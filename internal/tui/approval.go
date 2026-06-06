package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/tools"
)

// showNextApproval pops the next approval request from the queue and displays it.
func (a *App) showNextApproval() {
	if len(a.approvalQueue) == 0 {
		a.waitingForApproval = false
		a.pendingApprovalID = ""
		return
	}
	next := a.approvalQueue[0]
	a.approvalQueue = a.approvalQueue[1:]
	a.pendingApprovalID = next.approvalID
	a.waitingForApproval = true

	// Build all lines into one message to preserve order.
	var sb strings.Builder
	if len(a.approvalQueue) > 0 {
		sb.WriteString(warningStyle.Render(fmt.Sprintf("⚠️  Approval required for [%s] (%d more pending)", next.toolName, len(a.approvalQueue))))
	} else {
		sb.WriteString(warningStyle.Render(fmt.Sprintf("⚠️  Approval required for [%s]", next.toolName)))
	}
	sb.WriteByte('\n')
	if len(next.args) > 0 {
		sb.WriteString(warningStyle.Render(formatApprovalArgs(next.toolName, next.args)))
		sb.WriteByte('\n')
	}
	sb.WriteString(warningStyle.Render("Approve? (y/n): "))
	a.addMessage(sb.String())
}

func (a *App) clearApprovalState() {
	a.waitingForApproval = false
	a.pendingApprovalID = ""
	a.approvalQueue = a.approvalQueue[:0]
}

// showNextQuestion pops the next question request from the queue and displays it.
func (a *App) showNextQuestion() {
	if len(a.questionQueue) == 0 {
		a.waitingForQuestion = false
		a.pendingQuestionID = ""
		return
	}
	next := a.questionQueue[0]
	a.questionQueue = a.questionQueue[1:]
	a.pendingQuestionID = next.questionID
	a.waitingForQuestion = true

	// Build all lines into one message to preserve order (addMessage uses
	// async goroutines, so multiple calls can interleave).
	var sb strings.Builder
	if next.context != "" {
		sb.WriteString(warningStyle.Render("💬 " + next.context))
		sb.WriteByte('\n')
	}
	sb.WriteString(warningStyle.Render("❓ " + next.question))
	sb.WriteByte('\n')
	for i, opt := range next.options {
		sb.WriteString(statusStyle.Render(fmt.Sprintf("  [%d] %s", i+1, opt)))
		sb.WriteByte('\n')
	}
	sb.WriteString(statusStyle.Render(fmt.Sprintf("  [%d] ✍️  Custom input", len(next.options)+1)))
	sb.WriteByte('\n')
	sb.WriteString(warningStyle.Render("Enter number or custom text: "))
	a.addMessage(sb.String())
}

func (a *App) clearQuestionState() {
	a.waitingForQuestion = false
	a.pendingQuestionID = ""
	a.questionQueue = a.questionQueue[:0]
}

func formatApprovalArgs(toolName string, args map[string]any) string {
	if toolName == "edit" {
		return formatEditApprovalArgs(args)
	}

	safeArgs := make(map[string]any, len(args))
	for k, v := range args {
		if k == "content" {
			text := fmt.Sprintf("%v", v)
			safeArgs[k] = fmt.Sprintf("(%d bytes)", len(text))
			continue
		}
		safeArgs[k] = v
	}
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(safeArgs); err != nil {
		return fmt.Sprintf("%v", safeArgs)
	}
	return strings.TrimRight(buf.String(), "\n")
}

func formatEditApprovalArgs(args map[string]any) string {
	path, _ := args["path"].(string)
	if path == "" {
		path = "<unknown path>"
	}

	var diffs []string
	editList, ok := args["edits"].([]any)
	if ok {
		for _, e := range editList {
			editMap, ok := e.(map[string]any)
			if !ok {
				continue
			}
			oldText, _ := editMap["oldText"].(string)
			newText, _ := editMap["newText"].(string)
			diff := tools.BuildFileDiff(path, oldText, newText)
			if diff == nil || strings.TrimSpace(diff.Unified) == "" {
				continue
			}
			diffs = append(diffs, strings.TrimRight(diff.Unified, "\n"))
		}
	}

	if len(diffs) == 0 {
		return fmt.Sprintf("path: %s\ndiff: (empty)", path)
	}
	return fmt.Sprintf("path: %s\n%s", path, strings.Join(diffs, "\n"))
}
