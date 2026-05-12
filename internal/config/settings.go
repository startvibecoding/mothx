package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Settings holds all configuration for vibecoding.
type Settings struct {
	// Provider settings
	DefaultProvider     string `json:"defaultProvider,omitempty"`
	DefaultModel        string `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string `json:"defaultThinkingLevel,omitempty"`

	// Mode settings
	DefaultMode string `json:"defaultMode,omitempty"` // "plan", "agent", "yolo"

	// Context settings
	MaxContextTokens int  `json:"maxContextTokens,omitempty"` // 0 = use model default
	MaxOutputTokens  int  `json:"maxOutputTokens,omitempty"`  // 0 = use model default

	// Compaction settings
	Compaction CompactionSettings `json:"compaction"`

	// Sandbox settings
	Sandbox SandboxSettings `json:"sandbox"`

	// Session settings
	SessionDir string `json:"sessionDir,omitempty"`

	// Shell settings
	ShellPath          string `json:"shellPath,omitempty"`
	ShellCommandPrefix string `json:"shellCommandPrefix,omitempty"`

	// Theme
	Theme string `json:"theme,omitempty"`

	// Retry settings
	Retry RetrySettings `json:"retry"`
}

// CompactionSettings controls context compaction behavior.
type CompactionSettings struct {
	Enabled         bool `json:"enabled"`
	ReserveTokens   int  `json:"reserveTokens"`
	KeepRecentTokens int `json:"keepRecentTokens"`
}

// SandboxSettings controls sandbox behavior.
type SandboxSettings struct {
	Enabled      bool     `json:"enabled"`
	Level        string   `json:"level"`         // "strict", "standard", "none"
	BwrapPath    string   `json:"bwrapPath,omitempty"`
	AllowNetwork bool     `json:"allowNetwork"`
	AllowedRead  []string `json:"allowedRead,omitempty"`
	AllowedWrite []string `json:"allowedWrite,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`
	PassEnv      []string `json:"passEnv,omitempty"`
	TmpSize      string   `json:"tmpSize,omitempty"`
}

// RetrySettings controls retry behavior.
type RetrySettings struct {
	Enabled     bool `json:"enabled"`
	MaxRetries  int  `json:"maxRetries"`
	BaseDelayMs int  `json:"baseDelayMs"`
}

// DefaultSettings returns the default settings.
func DefaultSettings() *Settings {
	homeDir, _ := os.UserHomeDir()

	return &Settings{
		DefaultProvider:      "anthropic",
		DefaultModel:         "claude-sonnet-4-20250514",
		DefaultThinkingLevel: "medium",
		DefaultMode:          "agent",
		MaxContextTokens:     0,
		MaxOutputTokens:      0,
		Compaction: CompactionSettings{
			Enabled:          true,
			ReserveTokens:    16384,
			KeepRecentTokens: 20000,
		},
		Sandbox: SandboxSettings{
			Enabled: true,
			Level:   "standard",
			AllowedRead: []string{
				"/usr", "/lib", "/lib64", "/bin", "/sbin",
				"/etc/ld.so.cache", "/etc/ssl", "/etc/ca-certificates",
				"/dev/null", "/dev/urandom", "/dev/zero",
				"/proc/self", "/proc/meminfo", "/proc/cpuinfo",
			},
			DeniedPaths: []string{
				"/etc/shadow", "/etc/gshadow", "/etc/passwd",
				"/root", "/home",
			},
			PassEnv: []string{
				"PATH", "HOME", "USER", "SHELL",
				"GOPATH", "GOROOT", "GOPROXY", "GOMODCACHE",
				"NODE_PATH", "NPM_CONFIG_PREFIX",
				"LANG", "LC_ALL", "TERM",
			},
			TmpSize: "100m",
		},
		SessionDir: filepath.Join(homeDir, ".vibecoding", "sessions"),
		Theme:      "dark",
		Retry: RetrySettings{
			Enabled:     true,
			MaxRetries:  3,
			BaseDelayMs: 2000,
		},
	}
}

// ConfigDir returns the vibecoding config directory.
func ConfigDir() string {
	if dir := os.Getenv("VIBECODING_DIR"); dir != "" {
		return dir
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".vibecoding")
}

// GlobalSettingsPath returns the path to global settings.json.
func GlobalSettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

// ProjectSettingsPath returns the path to project-level settings.json.
func ProjectSettingsPath() string {
	return filepath.Join(".vibe", "settings.json")
}

// LoadSettings loads and merges global + project settings.
func LoadSettings() (*Settings, error) {
	s := DefaultSettings()

	// Load global settings
	if data, err := os.ReadFile(GlobalSettingsPath()); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("parse global settings: %w", err)
		}
	}

	// Load and merge project settings
	if data, err := os.ReadFile(ProjectSettingsPath()); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("parse project settings: %w", err)
		}
	}

	// Apply environment variable overrides
	if v := os.Getenv("VIBECODING_PROVIDER"); v != "" {
		s.DefaultProvider = v
	}
	if v := os.Getenv("VIBECODING_MODEL"); v != "" {
		s.DefaultModel = v
	}
	if v := os.Getenv("VIBECODING_MODE"); v != "" {
		s.DefaultMode = v
	}
	if v := os.Getenv("VIBECODING_THINKING"); v != "" {
		s.DefaultThinkingLevel = v
	}

	return s, nil
}

// AuthData holds authentication credentials.
type AuthData struct {
	Entries map[string]AuthEntry `json:"entries"`
}

// AuthEntry represents a single auth entry.
type AuthEntry struct {
	Type string `json:"type"` // "api_key"
	Key  string `json:"key"`
}

// AuthFilePath returns the path to auth.json.
func AuthFilePath() string {
	return filepath.Join(ConfigDir(), "auth.json")
}

// LoadAuth loads authentication data from auth.json.
func LoadAuth() (*AuthData, error) {
	data := &AuthData{
		Entries: make(map[string]AuthEntry),
	}

	raw, err := os.ReadFile(AuthFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(raw, &data.Entries); err != nil {
		return nil, fmt.Errorf("parse auth.json: %w", err)
	}

	return data, nil
}

// ResolveKey resolves an API key for a provider.
// Priority: auth.json > environment variable.
func ResolveKey(providerName string) string {
	// Try auth.json first
	auth, err := LoadAuth()
	if err == nil {
		if entry, ok := auth.Entries[providerName]; ok && entry.Key != "" {
			key := entry.Key
			// Shell command
			if strings.HasPrefix(key, "!") {
				return resolveShellCommand(key[1:])
			}
			// Environment variable reference
			if v := os.Getenv(key); v != "" {
				return v
			}
			// Literal value
			return key
		}
	}

	// Fall back to environment variables
	envMap := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
	}
	if envVar, ok := envMap[providerName]; ok {
		return os.Getenv(envVar)
	}

	return ""
}

func resolveShellCommand(cmd string) string {
	// Simple shell command execution for key resolution
	// In production, this would use os/exec with proper error handling
	return ""
}

// GetShell returns the shell path to use.
func (s *Settings) GetShell() string {
	if s.ShellPath != "" {
		return s.ShellPath
	}
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "/bin/bash"
}

// GetSessionDir returns the session directory.
func (s *Settings) GetSessionDir() string {
	if s.SessionDir != "" {
		if strings.HasPrefix(s.SessionDir, "~") {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, s.SessionDir[1:])
		}
		return s.SessionDir
	}
	return filepath.Join(ConfigDir(), "sessions")
}

// SaveGlobalSettings saves settings to the global config file.
func SaveGlobalSettings(s *Settings) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GlobalSettingsPath(), data, 0600)
}
