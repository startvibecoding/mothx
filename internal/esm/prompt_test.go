package esm

import (
	"strings"
	"testing"
)

func TestSteeringPromptRequiresFullObjectiveAudit(t *testing.T) {
	obj := &Objective{
		SessionID: "sess",
		ESMID:     "esm",
		Objective: "ship full goal <not demo> & verify",
		Status:    StatusActive,
	}

	prompt := SteeringPrompt(obj)
	required := []string{
		"Do not shrink it to a demo",
		"Completion audit before update_esm complete",
		"evidence proves every requirement",
		"Treat missing, weak, indirect, or uncertain evidence as not complete",
		"verification evidence in reason",
		"completion candidate only",
	}
	for _, want := range required {
		if !strings.Contains(prompt, want) {
			t.Fatalf("SteeringPrompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "<not demo>") {
		t.Fatalf("objective was not escaped:\n%s", prompt)
	}
	if !strings.Contains(prompt, "&lt;not demo&gt; &amp; verify") {
		t.Fatalf("escaped objective missing:\n%s", prompt)
	}
}

func TestUpdateToolRequiresReason(t *testing.T) {
	tool := NewUpdateTool(nil, nil)
	params := string(tool.Parameters())
	if !strings.Contains(params, `"required":["status","reason"]`) {
		t.Fatalf("update_esm parameters do not require reason: %s", params)
	}

	guidelines := strings.Join(tool.PromptGuidelines(), "\n")
	for _, want := range []string{
		"complete_candidate",
		"Do not mark complete for a demo",
		"three consecutive ESM agent runs",
	} {
		if !strings.Contains(guidelines, want) {
			t.Fatalf("update_esm guidelines missing %q:\n%s", want, guidelines)
		}
	}
}

func TestWorkerAndAuditPromptsUseIsolatedRoles(t *testing.T) {
	obj := &Objective{
		Objective:        "ship real feature",
		Status:           StatusCompleteCandidate,
		CompletionReason: "worker says complete",
		CompletionReview: "previous audit failed",
		BlockedCount:     1,
		BlockedReason:    "missing token",
	}

	worker := WorkerTaskPrompt(obj)
	for _, want := range []string{
		"ESM worker sub-agent",
		"Work toward the full objective, not a demo",
		"complete_candidate",
		"Do not call get_esm or update_esm",
		"Previous failed completion audit",
	} {
		if !strings.Contains(worker, want) {
			t.Fatalf("WorkerTaskPrompt missing %q:\n%s", want, worker)
		}
	}

	audit := AuditTaskPrompt(obj)
	for _, want := range []string{
		"ESM audit sub-agent",
		"must be skeptical",
		"Completion candidate evidence",
		"Pass only when your own tool-backed evidence proves the full objective is complete",
		"You must use tools to inspect the current repository state",
		`"verdict":"pass|fail"`,
	} {
		if !strings.Contains(audit, want) {
			t.Fatalf("AuditTaskPrompt missing %q:\n%s", want, audit)
		}
	}

	critic := CriticTaskPrompt(obj)
	for _, want := range []string{
		"ESM critic sub-agent",
		"challenge the worker's completion claim",
		"Look for demos",
		"Pass only when your own tool-backed inspection finds no hard blocker",
		"You must use tools to inspect the current repository state",
		`"verdict":"pass|fail"`,
	} {
		if !strings.Contains(critic, want) {
			t.Fatalf("CriticTaskPrompt missing %q:\n%s", want, critic)
		}
	}
}

func TestBudgetLimitPromptWrapsUpWithoutNewWork(t *testing.T) {
	budget := int64(10)
	obj := &Objective{
		SessionID:   "sess",
		ESMID:       "esm",
		Objective:   "ship <full> & verified",
		Status:      StatusBudgetLimited,
		TokenBudget: &budget,
		TokensUsed:  12,
	}

	prompt := BudgetLimitPrompt(obj)
	for _, want := range []string{
		"budget_limited",
		"Do not start new substantive work",
		"Wrap up soon",
		"Do not call update_esm unless the objective is actually complete",
		"ship &lt;full&gt; &amp; verified",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("BudgetLimitPrompt missing %q:\n%s", want, prompt)
		}
	}
}
