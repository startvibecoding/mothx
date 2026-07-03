package imageproc

import "strings"

type Family string

const (
	FamilyGeneric               Family = "generic"
	FamilyOpenAI                Family = "openai"
	FamilyAnthropic             Family = "anthropic"
	FamilyAnthropicBedrock      Family = "anthropic-bedrock"
	FamilyGemini                Family = "gemini"
	FamilyMistral               Family = "mistral"
	FamilyDoubaoSeed            Family = "doubao-seed"
	FamilyQwen                  Family = "qwen"
	FamilyKimi                  Family = "kimi"
	FamilyMiniMax               Family = "minimax"
	FamilyGLM                   Family = "glm"
	FamilyGrok                  Family = "grok"
	FamilyLlamaVision           Family = "llama-vision"
	FamilyGemmaVision           Family = "gemma-vision"
	FamilyMiMo                  Family = "mimo"
	FamilyAmazonNova            Family = "amazon-nova"
	FamilyDeepSeekGatewayVision Family = "deepseek-gateway-vision"
)

type Hint struct {
	ProviderID   string
	ProviderName string
	Vendor       string
	API          string
	BaseURL      string
	ModelID      string
}

func InferFamily(h Hint) Family {
	model := normalizeFamilyKey(h.ModelID)
	providerID := normalizeFamilyKey(h.ProviderID)
	providerName := normalizeFamilyKey(h.ProviderName)
	vendor := normalizeFamilyKey(h.Vendor)
	api := normalizeFamilyKey(h.API)
	baseURL := normalizeFamilyKey(h.BaseURL)
	providerText := strings.Join([]string{providerID, providerName, vendor, baseURL}, " ")

	if model != "" {
		if strings.Contains(model, "deepseek-v4") && isGatewayProvider(providerText) {
			return FamilyDeepSeekGatewayVision
		}
		if strings.Contains(model, "doubao") || strings.Contains(model, "seed-2") || strings.Contains(model, "seed2") {
			return FamilyDoubaoSeed
		}
		if strings.Contains(model, "minimax") {
			return FamilyMiniMax
		}
		if strings.Contains(model, "qwen") {
			return FamilyQwen
		}
		if strings.Contains(model, "kimi") || model == "k2p7" || strings.Contains(model, "k2p7") {
			return FamilyKimi
		}
		if strings.Contains(model, "glm") {
			return FamilyGLM
		}
		if strings.Contains(model, "mimo") {
			return FamilyMiMo
		}
		if strings.Contains(model, "grok") || strings.Contains(model, "x-ai") || strings.Contains(model, "xai/") {
			return FamilyGrok
		}
		if strings.Contains(model, "amazon.nova") || strings.Contains(model, "amazon-nova") {
			return FamilyAmazonNova
		}
		if strings.Contains(model, "llama") && (strings.Contains(model, "vision") || strings.Contains(model, "scout")) {
			return FamilyLlamaVision
		}
		if strings.Contains(model, "gemma") {
			return FamilyGemmaVision
		}
		if strings.Contains(model, "gemini") {
			return FamilyGemini
		}
		if strings.Contains(model, "pixtral") || strings.Contains(model, "mistral") || strings.Contains(model, "devstral") {
			return FamilyMistral
		}
		if strings.Contains(model, "claude") || strings.Contains(model, "anthropic.claude") || strings.Contains(model, "anthropic/claude") {
			if isBedrockProvider(providerText) || strings.HasPrefix(model, "anthropic.") {
				return FamilyAnthropicBedrock
			}
			return FamilyAnthropic
		}
		if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") ||
			strings.HasPrefix(model, "o4") || strings.Contains(model, "openai/gpt-") {
			return FamilyOpenAI
		}
	}

	switch {
	case strings.Contains(providerText, "xiaomi") || strings.Contains(providerText, "mimo"):
		return FamilyMiMo
	case strings.Contains(providerText, "minimax"):
		return FamilyMiniMax
	case strings.Contains(providerText, "moonshot") || strings.Contains(providerText, "kimi"):
		return FamilyKimi
	case strings.Contains(providerText, "zai") || strings.Contains(providerText, "bigmodel"):
		return FamilyGLM
	case strings.Contains(providerText, "xai") || strings.Contains(providerText, "x-ai"):
		return FamilyGrok
	case isBedrockProvider(providerText):
		return FamilyAmazonNova
	case strings.Contains(api, "google") || strings.Contains(providerText, "google-gemini") || strings.Contains(providerText, "google-vertex"):
		return FamilyGemini
	case strings.Contains(providerText, "mistral"):
		return FamilyMistral
	case strings.Contains(providerText, "volcengine"):
		return FamilyDoubaoSeed
	case strings.Contains(providerText, "alibaba") || strings.Contains(providerText, "bailian") || strings.Contains(providerText, "dashscope"):
		return FamilyQwen
	case strings.Contains(providerText, "anthropic") || api == "anthropic-messages":
		return FamilyAnthropic
	case providerName == "openai" || api == "openai-responses":
		return FamilyOpenAI
	default:
		return FamilyGeneric
	}
}

func PolicyForHint(h Hint, mode Mode) Policy {
	policy := DefaultPolicy(mode)
	switch InferFamily(h) {
	case FamilyOpenAI, FamilyAnthropic, FamilyGemini, FamilyGrok:
		raiseFileLimit(&policy, 20<<20)
	case FamilyAnthropicBedrock:
		capFileLimit(&policy, 4<<20)
		capOutputLimit(&policy, 3<<20)
	case FamilyAmazonNova:
		capOutputLimit(&policy, 4<<20)
	case FamilyDoubaoSeed, FamilyQwen, FamilyKimi, FamilyGLM:
		raiseDetailLongEdge(&policy, 2560)
	case FamilyMistral, FamilyMiniMax:
		capOutputLimit(&policy, 5<<20)
	case FamilyMiMo, FamilyLlamaVision, FamilyGemmaVision:
		capOutputLimit(&policy, 4<<20)
	}
	return policy
}

func normalizeFamilyKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("_", "-", " ", "-", ":", "-", ".", ".")
	return replacer.Replace(s)
}

func isBedrockProvider(s string) bool {
	return strings.Contains(s, "bedrock") || strings.Contains(s, "amazonaws.com")
}

func isGatewayProvider(s string) bool {
	for _, marker := range []string{
		"volcengine-agentplan",
		"volcengine-codingplan",
		"alibaba",
		"bailian",
		"dashscope",
		"gitee",
		"moark",
		"opencode",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func capFileLimit(policy *Policy, max int64) {
	if max <= 0 {
		return
	}
	if policy.MaxFileBytes <= 0 || policy.MaxFileBytes > max {
		policy.MaxFileBytes = max
	}
}

func raiseFileLimit(policy *Policy, min int64) {
	if min <= 0 {
		return
	}
	if policy.MaxFileBytes > 0 && policy.MaxFileBytes < min {
		policy.MaxFileBytes = min
	}
}

func capOutputLimit(policy *Policy, max int) {
	if max <= 0 {
		return
	}
	if policy.MaxOutputBytes <= 0 || policy.MaxOutputBytes > max {
		policy.MaxOutputBytes = max
	}
}

func raiseDetailLongEdge(policy *Policy, min int) {
	if policy.Mode == ModeDetail && policy.MaxLongEdge > 0 && policy.MaxLongEdge < min {
		policy.MaxLongEdge = min
	}
}
