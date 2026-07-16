//go:build windows

package sandbox

// newPlatformSandbox creates the platform-specific sandbox for Windows.
func newPlatformSandbox(projectDir string, level Level) Sandbox {
	return newPlatformSandboxWithOptions(projectDir, level, Options{})
}

func newPlatformSandboxWithOptions(projectDir string, level Level, opts Options) Sandbox {
	return newWinSandbox(projectDir, level)
}
