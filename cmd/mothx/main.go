package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/startvibecoding/mothx/internal/a2a"
	"github.com/startvibecoding/mothx/internal/acp"
	"github.com/startvibecoding/mothx/internal/agent"
	browserfeature "github.com/startvibecoding/mothx/internal/browser"
	"github.com/startvibecoding/mothx/internal/config"
	ctxpkg "github.com/startvibecoding/mothx/internal/context"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/debugpprof"
	"github.com/startvibecoding/mothx/internal/mcp"
	"github.com/startvibecoding/mothx/internal/platform"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/serve"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/systeminit"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/tui"
	"github.com/startvibecoding/mothx/internal/update"
	"github.com/startvibecoding/mothx/internal/workflow"
)

var version = "dev"

func main() {
	if cwd, err := os.Getwd(); err == nil {
		config.AutoMigrateLegacyDirs(cwd)
	} else {
		config.AutoMigrateLegacyDirs(".")
	}
	_ = platform.EnsureWindowsBusybox()
	rootCmd := newRootCommand(run, acp.Run)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand(runFn func([]string, runOptions) error, acpRunFn func(acp.RunOptions) error) *cobra.Command {
	flags := &cliFlags{}

	rootCmd := newCLICommand(flags, runFn)
	rootCmd.AddCommand(newACPCommand(flags, acpRunFn))
	rootCmd.AddCommand(newServeCommand(flags))
	rootCmd.AddCommand(newA2ACommand())
	rootCmd.AddCommand(newDoctorCommand())
	rootCmd.AddCommand(newSystemInitCommand(runFn, &flags.provider, &flags.model))
	rootCmd.AddCommand(newStatsCommand())
	rootCmd.AddCommand(newSpeedtestCommand())
	installFriendlyFlagErrors(rootCmd)
	return rootCmd
}

type cliFlags struct {
	provider        string
	model           string
	mode            string
	thinking        string
	continueSession bool
	resume          string
	session         string
	sandbox         bool
	print           bool
	verbose         bool
	debug           bool
	multiAgent      bool
	delegate        bool
	workflows       bool
	cron            bool
	webSearch       bool
	browser         bool
	initServe       bool
	force           bool
	enableA2AMaster bool
	initA2AMaster   bool
	workDir         string
	serveConfig     string
	servePort       string
	serveWebUIDir   string
	serveUnsafe     bool
	lobsterMode     bool
}

func newCLICommand(flags *cliFlags, runFn func([]string, runOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:     cliUseName(),
		Aliases: []string{"vc"},
		Short:   "MothX - AI coding assistant",
		Long:    "MothX is an AI-powered coding assistant that runs in your terminal.\nSupports OpenAI and Anthropic APIs with sandboxed execution.",
		Version: version,
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRootArgs(cmd, args, flags); err != nil {
				return err
			}
			return runCLICommand(args, flags, runFn)
		},
	}
	registerRootFlags(cmd.Flags(), flags)
	return cmd
}

func cliUseName() string {
	base := filepath.Base(os.Args[0])
	if strings.HasPrefix(base, "mothx") {
		return "mothx"
	}
	return "vibecoding"
}

func runCLICommand(args []string, flags *cliFlags, runFn func([]string, runOptions) error) error {
	if flags.initA2AMaster {
		path, err := a2a.InitA2AMasterConfig(flags.force)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Created a2a master config: %s\n", path)
		return nil
	}
	if flags.initServe {
		path, err := serve.InitConfig(flags.force)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Created serve config: %s\n", path)
		return nil
	}
	return runFn(args, flags.runOptions())
}

func newACPCommand(flags *cliFlags, acpRunFn func(acp.RunOptions) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acp",
		Short: "Run the Agent Client Protocol server",
		Long:  "Run vibecoding as an ACP-compliant stdio agent.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return acpRunFn(flags.acpOptions())
		},
	}
	registerACPFlags(cmd.Flags(), flags)
	return cmd
}

func registerRootFlags(fs *pflag.FlagSet, flags *cliFlags) {
	registerSharedProviderFlags(fs, flags)
	fs.BoolVarP(&flags.continueSession, "continue", "c", false, "Continue most recent session")
	fs.StringVarP(&flags.resume, "resume", "r", "", "Resume session by ID or path")
	fs.StringVar(&flags.session, "session", "", "Use specific session file or ID")
	fs.BoolVarP(&flags.print, "print", "P", false, "Print response and exit (non-interactive)")
	registerSharedExecutionFlags(fs, flags, "Enable configured web search provider for this run")
	fs.BoolVar(&flags.initServe, "init-serve", false, "Create serve.json config template")
	fs.BoolVar(&flags.force, "force", false, "Force overwrite existing files (used with --init-*)")
	fs.BoolVar(&flags.cron, "cron", false, "Enable scheduled task management (cron tool)")
	fs.BoolVar(&flags.enableA2AMaster, "enable-a2a-master", false, "Enable A2A master mode (dispatch tasks to remote agents)")
	fs.BoolVar(&flags.initA2AMaster, "init-a2a-master-config", false, "Create a2a-list.json config template")
}

func registerACPFlags(fs *pflag.FlagSet, flags *cliFlags) {
	registerSharedProviderFlags(fs, flags)
	registerSharedExecutionFlags(fs, flags, "Enable configured web search provider for this ACP run")
}

func registerSharedProviderFlags(fs *pflag.FlagSet, flags *cliFlags) {
	fs.StringVarP(&flags.provider, "provider", "p", "", "Provider (openai, anthropic, or custom provider name)")
	fs.StringVarP(&flags.model, "model", "m", "", "Model ID")
	fs.StringVarP(&flags.mode, "mode", "M", "", "Mode (plan, agent, yolo)")
	fs.StringVarP(&flags.thinking, "thinking", "t", "", "Thinking level (off, minimal, low, medium, high, xhigh)")
}

func registerSharedExecutionFlags(fs *pflag.FlagSet, flags *cliFlags, webSearchUsage string) {
	fs.BoolVar(&flags.sandbox, "sandbox", false, "Enable sandbox (bwrap) for secure execution")
	fs.BoolVar(&flags.verbose, "verbose", false, "Verbose output")
	fs.BoolVar(&flags.debug, "debug", false, "Enable debug logging")
	fs.BoolVar(&flags.multiAgent, "multi-agent", false, "Enable multi-agent mode (sub-agent tools)")
	fs.BoolVar(&flags.delegate, "delegate", false, "Enable delegation mode (blocking single sub-agent tool)")
	fs.BoolVar(&flags.workflows, "workflows", false, "Enable workflow mode (Elisp workflow tools)")
	fs.BoolVar(&flags.webSearch, "web-search", false, webSearchUsage)
	fs.BoolVar(&flags.browser, "browser", false, "Enable browser automation tool")
}

func (f *cliFlags) runOptions() runOptions {
	return runOptions{
		provider:        f.provider,
		model:           f.model,
		mode:            f.mode,
		thinking:        f.thinking,
		continue_:       f.continueSession,
		resume:          f.resume,
		session:         f.session,
		sandbox:         f.sandbox,
		print:           f.print,
		verbose:         f.verbose,
		debug:           f.debug,
		multiAgent:      f.multiAgent,
		delegate:        f.delegate,
		workflows:       f.workflows,
		cron:            f.cron,
		webSearch:       f.webSearch,
		browser:         f.browser,
		enableA2AMaster: f.enableA2AMaster,
	}
}

func (f *cliFlags) acpOptions() acp.RunOptions {
	return acp.RunOptions{
		Provider:   f.provider,
		Model:      f.model,
		Mode:       f.mode,
		Thinking:   f.thinking,
		Sandbox:    f.sandbox,
		Verbose:    f.verbose,
		Debug:      f.debug,
		MultiAgent: f.multiAgent,
		Delegate:   f.delegate,
		Workflows:  f.workflows,
		WebSearch:  f.webSearch,
		Browser:    f.browser,
	}
}

// newSystemInitCommand creates the `systeminit` subcommand which generates a
// project AGENTS.md non-interactively (CLI/print mode).
func newSystemInitCommand(runFn func([]string, runOptions) error, flagProvider, flagModel *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "systeminit [guidance...]",
		Short: "Generate a project AGENTS.md for AI agents",
		Long:  "Analyze the current project and write an AGENTS.md guide. Non-interactive; in the TUI use the /systeminit command for an interactive, question-driven setup.\n\nOptional trailing text is passed as extra guidance, e.g. `vibecoding systeminit write AGENTS.md in English`.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFn(nil, runOptions{
				provider:        *flagProvider,
				model:           *flagModel,
				print:           true,
				systemInit:      true,
				systemInitExtra: strings.Join(args, " "),
			})
		},
	}
	cmd.Flags().StringVarP(flagProvider, "provider", "p", "", "Provider (openai, anthropic, or custom provider name)")
	cmd.Flags().StringVarP(flagModel, "model", "m", "", "Model ID")
	return cmd
}

type runOptions struct {
	provider        string
	model           string
	mode            string
	thinking        string
	continue_       bool
	resume          string
	session         string
	sandbox         bool
	print           bool
	verbose         bool
	debug           bool
	multiAgent      bool
	delegate        bool
	workflows       bool
	cron            bool
	webSearch       bool
	browser         bool
	enableA2AMaster bool
	systemInit      bool
	systemInitExtra string
}

type contextFilesResult struct {
	context string
	info    string
}

type providerSelection struct {
	name          string
	modelID       string
	mode          string
	thinkingLevel string
}

type skillSetup struct {
	manager *skills.Manager
	context string
}

type sessionSetup struct {
	manager *session.Manager
	info    string
}

type runtimeSetup struct {
	agentManager  *agent.AgentManager
	cronStore     cron.CronStore
	cronScheduler *cron.Scheduler
	allow         *config.AllowConfig
	cleanup       func()
}

func run(args []string, opts runOptions) error {
	initRunEnvironment(opts)

	settings, settingsMeta, err := config.LoadSettingsWithMeta()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	applyRuntimeSettings(settings, opts)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	ruleContent := contextfiles.LoadRuleFile(cwd)
	contextFiles := loadContextFiles(cwd, settings, ruleContent)
	selection := resolveProviderSelection(settings, opts)
	args, opts, selection = applySystemInit(args, opts, selection)

	p, model, err := createProvider(settings, selection.name, selection.modelID)
	if err != nil {
		return err
	}

	skillSetup, err := loadSkills(cwd, settings, opts)
	if err != nil {
		return err
	}
	sbMgr, err := setupSandbox(cwd, settings, opts, selection.mode)
	if err != nil {
		return err
	}
	sbInfo := sandbox.FormatSandboxInfo(sbMgr.GetActive())

	sessionSetup, err := setupSession(cwd, settings, opts)
	if err != nil {
		return err
	}

	registry, mcpCleanup, err := setupToolRegistry(cwd, settings, opts, sbMgr, skillSetup.manager)
	if err != nil {
		return err
	}
	defer mcpCleanup()

	extraContext := contextFiles.context + skillSetup.context
	sessionID := ""
	if sessionSetup.manager != nil && sessionSetup.manager.GetHeader() != nil {
		sessionID = sessionSetup.manager.GetHeader().ID
	}
	runtime, err := setupAgentRuntime(p, selection.name, model, settings, opts, registry, sbMgr, extraContext, ruleContent, skillSetup.manager, sessionID, cwd)
	if err != nil {
		return err
	}
	defer runtime.cleanup()

	if opts.print {
		startUpdateCheck(settings, func(notice string) {
			fmt.Fprintln(os.Stderr, notice)
		})
		return runPrint(args, p, selection.name, model, selection.mode, provider.ThinkingLevel(selection.thinkingLevel), settings, registry, sessionSetup.manager, extraContext, ruleContent, opts.multiAgent, opts.delegate, opts.workflows, runtime.agentManager)
	}

	return runInteractive(runInteractiveConfig{
		provider:         p,
		providerKey:      selection.name,
		model:            model,
		settings:         settings,
		settingsMeta:     settingsMeta,
		session:          sessionSetup.manager,
		sessionInfo:      sessionSetup.info,
		registry:         registry,
		sandboxInfo:      sbInfo,
		extraContext:     extraContext,
		ruleContent:      ruleContent,
		contextFilesInfo: contextFiles.info,
		skillsManager:    skillSetup.manager,
		mode:             selection.mode,
		opts:             opts,
		runtime:          runtime,
	})
}

func initRunEnvironment(opts runOptions) {
	// Set Windows console to UTF-8 so CJK IME works correctly.
	if err := initConsole(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: init console: %v\n", err)
	}

	debugEnabled = opts.debug
	if debugEnabled && opts.print {
		fmt.Fprintf(os.Stderr, "[DEBUG] Debug logging enabled\n")
	}

	config.Verbose = opts.verbose || opts.debug
	if opts.debug {
		_ = os.Setenv("VIBECODING_DEBUG", "1")
		if opts.print {
			debugpprof.StartForDebug(os.Stderr)
		} else {
			_ = os.Setenv(provider.DebugLogOnlyEnv, "1")
			debugpprof.StartForDebug(io.Discard)
		}
	}
}

func applyRuntimeSettings(settings *config.Settings, opts runOptions) {
	if opts.webSearch {
		settings.WebSearch.Enabled = config.BoolPtr(true)
	}
}

func loadContextFiles(cwd string, settings *config.Settings, ruleContent string) contextFilesResult {
	cfResult := &contextfiles.LoadResult{}
	var contextStr string
	if settings.ContextFiles.Enabled {
		cfResult = contextfiles.LoadContextFiles(cwd, config.ConfigDir(), settings.ContextFiles.ExtraFiles)
		contextStr = contextfiles.BuildContextString(cfResult)
	}
	return contextFilesResult{
		context: contextStr,
		info:    formatContextFilesInfo(cfResult, cwd, ruleContent),
	}
}

func formatContextFilesInfo(result *contextfiles.LoadResult, cwd string, ruleContent string) string {
	var sb strings.Builder
	sb.WriteString("📄 Loaded context files:\n")
	for _, f := range result.GlobalFiles {
		sb.WriteString(fmt.Sprintf("  ✓ %s (global)\n", f.Name))
	}
	for _, f := range result.ParentFiles {
		sb.WriteString(fmt.Sprintf("  ✓ %s (parent: %s)\n", f.Name, filepath.Dir(f.Path)))
	}
	for _, f := range result.ProjectFiles {
		sb.WriteString(fmt.Sprintf("  ✓ %s (project)\n", f.Name))
	}
	appendRuleFileInfo(&sb, cwd, ruleContent)
	return sb.String()
}

func appendRuleFileInfo(sb *strings.Builder, cwd string, ruleContent string) {
	path := contextfiles.RuleFilePath(cwd)
	if _, err := os.Stat(path); err == nil {
		if strings.TrimSpace(ruleContent) == "" {
			sb.WriteString(fmt.Sprintf("  ⚠ %s (project rules, empty)\n", contextfiles.RuleFile))
			return
		}
		sb.WriteString(fmt.Sprintf("  ✓ %s (project rules)\n", contextfiles.RuleFile))
		return
	} else if os.IsNotExist(err) {
		sb.WriteString(fmt.Sprintf("  ! %s not found (run /rule to create default project rules)\n", contextfiles.RuleFile))
		return
	} else {
		sb.WriteString(fmt.Sprintf("  ! %s unavailable: %v\n", contextfiles.RuleFile, err))
	}
}

func resolveProviderSelection(settings *config.Settings, opts runOptions) providerSelection {
	selection := providerSelection{
		name:          opts.provider,
		modelID:       opts.model,
		mode:          opts.mode,
		thinkingLevel: opts.thinking,
	}
	if selection.name == "" {
		selection.name = settings.DefaultProvider
	}
	if selection.modelID == "" && opts.provider == "" {
		selection.modelID = settings.DefaultModel
	}
	if selection.mode == "" {
		selection.mode = settings.DefaultMode
	}
	if selection.mode == "" {
		selection.mode = "agent"
	}
	if selection.thinkingLevel == "" {
		selection.thinkingLevel = settings.DefaultThinkingLevel
	}
	return selection
}

func applySystemInit(args []string, opts runOptions, selection providerSelection) ([]string, runOptions, providerSelection) {
	// /systeminit on the CLI is non-interactive: force print mode and yolo so
	// the agent can write AGENTS.md without prompting for approval.
	if !opts.systemInit {
		return args, opts, selection
	}
	selection.mode = "yolo"
	opts.print = true
	args = []string{systeminit.Prompt(false, opts.systemInitExtra)}
	return args, opts, selection
}

func loadSkills(cwd string, settings *config.Settings, opts runOptions) (skillSetup, error) {
	if opts.workflows {
		path, created, err := workflow.EnsureProjectSkill(cwd)
		if err != nil {
			return skillSetup{}, fmt.Errorf("create workflow skill: %w", err)
		}
		if opts.verbose && created {
			fmt.Fprintf(os.Stderr, "Created workflow skill: %s\n", path)
		}
	}
	if opts.browser {
		path, created, err := browserfeature.EnsureProjectSkill(cwd)
		if err != nil {
			return skillSetup{}, fmt.Errorf("create browser skill: %w", err)
		}
		if opts.verbose && created {
			fmt.Fprintf(os.Stderr, "Created browser skill: %s\n", path)
		}
	}

	skillsMgr := skills.NewManagerWithProjectDirs(settings.GetGlobalSkillsDir(), skills.ProjectSkillDirs(cwd))
	if err := skillsMgr.Load(); err != nil && opts.verbose {
		fmt.Fprintf(os.Stderr, "Warning: load skills: %v\n", err)
	}
	skillsContext := skillsMgr.BuildAllSkillsContext()
	if opts.workflows {
		skillsContext += skillsMgr.BuildSkillContext(workflow.SkillName)
	}
	if opts.browser {
		skillsContext += skillsMgr.BuildSkillContext(browserfeature.SkillName)
	}
	if opts.verbose && skillsContext != "" {
		fmt.Fprintf(os.Stderr, "Loaded %d skills\n", len(skillsMgr.List()))
	}
	return skillSetup{manager: skillsMgr, context: skillsContext}, nil
}

func setupSandbox(cwd string, settings *config.Settings, opts runOptions, mode string) (*sandbox.Manager, error) {
	sbMgr := sandbox.NewManager(cwd)
	if !(opts.sandbox || settings.Sandbox.Enabled) {
		sbMgr.SetLevel(sandbox.LevelNone)
		return sbMgr, nil
	}

	targetLevel := sandboxLevelForMode(mode)
	// When the user explicitly passed --sandbox, verify the requested level
	// is actually available before allowing silent fallback to none.
	if opts.sandbox && targetLevel != sandbox.LevelNone {
		if _, err := sbMgr.GetForLevel(targetLevel); err != nil {
			return nil, fmt.Errorf("sandbox requested but unavailable: %w", err)
		}
	}
	if err := sbMgr.SetLevel(targetLevel); err != nil {
		if opts.sandbox {
			return nil, fmt.Errorf("sandbox requested but unavailable: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Warning: sandbox unavailable, continuing without: %v\n", err)
		sbMgr.SetLevel(sandbox.LevelNone)
	}
	return sbMgr, nil
}

func sandboxLevelForMode(mode string) sandbox.Level {
	switch mode {
	case "plan":
		return sandbox.LevelStrict
	case "yolo":
		return sandbox.LevelNone
	default:
		return sandbox.LevelStandard
	}
}

func setupSession(cwd string, settings *config.Settings, opts runOptions) (sessionSetup, error) {
	sessionDir := settings.GetSessionDir()
	switch {
	case opts.continue_:
		if opts.print {
			sess, err := session.ContinueRecent(cwd, sessionDir)
			if err != nil {
				return sessionSetup{}, fmt.Errorf("continue session: %w", err)
			}
			return sessionSetup{manager: sess, info: continuingSessionInfo(sess)}, nil
		}
		sessions, err := session.ListForDir(cwd, sessionDir)
		if err != nil {
			return sessionSetup{}, fmt.Errorf("continue session: %w", err)
		}
		if len(sessions) == 0 {
			return sessionSetup{}, nil
		}
		sess, err := session.ContinueRecent(cwd, sessionDir)
		if err != nil {
			return sessionSetup{}, fmt.Errorf("continue session: %w", err)
		}
		return sessionSetup{manager: sess, info: continuingSessionInfo(sess)}, nil
	case opts.session != "":
		sess, err := session.OpenByPathOrID(cwd, sessionDir, opts.session)
		if err != nil {
			return sessionSetup{}, fmt.Errorf("open session: %w", err)
		}
		return sessionSetup{manager: sess, info: fmt.Sprintf("📂 Opened session: %s", sess.GetHeader().ID)}, nil
	case opts.resume != "":
		sess, err := session.OpenByPathOrID(cwd, sessionDir, opts.resume)
		if err != nil {
			return sessionSetup{}, fmt.Errorf("resume session: %w", err)
		}
		return sessionSetup{manager: sess, info: fmt.Sprintf("📂 Resumed session: %s", sess.GetHeader().ID)}, nil
	default:
		if !opts.print && !opts.cron {
			return sessionSetup{}, nil
		}
		sess := session.New(cwd, sessionDir)
		if err := sess.Init(); err != nil {
			return sessionSetup{}, fmt.Errorf("init session: %w", err)
		}
		return sessionSetup{manager: sess}, nil
	}
}

func continuingSessionInfo(sess *session.Manager) string {
	if sess.GetHeader() == nil {
		return ""
	}
	info := fmt.Sprintf("📂 Continuing session: %s", sess.GetHeader().ID)
	if messages := sess.GetMessages(); len(messages) > 0 {
		info += fmt.Sprintf(" (%d messages)", len(messages))
	}
	return info
}

func setupToolRegistry(cwd string, settings *config.Settings, opts runOptions, sbMgr *sandbox.Manager, skillsMgr *skills.Manager) (*tools.Registry, func(), error) {
	registry := tools.NewRegistry(cwd, sbMgr.GetActive())
	registry.RegisterDefaultsWithPlanTool(settings.IsPlanToolEnabled())

	// Register the interactive question tool for TUI sessions (plan/agent modes).
	// Print mode is non-interactive, so it must not expose a tool that blocks
	// waiting for a user answer.
	if !opts.print {
		registry.Register(tools.NewQuestionTool(registry))
	}
	if skillsMgr != nil {
		registry.Register(tools.NewSkillRefTool(skillsMgr))
	}
	if opts.browser {
		browserfeature.RegisterTool(registry)
	}

	mcpServers, err := mcp.LoadConfiguredServers(cwd)
	if err != nil {
		return nil, nil, err
	}
	mcpClients, err := mcp.ConnectServers(context.Background(), mcpServers, registry, mcp.Callbacks{})
	if err != nil {
		return nil, nil, fmt.Errorf("connect MCP servers: %w", err)
	}
	cleanup := func() { mcp.CloseClients(mcpClients) }
	if err := registerA2AMasterTool(registry, opts); err != nil {
		cleanup()
		return nil, nil, err
	}
	return registry, cleanup, nil
}

func registerA2AMasterTool(registry *tools.Registry, opts runOptions) error {
	if !opts.enableA2AMaster {
		return nil
	}

	a2aListPath := a2a.ProjectAgentListConfigPath()
	if _, err := os.Stat(a2aListPath); err != nil {
		a2aListPath = a2a.AgentListConfigPath()
	}
	a2aListCfg, err := a2a.LoadAgentList(a2aListPath)
	if err != nil {
		return fmt.Errorf("load a2a-list.json: %w", err)
	}
	a2aMgr := a2a.NewA2AManager(a2aListCfg)
	registry.Register(tools.NewA2ADispatchTool(&a2aDispatcherAdapter{mgr: a2aMgr}))
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "A2A master mode enabled: %d agents loaded from %s\n", len(a2aMgr.List()), a2aListPath)
	}
	return nil
}

func setupAgentRuntime(p provider.Provider, providerName string, model *provider.Model, settings *config.Settings, opts runOptions, registry *tools.Registry, sbMgr *sandbox.Manager, extraContext string, ruleContent string, skillsMgr *skills.Manager, sessionID string, workDir string) (runtimeSetup, error) {
	allow := config.LoadAllow()
	factory := agent.NewAgentFactoryWithOptions(p, model, settings, sbMgr, extraContext, ruleContent, skillsMgr, compactionSettingsFromConfig(settings), nil, agent.AgentFactoryOptions{
		MultiAgentEnabled: true,
		DelegateEnabled:   opts.delegate,
		WorkflowsEnabled:  opts.workflows,
		ProviderName:      providerName,
		Allow:             allow,
	})
	agentMgr := agent.NewAgentManager(factory)
	runtime := runtimeSetup{
		agentManager: agentMgr,
		allow:        allow,
		cleanup:      func() {},
	}

	if opts.multiAgent {
		agent.RegisterSubAgentTools(registry, agentMgr)
	}
	if opts.cron {
		globalStore := cron.NewSQLiteCronStore(settings.GetSessionDir())
		runtime.cronStore = cron.NewSessionScopedStoreWithWorkDir(globalStore, sessionID, workDir)
		runtime.cronScheduler = cron.NewSchedulerWithSessionDir(globalStore, agentMgr, 30*time.Second, settings.GetSessionDir())
		runtime.cronScheduler.Start()
		runtime.cleanup = runtime.cronScheduler.Stop
		registry.Register(cron.NewCronTool(runtime.cronStore, runtime.cronScheduler))
	}
	if opts.delegate {
		agent.RegisterDelegateSubAgentTool(registry, agentMgr)
	}
	if opts.workflows {
		workflow.RegisterTools(registry, agentMgr, nil)
	}
	logRuntimeModes(opts)
	return runtime, nil
}

func compactionSettingsFromConfig(settings *config.Settings) ctxpkg.CompactionSettings {
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
	return compactionSettings
}

func logRuntimeModes(opts runOptions) {
	if !opts.verbose {
		return
	}
	if opts.multiAgent {
		fmt.Fprintf(os.Stderr, "Multi-agent mode enabled\n")
	}
	if opts.delegate {
		fmt.Fprintf(os.Stderr, "Delegate mode enabled\n")
	}
	if opts.workflows {
		fmt.Fprintf(os.Stderr, "Workflow mode enabled\n")
	}
	if opts.cron {
		fmt.Fprintf(os.Stderr, "Cron mode enabled\n")
	}
}

type runInteractiveConfig struct {
	provider         provider.Provider
	providerKey      string // user-configured settings.json key (e.g. "xiaomi")
	model            *provider.Model
	settings         *config.Settings
	settingsMeta     config.LoadMeta
	session          *session.Manager
	sessionInfo      string
	registry         *tools.Registry
	sandboxInfo      string
	extraContext     string
	ruleContent      string
	contextFilesInfo string
	skillsManager    *skills.Manager
	mode             string
	opts             runOptions
	runtime          runtimeSetup
}

func runInteractive(cfg runInteractiveConfig) error {
	// Clear any pending stdin input (e.g., terminal color queries).
	clearStdin()

	app := tui.NewAppWithWorkflowsAndAllow(
		cfg.provider,
		cfg.model,
		cfg.settings,
		cfg.session,
		cfg.registry,
		cfg.sandboxInfo,
		cfg.extraContext,
		cfg.ruleContent,
		cfg.skillsManager,
		cfg.mode,
		cfg.opts.multiAgent,
		cfg.opts.delegate,
		cfg.opts.workflows,
		cfg.runtime.agentManager,
		cfg.runtime.cronStore,
		cfg.runtime.cronScheduler,
		cfg.providerKey,
		cfg.runtime.allow,
	)
	if initialMsg := buildInitialMessage(cfg); initialMsg != "" {
		app.SetInitialMessage(initialMsg)
	}
	if cfg.settingsMeta.CreatedGlobalConfig {
		app.SetAutoOpenAuthDialog(true)
	}
	app.SetBrowserEnabled(cfg.opts.browser, cfg.opts.browser)
	p2 := tea.NewProgram(app, teaProgramOptions()...)
	app.SetProgram(p2)
	startUpdateCheck(cfg.settings, app.ShowUpdateNotice)
	if _, err := p2.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	if app.ReloadRequested() {
		return reexecFresh()
	}
	return nil
}

func buildInitialMessage(cfg runInteractiveConfig) string {
	parts := []string{}
	if cfg.contextFilesInfo != "" {
		parts = append(parts, cfg.contextFilesInfo)
	}
	if cfg.sessionInfo != "" {
		parts = append(parts, cfg.sessionInfo)
	}
	if cfg.settingsMeta.CreatedGlobalConfig {
		parts = append(parts, fmt.Sprintf("Created default config: %s\nOpening /auth to configure or confirm your provider token and model.", cfg.settingsMeta.GlobalSettingsPath))
	}
	return strings.Join(parts, "\n")
}

// reexecFresh restarts the program as a fresh process with a brand-new session
// (session-continuation flags are stripped). Used by the /reload command.
func reexecFresh() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("reload: locate executable: %w", err)
	}
	args := filterReloadArgs(os.Args[1:])
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("reload: %w", err)
	}
	os.Exit(0)
	return nil
}

// filterReloadArgs removes session-continuation flags so a reload starts fresh.
func filterReloadArgs(args []string) []string {
	out := make([]string, 0, len(args))
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		switch arg {
		case "-c", "--continue":
			continue
		case "-r", "--resume", "--session":
			skipNext = true
			continue
		}
		if strings.HasPrefix(arg, "--resume=") || strings.HasPrefix(arg, "--session=") {
			continue
		}
		out = append(out, arg)
	}
	return out
}

func startUpdateCheck(settings *config.Settings, notify func(string)) {
	if !settings.IsUpdateCheckEnabled() {
		return
	}
	update.CheckInBackground(version, notify)
}

// a2aDispatcherAdapter adapts a2a.A2AManager to tools.A2ADispatcher.
type a2aDispatcherAdapter struct {
	mgr *a2a.A2AManager
}

func (a *a2aDispatcherAdapter) List() []tools.AgentEntry {
	entries := a.mgr.List()
	result := make([]tools.AgentEntry, len(entries))
	for i, e := range entries {
		result[i] = tools.AgentEntry{Name: e.Name, URL: e.URL}
	}
	return result
}

func (a *a2aDispatcherAdapter) Dispatch(ctx context.Context, name, message string) (string, error) {
	return a.mgr.Dispatch(ctx, name, message)
}
