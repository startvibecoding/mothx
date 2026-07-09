//go:build !windows && !darwin

package main

import tea "github.com/charmbracelet/bubbletea"

func initConsole() error {
	return nil
}

func teaProgramOptions() []tea.ProgramOption {
	return []tea.ProgramOption{tea.WithInputTTY(), tea.WithReportFocus()}
}
