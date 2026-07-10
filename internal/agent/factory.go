package agent

import (
	"os"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/config"
	ctxpkg "github.com/startvibecoding/mothx/internal/context"
	"github.com/startvibecoding/mothx/internal/platform"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/tools"
)

// AgentFactory creates Agent instances with consistent configuration.
type AgentFactory struct {
	provider           provider.Provider
	providerName       string
	model              *provider.Model
	settings           *config.Settings
	allow              *config.AllowConfig
	sandboxMgr         *sandbox.Manager
	extraContext       string
	ruleContent        string
	skillsMgr          *skills.Manager
	compactionSettings ctxpkg.CompactionSettings
	approvalHandler    func(toolCallID, toolName string, args map[string]any) bool
	multiAgentEnabled  bool
	delegateEnabled    bool
	workflowsEnabled   bool
}

// NewAgentFactory creates a factory with shared configuration.
func NewAgentFactory(
	provider provider.Provider,
	model *provider.Model,
	settings *config.Settings,
	sandboxMgr *sandbox.Manager,
	extraContext string,
	ruleContent string,
	skillsMgr *skills.Manager,
	compactionSettings ctxpkg.CompactionSettings,
	approvalHandler func(toolCallID, toolName string, args map[string]any) bool,
) *AgentFactory {
	return NewAgentFactoryWithOptions(provider, model, settings, sandboxMgr, extraContext, ruleContent, skillsMgr, compactionSettings, approvalHandler, AgentFactoryOptions{
		MultiAgentEnabled: true,
	})
}

// AgentFactoryOptions configures AgentFactory behavior.
type AgentFactoryOptions struct {
	MultiAgentEnabled bool
	DelegateEnabled   bool
	WorkflowsEnabled  bool
	ProviderName      string
	Allow             *config.AllowConfig
}

// NewAgentFactoryWithOptions creates a factory with explicit behavior flags.
func NewAgentFactoryWithOptions(
	provider provider.Provider,
	model *provider.Model,
	settings *config.Settings,
	sandboxMgr *sandbox.Manager,
	extraContext string,
	ruleContent string,
	skillsMgr *skills.Manager,
	compactionSettings ctxpkg.CompactionSettings,
	approvalHandler func(toolCallID, toolName string, args map[string]any) bool,
	opts AgentFactoryOptions,
) *AgentFactory {
	allow := opts.Allow
	if allow == nil {
		allow = config.LoadAllow()
	}
	return &AgentFactory{
		provider:           provider,
		providerName:       opts.ProviderName,
		model:              model,
		settings:           settings,
		allow:              allow,
		sandboxMgr:         sandboxMgr,
		extraContext:       extraContext,
		ruleContent:        ruleContent,
		skillsMgr:          skillsMgr,
		compactionSettings: compactionSettings,
		approvalHandler:    approvalHandler,
		multiAgentEnabled:  opts.MultiAgentEnabled,
		delegateEnabled:    opts.DelegateEnabled,
		workflowsEnabled:   opts.WorkflowsEnabled,
	}
}

// AgentOptions specifies per-agent overrides.
type AgentOptions struct {
	ID                agentpkg.AgentID
	ParentID          agentpkg.AgentID
	Mode              string
	Model             *provider.Model
	WorkDir           string
	Tools             []string // optional: tool filter
	SystemPromptExtra string   // extra context for this agent
	MaxIterations     int
	ToolExecutionMode string
	Session           *session.Manager
	ApprovalHandler   func(toolCallID, toolName string, args map[string]any) bool // per-agent approval override
	MultiAgent        *bool                                                       // optional prompt override
	DelegateMode      *bool                                                       // optional prompt override
	Workflows         *bool                                                       // optional prompt override
}

// Create creates a new Agent with per-agent Registry.
// Each agent gets its own Registry (with its own workDir, sandbox, JobManager).
func (f *AgentFactory) Create(opts AgentOptions) agentpkg.Agent {
	workDir := opts.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	mode := opts.Mode
	if mode == "" {
		mode = "agent"
	}

	model := opts.Model
	if model == nil {
		model = f.model
	}

	maxIterations := opts.MaxIterations
	if maxIterations == 0 {
		maxIterations = 200
	}

	toolExecMode := opts.ToolExecutionMode
	if toolExecMode == "" {
		toolExecMode = "parallel"
	}

	// Create per-agent Registry with isolated workDir/sandbox/JobManager
	sb := f.sandboxForMode(mode)
	registry := tools.NewRegistryWithConfig(tools.RegistryConfig{
		WorkDir:        workDir,
		Sandbox:        sb,
		ToolFilter:     opts.Tools,
		SkillsMgr:      f.skillsMgr,
		EnablePlanTool: config.BoolPtr(f.settings == nil || f.settings.IsPlanToolEnabled()),
	})

	// Decision 5: Sub-agents cannot spawn sub-agents
	// Remove subagent_* tools from sub-agent registries
	if opts.ParentID != "" {
		registry.Remove("subagent_spawn")
		registry.Remove("subagent_status")
		registry.Remove("subagent_send")
		registry.Remove("subagent_destroy")
		registry.Remove("delegate_subagent")
	}

	// Build extra context: factory-level + per-agent
	extraContext := f.extraContext
	if opts.ParentID != "" {
		extraContext += "\n" + BuildSubAgentContext()
	}
	if opts.SystemPromptExtra != "" {
		extraContext += "\n" + opts.SystemPromptExtra
	}

	// Determine session
	sess := opts.Session
	if sess == nil {
		sess = f.defaultSession(workDir)
	}

	multiAgent := f.multiAgentEnabled && opts.ParentID == ""
	if opts.MultiAgent != nil {
		multiAgent = *opts.MultiAgent
	}
	delegateMode := f.delegateEnabled && opts.ParentID == ""
	if opts.DelegateMode != nil {
		delegateMode = *opts.DelegateMode
	}
	workflows := f.workflowsEnabled && opts.ParentID == ""
	if opts.Workflows != nil {
		workflows = *opts.Workflows
	}
	if opts.ParentID != "" {
		delegateMode = false
		workflows = false
	}

	cfg := Config{
		ID:       opts.ID,
		ParentID: opts.ParentID,
		Provider: f.provider,
		Vendor:   f.providerName,
		Model:    model,
		Mode:     mode,
		ThinkingLevel: func() provider.ThinkingLevel {
			if f.settings != nil {
				return provider.ThinkingLevel(f.settings.DefaultThinkingLevel)
			}
			return provider.ThinkingLevel(agentpkg.ThinkingMedium)
		}(),
		MaxTokens: func() int {
			return ResolveMaxTokens(f.settings, model)
		}(),
		SandboxMgr:         f.sandboxMgr,
		Settings:           f.settings,
		Allow:              f.allow,
		Session:            sess,
		ExtraContext:       extraContext,
		RuleContent:        f.ruleContent,
		CompactionSettings: f.compactionSettings,
		ApprovalHandler: func() func(toolCallID, toolName string, args map[string]any) bool {
			if opts.ApprovalHandler != nil {
				return opts.ApprovalHandler
			}
			return f.approvalHandler
		}(),
		MultiAgent:   multiAgent,
		DelegateMode: delegateMode,
		Workflows:    workflows,
	}

	loopCfg := AgentLoopConfig{
		Config:            cfg,
		ToolExecutionMode: toolExecMode,
		MaxIterations:     maxIterations,
	}

	a := NewWithLoopConfig(loopCfg, registry)
	return NewAgentAdapter(a)
}

func (f *AgentFactory) withParentRuntimeConfig(cfg AgentLoopConfig) *AgentFactory {
	if f == nil {
		return nil
	}
	clone := *f
	clone.provider = cfg.Provider
	clone.providerName = cfg.Vendor
	clone.model = cfg.Model
	clone.settings = cfg.Settings
	clone.allow = cfg.Allow
	clone.extraContext = cfg.ExtraContext
	clone.ruleContent = cfg.RuleContent
	clone.compactionSettings = cfg.CompactionSettings
	clone.approvalHandler = cfg.ApprovalHandler
	return &clone
}

func (f *AgentFactory) withRuntimeConfig(p provider.Provider, providerName string, model *provider.Model, settings *config.Settings, allow *config.AllowConfig) *AgentFactory {
	if f == nil {
		return nil
	}
	clone := *f
	if p != nil {
		clone.provider = p
	}
	clone.providerName = providerName
	if model != nil {
		clone.model = model
	}
	if settings != nil {
		clone.settings = settings
	}
	if allow != nil {
		clone.allow = allow
	}
	return &clone
}

// CreateFromPublicOptions creates an agent from public Builder options.
func (f *AgentFactory) CreateFromPublicOptions(b *agentpkg.Builder) agentpkg.Agent {
	if b == nil {
		return nil
	}
	agent, err := buildFromPublicBuilder(b)
	if err != nil {
		return nil
	}
	return agent
}

// sandboxForMode returns the appropriate sandbox for the given mode.
func (f *AgentFactory) sandboxForMode(mode string) sandbox.Sandbox {
	if f.sandboxMgr == nil {
		return sandbox.NewNoneSandbox()
	}
	switch mode {
	case "plan":
		return f.sandboxMgr.GetActive()
	case "agent":
		return f.sandboxMgr.GetActive()
	case "yolo":
		return sandbox.NewNoneSandbox()
	default:
		return f.sandboxMgr.GetActive()
	}
}

// defaultSession creates a default session manager for the given work directory.
func (f *AgentFactory) defaultSession(workDir string) *session.Manager {
	sessionDir := ""
	if f.settings != nil {
		sessionDir = f.settings.GetSessionDir()
	}
	if sessionDir == "" {
		sessionDir = platform.SessionDir()
	}
	return session.New(workDir, sessionDir)
}

// Provider returns the factory's provider (for Builder integration).
func (f *AgentFactory) Provider() provider.Provider { return f.provider }

// Settings returns the factory's settings.
func (f *AgentFactory) Settings() *config.Settings { return f.settings }

// --- Register the internal builder with the public agent package ---

func init() {
	agentpkg.SetBuilderFunc(buildFromPublicBuilder)
}

// buildFromPublicBuilder converts a public Builder into an internal Agent.
// This bridges the public agent.Builder API to the internal Agent implementation.
func buildFromPublicBuilder(b *agentpkg.Builder) (agentpkg.Agent, error) {
	cfg := b.Config()

	// Adapt the public Provider to the internal provider.Provider interface
	internalProvider := NewProviderAdapter(cfg.Provider)

	// Resolve the model from the provider
	model := internalProvider.GetModel(cfg.ModelID)
	if model == nil {
		// If the model is not found, create a minimal model entry
		model = &provider.Model{
			ID:   cfg.ModelID,
			Name: cfg.ModelID,
		}
	}

	// Build compaction settings
	compactionSettings := ctxpkg.CompactionSettings{
		Enabled:       cfg.CompactionEnabled,
		ReserveTokens: cfg.CompactionReserve,
	}
	if compactionSettings.ReserveTokens == 0 {
		compactionSettings.ReserveTokens = 16384
	}

	// Build sandbox
	var sandboxMgr *sandbox.Manager
	if cfg.SandboxEnabled {
		sandboxMgr = sandbox.NewManager(cfg.WorkDir)
	}

	// Build session
	var sess *session.Manager
	if cfg.SessionDir != "" {
		sess = session.New(cfg.WorkDir, cfg.SessionDir)
	}

	// Build the tool registry
	var sb sandbox.Sandbox
	if sandboxMgr != nil {
		sb = sandboxMgr.GetActive()
	} else {
		sb = sandbox.NewNoneSandbox()
	}
	var registry *tools.Registry
	if cfg.DisableBuiltinTools {
		// External-only mode: start with an empty registry so the agent may
		// use ONLY the host-provided external tools.
		registry = tools.NewRegistry(cfg.WorkDir, sb)
	} else {
		registry = tools.NewRegistryWithConfig(tools.RegistryConfig{
			WorkDir:    cfg.WorkDir,
			Sandbox:    sb,
			ToolFilter: cfg.Tools,
		})
	}
	// Register host-provided external tools.
	for _, et := range cfg.ExternalTools {
		if et == nil {
			continue
		}
		registry.Register(newExternalToolAdapter(et))
	}

	agentCfg := Config{
		Provider:           internalProvider,
		Model:              model,
		Mode:               cfg.Mode,
		ThinkingLevel:      provider.ThinkingLevel(cfg.ThinkingLevel),
		MaxTokens:          ResolveMaxTokensValue(cfg.MaxTokens, model),
		SandboxMgr:         sandboxMgr,
		Session:            sess,
		ExtraContext:       cfg.SystemPromptExtra,
		CompactionSettings: compactionSettings,
		ApprovalHandler:    cfg.ApprovalHandler,
		MultiAgent:         cfg.MultiAgent,
		DelegateMode:       cfg.DelegateMode,
	}

	loopCfg := AgentLoopConfig{
		Config:            agentCfg,
		ToolExecutionMode: cfg.ToolExecutionMode,
		MaxIterations:     cfg.MaxIterations,
	}

	a := NewWithLoopConfig(loopCfg, registry)
	return NewAgentAdapter(a), nil
}
