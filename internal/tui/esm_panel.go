package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"

	"github.com/startvibecoding/mothx/internal/esm"
	"github.com/startvibecoding/mothx/internal/tui/renderutil"
)

func (a *App) openESMPanel() {
	a.esmPanelOpen = true
	a.esmPanelScroll = 0
	a.refreshESMPanel()
}

func (a *App) closeESMPanel() {
	a.esmPanelOpen = false
	a.esmPanelScroll = 0
	a.esmPanelObjective = nil
	a.esmPanelErr = nil
}

func (a *App) refreshESMPanel() {
	a.esmMu.Lock()
	tracked := a.esmRunTracked
	a.esmMu.Unlock()
	if !a.esmPanelOpen && !tracked {
		return
	}
	obj, err := a.loadESMObjective(context.Background())
	if errors.Is(err, esm.ErrNotFound) {
		a.setESMFooter(nil)
		if a.esmPanelOpen {
			a.esmPanelObjective = nil
			a.esmPanelErr = nil
		}
		return
	}
	if err == nil {
		a.setESMFooter(obj)
	}
	if a.esmPanelOpen {
		a.esmPanelObjective = obj
		a.esmPanelErr = err
	}
}

func (a *App) scrollESMPanel(delta int) {
	a.esmPanelScroll += delta
	if a.esmPanelScroll < 0 {
		a.esmPanelScroll = 0
	}
	a.scheduleRender()
}

func (a *App) esmPanelPageSize() int {
	footerHeight := a.esmPanelFooterHeight()
	height := a.height - footerHeight - 5
	if a.height <= 0 {
		return 20
	}
	if height < 1 {
		return 1
	}
	return height
}

func (a *App) maxESMPanelOffset() int {
	width := esmPanelContentWidth(esmPanelWidth(a.width))
	maxOffset := len(a.esmPanelLines(width)) - a.esmPanelPageSize()
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

func (a *App) renderESMPanel() string {
	width := esmPanelWidth(a.width)
	innerWidth := esmPanelContentWidth(width)
	footerHeight := a.esmPanelFooterHeight()
	height := 20
	if a.height > 0 {
		height = a.height - footerHeight - 5
		if height < 0 {
			height = 0
		}
	}
	lines := a.esmPanelLines(innerWidth)
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if a.esmPanelScroll > maxOffset {
		a.esmPanelScroll = maxOffset
	}
	end := a.esmPanelScroll + height
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[a.esmPanelScroll:end], "\n")
	if visible == "" {
		visible = " "
	}
	position := fmt.Sprintf("lines %d-%d/%d", a.esmPanelScroll+1, end, len(lines))
	if len(lines) == 0 {
		position = "lines 0-0/0"
	} else if height == 0 {
		position = fmt.Sprintf("lines 0-0/%d", len(lines))
	}
	statusText := ""
	if obj := a.esmPanelObjective; obj != nil {
		statusText = fmt.Sprintf("  %s / %s", obj.Status, effectiveESMPhase(obj))
	}
	titleText := "ESM Progress" + statusText + "  " + position + "  Up/Down:scroll  PgUp/PgDn:page  Ctrl+E/Esc:close"
	suffix := "..."
	if innerWidth < len(suffix) {
		suffix = ""
	}
	title := statusStyle.Render(xansi.Truncate(titleText, innerWidth, suffix))
	divider := strings.Repeat("-", minInt(innerWidth, lipgloss.Width(title)))
	content := title + "\n" + divider + "\n" + visible
	return toolModalStyle.Width(width).Height(height + 3).Render(content)
}

func (a *App) esmPanelFooterHeight() int {
	if a.height > 0 && a.height < 8 {
		return 0
	}
	return lipgloss.Height(a.renderFooter())
}

func esmPanelWidth(terminalWidth int) int {
	if terminalWidth <= 0 {
		terminalWidth = 80
	}
	width := terminalWidth - 4
	if width < 1 {
		return 1
	}
	return width
}

func esmPanelContentWidth(width int) int {
	width -= toolModalStyle.GetHorizontalPadding()
	if width < 1 {
		return 1
	}
	return width
}

func (a *App) esmPanelLines(width int) []string {
	if a.esmPanelErr != nil {
		return []string{"Failed to load ESM progress: " + a.esmPanelErr.Error()}
	}
	obj := a.esmPanelObjective
	if obj == nil {
		return []string{
			"No Enable Supervisor Mode objective for this session.",
			"",
			"Create one with /esm <objective>.",
		}
	}

	phase := effectiveESMPhase(obj)
	lines := []string{
		"Enable Supervisor Mode",
		"",
		"Now: " + a.esmPanelNow(obj),
		esmPanelProgress(obj, phase),
		"Next: " + esmPanelNextStep(obj, phase),
		"",
		fmt.Sprintf("Status: %s", obj.Status),
		fmt.Sprintf("Stage: %s", esmPhaseLabel(phase)),
		"Pipeline: " + renderESMPipeline(phase, obj.Status),
	}
	if obj.Status == esm.StatusPaused && obj.RejectionCount >= esm.CompletionRejectionLimit {
		lines = append(lines, "Paused: completion rejection circuit breaker reached its limit; inspect the remaining work, then run /esm resume.")
	}
	lines = append(lines, "")
	lines = appendWrappedESMField(lines, "Objective", obj.Objective, width)

	if obj.ProgressSummary != "" {
		lines = append(lines, "")
		lines = appendWrappedESMField(lines, "Latest worker progress", obj.ProgressSummary, width)
	}
	if len(obj.RemainingWork) > 0 {
		lines = append(lines, "", fmt.Sprintf("Remaining work (%d):", len(obj.RemainingWork)))
		lines = appendESMItems(lines, obj.RemainingWork, width)
	}
	if obj.BlockedReason != "" {
		lines = append(lines, "")
		lines = appendWrappedESMField(lines, "Blocker", obj.BlockedReason, width)
		lines = append(lines, fmt.Sprintf("Repeated blocker audit: %d/3", obj.BlockedCount))
	}
	if obj.CompletionReview != "" {
		lines = append(lines, "")
		lines = appendWrappedESMField(lines, "Latest completion review", obj.CompletionReview, width)
	}
	if obj.RejectionCount > 0 {
		lines = append(lines, fmt.Sprintf("Consecutive completion rejections: %d/%d", obj.RejectionCount, esm.CompletionRejectionLimit))
	}
	if obj.RecoveryCount > 0 {
		lines = append(lines, "", fmt.Sprintf("Consecutive automatic recoveries: %d/%d", obj.RecoveryCount, esm.RecoveryLimit))
		if obj.RecoveryReason != "" {
			lines = appendWrappedESMField(lines, "Latest recovery reason", obj.RecoveryReason, width)
		}
	}
	if obj.CompletionReason != "" && obj.Status == esm.StatusCompleteCandidate {
		lines = append(lines, "")
		lines = appendWrappedESMField(lines, "Completion candidate", obj.CompletionReason, width)
	}

	if activity := a.activeESMPanelActivity(width); len(activity) > 0 {
		lines = append(lines, "", "Live details:")
		lines = append(lines, activity...)
	}

	lines = append(lines, "", fmt.Sprintf("Tokens: %d", obj.TokensUsed))
	if obj.TokenBudget != nil {
		lines[len(lines)-1] += fmt.Sprintf(" / %d", *obj.TokenBudget)
	}
	if obj.TimeUsedMS > 0 {
		lines = append(lines, "Time: "+formatDurationMSForPanel(obj.TimeUsedMS))
	}
	if !obj.UpdatedAt.IsZero() {
		lines = append(lines, "Last saved update: "+formatESMPanelUpdateTime(obj.UpdatedAt))
	}
	return wrapESMPanelLines(lines, width)
}

func (a *App) esmPanelNow(obj *esm.Objective) string {
	phase := effectiveESMPhase(obj)
	base := esmPhaseActivityLabel(phase, obj.Status)

	a.esmMu.Lock()
	id := a.esmActiveAgentID
	a.esmMu.Unlock()
	if id == "" {
		return base
	}
	act := a.agentActivities[id]
	if act == nil {
		return base + "; sub-agent is starting"
	}
	if act.LastTool != "" {
		return base + "; latest tool: " + act.LastTool
	}
	if act.LastResult != "" {
		return base + "; latest result: " + act.LastResult
	}
	if act.LastText != "" {
		return base + "; latest response: " + act.LastText
	}
	if act.LastThink != "" {
		return base + "; reasoning in progress"
	}
	return base + "; sub-agent is running"
}

func esmCompletedStages(phase esm.Phase) int {
	switch phase {
	case esm.PhaseCritic:
		return 1
	case esm.PhaseAudit:
		return 2
	case esm.PhaseComplete:
		return 3
	default:
		return 0
	}
}

func esmPanelProgress(obj *esm.Objective, phase esm.Phase) string {
	progress := fmt.Sprintf("Progress: %d/3 pipeline stages completed", esmCompletedStages(phase))
	if remaining := len(obj.RemainingWork); remaining > 0 {
		progress += fmt.Sprintf("; %d work item(s) remaining", remaining)
	}
	return progress
}

func esmPhaseActivityLabel(phase esm.Phase, status esm.Status) string {
	switch status {
	case esm.StatusPaused:
		return "ESM is paused"
	case esm.StatusBlocked:
		return "ESM is blocked"
	case esm.StatusBudgetLimited:
		return "ESM is waiting for more token budget"
	case esm.StatusUsageLimited:
		return "ESM is waiting for the provider limit to clear"
	case esm.StatusComplete:
		return "The objective has passed final audit"
	}
	switch phase {
	case esm.PhaseCritic:
		return "Critic is independently reviewing the worker evidence"
	case esm.PhaseAudit:
		return "Audit is independently verifying completion"
	default:
		return "Worker is investigating and implementing the objective"
	}
}

func esmPanelNextStep(obj *esm.Objective, phase esm.Phase) string {
	switch obj.Status {
	case esm.StatusPaused:
		return "Review the outstanding work, then run /esm resume"
	case esm.StatusBlocked:
		return "Resolve the blocker, then run /esm resume"
	case esm.StatusBudgetLimited:
		return "Raise or remove the token budget, then run /esm resume"
	case esm.StatusUsageLimited:
		return "Resolve the provider usage limit, then run /esm resume"
	case esm.StatusComplete:
		return "No further ESM work is scheduled"
	}
	switch phase {
	case esm.PhaseCritic:
		return "A passing critic review advances the candidate to final audit"
	case esm.PhaseAudit:
		return "A passing audit marks the objective complete; a failure returns it to the worker"
	default:
		return "Worker will record concrete progress and remaining work before the next run"
	}
}

func formatESMPanelUpdateTime(updatedAt time.Time) string {
	ago := time.Since(updatedAt)
	if ago < 0 {
		ago = 0
	}
	return updatedAt.Local().Format("2006-01-02 15:04:05") + " (" + formatDuration(ago) + " ago)"
}

func (a *App) activeESMPanelActivity(width int) []string {
	a.esmMu.Lock()
	id := a.esmActiveAgentID
	a.esmMu.Unlock()
	if id == "" {
		return nil
	}
	act := a.agentActivities[id]
	if act == nil {
		return []string{"  " + string(id) + " [starting]"}
	}
	lines := []string{fmt.Sprintf("  %s [%s]", id, act.State)}
	if act.LastTool != "" {
		lines = append(lines, "  Tool: "+act.LastTool)
	}
	if act.LastResult != "" {
		lines = append(lines, "  Latest: "+act.LastResult)
	} else if act.LastText != "" {
		lines = append(lines, "  Latest: "+act.LastText)
	} else if act.LastThink != "" {
		lines = append(lines, "  Thinking: "+act.LastThink)
	}
	return wrapESMPanelLines(lines, width)
}

func effectiveESMPhase(obj *esm.Objective) esm.Phase {
	if obj.Phase != "" {
		return obj.Phase
	}
	switch obj.Status {
	case esm.StatusComplete:
		return esm.PhaseComplete
	case esm.StatusCompleteCandidate:
		return esm.PhaseCritic
	default:
		return esm.PhaseWorker
	}
}

func esmPhaseLabel(phase esm.Phase) string {
	switch phase {
	case esm.PhaseCritic:
		return "Critic review"
	case esm.PhaseAudit:
		return "Final audit"
	case esm.PhaseComplete:
		return "Complete"
	default:
		return "Worker execution"
	}
}

func renderESMPipeline(phase esm.Phase, status esm.Status) string {
	stages := []struct {
		phase esm.Phase
		label string
	}{
		{esm.PhaseWorker, "Worker"},
		{esm.PhaseCritic, "Critic"},
		{esm.PhaseAudit, "Audit"},
	}
	current := esmPhaseIndex(phase)
	parts := make([]string, 0, len(stages))
	for i, stage := range stages {
		marker := " "
		switch {
		case phase == esm.PhaseComplete || i < current:
			marker = "x"
		case i == current && status == esm.StatusPaused:
			marker = "!"
		case i == current:
			marker = ">"
		}
		parts = append(parts, fmt.Sprintf("[%s] %s", marker, stage.label))
	}
	return strings.Join(parts, " -> ")
}

func esmPhaseIndex(phase esm.Phase) int {
	switch phase {
	case esm.PhaseCritic:
		return 1
	case esm.PhaseAudit:
		return 2
	case esm.PhaseComplete:
		return 3
	default:
		return 0
	}
}

func appendWrappedESMField(lines []string, label, value string, width int) []string {
	return append(lines, strings.Split(renderutil.WrapPlainText(label+": "+strings.TrimSpace(value), width), "\n")...)
}

func appendESMItems(lines []string, items []string, width int) []string {
	for i, item := range items {
		wrapped := renderutil.WrapPlainText(fmt.Sprintf("  %d. %s", i+1, item), width)
		lines = append(lines, strings.Split(wrapped, "\n")...)
	}
	return lines
}

func wrapESMPanelLines(lines []string, width int) []string {
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, strings.Split(renderutil.WrapPlainText(line, width), "\n")...)
	}
	return wrapped
}

func formatDurationMSForPanel(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return formatDuration(time.Duration(ms) * time.Millisecond)
}
