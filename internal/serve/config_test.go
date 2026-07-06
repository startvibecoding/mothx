package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/gateway"
	"github.com/startvibecoding/mothx/internal/hermes"
)

type fakeActiveSessionManager struct {
	sessions  []gateway.ActiveSessionInfo
	messages  []gateway.SessionMessageEntry
	deletedID string
	deleted   bool
	err       error
}

func (f *fakeActiveSessionManager) ListActiveSessions() []gateway.ActiveSessionInfo {
	return append([]gateway.ActiveSessionInfo(nil), f.sessions...)
}

func (f *fakeActiveSessionManager) DeleteActiveSession(id string) (bool, error) {
	f.deletedID = id
	return f.deleted, f.err
}

func (f *fakeActiveSessionManager) GetSessionMessages(id string) ([]gateway.SessionMessageEntry, error) {
	return append([]gateway.SessionMessageEntry(nil), f.messages...), f.err
}

type fakeWebSocketRuntime struct {
	connections int
	status      int
	stopped     bool
	model       string
	workDir     string
}

func (f *fakeWebSocketRuntime) ConnectionCount() int {
	return f.connections
}

func (f *fakeWebSocketRuntime) Stop(time.Duration) error {
	f.stopped = true
	return nil
}

func (f *fakeWebSocketRuntime) SetClientInfo(model, workDir string) {
	f.model = model
	f.workDir = workDir
}

func (f *fakeWebSocketRuntime) WebSocketHandler() http.Handler {
	status := f.status
	if status == 0 {
		status = http.StatusOK
	}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})
}

func TestLoadConfigFrom_LegacyNestedSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.json")
	data := `{
		"gateway": {
			"listen": ":9090",
			"provider": "deepseek",
			"model": "deepseek-chat",
			"defaultMode": "agent",
			"sandbox": {"enabled": true, "level": "strict"},
			"enableSubAgents": true
		},
		"channels": {
			"wechat": {"enabled": true, "auto_typing": false},
			"feishu": {"enabled": true, "app_id": "app-id", "app_secret": "secret"}
		},
		"webUI": {"enabled": false, "dir": "custom-ui"},
		"memory": {"enabled": true},
		"agent": {"max_turns": 42}
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("write serve config: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}

	if cfg.Gateway.Listen != ":9090" {
		t.Fatalf("listen = %q, want :9090", cfg.Gateway.Listen)
	}
	if cfg.Gateway.Provider != "deepseek" {
		t.Fatalf("provider = %q, want deepseek", cfg.Gateway.Provider)
	}
	if cfg.Gateway.Model != "deepseek-chat" {
		t.Fatalf("model = %q, want deepseek-chat", cfg.Gateway.Model)
	}
	if cfg.Gateway.DefaultMode != "agent" {
		t.Fatalf("mode = %q, want agent", cfg.Gateway.DefaultMode)
	}
	if !cfg.Gateway.Sandbox.Enabled || cfg.Gateway.Sandbox.Level != "strict" {
		t.Fatalf("sandbox = %#v, want enabled strict", cfg.Gateway.Sandbox)
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("enableSubAgents should be true")
	}
	if !cfg.Channels.Wechat.Enabled || cfg.Channels.Wechat.AutoTyping {
		t.Fatalf("wechat config = %#v", cfg.Channels.Wechat)
	}
	if !cfg.Channels.Feishu.Enabled || cfg.Channels.Feishu.AppID != "app-id" || cfg.Channels.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu config = %#v", cfg.Channels.Feishu)
	}
	if cfg.WebUI.Enabled || cfg.WebUI.Dir != "custom-ui" {
		t.Fatalf("webUI config = %#v", cfg.WebUI)
	}
	if !cfg.Memory.Enabled {
		t.Fatal("memory should be enabled")
	}
	if cfg.Agent.MaxTurns != 42 {
		t.Fatalf("agent max turns = %d, want 42", cfg.Agent.MaxTurns)
	}
}

func TestLoadConfigFrom_FlatSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.json")
	data := `{
		"listen": ":7777",
		"provider": "openai",
		"model": "gpt-4o",
		"mode": "agent",
		"workDir": "/tmp/project",
		"thinking": "high",
		"auth": {
			"enabled": true,
			"tokens": ["sk-test"]
		},
		"features": {
			"webUI": true,
			"openaiAPI": true,
			"wechat": true,
			"feishu": true,
			"multiAgent": true,
			"memory": true
		},
		"sandbox": {
			"enabled": true,
			"level": "strict"
		},
		"allowedWorkDirs": ["/tmp/project"],
		"channels": {
			"wechat": {"autoTyping": false},
			"feishu": {"appId": "app-id", "appSecret": "secret"}
		},
		"session": {
			"idleTimeoutSeconds": 99,
			"maxSessions": 7
		},
		"toolVisibility": {"mode": "content", "detail": "expanded"},
		"systemPromptMode": "ignore",
		"requestTimeoutSeconds": 66,
		"maxConcurrentRequests": 5,
		"agent": {"maxTurns": 12}
	}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("write serve config: %v", err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom: %v", err)
	}

	if cfg.Gateway.Listen != ":7777" {
		t.Fatalf("listen = %q, want :7777", cfg.Gateway.Listen)
	}
	if cfg.Gateway.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", cfg.Gateway.Provider)
	}
	if cfg.Gateway.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", cfg.Gateway.Model)
	}
	if cfg.Gateway.DefaultMode != "agent" {
		t.Fatalf("mode = %q, want agent", cfg.Gateway.DefaultMode)
	}
	if cfg.Gateway.WorkingDir != "/tmp/project" {
		t.Fatalf("workDir = %q, want /tmp/project", cfg.Gateway.WorkingDir)
	}
	if !cfg.Gateway.Auth.Enabled || len(cfg.Gateway.Auth.Tokens) != 1 || cfg.Gateway.Auth.Tokens[0] != "sk-test" {
		t.Fatalf("auth = %#v", cfg.Gateway.Auth)
	}
	if !cfg.Gateway.Sandbox.Enabled || cfg.Gateway.Sandbox.Level != "strict" {
		t.Fatalf("sandbox = %#v", cfg.Gateway.Sandbox)
	}
	if cfg.Gateway.AllowedWorkDirs == nil || len(*cfg.Gateway.AllowedWorkDirs) != 1 || (*cfg.Gateway.AllowedWorkDirs)[0] != "/tmp/project" {
		t.Fatalf("allowedWorkDirs = %#v", cfg.Gateway.AllowedWorkDirs)
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("enableSubAgents should be true")
	}
	if !cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be true")
	}
	if cfg.Gateway.Session.IdleTimeoutSeconds != 99 || cfg.Gateway.Session.MaxSessions != 7 {
		t.Fatalf("session config = %#v", cfg.Gateway.Session)
	}
	if cfg.Gateway.ToolVisibility.Mode != "content" || cfg.Gateway.ToolVisibility.Detail != "expanded" {
		t.Fatalf("tool visibility = %#v", cfg.Gateway.ToolVisibility)
	}
	if cfg.Gateway.SystemPromptMode != "ignore" {
		t.Fatalf("system prompt mode = %q, want ignore", cfg.Gateway.SystemPromptMode)
	}
	if cfg.Gateway.RequestTimeoutSecs != 66 {
		t.Fatalf("timeout = %d, want 66", cfg.Gateway.RequestTimeoutSecs)
	}
	if cfg.Gateway.MaxConcurrentReqs != 5 {
		t.Fatalf("max concurrent = %d, want 5", cfg.Gateway.MaxConcurrentReqs)
	}
	if !cfg.Channels.Wechat.Enabled || cfg.Channels.Wechat.AutoTyping {
		t.Fatalf("wechat config = %#v", cfg.Channels.Wechat)
	}
	if !cfg.Channels.Feishu.Enabled || cfg.Channels.Feishu.AppID != "app-id" || cfg.Channels.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu config = %#v", cfg.Channels.Feishu)
	}
	if !cfg.WebUI.Enabled {
		t.Fatal("webUI should be enabled")
	}
	if !cfg.Memory.Enabled {
		t.Fatal("memory should be enabled")
	}
	if cfg.Agent.MaxTurns != 12 {
		t.Fatalf("agent max turns = %d, want 12", cfg.Agent.MaxTurns)
	}
}

func TestDecodeConfigBytes_FlatSchema(t *testing.T) {
	data := []byte(`{
		"listen": ":8081",
		"features": {
			"webUI": false,
			"openaiAPI": false
		},
		"channels": {
			"wechat": {"autoTyping": false}
		}
	}`)

	cfg, err := DecodeConfigBytes(data)
	if err != nil {
		t.Fatalf("DecodeConfigBytes: %v", err)
	}
	if cfg.Gateway.Listen != ":8081" {
		t.Fatalf("listen = %q, want :8081", cfg.Gateway.Listen)
	}
	if cfg.Features.WebUI {
		t.Fatal("webUI feature should be false")
	}
	if cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be false")
	}
	if cfg.Channels.Wechat.AutoTyping {
		t.Fatal("wechat autoTyping should be false")
	}
}

func TestInitConfig_WritesFlatTemplate(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("MOTHX_CONFIG_DIR", "")
	t.Setenv("VIBECODING_DIR", "")

	path, err := InitConfig(false)
	if err != nil {
		t.Fatalf("InitConfig: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"listen": ":8080"`) {
		t.Fatalf("generated config missing flat listen field:\n%s", text)
	}
	if !strings.Contains(text, `"features": {`) {
		t.Fatalf("generated config missing features block:\n%s", text)
	}
	if !strings.Contains(text, `"openaiAPI": true`) {
		t.Fatalf("generated config missing openaiAPI feature flag:\n%s", text)
	}
	if !strings.Contains(text, `"auth": {`) {
		t.Fatalf("generated config missing auth block:\n%s", text)
	}
	if strings.Contains(text, `"gateway": {`) {
		t.Fatalf("generated config should prefer flat schema, got legacy gateway block:\n%s", text)
	}
}

func TestHandleChannels_ReturnsStableEntries(t *testing.T) {
	rt := &channelRuntime{
		cfg: &Config{
			Features: FeatureConfig{WebSocket: true},
			Channels: ChannelConfig{
				Wechat: hermes.WechatConfig{Enabled: true},
				Feishu: hermes.FeishuConfig{Enabled: false},
			},
		},
		wsGateway: &fakeWebSocketRuntime{connections: 2},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()
	rt.handleChannels(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var statuses []channelStatus
	if err := json.NewDecoder(w.Body).Decode(&statuses); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(statuses) != 3 {
		t.Fatalf("status count = %d, want 3", len(statuses))
	}
	if statuses[0] != (channelStatus{Name: "wechat", Enabled: true, Connected: false}) {
		t.Fatalf("wechat status = %#v", statuses[0])
	}
	if statuses[1] != (channelStatus{Name: "feishu", Enabled: false, Connected: false}) {
		t.Fatalf("feishu status = %#v", statuses[1])
	}
	if statuses[2] != (channelStatus{Name: "websocket", Enabled: true, Connected: true}) {
		t.Fatalf("websocket status = %#v", statuses[2])
	}
}

func TestRegisterServeRoutes_WebUIDisabledStillServesManageAPI(t *testing.T) {
	rt := &channelRuntime{
		cfg: &Config{
			Features: FeatureConfig{WebUI: false, OpenAIAPI: true},
			WebUI:    WebUIConfig{Enabled: false, Dir: "ui/dist"},
		},
	}

	mux := http.NewServeMux()
	registerServeRoutes(mux, rt, "/tmp/serve.json")

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("/api/channels status = %d, want 200", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("/ status = %d, want 404 when webUI disabled", w.Code)
	}
}

func TestRegisterServeRoutes_SessionsRequireGatewayServer(t *testing.T) {
	rt := &channelRuntime{cfg: DefaultConfig()}
	mux := http.NewServeMux()
	registerServeRoutes(mux, rt, "/tmp/serve.json")

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("/api/sessions status = %d, want 503", w.Code)
	}
}

func TestRegisterServeRoutes_WebSocketMounted(t *testing.T) {
	rt := &channelRuntime{
		cfg:       &Config{Features: FeatureConfig{WebSocket: true}, WebUI: WebUIConfig{Enabled: false}},
		wsGateway: &fakeWebSocketRuntime{status: http.StatusTeapot},
	}
	mux := http.NewServeMux()
	registerServeRoutes(mux, rt, "/tmp/serve.json")

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("/ws status = %d, want %d", w.Code, http.StatusTeapot)
	}
}

func TestHandleWebSocketDisabledReturnsNotFound(t *testing.T) {
	rt := &channelRuntime{
		cfg:       &Config{Features: FeatureConfig{WebSocket: false}},
		wsGateway: &fakeWebSocketRuntime{status: http.StatusTeapot},
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	rt.handleWebSocket(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleServeConfigPutSyncsWebSocketRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.json")
	ws := &fakeWebSocketRuntime{}
	cfg := DefaultConfig()
	cfg.Features.WebSocket = true
	rt := &channelRuntime{cfg: cfg, wsGateway: ws}

	req := httptest.NewRequest(http.MethodPut, "/api/serve/config", strings.NewReader(`{"features":{"websocket":false}}`))
	w := httptest.NewRecorder()
	rt.handleServeConfig(path).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.wsGateway != nil {
		t.Fatal("websocket runtime should be cleared when disabled")
	}
	if !ws.stopped {
		t.Fatal("websocket runtime should be stopped when disabled")
	}
}

func TestHandleServeConfigPutSyncsCronStorePath(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old-cron.json")
	newPath := filepath.Join(dir, "new-cron.json")
	cfg := DefaultConfig()
	cfg.Features.Cron = true
	cfg.Cron.Enabled = true
	cfg.Cron.StorePath = oldPath
	rt := &channelRuntime{cfg: cfg, cronStore: cron.NewFileCronStore(oldPath), cronStorePath: oldPath}

	body := fmt.Sprintf(`{"features":{"cron":true},"cron":{"enabled":true,"storePath":%q}}`, newPath)
	req := httptest.NewRequest(http.MethodPut, "/api/serve/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	rt.handleServeConfig(filepath.Join(dir, "serve.json")).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.cronStorePath != newPath {
		t.Fatalf("cronStorePath = %q, want %q", rt.cronStorePath, newPath)
	}
	if rt.cronStore == nil {
		t.Fatal("cron store should be available after config update")
	}
}

func TestHandleStatus_ReturnsRuntimeSummary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Gateway.Listen = "127.0.0.1:9090"
	cfg.Features.OpenAIAPI = false
	cfg.Features.Wechat = true
	cfg.Channels.Wechat.Enabled = true
	rt := &channelRuntime{cfg: cfg}
	sessions := &fakeActiveSessionManager{
		sessions: []gateway.ActiveSessionInfo{{ID: "s1"}, {ID: "s2"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	rt.handleStatus(sessions).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"openaiAPI":false`) || !strings.Contains(body, `"feishu":false`) {
		t.Fatalf("status response should include false feature flags: %s", body)
	}
	var got serveStatus
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "ok" || got.Listen != "127.0.0.1:9090" {
		t.Fatalf("status response = %#v", got)
	}
	if got.Features.OpenAIAPI || !got.Features.Wechat {
		t.Fatalf("features = %#v", got.Features)
	}
	if got.Sessions != 2 {
		t.Fatalf("sessions = %d, want 2", got.Sessions)
	}
	if len(got.Channels) != 3 || !got.Channels[0].Enabled || got.Channels[2].Name != "websocket" {
		t.Fatalf("channels = %#v", got.Channels)
	}
}

func TestHandleSessions_ReturnsActiveSessions(t *testing.T) {
	rt := &channelRuntime{cfg: DefaultConfig()}
	sessions := &fakeActiveSessionManager{
		sessions: []gateway.ActiveSessionInfo{{ID: "s1", WorkDir: "/tmp/a"}},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()
	rt.handleSessions(sessions).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got struct {
		Sessions []gateway.ActiveSessionInfo `json:"sessions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Sessions) != 1 || got.Sessions[0].ID != "s1" || got.Sessions[0].WorkDir != "/tmp/a" {
		t.Fatalf("sessions = %#v", got.Sessions)
	}
}

func TestHandleSessionByID_DeletesActiveSession(t *testing.T) {
	rt := &channelRuntime{cfg: DefaultConfig()}
	sessions := &fakeActiveSessionManager{deleted: true}

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/s1", nil)
	w := httptest.NewRecorder()
	rt.handleSessionByID(sessions).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if sessions.deletedID != "s1" {
		t.Fatalf("deletedID = %q, want s1", sessions.deletedID)
	}
	var got struct {
		ID      string `json:"id"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "s1" || !got.Deleted {
		t.Fatalf("response = %#v", got)
	}
}

func TestHandleMemoryDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.Memory = false
	rt := &channelRuntime{cfg: cfg}

	req := httptest.NewRequest(http.MethodGet, "/api/memory", nil)
	w := httptest.NewRecorder()
	rt.handleMemory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("GET body = %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/memory", strings.NewReader(`{"content":"# Memory"}`))
	w = httptest.NewRecorder()
	rt.handleMemory(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("PUT status = %d, want 403", w.Code)
	}
}

func TestHandleMemoryReadWrite(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.Memory = true
	cfg.Memory.Enabled = true
	cfg.Memory.Path = filepath.Join(t.TempDir(), "memory.md")
	rt := &channelRuntime{cfg: cfg}

	req := httptest.NewRequest(http.MethodPut, "/api/memory", strings.NewReader(`{"content":"# Memory\n\nhello"}`))
	w := httptest.NewRecorder()
	rt.handleMemory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", w.Code, w.Body.String())
	}
	if data, err := os.ReadFile(cfg.Memory.Path); err != nil || string(data) != "# Memory\n\nhello" {
		t.Fatalf("memory file = %q, err = %v", data, err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/memory", nil)
	w = httptest.NewRecorder()
	rt.handleMemory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", w.Code)
	}
	var got struct {
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
		Source  string `json:"source"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode memory response: %v", err)
	}
	if !got.Enabled || got.Path != cfg.Memory.Path || got.Source != "explicit" || got.Content != "# Memory\n\nhello" {
		t.Fatalf("memory response = %#v", got)
	}
}

func TestHandleCronDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.Cron = false
	rt := &channelRuntime{cfg: cfg}

	req := httptest.NewRequest(http.MethodGet, "/api/cron", nil)
	w := httptest.NewRecorder()
	rt.handleCron(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatalf("GET body = %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/cron", strings.NewReader(`{"name":"n","prompt":"p"}`))
	w = httptest.NewRecorder()
	rt.handleCron(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("POST status = %d, want 403", w.Code)
	}
}

func TestHandleCronCreateListUpdateDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve-cron.json")
	cfg := DefaultConfig()
	cfg.Features.Cron = true
	cfg.Cron.Enabled = true
	cfg.Cron.StorePath = path
	rt := &channelRuntime{cfg: cfg, cronStore: cron.NewFileCronStore(path), cronStorePath: path}

	req := httptest.NewRequest(http.MethodPost, "/api/cron", strings.NewReader(`{"name":"daily","prompt":"summarize","schedule":"@daily","mode":"agent"}`))
	w := httptest.NewRecorder()
	rt.handleCron(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", w.Code, w.Body.String())
	}
	var created struct {
		Job cron.CronJob `json:"job"`
	}
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode created job: %v", err)
	}
	if created.Job.ID == "" || created.Job.OneShot || created.Job.NextRun.IsZero() || created.Job.Mode != "agent" {
		t.Fatalf("created job = %#v", created.Job)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/cron", nil)
	w = httptest.NewRecorder()
	rt.handleCron(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", w.Code)
	}
	var status cronAPIResponse
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode cron status: %v", err)
	}
	if !status.Enabled || status.Path != path || len(status.Jobs) != 1 || status.Jobs[0].ID != created.Job.ID {
		t.Fatalf("cron status = %#v", status)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/cron/"+created.Job.ID, strings.NewReader(`{"enabled":false}`))
	w = httptest.NewRecorder()
	rt.handleCronByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, body = %s", w.Code, w.Body.String())
	}
	updated, err := rt.cronStore.Get(created.Job.ID)
	if err != nil {
		t.Fatalf("get updated job: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("job should be disabled: %#v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/cron/"+created.Job.ID, nil)
	w = httptest.NewRecorder()
	rt.handleCronByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, body = %s", w.Code, w.Body.String())
	}
	jobs, err := rt.cronStore.List()
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("jobs after delete = %#v", jobs)
	}
}

func TestHandleCronCreateRejectsInvalidSchedule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve-cron.json")
	cfg := DefaultConfig()
	cfg.Features.Cron = true
	cfg.Cron.Enabled = true
	cfg.Cron.StorePath = path
	rt := &channelRuntime{cfg: cfg, cronStore: cron.NewFileCronStore(path), cronStorePath: path}

	req := httptest.NewRequest(http.MethodPost, "/api/cron", strings.NewReader(`{"name":"bad","prompt":"run","schedule":"never"}`))
	w := httptest.NewRecorder()
	rt.handleCron(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

func TestLogHubPublishesLogEvents(t *testing.T) {
	hub := newLogHub()
	defer hub.close()
	ch, _, unsubscribe := hub.subscribe()
	defer unsubscribe()

	if _, err := hub.Write([]byte("serve log line\n")); err != nil {
		t.Fatalf("write log hub: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Type != "log" || ev.Message != "serve log line" || ev.Timestamp.IsZero() {
			t.Fatalf("log event = %#v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log event")
	}
}

func TestLogHubReplaysRecentHistory(t *testing.T) {
	hub := newLogHub()
	defer hub.close()
	hub.historySize = 2

	hub.publish(serveLogEvent{Type: "log", Message: "first", Timestamp: time.Now()})
	hub.publish(serveLogEvent{Type: "log", Message: "second", Timestamp: time.Now()})
	hub.publish(serveLogEvent{Type: "log", Message: "third", Timestamp: time.Now()})

	_, history, unsubscribe := hub.subscribe()
	defer unsubscribe()

	if len(history) != 2 {
		t.Fatalf("history length = %d, want 2", len(history))
	}
	if history[0].Message != "second" || history[1].Message != "third" {
		t.Fatalf("history = %#v", history)
	}
}

func TestUIHandlerMissingAssetsReturnsServiceUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	uiHandler(filepath.Join(t.TempDir(), "missing-dist")).ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Web UI assets not found") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestUIHandlerServesIndexFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>ok</main>"), 0600); err != nil {
		t.Fatalf("write index: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/nested/route", nil)
	w := httptest.NewRecorder()
	uiHandler(dir).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<main>ok</main>") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestHandleWebUIReflectsCurrentConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<main>ok</main>"), 0600); err != nil {
		t.Fatalf("write index: %v", err)
	}
	rt := &channelRuntime{cfg: &Config{WebUI: WebUIConfig{Enabled: false, Dir: dir}}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	rt.handleWebUI(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("disabled status = %d, want 404", w.Code)
	}

	rt.cfg.WebUI.Enabled = true
	w = httptest.NewRecorder()
	rt.handleWebUI(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("enabled status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<main>ok</main>") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestHandleBrowseRestrictsToWorkingDirByDefault(t *testing.T) {
	workDir := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "child"), 0700); err != nil {
		t.Fatalf("create child: %v", err)
	}
	cfg := DefaultConfig()
	cfg.Gateway.WorkingDir = workDir
	rt := &channelRuntime{cfg: cfg}

	req := httptest.NewRequest(http.MethodGet, "/api/browse", nil)
	w := httptest.NewRecorder()
	rt.handleBrowse(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("default status = %d, body = %s", w.Code, w.Body.String())
	}
	var got struct {
		Path    string `json:"path"`
		Parent  string `json:"parent"`
		Entries []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}
	if got.Path != workDir || got.Parent != workDir {
		t.Fatalf("browse root path=%q parent=%q, want both %q", got.Path, got.Parent, workDir)
	}
	if len(got.Entries) != 1 || got.Entries[0].Name != "child" {
		t.Fatalf("entries = %#v", got.Entries)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/browse?path="+url.QueryEscape(outside), nil)
	w = httptest.NewRecorder()
	rt.handleBrowse(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("outside status = %d, want 403; body = %s", w.Code, w.Body.String())
	}

	linkPath := filepath.Join(workDir, "outside-link")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Logf("skip symlink browse assertion: %v", err)
	} else {
		req = httptest.NewRequest(http.MethodGet, "/api/browse?path="+url.QueryEscape(linkPath), nil)
		w = httptest.NewRecorder()
		rt.handleBrowse(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("symlink outside status = %d, want 403; body = %s", w.Code, w.Body.String())
		}
	}
}

func TestHandleBrowseUsesAllowedWorkDirs(t *testing.T) {
	allowedRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(allowedRoot, "repo"), 0700); err != nil {
		t.Fatalf("create repo: %v", err)
	}
	cfg := DefaultConfig()
	cfg.Gateway.WorkingDir = t.TempDir()
	allowed := []string{allowedRoot}
	cfg.Gateway.AllowedWorkDirs = &allowed
	rt := &channelRuntime{cfg: cfg}

	req := httptest.NewRequest(http.MethodGet, "/api/browse?path="+url.QueryEscape(filepath.Join(allowedRoot, "repo")), nil)
	w := httptest.NewRecorder()
	rt.handleBrowse(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("allowed status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestResolveWebUIDirRelativeToWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "custom-ui")
	if err := os.MkdirAll(uiDir, 0700); err != nil {
		t.Fatalf("create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte("<main>ok</main>"), 0600); err != nil {
		t.Fatalf("write index: %v", err)
	}
	t.Chdir(dir)

	got := resolveWebUIDir("custom-ui")
	if got != uiDir {
		t.Fatalf("resolveWebUIDir = %q, want %q", got, uiDir)
	}
}

func TestHandleServeConfig_PutAcceptsFlatSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.json")
	rt := &channelRuntime{cfg: DefaultConfig()}

	body := `{
		"listen": ":9099",
		"features": {
			"webUI": false,
			"openaiAPI": false,
			"wechat": true
		},
		"channels": {
			"wechat": {"autoTyping": false}
		}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/serve/config", strings.NewReader(body))
	w := httptest.NewRecorder()
	rt.handleServeConfig(path).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if rt.cfg.Gateway.Listen != ":9099" {
		t.Fatalf("listen = %q, want :9099", rt.cfg.Gateway.Listen)
	}
	if rt.cfg.Features.WebUI {
		t.Fatal("webUI feature should be false after PUT")
	}
	if rt.cfg.Features.OpenAIAPI {
		t.Fatal("openaiAPI feature should be false after PUT")
	}
	if !rt.cfg.Channels.Wechat.Enabled {
		t.Fatal("wechat should be enabled after PUT")
	}
	if rt.cfg.Channels.Wechat.AutoTyping {
		t.Fatal("wechat autoTyping should be false after PUT")
	}

	saved, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("LoadConfigFrom(saved): %v", err)
	}
	if saved.Gateway.Listen != ":9099" || saved.Features.OpenAIAPI {
		t.Fatalf("saved config = %#v", saved)
	}
}

func TestApplyRuntimeFeaturesOverridesLegacyRuntimeFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.Wechat = false
	cfg.Features.Feishu = false
	cfg.Features.Cron = false
	cfg.Features.Memory = false
	cfg.Features.MultiAgent = false
	cfg.Features.WebUI = false
	cfg.Channels.Wechat.Enabled = true
	cfg.Channels.Feishu.Enabled = true
	cfg.Cron.Enabled = true
	cfg.Memory.Enabled = true
	cfg.Gateway.EnableSubAgents = true
	cfg.WebUI.Enabled = true

	applyRuntimeFeatures(cfg)

	if cfg.Channels.Wechat.Enabled {
		t.Fatal("wechat should be disabled by features")
	}
	if cfg.Channels.Feishu.Enabled {
		t.Fatal("feishu should be disabled by features")
	}
	if cfg.Cron.Enabled {
		t.Fatal("cron should be disabled by features")
	}
	if cfg.Memory.Enabled {
		t.Fatal("memory should be disabled by features")
	}
	if cfg.Gateway.EnableSubAgents {
		t.Fatal("multi-agent should be disabled by features")
	}
	if cfg.WebUI.Enabled {
		t.Fatal("webUI should be disabled by features")
	}
}

func TestApplyOverridesPreservesMultiAgentFlagThroughFeatureSync(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Features.MultiAgent = false
	cfg.Gateway.EnableSubAgents = false

	applyOverrides(cfg, RunOptions{MultiAgent: true})
	applyRuntimeFeatures(cfg)

	if !cfg.Features.MultiAgent {
		t.Fatal("multi-agent feature should be enabled by CLI override")
	}
	if !cfg.Gateway.EnableSubAgents {
		t.Fatal("gateway subagents should remain enabled after feature sync")
	}
}

func TestApplyOverridesWebUIDirEnablesWebUI(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WebUI.Enabled = false
	cfg.Features.WebUI = false

	applyOverrides(cfg, RunOptions{WebUIDir: "/tmp/mothx-ui"})
	applyRuntimeFeatures(cfg)

	if cfg.WebUI.Dir != "/tmp/mothx-ui" {
		t.Fatalf("webUI dir = %q, want /tmp/mothx-ui", cfg.WebUI.Dir)
	}
	if !cfg.WebUI.Enabled || !cfg.Features.WebUI {
		t.Fatalf("webUI should be enabled, config = %#v features = %#v", cfg.WebUI, cfg.Features)
	}
}

func TestApplyOverridesPortForms(t *testing.T) {
	tests := []struct {
		name string
		port string
		want string
	}{
		{name: "port only", port: "9090", want: ":9090"},
		{name: "colon port", port: ":9090", want: ":9090"},
		{name: "host port", port: "127.0.0.1:9090", want: "127.0.0.1:9090"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			applyOverrides(cfg, RunOptions{Port: tt.port})
			if cfg.Gateway.Listen != tt.want {
				t.Fatalf("listen = %q, want %q", cfg.Gateway.Listen, tt.want)
			}
		})
	}
}

func TestBuildHermesConfigFromServeConfigAppliesFeatureGating(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Gateway.Provider = "openai"
	cfg.Gateway.Model = "gpt-4o"
	cfg.Gateway.WorkingDir = "/tmp/project"
	cfg.Gateway.Sandbox.Enabled = true
	cfg.Features.MultiAgent = true
	cfg.Features.Wechat = false
	cfg.Features.Feishu = true
	cfg.Features.Cron = false
	cfg.Features.Memory = true
	cfg.Channels.Wechat.Enabled = true
	cfg.Channels.Wechat.AutoTyping = false
	cfg.Channels.Feishu.Enabled = false
	cfg.Channels.Feishu.AppID = "app-id"
	cfg.Channels.Feishu.AppSecret = "secret"
	cfg.Cron.Enabled = true
	cfg.Cron.StorePath = "/tmp/cron.json"
	cfg.Memory.Enabled = false
	cfg.Memory.Path = "/tmp/memory.md"
	cfg.Agent.MaxTurns = 33

	hCfg := buildHermesConfigFromServeConfig(cfg)

	if hCfg.Server.Host != "127.0.0.1" || hCfg.Server.Port != 0 {
		t.Fatalf("server = %#v", hCfg.Server)
	}
	if hCfg.DefaultProvider != "openai" || hCfg.DefaultModel != "gpt-4o" {
		t.Fatalf("provider/model = %q/%q", hCfg.DefaultProvider, hCfg.DefaultModel)
	}
	if hCfg.WorkDir != "/tmp/project" {
		t.Fatalf("workDir = %q, want /tmp/project", hCfg.WorkDir)
	}
	if !hCfg.Sandbox {
		t.Fatal("sandbox should be enabled")
	}
	if !hCfg.MultiAgent {
		t.Fatal("multi-agent should be enabled")
	}
	if hCfg.Wechat.Enabled {
		t.Fatal("wechat should be disabled by features")
	}
	if !hCfg.Feishu.Enabled || hCfg.Feishu.AppID != "app-id" || hCfg.Feishu.AppSecret != "secret" {
		t.Fatalf("feishu = %#v", hCfg.Feishu)
	}
	if hCfg.Cron.Enabled {
		t.Fatal("cron should be disabled by features")
	}
	if !hCfg.Memory.Enabled || hCfg.Memory.Path != "/tmp/memory.md" {
		t.Fatalf("memory = %#v", hCfg.Memory)
	}
	if hCfg.Agent.MaxTurns != 33 {
		t.Fatalf("agent max turns = %d, want 33", hCfg.Agent.MaxTurns)
	}
}

func TestBuildCronStoreHonorsCronFeature(t *testing.T) {
	disabled := buildCronStore(&hermes.HermesConfig{Cron: hermes.CronConfig{Enabled: false}})
	if disabled != nil {
		t.Fatal("cron store should be nil when cron is disabled")
	}

	enabled := buildCronStore(&hermes.HermesConfig{Cron: hermes.CronConfig{Enabled: true, StorePath: filepath.Join(t.TempDir(), "cron.json")}})
	if enabled == nil {
		t.Fatal("cron store should be created when cron is enabled")
	}
}
