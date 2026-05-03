package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvManager manages scoped environment variables
type EnvManager struct {
	// project-scoped vars (loaded from .env files)
	projectVars map[string]string
	// session vars (set manually during session)
	sessionVars map[string]string
	// track which .env file is currently loaded
	loadedEnvFile string
}

// global env manager instance
var envManager = &EnvManager{
	projectVars: make(map[string]string),
	sessionVars: make(map[string]string),
}

// ---- Core API ----

// Set sets a session-scoped environment variable
func (e *EnvManager) Set(key, value string) {
	e.sessionVars[key] = value
	os.Setenv(key, value)
}

// Get returns the value of a variable (session vars take priority)
func (e *EnvManager) Get(key string) (string, bool) {
	if val, ok := e.sessionVars[key]; ok {
		return val, true
	}
	if val, ok := e.projectVars[key]; ok {
		return val, true
	}
	// fall back to real OS env
	val := os.Getenv(key)
	return val, val != ""
}

// Clear removes a variable from session and project scope
func (e *EnvManager) Clear(key string) {
	delete(e.sessionVars, key)
	delete(e.projectVars, key)
	os.Unsetenv(key)
}

// List prints all managed variables
func (e *EnvManager) List() {
	if len(e.sessionVars) == 0 && len(e.projectVars) == 0 {
		fmt.Println("  No managed variables. Use 'env set KEY value' or 'env load .env'")
		return
	}

	if len(e.projectVars) > 0 {
		src := e.loadedEnvFile
		if src == "" {
			src = ".env"
		}
		fmt.Printf("  \033[1;33m[project — %s]\033[0m\n", src)
		for k, v := range e.projectVars {
			fmt.Printf("    \033[36m%s\033[0m = %s\n", k, maskSecret(k, v))
		}
	}

	if len(e.sessionVars) > 0 {
		fmt.Println("  \033[1;33m[session]\033[0m")
		for k, v := range e.sessionVars {
			fmt.Printf("    \033[36m%s\033[0m = %s\n", k, maskSecret(k, v))
		}
	}
}

// LoadFile parses a .env file and loads vars into project scope
func (e *EnvManager) LoadFile(path string) error {
	// resolve path
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("env: cannot open %s: %w", path, err)
	}
	defer f.Close()

	// clear previous project vars before loading new file
	e.projectVars = make(map[string]string)
	e.loadedEnvFile = filepath.Base(path)

	scanner := bufio.NewScanner(f)
	lineNum := 0
	loaded := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := parseEnvLine(line)
		if !ok {
			fmt.Fprintf(os.Stderr, "  env: skipping invalid line %d: %s\n", lineNum, line)
			continue
		}

		e.projectVars[key] = value
		os.Setenv(key, value)
		loaded++
	}

	fmt.Printf("  \033[32m✓\033[0m Loaded %d variable(s) from %s\n", loaded, filepath.Base(path))
	return nil
}

// UnloadFile clears all project-scoped vars and unsets them from OS
func (e *EnvManager) UnloadFile() {
	for k := range e.projectVars {
		os.Unsetenv(k)
	}
	e.projectVars = make(map[string]string)
	e.loadedEnvFile = ""
	fmt.Println("  Project env vars cleared.")
}

// AutoLoad checks the directory for a .env file and loads it silently
// Called automatically on cd
func (e *EnvManager) AutoLoad(dir string) {
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		// no .env file — unload previous if we had one
		if e.loadedEnvFile != "" {
			e.UnloadFile()
		}
		return
	}

	// don't reload if same file already loaded
	if e.loadedEnvFile == filepath.Base(envPath) {
		return
	}

	// silently load
	e.projectVars = make(map[string]string)
	e.loadedEnvFile = ""

	f, err := os.Open(envPath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		e.projectVars[key] = value
		os.Setenv(key, value)
		count++
	}

	if count > 0 {
		fmt.Printf("  \033[32m✓\033[0m Auto-loaded %d var(s) from .env\n", count)
		e.loadedEnvFile = ".env"
	}
}

// ---- Parsing ----

// parseEnvLine parses a line like KEY=value or KEY="value"
func parseEnvLine(line string) (string, string, bool) {
	// handle export KEY=value
	line = strings.TrimPrefix(line, "export ")
	line = strings.TrimSpace(line)

	idx := strings.IndexByte(line, '=')
	if idx <= 0 {
		return "", "", false
	}

	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])

	// validate key — must be alphanumeric + underscore
	for _, ch := range key {
		if !(ch >= 'A' && ch <= 'Z') && !(ch >= 'a' && ch <= 'z') &&
			!(ch >= '0' && ch <= '9') && ch != '_' {
			return "", "", false
		}
	}

	// strip surrounding quotes from value
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	// expand existing env vars in value
	value = os.ExpandEnv(value)

	return key, value, true
}

// maskSecret hides values for sensitive-looking keys
func maskSecret(key, value string) string {
	lower := strings.ToLower(key)
	sensitive := []string{"secret", "password", "passwd", "token", "api_key", "apikey", "private"}
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			if len(value) <= 4 {
				return "****"
			}
			return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
		}
	}
	return value
}

// ---- Built-in: env command ----

// runEnvCmd handles the `env` built-in
// usage:
//
//	env                  → list all managed vars
//	env list             → same as above
//	env set KEY value    → set a session variable
//	env get KEY          → print a variable's value
//	env clear KEY        → remove a variable
//	env load <file>      → load a .env file
//	env unload           → unload current .env file
func runEnvCmd(args []string) {
	if len(args) == 0 {
		envManager.List()
		return
	}

	switch args[0] {
	case "list":
		envManager.List()

	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: env set KEY value")
			return
		}
		key := args[1]
		value := strings.Join(args[2:], " ")
		envManager.Set(key, value)
		fmt.Printf("  \033[32m✓\033[0m %s = %s\n", key, maskSecret(key, value))

	case "get":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: env get KEY")
			return
		}
		val, ok := envManager.Get(args[1])
		if !ok {
			fmt.Fprintf(os.Stderr, "  env: %s is not set\n", args[1])
			return
		}
		fmt.Println(val)

	case "clear":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: env clear KEY")
			return
		}
		envManager.Clear(args[1])
		fmt.Printf("  \033[32m✓\033[0m Cleared %s\n", args[1])

	case "load":
		file := ".env"
		if len(args) >= 2 {
			file = args[1]
		}
		if err := envManager.LoadFile(file); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

	case "unload":
		envManager.UnloadFile()

	default:
		fmt.Fprintf(os.Stderr, "env: unknown subcommand '%s'\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: env [list|set|get|clear|load|unload]")
	}
}
