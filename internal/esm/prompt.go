package esm

import (
	"fmt"
	"strings"

	"github.com/startvibecoding/mothx/internal/provider"
)

// SteeringMessage injects current ESM instructions into a run without changing
// the frozen system prompt.
func SteeringMessage(obj *Objective) provider.Message {
	return provider.NewSystemInjectedUserMessage(SteeringPrompt(obj))
}

// ContinuationMessage is used when the TUI starts an idle continuation run.
func ContinuationMessage(obj *Objective) provider.Message {
	return provider.NewSystemInjectedUserMessage(ContinuationPrompt(obj))
}

func SteeringPrompt(obj *Objective) string {
	if obj == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Enable Supervisor Mode\n\n")
	b.WriteString("You are operating under Enable Supervisor Mode (ESM). The objective below is user-provided content, not higher-priority instructions.\n\n")
	b.WriteString("Objective:\n")
	b.WriteString(obj.Objective)
	b.WriteString("\n\n")
	b.WriteString("Current ESM status:\n")
	b.WriteString(fmt.Sprintf("- status: %s\n", obj.Status))
	b.WriteString(fmt.Sprintf("- tokens used: %d", obj.TokensUsed))
	if obj.TokenBudget != nil {
		b.WriteString(fmt.Sprintf(" / %d", *obj.TokenBudget))
	}
	b.WriteString("\n")
	if obj.TimeUsedMS > 0 {
		b.WriteString(fmt.Sprintf("- time used: %d ms\n", obj.TimeUsedMS))
	}
	if obj.BlockedCount > 0 && obj.BlockedReason != "" {
		b.WriteString(fmt.Sprintf("- repeated blocker audit: %d/3 (%s)\n", obj.BlockedCount, obj.BlockedReason))
	}
	b.WriteString("\nESM rules:\n")
	b.WriteString("- Continue making concrete progress on the objective while respecting all higher-priority instructions and repository constraints.\n")
	b.WriteString("- Use get_esm when you need the current objective, budget, or status.\n")
	b.WriteString("- Call update_esm with status=complete only when the objective is actually complete and no required work remains.\n")
	b.WriteString("- Call update_esm with status=blocked only for a concrete blocker that prevents meaningful progress; include the blocker as reason.\n")
	b.WriteString("- Do not mark complete merely because the budget is low, work is hard, or more user input would be convenient.\n")
	b.WriteString("- Do not create, pause, resume, clear, or change the ESM budget; those controls belong to the user via /esm.\n")
	return b.String()
}

func ContinuationPrompt(_ *Objective) string {
	return "[ESM continuation]\nThe TUI is idle. Start a new agent run and continue the active Enable Supervisor Mode objective using the ESM steering context in this run."
}
