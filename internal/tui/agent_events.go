package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/tools"
)

func (a *App) handleAgentEvent(event agent.Event) tea.Cmd {
	if a.isBackgroundAgentEvent(event) {
		a.recordAgentActivity(event)
		if event.Type == agent.EventStatus {
			a.refreshESMPanel()
		}
		a.scheduleRender()
		return a.listenAgentEvents()
	}

	switch event.Type {
	case agent.EventTextDelta:
		a.invalidateToolModalCache()
		if a.currentAssistantIdx >= 0 && a.currentAssistantIdx < len(a.messages) {
			a.appendAssistantDelta(a.currentAssistantIdx, event.TextDelta)
		} else {
			a.currentAssistantIdx = len(a.messages)
			a.assistantRaw[a.currentAssistantIdx] = ""
			a.appendAssistantDelta(a.currentAssistantIdx, event.TextDelta)
			// placeholder; actual display is built in updateViewportContent
			a.messages = append(a.messages, "")
		}
		a.assistantDirty[a.currentAssistantIdx] = true
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventThinkDelta:
		a.invalidateToolModalCache()
		if a.thinkRaw == nil {
			a.thinkRaw = make(map[int]string)
		}
		if a.currentThinkIdx >= 0 && a.currentThinkIdx < len(a.messages) {
			a.appendThinkDelta(a.currentThinkIdx, event.ThinkDelta)
		} else {
			if a.currentAssistantIdx >= 0 &&
				a.currentAssistantIdx == len(a.messages)-1 &&
				a.assistantRaw[a.currentAssistantIdx] == "" {
				a.currentThinkIdx = a.currentAssistantIdx
				delete(a.assistantRaw, a.currentAssistantIdx)
				delete(a.assistantBuilders, a.currentAssistantIdx)
				delete(a.assistantRendered, a.currentAssistantIdx)
				delete(a.assistantDirty, a.currentAssistantIdx)
				a.currentAssistantIdx = len(a.messages)
				a.assistantRaw[a.currentAssistantIdx] = ""
				a.assistantDirty[a.currentAssistantIdx] = true
				a.messages = append(a.messages, "")
			} else {
				a.currentThinkIdx = len(a.messages)
				a.messages = append(a.messages, "")
			}
			a.thinkRaw[a.currentThinkIdx] = ""
			a.appendThinkDelta(a.currentThinkIdx, event.ThinkDelta)
		}
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventTurnStart:
		a.invalidateToolModalCache()
		// Reserve display slots before streaming deltas arrive so later tool output
		// cannot shift the assistant message index underneath us.
		a.currentAssistantIdx = len(a.messages)
		a.assistantRaw[a.currentAssistantIdx] = ""
		a.messages = append(a.messages, "")
		return a.listenAgentEvents()

	case agent.EventToolCall:
		if event.ToolCall != nil {
			a.appendToolExecutionStart(event.ToolCall.ID, event.ToolCall.Name, event.ToolArgs)
		}
		return a.listenAgentEvents()

	case agent.EventToolExecutionStart:
		a.appendToolExecutionStart(event.ToolCallID, event.ToolName, event.ToolArgs)
		return a.listenAgentEvents()

	case agent.EventToolExecutionEnd:
		a.appendToolResult(event)
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventToolResult:
		a.appendToolResult(event)
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventPlanUpdate:
		a.currentPlan = event.Plan
		a.addMessage(statusStyle.Render(formatPlanForDisplay(event.Plan)))
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventToolApprovalRequest:
		a.commitActiveStream()
		nextApproval := pendingApproval{
			agentID:    event.AgentID,
			approvalID: event.ApprovalID,
			toolName:   event.ApprovalTool,
			args:       event.ApprovalArgs,
		}
		if a.hasPendingApproval(nextApproval) {
			a.scheduleRender()
			return a.listenAgentEvents()
		}
		// Queue the approval request
		a.approvalQueue = append(a.approvalQueue, nextApproval)
		// If not currently waiting, show the next one
		if !a.waitingForApproval {
			a.showNextApproval()
		}
		a.scheduleRender()
		if a.isThinking {
			return a.listenAgentEvents()
		}
		return tea.Batch(a.listenAgentEvents(), a.tickSpinner())

	case agent.EventQuestionRequest:
		a.commitActiveStream()
		// Queue the question request
		a.questionQueue = append(a.questionQueue, pendingQuestion{
			questionID: event.QuestionID,
			question:   event.QuestionText,
			options:    event.QuestionOptions,
			context:    event.QuestionContext,
		})
		// If not currently waiting for a question, show the next one
		if !a.waitingForQuestion {
			a.showNextQuestion()
		}
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventTurnEnd:
		a.invalidateToolModalCache()
		if event.ContextUsage != nil {
			a.contextUsage = event.ContextUsage
		}
		if a.currentThinkIdx >= 0 {
			a.finalizeThinkStream(a.currentThinkIdx)
			a.printMessageOnce(a.currentThinkIdx)
		}
		if a.currentAssistantIdx >= 0 {
			a.finalizeAssistantStream(a.currentAssistantIdx)
			a.printMessageOnce(a.currentAssistantIdx)
		}
		a.currentAssistantIdx = -1
		a.currentThinkIdx = -1
		a.updateViewportContent()
		return a.listenAgentEvents()

	case agent.EventDone:
		a.invalidateToolModalCache()
		a.isThinking = false
		a.finishRequestTimer()
		if event.ContextUsage != nil {
			a.contextUsage = event.ContextUsage
		}
		if a.currentThinkIdx >= 0 {
			a.finalizeThinkStream(a.currentThinkIdx)
			a.printMessageOnce(a.currentThinkIdx)
		}
		if a.currentAssistantIdx >= 0 {
			a.finalizeAssistantStream(a.currentAssistantIdx)
			a.printMessageOnce(a.currentAssistantIdx)
		}
		a.currentAssistantIdx = -1
		a.currentThinkIdx = -1
		a.refreshESMPanel()
		a.updateViewportContent()
		return tea.Batch(a.timer.Stop(), a.listenAgentEvents(), a.finishESMRun(nil))

	case agent.EventError:
		a.commitActiveStream()
		if (a.multiAgent || a.delegateMode) && a.agentMgr != nil && a.agent != nil {
			a.agentMgr.MarkError(a.agent.ID(), event.Error)
		}
		a.isThinking = false
		a.finishRequestTimer()
		if event.Error != nil {
			a.addMessage(errorStyle.Render("Error: ") + a.formatAgentError(event))
		}
		if event.StopReason != "" {
			a.addMessage(statusStyle.Render("Session ended: ") + event.StopReason)
		}
		a.pendingAbortReason = ""
		a.currentAssistantIdx = -1
		a.currentThinkIdx = -1
		a.refreshESMPanel()
		a.updateViewportContent()
		return tea.Batch(a.timer.Stop(), a.listenAgentEvents(), a.finishESMRun(event.Error))

	case agent.EventUsage:
		if event.ContextUsage != nil {
			a.contextUsage = event.ContextUsage
		}
		if event.Usage != nil {
			a.latestUsage = cloneUsage(event.Usage)
			a.recordESMUsage(event.Usage)
			// Accumulate cache stats
			a.totalInputTokens += event.Usage.TotalInputTokens()
			a.totalCacheRead += event.Usage.CacheRead
			a.totalCacheWrite += event.Usage.CacheWrite
			a.totalCostUSD += event.Usage.Cost.Total

			// Per-turn cache info
			cacheInfo := ""
			if info := event.Usage.CacheInfo(); info != "" {
				cacheInfo = " | " + info
			}
			costStr := fmt.Sprintf("Tokens: %d↓/%d↑ $%.4f%s",
				event.Usage.TotalInputTokens(), event.Usage.Output, event.Usage.Cost.Total, cacheInfo)
			a.addMessage(statusStyle.Render(costStr))
			a.refreshESMPanel()
		}
		a.scheduleRender()
		return a.listenAgentEvents()

	case agent.EventCompactionStart:
		a.addMessage(statusStyle.Render("⏳ Compacting context..."))
		return a.listenAgentEvents()

	case agent.EventCompactionEnd:
		if event.Error == nil && a.agent != nil {
			a.contextUsage = a.agent.GetContextUsage()
		}
		if event.Error != nil {
			a.addMessage(errorStyle.Render("Compaction failed: ") + event.Error.Error())
		} else if event.StopReason == "canceled" {
			a.addMessage(statusStyle.Render(event.StatusMessage))
		} else if event.StatusMessage != "" {
			a.addMessage(statusStyle.Render("✅ " + event.StatusMessage))
		} else {
			a.addMessage(statusStyle.Render("✅ Context compacted"))
		}
		return a.listenAgentEvents()

	case agent.EventContextPressure, agent.EventBudgetPressure:
		if event.ContextUsage != nil {
			a.contextUsage = event.ContextUsage
		}
		if event.PressureMessage != "" {
			a.addMessage(warningStyle.Render(event.PressureMessage))
		}
		return a.listenAgentEvents()

	case agent.EventStatus:
		if event.StatusMessage != "" {
			a.addMessage(statusStyle.Render(event.StatusMessage))
		}
		a.refreshESMPanel()
		return a.listenAgentEvents()

	case agent.EventMessageStart:
		if event.Message.Role == "user" && event.Message.Content != "" && !event.Message.SystemInjected {
			a.addMessage(userStyle.Render("You: ") + event.Message.Content)
		}
		return a.listenAgentEvents()

	default:
		return a.listenAgentEvents()
	}
}

func (a *App) formatAgentError(event agent.Event) string {
	if event.Error == nil {
		return ""
	}
	msg := event.Error.Error()
	if event.StopReason == "aborted" && a.pendingAbortReason != "" && !strings.Contains(msg, a.pendingAbortReason) {
		msg += " (reason: " + a.pendingAbortReason + ")"
	}
	return msg
}

func (a *App) appendToolExecutionStart(toolCallID, toolName string, toolArgs map[string]any) {
	if toolName == "" {
		return
	}
	if a.hasToolEntry(toolCallID, toolResultStatusRunning) || a.hasToolEntry(toolCallID, toolResultStatusCompleted) {
		return
	}

	a.invalidateToolModalCache()
	a.commitActiveStream()
	msgIdx := len(a.messages)
	runningEntry := toolResult{
		toolCallID: toolCallID,
		toolName:   toolName,
		toolArgs:   toolArgs,
		status:     toolResultStatusRunning,
		msgIndex:   msgIdx,
	}
	a.toolResults = append(a.toolResults, runningEntry)
	a.messages = append(a.messages, "")
	runningLine := formatToolExecutionStart(runningEntry)
	if runningLine != "" {
		a.messages[msgIdx] = toolStyle.Render(runningLine)
		a.printMessageOnce(msgIdx)
	}
	a.updateViewportContent()
}

func (a *App) appendToolResult(event agent.Event) {
	if a.hasToolEntry(event.ToolCallID, toolResultStatusCompleted) {
		return
	}

	a.invalidateToolModalCache()
	matchedArgs := event.ToolArgs
	matchedName := event.ToolName
	for j := len(a.toolResults) - 1; j >= 0; j-- {
		if a.toolResults[j].toolCallID == event.ToolCallID {
			if matchedArgs == nil {
				matchedArgs = a.toolResults[j].toolArgs
			}
			if matchedName == "" {
				matchedName = a.toolResults[j].toolName
			}
			break
		}
	}

	msgIdx := len(a.messages)
	resultEntry := toolResult{
		toolCallID:  event.ToolCallID,
		toolName:    matchedName,
		toolArgs:    matchedArgs,
		status:      toolResultStatusCompleted,
		msgIndex:    msgIdx,
		fullContent: event.ToolResult,
		diff:        event.ToolDiff,
		summary:     summarizeToolResult(matchedName, event.ToolResult, event.ToolDiff),
	}

	a.toolResults = append(a.toolResults, resultEntry)
	a.messages = append(a.messages, "")
	a.printMessageOnce(msgIdx)
}

func (a *App) hasToolEntry(toolCallID string, status toolResultStatus) bool {
	if toolCallID == "" {
		return false
	}
	for _, result := range a.toolResults {
		if result.toolCallID == toolCallID && result.status == status {
			return true
		}
	}
	return false
}

func summarizeToolResult(toolName, result string, diff *tools.FileDiff) string {
	switch toolName {
	case "bash":
		return compactBashOutput(result)
	case "read":
		lines := strings.Split(result, "\n")
		return fmt.Sprintf("%d lines", len(lines))
	case "ls":
		return compactBashOutput(result)
	case "write":
		if summary := summarizeFileDiff(diff); summary != "" {
			return summary
		}
		return summarizeWriteToolResult(result)
	case "edit":
		if summary := summarizeFileDiff(diff); summary != "" {
			return summary
		}
		return "Applied"
	default:
		return truncate(result, 50)
	}
}
