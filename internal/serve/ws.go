package serve

import (
	"context"
	"fmt"
	"os"

	"github.com/startvibecoding/mothx/internal/agent"
	channels "github.com/startvibecoding/mothx/internal/serve/channels"
	wsruntime "github.com/startvibecoding/mothx/internal/serve/ws"
	"github.com/startvibecoding/mothx/internal/util"
)

func (rt *channelRuntime) setupWebSocketRuntime(version string) {
	if rt == nil || rt.cfg == nil || !rt.cfg.Features.WebSocket {
		fmt.Fprintf(os.Stderr, "  WebSocket: disabled\n")
		return
	}
	if rt.dispatcher == nil {
		fmt.Fprintf(os.Stderr, "  WebSocket: enabled but dispatcher not ready\n")
		return
	}

	gw := wsruntime.NewRuntime("", "", version)
	gw.SetDispatcher(&serveWSDispatcherAdapter{dispatcher: rt.dispatcher})
	gw.SetClientInfo(rt.cfg.API.Model, rt.cfg.API.GetWorkDir())
	rt.wsRuntime = gw
	fmt.Fprintf(os.Stderr, "  WebSocket: enabled at /ws\n")
}

type serveWSDispatcherAdapter struct {
	dispatcher *channels.Dispatcher
}

func (a *serveWSDispatcherAdapter) HandleWSMessage(ctx context.Context, connID, text string, eventCh chan<- wsruntime.WSEvent) error {
	if a == nil || a.dispatcher == nil {
		eventCh <- wsruntime.WSEvent{Type: "error", Message: "dispatcher not ready"}
		return nil
	}

	agentCh := make(chan agent.Event, 100)
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.dispatcher.HandleWSMessage(ctx, connID, text, agentCh)
	}()

	for {
		select {
		case ev, ok := <-agentCh:
			if !ok {
				if err := <-errCh; err != nil {
					eventCh <- wsruntime.WSEvent{Type: "error", Message: err.Error()}
				}
				return nil
			}
			if out := serveAgentEventToWSEvent(ev); out.Type != "" {
				eventCh <- out
			}
		case err := <-errCh:
			for {
				select {
				case ev, ok := <-agentCh:
					if !ok {
						if err != nil {
							eventCh <- wsruntime.WSEvent{Type: "error", Message: err.Error()}
						}
						return nil
					}
					if out := serveAgentEventToWSEvent(ev); out.Type != "" {
						eventCh <- out
					}
				default:
					if err != nil {
						eventCh <- wsruntime.WSEvent{Type: "error", Message: err.Error()}
					}
					return nil
				}
			}
		}
	}
}

func (a *serveWSDispatcherAdapter) ListSessions() []wsruntime.SessionInfo {
	if a == nil || a.dispatcher == nil {
		return nil
	}
	sessions := a.dispatcher.ListSessions()
	result := make([]wsruntime.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		msgs := s.Manager.GetMessages()
		preview := ""
		for _, m := range msgs {
			if m.Role == "user" {
				preview = m.Content
				preview = util.TruncateWithSuffix(preview, 60, "...")
				break
			}
		}
		result = append(result, wsruntime.SessionInfo{
			ID:           s.ID,
			Platform:     s.Platform,
			UserID:       s.UserID,
			WorkDir:      s.WorkDir,
			Mode:         s.Mode,
			MessageCount: len(msgs),
			LastActive:   s.LastUsed,
			Preview:      preview,
		})
	}
	return result
}

func (a *serveWSDispatcherAdapter) RemoveSession(key string) {
	if a != nil && a.dispatcher != nil {
		a.dispatcher.RemoveSession(key)
	}
}

func (a *serveWSDispatcherAdapter) ResolveApproval(approvalID string, approved bool) bool {
	if a == nil || a.dispatcher == nil {
		return false
	}
	return a.dispatcher.ResolveApproval(approvalID, approved)
}

func (a *serveWSDispatcherAdapter) ResolveQuestion(questionID, answer string) bool {
	if a == nil || a.dispatcher == nil {
		return false
	}
	return a.dispatcher.ResolveQuestion(questionID, answer)
}

func serveAgentEventToWSEvent(ev agent.Event) wsruntime.WSEvent {
	switch ev.Type {
	case agent.EventTextDelta:
		return wsruntime.WSEvent{Type: "text_delta", Content: ev.TextDelta}
	case agent.EventThinkDelta:
		return wsruntime.WSEvent{Type: "think_delta", Content: ev.ThinkDelta}
	case agent.EventToolCall:
		out := wsruntime.WSEvent{
			Type:   "tool_call",
			Tool:   ev.ToolName,
			CallID: ev.ToolCallID,
			Args:   ev.ToolArgs,
		}
		if ev.ToolCall != nil {
			out.Tool = ev.ToolCall.Name
			out.CallID = ev.ToolCall.ID
		}
		return out
	case agent.EventToolExecutionEnd:
		name := ev.ToolName
		if name == "" && ev.ToolCall != nil {
			name = ev.ToolCall.Name
		}
		out := wsruntime.WSEvent{
			Type:   "tool_result",
			Tool:   name,
			CallID: ev.ToolCallID,
			Result: ev.ToolResult,
		}
		if ev.ToolError != nil {
			out.Code = "error"
			out.Message = ev.ToolError.Error()
		}
		if ev.ToolDiff != nil {
			out.Type = "tool_diff"
			out.Path = ev.ToolDiff.Path
			out.Diff = ev.ToolDiff.Unified
		}
		return out
	case agent.EventContextPressure, agent.EventBudgetPressure:
		return wsruntime.WSEvent{Type: "status", StatusMessage: ev.PressureMessage}
	case agent.EventToolApprovalRequest:
		return wsruntime.WSEvent{
			Type:         "approval_request",
			ApprovalID:   ev.ApprovalID,
			ApprovalTool: ev.ApprovalTool,
			ApprovalArgs: ev.ApprovalArgs,
		}
	case agent.EventQuestionRequest:
		return wsruntime.WSEvent{
			Type:            "question_request",
			QuestionID:      ev.QuestionID,
			Question:        ev.QuestionText,
			QuestionOptions: ev.QuestionOptions,
			QuestionContext: ev.QuestionContext,
		}
	case agent.EventPlanUpdate:
		var plan *wsruntime.PlanData
		if ev.Plan != nil {
			steps := make([]wsruntime.PlanStep, len(ev.Plan.Steps))
			for i, step := range ev.Plan.Steps {
				steps[i] = wsruntime.PlanStep{Title: step.Title, Status: step.Status}
			}
			plan = &wsruntime.PlanData{Title: ev.Plan.Title, Steps: steps}
		}
		return wsruntime.WSEvent{Type: "plan_update", Plan: plan}
	case agent.EventDone:
		return wsruntime.WSEvent{Type: "done", StopReason: ev.StopReason}
	case agent.EventStatus:
		return wsruntime.WSEvent{Type: "status", StatusMessage: ev.StatusMessage}
	case agent.EventCompactionStart:
		return wsruntime.WSEvent{Type: "compaction_start", StatusMessage: "Compacting context..."}
	case agent.EventCompactionEnd:
		msg := "Context compacted"
		if ev.Error != nil {
			msg = "Compaction failed: " + ev.Error.Error()
		} else if ev.StatusMessage != "" {
			msg = ev.StatusMessage
		}
		return wsruntime.WSEvent{Type: "compaction_end", StatusMessage: msg}
	case agent.EventError:
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Error()
		}
		return wsruntime.WSEvent{Type: "error", Message: msg, Code: ev.StopReason}
	case agent.EventUsage:
		out := wsruntime.WSEvent{Type: "usage"}
		if ev.Usage != nil {
			out.PromptTokens = ev.Usage.PromptTokens()
			out.CompletionTokens = ev.Usage.Output
			out.TotalTokens = ev.Usage.TotalTokens
			out.CacheReadTokens = ev.Usage.CacheRead
			out.CacheWriteTokens = ev.Usage.CacheWrite
		}
		return out
	case agent.EventMessageStart:
		if ev.Message.Role == "user" && ev.Message.Content != "" {
			return wsruntime.WSEvent{Type: "message_start", Content: ev.Message.Content}
		}
	}
	return wsruntime.WSEvent{}
}
