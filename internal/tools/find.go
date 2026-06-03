package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/vendored"
)

// FindTool searches for files by name pattern using fd.
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

	maxDepth := -1
	if v, ok := params["maxDepth"].(float64); ok && v > 0 {
		maxDepth = int(v)
	}

	maxResults := 100
	if v, ok := params["maxResults"].(float64); ok && v > 0 {
		maxResults = int(v)
	}

	// 选择可用的 fd 命令，当前平台没有内嵌 fd 时退回系统 find。
	fdPath, err := resolveFdPath()
	if err != nil {
		if errors.Is(err, vendored.ErrUnsupportedPlatform) {
			return executeNativeFind(ctx, pattern, searchPath, maxDepth, maxResults)
		}
		return ToolResult{}, err
	}

	// 构建 fd 命令参数
	args := []string{
		"--color=never",
		"--glob",
		fmt.Sprintf("--max-results=%d", maxResults),
	}

	if maxDepth >= 0 {
		args = append(args, fmt.Sprintf("--max-depth=%d", maxDepth))
	}

	args = append(args, "--", pattern, searchPath)

	// 执行 fd
	cmd := exec.CommandContext(ctx, fdPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// fd 返回 1 表示没有匹配
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return NewTextToolResult("(no files found)"), nil
		}
		// 其他错误
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return ToolResult{}, fmt.Errorf("fd 执行失败: %s", errMsg)
		}
		if isExecFormatError(err) {
			return executeNativeFind(ctx, pattern, searchPath, maxDepth, maxResults)
		}
		return ToolResult{}, fmt.Errorf("fd 执行失败: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return NewTextToolResult("(no files found)"), nil
	}

	// fd 输出就是每行一个路径，与原实现格式一致
	return NewTextToolResult(output), nil
}

func resolveFdPath() (string, error) {
	if !vendored.HasEmbeddedTools() {
		return "", fmt.Errorf("%w", vendored.ErrUnsupportedPlatform)
	}

	fdPath := vendored.FdPath()
	if fdPath == "" {
		return "", fmt.Errorf("无法确定 fd 路径")
	}

	// 缺失或不可执行时，尝试从 go:embed 释放到 ~/.vibecoding/bin/
	if err := vendored.Ensure(); err != nil {
		return "", fmt.Errorf("准备 fd 失败: %w", err)
	}

	return fdPath, nil
}

func executeNativeFind(ctx context.Context, pattern, searchPath string, maxDepth, maxResults int) (ToolResult, error) {
	findPath, err := exec.LookPath("find")
	if err != nil {
		return ToolResult{}, fmt.Errorf("fd is unsupported on this platform and system find was not found: %w", err)
	}

	args := []string{searchPath}
	if maxDepth >= 0 {
		args = append(args, "-maxdepth", fmt.Sprintf("%d", maxDepth))
	}
	args = append(args, "-type", "f")

	pathPattern := pattern
	if !filepath.IsAbs(pathPattern) {
		pathPattern = filepath.Join(searchPath, filepath.FromSlash(pattern))
	}
	args = append(args, "(", "-name", pattern, "-o", "-path", pathPattern, ")")

	cmd := exec.CommandContext(ctx, findPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return ToolResult{}, fmt.Errorf("find execution failed: %s", errMsg)
		}
		return ToolResult{}, fmt.Errorf("find execution failed: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return NewTextToolResult("(no files found)"), nil
	}

	lines := strings.Split(output, "\n")
	sort.Strings(lines)
	if maxResults > 0 && len(lines) > maxResults {
		lines = lines[:maxResults]
	}
	return NewTextToolResult(strings.Join(lines, "\n")), nil
}
