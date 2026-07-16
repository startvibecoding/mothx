package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Level defines the sandbox restriction level.
type Level int

const (
	LevelStrict   Level = iota // Required sandbox: read-only project
	LevelStandard              // Best-effort sandbox: read-write project
	LevelNone                  // Direct execution
)

// String returns the string representation of a Level.
func (l Level) String() string {
	switch l {
	case LevelStrict:
		return "strict"
	case LevelStandard:
		return "standard"
	case LevelNone:
		return "none"
	default:
		return "unknown"
	}
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) (Level, error) {
	switch s {
	case "strict":
		return LevelStrict, nil
	case "standard":
		return LevelStandard, nil
	case "none":
		return LevelNone, nil
	default:
		return LevelNone, fmt.Errorf("unknown sandbox level: %s", s)
	}
}

// ExecOpts contains options for executing a command in a sandbox.
type ExecOpts struct {
	WritablePaths []string          // Additional writable paths
	ReadOnlyPaths []string          // Additional read-only paths (for standard mode)
	NetworkAccess bool              // Deprecated: sandbox preserves host network access
	EnvVars       map[string]string // Additional environment variables
	WorkDir       string            // Working directory
	Timeout       time.Duration     // Command timeout
}

// Options controls sandbox backends. It deliberately uses primitive values so
// callers from every runtime can share the same policy without importing config.
type Options struct {
	BwrapPath    string
	AllowNetwork bool // Deprecated: sandbox always preserves host network access
	AllowedRead  []string
	AllowedWrite []string
	DeniedPaths  []string
	PassEnv      []string
	TmpSize      string
	ProtectGit   bool
}

// Sandbox is the interface for sandbox implementations.
type Sandbox interface {
	// WrapCommand wraps a command for execution inside the sandbox.
	WrapCommand(ctx context.Context, shell, cmd string, opts ExecOpts) *exec.Cmd

	// IsAvailable checks if the sandbox can be used on this system.
	IsAvailable() bool

	// Name returns the sandbox implementation name.
	Name() string

	// Level returns the sandbox level.
	Level() Level
}

// GitAccessSandbox is implemented by backends that can temporarily remove the
// protected Git deny rule for exactly one command.
type GitAccessSandbox interface {
	WrapCommandWithGitAccess(ctx context.Context, shell, cmd string, opts ExecOpts) *exec.Cmd
}

type CommandCleanupProvider interface {
	CleanupCommand(*exec.Cmd)
}

type AvailabilityErrorProvider interface {
	AvailabilityError() error
}

// Manager manages sandbox selection based on mode and availability.
type Manager struct {
	sandboxes   map[Level]Sandbox
	active      Sandbox
	initErr     error
	fallbackErr error
}

// NewManager creates a manager with the default sandbox policy.
func NewManager(projectDir string) *Manager {
	return NewManagerWithOptions(projectDir, Options{})
}

// NewManagerWithOptions creates a manager using the supplied sandbox policy.
func NewManagerWithOptions(projectDir string, opts Options) *Manager {
	normalized, normalizeErr := NormalizeOptions(projectDir, opts)
	if normalizeErr == nil {
		opts = normalized
	}
	m := &Manager{
		initErr:   normalizeErr,
		sandboxes: make(map[Level]Sandbox),
	}

	// Register sandbox implementations
	m.sandboxes[LevelNone] = NewNoneSandbox()
	m.sandboxes[LevelStandard] = newPlatformSandboxWithOptions(projectDir, LevelStandard, opts)
	m.sandboxes[LevelStrict] = newPlatformSandboxWithOptions(projectDir, LevelStrict, opts)

	return m
}

// SetLevel activates the requested execution policy. Standard sandboxing is
// best-effort and falls back to direct execution when the platform backend is
// unavailable. Strict sandboxing is required and never silently degrades.
func (m *Manager) SetLevel(level Level) error {
	m.fallbackErr = nil
	if m.initErr != nil && level != LevelNone {
		if level == LevelStandard {
			m.active = m.sandboxes[LevelNone]
			m.fallbackErr = fmt.Errorf("invalid sandbox policy: %w", m.initErr)
			return nil
		}
		return fmt.Errorf("invalid sandbox policy: %w", m.initErr)
	}
	sb, ok := m.sandboxes[level]
	if !ok {
		return fmt.Errorf("no sandbox for level %s", level)
	}
	if !sb.IsAvailable() {
		reason := fmt.Errorf("sandbox %s not available", level)
		if diagnostic, ok := sb.(AvailabilityErrorProvider); ok && diagnostic.AvailabilityError() != nil {
			reason = fmt.Errorf("sandbox %s not available: %w", level, diagnostic.AvailabilityError())
		}
		if level == LevelStandard {
			m.active = m.sandboxes[LevelNone]
			m.fallbackErr = reason
			return nil
		}
		return reason
	}
	m.active = sb
	return nil
}

// FallbackError reports why a best-effort sandbox fell back to direct execution.
func (m *Manager) FallbackError() error {
	return m.fallbackErr
}

// GetActive returns the active sandbox.
func (m *Manager) GetActive() Sandbox {
	if m.active == nil {
		return m.sandboxes[LevelNone]
	}
	return m.active
}

// GetForLevel returns the sandbox for a specific level, checking availability.
func (m *Manager) GetForLevel(level Level) (Sandbox, error) {
	if m.initErr != nil && level != LevelNone {
		return nil, fmt.Errorf("invalid sandbox policy: %w", m.initErr)
	}
	sb, ok := m.sandboxes[level]
	if !ok {
		return nil, fmt.Errorf("no sandbox for level %s", level)
	}
	if !sb.IsAvailable() {
		return nil, fmt.Errorf("sandbox %s not available", level)
	}
	return sb, nil
}
