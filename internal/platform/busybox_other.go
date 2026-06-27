//go:build !windows

package platform

// EnsureWindowsBusybox is a no-op on non-Windows platforms.
func EnsureWindowsBusybox() error { return nil }

// WindowsBusyboxPath is unavailable on non-Windows platforms.
func WindowsBusyboxPath() (string, bool) { return "", false }
