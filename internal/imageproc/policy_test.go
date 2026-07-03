package imageproc

import "testing"

func TestInferFamilyFromDefaultVisionModelIDs(t *testing.T) {
	tests := []struct {
		name string
		hint Hint
		want Family
	}{
		{
			name: "doubao seed code",
			hint: Hint{ProviderID: "volcengine-agentplan", ModelID: "doubao-seed-2-0-code"},
			want: FamilyDoubaoSeed,
		},
		{
			name: "seed shorthand",
			hint: Hint{ProviderID: "volcengine-agentplan", ModelID: "seed-2-pro"},
			want: FamilyDoubaoSeed,
		},
		{
			name: "qwen plus",
			hint: Hint{ProviderID: "alibaba-standard", ModelID: "qwen3.7-plus"},
			want: FamilyQwen,
		},
		{
			name: "qwen routed",
			hint: Hint{ProviderID: "openrouter", ModelID: "alibaba/qwen3.6-plus"},
			want: FamilyQwen,
		},
		{
			name: "kimi coding short id",
			hint: Hint{ProviderID: "kimi-coding", API: "anthropic-messages", ModelID: "k2p7"},
			want: FamilyKimi,
		},
		{
			name: "minimax over anthropic api",
			hint: Hint{ProviderID: "minimax-anthropic", API: "anthropic-messages", ModelID: "MiniMax-M3"},
			want: FamilyMiniMax,
		},
		{
			name: "bedrock claude",
			hint: Hint{ProviderID: "amazon-bedrock", ModelID: "anthropic.claude-sonnet-4-5-20250929-v1:0"},
			want: FamilyAnthropicBedrock,
		},
		{
			name: "amazon nova",
			hint: Hint{ProviderID: "amazon-bedrock", ModelID: "amazon.nova-pro-v1:0"},
			want: FamilyAmazonNova,
		},
		{
			name: "gateway deepseek vision",
			hint: Hint{ProviderID: "alibaba-standard", ModelID: "deepseek-v4-pro"},
			want: FamilyDeepSeekGatewayVision,
		},
		{
			name: "direct deepseek remains generic",
			hint: Hint{ProviderID: "deepseek-openai", ModelID: "deepseek-v4-flash"},
			want: FamilyGeneric,
		},
		{
			name: "llama vision",
			hint: Hint{ProviderID: "cloudflare-workers-ai", ModelID: "@cf/meta/llama-4-scout-17b-16e-instruct"},
			want: FamilyLlamaVision,
		},
		{
			name: "gemma vision",
			hint: Hint{ProviderID: "google-gemini", ModelID: "gemma-4-26b-a4b-it"},
			want: FamilyGemmaVision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferFamily(tt.hint); got != tt.want {
				t.Fatalf("InferFamily() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPolicyForHintAppliesProviderLimits(t *testing.T) {
	bedrock := PolicyForHint(Hint{
		ProviderID: "amazon-bedrock",
		ModelID:    "anthropic.claude-sonnet-4-5-20250929-v1:0",
	}, ModeDetail)
	if bedrock.MaxFileBytes != 4<<20 {
		t.Fatalf("bedrock MaxFileBytes = %d, want %d", bedrock.MaxFileBytes, 4<<20)
	}
	if bedrock.MaxOutputBytes != 3<<20 {
		t.Fatalf("bedrock MaxOutputBytes = %d, want %d", bedrock.MaxOutputBytes, 3<<20)
	}

	openai := PolicyForHint(Hint{ProviderID: "openai", ModelID: "gpt-4o"}, ModeAuto)
	if openai.MaxFileBytes != 20<<20 {
		t.Fatalf("openai MaxFileBytes = %d, want %d", openai.MaxFileBytes, 20<<20)
	}

	qwen := PolicyForHint(Hint{ProviderID: "alibaba-standard", ModelID: "qwen3.7-plus"}, ModeDetail)
	if qwen.MaxLongEdge != 2560 {
		t.Fatalf("qwen detail MaxLongEdge = %d, want %d", qwen.MaxLongEdge, 2560)
	}

	groq := PolicyForHint(Hint{
		ProviderID: "groq",
		BaseURL:    "https://api.groq.com/openai/v1",
		ModelID:    "meta-llama/llama-4-scout-17b-16e-instruct",
	}, ModeDetail)
	if groq.MaxOutputBytes != 3<<20 {
		t.Fatalf("groq MaxOutputBytes = %d, want %d", groq.MaxOutputBytes, 3<<20)
	}
}
