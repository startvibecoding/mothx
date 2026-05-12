package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/fuckvibecoding/vibecoding/internal/agent"
	"github.com/fuckvibecoding/vibecoding/internal/config"
	"github.com/fuckvibecoding/vibecoding/internal/contextfiles"
	"github.com/fuckvibecoding/vibecoding/internal/provider"
	"github.com/fuckvibecoding/vibecoding/internal/provider/anthropic"
	"github.com/fuckvibecoding/vibecoding/internal/provider/openai"
	"github.com/fuckvibecoding/vibecoding/internal/sandbox"
	"github.com/fuckvibecoding/vibecoding/internal/session"
	"github.com/fuckvibecoding/vibecoding/internal/skills"
	"github.com/fuckvibecoding/vibecoding/internal/tools"
	"github.com/fuckvibecoding/vibecoding/internal/tui"
)

var version = "dev"

func main() {
	var (
		flagProvider  string
		flagModel     string
		flagMode      string
		flagThinking  string
		flagContinue  bool
		flagResume    string
		flagSession   string
		flagSandbox    bool
		flagPrint     bool
		flagVerbose   bool
	)

	rootCmd := &cobra.Command{
		Use:     "vibecoding [message...]",
		Aliases: []string{"vc"},
		Short:   "VibeCoding - AI coding assistant",
		Long:    "VibeCoding is an AI-powered coding assistant that runs in your terminal.\nSupports OpenAI and Anthropic APIs with sandboxed execution.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, runOptions{
				provider:  flagProvider,
				model:     flagModel,
				mode:      flagMode,
				thinking:  flagThinking,
				continue_: flagContinue,
				resume:    flagResume,
				session:   flagSession,
				sandbox:   flagSandbox,
				print:     flagPrint,
				verbose:   flagVerbose,
			})
		},
	}

	flags := rootCmd.Flags()
	flags.StringVarP(&flagProvider, "provider", "p", "", "Provider (openai, anthropic, or custom provider name)")
	flags.StringVarP(&flagModel, "model", "m", "", "Model ID")
	flags.StringVarP(&flagMode, "mode", "M", "", "Mode (plan, agent, yolo)")
	flags.StringVarP(&flagThinking, "thinking", "t", "", "Thinking level (off, minimal, low, medium, high, xhigh)")
	flags.BoolVarP(&flagContinue, "continue", "c", false, "Continue most recent session")
	flags.StringVarP(&flagResume, "resume", "r", "", "Resume session by ID or path")
	flags.StringVar(&flagSession, "session", "", "Use specific session file or ID")
	flags.BoolVar(&flagSandbox, "sandbox", false, "Enable sandbox (bwrap) for secure execution")
	flags.BoolVarP(&flagPrint, "print", "P", false, "Print response and exit (non-interactive)")
	flags.BoolVar(&flagVerbose, "verbose", false, "Verbose output")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type runOptions struct {
	provider  string
	model     string
	mode      string
	thinking  string
	continue_ bool
	resume    string
	session   string
	sandbox   bool
	print     bool
	verbose   bool
}

func run(args []string, opts runOptions) error {
	// Load settings
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Determine provider
	providerName := opts.provider
	if providerName == "" {
		providerName = settings.DefaultProvider
	}

	// Determine model
	modelID := opts.model
	if modelID == "" {
		modelID = settings.DefaultModel
	}

	// Create provider from config
	p, model, err := createProvider(settings, providerName, modelID)
	if err != nil {
		return err
	}

	// Determine mode
	mode := opts.mode
	if mode == "" {
		mode = settings.DefaultMode
	}
	if mode == "" {
		mode = "agent"
	}

	// Determine thinking level
	thinkingLevel := opts.thinking
	if thinkingLevel == "" {
		thinkingLevel = settings.DefaultThinkingLevel
	}

	// Load context files
	var contextStr string
	if settings.ContextFiles.Enabled {
		cfResult := contextfiles.LoadContextFiles(cwd, config.ConfigDir(), settings.ContextFiles.ExtraFiles)
		contextStr = contextfiles.BuildContextString(cfResult)
		if opts.verbose && contextStr != "" {
			fmt.Fprintf(os.Stderr, "Loaded context files: %d global, %d parent, %d project\n",
				len(cfResult.GlobalFiles), len(cfResult.ParentFiles), len(cfResult.ProjectFiles))
		}
	}

	// Load skills
	skillsMgr := skills.NewManager(settings.GetGlobalSkillsDir(), cwd+"/.skills")
	if err := skillsMgr.Load(); err != nil && opts.verbose {
		fmt.Fprintf(os.Stderr, "Warning: load skills: %v\n", err)
	}
	skillsContext := skillsMgr.BuildAllSkillsContext()
	if opts.verbose && skillsContext != "" {
		fmt.Fprintf(os.Stderr, "Loaded %d skills\n", len(skillsMgr.List()))
	}

	// Setup sandbox
	sbMgr := sandbox.NewManager(cwd)

	// Sandbox is disabled by default, enabled via --sandbox flag or config
	sbEnabled := opts.sandbox || settings.Sandbox.Enabled

	if !sbEnabled {
		sbMgr.SetLevel(sandbox.LevelNone)
	} else {
		switch mode {
		case "plan":
			sbMgr.SetLevel(sandbox.LevelStrict)
		case "yolo":
			sbMgr.SetLevel(sandbox.LevelNone)
		default:
			sbMgr.SetLevel(sandbox.LevelStandard)
		}
	}
	sbInfo := sandbox.FormatSandboxInfo(sbMgr.GetActive())

	// Setup session
	var sess *session.Manager
	if opts.continue_ {
		sess, err = session.ContinueRecent(cwd, settings.GetSessionDir())
		if err != nil {
			return fmt.Errorf("continue session: %w", err)
		}
	} else if opts.session != "" {
		sess, err = session.Open(opts.session)
		if err != nil {
			return fmt.Errorf("open session: %w", err)
		}
	} else {
		sess = session.New(cwd, settings.GetSessionDir())
		if err := sess.Init(); err != nil {
			return fmt.Errorf("init session: %w", err)
		}
	}

	// Setup tools
	registry := tools.NewRegistry(cwd, sbMgr.GetActive())
	registry.RegisterDefaults()

	// Build extra system context
	extraContext := contextStr + skillsContext

	// Print mode: non-interactive
	if opts.print {
		return runPrint(args, p, model, mode, provider.ThinkingLevel(thinkingLevel), settings, registry, sess, extraContext)
	}

	// Interactive mode
	app := tui.NewApp(p, model, settings, sess, registry, sbInfo, extraContext, skillsMgr)
	p2 := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p2.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}

	return nil
}

// createProvider creates a provider from config based on provider name.
func createProvider(settings *config.Settings, providerName, modelID string) (provider.Provider, *provider.Model, error) {
	// Check if provider is in config
	pc := settings.GetProviderConfig(providerName)

	if pc != nil {
		// Custom provider from config
		apiKey := settings.ResolveKey(providerName)
		models := convertModelConfigs(providerName, pc.Models)

		api := pc.API
		if api == "" {
			// Auto-detect: if baseUrl contains "anthropic", use anthropic-messages
			if strings.Contains(strings.ToLower(pc.BaseURL), "anthropic") {
				api = "anthropic-messages"
			} else {
				api = "openai-chat"
			}
		}

		var p provider.Provider
		switch api {
		case "anthropic-messages":
			p = anthropic.NewProviderWithModels(apiKey, pc.BaseURL, models)
		case "openai-chat", "openai":
			p = openai.NewProviderWithModels(apiKey, pc.BaseURL, models)
		default:
			return nil, nil, fmt.Errorf("unsupported API type: %s (use 'openai-chat' or 'anthropic-messages')", api)
		}

		// Find model
		model := p.GetModel(modelID)
		if model == nil {
			if len(models) > 0 {
				model = models[0]
			} else {
				return nil, nil, fmt.Errorf("no models configured for provider %s", providerName)
			}
		}

		return p, model, nil
	}

	// Built-in providers (fallback)
	var p provider.Provider
	switch strings.ToLower(providerName) {
	case "openai":
		apiKey := settings.ResolveKey(providerName)
		p = openai.NewProvider(apiKey, "")
	case "anthropic":
		apiKey := settings.ResolveKey(providerName)
		p = anthropic.NewProvider(apiKey, "")
	default:
		return nil, nil, fmt.Errorf("unknown provider: %s (add it to settings.json providers section)", providerName)
	}

	model := p.GetModel(modelID)
	if model == nil {
		models := p.Models()
		if len(models) > 0 {
			model = models[0]
		} else {
			return nil, nil, fmt.Errorf("no models available for provider %s", providerName)
		}
	}

	return p, model, nil
}

// convertModelConfigs converts config.ModelConfig to provider.Model.
func convertModelConfigs(providerName string, models []config.ModelConfig) []*provider.Model {
	var result []*provider.Model
	for _, m := range models {
		input := m.Input
		if len(input) == 0 {
			input = []string{"text"}
		}
		var cost provider.ModelPricing
		if m.Cost != nil {
			cost = provider.ModelPricing{
				Input:      m.Cost.Input,
				Output:     m.Cost.Output,
				CacheRead:  m.Cost.CacheRead,
				CacheWrite: m.Cost.CacheWrite,
			}
		}
		result = append(result, &provider.Model{
			ID:            m.ID,
			Name:          m.Name,
			Provider:      providerName,
			Reasoning:     m.Reasoning,
			Input:         input,
			Cost:          cost,
			ContextWindow: m.ContextWindow,
			MaxTokens:     m.MaxTokens,
		})
	}
	return result
}

func runPrint(args []string, p provider.Provider, model *provider.Model, mode string, thinkingLevel provider.ThinkingLevel, settings *config.Settings, registry *tools.Registry, sess *session.Manager, extraContext string) error {
	input := strings.Join(args, " ")
	if input == "" {
		data, err := os.ReadFile("/dev/stdin")
		if err != nil {
			return fmt.Errorf("no input provided")
		}
		input = string(data)
	}

	fmt.Fprintf(os.Stderr, "Using %s/%s in %s mode\n", p.Name(), model.ID, mode)

	agentCfg := agent.Config{
		Provider:      p,
		Model:         model,
		Mode:          mode,
		ThinkingLevel: thinkingLevel,
		MaxTokens:     settings.MaxOutputTokens,
		Settings:      settings,
		Session:       sess,
		ExtraContext:  extraContext,
	}

	a := agent.New(agentCfg, registry)

	ctx := context.Background()
	eventCh := a.Run(ctx, input)

	for event := range eventCh {
		switch event.Type {
		case agent.EventTextDelta:
			fmt.Print(event.TextDelta)
		case agent.EventToolCall:
			fmt.Fprintf(os.Stderr, "\n[tool: %s]\n", event.ToolCall.Name)
		case agent.EventToolStart:
			fmt.Fprintf(os.Stderr, "[running: %s] ", event.ToolName)
		case agent.EventToolResult:
			fmt.Fprintf(os.Stderr, "done\n")
		case agent.EventDone:
			fmt.Println()
		case agent.EventError:
			if event.Error != nil {
				return event.Error
			}
		case agent.EventUsage:
			if event.Usage != nil {
				fmt.Fprintf(os.Stderr, "\nTokens: %d in / %d out | Cost: $%.4f\n",
					event.Usage.Input, event.Usage.Output, event.Usage.Cost.Total)
			}
		}
	}

	return nil
}
