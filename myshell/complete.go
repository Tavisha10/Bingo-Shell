package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// BuildCompleter returns a readline.AutoCompleter that knows about
// all built-in commands and their subcommands
func BuildCompleter() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		// cd — complete with directories
		readline.PcItem("cd",
			readline.PcItemDynamic(listDirs),
		),

		// history subcommands
		readline.PcItem("history",
			readline.PcItem("search"),
			readline.PcItem("here"),
			readline.PcItem("stats"),
		),

		// explore subcommands + flags
		readline.PcItem("explore",
			readline.PcItem("-a"),
			readline.PcItem("-d"),
			readline.PcItem("--all"),
			readline.PcItem("--depth"),
			readline.PcItem("search"),
			readline.PcItem("preview",
				readline.PcItemDynamic(listFiles),
			),
		),

		// tools subcommands
		readline.PcItem("tools",
			readline.PcItem("run",
				readline.PcItemDynamic(listProjectTools),
			),
		),

		// env subcommands
		readline.PcItem("env",
			readline.PcItem("set"),
			readline.PcItem("get",
				readline.PcItemDynamic(listEnvKeys),
			),
			readline.PcItem("clear",
				readline.PcItemDynamic(listEnvKeys),
			),
			readline.PcItem("load",
				readline.PcItemDynamic(listEnvFiles),
			),
			readline.PcItem("unload"),
			readline.PcItem("list"),
		),

		// tree — no subcommands, takes a command string
		readline.PcItem("tree"),

		// platform subcommands
		readline.PcItem("platform",
			readline.PcItem("which"),
			readline.PcItem("open",
				readline.PcItemDynamic(listFiles),
			),
		),

		// config subcommands
		readline.PcItem("config",
			readline.PcItem("get",
				readline.PcItemDynamic(listConfigKeys),
			),
			readline.PcItem("set",
				readline.PcItemDynamic(listConfigKeys),
			),
			readline.PcItem("alias"),
			readline.PcItem("edit"),
			readline.PcItem("reload"),
		),

		// session subcommands
		readline.PcItem("session",
			readline.PcItem("save"),
			readline.PcItem("restore"),
			readline.PcItem("clear"),
		),

		// hooks subcommands
		readline.PcItem("hooks",
			readline.PcItem("add",
				readline.PcItem("on_cd"),
				readline.PcItem("on_cmd"),
				readline.PcItem("on_enter"),
				readline.PcItem("on_exit"),
				readline.PcItem("on_startup"),
			),
			readline.PcItem("remove",
				readline.PcItemDynamic(listHookNames),
			),
			readline.PcItem("fire",
				readline.PcItem("on_cd"),
				readline.PcItem("on_cmd"),
				readline.PcItem("on_enter"),
				readline.PcItem("on_exit"),
				readline.PcItem("on_startup"),
			),
			readline.PcItem("dir"),
		),

		// highlight — complete with source files
		readline.PcItem("highlight",
			readline.PcItemDynamic(listSourceFiles),
		),

		// render subcommands
		readline.PcItem("render",
			readline.PcItem("table"),
			readline.PcItem("box"),
			readline.PcItem("progress"),
			readline.PcItem("diff",
				readline.PcItemDynamic(listFiles),
			),
			readline.PcItem("highlight",
				readline.PcItemDynamic(listSourceFiles),
			),
		),

		// status subcommands
		readline.PcItem("status",
			readline.PcItem("git"),
			readline.PcItem("toggle",
				readline.PcItem("git"),
				readline.PcItem("time"),
				readline.PcItem("project"),
				readline.PcItem("exit"),
			),
		),

		// help — complete with command names
		readline.PcItem("help",
			readline.PcItem("cd"),
			readline.PcItem("history"),
			readline.PcItem("explore"),
			readline.PcItem("tools"),
			readline.PcItem("env"),
			readline.PcItem("tree"),
			readline.PcItem("platform"),
			readline.PcItem("config"),
			readline.PcItem("session"),
			readline.PcItem("hooks"),
			readline.PcItem("highlight"),
			readline.PcItem("render"),
			readline.PcItem("status"),
			readline.PcItem("clear"),
		),

		readline.PcItem("clear"),
		readline.PcItem("exit"),
	)
}

// ---- Dynamic completers ----

// listDirs returns subdirectories of the current directory for tab completion
func listDirs(line string) []string {
	return listFSEntries(line, true, false)
}

// listFiles returns files in the current directory
func listFiles(line string) []string {
	return listFSEntries(line, true, true)
}

// listSourceFiles returns source code files
func listSourceFiles(line string) []string {
	cwd, _ := os.Getwd()
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil
	}

	srcExts := map[string]bool{
		".go": true, ".py": true, ".js": true,
		".ts": true, ".rs": true, ".jsx": true, ".tsx": true,
	}

	var results []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if srcExts[ext] {
			results = append(results, e.Name())
		}
	}
	return results
}

// listEnvFiles returns .env files in the current directory
func listEnvFiles(line string) []string {
	cwd, _ := os.Getwd()
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil
	}

	var results []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".env") || e.Name() == ".env") {
			results = append(results, e.Name())
		}
	}
	return results
}

// listEnvKeys returns currently managed env var names
func listEnvKeys(line string) []string {
	var keys []string
	for k := range envManager.sessionVars {
		keys = append(keys, k)
	}
	for k := range envManager.projectVars {
		keys = append(keys, k)
	}
	return keys
}

// listProjectTools returns tool names for the current project
func listProjectTools(line string) []string {
	cwd, _ := os.Getwd()
	project := DetectProject(cwd)
	tools, ok := toolSets[project.Type]
	if !ok {
		return []string{"all"}
	}

	names := []string{"all"}
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}

// listHookNames returns currently registered hook names
func listHookNames(line string) []string {
	var names []string
	for _, h := range hookManager.hooks {
		names = append(names, h.Name)
	}
	return names
}

// listConfigKeys returns all known config key names
func listConfigKeys(line string) []string {
	return []string{
		"history_limit",
		"show_git",
		"show_time",
		"show_project",
		"show_exit_code",
		"explorer_depth",
		"explorer_hidden",
		"autorunner",
	}
}

// listFSEntries lists files and/or directories for completion
func listFSEntries(line string, includeDirs, includeFiles bool) []string {
	cwd, _ := os.Getwd()
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return nil
	}

	var results []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue // skip hidden
		}
		if e.IsDir() && includeDirs {
			results = append(results, e.Name()+"/")
		} else if !e.IsDir() && includeFiles {
			results = append(results, e.Name())
		}
	}
	return results
}
