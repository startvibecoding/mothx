# TUI Rendering Refactor Proposal

> Date: 2026-06-14
> Status: In progress

## Current Status

- Phase 1 complete: transcript rendering is managed by `bubbles/viewport` in
  the local TUI.
- Phase 2 complete: custom ANSI parser/tracker wrapping was replaced with
  `charmbracelet/x/ansi` for width measurement, ANSI-aware wrapping, hard
  wrapping, stripping, and truncation.

## Problem

The current TUI mixes two rendering surfaces:

- completed messages are printed through `tea.Println` / `program.Println`
- streaming `think` and `assistant` text is rendered through `App.View()` as `liveContent`

Bubble Tea documents `Println` output as unmanaged program output. It persists
outside the normal render tree, so it is a poor fit for a transcript that also
has streaming content. When think deltas arrive quickly and the wrapped text
changes line breaks, the managed live region and unmanaged history can visually
interleave. This is especially visible with mixed CJK and ASCII text.

The current custom ANSI wrapping code also duplicates functionality already
available in Charmbracelet libraries, increasing the chance of width and escape
sequence bugs.

## Goals

- Keep the full transcript in one managed Bubble Tea render tree.
- Use `bubbles/viewport` for transcript sizing, scrolling, and clipping.
- Keep `think` and `assistant` rendering as separate pipelines.
- Prefer `charmbracelet/x/ansi` and Lipgloss width helpers over custom ANSI
  parsing where possible.
- Preserve existing TUI behavior for tools, approvals, question prompts, input,
  footer, and plan panels.

## Non-Goals

- Do not redesign the visual style in this pass.
- Do not change provider, agent, or session behavior.
- Do not replace all Markdown rendering at once.
- Do not remove tests that document existing TUI workflows.

## Proposed Architecture

### Transcript Viewport

Add a `viewport.Model` to `App` and render all transcript messages into it:

```text
plan panel (optional)
viewport transcript
input
footer
```

The transcript content is rebuilt from the existing message index model:

- `messages[idx]` for ordinary user/status messages
- `assistantRaw[idx]` rendered by `renderAssistantMessage`
- `thinkRaw[idx]` rendered by `renderThinkMessage`
- `toolResults` rendered by `renderToolResult`

`updateViewportContent()` becomes the single place that rebuilds the viewport
content. If the user was at the bottom before the rebuild, it keeps the viewport
pinned to the bottom. If the user has scrolled up, it preserves the current
offset so streaming output does not yank the view.

### Streaming

Streaming deltas should only update raw message buffers and mark cached renders
dirty. The next scheduled render rebuilds the managed viewport. Completed
messages should no longer be printed into unmanaged output.

### Wrapping

Rendering should eventually converge on:

- `think`: plain text wrapping, no Markdown
- `assistant plain`: plain text wrapping
- `assistant Markdown`: Glamour render with bounded `WithWordWrap`, then final
  width enforcement
- width and ANSI handling via `charmbracelet/x/ansi` or Lipgloss helpers

The first implementation phase kept the current renderers but moved their
output into the viewport. The second phase replaced custom wrapping with
`x/ansi` under focused tests.

## Implementation Plan

1. Add the proposal document.
2. Add `viewport.Model` to `internal/tui.App`.
3. Replace main transcript printing with managed viewport content.
4. Keep `tea.Println` only for cases that intentionally print outside the TUI,
   or remove the drain path if no longer needed.
5. Route viewport key and mouse messages through `viewport.Update`.
6. Update tests to assert viewport content instead of `liveContent` or
   `pendingPrints`.
7. After viewport stabilization, replace custom wrapping code with
   `charmbracelet/x/ansi` in a smaller follow-up.

## Test Strategy

- Unit tests for mixed CJK/ASCII `think` and assistant rendering.
- Tests that streaming think and assistant content appear in transcript order.
- Tests that initial message and loaded history appear in the viewport.
- Tests that limited height keeps the input and footer visible.
- Existing tool approval, command, and clear-state tests should keep passing.

## Risks

- Some tests rely on `pendingPrints`; they need to assert managed viewport
  content instead.
- Transcript scroll behavior must preserve user offset while still pinning to
  bottom during normal streaming.
- Remote TUI duplicates much of the local TUI code, so changes must stay in
  sync until the duplicate code is consolidated.
