package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/startvibecoding/mothx/internal/acp"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/contextfiles"
)

func TestRootPrintAcceptsMessageArgument(t *testing.T) {
	var gotArgs []string
	var gotOpts runOptions

	cmd := newRootCommand(
		func(args []string, opts runOptions) error {
			gotArgs = args
			gotOpts = opts
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	cmd.SetArgs([]string{"-P", "review"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !gotOpts.print {
		t.Fatal("expected print mode to be enabled")
	}
	if want := []string{"review"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestBuildInitialMessageForCreatedGlobalConfig(t *testing.T) {
	msg := buildInitialMessage(runInteractiveConfig{
		settings: config.DefaultSettings(),
		settingsMeta: config.LoadMeta{
			CreatedGlobalConfig: true,
			GlobalSettingsPath:  "/tmp/vibecoding/settings.json",
		},
	})

	if !strings.Contains(msg, "Created default config: /tmp/vibecoding/settings.json") {
		t.Fatalf("initial message = %q, want created config path", msg)
	}
	if !strings.Contains(msg, "Opening /auth") {
		t.Fatalf("initial message = %q, want /auth prompt", msg)
	}
}

func TestFormatContextFilesInfoIncludesLoadedRule(t *testing.T) {
	tmpDir := t.TempDir()
	rulePath := filepath.Join(tmpDir, contextfiles.RuleFile)
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	ruleContent := "project safety rules"
	if err := os.WriteFile(rulePath, []byte(ruleContent), 0644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	info := formatContextFilesInfo(&contextfiles.LoadResult{
		ProjectFiles: []contextfiles.FileContent{{Name: "AGENTS.md", Path: filepath.Join(tmpDir, "AGENTS.md"), Content: "# Agent"}},
	}, tmpDir, ruleContent)

	if !strings.Contains(info, "✓ AGENTS.md (project)") {
		t.Fatalf("info = %q, want project context file", info)
	}
	if !strings.Contains(info, "✓ "+contextfiles.RuleFile+" (project rules)") {
		t.Fatalf("info = %q, want loaded rule", info)
	}
}

func TestFormatContextFilesInfoPromptsRuleWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()

	info := formatContextFilesInfo(&contextfiles.LoadResult{}, tmpDir, "")

	if !strings.Contains(info, contextfiles.RuleFile+" not found") {
		t.Fatalf("info = %q, want missing rule", info)
	}
	if !strings.Contains(info, "run /rule") {
		t.Fatalf("info = %q, want /rule prompt", info)
	}
}

func TestRootParsesSessionFlags(t *testing.T) {
	var got runOptions

	cmd := newRootCommand(
		func(args []string, opts runOptions) error {
			got = opts
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	cmd.SetArgs([]string{
		"--provider", "openai",
		"--model", "gpt-test",
		"--mode", "plan",
		"--thinking", "high",
		"--continue",
		"--resume", "abc123",
		"--session", "def456",
		"--sandbox",
		"--web-search",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if got.provider != "openai" {
		t.Fatalf("provider = %q, want openai", got.provider)
	}
	if got.model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", got.model)
	}
	if got.mode != "plan" {
		t.Fatalf("mode = %q, want plan", got.mode)
	}
	if got.thinking != "high" {
		t.Fatalf("thinking = %q, want high", got.thinking)
	}
	if !got.continue_ {
		t.Fatal("expected continue flag")
	}
	if got.resume != "abc123" {
		t.Fatalf("resume = %q, want abc123", got.resume)
	}
	if got.session != "def456" {
		t.Fatalf("session = %q, want def456", got.session)
	}
	if !got.sandbox {
		t.Fatal("expected sandbox flag")
	}
	if !got.webSearch {
		t.Fatal("expected web-search flag")
	}
}

func TestRootParsesWorkflowFlagIndependently(t *testing.T) {
	var got runOptions

	cmd := newRootCommand(
		func(args []string, opts runOptions) error {
			got = opts
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	cmd.SetArgs([]string{"--workflows", "-P", "plan workflow"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !got.workflows {
		t.Fatal("expected workflows flag")
	}
	if got.multiAgent {
		t.Fatal("did not expect workflows to enable multi-agent")
	}
}

func TestRootMultiAgentDoesNotEnableWorkflows(t *testing.T) {
	var got runOptions

	cmd := newRootCommand(
		func(args []string, opts runOptions) error {
			got = opts
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	cmd.SetArgs([]string{"--multi-agent", "-P", "delegate"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !got.multiAgent {
		t.Fatal("expected multi-agent flag")
	}
	if got.workflows {
		t.Fatal("did not expect multi-agent to enable workflows")
	}
}

func TestACPParsesSharedFlagsWithoutRootFlags(t *testing.T) {
	var got acp.RunOptions

	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(opts acp.RunOptions) error {
			got = opts
			return nil
		},
	)
	cmd.SetArgs([]string{"acp", "-p", "anthropic", "-m", "claude-test", "-M", "yolo", "-t", "medium", "--sandbox", "--verbose", "--debug", "--workflows"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if got.Provider != "anthropic" {
		t.Fatalf("Provider = %q, want anthropic", got.Provider)
	}
	if got.Model != "claude-test" {
		t.Fatalf("Model = %q, want claude-test", got.Model)
	}
	if got.Mode != "yolo" {
		t.Fatalf("Mode = %q, want yolo", got.Mode)
	}
	if got.Thinking != "medium" {
		t.Fatalf("Thinking = %q, want medium", got.Thinking)
	}
	if !got.Sandbox || !got.Verbose || !got.Debug {
		t.Fatalf("flags = sandbox:%v verbose:%v debug:%v, want all true", got.Sandbox, got.Verbose, got.Debug)
	}
	if !got.Workflows {
		t.Fatal("expected workflows flag")
	}
	if got.MultiAgent {
		t.Fatal("did not expect workflows to enable multi-agent")
	}
}

func TestRootStillDispatchesACPSubcommand(t *testing.T) {
	var calledACP bool

	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(opts acp.RunOptions) error {
			calledACP = true
			if opts.Model != "test-model" {
				t.Fatalf("model = %q, want test-model", opts.Model)
			}
			return nil
		},
	)
	cmd.SetArgs([]string{"acp", "-m", "test-model"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if !calledACP {
		t.Fatal("expected ACP command execution")
	}
}

func TestUnknownRootFlagSuggestsSimilarFlag(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--modle", "hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	got := err.Error()
	for _, want := range []string{
		"invalid argument: unknown flag: --modle",
		"Did you mean --model?",
		"Run 'mothx --help' to see all commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q:\n%s", want, got)
		}
	}
}

func TestUnknownSubcommandFlagSuggestsSimilarFlag(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"gateway", "--wur-dir", "."})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	got := err.Error()
	for _, want := range []string{
		"invalid argument: unknown flag: --wur-dir",
		"Did you mean --work-dir?",
		"Run 'mothx --help' to see all commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q:\n%s", want, got)
		}
	}
}

func TestUnknownFlagWithoutSimilarFlagShowsHelpHint(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--definitely-not-a-real-option"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	got := err.Error()
	for _, want := range []string{
		"invalid argument: unknown flag: --definitely-not-a-real-option",
		"Run 'mothx --help' to see all commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q:\n%s", want, got)
		}
	}
}

func TestMistypedRootSubcommandSuggestsCommand(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"gatway"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected mistyped command error")
	}
	got := err.Error()
	for _, want := range []string{
		`invalid argument: unknown command "gatway" for "mothx"`,
		"Did you mean gateway?",
		"Run 'mothx --help' to see all commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q:\n%s", want, got)
		}
	}
}

func TestRootPrintMessageDoesNotRequireCommandSuggestion(t *testing.T) {
	var gotArgs []string

	cmd := newRootCommand(
		func(args []string, opts runOptions) error {
			gotArgs = args
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	cmd.SetArgs([]string{"-P", "explain", "this", "code"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if want := []string{"explain", "this", "code"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestRootArgsWithoutPrintAreUnknownCommand(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"explain"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unknown command error")
	}
	got := err.Error()
	for _, want := range []string{
		`invalid argument: unknown command "explain" for "mothx"`,
		"Run 'mothx --help' to see all commands.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q:\n%s", want, got)
		}
	}
}

func TestInputErrorDoesNotPrintFullHelp(t *testing.T) {
	cmd := newRootCommand(
		func([]string, runOptions) error {
			t.Fatal("unexpected root command execution")
			return nil
		},
		func(acp.RunOptions) error {
			t.Fatal("unexpected ACP command execution")
			return nil
		},
	)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"gatway"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected mistyped command error")
	}
	got := out.String()
	for _, notWant := range []string{
		"Available Commands:",
		"Usage:",
		"Flags:",
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("output should not include full help section %q:\n%s", notWant, got)
		}
	}
}
