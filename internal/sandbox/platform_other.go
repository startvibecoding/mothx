//go:build !linux && !darwin && !windows

package sandbox

// newPlatformSandbox creates the platform-specific sandbox for platforms
// without a dedicated sandbox backend (e.g. FreeBSD). It falls back to the
// no-op sandbox, so commands run without sandbox restrictions.
func newPlatformSandbox(projectDir string, level Level) Sandbox {
	return newPlatformSandboxWithOptions(projectDir, level, Options{})
}

func newPlatformSandboxWithOptions(projectDir string, level Level, opts Options) Sandbox {
	return NewNoneSandbox()
}
