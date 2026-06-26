package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/startvibecoding/go-ripgrep/pkg/globset"
	"github.com/startvibecoding/go-ripgrep/pkg/ignore"
	"github.com/startvibecoding/go-ripgrep/pkg/matcher"
	"github.com/startvibecoding/go-ripgrep/pkg/searcher"
)

const maxGrepOutputBytes = 200000

// GrepTool searches file contents using the go-ripgrep SDK.
type GrepTool struct {
	registry *Registry
}

// NewGrepTool creates a new grep tool.
func NewGrepTool(r *Registry) *GrepTool {
	return &GrepTool{registry: r}
}

func (t *GrepTool) Name() string { return "grep" }

func (t *GrepTool) Description() string {
	return "Search file contents using regex patterns. Returns matching lines with file paths and line numbers. Use for finding code patterns, function definitions, etc."
}

func (t *GrepTool) PromptSnippet() string {
	return "Search file contents for patterns (preferred for code search, respects .gitignore)"
}

func (t *GrepTool) PromptGuidelines() []string {
	return nil
}

func (t *GrepTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "Regex pattern to search for"
			},
			"path": {
				"type": "string",
				"description": "Directory or file to search in (default: current directory)"
			},
			"include": {
				"type": "string",
				"description": "File pattern to include (e.g. '*.go')"
			},
			"maxResults": {
				"type": "integer",
				"description": "Maximum number of results (default 100)"
			}
		},
		"required": ["pattern"]
	}`)
}

func (t *GrepTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
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

	include, _ := params["include"].(string)
	maxResults := 100
	if v, ok := params["maxResults"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	m, err := matcher.BuildMatcher(pattern, false, false, false)
	if err != nil {
		return ToolResult{}, fmt.Errorf("grep search failed: %w", err)
	}

	var includeGlob *globset.GlobSet
	if include != "" {
		var err error
		includeGlob, err = globset.NewGlobSet([]string{include})
		if err != nil {
			return ToolResult{}, fmt.Errorf("grep search failed: %w", err)
		}
	}

	searchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := searcher.NewSearcher(m, 0, 0, 0, false)
	s.SetContext(searchCtx)

	files, err := collectGrepFiles(searchCtx, searchPath, includeGlob)
	if err != nil {
		return ToolResult{}, fmt.Errorf("grep search failed: %w", err)
	}

	var lines []string
	bytesUsed := 0
	count := 0
	truncated := false

	for _, file := range files {
		select {
		case <-searchCtx.Done():
			return ToolResult{}, ctx.Err()
		default:
		}

		results, err := s.SearchFile(file.Path)
		if err != nil {
			continue
		}
		for _, res := range results {
			if res == nil {
				continue
			}
			for _, m := range res.Matches {
				if m.IsContext {
					continue
				}
				if maxResults > 0 && count >= maxResults {
					truncated = true
					break
				}
				line := fmt.Sprintf("%s:%d:%s", res.Path, m.LineNum, strings.TrimRight(m.Line, "\r\n"))
				if bytesUsed+len(line) > maxGrepOutputBytes {
					truncated = true
					break
				}
				lines = append(lines, line)
				bytesUsed += len(line)
				count++
			}
			if truncated {
				break
			}
		}
		if truncated {
			cancel()
			break
		}
	}

	if len(lines) == 0 {
		return NewTextToolResult("(no matches found)"), nil
	}

	output := strings.Join(lines, "\n")
	if truncated {
		if maxResults > 0 && count >= maxResults {
			output += fmt.Sprintf("\n... (truncated, showing first %d results)", maxResults)
		} else {
			output += fmt.Sprintf("\n... (truncated at %d bytes)", maxGrepOutputBytes)
		}
	}

	return NewTextToolResult(output), nil
}

type grepFile struct {
	Path string
	Rel  string
}

func collectGrepFiles(ctx context.Context, root string, includeGlob *globset.GlobSet) ([]grepFile, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if shouldIncludeGrepPath(root, filepath.Base(root), includeGlob) {
			return []grepFile{{Path: root, Rel: filepath.Base(root)}}, nil
		}
		return nil, nil
	}

	stack := ignore.NewIgnoreStack(false, false, 0)
	stack.LoadBaseRules(root)

	var files []grepFile
	err = walkGrepDir(ctx, root, root, stack, includeGlob, &files)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func walkGrepDir(ctx context.Context, root, dir string, stack *ignore.IgnoreStack, includeGlob *globset.GlobSet, files *[]grepFile) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := stack.Push(dir); err != nil {
		return err
	}
	defer stack.Pop()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path := filepath.Join(dir, entry.Name())
		isDir := entry.IsDir()
		if stack.IsIgnored(path, isDir) {
			continue
		}
		if isDir {
			if err := walkGrepDir(ctx, root, path, stack.Clone(), includeGlob, files); err != nil {
				return err
			}
			continue
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		if shouldIncludeGrepPath(path, rel, includeGlob) {
			*files = append(*files, grepFile{Path: path, Rel: rel})
		}
	}
	return nil
}

func shouldIncludeGrepPath(path, rel string, includeGlob *globset.GlobSet) bool {
	if ignore.ShouldIgnoreByType(filepath.Base(path), nil, nil) {
		return false
	}
	if includeGlob == nil {
		return true
	}
	return !includeGlob.MatchGlobFilter(filepath.ToSlash(rel))
}
