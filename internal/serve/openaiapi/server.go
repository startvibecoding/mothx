package openaiapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	browserfeature "github.com/startvibecoding/mothx/internal/browser"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/debugpprof"
	"github.com/startvibecoding/mothx/internal/provider"
	providerfactory "github.com/startvibecoding/mothx/internal/provider/factory"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/workflow"
)

// RunOptions controls the OpenAI-compatible API runtime used by serve.
type RunOptions struct {
	Config        *Config
	DisableAPI    bool
	Port          string
	Provider      string
	Model         string
	WorkDir       string
	Unsafe        bool
	Sandbox       bool
	MultiAgent    bool
	Delegate      bool
	Workflows     bool
	WebSearch     bool
	Browser       bool
	A2AMaster     bool
	CronStore     cron.CronStore
	CronScheduler *cron.Scheduler
	Verbose       bool
	Debug         bool
	ExtraRoutes   func(*Server, *http.ServeMux)
}

// Server is the OpenAI-compatible API HTTP server.
type Server struct {
	mu sync.RWMutex

	cfg              *Config
	settings         *config.Settings
	allow            *config.AllowConfig
	saveProjectAllow func(*config.AllowConfig) error
	version          string

	provider         provider.Provider
	providerName     string // user-configured vendor name (e.g. "longcat")
	providerOverride string
	modelOverride    string
	model            *provider.Model
	sandboxMgr       *sandbox.Manager
	skillsMgr        *skills.Manager
	pool             *SessionPool
	streamHub        *sessionStreamHub
	cronStore        cron.CronStore
	cronScheduler    *cron.Scheduler

	extraContext      string
	defaultSessionIDs map[string]string // key: workDir, used when x_session_id is empty
	sessionCreateMu   sync.Mutex
	runSlots          chan struct{}
}

// SettingsSkillHub returns a copy of marketplace settings for runtime adapters.
func (s *Server) SettingsSkillHub() config.SkillHubSettings {
	if s == nil {
		return config.SkillHubSettings{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.settings == nil {
		return config.SkillHubSettings{}
	}
	value := s.settings.SkillHub
	value.OfficialHandles = append([]string(nil), value.OfficialHandles...)
	value.Markets = append([]config.SkillHubMarketSettings(nil), value.Markets...)
	return value
}

func (s *Server) SessionDir() string {
	if s == nil || s.settings == nil {
		return ""
	}
	return s.settings.GetSessionDir()
}

// ApplySettings updates the runtime provider/model from a saved settings.json.
func (s *Server) ApplySettings(next *config.Settings) error {
	if s == nil || next == nil {
		return nil
	}
	s.mu.RLock()
	if s.cfg == nil {
		s.mu.RUnlock()
		return nil
	}
	cfg := *s.cfg
	providerOverride := s.providerOverride
	modelOverride := s.modelOverride
	s.mu.RUnlock()

	runtime := *next
	if cfg.EnableWebSearch {
		runtime.WebSearch.Enabled = config.BoolPtr(true)
	}
	providerName := cfg.Provider
	if providerOverride != "" {
		providerName = providerOverride
	}
	if providerName == "" {
		providerName = runtime.DefaultProvider
	}
	modelID := cfg.Model
	if modelOverride != "" {
		modelID = modelOverride
	}
	if modelID == "" {
		if providerOverride != "" || cfg.Provider != "" {
			modelID = ""
		} else {
			modelID = runtime.DefaultModel
		}
	}

	p, model, err := providerfactory.Create(&runtime, providerName, modelID)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	skillsMgr, extraContext, err := buildWorkDirContext(&runtime, cfg.GetWorkDir(), cfg.EnableWorkflows, cfg.EnableBrowser)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.settings = &runtime
	s.provider = p
	s.providerName = providerName
	s.model = model
	s.skillsMgr = skillsMgr
	s.extraContext = extraContext
	s.mu.Unlock()
	return nil
}

// Run starts the OpenAI-compatible API server.
func Run(opts RunOptions, version string) error {
	config.Verbose = opts.Verbose || opts.Debug
	if opts.Debug {
		_ = os.Setenv("VIBECODING_DEBUG", "1")
		debugpprof.StartForDebug(os.Stderr)
	}

	// Load settings.json
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	gCfg, err := loadRunConfig(opts)
	if err != nil {
		return fmt.Errorf("load API config: %w", err)
	}
	if err := validateListenSecurity(gCfg, opts.Unsafe); err != nil {
		return err
	}
	if gCfg.EnableWebSearch {
		settings.WebSearch.Enabled = config.BoolPtr(true)
	}

	// Resolve provider/model
	providerName := gCfg.Provider
	if opts.Provider != "" {
		providerName = opts.Provider
	}
	if providerName == "" {
		providerName = settings.DefaultProvider
	}

	modelID := gCfg.Model
	if opts.Model != "" {
		modelID = opts.Model
	}
	if modelID == "" {
		if opts.Provider != "" || gCfg.Provider != "" {
			modelID = ""
		} else {
			modelID = settings.DefaultModel
		}
	}

	p, model, err := providerfactory.Create(settings, providerName, modelID)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}

	// Setup working directory
	cwd := gCfg.GetWorkDir()

	// Setup sandbox
	sbMgr := sandbox.NewManagerWithOptions(cwd, settings.Sandbox.Options())
	sbEnabled := gCfg.Sandbox.Enabled
	if !sbEnabled {
		_ = sbMgr.SetLevel(sandbox.LevelNone)
	} else {
		level := sandbox.LevelStandard
		if gCfg.Sandbox.Level == "strict" {
			level = sandbox.LevelStrict
		}
		if err := sbMgr.SetLevel(level); err != nil {
			return fmt.Errorf("strict sandbox enabled but unavailable: %w", err)
		}
		if err := sbMgr.FallbackError(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: sandbox unavailable; using direct execution: %v\n", err)
		}
	}

	skillsMgr, extraContext, err := buildWorkDirContext(settings, cwd, gCfg.EnableWorkflows, gCfg.EnableBrowser)
	if err != nil {
		return err
	}

	// Build session pool
	idleTimeout := time.Duration(gCfg.Session.IdleTimeoutSeconds) * time.Second
	pool := NewSessionPool(gCfg.Session.MaxSessions, idleTimeout)
	var runSlots chan struct{}
	if gCfg.MaxConcurrentReqs > 0 {
		runSlots = make(chan struct{}, gCfg.MaxConcurrentReqs)
	}

	srv := &Server{
		cfg:               gCfg,
		settings:          settings,
		allow:             config.LoadAllow(),
		version:           version,
		provider:          p,
		providerName:      providerName,
		providerOverride:  opts.Provider,
		modelOverride:     opts.Model,
		model:             model,
		sandboxMgr:        sbMgr,
		skillsMgr:         skillsMgr,
		pool:              pool,
		streamHub:         newSessionStreamHub(),
		cronStore:         opts.CronStore,
		cronScheduler:     opts.CronScheduler,
		extraContext:      extraContext,
		defaultSessionIDs: make(map[string]string),
		runSlots:          runSlots,
	}

	// Build routes
	mux := http.NewServeMux()
	registerRoutes(mux, srv, opts)

	// Apply middleware stack (inside-out)
	var handler http.Handler = mux
	handler = ConcurrencyMiddleware(gCfg.MaxConcurrentReqs, handler)
	handler = CORSMiddleware(gCfg.CORS, handler)
	handler = LoggingMiddleware(handler)

	// Auth middleware wraps everything except /health
	authMux := http.NewServeMux()
	authMux.Handle("/health", LoggingMiddleware(http.HandlerFunc(srv.handleHealth)))
	authMux.Handle("/", AuthMiddleware(gCfg.Auth, handler))

	httpServer := &http.Server{
		Addr:         gCfg.GetListenAddr(),
		Handler:      authMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: time.Duration(gCfg.RequestTimeoutSecs+10) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "MothX Serve API %s starting on %s\n", version, gCfg.GetListenAddr())
		fmt.Fprintf(os.Stderr, "  Provider: %s | Model: %s | Mode: %s\n", p.Name(), model.ID, gCfg.DefaultMode)
		fmt.Fprintf(os.Stderr, "  WorkDir: %s\n", cwd)
		if gCfg.Auth.Enabled {
			fmt.Fprintf(os.Stderr, "  Auth: enabled (%d tokens)\n", len(gCfg.Auth.Tokens))
		} else {
			fmt.Fprintf(os.Stderr, "  Auth: disabled\n")
		}
		if warning := apiSecurityWarning(gCfg); warning != "" {
			fmt.Fprintf(os.Stderr, "  WARNING: %s\n", warning)
		}
		if gCfg.Sandbox.Enabled {
			fmt.Fprintf(os.Stderr, "  Sandbox: enabled (level: %s)\n", gCfg.Sandbox.Level)
		}
		if gCfg.EnableSubAgents {
			fmt.Fprintf(os.Stderr, "  Sub-Agents: enabled\n")
		}
		if gCfg.EnableDelegate {
			fmt.Fprintf(os.Stderr, "  Delegate: enabled\n")
		}
		if gCfg.EnableWorkflows {
			fmt.Fprintf(os.Stderr, "  Workflows: enabled\n")
		}
		if gCfg.EnableWebSearch {
			fmt.Fprintf(os.Stderr, "  Web search: enabled\n")
		}
		if gCfg.EnableBrowser {
			fmt.Fprintf(os.Stderr, "  Browser: enabled\n")
		}
		if gCfg.EnableA2AMaster {
			fmt.Fprintf(os.Stderr, "  A2A master: enabled\n")
		}
		fmt.Fprintf(os.Stderr, "  Tool visibility: %s | System prompt: %s\n", gCfg.ToolVisibility.Mode, gCfg.SystemPromptMode)
		fmt.Fprintf(os.Stderr, "\nReady to serve.\n")
		errCh <- httpServer.ListenAndServe()
	}()

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\nReceived %s, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool.Stop()
		if err := httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	return nil
}

func loadRunConfig(opts RunOptions) (*Config, error) {
	var cfg *Config
	if opts.Config != nil {
		cfg = cloneConfig(opts.Config)
		normalizeConfig(cfg)
	} else {
		cfg = DefaultConfig()
	}
	applyRunOverrides(cfg, opts)
	return cfg, nil
}

func applyRunOverrides(cfg *Config, opts RunOptions) {
	if cfg == nil {
		return
	}
	if opts.Port != "" {
		cfg.Listen = listenFromPortOverride(opts.Port)
	}
	if opts.Unsafe {
		cfg.ApplyUnsafeAccess()
	}
	if opts.MultiAgent {
		cfg.EnableSubAgents = true
	}
	if opts.Delegate {
		cfg.EnableDelegate = true
	}
	if opts.Workflows {
		cfg.EnableWorkflows = true
	}
	if opts.WebSearch {
		cfg.EnableWebSearch = true
	}
	if opts.Browser {
		cfg.EnableBrowser = true
	}
	if opts.A2AMaster {
		cfg.EnableA2AMaster = true
	}
	if opts.Sandbox {
		cfg.Sandbox.Enabled = true
	}
	if opts.WorkDir != "" {
		cfg.DefaultWorkDir = opts.WorkDir
		cfg.WorkingDir = ""
	}
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

func registerRoutes(mux *http.ServeMux, srv *Server, opts RunOptions) {
	if !opts.DisableAPI {
		mux.HandleFunc("/v1/chat/completions", srv.handleChatCompletions)
		mux.HandleFunc("/v1/models", srv.handleModels)
	}
	mux.HandleFunc("/health", srv.handleHealth)
	if opts.ExtraRoutes != nil {
		opts.ExtraRoutes(srv, mux)
	}
}

func buildWorkDirContext(settings *config.Settings, workDir string, workflows bool, browser bool) (*skills.Manager, string, error) {
	if workflows {
		if _, _, err := workflow.EnsureProjectSkill(workDir); err != nil {
			return nil, "", fmt.Errorf("create workflow skill: %w", err)
		}
	}
	if browser {
		if _, _, err := browserfeature.EnsureProjectSkill(workDir); err != nil {
			return nil, "", fmt.Errorf("create browser skill: %w", err)
		}
	}
	skillsMgr := skills.NewManagerWithProjectDirs(settings.GetGlobalSkillsDir(), skills.ProjectSkillDirs(workDir))
	_ = skillsMgr.Load()

	var extraContext string
	if settings.ContextFiles.Enabled {
		cfResult := contextfiles.LoadContextFiles(workDir, config.ConfigDir(), settings.ContextFiles.ExtraFiles)
		if ctx := contextfiles.BuildContextString(cfResult); ctx != "" {
			extraContext = ctx
		}
	}
	extraContext += skillsMgr.BuildAllSkillsContext()
	if workflows {
		extraContext += skillsMgr.BuildSkillContext(workflow.SkillName)
	}
	if browser {
		extraContext += skillsMgr.BuildSkillContext(browserfeature.SkillName)
	}
	return skillsMgr, extraContext, nil
}

// LoggingMiddleware logs each request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, lw.statusCode, time.Since(start).Round(time.Millisecond))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

func (lw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(lw.ResponseWriter).Hijack()
}

// Ensure loggingResponseWriter also satisfies http.Flusher for SSE.
func (lw *loggingResponseWriter) Flush() {
	_ = http.NewResponseController(lw.ResponseWriter).Flush()
}

func apiSecurityWarning(cfg *Config) string {
	if cfg.DefaultMode != "yolo" || (cfg.Auth.Enabled && len(cfg.Auth.Tokens) > 0) {
		return ""
	}
	listen := cfg.Listen
	if strings.HasPrefix(listen, ":") ||
		strings.HasPrefix(listen, "0.0.0.0:") ||
		strings.HasPrefix(listen, "[::]:") {
		return "API is listening beyond loopback in yolo mode without authentication"
	}
	return ""
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
		},
	}
	writeJSON(w, status, resp)
}
