package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/startvibecoding/mothx/internal/agent"
)

// renderAgentTabBar renders a horizontal tab bar showing all active agents.
func renderAgentTabBar(agentMgr *agent.AgentManager, activeID string, width int) string {
	if agentMgr == nil {
		return ""
	}

	ids := agentMgr.List()
	if len(ids) <= 1 {
		return ""
	}

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	stateIcon := func(state string) string {
		switch state {
		case "running":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("●")
		case "ready":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○")
		case "done":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("✓")
		case "error":
			return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
		default:
			return " "
		}
	}

	var tabs []string
	for _, id := range ids {
		st, ok := agentMgr.Status(id)
		state := ""
		if ok {
			state = st.State
		}

		name := string(id)
		label := "[ " + stateIcon(state) + " " + name + " ]"

		if string(id) == activeID {
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	row := strings.Join(tabs, " ")

	// Truncate to fit width
	if lipgloss.Width(row) > width {
		runes := []rune(row)
		if width > 3 && len(runes) > width-3 {
			row = string(runes[:width-3]) + "..."
		}
	}

	// Bottom border line
	border := lipgloss.NewStyle().
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Width(width)

	return lipgloss.JoinVertical(lipgloss.Left, row, border.Render(""))
}
