package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOS(t *testing.T) {
	os := OS()
	if os == "" {
		t.Error("expected non-empty OS")
	}

	if os != runtime.GOOS {
		t.Errorf("expected %s, got %s", runtime.GOOS, os)
	}
}

func TestIsWindows(t *testing.T) {
	expected := runtime.GOOS == "windows"
	if IsWindows() != expected {
		t.Errorf("expected %v, got %v", expected, IsWindows())
	}
}

func TestIsMacOS(t *testing.T) {
	expected := runtime.GOOS == "darwin"
	if IsMacOS() != expected {
		t.Errorf("expected %v, got %v", expected, IsMacOS())
	}
}

func TestIsLinux(t *testing.T) {
	expected := runtime.GOOS == "linux"
	if IsLinux() != expected {
		t.Errorf("expected %v, got %v", expected, IsLinux())
	}
}

func TestHomeDir(t *testing.T) {
	home := HomeDir()
	if home == "" {
		t.Error("expected non-empty home directory")
	}

	// Should be an absolute path
	if !filepath.IsAbs(home) {
		t.Errorf("expected absolute path, got %s", home)
	}
}

func TestConfigDir(t *testing.T) {
	// Test with env var
	os.Setenv("VIBECODING_DIR", "/tmp/test-vibecoding")
	dir := ConfigDir()
	if dir != "/tmp/test-vibecoding" {
		t.Errorf("expected '/tmp/test-vibecoding', got '%s'", dir)
	}
	os.Unsetenv("VIBECODING_DIR")

	// Test default
	dir = ConfigDir()
	if dir == "" {
		t.Error("expected non-empty config dir")
	}

	// Should be an absolute path
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %s", dir)
	}
}

func TestDataDir(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Error("expected non-empty data dir")
	}
}

func TestCacheDir(t *testing.T) {
	dir := CacheDir()
	if dir == "" {
		t.Error("expected non-empty cache dir")
	}
}

func TestSessionDir(t *testing.T) {
	dir := SessionDir()
	if dir == "" {
		t.Error("expected non-empty session dir")
	}

	// Should contain "sessions"
	if !contains(dir, "sessions") {
		t.Errorf("expected dir to contain 'sessions', got %s", dir)
	}
}

func TestSkillsDir(t *testing.T) {
	dir := SkillsDir()
	if dir == "" {
		t.Error("expected non-empty skills dir")
	}

	// Should contain "skills"
	if !contains(dir, "skills") {
		t.Errorf("expected dir to contain 'skills', got %s", dir)
	}
}

func TestDefaultShell(t *testing.T) {
	shell := DefaultShell()
	if shell == "" {
		t.Error("expected non-empty shell")
	}

	// On Linux, should be bash or zsh
	if runtime.GOOS == "linux" {
		if shell != "/bin/bash" && shell != "/bin/zsh" {
			t.Errorf("expected /bin/bash or /bin/zsh, got %s", shell)
		}
	}
}

func TestShellArgs(t *testing.T) {
	tests := []struct {
		shell    string
		command  string
		expected []string
	}{
		{"/bin/bash", "echo hello", []string{"-c", "echo hello"}},
		{"/bin/zsh", "echo hello", []string{"-c", "echo hello"}},
		{"powershell.exe", "echo hello", []string{"-NoProfile", "-NonInteractive", "-Command", "echo hello"}},
		{"cmd.exe", "echo hello", []string{"/c", "echo hello"}},
	}

	for _, tt := range tests {
		args := ShellArgs(tt.shell, tt.command)
		if len(args) != len(tt.expected) {
			t.Errorf("expected %d args, got %d", len(tt.expected), len(args))
			continue
		}

		for i, arg := range args {
			if arg != tt.expected[i] {
				t.Errorf("expected arg[%d] = '%s', got '%s'", i, tt.expected[i], arg)
			}
		}
	}
}

func TestPathSeparator(t *testing.T) {
	sep := PathSeparator()
	if sep == "" {
		t.Error("expected non-empty path separator")
	}

	if runtime.GOOS == "windows" {
		if sep != "\\" {
			t.Errorf("expected '\\', got '%s'", sep)
		}
	} else {
		if sep != "/" {
			t.Errorf("expected '/', got '%s'", sep)
		}
	}
}

func TestJoinPath(t *testing.T) {
	path := JoinPath("home", "user", "test")
	if path == "" {
		t.Error("expected non-empty path")
	}

	expected := filepath.Join("home", "user", "test")
	if path != expected {
		t.Errorf("expected '%s', got '%s'", expected, path)
	}
}

func TestNormalizePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		path := NormalizePath("home/user/test")
		if path != "home\\user\\test" {
			t.Errorf("expected 'home\\user\\test', got '%s'", path)
		}
	} else {
		path := NormalizePath("home/user/test")
		if path != "home/user/test" {
			t.Errorf("expected 'home/user/test', got '%s'", path)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := HomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result := ExpandHome(tt.input)
		if result != tt.expected {
			t.Errorf("ExpandHome(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestCommonPaths(t *testing.T) {
	paths := CommonPaths()
	if len(paths) == 0 {
		t.Error("expected non-empty common paths")
	}

	// Should have home directory
	if _, ok := paths["home"]; !ok {
		t.Error("expected 'home' in common paths")
	}

	// Should have temp directory
	if _, ok := paths["temp"]; !ok {
		t.Error("expected 'temp' in common paths")
	}
}

func TestSandboxPaths(t *testing.T) {
	paths := SandboxPaths()
	if len(paths) == 0 {
		t.Error("expected non-empty sandbox paths")
	}

	// Should have at least one path
	if len(paths) < 1 {
		t.Error("expected at least one sandbox path")
	}
}

func TestDeniedPaths(t *testing.T) {
	paths := DeniedPaths()
	if len(paths) == 0 {
		t.Error("expected non-empty denied paths")
	}
}

func TestDefaultEnvVars(t *testing.T) {
	vars := DefaultEnvVars()
	if len(vars) == 0 {
		t.Error("expected non-empty env vars")
	}

	// Should have PATH
	found := false
	for _, v := range vars {
		if v == "PATH" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'PATH' in env vars")
	}
}

func TestTempDir(t *testing.T) {
	dir := TempDir()
	if dir == "" {
		t.Error("expected non-empty temp dir")
	}
}

func TestExecutableExt(t *testing.T) {
	ext := ExecutableExt()
	if runtime.GOOS == "windows" {
		if ext != ".exe" {
			t.Errorf("expected '.exe', got '%s'", ext)
		}
	} else {
		if ext != "" {
			t.Errorf("expected empty string, got '%s'", ext)
		}
	}
}

func TestIsExecutable(t *testing.T) {
	// Test with executable file
	executable := os.FileMode(0755)
	if !IsExecutable(executable) {
		t.Error("expected executable file to be executable")
	}

	// Test with non-executable file
	nonExecutable := os.FileMode(0644)
	if runtime.GOOS != "windows" {
		if IsExecutable(nonExecutable) {
			t.Error("expected non-executable file to not be executable")
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
