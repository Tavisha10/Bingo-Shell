package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session holds everything needed to restore a shell session
type Session struct {
	WorkingDir  string            `json:"working_dir"`
	EnvVars     map[string]string `json:"env_vars"`
	LastProject string            `json:"last_project"`
	SavedAt     time.Time         `json:"saved_at"`
	Version     string            `json:"version"`
}

// sessionPath returns the path to the session file
func sessionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "myshell", "session.json")
}

// SaveSession writes the current session state to disk
func SaveSession() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// collect managed env vars
	envSnapshot := make(map[string]string)
	for k, v := range envManager.sessionVars {
		envSnapshot[k] = v
	}
	for k, v := range envManager.projectVars {
		envSnapshot[k] = v
	}

	project := DetectProject(cwd)

	session := Session{
		WorkingDir:  cwd,
		EnvVars:     envSnapshot,
		LastProject: string(project.Type),
		SavedAt:     time.Now(),
		Version:     "0.6",
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	// ensure dir exists
	if err := os.MkdirAll(filepath.Dir(sessionPath()), 0755); err != nil {
		return err
	}

	return os.WriteFile(sessionPath(), data, 0644)
}

// LoadSession reads and restores a previous session
func LoadSession() (*Session, error) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// RestoreSession applies a saved session to the current shell
func RestoreSession(session *Session) {
	if session == nil {
		return
	}

	// restore working directory
	if session.WorkingDir != "" {
		if err := os.Chdir(session.WorkingDir); err == nil {
			// auto-load .env if present
			envManager.AutoLoad(session.WorkingDir)
		}
	}

	// restore env vars
	for k, v := range session.EnvVars {
		os.Setenv(k, v)
		envManager.sessionVars[k] = v
	}
}

// InitSession loads session on startup, prints restore message if found
func InitSession() {
	session, err := LoadSession()
	if err != nil {
		// no session — fresh start, that's fine
		return
	}

	// don't restore very old sessions (older than 7 days)
	if time.Since(session.SavedAt) > 7*24*time.Hour {
		return
	}

	RestoreSession(session)

	home, _ := os.UserHomeDir()
	dir := session.WorkingDir
	for i, h := range []string{home} {
		_ = i
		if len(dir) >= len(h) && dir[:len(h)] == h {
			dir = "~" + dir[len(h):]
			break
		}
	}

	fmt.Printf("  %s Restored session from %s (%s ago)\n\n",
		ColorInfo("↩"),
		dir,
		formatDuration(time.Since(session.SavedAt)),
	)
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// ---- Built-in: session command ----

// runSessionCmd handles the `session` built-in
// usage:
//
//	session              → show current session info
//	session save         → save session now
//	session restore      → restore last session
//	session clear        → delete saved session
func runSessionCmd(args []string) {
	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "save":
		if err := SaveSession(); err != nil {
			PrintError("session save failed: " + err.Error())
			return
		}
		PrintSuccess("session saved to " + sessionPath())

	case "restore":
		session, err := LoadSession()
		if err != nil {
			PrintError("no saved session found")
			return
		}
		RestoreSession(session)
		PrintSuccess(fmt.Sprintf("session restored from %s ago",
			formatDuration(time.Since(session.SavedAt))))

	case "clear":
		if err := os.Remove(sessionPath()); err != nil {
			PrintError("no session to clear")
			return
		}
		PrintSuccess("session cleared")

	default:
		printSessionInfo()
	}
}

func printSessionInfo() {
	fmt.Println()
	PrintSection("session")

	cwd, _ := os.Getwd()
	project := DetectProject(cwd)

	PrintKV([][2]string{
		{"working dir", cwd},
		{"project", string(project.Type)},
		{"session file", sessionPath()},
	})

	// show saved session info if exists
	session, err := LoadSession()
	if err == nil {
		fmt.Println()
		PrintSection("last saved session")
		PrintKV([][2]string{
			{"saved at", session.SavedAt.Format("Mon Jan 2 15:04:05")},
			{"age", formatDuration(time.Since(session.SavedAt))},
			{"directory", session.WorkingDir},
			{"project", session.LastProject},
			{"env vars", fmt.Sprintf("%d", len(session.EnvVars))},
		})
	}

	fmt.Printf("  save with: %s\n\n", ColorInfo("session save"))
}
