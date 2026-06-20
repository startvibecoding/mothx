package hermes

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/contextfiles"
	"github.com/startvibecoding/vibecoding/internal/cron"
	"github.com/startvibecoding/vibecoding/internal/hermes/hooks"
	"github.com/startvibecoding/vibecoding/internal/mcp"
	"github.com/startvibecoding/vibecoding/internal/memory"
	"github.com/startvibecoding/vibecoding/internal/messaging"
	"github.com/startvibecoding/vibecoding/internal/provider"
	providerfactory "github.com/startvibecoding/vibecoding/internal/provider/factory"
	"github.com/startvibecoding/vibecoding/internal/sandbox"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/skills"
	"github.com/startvibecoding/vibecoding/internal/tools"
	"github.com/startvibecoding/vibecoding/internal/util"
)

// agentApprovalHandler is the callback signature for tool approval decisions.
type agentApprovalHandler func(toolCallID, toolName string, args map[string]any) bool

// Dispatcher routes messages to per-user agent sessions.
type Dispatcher struct {
	mu         sync.RWMutex
	cfg        *HermesConfig
	settings   *config.Settings
	version    string
	sessionDir string
	security   *Security
	hooksMgr   *hooks.Manager

	// Cached provider/model for creating agent instances
	provider provider.Provider
	model    *provider.Model

	// Multi-agent mode
	multiAgent bool
	agentMgr   *agent.AgentManager

	// Cron
	cronStore cron.CronStore
	scheduler *cron.Scheduler

	// Sandbox mode
	sandbox bool

	// Active sessions: key = "hermes/<platform>/<user_id>"
	sessions map[string]*HermesSession

	// Pending approvals for WebSocket clients: approvalID → channel
	approvalMu       sync.Mutex
	pendingApprovals map[string]chan bool
}

// HermesSession holds state for a single hermes user session.
type HermesSession struct {
	ID         string // e.g. "hermes/wechat/wxid_user1"
	Platform   string // "wechat", "feishu", "ws"
	UserID     string
	WorkDir    string
	Manager    *session.Manager
	Registry   *tools.Registry
	MCPClients []*mcp.Client // connected MCP clients (nil if none)
	Mode       string
	LastUsed   time.Time
	mu         sync.Mutex // serializes requests within this session
	// ForceCompact is set by /compact command and consumed by the next agent run.
	ForceCompact bool
}

// Lock acquires the session lock.
func (s *HermesSession) Lock() { s.mu.Lock() }

// Unlock releases the session lock.
func (s *HermesSession) Unlock() { s.mu.Unlock() }

// Touch updates the last-used timestamp.
func (s *HermesSession) Touch() { s.LastUsed = time.Now() }

// NewDispatcher creates a dispatcher with the given configuration.
func NewDispatcher(cfg *HermesConfig, settings *config.Settings, version string, cronStore cron.CronStore, scheduler *cron.Scheduler) (*Dispatcher, error) {
	providerName := cfg.GetDefaultProvider(settings.DefaultProvider)
	modelID := cfg.GetDefaultModel(settings.DefaultModel)

	p, model, err := providerfactory.Create(settings, providerName, modelID)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	d := &Dispatcher{
		cfg:              cfg,
		settings:         settings,
		version:          version,
		sessionDir:       settings.GetSessionDir(),
		security:         NewSecurity(cfg),
		hooksMgr:         hooks.NewManager(cfg.Hooks.PreToolCall, cfg.Hooks.PostToolCall),
		provider:         p,
		model:            model,
		multiAgent:       cfg.MultiAgent,
		sandbox:          cfg.Sandbox,
		cronStore:        cronStore,
		scheduler:        scheduler,
		sessions:         make(map[string]*HermesSession),
		pendingApprovals: make(map[string]chan bool),
	}

	// Multi-agent mode: create AgentFactory and AgentManager
	if cfg.MultiAgent {
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

		// Extra context will be loaded per-session in resolveSession; use empty here
		factory := agent.NewAgentFactory(p, model, settings, sandbox.NewManager("."), "", nil, compactionSettings, nil)
		d.agentMgr = agent.NewAgentManager(factory)
	}

	return d, nil
}

// HandleMessage processes an inbound message from any platform.
func (d *Dispatcher) HandleMessage(ctx context.Context, msg messaging.InboundMessage) (string, error) {
	log.Printf("[hermes] HandleMessage: platform=%s userID=%s text=%q", msg.Platform, msg.UserID, truncate(msg.Text, 80))

	// Check user whitelist
	if err := d.security.CheckUserAllowed(msg.Platform, msg.UserID); err != nil {
		return "", err
	}

	// Check if command
	if strings.HasPrefix(msg.Text, "/") {
		return d.handleCommand(msg)
	}

	sess, err := d.resolveSession(msg.Platform, msg.UserID)
	if err != nil {
		return "", fmt.Errorf("resolve session: %w", err)
	}

	sess.Lock()
	defer sess.Unlock()
	sess.Touch()

	return d.runAgent(ctx, sess, msg.Text, msg.ProgressFunc)
}

// HandleWSMessage processes a message from a WebSocket client.
func (d *Dispatcher) HandleWSMessage(ctx context.Context, connID, text string, eventCh chan<- agent.Event) error {
	if strings.HasPrefix(text, "/") {
		result := d.handleCommandForWS(connID, text)
		eventCh <- agent.Event{
			Type:          agent.EventStatus,
			StatusMessage: result,
		}
		eventCh <- agent.Event{Type: agent.EventDone, Done: true}
		return nil
	}

	sess, err := d.resolveSession("ws", connID)
	if err != nil {
		return fmt.Errorf("resolve session: %w", err)
	}

	sess.Lock()
	defer sess.Unlock()
	sess.Touch()

	return d.runAgentStreaming(ctx, sess, text, eventCh)
}

// resolveSession finds or creates the active session for a platform user.
func (d *Dispatcher) resolveSession(platform, userID string) (*HermesSession, error) {
	key := sessionKey(platform, userID)

	d.mu.RLock()
	if sess, ok := d.sessions[key]; ok {
		d.mu.RUnlock()
		log.Printf("[hermes] session reused: %s", key)
		return sess, nil
	}
	d.mu.RUnlock()

	log.Printf("[hermes] session not found in cache, creating: %s", key)

	// Create or load session
	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check after acquiring write lock
	if sess, ok := d.sessions[key]; ok {
		log.Printf("[hermes] session found after write lock: %s", key)
		return sess, nil
	}

	dir := d.hermesSessionDir(platform, userID)
	activePath := filepath.Join(dir, "active.jsonl")
	workDir := d.cfg.GetPlatformWorkDir(platform)
	if err := d.security.CheckWorkDirAllowed(workDir); err != nil {
		return nil, err
	}

	var mgr *session.Manager
	if _, err := os.Stat(activePath); err == nil {
		// Load existing active session
		var openErr error
		mgr, openErr = session.Open(activePath)
		if openErr != nil {
			// Corrupt session — archive it and create new
			d.archiveCorrupt(activePath)
			mgr = nil
		}
	}

	if mgr == nil {
		// Create new session
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("create session dir: %w", err)
		}
		mgr = session.New(workDir, dir)
		if err := mgr.Init(); err != nil {
			return nil, fmt.Errorf("init session: %w", err)
		}
		// Rename the auto-generated file to active.jsonl
		if mgr.GetFile() != activePath {
			if err := os.Rename(mgr.GetFile(), activePath); err != nil {
				return nil, fmt.Errorf("rename to active.jsonl: %w", err)
			}
			// Re-open from the renamed path
			var openErr error
			mgr, openErr = session.Open(activePath)
			if openErr != nil {
				return nil, fmt.Errorf("open renamed session: %w", openErr)
			}
		}
	}

	// Build tools registry
	sbMgr := sandbox.NewManager(workDir)
	if d.sandbox {
		sbMgr.SetLevel(sandbox.LevelStandard)
	} else {
		sbMgr.SetLevel(sandbox.LevelNone)
	}
	reg := tools.NewRegistry(workDir, sbMgr.GetActive())
	reg.RegisterDefaults()

	// Register memory tool
	memStore := memory.NewStore(d.cfg.Memory.Path, workDir)
	reg.Register(memory.NewMemoryTool(memStore))

	// Register subagent tools when multi-agent mode is enabled
	if d.agentMgr != nil {
		agent.RegisterSubAgentTools(reg, d.agentMgr)
	}

	// Register cron tool when cron store is available
	if d.cronStore != nil {
		reg.Register(cron.NewCronTool(d.cronStore, d.scheduler))
	}

	// Load and connect MCP servers
	var mcpClients []*mcp.Client
	mcpServers, err := mcp.LoadConfiguredServers(workDir)
	if err != nil {
		log.Printf("[hermes] load MCP servers: %v", err)
	} else if len(mcpServers) > 0 {
		clients, err := mcp.ConnectServers(context.Background(), mcpServers, reg, mcp.Callbacks{})
		if err != nil {
			log.Printf("[hermes] connect MCP servers: %v", err)
		} else {
			mcpClients = clients
			log.Printf("[hermes] connected %d MCP server(s) for %s/%s", len(clients), platform, userID)
		}
	}

	sess := &HermesSession{
		ID:         key,
		Platform:   platform,
		UserID:     userID,
		WorkDir:    workDir,
		Manager:    mgr,
		Registry:   reg,
		MCPClients: mcpClients,
		Mode:       "yolo",
		LastUsed:   time.Now(),
	}

	d.sessions[key] = sess
	log.Printf("[hermes] session created: %s (workDir=%s)", key, workDir)
	return sess, nil
}

// RotateSession archives the current session and creates a new one.
// Called when user sends /new.
func (d *Dispatcher) RotateSession(platform, userID string) error {
	key := sessionKey(platform, userID)
	log.Printf("[hermes] rotating session: %s", key)

	d.mu.Lock()
	defer d.mu.Unlock()

	dir := d.hermesSessionDir(platform, userID)
	activePath := filepath.Join(dir, "active.jsonl")

	// Archive existing active session
	if _, err := os.Stat(activePath); err == nil {
		mgr, err := session.Open(activePath)
		if err == nil {
			hdr := mgr.GetHeader()
			idPrefix := "unknown"
			if hdr != nil && len(hdr.ID) >= 8 {
				idPrefix = hdr.ID[:8]
			}
			archived := filepath.Join(dir, fmt.Sprintf("%s_%s.jsonl",
				time.Now().Format("20060102-150405"), idPrefix))
			os.Rename(activePath, archived)
		} else {
			// Can't parse — just rename with timestamp
			archived := filepath.Join(dir, fmt.Sprintf("%s_corrupt.jsonl",
				time.Now().Format("20060102-150405")))
			os.Rename(activePath, archived)
		}
	}

	// Close MCP clients and remove from cache so next message creates fresh session
	if sess, ok := d.sessions[key]; ok {
		if len(sess.MCPClients) > 0 {
			mcp.CloseClients(sess.MCPClients)
		}
	}
	delete(d.sessions, key)

	return nil
}

// GetSession returns a session by key, or nil if not found.
func (d *Dispatcher) GetSession(key string) *HermesSession {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.sessions[key]
}

// ListSessions returns all active session keys.
func (d *Dispatcher) ListSessions() []*HermesSession {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]*HermesSession, 0, len(d.sessions))
	for _, s := range d.sessions {
		result = append(result, s)
	}
	return result
}

// RemoveSession removes a session from the pool.
func (d *Dispatcher) RemoveSession(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if sess, ok := d.sessions[key]; ok {
		if len(sess.MCPClients) > 0 {
			mcp.CloseClients(sess.MCPClients)
		}
		delete(d.sessions, key)
	}
}

// buildAgent creates and configures an agent for a session.
// Returns the agent and a cleanup function to call with the run error.
func (d *Dispatcher) buildAgent(ctx context.Context, sess *HermesSession, approvalHandler agentApprovalHandler) (*agent.Agent, func(error)) {
	workDir := sess.WorkDir
	extraContext := d.buildExtraContext(workDir)
	compactionSettings := ctxpkg.NormalizeCompactionSettings(ctxpkg.CompactionSettings{
		Enabled:          d.settings.Compaction.Enabled,
		ReserveTokens:    d.settings.Compaction.ReserveTokens,
		KeepRecentTokens: d.settings.Compaction.KeepRecentTokens,
		Tokenizer:        d.settings.Compaction.Tokenizer,
		TokenizerModel:   d.settings.Compaction.TokenizerModel,
		Template:         d.settings.Compaction.Template,
	})

	agentCfg := agent.Config{
		Provider:           d.provider,
		Model:              d.model,
		Mode:               sess.Mode,
		ThinkingLevel:      provider.ThinkingLevel(d.settings.DefaultThinkingLevel),
		SandboxMgr:         sandbox.NewManager(workDir),
		Settings:           d.settings,
		Session:            sess.Manager,
		ExtraContext:       extraContext,
		CompactionSettings: compactionSettings,
		MultiAgent:         d.multiAgent,
		ApprovalHandler:    approvalHandler,
	}

	a := agent.NewWithLoopConfig(agent.AgentLoopConfig{
		Config:                   agentCfg,
		MaxIterations:            d.cfg.Agent.MaxTurns,
		ContextPressureThreshold: d.cfg.Agent.ContextPressureThreshold,
		BudgetPressureThreshold:  d.cfg.Agent.BudgetPressureThreshold,
		AfterToolCall: func(ctx2 agent.AfterToolCallContext) *agent.ToolCallResult {
			if d.hooksMgr.HasPostHook() {
				argsMap, _ := ctx2.Args.(map[string]any)
				errMsg := ""
				if ctx2.IsError {
					errMsg = ctx2.Result.Content
				}
				d.hooksMgr.PostToolCall(ctx, ctx2.ToolCall.Name, argsMap, ctx2.Result.Content, errMsg, sess.Platform, sess.UserID)
			}
			return nil
		},
	}, sess.Registry)

	var runErr error
	if d.agentMgr != nil {
		d.agentMgr.Register(agent.NewAgentAdapter(a))
	}
	cleanup := func(err error) {
		runErr = err
		if d.agentMgr != nil {
			d.agentMgr.Finish(a.ID(), runErr)
		}
	}

	if sess.ForceCompact {
		a.SetForceCompact()
		sess.ForceCompact = false
	}

	if replayState := sess.Manager.GetReplayState(); len(replayState.Messages) > 0 {
		a.LoadHistoryState(replayState.Messages, replayState.EntryIDs)
	}

	return a, cleanup
}

// messagingApprovalHandler returns an ApprovalHandler for messaging platforms.
// Medium risk → auto-approve + notify; high risk → auto-reject + notify.
func (d *Dispatcher) messagingApprovalHandler(ctx context.Context, sess *HermesSession, progress func(string)) agentApprovalHandler {
	return func(toolCallID, toolName string, args map[string]any) bool {
		if d.security.ShouldAutoApprove(toolName, args, sess.Mode) {
			return true
		}

		risk := "medium"
		if toolName == "bash" {
			if cmd, ok := args["command"]; ok {
				risk = CommandRiskLevel(fmt.Sprintf("%v", cmd))
			}
		}

		if d.hooksMgr.HasPreHook() {
			allowed, _, _ := d.hooksMgr.PreToolCall(ctx, toolName, args, sess.Platform, sess.UserID)
			if allowed {
				return true
			}
		}

		if risk == "medium" {
			if progress != nil {
				progress(FormatApprovalNotification(toolName, args, risk, true))
			}
			return true
		}

		if progress != nil {
			progress(FormatApprovalNotification(toolName, args, risk, false))
		}
		return false
	}
}

// wsApprovalHandler returns an ApprovalHandler for WebSocket clients.
// Medium risk → auto-approve; high risk → send approval request and wait.
func (d *Dispatcher) wsApprovalHandler(ctx context.Context, sess *HermesSession, eventCh chan<- agent.Event) agentApprovalHandler {
	return func(toolCallID, toolName string, args map[string]any) bool {
		if d.security.ShouldAutoApprove(toolName, args, sess.Mode) {
			return true
		}

		risk := "medium"
		if toolName == "bash" {
			if cmd, ok := args["command"]; ok {
				risk = CommandRiskLevel(fmt.Sprintf("%v", cmd))
			}
		}

		if d.hooksMgr.HasPreHook() {
			allowed, _, _ := d.hooksMgr.PreToolCall(ctx, toolName, args, sess.Platform, sess.UserID)
			if allowed {
				return true
			}
		}

		if risk == "medium" {
			eventCh <- agent.Event{
				Type:          agent.EventStatus,
				StatusMessage: FormatApprovalNotification(toolName, args, risk, true),
			}
			return true
		}

		approvalID := fmt.Sprintf("ap_%s_%d", toolCallID, time.Now().UnixNano())
		respCh := d.RegisterApproval(approvalID)

		eventCh <- agent.Event{
			Type:         agent.EventToolApprovalRequest,
			ApprovalID:   approvalID,
			ApprovalTool: toolName,
			ApprovalArgs: args,
		}

		select {
		case approved := <-respCh:
			if approved {
				eventCh <- agent.Event{
					Type:          agent.EventStatus,
					StatusMessage: fmt.Sprintf("✅ [%s] approved by user", toolName),
				}
			}
			return approved
		case <-time.After(5 * time.Minute):
			d.approvalMu.Lock()
			delete(d.pendingApprovals, approvalID)
			d.approvalMu.Unlock()
			eventCh <- agent.Event{
				Type:          agent.EventStatus,
				StatusMessage: fmt.Sprintf("⏰ [%s] approval timed out — blocked", toolName),
			}
			return false
		case <-ctx.Done():
			return false
		}
	}
}

// runAgent executes the agent loop synchronously (for messaging platforms).
func (d *Dispatcher) runAgent(ctx context.Context, sess *HermesSession, userInput string, progress func(string)) (string, error) {
	a, cleanup := d.buildAgent(ctx, sess, d.messagingApprovalHandler(ctx, sess, progress))
	var runErr error
	defer cleanup(runErr)

	eventCh := a.Run(ctx, userInput)

	var response strings.Builder
	var thinkBuf strings.Builder
	var eventCount int
	var toolCount int
	pendingToolArgs := make(map[string]map[string]any) // ToolCallID → args
	flushThink := func() {
		if progress != nil && thinkBuf.Len() > 0 {
			text := thinkBuf.String()
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			progress("💭 " + text)
			thinkBuf.Reset()
		}
	}
	for ev := range eventCh {
		eventCount++
		switch ev.Type {
		case agent.EventThinkDelta:
			thinkBuf.WriteString(ev.ThinkDelta)
		case agent.EventTextDelta:
			flushThink()
			response.WriteString(ev.TextDelta)
		case agent.EventToolExecutionStart:
			if ev.ToolCallID != "" && ev.ToolArgs != nil {
				pendingToolArgs[ev.ToolCallID] = ev.ToolArgs
			}
		case agent.EventToolExecutionEnd:
			flushThink()
			toolCount++
			if progress != nil {
				args := pendingToolArgs[ev.ToolCallID]
				delete(pendingToolArgs, ev.ToolCallID)
				line := formatToolProgress(ev, args)
				if line != "" {
					progress(line)
				}
			}
		case agent.EventContextPressure, agent.EventBudgetPressure:
			// Forward pressure warnings to messaging platform
			if progress != nil && ev.PressureMessage != "" {
				progress("\n" + ev.PressureMessage)
			}
			log.Printf("[hermes] %s pressure event for %s/%s: %s", ev.PressureType, sess.Platform, sess.UserID, ev.PressureMessage)
		case agent.EventError:
			flushThink()
			if ev.Error != nil {
				runErr = ev.Error
				log.Printf("[hermes] Agent error for %s/%s: %v", sess.Platform, sess.UserID, ev.Error)
				return "", ev.Error
			}
		}
	}

	result := response.String()
	log.Printf("[hermes] Agent completed for %s/%s: events=%d, tools=%d, response_len=%d", sess.Platform, sess.UserID, eventCount, toolCount, len(result))

	// If agent produced no text but executed tools, provide a fallback summary
	if result == "" && toolCount > 0 {
		result = fmt.Sprintf("✅ Done (%d tool calls completed)", toolCount)
	}

	return result, nil
}

// formatToolProgress formats a tool execution event into a concise one-line progress string.
func formatToolProgress(ev agent.Event, args map[string]any) string {
	name := ev.ToolName
	if name == "" && ev.ToolCall != nil {
		name = ev.ToolCall.Name
	}
	if name == "" {
		return ""
	}

	var icon string
	if ev.ToolError != nil {
		icon = "❌"
	} else {
		icon = "✅"
	}

	// Build a concise summary per tool type
	switch name {
	case "read", "write", "edit":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("[%s]: %s %s", name, path, icon)
		}
	case "bash":
		if cmd, ok := args["command"].(string); ok {
			if len(cmd) > 60 {
				cmd = cmd[:60] + "..."
			}
			return fmt.Sprintf("[bash]: %s %s", cmd, icon)
		}
	case "grep":
		if pat, ok := args["pattern"].(string); ok {
			return fmt.Sprintf("[grep]: %s %s", pat, icon)
		}
	case "find":
		if pat, ok := args["pattern"].(string); ok {
			return fmt.Sprintf("[find]: %s %s", pat, icon)
		}
	case "ls":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("[ls]: %s %s", path, icon)
		}
	}

	return fmt.Sprintf("[%s] %s", name, icon)
}

// runAgentStreaming executes the agent loop and sends events to the channel (for WebSocket).
// The eventCh is closed when the agent loop completes.
func (d *Dispatcher) runAgentStreaming(ctx context.Context, sess *HermesSession, userInput string, eventCh chan<- agent.Event) error {
	defer close(eventCh)

	a, cleanup := d.buildAgent(ctx, sess, d.wsApprovalHandler(ctx, sess, eventCh))
	var runErr error
	defer func() {
		UnregisterActiveAgent(string(a.ID()))
		cleanup(runErr)
	}()

	// Register agent for question resolution via WebSocket
	RegisterActiveAgent(string(a.ID()), a)

	agentCh := a.Run(ctx, userInput)

	for ev := range agentCh {
		if ev.Type == agent.EventError {
			runErr = ev.Error
		}
		eventCh <- ev
	}
	return nil
}

// buildExtraContext loads context files and skills for a working directory.
func (d *Dispatcher) buildExtraContext(workDir string) string {
	var extra string
	if d.settings.ContextFiles.Enabled {
		cfResult := contextfiles.LoadContextFiles(workDir, config.ConfigDir(), d.settings.ContextFiles.ExtraFiles)
		if ctx := contextfiles.BuildContextString(cfResult); ctx != "" {
			extra = ctx
		}
	}

	skillsMgr := skills.NewManagerWithProjectDirs(d.settings.GetGlobalSkillsDir(), skills.ProjectSkillDirs(workDir))
	_ = skillsMgr.Load()
	extra += skillsMgr.BuildAllSkillsContext()

	return extra
}

// handleCommand processes slash commands from messaging platforms.
func (d *Dispatcher) handleCommand(msg messaging.InboundMessage) (string, error) {
	parts := strings.Fields(msg.Text)
	if len(parts) == 0 {
		return "", nil
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/new":
		if err := d.RotateSession(msg.Platform, msg.UserID); err != nil {
			return "❌ Failed to create new session: " + err.Error(), nil
		}
		return "✅ New session created.", nil
	case "/clear":
		if err := d.RotateSession(msg.Platform, msg.UserID); err != nil {
			return "❌ Failed to clear session: " + err.Error(), nil
		}
		return "✅ Session cleared.", nil
	case "/status":
		sess := d.GetSession(sessionKey(msg.Platform, msg.UserID))
		if sess == nil {
			return "No active session.", nil
		}
		msgs := sess.Manager.GetMessages()
		return fmt.Sprintf("Session: %s\nMode: %s\nMessages: %d\nWorkDir: %s",
			sess.ID, sess.Mode, len(msgs), sess.WorkDir), nil
	case "/sessions":
		sessions := d.ListSessions()
		if len(sessions) == 0 {
			return "No active sessions.", nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Active sessions (%d):\n", len(sessions)))
		for _, s := range sessions {
			msgs := s.Manager.GetMessages()
			sb.WriteString(fmt.Sprintf("  • %s (%d msgs, %s)\n", s.ID, len(msgs), s.WorkDir))
		}
		return sb.String(), nil
	case "/mode":
		if len(parts) < 2 {
			sess := d.GetSession(sessionKey(msg.Platform, msg.UserID))
			if sess != nil {
				return fmt.Sprintf("Current mode: %s", sess.Mode), nil
			}
			return "No active session.", nil
		}
		mode := strings.ToLower(parts[1])
		switch mode {
		case "plan", "agent", "yolo":
			sess, err := d.resolveSession(msg.Platform, msg.UserID)
			if err != nil {
				return "❌ No active session.", nil
			}
			sess.Mode = mode
			return fmt.Sprintf("✅ Mode set to %s.", mode), nil
		default:
			return "Invalid mode. Use: plan, agent, yolo", nil
		}
	case "/compact":
		sess, err := d.resolveSession(msg.Platform, msg.UserID)
		if err != nil {
			return "❌ No active session.", nil
		}
		sess.Lock()
		defer sess.Unlock()
		if sess.Manager == nil || len(sess.Manager.GetMessages()) < 2 {
			return "Nothing to compact: conversation is too short.", nil
		}
		previousSummary := ""
		if compaction, ok := sess.Manager.GetLatestCompaction(); ok {
			previousSummary = compaction.Summary
		}
		compactionSettings := ctxpkg.NormalizeCompactionSettings(ctxpkg.CompactionSettings{
			Enabled:          d.settings.Compaction.Enabled,
			ReserveTokens:    d.settings.Compaction.ReserveTokens,
			KeepRecentTokens: d.settings.Compaction.KeepRecentTokens,
			Tokenizer:        d.settings.Compaction.Tokenizer,
			TokenizerModel:   d.settings.Compaction.TokenizerModel,
			Template:         d.settings.Compaction.Template,
		})
		if !ctxpkg.HasCompactableMessages(sess.Manager.GetReplayState().Messages, d.model, compactionSettings, previousSummary) {
			return "Nothing to compact: only recent context is available to keep.", nil
		}
		sess.ForceCompact = true
		return "✅ Context compaction will be triggered on the next message.", nil
	default:
		return fmt.Sprintf("Unknown command: %s\nAvailable: /new /clear /status /sessions /mode /compact", cmd), nil
	}
}

// handleCommandForWS processes slash commands from WebSocket clients.
func (d *Dispatcher) handleCommandForWS(connID, text string) string {
	msg := messaging.InboundMessage{
		Platform: "ws",
		UserID:   connID,
		Text:     text,
	}
	result, err := d.handleCommand(msg)
	if err != nil {
		return "❌ " + err.Error()
	}
	return result
}

// hermesSessionDir returns the directory for a platform user's sessions.
func (d *Dispatcher) hermesSessionDir(platform, userID string) string {
	return filepath.Join(d.sessionDir, "hermes", safeSessionPathComponent(platform), safeSessionPathComponent(userID))
}

// sessionKey builds a session pool key.
func sessionKey(platform, userID string) string {
	return fmt.Sprintf("hermes/%s/%s", platform, userID)
}

func safeSessionPathComponent(s string) string {
	if s == "" || s == "." || s == ".." {
		return "b64_" + base64.RawURLEncoding.EncodeToString([]byte(s))
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '-', '_', '.', '@':
			continue
		default:
			return "b64_" + base64.RawURLEncoding.EncodeToString([]byte(s))
		}
	}
	return s
}

// archiveCorrupt renames a corrupt session file.
func (d *Dispatcher) archiveCorrupt(path string) {
	dir := filepath.Dir(path)
	archived := filepath.Join(dir, fmt.Sprintf("%s_corrupt.jsonl",
		time.Now().Format("20060102-150405")))
	os.Rename(path, archived)
}

// RegisterApproval registers a pending approval and returns its channel.
func (d *Dispatcher) RegisterApproval(approvalID string) chan bool {
	ch := make(chan bool, 1)
	d.approvalMu.Lock()
	d.pendingApprovals[approvalID] = ch
	d.approvalMu.Unlock()
	return ch
}

// ResolveApproval resolves a pending approval with the given decision.
func (d *Dispatcher) ResolveApproval(approvalID string, approved bool) bool {
	d.approvalMu.Lock()
	ch, ok := d.pendingApprovals[approvalID]
	if ok {
		delete(d.pendingApprovals, approvalID)
	}
	d.approvalMu.Unlock()

	if ok {
		// Use select to avoid blocking if the channel was already consumed
		// (e.g., timeout raced with this call).
		select {
		case ch <- approved:
		default:
		}
		return true
	}
	return false
}

// activeAgents tracks running agents by ID for question resolution.
// Key: agent ID (string), Value: *agent.Agent
var activeAgents sync.Map

// RegisterActiveAgent registers a running agent for question resolution.
func RegisterActiveAgent(id string, a *agent.Agent) {
	activeAgents.Store(id, a)
}

// UnregisterActiveAgent removes an agent from the registry.
func UnregisterActiveAgent(id string) {
	activeAgents.Delete(id)
}

func truncate(s string, maxLen int) string {
	return util.TruncateWithSuffix(s, maxLen, "...")
}
