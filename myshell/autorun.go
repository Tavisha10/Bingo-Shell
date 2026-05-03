package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tool represents a runnable tool with a trigger condition
type Tool struct {
	Name        string
	Command     string                // command to run
	Description string                // what it does
	Check       func(dir string) bool // when to suggest it
}

// toolSets maps project types to their relevant tools
var toolSets = map[ProjectType][]Tool{
	ProjectNode: {
		{
			Name:        "npm install",
			Command:     "npm install",
			Description: "Install dependencies",
			Check: func(dir string) bool {
				_, err := os.Stat(filepath.Join(dir, "node_modules"))
				return os.IsNotExist(err)
			},
		},
		{
			Name:        "npm run dev",
			Command:     "npm run dev",
			Description: "Start dev server",
			Check:       always,
		},
		{
			Name:        "npm test",
			Command:     "npm test",
			Description: "Run tests",
			Check:       always,
		},
		{
			Name:        "npm run build",
			Command:     "npm run build",
			Description: "Build for production",
			Check:       always,
		},
	},
	ProjectGo: {
		{
			Name:        "go mod tidy",
			Command:     "go mod tidy",
			Description: "Tidy dependencies",
			Check: func(dir string) bool {
				_, err := os.Stat(filepath.Join(dir, "go.sum"))
				return os.IsNotExist(err)
			},
		},
		{
			Name:        "go build",
			Command:     "go build ./...",
			Description: "Build the project",
			Check:       always,
		},
		{
			Name:        "go test",
			Command:     "go test ./...",
			Description: "Run all tests",
			Check:       always,
		},
		{
			Name:        "go fmt",
			Command:     "go fmt ./...",
			Description: "Format code",
			Check:       always,
		},
	},
	ProjectRust: {
		{
			Name:        "cargo build",
			Command:     "cargo build",
			Description: "Build the project",
			Check:       always,
		},
		{
			Name:        "cargo run",
			Command:     "cargo run",
			Description: "Build and run",
			Check:       always,
		},
		{
			Name:        "cargo test",
			Command:     "cargo test",
			Description: "Run tests",
			Check:       always,
		},
		{
			Name:        "cargo fmt",
			Command:     "cargo fmt",
			Description: "Format code",
			Check:       always,
		},
		{
			Name:        "cargo clippy",
			Command:     "cargo clippy",
			Description: "Lint code",
			Check:       always,
		},
	},
	ProjectPython: {
		{
			Name:        "pip install",
			Command:     "pip install -r requirements.txt",
			Description: "Install dependencies",
			Check: func(dir string) bool {
				_, err := os.Stat(filepath.Join(dir, "requirements.txt"))
				return err == nil
			},
		},
		{
			Name:        "pytest",
			Command:     "python -m pytest",
			Description: "Run tests",
			Check:       always,
		},
		{
			Name:        "python main",
			Command:     "python main.py",
			Description: "Run main.py",
			Check: func(dir string) bool {
				_, err := os.Stat(filepath.Join(dir, "main.py"))
				return err == nil
			},
		},
	},
	ProjectDeno: {
		{
			Name:        "deno run",
			Command:     "deno run main.ts",
			Description: "Run the project",
			Check:       always,
		},
		{
			Name:        "deno test",
			Command:     "deno test",
			Description: "Run tests",
			Check:       always,
		},
		{
			Name:        "deno fmt",
			Command:     "deno fmt",
			Description: "Format code",
			Check:       always,
		},
	},
	ProjectRuby: {
		{
			Name:        "bundle install",
			Command:     "bundle install",
			Description: "Install gems",
			Check:       always,
		},
		{
			Name:        "rspec",
			Command:     "rspec",
			Description: "Run tests",
			Check:       always,
		},
	},
	ProjectJava: {
		{
			Name:        "mvn compile",
			Command:     "mvn compile",
			Description: "Compile the project",
			Check:       always,
		},
		{
			Name:        "mvn test",
			Command:     "mvn test",
			Description: "Run tests",
			Check:       always,
		},
	},
}

// always is a check function that always returns true
func always(_ string) bool { return true }

// ---- Auto-Runner ----

// AutoRunner manages tool suggestions and execution
type AutoRunner struct {
	lastDir     string
	lastProject ProjectType
}

var autoRunner = &AutoRunner{}

// OnDirChange is called every time the user changes directory.
// It checks for urgent suggestions (e.g. missing node_modules)
// and prints them once.
func (a *AutoRunner) OnDirChange(dir string, project ProjectInfo) {
	if project.Type == ProjectUnknown {
		a.lastDir = dir
		a.lastProject = project.Type
		return
	}

	// only run checks when dir or project changes
	if dir == a.lastDir && project.Type == a.lastProject {
		return
	}
	a.lastDir = dir
	a.lastProject = project.Type

	tools, ok := toolSets[project.Type]
	if !ok {
		return
	}

	// collect urgent suggestions (check returns true = something missing/needed)
	var urgent []Tool
	for _, t := range tools {
		if t.Check != nil && t.Check(dir) && t.Name != "" {
			// only show tools where Check is non-trivial (not "always")
			// we detect "always" by checking if it's a named urgent condition
			if !isAlways(t.Check) {
				urgent = append(urgent, t)
			}
		}
	}

	if len(urgent) > 0 {
		fmt.Printf("\n  \033[1;33m[autorun]\033[0m %s project detected\n", project.Type)
		for _, t := range urgent {
			fmt.Printf("  \033[33m!\033[0m %s — run '\033[1mtools run %s\033[0m'\n",
				t.Description, t.Name)
		}
		fmt.Println()
	}
}

// isAlways checks if a function is the always() function by testing it
func isAlways(f func(string) bool) bool {
	// always() ignores the dir and always returns true
	// non-always checks actually look at the filesystem
	// we use a sentinel non-existent path to detect this
	return f("/tmp/__myshell_sentinel_nonexistent__")
}

// ---- Built-in: tools command ----

// runToolsCmd handles the `tools` built-in
// usage:
//
//	tools            → list available tools for current project
//	tools run <name> → run a specific tool by name
//	tools run all    → run all suggested tools in order
func runToolsCmd(args []string) {
	cwd, _ := os.Getwd()
	project := DetectProject(cwd)

	if project.Type == ProjectUnknown {
		fmt.Println("  No project detected in current directory.")
		fmt.Println("  Navigate to a project folder to see available tools.")
		return
	}

	tools, ok := toolSets[project.Type]
	if !ok {
		fmt.Printf("  No tools configured for %s projects.\n", project.Type)
		return
	}

	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: tools run <tool-name>")
			fmt.Fprintln(os.Stderr, "  example: tools run npm install")
			return
		}

		toolName := strings.Join(args[1:], " ")

		if toolName == "all" {
			runAllTools(tools, cwd)
			return
		}

		// find the tool by name
		for _, t := range tools {
			if t.Name == toolName {
				runTool(t)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "  tools: unknown tool '%s'\n", toolName)
		fmt.Fprintln(os.Stderr, "  use 'tools' to see available tools")

	default:
		// list available tools
		fmt.Printf("\n  \033[1m%s project tools\033[0m\n\n", project.Type)
		for i, t := range tools {
			urgentMark := " "
			if t.Check != nil && !isAlways(t.Check) && t.Check(cwd) {
				urgentMark = "\033[33m!\033[0m"
			}
			fmt.Printf("  %s  \033[1;36m%-20s\033[0m  \033[2m%s\033[0m\n",
				urgentMark, t.Name, t.Description)
			_ = i
		}
		fmt.Println()
		fmt.Println("  run with: \033[1mtools run <name>\033[0m")
		fmt.Println()
	}
}

// runTool executes a single tool
func runTool(t Tool) {
	fmt.Printf("\n  \033[1;32m▶\033[0m Running: \033[1m%s\033[0m\n\n", t.Command)

	parts := strings.Fields(t.Command)
	if len(parts) == 0 {
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(os.Stderr, "  tools: %v\n", err)
		}
	}
}

// runAllTools runs every tool in sequence
func runAllTools(tools []Tool, dir string) {
	fmt.Printf("\n  \033[1mRunning all tools...\033[0m\n")
	for _, t := range tools {
		if t.Check != nil && t.Check(dir) {
			runTool(t)
		}
	}
}
