package agent

import (
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/provider"
)

// ResolveMaxTokens returns the explicit global override when set, then a
// per-model maxTokens value only when it came from user/runtime config. A zero
// return means the caller/provider should omit the output-token limit when the
// upstream API permits it.
func ResolveMaxTokens(settings *config.Settings, model *provider.Model) int {
	if settings != nil && settings.MaxOutputTokens > 0 {
		return settings.MaxOutputTokens
	}
	return ResolveMaxTokensValue(0, model)
}

// ResolveMaxTokensValue returns an explicit per-request value when set;
// otherwise it leaves the output-token limit unset.
func ResolveMaxTokensValue(explicit int, model *provider.Model) int {
	if explicit > 0 {
		return explicit
	}
	if model != nil && model.MaxTokensSet && model.MaxTokens > 0 {
		return model.MaxTokens
	}
	return 0
}
