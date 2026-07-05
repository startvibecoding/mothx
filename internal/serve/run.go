package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/gateway"
	"github.com/startvibecoding/mothx/internal/hermes"
	"github.com/startvibecoding/mothx/internal/messaging"
	"github.com/startvibecoding/mothx/internal/messaging/feishu"
	"github.com/startvibecoding/mothx/internal/messaging/wechat"
)

type RunOptions struct {
	ConfigPath string
	Port       string
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
	cfg        *Config
	dispatcher *hermes.Dispatcher
	platforms  []messaging.Platform
}

type channelStatus struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Connected bool   `json:"connected"`
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

	rt, err := startChannels(cfg, settings, version)
	if err != nil {
		return err
	}
	defer rt.stop()

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
		cfg.Gateway.Listen = ":" + opts.Port
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
	rt := &channelRuntime{cfg: cfg, dispatcher: dispatcher}
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
	var cronStore cron.CronStore
	if hCfg != nil && hCfg.Cron.Enabled {
		storePath := hCfg.Cron.StorePath
		if storePath == "" {
			storePath = filepath.Join(config.ConfigDir(), "serve-cron.json")
		}
		cronStore = cron.NewFileCronStore(storePath)
	}
	return cronStore
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
	for _, p := range rt.platforms {
		_ = p.Stop()
	}
}

func (rt *channelRuntime) routes(configPath string) func(*gateway.Server, *http.ServeMux) {
	return func(_ *gateway.Server, mux *http.ServeMux) {
		mux.HandleFunc("/api/serve/config", rt.handleServeConfig(configPath))
		mux.HandleFunc("/api/settings", rt.handleSettings)
		mux.HandleFunc("/api/channels", rt.handleChannels)
		if rt.cfg.WebUI.Enabled {
			mux.Handle("/", uiHandler(rt.cfg.WebUI.Dir))
		}
	}
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
			rt.cfg = next
			writeJSON(w, http.StatusOK, rt.cfg)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func (rt *channelRuntime) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	statuses := []channelStatus{
		{Name: "wechat", Enabled: rt.cfg.Channels.Wechat.Enabled, Connected: false},
		{Name: "feishu", Enabled: rt.cfg.Channels.Feishu.Enabled, Connected: false},
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
	writeJSON(w, http.StatusOK, statuses)
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

func uiHandler(dir string) http.Handler {
	if !filepath.IsAbs(dir) {
		if cwd, err := os.Getwd(); err == nil {
			dir = filepath.Join(cwd, dir)
		}
	}
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
