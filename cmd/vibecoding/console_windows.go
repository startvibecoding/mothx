//go:build windows

package main

import (
	"fmt"
	"os"

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
	if err := enableVirtualTerminal(os.Stdout.Fd(), windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return err
	}
	if err := enableVirtualTerminal(os.Stdin.Fd(), windows.ENABLE_EXTENDED_FLAGS); err != nil {
		return err
	}
	return nil
}

func enableVirtualTerminal(fd uintptr, flags uint32) error {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil
	}
	if err := windows.SetConsoleMode(handle, mode|flags); err != nil {
		return fmt.Errorf("set console mode: %w", err)
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
