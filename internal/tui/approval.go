package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/tui/renderutil"
)

type approvalAction int

const (
	approvalActionApprove approvalAction = iota
	approvalActionDeny
	approvalActionAllowCommand
	approvalActionAllowPrefix
)

type approvalOption struct {
	Action      approvalAction
	Title       string
	Description string
}

// showNextApproval pops the next approval request from the queue and displays it.
func (a *App) showNextApproval() {
	if len(a.approvalQueue) == 0 {
		a.waitingForApproval = false
		a.pendingApprovalID = ""
		a.currentApproval = pendingApproval{}
		a.currentApprovalIdx = -1
		a.approvalCursor = 0
		return
	}
	next := a.approvalQueue[0]
	a.approvalQueue = a.approvalQueue[1:]
	a.currentApproval = next
	a.pendingApprovalID = next.approvalID
	a.waitingForApproval = true
	a.approvalCursor = 0

	a.invalidateToolModalCache()
	a.currentApprovalIdx = len(a.messages)
	a.messages = append(a.messages, a.renderApprovalRequest(next, len(a.approvalQueue)))
	a.updateViewportContentWithFollow(true)
	a.scheduleRender()
}

func (a *App) clearApprovalState() {
	a.waitingForApproval = false
	a.pendingApprovalID = ""
	a.currentApproval = pendingApproval{}
	a.currentApprovalIdx = -1
	a.approvalCursor = 0
	a.approvalQueue = a.approvalQueue[:0]
}

func (a *App) handleApprovalResponse(approvalID string, approved bool) {
	current := a.currentApproval
	current.approvalID = approvalID
	a.handlePendingApprovalResponse(current, approved)
}

func (a *App) handlePendingApprovalResponse(p pendingApproval, approved bool) {
	if p.agentID != "" && a.agentMgr != nil {
		if target, ok := a.agentMgr.Get(p.agentID); ok {
			target.HandleApprovalResponse(p.approvalID, approved)
			return
		}
	}
	if a.agent != nil {
		a.agent.HandleApprovalResponse(p.approvalID, approved)
	}
}

func (a *App) handleApprovalKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !a.waitingForApproval {
		return false, nil
	}
	opts := a.approvalOptions()
	if len(opts) == 0 {
		return true, nil
	}
	if a.approvalCursor < 0 {
		a.approvalCursor = 0
	}
	if a.approvalCursor >= len(opts) {
		a.approvalCursor = len(opts) - 1
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return false, nil
	case tea.KeyEsc:
		return true, a.abortPendingRequest("user pressed Esc")
	case tea.KeyUp:
		if a.approvalCursor > 0 {
			a.approvalCursor--
			a.scheduleRender()
		}
		return true, nil
	case tea.KeyDown:
		if a.approvalCursor < len(opts)-1 {
			a.approvalCursor++
			a.scheduleRender()
		}
		return true, nil
	case tea.KeyEnter:
		a.confirmApprovalSelection(opts[a.approvalCursor].Action)
		return true, nil
	case tea.KeyRunes:
		switch strings.ToLower(strings.TrimSpace(string(msg.Runes))) {
		case "y":
			a.confirmApprovalSelection(approvalActionApprove)
			return true, nil
		case "n":
			a.confirmApprovalSelection(approvalActionDeny)
			return true, nil
		}
	}
	return true, nil
}

func (a *App) confirmApprovalSelection(action approvalAction) {
	switch action {
	case approvalActionApprove:
		a.finishApproval(true, "Approved once", false)
	case approvalActionDeny:
		a.finishApproval(false, "Denied", false)
	case approvalActionAllowCommand:
		command := a.currentApprovalCommand()
		if command == "" {
			a.addCommandError("Cannot save approval rule: missing bash command.")
			return
		}
		if err := a.saveApprovalBashRule(command, ""); err != nil {
			a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
			return
		}
		a.finishApproval(true, "Approved and remembered this command for this project", true)
	case approvalActionAllowPrefix:
		prefix := suggestApprovalCommandPrefix(a.currentApprovalCommand())
		if prefix == "" {
			a.addCommandError("Cannot save approval rule: missing bash command prefix.")
			return
		}
		if err := a.saveApprovalBashRule("", prefix); err != nil {
			a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
			return
		}
		a.finishApproval(true, fmt.Sprintf("Approved and remembered project command prefix: %s", prefix), true)
	}
}

func (a *App) saveApprovalBashRule(command string, prefix string) error {
	if a.allow == nil {
		a.allow = config.LoadAllow()
	}
	addedCommand := false
	addedPrefix := false
	if command != "" {
		addedCommand = a.allow.AddBashCommand(command)
	}
	if prefix != "" {
		addedPrefix = a.allow.AddBashPrefix(prefix)
	}
	if err := a.allow.SaveProject(); err != nil {
		if addedCommand {
			a.allow.RemoveBashCommand(command)
		}
		if addedPrefix {
			a.allow.RemoveBashPrefix(prefix)
		}
		return err
	}
	return nil
}

func (a *App) finishApproval(approved bool, label string, approveQueuedAllowed bool) {
	approvalIdx := a.currentApprovalIdx
	if approvalIdx >= 0 {
		a.printMessageOnce(approvalIdx)
	}
	a.handleApprovalResponse(a.pendingApprovalID, approved)
	if approved {
		a.addMessage(statusStyle.Render("✅ " + label))
	} else {
		a.addMessage(statusStyle.Render("❌ " + label))
	}
	if approveQueuedAllowed {
		if count := a.approveQueuedAllowedBashApprovals(); count > 0 {
			a.addMessage(statusStyle.Render(fmt.Sprintf("✅ Approved %d queued matching command(s)", count)))
		}
	}
	if len(a.approvalQueue) > 0 {
		a.showNextApproval()
	} else {
		a.waitingForApproval = false
		a.pendingApprovalID = ""
		a.currentApproval = pendingApproval{}
		a.currentApprovalIdx = -1
		a.approvalCursor = 0
	}
	a.input.Reset()
	a.resetInputHistoryNavigation()
	a.clearQueuedInput()
	a.scheduleRender()
}

func (a *App) approveQueuedAllowedBashApprovals() int {
	if a.allow == nil || len(a.approvalQueue) == 0 {
		return 0
	}
	kept := a.approvalQueue[:0]
	approved := 0
	for _, p := range a.approvalQueue {
		if p.toolName == "bash" && a.allow.MatchBashCommand(approvalCommand(p.args)) {
			a.handlePendingApprovalResponse(p, true)
			approved++
			continue
		}
		kept = append(kept, p)
	}
	a.approvalQueue = kept
	return approved
}

func (a *App) hasPendingApproval(p pendingApproval) bool {
	if p.approvalID == "" {
		return false
	}
	if a.waitingForApproval && a.currentApproval.approvalID == p.approvalID && a.currentApproval.agentID == p.agentID {
		return true
	}
	for _, queued := range a.approvalQueue {
		if queued.approvalID == p.approvalID && queued.agentID == p.agentID {
			return true
		}
	}
	return false
}

func (a *App) approvalOptions() []approvalOption {
	opts := []approvalOption{
		{Action: approvalActionApprove, Title: "Approve Once", Description: "Run only this pending tool call"},
		{Action: approvalActionDeny, Title: "Deny", Description: "Reject this pending tool call"},
	}
	command := a.currentApprovalCommand()
	if a.currentApproval.toolName != "bash" || strings.TrimSpace(command) == "" {
		return opts
	}
	opts = append(opts,
		approvalOption{
			Action:      approvalActionAllowCommand,
			Title:       "Always Allow Exact Command",
			Description: "Project rule: " + truncatePlain(command, 96),
		},
		approvalOption{
			Action:      approvalActionAllowPrefix,
			Title:       "Always Allow Command Prefix",
			Description: "Project rule: " + truncatePlain(suggestApprovalCommandPrefix(command), 96),
		},
	)
	return opts
}

func (a *App) currentApprovalCommand() string {
	return approvalCommand(a.currentApproval.args)
}

func approvalCommand(args map[string]any) string {
	for _, key := range []string{"command", "cmd"} {
		if command, ok := args[key].(string); ok {
			command = strings.TrimSpace(command)
			if command != "" {
				return command
			}
		}
	}
	return ""
}

func suggestApprovalCommandPrefix(command string) string {
	command = strings.TrimLeft(command, " \t\r\n")
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	want := 2
	if len(fields) < want {
		want = len(fields)
	}
	prefixText := strings.Join(fields[:want], " ")
	idx := strings.Index(command, prefixText)
	if idx >= 0 {
		prefixText = command[idx : idx+len(prefixText)]
	}
	if len(command) > len(prefixText) && command[len(prefixText)] == ' ' {
		return prefixText + " "
	}
	return prefixText
}

func (a *App) renderApprovalDialog() string {
	width := a.width - 4
	if width < 50 {
		width = 50
	}
	if width > 100 {
		width = 100
	}
	innerWidth := width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	title := fmt.Sprintf("Approval Required: %s", a.currentApproval.toolName)
	if len(a.approvalQueue) > 0 {
		title += fmt.Sprintf(" (%d more pending)", len(a.approvalQueue))
	}
	lines := []string{warningStyle.Render(title), ""}
	if detail := a.renderApprovalDialogDetails(innerWidth); detail != "" {
		lines = append(lines, detail, "")
	}

	opts := a.approvalOptions()
	for i, opt := range opts {
		cursor := "  "
		style := statusStyle
		if i == a.approvalCursor {
			cursor = "> "
			style = warningStyle
		}
		lines = append(lines, style.Render(cursor+opt.Title))
		if opt.Description != "" {
			desc := renderutil.WrapPlainText(opt.Description, innerWidth-4)
			lines = append(lines, indentLines(statusStyle.Render(desc), "    "))
		}
	}
	lines = append(lines, "", statusStyle.Render("Enter select · ↑/↓ move · y approve · n deny · Esc abort"))
	return authDialogStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (a *App) renderApprovalDialogDetails(width int) string {
	switch a.currentApproval.toolName {
	case "bash":
		return renderBashApprovalDialogDetails(a.currentApproval.args, width)
	case "edit":
		return renderWrappedApprovalDetail(formatEditApprovalArgs(a.currentApproval.args), width)
	case "write":
		return renderWrappedApprovalDetail(formatWriteApprovalArgs(a.currentApproval.args), width)
	default:
		return renderWrappedApprovalDetail(formatGenericApprovalArgs(a.currentApproval.args), width)
	}
}

func renderBashApprovalDialogDetails(args map[string]any, width int) string {
	var lines []string
	if command := approvalCommand(args); command != "" {
		lines = append(lines, statusStyle.Render("Command:"))
		lines = append(lines, indentLines(renderutil.WrapPlainText(command, width-2), "  "))
	}
	if timeout, ok := args["timeout"]; ok {
		lines = append(lines, fmt.Sprintf("Timeout: %v", timeout))
	}
	if async, ok := args["async"]; ok {
		lines = append(lines, fmt.Sprintf("Async: %v", async))
	}
	if len(lines) == 0 {
		return renderWrappedApprovalDetail(formatGenericApprovalArgs(args), width)
	}
	return strings.Join(lines, "\n")
}

func renderWrappedApprovalDetail(detail string, width int) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}
	return renderutil.WrapPlainText(detail, width)
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
	a.currentQuestion = next
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
	a.currentQuestion = pendingQuestion{}
	a.questionQueue = a.questionQueue[:0]
}

func (a *App) renderApprovalRequest(next pendingApproval, remaining int) string {
	var sb strings.Builder
	title := fmt.Sprintf("! Approval required: %s", next.toolName)
	if remaining > 0 {
		title += fmt.Sprintf(" (%d more pending)", remaining)
	}
	sb.WriteString(warningStyle.Render(title))
	sb.WriteByte('\n')

	if detail := formatApprovalArgs(next.toolName, next.args); strings.TrimSpace(detail) != "" {
		sb.WriteString(detail)
		sb.WriteByte('\n')
	}

	sb.WriteString(statusStyle.Render("Choose in the approval dialog (↑/↓, Enter, y/n)."))
	return sb.String()
}

func formatApprovalArgs(toolName string, args map[string]any) string {
	switch toolName {
	case "bash":
		return formatBashApprovalArgs(args)
	case "edit":
		return formatEditApprovalArgs(args)
	case "write":
		return formatWriteApprovalArgs(args)
	}

	return formatGenericApprovalArgs(args)
}

func formatBashApprovalArgs(args map[string]any) string {
	var lines []string
	if command := approvalCommand(args); command != "" {
		lines = append(lines, statusStyle.Render("command:"))
		lines = append(lines, indentLines(command, "  "))
	}
	if timeout, ok := args["timeout"]; ok {
		lines = append(lines, fmt.Sprintf("timeout: %v", timeout))
	}
	if async, ok := args["async"]; ok {
		lines = append(lines, fmt.Sprintf("async: %v", async))
	}
	return strings.Join(lines, "\n")
}

func formatWriteApprovalArgs(args map[string]any) string {
	var lines []string
	if path, ok := args["path"].(string); ok && path != "" {
		lines = append(lines, fmt.Sprintf("path: %s", path))
	}
	if content, ok := args["content"]; ok {
		text := fmt.Sprintf("%v", content)
		lines = append(lines, fmt.Sprintf("content: (%d bytes)", len(text)))
	}
	if len(lines) == 0 {
		return formatGenericApprovalArgs(args)
	}
	return strings.Join(lines, "\n")
}

func formatGenericApprovalArgs(args map[string]any) string {
	safeArgs := make(map[string]any, len(args))
	for k, v := range args {
		if k == "content" {
			text := fmt.Sprintf("%v", v)
			safeArgs[k] = fmt.Sprintf("(%d bytes)", len(text))
			continue
		}
		safeArgs[k] = v
	}

	keys := make([]string, 0, len(safeArgs))
	for k := range safeArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", k, formatApprovalValue(safeArgs[k])))
	}
	return strings.Join(lines, "\n")
}

func formatApprovalValue(v any) string {
	switch val := v.(type) {
	case string:
		if strings.Contains(val, "\n") {
			return "\n" + indentLines(val, "  ")
		}
		return val
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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
