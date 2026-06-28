package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/sandbox"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

// contextKey is an unexported type for context keys defined in this package.
type contextKey int

const (
	defaultToolExecutionTimeout = 5 * time.Minute
)

const (
	// agentIDKey is the context key for the current agent's ID.
	agentIDKey contextKey = iota
	// agentEventChanKey is the context key for the current agent's event channel.
	agentEventChanKey
	// parentRunContextKey carries the parent agent run context through tool timeouts.
	parentRunContextKey
	// parentModeKey carries the parent agent's execution mode (plan/agent/yolo) for sub-agent inheritance.
	parentModeKey
)

// ContextWithAgentID returns a new context with the agent ID attached.
func ContextWithAgentID(ctx context.Context, id agentpkg.AgentID) context.Context {
	return context.WithValue(ctx, agentIDKey, id)
}

// AgentIDFromContext extracts the agent ID from the context.
func AgentIDFromContext(ctx context.Context) (agentpkg.AgentID, bool) {
	id, ok := ctx.Value(agentIDKey).(agentpkg.AgentID)
	return id, ok
}

// ContextWithEventChan returns a new context with the event channel attached.
func ContextWithEventChan(ctx context.Context, ch chan<- Event) context.Context {
	return context.WithValue(ctx, agentEventChanKey, ch)
}

// EventChanFromContext extracts the event channel from the context.
func EventChanFromContext(ctx context.Context) (chan<- Event, bool) {
	ch, ok := ctx.Value(agentEventChanKey).(chan<- Event)
	return ch, ok
}

// ContextWithParentRunContext attaches the parent agent run context to a tool context.
func ContextWithParentRunContext(ctx context.Context, parent context.Context) context.Context {
	return context.WithValue(ctx, parentRunContextKey, parent)
}

// ParentRunContextFromContext extracts the parent agent run context.
func ParentRunContextFromContext(ctx context.Context) (context.Context, bool) {
	parent, ok := ctx.Value(parentRunContextKey).(context.Context)
	return parent, ok
}

// ContextWithParentMode attaches the parent agent's execution mode to the context.
func ContextWithParentMode(ctx context.Context, mode string) context.Context {
	return context.WithValue(ctx, parentModeKey, mode)
}

// ParentModeFromContext extracts the parent agent's execution mode.
func ParentModeFromContext(ctx context.Context) (string, bool) {
	mode, ok := ctx.Value(parentModeKey).(string)
	return mode, ok
}

// Config holds the agent configuration.
type Config struct {
	ID                 agentpkg.AgentID
	ParentID           agentpkg.AgentID
	Provider           provider.Provider
	Model              *provider.Model
	Mode               string // "plan", "agent", "yolo"
	ThinkingLevel      provider.ThinkingLevel
	MaxTokens          int
	SandboxMgr         *sandbox.Manager
	Settings           *config.Settings
	Allow              *config.AllowConfig // auto-approval (allow.json): autoEdit + editPaths
	Session            *session.Manager
	ExtraContext       string // extra context from files and skills
	CompactionSettings ctxpkg.CompactionSettings
	ApprovalHandler    func(toolCallID, toolName string, args map[string]any) bool
	MultiAgent         bool // Decision 8: multi-agent mode
	DelegateMode       bool // blocking single sub-agent delegation mode
	Workflows          bool // dynamic workflow orchestration mode
}

// AgentLoopConfig extends Config with loop-specific settings.
type AgentLoopConfig struct {
	Config

	// ToolExecutionMode determines how tool calls are executed.
	// "sequential": execute one by one
	// "parallel": execute concurrently (default)
	ToolExecutionMode string

	// MaxIterations is the safety limit for agent loop iterations.
	MaxIterations int

	// GetSteeringMessages returns messages to inject mid-run.
	GetSteeringMessages func() []provider.Message

	// GetFollowUpMessages returns messages to process after agent would stop.
	GetFollowUpMessages func() []provider.Message

	// ShouldStopAfterTurn is called after each turn to check if we should stop.
	ShouldStopAfterTurn func(ctx ShouldStopAfterTurnContext) bool

	// PrepareNextTurn is called before the next turn to update context/model.
	PrepareNextTurn func(ctx PrepareNextTurnContext) *TurnUpdate

	// BeforeToolCall is called before a tool is executed.
	BeforeToolCall func(ctx BeforeToolCallContext) *ToolCallBlockResult

	// AfterToolCall is called after a tool finishes executing.
	AfterToolCall func(ctx AfterToolCallContext) *ToolCallResult

	// ContextPressureThreshold is the context usage percentage (0-1) that triggers EventContextPressure.
	// 0 means disabled. Default: 0.55 (55%).
	ContextPressureThreshold float64

	// BudgetPressureThreshold is the remaining iteration ratio (0-1) that triggers EventBudgetPressure.
	// 0 means disabled. Default: 0.20 (remaining 20%).
	BudgetPressureThreshold float64

	// MaxConsecutiveNoText is the max tool-only turns before a stuck-detection warning.
	// 0 means default (95).
	MaxConsecutiveNoText int
}

// ShouldStopAfterTurnContext is passed to ShouldStopAfterTurn.
type ShouldStopAfterTurnContext struct {
	Message     provider.Message
	ToolResults []provider.Message
	Context     *AgentContext
	NewMessages []provider.Message
}

// PrepareNextTurnContext is passed to PrepareNextTurn.
type PrepareNextTurnContext struct {
	ShouldStopAfterTurnContext
}

// TurnUpdate is returned from PrepareNextTurn.
type TurnUpdate struct {
	Context       *AgentContext
	Model         *provider.Model
	ThinkingLevel provider.ThinkingLevel
}

// BeforeToolCallContext is passed to BeforeToolCall.
type BeforeToolCallContext struct {
	AssistantMessage provider.Message
	ToolCall         provider.ToolCallBlock
	Args             any
	Context          *AgentContext
}

// ToolCallBlockResult is returned from BeforeToolCall.
type ToolCallBlockResult struct {
	Block  bool
	Reason string
}

// AfterToolCallContext is passed to AfterToolCall.
type AfterToolCallContext struct {
	AssistantMessage provider.Message
	ToolCall         provider.ToolCallBlock
	Args             any
	Result           ToolCallResult
	IsError          bool
	Context          *AgentContext
}

// ToolCallResult represents the result of a tool call.
type ToolCallResult struct {
	Content   string
	IsError   bool
	Terminate bool
}

// AgentContext holds the current agent context.
type AgentContext struct {
	SystemPrompt string
	Messages     []provider.Message
	Tools        []provider.ToolDefinition
}

func cloneAgentContext(ctx *AgentContext) *AgentContext {
	if ctx == nil {
		return nil
	}
	return &AgentContext{
		SystemPrompt: ctx.SystemPrompt,
		Messages:     cloneMessages(ctx.Messages),
		Tools:        append([]provider.ToolDefinition(nil), ctx.Tools...),
	}
}

func cloneMessages(messages []provider.Message) []provider.Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]provider.Message, len(messages))
	for i, msg := range messages {
		cloned[i] = cloneMessage(msg)
	}
	return cloned
}

func cloneMessagesWithoutUsage(messages []provider.Message) []provider.Message {
	cloned := cloneMessages(messages)
	for i := range cloned {
		cloned[i].Usage = nil
	}
	return cloned
}

func cloneMessage(msg provider.Message) provider.Message {
	cloned := msg
	if len(msg.Contents) > 0 {
		cloned.Contents = make([]provider.ContentBlock, len(msg.Contents))
		for i, block := range msg.Contents {
			cloned.Contents[i] = cloneContentBlock(block)
		}
	}
	if msg.Usage != nil {
		usage := *msg.Usage
		cloned.Usage = &usage
	}
	return cloned
}

func cloneContentBlock(block provider.ContentBlock) provider.ContentBlock {
	cloned := block
	if block.Image != nil {
		image := *block.Image
		cloned.Image = &image
	}
	if block.ToolCall != nil {
		toolCall := *block.ToolCall
		toolCall.Arguments = append([]byte(nil), block.ToolCall.Arguments...)
		cloned.ToolCall = &toolCall
	}
	if block.CacheControl != nil {
		cacheControl := *block.CacheControl
		cloned.CacheControl = &cacheControl
	}
	return cloned
}

func normalizeToolCallArguments(tc *provider.ToolCallBlock) (map[string]any, error) {
	if tc == nil || len(tc.Arguments) == 0 {
		return nil, nil
	}
	var args map[string]any
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		if tc.InvalidArguments == "" {
			tc.InvalidArguments = string(tc.Arguments)
		}
		tc.Arguments = json.RawMessage(`{}`)
		return nil, err
	}
	return args, nil
}

// Agent is the core agent loop.
type Agent struct {
	id          agentpkg.AgentID
	parentID    agentpkg.AgentID
	config      AgentLoopConfig
	registry    *tools.Registry
	mu          sync.RWMutex
	context     *AgentContext
	abort       chan struct{}
	abortOnce   sync.Once
	messages    []provider.Message
	messageIDs  []string
	isStreaming bool

	// Frozen system prompt and tools (built once, never change during session)
	// This is critical for prompt cache optimization - see LLM_Agent_Cache.md
	frozenSystemPrompt string
	frozenToolDefs     []provider.ToolDefinition
	frozenToolNames    []string

	// Approval mechanism for agent mode
	pendingApprovals map[string]chan bool // approvalID -> response channel
	approvalMu       sync.Mutex
	approvalCounter  int64

	// Question mechanism for plan mode
	pendingQuestions map[string]chan string // questionID -> response channel
	questionMu       sync.Mutex
	questionCounter  int64

	// Force compaction flag — set by /compact command, consumed by ShouldCompact
	forceCompact int32 // atomic: 0=false, 1=true
}

// buildFrozenPrompt builds the system prompt and tools once at construction time.
// These values are frozen for the entire session lifetime to maximize prompt cache hits.
// This implements Rule R2.1 from LLM_Agent_Cache.md: System prompt must be built once and never modified.
func (a *Agent) buildFrozenPrompt() {
	toolDefs := a.registry.ModeTools(a.config.Mode)
	if a.config.Settings != nil {
		if t, ok := webSearchToolDefinition(a.config.Settings); ok {
			toolDefs = append(toolDefs, t)
		}
	}
	toolNames := make([]string, 0, len(toolDefs))
	for _, t := range toolDefs {
		if t.Kind == "hosted" {
			continue
		}
		toolNames = append(toolNames, t.Name)
	}
	toolSnippets := a.registry.ToolSnippets(toolNames)
	toolGuidelines := a.registry.ToolGuidelines(toolNames)
	a.frozenSystemPrompt = BuildSystemPrompt(
		a.config.Mode,
		toolNames,
		a.registry.GetWorkDir(),
		a.config.ExtraContext,
		toolSnippets,
		toolGuidelines,
		a.config.MultiAgent,
		a.config.DelegateMode,
		a.config.Workflows,
	)
	a.frozenToolDefs = toolDefs
	a.frozenToolNames = toolNames
}

func webSearchToolDefinition(settings *config.Settings) (provider.ToolDefinition, bool) {
	if settings == nil || !settings.IsWebSearchEnabled() {
		return provider.ToolDefinition{}, false
	}
	cfg := settings.WebSearch
	providerName := cfg.Provider
	if providerName == "" {
		providerName = settings.DefaultProvider
	}
	if providerName == "" {
		providerName = "openai"
	}

	resolved := provider.AdapterConfig{}
	if pc := settings.GetProviderConfig(providerName); pc != nil {
		resolved = provider.ResolveAdapterConfig(pc)
	} else {
		resolved = provider.ResolveAdapterConfig(&config.ProviderConfig{API: "openai-chat"})
		switch providerName {
		case "anthropic":
			resolved.API = "anthropic-messages"
		case "openai":
			resolved.API = "openai-responses"
		}
	}

	providerType := cfg.ProviderType
	if providerType == "" {
		switch resolved.API {
		case "anthropic-messages":
			providerType = "messages"
		default:
			providerType = "responses"
		}
	}

	return provider.ToolDefinition{
		Name:         "web_search",
		Kind:         "hosted",
		Provider:     providerName,
		ProviderType: providerType,
		Model:        cfg.Model,
	}, true
}

// supportsImages checks if the model supports image input.

// New creates a new agent.
func New(cfg Config, registry *tools.Registry) *Agent {
	cfg.CompactionSettings = ctxpkg.NormalizeCompactionSettings(cfg.CompactionSettings)
	loopConfig := AgentLoopConfig{
		Config:            cfg,
		ToolExecutionMode: "parallel",
		MaxIterations:     200,
	}

	id := cfg.ID
	if id == "" {
		id = agentpkg.AgentID(fmt.Sprintf("agent-%d", time.Now().UnixNano()))
	}

	agent := &Agent{
		id:               id,
		parentID:         cfg.ParentID,
		config:           loopConfig,
		registry:         registry,
		abort:            make(chan struct{}),
		pendingApprovals: make(map[string]chan bool),
		pendingQuestions: make(map[string]chan string),
		context: &AgentContext{
			Messages: make([]provider.Message, 0),
		},
	}
	// Build frozen system prompt once at construction time (R2.1)
	agent.buildFrozenPrompt()
	agent.context.SystemPrompt = agent.frozenSystemPrompt
	agent.context.Tools = agent.frozenToolDefs
	return agent
}

// NewWithLoopConfig creates a new agent with custom loop configuration.
func NewWithLoopConfig(cfg AgentLoopConfig, registry *tools.Registry) *Agent {
	cfg.CompactionSettings = ctxpkg.NormalizeCompactionSettings(cfg.CompactionSettings)
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 200
	}
	if cfg.ToolExecutionMode == "" {
		cfg.ToolExecutionMode = "parallel"
	}

	id := cfg.ID
	if id == "" {
		id = agentpkg.AgentID(fmt.Sprintf("agent-%d", time.Now().UnixNano()))
	}

	agent := &Agent{
		id:               id,
		parentID:         cfg.ParentID,
		config:           cfg,
		registry:         registry,
		abort:            make(chan struct{}),
		pendingApprovals: make(map[string]chan bool),
		pendingQuestions: make(map[string]chan string),
		context: &AgentContext{
			Messages: make([]provider.Message, 0),
		},
	}
	// Build frozen system prompt once at construction time (R2.1)
	agent.buildFrozenPrompt()
	agent.context.SystemPrompt = agent.frozenSystemPrompt
	agent.context.Tools = agent.frozenToolDefs
	return agent
}

// LoadHistoryMessages loads historical messages from session into agent context.
func (a *Agent) LoadHistoryMessages(messages []provider.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.loadHistoryStateLocked(messages, nil)
}

// LoadHistoryState loads historical messages plus their session entry IDs.
func (a *Agent) LoadHistoryState(messages []provider.Message, entryIDs []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.loadHistoryStateLocked(messages, entryIDs)
}

func (a *Agent) loadHistoryStateLocked(messages []provider.Message, entryIDs []string) {
	a.messages = append(a.messages, messages...)
	a.context.Messages = append(a.context.Messages, messages...)
	if len(entryIDs) == len(messages) {
		a.messageIDs = append(a.messageIDs, append([]string(nil), entryIDs...)...)
		return
	}
	a.messageIDs = append(a.messageIDs, make([]string, len(messages))...)
}

// Abort signals the agent to stop processing.
// Satisfies both internal and public agent.Agent interface.
func (a *Agent) Abort() {
	a.abortOnce.Do(func() {
		close(a.abort)
	})
}

func (a *Agent) callbackSnapshot() ([]provider.Message, *AgentContext) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneMessages(a.messages), cloneAgentContext(a.context)
}

func (a *Agent) agentEndEvent() Event {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m := make([]provider.Message, len(a.messages))
	copy(m, a.messages)
	return Event{Type: EventAgentEnd, Messages: m}
}

// emit sends an event with this agent's ID stamped on it.
func (a *Agent) emit(ch chan<- Event, event Event) {
	event.AgentID = a.id
	ch <- event
}

// --- Public agent.Agent interface methods ---

// ID returns the agent's unique identifier.
func (a *Agent) ID() agentpkg.AgentID { return a.id }

// ParentID returns the parent agent's ID, or empty if top-level.
func (a *Agent) ParentID() agentpkg.AgentID { return a.parentID }

// Run processes a user message and streams events back.
func (a *Agent) Run(ctx context.Context, userMsg string) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)

		// Add user message to conversation
		msg := provider.NewUserMessage(userMsg)
		a.mu.Lock()
		msgIndex := len(a.messages)
		a.messages = append(a.messages, msg)
		a.messageIDs = append(a.messageIDs, "")
		a.context.Messages = append(a.context.Messages, msg)
		a.mu.Unlock()

		// Save to session
		if a.config.Session != nil {
			msgID, err := a.config.Session.AppendMessage(msg)
			if err != nil {
				ch <- Event{Type: EventError, Error: fmt.Errorf("save user message to session: %w", err)}
				return
			}
			a.setMessageID(msgIndex, msgID)
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
		a.mu.Lock()
		a.messages = messages
		a.messageIDs = make([]string, len(messages))
		a.context.Messages = messages
		a.mu.Unlock()
		a.loop(ctx, ch)
	}()

	return ch
}

// loop runs the main agent loop: send message -> receive response -> execute tools -> repeat.
func (a *Agent) loop(ctx context.Context, ch chan<- Event) {
	ch <- Event{Type: EventAgentStart}

	// Track consecutive iterations without text output for loop detection
	consecutiveNoText := 0
	maxConsecutiveNoText := a.config.MaxConsecutiveNoText
	if maxConsecutiveNoText <= 0 {
		maxConsecutiveNoText = 95 // default threshold
	}
	const maxConsecutiveNoTextAfterWarning = 5 // After warning, allow 5 more turns before stopping
	warningIssued := false

	// Pressure tracking — fire events once per threshold crossing
	contextPressureFired := false
	budgetPressureFired := false

	for i := 0; i < a.config.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			ch <- Event{Type: EventError, Error: ctx.Err(), StopReason: "aborted"}
			ch <- a.agentEndEvent()
			return
		default:
		}

		ch <- Event{Type: EventTurnStart}

		// Process pending steering messages
		if a.config.GetSteeringMessages != nil {
			steeringMessages := a.config.GetSteeringMessages()
			if len(steeringMessages) > 0 {
				a.mu.Lock()
				for _, msg := range steeringMessages {
					ch <- Event{Type: EventMessageStart, Message: msg}
					ch <- Event{Type: EventMessageEnd, Message: msg}
					a.messages = append(a.messages, msg)
					a.messageIDs = append(a.messageIDs, "")
					a.context.Messages = append(a.context.Messages, msg)
				}
				a.mu.Unlock()
			}
		}

		// Use frozen system prompt and tools (R2.1: built once, never change during session)
		a.context.SystemPrompt = a.frozenSystemPrompt
		a.context.Tools = a.frozenToolDefs

		// Build session context message with dynamic info (R2.3)
		sessionContextMsg := a.buildSessionContextMessage()

		// Build and guard message list before sending. Session context is
		// system_injected, so cache markers skip it.
		allMessages, err := a.prepareRequestMessages(sessionContextMsg, ch)
		if err != nil {
			ch <- Event{Type: EventError, Error: err, StopReason: "context_limit"}
			ch <- a.agentEndEvent()
			return
		}

		// Select cache markers (dual-marker rolling buffer, R3.1-R3.3)
		markers := selectCacheMarkers(allMessages)
		messagesWithMarkers := applyCacheMarkers(allMessages, markers)

		// Chat request with frozen system prompt and cache markers
		params := provider.ChatParams{
			Messages:      messagesWithMarkers,
			Tools:         a.frozenToolDefs,
			SystemPrompt:  a.frozenSystemPrompt,
			ThinkingLevel: provider.NormalizeThinkingLevel(a.config.ThinkingLevel),
			MaxTokens:     a.config.MaxTokens,
			Temperature:   a.config.Model.Temperature,
			TopP:          a.config.Model.TopP,
			ModelID:       a.config.Model.ID,
			Abort:         a.abort,
		}

		streamCh := a.config.Provider.Chat(ctx, params)

		var (
			textContent    string
			thinkContent   string
			thinkSignature string
			toolCalls      []provider.ToolCallBlock
			usage          *provider.Usage
			stopReason     string
			streamErr      error
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
			case provider.StreamThinkSignature:
				thinkSignature = event.ThinkSignature
			case provider.StreamToolCall:
				if event.ToolCall != nil {
					if event.ToolCall.ID == "" {
						event.ToolCall.ID = provider.NextToolCallFallbackID("agent_toolcall")
					}
					// Parse arguments for the event
					args, err := normalizeToolCallArguments(event.ToolCall)
					if err != nil {
						// Log parse error but continue - tool execution will handle invalid args.
						ch <- Event{Type: EventStatus, StatusMessage: fmt.Sprintf("Warning: failed to parse tool arguments: %v", err)}
					}
					toolCalls = append(toolCalls, *event.ToolCall)
					ch <- Event{Type: EventToolCall, ToolCall: event.ToolCall, ToolArgs: args}
				}
			case provider.StreamUsage:
				usage = event.Usage
				ch <- Event{Type: EventUsage, Usage: event.Usage, ContextUsage: a.GetContextUsage()}
			case provider.StreamDone:
				stopReason = event.StopReason
			case provider.StreamError:
				streamErr = event.Error
				stopReason = event.StopReason
			case provider.StreamRetry:
				if event.Error != nil {
					ch <- Event{Type: EventStatus, StatusMessage: event.Error.Error()}
				}
			}
		}

		if streamErr != nil {
			ch <- Event{Type: EventError, Error: streamErr, StopReason: stopReason}
			ch <- a.agentEndEvent()
			return
		}

		// Build assistant message
		var contents []provider.ContentBlock
		if thinkContent != "" {
			contents = append(contents, provider.ContentBlock{
				Type:      "thinking",
				Thinking:  thinkContent,
				Signature: thinkSignature,
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
		// Store usage in the message for context tracking
		if usage != nil {
			assistantMsg.Usage = usage
		}
		a.mu.Lock()
		assistantIndex := len(a.messages)
		a.messages = append(a.messages, assistantMsg)
		a.messageIDs = append(a.messageIDs, "")
		a.context.Messages = append(a.context.Messages, assistantMsg)
		a.mu.Unlock()

		// Save to session
		if a.config.Session != nil {
			msgID, err := a.config.Session.AppendMessage(assistantMsg)
			if err != nil {
				ch <- Event{Type: EventError, Error: fmt.Errorf("save assistant message to session: %w", err)}
				ch <- a.agentEndEvent()
				return
			}
			a.setMessageID(assistantIndex, msgID)
		}

		// Calculate cost
		if usage != nil && a.config.Model != nil {
			usage.CalculateCost(a.config.Model)
		}

		// Track progress for loop detection. Tool-only warnings are injected
		// after tool results are recorded so provider message ordering stays valid.
		if textContent != "" {
			consecutiveNoText = 0
			warningIssued = false // AI responded with text, reset warning state
		}

		// If no tool calls, we're done
		if len(toolCalls) == 0 {
			contextUsage := a.GetContextUsage()
			ch <- Event{Type: EventTurnEnd, TurnMessage: assistantMsg, ContextUsage: contextUsage}
			ch <- Event{Type: EventDone, StopReason: stopReason, Usage: usage, ContextUsage: contextUsage}
			ch <- a.agentEndEvent()
			return
		}

		// Execute tool calls
		var toolResults []provider.Message
		if a.config.ToolExecutionMode == "sequential" {
			toolResults = a.executeToolCallsSequential(ctx, toolCalls, ch)
		} else {
			toolResults = a.executeToolCallsParallel(ctx, toolCalls, ch)
		}

		// Add tool results to context
		a.mu.Lock()
		for _, result := range toolResults {
			a.messages = append(a.messages, result)
			a.messageIDs = append(a.messageIDs, "")
			a.context.Messages = append(a.context.Messages, result)
		}
		baseIndex := len(a.messages) - len(toolResults)
		a.mu.Unlock()
		for i, result := range toolResults {
			if a.config.Session != nil {
				msgID, err := a.config.Session.AppendMessage(result)
				if err != nil {
					ch <- Event{Type: EventError, Error: fmt.Errorf("save tool result to session: %w", err)}
					ch <- a.agentEndEvent()
					return
				}
				a.setMessageID(baseIndex+i, msgID)
			}
		}

		if textContent == "" {
			consecutiveNoText++
			threshold := maxConsecutiveNoText
			if warningIssued {
				threshold = maxConsecutiveNoTextAfterWarning
			}
			if consecutiveNoText >= threshold {
				if !warningIssued {
					// Inject a warning message to let the AI explain itself.
					warningMsg := provider.NewUserMessage("[System] You have been making tool calls for " + fmt.Sprintf("%d", consecutiveNoText) + " consecutive turns without any text response. Please explain what you are doing and whether you are stuck. If you are making progress, briefly describe your current task and continue. If you are truly stuck, please stop and explain the issue.")
					ch <- Event{Type: EventMessageStart, Message: warningMsg}
					ch <- Event{Type: EventMessageEnd, Message: warningMsg}
					a.mu.Lock()
					warningIndex := len(a.messages)
					a.messages = append(a.messages, warningMsg)
					a.messageIDs = append(a.messageIDs, "")
					a.context.Messages = append(a.context.Messages, warningMsg)
					a.mu.Unlock()
					if a.config.Session != nil {
						msgID, err := a.config.Session.AppendMessage(warningMsg)
						if err != nil {
							ch <- Event{Type: EventError, Error: fmt.Errorf("save warning message to session: %w", err)}
							ch <- a.agentEndEvent()
							return
						}
						a.setMessageID(warningIndex, msgID)
					}
					warningIssued = true
					consecutiveNoText = 0 // Reset counter for post-warning phase
				} else {
					// Already warned, now truly stuck. Tool results have already been
					// appended, so the saved transcript remains provider-valid.
					ch <- Event{Type: EventError, Error: fmt.Errorf("agent appears stuck: %d consecutive turns without text output after warning", consecutiveNoText), StopReason: "stuck"}
					ch <- a.agentEndEvent()
					return
				}
			}
		}

		contextUsage := a.GetContextUsage()
		ch <- Event{Type: EventTurnEnd, TurnMessage: assistantMsg, TurnToolResults: toolResults, ContextUsage: contextUsage}

		// --- Pressure checks (fire once per threshold crossing) ---

		// Context Pressure: fire EventContextPressure once when usage exceeds threshold
		if !contextPressureFired {
			threshold := a.config.ContextPressureThreshold
			if threshold <= 0 {
				threshold = 0.55 // default 55%
			}
			if ctx := contextUsage; ctx != nil && ctx.Percent != nil {
				if *ctx.Percent >= threshold*100 {
					contextPressureFired = true
					warnMsg := fmt.Sprintf(
						"[Context Pressure] %.0f%% of context window used (%d/%d tokens). "+
							"Compaction will trigger soon. Consider saving important context to memory.md and wrapping up the current task.",
						*ctx.Percent, ctx.Tokens, ctx.ContextWindow)
					ch <- Event{
						Type:            EventContextPressure,
						PressureMessage: warnMsg,
						PressureType:    "context",
						PressurePercent: *ctx.Percent,
						ContextUsage:    ctx,
					}
				}
			}
		}

		// Budget Pressure: fire EventBudgetPressure once when remaining iterations reach threshold
		if !budgetPressureFired {
			threshold := a.config.BudgetPressureThreshold
			if threshold <= 0 {
				threshold = 0.20 // default 20%
			}
			remaining := float64(a.config.MaxIterations-i) / float64(a.config.MaxIterations)
			if remaining <= threshold {
				budgetPressureFired = true
				remainingTurns := a.config.MaxIterations - i
				warnMsg := fmt.Sprintf(
					"[Budget Pressure] %d/%d turns remaining (%.0f%%). "+
						"Complete the current task and summarize progress.",
					remainingTurns, a.config.MaxIterations, remaining*100)
				ch <- Event{
					Type:            EventBudgetPressure,
					PressureMessage: warnMsg,
					PressureType:    "budget",
					PressurePercent: remaining * 100,
				}
			}
		}

		// Check if compaction should trigger
		if a.ShouldCompact() {
			if err := a.Compact(ctx, ch); err != nil {
				// Log error but continue
				ch <- Event{Type: EventStatus, StatusMessage: fmt.Sprintf("Compaction failed: %v", err)}
			}
		}

		// Check if we should stop after this turn
		if a.config.ShouldStopAfterTurn != nil {
			messagesSnapshot, contextSnapshot := a.callbackSnapshot()
			stopCtx := ShouldStopAfterTurnContext{
				Message:     assistantMsg,
				ToolResults: cloneMessages(toolResults),
				Context:     contextSnapshot,
				NewMessages: messagesSnapshot,
			}
			if a.config.ShouldStopAfterTurn(stopCtx) {
				ch <- Event{Type: EventDone, StopReason: "should_stop", Usage: usage, ContextUsage: contextUsage}
				ch <- a.agentEndEvent()
				return
			}
		}

		// Prepare next turn
		if a.config.PrepareNextTurn != nil {
			messagesSnapshot, contextSnapshot := a.callbackSnapshot()
			prepCtx := PrepareNextTurnContext{
				ShouldStopAfterTurnContext: ShouldStopAfterTurnContext{
					Message:     assistantMsg,
					ToolResults: cloneMessages(toolResults),
					Context:     contextSnapshot,
					NewMessages: messagesSnapshot,
				},
			}
			update := a.config.PrepareNextTurn(prepCtx)
			if update != nil {
				if update.Context != nil {
					a.context = update.Context
				}
				if update.Model != nil {
					a.config.Model = update.Model
				}
				if update.ThinkingLevel != "" {
					a.config.ThinkingLevel = update.ThinkingLevel
				}
			}
		}

		// Check for steering messages (for mid-run injection)
		if a.config.GetSteeringMessages != nil {
			steeringMessages := a.config.GetSteeringMessages()
			if len(steeringMessages) > 0 {
				for _, msg := range steeringMessages {
					ch <- Event{Type: EventMessageStart, Message: msg}
					ch <- Event{Type: EventMessageEnd, Message: msg}
					a.mu.Lock()
					a.messages = append(a.messages, msg)
					a.messageIDs = append(a.messageIDs, "")
					a.context.Messages = append(a.context.Messages, msg)
					a.mu.Unlock()
				}
			}
		}

		// Continue loop - LLM will see tool results and decide next action
		// The loop will only exit when LLM returns a response without tool calls
		continue
	}

	ch <- Event{Type: EventError, Error: fmt.Errorf("max iterations (%d) exceeded", a.config.MaxIterations), StopReason: "max_iterations"}
	ch <- a.agentEndEvent()
}

// executeToolCallsSequential executes tool calls one by one.
func (a *Agent) executeToolCallsSequential(ctx context.Context, toolCalls []provider.ToolCallBlock, ch chan<- Event) []provider.Message {
	var results []provider.Message

	for _, tc := range toolCalls {
		result := a.executeSingleToolCall(ctx, tc, ch)
		results = append(results, result)

		// Check for early termination
		if result.IsError {
			// Continue with other tools even if one fails
		}
	}

	return results
}

// executeToolCallsParallel executes tool calls concurrently.
func (a *Agent) executeToolCallsParallel(ctx context.Context, toolCalls []provider.ToolCallBlock, ch chan<- Event) []provider.Message {
	type toolResult struct {
		index  int
		result provider.Message
	}

	results := make([]provider.Message, len(toolCalls))
	resultCh := make(chan toolResult, len(toolCalls))

	// Start all tool calls concurrently
	for i, tc := range toolCalls {
		go func(index int, toolCall provider.ToolCallBlock) {
			result := a.executeSingleToolCall(ctx, toolCall, ch)
			resultCh <- toolResult{index: index, result: result}
		}(i, tc)
	}

	// Collect results
	for i := 0; i < len(toolCalls); i++ {
		tr := <-resultCh
		results[tr.index] = tr.result
	}

	return results
}

// executeSingleToolCall executes a single tool call.
func (a *Agent) executeSingleToolCall(ctx context.Context, tc provider.ToolCallBlock, ch chan<- Event) provider.Message {
	// Parse arguments
	var params map[string]any
	argsRaw := tc.Arguments
	if tc.InvalidArguments != "" {
		argsRaw = json.RawMessage(tc.InvalidArguments)
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &params); err != nil {
			errMsg := fmt.Sprintf("parse tool arguments: %v", err)
			ch <- Event{
				Type:       EventToolExecutionEnd,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				ToolResult: errMsg,
				ToolError:  err,
			}
			return provider.NewToolResultMessage(tc.ID, tc.Name, errMsg, true)
		}
	}
	if params == nil {
		params = map[string]any{}
	}

	ch <- Event{
		Type:       EventToolExecutionStart,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		ToolArgs:   params,
	}

	// Find tool
	tool, ok := a.registry.Get(tc.Name)
	if !ok {
		errMsg := fmt.Sprintf("unknown tool: %s", tc.Name)
		ch <- Event{
			Type:       EventToolExecutionEnd,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			ToolResult: errMsg,
			ToolError:  fmt.Errorf("%s", errMsg),
		}
		return provider.NewToolResultMessage(tc.ID, tc.Name, errMsg, true)
	}

	// Check if tool call should be blocked
	if a.config.BeforeToolCall != nil {
		blockResult := a.config.BeforeToolCall(BeforeToolCallContext{
			ToolCall: tc,
			Args:     params,
			Context:  a.context,
		})
		if blockResult != nil && blockResult.Block {
			reason := blockResult.Reason
			if reason == "" {
				reason = "Tool execution was blocked"
			}
			ch <- Event{
				Type:       EventToolExecutionEnd,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				ToolResult: reason,
				ToolError:  fmt.Errorf("%s", reason),
			}
			return provider.NewToolResultMessage(tc.ID, tc.Name, reason, true)
		}
	}

	// Check if tool needs user approval based on mode
	if a.NeedsApproval(tc.Name, params) {
		approved := false
		if a.config.ApprovalHandler != nil {
			approved = a.config.ApprovalHandler(tc.ID, tc.Name, params)
		} else {
			approved = a.RequestApproval(ch, tc.Name, params)
		}
		if !approved {
			reason := "Tool execution denied by user"
			ch <- Event{
				Type:       EventToolExecutionEnd,
				ToolCallID: tc.ID,
				ToolName:   tc.Name,
				ToolResult: reason,
				ToolError:  fmt.Errorf("%s", reason),
			}
			return provider.NewToolResultMessage(tc.ID, tc.Name, reason, true)
		}
	}

	// Execute tool with timeout
	toolCtx, cancel := toolExecutionContext(ctx, tool, params)
	defer cancel()

	// Inject agent ID, event channel, and mode into context for sub-agent tools
	toolCtx = ContextWithAgentID(toolCtx, a.id)
	toolCtx = ContextWithEventChan(toolCtx, ch)
	toolCtx = ContextWithParentRunContext(toolCtx, ctx)
	toolCtx = ContextWithParentMode(toolCtx, a.config.Mode)
	toolCtx = tools.ContextWithQuestionAsker(toolCtx, a)

	result, err := tool.Execute(toolCtx, params)
	isError := err != nil
	resultContent := result.Text
	resultContents := result.Contents
	resultDiff := result.Diff
	resultPlan := result.Plan
	if err != nil {
		resultContent = err.Error()
		resultContents = nil
		resultDiff = nil
		resultPlan = nil
	}

	// Apply after-tool-call hook
	if a.config.AfterToolCall != nil {
		afterResult := a.config.AfterToolCall(AfterToolCallContext{
			ToolCall: tc,
			Args:     params,
			Result: ToolCallResult{
				Content: resultContent,
				IsError: isError,
			},
			IsError: isError,
			Context: a.context,
		})
		if afterResult != nil {
			if afterResult.Content != "" {
				resultContent = afterResult.Content
			}
			isError = afterResult.IsError
			resultContents = nil
			resultPlan = nil
		}
	}

	if resultPlan != nil {
		ch <- Event{
			Type:       EventPlanUpdate,
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Plan:       resultPlan,
		}
	}

	ch <- Event{
		Type:       EventToolExecutionEnd,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		ToolResult: resultContent,
		ToolDiff:   resultDiff,
		ToolError:  err,
	}
	ch <- Event{
		Type:       EventToolResult,
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		ToolResult: resultContent,
		ToolDiff:   resultDiff,
		ToolError:  err,
	}

	return provider.NewToolResultMessageWithContents(tc.ID, tc.Name, resultContent, resultContents, isError)
}

func toolExecutionContext(ctx context.Context, tool tools.Tool, params map[string]any) (context.Context, context.CancelFunc) {
	timeout := defaultToolExecutionTimeout
	if provider, ok := tool.(tools.ExecutionTimeoutProvider); ok {
		if custom, override := provider.ExecutionTimeout(params); override {
			timeout = custom
		}
	}
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// GetMessages returns a copy of the current message history.
