package agent

import (
	"github.com/startvibecoding/mothx/internal/provider"
)

// ResolveMaxTokens returns the output limit configured for the active model.
// Output limits are model-specific because provider defaults vary widely.
func ResolveMaxTokens(model *provider.Model) int {
	return ResolveMaxTokensValue(0, model)
}

// ResolveMaxTokensValue returns an explicit per-request value when set;
// otherwise it leaves the output-token limit unset.
func ResolveMaxTokensValue(explicit int, model *provider.Model) int {
	if explicit > 0 {
		return explicit
	}
	if model != nil && model.MaxTokens > 0 {
		return model.MaxTokens
	}
	return 0
}
