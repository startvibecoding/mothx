package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LsTool lists directory contents.
type LsTool struct {
	registry *Registry
}

// NewLsTool creates a new ls tool.
func NewLsTool(r *Registry) *LsTool {
	return &LsTool{registry: r}
}

func (t *LsTool) Name() string { return "ls" }

func (t *LsTool) Description() string {
	return "List directory contents with details. Shows files and directories with sizes and types."
}

func (t *LsTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Directory to list (default: current directory)"
			}
		}
	}`)
}

func (t *LsTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	dirPath := t.registry.GetWorkDir()
	if v, ok := params["path"].(string); ok && v != "" {
		dirPath = t.resolvePath(v)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("read directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	var sb strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("  📁 %s/\n", name))
		} else {
			size := formatSize(info.Size())
			sb.WriteString(fmt.Sprintf("  📄 %s (%s)\n", name, size))
		}
	}

	result := sb.String()
	if result == "" {
		return "(empty directory)", nil
	}
	return result, nil
}

func (t *LsTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.registry.GetWorkDir(), path)
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
