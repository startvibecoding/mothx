package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/tui/renderutil"
)

type toolModalTarget struct {
	AgentID agentpkg.AgentID
	Label   string
	Kind    string
}

func (a *App) openLatestToolModal() bool {
	if len(a.messages) == 0 && len(a.toolResults) == 0 && len(a.assistantRaw) == 0 && len(a.agentActivities) == 0 {
		return false
	}
	a.toolModalOpen = true
	a.toolModalPinnedBottom = true
	a.toolModalActive = 0
	a.toolModalOffset = 0
	return true
}

func (a *App) closeToolModal() {
	a.toolModalOpen = false
	a.toolModalOffset = 0
	a.toolModalPinnedBottom = false
	a.toolModalActive = 0
}

func (a *App) invalidateToolModalCache() {
	a.toolModalVersion++
	a.toolModalCacheValid = false
	a.toolModalCacheLines = nil
}

func formatToolModalContent(result toolResult) string {
	var parts []string
	if result.toolArgs != nil {
		if args := formatToolArgs(result.toolName, result.toolArgs); strings.TrimSpace(args) != "" {
			parts = append(parts, args)
		}
	}
	if result.fullContent != "" {
		parts = append(parts, "---", result.fullContent)
	}
	if result.diff != nil && result.diff.Unified != "" {
		parts = append(parts, "--- diff", result.diff.Unified)
	}
	if len(parts) == 0 {
		return "(no output)"
	}
	return strings.Join(parts, "\n")
}

func (a *App) renderExpandedTranscript() string {
	return strings.Join(a.toolModalLines(a.toolModalTargets()), "\n")
}

func (a *App) buildToolModalLines(targets []toolModalTarget) []string {
	if a.toolModalActive > 0 && a.toolModalActive < len(targets) {
		return strings.Split(a.renderAgentActivity(targets[a.toolModalActive].AgentID), "\n")
	}
	var parts []string
	for i := range a.messages {
		msg := a.renderExpandedMessageAt(i)
		if strings.TrimSpace(msg) != "" {
			parts = append(parts, msg)
		}
	}
	if len(parts) == 0 {
		return []string{"(no conversation yet)"}
	}
	lines := make([]string, 0, len(parts)*2)
	for i, part := range parts {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.Split(part, "\n")...)
	}
	return lines
}

func (a *App) toolModalLines(targets []toolModalTarget) []string {
	width := toolModalContentWidth(toolModalWidth(a.width))
	if a.toolModalCacheValid &&
		a.toolModalCacheActive == a.toolModalActive &&
		a.toolModalCacheVersion == a.toolModalVersion &&
		a.toolModalCacheWidth == width {
		return a.toolModalCacheLines
	}
	lines := a.buildToolModalLines(targets)
	content := strings.Join(lines, "\n")
	lines = strings.Split(renderutil.WrapANSI(content, width), "\n")
	a.toolModalCacheLines = lines
	a.toolModalCacheActive = a.toolModalActive
	a.toolModalCacheVersion = a.toolModalVersion
	a.toolModalCacheWidth = width
	a.toolModalCacheValid = true
	return lines
}

func (a *App) toolModalTargets() []toolModalTarget {
	targets := []toolModalTarget{{Label: "Main", Kind: "main"}}
	seen := map[agentpkg.AgentID]bool{"": true}
	if a.agent != nil {
		seen[a.agent.ID()] = true
	}
	for _, id := range a.agentActivityOrder {
		if id == "" || seen[id] {
			continue
		}
		targets = append(targets, a.toolModalTargetFor(id))
		seen[id] = true
	}
	if a.agentMgr != nil {
		for _, id := range a.agentMgr.List() {
			if id == "" || seen[id] {
				continue
			}
			targets = append(targets, a.toolModalTargetFor(id))
			seen[id] = true
		}
	}
	if a.toolModalActive >= len(targets) {
		a.toolModalActive = len(targets) - 1
	}
	if a.toolModalActive < 0 {
		a.toolModalActive = 0
	}
	return targets
}

func (a *App) toolModalTargetFor(id agentpkg.AgentID) toolModalTarget {
	kind := "subagent"
	label := string(id)
	if act := a.agentActivities[id]; act != nil {
		kind = act.Kind
		if act.State != "" {
			label += " " + act.State
		}
	}
	return toolModalTarget{AgentID: id, Label: label, Kind: kind}
}

func (a *App) renderAgentActivity(id agentpkg.AgentID) string {
	act := a.agentActivities[id]
	if act == nil {
		if a.agentMgr != nil {
			if st, ok := a.agentMgr.Status(id); ok {
				return fmt.Sprintf("%s [%s]\n\n(no activity captured yet)", id, st.State)
			}
		}
		return fmt.Sprintf("%s\n\n(no activity captured yet)", id)
	}
	var lines []string
	header := string(id)
	if act.Kind != "" {
		header += " (" + act.Kind + ")"
	}
	if act.State != "" {
		header += " [" + act.State + "]"
	}
	if !act.UpdatedAt.IsZero() {
		header += " updated " + formatActivityAge(act.UpdatedAt)
	}
	lines = append(lines, header)
	if act.LastTool != "" {
		lines = append(lines, "", "Tool: "+act.LastTool)
	}
	if act.LastThink != "" {
		lines = append(lines, "", "Thinking: "+act.LastThink)
	}
	if act.LastText != "" {
		lines = append(lines, "", "Text: "+act.LastText)
	}
	if act.LastResult != "" {
		lines = append(lines, "", "Result: "+act.LastResult)
	}
	if len(act.Events) > 0 {
		lines = append(lines, "", "Recent events:")
		for _, ev := range act.Events {
			prefix := ev.Time.Format("15:04:05")
			lines = append(lines, "  "+prefix+"  "+ev.Text)
		}
	}
	if len(lines) == 1 {
		lines = append(lines, "", "(no activity captured yet)")
	}
	return strings.Join(lines, "\n")
}

func formatActivityAge(t time.Time) string {
	d := time.Since(t).Round(time.Second)
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm ago", int(d.Minutes()))
}

func (a *App) renderToolModalTabs(targets []toolModalTarget, width int) string {
	if len(targets) <= 1 {
		return ""
	}
	active := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	inactive := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	parts := make([]string, 0, len(targets))
	for i, target := range targets {
		label := target.Label
		if label == "" {
			label = target.Kind
		}
		if i == a.toolModalActive {
			parts = append(parts, active.Render(label))
		} else {
			parts = append(parts, inactive.Render(label))
		}
	}
	row := strings.Join(parts, "  |  ")
	if lipgloss.Width(row) > width {
		return xansi.Truncate(row, width, "…")
	}
	return row
}

func (a *App) renderExpandedMessageAt(idx int) string {
	for i, tr := range a.toolResults {
		if tr.msgIndex == idx {
			return a.renderExpandedToolResultAt(i)
		}
	}
	if _, ok := a.assistantRaw[idx]; ok {
		return a.renderAssistantMessage(idx)
	}
	if _, ok := a.thinkRaw[idx]; ok {
		return a.renderThinkMessage(idx)
	}
	if idx >= 0 && idx < len(a.messages) {
		return a.messages[idx]
	}
	return ""
}

func (a *App) renderExpandedToolResultAt(idx int) string {
	if idx < 0 || idx >= len(a.toolResults) {
		return ""
	}
	if a.toolResults[idx].expanded != "" {
		return a.toolResults[idx].expanded
	}
	expanded := formatExpandedToolResult(a.toolResults[idx])
	a.toolResults[idx].expanded = expanded
	return expanded
}

func (a *App) renderExpandedToolResult(result toolResult) string {
	return formatExpandedToolResult(result)
}

func formatExpandedToolResult(result toolResult) string {
	content := formatToolHeader(result)
	if result.toolName == "edit" {
		content = formatExpandedEditHeader(result)
	}
	details := formatToolModalContent(result)
	if result.toolName == "bash" {
		return toolStyle.Render(formatToolHeader(result)) + "\n" + details
	}
	if result.toolName == "edit" {
		if strings.TrimSpace(details) != "" {
			return toolStyle.Render(content) + "\n" + details
		}
		return toolStyle.Render(content)
	}
	if strings.TrimSpace(details) != "" {
		content += "\n" + details
	}
	return toolStyle.Render(content)
}

func formatExpandedEditHeader(result toolResult) string {
	path := toolPath(result.toolArgs)
	if result.diff != nil && result.diff.Path != "" {
		path = result.diff.Path
	}
	if path == "" {
		path = "(unknown)"
	}

	summary := result.summary
	if result.diff != nil {
		summary = fmt.Sprintf("(+%d -%d)", result.diff.Added, result.diff.Deleted)
	}

	header := fmt.Sprintf("• Edited %s", path)
	if summary != "" {
		header += " " + summary
	}
	return header
}

func (a *App) renderToolModal() string {
	footerHeight := 0
	if a.height > 0 {
		footerHeight = lipgloss.Height(a.renderFooter())
	}
	return a.renderToolModalWithAvailableHeight(a.height - footerHeight)
}

func (a *App) renderToolModalWithAvailableHeight(availableHeight int) string {
	width := toolModalWidth(a.width)
	contentWidth := toolModalContentWidth(width)
	targets := a.toolModalTargets()
	height := a.toolModalPageSizeFor(targets, availableHeight)
	lines := a.toolModalLines(targets)
	maxOffset := maxToolModalOffsetFor(lines, height)
	if a.toolModalPinnedBottom {
		a.toolModalOffset = maxOffset
	}
	if a.toolModalOffset > maxOffset {
		a.toolModalOffset = maxOffset
	}
	end := a.toolModalOffset + height
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[a.toolModalOffset:end], "\n")
	if visible == "" {
		visible = " "
	}
	position := fmt.Sprintf("lines %d-%d/%d", a.toolModalOffset+1, end, len(lines))
	if len(lines) == 0 {
		position = "lines 0-0/0"
	}
	title := fmt.Sprintf("Agent details  %s  Left/Right:switch target  PgUp/PgDn:page  Up/Down:scroll  Esc:close", position)
	title = xansi.Truncate(title, contentWidth, "…")
	tabs := a.renderToolModalTabs(targets, contentWidth)
	header := title
	if tabs != "" {
		header += "\n" + tabs
	}
	content := header + "\n" + strings.Repeat("─", minInt(contentWidth, lipgloss.Width(title))) + "\n" + visible
	chrome := toolModalChrome(len(targets) > 1)
	return toolModalStyle.Width(width).Height(height + chrome).Render(content)
}

func toolModalWidth(terminalWidth int) int {
	width := terminalWidth - 4
	if width < 20 {
		return 20
	}
	return width
}

func toolModalContentWidth(width int) int {
	width -= toolModalStyle.GetHorizontalPadding()
	if width < 1 {
		return 1
	}
	return width
}

func (a *App) switchToolModalTarget(delta int) {
	targets := a.toolModalTargets()
	if len(targets) <= 1 {
		return
	}
	a.toolModalActive += delta
	if a.toolModalActive < 0 {
		a.toolModalActive = len(targets) - 1
	}
	if a.toolModalActive >= len(targets) {
		a.toolModalActive = 0
	}
	a.toolModalPinnedBottom = true
	a.toolModalOffset = 0
}

func (a *App) scrollToolModal(delta int) {
	a.toolModalOffset += delta
	if a.toolModalOffset < 0 {
		a.toolModalOffset = 0
	}
	maxOffset := a.maxToolModalOffset()
	if a.toolModalOffset > maxOffset {
		a.toolModalOffset = maxOffset
	}
	a.toolModalPinnedBottom = a.toolModalOffset == maxOffset
}

func (a *App) toolModalPageSize() int {
	footerHeight := 0
	if a.height > 0 {
		footerHeight = lipgloss.Height(a.renderFooter())
	}
	return a.toolModalPageSizeFor(a.toolModalTargets(), a.height-footerHeight)
}

func (a *App) toolModalPageSizeFor(targets []toolModalTarget, availableHeight int) int {
	pageSize := availableHeight - toolModalChrome(len(targets) > 1) - toolModalVerticalFrame()
	if pageSize < 1 {
		return 1
	}
	return pageSize
}

func toolModalChrome(hasTabs bool) int {
	if hasTabs {
		return 4
	}
	return 3
}

func toolModalVerticalFrame() int {
	return 2
}

func (a *App) maxToolModalOffset() int {
	targets := a.toolModalTargets()
	return maxToolModalOffsetFor(a.toolModalLines(targets), a.toolModalPageSize())
}

func maxToolModalOffsetFor(lines []string, pageSize int) int {
	maxOffset := len(lines) - pageSize
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}
