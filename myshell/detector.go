package main

import (
	"os"
	"path/filepath"
)

// ProjectType represents the detected project type
type ProjectType string

const (
	ProjectNode    ProjectType = "node"
	ProjectRust    ProjectType = "rust"
	ProjectGo      ProjectType = "go"
	ProjectPython  ProjectType = "python"
	ProjectDeno    ProjectType = "deno"
	ProjectJava    ProjectType = "java"
	ProjectRuby    ProjectType = "ruby"
	ProjectPHP     ProjectType = "php"
	ProjectUnknown ProjectType = "unknown"
)

// ProjectInfo holds everything detected about the current project
type ProjectInfo struct {
	Type    ProjectType
	Name    string   // project name if detectable
	Root    string   // root directory where marker was found
	Markers []string // which marker files were found
}

// markerMap maps filenames to project types
// order matters — first match wins if multiple exist
var markerMap = []struct {
	file    string
	project ProjectType
}{
	{"package.json", ProjectNode},
	{"deno.json", ProjectDeno},
	{"deno.jsonc", ProjectDeno},
	{"Cargo.toml", ProjectRust},
	{"go.mod", ProjectGo},
	{"pyproject.toml", ProjectPython},
	{"requirements.txt", ProjectPython},
	{"Pipfile", ProjectPython},
	{"setup.py", ProjectPython},
	{"pom.xml", ProjectJava},
	{"build.gradle", ProjectJava},
	{"Gemfile", ProjectRuby},
	{"composer.json", ProjectPHP},
}

// DetectProject walks up from dir looking for project marker files.
// It stops at the filesystem root or home directory.
func DetectProject(dir string) ProjectInfo {
	home, _ := os.UserHomeDir()
	current := dir

	for {
		info := scanDir(current)
		if info.Type != ProjectUnknown {
			return info
		}

		// Stop if we've hit home or root
		if current == home || current == filepath.Dir(current) {
			break
		}
		current = filepath.Dir(current)
	}

	return ProjectInfo{Type: ProjectUnknown, Root: dir}
}

// scanDir checks a single directory for marker files
func scanDir(dir string) ProjectInfo {
	var found []string
	detected := ProjectUnknown

	for _, m := range markerMap {
		path := filepath.Join(dir, m.file)
		if _, err := os.Stat(path); err == nil {
			found = append(found, m.file)
			if detected == ProjectUnknown {
				detected = m.project
			}
		}
	}

	if detected == ProjectUnknown {
		return ProjectInfo{Type: ProjectUnknown}
	}

	return ProjectInfo{
		Type:    detected,
		Root:    dir,
		Markers: found,
		Name:    projectName(detected, dir),
	}
}

// projectName tries to extract a human-readable project name
func projectName(t ProjectType, dir string) string {
	// default: use the directory name
	name := filepath.Base(dir)

	switch t {
	case ProjectGo:
		// read module name from go.mod first line
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			for _, line := range splitLines(string(data)) {
				if len(line) > 7 && line[:7] == "module " {
					name = line[7:]
					break
				}
			}
		}
	case ProjectNode:
		// could parse package.json for "name" field — keep it simple for now
		name = filepath.Base(dir)
	}

	return name
}

// splitLines splits a string into lines without importing strings
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// Emoji returns a small icon for the project type — used in the prompt
func (p ProjectType) Emoji() string {
	switch p {
	case ProjectNode:
		return "[node]"
	case ProjectDeno:
		return "[deno]"
	case ProjectRust:
		return "[rust]"
	case ProjectGo:
		return "[go]"
	case ProjectPython:
		return "[py]"
	case ProjectJava:
		return "[java]"
	case ProjectRuby:
		return "[rb]"
	case ProjectPHP:
		return "[php]"
	default:
		return ""
	}
}

// SuggestedTools returns tool commands relevant to the project type
func (p ProjectType) SuggestedTools() []string {
	switch p {
	case ProjectNode:
		return []string{"npm install", "npm run dev", "npm test", "npm run build"}
	case ProjectDeno:
		return []string{"deno run", "deno test", "deno fmt", "deno lint"}
	case ProjectRust:
		return []string{"cargo build", "cargo run", "cargo test", "cargo fmt"}
	case ProjectGo:
		return []string{"go build ./...", "go run .", "go test ./...", "go fmt ./..."}
	case ProjectPython:
		return []string{"python -m pytest", "pip install -r requirements.txt", "python main.py"}
	case ProjectJava:
		return []string{"mvn compile", "mvn test", "gradle build"}
	case ProjectRuby:
		return []string{"bundle install", "rspec", "rails server"}
	case ProjectPHP:
		return []string{"composer install", "php artisan serve", "phpunit"}
	default:
		return []string{}
	}
}
