//go:build windows

package tools

import (
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// Windows doesn't support Setpgid; nothing to do.
}

func killCommandProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
