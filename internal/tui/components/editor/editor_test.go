package editor

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBufferInsertAndValue(t *testing.T) {
	b := newBuffer()
	b.InsertRune('h')
	b.InsertRune('i')
	if got := b.Value(); got != "hi" {
		t.Errorf("Value() = %q, want %q", got, "hi")
	}
}

func TestBufferNewline(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello")
	b.cursorCol = 5
	b.InsertNewline()
	b.InsertString("world")

	if got := b.Value(); got != "hello\nworld" {
		t.Errorf("Value() = %q, want %q", got, "hello\nworld")
	}
	if b.LineCount() != 2 {
		t.Errorf("LineCount() = %d, want 2", b.LineCount())
	}
}

func TestBufferDeleteBack(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello")
	b.cursorCol = 5
	b.DeleteBack()
	if got := b.Value(); got != "hell" {
		t.Errorf("Value() = %q, want %q", got, "hell")
	}

	// Delete at start of line merges with previous
	b.InsertNewline()
	b.InsertString("world")
	b.cursorCol = 0
	b.DeleteBack()
	if got := b.Value(); got != "hellworld" {
		t.Errorf("after merge: Value() = %q, want %q", got, "hellworld")
	}
}

func TestBufferDeleteForward(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello")
	b.cursorCol = 0
	b.DeleteForward()
	if got := b.Value(); got != "ello" {
		t.Errorf("Value() = %q, want %q", got, "ello")
	}
}

func TestBufferDeleteWordBack(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello world")
	b.cursorCol = 11
	b.DeleteWordBack()
	if got := b.Value(); got != "hello " {
		t.Errorf("Value() = %q, want %q", got, "hello ")
	}
}

func TestBufferDeleteToLineEnd(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello world")
	b.cursorCol = 5
	b.DeleteToLineEnd()
	if got := b.Value(); got != "hello" {
		t.Errorf("Value() = %q, want %q", got, "hello")
	}
}

func TestBufferDeleteToLineStart(t *testing.T) {
	b := newBuffer()
	b.InsertString("hello world")
	b.cursorCol = 5
	b.DeleteToLineStart()
	if got := b.Value(); got != " world" {
		t.Errorf("Value() = %q, want %q", got, " world")
	}
}

func TestBufferMoveUpDown(t *testing.T) {
	b := newBuffer()
	b.SetValue("line1\nline2\nline3")

	b.cursorLine = 1
	b.cursorCol = 3
	b.preferredCol = 3

	// Move up
	if !b.MoveUp() {
		t.Error("MoveUp() returned false")
	}
	if b.cursorLine != 0 {
		t.Errorf("after MoveUp: cursorLine = %d, want 0", b.cursorLine)
	}
	if b.cursorCol != 3 {
		t.Errorf("after MoveUp: cursorCol = %d, want 3", b.cursorCol)
	}

	// Move down twice
	b.MoveDown()
	if b.cursorLine != 1 {
		t.Errorf("after MoveDown: cursorLine = %d, want 1", b.cursorLine)
	}
	b.MoveDown()
	if b.cursorLine != 2 {
		t.Errorf("after MoveDown x2: cursorLine = %d, want 2", b.cursorLine)
	}

	// Move down past end returns false
	if b.MoveDown() {
		t.Error("MoveDown() at end should return false")
	}
}

func TestBufferMoveLeftRight(t *testing.T) {
	b := newBuffer()
	b.SetValue("ab\ncd")

	b.cursorLine = 0
	b.cursorCol = 0
	b.preferredCol = 0

	// Move right twice
	b.MoveRight()
	b.MoveRight()
	// Move right across line boundary
	b.MoveRight()
	if b.cursorLine != 1 || b.cursorCol != 0 {
		t.Errorf("after cross-line right: (%d,%d), want (1,0)", b.cursorLine, b.cursorCol)
	}

	// Move left across line boundary
	b.MoveLeft()
	if b.cursorLine != 0 || b.cursorCol != 2 {
		t.Errorf("after cross-line left: (%d,%d), want (0,2)", b.cursorLine, b.cursorCol)
	}
}

func TestBufferHomeEnd(t *testing.T) {
	b := newBuffer()
	b.SetValue("hello world")
	b.cursorCol = 5

	b.MoveHome()
	if b.cursorCol != 0 {
		t.Errorf("after Home: cursorCol = %d, want 0", b.cursorCol)
	}

	b.MoveEnd()
	if b.cursorCol != 11 {
		t.Errorf("after End: cursorCol = %d, want 11", b.cursorCol)
	}
}

func TestBufferSetValue(t *testing.T) {
	b := newBuffer()
	b.SetValue("multi\nline\ntext")
	if b.LineCount() != 3 {
		t.Errorf("LineCount() = %d, want 3", b.LineCount())
	}
	if got := b.Value(); got != "multi\nline\ntext" {
		t.Errorf("Value() = %q, want %q", got, "multi\nline\ntext")
	}

	// Cursor should be at start
	line, col := b.CursorPos()
	if line != 0 || col != 0 {
		t.Errorf("CursorPos() = (%d,%d), want (0,0)", line, col)
	}
}

func TestBufferReset(t *testing.T) {
	b := newBuffer()
	b.SetValue("some text")
	b.Reset()
	if got := b.Value(); got != "" {
		t.Errorf("after Reset: Value() = %q, want %q", got, "")
	}
	if b.LineCount() != 1 {
		t.Errorf("after Reset: LineCount() = %d, want 1", b.LineCount())
	}
}

func TestBufferCJK(t *testing.T) {
	b := newBuffer()
	b.InsertString("你好世界")
	if got := b.Value(); got != "你好世界" {
		t.Errorf("Value() = %q, want %q", got, "你好世界")
	}
	// CJK characters should be counted correctly
	if b.cursorCol != 4 {
		t.Errorf("cursorCol = %d, want 4", b.cursorCol)
	}
}

func TestBufferInsertStringMultiline(t *testing.T) {
	b := newBuffer()
	b.InsertString("ab")
	b.cursorCol = 1 // cursor between 'a' and 'b'
	b.InsertString("x\ny")
	if got := b.Value(); got != "ax\nyb" {
		t.Errorf("Value() = %q, want %q", got, "ax\nyb")
	}
}

func TestEditorNewModel(t *testing.T) {
	m := New(80)
	if m.Focused() != true {
		t.Error("new editor should be focused")
	}
	if m.Value() != "" {
		t.Errorf("new editor Value() = %q, want empty", m.Value())
	}
}

func TestEditorSetValue(t *testing.T) {
	m := New(80)
	m = m.SetValue("hello\nworld")
	if got := m.Value(); got != "hello\nworld" {
		t.Errorf("Value() = %q, want %q", got, "hello\nworld")
	}
}

func TestEditorReset(t *testing.T) {
	m := New(80)
	m = m.SetValue("hello")
	m = m.Reset()
	if got := m.Value(); got != "" {
		t.Errorf("after Reset: Value() = %q, want empty", got)
	}
}

func TestEditorKeyRunes(t *testing.T) {
	m := New(80)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h', 'i'}})
	if got := m.Value(); got != "hi" {
		t.Errorf("Value() = %q, want %q", got, "hi")
	}
}

func TestEditorBackspace(t *testing.T) {
	m := New(80)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if got := m.Value(); got != "a" {
		t.Errorf("Value() = %q, want %q", got, "a")
	}
}

func TestEditorAltEnterNewline(t *testing.T) {
	m := New(80)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if got := m.Value(); got != "a\nb" {
		t.Errorf("Value() = %q, want %q", got, "a\nb")
	}
}

func TestEditorCtrlJNewline(t *testing.T) {
	m := New(80)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if got := m.Value(); got != "x\ny" {
		t.Errorf("Value() = %q, want %q", got, "x\ny")
	}
}

func TestEditorFocusBlur(t *testing.T) {
	m := New(80)
	m = m.Blur()
	if m.Focused() {
		t.Error("should not be focused after Blur()")
	}
	// Key input should be ignored when blurred
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if got := m.Value(); got != "" {
		t.Errorf("blurred editor Value() = %q, want empty", got)
	}

	m = m.Focus()
	if !m.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestEditorView(t *testing.T) {
	m := New(40)
	m = m.SetPlaceholder("type here...")
	view := m.View()
	if view == "" {
		t.Error("View() should not be empty")
	}
}

func TestWrapLine(t *testing.T) {
	tests := []struct {
		line   string
		width  int
		expect int
	}{
		{"hello", 10, 1},
		{"hello world", 5, 3}, // "hello", " worl", "d"
		{"", 10, 1},
	}
	for _, tt := range tests {
		result := wrapLine(tt.line, tt.width)
		if len(result) != tt.expect {
			t.Errorf("wrapLine(%q, %d) = %d lines, want %d", tt.line, tt.width, len(result), tt.expect)
		}
	}
}

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		s      string
		expect int
	}{
		{"hello", 5},
		{"你好", 4},     // CJK = 2 cells each
		{"", 0},
		{"abc", 3},
	}
	for _, tt := range tests {
		got := displayWidth(tt.s)
		if got != tt.expect {
			t.Errorf("displayWidth(%q) = %d, want %d", tt.s, got, tt.expect)
		}
	}
}
