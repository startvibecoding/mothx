package openaiapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the OpenAI-compatible API configuration used by serve.
type Config struct {
	Listen               string               `json:"listen,omitempty"`
	Auth                 AuthConfig           `json:"auth"`
	DefaultMode          string               `json:"defaultMode,omitempty"`
	DefaultThinkingLevel string               `json:"defaultThinkingLevel,omitempty"`
	EnableSubAgents      bool                 `json:"enableSubAgents,omitempty"`
	EnableDelegate       bool                 `json:"enableDelegate,omitempty"`
	EnableWorkflows      bool                 `json:"enableWorkflows,omitempty"`
	EnableWebSearch      bool                 `json:"enableWebSearch,omitempty"`
	EnableBrowser        bool                 `json:"enableBrowser,omitempty"`
	EnableA2AMaster      bool                 `json:"enableA2AMaster,omitempty"`
	Sandbox              SandboxConfig        `json:"sandbox"`
	AllowedWorkDirs      *[]string            `json:"allowedWorkDirs,omitempty"` // nil=no check, []=deny all overrides
	Session              SessionConfig        `json:"session"`
	WorkingDir           string               `json:"workingDir,omitempty"`
	CORS                 CORSConfig           `json:"cors"`
	Provider             string               `json:"provider,omitempty"`
	Model                string               `json:"model,omitempty"`
	ToolVisibility       ToolVisibilityConfig `json:"toolVisibility"`
	SystemPromptMode     string               `json:"systemPromptMode,omitempty"` // "append" (default), "ignore"
	RequestTimeoutSecs   int                  `json:"requestTimeoutSeconds,omitempty"`
	MaxConcurrentReqs    int                  `json:"maxConcurrentRequests,omitempty"`
	LogLevel             string               `json:"logLevel,omitempty"`
}

// AuthConfig controls bearer token authentication.
type AuthConfig struct {
	Enabled bool     `json:"enabled"`
	Tokens  []string `json:"tokens,omitempty"`
}

// SandboxConfig controls sandbox behavior for API requests.
type SandboxConfig struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level,omitempty"` // "none", "standard", "strict"; empty=auto from mode
}

// SessionConfig controls session pool behavior.
type SessionConfig struct {
	IdleTimeoutSeconds int `json:"idleTimeoutSeconds,omitempty"`
	MaxSessions        int `json:"maxSessions,omitempty"`
}

// CORSConfig controls cross-origin resource sharing.
type CORSConfig struct {
	Enabled      bool     `json:"enabled"`
	AllowOrigins []string `json:"allowOrigins,omitempty"`
}

// ToolVisibilityConfig controls how tool calls are exposed to the client.
type ToolVisibilityConfig struct {
	// Mode controls the transport for tool status:
	//   "content" (default) — tool output mixed into content stream
	//   "sse_event" — tool output via separate SSE events
	//   "none" — no tool output
	Mode string `json:"mode,omitempty"`

	// Detail controls the verbosity of tool output in content mode:
	//   "collapsed" (default) — one-line summary: 🔧 `read` main.go
	//                           edit always shows path + diff
	//   "expanded" — full output with code fences (Ctrl+O style)
	Detail string `json:"detail,omitempty"`
}

// DefaultConfig returns the default OpenAI-compatible API configuration.
func DefaultConfig() *Config {
	return &Config{
		Listen:               ":8080",
		Auth:                 AuthConfig{Enabled: false},
		DefaultMode:          "yolo",
		DefaultThinkingLevel: "medium",
		EnableSubAgents:      false,
		EnableDelegate:       false,
		EnableWorkflows:      false,
		Sandbox:              SandboxConfig{Enabled: false},
		Session:              SessionConfig{IdleTimeoutSeconds: 1800},
		CORS:                 CORSConfig{Enabled: false, AllowOrigins: []string{"*"}},
		ToolVisibility:       ToolVisibilityConfig{Mode: "content", Detail: "collapsed"},
		SystemPromptMode:     "append",
		RequestTimeoutSecs:   1800,
		LogLevel:             "info",
	}
}

func cloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Auth.Tokens = append([]string(nil), cfg.Auth.Tokens...)
	clone.CORS.AllowOrigins = append([]string(nil), cfg.CORS.AllowOrigins...)
	if cfg.AllowedWorkDirs != nil {
		allowed := append([]string(nil), (*cfg.AllowedWorkDirs)...)
		clone.AllowedWorkDirs = &allowed
	}
	return &clone
}

// normalizeConfig fills in defaults for empty fields.
func normalizeConfig(cfg *Config) {
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = "yolo"
	}
	if cfg.ToolVisibility.Mode == "" {
		cfg.ToolVisibility.Mode = "content"
	}
	if cfg.ToolVisibility.Detail == "" {
		cfg.ToolVisibility.Detail = "collapsed"
	}
	if cfg.SystemPromptMode == "" {
		cfg.SystemPromptMode = "append"
	}
	if cfg.RequestTimeoutSecs <= 0 {
		cfg.RequestTimeoutSecs = 1800
	}
}

// GetListenAddr returns the effective listen address.
func (c *Config) GetListenAddr() string {
	if c.Listen != "" {
		return c.Listen
	}
	return ":8080"
}

// GetWorkDir returns the effective working directory.
func (c *Config) GetWorkDir() string {
	if c.WorkingDir != "" {
		if strings.HasPrefix(c.WorkingDir, "~") {
			home, _ := os.UserHomeDir()
			if home != "" {
				return filepath.Join(home, c.WorkingDir[1:])
			}
		}
		return c.WorkingDir
	}
	cwd, _ := os.Getwd()
	return cwd
}

// GetToolDetail returns the effective tool detail level.
func (c *Config) GetToolDetail() string {
	if c.ToolVisibility.Detail != "" {
		return c.ToolVisibility.Detail
	}
	return "collapsed"
}

// ValidateWorkDir checks if the given directory is allowed by the allowedWorkDirs whitelist.
// Returns nil if allowed, an error describing the violation otherwise.
func (c *Config) ValidateWorkDir(dir string) error {
	// nil AllowedWorkDirs = no restriction
	if c.AllowedWorkDirs == nil {
		return nil
	}
	allowed := *c.AllowedWorkDirs
	// empty list = deny all overrides
	if len(allowed) == 0 {
		return fmt.Errorf("x_working_dir overrides are disabled")
	}

	cleanDir := filepath.Clean(dir)
	for _, a := range allowed {
		cleanAllowed := filepath.Clean(a)
		if cleanDir == cleanAllowed {
			return nil
		}
		// Prefix match with path separator boundary
		prefix := cleanAllowed + string(filepath.Separator)
		if strings.HasPrefix(cleanDir, prefix) {
			return nil
		}
	}
	return fmt.Errorf("directory %q is not in allowedWorkDirs", dir)
}
