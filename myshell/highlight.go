package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiRed       = "\033[31m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiBlue      = "\033[34m"
	ansiMagenta   = "\033[35m"
	ansiCyan      = "\033[36m"
	ansiBoldRed   = "\033[1;31m"
	ansiBoldGreen = "\033[1;32m"
	ansiBoldCyan  = "\033[1;36m"
	ansiBoldBlue  = "\033[1;34m"
)

// LangSpec defines keywords and comment styles for a language
type LangSpec struct {
	Keywords    []string
	Types       []string
	Builtins    []string
	LineComment string
	BlockOpen   string
	BlockClose  string
}

var langSpecs = map[string]LangSpec{
	"go": {
		Keywords:    []string{"package", "import", "func", "var", "const", "type", "struct", "interface", "map", "chan", "go", "defer", "return", "if", "else", "for", "range", "switch", "case", "default", "break", "continue", "select", "fallthrough", "goto"},
		Types:       []string{"string", "int", "int64", "int32", "float64", "float32", "bool", "byte", "rune", "error", "any"},
		Builtins:    []string{"make", "new", "len", "cap", "append", "copy", "delete", "close", "panic", "recover", "print", "println"},
		LineComment: "//",
		BlockOpen:   "/*",
		BlockClose:  "*/",
	},
	"python": {
		Keywords:    []string{"def", "class", "import", "from", "return", "if", "elif", "else", "for", "while", "in", "not", "and", "or", "is", "lambda", "with", "as", "try", "except", "finally", "raise", "pass", "break", "continue", "yield", "async", "await"},
		Types:       []string{"str", "int", "float", "bool", "list", "dict", "tuple", "set", "None", "True", "False"},
		Builtins:    []string{"print", "len", "range", "type", "isinstance", "enumerate", "zip", "map", "filter", "sorted", "open"},
		LineComment: "#",
	},
	"js": {
		Keywords:    []string{"const", "let", "var", "function", "return", "if", "else", "for", "while", "class", "import", "export", "default", "new", "this", "typeof", "switch", "case", "break", "continue", "try", "catch", "finally", "throw", "async", "await"},
		Types:       []string{"string", "number", "boolean", "undefined", "null", "object"},
		Builtins:    []string{"console", "Math", "JSON", "Array", "Object", "Promise", "setTimeout", "fetch"},
		LineComment: "//",
		BlockOpen:   "/*",
		BlockClose:  "*/",
	},
	"rust": {
		Keywords:    []string{"fn", "let", "mut", "pub", "use", "mod", "struct", "enum", "impl", "trait", "for", "in", "if", "else", "match", "return", "loop", "while", "break", "continue", "async", "await", "move", "type", "where", "unsafe"},
		Types:       []string{"String", "str", "i32", "i64", "u32", "u64", "f32", "f64", "bool", "char", "Vec", "Option", "Result"},
		Builtins:    []string{"Some", "None", "Ok", "Err"},
		LineComment: "//",
		BlockOpen:   "/*",
		BlockClose:  "*/",
	},
}

var extToLang = map[string]string{
	".go":  "go",
	".py":  "python",
	".js":  "js",
	".ts":  "js",
	".jsx": "js",
	".tsx": "js",
	".rs":  "rust",
}

// HighlightCode applies syntax highlighting to source code
func HighlightCode(code, ext string) string {
	lang := extToLang[strings.ToLower(ext)]
	spec, ok := langSpecs[lang]
	if !ok {
		return code
	}

	lines := strings.Split(code, "\n")
	var result strings.Builder
	inBlock := false

	for _, line := range lines {
		highlighted, stillIn := highlightLine(line, spec, inBlock)
		inBlock = stillIn
		result.WriteString(highlighted)
		result.WriteString("\n")
	}

	return result.String()
}

func highlightLine(line string, spec LangSpec, inBlock bool) (string, bool) {
	if inBlock {
		if spec.BlockClose != "" && strings.Contains(line, spec.BlockClose) {
			idx := strings.Index(line, spec.BlockClose) + len(spec.BlockClose)
			return ansiDim + line[:idx] + ansiReset + line[idx:], false
		}
		return ansiDim + line + ansiReset, true
	}

	if spec.BlockOpen != "" && strings.Contains(line, spec.BlockOpen) {
		idx := strings.Index(line, spec.BlockOpen)
		before := highlightTokens(line[:idx], spec)
		after := line[idx:]
		if spec.BlockClose != "" && strings.Contains(after, spec.BlockClose) {
			end := strings.Index(after, spec.BlockClose) + len(spec.BlockClose)
			return before + ansiDim + after[:end] + ansiReset + after[end:], false
		}
		return before + ansiDim + after + ansiReset, true
	}

	if spec.LineComment != "" {
		if idx := strings.Index(line, spec.LineComment); idx >= 0 {
			return highlightTokens(line[:idx], spec) + ansiDim + line[idx:] + ansiReset, false
		}
	}

	return highlightTokens(line, spec), false
}

func highlightTokens(line string, spec LangSpec) string {
	if line == "" {
		return line
	}

	var result strings.Builder
	runes := []rune(line)
	i := 0

	for i < len(runes) {
		ch := runes[i]

		// string literals
		if ch == '"' || ch == '\'' || ch == '`' {
			quote := ch
			j := i + 1
			for j < len(runes) && runes[j] != quote {
				if runes[j] == '\\' {
					j++
				}
				j++
			}
			if j < len(runes) {
				j++
			}
			result.WriteString(ansiYellow + string(runes[i:j]) + ansiReset)
			i = j
			continue
		}

		// numbers
		if ch >= '0' && ch <= '9' {
			j := i
			for j < len(runes) && ((runes[j] >= '0' && runes[j] <= '9') || runes[j] == '.') {
				j++
			}
			result.WriteString(ansiMagenta + string(runes[i:j]) + ansiReset)
			i = j
			continue
		}

		// identifiers and keywords
		if isLetter(ch) {
			j := i
			for j < len(runes) && (isLetter(runes[j]) || runes[j] == '_' || (runes[j] >= '0' && runes[j] <= '9') || runes[j] == '!') {
				j++
			}
			word := string(runes[i:j])

			switch {
			case sliceContains(spec.Keywords, word):
				result.WriteString(ansiBoldBlue + word + ansiReset)
			case sliceContains(spec.Types, word):
				result.WriteString(ansiCyan + word + ansiReset)
			case sliceContains(spec.Builtins, word):
				result.WriteString(ansiGreen + word + ansiReset)
			default:
				result.WriteString(word)
			}
			i = j
			continue
		}

		// operators — dim them slightly
		if strings.ContainsRune("+-*/%=<>!&|^~{}[]();,.", ch) {
			result.WriteString(ansiDim + string(ch) + ansiReset)
			i++
			continue
		}

		result.WriteRune(ch)
		i++
	}

	return result.String()
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func sliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ---- Color helpers used across the shell ----

func ColorSuccess(s string) string { return ansiBoldGreen + s + ansiReset }
func ColorError(s string) string   { return ansiBoldRed + s + ansiReset }
func ColorWarn(s string) string    { return ansiYellow + s + ansiReset }
func ColorInfo(s string) string    { return ansiBoldCyan + s + ansiReset }
func ColorDim(s string) string     { return ansiDim + s + ansiReset }
func ColorBold(s string) string    { return ansiBold + s + ansiReset }

// HighlightDiff colorizes unified diff output
func HighlightDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result strings.Builder
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			result.WriteString(ColorBold(line))
		case strings.HasPrefix(line, "+"):
			result.WriteString(ansiGreen + line + ansiReset)
		case strings.HasPrefix(line, "-"):
			result.WriteString(ansiRed + line + ansiReset)
		case strings.HasPrefix(line, "@@"):
			result.WriteString(ansiCyan + line + ansiReset)
		default:
			result.WriteString(line)
		}
		result.WriteString("\n")
	}
	return result.String()
}

// runHighlightCmd handles the `highlight` built-in
// usage:
//
//	highlight <file>        → print file with syntax highlighting
//	highlight <file> <n>    → print first n lines highlighted
func runHighlightCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "  usage: highlight <file> [lines]")
		return
	}

	path := NormalizePath(args[0])
	maxLines := 0
	if len(args) >= 2 {
		fmt.Sscanf(args[1], "%d", &maxLines)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  highlight: %v\n", err)
		return
	}

	content := string(data)
	if maxLines > 0 {
		lines := strings.Split(content, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		content = strings.Join(lines, "\n")
	}

	ext := ""
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		ext = path[idx:]
	}

	fmt.Println()
	fmt.Printf("  %s\n\n", ColorDim("── "+path+" ──"))

	highlighted := HighlightCode(content, ext)
	lineNum := 1
	for _, line := range strings.Split(highlighted, "\n") {
		fmt.Printf("  %s  %s\n", ColorDim(fmt.Sprintf("%3d", lineNum)), line)
		lineNum++
	}
	fmt.Println()
}
