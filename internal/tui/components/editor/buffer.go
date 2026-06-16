package editor

import (
	"strings"
	"unicode"

	xansi "github.com/charmbracelet/x/ansi"
)

// buffer is a Unicode-aware multi-line text buffer.
// Text is stored as a slice of lines (no trailing newline in each line).
// Cursor position is tracked as (cursorLine, cursorCol) where cursorCol is
// the rune offset within the line.
type buffer struct {
	lines          []string
	cursorLine     int
	cursorCol      int
	preferredCol   int  // preferred column for up/down navigation
	maxHeight      int  // 0 = unlimited
	width          int  // display width for wrapping
}

func newBuffer() *buffer {
	return &buffer{
		lines: []string{""},
	}
}

// Value returns the full text content.
func (b *buffer) Value() string {
	return strings.Join(b.lines, "\n")
}

// SetValue replaces all content and resets cursor.
func (b *buffer) SetValue(text string) {
	if text == "" {
		b.lines = []string{""}
	} else {
		b.lines = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	}
	b.cursorLine = 0
	b.cursorCol = 0
	b.preferredCol = 0
}

// Reset clears all content and cursor position.
func (b *buffer) Reset() {
	b.lines = []string{""}
	b.cursorLine = 0
	b.cursorCol = 0
	b.preferredCol = 0
}

// LineCount returns the number of lines.
func (b *buffer) LineCount() int {
	return len(b.lines)
}

// RuneCount returns the total number of runes in the buffer.
func (b *buffer) RuneCount() int {
	n := 0
	for i, line := range b.lines {
		n += len([]rune(line))
		if i < len(b.lines)-1 {
			n++ // newline character
		}
	}
	return n
}

// InsertRune inserts a single rune at the cursor position.
func (b *buffer) InsertRune(r rune) {
	b.clampCursor()
	line := b.lines[b.cursorLine]
	runes := []rune(line)
	if b.cursorCol > len(runes) {
		b.cursorCol = len(runes)
	}
	runes = append(runes[:b.cursorCol], append([]rune{r}, runes[b.cursorCol:]...)...)
	b.lines[b.cursorLine] = string(runes)
	b.cursorCol++
	b.preferredCol = b.cursorCol
}

// InsertString inserts a string at the cursor position, handling newlines.
func (b *buffer) InsertString(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if !strings.Contains(s, "\n") {
		for _, r := range s {
			b.InsertRune(r)
		}
		return
	}

	b.clampCursor()
	line := b.lines[b.cursorLine]
	runes := []rune(line)
	col := b.cursorCol
	if col > len(runes) {
		col = len(runes)
	}

	before := string(runes[:col])
	after := string(runes[col:])

	parts := strings.Split(s, "\n")
	newLines := make([]string, 0, len(parts)+1)
	newLines = append(newLines, before+parts[0])
	for i := 1; i < len(parts)-1; i++ {
		newLines = append(newLines, parts[i])
	}
	newLines = append(newLines, parts[len(parts)-1]+after)

	// Replace current line with expanded lines
	result := make([]string, 0, len(b.lines)+len(newLines)-1)
	result = append(result, b.lines[:b.cursorLine]...)
	result = append(result, newLines...)
	result = append(result, b.lines[b.cursorLine+1:]...)
	b.lines = result

	b.cursorLine += len(newLines) - 1
	b.cursorCol = len([]rune(parts[len(parts)-1]))
	b.preferredCol = b.cursorCol
}

// InsertNewline splits the current line at the cursor.
func (b *buffer) InsertNewline() {
	b.clampCursor()
	line := b.lines[b.cursorLine]
	runes := []rune(line)
	col := b.cursorCol
	if col > len(runes) {
		col = len(runes)
	}

	before := string(runes[:col])
	after := string(runes[col:])

	result := make([]string, 0, len(b.lines)+1)
	result = append(result, b.lines[:b.cursorLine]...)
	result = append(result, before, after)
	result = append(result, b.lines[b.cursorLine+1:]...)
	b.lines = result

	b.cursorLine++
	b.cursorCol = 0
	b.preferredCol = 0
}

// DeleteBack removes the character before the cursor (Backspace).
func (b *buffer) DeleteBack() {
	b.clampCursor()
	if b.cursorCol > 0 {
		runes := []rune(b.lines[b.cursorLine])
		b.lines[b.cursorLine] = string(append(runes[:b.cursorCol-1], runes[b.cursorCol:]...))
		b.cursorCol--
		b.preferredCol = b.cursorCol
	} else if b.cursorLine > 0 {
		// Merge with previous line
		prevLine := b.lines[b.cursorLine-1]
		currLine := b.lines[b.cursorLine]
		b.cursorCol = len([]rune(prevLine))
		b.lines[b.cursorLine-1] = prevLine + currLine
		b.lines = append(b.lines[:b.cursorLine], b.lines[b.cursorLine+1:]...)
		b.cursorLine--
		b.preferredCol = b.cursorCol
	}
}

// DeleteForward removes the character at the cursor (Delete).
func (b *buffer) DeleteForward() {
	b.clampCursor()
	runes := []rune(b.lines[b.cursorLine])
	if b.cursorCol < len(runes) {
		b.lines[b.cursorLine] = string(append(runes[:b.cursorCol], runes[b.cursorCol+1:]...))
	} else if b.cursorLine < len(b.lines)-1 {
		// Merge with next line
		currLine := b.lines[b.cursorLine]
		nextLine := b.lines[b.cursorLine+1]
		b.lines[b.cursorLine] = currLine + nextLine
		b.lines = append(b.lines[:b.cursorLine+1], b.lines[b.cursorLine+2:]...)
	}
}

// DeleteToLineEnd removes text from cursor to end of line (Ctrl+K).
func (b *buffer) DeleteToLineEnd() {
	b.clampCursor()
	runes := []rune(b.lines[b.cursorLine])
	if b.cursorCol < len(runes) {
		b.lines[b.cursorLine] = string(runes[:b.cursorCol])
	}
}

// DeleteToLineStart removes text from start of line to cursor (Ctrl+U).
func (b *buffer) DeleteToLineStart() {
	b.clampCursor()
	runes := []rune(b.lines[b.cursorLine])
	if b.cursorCol > 0 {
		b.lines[b.cursorLine] = string(runes[b.cursorCol:])
		b.cursorCol = 0
		b.preferredCol = 0
	}
}

// DeleteWordBack removes the word before the cursor (Ctrl+W).
func (b *buffer) DeleteWordBack() {
	b.clampCursor()
	if b.cursorCol == 0 {
		b.DeleteBack()
		return
	}

	runes := []rune(b.lines[b.cursorLine])
	end := b.cursorCol

	// Skip trailing spaces
	start := end
	for start > 0 && unicode.IsSpace(runes[start-1]) {
		start--
	}
	// Skip word characters
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	newRunes := make([]rune, 0, len(runes)-(end-start))
	newRunes = append(newRunes, runes[:start]...)
	newRunes = append(newRunes, runes[end:]...)
	b.lines[b.cursorLine] = string(newRunes)
	b.cursorCol = start
	b.preferredCol = b.cursorCol
}

// MoveLeft moves cursor one character left.
func (b *buffer) MoveLeft() {
	b.clampCursor()
	if b.cursorCol > 0 {
		b.cursorCol--
		b.preferredCol = b.cursorCol
	} else if b.cursorLine > 0 {
		b.cursorLine--
		b.cursorCol = len([]rune(b.lines[b.cursorLine]))
		b.preferredCol = b.cursorCol
	}
}

// MoveRight moves cursor one character right.
func (b *buffer) MoveRight() {
	b.clampCursor()
	lineLen := len([]rune(b.lines[b.cursorLine]))
	if b.cursorCol < lineLen {
		b.cursorCol++
		b.preferredCol = b.cursorCol
	} else if b.cursorLine < len(b.lines)-1 {
		b.cursorLine++
		b.cursorCol = 0
		b.preferredCol = 0
	}
}

// MoveUp moves cursor one line up.
func (b *buffer) MoveUp() bool {
	b.clampCursor()
	if b.cursorLine == 0 {
		return false
	}
	b.cursorLine--
	lineLen := len([]rune(b.lines[b.cursorLine]))
	if b.preferredCol > lineLen {
		b.cursorCol = lineLen
	} else {
		b.cursorCol = b.preferredCol
	}
	return true
}

// MoveDown moves cursor one line down.
func (b *buffer) MoveDown() bool {
	b.clampCursor()
	if b.cursorLine >= len(b.lines)-1 {
		return false
	}
	b.cursorLine++
	lineLen := len([]rune(b.lines[b.cursorLine]))
	if b.preferredCol > lineLen {
		b.cursorCol = lineLen
	} else {
		b.cursorCol = b.preferredCol
	}
	return true
}

// MoveHome moves cursor to the start of the current line.
func (b *buffer) MoveHome() {
	b.cursorCol = 0
	b.preferredCol = 0
}

// MoveEnd moves cursor to the end of the current line.
func (b *buffer) MoveEnd() {
	b.cursorCol = len([]rune(b.lines[b.cursorLine]))
	b.preferredCol = b.cursorCol
}

// CursorPos returns the current cursor position as (line, col).
func (b *buffer) CursorPos() (int, int) {
	return b.cursorLine, b.cursorCol
}

// clampCursor ensures the cursor is within valid bounds.
func (b *buffer) clampCursor() {
	if b.cursorLine < 0 {
		b.cursorLine = 0
	}
	if b.cursorLine >= len(b.lines) {
		b.cursorLine = len(b.lines) - 1
	}
	lineLen := len([]rune(b.lines[b.cursorLine]))
	if b.cursorCol < 0 {
		b.cursorCol = 0
	}
	if b.cursorCol > lineLen {
		b.cursorCol = lineLen
	}
}

// displayWidth returns the display width of a string in terminal cells.
func displayWidth(s string) int {
	return xansi.StringWidth(s)
}

// cursorDisplayCol returns the display column of the cursor (in terminal cells).
func (b *buffer) cursorDisplayCol() int {
	b.clampCursor()
	runes := []rune(b.lines[b.cursorLine])
	if b.cursorCol > len(runes) {
		b.cursorCol = len(runes)
	}
	return displayWidth(string(runes[:b.cursorCol]))
}
