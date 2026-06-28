package tui

import (
	"testing"

	"github.com/startvibecoding/vibecoding/internal/config"
)

func TestAuthBuildSettingsPreservesExistingModelConfig(t *testing.T) {
	s := config.DefaultSettings()
	a := &App{
		settings: s,
		auth: authDialogState{
			ProviderID:          "deepseek-openai",
			API:                 "openai-chat",
			BaseURL:             "https://api.deepseek.com",
			HTTPProxy:           "http://127.0.0.1:7890",
			ForceHTTP11:         true,
			APIKey:              "test-key",
			ModelIDs:            "deepseek-v4-pro, custom-model",
			ContextWindow:       "200000",
			MaxTokens:           "10000",
			Reasoning:           true,
			InputTypes:          "text,image",
			Temperature:         "0.7",
			TopP:                "0.9",
			SetDefault:          true,
			ContextWindowEdited: true,
			MaxTokensEdited:     true,
			ReasoningEdited:     true,
			InputTypesEdited:    true,
			TemperatureEdited:   true,
			TopPEdited:          true,
		},
	}

	next, modelID := a.buildAuthSettings()
	if modelID != "deepseek-v4-pro" {
		t.Fatalf("modelID = %q, want deepseek-v4-pro", modelID)
	}
	pc := next.GetProviderConfig("deepseek-openai")
	if pc == nil || len(pc.Models) != 2 {
		t.Fatalf("models = %#v, want 2 models", pc)
	}
	if pc.HTTPProxy != "http://127.0.0.1:7890" {
		t.Fatalf("httpProxy = %q", pc.HTTPProxy)
	}
	if !pc.ForceHTTP11 {
		t.Fatal("forceHTTP11 = false, want true")
	}
	if !pc.Models[0].Reasoning || pc.Models[0].ContextWindow != 200000 || pc.Models[0].MaxTokens != 10000 {
		t.Fatalf("existing model config/overrides unexpected: %#v", pc.Models[0])
	}
	if pc.Models[0].Temperature == nil || *pc.Models[0].Temperature != 0.7 {
		t.Fatalf("temperature override missing: %#v", pc.Models[0].Temperature)
	}
	if pc.Models[0].TopP == nil || *pc.Models[0].TopP != 0.9 {
		t.Fatalf("top_p override missing: %#v", pc.Models[0].TopP)
	}
	if pc.Models[1].ID != "custom-model" || pc.Models[1].ContextWindow != 200000 || pc.Models[1].MaxTokens != 10000 {
		t.Fatalf("custom model default config unexpected: %#v", pc.Models[1])
	}
	if next.DefaultProvider != "deepseek-openai" || next.DefaultModel != "deepseek-v4-pro" {
		t.Fatalf("defaults = %s/%s", next.DefaultProvider, next.DefaultModel)
	}
}

func TestAuthBuildSettingsFromUsesProvidedBase(t *testing.T) {
	runtime := config.DefaultSettings()
	runtime.DefaultProvider = "project-provider"
	runtime.Providers["project-only"] = &config.ProviderConfig{API: "openai-chat", BaseURL: "https://project.test", APIKey: "project", Models: []config.ModelConfig{{ID: "project-model", Name: "Project"}}}
	global := &config.Settings{Providers: map[string]*config.ProviderConfig{"xiaomi": runtime.Providers["xiaomi"]}}
	a := &App{settings: runtime, auth: authDialogState{ProviderID: "openrouter", API: "openai-chat", BaseURL: "https://openrouter.ai/api/v1", APIKey: "test", ModelIDs: "z-ai/glm-4.5-air:free", SetDefault: true}}

	next, _ := a.buildAuthSettingsFrom(global)
	if next.GetProviderConfig("project-only") != nil {
		t.Fatal("project-only provider leaked into global settings patch")
	}
	if next.GetProviderConfig("deepseek-openai") != nil {
		t.Fatal("default provider leaked into sparse global settings patch")
	}
	if next.GetProviderConfig("xiaomi") == nil || next.GetProviderConfig("openrouter") == nil {
		t.Fatalf("expected sparse global providers xiaomi/openrouter, got %#v", next.Providers)
	}
	if next.DefaultProvider != "openrouter" {
		t.Fatalf("DefaultProvider = %q, want openrouter", next.DefaultProvider)
	}
}

func TestAuthExistingCustomProvidersRemainVisible(t *testing.T) {
	s := &config.Settings{Providers: map[string]*config.ProviderConfig{
		"xiaomi":   {},
		"gitee":    {},
		"gitee-cc": {},
		"doubao":   {},
	}}
	ids := sortedAuthProviderIDs(s)
	for _, want := range []string{"xiaomi", "gitee", "gitee-cc", "doubao"} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("provider %q missing from auth list: %#v", want, ids)
		}
	}
}

func TestAuthSparsePatchPreservesExistingProviders(t *testing.T) {
	global := &config.Settings{Providers: map[string]*config.ProviderConfig{
		"xiaomi":   {API: "openai-chat", BaseURL: "https://old.xiaomi", APIKey: "old", Models: []config.ModelConfig{{ID: "mimo", Name: "MiMo"}}},
		"gitee":    {API: "openai-chat", BaseURL: "https://gitee.test", APIKey: "gitee", Models: []config.ModelConfig{{ID: "gitee-model", Name: "Gitee"}}},
		"gitee-cc": {API: "openai-chat", BaseURL: "https://cc.gitee.test", APIKey: "gitee-cc", Models: []config.ModelConfig{{ID: "cc-model", Name: "Gitee CC"}}},
		"doubao":   {API: "openai-chat", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", APIKey: "doubao", Models: []config.ModelConfig{{ID: "doubao-model", Name: "Doubao"}}},
	}}
	a := &App{settings: config.DefaultSettings(), auth: authDialogState{ProviderID: "xiaomi", API: "openai-chat", BaseURL: "https://new.xiaomi", APIKey: "new", ModelIDs: "mimo", SetDefault: true}}
	next, _ := a.buildAuthSettingsFrom(global)
	for _, want := range []string{"xiaomi", "gitee", "gitee-cc", "doubao"} {
		if next.GetProviderConfig(want) == nil {
			t.Fatalf("provider %q was dropped from sparse patch: %#v", want, next.Providers)
		}
	}
	if next.GetProviderConfig("openai") != nil || next.GetProviderConfig("deepseek-openai") != nil {
		t.Fatalf("default providers leaked into sparse patch: %#v", next.Providers)
	}
	if next.GetProviderConfig("xiaomi").BaseURL != "https://new.xiaomi" {
		t.Fatalf("xiaomi baseURL not updated: %#v", next.GetProviderConfig("xiaomi"))
	}
}

func TestAuthExistingProviderDoesNotSkipBaseURL(t *testing.T) {
	a := &App{auth: authDialogState{Open: true, View: authViewAPIKey, ProviderID: "gitee", API: "openai-chat", BaseURL: "https://old.gitee", ModelIDs: "gitee-model"}}
	a.prepareAuthInput()
	a.authInput = a.authInput.SetValue("new-key")
	a.submitAuthInput()
	if a.auth.View != authViewBaseURL {
		t.Fatalf("view = %v, want authViewBaseURL", a.auth.View)
	}
	if a.authInput.Value() != "https://old.gitee" {
		t.Fatalf("baseURL input = %q, want old base URL", a.authInput.Value())
	}
	a.authInput = a.authInput.SetValue("https://new.gitee")
	a.submitAuthInput()
	if a.auth.View != authViewHTTPProxy {
		t.Fatalf("view = %v, want authViewHTTPProxy", a.auth.View)
	}
}

func TestAuthCustomProviderAPIKeyAdvancesToModels(t *testing.T) {
	a := &App{auth: authDialogState{Open: true, View: authViewAPIKey, Mode: "custom", ProviderID: "openrouter", API: "openai-chat"}}
	a.prepareAuthInput()
	a.authInput = a.authInput.SetValue("test-key")
	a.submitAuthInput()
	if a.auth.View != authViewModels {
		t.Fatalf("view = %v, want authViewModels", a.auth.View)
	}
}

func TestAuthExistingProviderLoadsForceHTTP11(t *testing.T) {
	a := &App{
		settings: &config.Settings{Providers: map[string]*config.ProviderConfig{
			"custom": {API: "openai-chat", BaseURL: "https://custom.test", APIKey: "key", ForceHTTP11: true, Models: []config.ModelConfig{{ID: "model", Name: "Model"}}},
		}},
		auth: authDialogState{Open: true, View: authViewExistingProvider},
	}

	a.selectAuthOption()
	if !a.auth.ForceHTTP11 {
		t.Fatal("ForceHTTP11 was not loaded from provider config")
	}

	a.auth.Stack = []authView{authViewEditMenu}
	a.jumpAuthEdit("forceHTTP11")
	if a.auth.ForceHTTP11 {
		t.Fatal("ForceHTTP11 was not toggled from edit menu")
	}
	if a.auth.View != authViewReview {
		t.Fatalf("view = %v, want authViewReview", a.auth.View)
	}
}

func TestAuthProviderSortOrder(t *testing.T) {
	s := &config.Settings{Providers: map[string]*config.ProviderConfig{
		"google-gemini":      {},
		"anthropic":          {},
		"openai":             {},
		"doubao":             {},
		"xiaomi":             {},
		"deepseek-openai":    {},
		"moark":              {},
		"z-other":            {},
		"deepseek-anthropic": {},
	}}
	ids := sortedAuthProviderIDs(s)
	wantPrefix := []string{"moark", "deepseek-anthropic", "deepseek-openai", "xiaomi", "doubao", "openai", "anthropic", "google-gemini"}
	if len(ids) < len(wantPrefix) {
		t.Fatalf("ids too short: %#v", ids)
	}
	for i, want := range wantPrefix {
		if ids[i] != want {
			t.Fatalf("ids[%d] = %q, want %q (all: %#v)", i, ids[i], want, ids)
		}
	}
}

func TestAuthProviderSearchFilter(t *testing.T) {
	ids := []string{"moark", "deepseek-openai", "xiaomi", "openai", "moonshotai", "moonshotai-cn"}
	got := filterAuthProviderIDs(ids, "moon")
	want := []string{"moonshotai", "moonshotai-cn"}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
	got = filterAuthProviderIDs(ids, "ai")
	if len(got) == 0 || got[0] != "deepseek-openai" {
		t.Fatalf("contains search should keep priority ordering, got %#v", got)
	}
}

func TestAuthBaseURLOptionsForProvider(t *testing.T) {
	for _, tc := range []struct {
		provider string
		wantMin  int
	}{
		{provider: "minimax", wantMin: 2},
		{provider: "zai", wantMin: 2},
		{provider: "alibaba-standard", wantMin: 4},
		{provider: "alibaba-coding-plan", wantMin: 2},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			opts := baseURLOptionsForProvider(tc.provider)
			if len(opts) < tc.wantMin {
				t.Fatalf("got %d options, want at least %d", len(opts), tc.wantMin)
			}
			for _, opt := range opts {
				if opt.Value == "" || opt.Title == "" {
					t.Fatalf("invalid option: %#v", opt)
				}
			}
		})
	}
}

func TestAuthVisibleRange(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cursor    int
		total     int
		wantStart int
		wantEnd   int
	}{
		{name: "short", cursor: 0, total: 3, wantStart: 0, wantEnd: 3},
		{name: "top", cursor: 0, total: 12, wantStart: 0, wantEnd: 5},
		{name: "middle", cursor: 6, total: 12, wantStart: 4, wantEnd: 9},
		{name: "bottom", cursor: 11, total: 12, wantStart: 7, wantEnd: 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd := authVisibleRange(tc.cursor, tc.total, authMaxVisibleOptions)
			if gotStart != tc.wantStart || gotEnd != tc.wantEnd {
				t.Fatalf("range = %d:%d, want %d:%d", gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestRenderAuthPreviewTruncates(t *testing.T) {
	var preview string
	for i := 0; i < authMaxPreviewVisibleLines+5; i++ {
		preview += "line\n"
	}
	lines := renderAuthPreview(preview)
	if len(lines) != authMaxPreviewVisibleLines+1 {
		t.Fatalf("rendered %d lines, want %d", len(lines), authMaxPreviewVisibleLines+1)
	}
	if lines[len(lines)-1] == "line" {
		t.Fatalf("expected truncation marker, got %q", lines[len(lines)-1])
	}
}
