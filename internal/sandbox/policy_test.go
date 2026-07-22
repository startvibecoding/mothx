package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeTmpSize(t *testing.T) {
	for input, want := range map[string]string{"100m": "104857600", "1g": "1073741824", "4096": "4096", "2KB": "2048"} {
		got, err := normalizeTmpSize(input)
		if err != nil || got != want {
			t.Fatalf("normalizeTmpSize(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"0", "0m", "bad", "-1"} {
		if _, err := normalizeTmpSize(input); err == nil {
			t.Fatalf("normalizeTmpSize(%q) unexpectedly succeeded", input)
		}
	}
}

func TestNormalizeOptionsIgnoresLegacyLinuxHomeDeny(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux compatibility behavior")
	}
	project := "/home/free/src/vibecoding"
	opts, err := NormalizeOptions(project, Options{DeniedPaths: []string{"/home"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.DeniedPaths) != 0 {
		t.Fatalf("denied paths = %#v, want legacy /home removed", opts.DeniedPaths)
	}
}

func TestNormalizeOptionsRejectsOverlappingAllowAndDeny(t *testing.T) {
	project := t.TempDir()
	_, err := NormalizeOptions(project, Options{
		AllowedWrite: []string{project},
		DeniedPaths:  []string{filepath.Join(project, "secret")},
	})
	if err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("expected overlap error, got %v", err)
	}
}

func TestNormalizeOptionsKeepsGitVisible(t *testing.T) {
	project := t.TempDir()
	opts, err := NormalizeOptions(project, Options{
		DeniedPaths: []string{filepath.Join(project, ".git")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.DeniedPaths) != 0 {
		t.Fatalf("denied paths = %#v, want .git removed", opts.DeniedPaths)
	}
}

func TestNormalizeOptionsCanonicalizesRelativePaths(t *testing.T) {
	project := t.TempDir()
	cache := filepath.Join(project, "cache")
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	opts, err := NormalizeOptions(project, Options{AllowedRead: []string{"cache"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.AllowedRead) != 1 || opts.AllowedRead[0] != cache {
		t.Fatalf("allowed read = %#v, want %q", opts.AllowedRead, cache)
	}
}

func TestNormalizeOptionsRejectsDenyContainingProject(t *testing.T) {
	project := t.TempDir()
	_, err := NormalizeOptions(project, Options{DeniedPaths: []string{filepath.Dir(project)}})
	if err == nil || !strings.Contains(err.Error(), "contains project") {
		t.Fatalf("expected project containment error, got %v", err)
	}
}

func TestBwrapMasksDeniedPathAfterProjectBind(t *testing.T) {
	project := t.TempDir()
	gitDir := filepath.Join(project, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewBwrapSandboxWithOptions(project, LevelStandard, Options{DeniedPaths: []string{gitDir}})
	args := s.buildBwrapArgs(ExecOpts{WorkDir: project}, "/bin/sh", "true")
	projectBind := indexArgs(args, "--bind", project, project)
	mask := indexArgs(args, "--tmpfs", gitDir)
	if projectBind < 0 || mask < 0 || mask <= projectBind {
		t.Fatalf("project bind=%d deny mask=%d args=%#v", projectBind, mask, args)
	}
}

func TestBwrapDeniedFileIsMaskedWithNullDevice(t *testing.T) {
	project := t.TempDir()
	secret := filepath.Join(project, ".env")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewBwrapSandboxWithOptions(project, LevelStandard, Options{DeniedPaths: []string{secret}})
	args := s.buildBwrapArgs(ExecOpts{WorkDir: project}, "/bin/sh", "true")
	if indexArgs(args, "--ro-bind", "/dev/null", secret) < 0 {
		t.Fatalf("expected null-device mask, args=%#v", args)
	}
}

func TestBwrapProcIsReadable(t *testing.T) {
	project := t.TempDir()
	s := NewBwrapSandbox(project, LevelStandard)
	if !s.IsAvailable() {
		t.Skip("bwrap unavailable")
	}
	cmd := s.WrapCommand(context.Background(), "/bin/sh", "test -r /proc/self/status && test -d /proc/1", ExecOpts{WorkDir: project})
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("proc readability check failed: %v\n%s", err, output)
	}
}

func TestBwrapBasicDiagnosticsWork(t *testing.T) {
	project := t.TempDir()
	s := NewBwrapSandbox(project, LevelStandard)
	if !s.IsAvailable() {
		t.Skip("bwrap unavailable")
	}
	cmd := s.WrapCommand(context.Background(), "/bin/sh", "test \"$(hostname)\" = sandbox && ps >/dev/null", ExecOpts{WorkDir: project})
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sandbox diagnostics failed: %v\n%s", err, output)
	}
}

func TestBwrapStrictCannotWriteProject(t *testing.T) {
	if !NewBwrapSandbox(t.TempDir(), LevelStrict).IsAvailable() {
		t.Skip("bwrap unavailable")
	}
	project := t.TempDir()
	s := NewBwrapSandbox(project, LevelStrict)
	cmd := s.WrapCommand(context.Background(), "/bin/sh", "touch blocked", ExecOpts{WorkDir: project})
	if err := cmd.Run(); err == nil {
		t.Fatal("strict sandbox unexpectedly wrote project")
	}
}

func TestBwrapDeniedFileBlocksReadAndWrite(t *testing.T) {
	project := t.TempDir()
	if !NewBwrapSandbox(project, LevelStandard).IsAvailable() {
		t.Skip("bwrap unavailable")
	}
	secret := filepath.Join(project, ".env")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewBwrapSandboxWithOptions(project, LevelStandard, Options{DeniedPaths: []string{secret}})
	cmd := s.WrapCommand(context.Background(), "/bin/sh", `test "$(cat .env)" = "" && ! sh -c 'echo changed > .env'`, ExecOpts{WorkDir: project})
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("denied file was accessible: %v\n%s", err, output)
	}
	contents, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "secret" {
		t.Fatalf("host secret changed: %q", contents)
	}
}

func TestBwrapProtectGitLeavesGitMetadataVisible(t *testing.T) {
	project := t.TempDir()
	if !NewBwrapSandbox(project, LevelStandard).IsAvailable() {
		t.Skip("bwrap unavailable")
	}
	gitDir := filepath.Join(project, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("[core]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewBwrapSandboxWithOptions(project, LevelStandard, Options{ProtectGit: true})
	cmd := s.WrapCommand(context.Background(), "/bin/sh", "test -r .git/config && test -w .git/config", ExecOpts{WorkDir: project})
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git metadata was not visible: %v\n%s", err, output)
	}
}

func indexArgs(args []string, values ...string) int {
	for i := range args {
		if i+len(values) > len(args) {
			break
		}
		matched := true
		for j, value := range values {
			if args[i+j] != value {
				matched = false
				break
			}
		}
		if matched {
			return i
		}
	}
	return -1
}
