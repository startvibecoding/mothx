package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	if s.DefaultProvider != "deepseek-openai" {
		t.Errorf("expected default provider 'deepseek-openai', got '%s'", s.DefaultProvider)
	}

	if s.DefaultModel != "deepseek-v4-flash" {
		t.Errorf("expected default model 'deepseek-v4-flash', got '%s'", s.DefaultModel)
	}

	if s.DefaultMode != "agent" {
		t.Errorf("expected default mode 'agent', got '%s'", s.DefaultMode)
	}

	if len(s.Providers) < 35 {
		t.Errorf("expected at least 35 providers, got %d", len(s.Providers))
	}

	if s.Providers["openai"] == nil {
		t.Fatal("expected default openai provider")
	}
	if s.Providers["anthropic"] == nil {
		t.Fatal("expected default anthropic provider")
	}
	if s.Providers["xiaomi"] == nil {
		t.Fatal("expected default xiaomi provider")
	}
	if s.Providers["google-gemini"] == nil {
		t.Fatal("expected default google-gemini provider")
	}
	if s.Providers["google-vertex"] == nil {
		t.Fatal("expected default google-vertex provider")
	}

	for _, name := range []string{"openrouter", "minimax", "zai", "modelscope", "alibaba-standard", "alibaba-coding-plan", "alibaba-token-plan", "moark", "groq", "moonshotai", "xai", "together", "fireworks", "kimi-coding", "xiaomi-token-plan-cn"} {
		if s.Providers[name] == nil {
			t.Fatalf("expected default %s provider", name)
		}
	}
	if got := s.Providers["kimi-coding"].Headers["User-Agent"]; got != "opencode/1.17.18" {
		t.Fatalf("kimi-coding User-Agent = %q, want %q", got, "opencode/1.17.18")
	}
	for _, name := range []string{"openai", "codeok", "yescode"} {
		if got := s.Providers[name].Headers["User-Agent"]; got != "codex_cli_rs/0.144.4" {
			t.Fatalf("%s User-Agent = %q, want %q", name, got, "codex_cli_rs/0.144.4")
		}
	}
	kimiCoding := s.Providers["kimi-coding"]
	if kimiCoding.BaseURL != "https://api.kimi.com/coding/v1" || kimiCoding.API != "openai-chat" {
		t.Fatalf("kimi-coding endpoint = (%q, %q), want (%q, %q)", kimiCoding.BaseURL, kimiCoding.API, "https://api.kimi.com/coding/v1", "openai-chat")
	}
	for _, model := range kimiCoding.Models {
		if model.ID == "k2p7" {
			t.Fatal("kimi-coding must not include k2p7")
		}
	}

	if s.DefaultThinkingLevel != "medium" {
		t.Errorf("expected thinking level 'medium', got '%s'", s.DefaultThinkingLevel)
	}
	if s.StatusLine.Enabled {
		t.Fatalf("expected statusLine disabled by default")
	}
	if s.StatusLine.Type != "command" {
		t.Fatalf("expected statusLine.type command, got %q", s.StatusLine.Type)
	}
	if s.StatusLine.TimeoutMs != 800 {
		t.Fatalf("expected statusLine.timeoutMs 800, got %d", s.StatusLine.TimeoutMs)
	}
	if s.StatusLine.Fallback != "builtin" {
		t.Fatalf("expected statusLine.fallback builtin, got %q", s.StatusLine.Fallback)
	}
	if s.WebSearch.Enabled == nil || *s.WebSearch.Enabled {
		t.Fatalf("expected web search to be disabled by default, got %#v", s.WebSearch.Enabled)
	}
	if s.WebSearch.Provider != "openai" || s.WebSearch.ProviderType != "responses" {
		t.Fatalf("unexpected web search defaults: %#v", s.WebSearch)
	}
	if !s.Retry.Enabled || s.Retry.MaxRetries != 5 || s.Retry.BaseDelayMs != 3000 {
		t.Fatalf("unexpected retry defaults: %#v", s.Retry)
	}
	if s.WebSearch.Model != "" {
		t.Fatalf("expected empty web search model by default, got %q", s.WebSearch.Model)
	}
}

func TestGetProviderConfig(t *testing.T) {
	s := DefaultSettings()

	// Test existing provider (openai format)
	pc := s.GetProviderConfig("deepseek-openai")
	if pc == nil {
		t.Fatal("expected provider config, got nil")
	}

	if pc.API != "openai-chat" {
		t.Errorf("expected API 'openai-chat', got '%s'", pc.API)
	}

	// Test non-existing provider
	pc = s.GetProviderConfig("nonexistent")
	if pc != nil {
		t.Errorf("expected nil, got provider config")
	}
}

func TestGetModelConfig(t *testing.T) {
	s := DefaultSettings()

	// Test existing model
	mc := s.GetModelConfig("deepseek-openai", "deepseek-v4-flash")
	if mc == nil {
		t.Fatal("expected model config, got nil")
	}

	if mc.Name != "DeepSeek V4 Flash" {
		t.Errorf("expected name 'DeepSeek V4 Flash', got '%s'", mc.Name)
	}

	// Test non-existing model
	mc = s.GetModelConfig("deepseek-openai", "nonexistent")
	if mc != nil {
		t.Errorf("expected nil, got model config")
	}

	// Test non-existing provider
	mc = s.GetModelConfig("nonexistent", "model")
	if mc != nil {
		t.Errorf("expected nil, got model config")
	}
}

func TestMoarkModelMaxTokens(t *testing.T) {
	s := DefaultSettings()
	want := map[string]int{
		"glm-5.1":            131072,
		"qwen3.5-flash":      65536,
		"qwen3.6-flash":      65536,
		"qwen3.6-max":        65536,
		"qwen3.6-plus":       65536,
		"deepseek-v4-pro":    384000,
		"qwen3.7-max":        65536,
		"glm-5.2":            131072,
		"kimi-k2.5":          262144,
		"kimi-k2.7-code":     262144,
		"glm-5":              32768,
		"qwen3.7-plus":       65536,
		"minimax-m2.7":       131072,
		"minimax-m3":         128000,
		"mimo-v2.5-pro":      131072,
		"gemma-4-26b-a4b-it": 32768,
		"deepseek-v4-flash":  384000,
		"step-3.7-flash":     16384,
	}

	moark := s.Providers["moark"]
	if moark == nil {
		t.Fatal("expected moark provider")
	}
	if len(moark.Models) != len(want) {
		t.Fatalf("moark models = %d, want %d", len(moark.Models), len(want))
	}
	for _, model := range moark.Models {
		wantMaxTokens, ok := want[model.ID]
		if !ok {
			t.Fatalf("unexpected moark model %q", model.ID)
		}
		if model.MaxTokens != wantMaxTokens {
			t.Fatalf("moark %s MaxTokens = %d, want %d", model.ID, model.MaxTokens, wantMaxTokens)
		}
	}
}

func TestVolcenginePlanModelsUseSharedMaxTokens(t *testing.T) {
	s := DefaultSettings()
	for _, providerName := range []string{"volcengine-agentplan", "volcengine-codingplan"} {
		p := s.Providers[providerName]
		if p == nil {
			t.Fatalf("expected %s provider", providerName)
		}
		for _, model := range p.Models {
			if model.MaxTokens != 100000 {
				t.Fatalf("%s %s MaxTokens = %d, want 100000", providerName, model.ID, model.MaxTokens)
			}
		}
	}
}

func TestRoutedProviderModelMaxTokensAreExplicit(t *testing.T) {
	s := DefaultSettings()
	wantByProvider := map[string]map[string]int{
		"minimax": {
			"MiniMax-M3":             128000,
			"MiniMax-M2.7":           131072,
			"MiniMax-M2.7-highspeed": 131072,
			"MiniMax-M2.5":           131072,
			"MiniMax-M2.5-highspeed": 131072,
		},
		"modelscope": {
			"deepseek-ai/DeepSeek-V4-Flash": 384000,
			"Qwen/Qwen3.5-397B-A17B":        130000,
			"ZhipuAI/GLM-5.1":               131072,
		},
		"gitee": {
			"glm-5.1":           131072,
			"qwen3.6-max":       65536,
			"qwen3.6-plus":      65536,
			"deepseek-v4-pro":   384000,
			"qwen3.7-max":       65536,
			"glm-5.2":           131072,
			"kimi-k2.7-code":    262144,
			"glm-5":             32768,
			"qwen3.7-plus":      65536,
			"minimax-m2.7":      131072,
			"minimax-m3":        128000,
			"deepseek-v4-flash": 384000,
		},
		"alibaba-standard": {
			"qwen3.6-plus":      65536,
			"qwen3.7-plus":      65536,
			"qwen3.7-max":       65536,
			"glm-5.1":           131072,
			"deepseek-v4-pro":   384000,
			"deepseek-v4-flash": 384000,
		},
		"alibaba-coding-plan": {
			"qwen3.5-plus":         65536,
			"qwen3.6-plus":         65536,
			"qwen3.7-plus":         65536,
			"glm-5":                32768,
			"kimi-k2.5":            262144,
			"MiniMax-M2.5":         131072,
			"qwen3-coder-plus":     65536,
			"qwen3-coder-next":     65536,
			"qwen3-max-2026-01-23": 65536,
			"glm-4.7":              131072,
		},
		"alibaba-token-plan": {
			"qwen3.6-plus":      65536,
			"qwen3.7-max":       65536,
			"qwen3.6-flash":     65536,
			"deepseek-v4-pro":   384000,
			"deepseek-v4-flash": 384000,
			"deepseek-v3.2":     65536,
			"kimi-k2.6":         262144,
			"kimi-k2.5":         262144,
			"glm-5.1":           131072,
			"glm-5":             32768,
			"MiniMax-M2.5":      131072,
		},
	}

	for providerName, wantModels := range wantByProvider {
		provider := s.Providers[providerName]
		if provider == nil {
			t.Fatalf("expected %s provider", providerName)
		}
		gotModels := map[string]int{}
		for _, model := range provider.Models {
			gotModels[model.ID] = model.MaxTokens
		}
		for modelID, wantMaxTokens := range wantModels {
			got, ok := gotModels[modelID]
			if !ok {
				t.Fatalf("%s missing model %q", providerName, modelID)
			}
			if got != wantMaxTokens {
				t.Fatalf("%s %s MaxTokens = %d, want %d", providerName, modelID, got, wantMaxTokens)
			}
			if got == 8192 {
				t.Fatalf("%s %s still uses placeholder MaxTokens 8192", providerName, modelID)
			}
		}
	}
}

func TestResolveConfigJSONExplicitZeroValuesOverrideDefaults(t *testing.T) {
	var runtime Settings
	data := []byte(`{
		"providers": {
			"longcat": {
				"vendor": "",
				"baseUrl": "",
				"headers": {},
				"models": [
					{
						"id": "LongCat-2.0",
						"reasoning": false,
						"input": [],
						"cost": null
					}
				]
			}
		}
	}`)
	if err := json.Unmarshal(data, &runtime); err != nil {
		t.Fatalf("unmarshal runtime: %v", err)
	}

	pc := ResolveProviderConfig("longcat", &runtime)
	if pc.Vendor != "" {
		t.Fatalf("Vendor = %q, want explicit empty override", pc.Vendor)
	}
	if pc.BaseURL != "" {
		t.Fatalf("BaseURL = %q, want explicit empty override", pc.BaseURL)
	}
	if pc.Headers == nil || len(pc.Headers) != 0 {
		t.Fatalf("Headers = %#v, want explicit empty map", pc.Headers)
	}
	if len(pc.Models) != 1 {
		t.Fatalf("Models = %d, want explicit one-model override", len(pc.Models))
	}

	mc := ResolveModelConfig("longcat", "LongCat-2.0", &runtime)
	if mc == nil {
		t.Fatal("expected model config")
	}
	if mc.Reasoning {
		t.Fatal("Reasoning = true, want explicit false override")
	}
	if mc.Input == nil || len(mc.Input) != 0 {
		t.Fatalf("Input = %#v, want explicit empty slice", mc.Input)
	}
	if mc.Cost != nil {
		t.Fatalf("Cost = %#v, want explicit null override", mc.Cost)
	}
	if mc.ContextWindow == 0 {
		t.Fatal("ContextWindow lost builtin default")
	}
}

func TestResolveProviderConfigPreservesFieldPresenceAcrossGlobalAndProject(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	configDir := filepath.Join(tmpDir, "config")
	t.Setenv("VIBECODING_DIR", configDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(GlobalSettingsPath(), []byte(`{
		"providers": {
			"longcat": {"apiKey": "global-key"}
		}
	}`), 0600); err != nil {
		t.Fatalf("write global settings: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(ProjectSettingsPath()), 0700); err != nil {
		t.Fatalf("mkdir project config dir: %v", err)
	}
	if err := os.WriteFile(ProjectSettingsPath(), []byte(`{
		"providers": {
			"longcat": {"baseUrl": "https://project.longcat.test"}
		}
	}`), 0600); err != nil {
		t.Fatalf("write project settings: %v", err)
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	pc := ResolveProviderConfig("longcat", s)
	if pc.APIKey != "global-key" {
		t.Fatalf("APIKey = %q, want global-key", pc.APIKey)
	}
	if pc.BaseURL != "https://project.longcat.test" {
		t.Fatalf("BaseURL = %q, want project override", pc.BaseURL)
	}
}

func TestConfigDir(t *testing.T) {
	// Test with env var
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", "/tmp/test-vibecoding")
	dir := ConfigDir()
	if dir != "/tmp/test-vibecoding" {
		t.Errorf("expected '/tmp/test-vibecoding', got '%s'", dir)
	}
	t.Setenv("VIBECODING_DIR", "")

	// Test default
	dir = ConfigDir()
	if dir == "" {
		t.Error("expected non-empty config dir")
	}
}

func TestLoadSettingsCreatesMothXConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	})

	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", "")
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	if _, _, err := LoadSettingsWithMeta(); err != nil {
		t.Fatalf("load settings: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, ".mothx", "settings.json")); err != nil {
		t.Fatalf("expected .mothx settings file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".vibecoding")); !os.IsNotExist(err) {
		t.Fatalf("expected no .vibecoding directory, stat err=%v", err)
	}
}

func TestGlobalSettingsPath(t *testing.T) {
	path := GlobalSettingsPath()
	if path == "" {
		t.Error("expected non-empty path")
	}

	if !contains(path, "settings.json") {
		t.Error("expected path to contain 'settings.json'")
	}
}

func TestProjectSettingsPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	path := ProjectSettingsPath()
	if path != filepath.Join(ProjectDirName, "settings.json") {
		t.Errorf("expected '%s/settings.json', got '%s'", ProjectDirName, path)
	}
}

func TestLoadSettings(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Write test settings
	settingsJSON := `{
		"providers": {
			"test": {
				"baseUrl": "https://api.test.com",
				"apiKey": "test-key",
				"api": "openai-chat",
				"models": [
					{
						"id": "test-model",
						"name": "Test Model",
						"contextWindow": 100000,
						"maxTokens": 4096
					}
				]
			}
		},
		"defaultProvider": "test",
		"defaultModel": "test-model"
	}`

	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Load settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	s := DefaultSettings()
	if err := json.Unmarshal(data, s); err != nil {
		t.Fatal(err)
	}

	if s.DefaultProvider != "test" {
		t.Errorf("expected provider 'test', got '%s'", s.DefaultProvider)
	}
	if s.WebSearch.Model != "" {
		t.Errorf("expected empty webSearch.model, got '%s'", s.WebSearch.Model)
	}
}

func TestLoadSettingsAppliesProjectOverridesAndEnv(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	configDir := filepath.Join(tmpDir, "config")
	if err := os.Setenv("VIBECODING_DIR", configDir); err != nil {
		t.Fatalf("set VIBECODING_DIR: %v", err)
	}
	if err := os.Setenv("VIBECODING_PROVIDER", "env-provider"); err != nil {
		t.Fatalf("set VIBECODING_PROVIDER: %v", err)
	}
	if err := os.Setenv("VIBECODING_MODEL", "env-model"); err != nil {
		t.Fatalf("set VIBECODING_MODEL: %v", err)
	}
	if err := os.Setenv("VIBECODING_MODE", "plan"); err != nil {
		t.Fatalf("set VIBECODING_MODE: %v", err)
	}
	if err := os.Setenv("VIBECODING_THINKING", "high"); err != nil {
		t.Fatalf("set VIBECODING_THINKING: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("VIBECODING_DIR")
		_ = os.Unsetenv("VIBECODING_PROVIDER")
		_ = os.Unsetenv("VIBECODING_MODEL")
		_ = os.Unsetenv("VIBECODING_MODE")
		_ = os.Unsetenv("VIBECODING_THINKING")
	}()

	if err := os.MkdirAll(filepath.Dir(ProjectSettingsPath()), 0700); err != nil {
		t.Fatalf("mkdir project config dir: %v", err)
	}
	projectSettings := `{
		"sessionDir": "./sessions",
		"providers": {
			"project-provider": {
				"baseUrl": "https://example.test",
				"api": "openai-chat",
				"models": [{"id": "project-model", "name": "Project Model"}]
			}
		},
		"contextFiles": {"enabled": false, "extraFiles": ["extra.md"]},
		"approval": {"bashWhitelist": ["go test "]}
	}`
	if err := os.WriteFile(ProjectSettingsPath(), []byte(projectSettings), 0600); err != nil {
		t.Fatalf("write project settings: %v", err)
	}

	s, err := LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}

	if s.DefaultProvider != "env-provider" {
		t.Fatalf("DefaultProvider = %q, want env-provider", s.DefaultProvider)
	}
	if s.DefaultModel != "env-model" {
		t.Fatalf("DefaultModel = %q, want env-model", s.DefaultModel)
	}
	if s.DefaultMode != "plan" {
		t.Fatalf("DefaultMode = %q, want plan", s.DefaultMode)
	}
	if s.DefaultThinkingLevel != "high" {
		t.Fatalf("DefaultThinkingLevel = %q, want high", s.DefaultThinkingLevel)
	}
	if s.SessionDir != "./sessions" {
		t.Fatalf("SessionDir = %q, want ./sessions", s.SessionDir)
	}
	if s.GetProviderConfig("project-provider") == nil {
		t.Fatal("expected merged project provider")
	}
	if s.GetProviderConfig("deepseek-openai") == nil {
		t.Fatal("expected default provider to remain after project merge")
	}
	if s.ContextFiles.Enabled {
		t.Fatal("expected project contextFiles override to disable context files")
	}
	if len(s.ContextFiles.ExtraFiles) != 1 || s.ContextFiles.ExtraFiles[0] != "extra.md" {
		t.Fatalf("ExtraFiles = %#v, want extra.md", s.ContextFiles.ExtraFiles)
	}
	if len(s.Approval.BashWhitelist) != 1 || s.Approval.BashWhitelist[0] != "go test " {
		t.Fatalf("BashWhitelist = %#v, want go test", s.Approval.BashWhitelist)
	}
}

func TestDefaultSettingsConfirmBeforeWrite(t *testing.T) {
	s := DefaultSettings()
	if s.Approval.ConfirmBeforeWrite == nil || !*s.Approval.ConfirmBeforeWrite {
		t.Fatal("expected confirmBeforeWrite to be enabled by default")
	}
}

func TestDefaultSettingsEnablePlanTool(t *testing.T) {
	s := DefaultSettings()
	if s.EnablePlanTool == nil || !*s.EnablePlanTool {
		t.Fatal("expected enablePlanTool to be enabled by default")
	}
	if !s.IsPlanToolEnabled() {
		t.Fatal("expected IsPlanToolEnabled to return true by default")
	}
}

func TestResolveKey(t *testing.T) {
	s := &Settings{
		Providers: map[string]*ProviderConfig{
			"test": {
				APIKey: "test-api-key",
			},
		},
	}

	// Test direct key
	key := s.ResolveKey("test")
	if key != "test-api-key" {
		t.Errorf("expected 'test-api-key', got '%s'", key)
	}

	// Test env var
	os.Setenv("TEST_API_KEY", "env-key")
	s.Providers["env"] = &ProviderConfig{
		APIKey: "TEST_API_KEY",
	}
	key = s.ResolveKey("env")
	if key != "env-key" {
		t.Errorf("expected 'env-key', got '%s'", key)
	}
	os.Unsetenv("TEST_API_KEY")

	// Test missing key
	key = s.ResolveKey("nonexistent")
	if key != "" {
		t.Errorf("expected empty string, got '%s'", key)
	}
}

func TestResolveProviderHeaders(t *testing.T) {
	t.Setenv("CUSTOM_HEADER_VALUE", "env-header-value")
	s := &Settings{
		Providers: map[string]*ProviderConfig{
			"test": {
				Headers: map[string]string{
					"X-Static": "static-value",
					"X-Env":    "${CUSTOM_HEADER_VALUE}",
					" ":        "ignored",
				},
			},
		},
	}

	headers := s.ResolveProviderHeaders("test")
	if headers["X-Static"] != "static-value" {
		t.Fatalf("X-Static = %q, want static-value", headers["X-Static"])
	}
	if headers["X-Env"] != "env-header-value" {
		t.Fatalf("X-Env = %q, want env-header-value", headers["X-Env"])
	}
	if _, ok := headers[""]; ok {
		t.Fatal("expected empty header name to be ignored")
	}
	if got := s.ResolveProviderHeaders("missing"); got != nil {
		t.Fatalf("missing headers = %#v, want nil", got)
	}
}

func TestGetShell(t *testing.T) {
	s := &Settings{}

	// Test default
	shell := s.GetShell()
	if shell == "" {
		t.Error("expected non-empty shell")
	}

	// Test custom
	s.ShellPath = "/bin/zsh"
	shell = s.GetShell()
	if shell != "/bin/zsh" {
		t.Errorf("expected '/bin/zsh', got '%s'", shell)
	}
}

func TestGetSessionDir(t *testing.T) {
	s := &Settings{}

	// Test default
	dir := s.GetSessionDir()
	if dir == "" {
		t.Error("expected non-empty session dir")
	}

	// Test custom
	s.SessionDir = "/tmp/sessions"
	dir = s.GetSessionDir()
	if dir != "/tmp/sessions" {
		t.Errorf("expected '/tmp/sessions', got '%s'", dir)
	}

	// Test with tilde
	s.SessionDir = "~/sessions"
	dir = s.GetSessionDir()
	if dir == "" {
		t.Error("expected non-empty session dir")
	}
}

func TestGetSessionDirNormalizesLegacyDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	s := &Settings{SessionDir: "~/.vibecoding/sessions"}
	want := filepath.Join(home, ".mothx", "sessions")
	if got := s.GetSessionDir(); got != want {
		t.Fatalf("GetSessionDir() = %q, want %q", got, want)
	}

	s.SessionDir = filepath.Join(home, ".vibecoding", "sessions")
	if got := s.GetSessionDir(); got != want {
		t.Fatalf("GetSessionDir() absolute legacy = %q, want %q", got, want)
	}
}

func TestGetSessionDirPreservesCustomLegacyNamedPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	custom := filepath.Join(home, "projects", ".vibecoding", "sessions")
	s := &Settings{SessionDir: custom}
	if got := s.GetSessionDir(); got != custom {
		t.Fatalf("GetSessionDir() = %q, want custom %q", got, custom)
	}
}

func TestGetGlobalSkillsDir(t *testing.T) {
	s := &Settings{}

	// Test default
	dir := s.GetGlobalSkillsDir()
	if dir == "" {
		t.Error("expected non-empty skills dir")
	}

	// Test custom
	s.SkillsDir = "/tmp/skills"
	dir = s.GetGlobalSkillsDir()
	if dir != "/tmp/skills" {
		t.Errorf("expected '/tmp/skills', got '%s'", dir)
	}
}

func TestGetGlobalSkillsDirNormalizesLegacyDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	s := &Settings{SkillsDir: "~/.vibecoding/skills"}
	want := filepath.Join(home, ".mothx", "skills")
	if got := s.GetGlobalSkillsDir(); got != want {
		t.Fatalf("GetGlobalSkillsDir() = %q, want %q", got, want)
	}
}

func TestDefaultSkillHubSettings(t *testing.T) {
	settings := DefaultSettings()
	if settings.SkillHub.DefaultMarket != "skillhub.cn" {
		t.Fatalf("default SkillHub market = %q", settings.SkillHub.DefaultMarket)
	}
	if settings.SkillHub.DefaultInstallScope != "project" {
		t.Fatalf("default SkillHub scope = %q", settings.SkillHub.DefaultInstallScope)
	}
	if len(settings.SkillHub.OfficialHandles) != 1 || settings.SkillHub.OfficialHandles[0] != DefaultSkillHubOfficialHandle {
		t.Fatalf("default SkillHub official handles = %#v", settings.SkillHub.OfficialHandles)
	}
}

func TestSaveGlobalSettings(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	os.Setenv("VIBECODING_DIR", tmpDir)
	defer os.Unsetenv("VIBECODING_DIR")

	s := DefaultSettings()
	s.DefaultProvider = "test"

	err := SaveGlobalSettings(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Error("expected settings file to exist")
	}

	// Load and verify
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	loaded := &Settings{}
	if err := json.Unmarshal(data, loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.DefaultProvider != "test" {
		t.Errorf("expected provider 'test', got '%s'", loaded.DefaultProvider)
	}
}

func TestResolveKeyValue(t *testing.T) {
	// Test direct value
	key := resolveKeyValue("direct-key")
	if key != "direct-key" {
		t.Errorf("expected 'direct-key', got '%s'", key)
	}

	// Test env var
	os.Setenv("TEST_ENV_KEY", "env-value")
	key = resolveKeyValue("TEST_ENV_KEY")
	if key != "env-value" {
		t.Errorf("expected 'env-value', got '%s'", key)
	}
	os.Unsetenv("TEST_ENV_KEY")
}

func TestResolveKeyValueShellCommandRequiresOptIn(t *testing.T) {
	t.Setenv("VIBECODING_ALLOW_SHELL_CONFIG", "")
	if got := resolveKeyValue("!printf secret"); got != "!printf secret" {
		t.Fatalf("resolveKeyValue without opt-in = %q, want literal", got)
	}

	t.Setenv("VIBECODING_ALLOW_SHELL_CONFIG", "1")
	if got := resolveKeyValue("!printf secret"); got != "secret" {
		t.Fatalf("resolveKeyValue with opt-in = %q, want secret", got)
	}
}

func TestResolveModelConfigTracksUserSetMaxTokens(t *testing.T) {
	settings := DefaultSettings()
	settings.Providers["openai"] = &ProviderConfig{
		Models: []ModelConfig{{
			ID:        "gpt-4o",
			MaxTokens: 12345,
		}},
	}

	model := ResolveModelConfig("openai", "gpt-4o", settings)
	if model == nil {
		t.Fatal("ResolveModelConfig returned nil")
	}
	if !model.MaxTokensWasSet() {
		t.Fatal("MaxTokensWasSet = false, want true for runtime override")
	}
	if model.MaxTokens != 12345 {
		t.Fatalf("MaxTokens = %d, want 12345", model.MaxTokens)
	}

	builtin := DefaultModelConfig("openai", "gpt-4o")
	if builtin == nil {
		t.Fatal("DefaultModelConfig returned nil")
	}
	if builtin.MaxTokensWasSet() {
		t.Fatal("builtin MaxTokensWasSet = true, want false")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
