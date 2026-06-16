package tui

import "testing"

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
	if a.commandSuggestionsVisible() {
		t.Fatal("expected suggestions hidden after space")
	}
}
