package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandSuggestionsForSlash(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/au")
	a.updateCommandSuggestions()
	if !a.commandSuggestionsVisible() {
		t.Fatal("expected command suggestions to be visible")
	}
	if !a.applySelectedCommandSuggestion() {
		t.Fatal("expected selected command suggestion to apply")
	}
	if got := a.input.Value(); got != "/auth" {
		t.Fatalf("input = %q, want /auth", got)
	}
}

func TestCommandSuggestionsHiddenAfterSpace(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/mode ")
	a.updateCommandSuggestions()
	if !a.commandSuggestionsVisible() {
		t.Fatal("expected argument suggestions after space")
	}
}

func TestCommandSuggestionEnterFlushesQueuedInputBeforeApplying(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/mo")
	a.updateCommandSuggestions()
	if !a.commandSuggestionsVisible() {
		t.Fatal("expected command suggestions to be visible before queued input")
	}

	a.queueInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("de ")})
	a.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if got := a.input.Value(); got != "" {
		t.Fatalf("input = %q, want command to execute after queued input flush", got)
	}
	if a.commandSuggestionsVisible() {
		t.Fatal("expected suggestions hidden after queued input adds a space")
	}
}

func TestCommandArgumentSuggestionForMode(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/mode a")
	a.updateCommandSuggestions()
	if !a.commandSuggestionsVisible() {
		t.Fatal("expected mode argument suggestions to be visible")
	}
	if !a.applySelectedCommandSuggestion() {
		t.Fatal("expected selected mode argument suggestion to apply")
	}
	if got := a.input.Value(); got != "/mode agent" {
		t.Fatalf("input = %q, want /mode agent", got)
	}
}

func TestTabCompletesCommandArgumentInsteadOfCyclingMode(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/mode ")
	a.updateCommandSuggestions()

	a.Update(tea.KeyMsg{Type: tea.KeyTab})

	if got := a.mode; got != "agent" {
		t.Fatalf("mode = %q, want agent", got)
	}
	if got := a.input.Value(); got != "/mode plan" {
		t.Fatalf("input = %q, want /mode plan", got)
	}
}

func TestTabInSlashCommandWithoutArgumentSuggestionDoesNotCycleMode(t *testing.T) {
	a := NewApp(nil, nil, nil, nil, nil, "", "", nil, "agent", false, false, nil, nil, nil)
	a.input = a.input.SetValue("/skill ")
	a.updateCommandSuggestions()

	a.Update(tea.KeyMsg{Type: tea.KeyTab})

	if got := a.mode; got != "agent" {
		t.Fatalf("mode = %q, want agent", got)
	}
	if got := a.input.Value(); got != "/skill " {
		t.Fatalf("input = %q, want /skill ", got)
	}
}
