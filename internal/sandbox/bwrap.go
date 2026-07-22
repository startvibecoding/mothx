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
	level           Level
	projectDir      string
	bwrapPath       string
	availMu         sync.Mutex
	available       *bool // cached availability check
	availabilityErr error
	capabilities    *BwrapCapabilities
	options         Options
	gitPaths        []string
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
	gitPaths := []string{}
	if opts.ProtectGit {
		// .git is intentionally visible to sandboxed commands. Keep resolving
		// its paths for compatibility with one-shot Git access handling, but do
		// not add them to DeniedPaths.
		gitPaths = protectedGitPaths(absDir)
	}
	if opts.TmpSize != "" {
		// NewBwrapSandboxWithOptions is also used directly by tests and platform
		// integrations, outside Manager's NormalizeOptions path. Keep bwrap's
		// wire format valid here as well: --size accepts decimal bytes only.
		if normalizedSize, err := normalizeTmpSize(opts.TmpSize); err == nil {
			opts.TmpSize = normalizedSize
		} else {
			// Manager reports invalid policy through initErr. A direct backend
			// construction has no error return, so use the safe default rather
			// than passing an invalid value to bwrap.
			opts.TmpSize = "100000000"
		}
	}
	return &BwrapSandbox{level: level, projectDir: absDir, bwrapPath: bwrapPath, options: opts, gitPaths: uniquePaths(gitPaths)}
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

// BwrapCapabilities describes the flags and runtime features required by MothX.
type BwrapCapabilities struct {
	UnshareUser   bool
	UnsharePID    bool
	UnshareIPC    bool
	UnshareUTS    bool
	NewSession    bool
	DieWithParent bool
	MountProc     bool
	MountDev      bool
	MountTmpfs    bool
	TmpfsSize     bool
	MountBind     bool
	ChangeDir     bool
	Hostname      bool
}

func (c BwrapCapabilities) complete() bool {
	return c.UnshareUser && c.UnsharePID && c.UnshareIPC && c.UnshareUTS &&
		c.NewSession && c.DieWithParent && c.MountProc && c.MountDev &&
		c.MountTmpfs && c.TmpfsSize && c.MountBind && c.ChangeDir &&
		c.Hostname
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
		return s.markUnavailable("bubblewrap is only supported on Linux")
	}

	// Check if bwrap binary exists
	if s.bwrapPath == "" {
		return s.markUnavailable("bwrap binary not found in PATH")
	}

	caps, ok := probeBwrapCapabilities(s.bwrapPath)
	if !ok || !caps.complete() {
		s.capabilities = &caps
		if !ok {
			return s.markUnavailable("failed to query bwrap capabilities")
		}
		return s.markUnavailable("bwrap is missing required capabilities")
	}
	s.capabilities = &caps

	// Verify the exact profile used for commands, rather than a minimal
	// invocation. This catches bwrap versions and host policies that accept
	// --help but reject a flag or mount arrangement we actually require.
	shell, err := exec.LookPath("sh")
	if err != nil {
		return s.markUnavailable(fmt.Sprintf("shell not found: %v", err))
	}
	args := s.buildBwrapArgs(ExecOpts{WorkDir: s.projectDir}, shell, "test -r /proc/self/status && test -d /tmp && test -w /tmp")
	cmd := exec.Command(s.bwrapPath, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		reason := strings.TrimSpace(string(output))
		if reason == "" {
			reason = err.Error()
		} else {
			reason = fmt.Sprintf("%v: %s", err, reason)
		}
		return s.markUnavailable(fmt.Sprintf("bwrap probe failed: %s", reason))
	}

	t := true
	s.available = &t
	return true
}

func (s *BwrapSandbox) markUnavailable(reason string) bool {
	f := false
	s.available = &f
	s.availabilityErr = fmt.Errorf("%s", reason)
	return false
}

// AvailabilityError explains why bwrap could not be used.
func (s *BwrapSandbox) AvailabilityError() error {
	return s.availabilityErr
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

// WrapCommandWithGitAccess runs one command with the protected .git deny
// carveout removed. The sandbox instance itself is immutable; only this
// prepared command receives the temporary transform.
func (s *BwrapSandbox) WrapCommandWithGitAccess(ctx context.Context, shell, cmd string, opts ExecOpts) *exec.Cmd {
	// Construct a fresh BwrapSandbox value instead of copying *s by value,
	// which would duplicate the embedded sync.Mutex and trip go vet. The
	// derived instance is only used to build one command and is never shared,
	// so its availability cache (availMu/available) stays unused.
	clone := &BwrapSandbox{
		level:      s.level,
		projectDir: s.projectDir,
		bwrapPath:  s.bwrapPath,
		options:    s.options,
		gitPaths:   s.gitPaths,
	}
	clone.options.DeniedPaths = make([]string, 0, len(s.options.DeniedPaths))
	for _, path := range s.options.DeniedPaths {
		if !containsPath(s.gitPaths, path) {
			clone.options.DeniedPaths = append(clone.options.DeniedPaths, path)
		}
	}
	for _, path := range s.gitPaths {
		if _, err := os.Lstat(path); err == nil && !containsPath(clone.options.DeniedPaths, path) {
			clone.options.AllowedWrite = append(clone.options.AllowedWrite, path)
		}
	}
	args := clone.buildBwrapArgs(opts, shell, cmd)
	c := exec.CommandContext(ctx, clone.bwrapPath, args...)
	c.Dir = opts.WorkDir
	c.Env = clone.buildEnv(opts)
	return c
}

// buildBwrapArgs constructs the bwrap command arguments.
func (s *BwrapSandbox) buildBwrapArgs(opts ExecOpts, shell, cmd string) []string {
	args := []string{
		// Explicit user namespace avoids relying on bwrap's implicit behavior,
		// especially when invoked by uid 0 inside a container.
		"--unshare-user",
		"--new-session",
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts", // Required for --hostname

		// Die when parent dies
		"--die-with-parent",

		// Proc filesystem. bwrap must initialize PID entries before any
		// remount; forcing remount-ro here breaks /proc/<pid> creation.
		"--proc", "/proc",

		// Dev filesystem (minimal - null, zero, urandom)
		"--dev", "/dev",

		// Network access is intentionally preserved. The sandbox isolates process
		// and filesystem state, but does not create a network namespace.
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
		// /proc is created by --proc above. Binding entries from the host proc
		// tree is invalid after --unshare-pid because process paths change.
		// /dev is created by --dev above; rebinding host device nodes (notably
		// /dev/null) can turn them into regular read-only files and break tools
		// such as git that open them for read/write.
		if isProcPath(p) || isDevPath(p) {
			continue
		}
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
		if isProcPath(p) || isDevPath(p) {
			continue
		}
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

	// Denied paths are masked after every broad bind. bwrap cannot express a
	// deny rule directly; an empty tmpfs hides both the original directory and
	// all of its children.
	for _, p := range s.options.DeniedPaths {
		// The home tmpfs already hides a denied ancestor of the project. Masking
		// it after binding the project would also hide the project mount.
		if pathContains(p, s.projectDir) {
			continue
		}
		info, err := os.Lstat(p)
		if err != nil {
			// --dir creates the target only inside the new mount namespace;
			// masking it then prevents future creation through this path.
			args = append(args, "--dir", p, "--tmpfs", p)
			continue
		}
		if info.IsDir() {
			args = append(args, "--tmpfs", p)
		} else {
			args = append(args, "--ro-bind", "/dev/null", p)
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

func isProcPath(path string) bool {
	clean := filepath.Clean(path)
	return clean == "/proc" || strings.HasPrefix(clean, "/proc"+string(os.PathSeparator))
}

func isDevPath(path string) bool {
	clean := filepath.Clean(path)
	return clean == "/dev" || strings.HasPrefix(clean, "/dev"+string(os.PathSeparator))
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

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == filepath.Clean(target) {
			return true
		}
	}
	return false
}

func (s *BwrapSandbox) denied(path string) bool {
	path = filepath.Clean(path)
	for _, denied := range s.options.DeniedPaths {
		if pathsOverlap(path, denied) {
			return true
		}
	}
	return false
}

func pathContains(parent, path string) bool {
	return parent == path || strings.HasPrefix(path, parent+string(os.PathSeparator))
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
		return fmt.Sprintf("🔒 Strict sandbox [%s: %s] - read-only project, host network", name, available)
	case LevelStandard:
		return fmt.Sprintf("🔒 Standard sandbox [%s: %s] - read-write project, host network", name, available)
	default:
		return "🔓 No sandbox"
	}
}
