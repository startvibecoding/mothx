package config

import (
	"encoding/json"
	"testing"
)

func TestProjectUnmarshalSupportsFalseAndZeroOverrides(t *testing.T) {
	s := DefaultSettings()
	data := []byte(`{
		"maxContextTokens": 0,
		"maxOutputTokens": 0,
		"webSearch": {"model": "search-model"},
		"contextFiles": {"enabled": false},
		"compaction": {"enabled": false, "reserveTokens": 0, "keepRecentTokens": 0},
		"retry": {"enabled": false, "maxRetries": 0, "baseDelayMs": 0}
	}`)
	if err := json.Unmarshal(data, s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.MaxContextTokens != 0 || s.MaxOutputTokens != 0 {
		t.Fatalf("max token zero overrides not applied: %d/%d", s.MaxContextTokens, s.MaxOutputTokens)
	}
	if s.WebSearch.Model != "search-model" {
		t.Fatalf("webSearch.model = %q, want search-model", s.WebSearch.Model)
	}
	if s.ContextFiles.Enabled {
		t.Fatal("contextFiles.enabled=false override not applied")
	}
	if s.Compaction.Enabled || s.Compaction.ReserveTokens != 0 || s.Compaction.KeepRecentTokens != 0 {
		t.Fatalf("compaction zero/false overrides not applied: %#v", s.Compaction)
	}
	if s.Retry.Enabled || s.Retry.MaxRetries != 0 || s.Retry.BaseDelayMs != 0 {
		t.Fatalf("retry zero/false overrides not applied: %#v", s.Retry)
	}
}
