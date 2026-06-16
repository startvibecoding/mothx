package vscroll

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := New(80, 24)
	if m.width != 80 {
		t.Errorf("width = %d, want 80", m.width)
	}
	if m.height != 24 {
		t.Errorf("height = %d, want 24", m.height)
	}
	if m.offset != 0 {
		t.Errorf("offset = %d, want 0", m.offset)
	}
	if !m.followBottom {
		t.Error("followBottom should be true by default")
	}
	if m.ItemCount() != 0 {
		t.Errorf("ItemCount() = %d, want 0", m.ItemCount())
	}
}

func TestSetItems(t *testing.T) {
	m := New(40, 10)
	items := []string{"one", "two", "three"}
	m = m.SetItems(items)

	if m.ItemCount() != 3 {
		t.Errorf("ItemCount() = %d, want 3", m.ItemCount())
	}

	// Replace with fewer items.
	m = m.SetItems([]string{"only"})
	if m.ItemCount() != 1 {
		t.Errorf("ItemCount() = %d, want 1", m.ItemCount())
	}

	// Replace with empty.
	m = m.SetItems(nil)
	if m.ItemCount() != 0 {
		t.Errorf("ItemCount() = %d, want 0", m.ItemCount())
	}
}

func TestAppendItem(t *testing.T) {
	m := New(40, 10)
	m = m.AppendItem("first")
	m = m.AppendItem("second")

	if m.ItemCount() != 2 {
		t.Errorf("ItemCount() = %d, want 2", m.ItemCount())
	}
}

func TestAppendItemAutoFollow(t *testing.T) {
	// Height 3, each item is one line, separator is blank line between items.
	// Items: "a", "", "b", "", "c", "", "d" => 7 display lines for 4 items.
	m := New(20, 3)
	m = m.AppendItem("a")
	m = m.AppendItem("b")
	m = m.AppendItem("c")
	m = m.AppendItem("d")

	// followBottom should keep us at the bottom.
	if !m.followBottom {
		t.Error("followBottom should still be true after appending")
	}
	if !m.AtBottom() {
		t.Error("should be at bottom after appending with followBottom")
	}
}

func TestGotoBottom(t *testing.T) {
	m := New(20, 3)
	for i := 0; i < 10; i++ {
		m = m.AppendItem("line")
	}
	m = m.GotoTop()
	if m.AtBottom() {
		t.Error("should not be at bottom after GotoTop")
	}

	m = m.GotoBottom()
	if !m.AtBottom() {
		t.Error("should be at bottom after GotoBottom")
	}
	if !m.followBottom {
		t.Error("followBottom should be true after GotoBottom")
	}
}

func TestGotoTop(t *testing.T) {
	m := New(20, 3)
	for i := 0; i < 10; i++ {
		m = m.AppendItem("line")
	}

	m = m.GotoTop()
	if m.offset != 0 {
		t.Errorf("offset = %d, want 0 after GotoTop", m.offset)
	}
	if m.followBottom {
		t.Error("followBottom should be false after GotoTop")
	}
}

func TestAtBottom(t *testing.T) {
	m := New(20, 3)
	// Empty model is trivially at bottom.
	if !m.AtBottom() {
		t.Error("empty model should be at bottom")
	}

	m = m.AppendItem("a")
	if !m.AtBottom() {
		t.Error("single-item model should be at bottom")
	}
}

func TestViewBasic(t *testing.T) {
	// Width 10, height 5, 3 single-line items.
	// displayLines: ["aaa", "", "bbb", "", "ccc"] => 5 lines, fits exactly.
	m := New(10, 5)
	m = m.SetItems([]string{"aaa", "bbb", "ccc"})

	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) != 5 {
		t.Fatalf("View() returned %d lines, want 5", len(lines))
	}

	// First line should contain "aaa" padded to width 10.
	if !strings.Contains(lines[0], "aaa") {
		t.Errorf("line 0 = %q, want it to contain 'aaa'", lines[0])
	}
	// Second line is the separator (blank, padded with spaces).
	if strings.TrimSpace(lines[1]) != "" {
		t.Errorf("line 1 = %q, want blank separator", lines[1])
	}
	// Third line should contain "bbb".
	if !strings.Contains(lines[2], "bbb") {
		t.Errorf("line 2 = %q, want it to contain 'bbb'", lines[2])
	}
	// Fifth line should contain "ccc".
	if !strings.Contains(lines[4], "ccc") {
		t.Errorf("line 4 = %q, want it to contain 'ccc'", lines[4])
	}
}

func TestViewEmptyModel(t *testing.T) {
	m := New(10, 3)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		t.Fatalf("View() returned %d lines, want 3 (all blank/padded)", len(lines))
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			t.Errorf("line %d = %q, want blank", i, line)
		}
	}
}

func TestViewScrolling(t *testing.T) {
	// Create items that exceed viewport.
	// 6 items => displayLines: "0", "", "1", "", "2", "", "3", "", "4", "", "5" => 11 lines.
	m := New(10, 5)
	items := make([]string, 6)
	for i := range items {
		items[i] = strings.Repeat(string(rune('A'+i)), 5) // "AAAAA", "BBBBB", ...
	}
	m = m.SetItems(items)

	// At bottom by default (followBottom), visible should be last 5 lines.
	view := m.View()
	bottomLines := strings.Split(view, "\n")
	// Last item "FFFFF" should be visible.
	found := false
	for _, l := range bottomLines {
		if strings.Contains(l, "FFFFF") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("bottom view should contain FFFFF, got:\n%s", view)
	}

	// Scroll to top.
	m = m.GotoTop()
	view = m.View()
	topLines := strings.Split(view, "\n")
	// First item "AAAAA" should be visible.
	found = false
	for _, l := range topLines {
		if strings.Contains(l, "AAAAA") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("top view should contain AAAAA, got:\n%s", view)
	}

	// The first item "FFFFF" should NOT be visible at top.
	for _, l := range topLines {
		if strings.Contains(l, "FFFFF") {
			t.Error("top view should not contain FFFFF")
			break
		}
	}
}

func TestViewAutoFollow(t *testing.T) {
	m := New(10, 3)
	m = m.AppendItem("aaa")
	m = m.AppendItem("bbb")

	// Should be at bottom with follow.
	if !m.AtBottom() {
		t.Error("should be at bottom initially")
	}

	// Scroll up to disable follow.
	m = m.GotoTop()
	if m.followBottom {
		t.Error("followBottom should be false after GotoTop")
	}

	// Append while scrolled up - should NOT auto-follow.
	m = m.AppendItem("ccc")
	m = m.AppendItem("ddd")
	if m.followBottom {
		t.Error("followBottom should remain false after appending while scrolled up")
	}
	if m.offset != 0 {
		t.Errorf("offset = %d, want 0 (no auto-follow when scrolled up)", m.offset)
	}

	// Go to bottom, then append - should auto-follow.
	m = m.GotoBottom()
	if !m.followBottom {
		t.Error("followBottom should be true after GotoBottom")
	}
	oldOffset := m.offset
	m = m.AppendItem("eee")
	if m.offset <= oldOffset {
		t.Errorf("offset should increase after append at bottom: was %d, now %d", oldOffset, m.offset)
	}
}

func TestPageScroll(t *testing.T) {
	// Height 5, 20 single-line items.
	// displayLines: 20 items + 19 separators = 39 lines.
	m := New(10, 5)
	items := make([]string, 20)
	for i := range items {
		items[i] = strings.Repeat("x", 5)
	}
	m = m.SetItems(items)
	m = m.GotoTop()

	// PageDown should scroll by height (5 lines).
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m2.offset != 5 {
		t.Errorf("after PgDown: offset = %d, want 5", m2.offset)
	}

	// Another PageDown.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m3.offset != 10 {
		t.Errorf("after 2nd PgDown: offset = %d, want 10", m3.offset)
	}

	// PageUp.
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m4.offset != 5 {
		t.Errorf("after PgUp: offset = %d, want 5", m4.offset)
	}

	// PageUp again to top.
	m5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m5.offset != 0 {
		t.Errorf("after 2nd PgUp: offset = %d, want 0", m5.offset)
	}

	// PageUp at top stays at 0.
	m6, _ := m5.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m6.offset != 0 {
		t.Errorf("PgUp at top: offset = %d, want 0", m6.offset)
	}
}

func TestLineScroll(t *testing.T) {
	m := New(10, 5)
	items := make([]string, 20)
	for i := range items {
		items[i] = "line"
	}
	m = m.SetItems(items)
	m = m.GotoTop()

	// Down arrow scrolls by mouseWheelScrollLines (3).
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.offset != 3 {
		t.Errorf("after Down: offset = %d, want 3", m2.offset)
	}

	// Up arrow scrolls back.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m3.offset != 0 {
		t.Errorf("after Up: offset = %d, want 0", m3.offset)
	}
}

func TestHomeEndKeys(t *testing.T) {
	m := New(10, 3)
	for i := 0; i < 20; i++ {
		m = m.AppendItem("item")
	}

	// Home key.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m2.offset != 0 {
		t.Errorf("after Home: offset = %d, want 0", m2.offset)
	}
	if m2.followBottom {
		t.Error("followBottom should be false after Home")
	}

	// End key.
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !m3.AtBottom() {
		t.Error("should be at bottom after End")
	}
	if !m3.followBottom {
		t.Error("followBottom should be true after End")
	}
}

func TestMouseWheelScroll(t *testing.T) {
	m := New(10, 5)
	items := make([]string, 20)
	for i := range items {
		items[i] = "line"
	}
	m = m.SetItems(items)
	m = m.GotoTop()

	// Wheel down.
	m2, _ := m.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	})
	if m2.offset != 3 {
		t.Errorf("after wheel down: offset = %d, want 3", m2.offset)
	}

	// Wheel up.
	m3, _ := m2.Update(tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	})
	if m3.offset != 0 {
		t.Errorf("after wheel up: offset = %d, want 0", m3.offset)
	}
}

func TestSetSize(t *testing.T) {
	m := New(40, 10)
	for i := 0; i < 20; i++ {
		m = m.AppendItem("item")
	}

	// Shrink height - offset should be clamped.
	m = m.GotoBottom()
	m2 := m.SetSize(40, 5)
	if m2.width != 40 {
		t.Errorf("width = %d, want 40", m2.width)
	}
	if m2.height != 5 {
		t.Errorf("height = %d, want 5", m2.height)
	}
	// Should still be at bottom after resize.
	if !m2.AtBottom() {
		t.Error("should remain at bottom after shrinking")
	}

	// Grow height.
	m3 := m2.SetSize(80, 50)
	if m3.width != 80 {
		t.Errorf("width = %d, want 80", m3.width)
	}
	if m3.height != 50 {
		t.Errorf("height = %d, want 50", m3.height)
	}
}

func TestSetSizeClampsOffset(t *testing.T) {
	m := New(10, 3)
	for i := 0; i < 20; i++ {
		m = m.AppendItem("item")
	}
	m = m.GotoTop()
	// Scroll down a bit.
	m.offset = 10

	// Grow height so maxOffset shrinks - offset should be clamped.
	m = m.SetSize(10, 100)
	maxOff := m.maxOffset()
	if m.offset > maxOff {
		t.Errorf("offset %d exceeds maxOffset %d after growing height", m.offset, maxOff)
	}
}

func TestViewLineTruncation(t *testing.T) {
	m := New(5, 1)
	m = m.SetItems([]string{"Hello, World!"})

	view := m.View()
	// The view should be exactly 5 chars wide (truncated).
	if w := len(strings.TrimRight(view, " ")); w > 5 {
		t.Errorf("view width = %d, want <= 5, got %q", w, view)
	}
}

func TestMultilineItems(t *testing.T) {
	m := New(20, 10)
	m = m.SetItems([]string{"line1\nline2\nline3"})

	if m.ItemCount() != 1 {
		t.Errorf("ItemCount() = %d, want 1", m.ItemCount())
	}

	// displayLines should have 3 lines for this single multiline item.
	dl := m.displayLines()
	if len(dl) != 3 {
		t.Errorf("displayLines() returned %d lines, want 3", len(dl))
	}
	if dl[0] != "line1" || dl[1] != "line2" || dl[2] != "line3" {
		t.Errorf("displayLines() = %v, want [line1 line2 line3]", dl)
	}
}

func TestViewPadsToExactHeight(t *testing.T) {
	// 1 item in a height=5 viewport: should still produce 5 lines.
	m := New(10, 5)
	m = m.SetItems([]string{"only"})

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Errorf("View() returned %d lines, want 5", len(lines))
	}
}

func TestSetStyle(t *testing.T) {
	m := New(10, 3)
	m = m.SetItems([]string{"test"})

	// Just ensure SetStyle does not panic and returns a valid model.
	m = m.SetStyle(m.style)
	_ = m.View()
}
