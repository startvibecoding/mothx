//go:build linux

package sandbox

// newPlatformSandbox creates the platform-specific sandbox for Linux.
func newPlatformSandbox(projectDir string, level Level) Sandbox {
	return newPlatformSandboxWithOptions(projectDir, level, Options{})
}

func newPlatformSandboxWithOptions(projectDir string, level Level, opts Options) Sandbox {
	return NewBwrapSandboxWithOptions(projectDir, level, opts)
}
