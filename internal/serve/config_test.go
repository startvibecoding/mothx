package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startvibecoding/mothx/internal/hermes"
)

func TestLoadConfigFrom_LegacyNestedSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.json")
	data := `{
		"gateway": {
			"listen": ":9090",
			"provider": "deepseek",
			"model": "deepseek-chat",
			"defaultMode": "agent",
			"sandbox": {"enabled": true, "level": "strict"},
			"enableSubAgents": true
		},
		"channels": {
			"wechat": {"enabled": true, "auto_typing": false},
			"feishu": {"enabled": true, "app_id": "app-id", "app_secret": "secret"}
		},
		"webUI": {"enabled": false, "dir": "custom-ui"},
		"memory": {"enabled": true},
		"agent": {"max_turns": 42}
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("write serve config: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}

	if cfg.Gateway.Listen != ":9090" {
		t.Fatalf("listen = %q, want :9090", cfg.Gateway.Listen)
	}
	if cfg.Gateway.Provider != "deepseek" {
		t.Fatalf("provider = %q, want deepseek", cfg.Gateway.Provider)
	}
	if cfg.Gateway.Model != "deepseek-chat" {
		t.Fatalf("model = %q, want deepseek-chat", cfg.Gateway.Model)
	}
	if cfg.Gateway.DefaultMode != "agent" {
		t.Fatalf("mode = %q, want agent", cfg.Gateway.DefaultMode)
	}
	if !cfg.Gateway.Sandbox.Enabled || cfg.Gateway.Sandbox.Level != "strict" {
		t.Fatalf("sandbox = %#v, want enabled strict", cfg.Gateway.Sandbox)
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("enableSubAgents should be true")
	}
	if !cfg.Channels.Wechat.Enabled || cfg.Channels.Wechat.AutoTyping {
		t.Fatalf("wechat config = %#v", cfg.Channels.Wechat)
	}
	if !cfg.Channels.Feishu.Enabled || cfg.Channels.Feishu.AppID != "app-id" || cfg.Channels.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu config = %#v", cfg.Channels.Feishu)
	}
	if cfg.WebUI.Enabled || cfg.WebUI.Dir != "custom-ui" {
		t.Fatalf("webUI config = %#v", cfg.WebUI)
	}
	if !cfg.Memory.Enabled {
		t.Fatal("memory should be enabled")
	}
	if cfg.Agent.MaxTurns != 42 {
		t.Fatalf("agent max turns = %d, want 42", cfg.Agent.MaxTurns)
	}
}

func TestLoadConfigFrom_FlatSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.json")
	data := `{
		"listen": ":7777",
		"provider": "openai",
		"model": "gpt-4o",
		"mode": "agent",
		"workDir": "/tmp/project",
		"thinking": "high",
		"auth": {
			"enabled": true,
			"tokens": ["sk-test"]
		},
		"features": {
			"webUI": true,
			"openaiAPI": true,
			"wechat": true,
			"feishu": true,
			"multiAgent": true,
			"memory": true
		},
		"sandbox": {
			"enabled": true,
			"level": "strict"
		},
		"allowedWorkDirs": ["/tmp/project"],
		"channels": {
			"wechat": {"autoTyping": false},
			"feishu": {"appId": "app-id", "appSecret": "secret"}
		},
		"session": {
			"idleTimeoutSeconds": 99,
			"maxSessions": 7
		},
		"toolVisibility": {"mode": "content", "detail": "expanded"},
		"systemPromptMode": "ignore",
		"requestTimeoutSeconds": 66,
		"maxConcurrentRequests": 5,
		"agent": {"maxTurns": 12}
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("write serve config: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}

	if cfg.Gateway.Listen != ":7777" {
		t.Fatalf("listen = %q, want :7777", cfg.Gateway.Listen)
	}
	if cfg.Gateway.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", cfg.Gateway.Provider)
	}
	if cfg.Gateway.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", cfg.Gateway.Model)
	}
	if cfg.Gateway.DefaultMode != "agent" {
		t.Fatalf("mode = %q, want agent", cfg.Gateway.DefaultMode)
	}
	if cfg.Gateway.WorkingDir != "/tmp/project" {
		t.Fatalf("workDir = %q, want /tmp/project", cfg.Gateway.WorkingDir)
	}
	if !cfg.Gateway.Auth.Enabled || len(cfg.Gateway.Auth.Tokens) != 1 || cfg.Gateway.Auth.Tokens[0] != "sk-test" {
		t.Fatalf("auth = %#v", cfg.Gateway.Auth)
	}
	if !cfg.Gateway.Sandbox.Enabled || cfg.Gateway.Sandbox.Level != "strict" {
		t.Fatalf("sandbox = %#v", cfg.Gateway.Sandbox)
	}
	if cfg.Gateway.AllowedWorkDirs == nil || len(*cfg.Gateway.AllowedWorkDirs) != 1 || (*cfg.Gateway.AllowedWorkDirs)[0] != "/tmp/project" {
		t.Fatalf("allowedWorkDirs = %#v", cfg.Gateway.AllowedWorkDirs)
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("enableSubAgents should be true")
	}
	if !cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be true")
	}
	if cfg.Gateway.Session.IdleTimeoutSeconds != 99 || cfg.Gateway.Session.MaxSessions != 7 {
		t.Fatalf("session config = %#v", cfg.Gateway.Session)
	}
	if cfg.Gateway.ToolVisibility.Mode != "content" || cfg.Gateway.ToolVisibility.Detail != "expanded" {
		t.Fatalf("tool visibility = %#v", cfg.Gateway.ToolVisibility)
	}
	if cfg.Gateway.SystemPromptMode != "ignore" {
		t.Fatalf("system prompt mode = %q, want ignore", cfg.Gateway.SystemPromptMode)
	}
	if cfg.Gateway.RequestTimeoutSecs != 66 {
		t.Fatalf("timeout = %d, want 66", cfg.Gateway.RequestTimeoutSecs)
	}
	if cfg.Gateway.MaxConcurrentReqs != 5 {
		t.Fatalf("max concurrent = %d, want 5", cfg.Gateway.MaxConcurrentReqs)
	}
	if !cfg.Channels.Wechat.Enabled || cfg.Channels.Wechat.AutoTyping {
		t.Fatalf("wechat config = %#v", cfg.Channels.Wechat)
	}
	if !cfg.Channels.Feishu.Enabled || cfg.Channels.Feishu.AppID != "app-id" || cfg.Channels.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu config = %#v", cfg.Channels.Feishu)
	}
	if !cfg.WebUI.Enabled {
		t.Fatal("webUI should be enabled")
	}
	if !cfg.Memory.Enabled {
		t.Fatal("memory should be enabled")
	}
	if cfg.Agent.MaxTurns != 12 {
		t.Fatalf("agent max turns = %d, want 12", cfg.Agent.MaxTurns)
	}
}

func TestDecodeConfigBytes_FlatSchema(t *testing.T) {
	data := []byte(`{
		"listen": ":8081",
		"features": {
			"webUI": false,
			"openaiAPI": false
		},
		"channels": {
			"wechat": {"autoTyping": false}
		}
	}`)

	cfg, err := DecodeConfigBytes(data)
	if err != nil {
		t.Fatalf("DecodeConfigBytes: %v", err)
	}
	if cfg.Gateway.Listen != ":8081" {
		t.Fatalf("listen = %q, want :8081", cfg.Gateway.Listen)
	}
	if cfg.Features.WebUI {
		t.Fatal("webUI feature should be false")
	}
	if cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be false")
	}
	if cfg.Channels.Wechat.AutoTyping {
		t.Fatal("wechat autoTyping should be false")
	}
}

func TestInitConfig_WritesFlatTemplate(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("MOTHX_CONFIG_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	path, err := InitConfig(false)
	if err != nil {
		t.Fatalf("InitConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"listen": ":8080"`) {
		t.Fatalf("generated config missing flat listen field:\n%s", text)
	}
	if !strings.Contains(text, `"features": {`) {
		t.Fatalf("generated config missing features block:\n%s", text)
	}
	if !strings.Contains(text, `"openaiAPI": true`) {
		t.Fatalf("generated config missing openaiAPI feature flag:\n%s", text)
	}
	if !strings.Contains(text, `"auth": {`) {
		t.Fatalf("generated config missing auth block:\n%s", text)
	}
	if strings.Contains(text, `"gateway": {`) {
		t.Fatalf("generated config should prefer flat schema, got legacy gateway block:\n%s", text)
	}
}

func TestHandleChannels_ReturnsStableEntries(t *testing.T) {
	rt := &channelRuntime{
		cfg: &Config{
			Channels: ChannelConfig{
				Wechat: hermes.WechatConfig{Enabled: true},
				Feishu: hermes.FeishuConfig{Enabled: false},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()
	rt.handleChannels(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var statuses []channelStatus
	if err := json.NewDecoder(w.Body).Decode(&statuses); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("status count = %d, want 2", len(statuses))
	}
	if statuses[0] != (channelStatus{Name: "wechat", Enabled: true, Connected: false}) {
		t.Fatalf("wechat status = %#v", statuses[0])
	}
	if statuses[1] != (channelStatus{Name: "feishu", Enabled: false, Connected: false}) {
		t.Fatalf("feishu status = %#v", statuses[1])
	}
}

func TestRegisterServeRoutes_WebUIDisabledStillServesManageAPI(t *testing.T) {
	rt := &channelRuntime{
		cfg: &Config{
			Features: FeatureConfig{WebUI: false, OpenAIAPI: true},
			WebUI:    WebUIConfig{Enabled: false, Dir: "ui/dist"},
		},
	}

	mux := http.NewServeMux()
	registerServeRoutes(mux, rt, "/tmp/serve.json")

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/api/channels status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/ status = %d, want 404 when webUI disabled", w.Code)
	}
}

func TestHandleServeConfig_PutAcceptsFlatSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.json")
	rt := &channelRuntime{cfg: DefaultConfig()}

	body := `{
		"listen": ":9099",
		"features": {
			"webUI": false,
			"openaiAPI": false,
			"wechat": true
		},
		"channels": {
			"wechat": {"autoTyping": false}
		}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/serve/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	rt.handleServeConfig(path).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.cfg.Gateway.Listen != ":9099" {
		t.Fatalf("listen = %q, want :9099", rt.cfg.Gateway.Listen)
	}
	if rt.cfg.Features.WebUI {
		t.Fatal("webUI feature should be false after PUT")
	}
	if rt.cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be false after PUT")
	}
	if !rt.cfg.Channels.Wechat.Enabled {
		t.Fatal("wechat should be enabled after PUT")
	}
	if rt.cfg.Channels.Wechat.AutoTyping {
		t.Fatal("wechat autoTyping should be false after PUT")
	}

	saved, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom(saved): %v", err)
	}
	if saved.Gateway.Listen != ":9099" || saved.Features.OpenAIAPI {
		t.Fatalf("saved config = %#v", saved)
	}
}

func TestApplyRuntimeFeaturesOverridesLegacyRuntimeFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.Wechat = false
	cfg.Features.Feishu = false
	cfg.Features.Cron = false
	cfg.Features.Memory = false
	cfg.Features.MultiAgent = false
	cfg.Features.WebUI = false
	cfg.Channels.Wechat.Enabled = true
	cfg.Channels.Feishu.Enabled = true
	cfg.Cron.Enabled = true
	cfg.Memory.Enabled = true
	cfg.Gateway.EnableSubAgents = true
	cfg.WebUI.Enabled = true

	applyRuntimeFeatures(cfg)

	if cfg.Channels.Wechat.Enabled {
		t.Fatal("wechat should be disabled by features")
	}
	if cfg.Channels.Feishu.Enabled {
		t.Fatal("feishu should be disabled by features")
	}
	if cfg.Cron.Enabled {
		t.Fatal("cron should be disabled by features")
	}
	if cfg.Memory.Enabled {
		t.Fatal("memory should be disabled by features")
	}
	if cfg.Gateway.EnableSubAgents {
		t.Fatal("multi-agent should be disabled by features")
	}
	if cfg.WebUI.Enabled {
		t.Fatal("webUI should be disabled by features")
	}
}

func TestApplyOverridesPreservesMultiAgentFlagThroughFeatureSync(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.MultiAgent = false
	cfg.Gateway.EnableSubAgents = false

	applyOverrides(cfg, RunOptions{MultiAgent: true})
	applyRuntimeFeatures(cfg)

	if !cfg.Features.MultiAgent {
		t.Fatal("multi-agent feature should be enabled by CLI override")
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("gateway subagents should remain enabled after feature sync")
	}
}

func TestBuildHermesConfigFromServeConfigAppliesFeatureGating(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Gateway.Provider = "openai"
	cfg.Gateway.Model = "gpt-4o"
	cfg.Gateway.WorkingDir = "/tmp/project"
	cfg.Gateway.Sandbox.Enabled = true
	cfg.Features.MultiAgent = true
	cfg.Features.Wechat = false
	cfg.Features.Feishu = true
	cfg.Features.Cron = false
	cfg.Features.Memory = true
	cfg.Channels.Wechat.Enabled = true
	cfg.Channels.Wechat.AutoTyping = false
	cfg.Channels.Feishu.Enabled = false
	cfg.Channels.Feishu.AppID = "app-id"
	cfg.Channels.Feishu.AppSecret = "secret"
	cfg.Cron.Enabled = true
	cfg.Cron.StorePath = "/tmp/cron.json"
	cfg.Memory.Enabled = false
	cfg.Memory.Path = "/tmp/memory.md"
	cfg.Agent.MaxTurns = 33

	hCfg := buildHermesConfigFromServeConfig(cfg)

	if hCfg.Server.Host != "127.0.0.1" || hCfg.Server.Port != 0 {
		t.Fatalf("server = %#v", hCfg.Server)
	}
	if hCfg.DefaultProvider != "openai" || hCfg.DefaultModel != "gpt-4o" {
		t.Fatalf("provider/model = %q/%q", hCfg.DefaultProvider, hCfg.DefaultModel)
	}
	if hCfg.WorkDir != "/tmp/project" {
		t.Fatalf("workDir = %q, want /tmp/project", hCfg.WorkDir)
	}
	if !hCfg.Sandbox {
		t.Fatal("sandbox should be enabled")
	}
	if !hCfg.MultiAgent {
		t.Fatal("multi-agent should be enabled")
	}
	if hCfg.Wechat.Enabled {
		t.Fatal("wechat should be disabled by features")
	}
	if !hCfg.Feishu.Enabled || hCfg.Feishu.AppID != "app-id" || hCfg.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu = %#v", hCfg.Feishu)
	}
	if hCfg.Cron.Enabled {
		t.Fatal("cron should be disabled by features")
	}
	if !hCfg.Memory.Enabled || hCfg.Memory.Path != "/tmp/memory.md" {
		t.Fatalf("memory = %#v", hCfg.Memory)
	}
	if hCfg.Agent.MaxTurns != 33 {
		t.Fatalf("agent max turns = %d, want 33", hCfg.Agent.MaxTurns)
	}
}

func TestBuildCronStoreHonorsCronFeature(t *testing.T) {
	disabled := buildCronStore(&hermes.HermesConfig{Cron: hermes.CronConfig{Enabled: false}})
	if disabled != nil {
		t.Fatal("cron store should be nil when cron is disabled")
	}

	enabled := buildCronStore(&hermes.HermesConfig{Cron: hermes.CronConfig{Enabled: true, StorePath: filepath.Join(t.TempDir(), "cron.json")}})
	if enabled == nil {
		t.Fatal("cron store should be created when cron is enabled")
	}
}
