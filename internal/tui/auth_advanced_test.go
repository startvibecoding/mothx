package tui

import "testing"

func TestAuthAdvancedOptionsStartWithContinueAndSkip(t *testing.T) {
	a := &App{auth: authDialogState{View: authViewAdvanced}}
	opts := a.authOptions()
	if len(opts) < 2 {
		t.Fatalf("got %d options, want at least 2", len(opts))
	}
	if opts[0].Value != "continue" || opts[1].Value != "skip" {
		t.Fatalf("first options = %q/%q, want continue/skip", opts[0].Value, opts[1].Value)
	}
}
