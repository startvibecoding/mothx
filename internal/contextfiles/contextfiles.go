package contextfiles

import (
	"os"
	"path/filepath"
	"strings"
)

// Well-known context file names used by various AI coding tools.
var wellKnownFiles = []string{
	// VibeCoding
	"AGENTS.md",
	"CLAUDE.md",

	// Cursor
	".cursorrules",

	// Windsurf
	".windsurfrules",

	// Cline/Roo
	".clinerules",

	// GitHub Copilot
	".github/copilot-instructions.md",

	// Generic
	"CONVENTIONS.md",
	"CONTRIBUTING.md",
	"INSTRUCTIONS.md",
}

// LoadResult contains the loaded context files.
type LoadResult struct {
	GlobalFiles  []FileContent // files from ~/.vibecoding/
	ParentFiles  []FileContent // files from parent directories
	ProjectFiles []FileContent // files from current directory
}

// FileContent represents a loaded context file.
type FileContent struct {
	Path    string // absolute path
	Name    string // file name
	Content string // file content
}

// LoadContextFiles discovers and loads context files from all relevant locations.
// It walks up from cwd to the root, then checks the global config directory.
func LoadContextFiles(cwd string, globalConfigDir string, extraFiles []string) *LoadResult {
	result := &LoadResult{}

	// Combine well-known files with user-configured extra files
	fileNames := append([]string{}, wellKnownFiles...)
	fileNames = append(fileNames, extraFiles...)

	// Deduplicate
	seen := make(map[string]bool)
	var uniqueNames []string
	for _, name := range fileNames {
		if !seen[name] {
			seen[name] = true
			uniqueNames = append(uniqueNames, name)
		}
	}

	// 1. Load from current directory (highest priority)
	for _, name := range uniqueNames {
		path := filepath.Join(cwd, name)
		if content, err := os.ReadFile(path); err == nil {
			result.ProjectFiles = append(result.ProjectFiles, FileContent{
				Path:    path,
				Name:    name,
				Content: string(content),
			})
		}
	}

	// 2. Walk up from cwd to root, loading context files from parent directories
	dir := cwd
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		// Don't load from root or home directories to avoid noise
		if parent == "/" || parent == "" {
			break
		}

		for _, name := range uniqueNames {
			path := filepath.Join(parent, name)
			if content, err := os.ReadFile(path); err == nil {
				result.ParentFiles = append(result.ParentFiles, FileContent{
					Path:    path,
					Name:    name,
					Content: string(content),
				})
			}
		}
		dir = parent
	}

	// 3. Load from global config directory (~/.vibecoding/)
	if globalConfigDir != "" {
		for _, name := range uniqueNames {
			path := filepath.Join(globalConfigDir, name)
			if content, err := os.ReadFile(path); err == nil {
				result.GlobalFiles = append(result.GlobalFiles, FileContent{
					Path:    path,
					Name:    name,
					Content: string(content),
				})
			}
		}
	}

	return result
}

// BuildContextString concatenates all context files into a single string
// suitable for appending to the system prompt.
// Order: global -> parent (root to cwd) -> project (current dir)
func BuildContextString(result *LoadResult) string {
	var sb strings.Builder

	if len(result.GlobalFiles) == 0 && len(result.ParentFiles) == 0 && len(result.ProjectFiles) == 0 {
		return ""
	}

	sb.WriteString("\n## Project Context\n\n")
	sb.WriteString("The following context files have been loaded from the project and configuration directories:\n\n")

	// Global files (lowest priority)
	for _, f := range result.GlobalFiles {
		sb.WriteString(formatContextFile(f, "global"))
	}

	// Parent files (medium priority, root to cwd order)
	// Reverse so closer parents have higher priority
	for i := len(result.ParentFiles) - 1; i >= 0; i-- {
		sb.WriteString(formatContextFile(result.ParentFiles[i], "parent"))
	}

	// Project files (highest priority)
	for _, f := range result.ProjectFiles {
		sb.WriteString(formatContextFile(f, "project"))
	}

	return sb.String()
}

func formatContextFile(f FileContent, scope string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("File: `" + f.Path + "` (scope: " + scope + ")\n")
	sb.WriteString("---\n")
	sb.WriteString(f.Content)
	if !strings.HasSuffix(f.Content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}
