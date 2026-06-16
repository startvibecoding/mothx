---
name: bubble-tea-component-creation
description: Patterns and pitfalls for creating reusable Bubble Tea TUI components with proper integration
source: auto-skill
extracted_at: '2026-06-16T01:06:05.532Z'
---

# Bubble Tea Component Creation Patterns

When creating new TUI components for the vibecoding project (or similar Bubble Tea applications), follow these patterns and watch for common pitfalls.

## Component Structure

### Standard Model/View/Update Pattern

```go
package component

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Model struct {
    // Internal state
    width  int
    height int
    focus  bool
    // ... other fields
}

func New(width, height int) Model {
    return Model{
        width:  width,
        height: height,
        focus:  true,
    }
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    }
    return m, nil
}

func (m Model) View() string {
    // Render component
    return ""
}

func (m Model) SetSize(width, height int) Model {
    m.width = width
    m.height = height
    return m
}

func (m Model) Focus() Model {
    m.focus = true
    return m
}

func (m Model) Blur() Model {
    m.focus = false
    return m
}
```

### Key Handling Pitfalls

**Bubble Tea v1.3.4 does NOT support these key types:**
- ❌ `tea.KeyShiftEnter` — does not exist
- ❌ `tea.KeyCtrlShiftX` — modifier combinations limited

**Use these alternatives instead:**
- ✅ `tea.KeyCtrlJ` for newline insertion (Ctrl+J)
- ✅ Check `msg.Alt` for Alt+Enter: `if msg.Type == tea.KeyEnter && msg.Alt`
- ✅ `tea.KeyCtrlK`, `tea.KeyCtrlU`, `tea.KeyCtrlW` for text editing shortcuts

```go
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
    // Alt+Enter for newline (works on most terminals)
    if msg.Type == tea.KeyEnter && msg.Alt {
        m.buf.InsertNewline()
        return m, nil
    }
    
    switch msg.Type {
    case tea.KeyEnter:
        // Enter submits
        return m, func() tea.Msg { return SubmitMsg{} }
    
    case tea.KeyCtrlJ:
        // Ctrl+J also inserts newline
        m.buf.InsertNewline()
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
    }
    
    return m, nil
}
```

## ANSI-Aware Text Utilities

### Width Calculation and Truncation

**Do NOT use `lipgloss.Truncate`** — it doesn't exist in lipgloss v1.1.1.

**Use `xansi` from `github.com/charmbracelet/x/ansi` instead:**

```go
import xansi "github.com/charmbracelet/x/ansi"

// Calculate display width (handles ANSI codes and CJK)
width := xansi.StringWidth(text)

// Truncate text to max width with ellipsis
truncated := xansi.Truncate(text, maxWidth, "…")

// Wrap text to width (ANSI-aware)
wrapped := xansi.Wrap(text, width, "/")
```

### Rendering Patterns

```go
func (m Model) View() string {
    // Calculate available width
    availW := m.width
    
    // Truncate lines to fit
    lines := strings.Split(content, "\n")
    for i, line := range lines {
        if xansi.StringWidth(line) > availW {
            lines[i] = xansi.Truncate(line, availW, "…")
        }
    }
    
    // Join and apply style
    return m.style.Width(m.width).Render(strings.Join(lines, "\n"))
}
```

## Integration into Existing TUI

### Adding Components to View() Layout

When integrating a new component into the main `View()` function:

```go
func (a *App) View() string {
    if !a.ready {
        return "\n  Loading...\n"
    }
    
    footer := a.renderFooter()
    if a.toolModalOpen {
        return a.renderFixedHeight(lipgloss.JoinVertical(lipgloss.Left, 
            a.renderToolModal(), footer))
    }
    
    a.resizeViewport()
    var parts []string
    
    // 1. Main transcript
    parts = append(parts, a.viewport.View())
    
    // 2. Optional panels (only if non-empty)
    if panel := a.renderPlanPanel(); panel != "" {
        parts = append(parts, panel)
    }
    
    if todoList := renderStickyTodoList(a.currentPlan, a.width, 5); todoList != "" {
        parts = append(parts, todoList)
    }
    
    // 3. Loading indicator (only when thinking)
    if loading := renderLoadingIndicator(a.isThinking, ...); loading != "" {
        parts = append(parts, loading)
    }
    
    // 4. Input field
    parts = append(parts, a.input.View())
    
    // 5. Footer
    parts = append(parts, footer)
    
    // 6. Optional tab bar
    if tabBar := renderAgentTabBar(...); tabBar != "" {
        parts = append(parts, tabBar)
    }
    
    return a.renderFixedHeight(lipgloss.JoinVertical(lipgloss.Left, parts...))
}
```

### Updating Viewport Height Calculations

**Critical:** When adding new components, update the viewport height calculation to account for them:

```go
func (a *App) transcriptViewportHeight(footer string) int {
    height := a.height
    if height <= 0 {
        return 1
    }
    
    used := lipgloss.Height(a.input.View()) + lipgloss.Height(footer)
    
    // Account for all optional components
    if panel := a.renderPlanPanel(); panel != "" {
        used += lipgloss.Height(panel)
    }
    
    if todoList := renderStickyTodoList(...); todoList != "" {
        used += lipgloss.Height(todoList)
    }
    
    if loading := renderLoadingIndicator(...); loading != "" {
        used += lipgloss.Height(loading)
    }
    
    if tabBar := renderAgentTabBar(...); tabBar != "" {
        used += lipgloss.Height(tabBar)
    }
    
    available := height - used
    if available < 1 {
        return 1
    }
    return available
}
```

### Render Functions vs Bubble Tea Models

For simple UI elements, use **render functions** instead of full Bubble Tea models:

```go
// Simple render function (no state, no Update)
func renderLoadingIndicator(isThinking bool, spinnerIndex int, 
    elapsed time.Duration, tokens int, width int) string {
    if !isThinking {
        return ""
    }
    // Build and return rendered string
    return statusStyle.Render(fmt.Sprintf("%s Thinking... (%s · ↓ %s tokens)",
        spinnerChars[spinnerIndex],
        formatDuration(elapsed),
        formatTokenCount(tokens)))
}
```

Use full Bubble Tea models only when the component needs:
- Internal state management
- Keyboard/mouse event handling
- Timer/tick updates

## Testing Patterns

### Component Unit Tests

```go
func TestComponentBasicOperations(t *testing.T) {
    m := New(80, 24)
    
    // Test initial state
    if m.Focused() != true {
        t.Error("new component should be focused")
    }
    
    // Test key input
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
    if m.Value() != "a" {
        t.Errorf("Value() = %q, want %q", m.Value(), "a")
    }
    
    // Test special keys
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
    if m.Value() != "a\nb" {
        t.Errorf("Value() = %q, want %q", m.Value(), "a\nb")
    }
}

func TestComponentView(t *testing.T) {
    m := New(40, 10)
    view := m.View()
    
    if view == "" {
        t.Error("View() should not be empty")
    }
    
    // Check for expected content
    if !strings.Contains(view, "placeholder") {
        t.Error("View() should contain placeholder text")
    }
}
```

### Adapting Tests After Layout Changes

When you change the footer height or add new components, **tests that depend on viewport dimensions need updating:**

```go
// Before: height=8 worked
app.height = 8

// After: footer is now 3 lines instead of 1, need more height
app.height = 12

// Also update test data to ensure overflow
app.messages = []string{strings.Join([]string{
    "line 1", "line 2", "line 3", "line 4", "line 5",
    "line 6", "line 7", "line 8", "line 9", "line 10",
    "line 11", "line 12",
}, "\n")}
```

**Test pattern for checking line visibility:**

```go
viewContainsLine := func(view, line string) bool {
    for _, l := range strings.Split(view, "\n") {
        if strings.TrimSpace(l) == line {
            return true
        }
    }
    return false
}

// Check that a line is NOT visible initially
if viewContainsLine(before, "line 2") {
    t.Fatalf("precondition failed: line 2 should not be visible")
}

// Scroll up
app.Update(tea.MouseMsg{
    Action: tea.MouseActionPress,
    Button: tea.MouseButtonWheelUp,
})

// Check that the line IS now visible
after := stripANSI(app.View())
if !viewContainsLine(after, "line 2") {
    t.Fatalf("scrolling should make line 2 visible")
}
```

## Common Gotchas

1. **Pointer vs Value Receivers**: Use pointer receivers for methods that mutate state:
   ```go
   func (m *Model) clampOffset() {  // ✅ pointer receiver
       if m.offset < 0 {
           m.offset = 0
       }
   }
   ```

2. **CJK Width**: Always use `xansi.StringWidth` for display width, not `len()` or rune count

3. **ANSI Codes in Tests**: Strip ANSI codes before checking content:
   ```go
   func stripANSI(s string) string {
       return xansi.Strip(s)
   }
   ```

4. **Component Focus**: Always check `m.focus` before processing input:
   ```go
   func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
       if !m.focus {
           return m, nil
       }
       // ... handle input
   }
   ```

5. **Viewport Auto-Follow**: Track whether user is at bottom before appending:
   ```go
   wasAtBottom := m.AtBottom()
   m.AppendItem(newItem)
   if wasAtBottom {
       m = m.GotoBottom()
   }
   ```

## File Organization

```
internal/tui/
├── components/
│   ├── editor/
│   │   ├── buffer.go      # Text buffer logic
│   │   ├── editor.go      # Bubble Tea model
│   │   └── editor_test.go
│   ├── vscroll/
│   │   ├── vscroll.go     # Virtual scroll list
│   │   └── vscroll_test.go
│   └── suggest/
│       ├── suggest.go     # Autocomplete dropdown
│       └── suggest_test.go
├── header.go              # Render function
├── loading.go             # Render function
├── todo_list.go           # Render function
└── agent_tabbar.go        # Render function
```

Keep render-only components as simple functions in `internal/tui/`. Use subdirectories under `components/` only for full Bubble Tea models with state and tests.
