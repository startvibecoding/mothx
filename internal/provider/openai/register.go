package openai

import (
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/provider"
)

func resolveOpenAIModels(cfg *config.ProviderConfig) []*provider.Model {
	if cfg != nil && len(cfg.Models) > 0 {
		models := make([]*provider.Model, 0, len(cfg.Models))
		for _, m := range cfg.Models {
			input := m.Input
			if len(input) == 0 {
				input = []string{"text", "image"}
			}
			var cost provider.ModelPricing
			if m.Cost != nil {
				cost = provider.ModelPricing{
					Input:      m.Cost.Input,
					Output:     m.Cost.Output,
					CacheRead:  m.Cost.CacheRead,
					CacheWrite: m.Cost.CacheWrite,
				}
			}
			models = append(models, &provider.Model{
				ID:            m.ID,
				Name:          m.Name,
				Provider:      "openai",
				Reasoning:     m.Reasoning,
				Input:         input,
				Cost:          cost,
				ContextWindow: m.ContextWindow,
				MaxTokens:     m.MaxTokens,
				Temperature:   m.Temperature,
				TopP:          m.TopP,
				Compat:        convertCompat(m.Compat),
			})
		}
		return models
	}
	return DefaultModels()
}

func convertCompat(c *config.ModelCompat) *provider.ModelCompat {
	if c == nil {
		return nil
	}
	return &provider.ModelCompat{
		ThinkingFormat:                      c.ThinkingFormat,
		RequiresReasoningContentOnAssistant: c.RequiresReasoningContentOnAssistant || c.RequiresReasoningContentOnAssistantMessages,
		ForceAdaptiveThinking:               c.ForceAdaptiveThinking,
		ParseReasoningInContent:             c.ParseReasoningInContent,
		SupportsDeveloperRole:               cloneBool(c.SupportsDeveloperRole),
		SupportsStore:                       cloneBool(c.SupportsStore),
		SupportsReasoningEffort:             cloneBool(c.SupportsReasoningEffort),
		SupportsStrictMode:                  cloneBool(c.SupportsStrictMode),
		MaxTokensField:                      c.MaxTokensField,
		SupportsCacheControlOnTools:         cloneBool(c.SupportsCacheControlOnTools),
		SupportsLongCacheRetention:          cloneBool(c.SupportsLongCacheRetention),
		SupportsPromptCacheKey:              cloneBool(c.SupportsPromptCacheKey),
		SupportsReasoningSummary:            cloneBool(c.SupportsReasoningSummary),
		SendSessionAffinityHeaders:          c.SendSessionAffinityHeaders,
		SupportsEagerToolInputStreaming:     cloneBool(c.SupportsEagerToolInputStreaming),
	}
}

func cloneBool(v *bool) *bool {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

// init registers the generic OpenAI-compatible provider factory in the global
// provider registry so that agent.Builder.WithProviderByName (which resolves
// through provider.ResolveProvider) can construct OpenAI-style providers.
func init() {
	factory := func(cfg *config.ProviderConfig) (provider.Provider, error) {
		if cfg == nil {
			return NewProvider("", ""), nil
		}
		p, err := NewProviderWithModelsAndProxy(cfg.APIKey, cfg.BaseURL, cfg.HTTPProxy, resolveOpenAIModels(cfg))
		if err != nil {
			return nil, err
		}
		if cfg.API == "openai-responses" {
			p.SetUseResponsesAPI(true)
			p.SetResponsesConfig(cfg.Responses)
		}
		return p, nil
	}
	provider.Register("openai", factory)
	provider.Register("openai-chat", factory)
	provider.Register("openai-responses", factory)
	provider.Register("responses", factory)
}
