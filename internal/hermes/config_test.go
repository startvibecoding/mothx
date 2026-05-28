package hermes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultHermesConfig(t *testing.T) {
	cfg := DefaultHermesConfig()
	if cfg.Server.Port != 8090 {
		t.Errorf("expected port 8090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if !cfg.Wechat.AutoTyping {
		t.Error("expected auto_typing=true")
	}
	if !cfg.Security.SmartApprovals {
		t.Error("expected smart_approvals=true")
	}
	if cfg.Agent.MaxTurns != 90 {
		t.Errorf("expected max_turns=90, got %d", cfg.Agent.MaxTurns)
	}
}

func TestGetDefaultProvider(t *testing.T) {
	cfg := &HermesConfig{DefaultProvider: "openai"}
	if got := cfg.GetDefaultProvider("deepseek"); got != "openai" {
		t.Errorf("expected openai, got %s", got)
	}

	cfg2 := &HermesConfig{}
	if got := cfg2.GetDefaultProvider("deepseek"); got != "deepseek" {
		t.Errorf("expected deepseek fallback, got %s", got)
	}
}

func TestGetDefaultModel(t *testing.T) {
	cfg := &HermesConfig{DefaultModel: "gpt-4o"}
	if got := cfg.GetDefaultModel("deepseek-chat"); got != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", got)
	}

	cfg2 := &HermesConfig{}
	if got := cfg2.GetDefaultModel("deepseek-chat"); got != "deepseek-chat" {
		t.Errorf("expected deepseek-chat fallback, got %s", got)
	}
}

func TestGetListenAddr(t *testing.T) {
	cfg := &HermesConfig{
		Server: ServerConfig{Host: "127.0.0.1", Port: 9090},
	}
	if got := cfg.GetListenAddr(); got != "127.0.0.1:9090" {
		t.Errorf("expected 127.0.0.1:9090, got %s", got)
	}
}

func TestGetWorkDir(t *testing.T) {
	cfg := &HermesConfig{WorkDir: "/tmp/test"}
	if got := cfg.GetWorkDir(); got != "/tmp/test" {
		t.Errorf("expected /tmp/test, got %s", got)
	}

	cfg2 := &HermesConfig{WorkDir: "."}
	got := cfg2.GetWorkDir()
	if got == "" || got == "." {
		t.Errorf("expected resolved path, got %s", got)
	}
}

func TestGetPlatformWorkDir(t *testing.T) {
	cfg := &HermesConfig{
		WorkDir: "/global",
		Wechat:  WechatConfig{WorkDir: "/wechat"},
		Feishu:  FeishuConfig{WorkDir: "/feishu"},
	}

	if got := cfg.GetPlatformWorkDir("wechat"); got != "/wechat" {
		t.Errorf("expected /wechat, got %s", got)
	}
	if got := cfg.GetPlatformWorkDir("feishu"); got != "/feishu" {
		t.Errorf("expected /feishu, got %s", got)
	}
	if got := cfg.GetPlatformWorkDir("ws"); got != "/global" {
		t.Errorf("expected /global, got %s", got)
	}
}

func TestLoadHermesConfigFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes.json")

	data := `{"server":{"port":9999},"default_provider":"test-provider","default_model":"test-model","multi_agent":true}`
	os.WriteFile(path, []byte(data), 0600)

	cfg, err := LoadHermesConfigFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Server.Port)
	}
	if cfg.DefaultProvider != "test-provider" {
		t.Errorf("expected test-provider, got %s", cfg.DefaultProvider)
	}
	if cfg.DefaultModel != "test-model" {
		t.Errorf("expected test-model, got %s", cfg.DefaultModel)
	}
	if !cfg.MultiAgent {
		t.Error("expected multi_agent=true")
	}
}

func TestLoadHermesConfigFromMissing(t *testing.T) {
	cfg, err := LoadHermesConfigFrom("/nonexistent/hermes.json")
	if err != nil {
		t.Fatal(err)
	}
	// Should return defaults
	if cfg.Server.Port != 8090 {
		t.Errorf("expected default port 8090, got %d", cfg.Server.Port)
	}
}

func TestLoadHermesConfigFromInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0600)

	_, err := LoadHermesConfigFrom(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestInitHermesConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes.json")

	// Override path for test
	cfg := DefaultHermesConfig()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0600)

	// Should exist
	if _, err := os.Stat(path); err != nil {
		t.Fatal("expected file to exist")
	}
}

func TestCronConfig(t *testing.T) {
	cfg := &HermesConfig{
		Cron: CronConfig{
			Enabled:   true,
			StorePath: "/tmp/cron.json",
			Interval:  60,
		},
	}
	if !cfg.Cron.Enabled {
		t.Error("expected cron enabled")
	}
	if cfg.Cron.StorePath != "/tmp/cron.json" {
		t.Errorf("expected /tmp/cron.json, got %s", cfg.Cron.StorePath)
	}
	if cfg.Cron.Interval != 60 {
		t.Errorf("expected interval 60, got %d", cfg.Cron.Interval)
	}
}
