package agent

import (
	"context"
	"fmt"
	"strings"
)

// NeedsApproval checks if a tool call needs user approval based on the current mode.
func (a *Agent) NeedsApproval(toolName string, args map[string]any) bool {
	if (toolName == "write" || toolName == "edit") && a.config.Mode == "agent" {
		// Auto-approve edits globally when AllowAutoEdit is on.
		if a.config.Allow != nil && a.config.Allow.GetAutoEdit() {
			return false
		}
		// Auto-approve edits whose path matches the allow.json whitelist.
		if a.config.Allow != nil {
			if p, ok := args["path"].(string); ok && a.config.Allow.MatchEditPath(p) {
				return false
			}
		}
		return a.config.Settings != nil &&
			a.config.Settings.Approval.ConfirmBeforeWrite != nil &&
			*a.config.Settings.Approval.ConfirmBeforeWrite
	}
	if toolName != "bash" {
		return false
	}
	if a.isBashBlacklisted(args) {
		return true
	}
	switch a.config.Mode {
	case "plan":
		// Plan mode: no tools should be executed (read-only tools don't need approval)
		return false
	case "agent":
		// Agent mode: only whitelisted bash can skip approval.
		return !a.isBashWhitelisted(args)
	case "yolo":
		// YOLO mode: allow bash unless explicitly blacklisted above.
		return false
	default:
		return false
	}
}

func (a *Agent) isBashWhitelisted(args map[string]any) bool {
	if a.config.Settings == nil {
		return false
	}
	command, ok := args["command"].(string)
	if !ok {
		return false
	}
	for _, prefix := range a.config.Settings.Approval.BashWhitelist {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

func (a *Agent) isBashBlacklisted(args map[string]any) bool {
	if a.config.Settings == nil {
		return false
	}
	command, ok := args["command"].(string)
	if !ok {
		return false
	}
	for _, prefix := range a.config.Settings.Approval.BashBlacklist {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

// RequestApproval sends an approval request and waits for the user's response.
func (a *Agent) RequestApproval(ch chan<- Event, toolName string, args map[string]any) bool {
	a.approvalMu.Lock()
	a.approvalCounter++
	approvalID := fmt.Sprintf("approval-%d", a.approvalCounter)
	responseCh := make(chan bool, 1)
	a.pendingApprovals[approvalID] = responseCh
	a.approvalMu.Unlock()

	// Send approval request event
	ch <- Event{
		Type:         EventToolApprovalRequest,
		ApprovalID:   approvalID,
		ApprovalTool: toolName,
		ApprovalArgs: args,
	}

	// Wait for response or abort
	select {
	case approved := <-responseCh:
		return approved
	case <-a.abort:
		a.approvalMu.Lock()
		delete(a.pendingApprovals, approvalID)
		a.approvalMu.Unlock()
		return false
	}
}

// HandleApprovalResponse processes the user's approval response.
func (a *Agent) HandleApprovalResponse(approvalID string, approved bool) {
	a.approvalMu.Lock()
	defer a.approvalMu.Unlock()

	if ch, ok := a.pendingApprovals[approvalID]; ok {
		ch <- approved
		delete(a.pendingApprovals, approvalID)
	}
}

// RequestQuestion sends a question request and waits for the user's answer.
func (a *Agent) RequestQuestion(ch chan<- Event, question string, options []string, context string) string {
	a.questionMu.Lock()
	a.questionCounter++
	questionID := fmt.Sprintf("question-%d", a.questionCounter)
	responseCh := make(chan string, 1)
	a.pendingQuestions[questionID] = responseCh
	a.questionMu.Unlock()

	ch <- Event{
		Type:            EventQuestionRequest,
		QuestionID:      questionID,
		QuestionText:    question,
		QuestionOptions: options,
		QuestionContext: context,
	}

	select {
	case answer := <-responseCh:
		return answer
	case <-a.abort:
		a.questionMu.Lock()
		delete(a.pendingQuestions, questionID)
		a.questionMu.Unlock()
		return ""
	}
}

// HandleQuestionResponse processes the user's answer to a question.
func (a *Agent) HandleQuestionResponse(questionID string, answer string) {
	a.questionMu.Lock()
	defer a.questionMu.Unlock()

	if ch, ok := a.pendingQuestions[questionID]; ok {
		ch <- answer
		delete(a.pendingQuestions, questionID)
	}
}

// AskQuestion implements the tools.QuestionAsker interface.
// It gets the event channel from the context and delegates to RequestQuestion.
func (a *Agent) AskQuestion(ctx context.Context, question string, options []string, explanation string) string {
	eventCh, ok := EventChanFromContext(ctx)
	if !ok {
		return ""
	}
	return a.RequestQuestion(eventCh, question, options, explanation)
}
