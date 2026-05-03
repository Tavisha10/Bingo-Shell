package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ShellConfig holds all user-configurable settings
type ShellConfig struct {
	// General
	HistoryLimit int    `toml:"history_limit"`
	HistoryFile  string `toml:"history_file"`
	ConfigFile   string `toml:"-"`

	// Prompt
	ShowGitInPrompt     bool `toml:"show_git_in_prompt"`
	ShowTimeInPrompt    bool `toml:"show_time_in_prompt"`
	ShowProjectInPrompt bool `toml:"show_project_in_prompt"`
	ShowExitCode        bool `toml:"show_exit_code"`

	// Explorer
	ExplorerShowHidden bool `toml:"explorer_show_hidden"`
	ExplorerMaxDepth   int  `toml:"explorer_max_depth"`

	// Auto-runner
	AutoRunnerEnabled bool `toml:"autorunner_enabled"`

	// Aliases
	Aliases map[string]string `toml:"aliases"`

	// Custom env defaults
	DefaultEnv map[string]string `toml:"default_env"`
}

// defaultConfig returns a config with sensible defaults
func defaultShellConfig() *ShellConfig {
	home, _ := os.UserHomeDir()
	return &ShellConfig{
		HistoryLimit:        5000,
		HistoryFile:         filepath.Join(home, ".myshell_history"),
		ShowGitInPrompt:     true,
		ShowTimeInPrompt:    true,
		ShowProjectInPrompt: true,
		ShowExitCode:        true,
		ExplorerShowHidden:  false,
		ExplorerMaxDepth:    3,
		AutoRunnerEnabled:   true,
		Aliases:             make(map[string]string),
		DefaultEnv:          make(map[string]string),
	}
}

// shellConfig is the global config instance
var shellConfig *ShellConfig

// configPath returns the path to the config file
func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "myshell", "config.toml")
}

// InitConfig loads config from disk, creating defaults if missing
func InitConfig() error {
	shellConfig = defaultShellConfig()
	shellConfig.ConfigFile = configPath()

	// ensure config dir exists
	dir := filepath.Dir(shellConfig.ConfigFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("config: cannot create dir: %w", err)
	}

	// if file doesn't exist, write defaults
	if _, err := os.Stat(shellConfig.ConfigFile); os.IsNotExist(err) {
		return writeDefaultConfig(shellConfig.ConfigFile)
	}

	// load existing config
	return loadConfig(shellConfig.ConfigFile)
}

// loadConfig reads and parses a TOML config file (minimal hand-rolled parser)
func loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// section headers [section]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}

		// key = value
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = stripQuotes(value)

		applyConfigValue(shellConfig, section, key, value)
	}

	return scanner.Err()
}

// applyConfigValue sets a config field from a parsed key/value
func applyConfigValue(cfg *ShellConfig, section, key, value string) {
	switch section {
	case "", "general":
		switch key {
		case "history_limit":
			cfg.HistoryLimit, _ = strconv.Atoi(value)
		case "history_file":
			cfg.HistoryFile = value
		}

	case "prompt":
		switch key {
		case "show_git":
			cfg.ShowGitInPrompt = parseBool(value)
		case "show_time":
			cfg.ShowTimeInPrompt = parseBool(value)
		case "show_project":
			cfg.ShowProjectInPrompt = parseBool(value)
		case "show_exit_code":
			cfg.ShowExitCode = parseBool(value)
		}

	case "explorer":
		switch key {
		case "show_hidden":
			cfg.ExplorerShowHidden = parseBool(value)
		case "max_depth":
			cfg.ExplorerMaxDepth, _ = strconv.Atoi(value)
		}

	case "autorunner":
		switch key {
		case "enabled":
			cfg.AutoRunnerEnabled = parseBool(value)
		}

	case "aliases":
		cfg.Aliases[key] = value

	case "env":
		cfg.DefaultEnv[key] = value
	}
}

// ApplyConfig pushes loaded config values into live modules
func ApplyConfig() {
	if shellConfig == nil {
		return
	}

	// apply to status bar
	statusBar.ShowGit = shellConfig.ShowGitInPrompt
	statusBar.ShowTime = shellConfig.ShowTimeInPrompt
	statusBar.ShowProject = shellConfig.ShowProjectInPrompt
	statusBar.ShowExit = shellConfig.ShowExitCode

	// apply to explorer defaults
	defaultConfig.ShowHidden = shellConfig.ExplorerShowHidden
	defaultConfig.MaxDepth = shellConfig.ExplorerMaxDepth

	// apply default env vars
	for k, v := range shellConfig.DefaultEnv {
		os.Setenv(k, v)
		envManager.sessionVars[k] = v
	}
}

// SaveConfig writes the current config back to disk
func SaveConfig() error {
	if shellConfig == nil {
		return nil
	}
	return writeConfig(shellConfig, shellConfig.ConfigFile)
}

// writeDefaultConfig creates a config file with annotated defaults
func writeDefaultConfig(path string) error {
	content := `# myshell configuration
# Generated automatically — edit freely

[general]
history_limit = 5000

[prompt]
show_git     = true
show_time    = true
show_project = true
show_exit_code = true

[explorer]
show_hidden = false
max_depth   = 3

[autorunner]
enabled = true

# Add shell aliases here
# [aliases]
# ll = "ls -la"
# gs = "git status"

# Set default environment variables here
# [env]
# NODE_ENV = "development"
# EDITOR   = "vim"
`
	return os.WriteFile(path, []byte(content), 0644)
}

// writeConfig serializes the config struct to TOML
func writeConfig(cfg *ShellConfig, path string) error {
	var sb strings.Builder

	sb.WriteString("# myshell configuration\n\n")

	sb.WriteString("[general]\n")
	sb.WriteString(fmt.Sprintf("history_limit = %d\n", cfg.HistoryLimit))
	sb.WriteString(fmt.Sprintf("history_file  = %q\n", cfg.HistoryFile))

	sb.WriteString("\n[prompt]\n")
	sb.WriteString(fmt.Sprintf("show_git       = %v\n", cfg.ShowGitInPrompt))
	sb.WriteString(fmt.Sprintf("show_time      = %v\n", cfg.ShowTimeInPrompt))
	sb.WriteString(fmt.Sprintf("show_project   = %v\n", cfg.ShowProjectInPrompt))
	sb.WriteString(fmt.Sprintf("show_exit_code = %v\n", cfg.ShowExitCode))

	sb.WriteString("\n[explorer]\n")
	sb.WriteString(fmt.Sprintf("show_hidden = %v\n", cfg.ExplorerShowHidden))
	sb.WriteString(fmt.Sprintf("max_depth   = %d\n", cfg.ExplorerMaxDepth))

	sb.WriteString("\n[autorunner]\n")
	sb.WriteString(fmt.Sprintf("enabled = %v\n", cfg.AutoRunnerEnabled))

	if len(cfg.Aliases) > 0 {
		sb.WriteString("\n[aliases]\n")
		for k, v := range cfg.Aliases {
			sb.WriteString(fmt.Sprintf("%s = %q\n", k, v))
		}
	}

	if len(cfg.DefaultEnv) > 0 {
		sb.WriteString("\n[env]\n")
		for k, v := range cfg.DefaultEnv {
			sb.WriteString(fmt.Sprintf("%s = %q\n", k, v))
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// resolveAlias checks if a command matches an alias and returns the expansion
func resolveAlias(command string) (string, bool) {
	if shellConfig == nil {
		return "", false
	}
	expanded, ok := shellConfig.Aliases[command]
	return expanded, ok
}

// ---- Built-in: config command ----

// runConfigCmd handles the `config` built-in
// usage:
//
//	config              → show current config
//	config get <key>    → get a config value
//	config set <key> <val> → set a config value
//	config edit         → open config file in $EDITOR
//	config reload       → reload config from disk
//	config alias <name> <cmd> → add an alias
func runConfigCmd(args []string) {
	if len(args) == 0 {
		printConfig()
		return
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: config get <key>")
			return
		}
		printConfigKey(args[1])

	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "  usage: config set <key> <value>")
			return
		}
		applyConfigValue(shellConfig, "", args[1], args[2])
		ApplyConfig()
		if err := SaveConfig(); err != nil {
			PrintError(err.Error())
			return
		}
		PrintSuccess(fmt.Sprintf("%s = %s (saved)", args[1], args[2]))

	case "edit":
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		cmd := fmt.Sprintf("%s %s", editor, shellConfig.ConfigFile)
		if err := ExecuteInShell(cmd); err != nil {
			PrintError(err.Error())
			return
		}
		// reload after editing
		loadConfig(shellConfig.ConfigFile)
		ApplyConfig()
		PrintSuccess("config reloaded")

	case "reload":
		if err := loadConfig(shellConfig.ConfigFile); err != nil {
			PrintError(err.Error())
			return
		}
		ApplyConfig()
		PrintSuccess("config reloaded from " + shellConfig.ConfigFile)

	case "alias":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "  usage: config alias <name> <command>")
			return
		}
		name := args[1]
		cmd := strings.Join(args[2:], " ")
		shellConfig.Aliases[name] = cmd
		SaveConfig()
		PrintSuccess(fmt.Sprintf("alias %s = %s", name, cmd))

	default:
		fmt.Fprintf(os.Stderr, "  config: unknown subcommand '%s'\n", args[0])
	}
}

func printConfig() {
	if shellConfig == nil {
		PrintError("config not loaded")
		return
	}
	fmt.Println()
	PrintSection("myshell config")
	fmt.Printf("  %s\n\n", ColorDim(shellConfig.ConfigFile))

	PrintKV([][2]string{
		{"history_limit", fmt.Sprintf("%d", shellConfig.HistoryLimit)},
		{"show_git", fmt.Sprintf("%v", shellConfig.ShowGitInPrompt)},
		{"show_time", fmt.Sprintf("%v", shellConfig.ShowTimeInPrompt)},
		{"show_project", fmt.Sprintf("%v", shellConfig.ShowProjectInPrompt)},
		{"show_exit_code", fmt.Sprintf("%v", shellConfig.ShowExitCode)},
		{"explorer_hidden", fmt.Sprintf("%v", shellConfig.ExplorerShowHidden)},
		{"explorer_depth", fmt.Sprintf("%d", shellConfig.ExplorerMaxDepth)},
		{"autorunner", fmt.Sprintf("%v", shellConfig.AutoRunnerEnabled)},
	})

	if len(shellConfig.Aliases) > 0 {
		PrintSection("aliases")
		for k, v := range shellConfig.Aliases {
			fmt.Printf("  %-16s → %s\n", k, v)
		}
		fmt.Println()
	}

	fmt.Printf("  edit with: %s\n\n", ColorInfo("config edit"))
}

func printConfigKey(key string) {
	cfg := shellConfig
	var value string
	switch key {
	case "history_limit":
		value = fmt.Sprintf("%d", cfg.HistoryLimit)
	case "show_git":
		value = fmt.Sprintf("%v", cfg.ShowGitInPrompt)
	case "show_time":
		value = fmt.Sprintf("%v", cfg.ShowTimeInPrompt)
	case "show_project":
		value = fmt.Sprintf("%v", cfg.ShowProjectInPrompt)
	case "show_exit_code":
		value = fmt.Sprintf("%v", cfg.ShowExitCode)
	case "explorer_depth":
		value = fmt.Sprintf("%d", cfg.ExplorerMaxDepth)
	default:
		PrintWarn("unknown config key: " + key)
		return
	}
	fmt.Printf("  %s = %s\n", key, value)
}

// ---- Helpers ----

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}

func stripQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}
