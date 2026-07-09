//go:build darwin

package main

import tea "github.com/charmbracelet/bubbletea"

func initConsole() error {
	return nil
}

func teaProgramOptions() []tea.ProgramOption {
	// MothX handles split multi-line pastes in the TUI input queue. Leaving
	// Bubble Tea's bracketed paste parser enabled can make the UI appear frozen
	// on macOS if a terminal or PTY drops the bracketed-paste end marker.
	return []tea.ProgramOption{tea.WithInputTTY(), tea.WithReportFocus(), tea.WithoutBracketedPaste()}
}
