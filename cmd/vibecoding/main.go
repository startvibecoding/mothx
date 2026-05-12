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
	"github.com/fuckvibecoding/vibecoding/internal/provider"
	"github.com/fuckvibecoding/vibecoding/internal/provider/anthropic"
	"github.com/fuckvibecoding/vibecoding/internal/provider/openai"
	"github.com/fuckvibecoding/vibecoding/internal/sandbox"
	"github.com/fuckvibecoding/vibecoding/internal/session"
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
		flagNoSandbox bool
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
				noSandbox: flagNoSandbox,
				print:     flagPrint,
				verbose:   flagVerbose,
			})
		},
	}

	flags := rootCmd.Flags()
	flags.StringVarP(&flagProvider, "provider", "p", "", "Provider (openai, anthropic)")
	flags.StringVarP(&flagModel, "model", "m", "", "Model ID")
	flags.StringVarP(&flagMode, "mode", "M", "", "Mode (plan, agent, yolo)")
	flags.StringVarP(&flagThinking, "thinking", "t", "", "Thinking level (off, minimal, low, medium, high, xhigh)")
	flags.BoolVarP(&flagContinue, "continue", "c", false, "Continue most recent session")
	flags.StringVarP(&flagResume, "resume", "r", "", "Resume session by ID or path")
	flags.StringVar(&flagSession, "session", "", "Use specific session file or ID")
	flags.BoolVar(&flagNoSandbox, "no-sandbox", false, "Disable sandbox")
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
	noSandbox bool
	print     bool
	verbose   bool
}

func run(args []string, opts runOptions) error {
	// Load settings
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	// Determine provider
	providerName := opts.provider
	if providerName == "" {
		providerName = settings.DefaultProvider
	}

	// Resolve API key
	apiKey := config.ResolveKey(providerName)

	// Create provider
	var p provider.Provider
	switch strings.ToLower(providerName) {
	case "openai":
		p = openai.NewProvider(apiKey, "")
	case "anthropic":
		p = anthropic.NewProvider(apiKey, "")
	default:
		return fmt.Errorf("unknown provider: %s (supported: openai, anthropic)", providerName)
	}

	// Determine model
	modelID := opts.model
	if modelID == "" {
		modelID = settings.DefaultModel
	}

	model := p.GetModel(modelID)
	if model == nil {
		// Use first available model
		models := p.Models()
		if len(models) == 0 {
			return fmt.Errorf("no models available for provider %s", providerName)
		}
		model = models[0]
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

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Setup sandbox
	sbMgr := sandbox.NewManager(cwd)

	if opts.noSandbox {
		sbMgr.SetLevel(sandbox.LevelNone)
	} else {
		switch mode {
		case "plan":
			sbMgr.SetLevel(sandbox.LevelStrict)
		case "agent":
			sbMgr.SetLevel(sandbox.LevelStandard)
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

	// Print mode: non-interactive
	if opts.print {
		return runPrint(args, p, model, mode, provider.ThinkingLevel(thinkingLevel), settings, registry, sess)
	}

	// Interactive mode
	app := tui.NewApp(p, model, settings, sess, registry, sbInfo)
	p2 := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p2.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}

	return nil
}

func runPrint(args []string, p provider.Provider, model *provider.Model, mode string, thinkingLevel provider.ThinkingLevel, settings *config.Settings, registry *tools.Registry, sess *session.Manager) error {
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
	}

	a := agent.New(agentCfg, registry)

	ctx := context.Background()
	eventCh := a.Run(ctx, input)

	for event := range eventCh {
		switch event.Type {
		case agent.EventTextDelta:
			fmt.Print(event.TextDelta)
		case agent.EventThinkDelta:
			// Silently skip thinking in print mode
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
