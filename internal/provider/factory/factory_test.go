package factory

import (
	"testing"

	"github.com/startvibecoding/vibecoding/internal/config"
)

func TestCreateAppliesExplicitVendorDefaults(t *testing.T) {
	settings := config.DefaultSettings()
	settings.Providers = map[string]*config.ProviderConfig{
		"custom-deepseek": {
			Vendor:  "deepseek",
			BaseURL: "https://example.com/v1",
			APIKey:  "fake-key",
			API:     "openai-chat",
			Models: []config.ModelConfig{
				{ID: "m1", Name: "M1", Reasoning: true},
			},
		},
	}
	settings.DefaultProvider = "custom-deepseek"
	settings.DefaultModel = "m1"

	p, model, err := Create(settings, "", "")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if p.Name() != "openai" {
		t.Fatalf("provider name = %q, want openai", p.Name())
	}
	if model == nil || model.ID != "m1" {
		t.Fatalf("model = %#v, want m1", model)
	}
}

func TestConvertModelConfigsPreservesCompat(t *testing.T) {
	supportsReasoningEffort := false
	models := ConvertModelConfigs("test", []config.ModelConfig{
		{
			ID:        "m1",
			Name:      "M1",
			Reasoning: true,
			Compat: &config.ModelCompat{
				ThinkingFormat:          "deepseek",
				SupportsReasoningEffort: &supportsReasoningEffort,
				MaxTokensField:          "max_completion_tokens",
			},
		},
	})
	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1", len(models))
	}
	compat := models[0].Compat
	if compat == nil {
		t.Fatal("compat = nil")
	}
	if compat.ThinkingFormat != "deepseek" {
		t.Fatalf("ThinkingFormat = %q, want deepseek", compat.ThinkingFormat)
	}
	if compat.SupportsReasoningEffort == nil || *compat.SupportsReasoningEffort {
		t.Fatalf("SupportsReasoningEffort = %#v, want false", compat.SupportsReasoningEffort)
	}
	if compat.MaxTokensField != "max_completion_tokens" {
		t.Fatalf("MaxTokensField = %q, want max_completion_tokens", compat.MaxTokensField)
	}
}

func TestCreateOpenAIResponsesProvider(t *testing.T) {
	settings := &config.Settings{
		Providers: map[string]*config.ProviderConfig{
			"openai-responses-test": {
				APIKey:  "fake-key",
				BaseURL: "https://api.openai.com/v1",
				API:     "openai-responses",
				Responses: config.ResponsesConfig{
					ReasoningSummary:     "concise",
					PromptCacheKey:       "custom-cache-key",
					PromptCacheRetention: "24h",
				},
				Models: []config.ModelConfig{
					{ID: "gpt-test", Name: "GPT Test"},
				},
			},
		},
	}

	p, model, err := Create(settings, "openai-responses-test", "gpt-test")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}
	if model == nil || model.ID != "gpt-test" {
		t.Fatalf("model = %#v, want gpt-test", model)
	}
}

func TestCreateGoogleGeminiProvider(t *testing.T) {
	settings := &config.Settings{
		Providers: map[string]*config.ProviderConfig{
			"gemini-test": {
				APIKey:  "fake-key",
				BaseURL: "https://generativelanguage.googleapis.com/v1beta/models",
				API:     "google-gemini",
				Models: []config.ModelConfig{
					{ID: "gemini-test", Name: "Gemini Test", Reasoning: true},
				},
			},
		},
	}

	p, model, err := Create(settings, "gemini-test", "gemini-test")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if p.Name() != "google-gemini" {
		t.Fatalf("provider name = %q, want google-gemini", p.Name())
	}
	if model == nil || model.ID != "gemini-test" {
		t.Fatalf("model = %#v, want gemini-test", model)
	}
}

func TestCreateGoogleVertexProvider(t *testing.T) {
	settings := &config.Settings{
		Providers: map[string]*config.ProviderConfig{
			"vertex-test": {
				APIKey:  "fake-token",
				BaseURL: "https://aiplatform.googleapis.com/v1/projects/test/locations/global/publishers/google/models",
				API:     "google-vertex",
				Models: []config.ModelConfig{
					{ID: "gemini-test", Name: "Gemini Test", Reasoning: true},
				},
			},
		},
	}

	p, model, err := Create(settings, "vertex-test", "gemini-test")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if p.Name() != "google-vertex" {
		t.Fatalf("provider name = %q, want google-vertex", p.Name())
	}
	if model == nil || model.ID != "gemini-test" {
		t.Fatalf("model = %#v, want gemini-test", model)
	}
}

func TestCreateProviderRejectsInvalidHTTPProxy(t *testing.T) {
	settings := &config.Settings{
		Providers: map[string]*config.ProviderConfig{
			"bad-proxy": {
				APIKey:    "fake-key",
				BaseURL:   "https://api.openai.com/v1",
				API:       "openai-chat",
				HTTPProxy: "http://[::1",
				Models: []config.ModelConfig{
					{ID: "gpt-test", Name: "GPT Test"},
				},
			},
		},
	}

	if _, _, err := Create(settings, "bad-proxy", "gpt-test"); err == nil {
		t.Fatal("expected invalid http proxy error")
	}
}

func TestConvertModelConfigsSupportsReferenceReasoningAlias(t *testing.T) {
	models := ConvertModelConfigs("test", []config.ModelConfig{
		{
			ID:   "m1",
			Name: "M1",
			Compat: &config.ModelCompat{
				RequiresReasoningContentOnAssistantMessages: true,
			},
		},
	})
	compat := models[0].Compat
	if compat == nil || !compat.RequiresReasoningContentOnAssistant {
		t.Fatalf("RequiresReasoningContentOnAssistant = %#v, want true", compat)
	}
}

func TestCreateFallbackToFirstModel(t *testing.T) {
	settings := &config.Settings{
		Providers: map[string]*config.ProviderConfig{
			"custom-provider": {
				APIKey:  "fake-key",
				BaseURL: "https://api.openai.com/v1",
				API:     "openai",
				Models: []config.ModelConfig{
					{ID: "model-one", Name: "Model One"},
					{ID: "model-two", Name: "Model Two"},
				},
			},
		},
		DefaultProvider: "custom-provider",
		DefaultModel:    "model-two",
	}

	// When provider is specified but modelID is "", it should fall back to the first model under the provider (model-one).
	_, model, err := Create(settings, "custom-provider", "")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if model == nil || model.ID != "model-one" {
		t.Fatalf("model = %#v, want model-one", model)
	}

	// When built-in provider is specified but modelID is "", it should fall back to the first model of that built-in provider.
	p2, model2, err := Create(settings, "openai", "")
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	available := p2.Models()
	if len(available) == 0 {
		t.Fatal("expected built-in openai to have models")
	}
	if model2 == nil || model2.ID != available[0].ID {
		t.Fatalf("model = %#v, want first model %s", model2, available[0].ID)
	}
}
