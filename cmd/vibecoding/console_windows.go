//go:build windows

package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sys/windows"
)

const cpUTF8 = 65001

// initConsole sets the Windows console to UTF-8 code page for both input
// and output. This is required for CJK IME (Chinese, Japanese, Korean)
// to work correctly in the TUI.
func initConsole() error {
	// Set input code page to UTF-8 so IME compositions are delivered as UTF-8
	if err := windows.SetConsoleCP(cpUTF8); err != nil {
		return fmt.Errorf("set console input code page: %w", err)
	}
	// Set output code page to UTF-8 so text renders correctly
	if err := windows.SetConsoleOutputCP(cpUTF8); err != nil {
		return fmt.Errorf("set console output code page: %w", err)
	}
	return nil
}

// teaProgramOptions returns platform-specific BubbleTea program options.
// On Windows we must NOT use tea.WithInputTTY() because the Windows input
// path reads directly from the console via coninput; TTY mode is a Unix
// concept and can break IME input.
func teaProgramOptions() []tea.ProgramOption {
	return []tea.ProgramOption{tea.WithReportFocus()}
}
