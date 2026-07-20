package agent

import (
	"github.com/startvibecoding/mothx/internal/provider"
)

// ResolveMaxTokens returns the output limit configured for the active model.
// Output limits are model-specific because provider defaults vary widely.
func ResolveMaxTokens(model *provider.Model) int {
	return ResolveMaxTokensValue(0, model)
}

// ResolveMaxTokensValue returns an explicit per-request value when set.
// An explicit zero on a model disables the output-token parameter; otherwise
// the configured model limit is used.
func ResolveMaxTokensValue(explicit int, model *provider.Model) int {
	if explicit > 0 {
		return explicit
	}
	if model != nil {
		if model.MaxTokensSet && model.MaxTokens == 0 {
			return 0
		}
		if model.MaxTokens > 0 {
			return model.MaxTokens
		}
	}
	return 0
}
