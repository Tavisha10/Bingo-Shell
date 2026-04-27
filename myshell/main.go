package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

var history []string

func addToHistory(command string) {
	if command != "" {
		history = append(history, command)
	}
}

func saveHistory(rl *readline.Instance) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	rl.SaveHistory(filepath.Join(home, ".myshell_history"))
}

func loadHistory() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	f, err := os.Open(filepath.Join(home, ".myshell_history"))
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		history = append(history, scanner.Text())
	}
}

// prompt

func getPrompt() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "?"
	}
	home, _ := os.UserHomeDir()
	display := strings.Replace(cwd, home, "~", 1)

	// Detect project and show type in prompt
	project := DetectProject(cwd)
	projectTag := ""
	if project.Type != ProjectUnknown {
		projectTag = fmt.Sprintf("\033[0;33m%s\033[0m ", project.Type.Emoji())
	}

	return fmt.Sprintf("\033[1;36m%s\033[0m %s\033[1;32m❯\033[0m ", display, projectTag)
}

// Built-in commands

var prevDir string // add this near the top, next to the history variable

func runCD(args []string) {
	current, _ := os.Getwd()

	var target string
	if len(args) == 0 {
		target, _ = os.UserHomeDir()
	} else if args[0] == "-" {
		if prevDir == "" {
			fmt.Fprintln(os.Stderr, "cd: no previous directory")
			return
		}
		target = prevDir
		fmt.Println(target)
	} else {
		target = args[0]
		// expand ~ to home directory
		if target == "~" {
			target, _ = os.UserHomeDir()
		} else if len(target) > 1 && target[:2] == "~/" {
			home, _ := os.UserHomeDir()
			target = home + target[1:]
		}
	}

	if err := os.Chdir(target); err != nil {
		fmt.Fprintf(os.Stderr, "cd: %v\n", err)
		return
	}
	prevDir = current
	// auto-load .env if present
	envManager.AutoLoad(target)
}

// ---------- External command execution ----------

func runExternal(name string, args []string) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Don't print error for normal non-zero exits (e.g. grep with no match)
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(os.Stderr, "myshell: %s: %v\n", name, err)
		}
	}
}

//---------- Tokenization ----------

// tokenize splits a line into tokens, respecting quoted strings.
// Example: `echo "hello world"` → ["echo", "hello world"]
func tokenize(line string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range line {
		switch {
		case inQuote:
			if ch == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(ch)
			}
		case ch == '"' || ch == '\'':
			inQuote = true
			quoteChar = ch
		case ch == ' ' || ch == '\t':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// ---------- Redirect parsing ----------

// parseRedirects strips >, >>, < tokens from args and returns
// the cleaned args plus the redirect targets.
type redirects struct {
	stdout    string // > file
	stdoutApp string // >> file
	stdin     string // < file
}

func parseRedirects(tokens []string) ([]string, redirects) {
	var clean []string
	var r redirects

	i := 0
	for i < len(tokens) {
		switch tokens[i] {
		case ">":
			if i+1 < len(tokens) {
				r.stdout = tokens[i+1]
				i += 2
			}
		case ">>":
			if i+1 < len(tokens) {
				r.stdoutApp = tokens[i+1]
				i += 2
			}
		case "<":
			if i+1 < len(tokens) {
				r.stdin = tokens[i+1]
				i += 2
			}
		default:
			clean = append(clean, tokens[i])
			i++
		}
	}
	return clean, r
}

// ---------- Single command runner ----------

// runCommand builds and starts an exec.Cmd. stdin/stdout can be
// overridden for piping; if nil, defaults to os.Stdin/os.Stdout.
func runCommand(tokens []string, stdin io.Reader, stdout io.Writer) error {
	tokens, r := parseRedirects(tokens)
	if len(tokens) == 0 {
		return nil
	}

	name := tokens[0]
	args := tokens[1:]

	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr

	// stdin source
	if r.stdin != "" {
		f, err := os.Open(r.stdin)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		defer f.Close()
		cmd.Stdin = f
	} else if stdin != nil {
		cmd.Stdin = stdin
	} else {
		cmd.Stdin = os.Stdin
	}

	// stdout destination
	if r.stdout != "" {
		f, err := os.Create(r.stdout)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		defer f.Close()
		cmd.Stdout = f
	} else if r.stdoutApp != "" {
		f, err := os.OpenFile(r.stdoutApp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		defer f.Close()
		cmd.Stdout = f
	} else if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}

	return cmd.Run()
}

// ---------- Pipeline runner ----------

// splitPipe splits tokens on "|" into segments.
func splitPipe(tokens []string) [][]string {
	var segments [][]string
	var current []string
	for _, t := range tokens {
		if t == "|" {
			if len(current) > 0 {
				segments = append(segments, current)
				current = nil
			}
		} else {
			current = append(current, t)
		}
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}

func runPipeline(segments [][]string) {
	if len(segments) == 1 {
		// No pipe — simple command
		if err := runCommand(segments[0], nil, nil); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				fmt.Fprintf(os.Stderr, "myshell: %v\n", err)
			}
		}
		return
	}

	// Build a chain of pipes between N commands
	cmds := make([]*exec.Cmd, len(segments))
	for i, seg := range segments {
		seg, _ = parseRedirects(seg) // redirects inside pipes are rare but handle it
		cmds[i] = exec.Command(seg[0], seg[1:]...)
		cmds[i].Stderr = os.Stderr
	}

	// Wire up stdin → cmd[0] → pipe → cmd[1] → ... → os.Stdout
	cmds[0].Stdin = os.Stdin
	cmds[len(cmds)-1].Stdout = os.Stdout

	for i := 0; i < len(cmds)-1; i++ {
		pipe, err := cmds[i].StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "myshell: pipe: %v\n", err)
			return
		}
		cmds[i+1].Stdin = pipe
	}

	// Start all commands
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "myshell: %v\n", err)
			return
		}
	}

	// Wait for all commands to finish
	for _, cmd := range cmds {
		cmd.Wait()
	}
}

// ---------- Execute a full line ----------
func executeLine(line string) bool {
	tokens := tokenize(line)
	tokens = expandEnv(tokens)
	if len(tokens) == 0 {
		return true
	}

	// Built-ins (must run in shell process, not subprocess)
	switch tokens[0] {
	case "exit":
		return false
	case "cd":
		runCD(tokens[1:])
		return true
	case "history":
		runHistoryCmd(tokens[1:])
		return true
	case "env":
		runEnvCmd(tokens[1:])
		return true
	case "tree":
		runTreeCmd(tokens[1:])
		return true
	}

	// Split on pipes and run
	segments := splitPipe(tokens)
	runPipeline(segments)
	return true
}

// ---------- Environment variable expansion ----------

// expandEnv replaces $VAR references in tokens with their values
func expandEnv(tokens []string) []string {
	expanded := make([]string, len(tokens))
	for i, t := range tokens {
		expanded[i] = os.ExpandEnv(t)
	}
	return expanded
}

// ---------- Main REPL ----------

func main() {
	home, _ := os.UserHomeDir()
	historyFile := filepath.Join(home, ".myshell_history")

	// Init SQLite history
	if err := InitHistory(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: history db failed:", err)
	}
	defer historyDB.Close()

	rl, err := readline.NewEx(&readline.Config{
		HistoryFile:         historyFile,
		HistoryLimit:        5000,
		HistorySearchFold:   true,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		ForceUseInteractive: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "readline init error:", err)
		os.Exit(1)
	}
	defer rl.Close()

	fmt.Println("myshell 0.3 — SQLite history · project detection")

	for {
		rl.SetPrompt(getPrompt())

		line, err := rl.Readline()
		if err != nil {
			fmt.Println("\nBye!")
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		addToHistory(line)

		// Save to SQLite with context
		cwd, _ := os.Getwd()
		project := DetectProject(cwd)
		if historyDB != nil {
			historyDB.Add(line, cwd, string(project.Type), 0)
		}

		if !executeLine(line) {
			fmt.Println("Bye!")
			saveHistory(rl)
			break
		}
	}
}
