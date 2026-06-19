package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	internalagent "github.com/startvibecoding/vibecoding/internal/agent"
)

// AgentHost runs workflow tasks through the existing AgentManager.
type AgentHost struct {
	Manager       *internalagent.AgentManager
	ParentID      agentpkg.AgentID
	ParentMode    string
	ParentEventCh chan<- internalagent.Event
	ParentRunCtx  context.Context
}

func (h *AgentHost) RunAgent(ctx context.Context, task AgentTask) (AgentResult, error) {
	if h.Manager == nil {
		return AgentResult{}, fmt.Errorf("agent manager is not initialized")
	}
	mode := task.Mode
	if mode == "" {
		mode = h.ParentMode
	}
	if mode == "" {
		mode = "agent"
	}
	maxIter := task.MaxIterations
	if maxIter <= 0 {
		maxIter = 50
	}

	multiAgent := false
	delegateMode := false
	workflows := false
	runCtx := ctx
	if h.ParentRunCtx != nil {
		var cancel context.CancelFunc
		runCtx, cancel = contextWithEitherCancel(ctx, h.ParentRunCtx)
		defer cancel()
	}

	a, err := h.Manager.Create(internalagent.AgentOptions{
		ParentID:          h.ParentID,
		Mode:              mode,
		WorkDir:           task.WorkDir,
		Tools:             task.Tools,
		SystemPromptExtra: task.SystemPromptExtra,
		MaxIterations:     maxIter,
		MultiAgent:        &multiAgent,
		DelegateMode:      &delegateMode,
		Workflows:         &workflows,
	})
	if err != nil {
		return AgentResult{}, fmt.Errorf("create workflow worker: %w", err)
	}
	defer func() { _ = h.Manager.Destroy(a.ID()) }()

	started := time.Now()
	h.Manager.MarkRunning(a.ID())
	var runErr error
	completed := false
	for ev := range a.Run(runCtx, buildTaskPrompt(task)) {
		if ev.Type == agentpkg.EventToolApprovalRequest && h.ParentEventCh != nil {
			_ = sendParentEvent(runCtx, h.ParentEventCh, internalagent.Event{
				Type:         internalagent.EventToolApprovalRequest,
				AgentID:      a.ID(),
				ApprovalID:   ev.ApprovalID,
				ApprovalTool: ev.ApprovalTool,
				ApprovalArgs: ev.ApprovalArgs,
			})
		}
		internalagent.ForwardChildAgentEvent(runCtx, h.ParentEventCh, a.ID(), ev)
		switch ev.Type {
		case agentpkg.EventDone:
			completed = true
			h.Manager.MarkDone(a.ID(), lastAssistantResponse(a))
		case agentpkg.EventError:
			completed = true
			runErr = ev.Error
			h.Manager.MarkError(a.ID(), ev.Error)
		}
	}
	if !completed && runCtx.Err() != nil {
		runErr = runCtx.Err()
		h.Manager.MarkError(a.ID(), runErr)
	}

	result := AgentResult{
		Name:       task.Name,
		Phase:      task.Phase,
		Status:     StatusDone,
		Result:     lastAssistantResponse(a),
		StartedAt:  started,
		FinishedAt: time.Now(),
	}
	if runErr != nil {
		result.Status = StatusError
		result.Error = runErr.Error()
		return result, runErr
	}
	return result, nil
}

func buildTaskPrompt(task AgentTask) string {
	return fmt.Sprintf(`Workflow task: %s
Phase: %s

%s

Return a concise final result with evidence and risks.`, task.Name, task.Phase, strings.TrimSpace(task.Prompt))
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

func sendParentEvent(ctx context.Context, ch chan<- internalagent.Event, ev internalagent.Event) (ok bool) {
	defer func() {
		if recover() != nil {
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

func contextWithEitherCancel(a context.Context, b context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(a)
	go func() {
		select {
		case <-b.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}
