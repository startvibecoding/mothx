package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fuckvibecoding/vibecoding/internal/config"
	"github.com/fuckvibecoding/vibecoding/internal/provider"
	"github.com/fuckvibecoding/vibecoding/internal/sandbox"
	"github.com/fuckvibecoding/vibecoding/internal/session"
	"github.com/fuckvibecoding/vibecoding/internal/tools"
)

// Event represents an event from the agent to the UI.
type Event struct {
	Type EventType

	// Stream events
	TextDelta  string
	ThinkDelta string
	ToolCall   *provider.ToolCallBlock

	// Tool events
	ToolName   string
	ToolResult string
	ToolError  error

	// Status
	Message string

	// Completion
	Done       bool
	StopReason string
	Error      error

	// Usage
	Usage *provider.Usage
}

// EventType identifies the type of agent event.
type EventType int

const (
	EventTextDelta    EventType = iota
	EventThinkDelta
	EventToolCall
	EventToolStart
	EventToolResult
	EventStatus
	EventDone
	EventError
	EventUsage
)

// Config holds the agent configuration.
type Config struct {
	Provider      provider.Provider
	Model         *provider.Model
	Mode          string // "plan", "agent", "yolo"
	ThinkingLevel provider.ThinkingLevel
	MaxTokens     int
	SandboxMgr    *sandbox.Manager
	Settings      *config.Settings
	Session       *session.Manager
	ExtraContext  string // extra context from files and skills
}

// Agent is the core agent loop.
type Agent struct {
	config    Config
	registry  *tools.Registry
	messages  []provider.Message
	abort     chan struct{}
	abortOnce sync.Once
}

// New creates a new agent.
func New(cfg Config, registry *tools.Registry) *Agent {
	return &Agent{
		config:   cfg,
		registry: registry,
		abort:    make(chan struct{}),
	}
}

// Abort signals the agent to stop processing.
func (a *Agent) Abort() {
	a.abortOnce.Do(func() {
		close(a.abort)
		a.abort = make(chan struct{})
	})
}

// Run processes a user message and streams events back.
func (a *Agent) Run(ctx context.Context, userMsg string) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)

		// Add user message to conversation
		msg := provider.NewUserMessage(userMsg)
		a.messages = append(a.messages, msg)

		// Save to session
		if a.config.Session != nil {
			a.config.Session.AppendMessage(msg)
		}

		// Run agent loop
		a.loop(ctx, ch)
	}()

	return ch
}

// RunWithMessages processes with explicit message history.
func (a *Agent) RunWithMessages(ctx context.Context, messages []provider.Message) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		a.messages = messages
		a.loop(ctx, ch)
	}()

	return ch
}

// loop runs the main agent loop: send message -> receive response -> execute tools -> repeat.
func (a *Agent) loop(ctx context.Context, ch chan<- Event) {
	maxIterations := 50 // Safety limit

	for i := 0; i < maxIterations; i++ {
		select {
		case <-ctx.Done():
			ch <- Event{Type: EventError, Error: ctx.Err()}
			return
		default:
		}

		// Build system prompt
		toolNames := make([]string, 0)
		for _, t := range a.registry.ModeTools(a.config.Mode) {
			toolNames = append(toolNames, t.Name)
		}
		systemPrompt := BuildSystemPrompt(a.config.Mode, toolNames, a.registry.GetWorkDir(), a.config.ExtraContext)

		// Build context messages
		messages := a.buildContext(systemPrompt)

		// Get tool definitions for current mode
		toolDefs := a.registry.ModeTools(a.config.Mode)

		// Chat request
		params := provider.ChatParams{
			Messages:      messages,
			Tools:         toolDefs,
			SystemPrompt:  systemPrompt,
			ThinkingLevel: a.config.ThinkingLevel,
			MaxTokens:     a.config.MaxTokens,
			Abort:         a.abort,
		}

		streamCh := a.config.Provider.Chat(ctx, params)

		var (
			textContent   string
			thinkContent  string
			toolCalls     []provider.ToolCallBlock
			usage         *provider.Usage
			stopReason    string
			streamErr     error
		)

		// Process stream events
		for event := range streamCh {
			switch event.Type {
			case provider.StreamStart:
				// Stream started
			case provider.StreamTextDelta:
				textContent += event.TextDelta
				ch <- Event{Type: EventTextDelta, TextDelta: event.TextDelta}
			case provider.StreamThinkDelta:
				thinkContent += event.ThinkDelta
				ch <- Event{Type: EventThinkDelta, ThinkDelta: event.ThinkDelta}
			case provider.StreamToolCall:
				if event.ToolCall != nil {
					toolCalls = append(toolCalls, *event.ToolCall)
					ch <- Event{Type: EventToolCall, ToolCall: event.ToolCall}
				}
			case provider.StreamUsage:
				usage = event.Usage
				ch <- Event{Type: EventUsage, Usage: event.Usage}
			case provider.StreamDone:
				stopReason = event.StopReason
			case provider.StreamError:
				streamErr = event.Error
				stopReason = event.StopReason
			}
		}

		if streamErr != nil {
			ch <- Event{Type: EventError, Error: streamErr}
			return
		}

		// Build assistant message
		var contents []provider.ContentBlock
		if thinkContent != "" {
			contents = append(contents, provider.ContentBlock{
				Type:     "thinking",
				Thinking: thinkContent,
			})
		}
		if textContent != "" {
			contents = append(contents, provider.ContentBlock{
				Type: "text",
				Text: textContent,
			})
		}
		for _, tc := range toolCalls {
			tc := tc
			contents = append(contents, provider.ContentBlock{
				Type:     "toolCall",
				ToolCall: &tc,
			})
		}

		assistantMsg := provider.NewAssistantMessage(contents)
		a.messages = append(a.messages, assistantMsg)

		// Save to session
		if a.config.Session != nil {
			a.config.Session.AppendMessage(assistantMsg)
		}

		// Calculate cost
		if usage != nil && a.config.Model != nil {
			usage.CalculateCost(a.config.Model)
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			ch <- Event{Type: EventDone, StopReason: stopReason}
			return
		}

		// Execute tool calls
		for _, tc := range toolCalls {
			ch <- Event{Type: EventToolStart, ToolName: tc.Name}

			result, err := a.executeTool(ctx, tc)

			isError := err != nil
			resultContent := result
			if err != nil {
				resultContent = err.Error()
			}

			toolResultMsg := provider.NewToolResultMessage(tc.ID, tc.Name, resultContent, isError)
			a.messages = append(a.messages, toolResultMsg)

			// Save to session
			if a.config.Session != nil {
				a.config.Session.AppendMessage(toolResultMsg)
			}

			ch <- Event{
				Type:       EventToolResult,
				ToolName:   tc.Name,
				ToolResult: resultContent,
				ToolError:  err,
			}
		}

		// Continue loop with tool results
	}

	ch <- Event{Type: EventError, Error: fmt.Errorf("max iterations (%d) exceeded", maxIterations)}
}

// executeTool executes a single tool call.
func (a *Agent) executeTool(ctx context.Context, tc provider.ToolCallBlock) (string, error) {
	tool, ok := a.registry.Get(tc.Name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", tc.Name)
	}

	var params map[string]any
	if len(tc.Arguments) > 0 {
		if err := json.Unmarshal(tc.Arguments, &params); err != nil {
			return "", fmt.Errorf("parse tool arguments: %w", err)
		}
	}

	// Add timeout for tool execution
	toolCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return tool.Execute(toolCtx, params)
}

// buildContext builds the message list for the LLM, respecting context limits.
func (a *Agent) buildContext(systemPrompt string) []provider.Message {
	messages := a.messages

	// Estimate tokens and trim if needed
	maxContext := 200000 // default
	if a.config.Model != nil && a.config.Model.ContextWindow > 0 {
		maxContext = a.config.Model.ContextWindow
	}
	if a.config.Settings != nil && a.config.Settings.MaxContextTokens > 0 {
		maxContext = a.config.Settings.MaxContextTokens
	}

	// Reserve tokens for output
	reserve := 16384
	if a.config.Settings != nil && a.config.Settings.Compaction.ReserveTokens > 0 {
		reserve = a.config.Settings.Compaction.ReserveTokens
	}

	targetTokens := maxContext - reserve

	// Estimate tokens (rough: 1 token ≈ 4 chars)
	totalChars := 0
	for _, m := range messages {
		totalChars += len(m.Content)
		for _, c := range m.Contents {
			totalChars += len(c.Text) + len(c.Thinking)
		}
	}
	estimatedTokens := totalChars / 3 // conservative estimate

	if estimatedTokens > targetTokens && len(messages) > 4 {
		// Keep the last few messages and summarize the rest
		// In a full implementation, this would call the LLM to compact
		// For now, just keep recent messages
		keepCount := len(messages) / 2
		if keepCount < 4 {
			keepCount = 4
		}
		if keepCount > len(messages) {
			keepCount = len(messages)
		}
		messages = messages[len(messages)-keepCount:]
	}

	return messages
}

// GetMessages returns the current message history.
func (a *Agent) GetMessages() []provider.Message {
	return a.messages
}

// SetMessages replaces the message history.
func (a *Agent) SetMessages(msgs []provider.Message) {
	a.messages = msgs
}
