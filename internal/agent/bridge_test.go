package agent

import (
	"testing"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/provider"
)

func TestChatParamsBridgePreservesModelID(t *testing.T) {
	pub := agentpkg.ChatParams{
		ModelID:      "kimi-2.5",
		SystemPrompt: "test",
		MaxTokens:    1234,
	}

	internal := ChatParamsFromPublic(pub)
	if internal.ModelID != "kimi-2.5" {
		t.Fatalf("internal ModelID = %q, want kimi-2.5", internal.ModelID)
	}

	back := ChatParamsToPublic(provider.ChatParams{
		ModelID:      "kimi-2.5",
		SystemPrompt: "test",
		MaxTokens:    1234,
	})
	if back.ModelID != "kimi-2.5" {
		t.Fatalf("public ModelID = %q, want kimi-2.5", back.ModelID)
	}
}
