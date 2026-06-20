package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/platform"
)

// Verbose controls whether config loading prints diagnostic messages to stderr.
var Verbose bool

// Settings holds all configuration for vibecoding.
type Settings struct {
	Providers            map[string]*ProviderConfig `json:"providers,omitempty"`
	DefaultProvider      string                     `json:"defaultProvider,omitempty"`
	DefaultModel         string                     `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string                     `json:"defaultThinkingLevel,omitempty"`
	DefaultMode          string                     `json:"defaultMode,omitempty"`
	EnablePlanTool       *bool                      `json:"enablePlanTool,omitempty"`
	WebSearch            WebSearchSettings          `json:"webSearch"`
	MaxContextTokens     int                        `json:"maxContextTokens,omitempty"`
	MaxOutputTokens      int                        `json:"maxOutputTokens,omitempty"`
	ContextFiles         ContextFilesSettings       `json:"contextFiles"`
	SkillsDir            string                     `json:"skillsDir,omitempty"`
	Compaction           CompactionSettings         `json:"compaction"`
	Sandbox              SandboxSettings            `json:"sandbox"`
	SessionDir           string                     `json:"sessionDir,omitempty"`
	ShellPath            string                     `json:"shellPath,omitempty"`
	ShellCommandPrefix   string                     `json:"shellCommandPrefix,omitempty"`
	Theme                string                     `json:"theme,omitempty"`
	Retry                RetrySettings              `json:"retry"`
	Approval             ApprovalSettings           `json:"approval"`
}

type ProviderConfig struct {
	Vendor         string            `json:"vendor,omitempty"`    // Explicit vendor adapter (Decision 12/13)
	APIKey         string            `json:"apiKey,omitempty"`    // API key or env/shell reference
	BaseURL        string            `json:"baseUrl,omitempty"`   // API base URL
	HTTPProxy      string            `json:"httpProxy,omitempty"` // optional per-provider HTTP proxy URL, e.g. http://127.0.0.1:7890
	Headers        map[string]string `json:"headers,omitempty"`   // optional per-provider HTTP headers
	API            string            `json:"api,omitempty"`
	ThinkingFormat string            `json:"thinkingFormat,omitempty"` // "", "openai", "anthropic", "deepseek", "xiaomi"
	CacheControl   *bool             `json:"cacheControl,omitempty"`   // enable Anthropic prompt caching (nil/false=off, true=on; set true for Claude models)
	Responses      ResponsesConfig   `json:"responses,omitempty"`
	Models         []ModelConfig     `json:"models"`
}

type ResponsesConfig struct {
	ReasoningSummary     string `json:"reasoningSummary,omitempty"`     // "auto" (default), "concise", or "detailed"
	PromptCacheEnabled   *bool  `json:"promptCacheEnabled,omitempty"`   // nil/true = on, false = off
	PromptCacheKey       string `json:"promptCacheKey,omitempty"`       // optional explicit cache key; defaults to provider/model stable key
	PromptCacheRetention string `json:"promptCacheRetention,omitempty"` // optional OpenAI prompt cache retention value
}

type WebSearchSettings struct {
	Enabled      *bool  `json:"enabled,omitempty"`
	Provider     string `json:"provider,omitempty"`
	ProviderType string `json:"providerType,omitempty"`
	Model        string `json:"model,omitempty"`
}

type ModelConfig struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Reasoning     bool         `json:"reasoning,omitempty"`
	ContextWindow int          `json:"contextWindow,omitempty"`
	MaxTokens     int          `json:"maxTokens,omitempty"`
	Temperature   *float64     `json:"temperature,omitempty"` // nil = use API default
	TopP          *float64     `json:"top_p,omitempty"`       // nil = use API default
	Cost          *CostConfig  `json:"cost,omitempty"`
	Input         []string     `json:"input,omitempty"`
	Compat        *ModelCompat `json:"compat,omitempty"` // Vendor compatibility flags (Decision 14)
}

type CostConfig struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead,omitempty"`
	CacheWrite float64 `json:"cacheWrite,omitempty"`
}

// ModelCompat defines per-model compatibility flags (Decision 14).
// Reference: pi/packages/ai/src/models.generated.ts compat field
type ModelCompat struct {
	// Thinking/reasoning
	ThinkingFormat                              string `json:"thinkingFormat,omitempty"`
	RequiresReasoningContentOnAssistant         bool   `json:"requiresReasoningContentOnAssistant,omitempty"`
	RequiresReasoningContentOnAssistantMessages bool   `json:"requiresReasoningContentOnAssistantMessages,omitempty"`
	ForceAdaptiveThinking                       bool   `json:"forceAdaptiveThinking,omitempty"`

	// API parameter compatibility
	SupportsDeveloperRole   *bool  `json:"supportsDeveloperRole,omitempty"`
	SupportsStore           *bool  `json:"supportsStore,omitempty"`
	SupportsReasoningEffort *bool  `json:"supportsReasoningEffort,omitempty"`
	SupportsStrictMode      *bool  `json:"supportsStrictMode,omitempty"`
	MaxTokensField          string `json:"maxTokensField,omitempty"`

	// Cache
	SupportsCacheControlOnTools *bool `json:"supportsCacheControlOnTools,omitempty"`
	SupportsLongCacheRetention  *bool `json:"supportsLongCacheRetention,omitempty"`
	SupportsPromptCacheKey      *bool `json:"supportsPromptCacheKey,omitempty"`
	SupportsReasoningSummary    *bool `json:"supportsReasoningSummary,omitempty"`
	SendSessionAffinityHeaders  bool  `json:"sendSessionAffinityHeaders,omitempty"`

	// Streaming
	SupportsEagerToolInputStreaming *bool `json:"supportsEagerToolInputStreaming,omitempty"`
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(v bool) *bool { return &v }

type ContextFilesSettings struct {
	Enabled    bool     `json:"enabled"`
	ExtraFiles []string `json:"extraFiles,omitempty"`
}

type CompactionSettings struct {
	Enabled          bool   `json:"enabled"`
	ReserveTokens    int    `json:"reserveTokens"`
	KeepRecentTokens int    `json:"keepRecentTokens"`
	Tokenizer        string `json:"tokenizer,omitempty"`
	TokenizerModel   string `json:"tokenizerModel,omitempty"`
	Template         string `json:"template,omitempty"`

	// Idle compression settings (R5.1-R5.5)
	IdleCompressionEnabled   bool `json:"idleCompressionEnabled,omitempty"`   // R5.1: off by default
	IdleTimeoutSeconds       int  `json:"idleTimeoutSeconds,omitempty"`       // seconds of inactivity (default: 90)
	IdleMinTokensForCompress int  `json:"idleMinTokensForCompress,omitempty"` // minimum tokens to trigger (default: 150000)
}

type SandboxSettings struct {
	Enabled      bool     `json:"enabled"`
	Level        string   `json:"level"`
	BwrapPath    string   `json:"bwrapPath,omitempty"`
	AllowNetwork bool     `json:"allowNetwork"`
	AllowedRead  []string `json:"allowedRead,omitempty"`
	AllowedWrite []string `json:"allowedWrite,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`
	PassEnv      []string `json:"passEnv,omitempty"`
	TmpSize      string   `json:"tmpSize,omitempty"`
}

type RetrySettings struct {
	Enabled     bool `json:"enabled"`
	MaxRetries  int  `json:"maxRetries"`
	BaseDelayMs int  `json:"baseDelayMs"`
}

type ApprovalSettings struct {
	// BashWhitelist is a list of command prefixes that auto-approve in agent mode
	BashWhitelist []string `json:"bashWhitelist,omitempty"`
	// BashBlacklist is a list of command prefixes that always require approval (even in yolo mode if configured)
	BashBlacklist []string `json:"bashBlacklist,omitempty"`
	// ConfirmBeforeWrite requires user approval before write/edit tools run in agent mode.
	ConfirmBeforeWrite *bool `json:"confirmBeforeWrite,omitempty"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Providers: map[string]*ProviderConfig{
			"anthropic": &ProviderConfig{
				BaseURL: "https://api.anthropic.com",
				APIKey:  "${ANTHROPIC_API_KEY}",
				API:     "anthropic-messages",
				Models: []ModelConfig{
					{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75}, Input: []string{"text", "image"}},
					{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75}, Input: []string{"text", "image"}},
					{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Reasoning: true, ContextWindow: 1000000, MaxTokens: 64000, Cost: &CostConfig{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75}, Input: []string{"text", "image"}},
					{ID: "claude-opus-4-5-20251101", Name: "Claude Opus 4.5", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25}, Input: []string{"text", "image"}},
					{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", Reasoning: true, ContextWindow: 1000000, MaxTokens: 128000, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25}, Input: []string{"text", "image"}},
					{ID: "claude-opus-4-7", Name: "Claude Opus 4.7", Reasoning: true, ContextWindow: 1000000, MaxTokens: 128000, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25}, Input: []string{"text", "image"}},
					{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 1, Output: 5, CacheRead: 0.1, CacheWrite: 1.25}, Input: []string{"text", "image"}},
					{ID: "claude-3-5-sonnet-20241022", Name: "Claude Sonnet 3.5 v2", ContextWindow: 200000, MaxTokens: 8192, Cost: &CostConfig{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75}, Input: []string{"text", "image"}},
					{ID: "claude-3-5-haiku-20241022", Name: "Claude Haiku 3.5", ContextWindow: 200000, MaxTokens: 8192, Cost: &CostConfig{Input: 0.8, Output: 4, CacheRead: 0.08, CacheWrite: 1}, Input: []string{"text", "image"}},
				},
			},
			"deepseek-anthropic": &ProviderConfig{
				BaseURL: "https://api.deepseek.com/anthropic",
				APIKey:  "${DEEPSEEK_API_KEY}",
				API:     "anthropic-messages",
				Models: []ModelConfig{
					{ID: "deepseek-v4-flash", Name: "DeepSeek-V4-Flash", ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 0.5, Output: 2}, Input: []string{"text"}},
					{ID: "deepseek-v4-pro", Name: "DeepSeek-V4-Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 1, Output: 4}, Input: []string{"text"}},
				},
			},
			"deepseek-openai": &ProviderConfig{
				BaseURL: "https://api.deepseek.com",
				APIKey:  "${DEEPSEEK_API_KEY}",
				API:     "openai-chat",
				Models: []ModelConfig{
					{ID: "deepseek-v4-flash", Name: "DeepSeek-V4-Flash", ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 0.5, Output: 2}, Input: []string{"text"}},
					{ID: "deepseek-v4-pro", Name: "DeepSeek-V4-Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 1, Output: 4}, Input: []string{"text"}},
				},
			},
			"openai": &ProviderConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "${OPENAI_API_KEY}",
				API:     "openai-responses",
				Models: []ModelConfig{
					{ID: "gpt-4o", Name: "GPT-4o", ContextWindow: 128000, MaxTokens: 16384, Cost: &CostConfig{Input: 2.5, Output: 10, CacheRead: 1.25}, Input: []string{"text", "image"}},
					{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ContextWindow: 128000, MaxTokens: 16384, Cost: &CostConfig{Input: 0.15, Output: 0.6, CacheRead: 0.075}, Input: []string{"text", "image"}},
					{ID: "gpt-5", Name: "GPT-5", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 1.25, Output: 10, CacheRead: 0.125}, Input: []string{"text", "image"}},
					{ID: "gpt-5-mini", Name: "GPT-5 Mini", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.25, Output: 2, CacheRead: 0.025}, Input: []string{"text", "image"}},
					{ID: "gpt-5-nano", Name: "GPT-5 Nano", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.05, Output: 0.4, CacheRead: 0.005}, Input: []string{"text", "image"}},
					{ID: "gpt-5.1", Name: "GPT-5.1", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 1.25, Output: 10, CacheRead: 0.125}, Input: []string{"text", "image"}},
					{ID: "gpt-5.2", Name: "GPT-5.2", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 1.75, Output: 14, CacheRead: 0.175}, Input: []string{"text", "image"}},
					{ID: "gpt-5.4", Name: "GPT-5.4", Reasoning: true, ContextWindow: 272000, MaxTokens: 128000, Cost: &CostConfig{Input: 2.5, Output: 15, CacheRead: 0.25}, Input: []string{"text", "image"}},
					{ID: "gpt-5.4-mini", Name: "GPT-5.4 Mini", Reasoning: true, ContextWindow: 400000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.75, Output: 4.5, CacheRead: 0.075}, Input: []string{"text", "image"}},
					{ID: "o1", Name: "o1", Reasoning: true, ContextWindow: 200000, MaxTokens: 100000, Cost: &CostConfig{Input: 15, Output: 60, CacheRead: 7.5}, Input: []string{"text", "image"}},
					{ID: "o3-mini", Name: "o3-mini", Reasoning: true, ContextWindow: 200000, MaxTokens: 100000, Cost: &CostConfig{Input: 1.1, Output: 4.4, CacheRead: 0.55}, Input: []string{"text"}},
				},
			},
			"google-gemini": &ProviderConfig{
				BaseURL: "https://generativelanguage.googleapis.com/v1beta/models",
				APIKey:  "${GOOGLE_API_KEY}",
				API:     "google-gemini",
				Models: []ModelConfig{
					{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 1.25, Output: 10, CacheRead: 0.125}, Input: []string{"text", "image"}},
					{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.3, Output: 2.5, CacheRead: 0.03}, Input: []string{"text", "image"}},
					{ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash-Lite", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.1, Output: 0.4, CacheRead: 0.01}, Input: []string{"text", "image"}},
					{ID: "gemini-3-flash-preview", Name: "Gemini 3 Flash Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.5, Output: 3, CacheRead: 0.05}, Input: []string{"text", "image"}},
					{ID: "gemini-3-pro-preview", Name: "Gemini 3 Pro Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 2, Output: 12, CacheRead: 0.2}, Input: []string{"text", "image"}},
					{ID: "gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash Lite", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.25, Output: 1.5, CacheRead: 0.025}, Input: []string{"text", "image"}},
					{ID: "gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 2, Output: 12, CacheRead: 0.2}, Input: []string{"text", "image"}},
					{ID: "gemini-3.5-flash", Name: "Gemini 3.5 Flash", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 1.5, Output: 9, CacheRead: 0.15}, Input: []string{"text", "image"}},
				},
			},
			"google-vertex": &ProviderConfig{
				BaseURL: "https://aiplatform.googleapis.com/v1/publishers/google/models",
				APIKey:  "${GOOGLE_CLOUD_API_KEY}",
				API:     "google-vertex",
				Models: []ModelConfig{
					{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 1.25, Output: 10, CacheRead: 0.125}, Input: []string{"text", "image"}},
					{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.3, Output: 2.5, CacheRead: 0.03}, Input: []string{"text", "image"}},
					{ID: "gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash-Lite", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.1, Output: 0.4, CacheRead: 0.01}, Input: []string{"text", "image"}},
					{ID: "gemini-3-flash-preview", Name: "Gemini 3 Flash Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.5, Output: 3, CacheRead: 0.05}, Input: []string{"text", "image"}},
					{ID: "gemini-3-pro-preview", Name: "Gemini 3 Pro Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 2, Output: 12, CacheRead: 0.2}, Input: []string{"text", "image"}},
					{ID: "gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash Lite", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 0.25, Output: 1.5, CacheRead: 0.025}, Input: []string{"text", "image"}},
					{ID: "gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro Preview", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 2, Output: 12, CacheRead: 0.2}, Input: []string{"text", "image"}},
					{ID: "gemini-3.5-flash", Name: "Gemini 3.5 Flash", Reasoning: true, ContextWindow: 1048576, MaxTokens: 65536, Cost: &CostConfig{Input: 1.5, Output: 9, CacheRead: 0.15}, Input: []string{"text", "image"}},
				},
			},
			"xiaomi":                {BaseURL: "https://api.xiaomimimo.com/v1", APIKey: "${XIAOMI_API_KEY}", API: "openai-chat", ThinkingFormat: "xiaomi", Models: []ModelConfig{{ID: "mimo-v2.5-pro", Name: "MiMo-V2.5-Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.435, Output: 0.87, CacheRead: 0.0036}, Input: []string{"text"}}, {ID: "mimo-v2.5", Name: "MiMo-V2.5", Reasoning: true, ContextWindow: 1000000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.14, Output: 0.28, CacheRead: 0.0028}, Input: []string{"text", "image", "audio", "video"}}, {ID: "mimo-v2-flash", Name: "MiMo-V2-Flash", Reasoning: true, ContextWindow: 256000, MaxTokens: 64000, Cost: &CostConfig{Input: 0.10, Output: 0.30, CacheRead: 0.01}, Input: []string{"text"}}}},
			"openrouter":            {Vendor: "openrouter", BaseURL: "https://openrouter.ai/api/v1", APIKey: "${OPENROUTER_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "z-ai/glm-4.5-air:free", Name: "OpenRouter GLM-4.5-Air Free", ContextWindow: 128000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "openai/gpt-oss-120b:free", Name: "OpenRouter GPT-OSS-120B Free", ContextWindow: 131072, MaxTokens: 8192, Input: []string{"text"}}}},
			"minimax":               {Vendor: "minimax", BaseURL: "https://api.minimax.io/v1", APIKey: "${MINIMAX_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "MiniMax-M3", Name: "MiniMax-M3", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "MiniMax-M2.7", Name: "MiniMax-M2.7", ContextWindow: 204800, MaxTokens: 8192, Input: []string{"text"}}, {ID: "MiniMax-M2.7-highspeed", Name: "MiniMax-M2.7-highspeed", ContextWindow: 204800, MaxTokens: 8192, Input: []string{"text"}}, {ID: "MiniMax-M2.5", Name: "MiniMax-M2.5", ContextWindow: 196608, MaxTokens: 8192, Input: []string{"text"}}, {ID: "MiniMax-M2.5-highspeed", Name: "MiniMax-M2.5-highspeed", ContextWindow: 196608, MaxTokens: 8192, Input: []string{"text"}}}},
			"zai":                   {Vendor: "zai", BaseURL: "https://api.z.ai/api/coding/paas/v4", APIKey: "${ZAI_API_KEY}", API: "openai-chat", ThinkingFormat: "zai", Models: []ModelConfig{{ID: "glm-4.5-air", Name: "GLM-4.5-Air", Reasoning: true, ContextWindow: 131072, MaxTokens: 98304, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-4.7", Name: "GLM-4.7", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5-turbo", Name: "GLM-5-Turbo", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.2", Name: "GLM-5.2", Reasoning: true, ContextWindow: 1000000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5v-turbo", Name: "GLM-5V-Turbo", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"modelscope":            {BaseURL: "https://api-inference.modelscope.cn/v1", APIKey: "${MODELSCOPE_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "deepseek-ai/DeepSeek-V4-Flash", Name: "DeepSeek-V4-Flash", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "Qwen/Qwen3.5-397B-A17B", Name: "Qwen3.5-397B-A17B", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "ZhipuAI/GLM-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}}},
			"alibaba-coding-plan":   {Vendor: "bailian", BaseURL: "https://coding.dashscope.aliyuncs.com/v1", APIKey: "${BAILIAN_CODING_PLAN_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "qwen3.5-plus", Name: "Qwen3.5 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "qwen3.6-plus", Name: "Qwen3.6 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "qwen3.7-plus", Name: "Qwen3.7 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "glm-5", Name: "GLM-5", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}, {ID: "kimi-k2.5", Name: "Kimi-K2.5", Reasoning: true, ContextWindow: 262144, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "MiniMax-M2.5", Name: "MiniMax-M2.5", Reasoning: true, ContextWindow: 196608, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3-coder-plus", Name: "Qwen3 Coder Plus", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3-coder-next", Name: "Qwen3 Coder Next", ContextWindow: 262144, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3-max-2026-01-23", Name: "Qwen3 Max", Reasoning: true, ContextWindow: 262144, MaxTokens: 8192, Input: []string{"text"}}, {ID: "glm-4.7", Name: "GLM-4.7", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}}},
			"alibaba-token-plan":    {Vendor: "bailian", BaseURL: "https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1", APIKey: "${BAILIAN_TOKEN_PLAN_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "qwen3.6-plus", Name: "Qwen3.6 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "qwen3.7-max", Name: "Qwen3.7 Max", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3.6-flash", Name: "Qwen3.6 Flash", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "deepseek-v4-pro", Name: "DeepSeek-V4-Pro", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "deepseek-v4-flash", Name: "DeepSeek-V4-Flash", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "deepseek-v3.2", Name: "DeepSeek-V3.2", ContextWindow: 131072, MaxTokens: 8192, Input: []string{"text"}}, {ID: "kimi-k2.6", Name: "Kimi-K2.6", Reasoning: true, ContextWindow: 262144, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "kimi-k2.5", Name: "Kimi-K2.5", Reasoning: true, ContextWindow: 262144, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}, {ID: "glm-5", Name: "GLM-5", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}, {ID: "MiniMax-M2.5", Name: "MiniMax-M2.5", ContextWindow: 196608, MaxTokens: 8192, Input: []string{"text"}}}},
			"moark":                 {BaseURL: "https://api.moark.com/v1", APIKey: "${MOARK_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "qwen3.6-plus", Name: "Qwen3.6 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3.7-max", Name: "Qwen3.7 Max", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "deepseek-v4-pro", Name: "DeepSeek-V4-Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "deepseek-v4-flash", Name: "DeepSeek-V4-Flash", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}}},
			"alibaba-standard":      {Vendor: "bailian", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", APIKey: "${DASHSCOPE_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "qwen3.6-plus", Name: "Qwen3.6 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3.7-plus", Name: "Qwen3.7 Plus", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "qwen3.7-max", Name: "Qwen3.7 Max", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 202752, MaxTokens: 8192, Input: []string{"text"}}, {ID: "deepseek-v4-pro", Name: "DeepSeek-V4-Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text", "image", "video"}}, {ID: "deepseek-v4-flash", Name: "DeepSeek-V4-Flash", ContextWindow: 1000000, MaxTokens: 8192, Input: []string{"text"}}}},
			"ant-ling":              {BaseURL: "https://api.ant-ling.com/v1", APIKey: "${ANT_LING_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "Ling-2.6-1T", Name: "Ling 2.6 1T", ContextWindow: 262144, MaxTokens: 65536, Cost: &CostConfig{Input: 0.06, Output: 0.25, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Ling-2.6-flash", Name: "Ling 2.6 Flash", ContextWindow: 262144, MaxTokens: 65536, Cost: &CostConfig{Input: 0.01, Output: 0.02, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Ring-2.6-1T", Name: "Ring 2.6 1T", Reasoning: true, ContextWindow: 262144, MaxTokens: 65536, Cost: &CostConfig{Input: 0.06, Output: 0.25, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}}},
			"cerebras":              {BaseURL: "https://api.cerebras.ai/v1", APIKey: "${CEREBRAS_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "gpt-oss-120b", Name: "GPT OSS 120B", Reasoning: true, ContextWindow: 131072, MaxTokens: 40960, Cost: &CostConfig{Input: 0.35, Output: 0.75, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "zai-glm-4.7", Name: "Z.AI GLM-4.7", Reasoning: true, ContextWindow: 131072, MaxTokens: 40960, Cost: &CostConfig{Input: 2.25, Output: 2.75, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}}},
			"groq":                  {BaseURL: "https://api.groq.com/openai/v1", APIKey: "${GROQ_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B", ContextWindow: 131072, MaxTokens: 131072, Cost: &CostConfig{Input: 0.05, Output: 0.08, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B", ContextWindow: 131072, MaxTokens: 32768, Cost: &CostConfig{Input: 0.59, Output: 0.79, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "meta-llama/llama-4-scout-17b-16e-instruct", Name: "Llama 4 Scout 17B 16E", ContextWindow: 131072, MaxTokens: 8192, Cost: &CostConfig{Input: 0.11, Output: 0.34, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "openai/gpt-oss-120b", Name: "GPT OSS 120B", Reasoning: true, ContextWindow: 131072, MaxTokens: 65536, Cost: &CostConfig{Input: 0.15, Output: 0.6, CacheRead: 0.075, CacheWrite: 0}, Input: []string{"text"}}, {ID: "openai/gpt-oss-20b", Name: "GPT OSS 20B", Reasoning: true, ContextWindow: 131072, MaxTokens: 65536, Cost: &CostConfig{Input: 0.075, Output: 0.3, CacheRead: 0.0375, CacheWrite: 0}, Input: []string{"text"}}}},
			"moonshotai":            {BaseURL: "https://api.moonshot.ai/v1", APIKey: "${MOONSHOTAI_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "kimi-k2-0711-preview", Name: "Kimi K2 0711", ContextWindow: 131072, MaxTokens: 16384, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-0905-preview", Name: "Kimi K2 0905", ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-thinking", Name: "Kimi K2 Thinking", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-thinking-turbo", Name: "Kimi K2 Thinking Turbo", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 1.15, Output: 8, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-turbo-preview", Name: "Kimi K2 Turbo", ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 2.4, Output: 10, CacheRead: 0.6, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2.7-code", Name: "Kimi K2.7 Code", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.95, Output: 4, CacheRead: 0.19, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "kimi-k2.7-code-highspeed", Name: "Kimi K2.7 Code HighSpeed", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 1.9, Output: 8, CacheRead: 0.38, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"moonshotai-cn":         {BaseURL: "https://api.moonshot.cn/v1", APIKey: "${MOONSHOTAI_CN_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "kimi-k2-0711-preview", Name: "Kimi K2 0711", ContextWindow: 131072, MaxTokens: 16384, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-0905-preview", Name: "Kimi K2 0905", ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-thinking", Name: "Kimi K2 Thinking", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.6, Output: 2.5, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-thinking-turbo", Name: "Kimi K2 Thinking Turbo", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 1.15, Output: 8, CacheRead: 0.15, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2-turbo-preview", Name: "Kimi K2 Turbo", ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 2.4, Output: 10, CacheRead: 0.6, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2.7-code", Name: "Kimi K2.7 Code", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.95, Output: 4, CacheRead: 0.19, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "kimi-k2.7-code-highspeed", Name: "Kimi K2.7 Code HighSpeed", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 1.9, Output: 8, CacheRead: 0.38, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"nvidia":                {BaseURL: "https://integrate.api.nvidia.com/v1", APIKey: "${NVIDIA_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "meta/llama-3.1-70b-instruct", Name: "Llama 3.1 70b Instruct", ContextWindow: 128000, MaxTokens: 4096, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "meta/llama-3.1-8b-instruct", Name: "Llama 3.1 8B Instruct", ContextWindow: 16000, MaxTokens: 4096, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "meta/llama-3.2-11b-vision-instruct", Name: "Llama 3.2 11b Vision Instruct", ContextWindow: 128000, MaxTokens: 4096, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "meta/llama-3.2-90b-vision-instruct", Name: "Llama-3.2-90B-Vision-Instruct", ContextWindow: 128000, MaxTokens: 8192, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "meta/llama-3.3-70b-instruct", Name: "Llama 3.3 70b Instruct", ContextWindow: 128000, MaxTokens: 4096, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}}},
			"together":              {BaseURL: "https://api.together.ai/v1", APIKey: "${TOGETHER_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "MiniMaxAI/MiniMax-M2.7", Name: "MiniMax-M2.7", Reasoning: true, ContextWindow: 202752, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.06, CacheWrite: 0}, Input: []string{"text"}}, {ID: "MiniMaxAI/MiniMax-M3", Name: "MiniMax-M3", Reasoning: true, ContextWindow: 524288, MaxTokens: 250000, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.06, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "Qwen/Qwen2.5-7B-Instruct-Turbo", Name: "Qwen 2.5 7B Instruct Turbo", ContextWindow: 32768, MaxTokens: 32768, Cost: &CostConfig{Input: 0.3, Output: 0.3, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Qwen/Qwen3-235B-A22B-Instruct-2507-tput", Name: "Qwen3 235B A22B Instruct 2507 FP8", ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.2, Output: 0.6, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Qwen/Qwen3.5-397B-A17B", Name: "Qwen3.5 397B A17B", Reasoning: true, ContextWindow: 262144, MaxTokens: 130000, Cost: &CostConfig{Input: 0.6, Output: 3.6, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"xai":                   {BaseURL: "https://api.x.ai/v1", APIKey: "${XAI_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "grok-3", Name: "Grok 3", ContextWindow: 131072, MaxTokens: 8192, Cost: &CostConfig{Input: 3, Output: 15, CacheRead: 0.75, CacheWrite: 0}, Input: []string{"text"}}, {ID: "grok-3-fast", Name: "Grok 3 Fast", ContextWindow: 131072, MaxTokens: 8192, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 1.25, CacheWrite: 0}, Input: []string{"text"}}, {ID: "grok-4.20-0309-non-reasoning", Name: "Grok 4.20 (Non-Reasoning)", ContextWindow: 1000000, MaxTokens: 30000, Cost: &CostConfig{Input: 1.25, Output: 2.5, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "grok-4.20-0309-reasoning", Name: "Grok 4.20 (Reasoning)", Reasoning: true, ContextWindow: 1000000, MaxTokens: 30000, Cost: &CostConfig{Input: 1.25, Output: 2.5, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "grok-4.3", Name: "Grok 4.3", Reasoning: true, ContextWindow: 1000000, MaxTokens: 30000, Cost: &CostConfig{Input: 1.25, Output: 2.5, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"kimi-coding":           {BaseURL: "https://api.kimi.com/coding", APIKey: "${KIMI_CODING_API_KEY}", API: "anthropic-messages", Headers: map[string]string{"User-Agent": "KimiCLI/1.5"}, Models: []ModelConfig{{ID: "k2p7", Name: "Kimi K2.7 Code", Reasoning: true, ContextWindow: 262144, MaxTokens: 32768, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "kimi-for-coding", Name: "Kimi For Coding", Reasoning: true, ContextWindow: 262144, MaxTokens: 32768, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "kimi-k2-thinking", Name: "Kimi K2 Thinking", Reasoning: true, ContextWindow: 262144, MaxTokens: 32768, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}}},
			"fireworks":             {BaseURL: "https://api.fireworks.ai/inference", APIKey: "${FIREWORKS_API_KEY}", API: "anthropic-messages", Models: []ModelConfig{{ID: "accounts/fireworks/models/deepseek-v4-flash", Name: "DeepSeek V4 Flash", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 0.14, Output: 0.28, CacheRead: 0.03, CacheWrite: 0}, Input: []string{"text"}}, {ID: "accounts/fireworks/models/deepseek-v4-pro", Name: "DeepSeek V4 Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 1.74, Output: 3.48, CacheRead: 0.145, CacheWrite: 0}, Input: []string{"text"}}, {ID: "accounts/fireworks/models/glm-5p1", Name: "GLM 5.1", Reasoning: true, ContextWindow: 202800, MaxTokens: 131072, Cost: &CostConfig{Input: 1.4, Output: 4.4, CacheRead: 0.26, CacheWrite: 0}, Input: []string{"text"}}, {ID: "accounts/fireworks/models/kimi-k2p7-code", Name: "Kimi K2.7 Code", Reasoning: true, ContextWindow: 262000, MaxTokens: 262000, Cost: &CostConfig{Input: 0.95, Output: 4, CacheRead: 0.19, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "accounts/fireworks/routers/kimi-k2p7-code-fast", Name: "Kimi K2.7 Code Fast", Reasoning: true, ContextWindow: 262000, MaxTokens: 262000, Cost: &CostConfig{Input: 2, Output: 8, CacheRead: 0.38, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "accounts/fireworks/models/gpt-oss-120b", Name: "GPT OSS 120B", Reasoning: true, ContextWindow: 131072, MaxTokens: 32768, Cost: &CostConfig{Input: 0.15, Output: 0.6, CacheRead: 0.01, CacheWrite: 0}, Input: []string{"text"}}, {ID: "accounts/fireworks/models/gpt-oss-20b", Name: "GPT OSS 20B", Reasoning: true, ContextWindow: 131072, MaxTokens: 32768, Cost: &CostConfig{Input: 0.07, Output: 0.3, CacheRead: 0.035, CacheWrite: 0}, Input: []string{"text"}}}},
			"huggingface":           {BaseURL: "https://router.huggingface.co/v1", APIKey: "${HUGGINGFACE_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "MiniMaxAI/MiniMax-M2.1", Name: "MiniMax-M2.1", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "MiniMaxAI/MiniMax-M2.5", Name: "MiniMax-M2.5", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.03, CacheWrite: 0}, Input: []string{"text"}}, {ID: "MiniMaxAI/MiniMax-M2.7", Name: "MiniMax-M2.7", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.06, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Qwen/Qwen3-235B-A22B-Thinking-2507", Name: "Qwen3-235B-A22B-Thinking-2507", Reasoning: true, ContextWindow: 262144, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 3, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "Qwen/Qwen3-Coder-480B-A35B-Instruct", Name: "Qwen3-Coder-480B-A35B-Instruct", ContextWindow: 262144, MaxTokens: 66536, Cost: &CostConfig{Input: 2, Output: 2, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}}},
			"xiaomi-token-plan-ams": {BaseURL: "https://token-plan-ams.xiaomimimo.com/v1", APIKey: "${XIAOMI_TOKEN_PLAN_AMS_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "mimo-v2-omni", Name: "MiMo-V2-Omni", Reasoning: true, ContextWindow: 262144, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2-pro", Name: "MiMo-V2-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5", Name: "MiMo-V2.5", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2.5-pro", Name: "MiMo-V2.5-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5-pro-ultraspeed", Name: "MiMo-V2.5-Pro-UltraSpeed", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1.305, Output: 2.61, CacheRead: 0.0108, CacheWrite: 0}, Input: []string{"text"}}}},
			"xiaomi-token-plan-cn":  {BaseURL: "https://token-plan-cn.xiaomimimo.com/v1", APIKey: "${XIAOMI_TOKEN_PLAN_CN_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "mimo-v2-omni", Name: "MiMo-V2-Omni", Reasoning: true, ContextWindow: 262144, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2-pro", Name: "MiMo-V2-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5", Name: "MiMo-V2.5", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2.5-pro", Name: "MiMo-V2.5-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5-pro-ultraspeed", Name: "MiMo-V2.5-Pro-UltraSpeed", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1.305, Output: 2.61, CacheRead: 0.0108, CacheWrite: 0}, Input: []string{"text"}}}},
			"xiaomi-token-plan-sgp": {BaseURL: "https://token-plan-sgp.xiaomimimo.com/v1", APIKey: "${XIAOMI_TOKEN_PLAN_SGP_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "mimo-v2-omni", Name: "MiMo-V2-Omni", Reasoning: true, ContextWindow: 262144, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2-pro", Name: "MiMo-V2-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5", Name: "MiMo-V2.5", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 0.4, Output: 2, CacheRead: 0.08, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "mimo-v2.5-pro", Name: "MiMo-V2.5-Pro", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1, Output: 3, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "mimo-v2.5-pro-ultraspeed", Name: "MiMo-V2.5-Pro-UltraSpeed", Reasoning: true, ContextWindow: 1048576, MaxTokens: 131072, Cost: &CostConfig{Input: 1.305, Output: 2.61, CacheRead: 0.0108, CacheWrite: 0}, Input: []string{"text"}}}},
			"zai-coding-cn":         {Vendor: "zai", BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4", APIKey: "${ZAI_CODING_CN_API_KEY}", API: "openai-chat", ThinkingFormat: "zai", Models: []ModelConfig{{ID: "glm-4.5-air", Name: "GLM-4.5-Air", Reasoning: true, ContextWindow: 131072, MaxTokens: 98304, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-4.7", Name: "GLM-4.7", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5-turbo", Name: "GLM-5-Turbo", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.2", Name: "GLM-5.2", Reasoning: true, ContextWindow: 1000000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5v-turbo", Name: "GLM-5V-Turbo", Reasoning: true, ContextWindow: 200000, MaxTokens: 131072, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"minimax-anthropic":     {BaseURL: "https://api.minimax.io/anthropic", APIKey: "${MINIMAX_ANTHROPIC_API_KEY}", API: "anthropic-messages", Models: []ModelConfig{{ID: "MiniMax-M2.7", Name: "MiniMax-M2.7", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.06, CacheWrite: 0.375}, Input: []string{"text"}}, {ID: "MiniMax-M2.7-highspeed", Name: "MiniMax-M2.7-highspeed", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.6, Output: 2.4, CacheRead: 0.06, CacheWrite: 0.375}, Input: []string{"text"}}, {ID: "MiniMax-M3", Name: "MiniMax-M3", Reasoning: true, ContextWindow: 512000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.6, Output: 2.4, CacheRead: 0.12, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"minimax-cn-anthropic":  {BaseURL: "https://api.minimaxi.com/anthropic", APIKey: "${MINIMAX_CN_ANTHROPIC_API_KEY}", API: "anthropic-messages", Models: []ModelConfig{{ID: "MiniMax-M2.7", Name: "MiniMax-M2.7", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.3, Output: 1.2, CacheRead: 0.06, CacheWrite: 0.375}, Input: []string{"text"}}, {ID: "MiniMax-M2.7-highspeed", Name: "MiniMax-M2.7-highspeed", Reasoning: true, ContextWindow: 204800, MaxTokens: 131072, Cost: &CostConfig{Input: 0.6, Output: 2.4, CacheRead: 0.06, CacheWrite: 0.375}, Input: []string{"text"}}, {ID: "MiniMax-M3", Name: "MiniMax-M3", Reasoning: true, ContextWindow: 512000, MaxTokens: 128000, Cost: &CostConfig{Input: 0.6, Output: 2.4, CacheRead: 0.12, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"opencode":              {BaseURL: "https://opencode.ai/zen/v1", APIKey: "${OPENCODE_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "big-pickle", Name: "Big Pickle", Reasoning: true, ContextWindow: 200000, MaxTokens: 32000, Cost: &CostConfig{Input: 0, Output: 0, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 1, Output: 5, CacheRead: 0.1, CacheWrite: 1.25}, Input: []string{"text", "image"}}, {ID: "claude-opus-4-1", Name: "Claude Opus 4.1", Reasoning: true, ContextWindow: 200000, MaxTokens: 32000, Cost: &CostConfig{Input: 15, Output: 75, CacheRead: 1.5, CacheWrite: 18.75}, Input: []string{"text", "image"}}, {ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Reasoning: true, ContextWindow: 200000, MaxTokens: 64000, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25}, Input: []string{"text", "image"}}, {ID: "claude-opus-4-6", Name: "Claude Opus 4.6", Reasoning: true, ContextWindow: 1000000, MaxTokens: 128000, Cost: &CostConfig{Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25}, Input: []string{"text", "image"}}}},
			"opencode-go":           {BaseURL: "https://opencode.ai/zen/go/v1", APIKey: "${OPENCODE_GO_API_KEY}", API: "openai-chat", Models: []ModelConfig{{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 0.14, Output: 0.28, CacheRead: 0.0028, CacheWrite: 0}, Input: []string{"text"}}, {ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro", Reasoning: true, ContextWindow: 1000000, MaxTokens: 384000, Cost: &CostConfig{Input: 1.74, Output: 3.48, CacheRead: 0.0145, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5", Name: "GLM-5", Reasoning: true, ContextWindow: 202752, MaxTokens: 32768, Cost: &CostConfig{Input: 1, Output: 3.2, CacheRead: 0.2, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.1", Name: "GLM-5.1", Reasoning: true, ContextWindow: 202752, MaxTokens: 32768, Cost: &CostConfig{Input: 1.4, Output: 4.4, CacheRead: 0.26, CacheWrite: 0}, Input: []string{"text"}}, {ID: "glm-5.2", Name: "GLM-5.2", Reasoning: true, ContextWindow: 1000000, MaxTokens: 262144, Cost: &CostConfig{Input: 1.4, Output: 4.4, CacheRead: 0.26, CacheWrite: 0}, Input: []string{"text"}}, {ID: "kimi-k2.6", Name: "Kimi K2.6", Reasoning: true, ContextWindow: 262144, MaxTokens: 65536, Cost: &CostConfig{Input: 0.95, Output: 4, CacheRead: 0.16, CacheWrite: 0}, Input: []string{"text", "image"}}, {ID: "kimi-k2.7-code", Name: "Kimi K2.7 Code", Reasoning: true, ContextWindow: 262144, MaxTokens: 262144, Cost: &CostConfig{Input: 0.95, Output: 4, CacheRead: 0.19, CacheWrite: 0}, Input: []string{"text", "image"}}}},
			"vercel-ai-gateway":     {BaseURL: "https://ai-gateway.vercel.sh", APIKey: "${VERCEL_AI_GATEWAY_API_KEY}", API: "anthropic-messages", Models: []ModelConfig{{ID: "alibaba/qwen-3-14b", Name: "Qwen3-14B", Reasoning: true, ContextWindow: 40960, MaxTokens: 16384, Cost: &CostConfig{Input: 0.12, Output: 0.24, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "alibaba/qwen-3-235b", Name: "Qwen3 235B A22B", Reasoning: true, ContextWindow: 262144, MaxTokens: 16384, Cost: &CostConfig{Input: 0.22, Output: 0.88, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "alibaba/qwen-3-30b", Name: "Qwen3-30B-A3B", Reasoning: true, ContextWindow: 40960, MaxTokens: 16384, Cost: &CostConfig{Input: 0.12, Output: 0.5, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "alibaba/qwen-3-32b", Name: "Qwen 3 32B", Reasoning: true, ContextWindow: 128000, MaxTokens: 8192, Cost: &CostConfig{Input: 0.16, Output: 0.64, CacheRead: 0, CacheWrite: 0}, Input: []string{"text"}}, {ID: "alibaba/qwen-3.6-max-preview", Name: "Qwen 3.6 Max Preview", Reasoning: true, ContextWindow: 240000, MaxTokens: 64000, Cost: &CostConfig{Input: 1.3, Output: 7.8, CacheRead: 0.26, CacheWrite: 1.625}, Input: []string{"text"}}}},
		},
		DefaultProvider:      "deepseek-openai",
		DefaultModel:         "deepseek-v4-flash",
		DefaultThinkingLevel: "medium",
		DefaultMode:          "agent",
		EnablePlanTool:       boolPtr(true),
		WebSearch:            WebSearchSettings{Enabled: boolPtr(false), Provider: "openai", ProviderType: "responses"},
		ContextFiles:         ContextFilesSettings{Enabled: true},
		SkillsDir:            platform.SkillsDir(),
		Compaction:           CompactionSettings{Enabled: true, ReserveTokens: 16384, KeepRecentTokens: 20000},
		Sandbox: SandboxSettings{
			Enabled:     false,
			Level:       "none",
			AllowedRead: platform.SandboxPaths(),
			DeniedPaths: platform.DeniedPaths(),
			PassEnv:     platform.DefaultEnvVars(),
			TmpSize:     "100m",
		},
		SessionDir: platform.SessionDir(),
		Theme:      "dark",
		Retry:      RetrySettings{Enabled: true, MaxRetries: 3, BaseDelayMs: 2000},
		Approval: ApprovalSettings{
			BashWhitelist:      []string{"go ", "make ", "git ", "npm ", "yarn ", "node ", "python ", "pip "},
			ConfirmBeforeWrite: boolPtr(true),
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func ConfigDir() string {
	return platform.ConfigDir()
}

func GlobalSettingsPath() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

func ProjectSettingsPath() string {
	return filepath.Join(".vibe", "settings.json")
}

func LoadSettings() (*Settings, error) {
	s, _, err := LoadSettingsWithMeta()
	return s, err
}

// LoadMeta describes side effects and paths from settings loading.
type LoadMeta struct {
	CreatedGlobalConfig bool
	GlobalSettingsPath  string
}

// LoadSettingsWithMeta loads settings and reports whether the global settings
// file was created during this call. The loaded schema is the same as LoadSettings.
func LoadSettingsWithMeta() (*Settings, LoadMeta, error) {
	s := DefaultSettings()
	meta := LoadMeta{GlobalSettingsPath: GlobalSettingsPath()}

	created, err := ensureConfigExists(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create config: %v\n", err)
	} else {
		meta.CreatedGlobalConfig = created
	}

	globalPath := GlobalSettingsPath()
	if Verbose {
		fmt.Fprintf(os.Stderr, "[config] Loading global settings: %s\n", globalPath)
	}
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, meta, fmt.Errorf("parse global settings: %w", err)
		}
		if Verbose {
			fmt.Fprintf(os.Stderr, "[config] Loaded global settings\n")
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not read global settings %s: %v\n", globalPath, err)
	} else if Verbose {
		fmt.Fprintf(os.Stderr, "[config] Global settings not found: %s\n", globalPath)
	}

	projectPath := ProjectSettingsPath()
	if Verbose {
		fmt.Fprintf(os.Stderr, "[config] Loading project settings: %s\n", projectPath)
	}
	if data, err := os.ReadFile(projectPath); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, meta, fmt.Errorf("parse project settings: %w", err)
		}
		if Verbose {
			fmt.Fprintf(os.Stderr, "[config] Loaded project settings\n")
		}
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Warning: could not read project settings %s: %v\n", projectPath, err)
	} else if Verbose {
		fmt.Fprintf(os.Stderr, "[config] Project settings not found: %s\n", projectPath)
		// Detect common typo: .vibe/setting.json (singular)
		if _, err2 := os.Stat(".vibe/setting.json"); err2 == nil {
			fmt.Fprintf(os.Stderr, "[config] Found .vibe/setting.json (singular) — expected .vibe/settings.json (plural). Please rename the file.\n")
		}
	}

	if v := os.Getenv("VIBECODING_PROVIDER"); v != "" {
		s.DefaultProvider = v
	}
	if v := os.Getenv("VIBECODING_MODEL"); v != "" {
		s.DefaultModel = v
	}
	if v := os.Getenv("VIBECODING_MODE"); v != "" {
		s.DefaultMode = v
	}
	if v := os.Getenv("VIBECODING_THINKING"); v != "" {
		s.DefaultThinkingLevel = v
	}

	return s, meta, nil
}

func ensureConfigExists(defaults *Settings) (bool, error) {
	configDir := ConfigDir()
	settingsPath := GlobalSettingsPath()

	if _, err := os.Stat(settingsPath); err == nil {
		return false, nil
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return false, fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(defaults, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal default settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0600); err != nil {
		return false, fmt.Errorf("write settings file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created default config: %s\n", settingsPath)
	return true, nil
}

// LoadGlobalSettingsOrDefault loads only the global settings file over defaults.
// It intentionally does not apply project settings or environment overrides, so
// callers that need a full runnable global config can avoid persisting runtime state.
func LoadGlobalSettingsOrDefault() (*Settings, error) {
	s := DefaultSettings()
	globalPath := GlobalSettingsPath()
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("parse global settings: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read global settings %s: %w", globalPath, err)
	}
	return s, nil
}

// LoadGlobalSettingsSparse loads only fields explicitly present in the global
// settings file. If the file does not exist, it returns an empty Settings.
// Use this for patch-style writes so defaults are not expanded into settings.json.
func LoadGlobalSettingsSparse() (*Settings, error) {
	s := &Settings{}
	globalPath := GlobalSettingsPath()
	if data, err := os.ReadFile(globalPath); err == nil {
		if err := json.Unmarshal(data, s); err != nil {
			return nil, fmt.Errorf("parse global settings: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read global settings %s: %w", globalPath, err)
	}
	if s.Providers == nil {
		s.Providers = map[string]*ProviderConfig{}
	}
	return s, nil
}

// SaveGlobalSettings writes settings.json atomically with private permissions.
func SaveGlobalSettings(s *Settings) error {
	if s == nil {
		return fmt.Errorf("settings is nil")
	}
	configDir := ConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	settingsPath := GlobalSettingsPath()
	tmp, err := os.CreateTemp(configDir, "settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp settings: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp settings: %w", err)
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp settings: %w", err)
	}
	if err := os.Rename(tmpName, settingsPath); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}

func (s *Settings) ResolveKey(providerName string) string {
	// 1. Use apiKey from provider config (supports ${VAR} env references)
	if pc, ok := s.Providers[providerName]; ok && pc != nil && pc.APIKey != "" {
		return resolveKeyValue(pc.APIKey)
	}
	// 2. Fallback: derive env var from provider name, e.g. "deepseek-openai" → "DEEPSEEK_OPENAI_API_KEY"
	envVar := providerToEnvVar(providerName)
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return ""
}

// ResolveProviderHeaders resolves configured per-provider HTTP header values.
// Header values use the same env-var and shell-command resolution rules as apiKey.
func (s *Settings) ResolveProviderHeaders(providerName string) map[string]string {
	if s == nil {
		return nil
	}
	pc := s.GetProviderConfig(providerName)
	if pc == nil || len(pc.Headers) == 0 {
		return nil
	}
	headers := make(map[string]string, len(pc.Headers))
	for name, value := range pc.Headers {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		headers[name] = resolveKeyValue(value)
	}
	return headers
}

// providerToEnvVar converts a provider name to a conventional environment variable name.
// e.g. "deepseek-openai" → "DEEPSEEK_OPENAI_API_KEY", "my-provider" → "MY_PROVIDER_API_KEY".
func providerToEnvVar(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_API_KEY"
}

func resolveKeyValue(key string) string {
	if strings.HasPrefix(key, "!") {
		if os.Getenv("VIBECODING_ALLOW_SHELL_CONFIG") != "1" {
			return key
		}
		return resolveShellCommand(key[1:])
	}

	// Handle ${VAR} syntax: look up the variable name inside ${}
	envName := key
	if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
		envName = key[2 : len(key)-1]
	}

	if !strings.Contains(envName, " ") {
		if v := os.Getenv(envName); v != "" {
			return v
		}
	}

	return key
}

func (s *Settings) GetProviderConfig(name string) *ProviderConfig {
	return s.Providers[name]
}

func (s *Settings) GetModelConfig(providerName, modelID string) *ModelConfig {
	pc := s.GetProviderConfig(providerName)
	if pc == nil {
		return nil
	}
	for _, m := range pc.Models {
		if m.ID == modelID {
			return &m
		}
	}
	return nil
}

func resolveShellCommand(cmd string) string {
	if cmd == "" {
		return ""
	}
	var out []byte
	var err error
	if platform.IsWindows() {
		out, err = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	} else {
		out, err = exec.Command("sh", "-c", cmd).Output()
	}
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (s *Settings) GetShell() string {
	if s.ShellPath != "" {
		return s.ShellPath
	}
	return platform.DefaultShell()
}

func (s *Settings) GetSessionDir() string {
	if s.SessionDir != "" {
		if strings.HasPrefix(s.SessionDir, "~") {
			return platform.ExpandHome(s.SessionDir)
		}
		return s.SessionDir
	}
	return platform.SessionDir()
}

func (s *Settings) GetGlobalSkillsDir() string {
	if s.SkillsDir != "" {
		if strings.HasPrefix(s.SkillsDir, "~") {
			return platform.ExpandHome(s.SkillsDir)
		}
		return s.SkillsDir
	}
	return platform.SkillsDir()
}

func (s *Settings) IsPlanToolEnabled() bool {
	if s.EnablePlanTool == nil {
		return true
	}
	return *s.EnablePlanTool
}

func (s *Settings) IsWebSearchEnabled() bool {
	if s == nil || s.WebSearch.Enabled == nil {
		return false
	}
	return *s.WebSearch.Enabled
}

func mergeWebSearchSettings(base, override WebSearchSettings) WebSearchSettings {
	if override.Enabled != nil {
		base.Enabled = boolPtr(*override.Enabled)
	}
	if override.Provider != "" {
		base.Provider = override.Provider
		if override.ProviderType == "" {
			base.ProviderType = ""
		}
	}
	if override.ProviderType != "" {
		base.ProviderType = override.ProviderType
	}
	if override.Model != "" {
		base.Model = override.Model
	}
	return normalizeWebSearchSettings(base)
}

func normalizeWebSearchSettings(cfg WebSearchSettings) WebSearchSettings {
	if cfg.Enabled == nil {
		cfg.Enabled = boolPtr(false)
	}
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	if cfg.ProviderType == "" {
		switch cfg.Provider {
		case "anthropic":
			cfg.ProviderType = "messages"
		default:
			cfg.ProviderType = "responses"
		}
	}
	return cfg
}
