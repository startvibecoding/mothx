package agent

import (
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/provider"
)

// ResolveMaxTokens returns the explicit global override when set; otherwise it
// falls back to the selected model's advertised output limit. A zero return
// means the caller/provider should use its own fallback.
func ResolveMaxTokens(settings *config.Settings, model *provider.Model) int {
	if settings != nil && settings.MaxOutputTokens > 0 {
		return settings.MaxOutputTokens
	}
	return ResolveMaxTokensValue(0, model)
}

// ResolveMaxTokensValue returns an explicit per-request value when set;
// otherwise it falls back to the selected model's advertised output limit.
func ResolveMaxTokensValue(explicit int, model *provider.Model) int {
	if explicit > 0 {
		return explicit
	}
	if model != nil && model.MaxTokens > 0 {
		return model.MaxTokens
	}
	return 0
}
