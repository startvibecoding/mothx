package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const mothxLogo = `██   ██  ███  ████ █  █ █  █
███ ███ █   █  ██  █  █  ██
█ ███ █ █   █  ██  ████  ██
█  █  █ █   █  ██  █  █ █  █
█     █  ███   ██  █  █ █  █`

const renameNotice = "Renamed: VibeCoding -> MothX. Use mothx."

func logoWidth() int {
	lines := strings.Split(mothxLogo, "\n")
	max := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > max {
			max = w
		}
	}
	return max
}

func renderHeader(width int, version string, providerName string, modelName string, cwd string) string {
	logoW := logoWidth()

	infoStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 0)

	line1 := lipgloss.NewStyle().Bold(true).Render("MothX (" + version + ")")
	line2 := providerName + " | " + modelName
	line3 := cwd

	infoContent := line1 + "\n" + line2 + "\n" + line3 + "\n" + renameNotice
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
		infoContent = line1 + "\n" + line2 + "\n" + line3 + "\n" + renameNotice
		infoPanel = infoStyle.Render(infoContent)
	}

	boxHeight := lipgloss.Height(infoPanel)
	logoStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Height(boxHeight).
		AlignVertical(lipgloss.Center).
		MarginRight(gap).
		Render(mothxLogo)
	return lipgloss.JoinHorizontal(lipgloss.Top, logoStyled, infoPanel)
}
