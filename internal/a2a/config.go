// Package a2a implements the A2A (Agent-to-Agent) protocol server.
// It provides a JSON-RPC 2.0 endpoint for other agents to send tasks to VibeCoding.
// Supports both standalone mode (vibecoding a2a start) and integration mode (hermes + a2a.enabled).
package a2a

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/startvibecoding/vibecoding/internal/config"
)

// Config holds A2A server configuration.
type Config struct {
	Enabled    bool          `json:"enabled"`
	Port       int           `json:"port"`
	Host       string        `json:"host"`
	AuthToken  string        `json:"auth_token,omitempty"`
	WorkDir    string        `json:"work_dir,omitempty"`
	AgentCard  *AgentCardCfg `json:"agent_card,omitempty"`
}

// AgentCardCfg holds customizable Agent Card fields.
type AgentCardCfg struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// DefaultConfig returns default A2A configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: false,
		Port:    8093,
		Host:    "0.0.0.0",
	}
}

// ConfigPath returns the path to the global a2a.json.
func ConfigPath() string {
	return filepath.Join(config.ConfigDir(), "a2a.json")
}

// ProjectConfigPath returns the path to the project-level .vibe/a2a.json.
func ProjectConfigPath() string {
	return filepath.Join(".vibe", "a2a.json")
}

// GetListenAddr returns the listen address.
func (c *Config) GetListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetWorkDir returns the resolved working directory.
func (c *Config) GetWorkDir() string {
	if c.WorkDir != "" && c.WorkDir != "." {
		return c.WorkDir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
