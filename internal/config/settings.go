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
	Providers          map[string]ProviderConfig `json:"providers,omitempty"`
	DefaultProvider    string                    `json:"defaultProvider,omitempty"`
	DefaultModel       string                    `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string                  `json:"defaultThinkingLevel,omitempty"`

	// Mode settings
	DefaultMode string `json:"defaultMode,omitempty"` // "plan", "agent", "yolo"

	// Context settings
	MaxContextTokens int  `json:"maxContextTokens,omitempty"` // 0 = use model default
	MaxOutputTokens  int  `json:"maxOutputTokens,omitempty"`  // 0 = use model default

	// Context files settings
	ContextFiles ContextFilesSettings `json:"contextFiles"`

	// Skills settings
	SkillsDir string `json:"skillsDir,omitempty"` // global skills dir, default ~/.vibecoding/skills

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

// ProviderConfig holds configuration for a single provider.
type ProviderConfig struct {
	APIKey  string        `json:"apiKey,omitempty"`
	BaseURL string        `json:"baseUrl,omitempty"`
	API     string        `json:"api,omitempty"`      // "openai-chat", "anthropic-messages" (default: "openai-chat")
	Models  []ModelConfig `json:"models"`
}

// ModelConfig holds configuration for a single model.
type ModelConfig struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Reasoning     bool     `json:"reasoning,omitempty"`
	ContextWindow int      `json:"contextWindow,omitempty"`
	MaxTokens     int      `json:"maxTokens,omitempty"`
	CostInput     float64  `json:"costInput,omitempty"`
	CostOutput    float64  `json:"costOutput,omitempty"`
	Input         []string `json:"input,omitempty"` // "text", "image"
}

// ContextFilesSettings controls which context files to load.
type ContextFilesSettings struct {
	Enabled   bool     `json:"enabled"`
	ExtraFiles []string `json:"extraFiles,omitempty"` // additional file names to look for
}

// CompactionSettings controls context compaction behavior.
type CompactionSettings struct {
	Enabled          bool `json:"enabled"`
	ReserveTokens    int  `json:"reserveTokens"`
	KeepRecentTokens int  `json:"keepRecentTokens"`
}

// SandboxSettings controls sandbox behavior.
type SandboxSettings struct {
	Enabled      bool     `json:"enabled"`
	Level        string   `json:"level"` // "strict", "standard", "none"
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
		Providers: map[string]ProviderConfig{
			"anthropic": {
				Models: []ModelConfig{
					{ID: "claude-sonnet-4-20250514", Name: "Claude 4 Sonnet", Reasoning: true, ContextWindow: 200000, MaxTokens: 16384, CostInput: 3.0, CostOutput: 15.0, Input: []string{"text", "image"}},
					{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Reasoning: false, ContextWindow: 200000, MaxTokens: 8192, CostInput: 3.0, CostOutput: 15.0, Input: []string{"text", "image"}},
					{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Reasoning: false, ContextWindow: 200000, MaxTokens: 8192, CostInput: 0.8, CostOutput: 4.0, Input: []string{"text", "image"}},
					{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Reasoning: false, ContextWindow: 200000, MaxTokens: 4096, CostInput: 15.0, CostOutput: 75.0, Input: []string{"text", "image"}},
				},
			},
			"openai": {
				Models: []ModelConfig{
					{ID: "gpt-4o", Name: "GPT-4o", Reasoning: false, ContextWindow: 128000, MaxTokens: 16384, CostInput: 2.5, CostOutput: 10.0, Input: []string{"text", "image"}},
					{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Reasoning: false, ContextWindow: 128000, MaxTokens: 16384, CostInput: 0.15, CostOutput: 0.6, Input: []string{"text", "image"}},
					{ID: "o1", Name: "o1", Reasoning: true, ContextWindow: 200000, MaxTokens: 100000, CostInput: 15.0, CostOutput: 60.0, Input: []string{"text", "image"}},
					{ID: "o3-mini", Name: "o3-mini", Reasoning: true, ContextWindow: 200000, MaxTokens: 100000, CostInput: 1.1, CostOutput: 4.4, Input: []string{"text", "image"}},
				},
			},
		},
		DefaultProvider:      "anthropic",
		DefaultModel:         "claude-sonnet-4-20250514",
		DefaultThinkingLevel: "medium",
		DefaultMode:          "agent",
		MaxContextTokens:     0,
		MaxOutputTokens:      0,
		ContextFiles: ContextFilesSettings{
			Enabled: true,
		},
		SkillsDir: filepath.Join(homeDir, ".vibecoding", "skills"),
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
// If the config directory or settings.json doesn't exist, creates them with defaults.
func LoadSettings() (*Settings, error) {
	s := DefaultSettings()

	// Ensure config directory and settings.json exist
	if err := ensureConfigExists(s); err != nil {
		// Non-fatal: continue with defaults even if we can't create config
		fmt.Fprintf(os.Stderr, "Warning: could not create config: %v\n", err)
	}

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

// ensureConfigExists creates the config directory and settings.json if they don't exist.
func ensureConfigExists(defaults *Settings) error {
	configDir := ConfigDir()
	settingsPath := GlobalSettingsPath()

	// Check if settings.json already exists
	if _, err := os.Stat(settingsPath); err == nil {
		return nil // already exists
	}

	// Create config directory with proper permissions
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Marshal default settings with nice formatting
	data, err := json.MarshalIndent(defaults, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal default settings: %w", err)
	}

	// Write settings file
	if err := os.WriteFile(settingsPath, data, 0600); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created default config: %s\n", settingsPath)
	return nil
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
// Priority: settings.providers[name].apiKey > auth.json > environment variable.
func (s *Settings) ResolveKey(providerName string) string {
	// 1. Check settings.providers
	if pc, ok := s.Providers[providerName]; ok && pc.APIKey != "" {
		return resolveKeyValue(pc.APIKey)
	}

	// 2. Check auth.json
	auth, err := LoadAuth()
	if err == nil {
		if entry, ok := auth.Entries[providerName]; ok && entry.Key != "" {
			return resolveKeyValue(entry.Key)
		}
	}

	// 3. Environment variables
	envMap := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
	}
	if envVar, ok := envMap[providerName]; ok {
		return os.Getenv(envVar)
	}

	return ""
}

// resolveKeyValue resolves a key value that could be a shell command, env var, or literal.
func resolveKeyValue(key string) string {
	// Shell command
	if strings.HasPrefix(key, "!") {
		return resolveShellCommand(key[1:])
	}
	// Environment variable reference (single word that looks like an env var)
	if v := os.Getenv(key); v != "" && !strings.Contains(key, " ") {
		return v
	}
	// Literal value
	return key
}

// GetProviderConfig returns the provider config for the given name.
func (s *Settings) GetProviderConfig(name string) *ProviderConfig {
	if pc, ok := s.Providers[name]; ok {
		return &pc
	}
	return nil
}

// GetModelConfig returns the model config for the given provider and model ID.
func (s *Settings) GetModelConfig(providerName, modelID string) *ModelConfig {
	pc := s.GetProviderConfig(providerName)
	if pc == nil {
		return nil
	}
	for _, m := range pc.Models {
		if m.ID == modelID {
			return &m
		}
	}
	return nil
}

func resolveShellCommand(cmd string) string {
	// Simple shell command execution for key resolution
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

// GetGlobalSkillsDir returns the global skills directory.
func (s *Settings) GetGlobalSkillsDir() string {
	if s.SkillsDir != "" {
		if strings.HasPrefix(s.SkillsDir, "~") {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, s.SkillsDir[1:])
		}
		return s.SkillsDir
	}
	return filepath.Join(ConfigDir(), "skills")
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
