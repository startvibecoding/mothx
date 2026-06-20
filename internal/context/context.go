package context

import "github.com/startvibecoding/vibecoding/internal/provider"

// ContextUsage holds the current context usage information.
type ContextUsage struct {
	Tokens        int      // Current estimated context tokens
	ContextWindow int      // Maximum context window size
	Percent       *float64 // Usage percentage, nil if unknown
}

// EstimateTokens estimates token count for a message using the default estimator.
func EstimateTokens(msg provider.Message) int {
	return GenericTokenEstimator{}.EstimateTokens(msg)
}

// CalculateContextTokens calculates total context tokens from usage.
// Uses the totalTokens field when available, falls back to computing from components.
func CalculateContextTokens(usage *provider.Usage) int {
	if usage == nil {
		return 0
	}
	if usage.TotalTokens > 0 {
		return usage.TotalTokens
	}
	return usage.Input + usage.Output + usage.CacheRead + usage.CacheWrite
}

// EstimateContextTokens estimates context tokens from messages.
// Uses the last assistant's usage when available, then estimates trailing messages.
func EstimateContextTokens(messages []provider.Message) (tokens int, lastUsageIndex int) {
	return EstimateContextTokensWithEstimator(messages, GenericTokenEstimator{})
}

// EstimateContextTokensWithEstimator estimates context tokens using provider
// usage when available, then the supplied estimator for trailing messages.
func EstimateContextTokensWithEstimator(messages []provider.Message, estimator TokenEstimator) (tokens int, lastUsageIndex int) {
	if estimator == nil {
		estimator = GenericTokenEstimator{}
	}
	lastUsageIndex = -1
	usageTokens := 0

	// Find last assistant message with usage
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "assistant" && msg.Usage != nil {
			totalTokens := CalculateContextTokens(msg.Usage)
			if totalTokens > 0 {
				usageTokens = totalTokens
				lastUsageIndex = i
				break
			}
		}
	}

	// If we found usage, estimate trailing messages
	if lastUsageIndex >= 0 {
		trailingTokens := 0
		for i := lastUsageIndex + 1; i < len(messages); i++ {
			trailingTokens += estimator.EstimateTokens(messages[i])
		}
		return usageTokens + trailingTokens, lastUsageIndex
	}

	// No usage data, estimate all messages
	return estimator.EstimateMessagesTokens(messages), -1
}

// ShouldCompact checks if compaction should trigger based on context usage.
func ShouldCompact(contextTokens int, contextWindow int, reserveTokens int) bool {
	if contextWindow <= 0 {
		return false
	}
	return contextTokens > contextWindow-reserveTokens
}
