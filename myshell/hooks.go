package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HookEvent represents when a hook fires
type HookEvent string

const (
	HookOnCD      HookEvent = "on_cd"      // fires when directory changes
	HookOnCmd     HookEvent = "on_cmd"     // fires after every command
	HookOnEnter   HookEvent = "on_enter"   // fires when entering a project dir
	HookOnExit    HookEvent = "on_exit"    // fires when shell exits
	HookOnStartup HookEvent = "on_startup" // fires once on shell start
)

// Hook is a single registered hook
type Hook struct {
	Event   HookEvent
	Name    string // human label
	Command string // shell command to run
	Once    bool   // fire only once
	fired   bool   // internal — tracks if already fired
}

// HookManager manages all registered hooks
type HookManager struct {
	hooks []*Hook
}

var hookManager = &HookManager{}

// ---- Registration ----

// Register adds a new hook
func (h *HookManager) Register(event HookEvent, name, command string) {
	h.hooks = append(h.hooks, &Hook{
		Event:   event,
		Name:    name,
		Command: command,
	})
}

// RegisterOnce adds a hook that fires only once
func (h *HookManager) RegisterOnce(event HookEvent, name, command string) {
	h.hooks = append(h.hooks, &Hook{
		Event:   event,
		Name:    name,
		Command: command,
		Once:    true,
	})
}

// Remove deletes a hook by name
func (h *HookManager) Remove(name string) bool {
	for i, hook := range h.hooks {
		if hook.Name == name {
			h.hooks = append(h.hooks[:i], h.hooks[i+1:]...)
			return true
		}
	}
	return false
}

// ---- Firing ----

// Fire runs all hooks registered for an event
func (h *HookManager) Fire(event HookEvent, ctx map[string]string) {
	for _, hook := range h.hooks {
		if hook.Event != event {
			continue
		}
		if hook.Once && hook.fired {
			continue
		}
		hook.fired = true
		runHook(hook, ctx)
	}
}

// runHook executes a single hook command with context variables injected
func runHook(hook *Hook, ctx map[string]string) {
	command := hook.Command

	// substitute context variables like $MYSHELL_DIR, $MYSHELL_PROJECT
	for k, v := range ctx {
		command = strings.ReplaceAll(command, "$"+k, v)
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), contextToEnv(ctx)...)

	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(os.Stderr, "  hook [%s] error: %v\n", hook.Name, err)
		}
	}
}

// contextToEnv converts a context map to KEY=VALUE strings
func contextToEnv(ctx map[string]string) []string {
	var env []string
	for k, v := range ctx {
		env = append(env, k+"="+v)
	}
	return env
}

// ---- Built-in hook fires ----

// FireCD fires the on_cd hook with directory context
func FireCD(newDir string) {
	project := DetectProject(newDir)
	hookManager.Fire(HookOnCD, map[string]string{
		"MYSHELL_DIR":     newDir,
		"MYSHELL_PROJECT": string(project.Type),
	})
}

// FireCmd fires the on_cmd hook after a command runs
func FireCmd(command string) {
	cwd, _ := os.Getwd()
	hookManager.Fire(HookOnCmd, map[string]string{
		"MYSHELL_CMD": command,
		"MYSHELL_DIR": cwd,
	})
}

// FireEnter fires the on_enter hook when entering a project root
func FireEnter(dir string, project ProjectInfo) {
	hookManager.Fire(HookOnEnter, map[string]string{
		"MYSHELL_DIR":          dir,
		"MYSHELL_PROJECT":      string(project.Type),
		"MYSHELL_PROJECT_NAME": project.Name,
	})
}

// FireExit fires the on_exit hook before shutdown
func FireExit() {
	hookManager.Fire(HookOnExit, map[string]string{})
}

// FireStartup fires the on_startup hook once on launch
func FireStartup() {
	cwd, _ := os.Getwd()
	hookManager.Fire(HookOnStartup, map[string]string{
		"MYSHELL_DIR": cwd,
	})
}

// ---- Hook script loader ----

// LoadHooksFromDir loads .sh hook scripts from ~/.config/myshell/hooks/
// Files named on_cd.sh, on_cmd.sh etc. are auto-registered
func LoadHooksFromDir() {
	home, _ := os.UserHomeDir()
	hooksDir := filepath.Join(home, ".config", "myshell", "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return
	}

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return
	}

	eventMap := map[string]HookEvent{
		"on_cd":      HookOnCD,
		"on_cmd":     HookOnCmd,
		"on_enter":   HookOnEnter,
		"on_exit":    HookOnExit,
		"on_startup": HookOnStartup,
	}

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), ".sh")
		event, ok := eventMap[base]
		if !ok {
			continue
		}

		scriptPath := filepath.Join(hooksDir, entry.Name())
		hookManager.Register(event, base, "sh "+scriptPath)
		loaded++
	}

	if loaded > 0 {
		fmt.Printf("  %s Loaded %d hook(s) from %s\n\n",
			ColorDim("⚡"), loaded, hooksDir)
	}
}

// WriteExampleHooks writes example hook scripts if none exist
func WriteExampleHooks() {
	home, _ := os.UserHomeDir()
	hooksDir := filepath.Join(home, ".config", "myshell", "hooks")
	os.MkdirAll(hooksDir, 0755)

	examples := map[string]string{
		"on_cd.sh": `#!/bin/sh
# Fires every time you cd into a new directory
# Available variables: MYSHELL_DIR, MYSHELL_PROJECT
# echo "moved to $MYSHELL_DIR"
`,
		"on_enter.sh": `#!/bin/sh
# Fires when entering a project root directory
# Available variables: MYSHELL_DIR, MYSHELL_PROJECT, MYSHELL_PROJECT_NAME
# echo "entered $MYSHELL_PROJECT project: $MYSHELL_PROJECT_NAME"
`,
		"on_startup.sh": `#!/bin/sh
# Fires once when the shell starts
# Available variables: MYSHELL_DIR
# echo "welcome back!"
`,
	}

	for name, content := range examples {
		path := filepath.Join(hooksDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			os.WriteFile(path, []byte(content), 0755)
		}
	}
}

// ---- Built-in: hooks command ----

// runHooksCmd handles the `hooks` built-in
// usage:
//
//	hooks                         → list all registered hooks
//	hooks add <event> <name> <cmd>→ add a hook
//	hooks remove <name>           → remove a hook by name
//	hooks fire <event>            → manually fire an event
//	hooks dir                     → open hooks directory
func runHooksCmd(args []string) {
	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "add":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "  usage: hooks add <event> <name> <command>")
			fmt.Fprintln(os.Stderr, "  events: on_cd · on_cmd · on_enter · on_exit · on_startup")
			return
		}
		event := HookEvent(args[1])
		name := args[2]
		command := strings.Join(args[3:], " ")
		hookManager.Register(event, name, command)
		PrintSuccess(fmt.Sprintf("hook '%s' registered for %s", name, event))

	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: hooks remove <name>")
			return
		}
		if hookManager.Remove(args[1]) {
			PrintSuccess("hook '" + args[1] + "' removed")
		} else {
			PrintError("hook '" + args[1] + "' not found")
		}

	case "fire":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: hooks fire <event>")
			return
		}
		cwd, _ := os.Getwd()
		hookManager.Fire(HookEvent(args[1]), map[string]string{
			"MYSHELL_DIR": cwd,
		})
		PrintSuccess(fmt.Sprintf("fired event: %s", args[1]))

	case "dir":
		home, _ := os.UserHomeDir()
		hooksDir := filepath.Join(home, ".config", "myshell", "hooks")
		fmt.Printf("  hooks directory: %s\n\n", ColorInfo(hooksDir))
		OpenFile(hooksDir)

	default:
		printHooksList()
	}
}

func printHooksList() {
	fmt.Println()
	PrintSection("registered hooks")

	if len(hookManager.hooks) == 0 {
		fmt.Println("  No hooks registered.")
		fmt.Println()
		fmt.Printf("  Add hooks with: %s\n", ColorInfo("hooks add <event> <name> <cmd>"))
		fmt.Printf("  Or drop .sh scripts in: %s\n", ColorInfo("hooks dir"))
		fmt.Println()
		return
	}

	t := NewTable("event", "name", "command", "once")
	for _, h := range hookManager.hooks {
		once := ""
		if h.Once {
			once = "yes"
		}
		t.AddRow(string(h.Event), h.Name, h.Command, once)
	}
	t.Print()

	fmt.Printf("  remove with: %s\n\n", ColorInfo("hooks remove <name>"))
}
