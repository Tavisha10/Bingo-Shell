package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// HistoryEntry represents a single command in history
type HistoryEntry struct {
	ID        int
	Command   string
	Dir       string
	Project   string
	ExitCode  int
	Timestamp time.Time
}

// HistoryDB wraps the SQLite database
type HistoryDB struct {
	db *sql.DB
}

// historyDB is the global instance
var historyDB *HistoryDB

// InitHistory opens (or creates) the history database
func InitHistory() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(home, ".myshell_history.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open history db: %w", err)
	}

	historyDB = &HistoryDB{db: db}
	return historyDB.migrate()
}

// migrate creates the schema if it doesn't exist
func (h *HistoryDB) migrate() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			command   TEXT NOT NULL,
			dir       TEXT NOT NULL DEFAULT '',
			project   TEXT NOT NULL DEFAULT '',
			exit_code INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_history_command   ON history(command);
		CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history(timestamp);
	`)
	return err
}

// Add saves a command to the history database
func (h *HistoryDB) Add(command, dir, project string, exitCode int) error {
	_, err := h.db.Exec(
		`INSERT INTO history (command, dir, project, exit_code, timestamp) VALUES (?, ?, ?, ?, ?)`,
		command, dir, project, exitCode, time.Now().Unix(),
	)
	return err
}

// Search does a fuzzy search over history, returning the most recent matches first
func (h *HistoryDB) Search(query string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	// Simple contains search — fast enough for tens of thousands of entries
	rows, err := h.db.Query(`
		SELECT id, command, dir, project, exit_code, timestamp
		FROM history
		WHERE command LIKE ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Recent returns the N most recent history entries
func (h *HistoryDB) Recent(limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := h.db.Query(`
		SELECT id, command, dir, project, exit_code, timestamp
		FROM history
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// InDir returns history entries run from a specific directory
func (h *HistoryDB) InDir(dir string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := h.db.Query(`
		SELECT id, command, dir, project, exit_code, timestamp
		FROM history
		WHERE dir = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, dir, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Stats returns a summary of history usage
func (h *HistoryDB) Stats() {
	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&total)

	fmt.Printf("  Total commands: %d\n", total)

	rows, err := h.db.Query(`
		SELECT command, COUNT(*) as cnt
		FROM history
		GROUP BY command
		ORDER BY cnt DESC
		LIMIT 5
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	fmt.Println("  Most used commands:")
	for rows.Next() {
		var cmd string
		var cnt int
		rows.Scan(&cmd, &cnt)
		// truncate long commands for display
		display := cmd
		if len(display) > 40 {
			display = display[:37] + "..."
		}
		fmt.Printf("    %3dx  %s\n", cnt, display)
	}
}

// Close closes the database
func (h *HistoryDB) Close() {
	if h.db != nil {
		h.db.Close()
	}
}

// scanEntries reads rows into HistoryEntry slice
func scanEntries(rows *sql.Rows) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var ts int64
		err := rows.Scan(&e.ID, &e.Command, &e.Dir, &e.Project, &e.ExitCode, &ts)
		if err != nil {
			continue
		}
		e.Timestamp = time.Unix(ts, 0)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ---- Built-in: history command ----

// runHistoryCmd handles the `history` built-in with subcommands
// usage:
//
//	history           → show 50 most recent
//	history search <q>→ search history
//	history here      → commands run in current dir
//	history stats     → usage statistics
func runHistoryCmd(args []string) {
	if historyDB == nil {
		fmt.Fprintln(os.Stderr, "history: database not initialized")
		return
	}

	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "search":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: history search <query>")
			return
		}
		query := strings.Join(args[1:], " ")
		entries, err := historyDB.Search(query, 20)
		if err != nil {
			fmt.Fprintln(os.Stderr, "history search error:", err)
			return
		}
		printEntries(entries)

	case "here":
		cwd, _ := os.Getwd()
		entries, err := historyDB.InDir(cwd, 20)
		if err != nil {
			fmt.Fprintln(os.Stderr, "history here error:", err)
			return
		}
		fmt.Printf("  Commands run in %s:\n", cwd)
		printEntries(entries)

	case "stats":
		historyDB.Stats()

	default:
		entries, err := historyDB.Recent(50)
		if err != nil {
			fmt.Fprintln(os.Stderr, "history error:", err)
			return
		}
		printEntries(entries)
	}
}

func printEntries(entries []HistoryEntry) {
	if len(entries) == 0 {
		fmt.Println("  No history found.")
		return
	}
	for _, e := range entries {
		ts := e.Timestamp.Format("01/02 15:04")
		dir := e.Dir
		home, _ := os.UserHomeDir()
		dir = strings.Replace(dir, home, "~", 1)
		if len(dir) > 25 {
			dir = "..." + dir[len(dir)-22:]
		}
		fmt.Printf("  %s  %-26s  %s\n", ts, dir, e.Command)
	}
}
