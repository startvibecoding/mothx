package acp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExtractSamplingInput(t *testing.T) {
	raw := json.RawMessage(`{"maxTokens":512,"messages":[{"role":"system","content":"sys"},{"role":"user","content":"hello"}]}`)
	prompt, systemPrompt, maxTokens := extractSamplingInput(raw)
	if prompt != "hello" {
		t.Errorf("prompt: got %q", prompt)
	}
	if systemPrompt != "sys" {
		t.Errorf("systemPrompt: got %q", systemPrompt)
	}
	if maxTokens != 512 {
		t.Errorf("maxTokens: got %d", maxTokens)
	}
}

func TestParseJSONRawToMap(t *testing.T) {
	raw := json.RawMessage("{}")
	m := parseJSONRawToMap(raw)
	if m == nil {
		t.Fatal("expected map")
	}
	m = parseJSONRawToMap(json.RawMessage("bad"))
	if m != nil {
		t.Error("expected nil")
	}
}

func TestRequestPermissionTimeoutCleansPending(t *testing.T) {
	s := &server{
		pending:           make(map[string]chan json.RawMessage),
		w:                 &bytes.Buffer{},
		permissionTimeout: time.Millisecond,
	}

	if s.requestPermission("session-1", "tool-1", "bash", map[string]any{"command": "date"}) {
		t.Fatal("requestPermission returned true, want false on timeout")
	}

	if len(s.pending) != 0 {
		t.Fatalf("pending len = %d, want 0", len(s.pending))
	}
}

func TestWriteMessageReturnsWriteError(t *testing.T) {
	s := &server{w: errWriter{}}

	if err := s.writeMessage(map[string]any{"jsonrpc": "2.0"}); err == nil {
		t.Fatal("writeMessage error = nil, want error")
	}
}

func TestReadRequestRejectsOversizedMessage(t *testing.T) {
	s := &server{r: bufio.NewReader(strings.NewReader(strings.Repeat("x", maxRequestBytes+1) + "\n"))}

	if _, err := s.readRequest(); err == nil {
		t.Fatal("readRequest error = nil, want oversized error")
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
