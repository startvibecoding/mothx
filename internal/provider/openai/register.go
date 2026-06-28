package openai

import (
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/provider"
)

// init registers the generic OpenAI-compatible provider factory in the global
// provider registry so that agent.Builder.WithProviderByName (which resolves
// through provider.ResolveProvider) can construct OpenAI-style providers.
func init() {
	factory := func(cfg *config.ProviderConfig) (provider.Provider, error) {
		if cfg == nil {
			return NewProvider("", ""), nil
		}
		p, err := NewProviderWithModelsAndProxy(cfg.APIKey, cfg.BaseURL, cfg.HTTPProxy, DefaultModels())
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
