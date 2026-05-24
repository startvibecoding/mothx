package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PlanTool publishes a structured task plan for UI and audit surfaces.
type PlanTool struct {
	registry *Registry
}

// NewPlanTool creates a new plan tool.
func NewPlanTool(r *Registry) *PlanTool {
	return &PlanTool{registry: r}
}

func (t *PlanTool) Name() string { return "plan" }

func (t *PlanTool) Description() string {
	return "Publish or update a structured task plan with step statuses."
}

func (t *PlanTool) PromptSnippet() string {
	return "Publish a visible task plan with pending, running, done, or failed steps"
}

func (t *PlanTool) PromptGuidelines() []string {
	return []string{
		"Use plan before making code changes for multi-step tasks.",
		"Update plan step statuses as work progresses.",
		"Keep plan steps concise and actionable.",
	}
}

func (t *PlanTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"title": {
				"type": "string",
				"description": "Short title for the current task plan"
			},
			"steps": {
				"type": "array",
				"description": "Ordered task steps with statuses",
				"items": {
					"type": "object",
					"properties": {
						"title": {
							"type": "string",
							"description": "Concise step description"
						},
						"status": {
							"type": "string",
							"enum": ["pending", "running", "done", "failed"],
							"description": "Current step status"
						}
					},
					"required": ["title", "status"]
				}
			},
			"note": {
				"type": "string",
				"description": "Optional short note about risks, blockers, or next action"
			}
		},
		"required": ["steps"]
	}`)
}

func (t *PlanTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	title, _ := params["title"].(string)
	note, _ := params["note"].(string)
	stepsRaw, ok := params["steps"].([]any)
	if !ok || len(stepsRaw) == 0 {
		return ToolResult{}, fmt.Errorf("steps array is required and must not be empty")
	}

	plan := &TaskPlan{
		Title: strings.TrimSpace(title),
		Note:  strings.TrimSpace(note),
		Steps: make([]PlanStep, 0, len(stepsRaw)),
	}
	for i, raw := range stepsRaw {
		m, ok := raw.(map[string]any)
		if !ok {
			return ToolResult{}, fmt.Errorf("step %d: invalid step format", i)
		}
		stepTitle, _ := m["title"].(string)
		stepTitle = strings.TrimSpace(stepTitle)
		if stepTitle == "" {
			return ToolResult{}, fmt.Errorf("step %d: title is required", i)
		}
		status, _ := m["status"].(string)
		status = normalizePlanStatus(status)
		if status == "" {
			return ToolResult{}, fmt.Errorf("step %d: status must be pending, running, done, or failed", i)
		}
		plan.Steps = append(plan.Steps, PlanStep{Title: stepTitle, Status: status})
	}

	return NewPlanToolResult(formatTaskPlan(plan), plan), nil
}

func normalizePlanStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "running", "done", "failed":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return ""
	}
}

func formatTaskPlan(plan *TaskPlan) string {
	if plan == nil {
		return "Plan updated."
	}
	var sb strings.Builder
	if plan.Title != "" {
		sb.WriteString("Plan: " + plan.Title + "\n")
	} else {
		sb.WriteString("Plan updated:\n")
	}
	for _, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", step.Status, step.Title))
	}
	if plan.Note != "" {
		sb.WriteString("Note: " + plan.Note)
	}
	return strings.TrimRight(sb.String(), "\n")
}
