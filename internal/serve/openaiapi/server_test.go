package openaiapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/workflow"
)

type recordingAPIProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

func newRecordingAPIProvider() *recordingAPIProvider {
	return &recordingAPIProvider{
		models: []*provider.Model{{ID: "m1", Name: "Model 1", ContextWindow: 32768, MaxTokens: 2048}},
	}
}

func (p *recordingAPIProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	p.calls = append(p.calls, provider.ChatParams{
		Messages:     append([]provider.Message(nil), params.Messages...),
		SystemPrompt: params.SystemPrompt,
	})
	ch := make(chan provider.StreamEvent, 3)
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "ok"}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *recordingAPIProvider) Name() string              { return "recording-API" }
func (p *recordingAPIProvider) API() string               { return "openai-chat" }
func (p *recordingAPIProvider) Models() []*provider.Model { return p.models }
func (p *recordingAPIProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// --- Config tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Listen != ":8080" {
		t.Errorf("default listen = %q, want :8080", cfg.Listen)
	}
	if cfg.DefaultMode != "yolo" {
		t.Errorf("default mode = %q, want yolo", cfg.DefaultMode)
	}
	if cfg.ToolVisibility.Mode != "content" {
		t.Errorf("default tool visibility = %q, want content", cfg.ToolVisibility.Mode)
	}
	if cfg.SystemPromptMode != "append" {
		t.Errorf("default system prompt mode = %q, want append", cfg.SystemPromptMode)
	}
	if cfg.RequestTimeoutSecs != 1800 {
		t.Errorf("default timeout = %d, want 1800", cfg.RequestTimeoutSecs)
	}
	if cfg.Auth.Enabled {
		t.Error("auth should be disabled by default")
	}
}

func TestValidateWorkDir(t *testing.T) {
	tests := []struct {
		name    string
		allowed *[]string
		dir     string
		wantErr bool
	}{
		{"nil=no check", nil, "/any/path", false},
		{"empty=deny all", &[]string{}, "/any/path", true},
		{"exact match", &[]string{"/home/user/projects"}, "/home/user/projects", false},
		{"prefix match", &[]string{"/home/user/projects"}, "/home/user/projects/foo", false},
		{"evil prefix", &[]string{"/home/user/projects"}, "/home/user/projects-evil", true},
		{"no match", &[]string{"/opt/repos"}, "/home/user/projects", true},
		{"multi allowed", &[]string{"/opt/repos", "/home/user"}, "/home/user/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{AllowedWorkDirs: tt.allowed}
			err := cfg.ValidateWorkDir(tt.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkDir(%q) error = %v, wantErr = %v", tt.dir, err, tt.wantErr)
			}
		})
	}
}

func TestCloneConfig(t *testing.T) {
	allowed := []string{"/home/test", "/opt/repos"}
	cfg := &Config{
		Listen: ":8080",
		Auth: AuthConfig{
			Enabled: true,
			Tokens:  []string{"sk-a", "sk-b"},
		},
		CORS: CORSConfig{
			Enabled:      true,
			AllowOrigins: []string{"http://localhost:3000"},
		},
		AllowedWorkDirs: &allowed,
	}

	clone := cloneConfig(cfg)
	if clone == cfg {
		t.Fatal("cloneConfig returned the original pointer")
	}

	clone.Listen = ":9090"
	clone.Auth.Tokens[0] = "sk-mutated"
	clone.CORS.AllowOrigins[0] = "http://mutated"
	(*clone.AllowedWorkDirs)[0] = "/tmp/mutated"

	if cfg.Listen != ":8080" {
		t.Fatalf("original listen mutated: %q", cfg.Listen)
	}
	if got := cfg.Auth.Tokens[0]; got != "sk-a" {
		t.Fatalf("original auth tokens mutated: %q", got)
	}
	if got := cfg.CORS.AllowOrigins[0]; got != "http://localhost:3000" {
		t.Fatalf("original CORS origins mutated: %q", got)
	}
	if got := (*cfg.AllowedWorkDirs)[0]; got != "/home/test" {
		t.Fatalf("original allowedWorkDirs mutated: %q", got)
	}
}

func TestApplyRunOverrides(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Listen = ":8080"
	cfg.EnableSubAgents = false
	cfg.EnableDelegate = false
	cfg.EnableWorkflows = false
	cfg.Sandbox.Enabled = false
	cfg.WorkingDir = "/tmp/original"

	applyRunOverrides(cfg, RunOptions{
		Port:       "9090",
		WorkDir:    "/tmp/override",
		Sandbox:    true,
		MultiAgent: true,
		Delegate:   true,
		Workflows:  true,
	})

	if cfg.Listen != ":9090" {
		t.Fatalf("listen = %q, want :9090", cfg.Listen)
	}
	if cfg.WorkingDir != "/tmp/override" {
		t.Fatalf("workDir = %q, want /tmp/override", cfg.WorkingDir)
	}
	if !cfg.Sandbox.Enabled {
		t.Fatal("sandbox should be enabled")
	}
	if !cfg.EnableSubAgents {
		t.Fatal("multi-agent should be enabled")
	}
	if !cfg.EnableDelegate {
		t.Fatal("delegate should be enabled")
	}
	if !cfg.EnableWorkflows {
		t.Fatal("workflows should be enabled")
	}
}

func TestApplyRunOverrides_PortForms(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "port only", port: "9090", want: ":9090"},
		{name: "colon port", port: ":9090", want: ":9090"},
		{name: "host port", port: "127.0.0.1:9090", want: "127.0.0.1:9090"},
		{name: "all interfaces host port", port: "0.0.0.0:9090", want: "0.0.0.0:9090"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			applyRunOverrides(cfg, RunOptions{Port: tt.port})
			if cfg.Listen != tt.want {
				t.Fatalf("listen = %q, want %q", cfg.Listen, tt.want)
			}
		})
	}
}

func TestLoadRunConfig_UsesInMemoryConfigAndClones(t *testing.T) {
	allowed := []string{"/home/test"}
	original := &Config{
		Listen: ":8080",
		Auth: AuthConfig{
			Enabled: true,
			Tokens:  []string{"sk-a"},
		},
		CORS: CORSConfig{
			Enabled:      true,
			AllowOrigins: []string{"http://localhost:3000"},
		},
		AllowedWorkDirs: &allowed,
	}

	cfg, err := loadRunConfig(RunOptions{
		Config:     original,
		Port:       "9090",
		WorkDir:    "/tmp/work",
		Sandbox:    true,
		MultiAgent: true,
		Delegate:   true,
		Workflows:  true,
	})
	if err != nil {
		t.Fatalf("loadRunConfig: %v", err)
	}

	if cfg == original {
		t.Fatal("loadRunConfig returned original config pointer")
	}
	if cfg.Listen != ":9090" {
		t.Fatalf("listen = %q, want :9090", cfg.Listen)
	}
	if cfg.WorkingDir != "/tmp/work" {
		t.Fatalf("workDir = %q, want /tmp/work", cfg.WorkingDir)
	}
	if !cfg.Sandbox.Enabled || !cfg.EnableSubAgents || !cfg.EnableDelegate || !cfg.EnableWorkflows {
		t.Fatal("expected overrides to be applied")
	}

	if original.Listen != ":8080" {
		t.Fatalf("original listen mutated: %q", original.Listen)
	}
	if original.WorkingDir != "" {
		t.Fatalf("original workDir mutated: %q", original.WorkingDir)
	}
	if original.Sandbox.Enabled || original.EnableSubAgents || original.EnableDelegate || original.EnableWorkflows {
		t.Fatal("original config booleans mutated")
	}
}

func TestRegisterRoutes_DisableAPI(t *testing.T) {
	srv := newTestServer(t)
	mux := http.NewServeMux()

	registerRoutes(mux, srv, RunOptions{DisableAPI: true})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/v1/chat/completions status = %d, want 404", w.Code)
	}

	req = httptest.NewRequest("GET", "/v1/models", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/v1/models status = %d, want 404", w.Code)
	}
}

type hijackableResponseWriter struct {
	header http.Header
	conn   net.Conn
	peer   net.Conn
}

func (w *hijackableResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *hijackableResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *hijackableResponseWriter) WriteHeader(statusCode int) {}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.conn, w.peer = net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn))
	return w.conn, rw, nil
}

func TestLoggingResponseWriterSupportsHijack(t *testing.T) {
	base := &hijackableResponseWriter{}
	lw := &loggingResponseWriter{ResponseWriter: base, statusCode: http.StatusOK}

	conn, _, err := lw.Hijack()
	if err != nil {
		t.Fatalf("Hijack: %v", err)
	}
	defer conn.Close()
	if base.peer != nil {
		defer base.peer.Close()
	}
}

// --- Auth middleware tests ---

func TestAuthMiddleware_Disabled(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: false}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: true, Tokens: []string{"sk-test"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-test")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: true, Tokens: []string{"sk-test"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: true, Tokens: []string{"sk-test"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_EnabledWithoutTokensRejects(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: true}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer anything")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- CORS middleware tests ---

func TestCORSMiddleware_Enabled(t *testing.T) {
	handler := CORSMiddleware(CORSConfig{Enabled: true, AllowOrigins: []string{"http://example.com"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Errorf("CORS origin = %q, want http://example.com", got)
	}
}

func TestCORSMiddleware_MultipleOriginsEchoesRequestOrigin(t *testing.T) {
	handler := CORSMiddleware(CORSConfig{Enabled: true, AllowOrigins: []string{"http://a.example", "http://b.example"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://b.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://b.example" {
		t.Errorf("CORS origin = %q, want http://b.example", got)
	}
}

func TestCORSMiddleware_MultipleOriginsRejectsUnknownOrigin(t *testing.T) {
	handler := CORSMiddleware(CORSConfig{Enabled: true, AllowOrigins: []string{"http://a.example", "http://b.example"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("CORS origin = %q, want empty", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := CORSMiddleware(CORSConfig{Enabled: true}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

// --- Concurrency middleware tests ---

func TestConcurrencyMiddleware_NoLimit(t *testing.T) {
	handler := ConcurrencyMiddleware(0, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// --- SessionPool tests ---

func TestSessionPool_PutGet(t *testing.T) {
	pool := NewSessionPool(0, 0)
	defer pool.Stop()

	sess := &APISession{ID: "sess-1", WorkDir: "/tmp", LastUsed: time.Now()}
	if err := pool.Put(sess); err != nil {
		t.Fatalf("put: %v", err)
	}
	got := pool.Get("sess-1")
	if got == nil || got.ID != "sess-1" {
		t.Error("expected to get session back")
	}
	if pool.Count() != 1 {
		t.Errorf("count = %d, want 1", pool.Count())
	}
}

func TestSessionPool_MaxSessions(t *testing.T) {
	pool := NewSessionPool(1, 0)
	defer pool.Stop()

	sess1 := &APISession{ID: "sess-1", LastUsed: time.Now()}
	if err := pool.Put(sess1); err != nil {
		t.Fatalf("put 1: %v", err)
	}
	sess2 := &APISession{ID: "sess-2", LastUsed: time.Now()}
	if err := pool.Put(sess2); err == nil {
		t.Error("expected pool full error")
	}
}

func TestSessionPool_Remove(t *testing.T) {
	pool := NewSessionPool(0, 0)
	defer pool.Stop()

	pool.Put(&APISession{ID: "sess-1", LastUsed: time.Now()})
	pool.Remove("sess-1")
	if pool.Get("sess-1") != nil {
		t.Error("session should be removed")
	}
}

func TestSessionPool_List(t *testing.T) {
	pool := NewSessionPool(0, 0)
	defer pool.Stop()

	pool.Put(&APISession{ID: "a", LastUsed: time.Now()})
	pool.Put(&APISession{ID: "b", LastUsed: time.Now()})
	ids := pool.List()
	if len(ids) != 2 {
		t.Errorf("list len = %d, want 2", len(ids))
	}
}

func TestListActiveSessions(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("s1"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewUserMessage("hello")); err != nil {
		t.Fatalf("append message: %v", err)
	}
	otherMgr := session.New("/tmp/history-only", srv.settings.GetSessionDir())
	if err := otherMgr.InitWithID("s3"); err != nil {
		t.Fatalf("init other session: %v", err)
	}
	if _, err := otherMgr.AppendMessage(provider.NewUserMessage("history only")); err != nil {
		t.Fatalf("append other message: %v", err)
	}
	older := time.Now().Add(-time.Minute)
	newer := time.Now()
	if err := srv.pool.Put(&APISession{ID: "s1", WorkDir: srv.cfg.GetWorkDir(), Manager: mgr, Mode: "agent", LastUsed: older}); err != nil {
		t.Fatalf("put s1: %v", err)
	}
	if err := srv.pool.Put(&APISession{ID: "s2", WorkDir: "/tmp/other", LastUsed: newer}); err != nil {
		t.Fatalf("put s2: %v", err)
	}

	sessions := srv.ListActiveSessions()
	if len(sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sessions))
	}
	byID := make(map[string]ActiveSessionInfo)
	for _, sess := range sessions {
		byID[sess.ID] = sess
	}
	if !byID["s1"].Active || byID["s1"].Mode != "agent" || byID["s1"].MessageCount != 1 {
		t.Fatalf("s1 details = %#v", byID["s1"])
	}
	if !byID["s2"].Active || byID["s2"].WorkDir != "/tmp/other" {
		t.Fatalf("s2 details = %#v", byID["s2"])
	}
	if byID["s3"].Active || byID["s3"].WorkDir != "/tmp/history-only" || byID["s3"].Preview != "history only" {
		t.Fatalf("s3 details = %#v", byID["s3"])
	}
}

func TestGetSessionMessagesReadsFromSessionDB(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("db-history"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	userMsg := provider.NewUserMessage("describe this")
	userMsg.Contents = []provider.ContentBlock{
		{Type: "text", Text: "describe this"},
		{Type: "image", Image: &provider.ImageContent{MimeType: "image/png", Data: "aW1n", Bytes: 3}},
	}
	if _, err := mgr.AppendMessage(userMsg); err != nil {
		t.Fatalf("append user message: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "ok"}})); err != nil {
		t.Fatalf("append assistant message: %v", err)
	}

	messages, err := srv.GetSessionMessages("db-history")
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "describe this" {
		t.Fatalf("first message = %#v", messages[0])
	}
	if len(messages[0].Contents) != 2 || messages[0].Contents[1].Image == nil || messages[0].Contents[1].Image.Data != "aW1n" {
		t.Fatalf("first message contents = %#v", messages[0].Contents)
	}
	if messages[1].Role != "assistant" || messages[1].Content != "ok" {
		t.Fatalf("second message = %#v", messages[1])
	}
}

func TestGetSessionMessagesIncludesToolCallsAndCollapsedResults(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("tool-history"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewUserMessage("list files")); err != nil {
		t.Fatalf("append user message: %v", err)
	}
	assistant := provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "text", Text: "I will inspect the tree."},
		{Type: "toolCall", ToolCall: &provider.ToolCallBlock{
			ID:        "call-1",
			Name:      "bash",
			Arguments: json.RawMessage(`{"command":"ls -la"}`),
		}},
	})
	if _, err := mgr.AppendMessage(assistant); err != nil {
		t.Fatalf("append assistant message: %v", err)
	}
	fullOutput := "total 8\n-rw-r--r-- file.txt\n"
	if _, err := mgr.AppendMessage(provider.NewToolResultMessage("call-1", "bash", fullOutput, false)); err != nil {
		t.Fatalf("append tool result: %v", err)
	}

	messages, err := srv.GetSessionMessages("tool-history")
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("messages len = %d, want 4: %#v", len(messages), messages)
	}
	if messages[1].Role != "assistant" || messages[1].Content != "I will inspect the tree." {
		t.Fatalf("assistant message = %#v", messages[1])
	}
	if messages[2].Role != "toolCall" || messages[2].ToolCallID != "call-1" || messages[2].ToolName != "bash" {
		t.Fatalf("tool call entry = %#v", messages[2])
	}
	if string(messages[2].Arguments) != `{"command":"ls -la"}` {
		t.Fatalf("tool call args = %s", messages[2].Arguments)
	}
	if messages[3].Role != "toolResult" || messages[3].Content != "" || !messages[3].HasDetail {
		t.Fatalf("tool result summary entry = %#v", messages[3])
	}
	if messages[3].Summary != "total 8" {
		t.Fatalf("tool result summary = %q", messages[3].Summary)
	}

	detail, err := srv.GetSessionToolResult("tool-history", "call-1")
	if err != nil {
		t.Fatalf("GetSessionToolResult: %v", err)
	}
	if detail.Content != fullOutput || detail.ToolName != "bash" {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestGetSessionMessagesExtractsPlanToolCall(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("plan-history"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	assistant := provider.NewAssistantMessage([]provider.ContentBlock{
		{Type: "toolCall", ToolCall: &provider.ToolCallBlock{
			ID:   "plan-call",
			Name: "plan",
			Arguments: json.RawMessage(`{
				"title":"Ship WebUI plan",
				"steps":[
					{"title":"Read current UI","status":"done"},
					{"title":"Render todo card","status":"running"},
					{"title":"Build frontend","status":"pending"}
				],
				"note":"Keep output compact"
			}`),
		}},
	})
	if _, err := mgr.AppendMessage(assistant); err != nil {
		t.Fatalf("append assistant message: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewToolResultMessage("plan-call", "plan", "Plan updated.", false)); err != nil {
		t.Fatalf("append tool result: %v", err)
	}

	messages, err := srv.GetSessionMessages("plan-history")
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2: %#v", len(messages), messages)
	}
	if messages[0].Role != "toolCall" || messages[0].ToolName != "plan" || messages[0].Plan == nil {
		t.Fatalf("plan tool call = %#v", messages[0])
	}
	if messages[0].Plan.Title != "Ship WebUI plan" || messages[0].Plan.Note != "Keep output compact" {
		t.Fatalf("plan = %#v", messages[0].Plan)
	}
	if len(messages[0].Plan.Steps) != 3 || messages[0].Plan.Steps[1].Status != "running" {
		t.Fatalf("plan steps = %#v", messages[0].Plan.Steps)
	}
	if messages[1].Role != "toolResult" || messages[1].ToolName != "plan" {
		t.Fatalf("plan result = %#v", messages[1])
	}
}

func TestDeleteActiveSession(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("delete-me"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewUserMessage("hello")); err != nil {
		t.Fatalf("append message: %v", err)
	}
	if err := srv.pool.Put(&APISession{ID: "delete-me", WorkDir: srv.cfg.GetWorkDir(), Manager: mgr, LastUsed: time.Now()}); err != nil {
		t.Fatalf("put session: %v", err)
	}
	srv.defaultSessionIDs = map[string]string{srv.cfg.GetWorkDir(): "delete-me"}

	deleted, err := srv.DeleteActiveSession("delete-me")
	if err != nil {
		t.Fatalf("DeleteActiveSession: %v", err)
	}
	if !deleted {
		t.Fatal("session should be deleted")
	}
	if srv.pool.Get("delete-me") != nil {
		t.Fatal("session should be removed from pool")
	}
	if srv.defaultSessionIDs[srv.cfg.GetWorkDir()] != "" {
		t.Fatalf("default session ID was not cleared: %#v", srv.defaultSessionIDs)
	}
	sessions, err := session.ListForDir(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err != nil {
		t.Fatalf("ListForDir: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("persisted sessions = %d, want 0", len(sessions))
	}
}

func TestDeleteActiveSessionDeletesPersistedSession(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	mgr := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := mgr.InitWithID("persisted-delete"); err != nil {
		t.Fatalf("init session: %v", err)
	}
	if _, err := mgr.AppendMessage(provider.NewUserMessage("hello")); err != nil {
		t.Fatalf("append message: %v", err)
	}

	deleted, err := srv.DeleteActiveSession("persisted-delete")
	if err != nil {
		t.Fatalf("DeleteActiveSession: %v", err)
	}
	if !deleted {
		t.Fatal("persisted session should be deleted")
	}
	if _, err := session.OpenByIDExact(srv.settings.GetSessionDir(), "persisted-delete"); err == nil {
		t.Fatal("session should not exist after delete")
	}
}

func TestDeleteActiveSessionAmbiguousID(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	if err := srv.pool.Put(&APISession{ID: "same", WorkDir: "/tmp/a", LastUsed: time.Now()}); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := srv.pool.Put(&APISession{ID: "same", WorkDir: "/tmp/b", LastUsed: time.Now()}); err != nil {
		t.Fatalf("put b: %v", err)
	}

	deleted, err := srv.DeleteActiveSession("same")
	if !errors.Is(err, ErrActiveSessionIDAmbiguous) {
		t.Fatalf("err = %v, want ErrActiveSessionIDAmbiguous", err)
	}
	if deleted {
		t.Fatal("ambiguous session should not be deleted")
	}
	if srv.pool.Count() != 2 {
		t.Fatalf("pool count = %d, want 2", srv.pool.Count())
	}
}

// --- parseMessages tests ---

func TestParseMessages(t *testing.T) {
	msgs := []RequestMessage{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "explain main.go"},
	}
	lastUser, sysMsgs, history := parseMessages(msgs)
	if lastUser.Content != "explain main.go" {
		t.Errorf("lastUser = %q", lastUser.Content)
	}
	if len(sysMsgs) != 1 || sysMsgs[0] != "you are helpful" {
		t.Errorf("sysMsgs = %v", sysMsgs)
	}
	if len(history) != 2 { // "hello" and "hi there"
		t.Errorf("history len = %d, want 2", len(history))
	}
}

func TestParseMessages_NoUser(t *testing.T) {
	msgs := []RequestMessage{
		{Role: "system", Content: "test"},
	}
	lastUser, _, _ := parseMessages(msgs)
	if lastUser.Content != "" {
		t.Errorf("expected empty lastUser, got %q", lastUser.Content)
	}
}

func TestRequestMessageMultimodalContent(t *testing.T) {
	var msg RequestMessage
	body := `{"role":"user","content":[{"type":"text","text":"describe this"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aW1n","detail":"auto"}}]}`
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Content != "describe this" {
		t.Fatalf("content = %q", msg.Content)
	}
	providerMsg, err := buildUserMessage(msg)
	if err != nil {
		t.Fatalf("buildUserMessage: %v", err)
	}
	if len(providerMsg.Contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(providerMsg.Contents))
	}
	if providerMsg.Contents[1].Image == nil || providerMsg.Contents[1].Image.MimeType != "image/png" || providerMsg.Contents[1].Image.Data != "aW1n" {
		t.Fatalf("image content = %#v", providerMsg.Contents[1].Image)
	}
}

func TestChatHandlerRejectsImageForTextOnlyModel(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()
	srv.model.Input = []string{"text"}

	body := `{"messages":[{"role":"user","content":[{"type":"text","text":"describe"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aW1n"}}]}],"stream":false}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "does not support image input") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

// --- SSE writer tests ---

func TestSSEWriter_ContentDelta(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w, "test-model", "sess-1")
	sse.WriteContentDelta("hello")
	body := w.Body.String()
	if !strings.Contains(body, `"content":"hello"`) {
		t.Errorf("body doesn't contain content delta: %s", body)
	}
	if !strings.HasPrefix(body, "data: ") {
		t.Error("SSE data should start with 'data: '")
	}
}

func TestSSEWriter_Done(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w, "test-model", "sess-1")
	sse.WriteDone(&CompletionUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150})
	body := w.Body.String()
	if !strings.Contains(body, `"finish_reason":"stop"`) {
		t.Errorf("missing finish_reason: %s", body)
	}
	if !strings.Contains(body, "[DONE]") {
		t.Error("missing [DONE] sentinel")
	}
}

func TestSSEWriter_ToolStatusContent(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w, "test-model", "")
	sse.WriteToolStatusContent("🔧 [read] main.go", "running")
	body := w.Body.String()
	if !strings.Contains(body, "[running]") {
		t.Errorf("missing status in content: %s", body)
	}
	if !strings.Contains(body, "read") {
		t.Errorf("missing tool name in content: %s", body)
	}
}

func TestSSEWriter_ToolStatusEvent(t *testing.T) {
	w := httptest.NewRecorder()
	sse := NewSSEWriter(w, "test-model", "")
	sse.WriteToolStatusEvent(ToolStatusEvent{
		Tool:       "bash",
		ToolCallID: "call-1",
		Status:     "running",
		Args:       map[string]any{"command": "ls"},
	})
	body := w.Body.String()
	if !strings.Contains(body, "event: tool_status") {
		t.Errorf("missing tool_status event: %s", body)
	}
	if !strings.Contains(body, `"tool":"bash"`) {
		t.Errorf("missing tool name: %s", body)
	}
	if !strings.Contains(body, `"toolCallId":"call-1"`) {
		t.Errorf("missing tool call id: %s", body)
	}
}

// --- writeError / writeJSON tests ---

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "bad input", "invalid_request_error")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error.Message != "bad input" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
}

// --- Health handler test ---

func TestHealthHandler(t *testing.T) {
	srv := &Server{
		version: "test",
		pool:    NewSessionPool(0, 0),
	}
	defer srv.pool.Stop()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q", resp.Status)
	}
	if resp.Version != "test" {
		t.Errorf("version = %q", resp.Version)
	}
}

// --- Models handler test ---

func TestModelsHandler(t *testing.T) {
	mockP := provider.NewMockProvider("test", []*provider.Model{
		{ID: "m1", Name: "Model 1", Input: []string{"text", "image"}},
		{ID: "m2", Name: "Model 2"},
	}, nil)
	srv := &Server{
		provider: mockP,
	}
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	srv.handleModels(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp ModelListResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Object != "list" {
		t.Errorf("object = %q", resp.Object)
	}
	if len(resp.Data) != 2 {
		t.Errorf("models = %d, want 2", len(resp.Data))
	}
	if len(resp.Data[0].Input) != 2 || resp.Data[0].Input[0] != "text" || resp.Data[0].Input[1] != "image" {
		t.Errorf("model input = %#v, want text/image", resp.Data[0].Input)
	}
}

// --- Chat handler slash command test ---

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cwd := t.TempDir()
	models := []*provider.Model{
		{ID: "m1", Name: "Model 1"},
	}
	mockP := provider.NewMockProvider("test", models, nil)

	settings := config.DefaultSettings()
	settings.SessionDir = filepath.Join(cwd, "sessions")

	sbMgr := sandbox.NewManager(cwd)
	sbMgr.SetLevel(sandbox.LevelNone)

	skillsMgr := skills.NewManager(filepath.Join(cwd, "skills-global"), filepath.Join(cwd, "skills-project"))

	pool := NewSessionPool(0, 0)

	cfg := DefaultConfig()
	cfg.WorkingDir = cwd

	return &Server{
		cfg:        cfg,
		settings:   settings,
		version:    "test",
		provider:   mockP,
		model:      models[0],
		sandboxMgr: sbMgr,
		skillsMgr:  skillsMgr,
		pool:       pool,
	}
}

func newRecordingAPIServer(t *testing.T) (*Server, *recordingAPIProvider) {
	t.Helper()
	srv := newTestServer(t)
	p := newRecordingAPIProvider()
	srv.provider = p
	srv.model = p.models[0]
	return srv, p
}

func TestCloneModelCopiesMutableFields(t *testing.T) {
	model := &provider.Model{
		ID:     "m1",
		Input:  []string{"text"},
		Compat: &provider.ModelCompat{ThinkingFormat: "anthropic"},
	}

	clone := cloneModel(model)
	clone.Input[0] = "image"
	clone.Compat.ThinkingFormat = "deepseek"

	if model.Input[0] != "text" {
		t.Fatalf("original input mutated: %v", model.Input)
	}
	if model.Compat.ThinkingFormat != "anthropic" {
		t.Fatalf("original compat mutated: %s", model.Compat.ThinkingFormat)
	}
}

func TestChatHandler_SlashHelp(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body := `{"messages":[{"role":"user","content":"/help"}],"stream":false}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp ChatCompletionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.XCommand != "/help" {
		t.Errorf("x_command = %q, want /help", resp.XCommand)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		t.Fatal("missing choice")
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "/clear") {
		t.Error("help output should mention /clear")
	}
}

func TestChatHandler_SlashClear(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	sess, err := srv.getOrCreateSession("test-sess", srv.cfg.GetWorkDir())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := sess.Manager.AppendMessage(provider.NewUserMessage("hello")); err != nil {
		t.Fatalf("append message: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"/clear"}],"stream":false,"x_session_id":"test-sess"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp ChatCompletionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.XCommand != "/clear" {
		t.Errorf("x_command = %q, want /clear", resp.XCommand)
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "Conversation cleared") {
		t.Errorf("expected clear confirmation, got %q", resp.Choices[0].Message.Content)
	}
	if resp.XSessionID != "test-sess" {
		t.Fatalf("clear response session ID = %q, want test-sess", resp.XSessionID)
	}
	if srv.pool.Count() != 1 {
		t.Fatalf("pool count = %d, want 1", srv.pool.Count())
	}
	cleared := srv.pool.Get("test-sess")
	if cleared == nil || cleared.Manager == nil {
		t.Fatal("expected cleared session in pool")
	}
	if msgs := cleared.Manager.GetMessages(); len(msgs) != 0 {
		t.Fatalf("clear should reset messages, got %d", len(msgs))
	}
}

func TestChatHandler_SlashMode(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body := `{"messages":[{"role":"user","content":"/mode plan"}],"stream":false,"x_session_id":"mode-sess"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp ChatCompletionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Choices[0].Message.Content, "PLAN") {
		t.Errorf("expected PLAN in response, got %q", resp.Choices[0].Message.Content)
	}
}

func TestChatHandler_EmptyMessages(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body := `{"messages":[]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestChatHandler_InvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestChatHandler_WorkDirForbidden(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	// Set restrictive allowedWorkDirs
	allowed := []string{"/opt/allowed"}
	srv.cfg.AllowedWorkDirs = &allowed

	body := `{"messages":[{"role":"user","content":"hi"}],"x_working_dir":"/etc/evil"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

// --- Commands tests ---

func TestCommands_UnknownCommand(t *testing.T) {
	srv := newTestServer(t)
	result := srv.handleCommand(nil, "/foobar")
	if result == nil {
		t.Fatal("expected result for unknown command")
	}
	if !result.Error {
		t.Error("expected error=true for unknown command")
	}
}

func TestCommands_NotACommand(t *testing.T) {
	srv := newTestServer(t)
	result := srv.handleCommand(nil, "hello world")
	if result != nil {
		t.Error("non-command should return nil")
	}
}

func TestCommands_Status(t *testing.T) {
	srv := newTestServer(t)
	sess := &APISession{ID: "test-sess", WorkDir: "/tmp", Mode: "agent"}
	result := srv.cmdStatus(sess)
	if result == nil {
		t.Fatal("expected result")
	}
	if !strings.Contains(result.Message, "AGENT") {
		t.Errorf("status should show mode, got %q", result.Message)
	}
	if !strings.Contains(result.Message, "test-sess") {
		t.Errorf("status should show session ID, got %q", result.Message)
	}
}

func TestCommands_CompactNoSession(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdCompact(nil)
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.Error {
		t.Error("expected error for nil session")
	}
}

func TestCommands_CompactTooShort(t *testing.T) {
	srv := newTestServer(t)
	// Create a session with less than 2 messages
	sess := &APISession{ID: "test-sess", WorkDir: "/tmp"}
	mgr := session.New(t.TempDir(), t.TempDir())
	mgr.Init()
	sess.Manager = mgr
	result := srv.cmdCompact(sess)
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.Error {
		t.Error("expected error for too-short conversation")
	}
	if !strings.Contains(result.Message, "too short") {
		t.Errorf("expected 'too short' message, got %q", result.Message)
	}
}

func TestCommands_CompactSetsFlag(t *testing.T) {
	srv := newTestServer(t)
	srv.settings.Compaction.KeepRecentTokens = 1
	sess := &APISession{ID: "test-sess", WorkDir: t.TempDir()}
	mgr := session.New(sess.WorkDir, t.TempDir())
	mgr.Init()
	// Append enough history so there is an older turn to summarize.
	mgr.AppendMessage(provider.NewUserMessage("old hello"))
	mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old hi"}}))
	mgr.AppendMessage(provider.NewUserMessage("recent hello"))
	mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent hi"}}))
	sess.Manager = mgr

	result := srv.cmdCompact(sess)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error {
		t.Errorf("unexpected error: %s", result.Message)
	}
	if !sess.ForceCompact {
		t.Error("expected ForceCompact to be set")
	}
	if !strings.Contains(result.Message, "compaction") {
		t.Errorf("expected compaction confirmation, got %q", result.Message)
	}
}

func TestCommands_RuleCreatesFile(t *testing.T) {
	srv := newTestServer(t)
	workDir := t.TempDir()
	sess := &APISession{ID: "test-sess", WorkDir: workDir}

	result := srv.cmdRule(sess, []string{"/rule"})
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Error {
		t.Fatalf("unexpected error: %s", result.Message)
	}
	rulePath := contextfiles.RuleFilePath(workDir)
	data, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read rule file: %v", err)
	}
	if string(data) != sess.RuleContent {
		t.Fatal("session RuleContent does not match file content")
	}
	if !strings.Contains(sess.RuleContent, "Never use sudo") {
		t.Fatalf("unexpected rule content: %q", sess.RuleContent)
	}
}

func TestCommands_RulePreservesExistingUnlessForced(t *testing.T) {
	srv := newTestServer(t)
	workDir := t.TempDir()
	rulePath := contextfiles.RuleFilePath(workDir)
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte("custom rule"), 0644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}
	sess := &APISession{ID: "test-sess", WorkDir: workDir}

	result := srv.cmdRule(sess, []string{"/rule"})
	if result == nil || result.Error {
		t.Fatalf("unexpected result: %#v", result)
	}
	if sess.RuleContent != "custom rule" {
		t.Fatalf("RuleContent = %q", sess.RuleContent)
	}
	data, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read rule file: %v", err)
	}
	if string(data) != "custom rule" {
		t.Fatalf("rule overwritten without force: %q", string(data))
	}

	result = srv.cmdRule(sess, []string{"/rule", "force"})
	if result == nil || result.Error {
		t.Fatalf("unexpected force result: %#v", result)
	}
	data, err = os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read forced rule file: %v", err)
	}
	if string(data) != sess.RuleContent || !strings.Contains(string(data), "Treat repository files") {
		t.Fatalf("force did not write default rule: %q", string(data))
	}
}

func TestCommands_CompactNoCompactableMessages(t *testing.T) {
	srv := newTestServer(t)
	sess := &APISession{ID: "test-sess", WorkDir: t.TempDir()}
	mgr := session.New(sess.WorkDir, t.TempDir())
	mgr.Init()
	mgr.AppendMessage(provider.NewUserMessage("hello"))
	mgr.AppendMessage(provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "hi"}}))
	sess.Manager = mgr

	result := srv.cmdCompact(sess)
	if result == nil {
		t.Fatal("expected result")
	}
	if !result.Error {
		t.Fatal("expected error for non-compactable conversation")
	}
	if sess.ForceCompact {
		t.Fatal("ForceCompact should not be set for non-compactable conversation")
	}
	if !strings.Contains(result.Message, "only recent context") {
		t.Errorf("expected only recent context message, got %q", result.Message)
	}
}

func TestChatHandler_SlashCompact(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body := `{"messages":[{"role":"user","content":"/compact"}],"stream":false,"x_session_id":"compact-sess"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp ChatCompletionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.XCommand != "/compact" {
		t.Errorf("x_command = %q, want /compact", resp.XCommand)
	}
}

func TestChatHandler_LoadsReplayStateFromSession(t *testing.T) {
	srv, p := newRecordingAPIServer(t)
	defer srv.pool.Stop()

	workDir := t.TempDir()
	mgr := session.New(workDir, srv.settings.SessionDir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	oldUser := provider.NewUserMessage("old user context")
	oldAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant context"}})
	recentUser := provider.NewUserMessage("recent user context")
	recentAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant context"}})
	_, _ = mgr.AppendMessage(oldUser)
	_, _ = mgr.AppendMessage(oldAssistant)
	recentUserID, _ := mgr.AppendMessage(recentUser)
	_, _ = mgr.AppendMessage(recentAssistant)
	_, _ = mgr.AppendCompaction("## Goal\ncompacted checkpoint", recentUserID, 100)

	registry := tools.NewRegistry(workDir, sandbox.NewNoneSandbox())
	sess := &APISession{
		ID:       "replay-sess",
		WorkDir:  workDir,
		Manager:  mgr,
		Registry: registry,
		LastUsed: time.Now(),
	}
	if err := srv.pool.Put(sess); err != nil {
		t.Fatalf("pool put: %v", err)
	}

	body := `{"messages":[{"role":"user","content":"continue"}],"stream":false,"x_session_id":"replay-sess"}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(p.calls) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(p.calls))
	}

	foundSummary := false
	foundOldUser := false
	foundRecentUser := false
	for _, msg := range p.calls[0].Messages {
		if msg.SystemInjected && msg.Content == "## Goal\ncompacted checkpoint" {
			foundSummary = true
		}
		if msg.Content == oldUser.Content {
			foundOldUser = true
		}
		if msg.Content == recentUser.Content {
			foundRecentUser = true
		}
	}

	if !foundSummary {
		t.Fatal("API run did not replay compacted summary")
	}
	if foundOldUser {
		t.Fatal("API run still included pre-compaction old user message")
	}
	if !foundRecentUser {
		t.Fatal("API run lost recent user message from replay state")
	}
}

func TestChatHandlerUsesSessionRuleContent(t *testing.T) {
	srv, p := newRecordingAPIServer(t)
	defer srv.pool.Stop()

	workDir := t.TempDir()
	rulePath := contextfiles.RuleFilePath(workDir)
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte("session-specific rule"), 0644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	body := fmt.Sprintf(`{"messages":[{"role":"user","content":"hi"}],"stream":false,"x_session_id":"rule-sess","x_working_dir":%q}`, workDir)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(p.calls) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(p.calls))
	}
	if !strings.Contains(p.calls[0].SystemPrompt, "session-specific rule") {
		t.Fatalf("system prompt did not include session rule:\n%s", p.calls[0].SystemPrompt)
	}
}

func TestChatHandlerDoesNotFallbackToOtherWorkDirRule(t *testing.T) {
	srv, p := newRecordingAPIServer(t)
	defer srv.pool.Stop()

	serverWorkDir := srv.cfg.GetWorkDir()
	serverRulePath := contextfiles.RuleFilePath(serverWorkDir)
	if err := os.MkdirAll(filepath.Dir(serverRulePath), 0755); err != nil {
		t.Fatalf("mkdir server rule dir: %v", err)
	}
	if err := os.WriteFile(serverRulePath, []byte("server-only rule"), 0644); err != nil {
		t.Fatalf("write server rule file: %v", err)
	}
	workDir := t.TempDir()

	body := fmt.Sprintf(`{"messages":[{"role":"user","content":"hi"}],"stream":false,"x_session_id":"no-rule-sess","x_working_dir":%q}`, workDir)
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(p.calls) != 1 {
		t.Fatalf("provider call count = %d, want 1", len(p.calls))
	}
	if strings.Contains(p.calls[0].SystemPrompt, "server-only rule") {
		t.Fatalf("system prompt used another workDir rule:\n%s", p.calls[0].SystemPrompt)
	}
}

// --- Tool format tests ---

func TestFormatToolExpanded_Read(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "read",
		Args:   map[string]any{"path": "main.go"},
		Status: "completed",
		Result: "package main\n\nfunc main() {}\n",
	}
	text := formatToolExpanded(tc)
	// Markdown header
	if !strings.Contains(text, "🔧 read: main.go") {
		t.Errorf("missing markdown header: %s", text)
	}
	// Code fence with language
	if !strings.Contains(text, "```go\n") {
		t.Errorf("missing go code fence: %s", text)
	}
	if !strings.Contains(text, "package main") {
		t.Errorf("missing result content: %s", text)
	}
	// Closing fence
	if !strings.Contains(text, "\n```") {
		t.Errorf("missing closing fence: %s", text)
	}
}

func TestFormatToolExpanded_Bash(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "bash",
		Args:   map[string]any{"command": "go test ./..."},
		Status: "completed",
		Result: "ok  pkg 0.5s\n",
	}
	text := formatToolExpanded(tc)
	if !strings.Contains(text, "🔧 bash: go test ./...") {
		t.Errorf("missing markdown header: %s", text)
	}
	if !strings.Contains(text, "```bash\n") {
		t.Errorf("missing bash code fence: %s", text)
	}
	if !strings.Contains(text, "ok  pkg") {
		t.Errorf("missing stdout: %s", text)
	}
}

func TestFormatToolExpanded_EditWithDiff(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "edit",
		Args:   map[string]any{"path": "main.go"},
		Status: "completed",
		Diff:   &tools.FileDiff{Path: "main.go", Added: 2, Deleted: 1, Unified: "+func new1() {}\n-func old() {}\n"},
	}
	text := formatToolExpanded(tc)
	if !strings.Contains(text, "```diff\n") {
		t.Errorf("missing diff code fence: %s", text)
	}
	if !strings.Contains(text, "+func new1") {
		t.Errorf("missing diff content: %s", text)
	}
}

func TestFormatToolExpanded_Error(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "bash",
		Args:   map[string]any{"command": "false"},
		Status: "failed",
		Error:  fmt.Errorf("exit code 1"),
	}
	text := formatToolExpanded(tc)
	if !strings.Contains(text, "Error: exit code 1") {
		t.Errorf("missing error: %s", text)
	}
}

func TestFormatToolExpanded_ReadJSON(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "read",
		Args:   map[string]any{"path": "package.json"},
		Status: "completed",
		Result: `{"name": "test"}`,
	}
	text := formatToolExpanded(tc)
	if !strings.Contains(text, "```json\n") {
		t.Errorf("should use json fence for .json file: %s", text)
	}
}

func TestFormatToolExpanded_GrepPlain(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "grep",
		Args:   map[string]any{"pattern": "TODO", "path": "./src"},
		Status: "completed",
		Result: "src/main.go:10: // TODO fix this\n",
	}
	text := formatToolExpanded(tc)
	// grep should use plain text fence (no language)
	if !strings.Contains(text, "```\n") {
		t.Errorf("grep should use plain code fence: %s", text)
	}
}

func TestFormatToolRunning(t *testing.T) {
	text := formatToolRunning("read", map[string]any{"path": "main.go"})
	if !strings.Contains(text, "\u23f3") {
		t.Errorf("missing hourglass: %s", text)
	}
	if !strings.Contains(text, "read") {
		t.Errorf("missing tool name: %s", text)
	}
}

func TestInferCodeLang(t *testing.T) {
	tests := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"bash", nil, "bash"},
		{"read", map[string]any{"path": "main.go"}, "go"},
		{"read", map[string]any{"path": "app.py"}, "python"},
		{"read", map[string]any{"path": "style.css"}, "css"},
		{"read", map[string]any{"path": "Makefile"}, "makefile"},
		{"read", map[string]any{"path": "Dockerfile"}, "dockerfile"},
		{"read", map[string]any{"path": "data.json"}, "json"},
		{"grep", map[string]any{"pattern": "x"}, ""},
		{"ls", nil, ""},
	}
	for _, tt := range tests {
		got := inferCodeLang(tt.tool, tt.args)
		if got != tt.want {
			t.Errorf("inferCodeLang(%q, %v) = %q, want %q", tt.tool, tt.args, got, tt.want)
		}
	}
}

func TestToolKeyArg(t *testing.T) {
	tests := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"read path", "read", map[string]any{"path": "main.go"}, "main.go"},
		{"bash command", "bash", map[string]any{"command": "ls -la"}, "ls -la"},
		{"grep", "grep", map[string]any{"pattern": "TODO", "path": "src/"}, "TODO src/"},
		{"nil args", "read", nil, ""},
		{"unknown tool", "foo", map[string]any{"name": "bar"}, "bar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolKeyArg(tt.tool, tt.args)
			if got != tt.want {
				t.Errorf("toolKeyArg(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestChatHandler_SlashHelp_Streaming(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body := `{"messages":[{"role":"user","content":"/help"}],"stream":true}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	resBody := w.Body.String()
	if !strings.Contains(resBody, "data: ") {
		t.Error("streaming response should contain SSE data lines")
	}
	if !strings.Contains(resBody, "[DONE]") {
		t.Error("streaming response should end with [DONE]")
	}
	if !strings.Contains(resBody, "/clear") {
		t.Error("help content should mention /clear")
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}

// --- Collapsed mode tests ---

func TestFormatToolCollapsed_Read(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "read",
		Args:   map[string]any{"path": "main.go"},
		Status: "completed",
		Result: "package main\n\nfunc main() {}\n",
	}
	text := formatToolCollapsed(tc)
	if !strings.Contains(text, "read") {
		t.Errorf("missing tool name: %s", text)
	}
	if !strings.Contains(text, "main.go") {
		t.Errorf("missing path: %s", text)
	}
	if !strings.Contains(text, "✅") {
		t.Errorf("missing success marker: %s", text)
	}
	// Should NOT contain the file content
	if strings.Contains(text, "package main") {
		t.Errorf("collapsed should not contain file content: %s", text)
	}
	if strings.Contains(text, "```") {
		t.Errorf("collapsed should not contain code fences: %s", text)
	}
}

func TestFormatToolCollapsed_EditShowsDiff(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "edit",
		Args:   map[string]any{"path": "main.go"},
		Status: "completed",
		Diff:   &tools.FileDiff{Path: "main.go", Added: 1, Deleted: 1, Unified: "+new line\n-old line\n"},
	}
	text := formatToolCollapsed(tc)
	// edit with diff should always show the diff even in collapsed mode
	if !strings.Contains(text, "```diff") {
		t.Errorf("collapsed edit should show diff fence: %s", text)
	}
	if !strings.Contains(text, "+new line") {
		t.Errorf("collapsed edit should show diff content: %s", text)
	}
}

func TestFormatToolCollapsed_ErrorAlwaysShown(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "bash",
		Args:   map[string]any{"command": "false"},
		Status: "failed",
		Error:  fmt.Errorf("exit code 1"),
	}
	text := formatToolCollapsed(tc)
	if !strings.Contains(text, "Error: exit code 1") {
		t.Errorf("collapsed error should always show: %s", text)
	}
}

func TestFormatToolCollapsed_BashNoOutput(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "bash",
		Args:   map[string]any{"command": "go test ./..."},
		Status: "completed",
		Result: "ok  pkg 0.5s\n",
	}
	text := formatToolCollapsed(tc)
	if !strings.Contains(text, "✅") {
		t.Errorf("missing success marker: %s", text)
	}
	if strings.Contains(text, "ok  pkg") {
		t.Errorf("collapsed bash should not show stdout: %s", text)
	}
}

// --- Dispatcher test ---

func TestFormatToolResult_Dispatches(t *testing.T) {
	tc := &toolCallInfo{
		Name:   "read",
		Args:   map[string]any{"path": "main.go"},
		Status: "completed",
		Result: "package main\n",
	}

	collapsed := formatToolResult(tc, "collapsed")
	expanded := formatToolResult(tc, "expanded")

	if strings.Contains(collapsed, "```go") {
		t.Error("collapsed should not have code fence")
	}
	if !strings.Contains(expanded, "```go") {
		t.Error("expanded should have code fence")
	}
}

func TestToolVisibility_DefaultDetail(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GetToolDetail() != "collapsed" {
		t.Errorf("default detail = %q, want collapsed", cfg.GetToolDetail())
	}
}

// --- CORS middleware disabled test ---

func TestCORSMiddleware_Disabled(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORSMiddleware(CORSConfig{Enabled: false}, inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	// CORS headers should NOT be set
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("CORS origin should be empty, got %q", got)
	}
}

func TestCORSMiddleware_DefaultOrigins(t *testing.T) {
	handler := CORSMiddleware(CORSConfig{Enabled: true}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want *", got)
	}
}

func TestAPISecurityWarning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Listen = ":8080"
	cfg.DefaultMode = "yolo"
	cfg.Auth.Enabled = false
	if got := apiSecurityWarning(cfg); got == "" {
		t.Fatal("expected warning for public yolo API without auth")
	}

	cfg.Listen = "127.0.0.1:8080"
	if got := apiSecurityWarning(cfg); got != "" {
		t.Fatalf("warning for loopback = %q, want empty", got)
	}

	cfg.Listen = ":8080"
	cfg.Auth.Enabled = true
	if got := apiSecurityWarning(cfg); got == "" {
		t.Fatal("expected warning when auth is enabled without tokens")
	}

	cfg.Auth.Tokens = []string{"sk-test"}
	if got := apiSecurityWarning(cfg); got != "" {
		t.Fatalf("warning with auth = %q, want empty", got)
	}
}

// --- Concurrency middleware at capacity test ---

func TestConcurrencyMiddleware_AtCapacity(t *testing.T) {
	blocking := make(chan struct{})
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocking // block until released
		w.WriteHeader(http.StatusOK)
	})
	handler := ConcurrencyMiddleware(1, inner)

	// Fill the single slot
	go func() {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}()

	// Give goroutine time to start
	time.Sleep(20 * time.Millisecond)

	// Second request should be rejected
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}

	// Release the blocking goroutine
	close(blocking)
}

// --- Auth with non-Bearer prefix ---

func TestAuthMiddleware_NonBearerPrefix(t *testing.T) {
	handler := AuthMiddleware(AuthConfig{Enabled: true, Tokens: []string{"sk-test"}}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// --- extractBearerToken tests ---

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{"empty", "", ""},
		{"bearer", "Bearer sk-test", "sk-test"},
		{"bearer with spaces", "Bearer  sk-test ", "sk-test"},
		{"basic", "Basic dXNlcjpwYXNz", ""},
		{"no prefix", "sk-test", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			got := extractBearerToken(req)
			if got != tt.want {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tt.auth, got, tt.want)
			}
		})
	}
}

// --- SessionPool advanced tests ---

func TestSessionPool_ReplaceSameID(t *testing.T) {
	pool := NewSessionPool(1, 0)
	defer pool.Stop()

	sess1 := &APISession{ID: "sess-1", WorkDir: "/tmp/a", LastUsed: time.Now()}
	if err := pool.Put(sess1); err != nil {
		t.Fatalf("put 1: %v", err)
	}

	// Replace same ID within the same workDir should succeed even at max capacity.
	sess1v2 := &APISession{ID: "sess-1", WorkDir: "/tmp/a", LastUsed: time.Now()}
	if err := pool.Put(sess1v2); err != nil {
		t.Fatalf("replace same ID should succeed: %v", err)
	}

	got := pool.Get("sess-1")
	if got.WorkDir != "/tmp/a" {
		t.Errorf("workdir = %q, want /tmp/a", got.WorkDir)
	}
}

func TestSessionPool_EvictIdle(t *testing.T) {
	pool := NewSessionPool(0, 50*time.Millisecond)
	defer pool.Stop()

	sess := &APISession{ID: "sess-1", LastUsed: time.Now()}
	pool.Put(sess)
	// Manually backdate LastUsed after Put (which calls Touch)
	sess.LastUsed = time.Now().Add(-time.Hour)

	pool.evictIdle()

	if pool.Get("sess-1") != nil {
		t.Error("idle session should be evicted")
	}
}

func TestSessionPool_EvictIdleKeepsFresh(t *testing.T) {
	pool := NewSessionPool(0, time.Hour)
	defer pool.Stop()

	sess := &APISession{ID: "sess-1", LastUsed: time.Now()}
	pool.Put(sess)

	pool.evictIdle()

	if pool.Get("sess-1") == nil {
		t.Error("fresh session should not be evicted")
	}
}

func TestPoolFullError_Error(t *testing.T) {
	e := &PoolFullError{Max: 5}
	if e.Error() != "session pool is at capacity" {
		t.Errorf("error = %q", e.Error())
	}
}

// --- parseMessages advanced tests ---

func TestParseMessages_MultipleSystem(t *testing.T) {
	msgs := []RequestMessage{
		{Role: "system", Content: "sys1"},
		{Role: "system", Content: "sys2"},
		{Role: "user", Content: "hello"},
	}
	lastUser, sysMsgs, history := parseMessages(msgs)
	if lastUser.Content != "hello" {
		t.Errorf("lastUser = %q", lastUser.Content)
	}
	if len(sysMsgs) != 2 {
		t.Errorf("sysMsgs len = %d, want 2", len(sysMsgs))
	}
	if len(history) != 0 {
		t.Errorf("history len = %d, want 0", len(history))
	}
}

func TestParseMessages_SingleUser(t *testing.T) {
	msgs := []RequestMessage{
		{Role: "user", Content: "only message"},
	}
	lastUser, sysMsgs, history := parseMessages(msgs)
	if lastUser.Content != "only message" {
		t.Errorf("lastUser = %q", lastUser.Content)
	}
	if len(sysMsgs) != 0 {
		t.Errorf("sysMsgs len = %d", len(sysMsgs))
	}
	if len(history) != 0 {
		t.Errorf("history len = %d", len(history))
	}
}

// --- convertHistoryMessages tests ---

func TestConvertHistoryMessages(t *testing.T) {
	msgs := []RequestMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
		{Role: "system", Content: "ignored"},
	}
	result := convertHistoryMessages(msgs)
	if len(result) != 2 {
		t.Fatalf("result len = %d, want 2", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("result[0].Role = %q", result[0].Role)
	}
	if result[1].Role != "assistant" {
		t.Errorf("result[1].Role = %q", result[1].Role)
	}
}

func TestConvertHistoryMessages_Empty(t *testing.T) {
	result := convertHistoryMessages(nil)
	if len(result) != 0 {
		t.Errorf("result len = %d, want 0", len(result))
	}
}

// --- resolveToolEvent tests ---

func TestResolveToolEvent_FromTopLevel(t *testing.T) {
	ev := agent.Event{
		ToolName:   "read",
		ToolCallID: "call-1",
	}
	name, callID := resolveToolEvent(ev)
	if name != "read" {
		t.Errorf("name = %q", name)
	}
	if callID != "call-1" {
		t.Errorf("callID = %q", callID)
	}
}

func TestResolveToolEvent_FallbackToToolCall(t *testing.T) {
	ev := agent.Event{
		ToolCall: &provider.ToolCallBlock{
			ID:   "call-2",
			Name: "bash",
		},
	}
	name, callID := resolveToolEvent(ev)
	if name != "bash" {
		t.Errorf("name = %q", name)
	}
	if callID != "call-2" {
		t.Errorf("callID = %q", callID)
	}
}

func TestResolveToolEvent_TopLevelTakesPrecedence(t *testing.T) {
	ev := agent.Event{
		ToolName:   "read",
		ToolCallID: "call-1",
		ToolCall: &provider.ToolCallBlock{
			ID:   "call-2",
			Name: "bash",
		},
	}
	name, callID := resolveToolEvent(ev)
	if name != "read" {
		t.Errorf("name = %q, want read", name)
	}
	if callID != "call-1" {
		t.Errorf("callID = %q, want call-1", callID)
	}
}

// --- Commands: mode/model/sessions edge cases ---

func TestCommands_ModeInvalid(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdMode(nil, []string{"/mode", "invalid"})
	if !result.Error {
		t.Error("expected error for invalid mode")
	}
}

func TestCommands_ModeShowCurrent(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdMode(nil, []string{"/mode"})
	if result.Error {
		t.Error("unexpected error")
	}
	if !strings.Contains(result.Message, "YOLO") {
		t.Errorf("expected current mode YOLO, got %q", result.Message)
	}
}

func TestCommands_ModeShowSessionOverride(t *testing.T) {
	srv := newTestServer(t)
	sess := &APISession{ID: "s1", Mode: "plan"}
	result := srv.cmdMode(sess, []string{"/mode"})
	if !strings.Contains(result.Message, "PLAN") {
		t.Errorf("expected PLAN, got %q", result.Message)
	}
}

func TestCommands_ModelNotFound(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdModel([]string{"/model", "nonexistent"})
	if !result.Error {
		t.Error("expected error for unknown model")
	}
}

func TestCommands_ModelShowCurrent(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdModel([]string{"/model"})
	if result.Error {
		t.Error("unexpected error")
	}
	if !strings.Contains(result.Message, "Model 1") {
		t.Errorf("expected Model 1, got %q", result.Message)
	}
}

func TestCommands_SessionsList(t *testing.T) {
	srv := newTestServer(t)
	workDir := srv.cfg.GetWorkDir()
	srv.pool.Put(&APISession{ID: "s1", WorkDir: workDir, LastUsed: time.Now()})
	srv.pool.Put(&APISession{ID: "s2", WorkDir: workDir, LastUsed: time.Now()})

	result := srv.cmdSessions([]string{"/sessions"})
	if result.Error {
		t.Error("unexpected error")
	}
	if !strings.Contains(result.Message, "s1") || !strings.Contains(result.Message, "s2") {
		t.Errorf("expected both session IDs, got %q", result.Message)
	}
}

func TestCommands_SessionsEmpty(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSessions([]string{"/sessions"})
	if !strings.Contains(result.Message, "No active sessions") {
		t.Errorf("expected no sessions message, got %q", result.Message)
	}
}

func TestCommands_SessionsDelete(t *testing.T) {
	srv := newTestServer(t)
	current := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := current.Init(); err != nil {
		t.Fatalf("init current session: %v", err)
	}
	target := session.New(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir())
	if err := target.Init(); err != nil {
		t.Fatalf("init target session: %v", err)
	}
	result := srv.cmdSessionsForSession(&APISession{ID: current.GetHeader().ID, Manager: current}, []string{"/sessions", "del", target.GetHeader().ID[:8]})
	if result.Error {
		t.Error("unexpected error")
	}
	if sessions, err := session.ListForDir(srv.cfg.GetWorkDir(), srv.settings.GetSessionDir()); err != nil {
		t.Fatalf("list sessions: %v", err)
	} else if len(sessions) != 1 {
		t.Fatalf("expected 1 session remaining, got %d", len(sessions))
	}
	if srv.pool.Get(target.GetHeader().ID) != nil {
		t.Error("deleted session should not remain in pool")
	}
}

func TestCommands_SessionsDeleteNotFound(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSessions([]string{"/sessions", "del", "nonexistent"})
	if !result.Error {
		t.Error("expected error for missing session")
	}
}

func TestCommands_SessionsDeleteMissingID(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSessions([]string{"/sessions", "del"})
	if !result.Error {
		t.Error("expected error for missing ID")
	}
}

func TestGetOrCreateSessionConcurrentDefaultReuse(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := srv.getOrCreateSession("", srv.cfg.GetWorkDir())
			if err != nil {
				errCh <- err
				return
			}
			if sess == nil || sess.ID == "" {
				errCh <- fmt.Errorf("missing session")
				return
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent getOrCreateSession failed: %v", err)
		}
	}
	if srv.pool.Count() != 1 {
		t.Fatalf("pool count = %d, want 1", srv.pool.Count())
	}
}

func TestCommands_SessionsUnknownSubcmd(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSessions([]string{"/sessions", "badcmd"})
	if !result.Error {
		t.Error("expected error for unknown subcmd")
	}
}

func TestCommands_StatusNoSession(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdStatus(nil)
	if !result.Error {
		t.Error("expected error for nil session")
	}
}

func TestCommands_SkillNoManager(t *testing.T) {
	srv := newTestServer(t)
	srv.skillsMgr = nil
	result := srv.cmdSkill(nil, []string{"/skill", "test"})
	if !result.Error {
		t.Error("expected error when no skills manager")
	}
}

func TestCommands_SkillNotFound(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSkill(nil, []string{"/skill", "nonexistent"})
	if !result.Error {
		t.Error("expected error for unknown skill")
	}
}

func TestCommands_SkillsEmpty(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdSkills(nil)
	if !strings.Contains(result.Message, "No skills found") {
		t.Errorf("expected no skills message, got %q", result.Message)
	}
}

func TestAPISessionCreatesAndActivatesWorkflowSkillForWorkDir(t *testing.T) {
	srv := newTestServer(t)
	workDir := t.TempDir()
	srv.cfg.EnableWorkflows = true

	sess, err := srv.getOrCreateSession("workflow-sess", workDir)
	if err != nil {
		t.Fatalf("getOrCreateSession() error = %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
	skillPath := filepath.Join(workDir, ".skills", workflow.SkillName, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("expected workflow skill at %s: %v", skillPath, err)
	}
	if sess.SkillsMgr == nil || sess.SkillsMgr.Get(workflow.SkillName) == nil {
		t.Fatal("expected session skills manager to load workflow skill")
	}
	if !strings.Contains(sess.ExtraContext, "## Active Skill: "+workflow.SkillName) {
		t.Fatalf("expected workflow skill to be active in session context")
	}
}

func TestCommands_Help(t *testing.T) {
	srv := newTestServer(t)
	result := srv.cmdHelp()
	for _, cmd := range []string{"/clear", "/mode", "/model", "/compact", "/workflows", "/help"} {
		if !strings.Contains(result.Message, cmd) {
			t.Errorf("help missing %s", cmd)
		}
	}
}

func TestCommands_WorkflowsCancelActiveRun(t *testing.T) {
	srv := newTestServer(t)
	active := workflow.DefaultActiveRegistry()
	canceled := false
	if err := active.Register("API-run", func() { canceled = true }); err != nil {
		t.Fatalf("register active workflow: %v", err)
	}
	defer active.Unregister("API-run")

	result := srv.cmdWorkflows([]string{"/workflows", "cancel", "API-run"})
	if result.Error {
		t.Fatalf("expected cancel success, got %q", result.Message)
	}
	if !canceled {
		t.Fatal("expected cancel function to be called")
	}
}

// --- Chat handler method-not-allowed test ---

func TestChatHandler_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	req := httptest.NewRequest("GET", "/v1/chat/completions", nil)
	w := httptest.NewRecorder()
	srv.handleChatCompletions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

// --- Type helper tests ---

func TestNewCompletionID(t *testing.T) {
	id := newCompletionID()
	if !strings.HasPrefix(id, "chatcmpl-") {
		t.Errorf("id = %q, want chatcmpl- prefix", id)
	}
}

func TestNewCommandCompletionID(t *testing.T) {
	id := newCommandCompletionID()
	if !strings.HasPrefix(id, "chatcmpl-cmd-") {
		t.Errorf("id = %q, want chatcmpl-cmd- prefix", id)
	}
}

func TestStringPtr(t *testing.T) {
	p := stringPtr("test")
	if *p != "test" {
		t.Errorf("*p = %q", *p)
	}
}

func TestMarshalJSON(t *testing.T) {
	data := marshalJSON(map[string]string{"key": "val"})
	if !strings.Contains(string(data), "key") {
		t.Errorf("data = %s", data)
	}
}

// --- langFromPath extended tests ---

func TestLangFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"app.ts", "typescript"},
		{"comp.tsx", "tsx"},
		{"comp.jsx", "jsx"},
		{"main.rs", "rust"},
		{"app.rb", "ruby"},
		{"Main.java", "java"},
		{"main.c", "c"},
		{"main.h", "c"},
		{"main.cpp", "cpp"},
		{"main.cc", "cpp"},
		{"main.cs", "csharp"},
		{"main.swift", "swift"},
		{"main.kt", "kotlin"},
		{"script.sh", "bash"},
		{"script.bash", "bash"},
		{"script.zsh", "zsh"},
		{"script.ps1", "powershell"},
		{"query.sql", "sql"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"style.scss", "scss"},
		{"data.json", "json"},
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"config.toml", "toml"},
		{"data.xml", "xml"},
		{"README.md", "markdown"},
		{"main.tf", "hcl"},
		{"main.lua", "lua"},
		{"main.php", "php"},
		{"main.pl", "perl"},
		{"main.ex", "elixir"},
		{"main.erl", "erlang"},
		{"main.hs", "haskell"},
		{"main.scala", "scala"},
		{"main.clj", "clojure"},
		{"main.vim", "vim"},
		{"schema.proto", "protobuf"},
		{"schema.graphql", "graphql"},
		{"config.ini", "ini"},
		{".env", "bash"},
		{"Makefile", "makefile"},
		{"Dockerfile", "dockerfile"},
		{"Gemfile", "ruby"},
		{"unknown.xyz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := langFromPath(tt.path)
			if got != tt.want {
				t.Errorf("langFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// --- formatToolHeaderMD tests ---

func TestFormatToolHeaderMD(t *testing.T) {
	got := formatToolHeaderMD("read", map[string]any{"path": "main.go"})
	if got != "🔧 read: main.go" {
		t.Errorf("got %q", got)
	}
	got2 := formatToolHeaderMD("plan", nil)
	if got2 != "🔧 plan" {
		t.Errorf("got %q", got2)
	}
}

// --- formatToolHeader tests ---

func TestFormatToolHeader(t *testing.T) {
	got := formatToolHeader("bash", map[string]any{"command": "ls"})
	if got != "🔧 [bash] ls" {
		t.Errorf("got %q", got)
	}
	got2 := formatToolHeader("plan", nil)
	if got2 != "🔧 [plan]" {
		t.Errorf("got %q", got2)
	}
}

// --- toolKeyArg: bash long command truncation ---

func TestToolKeyArg_BashLongCommand(t *testing.T) {
	longCmd := strings.Repeat("a", 200)
	got := toolKeyArg("bash", map[string]any{"command": longCmd})
	if len(got) > 124 { // 120 + "..."
		t.Errorf("expected truncated, got len %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("expected ... suffix")
	}
}

// --- APISession Touch/Lock ---

func TestAPISession_Touch(t *testing.T) {
	sess := &APISession{ID: "s1"}
	sess.Touch()
	if sess.LastUsed.IsZero() {
		t.Error("expected non-zero LastUsed after Touch")
	}
}

func TestAPISession_LockUnlock(t *testing.T) {
	sess := &APISession{ID: "s1"}
	sess.Lock()
	sess.Unlock()
	// No panic = pass
}

// --- Default session ID tests ---

func TestDefaultSessionID_EmptyReusesSession(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	// First request without x_session_id — should create a session
	body1 := `{"messages":[{"role":"user","content":"/status"}],"stream":false}`
	req1 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body1))
	w1 := httptest.NewRecorder()
	srv.handleChatCompletions(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("req1 status = %d, body = %s", w1.Code, w1.Body.String())
	}
	var resp1 ChatCompletionResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	sessID1 := resp1.XSessionID
	if sessID1 == "" {
		t.Fatal("first request should return a session ID")
	}

	// Second request without x_session_id — should reuse the same session
	body2 := `{"messages":[{"role":"user","content":"/status"}],"stream":false}`
	req2 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body2))
	w2 := httptest.NewRecorder()
	srv.handleChatCompletions(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("req2 status = %d", w2.Code)
	}
	var resp2 ChatCompletionResponse
	json.NewDecoder(w2.Body).Decode(&resp2)

	if resp2.XSessionID != sessID1 {
		t.Errorf("second request should reuse session: got %q, want %q", resp2.XSessionID, sessID1)
	}
}

func TestDefaultSessionID_ExplicitIDOverrides(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	// First request without x_session_id
	body1 := `{"messages":[{"role":"user","content":"/status"}],"stream":false}`
	req1 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body1))
	w1 := httptest.NewRecorder()
	srv.handleChatCompletions(w1, req1)
	var resp1 ChatCompletionResponse
	json.NewDecoder(w1.Body).Decode(&resp1)
	defaultID := resp1.XSessionID

	// Second request WITH explicit x_session_id — should use that, not default
	body2 := `{"messages":[{"role":"user","content":"/status"}],"stream":false,"x_session_id":"explicit-sess"}`
	req2 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body2))
	w2 := httptest.NewRecorder()
	srv.handleChatCompletions(w2, req2)
	var resp2 ChatCompletionResponse
	json.NewDecoder(w2.Body).Decode(&resp2)

	if resp2.XSessionID != "explicit-sess" {
		t.Errorf("explicit session should be used: got %q", resp2.XSessionID)
	}

	// Third request without x_session_id — should still use the default, not "explicit-sess"
	body3 := `{"messages":[{"role":"user","content":"/status"}],"stream":false}`
	req3 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body3))
	w3 := httptest.NewRecorder()
	srv.handleChatCompletions(w3, req3)
	var resp3 ChatCompletionResponse
	json.NewDecoder(w3.Body).Decode(&resp3)

	if resp3.XSessionID != defaultID {
		t.Errorf("third request should reuse default: got %q, want %q", resp3.XSessionID, defaultID)
	}
}

func TestDefaultSessionID_PoolCount(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	// Multiple requests without x_session_id should all share one session
	for i := 0; i < 5; i++ {
		body := `{"messages":[{"role":"user","content":"/help"}],"stream":false}`
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		w := httptest.NewRecorder()
		srv.handleChatCompletions(w, req)
	}

	if srv.pool.Count() != 1 {
		t.Errorf("pool count = %d, want 1 (all should share default session)", srv.pool.Count())
	}
}

func TestDefaultSessionID_IsolatedByWorkDir(t *testing.T) {
	srv := newTestServer(t)
	defer srv.pool.Stop()

	body1 := fmt.Sprintf(`{"messages":[{"role":"user","content":"/status"}],"stream":false,"x_working_dir":%q}`, filepath.Join(srv.cfg.GetWorkDir(), "a"))

	req1 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.handleChatCompletions(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("req1 status = %d, body = %s", w1.Code, w1.Body.String())
	}
	var resp1 ChatCompletionResponse
	if err := json.NewDecoder(w1.Body).Decode(&resp1); err != nil {
		t.Fatalf("decode resp1: %v", err)
	}

	body2 := fmt.Sprintf(`{"messages":[{"role":"user","content":"/status"}],"stream":false,"x_working_dir":%q}`, filepath.Join(srv.cfg.GetWorkDir(), "b"))

	req2 := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.handleChatCompletions(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("req2 status = %d, body = %s", w2.Code, w2.Body.String())
	}
	var resp2 ChatCompletionResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode resp2: %v", err)
	}

	if resp1.XSessionID == resp2.XSessionID {
		t.Fatalf("expected different default sessions for different workdirs, got %q", resp1.XSessionID)
	}
	if srv.pool.Count() != 2 {
		t.Fatalf("pool count = %d, want 2", srv.pool.Count())
	}
}
