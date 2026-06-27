package agent

import "context"

// ExternalTool is a custom tool supplied by an embedding application
// (for example, PostgreBase's schema/data tools).
//
// External tools let host applications expose their own controlled
// capabilities to the agent without depending on the agent's built-in
// coding tools. Combine with Builder.WithoutBuiltinTools to run an agent
// that may ONLY use the host-provided tools.
type ExternalTool interface {
	// Name returns the tool's name (must match ^[a-zA-Z0-9_-]+$ for most providers).
	Name() string

	// Description returns a human-readable description of what the tool does.
	Description() string

	// Parameters returns the JSON Schema (as raw JSON bytes) for the tool's
	// input parameters.
	Parameters() []byte

	// Execute runs the tool with the decoded parameters.
	Execute(ctx context.Context, params map[string]any) (ExternalToolResult, error)
}

// ExternalToolResult is the normalized result returned by an ExternalTool.
type ExternalToolResult struct {
	// Text is the plain-text result surfaced to the model and logs.
	Text string

	// IsError marks the result as an error so the agent can react accordingly.
	IsError bool

	// Contents optionally carries rich content blocks (e.g. images) for
	// multimodal results. When empty, Text is used.
	Contents []ContentBlock
}

// ExternalToolPromptInfo is an optional interface an ExternalTool may implement
// to contribute richer system-prompt hints. When not implemented, the agent
// falls back to the tool name and description.
type ExternalToolPromptInfo interface {
	// PromptSnippet returns a short one-line description for the system prompt.
	PromptSnippet() string
	// PromptGuidelines returns guideline bullets for the system prompt.
	PromptGuidelines() []string
}
