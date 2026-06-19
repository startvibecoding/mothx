package tui

import (
	"fmt"
	"strings"
	"time"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/agent"
)

const maxActivityLines = 200

type activityLine struct {
	Time time.Time
	Text string
}

type agentActivity struct {
	AgentID    agentpkg.AgentID
	Kind       string
	State      string
	LastThink  string
	LastText   string
	LastTool   string
	LastResult string
	UpdatedAt  time.Time
	Events     []activityLine
}

func (a *App) isBackgroundAgentEvent(event agent.Event) bool {
	if event.AgentID == "" {
		return false
	}
	if a.agent != nil && event.AgentID == a.agent.ID() {
		return false
	}
	switch event.Type {
	case agent.EventToolApprovalRequest,
		agent.EventQuestionRequest,
		agent.EventToolApprovalResponse,
		agent.EventQuestionResponse:
		return false
	}
	return true
}

func (a *App) recordAgentActivity(event agent.Event) {
	if event.AgentID == "" {
		return
	}
	if a.agentActivities == nil {
		a.agentActivities = make(map[agentpkg.AgentID]*agentActivity)
	}
	act := a.agentActivities[event.AgentID]
	if act == nil {
		act = &agentActivity{
			AgentID: event.AgentID,
			Kind:    "subagent",
			State:   "running",
		}
		if strings.HasPrefix(string(event.AgentID), "workflow:") {
			act.Kind = "workflow"
		}
		a.agentActivities[event.AgentID] = act
		a.agentActivityOrder = appendUniqueActivityID(a.agentActivityOrder, event.AgentID)
	}

	now := time.Now()
	act.UpdatedAt = now
	switch event.Type {
	case agent.EventStatus:
		if event.StatusMessage != "" {
			act.LastResult = truncatePlain(event.StatusMessage, 160)
			act.Events = appendActivityLine(act.Events, now, event.StatusMessage)
		}
	case agent.EventThinkDelta:
		act.State = "running"
		act.LastThink = truncatePlain(act.LastThink+event.ThinkDelta, 240)
	case agent.EventTextDelta:
		act.State = "running"
		act.LastText = truncatePlain(act.LastText+event.TextDelta, 320)
	case agent.EventToolCall, agent.EventToolExecutionStart:
		act.State = "running"
		name := event.ToolName
		if name == "" && event.ToolCall != nil {
			name = event.ToolCall.Name
		}
		if name != "" {
			act.LastTool = formatActivityTool(name, event.ToolArgs)
			act.Events = appendActivityLine(act.Events, now, "tool: "+act.LastTool)
		}
	case agent.EventToolResult, agent.EventToolExecutionEnd:
		name := event.ToolName
		if name == "" && event.ToolCall != nil {
			name = event.ToolCall.Name
		}
		result := strings.TrimSpace(event.ToolResult)
		if event.ToolError != nil {
			act.State = "error"
			result = event.ToolError.Error()
		}
		if result != "" {
			act.LastResult = truncatePlain(result, 320)
		}
		if name != "" || result != "" {
			line := truncatePlain(result, 140)
			if name != "" && line != "" {
				line = name + ": " + line
			} else if name != "" {
				line = name
			}
			act.Events = appendActivityLine(act.Events, now, "result: "+line)
		}
	case agent.EventDone:
		act.State = "done"
		act.Events = appendActivityLine(act.Events, now, "done")
	case agent.EventError:
		act.State = "error"
		if event.Error != nil {
			act.LastResult = truncatePlain(event.Error.Error(), 320)
			act.Events = appendActivityLine(act.Events, now, "error: "+event.Error.Error())
		}
	}
}

func appendUniqueActivityID(ids []agentpkg.AgentID, id agentpkg.AgentID) []agentpkg.AgentID {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

func appendActivityLine(lines []activityLine, t time.Time, text string) []activityLine {
	text = strings.TrimSpace(text)
	if text == "" {
		return lines
	}
	lines = append(lines, activityLine{Time: t, Text: truncatePlain(text, 500)})
	if len(lines) > maxActivityLines {
		lines = lines[len(lines)-maxActivityLines:]
	}
	return lines
}

func formatActivityTool(name string, args map[string]any) string {
	if len(args) == 0 {
		return name
	}
	var parts []string
	for _, key := range []string{"path", "cmd", "query", "pattern", "handle", "message", "source", "task"} {
		if v, ok := args[key]; ok {
			parts = append(parts, fmt.Sprintf("%s=%q", key, truncatePlain(fmt.Sprint(v), 80)))
		}
	}
	if len(parts) == 0 {
		return name
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
}

func truncatePlain(s string, max int) string {
	s = strings.TrimSpace(strings.Join(strings.Fields(s), " "))
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

func (a *App) renderActivitySummary(width int) string {
	if len(a.agentActivityOrder) == 0 {
		return ""
	}
	limit := 4
	var lines []string
	for i := len(a.agentActivityOrder) - 1; i >= 0 && len(lines) < limit; i-- {
		id := a.agentActivityOrder[i]
		act := a.agentActivities[id]
		if act == nil {
			continue
		}
		detail := act.LastTool
		if detail == "" {
			detail = act.LastResult
		}
		if detail == "" {
			detail = act.LastText
		}
		if detail == "" {
			detail = act.LastThink
		}
		state := act.State
		if state == "" {
			state = "running"
		}
		line := fmt.Sprintf("%s [%s]", id, state)
		if detail != "" {
			line += " " + detail
		}
		if width > 0 {
			line = truncatePlain(line, width-2)
		}
		lines = append(lines, statusStyle.Render(line))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
