package vscroll

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// mouseWheelScrollLines is the number of lines scrolled per mouse wheel event
// or arrow key press.
const mouseWheelScrollLines = 3

// Model is a virtualized scroll list that only renders visible items.
type Model struct {
	items        []string
	width        int
	height       int
	offset       int
	followBottom bool
	style        lipgloss.Style
}

// New creates a new Model with the given viewport dimensions.
func New(width, height int) Model {
	return Model{
		width:        width,
		height:       height,
		followBottom: true,
	}
}

// SetSize returns a new Model with updated viewport dimensions.
func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

// SetItems replaces all items and returns the updated Model.
func (m Model) SetItems(items []string) Model {
	m.items = items
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

// AppendItem adds one item to the end. If followBottom is true, the view
// auto-scrolls to show the new content.
func (m Model) AppendItem(item string) Model {
	m.items = append(m.items, item)
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	return m
}

// GotoBottom scrolls to the very bottom.
func (m Model) GotoBottom() Model {
	m.followBottom = true
	m.offset = m.maxOffset()
	return m
}

// GotoTop scrolls to the very top.
func (m Model) GotoTop() Model {
	m.followBottom = false
	m.offset = 0
	return m
}

// AtBottom reports whether the viewport is scrolled to the bottom.
func (m Model) AtBottom() bool {
	return m.offset >= m.maxOffset()
}

// ItemCount returns the number of items.
func (m Model) ItemCount() int {
	return len(m.items)
}

// Update handles keyboard and mouse messages for scrolling.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup":
			m.followBottom = false
			m.offset -= m.height
			m.clampOffset()
		case "pgdown":
			m.offset += m.height
			if m.AtBottom() {
				m.followBottom = true
			}
			m.clampOffset()
		case "up":
			m.followBottom = false
			m.offset -= mouseWheelScrollLines
			m.clampOffset()
		case "down":
			m.offset += mouseWheelScrollLines
			if m.AtBottom() {
				m.followBottom = true
			}
			m.clampOffset()
		case "home":
			m = m.GotoTop()
		case "end":
			m = m.GotoBottom()
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			break
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.followBottom = false
			m.offset -= mouseWheelScrollLines
			m.clampOffset()
		case tea.MouseButtonWheelDown:
			m.offset += mouseWheelScrollLines
			if m.AtBottom() {
				m.followBottom = true
			}
			m.clampOffset()
		}
	}
	return m, nil
}

// View renders the visible window of the scroll list.
func (m Model) View() string {
	if m.height <= 0 || m.width <= 0 {
		return ""
	}

	lines := m.displayLines()

	// Apply followBottom before slicing.
	offset := m.offset
	if m.followBottom {
		offset = max(0, len(lines)-m.height)
	}

	// Slice the visible window.
	var visible []string
	if offset >= len(lines) {
		visible = make([]string, 0)
	} else {
		end := offset + m.height
		if end > len(lines) {
			end = len(lines)
		}
		visible = lines[offset:end]
	}

	// Pad or trim to exactly height lines, and truncate each to width.
	var b strings.Builder
	for i := 0; i < m.height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		var line string
		if i < len(visible) {
			line = visible[i]
		}
		line = m.fitLine(line)
		b.WriteString(line)
	}

	return m.style.Render(b.String())
}

// SetStyle returns a new Model with the given lipgloss style applied.
func (m Model) SetStyle(style lipgloss.Style) Model {
	m.style = style
	return m
}

// displayLines flattens all items into individual display lines.
// Items are separated by a blank line (empty string between items).
func (m Model) displayLines() []string {
	if len(m.items) == 0 {
		return nil
	}
	var lines []string
	for i, item := range m.items {
		if i > 0 {
			// Blank separator line between items.
			lines = append(lines, "")
		}
		parts := strings.Split(item, "\n")
		lines = append(lines, parts...)
	}
	return lines
}

// fitLine truncates or pads a line to exactly m.width display columns.
func (m Model) fitLine(line string) string {
	w := xansi.StringWidth(line)
	if w > m.width {
		return xansi.Truncate(line, m.width, "")
	}
	if w < m.width {
		return line + strings.Repeat(" ", m.width-w)
	}
	return line
}

// maxOffset returns the maximum valid scroll offset.
func (m Model) maxOffset() int {
	total := len(m.displayLines())
	v := total - m.height
	if v < 0 {
		return 0
	}
	return v
}

// clampOffset ensures offset is within [0, maxOffset].
func (m *Model) clampOffset() {
	max := m.maxOffset()
	if m.offset < 0 {
		m.offset = 0
	}
	if m.offset > max {
		m.offset = max
	}
}
