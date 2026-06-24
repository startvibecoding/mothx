package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/sandbox"
	"github.com/startvibecoding/vibecoding/internal/skills"
)

// writeFileAtomic writes data to path atomically using a temporary file and rename.
// It preserves the existing file's permissions if the file already exists.
func writeFileAtomic(path string, data []byte) error {
	// Determine target permissions: preserve existing or use default
	perm := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	// Create temp file in the same directory for atomic rename
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// ToolResult represents the result of a tool execution.
// It can contain plain text and optional rich content blocks (e.g. images).
type ToolResult struct {
	Text     string                  // Plain text result (always populated for display/logging)
	Contents []provider.ContentBlock // Rich content blocks (text + images) for the LLM
	Diff     *FileDiff               // Optional structured file diff for UI/reporting
	Plan     *TaskPlan               // Optional structured task plan for UI/reporting
}

// FileDiff describes a file change produced by a write-like tool.
type FileDiff struct {
	Path         string
	Added        int
	Deleted      int
	AddedLines   []int
	DeletedLines []int
	Unified      string
	Truncated    bool
}

// TaskPlan describes a structured task plan emitted by the plan tool.
type TaskPlan struct {
	Title string
	Steps []PlanStep
	Note  string
}

// PlanStep describes one step in a task plan.
type PlanStep struct {
	Title  string
	Status string
}

// NewTextToolResult creates a plain text tool result.
func NewTextToolResult(text string) ToolResult {
	return ToolResult{Text: text}
}

// NewDiffToolResult creates a text tool result with structured diff metadata.
func NewDiffToolResult(text string, diff *FileDiff) ToolResult {
	return ToolResult{Text: text, Diff: diff}
}

// NewPlanToolResult creates a text tool result with structured plan metadata.
func NewPlanToolResult(text string, plan *TaskPlan) ToolResult {
	return ToolResult{Text: text, Plan: plan}
}

// NewImageToolResult creates a tool result that includes an image.
// text is the human-readable description, mimeType and base64Data are the image payload.
func NewImageToolResult(text, mimeType, base64Data string) ToolResult {
	return ToolResult{
		Text: text,
		Contents: []provider.ContentBlock{
			{Type: "text", Text: text},
			{Type: "image", Image: &provider.ImageContent{MimeType: mimeType, Data: base64Data}},
		},
	}
}

// Tool is the interface that all tools must implement.
type Tool interface {
	// Name returns the tool's name.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// PromptSnippet returns a short one-line description for the system prompt's Available tools section.
	PromptSnippet() string

	// PromptGuidelines returns guideline bullets for the system prompt's Guidelines section.
	PromptGuidelines() []string

	// Parameters returns the JSON Schema for the tool's parameters.
	Parameters() json.RawMessage

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params map[string]any) (ToolResult, error)
}

// ExecutionTimeoutProvider lets a tool override the agent's default execution
// timeout. The bool reports whether an override is provided. A non-positive
// duration disables the agent-level deadline while preserving parent
// cancellation.
type ExecutionTimeoutProvider interface {
	ExecutionTimeout(params map[string]any) (time.Duration, bool)
}

// ToolDefinition converts a Tool to a provider.ToolDefinition.
func ToolDefinition(t Tool) provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

// Registry manages available tools.
type Registry struct {
	mu         sync.RWMutex
	tools      map[string]Tool
	order      []string
	sandbox    sandbox.Sandbox
	workDir    string
	jobManager *JobManager
	skillsMgr  *skills.Manager
	fileLocks  *FileLockManager
}

// NewRegistry creates a new tool registry.
func NewRegistry(workDir string, sb sandbox.Sandbox) *Registry {
	return &Registry{
		tools:      make(map[string]Tool),
		workDir:    workDir,
		sandbox:    sb,
		jobManager: NewJobManager(),
		fileLocks:  DefaultFileLockManager(),
	}
}

// RegistryConfig configures a Registry instance.
type RegistryConfig struct {
	WorkDir        string
	Sandbox        sandbox.Sandbox
	ToolFilter     []string         // optional: only register these tools (empty = all)
	SkillsMgr      *skills.Manager  // optional: skills manager for skill_ref tool
	EnablePlanTool *bool            // optional: defaults to true when nil
	FileLocks      *FileLockManager // optional: defaults to process-wide manager
}

// NewRegistryWithConfig creates a Registry with the given config.
func NewRegistryWithConfig(cfg RegistryConfig) *Registry {
	fileLocks := cfg.FileLocks
	if fileLocks == nil {
		fileLocks = DefaultFileLockManager()
	}
	r := &Registry{
		tools:      make(map[string]Tool),
		workDir:    cfg.WorkDir,
		sandbox:    cfg.Sandbox,
		jobManager: NewJobManager(),
		skillsMgr:  cfg.SkillsMgr,
		fileLocks:  fileLocks,
	}
	enablePlanTool := true
	if cfg.EnablePlanTool != nil {
		enablePlanTool = *cfg.EnablePlanTool
	}
	if len(cfg.ToolFilter) == 0 {
		r.RegisterDefaultsWithPlanTool(enablePlanTool)
	} else {
		r.RegisterFiltered(cfg.ToolFilter)
	}
	return r
}

// JobManager returns the registry's per-instance job manager.
func (r *Registry) JobManager() *JobManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.jobManager
}

// FileLocks returns the registry's in-memory file lock manager.
func (r *Registry) FileLocks() *FileLockManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.fileLocks
}

func (r *Registry) acquireFileLock(ctx context.Context, path, owner string) (func(), error) {
	mgr := r.FileLocks()
	if mgr == nil {
		return func() {}, nil
	}
	return mgr.Acquire(ctx, path, owner)
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := t.Name()
	if _, exists := r.tools[name]; !exists {
		r.order = append(r.order, name)
	}
	r.tools[name] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Remove removes a tool by name. No-op if not found.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; ok {
		delete(r.tools, name)
		// Also remove from order
		for i, n := range r.order {
			if n == name {
				r.order = append(r.order[:i], r.order[i+1:]...)
				break
			}
		}
	}
}

// All returns all registered tools in order.
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []Tool
	for _, name := range r.order {
		if t, ok := r.tools[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// Definitions returns tool definitions for all registered tools.
func (r *Registry) Definitions() []provider.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var defs []provider.ToolDefinition
	for _, name := range r.order {
		if t, ok := r.tools[name]; ok {
			defs = append(defs, ToolDefinition(t))
		}
	}
	return defs
}

// GetSandbox returns the registry's sandbox.
func (r *Registry) GetSandbox() sandbox.Sandbox {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sandbox
}

// GetWorkDir returns the registry's working directory.
func (r *Registry) GetWorkDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.workDir
}

// ResolvePath resolves a user-provided path to an absolute path constrained to the work directory.
func (r *Registry) ResolvePath(path string) (string, error) {
	r.mu.RLock()
	workDir := r.workDir
	r.mu.RUnlock()

	// Expand ~ (only ~/ prefix, not arbitrary ~user)
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = home
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert relative paths to absolute within workDir
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	// Clean to resolve .. segments
	path = filepath.Clean(path)

	// Validate: path must not escape workDir
	workDir = filepath.Clean(workDir)
	rel, err := filepath.Rel(workDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s escapes working directory %s", path, workDir)
	}

	return path, nil
}

// SetSandbox updates the sandbox used by tools.
func (r *Registry) SetSandbox(sb sandbox.Sandbox) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sandbox = sb
}

// RegisterDefaults registers all default tools.
func (r *Registry) RegisterDefaults() {
	r.RegisterDefaultsWithPlanTool(true)
}

// RegisterDefaultsWithPlanTool registers all default tools, optionally including the plan tool.
func (r *Registry) RegisterDefaultsWithPlanTool(enablePlanTool bool) {
	r.Register(NewReadTool(r))
	r.Register(NewLsTool(r))
	r.Register(NewGrepTool(r))
	r.Register(NewFindTool(r))
	if enablePlanTool {
		r.Register(NewPlanTool(r))
	}
	r.Register(NewWriteTool(r))
	r.Register(NewEditTool(r))
	bashTool := NewBashToolWithJM(r, r.jobManager)
	r.Register(bashTool)
	r.Register(NewJobsTool(r, bashTool))
	r.Register(NewKillTool(r, bashTool))
	if r.skillsMgr != nil {
		r.Register(NewSkillRefTool(r.skillsMgr))
	}
}

// RegisterFiltered registers only the specified tools by name.
func (r *Registry) RegisterFiltered(toolNames []string) {
	allTools := map[string]func() Tool{
		"read":  func() Tool { return NewReadTool(r) },
		"ls":    func() Tool { return NewLsTool(r) },
		"grep":  func() Tool { return NewGrepTool(r) },
		"find":  func() Tool { return NewFindTool(r) },
		"plan":  func() Tool { return NewPlanTool(r) },
		"write": func() Tool { return NewWriteTool(r) },
		"edit":  func() Tool { return NewEditTool(r) },
	}
	bashTool := NewBashToolWithJM(r, r.jobManager)
	allTools["bash"] = func() Tool { return bashTool }
	allTools["jobs"] = func() Tool { return NewJobsTool(r, bashTool) }
	allTools["kill"] = func() Tool { return NewKillTool(r, bashTool) }
	if r.skillsMgr != nil {
		allTools["skill_ref"] = func() Tool { return NewSkillRefTool(r.skillsMgr) }
	}

	for _, name := range toolNames {
		if factory, ok := allTools[name]; ok {
			r.Register(factory())
		}
	}
}

// ModeTools returns tool definitions appropriate for the given mode.
func (r *Registry) ModeTools(mode string) []provider.ToolDefinition {
	switch mode {
	case "plan":
		// Plan mode: read-only tools + any extras like question
		var defs []provider.ToolDefinition
		for _, t := range r.All() {
			switch t.Name() {
			case "read", "grep", "find", "ls", "plan":
				defs = append(defs, ToolDefinition(t))
			case "question":
				defs = append(defs, ToolDefinition(t))
			}
		}
		return defs
	case "agent":
		// Agent mode: all tools, including the interactive question tool.
		var defs []provider.ToolDefinition
		for _, t := range r.All() {
			defs = append(defs, ToolDefinition(t))
		}
		return defs
	default:
		// YOLO (and unattended modes): all tools except question (interactive only)
		var defs []provider.ToolDefinition
		for _, t := range r.All() {
			if t.Name() == "question" {
				continue
			}
			defs = append(defs, ToolDefinition(t))
		}
		return defs
	}
}

// ToolSnippets returns prompt snippets for the given tool names.
func (r *Registry) ToolSnippets(toolNames []string) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	snippets := make(map[string]string)
	for _, name := range toolNames {
		if t, ok := r.tools[name]; ok {
			if snippet := t.PromptSnippet(); snippet != "" {
				snippets[name] = snippet
			}
		}
	}
	return snippets
}

// ToolGuidelines returns prompt guidelines for the given tool names.
func (r *Registry) ToolGuidelines(toolNames []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var guidelines []string
	seen := make(map[string]bool)
	for _, name := range toolNames {
		if t, ok := r.tools[name]; ok {
			for _, g := range t.PromptGuidelines() {
				if !seen[g] {
					seen[g] = true
					guidelines = append(guidelines, g)
				}
			}
		}
	}
	return guidelines
}
