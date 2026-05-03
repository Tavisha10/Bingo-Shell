package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// StatusBar holds the current state shown in the prompt area
type StatusBar struct {
	ShowGit     bool
	ShowTime    bool
	ShowProject bool
	ShowExit    bool
	lastExit    int
}

var statusBar = &StatusBar{
	ShowGit:     true,
	ShowTime:    true,
	ShowProject: true,
	ShowExit:    true,
}

// ---- Git info ----

// GitInfo holds current git repository state
type GitInfo struct {
	Branch   string
	Modified int // number of modified files
	Staged   int // number of staged files
	Ahead    int // commits ahead of remote
	Behind   int // commits behind remote
	IsRepo   bool
}

// GetGitInfo reads git status for the current directory
func GetGitInfo() GitInfo {
	info := GitInfo{}

	// check if inside a git repo
	out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return info
	}
	info.IsRepo = true

	// get branch name
	if branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		info.Branch = strings.TrimSpace(string(branch))
	}

	// get status counts
	if status, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		for _, line := range strings.Split(string(status), "\n") {
			if len(line) < 2 {
				continue
			}
			x := line[0] // index (staged)
			y := line[1] // worktree (unstaged)
			if x != ' ' && x != '?' {
				info.Staged++
			}
			if y != ' ' && y != '?' {
				info.Modified++
			}
		}
	}

	// check ahead/behind
	if ab, err := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{u}").Output(); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(ab)), "%d\t%d", &info.Ahead, &info.Behind)
	}

	return info
}

// FormatGit returns a colored git status string for the prompt
func FormatGit(info GitInfo) string {
	if !info.IsRepo {
		return ""
	}

	branch := info.Branch
	if len(branch) > 20 {
		branch = branch[:17] + "..."
	}

	// color branch based on status
	branchColor := ansiGreen
	if info.Modified > 0 {
		branchColor = ansiYellow
	}
	if info.Staged > 0 {
		branchColor = ansiCyan
	}

	result := fmt.Sprintf("%s%s%s", branchColor, branch, ansiReset)

	if info.Modified > 0 {
		result += fmt.Sprintf(" %s~%d%s", ansiYellow, info.Modified, ansiReset)
	}
	if info.Staged > 0 {
		result += fmt.Sprintf(" %s+%d%s", ansiCyan, info.Staged, ansiReset)
	}
	if info.Ahead > 0 {
		result += fmt.Sprintf(" %s↑%d%s", ansiGreen, info.Ahead, ansiReset)
	}
	if info.Behind > 0 {
		result += fmt.Sprintf(" %s↓%d%s", ansiRed, info.Behind, ansiReset)
	}

	return "(" + result + ")"
}

// ---- Enhanced prompt ----

// BuildPrompt constructs the full multi-segment prompt string
func BuildPrompt() string {
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	display := strings.Replace(cwd, home, "~", 1)

	// shorten deep paths: keep last 2 segments
	parts := strings.Split(display, "/")
	if len(parts) > 4 {
		display = "…/" + strings.Join(parts[len(parts)-2:], "/")
	}

	var segments []string

	// directory segment
	segments = append(segments, fmt.Sprintf("\033[1;36m%s\033[0m", display))

	// project type
	if statusBar.ShowProject {
		project := DetectProject(cwd)
		if project.Type != ProjectUnknown {
			segments = append(segments, fmt.Sprintf("\033[0;33m%s\033[0m", project.Type.Emoji()))
		}
	}

	// git segment
	if statusBar.ShowGit {
		git := GetGitInfo()
		if git.IsRepo {
			segments = append(segments, FormatGit(git))
		}
	}

	// time segment
	if statusBar.ShowTime {
		t := time.Now().Format("15:04")
		segments = append(segments, fmt.Sprintf("%s%s%s", ansiDim, t, ansiReset))
	}

	// exit code of last command
	if statusBar.ShowExit && statusBar.lastExit != 0 {
		segments = append(segments, fmt.Sprintf("%s[%d]%s", ansiBoldRed, statusBar.lastExit, ansiReset))
	}

	prompt := strings.Join(segments, " ")
	prompt += fmt.Sprintf(" \033[1;32m❯\033[0m ")

	return prompt
}

// SetLastExit records the exit code of the last command
func SetLastExit(code int) {
	statusBar.lastExit = code
}

// ---- Built-in: status command ----

// runStatusCmd handles the `status` built-in
// usage:
//
//	status              → show full status panel
//	status git          → show git details
//	status toggle <key> → toggle a status bar segment
func runStatusCmd(args []string) {
	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "git":
		printGitStatus()

	case "toggle":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "  usage: status toggle [git|time|project|exit]")
			return
		}
		switch args[1] {
		case "git":
			statusBar.ShowGit = !statusBar.ShowGit
			fmt.Printf("  git status: %s\n", onOff(statusBar.ShowGit))
		case "time":
			statusBar.ShowTime = !statusBar.ShowTime
			fmt.Printf("  time: %s\n", onOff(statusBar.ShowTime))
		case "project":
			statusBar.ShowProject = !statusBar.ShowProject
			fmt.Printf("  project tag: %s\n", onOff(statusBar.ShowProject))
		case "exit":
			statusBar.ShowExit = !statusBar.ShowExit
			fmt.Printf("  exit code: %s\n", onOff(statusBar.ShowExit))
		default:
			fmt.Fprintf(os.Stderr, "  unknown toggle: %s\n", args[1])
		}

	default:
		printFullStatus()
	}
}

func printFullStatus() {
	fmt.Println()
	PrintSection("shell status")

	cwd, _ := os.Getwd()
	project := DetectProject(cwd)
	git := GetGitInfo()

	pairs := [][2]string{
		{"directory", cwd},
		{"project", fmt.Sprintf("%s  %s", project.Type.Emoji(), string(project.Type))},
		{"time", time.Now().Format("Mon Jan 2 15:04:05")},
	}

	if platform != nil {
		pairs = append(pairs, [2]string{"platform", string(platform.OS)})
		pairs = append(pairs, [2]string{"shell", platform.Shell})
	}

	PrintKV(pairs)

	if git.IsRepo {
		printGitStatus()
	}

	// status bar toggles
	PrintSection("status bar")
	PrintKV([][2]string{
		{"git", onOff(statusBar.ShowGit)},
		{"time", onOff(statusBar.ShowTime)},
		{"project", onOff(statusBar.ShowProject)},
		{"exit code", onOff(statusBar.ShowExit)},
	})
	fmt.Println("  toggle with: status toggle <git|time|project|exit>")
	fmt.Println()
}

func printGitStatus() {
	git := GetGitInfo()
	if !git.IsRepo {
		PrintWarn("not inside a git repository")
		return
	}

	fmt.Println()
	PrintSection("git status")
	PrintKV([][2]string{
		{"branch", git.Branch},
		{"modified", fmt.Sprintf("%d file(s)", git.Modified)},
		{"staged", fmt.Sprintf("%d file(s)", git.Staged)},
		{"ahead", fmt.Sprintf("%d commit(s)", git.Ahead)},
		{"behind", fmt.Sprintf("%d commit(s)", git.Behind)},
	})
}

func onOff(b bool) string {
	if b {
		return ColorSuccess("on")
	}
	return ColorDim("off")
}
