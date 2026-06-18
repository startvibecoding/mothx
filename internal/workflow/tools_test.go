package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	internalagent "github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

func TestRegisterToolsRegistersOnlyWorkflowTools(t *testing.T) {
	registry := tools.NewRegistry(t.TempDir(), nil)
	manager := internalagent.NewAgentManager(&internalagent.AgentFactory{})

	RegisterTools(registry, manager, &memoryStore{})

	for _, name := range []string{"workflow_run", "workflow_status", "workflow_cancel"} {
		if _, ok := registry.Get(name); !ok {
			t.Fatalf("expected %s to be registered", name)
		}
	}
	for _, name := range []string{"subagent_spawn", "subagent_status", "subagent_send", "subagent_destroy", "delegate_subagent"} {
		if _, ok := registry.Get(name); ok {
			t.Fatalf("did not expect %s to be registered by workflow tools", name)
		}
	}
}

func TestRunToolPromptGuidelinesRequireCompleteElispSource(t *testing.T) {
	tool := NewRunTool(nil, nil)
	guidelines := strings.Join(tool.PromptGuidelines(), "\n")
	for _, want := range []string{
		"plain Elisp syntax",
		"do not use Markdown code fences",
		"balanced parentheses",
		"closed double-quoted strings",
		":tools '(\"read\" \"grep\")",
	} {
		if !strings.Contains(guidelines, want) {
			t.Fatalf("workflow_run guidelines missing %q:\n%s", want, guidelines)
		}
	}

	params := string(tool.Parameters())
	for _, want := range []string{
		"Complete raw Elisp workflow DSL source",
		"one balanced (workflow",
		"closed double-quoted strings",
		"Markdown fences",
	} {
		if !strings.Contains(params, want) {
			t.Fatalf("workflow_run schema missing %q:\n%s", want, params)
		}
	}
	if strings.Contains(guidelines, "JSON DSL") || strings.Contains(params, "JSON DSL") {
		t.Fatalf("workflow_run prompt text should avoid JSON DSL negative guidance:\n%s\n%s", guidelines, params)
	}
}

func TestCancelToolCancelsActiveRun(t *testing.T) {
	active := NewActiveRegistry()
	canceled := false
	if err := active.Register("run-1", func() { canceled = true }); err != nil {
		t.Fatalf("register active run: %v", err)
	}

	result, err := NewCancelTool(active).Execute(context.Background(), map[string]any{"id": "run-1"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !canceled {
		t.Fatal("expected active run cancel function to be called")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result.Text), &parsed); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if parsed["status"] != StatusCanceled {
		t.Fatalf("status = %v, want canceled", parsed["status"])
	}
}

func TestCancelToolRejectsInactiveRun(t *testing.T) {
	_, err := NewCancelTool(NewActiveRegistry()).Execute(context.Background(), map[string]any{"id": "missing"})
	if err == nil {
		t.Fatal("expected inactive workflow error")
	}
}
