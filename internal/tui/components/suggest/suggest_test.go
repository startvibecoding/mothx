package suggest

import (
	"strings"
	"testing"
)

func testItems() []Item {
	return []Item{
		{Label: "mode", Description: "set mode", Value: "mode"},
		{Label: "model", Description: "set model", Value: "model"},
		{Label: "provider", Description: "set provider", Value: "provider"},
		{Label: "compact", Description: "compact context", Value: "compact"},
	}
}

func TestNew(t *testing.T) {
	m := New(40)
	if m.Visible() {
		t.Error("new model should not be visible")
	}
	if m.maxVisible != 8 {
		t.Errorf("expected maxVisible=8, got %d", m.maxVisible)
	}
	if m.width != 40 {
		t.Errorf("expected width=40, got %d", m.width)
	}
}

func TestSetItems(t *testing.T) {
	m := New(40)
	items := testItems()
	m = m.SetItems(items)
	if len(m.items) != 4 {
		t.Errorf("expected 4 items, got %d", len(m.items))
	}
	// items stored even when query is empty
	if len(m.filtered) != 4 {
		t.Errorf("expected 4 filtered items, got %d", len(m.filtered))
	}
	// but not visible because query is empty
	if m.Visible() {
		t.Error("should not be visible with empty query")
	}
}

func TestFilterPrefix(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("mo")
	if !m.Visible() {
		t.Error("should be visible with matching query")
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 matches for 'mo', got %d", len(m.filtered))
	}
	labels := []string{m.filtered[0].Label, m.filtered[1].Label}
	if labels[0] != "mode" || labels[1] != "model" {
		t.Errorf("expected [mode, model], got %v", labels)
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("MO")
	if !m.Visible() {
		t.Error("should be visible for case-insensitive match")
	}
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 matches for 'MO', got %d", len(m.filtered))
	}
}

func TestFilterEmpty(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("mo")
	if !m.Visible() {
		t.Fatal("should be visible after matching query")
	}
	m = m.Update("")
	if m.Visible() {
		t.Error("empty query should hide dropdown")
	}
}

func TestFilterNoMatch(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("xyz")
	if m.Visible() {
		t.Error("no matches should hide dropdown")
	}
	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered items, got %d", len(m.filtered))
	}
}

func TestCursorNavigation(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("mo") // 2 items: mode, model

	if m.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.cursor)
	}

	m = m.CursorDown()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1 after down, got %d", m.cursor)
	}

	// wrap around
	m = m.CursorDown()
	if m.cursor != 0 {
		t.Errorf("cursor should wrap to 0, got %d", m.cursor)
	}

	// wrap up
	m = m.CursorUp()
	if m.cursor != 1 {
		t.Errorf("cursor should wrap to 1 on up from 0, got %d", m.cursor)
	}

	m = m.CursorUp()
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 after up, got %d", m.cursor)
	}
}

func TestSelected(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("mo") // 2 items: mode, model

	item, ok := m.Selected()
	if !ok {
		t.Fatal("expected selected item")
	}
	if item.Label != "mode" {
		t.Errorf("expected 'mode', got %q", item.Label)
	}

	m = m.CursorDown()
	item, ok = m.Selected()
	if !ok {
		t.Fatal("expected selected item")
	}
	if item.Label != "model" {
		t.Errorf("expected 'model', got %q", item.Label)
	}

	// no items
	empty := New(40)
	_, ok = empty.Selected()
	if ok {
		t.Error("expected no selection on empty model")
	}
}

func TestView(t *testing.T) {
	m := New(40).SetItems(testItems())
	m = m.Update("mo")

	// should render without panicking
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view when visible")
	}
	if !strings.Contains(view, "mode") {
		t.Error("view should contain 'mode'")
	}
	if !strings.Contains(view, "model") {
		t.Error("view should contain 'model'")
	}

	// hidden model returns empty
	hidden := New(40).SetItems(testItems())
	if v := hidden.View(); v != "" {
		t.Errorf("hidden model should return empty view, got %q", v)
	}
}
