package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// QuestionTool asks the user a multiple-choice question during plan mode.
type QuestionTool struct {
	registry *Registry
}

// NewQuestionTool creates a new question tool.
func NewQuestionTool(r *Registry) *QuestionTool {
	return &QuestionTool{registry: r}
}

func (t *QuestionTool) Name() string { return "question" }

func (t *QuestionTool) Description() string {
	return "Ask the user a question with predefined options to clarify requirements before forming a plan. The user selects an option or provides a custom answer."
}

func (t *QuestionTool) PromptSnippet() string {
	return "Ask the user a multiple-choice question to clarify requirements"
}

func (t *QuestionTool) PromptGuidelines() []string {
	return []string{
		"Use question when you need the user to make a decision or clarify requirements before planning",
		"Provide clear, concise options that cover the main choices",
		"The last option is always 'Custom input' — the user can type their own answer",
		"Use context to explain why you're asking and what each option means",
		"Ask one question at a time for clarity",
	}
}

func (t *QuestionTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"question": {
				"type": "string",
				"description": "The question to ask the user"
			},
			"options": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Predefined options for the user to choose from"
			},
			"context": {
				"type": "string",
				"description": "Optional context or explanation for why you're asking this question"
			}
		},
		"required": ["question", "options"]
	}`)
}

// QuestionAsker is the interface the tool uses to interact with the user.
// The agent implements this via RequestQuestion.
type QuestionAsker interface {
	AskQuestion(ctx context.Context, question string, options []string, context string) string
}

func (t *QuestionTool) Execute(ctx context.Context, params map[string]any) (ToolResult, error) {
	question, _ := params["question"].(string)
	if question == "" {
		return ToolResult{}, fmt.Errorf("question is required")
	}

	optionsRaw, ok := params["options"].([]any)
	if !ok || len(optionsRaw) == 0 {
		return ToolResult{}, fmt.Errorf("options array is required and must not be empty")
	}

	var options []string
	for i, raw := range optionsRaw {
		opt, ok := raw.(string)
		if !ok {
			return ToolResult{}, fmt.Errorf("option %d must be a string", i)
		}
		options = append(options, strings.TrimSpace(opt))
	}

	explanation, _ := params["context"].(string)

	// Look for the QuestionAsker in the context
	asker, ok := ctx.Value(questionAskerKey{}).(QuestionAsker)
	if !ok {
		return ToolResult{}, fmt.Errorf("question tool: no question handler available in context")
	}

	answer := asker.AskQuestion(ctx, question, options, explanation)
	if answer == "" {
		return ToolResult{}, fmt.Errorf("no answer received (user may have aborted)")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("User answered: %s\n", answer))
	return NewTextToolResult(sb.String()), nil
}

// questionAskerKey is the context key for the QuestionAsker.
type questionAskerKey struct{}

// ContextWithQuestionAsker attaches a QuestionAsker to the context.
func ContextWithQuestionAsker(ctx context.Context, asker QuestionAsker) context.Context {
	return context.WithValue(ctx, questionAskerKey{}, asker)
}
