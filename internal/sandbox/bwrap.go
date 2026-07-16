package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// BwrapSandbox implements sandbox using bubblewrap (bwrap).
type BwrapSandbox struct {
	level      Level
	projectDir string
	bwrapPath  string
	availMu    sync.Mutex
	available  *bool // cached availability check
	options    Options
}

// NewBwrapSandbox creates a new bubblewrap sandbox.
func NewBwrapSandbox(projectDir string, level Level) *BwrapSandbox {
	return NewBwrapSandboxWithOptions(projectDir, level, Options{})
}

// NewBwrapSandboxWithOptions creates a bubblewrap sandbox with a policy.
func NewBwrapSandboxWithOptions(projectDir string, level Level, opts Options) *BwrapSandbox {
	absDir, _ := filepath.Abs(projectDir)
	bwrapPath := opts.BwrapPath
	if bwrapPath == "" {
		bwrapPath = findBwrap()
	}
	return &BwrapSandbox{level: level, projectDir: absDir, bwrapPath: bwrapPath, options: opts}
}

// findBwrap locates the bwrap binary.
func findBwrap() string {
	// Check common locations
	candidates := []string{
		"/usr/bin/bwrap",
		"/usr/local/bin/bwrap",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Try PATH
	if path, err := exec.LookPath("bwrap"); err == nil {
		return path
	}

	return ""
}

// IsAvailable checks if bwrap is available on this system.
func (s *BwrapSandbox) IsAvailable() bool {
	s.availMu.Lock()
	defer s.availMu.Unlock()

	if s.available != nil {
		return *s.available
	}

	// bwrap is Linux only
	if runtime.GOOS != "linux" {
		f := false
		s.available = &f
		return false
	}

	// Check if bwrap binary exists
	if s.bwrapPath == "" {
		f := false
		s.available = &f
		return false
	}

	// Test that bwrap works with a minimal but complete sandbox.
	// We need to mount enough of the system for /bin/true to execute,
	// including /lib64 for the dynamic linker on multiarch systems.
	cmd := exec.Command(s.bwrapPath,
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", "/bin", "/bin",
		"/bin/true",
	)
	if err := cmd.Run(); err != nil {
		f := false
		s.available = &f
		return false
	}

	t := true
	s.available = &t
	return true
}

// Name returns "bwrap".
func (s *BwrapSandbox) Name() string {
	return "bwrap"
}

// Level returns the sandbox level.
func (s *BwrapSandbox) Level() Level {
	return s.level
}

// WrapCommand wraps a command for execution inside bubblewrap.
func (s *BwrapSandbox) WrapCommand(ctx context.Context, shell, cmd string, opts ExecOpts) *exec.Cmd {
	args := s.buildBwrapArgs(opts, shell, cmd)
	c := exec.CommandContext(ctx, s.bwrapPath, args...)
	c.Dir = opts.WorkDir

	// Pass through allowed environment variables.
	// bwrap inherits c.Env into the sandbox; --setenv in buildBwrapArgs overrides specific keys.
	c.Env = s.buildEnv(opts)

	return c
}

// buildBwrapArgs constructs the bwrap command arguments.
func (s *BwrapSandbox) buildBwrapArgs(opts ExecOpts, shell, cmd string) []string {
	args := []string{
		// Unshare namespaces
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts", // Required for --hostname

		// Die when parent dies
		"--die-with-parent",

		// Proc filesystem (minimal)
		"--proc", "/proc",

		// Dev filesystem (minimal - null, zero, urandom)
		"--dev", "/dev",

		// Network isolation (unless explicitly allowed)
	}
	if !opts.NetworkAccess && !s.options.AllowNetwork {
		args = append(args, "--unshare-net")
	}

	// Tmp filesystem with size limit.
	// --size must immediately precede --tmpfs.
	tmpSize := s.options.TmpSize
	if tmpSize == "" {
		tmpSize = "100000000"
	}
	args = append(args, "--size", tmpSize, "--tmpfs", "/tmp")

	// System libraries (read-only)
	systemPaths := []string{"/usr", "/lib", "/lib64", "/bin", "/sbin"}
	for _, p := range systemPaths {
		if _, err := os.Stat(p); err == nil {
			args = append(args, "--ro-bind", p, p)
		}
	}

	// Additional system paths
	roPaths := []string{
		"/etc/ld.so.cache",
		"/etc/ssl",
		"/etc/ca-certificates",
		"/etc/resolv.conf",
		"/etc/hosts",
		"/etc/nsswitch.conf",
	}
	for _, p := range roPaths {
		if _, err := os.Stat(p); err == nil {
			args = append(args, "--ro-bind", p, p)
		}
	}

	// Home directory: use tmpfs to prevent access to real home
	// NOTE: This must be set BEFORE project directory binding if project is under home
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		args = append(args, "--tmpfs", homeDir)
	}

	// Project directory binding (must be after home tmpfs if project is under home)
	if s.projectDir != "" {
		if s.level == LevelStrict {
			// Read-only in strict mode
			args = append(args, "--ro-bind", s.projectDir, s.projectDir)
		} else {
			// Read-write in standard mode
			args = append(args, "--bind", s.projectDir, s.projectDir)
		}
	}

	// Configured paths are applied after the project bind so explicit policy
	// paths are also visible to commands. A denied path is never bound.
	for _, p := range s.options.AllowedRead {
		if !s.denied(p) {
			if _, err := os.Stat(p); err == nil {
				args = append(args, "--ro-bind", p, p)
			}
		}
	}
	for _, p := range s.options.AllowedWrite {
		if !s.denied(p) {
			if _, err := os.Stat(p); err == nil {
				args = append(args, "--bind", p, p)
			}
		}
	}

	// Additional read-only paths from options
	for _, p := range opts.ReadOnlyPaths {
		if !s.denied(p) {
			if _, err := os.Stat(p); err == nil {
				args = append(args, "--ro-bind", p, p)
			}
		}
	}

	// Additional writable paths from options
	for _, p := range opts.WritablePaths {
		if !s.denied(p) {
			if _, err := os.Stat(p); err == nil {
				args = append(args, "--bind", p, p)
			}
		}
	}

	// Set hostname
	args = append(args, "--hostname", "sandbox")

	// Working directory
	if opts.WorkDir != "" {
		args = append(args, "--chdir", opts.WorkDir)
	} else if s.projectDir != "" {
		args = append(args, "--chdir", s.projectDir)
	}

	// Environment variables: override specific keys inside the sandbox
	for k, v := range opts.EnvVars {
		args = append(args, "--setenv", k, v)
	}

	// The actual command
	args = append(args, shell, "-c", cmd)

	return args
}

// buildEnv constructs the environment for the sandboxed process.
func (s *BwrapSandbox) buildEnv(opts ExecOpts) []string {
	var env []string

	// Default pass-through variables
	defaultPass := []string{
		"PATH", "LANG", "LC_ALL", "TERM",
		"GOPATH", "GOROOT", "GOPROXY", "GOMODCACHE",
		"NODE_PATH", "NPM_CONFIG_PREFIX",
		"HOME", "USER", "SHELL",
	}

	defaultPass = append(defaultPass, s.options.PassEnv...)

	passVars := make(map[string]bool)
	for _, v := range defaultPass {
		passVars[v] = true
	}

	// Add explicitly passed env vars
	if v := os.Getenv("VIBECODING_SANDBOX_PASS_ENV"); v != "" {
		for _, name := range strings.Split(v, ",") {
			passVars[strings.TrimSpace(name)] = true
		}
	}

	// Copy allowed env vars from current environment
	for _, e := range os.Getenv("PATH") {
		_ = e
	}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if passVars[name] {
			env = append(env, entry)
		}
	}

	for k, v := range opts.EnvVars {
		env = replaceEnv(env, k, v)
	}

	// Set HOME to the sandbox-isolated home directory (tmpfs mounted over real home).
	// Only override if the caller did not explicitly set HOME via EnvVars.
	if _, ok := opts.EnvVars["HOME"]; !ok {
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			env = append(env, "HOME="+homeDir)
		} else {
			env = append(env, "HOME=/tmp")
		}
	}

	return env
}

func (s *BwrapSandbox) denied(path string) bool {
	path = filepath.Clean(path)
	for _, denied := range s.options.DeniedPaths {
		denied = filepath.Clean(denied)
		if path == denied || strings.HasPrefix(path, denied+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// FormatSandboxInfo returns a human-readable description of the sandbox state.
func FormatSandboxInfo(s Sandbox) string {
	if s == nil || s.Level() == LevelNone {
		return "🔓 No sandbox"
	}

	available := "✓"
	if !s.IsAvailable() {
		available = "✗"
	}

	name := s.Name()
	switch s.Level() {
	case LevelStrict:
		return fmt.Sprintf("🔒 Strict sandbox [%s: %s] - read-only project, no network", name, available)
	case LevelStandard:
		return fmt.Sprintf("🔒 Standard sandbox [%s: %s] - read-write project, no network", name, available)
	default:
		return "🔓 No sandbox"
	}
}
