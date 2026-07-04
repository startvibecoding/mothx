package provider

import (
	"strings"
)

func init() {
	RegisterVendorAdapter(tencentHYPlanAdapter{})
}

// tencentHYPlanAdapter is a vendor that users explicitly select via
// vendor="tencent-hy-plan". It supports both OpenAI-compatible and
// Anthropic-compatible endpoints via base URL path detection.
type tencentHYPlanAdapter struct{}

func (a tencentHYPlanAdapter) Name() string { return "tencent-hy-plan" }

func (a tencentHYPlanAdapter) MatchBaseURL(baseURL string) bool { return false }

func (a tencentHYPlanAdapter) Apply(cfg *AdapterConfig) {
	if cfg.API == "" {
		lower := strings.ToLower(cfg.BaseURL)
		if strings.Contains(lower, "/plan/anthropic") {
			cfg.API = "anthropic-messages"
		} else {
			cfg.API = "openai-chat"
		}
	}
}
