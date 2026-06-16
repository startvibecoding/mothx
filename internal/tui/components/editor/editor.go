package editor

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// cursorBlinkMsg is sent to toggle cursor visibility.
type cursorBlinkMsg time.Time

const cursorBlinkInterval = 530 * time.Millisecond

// SubmitMsg is sent when the user presses Enter to submit.
type SubmitMsg struct{}

// Model is a multi-line text editor Bubble Tea component.
// Enter submits, Shift+Enter / Ctrl+J inserts a newline.
type Model struct {
	buf         *buffer
	focus       bool
	cursorOn    bool
	width       int
	maxLines    int
	placeholder string
	prompt      string
	style       lipgloss.Style
	cursorStyle lipgloss.Style
}

// New creates a new editor Model.
func New(width int) Model {
	return Model{
		buf:         newBuffer(),
		focus:       true,
		cursorOn:    true,
		width:       width,
		maxLines:    5,
		placeholder: "Type a message...",
		prompt:      "",
		style:       lipgloss.NewStyle(),
		cursorStyle: lipgloss.NewStyle().Reverse(true),
	}
}

// Focus activates the editor.
func (m Model) Focus() Model {
	m.focus = true
	m.cursorOn = true
	return m
}

// Blur deactivates the editor.
func (m Model) Blur() Model {
	m.focus = false
	return m
}

// Focused returns whether the editor has focus.
func (m Model) Focused() bool {
	return m.focus
}

// Value returns the full text content.
func (m Model) Value() string {
	return m.buf.Value()
}

// SetValue replaces the editor content.
func (m Model) SetValue(text string) Model {
	m.buf.SetValue(text)
	return m
}

// Reset clears the editor content and cursor.
func (m Model) Reset() Model {
	m.buf.Reset()
	return m
}

// SetWidth sets the editor display width.
func (m Model) SetWidth(w int) Model {
	m.width = w
	return m
}

// SetMaxLines sets the maximum visible lines.
func (m Model) SetMaxLines(n int) Model {
	m.maxLines = n
	return m
}

// SetPlaceholder sets the placeholder text shown when empty.
func (m Model) SetPlaceholder(s string) Model {
	m.placeholder = s
	return m
}

// SetPrompt sets the prompt prefix (e.g. "> ").
func (m Model) SetPrompt(s string) Model {
	m.prompt = s
	return m
}

// SetStyle sets the editor style.
func (m Model) SetStyle(s lipgloss.Style) Model {
	m.style = s
	return m
}

// SetCursorStyle sets the cursor style.
func (m Model) SetCursorStyle(s lipgloss.Style) Model {
	m.cursorStyle = s
	return m
}

// LineCount returns the current number of lines.
func (m Model) LineCount() int {
	return m.buf.LineCount()
}

// CursorPos returns the cursor position (line, col).
func (m Model) CursorPos() (int, int) {
	return m.buf.CursorPos()
}

// Init starts the cursor blink timer.
func (m Model) Init() tea.Cmd {
	if m.focus {
		return blinkCursor()
	}
	return nil
}

func blinkCursor() tea.Cmd {
	return tea.Tick(cursorBlinkInterval, func(t time.Time) tea.Msg {
		return cursorBlinkMsg(t)
	})
}

// Update processes messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	switch msg := msg.(type) {
	case cursorBlinkMsg:
		m.cursorOn = !m.cursorOn
		return m, blinkCursor()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Alt+Enter inserts a newline (works on most terminals)
	if msg.Type == tea.KeyEnter && msg.Alt {
		m.buf.InsertNewline()
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		// Enter submits
		return m, func() tea.Msg { return SubmitMsg{} }

	case tea.KeyCtrlJ:
		// Ctrl+J also inserts a newline
		m.buf.InsertNewline()
		return m, nil

	case tea.KeyBackspace:
		m.buf.DeleteBack()
		return m, nil

	case tea.KeyDelete:
		m.buf.DeleteForward()
		return m, nil

	case tea.KeyLeft:
		m.buf.MoveLeft()
		return m, nil

	case tea.KeyRight:
		m.buf.MoveRight()
		return m, nil

	case tea.KeyUp:
		m.buf.MoveUp()
		return m, nil

	case tea.KeyDown:
		m.buf.MoveDown()
		return m, nil

	case tea.KeyHome, tea.KeyCtrlA:
		m.buf.MoveHome()
		return m, nil

	case tea.KeyEnd, tea.KeyCtrlE:
		m.buf.MoveEnd()
		return m, nil

	case tea.KeyCtrlK:
		m.buf.DeleteToLineEnd()
		return m, nil

	case tea.KeyCtrlU:
		m.buf.DeleteToLineStart()
		return m, nil

	case tea.KeyCtrlW:
		m.buf.DeleteWordBack()
		return m, nil

	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if r == '\n' {
				m.buf.InsertNewline()
			} else {
				m.buf.InsertRune(r)
			}
		}
		return m, nil

	case tea.KeySpace:
		m.buf.InsertRune(' ')
		return m, nil

	case tea.KeyTab:
		// Tab inserts two spaces (used for indentation, not submission)
		m.buf.InsertString("  ")
		return m, nil
	}

	return m, nil
}

// View renders the editor.
func (m Model) View() string {
	promptW := displayWidth(m.prompt)
	availW := m.width - promptW
	if availW < 1 {
		availW = 1
	}

	text := m.buf.Value()
	isEmpty := text == ""

	var displayLines []string
	if isEmpty && !m.focus {
		// Empty, blurred: show nothing
		displayLines = []string{""}
	} else if isEmpty {
		// Empty, focused: show placeholder with cursor
		displayLines = []string{m.renderPlaceholder()}
	} else {
		rawLines := strings.Split(text, "\n")
		for _, line := range rawLines {
			wrapped := wrapLine(line, availW)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			displayLines = append(displayLines, wrapped...)
		}
	}

	// Determine visible range
	maxVis := m.maxLines
	if maxVis < 1 {
		maxVis = 1
	}
	totalLines := len(displayLines)

	// Compute the display line index of the cursor
	cursorDispLine := m.cursorDisplayLine(availW)

	// Window the visible lines around the cursor
	startLine := 0
	if totalLines > maxVis {
		startLine = cursorDispLine - maxVis/2
		if startLine < 0 {
			startLine = 0
		}
		if startLine+maxVis > totalLines {
			startLine = totalLines - maxVis
		}
	}
	endLine := startLine + maxVis
	if endLine > totalLines {
		endLine = totalLines
	}

	// Render visible lines with cursor
	var renderedLines []string
	cursorBufLine, cursorBufCol := m.buf.CursorPos()

	for i := startLine; i < endLine; i++ {
		line := displayLines[i]

		// Check if the cursor is on this display line
		if m.focus && m.cursorOn && i == cursorDispLine {
			line = m.insertCursor(line, cursorBufLine, cursorBufCol, availW)
		}

		renderedLines = append(renderedLines, m.prompt+line)
	}

	view := strings.Join(renderedLines, "\n")
	return m.style.Width(m.width).Render(view)
}

// cursorDisplayLine computes which display line the cursor is on,
// accounting for line wrapping.
func (m Model) cursorDisplayLine(availW int) int {
	text := m.buf.Value()
	if text == "" {
		return 0
	}

	rawLines := strings.Split(text, "\n")
	cursorLine, cursorCol := m.buf.CursorPos()
	dispLine := 0

	for i, line := range rawLines {
		if i == cursorLine {
			// Calculate which wrapped line the cursor is on
			remaining := cursorCol
			wrapped := wrapLine(line, availW)
			for _, wl := range wrapped {
				wlRunes := len([]rune(wl))
				if remaining <= wlRunes {
					return dispLine
				}
				remaining -= wlRunes
				dispLine++
			}
			return dispLine
		}
		wrapped := wrapLine(line, availW)
		dispLine += len(wrapped)
	}
	return dispLine
}

// insertCursor inserts the cursor visual marker into a display line.
func (m Model) insertCursor(line string, bufLine, bufCol, availW int) string {
	runes := []rune(line)

	// Calculate the position within this display line
	// For the cursor's buffer line, find the offset within this wrapped segment
	text := m.buf.Value()
	rawLines := strings.Split(text, "\n")
	if bufLine >= len(rawLines) {
		return line
	}

	lineRunes := []rune(rawLines[bufLine])
	cursorColInLine := bufCol
	if cursorColInLine > len(lineRunes) {
		cursorColInLine = len(lineRunes)
	}

	// For single-line case or when cursor is within this wrapped segment
	displayCol := displayWidth(string(lineRunes[:cursorColInLine]))
	if displayCol > len(runes) {
		displayCol = len(runes)
	}

	// Place cursor at the rune position corresponding to display column
	runePos := runePosForDisplayCol(runes, displayCol)
	if runePos > len(runes) {
		runePos = len(runes)
	}

	if runePos < len(runes) {
		// Cursor on a character: reverse that character
		ch := string(runes[runePos])
		before := string(runes[:runePos])
		after := string(runes[runePos+1:])
		return before + m.cursorStyle.Render(ch) + after
	}
	// Cursor at end of line: render a space with cursor style
	return string(runes) + m.cursorStyle.Render(" ")
}

// runePosForDisplayCol converts a display column to a rune index.
func runePosForDisplayCol(runes []rune, dispCol int) int {
	w := 0
	for i, r := range runes {
		if w >= dispCol {
			return i
		}
		w += runeWidth(r)
	}
	return len(runes)
}

// runeWidth returns the display width of a single rune.
func runeWidth(r rune) int {
	return displayWidth(string(r))
}

// renderPlaceholder renders the placeholder text with dim styling.
func (m Model) renderPlaceholder() string {
	if m.placeholder == "" {
		return m.cursorStyle.Render(" ")
	}
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return dimStyle.Render(m.placeholder[:1]) + dimStyle.Render(m.placeholder[1:])
}

// wrapLine wraps a single line to fit within width, returning display lines.
func wrapLine(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	if displayWidth(line) <= width {
		return []string{line}
	}

	runes := []rune(line)
	var result []string
	current := ""
	currentW := 0

	for _, r := range runes {
		rw := runeWidth(r)
		if currentW+rw > width {
			result = append(result, current)
			current = string(r)
			currentW = rw
		} else {
			current += string(r)
			currentW += rw
		}
	}
	if current != "" {
		result = append(result, current)
	}
	if len(result) == 0 {
		result = []string{""}
	}
	return result
}
