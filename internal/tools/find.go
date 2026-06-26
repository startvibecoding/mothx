package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	gofd "github.com/startvibecoding/go-fd"
)

// FindTool searches for files by name pattern using the go-fd SDK.
type FindTool struct {
	registry *Registry
}

// NewFindTool creates a new find tool.
func NewFindTool(r *Registry) *FindTool {
	return &FindTool{registry: r}
}

func (t *FindTool) Name() string { return "find" }

func (t *FindTool) Description() string {
	return "Search for files by name pattern. Supports glob patterns. Use for finding files by name, extension, or path pattern."
}

func (t *FindTool) PromptSnippet() string {
	return "Find files by glob pattern (preferred for locating files, respects .gitignore)"
}

func (t *FindTool) PromptGuidelines() []string {
	return nil
}

func (t *FindTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Glob pattern to match file names (e.g. '*.go', '*.test.*')"
			},
			"path": {
				"type": "string",
				"description": "Directory to search in (default: current directory)"
			},
			"maxDepth": {
				"type": "integer",
				"description": "Maximum directory depth (default: unlimited)"
			},
			"maxResults": {
				"type": "integer",
				"description": "Maximum number of results (default 100)"
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *FindTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	pattern, _ := params["pattern"].(string)
	if pattern == "" {
		return ToolResult{}, fmt.Errorf("pattern is required")
	}

	searchPath := t.registry.GetWorkDir()
	if v, ok := params["path"].(string); ok && v != "" {
		var err error
		searchPath, err = t.registry.ResolvePath(v)
		if err != nil {
			return ToolResult{}, fmt.Errorf("invalid path: %w", err)
		}
	}
	if _, err := os.Stat(searchPath); err != nil {
		return ToolResult{}, fmt.Errorf("invalid path: %w", err)
	}

	maxDepth := 0
	if v, ok := params["maxDepth"].(float64); ok && v > 0 {
		maxDepth = int(v)
	}

	maxResults := 100
	if v, ok := params["maxResults"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	paths, err := gofd.Find(ctx, gofd.Options{
		Pattern:    pattern,
		Paths:      []string{searchPath},
		Glob:       true,
		MaxDepth:   maxDepth,
		MaxResults: maxResults,
	})
	if err != nil {
		return ToolResult{}, fmt.Errorf("find search failed: %w", err)
	}

	if len(paths) == 0 {
		return NewTextToolResult("(no files found)"), nil
	}

	return NewTextToolResult(strings.Join(paths, "\n")), nil
}
