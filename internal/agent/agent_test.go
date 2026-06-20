package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/sandbox"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

type loopingToolProvider struct {
	models    []*provider.Model
	callCount int
}

type recordingToolProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

type compactionReplayProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

type workflowRunToolProvider struct {
	models []*provider.Model
	calls  []provider.ChatParams
}

func newRecordingToolProvider() *recordingToolProvider {
	return &recordingToolProvider{
		models: []*provider.Model{{ID: "model1", Name: "Model 1"}},
	}
}

func newCompactionReplayProvider() *compactionReplayProvider {
	return &compactionReplayProvider{
		models: []*provider.Model{{
			ID:            "model1",
			Name:          "Model 1",
			ContextWindow: 4096,
			MaxTokens:     1024,
		}},
	}
}

func (p *recordingToolProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	p.calls = append(p.calls, provider.ChatParams{
		Messages: cloneMessages(params.Messages),
		Tools:    append([]provider.ToolDefinition(nil), params.Tools...),
	})

	ch := make(chan provider.StreamEvent, 3)
	callNumber := len(p.calls)
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		if callNumber == 1 {
			ch <- provider.StreamEvent{Type: provider.StreamToolCall, ToolCall: &provider.ToolCallBlock{
				ID:        "call_1",
				Name:      "bash",
				Arguments: []byte(`{"command":"echo visible"}`),
			}}
		} else {
			ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "done"}
		}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *recordingToolProvider) Name() string {
	return "recording"
}

func (p *recordingToolProvider) Models() []*provider.Model {
	return p.models
}

func (p *recordingToolProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (p *compactionReplayProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	p.calls = append(p.calls, provider.ChatParams{
		Messages: cloneMessages(params.Messages),
		Tools:    append([]provider.ToolDefinition(nil), params.Tools...),
	})

	ch := make(chan provider.StreamEvent, 3)
	callNumber := len(p.calls)
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		switch callNumber {
		case 1:
			ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "## Goal\ncheckpoint"}
		default:
			ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "continued"}
		}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *compactionReplayProvider) Name() string {
	return "compaction-replay"
}

func (p *compactionReplayProvider) Models() []*provider.Model {
	return p.models
}

func (p *compactionReplayProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (p *workflowRunToolProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	p.calls = append(p.calls, provider.ChatParams{
		Messages: cloneMessages(params.Messages),
		Tools:    append([]provider.ToolDefinition(nil), params.Tools...),
	})

	ch := make(chan provider.StreamEvent, 3)
	callNumber := len(p.calls)
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		if callNumber == 1 {
			ch <- provider.StreamEvent{Type: provider.StreamToolCall, ToolCall: &provider.ToolCallBlock{
				ID:        "workflow_call_1",
				Name:      "workflow_run",
				Arguments: []byte(`{"source":"(workflow \"slow\")"}`),
			}}
		} else {
			ch <- provider.StreamEvent{Type: provider.StreamTextDelta, TextDelta: "workflow complete"}
		}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *workflowRunToolProvider) Name() string {
	return "workflow-run-tool"
}

func (p *workflowRunToolProvider) Models() []*provider.Model {
	return p.models
}

func (p *workflowRunToolProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

type fixedBashTool struct{}

func (fixedBashTool) Name() string { return "bash" }

func (fixedBashTool) Description() string { return "fake bash for tests" }

func (fixedBashTool) PromptSnippet() string { return "fake bash" }

func (fixedBashTool) PromptGuidelines() []string { return nil }

func (fixedBashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"}}}`)
}

func (fixedBashTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	return tools.NewTextToolResult("bash output visible to the next model turn"), nil
}

type hugeBashTool struct {
	fixedBashTool
	output string
}

func (t hugeBashTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	return tools.NewTextToolResult(t.output), nil
}

type timeoutBashTool struct {
	fixedBashTool
	timeout time.Duration
}

func (t timeoutBashTool) ExecutionTimeout(params map[string]any) (time.Duration, bool) {
	return t.timeout, true
}

type blockingWorkflowRunTool struct {
	delay time.Duration
}

func (blockingWorkflowRunTool) Name() string { return "workflow_run" }

func (blockingWorkflowRunTool) Description() string { return "fake workflow_run for tests" }

func (blockingWorkflowRunTool) PromptSnippet() string { return "fake workflow_run" }

func (blockingWorkflowRunTool) PromptGuidelines() []string { return nil }

func (blockingWorkflowRunTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"source":{"type":"string"}}}`)
}

func (t blockingWorkflowRunTool) Execute(ctx context.Context, params map[string]any) (tools.ToolResult, error) {
	timer := time.NewTimer(t.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return tools.ToolResult{}, ctx.Err()
	case <-timer.C:
		return tools.NewTextToolResult(`{"id":"run-1","status":"done"}`), nil
	}
}

func TestToolExecutionContextUsesToolTimeoutOverride(t *testing.T) {
	parent := context.Background()

	defaultCtx, defaultCancel := toolExecutionContext(parent, fixedBashTool{}, nil)
	defer defaultCancel()
	if _, ok := defaultCtx.Deadline(); !ok {
		t.Fatal("expected default tool execution deadline")
	}

	customCtx, customCancel := toolExecutionContext(parent, timeoutBashTool{timeout: 2 * time.Second}, nil)
	defer customCancel()
	deadline, ok := customCtx.Deadline()
	if !ok {
		t.Fatal("expected custom tool execution deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > 3*time.Second {
		t.Fatalf("custom deadline remaining = %s, want about 2s", remaining)
	}

	noDeadlineCtx, noDeadlineCancel := toolExecutionContext(parent, timeoutBashTool{timeout: 0}, nil)
	defer noDeadlineCancel()
	if _, ok := noDeadlineCtx.Deadline(); ok {
		t.Fatal("expected no agent-level deadline when tool timeout is zero")
	}
}

func newLoopingToolProvider() *loopingToolProvider {
	return &loopingToolProvider{
		models: []*provider.Model{{ID: "model1", Name: "Model 1"}},
	}
}

func (p *loopingToolProvider) Chat(ctx context.Context, params provider.ChatParams) <-chan provider.StreamEvent {
	ch := make(chan provider.StreamEvent, 3)
	p.callCount++
	toolCall := &provider.ToolCallBlock{
		ID:        fmt.Sprintf("call_%d", p.callCount),
		Name:      "unknown_tool",
		Arguments: []byte(`{}`),
	}
	go func() {
		defer close(ch)
		ch <- provider.StreamEvent{Type: provider.StreamStart}
		ch <- provider.StreamEvent{Type: provider.StreamToolCall, ToolCall: toolCall}
		ch <- provider.StreamEvent{Type: provider.StreamDone}
	}()
	return ch
}

func (p *loopingToolProvider) Name() string {
	return "looping"
}

func (p *loopingToolProvider) Models() []*provider.Model {
	return p.models
}

func (p *loopingToolProvider) GetModel(id string) *provider.Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func TestNewAgent(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	if a == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestNewWithLoopConfig(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     100,
	}

	a := NewWithLoopConfig(cfg, registry)

	if a == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentAbort(t *testing.T) {
	// Use a slow provider that gives us time to abort
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamTextDelta, TextDelta: "Hello"},
		{Type: provider.StreamDone},
	}

	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	// Run and collect events
	ch := a.Run(context.Background(), "test")

	// Abort after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		a.Abort()
	}()

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	// Should have events (abort may or may not cause error depending on timing)
	if len(events) == 0 {
		t.Error("expected at least one event")
	}
}

func TestAgentRun(t *testing.T) {
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamTextDelta, TextDelta: "Hello"},
		{Type: provider.StreamTextDelta, TextDelta: " World"},
		{Type: provider.StreamDone},
	}

	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)
	ch := a.Run(context.Background(), "test")

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	// Should have: AgentStart, TurnStart, TextDelta, TextDelta, TurnEnd, Done, AgentEnd
	if len(events) < 5 {
		t.Errorf("expected at least 5 events, got %d", len(events))
	}

	// Check first event is AgentStart
	if events[0].Type != EventAgentStart {
		t.Errorf("expected first event to be AgentStart, got %d", events[0].Type)
	}

	// Check last event is AgentEnd
	lastEvent := events[len(events)-1]
	if lastEvent.Type != EventAgentEnd {
		t.Errorf("expected last event to be AgentEnd, got %d", lastEvent.Type)
	}
}

func TestRunWithMessages(t *testing.T) {
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamTextDelta, TextDelta: "Response"},
		{Type: provider.StreamDone},
	}

	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	messages := []provider.Message{
		provider.NewUserMessage("Hello"),
	}

	ch := a.RunWithMessages(context.Background(), messages)

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	if len(events) < 3 {
		t.Errorf("expected at least 3 events, got %d", len(events))
	}
}

func TestGetSetMessages(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	messages := []provider.Message{
		provider.NewUserMessage("Hello"),
		provider.NewAssistantMessage([]provider.ContentBlock{
			{Type: "text", Text: "Hi"},
		}),
	}

	a.SetMessages(messages)

	got := a.GetMessages()
	if len(got) != 2 {
		t.Errorf("expected 2 messages, got %d", len(got))
	}
}

func TestGetSetContext(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	ctx := &AgentContext{
		SystemPrompt: "test prompt",
		Messages:     []provider.Message{provider.NewUserMessage("Hello")},
	}

	a.SetContext(ctx)

	got := a.GetContext()
	if got.SystemPrompt != "test prompt" {
		t.Errorf("expected system prompt 'test prompt', got '%s'", got.SystemPrompt)
	}
}

func TestAgentRunWithToolCall(t *testing.T) {
	toolCall := &provider.ToolCallBlock{
		ID:        "call_1",
		Name:      "ls",
		Arguments: []byte(`{"path": "."}`),
	}

	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamToolCall, ToolCall: toolCall},
		{Type: provider.StreamDone},
	}

	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)
	ch := a.Run(context.Background(), "list files")

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	// Check that tool events are present
	hasToolCall := false
	hasToolExecution := false
	for _, event := range events {
		if event.Type == EventToolCall {
			hasToolCall = true
		}
		if event.Type == EventToolExecutionStart {
			hasToolExecution = true
		}
	}

	if !hasToolCall {
		t.Error("expected tool call event")
	}

	if !hasToolExecution {
		t.Error("expected tool execution event")
	}
}

func TestToolResultIsIncludedInNextProviderTurn(t *testing.T) {
	recorder := newRecordingToolProvider()

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)
	registry.Register(fixedBashTool{})

	cfg := AgentLoopConfig{
		Config: Config{
			Provider: recorder,
			Model:    recorder.Models()[0],
			Mode:     "yolo",
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     2,
	}

	a := NewWithLoopConfig(cfg, registry)
	for range a.Run(context.Background(), "run bash") {
	}

	if len(recorder.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(recorder.calls))
	}

	var foundToolResult bool
	for _, msg := range recorder.calls[1].Messages {
		if msg.Role == "toolResult" && msg.ToolName == "bash" && strings.Contains(messageTextForTest(msg), "bash output visible") {
			foundToolResult = true
			break
		}
	}
	if !foundToolResult {
		t.Fatalf("second provider call did not include bash tool result: %#v", recorder.calls[1].Messages)
	}
}

func TestWorkflowRunWaitDoesNotConsumeMainAgentIterations(t *testing.T) {
	delay := 75 * time.Millisecond
	recorder := &workflowRunToolProvider{
		models: []*provider.Model{{ID: "model1", Name: "Model 1"}},
	}

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)
	registry.Register(blockingWorkflowRunTool{delay: delay})

	cfg := AgentLoopConfig{
		Config: Config{
			Provider: recorder,
			Model:    recorder.Models()[0],
			Mode:     "yolo",
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     2,
	}

	a := NewWithLoopConfig(cfg, registry)
	start := time.Now()
	var turnStarts int
	var toolStarted bool
	var toolEnded bool
	var runErr error
	for ev := range a.Run(context.Background(), "run workflow") {
		switch ev.Type {
		case EventTurnStart:
			turnStarts++
		case EventToolExecutionStart:
			if ev.ToolName == "workflow_run" {
				toolStarted = true
			}
		case EventToolExecutionEnd:
			if ev.ToolName == "workflow_run" {
				toolEnded = true
			}
		case EventError:
			runErr = ev.Error
		}
	}
	elapsed := time.Since(start)

	if runErr != nil {
		t.Fatalf("Run() error = %v", runErr)
	}
	if elapsed < delay {
		t.Fatalf("workflow_run did not block long enough: elapsed %s, delay %s", elapsed, delay)
	}
	if len(recorder.calls) != 2 {
		t.Fatalf("expected exactly 2 provider calls, got %d", len(recorder.calls))
	}
	if turnStarts != 2 {
		t.Fatalf("expected exactly 2 main-agent turns, got %d", turnStarts)
	}
	if !toolStarted || !toolEnded {
		t.Fatalf("expected workflow_run tool start/end events, started=%v ended=%v", toolStarted, toolEnded)
	}
}

func TestOversizedToolResultIsOmittedBeforeNextProviderTurn(t *testing.T) {
	recorder := &recordingToolProvider{
		models: []*provider.Model{{
			ID:            "model1",
			Name:          "Model 1",
			ContextWindow: 12000,
			MaxTokens:     512,
		}},
	}

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)
	registry.Register(hugeBashTool{output: strings.Repeat("x", 60000)})

	cfg := AgentLoopConfig{
		Config: Config{
			Provider:  recorder,
			Model:     recorder.Models()[0],
			Mode:      "yolo",
			MaxTokens: 512,
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     2,
	}

	a := NewWithLoopConfig(cfg, registry)
	for range a.Run(context.Background(), "run bash") {
	}

	if len(recorder.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(recorder.calls))
	}

	var guardMessage string
	for _, msg := range recorder.calls[1].Messages {
		if msg.Role == "toolResult" && msg.ToolName == "bash" {
			guardMessage = messageTextForTest(msg)
			if strings.Contains(guardMessage, strings.Repeat("x", 1000)) {
				t.Fatal("second provider call included oversized raw tool result")
			}
			break
		}
	}
	if !strings.Contains(guardMessage, "[Context guard]") {
		t.Fatalf("second provider call missing context guard tool result: %#v", recorder.calls[1].Messages)
	}
	if !strings.Contains(guardMessage, "offset/limit") || !strings.Contains(guardMessage, "maxResults") {
		t.Fatalf("context guard did not instruct the model to narrow scope: %q", guardMessage)
	}
}

func messageTextForTest(msg provider.Message) string {
	if msg.Content != "" || len(msg.Contents) == 0 {
		return msg.Content
	}
	var parts []string
	for _, block := range msg.Contents {
		if block.Type == "text" && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func TestToolOnlyWarningAppendedAfterToolResults(t *testing.T) {
	mockProvider := newLoopingToolProvider()

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	var stopped bool
	cfg := AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     95,
		ShouldStopAfterTurn: func(ctx ShouldStopAfterTurnContext) bool {
			for _, msg := range ctx.NewMessages {
				if msg.Role == "user" && contains(msg.Content, "You have been making tool calls") {
					stopped = true
					return true
				}
			}
			return false
		},
	}

	a := NewWithLoopConfig(cfg, registry)
	ch := a.Run(context.Background(), "keep using tools")

	for range ch {
	}

	if !stopped {
		t.Fatal("expected warning-triggered stop")
	}

	messages := a.GetMessages()
	warningIndex := -1
	for i, msg := range messages {
		if msg.Role == "user" && contains(msg.Content, "You have been making tool calls") {
			warningIndex = i
			break
		}
	}
	if warningIndex < 2 {
		t.Fatalf("warning index = %d, want at least 2", warningIndex)
	}
	if messages[warningIndex-1].Role != "toolResult" {
		t.Fatalf("message before warning role = %q, want toolResult", messages[warningIndex-1].Role)
	}
	if messages[warningIndex-2].Role != "assistant" {
		t.Fatalf("message before tool result role = %q, want assistant", messages[warningIndex-2].Role)
	}
}

func TestShouldStopAfterTurnDoneIncludesFinalMetadata(t *testing.T) {
	toolCall := &provider.ToolCallBlock{
		ID:        "call_1",
		Name:      "unknown_tool",
		Arguments: []byte(`{}`),
	}
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamToolCall, ToolCall: toolCall},
		{Type: provider.StreamUsage, Usage: &provider.Usage{Input: 10, Output: 3}},
		{Type: provider.StreamDone, StopReason: "tool_use"},
	}
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1", ContextWindow: 50000, MaxTokens: 512},
	}, responses)

	cfg := AgentLoopConfig{
		Config: Config{
			Provider:  mockProvider,
			Model:     mockProvider.Models()[0],
			Mode:      "agent",
			MaxTokens: 512,
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     2,
		ShouldStopAfterTurn: func(ctx ShouldStopAfterTurnContext) bool {
			return true
		},
	}
	a := NewWithLoopConfig(cfg, tools.NewRegistry(t.TempDir(), sandbox.NewNoneSandbox()))

	var done *Event
	for event := range a.Run(context.Background(), "use a tool") {
		if event.Type == EventDone {
			ev := event
			done = &ev
		}
	}

	if done == nil {
		t.Fatal("expected EventDone")
	}
	if done.StopReason != "should_stop" {
		t.Fatalf("stop reason = %q, want should_stop", done.StopReason)
	}
	if done.Usage == nil {
		t.Fatal("expected EventDone to include usage")
	}
	if done.ContextUsage == nil {
		t.Fatal("expected EventDone to include context usage")
	}
}

func TestSessionSaveErrorEmitsAgentEnd(t *testing.T) {
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamTextDelta, TextDelta: "hello"},
		{Type: provider.StreamDone},
	}
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sess := session.New(t.TempDir(), t.TempDir())
	if err := sess.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}
	sessionFile := sess.GetFile()
	if err := os.Chmod(sessionFile, 0400); err != nil {
		t.Fatalf("chmod session file read-only: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(sessionFile, 0600)
	})

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
		Session:  sess,
	}
	a := New(cfg, tools.NewRegistry(t.TempDir(), sandbox.NewNoneSandbox()))

	var events []Event
	for event := range a.RunWithMessages(context.Background(), []provider.Message{provider.NewUserMessage("test")}) {
		events = append(events, event)
	}

	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[len(events)-1].Type != EventAgentEnd {
		t.Fatalf("last event = %v, want EventAgentEnd", events[len(events)-1].Type)
	}
	var sawError bool
	for _, event := range events {
		if event.Type == EventError && event.Error != nil && strings.Contains(event.Error.Error(), "save assistant message to session") {
			sawError = true
			break
		}
	}
	if !sawError {
		t.Fatal("expected save assistant message error")
	}
}

func TestInvalidToolArgumentsDoNotBreakSessionSave(t *testing.T) {
	responses := []provider.StreamEvent{
		{Type: provider.StreamStart},
		{Type: provider.StreamToolCall, ToolCall: &provider.ToolCallBlock{
			ID:        "call_1",
			Name:      "bash",
			Arguments: json.RawMessage(`]`),
		}},
		{Type: provider.StreamDone},
	}
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses)

	sess := session.New(t.TempDir(), t.TempDir())
	if err := sess.Init(); err != nil {
		t.Fatalf("init session: %v", err)
	}

	cfg := AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
			Session:  sess,
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     1,
	}
	a := NewWithLoopConfig(cfg, tools.NewRegistry(t.TempDir(), sandbox.NewNoneSandbox()))

	var events []Event
	for event := range a.Run(context.Background(), "test") {
		events = append(events, event)
	}

	var sawParseError bool
	for _, event := range events {
		if event.Type == EventError && event.Error != nil && strings.Contains(event.Error.Error(), "save assistant message to session") {
			t.Fatalf("unexpected session save error: %v", event.Error)
		}
		if event.Type == EventToolExecutionEnd && event.ToolError != nil && strings.Contains(event.ToolError.Error(), "invalid character ']'") {
			sawParseError = true
		}
	}
	if !sawParseError {
		t.Fatal("expected tool argument parse error")
	}

	rawSession, err := os.ReadFile(sess.GetFile())
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if !strings.Contains(string(rawSession), `"arguments":{}`) {
		t.Fatalf("expected sanitized arguments in session, got:\n%s", string(rawSession))
	}
	if !strings.Contains(string(rawSession), `"invalidArguments":"]"`) {
		t.Fatalf("expected original invalid arguments in session, got:\n%s", string(rawSession))
	}
	if _, err := session.Open(sess.GetFile()); err != nil {
		t.Fatalf("reopen session: %v", err)
	}
}

func TestCallbackSnapshotDoesNotExposeInternalSlices(t *testing.T) {
	mockProvider := newMockProvider()
	a := New(Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}, tools.NewRegistry(t.TempDir(), sandbox.NewNoneSandbox()))

	a.messages = []provider.Message{
		provider.NewAssistantMessage([]provider.ContentBlock{{
			Type: "toolCall",
			ToolCall: &provider.ToolCallBlock{
				ID:        "call-1",
				Name:      "read",
				Arguments: json.RawMessage(`{"path":"a"}`),
			},
		}}),
	}
	a.context.Messages = a.messages

	messages, ctx := a.callbackSnapshot()
	messages[0].Contents[0].ToolCall.Name = "mutated"
	ctx.Messages[0].Contents[0].ToolCall.Arguments[0] = '{'

	if a.messages[0].Contents[0].ToolCall.Name != "read" {
		t.Fatalf("internal tool name mutated: %s", a.messages[0].Contents[0].ToolCall.Name)
	}
	if string(a.context.Messages[0].Contents[0].ToolCall.Arguments) != `{"path":"a"}` {
		t.Fatalf("internal arguments mutated: %s", string(a.context.Messages[0].Contents[0].ToolCall.Arguments))
	}
}

func TestAgentRunSequential(t *testing.T) {
	toolCall1 := &provider.ToolCallBlock{
		ID:        "call_1",
		Name:      "ls",
		Arguments: []byte(`{"path": "."}`),
	}

	// First call returns tool call, second call returns text
	callCount := 0
	responses := func() []provider.StreamEvent {
		callCount++
		if callCount == 1 {
			return []provider.StreamEvent{
				{Type: provider.StreamStart},
				{Type: provider.StreamToolCall, ToolCall: toolCall1},
				{Type: provider.StreamDone},
			}
		}
		return []provider.StreamEvent{
			{Type: provider.StreamStart},
			{Type: provider.StreamTextDelta, TextDelta: "Done"},
			{Type: provider.StreamDone},
		}
	}

	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, responses())

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry("/tmp", sb)
	registry.RegisterDefaults()

	cfg := AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
		},
		ToolExecutionMode: "sequential",
		MaxIterations:     10,
	}

	a := NewWithLoopConfig(cfg, registry)
	ch := a.Run(context.Background(), "test")

	var events []Event
	for event := range ch {
		events = append(events, event)
	}

	// Should have tool execution and text events
	hasToolExecution := false
	for _, event := range events {
		if event.Type == EventToolExecutionStart {
			hasToolExecution = true
		}
	}

	if !hasToolExecution {
		t.Error("expected tool execution event")
	}
}

func TestWebSearchToolDefinitionCarriesModelMetadata(t *testing.T) {
	settings := &config.Settings{
		WebSearch: config.WebSearchSettings{
			Enabled:      config.BoolPtr(true),
			Provider:     "anthropic",
			ProviderType: "messages",
			Model:        "claude-sonnet-4-20250514",
		},
	}
	def, ok := webSearchToolDefinition(settings)
	if !ok {
		t.Fatal("expected web search tool definition")
	}
	if def.Name != "web_search" {
		t.Fatalf("name = %q, want web_search", def.Name)
	}
	if def.Provider != "anthropic" {
		t.Fatalf("provider = %q, want anthropic", def.Provider)
	}
	if def.ProviderType != "messages" {
		t.Fatalf("providerType = %q, want messages", def.ProviderType)
	}
	if def.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("model = %q, want claude-sonnet-4-20250514", def.Model)
	}
}

func TestWebSearchToolDefinitionResolvesProviderReference(t *testing.T) {
	settings := &config.Settings{
		DefaultProvider: "gpt",
		WebSearch: config.WebSearchSettings{
			Enabled:      config.BoolPtr(true),
			Provider:     "gpt",
			ProviderType: "responses",
		},
		Providers: map[string]*config.ProviderConfig{
			"gpt": {
				BaseURL: "https://co.yes.vg/v1",
				API:     "openai-responses",
			},
		},
	}
	def, ok := webSearchToolDefinition(settings)
	if !ok {
		t.Fatal("expected web search tool definition")
	}
	if def.Provider != "gpt" {
		t.Fatalf("provider = %q, want gpt", def.Provider)
	}
	if def.ProviderType != "responses" {
		t.Fatalf("providerType = %q, want responses", def.ProviderType)
	}
	if def.Provider == "" {
		t.Fatal("expected hosted provider to be resolved")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	toolNames := []string{"read", "write", "bash"}
	cwd := "/home/user/project"
	extraContext := "## Extra\nSome extra context"
	toolSnippets := map[string]string{
		"read":  "Read file contents",
		"write": "Create or overwrite files",
		"bash":  "Execute bash commands",
	}
	toolGuidelines := []string{"Use read to examine files instead of cat or sed."}

	prompt := BuildSystemPrompt("agent", toolNames, cwd, extraContext, toolSnippets, toolGuidelines, false, false, false)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	// Check that prompt contains expected content
	if !contains(prompt, "VibeCoding") {
		t.Error("expected prompt to contain 'VibeCoding'")
	}

	if !contains(prompt, "/home/user/project") {
		t.Error("expected prompt to contain working directory")
	}

	if !contains(prompt, "read") {
		t.Error("expected prompt to contain tool names")
	}

	if !contains(prompt, "Extra") {
		t.Error("expected prompt to contain extra context")
	}
}

func TestBuildSystemPromptModes(t *testing.T) {
	// Test plan mode
	planPrompt := BuildSystemPrompt("plan", nil, "/tmp", "", nil, nil, false, false, false)
	if !contains(planPrompt, "PLAN") {
		t.Error("expected plan prompt to contain 'PLAN'")
	}

	if !contains(planPrompt, "READ-ONLY") {
		t.Error("expected plan prompt to contain 'READ-ONLY'")
	}

	// Test agent mode
	agentPrompt := BuildSystemPrompt("agent", nil, "/tmp", "", nil, nil, false, false, false)
	if !contains(agentPrompt, "AGENT") {
		t.Error("expected agent prompt to contain 'AGENT'")
	}

	// Test yolo mode
	yoloPrompt := BuildSystemPrompt("yolo", nil, "/tmp", "", nil, nil, false, false, false)
	if !contains(yoloPrompt, "YOLO") {
		t.Error("expected yolo prompt to contain 'YOLO'")
	}

	// Test unknown mode
	unknownPrompt := BuildSystemPrompt("custom", nil, "/tmp", "", nil, nil, false, false, false)
	if !contains(unknownPrompt, "CUSTOM") {
		t.Error("expected unknown prompt to contain mode name")
	}
}

func TestBuildSystemPromptMultiAgentGated(t *testing.T) {
	defaultPrompt := BuildSystemPrompt("agent", nil, "/tmp", "", nil, nil, false, false, false)
	if contains(defaultPrompt, "Sub-Agent Tools") {
		t.Error("expected default prompt to omit sub-agent instructions")
	}

	multiPrompt := BuildSystemPrompt("agent", []string{"subagent_spawn"}, "/tmp", "", nil, nil, true, false, false)
	if !contains(multiPrompt, "Sub-Agent Tools") {
		t.Error("expected multi-agent prompt to include sub-agent instructions")
	}
	if !contains(multiPrompt, "Act as the orchestrator") {
		t.Error("expected multi-agent prompt to include orchestration guidance")
	}
}

func TestBuildSystemPromptDelegateModeGated(t *testing.T) {
	defaultPrompt := BuildSystemPrompt("agent", nil, "/tmp", "", nil, nil, false, false, false)
	if contains(defaultPrompt, "Delegation Mode") {
		t.Error("expected default prompt to omit delegation instructions")
	}

	delegatePrompt := BuildSystemPrompt("agent", []string{"delegate_subagent"}, "/tmp", "", nil, nil, false, true, false)
	if !contains(delegatePrompt, "Delegation Mode") {
		t.Error("expected delegate prompt to include delegation instructions")
	}
	if !contains(delegatePrompt, "delegate_subagent") {
		t.Error("expected delegate prompt to mention delegate_subagent")
	}
}

func TestBuildSystemPromptWorkflowGated(t *testing.T) {
	defaultPrompt := BuildSystemPrompt("agent", nil, "/tmp", "", nil, nil, false, false, false)
	if contains(defaultPrompt, "Workflow Tools") {
		t.Error("expected default prompt to omit workflow instructions")
	}
	if contains(defaultPrompt, "Sub-Agent Tools") {
		t.Error("expected default prompt to omit sub-agent instructions")
	}
	if contains(defaultPrompt, "Elisp VM scope") || contains(defaultPrompt, "Workflow DSL forms") || contains(defaultPrompt, "Syntax checklist before workflow_run") {
		t.Error("expected default prompt to omit workflow Elisp DSL reference")
	}

	workflowPrompt := BuildSystemPrompt("agent", []string{"workflow_run"}, "/tmp", "", nil, nil, false, false, true)
	if !contains(workflowPrompt, "Workflow Tools") {
		t.Error("expected workflow prompt to include workflow instructions")
	}
	if contains(workflowPrompt, "Sub-Agent Tools") {
		t.Error("expected workflow prompt to omit sub-agent instructions")
	}
	for _, want := range []string{
		"raw Elisp text, not Markdown",
		"active workflow-elisp skill",
		"workflow, phase, and agent names must be string literals",
		"defun and defmacro support fixed argument lists only",
	} {
		if !contains(workflowPrompt, want) {
			t.Errorf("expected workflow prompt to contain %q", want)
		}
	}
	for _, unwanted := range []string{
		"Elisp VM scope",
		"Workflow DSL forms",
		"Supported special forms:",
		"Supported builtins:",
		"Minimal valid skeleton",
		"Syntax checklist before workflow_run",
		"(workflow \"auth audit\"",
	} {
		if contains(workflowPrompt, unwanted) {
			t.Errorf("expected workflow prompt to omit detailed tutorial text %q", unwanted)
		}
	}
	if contains(workflowPrompt, "JSON DSL") {
		t.Error("expected workflow prompt to avoid JSON DSL negative guidance")
	}
}

// --- stripImageContent tests ---

func TestStripImageContent(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "hello"},
		{Role: "toolResult", ToolName: "read", Contents: []provider.ContentBlock{
			{Type: "text", Text: "[Image file: test.png]"},
			{Type: "image", Image: &provider.ImageContent{MimeType: "image/png", Data: "base64data"}},
		}},
		{Role: "assistant", Contents: []provider.ContentBlock{
			{Type: "text", Text: "I see the image"},
		}},
	}

	result := stripImageContent(messages)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// Second message should have image stripped
	if len(result[1].Contents) != 1 {
		t.Errorf("expected 1 content block after stripping, got %d", len(result[1].Contents))
	}
	if result[1].Contents[0].Type == "image" {
		t.Error("image content should have been stripped")
	}
}

func TestStripImageContentOnlyImage(t *testing.T) {
	messages := []provider.Message{
		{Role: "user", Content: "hello"},
		{Role: "toolResult", ToolName: "read", Contents: []provider.ContentBlock{
			{Type: "image", Image: &provider.ImageContent{MimeType: "image/png", Data: "base64data"}},
		}},
	}

	result := stripImageContent(messages)
	// Message with only image and no text should be skipped
	if len(result) != 1 {
		t.Fatalf("expected 1 message (image-only skipped), got %d", len(result))
	}
}

func TestSupportsImages(t *testing.T) {
	a := &Agent{config: AgentLoopConfig{}}
	a.config.Model = &provider.Model{Input: []string{"text"}}
	if a.supportsImages() {
		t.Error("expected false for text-only model")
	}

	a.config.Model = &provider.Model{Input: []string{"text", "image"}}
	if !a.supportsImages() {
		t.Error("expected true for text+image model")
	}

	a.config.Model = nil
	if a.supportsImages() {
		t.Error("expected false for nil model")
	}
}

func TestFormatToolListWithSnippets(t *testing.T) {
	// Test with tools and snippets
	tools := []string{"read", "write", "bash"}
	snippets := map[string]string{"read": "Read a file", "write": "Write a file"}
	list := formatToolListWithSnippets(tools, snippets)

	if !contains(list, "read") {
		t.Error("expected list to contain 'read'")
	}

	if !contains(list, "Read a file") {
		t.Error("expected list to contain snippet")
	}

	// Test empty
	emptyList := formatToolListWithSnippets(nil, nil)
	if emptyList != "(none)" {
		t.Errorf("expected empty list to say '(none)', got %q", emptyList)
	}
}

func TestBuildSkillsContext(t *testing.T) {
	skills := []SkillInfo{
		{Name: "test", Description: "Test skill", Path: "/path/to/skill"},
	}

	context := BuildSkillsContext(skills)

	if context == "" {
		t.Fatal("expected non-empty context")
	}

	if !contains(context, "test") {
		t.Error("expected context to contain skill name")
	}

	// Test empty
	emptyContext := BuildSkillsContext(nil)
	if emptyContext != "" {
		t.Error("expected empty context for nil skills")
	}
}

func TestBuildContextFilesContext(t *testing.T) {
	files := []ContextFileInfo{
		{Name: "AGENTS.md", Path: "/path", Scope: "project", Content: "# Test"},
	}

	context := BuildContextFilesContext(files)

	if context == "" {
		t.Fatal("expected non-empty context")
	}

	if !contains(context, "AGENTS.md") {
		t.Error("expected context to contain file name")
	}

	// Test empty
	emptyContext := BuildContextFilesContext(nil)
	if emptyContext != "" {
		t.Error("expected empty context for nil files")
	}
}

func TestBaseProvider(t *testing.T) {
	models := []*provider.Model{
		{ID: "model1", Name: "Model 1"},
		{ID: "model2", Name: "Model 2"},
	}

	p := NewBaseProvider("test", models)

	if p.Name() != "test" {
		t.Errorf("expected name 'test', got '%s'", p.Name())
	}

	if len(p.Models()) != 2 {
		t.Errorf("expected 2 models, got %d", len(p.Models()))
	}

	m := p.GetModel("model1")
	if m == nil {
		t.Fatal("expected model, got nil")
	}

	if m.Name != "Model 1" {
		t.Errorf("expected name 'Model 1', got '%s'", m.Name)
	}

	m = p.GetModel("nonexistent")
	if m != nil {
		t.Error("expected nil for nonexistent model")
	}
}

// --- ContextWithAgentID tests ---

func TestContextWithAgentID(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithAgentID(ctx, "test-agent")

	id, ok := AgentIDFromContext(ctx)
	if !ok {
		t.Fatal("expected agent ID in context")
	}
	if id != "test-agent" {
		t.Errorf("agent ID = %q, want 'test-agent'", id)
	}

	// Missing from context
	_, ok = AgentIDFromContext(context.Background())
	if ok {
		t.Error("expected no agent ID in empty context")
	}
}

func TestContextWithEventChan(t *testing.T) {
	ch := make(chan Event, 1)
	ctx := ContextWithEventChan(context.Background(), ch)

	got, ok := EventChanFromContext(ctx)
	if !ok {
		t.Fatal("expected event chan in context")
	}
	if got == nil {
		t.Fatal("expected non-nil event chan")
	}

	_, ok = EventChanFromContext(context.Background())
	if ok {
		t.Error("expected no event chan in empty context")
	}
}

func TestContextWithParentRunContext(t *testing.T) {
	parent := context.Background()
	ctx := ContextWithParentRunContext(context.Background(), parent)

	got, ok := ParentRunContextFromContext(ctx)
	if !ok {
		t.Fatal("expected parent run context")
	}
	if got != parent {
		t.Fatal("unexpected parent run context")
	}

	_, ok = ParentRunContextFromContext(context.Background())
	if ok {
		t.Error("expected no parent run context in empty context")
	}
}

// --- Manager status tests ---

func TestAgentManagerMarkRunning(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	m.Create(AgentOptions{ID: "a1"})
	m.MarkRunning("a1")
	st, ok := m.Status("a1")
	if !ok {
		t.Fatal("expected status")
	}
	if st.State != "running" {
		t.Errorf("state = %q, want running", st.State)
	}
}

func TestAgentManagerMarkDone(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	m.Create(AgentOptions{ID: "a1"})
	m.MarkDone("a1", "completed")
	st, _ := m.Status("a1")
	if st.State != "done" {
		t.Errorf("state = %q, want done", st.State)
	}
	if st.Result != "completed" {
		t.Errorf("result = %q, want completed", st.Result)
	}
}

func TestAgentManagerMarkError(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	m.Create(AgentOptions{ID: "a1"})
	m.MarkError("a1", fmt.Errorf("test error"))
	st, _ := m.Status("a1")
	if st.State != "error" {
		t.Errorf("state = %q, want error", st.State)
	}
	if st.Error != "test error" {
		t.Errorf("error = %q, want 'test error'", st.Error)
	}
}

func TestAgentManagerMarkErrorNil(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	m.Create(AgentOptions{ID: "a1"})
	m.MarkError("a1", nil)
	st, _ := m.Status("a1")
	if st.Error != "" {
		t.Errorf("error = %q, want empty", st.Error)
	}
}

func TestAgentManagerRegister(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	// Create an agent through factory to get a valid agentpkg.Agent
	a, _ := m.Create(AgentOptions{ID: "parent"})
	m.Destroy("parent")
	// Re-register
	m.Register(a)
	if m.Count() != 1 {
		t.Errorf("count = %d, want 1", m.Count())
	}
}

func TestAgentManagerRegisterNil(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	m.Register(nil) // Should not panic
	if m.Count() != 0 {
		t.Errorf("count = %d, want 0", m.Count())
	}
}

func TestAgentManagerStatusNotFound(t *testing.T) {
	m := NewAgentManager(&AgentFactory{})
	_, ok := m.Status("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- ForceCompact tests ---

func TestSetForceCompact_ShouldCompactReturnsTrue(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1", ContextWindow: 100000},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
		CompactionSettings: ctxpkg.CompactionSettings{
			KeepRecentTokens: 1,
		},
	}

	a := New(cfg, registry)

	// Load some messages so there's something to compact
	a.LoadHistoryMessages([]provider.Message{
		provider.NewUserMessage("Hello"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "Hi there"}}),
		provider.NewUserMessage("Second turn"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "Second response"}}),
	})

	// Without force, ShouldCompact should be false (context is tiny)
	if a.ShouldCompact() {
		t.Fatal("ShouldCompact should be false without force and small context")
	}

	// Set force flag
	a.SetForceCompact()

	// Now ShouldCompact should return true (force flag set)
	if !a.ShouldCompact() {
		t.Fatal("ShouldCompact should be true after SetForceCompact")
	}

	// Force flag is consumed — second call should return false
	if a.ShouldCompact() {
		t.Fatal("ShouldCompact should be false after force flag was consumed")
	}
}

func TestSetForceCompact_NoMessagesDoesNotForce(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1", ContextWindow: 100000},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
	}

	a := New(cfg, registry)

	// No messages loaded — force should not trigger (nothing to compact)
	a.SetForceCompact()
	if a.ShouldCompact() {
		t.Fatal("ShouldCompact should be false with force but no messages")
	}
}

func TestShouldCompact_OverThresholdButNoCompactableMessages(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1", ContextWindow: 100},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    mockProvider.Models()[0],
		Mode:     "agent",
		CompactionSettings: ctxpkg.CompactionSettings{
			Enabled:          true,
			ReserveTokens:    10,
			KeepRecentTokens: 20,
		},
	}

	a := New(cfg, registry)
	a.LoadHistoryMessages([]provider.Message{
		provider.NewUserMessage("current request"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: strings.Repeat("x", 500)}}),
	})

	if a.ShouldCompact() {
		t.Fatal("ShouldCompact should be false when only the current turn would be kept")
	}
}

func TestCompactClearsKeptMessageUsage(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")
	sess := session.New(tmpDir, sessionDir)
	if err := sess.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	p := newCompactionReplayProvider()
	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(tmpDir, sb)
	a := New(Config{
		Provider: p,
		Model:    p.models[0],
		Mode:     "agent",
		Session:  sess,
		CompactionSettings: ctxpkg.CompactionSettings{
			Enabled:          true,
			ReserveTokens:    256,
			KeepRecentTokens: 1,
		},
	}, registry)

	history := []provider.Message{
		provider.NewUserMessage("old user context"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "old assistant context"}}),
		provider.NewUserMessage("recent user context"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "recent assistant context"}}),
	}
	history[3].Usage = &provider.Usage{Input: 1000, Output: 50, TotalTokens: 1050}
	for _, msg := range history {
		if _, err := sess.AppendMessage(msg); err != nil {
			t.Fatalf("AppendMessage() error = %v", err)
		}
	}
	a.LoadHistoryState(sess.GetReplayState().Messages, sess.GetReplayState().EntryIDs)

	eventCh := make(chan Event, 32)
	go func() {
		defer close(eventCh)
		if err := a.Compact(context.Background(), eventCh); err != nil {
			t.Errorf("Compact() error = %v", err)
		}
	}()
	for range eventCh {
	}

	messages := a.GetMessages()
	if len(messages) != 3 {
		t.Fatalf("messages len = %d, want 3", len(messages))
	}
	if messages[2].Usage != nil {
		t.Fatalf("kept assistant usage = %#v, want nil stale usage", messages[2].Usage)
	}
	usage := a.GetContextUsage()
	if usage == nil {
		t.Fatal("expected context usage")
	}
	if usage.Tokens >= 1050 {
		t.Fatalf("context usage tokens = %d, still using stale assistant usage", usage.Tokens)
	}
}

func TestSetForceCompact_NoModelDoesNotForce(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{
		{ID: "model1", Name: "Model 1"},
	}, nil)

	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	cfg := Config{
		Provider: mockProvider,
		Model:    nil, // no model
		Mode:     "agent",
	}

	a := New(cfg, registry)
	a.LoadHistoryMessages([]provider.Message{
		provider.NewUserMessage("Hello"),
		provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: "Hi"}}),
	})

	a.SetForceCompact()
	if a.ShouldCompact() {
		t.Fatal("ShouldCompact should be false with force but no model")
	}
}

func TestCompactionReplayPersistsAcrossSessionReload(t *testing.T) {
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions")
	sess := session.New(tmpDir, sessionDir)
	if err := sess.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	p := newCompactionReplayProvider()
	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(tmpDir, sb)
	a := New(Config{
		Provider: p,
		Model:    p.models[0],
		Mode:     "agent",
		Session:  sess,
		CompactionSettings: ctxpkg.CompactionSettings{
			Enabled:          true,
			ReserveTokens:    256,
			KeepRecentTokens: 48,
		},
	}, registry)

	oldUser := provider.NewUserMessage(strings.Repeat("old user ", 24))
	oldAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: strings.Repeat("old assistant ", 24)}})
	recentUser := provider.NewUserMessage(strings.Repeat("recent user ", 20))
	recentAssistant := provider.NewAssistantMessage([]provider.ContentBlock{{Type: "text", Text: strings.Repeat("recent assistant ", 20)}})
	history := []provider.Message{oldUser, oldAssistant, recentUser, recentAssistant}
	for _, msg := range history {
		if _, err := sess.AppendMessage(msg); err != nil {
			t.Fatalf("AppendMessage() error = %v", err)
		}
	}
	a.LoadHistoryState(sess.GetReplayState().Messages, sess.GetReplayState().EntryIDs)

	eventCh := make(chan Event, 32)
	go func() {
		defer close(eventCh)
		if err := a.Compact(context.Background(), eventCh); err != nil {
			t.Errorf("Compact() error = %v", err)
		}
	}()
	for range eventCh {
	}

	reopened, err := session.Open(sess.GetFile())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	replay := reopened.GetReplayState()

	a2 := New(Config{
		Provider: p,
		Model:    p.models[0],
		Mode:     "agent",
		Session:  reopened,
		CompactionSettings: ctxpkg.CompactionSettings{
			Enabled:          true,
			ReserveTokens:    256,
			KeepRecentTokens: 48,
		},
	}, registry)
	a2.LoadHistoryState(replay.Messages, replay.EntryIDs)

	runCh := a2.Run(context.Background(), "next step")
	for range runCh {
	}

	if len(p.calls) != 2 {
		t.Fatalf("provider call count = %d, want 2", len(p.calls))
	}

	continued := p.calls[1].Messages
	if len(continued) < 4 {
		t.Fatalf("continued call messages = %d, want at least 4", len(continued))
	}

	foundSummary := false
	foundOldUser := false
	foundRecentUser := false
	for _, msg := range continued {
		if msg.SystemInjected && msg.Content == "## Goal\ncheckpoint" {
			foundSummary = true
		}
		if msg.Content == oldUser.Content {
			foundOldUser = true
		}
		if msg.Content == recentUser.Content {
			foundRecentUser = true
		}
	}

	if !foundSummary {
		t.Fatal("continued run did not include compacted summary")
	}
	if foundOldUser {
		t.Fatal("continued run still included pre-compaction old user message")
	}
	if !foundRecentUser {
		t.Fatal("continued run lost recent user message after compaction replay")
	}
}

// --- MaxConsecutiveNoText tests ---

func TestMaxConsecutiveNoText_Default(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{{ID: "m1", Name: "M1"}}, nil)
	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	a := NewWithLoopConfig(AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
		},
	}, registry)

	// Default MaxConsecutiveNoText should be 200 (MaxIterations default)
	// but the threshold is 95. Verify the config field is 0 (uses default).
	if a.config.MaxConsecutiveNoText != 0 {
		t.Fatalf("expected default MaxConsecutiveNoText=0, got %d", a.config.MaxConsecutiveNoText)
	}
}

func TestMaxConsecutiveNoText_Custom(t *testing.T) {
	mockProvider := provider.NewMockProvider("mock", []*provider.Model{{ID: "m1", Name: "M1"}}, nil)
	sb := sandbox.NewNoneSandbox()
	registry := tools.NewRegistry(t.TempDir(), sb)

	a := NewWithLoopConfig(AgentLoopConfig{
		Config: Config{
			Provider: mockProvider,
			Model:    mockProvider.Models()[0],
			Mode:     "agent",
		},
		MaxConsecutiveNoText: 10,
	}, registry)

	if a.config.MaxConsecutiveNoText != 10 {
		t.Fatalf("expected MaxConsecutiveNoText=10, got %d", a.config.MaxConsecutiveNoText)
	}
}
