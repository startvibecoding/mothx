package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/startvibecoding/mothx/internal/serve"
)

func newServeCommand(flags *cliFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the unified OpenAI API, Web UI, and messaging channels",
		Long:  "Start MothX as a unified server exposing OpenAI-compatible APIs, a Web UI, and optional Feishu/WeChat channels.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve.Run(flags.serveOptions(), version)
		},
	}
	registerServeFlags(cmd.Flags(), flags)
	cmd.AddCommand(newServeInitConfigCommand())
	return cmd
}

func newServeInitConfigCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init-config [global|project]",
		Short: "Create serve.json config template",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := "global"
			if len(args) > 0 {
				scope = args[0]
			}
			var project bool
			switch scope {
			case "global":
				project = false
			case "project":
				project = true
			default:
				return fmt.Errorf("invalid scope %q: expected global or project", scope)
			}
			path, err := serve.InitConfigForProject(project, force)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Created serve config: %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing file")
	return cmd
}

func registerServeFlags(fs *pflag.FlagSet, flags *cliFlags) {
	fs.StringVar(&flags.servePort, "port", "", "Listen port (default: from serve.json or 8080)")
	fs.StringVar(&flags.serveConfig, "config", "", "Path to serve.json")
	fs.StringVar(&flags.serveWebUIDir, "web-ui-dir", "", "Directory containing built Serve Web UI assets")
	fs.StringVar(&flags.gatewayWorkDir, "work-dir", "", "Default working directory")
	fs.StringVarP(&flags.provider, "provider", "p", "", "Provider (openai, anthropic, or custom provider name)")
	fs.StringVarP(&flags.model, "model", "m", "", "Model ID")
	fs.BoolVar(&flags.sandbox, "sandbox", false, "Enable sandbox (bwrap) for secure execution")
	fs.BoolVar(&flags.multiAgent, "multi-agent", false, "Enable multi-agent mode (sub-agent tools)")
	fs.BoolVar(&flags.delegate, "delegate", false, "Enable delegation mode (blocking single sub-agent tool)")
	fs.BoolVar(&flags.workflows, "workflows", false, "Enable workflow mode (Elisp workflow tools)")
	fs.BoolVar(&flags.lobsterMode, "lobster", false, "Enable lobster mode (yolo, no sandbox, sub-agents on)")
	fs.BoolVar(&flags.verbose, "verbose", false, "Verbose output")
	fs.BoolVar(&flags.debug, "debug", false, "Enable debug logging")
}

func (f *cliFlags) serveOptions() serve.RunOptions {
	return serve.RunOptions{
		ConfigPath: f.serveConfig,
		Port:       f.servePort,
		WebUIDir:   f.serveWebUIDir,
		Provider:   f.provider,
		Model:      f.model,
		WorkDir:    f.gatewayWorkDir,
		Sandbox:    f.sandbox,
		MultiAgent: f.multiAgent,
		Delegate:   f.delegate,
		Workflows:  f.workflows,
		Lobster:    f.lobsterMode,
		Verbose:    f.verbose,
		Debug:      f.debug,
	}
}
