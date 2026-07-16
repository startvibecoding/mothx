//go:build darwin

package sandbox

// newPlatformSandbox creates the platform-specific sandbox for macOS.
func newPlatformSandbox(projectDir string, level Level) Sandbox {
	return newPlatformSandboxWithOptions(projectDir, level, Options{})
}

func newPlatformSandboxWithOptions(projectDir string, level Level, opts Options) Sandbox {
	return newMacSandbox(projectDir, level)
}
