package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/vibecoding/internal/tui/components/suggest"
)

func commandSuggestionItems() []suggest.Item {
	return []suggest.Item{
		{Label: "/auth", Value: "/auth", Description: "Configure provider token, base URL, proxy and models"},
		{Label: "/mode", Value: "/mode ", Description: "Switch or show mode"},
		{Label: "/model", Value: "/model ", Description: "Switch or show model"},
		{Label: "/skills", Value: "/skills", Description: "List available skills"},
		{Label: "/skill", Value: "/skill ", Description: "Activate a skill"},
		{Label: "/clear", Value: "/clear", Description: "Clear conversation"},
		{Label: "/compact", Value: "/compact", Description: "Trigger context compaction"},
		{Label: "/sessions", Value: "/sessions", Description: "List/switch sessions"},
		{Label: "/init_mcp", Value: "/init_mcp ", Description: "Init mcp.json"},
		{Label: "/mcps", Value: "/mcps", Description: "List MCP servers"},
		{Label: "/delegate", Value: "/delegate ", Description: "Toggle delegation mode"},
		{Label: "/statusline", Value: "/statusline ", Description: "Inspect or toggle the TUI status line"},
		{Label: "/alloweditpath", Value: "/alloweditpath ", Description: "Manage auto-edit path whitelist (glob)"},
		{Label: "/allowautoedit", Value: "/allowautoedit ", Description: "Toggle full auto-edit in agent mode"},
		{Label: "/btw", Value: "/btw ", Description: "Ask a side question without touching the main task"},
		{Label: "/systeminit", Value: "/systeminit ", Description: "Generate/refresh AGENTS.md; optional guidance (e.g. ask me in Chinese)"},
		{Label: "/reload", Value: "/reload", Description: "Restart as a fresh process with a new session"},
		{Label: "/workflows", Value: "/workflows ", Description: "Workflow run commands"},
		{Label: "/agent", Value: "/agent ", Description: "Multi-agent commands"},
		{Label: "/cron", Value: "/cron ", Description: "Scheduled task commands"},
		{Label: "/help", Value: "/help", Description: "Show help"},
		{Label: "/quit", Value: "/quit", Description: "Exit"},
	}
}

func (a *App) updateCommandSuggestions() {
	value := a.input.Value()
	if a.auth.Open || a.toolModalOpen || a.waitingForApproval || a.waitingForQuestion || !strings.HasPrefix(value, "/") || strings.ContainsAny(value, " \t\n") {
		a.suggest = a.suggest.Update("")
		return
	}
	a.suggest = a.suggest.Update(value)
}

func (a *App) commandSuggestionsVisible() bool {
	return a.suggest.Visible()
}

func (a *App) applySelectedCommandSuggestion() bool {
	item, ok := a.suggest.Selected()
	if !ok {
		return false
	}
	a.input = a.input.SetValue(item.Value)
	a.input = a.input.CursorEnd()
	a.suggest = a.suggest.Update("")
	a.scheduleRender()
	return true
}

func (a *App) handleCommandSuggestionKey(msg tea.KeyMsg) bool {
	if !a.commandSuggestionsVisible() {
		return false
	}
	switch msg.Type {
	case tea.KeyUp:
		a.suggest = a.suggest.CursorUp()
		a.scheduleRender()
		return true
	case tea.KeyDown:
		a.suggest = a.suggest.CursorDown()
		a.scheduleRender()
		return true
	case tea.KeyTab:
		return a.applySelectedCommandSuggestion()
	}
	return false
}
