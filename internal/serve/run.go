package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/debugpprof"
	"github.com/startvibecoding/mothx/internal/memory"
	"github.com/startvibecoding/mothx/internal/messaging"
	"github.com/startvibecoding/mothx/internal/messaging/feishu"
	"github.com/startvibecoding/mothx/internal/messaging/wechat"
	channels "github.com/startvibecoding/mothx/internal/serve/channels"
	openaiapi "github.com/startvibecoding/mothx/internal/serve/openaiapi"
	webui "github.com/startvibecoding/mothx/ui"
)

type RunOptions struct {
	ConfigPath string
	Port       string
	WebUIDir   string
	Provider   string
	Model      string
	WorkDir    string
	Sandbox    bool
	MultiAgent bool
	Delegate   bool
	Workflows  bool
	WebSearch  bool
	Browser    bool
	A2AMaster  bool
	Lobster    bool
	Verbose    bool
	Debug      bool
}

type channelRuntime struct {
	cfg           *Config
	version       string
	dispatcher    *channels.Dispatcher
	platforms     []messaging.Platform
	logHub        *logHub
	wsRuntime     websocketRuntime
	cronStore     cron.CronStore
	cronStorePath string
	cronScheduler *cron.Scheduler
}

type channelStatus struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected"`
}

type activeSessionManager interface {
	ListActiveSessions() []openaiapi.ActiveSessionInfo
	DeleteActiveSession(id string) (bool, error)
	GetSessionMessages(id string) ([]openaiapi.SessionMessageEntry, error)
	GetSessionToolResult(id, toolCallID string) (*openaiapi.SessionToolResultDetail, error)
	CapabilityOverview() openaiapi.CapabilityOverview
	GetSessionCapabilities(id string) (*openaiapi.SessionCapabilities, error)
	PatchSessionCapabilities(id string, patch openaiapi.SessionCapabilityPatch) (*openaiapi.SessionCapabilities, error)
}

type websocketRuntime interface {
	ConnectionCount() int
	Stop(time.Duration) error
	SetClientInfo(model, workDir string)
	WebSocketHandler() http.Handler
}

type featureStatus struct {
	WebUI      bool `json:"webUI"`
	OpenAIAPI  bool `json:"openaiAPI"`
	Wechat     bool `json:"wechat"`
	Feishu     bool `json:"feishu"`
	WebSocket  bool `json:"websocket"`
	MultiAgent bool `json:"multiAgent"`
	Delegate   bool `json:"delegate"`
	WebSearch  bool `json:"webSearch"`
	Browser    bool `json:"browser"`
	A2AMaster  bool `json:"a2aMaster"`
	Cron       bool `json:"cron"`
	Memory     bool `json:"memory"`
}

type serveStatus struct {
	Status   string          `json:"status"`
	Listen   string          `json:"listen"`
	Features featureStatus   `json:"features"`
	WebUI    WebUIConfig     `json:"webUI"`
	Channels []channelStatus `json:"channels"`
	Sessions int             `json:"sessions"`
}

func registerServeRoutes(mux *http.ServeMux, rt *channelRuntime, configPath string) {
	if rt == nil {
		return
	}
	rt.routes(configPath)(nil, mux)
}

func Run(opts RunOptions, version string) error {
	config.Verbose = opts.Verbose || opts.Debug
	if opts.Debug {
		_ = os.Setenv("VIBECODING_DEBUG", "1")
		debugpprof.StartForDebug(os.Stderr)
	}

	cfg, path, err := loadRunConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	applyOverrides(cfg, opts)
	applyRuntimeFeatures(cfg)

	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if cfg.API.EnableWebSearch {
		settings.WebSearch.Enabled = config.BoolPtr(true)
	}

	fmt.Fprintf(os.Stderr, "MothX Serve v%s starting\n", version)
	displayAddr := displayListenAddr(cfg.API.GetListenAddr())
	if cfg.Features.OpenAIAPI {
		fmt.Fprintf(os.Stderr, "  OpenAI API: http://%s/v1/chat/completions\n", displayAddr)
	} else {
		fmt.Fprintf(os.Stderr, "  OpenAI API: disabled\n")
	}
	if cfg.WebUI.Enabled {
		fmt.Fprintf(os.Stderr, "  Web UI: http://%s/\n", displayAddr)
	} else {
		fmt.Fprintf(os.Stderr, "  Web UI: disabled\n")
	}

	logHub := newLogHub()
	restoreLogs := installLogHub(logHub)
	defer restoreLogs()

	rt, err := startChannels(cfg, settings, version)
	if err != nil {
		return err
	}
	rt.logHub = logHub
	defer rt.stop()

	if cfg.LobsterMode {
		fmt.Fprintf(os.Stderr, "  Lobster mode: enabled (yolo, no sandbox, sub-agents on)\n")
	}
	fmt.Fprintf(os.Stderr, "  Config: %s\n", path)

	return openaiapi.Run(openaiapi.RunOptions{
		Config:      &cfg.API,
		DisableAPI:  !cfg.Features.OpenAIAPI,
		Provider:    opts.Provider,
		Model:       opts.Model,
		WorkDir:     opts.WorkDir,
		Sandbox:     opts.Sandbox,
		MultiAgent:  opts.MultiAgent,
		Delegate:    opts.Delegate,
		Workflows:   opts.Workflows,
		WebSearch:   opts.WebSearch,
		Browser:     opts.Browser,
		A2AMaster:   opts.A2AMaster,
		Verbose:     opts.Verbose,
		Debug:       opts.Debug,
		ExtraRoutes: rt.routes(path),
	}, version)
}

func displayListenAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	return addr
}

func loadRunConfig(path string) (*Config, string, error) {
	if path != "" {
		cfg, err := LoadConfigFrom(path)
		return cfg, path, err
	}
	cfg, err := LoadConfig()
	return cfg, ConfigPath(), err
}

func applyOverrides(cfg *Config, opts RunOptions) {
	if opts.Port != "" {
		cfg.API.Listen = listenFromPortOverride(opts.Port)
	}
	if opts.WebUIDir != "" {
		webUIDir := opts.WebUIDir
		if useEmbeddedWebUI(webUIDir) {
			if abs, err := filepath.Abs(webUIDir); err == nil {
				webUIDir = abs
			}
		}
		cfg.WebUI.Dir = webUIDir
		cfg.WebUI.Enabled = true
		cfg.Features.WebUI = true
	}
	if opts.WorkDir != "" {
		cfg.API.WorkingDir = opts.WorkDir
	}
	if opts.Provider != "" {
		cfg.API.Provider = opts.Provider
	}
	if opts.Model != "" {
		cfg.API.Model = opts.Model
	}
	if opts.Sandbox {
		cfg.API.Sandbox.Enabled = true
	}
	if opts.MultiAgent {
		cfg.API.EnableSubAgents = true
		cfg.Features.MultiAgent = true
	}
	if opts.Delegate {
		cfg.API.EnableDelegate = true
	}
	if opts.Workflows {
		cfg.API.EnableWorkflows = true
	}
	if opts.WebSearch {
		cfg.API.EnableWebSearch = true
	}
	if opts.Browser {
		cfg.API.EnableBrowser = true
	}
	if opts.A2AMaster {
		cfg.API.EnableA2AMaster = true
	}
	if opts.Lobster {
		cfg.LobsterMode = true
	}
	normalize(cfg)
}

func listenFromPortOverride(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ""
	}
	if strings.HasPrefix(port, ":") || strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func applyRuntimeFeatures(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.WebUI.Enabled = cfg.Features.WebUI
	cfg.API.EnableSubAgents = cfg.Features.MultiAgent
	cfg.Channels.Wechat.Enabled = cfg.Features.Wechat
	cfg.Channels.Feishu.Enabled = cfg.Features.Feishu
	cfg.Cron.Enabled = cfg.Features.Cron
	cfg.Memory.Enabled = cfg.Features.Memory
}

func startChannels(cfg *Config, settings *config.Settings, version string) (*channelRuntime, error) {
	applyRuntimeFeatures(cfg)

	hCfg := buildConfigFromServeConfig(cfg)
	cronStore := buildCronStore(hCfg)

	dispatcher, err := channels.NewDispatcher(hCfg, settings, version, cronStore, nil)
	if err != nil {
		return nil, fmt.Errorf("create channel dispatcher: %w", err)
	}
	rt := &channelRuntime{cfg: cfg, version: version, dispatcher: dispatcher, cronStore: cronStore, cronStorePath: cronStorePath(hCfg)}
	rt.setupCronScheduler(hCfg)
	rt.setupWebSocketRuntime(version)
	rt.startPlatforms()
	return rt, nil
}

func buildConfigFromServeConfig(cfg *Config) *channels.Config {
	hCfg := channels.DefaultConfig()
	if cfg == nil {
		return hCfg
	}
	applyRuntimeFeatures(cfg)
	hCfg.Server.Host = "127.0.0.1"
	hCfg.Server.Port = 0
	hCfg.DefaultProvider = cfg.API.Provider
	hCfg.DefaultModel = cfg.API.Model
	hCfg.MultiAgent = cfg.API.EnableSubAgents
	hCfg.Sandbox = cfg.API.Sandbox.Enabled
	hCfg.WebSearch = cfg.API.EnableWebSearch
	hCfg.Browser = cfg.API.EnableBrowser
	hCfg.A2AMaster = cfg.API.EnableA2AMaster
	hCfg.WorkDir = cfg.API.GetWorkDir()
	hCfg.Wechat = cfg.Channels.Wechat
	hCfg.Feishu = cfg.Channels.Feishu
	hCfg.Cron = cfg.Cron
	hCfg.Memory = cfg.Memory
	hCfg.Security = cfg.Security
	hCfg.Hooks = cfg.Hooks
	hCfg.Agent = cfg.Agent
	return hCfg
}

func buildCronStore(hCfg *channels.Config) cron.CronStore {
	if hCfg != nil && hCfg.Cron.Enabled {
		return cron.NewFileCronStore(cronStorePath(hCfg))
	}
	return nil
}

func cronStorePath(hCfg *channels.Config) string {
	if hCfg != nil && hCfg.Cron.StorePath != "" {
		return hCfg.Cron.StorePath
	}
	return filepath.Join(config.ConfigDir(), "serve-cron.json")
}

func (rt *channelRuntime) setupCronScheduler(hCfg *channels.Config) {
	if hCfg == nil || !hCfg.Cron.Enabled {
		fmt.Fprintf(os.Stderr, "  Cron: disabled\n")
		return
	}
	if rt.cronStore == nil || rt.dispatcher == nil || rt.dispatcher.AgentManager() == nil {
		fmt.Fprintf(os.Stderr, "  Cron: disabled (requires multi-agent)\n")
		return
	}
	interval := time.Duration(hCfg.Cron.Interval) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	rt.cronScheduler = cron.NewScheduler(rt.cronStore, rt.dispatcher.AgentManager(), interval)
	rt.dispatcher.SetCronScheduler(rt.cronScheduler)
	rt.cronScheduler.Start()
	fmt.Fprintf(os.Stderr, "  Cron: enabled\n")
}

func (rt *channelRuntime) applyConfigUpdate(next *Config) {
	if rt == nil {
		return
	}
	applyRuntimeFeatures(next)
	rt.cfg = next
	rt.syncCronRuntime()
	rt.syncWebSocketRuntime()
}

func (rt *channelRuntime) syncCronRuntime() {
	if rt == nil {
		return
	}
	if !rt.cronEnabled() {
		rt.stopCronScheduler()
		rt.cronStore = nil
		rt.cronStorePath = ""
		return
	}

	hCfg := buildConfigFromServeConfig(rt.cfg)
	nextPath := cronStorePath(hCfg)
	if rt.cronStore == nil || rt.cronStorePath != nextPath {
		rt.stopCronScheduler()
		rt.cronStorePath = nextPath
		rt.cronStore = buildCronStore(hCfg)
	}
	if rt.cronStore == nil || rt.dispatcher == nil || rt.dispatcher.AgentManager() == nil {
		rt.stopCronScheduler()
		return
	}
	if rt.cronScheduler == nil || !rt.cronScheduler.IsRunning() {
		interval := time.Duration(hCfg.Cron.Interval) * time.Second
		if interval <= 0 {
			interval = 30 * time.Second
		}
		rt.cronScheduler = cron.NewScheduler(rt.cronStore, rt.dispatcher.AgentManager(), interval)
		rt.dispatcher.SetCronScheduler(rt.cronScheduler)
		rt.cronScheduler.Start()
	}
}

func (rt *channelRuntime) stopCronScheduler() {
	if rt == nil {
		return
	}
	if rt.cronScheduler != nil {
		rt.cronScheduler.Stop()
		rt.cronScheduler = nil
	}
	if rt.dispatcher != nil {
		rt.dispatcher.SetCronScheduler(nil)
	}
}

func (rt *channelRuntime) syncWebSocketRuntime() {
	if rt == nil || rt.cfg == nil || !rt.cfg.Features.WebSocket {
		if rt != nil && rt.wsRuntime != nil {
			_ = rt.wsRuntime.Stop(5 * time.Second)
			rt.wsRuntime = nil
		}
		return
	}
	if rt.wsRuntime == nil {
		rt.setupWebSocketRuntime(rt.version)
		return
	}
	rt.wsRuntime.SetClientInfo(rt.cfg.API.Model, rt.cfg.API.GetWorkDir())
}

func (rt *channelRuntime) startPlatforms() {
	if rt.cfg.Channels.Wechat.Enabled {
		credPath := rt.cfg.Channels.Wechat.CredPath
		if credPath == "" {
			credPath = filepath.Join(config.ConfigDir(), "wechat-credentials.json")
		}
		if creds, err := wechat.LoadCredentials(credPath); err != nil || creds == nil {
			fmt.Fprintf(os.Stderr, "  WeChat: enabled but not logged in\n")
		} else {
			bot := wechat.NewBot(wechat.BotOptions{CredPath: credPath, AutoTyping: rt.cfg.Channels.Wechat.AutoTyping})
			rt.platforms = append(rt.platforms, bot)
			go rt.runPlatform(bot)
			fmt.Fprintf(os.Stderr, "  WeChat: connected\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "  WeChat: disabled\n")
	}

	if rt.cfg.Channels.Feishu.Enabled {
		if rt.cfg.Channels.Feishu.AppID == "" || rt.cfg.Channels.Feishu.AppSecret == "" {
			fmt.Fprintf(os.Stderr, "  Feishu: enabled but app_id/app_secret not configured\n")
		} else {
			bot := feishu.NewBot(feishu.BotOptions{
				AppID:     rt.cfg.Channels.Feishu.AppID,
				AppSecret: rt.cfg.Channels.Feishu.AppSecret,
			})
			rt.platforms = append(rt.platforms, bot)
			go rt.runPlatform(bot)
			fmt.Fprintf(os.Stderr, "  Feishu: connecting\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "  Feishu: disabled\n")
	}
}

func (rt *channelRuntime) runPlatform(p messaging.Platform) {
	if err := p.Start(context.Background(), rt.dispatcher.HandleMessage); err != nil {
		log.Printf("[serve] %s stopped: %v", p.Name(), err)
	}
}

func (rt *channelRuntime) stop() {
	rt.stopCronScheduler()
	for _, p := range rt.platforms {
		_ = p.Stop()
	}
	if rt.wsRuntime != nil {
		_ = rt.wsRuntime.Stop(5 * time.Second)
	}
}

func (rt *channelRuntime) routes(configPath string) func(*openaiapi.Server, *http.ServeMux) {
	return func(srv *openaiapi.Server, mux *http.ServeMux) {
		sessions := activeSessionManagerFromAPI(srv)
		mux.HandleFunc("/api/status", rt.handleStatus(sessions))
		mux.HandleFunc("/api/serve/config", rt.handleServeConfig(configPath))
		mux.HandleFunc("/api/capabilities", rt.handleCapabilities(sessions))
		mux.HandleFunc("/api/sessions", rt.handleSessions(sessions))
		mux.HandleFunc("/api/sessions/", rt.handleSessionByID(sessions))
		mux.HandleFunc("/api/settings", rt.handleSettings)
		mux.HandleFunc("/api/memory", rt.handleMemory)
		mux.HandleFunc("/api/cron", rt.handleCron)
		mux.HandleFunc("/api/cron/", rt.handleCronByID)
		mux.HandleFunc("/api/channels", rt.handleChannels)
		mux.Handle("/ws/logs", rt.handleLogs(sessions))
		mux.HandleFunc("/ws", rt.handleWebSocket)
		mux.HandleFunc("/api/browse", rt.handleBrowse)
		mux.HandleFunc("/", rt.handleWebUI)
	}
}

func activeSessionManagerFromAPI(srv *openaiapi.Server) activeSessionManager {
	if srv == nil {
		return nil
	}
	return srv
}

func (rt *channelRuntime) handleServeConfig(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, rt.cfg)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			next, err := DecodeConfigBytes(body)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			if err := SaveConfig(path, next); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			rt.applyConfigUpdate(next)
			writeJSON(w, http.StatusOK, rt.cfg)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (rt *channelRuntime) handleStatus(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, rt.statusSnapshot(sessions))
	}
}

func (rt *channelRuntime) statusSnapshot(sessions activeSessionManager) serveStatus {
	sessionCount := 0
	if sessions != nil {
		sessionCount = len(sessions.ListActiveSessions())
	}
	status := serveStatus{
		Status:   "ok",
		Channels: rt.channelStatuses(),
		Sessions: sessionCount,
	}
	if rt.cfg != nil {
		status.Listen = rt.cfg.API.GetListenAddr()
		status.Features = featureStatusFromConfig(rt.cfg)
		status.WebUI = rt.cfg.WebUI
	}
	return status
}

func featureStatusFromConfig(cfg *Config) featureStatus {
	if cfg == nil {
		return featureStatus{}
	}
	return featureStatus{
		WebUI:      cfg.Features.WebUI,
		OpenAIAPI:  cfg.Features.OpenAIAPI,
		Wechat:     cfg.Features.Wechat,
		Feishu:     cfg.Features.Feishu,
		WebSocket:  cfg.Features.WebSocket,
		MultiAgent: cfg.Features.MultiAgent,
		Delegate:   cfg.API.EnableDelegate,
		WebSearch:  cfg.API.EnableWebSearch,
		Browser:    cfg.API.EnableBrowser,
		A2AMaster:  cfg.API.EnableA2AMaster,
		Cron:       cfg.Features.Cron,
		Memory:     cfg.Features.Memory,
	}
}

func (rt *channelRuntime) handleSessions(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
			return
		}
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			scope = "all"
		}
		list := sessions.ListActiveSessions()
		switch scope {
		case "all":
		case "active":
			list = filterActiveSessions(list)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid scope: expected all or active"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
	}
}

func (rt *channelRuntime) handleCapabilities(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
			return
		}
		writeJSON(w, http.StatusOK, sessions.CapabilityOverview())
	}
}

func (rt *channelRuntime) handleSessionByID(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/"), "/")
		if len(parts) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session ID required"})
			return
		}
		id, err := url.PathUnescape(parts[0])
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session ID"})
			return
		}
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session ID required"})
			return
		}
		if len(parts) == 1 && id == "active" && r.Method == http.MethodGet {
			if sessions == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"sessions": filterActiveSessions(sessions.ListActiveSessions())})
			return
		}
		if len(parts) == 2 && parts[1] == "capabilities" {
			if sessions == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
				return
			}
			switch r.Method {
			case http.MethodGet:
				caps, err := sessions.GetSessionCapabilities(id)
				if errors.Is(err, openaiapi.ErrSessionNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
					return
				}
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, caps)
				return
			case http.MethodPatch:
				body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read request body"})
					return
				}
				var patch openaiapi.SessionCapabilityPatch
				if len(strings.TrimSpace(string(body))) > 0 {
					if err := json.Unmarshal(body, &patch); err != nil {
						writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
						return
					}
				}
				caps, err := sessions.PatchSessionCapabilities(id, patch)
				if errors.Is(err, openaiapi.ErrSessionNotFound) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
					return
				}
				if errors.Is(err, openaiapi.ErrInvalidCapability) {
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
					return
				}
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, caps)
				return
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
		}
		if len(parts) == 2 && parts[1] == "messages" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if sessions == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
				return
			}
			msgs, err := sessions.GetSessionMessages(id)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"messages": msgs})
			return
		}
		if len(parts) == 3 && parts[1] == "tool-results" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if sessions == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
				return
			}
			toolCallID, err := url.PathUnescape(parts[2])
			if err != nil || toolCallID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tool call ID"})
				return
			}
			detail, err := sessions.GetSessionToolResult(id, toolCallID)
			if errors.Is(err, openaiapi.ErrSessionToolResultNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			if detail == nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": openaiapi.ErrSessionToolResultNotFound.Error()})
				return
			}
			writeJSON(w, http.StatusOK, detail)
			return
		}
		if len(parts) != 1 {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "API server not ready"})
			return
		}
		deleted, err := sessions.DeleteActiveSession(id)
		if errors.Is(err, openaiapi.ErrActiveSessionIDAmbiguous) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !deleted {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
	}
}

func filterActiveSessions(list []openaiapi.ActiveSessionInfo) []openaiapi.ActiveSessionInfo {
	active := make([]openaiapi.ActiveSessionInfo, 0, len(list))
	for _, sess := range list {
		if sess.Active {
			active = append(active, sess)
		}
	}
	return active
}

func (rt *channelRuntime) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, rt.channelStatuses())
}

func (rt *channelRuntime) channelStatuses() []channelStatus {
	if rt == nil || rt.cfg == nil {
		return []channelStatus{
			{Name: "wechat", Enabled: false},
			{Name: "feishu", Enabled: false},
			{Name: "websocket", Enabled: false},
		}
	}
	statuses := []channelStatus{
		{Name: "wechat", Enabled: rt.cfg.Channels.Wechat.Enabled, Connected: false},
		{Name: "feishu", Enabled: rt.cfg.Channels.Feishu.Enabled, Connected: false},
		{Name: "websocket", Enabled: rt.cfg.Features.WebSocket, Connected: rt.wsRuntime != nil && rt.wsRuntime.ConnectionCount() > 0},
	}
	byName := map[string]int{
		"wechat": 0,
		"feishu": 1,
	}
	for _, p := range rt.platforms {
		if idx, ok := byName[p.Name()]; ok {
			statuses[idx].Connected = p.IsConnected()
			continue
		}
		statuses = append(statuses, channelStatus{Name: p.Name(), Enabled: true, Connected: p.IsConnected()})
	}
	return statuses
}

func (rt *channelRuntime) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := config.LoadSettings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		var settings config.Settings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := config.SaveGlobalSettings(&settings); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, settings)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *channelRuntime) handleMemory(w http.ResponseWriter, r *http.Request) {
	if rt == nil || rt.cfg == nil || !rt.cfg.Features.Memory {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "content": ""})
		case http.MethodPut:
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "memory is disabled"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	store := memory.NewStore(rt.cfg.Memory.Path, rt.cfg.API.GetWorkDir())
	switch r.Method {
	case http.MethodGet:
		content, path, source, err := store.Read()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": true,
			"path":    path,
			"source":  source,
			"content": content,
		})
	case http.MethodPut:
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if err := store.WriteAll(body.Content); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		content, path, source, err := store.Read()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": true,
			"path":    path,
			"source":  source,
			"content": content,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (rt *channelRuntime) handleWebUI(w http.ResponseWriter, r *http.Request) {
	if rt == nil || rt.cfg == nil || !rt.cfg.WebUI.Enabled {
		http.NotFound(w, r)
		return
	}
	uiHandler(rt.cfg.WebUI.Dir).ServeHTTP(w, r)
}

func (rt *channelRuntime) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if rt == nil || rt.cfg == nil || !rt.cfg.Features.WebSocket {
		http.NotFound(w, r)
		return
	}
	if rt.wsRuntime == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "websocket runtime not ready"})
		return
	}
	rt.wsRuntime.WebSocketHandler().ServeHTTP(w, r)
}

func (rt *channelRuntime) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		path = rt.browseDefaultDir()
	}
	abs, parent, err := rt.resolveBrowseDir(path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	type dirEntry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	}
	var dirs []dirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, dirEntry{Name: name, Path: filepath.Join(abs, name), IsDir: true})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    abs,
		"parent":  parent,
		"entries": dirs,
	})
}

func (rt *channelRuntime) browseDefaultDir() string {
	if rt != nil && rt.cfg != nil {
		return rt.cfg.API.GetWorkDir()
	}
	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (rt *channelRuntime) resolveBrowseDir(path string) (string, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	abs = filepath.Clean(abs)
	realAbs, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", "", fmt.Errorf("invalid path: %w", err)
	}
	realAbs = filepath.Clean(realAbs)
	roots, err := rt.browseAllowedRoots()
	if err != nil {
		return "", "", err
	}
	if !pathWithinAnyRoot(realAbs, roots) {
		return "", "", fmt.Errorf("directory %q is not in allowed browse roots", path)
	}
	parent := filepath.Dir(realAbs)
	if parent == realAbs || !pathWithinAnyRoot(parent, roots) {
		parent = realAbs
	}
	return realAbs, parent, nil
}

func (rt *channelRuntime) browseAllowedRoots() ([]string, error) {
	if rt == nil || rt.cfg == nil {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve working directory: %w", err)
		}
		return []string{filepath.Clean(cwd)}, nil
	}
	var configured []string
	if rt.cfg.API.AllowedWorkDirs != nil {
		configured = append(configured, (*rt.cfg.API.AllowedWorkDirs)...)
	} else if len(rt.cfg.Security.AllowedWorkDirs) > 0 {
		configured = append(configured, rt.cfg.Security.AllowedWorkDirs...)
	} else {
		configured = []string{rt.cfg.API.GetWorkDir()}
	}
	if len(configured) == 0 {
		return nil, fmt.Errorf("directory browsing is disabled")
	}
	roots := make([]string, 0, len(configured))
	for _, root := range configured {
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("invalid browse root %q: %w", root, err)
		}
		abs = filepath.Clean(abs)
		if realRoot, err := filepath.EvalSymlinks(abs); err == nil {
			abs = filepath.Clean(realRoot)
		}
		roots = append(roots, abs)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("directory browsing is disabled")
	}
	return roots, nil
}

func pathWithinAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

const defaultWebUIDir = "ui/dist"

func uiHandler(dir string) http.Handler {
	if useEmbeddedWebUI(dir) {
		return uiFSHandler(webui.DistFS(), "embedded Web UI assets not found")
	}
	return uiFSHandler(os.DirFS(resolveWebUIDir(dir)), "Web UI assets not found. Build ui/dist or set webUI.dir to a built frontend directory.")
}

func uiFSHandler(fsys fs.FS, missingMessage string) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(pathpkg.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if st, err := fs.Stat(fsys, name); err == nil && !st.IsDir() {
			if name == "index.html" {
				serveUIIndex(w, fsys)
				return
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		if st, err := fs.Stat(fsys, "index.html"); err == nil && !st.IsDir() {
			serveUIIndex(w, fsys)
			return
		}
		http.Error(w, missingMessage, http.StatusServiceUnavailable)
	})
}

func serveUIIndex(w http.ResponseWriter, fsys fs.FS) {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "Web UI index not found", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func resolveWebUIDir(dir string) string {
	if dir == "" {
		dir = defaultWebUIDir
	}
	if filepath.IsAbs(dir) {
		return dir
	}

	var fallback string
	if cwd, err := os.Getwd(); err == nil {
		fallback = filepath.Join(cwd, dir)
		if hasUIIndex(fallback) {
			return fallback
		}
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, candidate := range []string{
			filepath.Join(exeDir, dir),
			filepath.Join(exeDir, "..", "share", "mothx", dir),
		} {
			if hasUIIndex(candidate) {
				return candidate
			}
		}
	}
	if fallback != "" {
		return fallback
	}
	return dir
}

func useEmbeddedWebUI(dir string) bool {
	return dir == "" || filepath.ToSlash(filepath.Clean(dir)) == defaultWebUIDir
}

func hasUIIndex(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !st.IsDir()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
