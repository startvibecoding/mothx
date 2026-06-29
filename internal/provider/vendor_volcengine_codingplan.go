package provider

import (
	"strings"
)

func init() {
	RegisterVendorAdapter(volccodingplanAdapter{})
}

// volccodingplanAdapter is a vendor that users explicitly select via
// vendor="volcengine-codingplan". It auto-detects the API protocol from
// the baseURL path: /api/coding/v3 → openai-chat, /api/coding → anthropic-messages.
// It does not participate in automatic domain-based vendor detection.
type volccodingplanAdapter struct{}

func (a volccodingplanAdapter) Name() string { return "volcengine-codingplan" }

func (a volccodingplanAdapter) MatchBaseURL(baseURL string) bool { return false }

func (a volccodingplanAdapter) Apply(cfg *AdapterConfig) {
	if cfg.API == "" {
		if strings.Contains(strings.ToLower(cfg.BaseURL), "/api/coding/v3") {
			cfg.API = "openai-chat"
		} else {
			cfg.API = "anthropic-messages"
		}
	}
}
