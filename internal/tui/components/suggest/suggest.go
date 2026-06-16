package suggest

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Item represents a single autocomplete suggestion.
type Item struct {
	Label       string // display text
	Description string // optional description
	Value       string // the actual value to insert
}

// Model is the autocomplete suggestion dropdown component.
type Model struct {
	items         []Item
	filtered      []Item
	cursor        int
	maxVisible    int
	visible       bool
	query         string
	width         int
	style         lipgloss.Style
	selectedStyle lipgloss.Style
}

// New creates a new suggestion Model with default settings.
func New(width int) Model {
	return Model{
		maxVisible: 8,
		width:      width,
		style:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true),
	}
}

// SetItems sets the available completion items and re-filters.
func (m Model) SetItems(items []Item) Model {
	m.items = items
	return m.filter()
}

// SetWidth sets the dropdown width.
func (m Model) SetWidth(width int) Model {
	m.width = width
	return m
}

// Update updates the filter based on the current input query.
func (m Model) Update(query string) Model {
	m.query = query
	return m.filter()
}

// Visible returns whether the dropdown should be shown.
func (m Model) Visible() bool {
	return m.visible
}

// Selected returns the currently selected item, or false if nothing is selected.
func (m Model) Selected() (Item, bool) {
	if len(m.filtered) == 0 {
		return Item{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return Item{}, false
	}
	return m.filtered[m.cursor], true
}

// CursorUp moves the selection up, wrapping to the bottom at the top.
func (m Model) CursorUp() Model {
	if len(m.filtered) == 0 {
		return m
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.filtered) - 1
	}
	return m
}

// CursorDown moves the selection down, wrapping to the top at the bottom.
func (m Model) CursorDown() Model {
	if len(m.filtered) == 0 {
		return m
	}
	m.cursor++
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
	return m
}

// View renders the suggestion dropdown.
func (m Model) View() string {
	if !m.visible || len(m.filtered) == 0 {
		return ""
	}

	contentWidth := m.width - 2 // account for border left+right
	if contentWidth < 1 {
		contentWidth = 1
	}

	var b strings.Builder

	start := 0
	end := len(m.filtered)
	hasMore := false

	if end > m.maxVisible {
		// scroll window centered on cursor
		start = m.cursor - m.maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + m.maxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
			start = end - m.maxVisible
			if start < 0 {
				start = 0
			}
		}
		hasMore = true
	}

	for i := start; i < end; i++ {
		item := m.filtered[i]
		line := m.renderItem(item, i == m.cursor, contentWidth)
		b.WriteString(line)
		if i < end-1 {
			b.WriteByte('\n')
		}
	}

	if hasMore {
		indicator := "  ↑↓ more"
		padded := indicator + strings.Repeat(" ", contentWidth-lipgloss.Width(indicator))
		b.WriteByte('\n')
		b.WriteString(m.style.Render(padded))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(contentWidth).
		Render(b.String())
}

// renderItem formats a single item line.
func (m Model) renderItem(item Item, selected bool, maxWidth int) string {
	label := item.Label
	desc := ""
	if item.Description != "" {
		desc = " " + item.Description
	}

	line := label + desc
	if lipgloss.Width(line) > maxWidth {
		line = truncate(line, maxWidth)
	} else {
		line = line + strings.Repeat(" ", maxWidth-lipgloss.Width(line))
	}

	if selected {
		return m.selectedStyle.Render(line)
	}
	return m.style.Render(line)
}

// filter applies prefix matching and updates visibility.
func (m Model) filter() Model {
	q := strings.ToLower(m.query)

	if q == "" {
		m.filtered = m.items
		m.visible = false
		m.cursor = clampCursor(m.cursor, m.filtered)
		return m
	}

	var matched []Item
	for _, item := range m.items {
		if strings.HasPrefix(strings.ToLower(item.Label), q) ||
			strings.HasPrefix(strings.ToLower(item.Value), q) {
			matched = append(matched, item)
		}
	}

	m.filtered = matched
	m.visible = len(matched) > 0
	m.cursor = clampCursor(m.cursor, m.filtered)
	return m
}

// clampCursor ensures the cursor is within bounds.
func clampCursor(cursor int, items []Item) int {
	if len(items) == 0 {
		return 0
	}
	if cursor >= len(items) {
		return len(items) - 1
	}
	if cursor < 0 {
		return 0
	}
	return cursor
}

// truncate shortens a string to fit within maxWidth, appending "..." if needed.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	return string(runes[:maxWidth-3]) + "..."
}
