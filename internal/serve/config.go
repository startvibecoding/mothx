package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/gateway"
	"github.com/startvibecoding/mothx/internal/hermes"
)

type Config struct {
	Gateway     gateway.GatewayConfig `json:"gateway"`
	Features    FeatureConfig         `json:"features"`
	Channels    ChannelConfig         `json:"channels"`
	WebUI       WebUIConfig           `json:"webUI"`
	LobsterMode bool                  `json:"lobsterMode,omitempty"`
	Cron        hermes.CronConfig     `json:"cron"`
	Memory      hermes.MemoryConfig   `json:"memory"`
	Security    hermes.SecurityConfig `json:"security"`
	Hooks       hermes.HooksConfig    `json:"hooks"`
	Agent       hermes.AgentConfig    `json:"agent"`
}

type FeatureConfig struct {
	WebUI      bool `json:"webUI,omitempty"`
	OpenAIAPI  bool `json:"openaiAPI,omitempty"`
	Wechat     bool `json:"wechat,omitempty"`
	Feishu     bool `json:"feishu,omitempty"`
	WebSocket  bool `json:"websocket,omitempty"`
	MultiAgent bool `json:"multiAgent,omitempty"`
	Cron       bool `json:"cron,omitempty"`
	Memory     bool `json:"memory,omitempty"`
}

type ChannelConfig struct {
	Wechat hermes.WechatConfig `json:"wechat"`
	Feishu hermes.FeishuConfig `json:"feishu"`
}

type WebUIConfig struct {
	Enabled bool   `json:"enabled"`
	Dir     string `json:"dir,omitempty"`
}

func DefaultConfig() *Config {
	gw := gateway.DefaultGatewayConfig()
	gw.Listen = ":8080"
	gw.DefaultMode = "yolo"
	h := hermes.DefaultHermesConfig()
	return &Config{
		Gateway: *gw,
		Features: FeatureConfig{
			WebUI:      true,
			OpenAIAPI:  true,
			Wechat:     false,
			Feishu:     false,
			WebSocket:  false,
			MultiAgent: gw.EnableSubAgents,
			Cron:       h.Cron.Enabled,
			Memory:     h.Memory.Enabled,
		},
		WebUI:    WebUIConfig{Enabled: true, Dir: "ui/dist"},
		Cron:     h.Cron,
		Memory:   h.Memory,
		Security: h.Security,
		Hooks:    h.Hooks,
		Agent:    h.Agent,
	}
}

func ConfigPath() string {
	return filepath.Join(config.ConfigDir(), "serve.json")
}

func ProjectConfigPath() string {
	return config.ProjectPath("serve.json")
}

func LoadConfig() (*Config, error) {
	cfg, err := LoadConfigFrom(ConfigPath())
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(ProjectConfigPath()); err == nil {
		if err := DecodeConfigBytesInto(cfg, data); err != nil {
			return nil, fmt.Errorf("parse project serve config %s: %w", ProjectConfigPath(), err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read project serve config %s: %w", ProjectConfigPath(), err)
	}
	normalize(cfg)
	return cfg, nil
}

func LoadConfigFrom(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read serve config %s: %w", path, err)
	}
	if err := DecodeConfigBytesInto(cfg, data); err != nil {
		return nil, fmt.Errorf("parse serve config %s: %w", path, err)
	}
	normalize(cfg)
	return cfg, nil
}

func normalize(cfg *Config) {
	if cfg.Gateway.Listen == "" {
		cfg.Gateway.Listen = ":8080"
	}
	if cfg.Gateway.DefaultMode == "" {
		cfg.Gateway.DefaultMode = "yolo"
	}
	if cfg.Gateway.ToolVisibility.Mode == "" {
		cfg.Gateway.ToolVisibility.Mode = "content"
	}
	if cfg.Gateway.ToolVisibility.Detail == "" {
		cfg.Gateway.ToolVisibility.Detail = "collapsed"
	}
	if cfg.Gateway.SystemPromptMode == "" {
		cfg.Gateway.SystemPromptMode = "append"
	}
	if cfg.Gateway.DefaultThinkingLevel == "" {
		cfg.Gateway.DefaultThinkingLevel = "medium"
	}
	if cfg.Gateway.RequestTimeoutSecs <= 0 {
		cfg.Gateway.RequestTimeoutSecs = 1800
	}
	if cfg.WebUI.Dir == "" {
		cfg.WebUI.Dir = "ui/dist"
	}
	if cfg.Agent.MaxTurns == 0 {
		cfg.Agent = hermes.DefaultHermesConfig().Agent
	}
	if cfg.LobsterMode {
		cfg.Gateway.DefaultMode = "yolo"
		cfg.Gateway.Sandbox.Enabled = false
		cfg.Gateway.EnableSubAgents = true
	}
	cfg.Features.WebUI = cfg.WebUI.Enabled
	cfg.Features.Wechat = cfg.Channels.Wechat.Enabled
	cfg.Features.Feishu = cfg.Channels.Feishu.Enabled
	cfg.Features.MultiAgent = cfg.Gateway.EnableSubAgents
	cfg.Features.Cron = cfg.Cron.Enabled
	cfg.Features.Memory = cfg.Memory.Enabled
}

func SaveConfig(path string, cfg *Config) error {
	normalize(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal serve config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func InitConfig(force bool) (string, error) {
	return InitConfigForProject(false, force)
}

func InitConfigForProject(project bool, force bool) (string, error) {
	path := ConfigPath()
	if project {
		path = ProjectConfigPath()
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return path, fmt.Errorf("serve.json already exists: %s", path)
		}
	}
	cfg := DefaultConfig()
	cfg.Gateway.Auth.Tokens = []string{"sk-change-me-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/home/user"
	}
	cfg.Gateway.WorkingDir = filepath.Join(home, "projects")
	allowed := []string{filepath.Join(home, "projects")}
	cfg.Gateway.AllowedWorkDirs = &allowed
	return path, SaveConfig(path, cfg)
}
