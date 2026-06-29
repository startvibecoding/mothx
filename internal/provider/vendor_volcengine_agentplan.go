package provider

import (
	"strings"
)

func init() {
	RegisterVendorAdapter(volcagentplanAdapter{})
}

// volcagentplanAdapter is a vendor that users explicitly select via
// vendor="volcengine-agentplan". It auto-detects the API protocol from
// the baseURL path: /api/plan/v3 → openai-chat, /api/plan → anthropic-messages.
// It does not participate in automatic domain-based vendor detection.
type volcagentplanAdapter struct{}

func (a volcagentplanAdapter) Name() string { return "volcengine-agentplan" }

func (a volcagentplanAdapter) MatchBaseURL(baseURL string) bool { return false }

func (a volcagentplanAdapter) Apply(cfg *AdapterConfig) {
	if cfg.API == "" {
		if strings.Contains(strings.ToLower(cfg.BaseURL), "/api/plan/v3") {
			cfg.API = "openai-chat"
		} else {
			cfg.API = "anthropic-messages"
		}
	}
}
