package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExplorerConfig controls display options
type ExplorerConfig struct {
	ShowHidden  bool // show dotfiles
	MaxDepth    int  // max tree depth
	ShowSize    bool // show file sizes
	ShowPreview bool // show file preview for text files
}

var defaultConfig = ExplorerConfig{
	ShowHidden:  false,
	MaxDepth:    3,
	ShowSize:    true,
	ShowPreview: true,
}

// FileNode represents a file or directory in the tree
type FileNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Children []*FileNode
	Err      error
}

// ---- Tree builder ----

// BuildTree builds a file tree from the given root up to maxDepth
func BuildTree(root string, cfg ExplorerConfig) (*FileNode, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	node := &FileNode{
		Name:  filepath.Base(root),
		Path:  root,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}

	if info.IsDir() {
		node.Children = buildChildren(root, cfg, 0)
	}

	return node, nil
}

func buildChildren(dir string, cfg ExplorerConfig, depth int) []*FileNode {
	if depth >= cfg.MaxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var nodes []*FileNode
	for _, entry := range entries {
		// skip hidden files unless configured
		if !cfg.ShowHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		node := &FileNode{
			Name:  entry.Name(),
			Path:  filepath.Join(dir, entry.Name()),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		}

		if entry.IsDir() {
			node.Children = buildChildren(node.Path, cfg, depth+1)
		}

		nodes = append(nodes, node)
	}

	// sort: dirs first, then files, both alphabetically
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})

	return nodes
}

// ---- Renderer ----

// PrintTree renders the file tree with box-drawing characters
func PrintFileTree(node *FileNode, cfg ExplorerConfig) {
	if node == nil {
		return
	}
	fmt.Println()
	// print root
	icon := dirIcon(node)
	fmt.Printf("  %s \033[1;36m%s\033[0m\n", icon, node.Name)
	printFileChildren(node.Children, "  ", cfg)
	fmt.Println()
}

func printFileChildren(nodes []*FileNode, prefix string, cfg ExplorerConfig) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		icon := fileIcon(node)
		name := formatName(node)
		size := ""
		if cfg.ShowSize && !node.IsDir {
			size = fmt.Sprintf(" \033[2m%s\033[0m", formatSize(node.Size))
		}

		// truncate very long names
		display := node.Name
		if len(display) > 40 {
			display = display[:37] + "..."
		}

		fmt.Printf("%s%s%s %s%s%s\n", prefix, connector, icon, name, display, size)

		if node.IsDir && len(node.Children) > 0 {
			printFileChildren(node.Children, childPrefix, cfg)
		} else if node.IsDir && node.Children == nil {
			// max depth reached
			fmt.Printf("%s    \033[2m...\033[0m\n", childPrefix)
		}
	}
}

// formatName adds color based on file type
func formatName(node *FileNode) string {
	if node.IsDir {
		return "\033[1;34m"
	}
	ext := strings.ToLower(filepath.Ext(node.Name))
	switch ext {
	case ".go", ".rs", ".py", ".js", ".ts", ".rb", ".java", ".c", ".cpp", ".h":
		return "\033[0;32m" // green for source files
	case ".json", ".toml", ".yaml", ".yml", ".xml", ".env":
		return "\033[0;33m" // amber for config files
	case ".md", ".txt", ".rst":
		return "\033[0;36m" // cyan for docs
	case ".sh", ".bash", ".zsh", ".fish":
		return "\033[0;35m" // purple for scripts
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return "\033[0;31m" // red for images
	default:
		return "\033[0m" // default
	}
}

// fileIcon returns a text icon for the file type
func fileIcon(node *FileNode) string {
	if node.IsDir {
		return "▸"
	}
	ext := strings.ToLower(filepath.Ext(node.Name))
	switch ext {
	case ".go":
		return "○"
	case ".rs":
		return "○"
	case ".py":
		return "○"
	case ".js", ".ts":
		return "○"
	case ".json", ".toml", ".yaml", ".yml":
		return "◆"
	case ".md", ".txt":
		return "◇"
	case ".sh", ".bash", ".zsh":
		return "▷"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		return "◈"
	case ".env":
		return "◆"
	default:
		return "·"
	}
}

func dirIcon(node *FileNode) string {
	if node.IsDir {
		return "▸"
	}
	return "·"
}

// formatSize returns a human-readable file size
func formatSize(size int64) string {
	switch {
	case size >= 1024*1024*1024:
		return fmt.Sprintf("%.1fG", float64(size)/float64(1024*1024*1024))
	case size >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(size)/float64(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.1fK", float64(size)/float64(1024))
	default:
		return fmt.Sprintf("%dB", size)
	}
}

// ---- File preview ----

// PreviewFile prints the first N lines of a text file
func PreviewFile(path string, lines int) {
	if lines <= 0 {
		lines = 20
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  preview: %v\n", err)
		return
	}
	defer f.Close()

	// check if binary by reading first 512 bytes
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if isBinary(buf[:n]) {
		fmt.Println("  \033[2m[binary file]\033[0m")
		return
	}

	// reopen and read lines
	f.Seek(0, 0)
	content := make([]byte, 8192)
	total, _ := f.Read(content)
	text := string(content[:total])

	allLines := strings.Split(text, "\n")
	if len(allLines) > lines {
		allLines = allLines[:lines]
	}

	fmt.Printf("\n  \033[2m── preview: %s ──\033[0m\n", filepath.Base(path))
	for i, line := range allLines {
		fmt.Printf("  \033[2m%3d\033[0m  %s\n", i+1, line)
	}
	if len(strings.Split(text, "\n")) > lines {
		fmt.Printf("  \033[2m... (%d more lines)\033[0m\n",
			len(strings.Split(text, "\n"))-lines)
	}
	fmt.Println()
}

// isBinary checks if a byte slice looks like binary content
func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// ---- Search ----

// SearchFiles searches for files matching a pattern under root
func SearchFiles(root, pattern string, cfg ExplorerConfig) {
	pattern = strings.ToLower(pattern)
	found := 0

	fmt.Printf("\n  \033[1mSearching for '%s' in %s\033[0m\n\n", pattern, root)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := strings.ToLower(info.Name())

		// skip hidden unless configured
		if !cfg.ShowHidden && strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// skip common noise directories
		if info.IsDir() && isIgnoredDir(info.Name()) {
			return filepath.SkipDir
		}

		if strings.Contains(name, pattern) {
			rel, _ := filepath.Rel(root, path)
			icon := "·"
			color := "\033[0m"
			if info.IsDir() {
				icon = "▸"
				color = "\033[1;34m"
			}
			fmt.Printf("  %s %s%s\033[0m\n", icon, color, rel)
			found++
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "  search: %v\n", err)
	}

	if found == 0 {
		fmt.Println("  No files found.")
	} else {
		fmt.Printf("\n  Found %d result(s)\n", found)
	}
	fmt.Println()
}

// isIgnoredDir returns true for directories that should be skipped
func isIgnoredDir(name string) bool {
	ignored := []string{
		"node_modules", ".git", ".svn", "vendor",
		"target", "dist", "build", "__pycache__",
		".cache", ".gradle",
	}
	for _, ig := range ignored {
		if name == ig {
			return true
		}
	}
	return false
}

// ---- Built-in: ls and explore commands ----

// runExploreCmd handles the `explore` built-in
// usage:
//
//	explore              → tree of current directory
//	explore <path>       → tree of given path
//	explore -a           → include hidden files
//	explore -d <n>       → set max depth
//	explore search <q>   → search for files
//	explore preview <f>  → preview a file
func runExploreCmd(args []string) {
	cfg := defaultConfig
	root := "."
	subcommand := ""

	// parse args
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-a", "--all":
			cfg.ShowHidden = true
		case "-d", "--depth":
			if i+1 < len(args) {
				i++
				depth := 0
				fmt.Sscanf(args[i], "%d", &depth)
				if depth > 0 {
					cfg.MaxDepth = depth
				}
			}
		case "search":
			subcommand = "search"
			if i+1 < len(args) {
				i++
				subcommand = "search:" + args[i]
			}
		case "preview":
			subcommand = "preview"
			if i+1 < len(args) {
				i++
				subcommand = "preview:" + args[i]
			}
		default:
			if !strings.HasPrefix(args[i], "-") {
				root = args[i]
			}
		}
		i++
	}

	// resolve root
	if root == "." {
		var err error
		root, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "explore: cannot get working directory")
			return
		}
	} else {
		root = NormalizePath(root)
	}

	// handle subcommands
	if strings.HasPrefix(subcommand, "search:") {
		pattern := strings.TrimPrefix(subcommand, "search:")
		SearchFiles(root, pattern, cfg)
		return
	}

	if strings.HasPrefix(subcommand, "preview:") {
		path := strings.TrimPrefix(subcommand, "preview:")
		path = NormalizePath(path)
		PreviewFile(path, 30)
		return
	}

	// default: show tree
	node, err := BuildTree(root, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explore: %v\n", err)
		return
	}

	PrintFileTree(node, cfg)

	// show project context
	project := DetectProject(root)
	if project.Type != ProjectUnknown {
		fmt.Printf("  \033[2mProject: %s · %s\033[0m\n\n",
			project.Type, project.Name)
	}
}
