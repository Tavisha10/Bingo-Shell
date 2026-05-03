package main

import (
	"fmt"
	"os"
	"strings"
)

// HelpEntry defines the help content for a built-in command
type HelpEntry struct {
	Command     string
	Short       string // one-line description
	Usage       string // usage string
	Description string // longer description
	Examples    []string
	Subcommands []SubcmdHelp
}

type SubcmdHelp struct {
	Name        string
	Description string
}

// helpRegistry maps command names to their help entries
var helpRegistry = map[string]HelpEntry{
	"cd": {
		Command:     "cd",
		Short:       "Change the working directory",
		Usage:       "cd [directory]",
		Description: "Changes the current working directory. Supports ~ for home, - for previous directory, and relative or absolute paths.",
		Examples: []string{
			"cd ~             # go home",
			"cd ..            # go up one level",
			"cd -             # go to previous directory",
			"cd ~/Documents   # go to Documents",
		},
	},
	"history": {
		Command:     "history",
		Short:       "Search and view command history",
		Usage:       "history [subcommand]",
		Description: "Manages shell history stored in SQLite. Supports full-text search, filtering by directory, and usage statistics.",
		Examples: []string{
			"history              # show last 50 commands",
			"history search git   # find commands containing 'git'",
			"history here         # commands run in this directory",
			"history stats        # most used commands",
		},
		Subcommands: []SubcmdHelp{
			{"search <query>", "fuzzy search history"},
			{"here", "commands run in current directory"},
			{"stats", "usage statistics"},
		},
	},
	"explore": {
		Command:     "explore",
		Short:       "Browse the file system as a tree",
		Usage:       "explore [path] [flags]",
		Description: "Displays a color-coded file tree. Supports hidden files, depth limiting, file search, and preview.",
		Examples: []string{
			"explore                  # tree of current dir",
			"explore ~/Documents      # tree of a specific path",
			"explore -a               # include hidden files",
			"explore -d 2             # limit to 2 levels deep",
			"explore search .go       # find .go files",
			"explore preview main.go  # preview a file",
		},
		Subcommands: []SubcmdHelp{
			{"search <pattern>", "search for files by name"},
			{"preview <file>", "preview file contents"},
			{"-a / --all", "show hidden files"},
			{"-d / --depth <n>", "set max depth"},
		},
	},
	"tools": {
		Command:     "tools",
		Short:       "Run project-specific tools",
		Usage:       "tools [run <name>]",
		Description: "Detects the current project type and lists relevant tools. Suggests tools when entering a project directory.",
		Examples: []string{
			"tools                  # list available tools",
			"tools run go test      # run a specific tool",
			"tools run all          # run all tools in order",
		},
		Subcommands: []SubcmdHelp{
			{"run <name>", "run a tool by name"},
			{"run all", "run all tools in sequence"},
		},
	},
	"env": {
		Command:     "env",
		Short:       "Manage environment variables",
		Usage:       "env [subcommand]",
		Description: "Manages session and project-scoped environment variables. Supports .env file loading and automatic loading on cd.",
		Examples: []string{
			"env                      # list all managed vars",
			"env set NAME tavisha     # set a variable",
			"env get NAME             # get a variable",
			"env clear NAME           # remove a variable",
			"env load .env            # load a .env file",
			"env unload               # unload current .env",
		},
		Subcommands: []SubcmdHelp{
			{"set <key> <value>", "set a session variable"},
			{"get <key>", "print a variable's value"},
			{"clear <key>", "remove a variable"},
			{"load [file]", "load a .env file"},
			{"unload", "unload project env vars"},
			{"list", "list all managed vars"},
		},
	},
	"tree": {
		Command:     "tree",
		Short:       "Visualize a command's structure",
		Usage:       "tree <command>",
		Description: "Parses and displays a command as a tree, showing pipes, redirects, and chained commands visually.",
		Examples: []string{
			"tree ls | grep .go | wc -l",
			"tree git add . && git commit -m 'msg'",
			"tree cat file.txt > output.txt",
		},
	},
	"platform": {
		Command:     "platform",
		Short:       "Show platform and environment info",
		Usage:       "platform [subcommand]",
		Description: "Displays OS, shell, home directory, and available dev tools. Supports cross-platform path normalization.",
		Examples: []string{
			"platform              # full platform info",
			"platform which git    # find a command's path",
			"platform open file    # open file in default app",
		},
		Subcommands: []SubcmdHelp{
			{"which <cmd>", "find command path"},
			{"open <file>", "open with default app"},
		},
	},
	"config": {
		Command:     "config",
		Short:       "View and edit shell configuration",
		Usage:       "config [subcommand]",
		Description: "Manages the shell config file at ~/.config/myshell/config.toml. Supports aliases, prompt toggles, and env defaults.",
		Examples: []string{
			"config                        # show all settings",
			"config get show_git           # get a value",
			"config set show_time false    # set a value",
			"config alias ll 'ls -la'      # create an alias",
			"config edit                   # open in $EDITOR",
			"config reload                 # reload from disk",
		},
		Subcommands: []SubcmdHelp{
			{"get <key>", "get a config value"},
			{"set <key> <val>", "set a config value"},
			{"alias <name> <cmd>", "create a command alias"},
			{"edit", "open config in $EDITOR"},
			{"reload", "reload config from disk"},
		},
	},
	"session": {
		Command:     "session",
		Short:       "Save and restore shell sessions",
		Usage:       "session [subcommand]",
		Description: "Saves and restores working directory, environment variables, and project context across shell restarts.",
		Examples: []string{
			"session              # show session info",
			"session save         # save current session",
			"session restore      # restore last session",
			"session clear        # delete saved session",
		},
		Subcommands: []SubcmdHelp{
			{"save", "save current session"},
			{"restore", "restore last session"},
			{"clear", "delete saved session"},
		},
	},
	"hooks": {
		Command:     "hooks",
		Short:       "Manage shell event hooks",
		Usage:       "hooks [subcommand]",
		Description: "Registers shell scripts to run on events like cd, command execution, or startup. Scripts can also be dropped in ~/.config/myshell/hooks/.",
		Examples: []string{
			"hooks                              # list all hooks",
			"hooks add on_cd logger 'echo $MYSHELL_DIR'",
			"hooks remove logger               # remove a hook",
			"hooks fire on_cd                  # manually fire",
			"hooks dir                         # open hooks folder",
		},
		Subcommands: []SubcmdHelp{
			{"add <event> <name> <cmd>", "register a hook"},
			{"remove <name>", "remove a hook"},
			{"fire <event>", "manually fire an event"},
			{"dir", "open hooks directory"},
		},
	},
	"highlight": {
		Command:     "highlight",
		Short:       "Syntax highlight a source file",
		Usage:       "highlight <file> [lines]",
		Description: "Prints a file with syntax highlighting. Supports Go, Python, JavaScript, TypeScript, and Rust.",
		Examples: []string{
			"highlight main.go         # highlight entire file",
			"highlight main.go 30      # first 30 lines",
			"highlight detector.go 10  # first 10 lines",
		},
	},
	"render": {
		Command:     "render",
		Short:       "Render tables, boxes, diffs, and more",
		Usage:       "render <subcommand>",
		Description: "Unified output renderer. Draws aligned tables, labeled boxes, progress bars, and colorized diffs.",
		Examples: []string{
			"render table              # demo table",
			"render box title content  # labeled box",
			"render progress           # progress bar demo",
			"render diff file.patch    # colorized diff",
			"render highlight file.go  # syntax highlight",
		},
		Subcommands: []SubcmdHelp{
			{"table", "demo aligned table"},
			{"box <title> <text>", "draw a labeled box"},
			{"progress", "demo progress bars"},
			{"diff <file>", "colorize a patch file"},
			{"highlight <file>", "syntax highlight"},
		},
	},
	"status": {
		Command:     "status",
		Short:       "Show shell and git status",
		Usage:       "status [subcommand]",
		Description: "Displays current shell state including git branch, project type, platform info, and prompt toggle settings.",
		Examples: []string{
			"status                    # full status panel",
			"status git                # git details only",
			"status toggle time        # hide clock in prompt",
			"status toggle git         # hide git in prompt",
		},
		Subcommands: []SubcmdHelp{
			{"git", "show git status details"},
			{"toggle <key>", "toggle prompt segment (git|time|project|exit)"},
		},
	},
	"clear": {
		Command:     "clear",
		Short:       "Clear the terminal screen",
		Usage:       "clear",
		Description: "Clears the terminal screen. Works cross-platform on Mac, Linux, and Windows.",
		Examples:    []string{"clear"},
	},
}

// ---- Built-in: help command ----

// runHelpCmd handles the `help` built-in
// usage:
//
//	help              → list all commands
//	help <command>    → detailed help for a command
func runHelpCmd(args []string) {
	if len(args) == 0 {
		printAllHelp()
		return
	}

	cmd := strings.ToLower(args[0])
	entry, ok := helpRegistry[cmd]
	if !ok {
		fmt.Fprintf(os.Stderr, "  help: unknown command '%s'\n", cmd)
		fmt.Fprintln(os.Stderr, "  run 'help' to see all commands")
		return
	}

	printDetailedHelp(entry)
}

func printAllHelp() {
	fmt.Println()
	PrintSection("myshell — built-in commands")

	// group commands
	groups := []struct {
		title    string
		commands []string
	}{
		{"Navigation", []string{"cd", "explore", "clear"}},
		{"History", []string{"history"}},
		{"Project", []string{"tools", "env", "platform"}},
		{"Visualization", []string{"tree", "highlight", "render"}},
		{"Shell", []string{"config", "session", "hooks", "status"}},
	}

	for _, group := range groups {
		fmt.Printf("  %s%s%s\n", ansiDim, group.title, ansiReset)
		for _, cmd := range group.commands {
			if entry, ok := helpRegistry[cmd]; ok {
				fmt.Printf("    %-16s %s\n", ColorInfo(entry.Command), entry.Short)
			}
		}
		fmt.Println()
	}

	fmt.Printf("  detailed help: %s\n\n", ColorInfo("help <command>"))
}

func printDetailedHelp(e HelpEntry) {
	fmt.Println()
	fmt.Printf("  %s%s%s  —  %s\n\n", ansiBold, e.Command, ansiReset, e.Short)

	fmt.Printf("  %sUsage:%s  %s\n\n", ansiDim, ansiReset, e.Usage)

	if e.Description != "" {
		// word-wrap at 60 chars
		words := strings.Fields(e.Description)
		line := "  "
		for _, w := range words {
			if len(line)+len(w)+1 > 64 {
				fmt.Println(line)
				line = "  " + w
			} else {
				if line == "  " {
					line += w
				} else {
					line += " " + w
				}
			}
		}
		if line != "  " {
			fmt.Println(line)
		}
		fmt.Println()
	}

	if len(e.Subcommands) > 0 {
		fmt.Printf("  %sSubcommands:%s\n", ansiDim, ansiReset)
		for _, sub := range e.Subcommands {
			fmt.Printf("    %-32s %s\n", ColorInfo(sub.Name), sub.Description)
		}
		fmt.Println()
	}

	if len(e.Examples) > 0 {
		fmt.Printf("  %sExamples:%s\n", ansiDim, ansiReset)
		for _, ex := range e.Examples {
			// split on # for inline comment
			parts := strings.SplitN(ex, "#", 2)
			code := strings.TrimSpace(parts[0])
			comment := ""
			if len(parts) > 1 {
				comment = "  " + ColorDim("# "+strings.TrimSpace(parts[1]))
			}
			fmt.Printf("    %s%s%s%s\n", ansiGreen, code, ansiReset, comment)
		}
		fmt.Println()
	}
}
