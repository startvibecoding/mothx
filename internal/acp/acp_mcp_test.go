package acp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
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

func TestInitializeAdvertisesStandardSessionLifecycleCapabilities(t *testing.T) {
	var out bytes.Buffer
	s := &server{w: &out}
	s.handleInitialize(rpcRequest{ID: json.RawMessage("1")})

	message := jsonLines(t, &out)[0]
	result := message["result"].(map[string]any)
	caps := result["agentCapabilities"].(map[string]any)["sessionCapabilities"].(map[string]any)
	if _, ok := caps["close"].(map[string]any); !ok {
		t.Fatalf("close capability = %#v, want object", caps["close"])
	}
	if _, ok := caps["list"].(map[string]any); !ok {
		t.Fatalf("list capability = %#v, want object", caps["list"])
	}
	if _, ok := caps["delete"].(map[string]any); !ok {
		t.Fatalf("delete capability = %#v, want object", caps["delete"])
	}
	if _, ok := caps["resume"].(map[string]any); !ok {
		t.Fatalf("resume capability = %#v, want object", caps["resume"])
	}
	mcpCaps := result["agentCapabilities"].(map[string]any)["mcpCapabilities"].(map[string]any)
	if _, ok := mcpCaps["stdio"]; ok {
		t.Fatalf("stdio must not be advertised as an MCP extension: %#v", mcpCaps)
	}
	meta := result["agentCapabilities"].(map[string]any)["_meta"].(map[string]any)
	if _, ok := meta[mothxExtensionNamespace]; !ok {
		t.Fatalf("missing MothX extension capability: %#v", meta)
	}
}

func TestCloseSessionCancelsAndRemovesRuntime(t *testing.T) {
	var out bytes.Buffer
	cancelled := make(chan struct{})
	s := &server{
		w: &out,
		sessions: map[string]*sessionRuntime{
			"session-1": {cancel: func() { close(cancelled) }},
		},
	}
	s.handleCloseSession(rpcRequest{
		ID:     json.RawMessage("1"),
		Params: json.RawMessage(`{"sessionId":"session-1"}`),
	})

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("close did not cancel active session")
	}
	if _, ok := s.sessions["session-1"]; ok {
		t.Fatal("closed session remains in runtime map")
	}
	message := jsonLines(t, &out)[0]
	if _, ok := message["result"].(map[string]any); !ok {
		t.Fatalf("close result = %#v, want empty object", message["result"])
	}
}

func TestListSessionsReturnsPersistedSessions(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	newTestSession(t, cwd, dir, "session-one", 1)
	newTestSession(t, cwd, dir, "session-two", 2)

	var out bytes.Buffer
	s := &server{settings: &config.Settings{SessionDir: dir}, w: &out}
	s.handleListSessions(rpcRequest{
		ID:     json.RawMessage("1"),
		Params: json.RawMessage(fmt.Sprintf(`{"cwd":%q}`, cwd)),
	})

	message := jsonLines(t, &out)[0]
	result := message["result"].(map[string]any)
	sessions := result["sessions"].([]any)
	if len(sessions) != 2 {
		t.Fatalf("listed sessions = %d, want 2", len(sessions))
	}
	for _, item := range sessions {
		listed := item.(map[string]any)
		if listed["cwd"] != cwd {
			t.Fatalf("listed cwd = %q, want %q", listed["cwd"], cwd)
		}
		if listed["sessionId"] == "" {
			t.Fatalf("missing session ID: %#v", listed)
		}
	}
}

func TestLoadSessionReplaysAllMessages(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	newTestSession(t, cwd, dir, "full-history", 41)

	var out bytes.Buffer
	s := &server{
		settings:   &config.Settings{SessionDir: dir},
		sbMgr:      sandbox.NewManager(cwd),
		sessions:   make(map[string]*sessionRuntime),
		toolTitles: make(map[string]string),
		mcpNotify:  make(map[string]bool),
		w:          &out,
	}
	s.handleLoadSession(rpcRequest{
		ID:     json.RawMessage("1"),
		Params: json.RawMessage(fmt.Sprintf(`{"sessionId":"full-history","cwd":%q}`, cwd)),
	})

	messages := jsonLines(t, &out)
	updates := 0
	for _, message := range messages {
		if message["method"] == "session/update" {
			updates++
		}
	}
	if updates != 41 {
		t.Fatalf("replayed updates = %d, want 41", updates)
	}
}

func TestResumeSessionDoesNotReplayMessages(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	newTestSession(t, cwd, dir, "resume-history", 2)

	var out bytes.Buffer
	s := testSessionServer(cwd, dir, &out)
	s.handleResumeSession(rpcRequest{
		ID:     json.RawMessage("1"),
		Params: json.RawMessage(fmt.Sprintf(`{"sessionId":"resume-history","cwd":%q}`, cwd)),
	})

	messages := jsonLines(t, &out)
	if len(messages) != 1 || messages[0]["result"] == nil {
		t.Fatalf("resume messages = %#v, want one response without replay", messages)
	}
}

func TestDeleteSessionRemovesPersistedSession(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	newTestSession(t, cwd, dir, "delete-me", 1)

	var out bytes.Buffer
	s := &server{settings: &config.Settings{SessionDir: dir}, w: &out}
	s.handleDeleteSession(rpcRequest{
		ID:     json.RawMessage("1"),
		Params: json.RawMessage(`{"sessionId":"delete-me"}`),
	})
	if _, err := session.OpenByID(cwd, dir, "delete-me"); err == nil {
		t.Fatal("deleted session is still available")
	}
}

func TestNewSessionMCPFailureRollsBackPersistedSession(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	var out bytes.Buffer
	s := testSessionServer(cwd, dir, &out)

	s.handleNewSession(rpcRequest{
		ID: json.RawMessage("1"),
		Params: json.RawMessage(fmt.Sprintf(
			`{"cwd":%q,"mcpServers":[{"name":"broken","type":"stdio"}]}`,
			cwd,
		)),
	})

	messages := jsonLines(t, &out)
	if len(messages) != 1 || messages[0]["error"] == nil {
		t.Fatalf("new session response = %#v, want MCP error", messages)
	}
	if len(s.sessions) != 0 {
		t.Fatalf("runtime sessions = %d, want 0", len(s.sessions))
	}
	persisted, err := session.ListForDir(cwd, dir)
	if err != nil {
		t.Fatalf("list persisted sessions: %v", err)
	}
	if len(persisted) != 0 {
		t.Fatalf("persisted sessions = %#v, want none after MCP failure", persisted)
	}
}

func TestCancelRequestCancelsMatchingPrompt(t *testing.T) {
	cancelled := make(chan struct{})
	s := &server{sessions: map[string]*sessionRuntime{
		"session-1": {promptID: "prompt-1", cancel: func() { close(cancelled) }},
	}}
	s.handleCancelRequest(rpcRequest{Params: json.RawMessage(`{"requestId":"prompt-1"}`)})
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("$/cancel_request did not cancel matching prompt")
	}
}

func TestPlanUpdateUsesStandardPlanVariant(t *testing.T) {
	var out bytes.Buffer
	s := &server{w: &out}
	s.handleAgentEvent("session-1", agentpkg.Event{
		Type: agentpkg.EventPlanUpdate,
		Plan: &agentpkg.TaskPlan{
			Title: "Implementation",
			Steps: []agentpkg.PlanStep{{Title: "Inspect", Status: "running"}},
		},
	})
	update := jsonLines(t, &out)[0]["params"].(map[string]any)["update"].(map[string]any)
	if update["sessionUpdate"] != "plan" {
		t.Fatalf("plan update type = %#v", update)
	}
	entry := update["entries"].([]any)[0].(map[string]any)
	if entry["content"] != "Inspect" || entry["priority"] != "medium" || entry["status"] != "in_progress" {
		t.Fatalf("plan entry = %#v", entry)
	}
}

func TestMothxStatusUsesExtensionNotification(t *testing.T) {
	var out bytes.Buffer
	s := &server{w: &out}
	s.handleAgentEvent("session-1", agentpkg.Event{Type: agentpkg.EventStatus, StatusMessage: "working"})
	message := jsonLines(t, &out)[0]
	if message["method"] != "_mothx/session_event" {
		t.Fatalf("status method = %#v, want extension notification", message["method"])
	}
}

func TestPromptSupportsResourceLinksAndRejectsUnadvertisedContent(t *testing.T) {
	text, err := promptToText([]contentBlock{{Type: "resource_link", Name: "notes", URI: "file:///notes.md"}})
	if err != nil || text != "notes: file:///notes.md" {
		t.Fatalf("resource link = %q, %v", text, err)
	}
	if _, err := promptToText([]contentBlock{{Type: "image"}}); err == nil {
		t.Fatal("unadvertised image content was accepted")
	}
}

func TestUsageEventEmitsUsageUpdate(t *testing.T) {
	var out bytes.Buffer
	s := &server{
		m: &provider.Model{ContextWindow: 100, Cost: provider.ModelPricing{Input: 1, Output: 2}},
		sessions: map[string]*sessionRuntime{
			"session-1": {},
		},
		w: &out,
	}
	s.handleAgentEvent("session-1", agentpkg.Event{
		Type: agentpkg.EventUsage,
		Usage: &agentpkg.Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
		ContextUsage: &agentpkg.ContextUsage{Tokens: 20, ContextWindow: 100},
	})

	message := jsonLines(t, &out)[0]
	update := message["params"].(map[string]any)["update"].(map[string]any)
	if update["sessionUpdate"] != "usage_update" || update["used"] != float64(20) || update["size"] != float64(100) {
		t.Fatalf("usage update = %#v", update)
	}
	cost := update["cost"].(map[string]any)
	if cost["currency"] != "USD" || cost["amount"] != 0.00002 {
		t.Fatalf("usage cost = %#v", cost)
	}
}

func newTestSession(t *testing.T, cwd, dir, id string, messages int) {
	t.Helper()
	mgr := session.New(cwd, dir)
	if err := mgr.InitWithID(id); err != nil {
		t.Fatalf("initialize session: %v", err)
	}
	for i := 0; i < messages; i++ {
		if _, err := mgr.AppendMessage(provider.NewUserMessage(fmt.Sprintf("message %d", i))); err != nil {
			t.Fatalf("append message %d: %v", i, err)
		}
	}
}

func testSessionServer(cwd, dir string, out *bytes.Buffer) *server {
	return &server{
		settings:   &config.Settings{SessionDir: dir},
		sbMgr:      sandbox.NewManager(cwd),
		sessions:   make(map[string]*sessionRuntime),
		toolTitles: make(map[string]string),
		mcpNotify:  make(map[string]bool),
		w:          out,
	}
}

func jsonLines(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var messages []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var message map[string]any
		if err := json.Unmarshal([]byte(line), &message); err != nil {
			t.Fatalf("decode JSON-RPC message %q: %v", line, err)
		}
		messages = append(messages, message)
	}
	return messages
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
