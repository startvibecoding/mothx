package tui

import (
	"fmt"
	"strings"
	"time"
)

// formatTokenCount formats a token count as a human-friendly string:
// "500", "1.2k", "3.5M", etc.
func formatTokenCount(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		k := float64(n) / 1000
		if k < 10 {
			return fmt.Sprintf("%.1fk", k)
		}
		return fmt.Sprintf("%.0fk", k)
	default:
		m := float64(n) / 1_000_000
		if m < 10 {
			return fmt.Sprintf("%.1fM", m)
		}
		return fmt.Sprintf("%.0fM", m)
	}
}

// renderLoadingIndicator builds the streaming status line shown in the footer.
// Returns "" when not thinking. Uses package-level spinnerChars, statusStyle,
// and warningStyle from app.go. Reuses formatDuration for elapsed formatting.
func renderLoadingIndicator(isThinking bool, spinnerIndex int, elapsed time.Duration, streamingTokens int, width int) string {
	if !isThinking {
		return ""
	}

	char := spinnerChars[spinnerIndex%len(spinnerChars)]
	phrase := "Thinking..."

	var parts []string
	parts = append(parts, formatDuration(elapsed))
	if streamingTokens > 0 {
		parts = append(parts, fmt.Sprintf("↓ %s tokens", formatTokenCount(streamingTokens)))
	}
	parts = append(parts, "esc to cancel")

	parenthetical := statusStyle.Render("(" + strings.Join(parts, " · ") + ")")
	spinnerPart := warningStyle.Render(char)

	line := fmt.Sprintf("%s %s %s", spinnerPart, phrase, parenthetical)
	return line
}
