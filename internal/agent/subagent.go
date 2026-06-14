package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

// SubAgentSpawnTool creates and starts a sub-agent.
type SubAgentSpawnTool struct {
	manager *AgentManager
}

// NewSubAgentSpawnTool creates a new subagent_spawn tool.
func NewSubAgentSpawnTool(m *AgentManager) *SubAgentSpawnTool {
	return &SubAgentSpawnTool{manager: m}
}

// DelegateSubAgentTool runs exactly one delegated sub-agent task synchronously.
type DelegateSubAgentTool struct {
	manager *AgentManager
	busy    atomic.Bool
}

// NewDelegateSubAgentTool creates a blocking delegate_subagent tool.
func NewDelegateSubAgentTool(m *AgentManager) *DelegateSubAgentTool {
	return &DelegateSubAgentTool{manager: m}
}

func (t *DelegateSubAgentTool) Name() string { return "delegate_subagent" }
func (t *DelegateSubAgentTool) Description() string {
	return "Delegate one bounded independent subtask to a blocking sub-agent. Waits until completion and returns a summarized result."
}
func (t *DelegateSubAgentTool) PromptSnippet() string {
	return "Delegate one bounded independent subtask to a blocking sub-agent"
}
func (t *DelegateSubAgentTool) PromptGuidelines() []string {
	return []string{
		"Use delegate_subagent when the subtask requires multi-step exploration (grep many files, trace code paths, run multiple commands) but you only need the final answer — the intermediate steps would bloat your context",
		"Do NOT delegate single-tool tasks (read one file, run one command) — direct execution is cheaper",
		"Do NOT delegate tasks smaller than ~3 tool calls, tasks needing user clarification mid-way, or highly stateful work depending on conversation history",
		"Write a specific task: state the exact goal, list relevant file paths/names, specify expected output format, and include stop conditions",
		"Only one delegated sub-agent can run at a time; review its result before acting — treat the output as evidence, not ground truth",
	}
}
func (t *DelegateSubAgentTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task": {"type": "string", "description": "A specific, bounded task description. Must include: (1) the exact goal or question, (2) relevant file paths or search patterns, (3) expected output format, (4) stop conditions. Example: 'Find all Go files in internal/gateway/ that import net/http but do not call http.Error. Return file paths with line numbers.'"},
			"mode": {"type": "string", "enum": ["plan", "agent", "yolo"], "default": "agent", "description": "Sub-agent execution mode. 'agent' for balanced safety, 'yolo' for unrestricted access, 'plan' for read-only analysis."},
			"work_dir": {"type": "string", "description": "Working directory for the sub-agent (defaults to current directory). Set explicitly if the task targets a different directory."},
			"tools": {"type": "array", "items": {"type": "string"}, "description": "Restrict sub-agent to specific tools (empty = all tools except nested sub-agent/delegate). Use to narrow scope, e.g. ['read', 'grep', 'find'] for investigation-only tasks."},
			"max_iterations": {"type": "integer", "default": 50, "description": "Maximum tool-call iterations. Lower for simple tasks (10-20), higher for complex exploration (50-100)."},
			"system_prompt_extra": {"type": "string", "description": "Additional context or constraints for the sub-agent. Use to pass domain knowledge, coding conventions, or specific instructions not in the task description."}
		},
		"required": ["task"]
	}`)
}

func (t *DelegateSubAgentTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	if t.manager == nil {
		return tools.ToolResult{}, fmt.Errorf("agent manager is not initialized")
	}
	if !t.busy.CompareAndSwap(false, true) {
		return tools.ToolResult{}, fmt.Errorf("a delegated sub-agent is already running")
	}
	defer t.busy.Store(false)

	started := time.Now()
	task, _ := params["task"].(string)
	task = strings.TrimSpace(task)
	if task == "" {
		return tools.ToolResult{}, fmt.Errorf("task is required")
	}

	mode, _ := params["mode"].(string)
	if mode == "" {
		// Inherit parent agent's mode (yolo/agent/plan) instead of hardcoding "agent"
		if parentMode, ok := ParentModeFromContext(ctx); ok && parentMode != "" {
			mode = parentMode
		} else {
			mode = "agent"
		}
	}
	workDir, _ := params["work_dir"].(string)
	maxIter := 50
	if v, ok := params["max_iterations"].(float64); ok && v > 0 {
		maxIter = int(v)
	}
	extra, _ := params["system_prompt_extra"].(string)

	var toolFilter []string
	if ts, ok := params["tools"].([]any); ok {
		for _, tt := range ts {
			if s, ok := tt.(string); ok {
				toolFilter = append(toolFilter, s)
			}
		}
	}

	parentID, _ := AgentIDFromContext(ctx)
	parentEventCh, _ := EventChanFromContext(ctx)
	parentRunCtx, ok := ParentRunContextFromContext(ctx)
	if !ok || parentRunCtx == nil {
		parentRunCtx = ctx
	}
	policy := DefaultSubAgentPolicy()
	runCtx, cancel := context.WithTimeout(parentRunCtx, policy.TimeoutPerAgent)
	defer cancel()

	a, err := t.manager.Create(AgentOptions{
		ParentID:          parentID,
		Mode:              mode,
		WorkDir:           workDir,
		Tools:             toolFilter,
		SystemPromptExtra: extra,
		MaxIterations:     maxIter,
	})
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("create delegated sub-agent: %w", err)
	}
	defer func() { _ = t.manager.Destroy(a.ID()) }()

	t.manager.MarkRunning(a.ID())
	t.manager.SetCancel(a.ID(), cancel)
	defer t.manager.SetCancel(a.ID(), nil)

	var runErr error
	completed := false
	toolCallCount := 0
	toolNames := make(map[string]int)
	ch := a.Run(runCtx, buildSubAgentTask(task))
	for e := range ch {
		if e.Type == agentpkg.EventToolApprovalRequest && parentEventCh != nil {
			_ = sendParentEvent(runCtx, parentEventCh, Event{
				Type:         EventToolApprovalRequest,
				AgentID:      a.ID(),
				ApprovalID:   e.ApprovalID,
				ApprovalTool: e.ApprovalTool,
				ApprovalArgs: e.ApprovalArgs,
			})
		}
		if e.Type == agentpkg.EventToolCall {
			toolCallCount++
			if e.ToolName != "" {
				toolNames[e.ToolName]++
			} else if e.ToolCall != nil && e.ToolCall.Name != "" {
				toolNames[e.ToolCall.Name]++
			}
		}
		switch e.Type {
		case agentpkg.EventDone:
			completed = true
			t.manager.MarkDone(a.ID(), lastAssistantResponse(a))
		case agentpkg.EventError:
			completed = true
			runErr = e.Error
			t.manager.MarkError(a.ID(), e.Error)
		}
	}
	if !completed && runCtx.Err() != nil {
		runErr = runCtx.Err()
		t.manager.MarkError(a.ID(), runErr)
	}

	response := lastAssistantResponse(a)
	result := map[string]any{
		"status":         "done",
		"result":         response,
		"duration":       time.Since(started).Round(time.Millisecond).String(),
		"tool_calls":     toolCallCount,
		"tool_breakdown": toolNames,
	}
	if runErr != nil {
		result["status"] = "error"
		result["error"] = runErr.Error()
		if response != "" {
			result["partial_result"] = response
		}
	}
	data, _ := json.Marshal(result)
	return tools.NewTextToolResult(string(data)), nil
}

func (t *SubAgentSpawnTool) Name() string { return "subagent_spawn" }
func (t *SubAgentSpawnTool) Description() string {
	return "Create and start a bounded sub-agent task. Returns a handle for status/result polling."
}
func (t *SubAgentSpawnTool) PromptSnippet() string {
	return "Create a bounded sub-agent task for independent work"
}
func (t *SubAgentSpawnTool) PromptGuidelines() []string {
	return []string{
		"Use subagent_spawn only for independent subtasks with clear scope, expected output, and stop conditions",
		"Spawn multiple sub-agents in parallel for independent investigation or review work, then reconcile their results in the main agent",
		"Use subagent_status to poll results and verify important claims before acting on them",
		"Use subagent_destroy to clean up finished sub-agents",
	}
}

func (t *SubAgentSpawnTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task": {"type": "string", "description": "Focused task for the sub-agent, including scope, relevant paths/context, expected artifact, and stop conditions"},
			"mode": {"type": "string", "enum": ["plan", "agent", "yolo"], "default": "agent", "description": "Agent mode"},
			"work_dir": {"type": "string", "description": "Working directory for the sub-agent (defaults to current)"},
			"tools": {"type": "array", "items": {"type": "string"}, "description": "Allowed tools (empty = all)"},
			"max_iterations": {"type": "integer", "default": 50, "description": "Maximum iterations"},
			"system_prompt_extra": {"type": "string", "description": "Extra context for the sub-agent"}
		},
		"required": ["task"]
	}`)
}

func (t *SubAgentSpawnTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	task, _ := params["task"].(string)
	if task == "" {
		return tools.ToolResult{}, fmt.Errorf("task is required")
	}

	mode, _ := params["mode"].(string)
	if mode == "" {
		// Inherit parent agent's mode (yolo/agent/plan) instead of hardcoding "agent"
		if parentMode, ok := ParentModeFromContext(ctx); ok && parentMode != "" {
			mode = parentMode
		} else {
			mode = "agent"
		}
	}

	workDir, _ := params["work_dir"].(string)

	maxIter := 50
	if v, ok := params["max_iterations"].(float64); ok && v > 0 {
		maxIter = int(v)
	}

	extra, _ := params["system_prompt_extra"].(string)

	var toolFilter []string
	if ts, ok := params["tools"].([]any); ok {
		for _, tt := range ts {
			if s, ok := tt.(string); ok {
				toolFilter = append(toolFilter, s)
			}
		}
	}

	// Extract parent agent ID from context (injected by executeTool)
	parentID, _ := AgentIDFromContext(ctx)

	// Extract parent's event channel from context (injected by executeTool)
	parentEventCh, _ := EventChanFromContext(ctx)

	// Apply per-agent timeout from default policy, tied to the parent run context.
	policy := DefaultSubAgentPolicy()
	parentRunCtx, ok := ParentRunContextFromContext(ctx)
	if !ok || parentRunCtx == nil {
		parentRunCtx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(parentRunCtx, policy.TimeoutPerAgent)

	a, err := t.manager.Create(AgentOptions{
		ParentID:          parentID,
		Mode:              mode,
		WorkDir:           workDir,
		Tools:             toolFilter,
		SystemPromptExtra: extra,
		MaxIterations:     maxIter,
	})
	if err != nil {
		cancel()
		return tools.ToolResult{}, fmt.Errorf("create sub-agent: %w", err)
	}
	t.manager.MarkRunning(a.ID())
	t.manager.SetCancel(a.ID(), cancel)

	// Start the sub-agent asynchronously, forward events to parent
	go func() {
		defer func() {
			cancel()
			t.manager.SetCancel(a.ID(), nil)
		}()
		ch := a.Run(runCtx, buildSubAgentTask(task))
		for e := range ch {
			// Forward approval events to parent so the UI can handle them
			if e.Type == agentpkg.EventToolApprovalRequest && parentEventCh != nil {
				_ = sendParentEvent(runCtx, parentEventCh, Event{
					Type:         EventToolApprovalRequest,
					AgentID:      a.ID(),
					ApprovalID:   e.ApprovalID,
					ApprovalTool: e.ApprovalTool,
					ApprovalArgs: e.ApprovalArgs,
				})
			}
			switch e.Type {
			case agentpkg.EventDone:
				t.manager.MarkDone(a.ID(), lastAssistantResponse(a))
			case agentpkg.EventError:
				t.manager.MarkError(a.ID(), e.Error)
			}
		}
		if runCtx.Err() != nil {
			if st, ok := t.manager.Status(a.ID()); !ok || st.State != "done" {
				t.manager.MarkError(a.ID(), runCtx.Err())
			}
		}
	}()

	result := map[string]any{
		"handle":  string(a.ID()),
		"status":  "running",
		"timeout": policy.TimeoutPerAgent.String(),
	}
	data, _ := json.Marshal(result)
	return tools.NewTextToolResult(string(data)), nil
}

func sendParentEvent(ctx context.Context, ch chan<- Event, ev Event) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[agent] sendParentEvent recovered from panic: %v (event type=%d)", r, ev.Type)
			ok = false
		}
	}()
	select {
	case ch <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

// SubAgentStatusTool queries sub-agent status and results.
type SubAgentStatusTool struct {
	manager *AgentManager
}

func NewSubAgentStatusTool(m *AgentManager) *SubAgentStatusTool {
	return &SubAgentStatusTool{manager: m}
}

func (t *SubAgentStatusTool) Name() string { return "subagent_status" }
func (t *SubAgentStatusTool) Description() string {
	return "Query the status and results of a sub-agent."
}
func (t *SubAgentStatusTool) PromptSnippet() string      { return "Check sub-agent status and get results" }
func (t *SubAgentStatusTool) PromptGuidelines() []string { return nil }

func (t *SubAgentStatusTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"handle": {"type": "string", "description": "The sub-agent handle ID"}
		},
		"required": ["handle"]
	}`)
}

func (t *SubAgentStatusTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	handle, _ := params["handle"].(string)
	if handle == "" {
		return tools.ToolResult{}, fmt.Errorf("handle is required")
	}

	st, statusOK := t.manager.Status(agentpkg.AgentID(handle))
	a, agentOK := t.manager.Get(agentpkg.AgentID(handle))
	if !statusOK && !agentOK {
		return tools.ToolResult{}, fmt.Errorf("sub-agent %q not found", handle)
	}

	status := st.State
	if status == "" {
		status = "unknown"
	}
	lastResponse := st.Result
	messageCount := 0
	if agentOK {
		messages := a.GetMessages()
		messageCount = len(messages)
	}
	if lastResponse == "" && agentOK {
		lastResponse = lastAssistantResponse(a)
	}

	result := map[string]any{
		"handle":        handle,
		"status":        status,
		"message_count": messageCount,
	}
	if lastResponse != "" {
		result["last_response"] = lastResponse
	}
	if st.Error != "" {
		result["error"] = st.Error
	}
	if !st.UpdatedAt.IsZero() {
		result["updated_at"] = st.UpdatedAt.Format(time.RFC3339)
	}

	data, _ := json.Marshal(result)
	return tools.NewTextToolResult(string(data)), nil
}

// SubAgentSendTool sends a follow-up message to a running sub-agent.
type SubAgentSendTool struct {
	manager *AgentManager
}

func NewSubAgentSendTool(m *AgentManager) *SubAgentSendTool {
	return &SubAgentSendTool{manager: m}
}

func (t *SubAgentSendTool) Name() string { return "subagent_send" }
func (t *SubAgentSendTool) Description() string {
	return "Send a follow-up message to a running sub-agent."
}
func (t *SubAgentSendTool) PromptSnippet() string {
	return "Send follow-up instructions to a sub-agent"
}
func (t *SubAgentSendTool) PromptGuidelines() []string { return nil }

func (t *SubAgentSendTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"handle": {"type": "string", "description": "The sub-agent handle ID"},
			"message": {"type": "string", "description": "The follow-up message"}
		},
		"required": ["handle", "message"]
	}`)
}

func (t *SubAgentSendTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	handle, _ := params["handle"].(string)
	message, _ := params["message"].(string)
	if handle == "" || message == "" {
		return tools.ToolResult{}, fmt.Errorf("handle and message are required")
	}

	a, ok := t.manager.Get(agentpkg.AgentID(handle))
	if !ok {
		return tools.ToolResult{}, fmt.Errorf("sub-agent %q not found", handle)
	}

	// Apply per-agent timeout for follow-up messages too
	policy := DefaultSubAgentPolicy()
	parentRunCtx, ok := ParentRunContextFromContext(ctx)
	if !ok || parentRunCtx == nil {
		parentRunCtx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(parentRunCtx, policy.TimeoutPerAgent)
	t.manager.MarkRunning(a.ID())
	t.manager.SetCancel(a.ID(), cancel)

	// Extract parent's event channel for approval forwarding
	parentEventCh, _ := EventChanFromContext(ctx)

	go func() {
		defer func() {
			cancel()
			t.manager.SetCancel(a.ID(), nil)
		}()
		ch := a.Run(runCtx, message)
		for e := range ch {
			// Forward approval events to parent
			if e.Type == agentpkg.EventToolApprovalRequest && parentEventCh != nil {
				_ = sendParentEvent(runCtx, parentEventCh, Event{
					Type:         EventToolApprovalRequest,
					AgentID:      a.ID(),
					ApprovalID:   e.ApprovalID,
					ApprovalTool: e.ApprovalTool,
					ApprovalArgs: e.ApprovalArgs,
				})
			}
			switch e.Type {
			case agentpkg.EventDone:
				t.manager.MarkDone(a.ID(), lastAssistantResponse(a))
			case agentpkg.EventError:
				t.manager.MarkError(a.ID(), e.Error)
			}
		}
		if runCtx.Err() != nil {
			if st, ok := t.manager.Status(a.ID()); !ok || st.State != "done" {
				t.manager.MarkError(a.ID(), runCtx.Err())
			}
		}
	}()

	return tools.NewTextToolResult(fmt.Sprintf(`{"handle":%q,"status":"message_sent"}`, handle)), nil
}

func buildSubAgentTask(task string) string {
	task = strings.TrimSpace(task)
	return fmt.Sprintf(`Delegated task:
%s

Execute this task precisely. When done, structure your final response using this format:

Result: <the direct answer or completed change>
Evidence: <files inspected, commands run, test outputs — summarized>
Changes: <files modified with brief description, or "None">
Risks: <assumptions, uncertainty, follow-up needed, or "None">
`, task)
}

func lastAssistantResponse(a agentpkg.Agent) string {
	messages := a.GetMessages()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == agentpkg.RoleAssistant {
			if messages[i].Content != "" {
				return messages[i].Content
			}
			var sb strings.Builder
			for _, block := range messages[i].Contents {
				if block.Type == "text" && block.Text != "" {
					sb.WriteString(block.Text)
				}
			}
			return sb.String()
		}
	}
	return ""
}

// SubAgentDestroyTool destroys a sub-agent and releases resources.
type SubAgentDestroyTool struct {
	manager *AgentManager
}

func NewSubAgentDestroyTool(m *AgentManager) *SubAgentDestroyTool {
	return &SubAgentDestroyTool{manager: m}
}

func (t *SubAgentDestroyTool) Name() string { return "subagent_destroy" }
func (t *SubAgentDestroyTool) Description() string {
	return "Destroy a sub-agent and release resources."
}
func (t *SubAgentDestroyTool) PromptSnippet() string      { return "Destroy a finished sub-agent" }
func (t *SubAgentDestroyTool) PromptGuidelines() []string { return nil }

func (t *SubAgentDestroyTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"handle": {"type": "string", "description": "The sub-agent handle ID"}
		},
		"required": ["handle"]
	}`)
}

func (t *SubAgentDestroyTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	handle, _ := params["handle"].(string)
	if handle == "" {
		return tools.ToolResult{}, fmt.Errorf("handle is required")
	}

	if err := t.manager.Destroy(agentpkg.AgentID(handle)); err != nil {
		return tools.ToolResult{}, fmt.Errorf("destroy sub-agent: %w", err)
	}

	return tools.NewTextToolResult(fmt.Sprintf(`{"handle":%q,"status":"destroyed"}`, handle)), nil
}

// SubAgentPolicy defines security constraints for sub-agents.
type SubAgentPolicy struct {
	MaxChildren     int           // Maximum number of sub-agents (default 5)
	AllowedModes    []string      // Allowed modes for sub-agents (default ["agent"])
	InheritSandbox  bool          // Inherit parent's sandbox (default true)
	TimeoutPerAgent time.Duration // Per-agent timeout (default 10min)
	TotalTimeout    time.Duration // Total timeout for all sub-agents (default 30min)
}

// DefaultSubAgentPolicy returns the default policy.
func DefaultSubAgentPolicy() SubAgentPolicy {
	return SubAgentPolicy{
		MaxChildren:     5,
		AllowedModes:    []string{"plan", "agent", "yolo"},
		InheritSandbox:  true,
		TimeoutPerAgent: 10 * time.Minute,
		TotalTimeout:    30 * time.Minute,
	}
}

// Validate checks if a sub-agent creation request is allowed.
func (p *SubAgentPolicy) Validate(parentID string, mode string, currentChildCount int) error {
	if parentID == "" {
		return nil
	}
	if currentChildCount >= p.MaxChildren {
		return fmt.Errorf("maximum %d sub-agents allowed", p.MaxChildren)
	}
	allowed := false
	for _, m := range p.AllowedModes {
		if m == mode {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("mode %q is not allowed for sub-agents; allowed: %v", mode, p.AllowedModes)
	}
	return nil
}
