package acp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/agent"
	browserfeature "github.com/startvibecoding/mothx/internal/browser"
	"github.com/startvibecoding/mothx/internal/config"
	ctxpkg "github.com/startvibecoding/mothx/internal/context"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/debugpprof"
	"github.com/startvibecoding/mothx/internal/mcp"
	"github.com/startvibecoding/mothx/internal/provider"
	providerfactory "github.com/startvibecoding/mothx/internal/provider/factory"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/systeminit"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/workflow"
)

const protocolVersion = 1
const maxRequestBytes = 10 << 20

const mothxExtensionNamespace = "mothx.dev"

type RunOptions struct {
	Provider   string
	Model      string
	Mode       string
	Thinking   string
	Sandbox    bool
	Verbose    bool
	Debug      bool
	MultiAgent bool
	Delegate   bool
	Workflows  bool
	WebSearch  bool
	Browser    bool
}

type server struct {
	mu  sync.Mutex
	wmu sync.Mutex

	settings *config.Settings
	allow    *config.AllowConfig
	cwd      string

	p            provider.Provider
	providerName string
	m            *provider.Model

	mode          string
	thinkingLevel provider.ThinkingLevel
	sbMgr         *sandbox.Manager
	skillsMgr     *skills.Manager
	extraContext  string
	ruleContent   string
	contextFiles  string

	multiAgent bool
	delegate   bool
	workflows  bool
	browser    bool
	factory    *agent.AgentFactory
	agentMgr   *agent.AgentManager

	sessions map[string]*sessionRuntime
	pending  map[string]chan json.RawMessage

	toolTitles map[string]string
	mcpNotify  map[string]bool

	nextID int64
	r      *bufio.Reader
	w      io.Writer

	permissionTimeout time.Duration
}

type sessionRuntime struct {
	id       string
	mgr      *session.Manager
	agent    agentpkg.Agent
	registry *tools.Registry
	cancel   context.CancelFunc
	promptID string
	closed   bool
	cancelMu sync.Mutex
	mcp      []*mcp.Client
	agentMgr *agent.AgentManager

	usageMu sync.Mutex
	cost    float64
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcp.RPCError   `json:"error,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name,omitempty"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

type initializeRequest struct {
	ProtocolVersion    int            `json:"protocolVersion"`
	ClientCapabilities map[string]any `json:"clientCapabilities,omitempty"`
	ClientInfo         clientInfo     `json:"clientInfo,omitempty"`
}

type initializeResult struct {
	ProtocolVersion   int          `json:"protocolVersion"`
	AgentCapabilities agentCaps    `json:"agentCapabilities"`
	AgentInfo         clientInfo   `json:"agentInfo"`
	AuthMethods       []authMethod `json:"authMethods"`
}

type authMethod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type agentCaps struct {
	LoadSession         bool           `json:"loadSession"`
	PromptCapabilities  promptCaps     `json:"promptCapabilities"`
	SessionCapabilities sessionCaps    `json:"sessionCapabilities"`
	McPCapabilities     mcpCaps        `json:"mcpCapabilities"`
	Meta                map[string]any `json:"_meta,omitempty"`
}

type mcpCaps struct {
	HTTP bool `json:"http"`
	SSE  bool `json:"sse"`
}

type promptCaps struct {
	Image           bool `json:"image"`
	Audio           bool `json:"audio"`
	EmbeddedContext bool `json:"embeddedContext"`
}

type sessionCaps struct {
	// session/new, session/prompt, session/cancel, and session/update are
	// required ACP v1 baseline methods, so they are not capability flags.
	Close  *struct{} `json:"close,omitempty"`
	Delete *struct{} `json:"delete,omitempty"`
	List   *struct{} `json:"list,omitempty"`
	Resume *struct{} `json:"resume,omitempty"`
}

type newSessionRequest struct {
	Cwd        string             `json:"cwd"`
	McpServers []mcp.ServerConfig `json:"mcpServers,omitempty"`
}

type newSessionResult struct {
	SessionID string `json:"sessionId"`
}

type loadSessionRequest struct {
	SessionID  string             `json:"sessionId"`
	Cwd        string             `json:"cwd"`
	McpServers []mcp.ServerConfig `json:"mcpServers,omitempty"`
}

type resumeSessionRequest struct {
	SessionID  string             `json:"sessionId"`
	Cwd        string             `json:"cwd"`
	McpServers []mcp.ServerConfig `json:"mcpServers,omitempty"`
}

type promptRequest struct {
	SessionID string         `json:"sessionId"`
	Prompt    []contentBlock `json:"prompt"`
}

type promptResult struct {
	StopReason string `json:"stopReason"`
}

type cancelRequest struct {
	SessionID string `json:"sessionId"`
}

type closeSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type deleteSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type cancelRequestNotification struct {
	RequestID json.RawMessage `json:"requestId"`
}

type listSessionsRequest struct {
	Cwd    string `json:"cwd,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type listSessionsResult struct {
	Sessions   []listedSession `json:"sessions"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

type listedSession struct {
	SessionID string         `json:"sessionId"`
	Cwd       string         `json:"cwd"`
	Title     string         `json:"title,omitempty"`
	UpdatedAt string         `json:"updatedAt,omitempty"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

type requestPermissionRequest struct {
	SessionID string             `json:"sessionId"`
	ToolCall  permissionToolCall `json:"toolCall"`
	Options   []permissionOption `json:"options"`
}

type permissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

type contentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
	Name     string `json:"name,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type sessionUpdate struct {
	SessionUpdate string         `json:"sessionUpdate"`
	Content       any            `json:"content,omitempty"`
	ToolCallID    string         `json:"toolCallId,omitempty"`
	Title         string         `json:"title,omitempty"`
	Kind          string         `json:"kind,omitempty"`
	Status        string         `json:"status,omitempty"`
	RawInput      map[string]any `json:"rawInput,omitempty"`
	RawOutput     map[string]any `json:"rawOutput,omitempty"`
	Used          *int           `json:"used,omitempty"`
	Size          *int           `json:"size,omitempty"`
	Cost          *usageCost     `json:"cost,omitempty"`
	Entries       []planEntry    `json:"entries,omitempty"`
	Meta          map[string]any `json:"_meta,omitempty"`
}

type toolCallContent struct {
	Type    string       `json:"type"`
	Content contentBlock `json:"content"`
}

type planEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
}

type usageCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type permissionToolCall struct {
	ToolCallID string         `json:"toolCallId"`
	Title      string         `json:"title,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	Status     string         `json:"status,omitempty"`
	RawInput   map[string]any `json:"rawInput,omitempty"`
}

type questionRequest struct {
	SessionID   string   `json:"sessionId"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Explanation string   `json:"explanation,omitempty"`
	TimeoutMs   int64    `json:"timeoutMs"`
}

type questionResult struct {
	Answer string `json:"answer,omitempty"`
}

type permissionResult struct {
	Outcome *permissionOutcome `json:"outcome,omitempty"`
}

type permissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// Run starts the ACP stdio server.
func Run(opts RunOptions) error {
	config.Verbose = opts.Verbose || opts.Debug
	if opts.Debug {
		_ = os.Setenv("VIBECODING_DEBUG", "1")
		debugpprof.StartForDebug(os.Stderr)
	}

	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	if opts.WebSearch {
		settings.WebSearch.Enabled = config.BoolPtr(true)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	srv := &server{
		settings:   settings,
		allow:      config.LoadAllow(),
		cwd:        cwd,
		multiAgent: opts.MultiAgent,
		delegate:   opts.Delegate,
		workflows:  opts.Workflows,
		browser:    opts.Browser,
		sessions:   make(map[string]*sessionRuntime),
		pending:    make(map[string]chan json.RawMessage),
		toolTitles: make(map[string]string),
		mcpNotify:  make(map[string]bool),
		r:          bufio.NewReader(os.Stdin),
		w:          os.Stdout,
	}

	p, model, err := createProvider(settings, opts.Provider, opts.Model)
	if err != nil {
		return err
	}
	srv.p = p
	srv.providerName = opts.Provider
	if srv.providerName == "" {
		srv.providerName = settings.DefaultProvider
	}
	srv.m = model

	mode := opts.Mode
	if mode == "" {
		mode = settings.DefaultMode
	}
	if mode == "" {
		mode = "agent"
	}
	srv.mode = mode

	thinkingLevel := opts.Thinking
	if thinkingLevel == "" {
		thinkingLevel = settings.DefaultThinkingLevel
	}
	srv.thinkingLevel = provider.ThinkingLevel(thinkingLevel)

	sbMgr := sandbox.NewManager(cwd)
	sbEnabled := opts.Sandbox || settings.Sandbox.Enabled
	if !sbEnabled {
		sbMgr.SetLevel(sandbox.LevelNone)
	} else {
		level := sandbox.LevelStandard
		if mode == "plan" {
			level = sandbox.LevelStrict
		} else if mode == "yolo" {
			level = sandbox.LevelNone
		}
		if err := sbMgr.SetLevel(level); err != nil {
			if opts.Sandbox {
				return fmt.Errorf("sandbox requested but unavailable: %w", err)
			}
			sbMgr.SetLevel(sandbox.LevelNone)
		}
	}
	srv.sbMgr = sbMgr

	if opts.Workflows {
		if _, _, err := workflow.EnsureProjectSkill(cwd); err != nil {
			return fmt.Errorf("create workflow skill: %w", err)
		}
	}
	if opts.Browser {
		if _, _, err := browserfeature.EnsureProjectSkill(cwd); err != nil {
			return fmt.Errorf("create browser skill: %w", err)
		}
	}
	skillsMgr := skills.NewManagerWithProjectDirs(settings.GetGlobalSkillsDir(), skills.ProjectSkillDirs(cwd))
	_ = skillsMgr.Load()
	srv.skillsMgr = skillsMgr

	cfResult := contextfiles.LoadContextFiles(cwd, config.ConfigDir(), settings.ContextFiles.ExtraFiles)
	if ctx := contextfiles.BuildContextString(cfResult); ctx != "" {
		srv.extraContext = ctx
	}
	srv.ruleContent = contextfiles.LoadRuleFile(cwd)
	srv.extraContext += skillsMgr.BuildAllSkillsContext()
	if opts.Workflows {
		srv.extraContext += skillsMgr.BuildSkillContext(workflow.SkillName)
	}
	if opts.Browser {
		srv.extraContext += skillsMgr.BuildSkillContext(browserfeature.SkillName)
	}

	// Agent manager backs multi-agent and delegate workflows.
	if opts.MultiAgent || opts.Delegate || opts.Workflows {
		compactionSettings := ctxpkg.CompactionSettings{
			Enabled:          settings.Compaction.Enabled,
			ReserveTokens:    settings.Compaction.ReserveTokens,
			KeepRecentTokens: settings.Compaction.KeepRecentTokens,
			Tokenizer:        settings.Compaction.Tokenizer,
			TokenizerModel:   settings.Compaction.TokenizerModel,
			Template:         settings.Compaction.Template,
		}
		if compactionSettings.ReserveTokens == 0 {
			compactionSettings.ReserveTokens = 16384
		}
		if compactionSettings.KeepRecentTokens == 0 {
			compactionSettings.KeepRecentTokens = 20000
		}

		srv.factory = agent.NewAgentFactoryWithOptions(p, model, settings, sbMgr, srv.extraContext, srv.ruleContent, nil, compactionSettings, nil, agent.AgentFactoryOptions{
			MultiAgentEnabled: true,
			DelegateEnabled:   opts.Delegate,
			WorkflowsEnabled:  opts.Workflows,
			ProviderName:      srv.providerName,
			Allow:             srv.allow,
		})
		srv.agentMgr = agent.NewAgentManager(srv.factory)
	}

	for {
		req, err := srv.readRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			if err := srv.writeMessage(map[string]any{
				"jsonrpc": "2.0",
				"error":   &mcp.RPCError{Code: -32700, Message: err.Error()},
			}); err != nil {
				return err
			}
			continue
		}

		if len(req.Method) == 0 && len(req.ID) > 0 {
			srv.deliverResponse(req.ID, req.Result, req.Error)
			continue
		}

		switch req.Method {
		case "initialize":
			srv.handleInitialize(req)
		case "session/new":
			srv.handleNewSession(req)
		case "session/load":
			srv.handleLoadSession(req)
		case "session/resume":
			srv.handleResumeSession(req)
		case "session/prompt":
			srv.handlePrompt(req)
		case "session/cancel":
			srv.handleCancel(req)
		case "$/cancel_request":
			srv.handleCancelRequest(req)
		case "session/close":
			srv.handleCloseSession(req)
		case "session/delete":
			srv.handleDeleteSession(req)
		case "session/list":
			srv.handleListSessions(req)
		default:
			if len(req.ID) > 0 {
				srv.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32601, Message: "method not found"})
			}
		}
	}
}

func createProvider(settings *config.Settings, providerName, modelID string) (provider.Provider, *provider.Model, error) {
	enabled := true
	return providerfactory.CreateWithOptions(settings, providerName, modelID, providerfactory.Options{
		BuiltinAnthropicCacheControl: &enabled,
	})
}

func (s *server) newToolRegistry(cwd string) *tools.Registry {
	if cwd == "" {
		cwd = s.cwd
	}
	registry := tools.NewRegistry(cwd, s.sbMgr.GetActive())
	registry.RegisterDefaultsWithPlanTool(s.settings.IsPlanToolEnabled())
	// The interactive question tool is exposed in plan/agent modes (see
	// Registry.ModeTools) so the agent can ask the user clarifying questions,
	// e.g. during /systeminit. ACP surfaces questions via request_permission.
	registry.Register(tools.NewQuestionTool(registry))
	if s.skillsMgr != nil {
		registry.Register(tools.NewSkillRefTool(s.skillsMgr))
	}
	if s.browser {
		browserfeature.RegisterTool(registry)
	}
	if s.agentMgr != nil {
		if s.multiAgent {
			agent.RegisterSubAgentTools(registry, s.agentMgr)
		}
		if s.delegate {
			agent.RegisterDelegateSubAgentTool(registry, s.agentMgr)
		}
		if s.workflows {
			workflow.RegisterTools(registry, s.agentMgr, nil)
		}
	}
	return registry
}

func (s *server) handleInitialize(req rpcRequest) {
	var in initializeRequest
	_ = json.Unmarshal(req.Params, &in)
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		AgentCapabilities: agentCaps{
			LoadSession: true,
			PromptCapabilities: promptCaps{
				Image:           false,
				Audio:           false,
				EmbeddedContext: false,
			},
			SessionCapabilities: sessionCaps{
				Close:  &struct{}{},
				Delete: &struct{}{},
				List:   &struct{}{},
				Resume: &struct{}{},
			},
			McPCapabilities: mcpCaps{HTTP: true, SSE: true},
			Meta: map[string]any{
				mothxExtensionNamespace: map[string]any{
					"requestQuestion": true,
					"sessionEvent":    true,
				},
			},
		},
		AgentInfo: clientInfo{
			Name:    "vibecoding",
			Title:   "VibeCoding",
			Version: "dev",
		},
		AuthMethods: []authMethod{},
	}
	s.writeResponse(req.ID, result, nil)
}

func (s *server) handleNewSession(req rpcRequest) {
	var in newSessionRequest
	if err := json.Unmarshal(req.Params, &in); err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	if strings.TrimSpace(in.Cwd) == "" {
		in.Cwd = s.cwd
	}
	if !filepath.IsAbs(in.Cwd) {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "cwd must be an absolute path"})
		return
	}
	mgr := session.New(in.Cwd, s.settings.GetSessionDir())
	if err := mgr.InitWithID(""); err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	id := mgr.GetHeader().ID
	registry := s.newToolRegistry(in.Cwd)
	mcpClients, err := mcp.ConnectServers(context.Background(), in.McpServers, registry, s.buildMCPCallbacks(id))
	if err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	s.mu.Lock()
	if old := s.sessions[id]; old != nil {
		mcp.CloseClients(old.mcp)
	}
	s.sessions[id] = &sessionRuntime{id: id, mgr: mgr, registry: registry, mcp: mcpClients}
	s.mu.Unlock()
	s.writeResponse(req.ID, newSessionResult{SessionID: id}, nil)
}

func (s *server) handleLoadSession(req rpcRequest) {
	var in loadSessionRequest
	if err := json.Unmarshal(req.Params, &in); err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	if strings.TrimSpace(in.Cwd) == "" {
		in.Cwd = s.cwd
	}
	if !filepath.IsAbs(in.Cwd) {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "cwd must be an absolute path"})
		return
	}
	rt, err := s.openSessionRuntime(in.SessionID, in.Cwd, in.McpServers)
	if err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	s.installSessionRuntime(rt)
	allMsgs := rt.mgr.GetMessages()
	for _, msg := range allMsgs {
		s.emitMessage(in.SessionID, msg)
	}
	s.writeResponse(req.ID, nil, nil)
}

func (s *server) handleResumeSession(req rpcRequest) {
	var in resumeSessionRequest
	if err := json.Unmarshal(req.Params, &in); err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	if !filepath.IsAbs(in.Cwd) {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "cwd must be an absolute path"})
		return
	}
	rt, err := s.openSessionRuntime(in.SessionID, in.Cwd, in.McpServers)
	if err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	s.installSessionRuntime(rt)
	s.writeResponse(req.ID, map[string]any{}, nil)
}

func (s *server) openSessionRuntime(sessionID, cwd string, servers []mcp.ServerConfig) (*sessionRuntime, error) {
	registry := s.newToolRegistry(cwd)
	mcpClients, err := mcp.ConnectServers(context.Background(), servers, registry, s.buildMCPCallbacks(sessionID))
	if err != nil {
		return nil, err
	}
	mgr, err := session.OpenByID(cwd, s.settings.GetSessionDir(), sessionID)
	if err != nil {
		mcp.CloseClients(mcpClients)
		return nil, err
	}
	return &sessionRuntime{
		id:       sessionID,
		mgr:      mgr,
		registry: registry,
		mcp:      mcpClients,
		cost:     s.persistedSessionCost(mgr),
	}, nil
}

func (s *server) installSessionRuntime(rt *sessionRuntime) {
	s.mu.Lock()
	old := s.sessions[rt.id]
	s.sessions[rt.id] = rt
	s.mu.Unlock()
	if old != nil {
		old.cancelMu.Lock()
		old.closed = true
		cancel := old.cancel
		old.cancelMu.Unlock()
		if cancel != nil {
			cancel()
		} else {
			mcp.CloseClients(old.mcp)
		}
	}
}

func (s *server) handlePrompt(req rpcRequest) {
	var in promptRequest
	if err := json.Unmarshal(req.Params, &in); err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	rt := s.sessionForPrompt(in.SessionID)
	if rt == nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: "unknown session"})
		return
	}
	userText, err := promptToText(in.Prompt)
	if err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: err.Error()})
		return
	}
	if userText == "" {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "empty prompt"})
		return
	}
	effectiveMode := s.mode
	// Expand the /systeminit slash command into the full instruction prompt.
	// In ACP the question tool is available, so use the interactive variant.
	// /systeminit must also be able to write AGENTS.md, so upgrade plan mode to
	// agent for this prompt only.
	if fields := strings.Fields(strings.TrimSpace(userText)); len(fields) > 0 && fields[0] == systeminit.Command {
		extra := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(userText), systeminit.Command))
		userText = systeminit.Prompt(true, extra)
		if effectiveMode == "plan" {
			effectiveMode = "agent"
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	promptKey := mcp.RawIDKey(req.ID)
	rt.cancelMu.Lock()
	if rt.cancel != nil {
		rt.cancelMu.Unlock()
		cancel()
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: "session already has an active prompt"})
		return
	}
	rt.cancel = cancel
	rt.promptID = promptKey
	rt.cancelMu.Unlock()

	var a agentpkg.Agent
	if s.agentMgr != nil {
		var err error
		a, err = s.agentMgr.Create(agent.AgentOptions{
			Mode:    effectiveMode,
			Model:   s.m,
			Session: rt.mgr,
		})
		if err != nil {
			cancel()
			s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
			return
		}
	} else {
		inner := agent.New(agent.Config{
			Provider:      s.p,
			Vendor:        s.providerName,
			Model:         s.m,
			Mode:          effectiveMode,
			ThinkingLevel: s.thinkingLevel,
			MaxTokens:     agent.ResolveMaxTokens(s.m),
			SandboxMgr:    s.sbMgr,
			Settings:      s.settings,
			Allow:         s.allow,
			Session:       rt.mgr,
			ExtraContext:  s.extraContext,
			RuleContent:   s.ruleContent,
			CompactionSettings: ctxpkg.CompactionSettings{
				Enabled:          s.settings.Compaction.Enabled,
				ReserveTokens:    s.settings.Compaction.ReserveTokens,
				KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
			},
			ApprovalHandler: func(toolCallID, toolName string, args map[string]any) bool {
				return s.requestPermissionContext(ctx, rt.id, toolCallID, toolName, args)
			},
			MultiAgent:   s.multiAgent,
			DelegateMode: s.delegate,
			Workflows:    s.workflows,
		}, rt.registry)
		a = agent.NewAgentAdapter(inner)
	}
	rt.agent = a
	go func() {
		stopReason := "end_turn"
		var runErr error
		defer func() {
			if s.agentMgr != nil && rt.agent != nil {
				s.agentMgr.Finish(rt.agent.ID(), runErr)
			}
			rt.cancelMu.Lock()
			closed := rt.closed
			if rt.promptID == promptKey {
				rt.cancel = nil
				rt.promptID = ""
			}
			rt.cancelMu.Unlock()
			if closed {
				mcp.CloseClients(rt.mcp)
			}
			cancel()
		}()
		events := rt.agent.Run(ctx, userText)
		for ev := range events {
			s.handleAgentEvent(rt.id, ev)
			switch ev.Type {
			case agentpkg.EventQuestionRequest:
				go s.handleQuestion(ctx, rt, ev)
			case agentpkg.EventDone:
				stopReason = normalizeStopReason(ev.StopReason)
			case agentpkg.EventError:
				if ev.Error != nil {
					runErr = ev.Error
				}
				stopReason = normalizeStopReason(ev.StopReason)
			}
		}
		if runErr != nil && stopReason != "cancelled" {
			s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: runErr.Error()})
			return
		}
		s.writeResponse(req.ID, promptResult{StopReason: stopReason}, nil)
	}()
}

func (s *server) handleCancel(req rpcRequest) {
	var in cancelRequest
	_ = json.Unmarshal(req.Params, &in)
	s.mu.Lock()
	rt := s.sessions[in.SessionID]
	s.mu.Unlock()
	if rt != nil {
		rt.cancelMu.Lock()
		if rt.cancel != nil {
			rt.cancel()
		}
		rt.cancelMu.Unlock()
	}
	if len(req.ID) > 0 {
		s.writeResponse(req.ID, map[string]any{}, nil)
	}
}

func (s *server) handleCancelRequest(req rpcRequest) {
	var in cancelRequestNotification
	if err := json.Unmarshal(req.Params, &in); err != nil || len(in.RequestID) == 0 {
		return
	}
	key := mcp.RawIDKey(in.RequestID)
	s.mu.Lock()
	pending := s.pending[key]
	if pending != nil {
		delete(s.pending, key)
	}
	var cancel context.CancelFunc
	for _, rt := range s.sessions {
		rt.cancelMu.Lock()
		if rt.promptID == key {
			cancel = rt.cancel
		}
		rt.cancelMu.Unlock()
		if cancel != nil {
			break
		}
	}
	s.mu.Unlock()
	if pending != nil {
		pending <- json.RawMessage(`{"outcome":{"outcome":"cancelled"}}`)
	}
	if cancel != nil {
		cancel()
	}
}

func (s *server) handleCloseSession(req rpcRequest) {
	var in closeSessionRequest
	if err := json.Unmarshal(req.Params, &in); err != nil || strings.TrimSpace(in.SessionID) == "" {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}

	s.closeSessionRuntime(in.SessionID)
	s.writeResponse(req.ID, map[string]any{}, nil)
}

func (s *server) closeSessionRuntime(sessionID string) *sessionRuntime {
	s.mu.Lock()
	rt := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	if rt == nil {
		return nil
	}
	rt.cancelMu.Lock()
	rt.closed = true
	cancel := rt.cancel
	rt.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	} else {
		mcp.CloseClients(rt.mcp)
	}
	return rt
}

func (s *server) handleDeleteSession(req rpcRequest) {
	var in deleteSessionRequest
	if err := json.Unmarshal(req.Params, &in); err != nil || strings.TrimSpace(in.SessionID) == "" {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	s.mu.Lock()
	active := s.sessions[in.SessionID]
	s.mu.Unlock()
	if active != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: "cannot delete an active session"})
		return
	}
	mgr, err := session.OpenByIDExact(s.settings.GetSessionDir(), in.SessionID)
	if err == nil {
		err = session.DeleteSession(mgr.GetFile(), s.settings.GetSessionDir())
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	s.writeResponse(req.ID, map[string]any{}, nil)
}

const sessionListPageSize = 50

func (s *server) handleListSessions(req rpcRequest) {
	var in listSessionsRequest
	if len(req.Params) > 0 && json.Unmarshal(req.Params, &in) != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid params"})
		return
	}
	if in.Cwd != "" && !filepath.IsAbs(in.Cwd) {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "cwd must be an absolute path"})
		return
	}

	offset := 0
	if in.Cursor != "" {
		var err error
		offset, err = strconv.Atoi(in.Cursor)
		if err != nil || offset < 0 {
			s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid cursor"})
			return
		}
	}

	var (
		details []session.SessionDetail
		err     error
	)
	if in.Cwd == "" {
		details, err = session.ListAllDetailed(s.settings.GetSessionDir())
	} else {
		details, err = session.ListForDirDetailed(in.Cwd, s.settings.GetSessionDir())
	}
	if err != nil {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32000, Message: err.Error()})
		return
	}
	if offset > len(details) {
		s.writeResponse(req.ID, nil, &mcp.RPCError{Code: -32602, Message: "invalid cursor"})
		return
	}

	end := offset + sessionListPageSize
	if end > len(details) {
		end = len(details)
	}
	result := listSessionsResult{Sessions: make([]listedSession, 0, end-offset)}
	for _, detail := range details[offset:end] {
		title := detail.Name
		if title == "" {
			title = detail.Preview
		}
		result.Sessions = append(result.Sessions, listedSession{
			SessionID: detail.ID,
			Cwd:       detail.Cwd,
			Title:     title,
			UpdatedAt: detail.ModTime.UTC().Format(time.RFC3339),
			Meta:      map[string]any{"messageCount": detail.MessageCount},
		})
	}
	if end < len(details) {
		result.NextCursor = strconv.Itoa(end)
	}
	s.writeResponse(req.ID, result, nil)
}

func (s *server) sessionForPrompt(sessionID string) *sessionRuntime {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[sessionID]
}

func (s *server) handleAgentEvent(sessionID string, ev agentpkg.Event) {
	switch ev.Type {
	case agentpkg.EventTextDelta:
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "agent_message_chunk",
			Content:       &contentBlock{Type: "text", Text: ev.TextDelta},
		})
	case agentpkg.EventThinkDelta:
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "agent_thought_chunk",
			Content:       &contentBlock{Type: "text", Text: ev.ThinkDelta},
		})
	case agentpkg.EventToolCall:
		if ev.ToolCall != nil {
			title := s.rememberToolTitle(ev.ToolCall.ID, ev.ToolCall.Name, ev.ToolArgs)
			s.notify(sessionID, sessionUpdate{
				SessionUpdate: "tool_call",
				ToolCallID:    ev.ToolCall.ID,
				Title:         title,
				Kind:          acpToolKind(ev.ToolCall.Name),
				Status:        "pending",
				RawInput:      toolRawInput(ev.ToolArgs),
			})
		}
	case agentpkg.EventToolExecutionStart:
		title := s.rememberToolTitle(ev.ToolCallID, ev.ToolName, ev.ToolArgs)
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    ev.ToolCallID,
			Title:         title,
			Kind:          acpToolKind(ev.ToolName),
			Status:        "in_progress",
			RawInput:      toolRawInput(ev.ToolArgs),
		})
	case agentpkg.EventToolExecutionEnd:
		status := "completed"
		if ev.ToolError != nil {
			status = "failed"
		}
		rawOutput := map[string]any{"content": ev.ToolResult}
		if ev.ToolDiff != nil {
			rawOutput["diff"] = ev.ToolDiff
		}
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    ev.ToolCallID,
			Title:         s.toolTitleFor(ev.ToolCallID, ev.ToolName),
			Kind:          acpToolKind(ev.ToolName),
			Status:        status,
			Content:       textToolContent(ev.ToolResult),
			RawOutput:     rawOutput,
		})
	case agentpkg.EventToolExecutionUpdate:
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    ev.ToolCallID,
			Content:       textToolContent(fmt.Sprint(ev.PartialResult)),
		})
	case agentpkg.EventToolResult:
	case agentpkg.EventPlanUpdate:
		if ev.Plan != nil {
			s.notify(sessionID, sessionUpdate{
				SessionUpdate: "plan",
				Entries:       acpPlanEntries(ev.Plan),
				Meta:          acpPlanMeta(ev.Plan),
			})
		}
	case agentpkg.EventUsage:
		s.emitUsageUpdate(sessionID, ev, true)
	case agentpkg.EventDone:
		s.emitUsageUpdate(sessionID, ev, false)
	case agentpkg.EventStatus:
		s.notifyExtension("_mothx/session_event", map[string]any{
			"sessionId": sessionID,
			"event":     "status",
			"message":   ev.StatusMessage,
		})
	case agentpkg.EventCompactionStart, agentpkg.EventCompactionEnd, agentpkg.EventTurnStart, agentpkg.EventTurnEnd:
		s.notifyExtension("_mothx/session_event", map[string]any{
			"sessionId": sessionID,
			"event":     acpEventName(ev.Type),
			"message":   ev.StatusMessage,
		})
	}
}

func acpEventName(eventType agentpkg.EventType) string {
	switch eventType {
	case agentpkg.EventCompactionStart:
		return "compaction_started"
	case agentpkg.EventCompactionEnd:
		return "compaction_finished"
	case agentpkg.EventTurnStart:
		return "turn_started"
	case agentpkg.EventTurnEnd:
		return "turn_finished"
	default:
		return "unknown"
	}
}

func acpToolKind(name string) string {
	switch name {
	case "read", "ls":
		return "read"
	case "write", "edit":
		return "edit"
	case "grep", "find":
		return "search"
	case "bash":
		return "execute"
	case "plan":
		return "think"
	default:
		return "other"
	}
}

func textToolContent(text string) []toolCallContent {
	if text == "" {
		return nil
	}
	return []toolCallContent{{Type: "content", Content: contentBlock{Type: "text", Text: text}}}
}

func acpPlanEntries(plan *agentpkg.TaskPlan) []planEntry {
	entries := make([]planEntry, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		status := "pending"
		switch step.Status {
		case "running":
			status = "in_progress"
		case "done", "failed":
			status = "completed"
		}
		entries = append(entries, planEntry{Content: step.Title, Priority: "medium", Status: status})
	}
	return entries
}

func acpPlanMeta(plan *agentpkg.TaskPlan) map[string]any {
	if plan.Title == "" && plan.Note == "" {
		return nil
	}
	return map[string]any{mothxExtensionNamespace: map[string]string{"title": plan.Title, "note": plan.Note}}
}

func (s *server) emitUsageUpdate(sessionID string, ev agentpkg.Event, addCost bool) {
	s.mu.Lock()
	rt := s.sessions[sessionID]
	s.mu.Unlock()
	if rt == nil {
		return
	}

	used, size := usageContext(ev.ContextUsage, ev.Usage, s.m)
	rt.usageMu.Lock()
	if addCost && ev.Usage != nil {
		if s.m != nil {
			ev.Usage.CalculateCost(s.m.Cost.Input, s.m.Cost.Output, s.m.Cost.CacheRead, s.m.Cost.CacheWrite)
		}
		rt.cost += ev.Usage.Cost.Total
	}
	cost := rt.cost
	rt.usageMu.Unlock()

	update := sessionUpdate{
		SessionUpdate: "usage_update",
		Used:          &used,
		Size:          &size,
	}
	if cost > 0 {
		update.Cost = &usageCost{Amount: cost, Currency: "USD"}
	}
	s.notify(sessionID, update)
}

func usageContext(contextUsage *agentpkg.ContextUsage, usage *agentpkg.Usage, model *provider.Model) (int, int) {
	used := 0
	size := 0
	if contextUsage != nil {
		used = contextUsage.Tokens
		size = contextUsage.ContextWindow
	}
	if used == 0 && usage != nil {
		used = usage.TotalTokens
	}
	if size == 0 && model != nil {
		size = model.ContextWindow
	}
	return used, size
}

func (s *server) persistedSessionCost(mgr *session.Manager) float64 {
	if mgr == nil {
		return 0
	}
	var total float64
	for _, msg := range mgr.GetMessages() {
		if msg.Usage == nil {
			continue
		}
		if msg.Usage.Cost.Total == 0 {
			msg.Usage.CalculateCost(s.m)
		}
		total += msg.Usage.Cost.Total
	}
	return total
}

func formatACPPlan(plan *agentpkg.TaskPlan) string {
	if plan == nil || len(plan.Steps) == 0 {
		return "Plan updated."
	}
	var b strings.Builder
	title := plan.Title
	if title == "" {
		title = "Plan"
	}
	b.WriteString(title)
	for _, step := range plan.Steps {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("%s %s", planStatusMarker(step.Status), step.Title))
	}
	if plan.Note != "" {
		b.WriteString("\nnote: " + plan.Note)
	}
	return b.String()
}

func planStatusMarker(status string) string {
	switch status {
	case "running":
		return ">"
	case "done":
		return "x"
	case "failed":
		return "!"
	default:
		return "-"
	}
}

func (s *server) buildMCPCallbacks(sessionID string) mcp.Callbacks {
	return mcp.Callbacks{
		OnNotification: func(serverName, method string, params json.RawMessage) {
			s.handleMCPNotification(sessionID, serverName, method, params)
		},
		OnSamplingCreateMessage: func(ctx context.Context, serverName string, params json.RawMessage) (json.RawMessage, *mcp.RPCError) {
			return s.handleMCPSamplingCreateMessage(ctx, sessionID, serverName, params)
		},
	}
}

func (s *server) handleMCPNotification(sessionID, serverName, method string, params json.RawMessage) {
	callID := "mcp-notify-" + mcp.SanitizeToolName(serverName)
	title := "mcp_notification: " + serverName
	s.mu.Lock()
	if !s.mcpNotify[callID] {
		s.mcpNotify[callID] = true
		s.mu.Unlock()
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call",
			ToolCallID:    callID,
			Title:         title,
			Kind:          "other",
			Status:        "pending",
		})
	} else {
		s.mu.Unlock()
	}

	rawOut := map[string]any{
		"method": method,
	}
	if parsed := parseJSONRawToMap(params); parsed != nil {
		rawOut["params"] = parsed
	} else if trimmed := strings.TrimSpace(string(params)); trimmed != "" && trimmed != "null" {
		rawOut["paramsText"] = trimmed
	}

	switch method {
	case "notifications/progress", "notifications/message", "logging/message", "notifications/cancelled":
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    callID,
			Title:         title,
			Status:        "in_progress",
			RawOutput:     rawOut,
		})
	}
}

func (s *server) handleMCPSamplingCreateMessage(ctx context.Context, sessionID, serverName string, params json.RawMessage) (json.RawMessage, *mcp.RPCError) {
	prompt, systemPrompt, maxTokens := extractSamplingInput(params)
	if strings.TrimSpace(prompt) == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "sampling/createMessage requires non-empty messages"}
	}
	if maxTokens <= 0 {
		maxTokens = agent.ResolveMaxTokens(s.m)
	}
	modelID := ""
	if s.m != nil {
		modelID = s.m.ID
	}
	chatCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	events := s.p.Chat(chatCtx, provider.ChatParams{
		Messages:      []provider.Message{provider.NewUserMessage(prompt)},
		SystemPrompt:  systemPrompt,
		ThinkingLevel: s.thinkingLevel,
		MaxTokens:     maxTokens,
		Temperature:   s.m.Temperature,
		TopP:          s.m.TopP,
		ModelID:       modelID,
	})
	var outText strings.Builder
	for ev := range events {
		switch ev.Type {
		case provider.StreamTextDelta:
			outText.WriteString(ev.TextDelta)
		case provider.StreamDone:
			// noop
		case provider.StreamError:
			if ev.Error != nil {
				return nil, &mcp.RPCError{Code: -32000, Message: ev.Error.Error()}
			}
		}
	}
	text := strings.TrimSpace(outText.String())
	if text == "" {
		text = "(empty response)"
	}
	result := map[string]any{
		"model": modelID,
		"role":  "assistant",
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32000, Message: err.Error()}
	}
	s.notify(sessionID, sessionUpdate{
		SessionUpdate: "agent_message_chunk",
		Content:       &contentBlock{Type: "text", Text: "MCP[" + serverName + "] sampling/createMessage completed"},
	})
	return data, nil
}

func extractSamplingPrompt(params json.RawMessage) string {
	prompt, _, _ := extractSamplingInput(params)
	return prompt
}

func extractSamplingInput(params json.RawMessage) (prompt string, systemPrompt string, maxTokens int) {
	maxTokens = 0
	if len(params) == 0 {
		return "", "", maxTokens
	}
	var raw map[string]any
	if err := json.Unmarshal(params, &raw); err != nil {
		return strings.TrimSpace(string(params)), "", maxTokens
	}
	if v, ok := raw["maxTokens"].(float64); ok && int(v) > 0 {
		maxTokens = int(v)
	}
	msgs, _ := raw["messages"].([]any)
	var parts []string
	for _, m := range msgs {
		msgMap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		content := msgMap["content"]
		role, _ := msgMap["role"].(string)
		switch v := content.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				if role == "system" {
					if systemPrompt == "" {
						systemPrompt = v
					}
					continue
				}
				parts = append(parts, v)
			}
		case []any:
			var blockTexts []string
			for _, item := range v {
				block, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if t, _ := block["type"].(string); t == "text" {
					if txt, _ := block["text"].(string); strings.TrimSpace(txt) != "" {
						blockTexts = append(blockTexts, txt)
					}
				}
			}
			if len(blockTexts) == 0 {
				continue
			}
			joined := strings.Join(blockTexts, "\n")
			if role == "system" {
				if systemPrompt == "" {
					systemPrompt = joined
				}
				continue
			}
			parts = append(parts, joined)
		}
	}
	return strings.Join(parts, "\n"), systemPrompt, maxTokens
}

func parseJSONRawToMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// handleQuestion routes MothX's interactive question tool through its own ACP
// extension instead of overloading the standard permission request.
func (s *server) handleQuestion(ctx context.Context, rt *sessionRuntime, ev agentpkg.Event) {
	qh, ok := rt.agent.(agentpkg.QuestionHandler)
	if !ok {
		return
	}
	answer := s.requestQuestion(ctx, rt.id, ev.QuestionText, ev.QuestionOptions, ev.QuestionContext)
	qh.HandleQuestionResponse(ev.QuestionID, answer)
}

// requestQuestion sends a multiple-choice question to ACP clients that opt into
// the MothX extension and returns an empty answer if cancelled or timed out.
func (s *server) requestQuestion(ctx context.Context, sessionID, question string, options []string, explanation string) string {
	id := s.nextRequestID()
	ch := make(chan json.RawMessage, 1)
	s.mu.Lock()
	s.pending[id] = ch
	s.mu.Unlock()

	if err := s.notifyRequest(id, "_mothx/request_question", questionRequest{
		SessionID:   sessionID,
		Question:    question,
		Options:     options,
		Explanation: explanation,
		TimeoutMs:   int64((5 * time.Minute).Milliseconds()),
	}); err != nil {
		s.deletePending(id)
		return ""
	}
	select {
	case <-ctx.Done():
		s.deletePending(id)
		return ""
	case <-time.After(5 * time.Minute):
		s.deletePending(id)
		return ""
	case resp := <-ch:
		var out questionResult
		_ = json.Unmarshal(resp, &out)
		for _, option := range options {
			if out.Answer == option {
				return option
			}
		}
		return ""
	}
}

func (s *server) requestPermission(sessionID, toolCallID, toolName string, args map[string]any) bool {
	return s.requestPermissionContext(context.Background(), sessionID, toolCallID, toolName, args)
}

func (s *server) requestPermissionContext(ctx context.Context, sessionID, toolCallID, toolName string, args map[string]any) bool {
	id := s.nextRequestID()
	ch := make(chan json.RawMessage, 1)
	s.mu.Lock()
	s.pending[id] = ch
	s.mu.Unlock()
	if err := s.notifyRequest(id, "session/request_permission", requestPermissionRequest{
		SessionID: sessionID,
		ToolCall: permissionToolCall{
			ToolCallID: toolCallID,
			Title:      toolName,
			Kind:       acpToolKind(toolName),
			Status:     "pending",
			RawInput:   toolRawInput(args),
		},
		Options: []permissionOption{
			{OptionID: "allow-once", Name: "Allow once", Kind: "allow_once"},
			{OptionID: "reject-once", Name: "Reject", Kind: "reject_once"},
		},
	}); err != nil {
		s.deletePending(id)
		return false
	}
	timeout := s.permissionTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	select {
	case <-ctx.Done():
		s.deletePending(id)
		return false
	case <-time.After(timeout):
		s.deletePending(id)
		return false
	case resp := <-ch:
		var out permissionResult
		_ = json.Unmarshal(resp, &out)
		return out.Outcome != nil && out.Outcome.Outcome == "selected" && out.Outcome.OptionID == "allow-once"
	}
}

func (s *server) deletePending(id string) {
	s.mu.Lock()
	delete(s.pending, id)
	s.mu.Unlock()
}

func (s *server) deliverResponse(id json.RawMessage, result json.RawMessage, errMsg json.RawMessage) {
	key := mcp.RawIDKey(id)
	s.mu.Lock()
	ch, ok := s.pending[key]
	if ok {
		delete(s.pending, key)
	}
	s.mu.Unlock()
	if ok {
		if len(errMsg) > 0 {
			ch <- errMsg
			return
		}
		ch <- result
	}
}

func (s *server) emitMessage(sessionID string, msg provider.Message) {
	if msg.Role == "assistant" {
		for _, c := range msg.Contents {
			if c.Type == "thinking" && c.Thinking != "" {
				s.notify(sessionID, sessionUpdate{SessionUpdate: "agent_thought_chunk", Content: &contentBlock{Type: "text", Text: c.Thinking}})
			} else if c.Type == "text" && c.Text != "" {
				s.notify(sessionID, sessionUpdate{SessionUpdate: "agent_message_chunk", Content: &contentBlock{Type: "text", Text: c.Text}})
			} else if c.Type == "toolCall" && c.ToolCall != nil {
				var rawInput map[string]any
				_ = json.Unmarshal(c.ToolCall.Arguments, &rawInput)
				title := s.rememberToolTitle(c.ToolCall.ID, c.ToolCall.Name, rawInput)
				s.notify(sessionID, sessionUpdate{
					SessionUpdate: "tool_call",
					ToolCallID:    c.ToolCall.ID,
					Title:         title,
					Kind:          acpToolKind(c.ToolCall.Name),
					Status:        "pending",
					RawInput:      toolRawInput(rawInput),
				})
			}
		}
		return
	}
	if msg.Role == "user" {
		text := msg.Content
		if text == "" {
			for _, c := range msg.Contents {
				if c.Type == "text" && c.Text != "" {
					text = c.Text
					break
				}
			}
		}
		if text != "" {
			s.notify(sessionID, sessionUpdate{SessionUpdate: "user_message_chunk", Content: &contentBlock{Type: "text", Text: text}})
		}
		return
	}
	if msg.Role == "toolResult" {
		rawOutput := map[string]any{"content": msg.Content}
		status := "completed"
		if msg.IsError {
			status = "failed"
		}
		title := s.toolTitleFor(msg.ToolCallID, msg.ToolName)
		s.notify(sessionID, sessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    msg.ToolCallID,
			Title:         title,
			Kind:          acpToolKind(msg.ToolName),
			Status:        status,
			Content:       textToolContent(msg.Content),
			RawOutput:     rawOutput,
		})
	}
}

func promptToText(blocks []contentBlock) (string, error) {
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				parts = append(parts, b.Text)
			}
		case "resource_link":
			if b.Name == "" || b.URI == "" {
				return "", fmt.Errorf("resource_link requires name and uri")
			}
			parts = append(parts, b.Name+": "+b.URI)
		case "image", "audio", "resource":
			return "", fmt.Errorf("unsupported prompt content type: %s", b.Type)
		default:
			return "", fmt.Errorf("unsupported prompt content type: %s", b.Type)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func toolRawInput(args map[string]any) map[string]any {
	raw := map[string]any{"args": args}
	for key, value := range args {
		raw[key] = value
	}
	return raw
}

func (s *server) rememberToolTitle(toolCallID, name string, args map[string]any) string {
	title := toolTitle(name, args)
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing := s.toolTitles[toolCallID]; existing != "" && existing != name {
		return existing
	}
	s.toolTitles[toolCallID] = title
	return title
}

func (s *server) toolTitleFor(toolCallID, fallback string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if title := s.toolTitles[toolCallID]; title != "" {
		return title
	}
	return fallback
}

func toolTitle(name string, args map[string]any) string {
	if args == nil {
		return name
	}

	var details []string
	switch name {
	case "bash":
		details = appendStringArg(details, "command", args)
	case "read", "write", "edit", "ls":
		details = appendStringArg(details, "path", args)
	case "grep":
		details = appendStringArg(details, "pattern", args)
		details = appendStringArg(details, "path", args)
	case "find":
		details = appendStringArg(details, "pattern", args)
		details = appendStringArg(details, "path", args)
	default:
		for _, key := range []string{"command", "path", "pattern", "query", "name"} {
			details = appendStringArg(details, key, args)
			if len(details) > 0 {
				break
			}
		}
	}

	if len(details) == 0 {
		return name
	}
	return name + ": " + truncateTitle(strings.Join(details, " "))
}

func appendStringArg(details []string, key string, args map[string]any) []string {
	value, ok := args[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return details
	}
	if key == "command" {
		return append(details, value)
	}
	return append(details, key+"="+value)
}

func truncateTitle(title string) string {
	const maxTitleLength = 160
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	if len(title) <= maxTitleLength {
		return title
	}
	return title[:maxTitleLength-3] + "..."
}

func normalizeStopReason(reason string) string {
	switch reason {
	case "", "stop", "end_turn", "tool_use":
		return "end_turn"
	case "max_tokens", "length":
		return "max_tokens"
	case "cancelled", "aborted":
		return "cancelled"
	default:
		return "refusal"
	}
}

func (s *server) nextRequestID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("acp-%d", s.nextID)
}

func (s *server) readRequest() (rpcRequest, error) {
	var req rpcRequest
	var buf bytes.Buffer
	for {
		part, err := s.r.ReadSlice('\n')
		if len(part) > 0 {
			if buf.Len()+len(part) > maxRequestBytes {
				return req, fmt.Errorf("message exceeds maximum size of %d bytes", maxRequestBytes)
			}
			buf.Write(part)
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			return req, err
		}
		break
	}
	payload := strings.TrimRight(buf.String(), "\r\n")
	if strings.TrimSpace(payload) == "" {
		return req, fmt.Errorf("empty message")
	}
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return req, err
	}
	return req, nil
}

func (s *server) writeResponse(id json.RawMessage, result any, errResp *mcp.RPCError) error {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
	}
	if errResp != nil {
		resp["error"] = errResp
	} else {
		resp["result"] = result
	}
	return s.writeMessage(resp)
}

func (s *server) notify(sessionID string, update sessionUpdate) error {
	return s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"method":  "session/update",
		"params": map[string]any{
			"sessionId": sessionID,
			"update":    update,
		},
	})
}

func (s *server) notifyExtension(method string, params any) error {
	return s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (s *server) notifyRequest(id string, method string, params any) error {
	return s.writeMessage(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
}

func (s *server) writeMessage(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.w.Write(data); err != nil {
		return err
	}
	if _, err := s.w.Write([]byte("\n")); err != nil {
		return err
	}
	if f, ok := s.w.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			return err
		}
	}
	return nil
}
