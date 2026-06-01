package main

import (
	"reflect"
	"testing"

	"github.com/startvibecoding/vibecoding/internal/acp"
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
	cmd.SetArgs([]string{"acp", "-p", "anthropic", "-m", "claude-test", "-M", "yolo", "-t", "medium", "--sandbox", "--verbose", "--debug"})

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
