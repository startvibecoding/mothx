package remotetui

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/websocket"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/tools"
)

type wsEvent struct {
	Type             string         `json:"type"`
	Content          string         `json:"content,omitempty"`
	SessionID        string         `json:"session_id,omitempty"`
	Version          string         `json:"version,omitempty"`
	Model            string         `json:"model,omitempty"`
	WorkDir          string         `json:"work_dir,omitempty"`
	Tool             string         `json:"tool,omitempty"`
	CallID           string         `json:"call_id,omitempty"`
	Args             map[string]any `json:"args,omitempty"`
	Result           string         `json:"result,omitempty"`
	Path             string         `json:"path,omitempty"`
	Diff             string         `json:"diff,omitempty"`
	ApprovalID       string         `json:"approval_id,omitempty"`
	ApprovalTool     string         `json:"approval_tool,omitempty"`
	ApprovalArgs     map[string]any `json:"approval_args,omitempty"`
	QuestionID       string         `json:"question_id,omitempty"`
	Question         string         `json:"question,omitempty"`
	QuestionOptions  []string       `json:"question_options,omitempty"`
	QuestionContext  string         `json:"question_context,omitempty"`
	StatusMessage    string         `json:"status_message,omitempty"`
	StopReason       string         `json:"stop_reason,omitempty"`
	Message          string         `json:"message,omitempty"`
	Code             string         `json:"code,omitempty"`
	PromptTokens     int            `json:"prompt_tokens,omitempty"`
	CompletionTokens int            `json:"completion_tokens,omitempty"`
	TotalTokens      int            `json:"total_tokens,omitempty"`
	CacheReadTokens  int            `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens int            `json:"cache_write_tokens,omitempty"`
	Plan             *wsPlanData    `json:"plan,omitempty"`
}

type wsPlanData struct {
	Title string       `json:"title"`
	Steps []wsPlanStep `json:"steps"`
}

type wsPlanStep struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

type wsMessage struct {
	Type       string `json:"type"`
	Content    string `json:"content,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
	Approved   bool   `json:"approved,omitempty"`
	QuestionID string `json:"question_id,omitempty"`
	Answer     string `json:"answer,omitempty"`
}

func (a *App) connectWS() tea.Cmd {
	return func() tea.Msg {
		wsCfg, err := websocket.NewConfig(a.wsURL, "http://localhost/")
		if err != nil {
			return agentDoneMsg{err: fmt.Errorf("websocket config: %w", err)}
		}
		if a.authToken != "" {
			if wsCfg.Header == nil {
				wsCfg.Header = http.Header{}
			}
			wsCfg.Header.Set("Authorization", "Bearer "+a.authToken)
		}
		ws, err := websocket.DialConfig(wsCfg)
		if err != nil {
			return agentDoneMsg{err: fmt.Errorf("connect: %w", err)}
		}
		a.ws = ws
		return agentEventMsg{event: agent.Event{
			Type:          agent.EventStatus,
			StatusMessage: "✓ Connected to Hermes gateway",
		}}
	}
}

func (a *App) sendWS(msg wsMessage) error {
	if a.ws == nil {
		return fmt.Errorf("websocket is not connected")
	}
	return websocket.JSON.Send(a.ws, msg)
}

func (a *App) closeWS() {
	if a.ws != nil {
		a.ws.Close()
		a.ws = nil
	}
}

func (ev wsEvent) agentEvent() agent.Event {
	switch ev.Type {
	case "connected":
		msg := "✓ Connected"
		if ev.SessionID != "" {
			msg += " (session: " + ev.SessionID + ")"
		}
		if ev.Version != "" {
			msg += " version: " + ev.Version
		}
		return agent.Event{Type: agent.EventStatus, StatusMessage: msg}
	case "text_delta":
		return agent.Event{Type: agent.EventTextDelta, TextDelta: ev.Content}
	case "think_delta":
		return agent.Event{Type: agent.EventThinkDelta, ThinkDelta: ev.Content}
	case "tool_call":
		args, _ := json.Marshal(ev.Args)
		return agent.Event{
			Type:       agent.EventToolCall,
			ToolCallID: ev.CallID,
			ToolName:   ev.Tool,
			ToolArgs:   ev.Args,
			ToolCall: &provider.ToolCallBlock{
				ID:        ev.CallID,
				Name:      ev.Tool,
				Arguments: args,
			},
		}
	case "tool_result", "tool_diff":
		out := ev.Result
		if out == "" {
			out = ev.Message
		}
		event := agent.Event{
			Type:       agent.EventToolResult,
			ToolCallID: ev.CallID,
			ToolName:   ev.Tool,
			ToolResult: out,
		}
		if ev.Type == "tool_diff" || ev.Diff != "" {
			event.ToolDiff = fileDiffFromUnified(ev.Path, ev.Diff)
		}
		return event
	case "approval_request":
		return agent.Event{
			Type:         agent.EventToolApprovalRequest,
			ApprovalID:   ev.ApprovalID,
			ApprovalTool: ev.ApprovalTool,
			ApprovalArgs: ev.ApprovalArgs,
		}
	case "question_request":
		return agent.Event{
			Type:            agent.EventQuestionRequest,
			QuestionID:      ev.QuestionID,
			QuestionText:    ev.Question,
			QuestionOptions: ev.QuestionOptions,
			QuestionContext: ev.QuestionContext,
		}
	case "plan_update":
		return agent.Event{Type: agent.EventPlanUpdate, Plan: ev.taskPlan()}
	case "status", "command_result":
		return agent.Event{Type: agent.EventStatus, StatusMessage: firstNonEmpty(ev.StatusMessage, ev.Message)}
	case "compaction_start":
		return agent.Event{Type: agent.EventCompactionStart}
	case "compaction_end":
		return agent.Event{Type: agent.EventCompactionEnd, StatusMessage: ev.StatusMessage}
	case "usage":
		return agent.Event{Type: agent.EventUsage, Usage: &provider.Usage{
			Input:       ev.PromptTokens,
			Output:      ev.CompletionTokens,
			CacheRead:   ev.CacheReadTokens,
			CacheWrite:  ev.CacheWriteTokens,
			TotalTokens: ev.TotalTokens,
		}}
	case "message_start":
		return agent.Event{Type: agent.EventStatus}
	case "done":
		return agent.Event{Type: agent.EventDone, Done: true, StopReason: ev.StopReason}
	case "error":
		return agent.Event{Type: agent.EventError, Error: errors.New(firstNonEmpty(ev.Message, ev.StatusMessage, ev.Code))}
	default:
		return agent.Event{Type: agent.EventStatus, StatusMessage: firstNonEmpty(ev.StatusMessage, ev.Message)}
	}
}

func (ev wsEvent) taskPlan() *tools.TaskPlan {
	if ev.Plan == nil {
		return nil
	}
	plan := &tools.TaskPlan{Title: ev.Plan.Title}
	for _, step := range ev.Plan.Steps {
		plan.Steps = append(plan.Steps, tools.PlanStep{
			Title:  step.Title,
			Status: step.Status,
		})
	}
	return plan
}

func fileDiffFromUnified(path, unified string) *tools.FileDiff {
	if unified == "" {
		return nil
	}
	diff := &tools.FileDiff{Path: path, Unified: unified}
	for _, line := range strings.Split(unified, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			continue
		case strings.HasPrefix(line, "+"):
			diff.Added++
		case strings.HasPrefix(line, "-"):
			diff.Deleted++
		}
	}
	return diff
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// Run starts the remote TUI application.
func Run(opts Options) error {
	app := NewApp(opts)
	p := tea.NewProgram(app)
	app.SetProgram(p)
	_, err := p.Run()
	return err
}
