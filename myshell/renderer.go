package main

import (
	"fmt"
	"os"
	"strings"
)

// ---- Table renderer ----

// Table renders data as an aligned ASCII table
type Table struct {
	Headers []string
	Rows    [][]string
}

// NewTable creates a new table with the given headers
func NewTable(headers ...string) *Table {
	return &Table{Headers: headers}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cols ...string) {
	t.Rows = append(t.Rows, cols)
}

// Print renders the table to stdout
func (t *Table) Print() {
	if len(t.Headers) == 0 {
		return
	}

	// compute column widths
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, row := range t.Rows {
		for i, col := range row {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// top border
	fmt.Print("  ")
	printTableBorder(widths, "┌", "┬", "┐")

	// headers
	fmt.Print("  │ ")
	for i, h := range t.Headers {
		fmt.Printf("%s%s%s", ansiBold, padRight(h, widths[i]), ansiReset)
		if i < len(t.Headers)-1 {
			fmt.Print(" │ ")
		}
	}
	fmt.Println(" │")

	// header separator
	fmt.Print("  ")
	printTableBorder(widths, "├", "┼", "┤")

	// rows
	for _, row := range t.Rows {
		fmt.Print("  │ ")
		for i := 0; i < len(t.Headers); i++ {
			col := ""
			if i < len(row) {
				col = row[i]
			}
			fmt.Printf("%s", padRight(col, widths[i]))
			if i < len(t.Headers)-1 {
				fmt.Print(" │ ")
			}
		}
		fmt.Println(" │")
	}

	// bottom border
	fmt.Print("  ")
	printTableBorder(widths, "└", "┴", "┘")
	fmt.Println()
}

func printTableBorder(widths []int, left, mid, right string) {
	fmt.Print(left)
	for i, w := range widths {
		fmt.Print(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			fmt.Print(mid)
		}
	}
	fmt.Println(right)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// ---- Box renderer ----

// PrintBox draws a labeled box around content
func PrintBox(title, content string) {
	lines := strings.Split(content, "\n")
	maxWidth := len(title) + 4
	for _, l := range lines {
		if len(l)+4 > maxWidth {
			maxWidth = len(l) + 4
		}
	}

	top := "  ┌─ " + ansiBold + title + ansiReset + " " + strings.Repeat("─", maxWidth-len(title)-4) + "┐"
	fmt.Println(top)
	for _, l := range lines {
		fmt.Printf("  │ %-*s │\n", maxWidth-4, l)
	}
	fmt.Println("  └" + strings.Repeat("─", maxWidth-2) + "┘")
	fmt.Println()
}

// ---- Section header ----

// PrintSection prints a styled section header
func PrintSection(title string) {
	fmt.Printf("\n  %s%s%s\n", ansiBold, title, ansiReset)
	fmt.Printf("  %s\n\n", strings.Repeat("─", len(title)))
}

// ---- Key-value renderer ----

// PrintKV prints a list of key-value pairs aligned
func PrintKV(pairs [][2]string) {
	maxKey := 0
	for _, p := range pairs {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	for _, p := range pairs {
		fmt.Printf("  %s%s%s%s  %s\n",
			ansiDim,
			padRight(p[0], maxKey),
			ansiReset,
			ansiDim+"│"+ansiReset,
			p[1],
		)
	}
	fmt.Println()
}

// ---- Progress bar ----

// PrintProgress renders a simple progress bar
func PrintProgress(label string, current, total int) {
	if total == 0 {
		return
	}
	pct := float64(current) / float64(total)
	barWidth := 30
	filled := int(pct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("  %s  [%s%s%s]  %d%%\n",
		padRight(label, 16),
		ansiGreen, bar, ansiReset,
		int(pct*100),
	)
}

// ---- Error / success / warning formatters ----

func PrintSuccess(msg string) {
	fmt.Printf("  %s✓%s  %s\n", ansiBoldGreen, ansiReset, msg)
}

func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "  %s✗%s  %s\n", ansiBoldRed, ansiReset, msg)
}

func PrintWarn(msg string) {
	fmt.Printf("  %s!%s  %s\n", ansiYellow, ansiReset, msg)
}

func PrintInfo(msg string) {
	fmt.Printf("  %si%s  %s\n", ansiBoldCyan, ansiReset, msg)
}

// ---- Built-in: render command (demo / test) ----

// runRenderCmd handles the `render` built-in — demo of renderer features
func runRenderCmd(args []string) {
	if len(args) == 0 {
		printRenderHelp()
		return
	}

	switch args[0] {
	case "table":
		demoTable()
	case "box":
		if len(args) >= 3 {
			PrintBox(args[1], strings.Join(args[2:], " "))
		} else {
			demoBox()
		}
	case "progress":
		demoProgress()
	case "diff":
		if len(args) >= 2 {
			data, err := os.ReadFile(NormalizePath(args[1]))
			if err != nil {
				PrintError(err.Error())
				return
			}
			fmt.Print(HighlightDiff(string(data)))
		} else {
			PrintWarn("usage: render diff <file.patch>")
		}
	case "highlight":
		if len(args) >= 2 {
			runHighlightCmd(args[1:])
		}
	default:
		printRenderHelp()
	}
}

func printRenderHelp() {
	PrintSection("render — output renderer")
	PrintKV([][2]string{
		{"render table", "demo aligned table"},
		{"render box <title> <text>", "draw a labeled box"},
		{"render progress", "demo progress bars"},
		{"render diff <file>", "colorize a diff file"},
		{"render highlight <file>", "syntax highlight a file"},
	})
}

func demoTable() {
	t := NewTable("command", "description", "status")
	t.AddRow("explore", "file tree browser", ColorSuccess("ready"))
	t.AddRow("history", "SQLite history search", ColorSuccess("ready"))
	t.AddRow("tools", "project tool runner", ColorSuccess("ready"))
	t.AddRow("platform", "OS detection", ColorSuccess("ready"))
	t.AddRow("highlight", "syntax highlighter", ColorSuccess("ready"))
	t.Print()
}

func demoBox() {
	PrintBox("myshell", "version 0.5\nphase 5 — renderer\nall systems go")
}

func demoProgress() {
	fmt.Println()
	PrintProgress("downloading", 75, 100)
	PrintProgress("installing", 45, 100)
	PrintProgress("complete", 100, 100)
	fmt.Println()
}
