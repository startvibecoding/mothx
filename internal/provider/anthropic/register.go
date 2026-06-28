package anthropic

import (
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/provider"
)

// init registers the generic Anthropic-compatible provider factory in the
// global provider registry so that agent.Builder.WithProviderByName (which
// resolves through provider.ResolveProvider) can construct Anthropic-style
// providers.
func init() {
	factory := func(cfg *config.ProviderConfig) (provider.Provider, error) {
		if cfg == nil {
			return NewProvider("", ""), nil
		}
		return NewProviderWithModelsAndProxy(cfg.APIKey, cfg.BaseURL, cfg.HTTPProxy, DefaultModels())
	}
	provider.Register("anthropic", factory)
	provider.Register("anthropic-messages", factory)
}
