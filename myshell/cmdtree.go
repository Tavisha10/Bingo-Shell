package main

import (
	"fmt"
	"strings"
)

// NodeType represents what kind of node this is in the command tree
type NodeType string

const (
	NodePipeline NodeType = "Pipeline"
	NodeCommand  NodeType = "Command"
	NodeAnd      NodeType = "AND (&&)"
	NodeOr       NodeType = "OR (||)"
	NodeSequence NodeType = "Sequence (;)"
)

// CmdNode is a node in the command tree
type CmdNode struct {
	Type     NodeType
	Parts    []string   // tokens for a command node
	Redirect redirects  // any redirects on this node
	Children []*CmdNode // for pipeline/and/or/sequence nodes
}

// ---- Parser ----

// ParseCommandTree takes a raw input line and builds a tree
func ParseCommandTree(line string) *CmdNode {
	tokens := tokenize(line)
	tokens = expandEnv(tokens)
	return parseSequence(tokens)
}

// parseSequence splits on ; into a sequence node
func parseSequence(tokens []string) *CmdNode {
	segments := splitOn(tokens, ";")
	if len(segments) == 1 {
		return parseAndOr(segments[0])
	}
	node := &CmdNode{Type: NodeSequence}
	for _, seg := range segments {
		node.Children = append(node.Children, parseAndOr(seg))
	}
	return node
}

// parseAndOr splits on && and ||
func parseAndOr(tokens []string) *CmdNode {
	// find && or ||
	for i, t := range tokens {
		if t == "&&" {
			node := &CmdNode{Type: NodeAnd}
			node.Children = append(node.Children, parsePipeline(tokens[:i]))
			node.Children = append(node.Children, parseAndOr(tokens[i+1:]))
			return node
		}
		if t == "||" {
			node := &CmdNode{Type: NodeOr}
			node.Children = append(node.Children, parsePipeline(tokens[:i]))
			node.Children = append(node.Children, parseAndOr(tokens[i+1:]))
			return node
		}
	}
	return parsePipeline(tokens)
}

// parsePipeline splits on | into a pipeline node
func parsePipeline(tokens []string) *CmdNode {
	segments := splitOn(tokens, "|")
	if len(segments) == 1 {
		return parseCommand(segments[0])
	}
	node := &CmdNode{Type: NodePipeline}
	for _, seg := range segments {
		node.Children = append(node.Children, parseCommand(seg))
	}
	return node
}

// parseCommand builds a leaf command node
func parseCommand(tokens []string) *CmdNode {
	clean, r := parseRedirects(tokens)
	return &CmdNode{
		Type:     NodeCommand,
		Parts:    clean,
		Redirect: r,
	}
}

// splitOn splits a token slice on a delimiter token
func splitOn(tokens []string, delim string) [][]string {
	var segments [][]string
	var current []string
	for _, t := range tokens {
		if t == delim {
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
	if len(segments) == 0 {
		return [][]string{{}}
	}
	return segments
}

// ---- Renderer ----

// PrintTree prints the command tree with box-drawing characters
func PrintTree(node *CmdNode) {
	if node == nil {
		return
	}
	fmt.Println()
	printNode(node, "", true)
	fmt.Println()
}

func printNode(node *CmdNode, prefix string, isLast bool) {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	switch node.Type {
	case NodeCommand:
		label := formatCommand(node)
		fmt.Printf("%s%s\033[1;36m%s\033[0m\n", prefix, connector, label)

	case NodePipeline:
		fmt.Printf("%s%s\033[1;33m%s\033[0m\n", prefix, connector, string(node.Type))
		for i, child := range node.Children {
			printNode(child, childPrefix, i == len(node.Children)-1)
		}

	case NodeAnd, NodeOr, NodeSequence:
		fmt.Printf("%s%s\033[1;35m%s\033[0m\n", prefix, connector, string(node.Type))
		for i, child := range node.Children {
			printNode(child, childPrefix, i == len(node.Children)-1)
		}
	}
}

// formatCommand builds a display string for a command node
func formatCommand(node *CmdNode) string {
	if len(node.Parts) == 0 {
		return "(empty)"
	}

	// command name in bold
	parts := make([]string, len(node.Parts))
	copy(parts, node.Parts)

	// color the command name differently from args
	cmd := "\033[1m" + parts[0] + "\033[0m\033[1;36m"
	args := ""
	if len(parts) > 1 {
		args = " " + strings.Join(parts[1:], " ")
	}

	result := cmd + args

	// append redirect info
	if node.Redirect.stdout != "" {
		result += fmt.Sprintf("  \033[33m→ %s\033[0m", node.Redirect.stdout)
	}
	if node.Redirect.stdoutApp != "" {
		result += fmt.Sprintf("  \033[33m»  %s\033[0m", node.Redirect.stdoutApp)
	}
	if node.Redirect.stdin != "" {
		result += fmt.Sprintf("  \033[33m← %s\033[0m", node.Redirect.stdin)
	}

	return result
}

// ---- Built-in: tree command ----

// runTreeCmd handles the `tree` built-in
// usage:
//
//	tree <command>   → parse and display the tree of a command
//	tree             → show usage
func runTreeCmd(args []string) {
	if len(args) == 0 {
		fmt.Println("  usage: tree <command>")
		fmt.Println("  example: tree ls | grep .go | wc -l")
		fmt.Println("  example: tree git add . && git commit -m 'msg'")
		return
	}

	line := strings.Join(args, " ")
	node := ParseCommandTree(line)

	fmt.Printf("  \033[1mCommand tree for:\033[0m %s", line)
	printTreeFromRoot(node, "  ")
}

// printTreeFromRoot prints the full tree starting from root (no connector prefix)
func printTreeFromRoot(node *CmdNode, indent string) {
	if node == nil {
		return
	}
	fmt.Println()

	switch node.Type {
	case NodeCommand:
		fmt.Printf("%s\033[1;36m%s\033[0m\n", indent, formatCommand(node))

	case NodePipeline, NodeAnd, NodeOr, NodeSequence:
		color := "\033[1;33m"
		if node.Type == NodeAnd || node.Type == NodeOr {
			color = "\033[1;35m"
		}
		fmt.Printf("%s%s%s\033[0m\n", indent, color, string(node.Type))
		for i, child := range node.Children {
			printNode(child, indent, i == len(node.Children)-1)
		}
	}
	fmt.Println()
}
