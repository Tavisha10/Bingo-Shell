package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform represents the detected OS/environment
type Platform string

const (
	PlatformMac     Platform = "mac"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformWSL     Platform = "wsl"
)

// PlatformInfo holds everything detected about the current platform
type PlatformInfo struct {
	OS          Platform
	Shell       string // default shell (zsh, bash, fish, powershell)
	PathSep     string // / or \
	HomePath    string // home directory
	TempPath    string // temp directory
	IsWSL       bool   // running inside WSL
	WindowsHome string // windows home if WSL (e.g. /mnt/c/Users/tavisha)
}

// platform is the global detected platform info
var platform *PlatformInfo

// InitPlatform detects and stores the current platform info
func InitPlatform() {
	platform = &PlatformInfo{}
	platform.detect()
}

func (p *PlatformInfo) detect() {
	// detect OS
	switch runtime.GOOS {
	case "darwin":
		p.OS = PlatformMac
		p.PathSep = "/"
	case "linux":
		if isWSL() {
			p.OS = PlatformWSL
			p.IsWSL = true
		} else {
			p.OS = PlatformLinux
		}
		p.PathSep = "/"
	case "windows":
		p.OS = PlatformWindows
		p.PathSep = "\\"
	default:
		p.OS = PlatformLinux
		p.PathSep = "/"
	}

	// detect home
	home, err := os.UserHomeDir()
	if err == nil {
		p.HomePath = home
	}

	// detect temp
	p.TempPath = os.TempDir()

	// detect shell
	p.Shell = p.detectShell()

	// if WSL, find windows home
	if p.IsWSL {
		p.WindowsHome = p.detectWindowsHome()
	}
}

// isWSL checks if we're running inside Windows Subsystem for Linux
func isWSL() bool {
	// check /proc/version for microsoft/WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl") {
			return true
		}
	}
	// check WSL_DISTRO_NAME env var
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	return false
}

// detectShell finds the best shell available on this system
func (p *PlatformInfo) detectShell() string {
	// check $SHELL env first
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}

	// fallback: probe common shells in order of preference
	var preferred []string
	switch p.OS {
	case PlatformMac:
		preferred = []string{"zsh", "bash", "sh"}
	case PlatformWindows:
		preferred = []string{"powershell", "cmd"}
	default:
		preferred = []string{"bash", "zsh", "sh"}
	}

	for _, sh := range preferred {
		if path, err := exec.LookPath(sh); err == nil {
			return path
		}
	}
	return "sh"
}

// detectWindowsHome finds the windows user home from inside WSL
func (p *PlatformInfo) detectWindowsHome() string {
	// try USERPROFILE first (sometimes set in WSL)
	if up := os.Getenv("USERPROFILE"); up != "" {
		return up
	}
	// derive from windows username via /mnt/c/Users/
	out, err := exec.Command("cmd.exe", "/c", "echo %USERPROFILE%").Output()
	if err != nil {
		return ""
	}
	winPath := strings.TrimSpace(string(out))
	// convert C:\Users\tavisha → /mnt/c/Users/tavisha
	return winPathToWSL(winPath)
}

// winPathToWSL converts a Windows path to its WSL equivalent
// e.g. C:\Users\tavisha → /mnt/c/Users/tavisha
func winPathToWSL(winPath string) string {
	winPath = strings.ReplaceAll(winPath, "\\", "/")
	if len(winPath) >= 2 && winPath[1] == ':' {
		drive := strings.ToLower(string(winPath[0]))
		rest := winPath[2:]
		return "/mnt/" + drive + rest
	}
	return winPath
}

// NormalizePath cleans and normalizes a path for the current platform
func NormalizePath(path string) string {
	if platform == nil {
		return filepath.Clean(path)
	}

	// expand ~ on all platforms
	if path == "~" {
		return platform.HomePath
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(platform.HomePath, path[2:])
	}

	// on windows, convert forward slashes
	if platform.OS == PlatformWindows {
		path = strings.ReplaceAll(path, "/", "\\")
	}

	return filepath.Clean(path)
}

// ExecuteInShell runs a command string through the system shell
// useful for shell-specific features like globbing
func ExecuteInShell(command string) error {
	if platform == nil {
		InitPlatform()
	}

	var cmd *exec.Cmd
	switch platform.OS {
	case PlatformWindows:
		cmd = exec.Command("cmd", "/c", command)
	default:
		shell := platform.Shell
		if shell == "" {
			shell = "sh"
		}
		cmd = exec.Command(shell, "-c", command)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Which finds the full path of a command, cross-platform
func Which(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

// OpenFile opens a file with the system default application
func OpenFile(path string) error {
	if platform == nil {
		InitPlatform()
	}

	var cmd *exec.Cmd
	switch platform.OS {
	case PlatformMac:
		cmd = exec.Command("open", path)
	case PlatformWindows:
		cmd = exec.Command("cmd", "/c", "start", path)
	case PlatformWSL:
		cmd = exec.Command("explorer.exe", path)
	default:
		// Linux — try xdg-open
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}

// ClearScreen clears the terminal screen cross-platform
func ClearScreen() {
	if platform == nil {
		InitPlatform()
	}

	switch platform.OS {
	case PlatformWindows:
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Print("\033[H\033[2J")
	}
}

// ---- Built-in: platform command ----

// runPlatformCmd handles the `platform` built-in
// usage:
//
//	platform         → show current platform info
//	platform which <cmd> → find command path
//	platform open <file> → open file with default app
func runPlatformCmd(args []string) {
	if platform == nil {
		InitPlatform()
	}

	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "which":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: platform which <command>")
			return
		}
		if path, ok := Which(args[1]); ok {
			fmt.Printf("  \033[32m✓\033[0m %s → %s\n", args[1], path)
		} else {
			fmt.Printf("  \033[31m✗\033[0m %s not found\n", args[1])
		}

	case "open":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: platform open <file>")
			return
		}
		if err := OpenFile(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "  platform: %v\n", err)
		}

	default:
		// show platform info
		fmt.Println()
		fmt.Printf("  \033[1mPlatform info\033[0m\n\n")
		fmt.Printf("  %-16s \033[1;36m%s\033[0m\n", "OS:", platform.OS)
		fmt.Printf("  %-16s %s\n", "Shell:", platform.Shell)
		fmt.Printf("  %-16s %s\n", "Home:", platform.HomePath)
		fmt.Printf("  %-16s %s\n", "Temp:", platform.TempPath)
		fmt.Printf("  %-16s %s\n", "Path sep:", platform.PathSep)
		if platform.IsWSL {
			fmt.Printf("  %-16s %s\n", "Windows home:", platform.WindowsHome)
		}

		// check for common dev tools
		fmt.Println()
		fmt.Printf("  \033[1mDev tools\033[0m\n\n")
		tools := []string{"git", "node", "npm", "go", "python3", "cargo", "docker", "make"}
		for _, t := range tools {
			if path, ok := Which(t); ok {
				fmt.Printf("  \033[32m✓\033[0m  %-12s %s\n", t, path)
			} else {
				fmt.Printf("  \033[2m✗  %s\033[0m\n", t)
			}
		}
		fmt.Println()
	}
}
