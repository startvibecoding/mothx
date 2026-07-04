package agent

import (
	"testing"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/provider"
)

func TestResolveMaxTokensPrefersSettingsOverride(t *testing.T) {
	settings := config.DefaultSettings()
	settings.MaxOutputTokens = 1234
	model := &provider.Model{ID: "m", MaxTokens: 64000}

	if got := ResolveMaxTokens(settings, model); got != 1234 {
		t.Fatalf("ResolveMaxTokens = %d, want 1234", got)
	}
}

func TestResolveMaxTokensLeavesUnsetWhenOnlyModelHasMaxTokens(t *testing.T) {
	settings := config.DefaultSettings()
	settings.MaxOutputTokens = 0
	model := &provider.Model{ID: "m", ContextWindow: 128000, MaxTokens: 64000}

	if got := ResolveMaxTokens(settings, model); got != 0 {
		t.Fatalf("ResolveMaxTokens = %d, want 0", got)
	}
}

func TestResolveMaxTokensUsesUserSetModelMaxTokens(t *testing.T) {
	settings := config.DefaultSettings()
	settings.MaxOutputTokens = 0
	model := &provider.Model{ID: "m", ContextWindow: 128000, MaxTokens: 64000, MaxTokensSet: true}

	if got := ResolveMaxTokens(settings, model); got != 64000 {
		t.Fatalf("ResolveMaxTokens = %d, want 64000", got)
	}
}

func TestResolveMaxTokensKeepsSettingsOverrideEvenWhenLarge(t *testing.T) {
	settings := config.DefaultSettings()
	settings.MaxOutputTokens = 262144
	model := &provider.Model{ID: "m", ContextWindow: 262144, MaxTokens: 262144}

	if got := ResolveMaxTokens(settings, model); got != 262144 {
		t.Fatalf("ResolveMaxTokens = %d, want 262144", got)
	}
}

func TestResolveMaxTokensValuePrefersExplicit(t *testing.T) {
	model := &provider.Model{ID: "m", MaxTokens: 64000}

	if got := ResolveMaxTokensValue(4096, model); got != 4096 {
		t.Fatalf("ResolveMaxTokensValue = %d, want 4096", got)
	}
}

func TestResolveMaxTokensReturnsZeroWhenUnknown(t *testing.T) {
	if got := ResolveMaxTokens(nil, nil); got != 0 {
		t.Fatalf("ResolveMaxTokens = %d, want 0", got)
	}
}

func TestClampMaxTokensToContext(t *testing.T) {
	if got := clampMaxTokensToContext(10000, 12000, 3000); got != 9000 {
		t.Fatalf("clampMaxTokensToContext = %d, want 9000", got)
	}
}

func TestClampMaxTokensToContextKeepsValueWhenItFits(t *testing.T) {
	if got := clampMaxTokensToContext(4000, 12000, 3000); got != 4000 {
		t.Fatalf("clampMaxTokensToContext = %d, want 4000", got)
	}
}

func TestClampMaxTokensToContextKeepsZeroFallback(t *testing.T) {
	if got := clampMaxTokensToContext(0, 12000, 3000); got != 0 {
		t.Fatalf("clampMaxTokensToContext = %d, want 0", got)
	}
}
