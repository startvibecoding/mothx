package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadTool reads file contents.
type ReadTool struct {
	registry *Registry
}

// NewReadTool creates a new read tool.
func NewReadTool(r *Registry) *ReadTool {
	return &ReadTool{registry: r}
}

func (t *ReadTool) Name() string { return "read" }

func (t *ReadTool) Description() string {
	return "Read the contents of a file. Supports text files and images (jpg, png, gif, webp). For text files, output is truncated at 2000 lines or 50KB. Use offset/limit for large files."
}

func (t *ReadTool) PromptSnippet() string {
	return "Read file contents"
}

func (t *ReadTool) PromptGuidelines() []string {
	return []string{"Use read to examine files instead of cat or sed."}
}

func (t *ReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to the file to read"
			},
			"offset": {
				"type": "integer",
				"description": "Line number to start reading from (1-indexed)"
			},
			"limit": {
				"type": "integer",
				"description": "Maximum number of lines to read"
			}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	path, _ := params["path"].(string)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	path = t.resolvePath(path)
	path = filepath.Clean(path)

	// Check for image files
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if imageExts[ext] {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("cannot access file: %w", err)
		}
		return fmt.Sprintf("[Image file: %s, size: %d bytes]", path, info.Size()), nil
	}

	// Read text file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	offset := 0
	if v, ok := params["offset"].(float64); ok && v > 0 {
		offset = int(v) - 1 // Convert to 0-indexed
	}

	limit := len(lines)
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	// Clamp
	if offset >= len(lines) {
		return "(end of file)", nil
	}
	end := offset + limit
	if end > len(lines) {
		end = len(lines)
	}

	selected := lines[offset:end]

	// Number lines
	var sb strings.Builder
	for i, line := range selected {
		lineNum := offset + i + 1
		sb.WriteString(fmt.Sprintf("%d\t%s\n", lineNum, line))
	}

	result := sb.String()

	// Truncate
	const maxBytes = 50000
	if len(result) > maxBytes {
		result = result[:maxBytes] + fmt.Sprintf("\n... (truncated, total %d lines)", len(lines))
	}

	return result, nil
}

func (t *ReadTool) resolvePath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return strings.Replace(path, "~", home, 1)
	}
	if !filepath.IsAbs(path) {
		return filepath.Join(t.registry.GetWorkDir(), path)
	}
	return path
}
