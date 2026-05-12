package sandbox

import (
	"context"
	"os/exec"
)

// NoneSandbox executes commands without any sandbox restrictions.
type NoneSandbox struct{}

// NewNoneSandbox creates a new no-op sandbox.
func NewNoneSandbox() *NoneSandbox {
	return &NoneSandbox{}
}

// WrapCommand returns a plain command without any wrapping.
func (s *NoneSandbox) WrapCommand(ctx context.Context, shell, cmd string, opts ExecOpts) *exec.Cmd {
	c := exec.CommandContext(ctx, shell, "-c", cmd)

	if opts.WorkDir != "" {
		c.Dir = opts.WorkDir
	}

	// Set environment variables
	for k, v := range opts.EnvVars {
		c.Env = append(c.Env, k+"="+v)
	}

	return c
}

// IsAvailable always returns true.
func (s *NoneSandbox) IsAvailable() bool {
	return true
}

// Name returns "none".
func (s *NoneSandbox) Name() string {
	return "none"
}

// Level returns LevelNone.
func (s *NoneSandbox) Level() Level {
	return LevelNone
}
