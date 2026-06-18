package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	internalagent "github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

// RegisterTools registers workflow tools.
func RegisterTools(registry *tools.Registry, manager *internalagent.AgentManager, store Store) {
	if registry == nil || manager == nil {
		return
	}
	if store == nil {
		store = DefaultStore()
	}
	active := DefaultActiveRegistry()
	registry.Register(NewRunToolWithActive(manager, store, active))
	registry.Register(NewStatusTool(store))
	registry.Register(NewCancelTool(active))
}

func DefaultStore() Store {
	return NewFileStore(filepath.Join(config.ConfigDir(), "workflows", "runs"))
}

type RunTool struct {
	manager *internalagent.AgentManager
	store   Store
	active  *ActiveRegistry
}

func NewRunTool(manager *internalagent.AgentManager, store Store) *RunTool {
	return NewRunToolWithActive(manager, store, DefaultActiveRegistry())
}

func NewRunToolWithActive(manager *internalagent.AgentManager, store Store, active *ActiveRegistry) *RunTool {
	if active == nil {
		active = DefaultActiveRegistry()
	}
	return &RunTool{manager: manager, store: store, active: active}
}

func (t *RunTool) Name() string { return "workflow_run" }
func (t *RunTool) Description() string {
	return "Run an Elisp workflow DSL script that orchestrates worker agents."
}
func (t *RunTool) PromptSnippet() string {
	return "Run a multi-phase Elisp workflow with worker agents"
}
func (t *RunTool) PromptGuidelines() []string {
	return []string{
		"Use workflow_run for multi-phase tasks with independent worker-agent branches and fan-in verification.",
		"Write workflow DSL using plain Elisp syntax; do not use Markdown code fences.",
		"Before calling workflow_run, ensure the source is one complete (workflow \"name\" ...) form with balanced parentheses and closed double-quoted strings.",
		"Use quoted string lists for tools, for example :tools '(\"read\" \"grep\").",
		"Keep worker prompts explicit and bounded; use result to pass prior phase outputs into later phases.",
	}
}
func (t *RunTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
			"type": "object",
			"properties": {
				"source": {
					"type": "string",
					"description": "Complete raw Elisp workflow DSL source. Must be one balanced (workflow \"name\" ...) form with closed double-quoted strings; do not pass Markdown fences."
				}
			},
			"required": ["source"]
		}`)
}

func (t *RunTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	source, _ := params["source"].(string)
	source = strings.TrimSpace(source)
	if source == "" {
		return tools.ToolResult{}, fmt.Errorf("source is required")
	}
	parentID, _ := internalagent.AgentIDFromContext(ctx)
	parentEventCh, _ := internalagent.EventChanFromContext(ctx)
	parentRunCtx, _ := internalagent.ParentRunContextFromContext(ctx)
	parentMode, _ := internalagent.ParentModeFromContext(ctx)

	host := &AgentHost{
		Manager:       t.manager,
		ParentID:      parentID,
		ParentMode:    parentMode,
		ParentEventCh: parentEventCh,
		ParentRunCtx:  parentRunCtx,
	}
	state, err := (&Runner{Host: host, Store: t.store, Active: t.active}).Run(ctx, source)
	if err != nil {
		if errors.Is(err, context.Canceled) && state != nil {
			return runToolResult(state), nil
		}
		return tools.ToolResult{}, err
	}
	return runToolResult(state), nil
}

func runToolResult(state *RunState) tools.ToolResult {
	data, _ := json.Marshal(map[string]any{
		"id":      state.ID,
		"name":    state.Name,
		"status":  state.Status,
		"results": summarizeResults(state),
	})
	return tools.NewTextToolResult(string(data))
}

type StatusTool struct {
	store Store
}

func NewStatusTool(store Store) *StatusTool {
	return &StatusTool{store: store}
}

func (t *StatusTool) Name() string { return "workflow_status" }
func (t *StatusTool) Description() string {
	return "Show workflow run status. Pass an id for details, or omit id to list recent runs."
}
func (t *StatusTool) PromptSnippet() string { return "Inspect workflow run status and results" }
func (t *StatusTool) PromptGuidelines() []string {
	return []string{"Use workflow_status to inspect workflow run state without invoking the LLM."}
}
func (t *StatusTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "Workflow run id. Omit to list recent runs."}
		}
	}`)
}

func (t *StatusTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	id, _ := params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		runs, err := t.store.List(ctx)
		if err != nil {
			return tools.ToolResult{}, err
		}
		data, _ := json.Marshal(runs)
		return tools.NewTextToolResult(string(data)), nil
	}
	state, err := t.store.Load(ctx, id)
	if err != nil {
		return tools.ToolResult{}, err
	}
	data, _ := json.Marshal(state)
	return tools.NewTextToolResult(string(data)), nil
}

type CancelTool struct {
	active *ActiveRegistry
}

func NewCancelTool(active *ActiveRegistry) *CancelTool {
	if active == nil {
		active = DefaultActiveRegistry()
	}
	return &CancelTool{active: active}
}

func (t *CancelTool) Name() string { return "workflow_cancel" }
func (t *CancelTool) Description() string {
	return "Cancel an active workflow run by id."
}
func (t *CancelTool) PromptSnippet() string { return "Cancel an active workflow run" }
func (t *CancelTool) PromptGuidelines() []string {
	return []string{"Use workflow_cancel only for active workflow runs that should be interrupted."}
}
func (t *CancelTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "Workflow run id"}
		},
		"required": ["id"]
	}`)
}

func (t *CancelTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	id, _ := params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return tools.ToolResult{}, fmt.Errorf("id is required")
	}
	if !t.active.Cancel(id) {
		return tools.ToolResult{}, fmt.Errorf("workflow run %q is not active", id)
	}
	data, _ := json.Marshal(map[string]any{
		"id":     id,
		"status": StatusCanceled,
	})
	return tools.NewTextToolResult(string(data)), nil
}

func summarizeResults(state *RunState) map[string]string {
	out := make(map[string]string, len(state.Results))
	for key, result := range state.Results {
		out[key] = result.Status
	}
	return out
}
