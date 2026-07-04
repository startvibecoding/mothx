//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func killCommandProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Setsid makes the shell the process-group leader. Kill the group so auth
	// helpers or grandchildren cannot keep the terminal/session wedged.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
