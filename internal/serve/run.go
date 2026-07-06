package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/gateway"
	"github.com/startvibecoding/mothx/internal/hermes"
	"github.com/startvibecoding/mothx/internal/memory"
	"github.com/startvibecoding/mothx/internal/messaging"
	"github.com/startvibecoding/mothx/internal/messaging/feishu"
	"github.com/startvibecoding/mothx/internal/messaging/wechat"
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
	Lobster    bool
	Verbose    bool
	Debug      bool
}

type channelRuntime struct {
	cfg           *Config
	version       string
	dispatcher    *hermes.Dispatcher
	platforms     []messaging.Platform
	logHub        *logHub
	wsGateway     websocketRuntime
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
	ListActiveSessions() []gateway.ActiveSessionInfo
	DeleteActiveSession(id string) (bool, error)
	GetSessionMessages(id string) ([]gateway.SessionMessageEntry, error)
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

	fmt.Fprintf(os.Stderr, "MothX Serve v%s starting\n", version)
	displayAddr := displayListenAddr(cfg.Gateway.GetListenAddr())
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

	return gateway.Run(gateway.RunOptions{
		Config:      &cfg.Gateway,
		DisableAPI:  !cfg.Features.OpenAIAPI,
		Provider:    opts.Provider,
		Model:       opts.Model,
		WorkDir:     opts.WorkDir,
		Sandbox:     opts.Sandbox,
		MultiAgent:  opts.MultiAgent,
		Delegate:    opts.Delegate,
		Workflows:   opts.Workflows,
		Verbose:     opts.Verbose,
		Debug:       opts.Debug,
		ExtraRoutes: rt.routes(path),
	}, version)
}

func displayListenAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
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
		cfg.Gateway.Listen = listenFromPortOverride(opts.Port)
	}
	if opts.WebUIDir != "" {
		cfg.WebUI.Dir = opts.WebUIDir
		cfg.WebUI.Enabled = true
		cfg.Features.WebUI = true
	}
	if opts.WorkDir != "" {
		cfg.Gateway.WorkingDir = opts.WorkDir
	}
	if opts.Provider != "" {
		cfg.Gateway.Provider = opts.Provider
	}
	if opts.Model != "" {
		cfg.Gateway.Model = opts.Model
	}
	if opts.Sandbox {
		cfg.Gateway.Sandbox.Enabled = true
	}
	if opts.MultiAgent {
		cfg.Gateway.EnableSubAgents = true
		cfg.Features.MultiAgent = true
	}
	if opts.Delegate {
		cfg.Gateway.EnableDelegate = true
	}
	if opts.Workflows {
		cfg.Gateway.EnableWorkflows = true
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
	cfg.Gateway.EnableSubAgents = cfg.Features.MultiAgent
	cfg.Channels.Wechat.Enabled = cfg.Features.Wechat
	cfg.Channels.Feishu.Enabled = cfg.Features.Feishu
	cfg.Cron.Enabled = cfg.Features.Cron
	cfg.Memory.Enabled = cfg.Features.Memory
}

func startChannels(cfg *Config, settings *config.Settings, version string) (*channelRuntime, error) {
	applyRuntimeFeatures(cfg)

	hCfg := buildHermesConfigFromServeConfig(cfg)
	cronStore := buildCronStore(hCfg)

	dispatcher, err := hermes.NewDispatcher(hCfg, settings, version, cronStore, nil)
	if err != nil {
		return nil, fmt.Errorf("create channel dispatcher: %w", err)
	}
	rt := &channelRuntime{cfg: cfg, version: version, dispatcher: dispatcher, cronStore: cronStore, cronStorePath: cronStorePath(hCfg)}
	rt.setupCronScheduler(hCfg)
	rt.setupWebSocketGateway(version)
	rt.startPlatforms()
	return rt, nil
}

func buildHermesConfigFromServeConfig(cfg *Config) *hermes.HermesConfig {
	hCfg := hermes.DefaultHermesConfig()
	if cfg == nil {
		return hCfg
	}
	applyRuntimeFeatures(cfg)
	hCfg.Server.Host = "127.0.0.1"
	hCfg.Server.Port = 0
	hCfg.DefaultProvider = cfg.Gateway.Provider
	hCfg.DefaultModel = cfg.Gateway.Model
	hCfg.MultiAgent = cfg.Gateway.EnableSubAgents
	hCfg.Sandbox = cfg.Gateway.Sandbox.Enabled
	hCfg.WorkDir = cfg.Gateway.GetWorkDir()
	hCfg.Wechat = cfg.Channels.Wechat
	hCfg.Feishu = cfg.Channels.Feishu
	hCfg.Cron = cfg.Cron
	hCfg.Memory = cfg.Memory
	hCfg.Security = cfg.Security
	hCfg.Hooks = cfg.Hooks
	hCfg.Agent = cfg.Agent
	return hCfg
}

func buildCronStore(hCfg *hermes.HermesConfig) cron.CronStore {
	if hCfg != nil && hCfg.Cron.Enabled {
		return cron.NewFileCronStore(cronStorePath(hCfg))
	}
	return nil
}

func cronStorePath(hCfg *hermes.HermesConfig) string {
	if hCfg != nil && hCfg.Cron.StorePath != "" {
		return hCfg.Cron.StorePath
	}
	return filepath.Join(config.ConfigDir(), "serve-cron.json")
}

func (rt *channelRuntime) setupCronScheduler(hCfg *hermes.HermesConfig) {
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

	hCfg := buildHermesConfigFromServeConfig(rt.cfg)
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
		if rt != nil && rt.wsGateway != nil {
			_ = rt.wsGateway.Stop(5 * time.Second)
			rt.wsGateway = nil
		}
		return
	}
	if rt.wsGateway == nil {
		rt.setupWebSocketGateway(rt.version)
		return
	}
	rt.wsGateway.SetClientInfo(rt.cfg.Gateway.Model, rt.cfg.Gateway.GetWorkDir())
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
	if rt.wsGateway != nil {
		_ = rt.wsGateway.Stop(5 * time.Second)
	}
}

func (rt *channelRuntime) routes(configPath string) func(*gateway.Server, *http.ServeMux) {
	return func(srv *gateway.Server, mux *http.ServeMux) {
		sessions := activeSessionManagerFromGateway(srv)
		mux.HandleFunc("/api/status", rt.handleStatus(sessions))
		mux.HandleFunc("/api/serve/config", rt.handleServeConfig(configPath))
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

func activeSessionManagerFromGateway(srv *gateway.Server) activeSessionManager {
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
		status.Listen = rt.cfg.Gateway.GetListenAddr()
		status.Features = featureStatusFromConfig(rt.cfg.Features)
		status.WebUI = rt.cfg.WebUI
	}
	return status
}

func featureStatusFromConfig(cfg FeatureConfig) featureStatus {
	return featureStatus{
		WebUI:      cfg.WebUI,
		OpenAIAPI:  cfg.OpenAIAPI,
		Wechat:     cfg.Wechat,
		Feishu:     cfg.Feishu,
		WebSocket:  cfg.WebSocket,
		MultiAgent: cfg.MultiAgent,
		Cron:       cfg.Cron,
		Memory:     cfg.Memory,
	}
}

func (rt *channelRuntime) handleSessions(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "gateway server not ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions.ListActiveSessions()})
	}
}

func (rt *channelRuntime) handleSessionByID(sessions activeSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session ID required"})
			return
		}
		// Check if the path ends with /messages
		if strings.HasSuffix(id, "/messages") {
			id = strings.TrimSuffix(id, "/messages")
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if sessions == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "gateway server not ready"})
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
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if sessions == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "gateway server not ready"})
			return
		}
		deleted, err := sessions.DeleteActiveSession(id)
		if errors.Is(err, gateway.ErrActiveSessionIDAmbiguous) {
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
		{Name: "websocket", Enabled: rt.cfg.Features.WebSocket, Connected: rt.wsGateway != nil && rt.wsGateway.ConnectionCount() > 0},
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

	store := memory.NewStore(rt.cfg.Memory.Path, rt.cfg.Gateway.GetWorkDir())
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
	if rt.wsGateway == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "websocket gateway not ready"})
		return
	}
	rt.wsGateway.WebSocketHandler().ServeHTTP(w, r)
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
		return rt.cfg.Gateway.GetWorkDir()
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
	if rt.cfg.Gateway.AllowedWorkDirs != nil {
		configured = append(configured, (*rt.cfg.Gateway.AllowedWorkDirs)...)
	} else if len(rt.cfg.Security.AllowedWorkDirs) > 0 {
		configured = append(configured, rt.cfg.Security.AllowedWorkDirs...)
	} else {
		configured = []string{rt.cfg.Gateway.GetWorkDir()}
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

func uiHandler(dir string) http.Handler {
	dir = resolveWebUIDir(dir)
	indexPath := filepath.Join(dir, "index.html")
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		if st, err := os.Stat(indexPath); err == nil && !st.IsDir() {
			http.ServeFile(w, r, indexPath)
			return
		}
		http.Error(w, "Web UI assets not found. Build ui/dist or set webUI.dir to a built frontend directory.", http.StatusServiceUnavailable)
	})
}

func resolveWebUIDir(dir string) string {
	if dir == "" {
		dir = "ui/dist"
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

func hasUIIndex(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !st.IsDir()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
