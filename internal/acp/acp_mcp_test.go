package acp

import (
	"encoding/json"
	"testing"
)

func TestExtractSamplingInput(t *testing.T) {
	raw := json.RawMessage(`{
		"maxTokens": 512,
		"messages": [
			{"role":"system","content":"you are concise"},
			{"role":"user","content":"hello"},
			{"role":"user","content":[{"type":"text","text":"world"}]}
		]
	}`)
	prompt, systemPrompt, maxTokens := extractSamplingInput(raw)
	if prompt != "hello\nworld" {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
	if systemPrompt != "you are concise" {
		t.Fatalf("unexpected system prompt: %q", systemPrompt)
	}
	if maxTokens != 512 {
		t.Fatalf("unexpected maxTokens: %d", maxTokens)
	}
}

func TestParseJSONRawToMap(t *testing.T) {
	raw := json.RawMessage(`{"a":1}`)
	m := parseJSONRawToMap(raw)
	if m == nil {
		t.Fatal("expected map, got nil")
	}
	if _, ok := m["a"]; !ok {
		t.Fatalf("missing key a: %#v", m)
	}
}
