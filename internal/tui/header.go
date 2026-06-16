package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const vibeLogo = ` _    ___ __       
| |  / (_) /_  ___ 
| | / / / __ \/ _ \
| |/ / / /_/ /  __/
|___/_/_.___/\___/ 
                   `

func asciiLogoWidth() int {
	lines := strings.Split(vibeLogo, "\n")
	max := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > max {
			max = w
		}
	}
	return max
}

func renderHeader(width int, version string, providerName string, modelName string, cwd string) string {
	logoW := asciiLogoWidth()

	infoStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 0)

	line1 := lipgloss.NewStyle().Bold(true).Render(">_ Vibecoding (" + version + ")")
	line2 := providerName + " | " + modelName
	line3 := cwd

	infoContent := line1 + "\n" + line2 + "\n" + line3
	infoPanel := infoStyle.Render(infoContent)
	infoW := lipgloss.Width(infoPanel)

	gap := 2
	if width < logoW+infoW+gap+2 {
		// Responsive: show info panel only at full width
		fullInfo := infoStyle.Width(width - 2).Render(infoContent)
		return fullInfo
	}

	// Truncate cwd to fit
	available := width - logoW - gap - 4 // border chars
	if lipgloss.Width(line3) > available && available > 3 {
		line3 = lipgloss.NewStyle().MaxWidth(available).Render(cwd)
		infoContent = line1 + "\n" + line2 + "\n" + line3
		infoPanel = infoStyle.Render(infoContent)
	}

	logoStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(vibeLogo)
	return lipgloss.JoinHorizontal(lipgloss.Top, logoStyled, infoPanel)
}
