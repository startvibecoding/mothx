package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/mothx/internal/tui/components/suggest"
)

func commandSuggestionItems() []suggest.Item {
	return []suggest.Item{
		{Label: "/auth", Value: "/auth", Description: "Configure provider token, base URL, proxy and models"},
		{Label: "/settings", Value: "/settings", Description: "Configure settings.json groups, including providers"},
		{Label: "/mode", Value: "/mode ", Description: "Switch or show execution mode (plan/agent/yolo)"},
		{Label: "/model", Value: "/model ", Description: "Switch or show model"},
		{Label: "/defaultModel", Value: "/defaultModel ", Description: "Set default provider/model; defaults to global settings"},
		{Label: "/skills", Value: "/skills", Description: "List available skills"},
		{Label: "/skill", Value: "/skill ", Description: "Activate a skill"},
		{Label: "/paste-image", Value: "/paste-image", Description: "Paste clipboard image as a local file path"},
		{Label: "/clear", Value: "/clear", Description: "Clear conversation"},
		{Label: "/compact", Value: "/compact", Description: "Trigger context compaction"},
		{Label: "/sessions", Value: "/sessions", Description: "List/switch sessions"},
		{Label: "/init_mcp", Value: "/init_mcp ", Description: "Init mcp.json"},
		{Label: "/mcps", Value: "/mcps", Description: "List MCP servers"},
		{Label: "/delegate", Value: "/delegate ", Description: "Toggle delegation mode"},
		{Label: "/browser", Value: "/browser ", Description: "Toggle browser automation tool"},
		{Label: "/stats", Value: "/stats ", Description: "Open usage stats dashboard or TUI summary"},
		{Label: "/statusline", Value: "/statusline ", Description: "Inspect or toggle the TUI status line"},
		{Label: "/alloweditpath", Value: "/alloweditpath ", Description: "Manage auto-edit path whitelist (glob)"},
		{Label: "/allowautoedit", Value: "/allowautoedit ", Description: "Toggle full auto-edit in agent mode"},
		{Label: "/btw", Value: "/btw ", Description: "Ask a side question without touching the main task"},
		{Label: "/systeminit", Value: "/systeminit ", Description: "Generate/refresh AGENTS.md; optional guidance (e.g. ask me in Chinese)"},
		{Label: "/reload", Value: "/reload", Description: "Restart as a fresh process with a new session"},
		{Label: "/workflows", Value: "/workflows ", Description: "Workflow run commands"},
		{Label: "/agent", Value: "/agent ", Description: "Multi-agent commands (not execution mode)"},
		{Label: "/cron", Value: "/cron ", Description: "Scheduled task commands"},
		{Label: "/help", Value: "/help", Description: "Show help"},
		{Label: "/quit", Value: "/quit", Description: "Exit"},
	}
}

func (a *App) updateCommandSuggestions() {
	value := a.input.Value()
	items, query, ok := commandSuggestionItemsForInput(value)
	if a.auth.Open || a.defaultModelDialog.Open || a.modelDialog.Open || a.sessionsDialog.Open || a.toolModalOpen || a.statsOverlayOpen || a.waitingForApproval || a.waitingForQuestion || !ok {
		a.suggest = a.suggest.SetItems(commandSuggestionItems())
		a.suggest = a.suggest.Update("")
		return
	}
	a.suggest = a.suggest.SetItems(items)
	a.suggest = a.suggest.Update(query)
}

func (a *App) commandSuggestionsVisible() bool {
	return a.suggest.Visible()
}

func (a *App) commandInputActive() bool {
	value := a.input.Value()
	return strings.HasPrefix(value, "/") && !strings.Contains(value, "\n")
}

func (a *App) commandNameInputActive() bool {
	value := a.input.Value()
	return strings.HasPrefix(value, "/") && !strings.ContainsAny(value, " \t\n")
}

func (a *App) applySelectedCommandSuggestion() bool {
	item, ok := a.suggest.Selected()
	if !ok {
		return false
	}
	if item.Value == a.input.Value() {
		return false
	}
	a.input = a.input.SetValue(item.Value)
	a.input = a.input.CursorEnd()
	a.updateCommandSuggestions()
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

func commandSuggestionItemsForInput(value string) ([]suggest.Item, string, bool) {
	if !strings.HasPrefix(value, "/") || strings.Contains(value, "\n") {
		return nil, "", false
	}
	if !strings.ContainsAny(value, " \t") {
		return commandSuggestionItems(), value, true
	}

	items := commandArgumentSuggestionItems(value)
	if len(items) == 0 {
		return nil, "", false
	}
	return items, value, true
}

func commandArgumentSuggestionItems(value string) []suggest.Item {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}
	cmd := fields[0]
	argIndex := len(fields) - 1
	if strings.HasSuffix(value, " ") || strings.HasSuffix(value, "\t") {
		argIndex = len(fields)
	}
	if argIndex < 1 {
		argIndex = 1
	}

	switch cmd {
	case "/mode":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"plan", "agent", "yolo"})
		}
	case "/defaultModel":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"project", "global"})
		}
	case "/sessions":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"ls", "set", "clear", "del"})
		}
	case "/delegate":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"on", "off", "status"})
		}
	case "/browser":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"on", "off", "status"})
		}
	case "/stats":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"server", "stop-server", "tui"})
		}
	case "/alloweditpath":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"add", "remove", "clear"})
		}
	case "/allowautoedit":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"on", "off"})
		}
		if argIndex == 2 && len(fields) >= 2 && (fields[1] == "on" || fields[1] == "off") {
			return commandArgumentItems(cmd+" "+fields[1], []string{"global"})
		}
	case "/statusline":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"status", "on", "off", "command", "refresh"})
		}
		if argIndex == 2 && len(fields) >= 2 && (fields[1] == "on" || fields[1] == "off") {
			return commandArgumentItems(cmd+" "+fields[1], []string{"project", "global"})
		}
	case "/agent":
		if argIndex == 1 {
			return commandArgumentItems(cmd, []string{"list", "switch", "destroy"})
		}
	}

	return nil
}

func commandArgumentItems(prefix string, args []string) []suggest.Item {
	items := make([]suggest.Item, 0, len(args))
	for _, arg := range args {
		value := prefix + " " + arg
		items = append(items, suggest.Item{
			Label: value,
			Value: value,
		})
	}
	return items
}
