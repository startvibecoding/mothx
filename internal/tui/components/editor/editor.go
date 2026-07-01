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
// Enter submits, Alt+Enter / Ctrl+J inserts a newline.
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
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Padding(0, 1),
		cursorStyle: lipgloss.NewStyle().Background(lipgloss.Color("236")).Reverse(true),
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
	m.buf.MoveEndAll()
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

// Placeholder returns the placeholder text shown when the editor is empty.
func (m Model) Placeholder() string {
	return m.placeholder
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

// CursorEnd moves the cursor to the end of the buffer.
func (m Model) CursorEnd() Model {
	m.buf.MoveEndAll()
	return m
}

// InsertString inserts text at the current cursor position.
func (m Model) InsertString(s string) Model {
	m.buf.InsertString(s)
	return m
}

// AtFirstLine reports whether the cursor is on the first logical line.
func (m Model) AtFirstLine() bool {
	line, _ := m.buf.CursorPos()
	return line == 0
}

// AtLastLine reports whether the cursor is on the last logical line.
func (m Model) AtLastLine() bool {
	line, _ := m.buf.CursorPos()
	return line >= m.buf.LineCount()-1
}

// Init starts the cursor blink timer.
func (m Model) Init() tea.Cmd {
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
		m.cursorOn = true
		return m, nil

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
		if msg.Alt {
			m.buf.MoveWordLeft()
		} else {
			m.buf.MoveLeft()
		}
		return m, nil

	case tea.KeyRight:
		if msg.Alt {
			m.buf.MoveWordRight()
		} else {
			m.buf.MoveRight()
		}
		return m, nil

	case tea.KeyCtrlLeft:
		m.buf.MoveWordLeft()
		return m, nil

	case tea.KeyCtrlRight:
		m.buf.MoveWordRight()
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
	frameW := m.style.GetHorizontalFrameSize()
	contentW := m.width - frameW
	if contentW < 1 {
		contentW = 1
	}
	availW := contentW - promptW
	if availW < 1 {
		availW = 1
	}

	text := m.buf.Value()
	isEmpty := text == ""

	var displayLines []displayLine
	if isEmpty && !m.focus {
		// Empty, blurred: show nothing
		displayLines = []displayLine{{text: ""}}
	} else if isEmpty {
		displayLines = []displayLine{{text: m.renderEmptyLine()}}
	} else {
		displayLines = m.buildDisplayLines(availW)
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
		line := displayLines[i].text

		// Check if the cursor is on this display line
		if !isEmpty && m.focus && m.cursorOn && i == cursorDispLine {
			line = m.insertCursor(displayLines[i], cursorBufLine, cursorBufCol)
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

	cursorLine, cursorCol := m.buf.CursorPos()
	displayLines := m.buildDisplayLines(availW)
	for i, line := range displayLines {
		if line.bufLine != cursorLine {
			continue
		}
		if cursorCol >= line.startCol && cursorCol <= line.endCol {
			return i
		}
	}
	if len(displayLines) == 0 {
		return 0
	}
	return len(displayLines) - 1
}

// insertCursor inserts the cursor visual marker into a display line.
func (m Model) insertCursor(line displayLine, bufLine, bufCol int) string {
	if line.bufLine != bufLine {
		return line.text
	}
	runes := []rune(line.text)
	runePos := bufCol - line.startCol
	if runePos < 0 {
		runePos = 0
	}
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

// runeWidth returns the display width of a single rune.
func runeWidth(r rune) int {
	return displayWidth(string(r))
}

// renderEmptyLine renders placeholder and cursor without splitting ANSI escapes.
func (m Model) renderEmptyLine() string {
	if m.placeholder == "" {
		if m.cursorOn {
			return m.cursorStyle.Render(" ")
		}
		return " "
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(lipgloss.Color("236"))
	if !m.cursorOn {
		return dimStyle.Render(m.placeholder)
	}

	runes := []rune(m.placeholder)
	return m.cursorStyle.Render(string(runes[0])) + dimStyle.Render(string(runes[1:]))
}

// wrapLine wraps a single line to fit within width, returning display lines.
func wrapLine(line string, width int) []string {
	segments := wrapLineSegments(line, width, 0, 0)
	result := make([]string, 0, len(segments))
	for _, segment := range segments {
		result = append(result, segment.text)
	}
	return result
}

type displayLine struct {
	text     string
	bufLine  int
	startCol int
	endCol   int
}

func (m Model) buildDisplayLines(availW int) []displayLine {
	rawLines := strings.Split(m.buf.Value(), "\n")
	displayLines := make([]displayLine, 0, len(rawLines))
	for lineNum, line := range rawLines {
		displayLines = append(displayLines, wrapLineSegments(line, availW, lineNum, 0)...)
	}
	if len(displayLines) == 0 {
		return []displayLine{{text: ""}}
	}
	return displayLines
}

func wrapLineSegments(line string, width int, bufLine int, startCol int) []displayLine {
	if width <= 0 {
		return []displayLine{{text: line, bufLine: bufLine, startCol: startCol, endCol: startCol + len([]rune(line))}}
	}
	if displayWidth(line) <= width {
		return []displayLine{{text: line, bufLine: bufLine, startCol: startCol, endCol: startCol + len([]rune(line))}}
	}

	runes := []rune(line)
	var result []displayLine
	var current []rune
	currentW := 0
	segmentStart := startCol

	for i, r := range runes {
		rw := runeWidth(r)
		if currentW > 0 && currentW+rw > width {
			result = append(result, displayLine{
				text:     string(current),
				bufLine:  bufLine,
				startCol: segmentStart,
				endCol:   startCol + i,
			})
			segmentStart = startCol + i
			current = []rune{r}
			currentW = rw
		} else {
			current = append(current, r)
			currentW += rw
		}
	}
	if len(current) > 0 {
		result = append(result, displayLine{
			text:     string(current),
			bufLine:  bufLine,
			startCol: segmentStart,
			endCol:   startCol + len(runes),
		})
	}
	if len(result) == 0 {
		result = []displayLine{{text: "", bufLine: bufLine, startCol: startCol, endCol: startCol}}
	}
	return result
}
