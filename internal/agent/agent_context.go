package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	ctxpkg "github.com/startvibecoding/mothx/internal/context"
	"github.com/startvibecoding/mothx/internal/provider"
)

const defaultCompactionTimeout = 5 * time.Minute
const defaultAutoCompactionThreshold = 0.80

// supportsImages checks if the model supports image input.
func (a *Agent) supportsImages() bool {
	if a.config.Model == nil {
		return false
	}
	for _, input := range a.config.Model.Input {
		if input == "image" {
			return true
		}
	}
	return false
}

// stripImageContent removes image content blocks from messages.
// This prevents 404 errors when sending to models that don't support image input.
func stripImageContent(messages []provider.Message) []provider.Message {
	result := make([]provider.Message, 0, len(messages))
	for _, msg := range messages {
		if len(msg.Contents) > 0 {
			var filtered []provider.ContentBlock
			for _, c := range msg.Contents {
				if c.Type != "image" {
					filtered = append(filtered, c)
				}
			}
			if len(filtered) == 0 && msg.Content == "" {
				continue // skip message with only image content and no text
			}
			msg.Contents = filtered
		}
		result = append(result, msg)
	}
	return result
}

func estimateTextTokens(s string) int {
	return (len(s) + 3) / 4
}

func estimateToolDefinitionTokens(tools []provider.ToolDefinition) int {
	if len(tools) == 0 {
		return 0
	}
	data, err := json.Marshal(tools)
	if err != nil {
		return 0
	}
	return estimateTextTokens(string(data))
}

func estimateChatRequestTokens(systemPrompt string, messages []provider.Message, tools []provider.ToolDefinition, estimator ctxpkg.TokenEstimator) int {
	if estimator == nil {
		estimator = ctxpkg.GenericTokenEstimator{}
	}
	total := estimateTextTokens(systemPrompt)
	total += estimateToolDefinitionTokens(tools)
	for _, msg := range messages {
		total += estimator.EstimateTokens(msg)
	}
	return total
}

// buildSessionContextMessage builds the [session context] message with dynamic information.
// This implements Rule R2.3 from LLM_Agent_Cache.md: dynamic info goes into a separate message.
// The message is marked as SystemInjected so cache markers skip it.
func (a *Agent) buildSessionContextMessage() provider.Message {
	modelID := "unknown"
	modelName := "unknown"
	if a.config.Model != nil {
		modelID = a.config.Model.ID
		modelName = a.config.Model.Name
	}

	context := fmt.Sprintf(`[session context]
- Current date: %s
- Model: %s (%s)
- Working directory: %s
- Mode: %s
`,
		time.Now().Format("2006-01-02"),
		modelName,
		modelID,
		a.registry.GetWorkDir(),
		a.config.Mode,
	)

	return provider.NewSystemInjectedUserMessage(context)
}

func (a *Agent) outputReserveTokens() int {
	reserve := a.config.MaxTokens
	if reserve <= 0 {
		reserve = 16384
	}
	if a.config.Model != nil && a.config.Model.ContextWindow > 0 && reserve >= a.config.Model.ContextWindow {
		return a.config.Model.ContextWindow / 2
	}
	return reserve
}

func (a *Agent) requestTokenBudget() (budget int, reserve int, contextWindow int, ok bool) {
	if a.config.Model == nil || a.config.Model.ContextWindow <= 0 {
		return 0, 0, 0, false
	}
	contextWindow = a.config.Model.ContextWindow
	reserve = a.outputReserveTokens()
	budget = contextWindow - reserve
	if budget <= 0 {
		budget = contextWindow / 2
	}
	return budget, reserve, contextWindow, true
}

func (a *Agent) buildRequestMessages(sessionContextMsg provider.Message) []provider.Message {
	a.mu.RLock()
	allMessages := make([]provider.Message, 0, len(a.messages)+1)
	allMessages = append(allMessages, sessionContextMsg)
	allMessages = append(allMessages, a.messages...)
	a.mu.RUnlock()

	if !a.supportsImages() {
		allMessages = stripImageContent(allMessages)
	}
	return allMessages
}

func isContextGuardToolResult(msg provider.Message) bool {
	return msg.Role == "toolResult" && strings.HasPrefix(msg.Content, "[Context guard]")
}

func contextGuardToolResult(msg provider.Message, estimatedTokens, budgetTokens, contextWindow, reserveTokens int) provider.Message {
	toolName := msg.ToolName
	if toolName == "" {
		toolName = "tool"
	}
	content := fmt.Sprintf("[Context guard] The %q tool output was omitted because sending it would exceed the model context window (estimated request: %d tokens; input budget: %d tokens; context window: %d; reserved for output: %d). Retry with a narrower scope: use read with offset/limit, grep/find with path/include/maxResults, or request smaller chunks and summarize incrementally.", toolName, estimatedTokens, budgetTokens, contextWindow, reserveTokens)
	return provider.Message{
		Role:       "toolResult",
		Content:    content,
		ToolCallID: msg.ToolCallID,
		ToolName:   msg.ToolName,
		IsError:    true,
		Timestamp:  msg.Timestamp,
	}
}

func (a *Agent) replaceLargestToolResultForContext(estimatedTokens, budgetTokens, contextWindow, reserveTokens int) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	estimator := ctxpkg.ResolveTokenEstimator(a.config.CompactionSettings, a.config.Model)
	bestIndex := -1
	bestTokens := 0
	for i, msg := range a.messages {
		if msg.Role != "toolResult" || isContextGuardToolResult(msg) {
			continue
		}
		tokens := estimator.EstimateTokens(msg)
		if tokens > bestTokens {
			bestIndex = i
			bestTokens = tokens
		}
	}
	if bestIndex < 0 {
		return "", false
	}

	original := a.messages[bestIndex]
	a.messages[bestIndex] = contextGuardToolResult(original, estimatedTokens, budgetTokens, contextWindow, reserveTokens)
	if a.context != nil {
		if len(a.context.Messages) == len(a.messages) {
			a.context.Messages[bestIndex] = a.messages[bestIndex]
		} else {
			a.context.Messages = a.messages
		}
	}
	return original.ToolName, true
}

func (a *Agent) prepareRequestMessages(sessionContextMsg provider.Message, ch chan<- Event) ([]provider.Message, error) {
	budgetTokens, reserveTokens, contextWindow, ok := a.requestTokenBudget()
	if !ok {
		return a.buildRequestMessages(sessionContextMsg), nil
	}

	estimator := ctxpkg.ResolveTokenEstimator(a.config.CompactionSettings, a.config.Model)
	for attempts := 0; attempts < 16; attempts++ {
		messages := a.buildRequestMessages(sessionContextMsg)
		estimatedTokens := estimateChatRequestTokens(a.frozenSystemPrompt, messages, a.frozenToolDefs, estimator)
		if estimatedTokens <= budgetTokens {
			return messages, nil
		}
		toolName, replaced := a.replaceLargestToolResultForContext(estimatedTokens, budgetTokens, contextWindow, reserveTokens)
		if !replaced {
			return nil, fmt.Errorf("estimated request tokens %d exceed input budget %d for context window %d (reserved output: %d). Narrow the request or reduce context before retrying", estimatedTokens, budgetTokens, contextWindow, reserveTokens)
		}
		if toolName == "" {
			toolName = "tool"
		}
		ch <- Event{Type: EventStatus, StatusMessage: fmt.Sprintf("Context guard omitted oversized %s output; asking model to retry with a narrower scope.", toolName)}
	}

	return nil, fmt.Errorf("estimated request still exceeds context after omitting oversized tool outputs")
}

const contextTokenSafetyMargin = 512

func clampMaxTokensToContext(maxTokens, contextWindow, estimatedInputTokens int) int {
	if maxTokens <= 0 || contextWindow <= 0 || estimatedInputTokens <= 0 {
		return maxTokens
	}
	available := contextWindow - estimatedInputTokens - contextTokenSafetyMargin
	if available < 1 {
		available = 1
	}
	if maxTokens > available {
		return available
	}
	return maxTokens
}

func (a *Agent) maxTokensForRequest(messages []provider.Message) int {
	maxTokens := a.config.MaxTokens
	if a.config.Model == nil || a.config.Model.ContextWindow <= 0 || maxTokens <= 0 {
		return maxTokens
	}
	estimator := ctxpkg.ResolveTokenEstimator(a.config.CompactionSettings, a.config.Model)
	estimatedTokens := estimateChatRequestTokens(a.frozenSystemPrompt, messages, a.frozenToolDefs, estimator)
	return clampMaxTokensToContext(maxTokens, a.config.Model.ContextWindow, estimatedTokens)
}

func selectCacheMarkers(messages []provider.Message) [2]int {
	var markers [2]int
	markers[0] = -1
	markers[1] = -1

	count := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].SystemInjected {
			continue
		}
		if count == 0 {
			markers[1] = i // newest marker
		} else if count == 1 {
			markers[0] = i // second newest marker
			break
		}
		count++
	}
	return markers
}

func applyCacheMarkers(messages []provider.Message, markers [2]int) []provider.Message {
	if markers[0] == -1 && markers[1] == -1 {
		return messages
	}

	// Create a deep copy to avoid modifying the original messages
	result := make([]provider.Message, len(messages))
	for i, msg := range messages {
		result[i] = msg
		// Deep copy Contents slice and pointer fields
		if len(msg.Contents) > 0 {
			result[i].Contents = make([]provider.ContentBlock, len(msg.Contents))
			for j, cb := range msg.Contents {
				result[i].Contents[j] = cb
				if cb.Image != nil {
					imgCopy := *cb.Image
					result[i].Contents[j].Image = &imgCopy
				}
				if cb.ToolCall != nil {
					tcCopy := *cb.ToolCall
					result[i].Contents[j].ToolCall = &tcCopy
				}
				if cb.CacheControl != nil {
					ccCopy := *cb.CacheControl
					result[i].Contents[j].CacheControl = &ccCopy
				}
			}
		}
	}

	for _, idx := range markers {
		if idx < 0 || idx >= len(result) {
			continue
		}
		msg := &result[idx]
		if len(msg.Contents) > 0 {
			// Add cache_control to the last content block
			lastIdx := len(msg.Contents) - 1
			msg.Contents[lastIdx].CacheControl = &provider.CacheControl{Type: "ephemeral"}
		} else if msg.Content != "" {
			// Convert simple text to content blocks with cache_control
			msg.Contents = []provider.ContentBlock{
				{
					Type:         "text",
					Text:         msg.Content,
					CacheControl: &provider.CacheControl{Type: "ephemeral"},
				},
			}
			msg.Content = ""
		}
	}

	return result
}

// GetMessages returns a copy of the current message history.
func (a *Agent) GetMessages() []provider.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]provider.Message, len(a.messages))
	copy(result, a.messages)
	return result
}

// GetHistoryState returns a copy of message history plus aligned session entry IDs.
func (a *Agent) GetHistoryState() ([]provider.Message, []string) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	msgs := make([]provider.Message, len(a.messages))
	copy(msgs, a.messages)
	ids := append([]string(nil), a.messageIDs...)
	return msgs, ids
}

// SetMessages replaces the message history.
func (a *Agent) SetMessages(msgs []provider.Message) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.messages = msgs
	a.messageIDs = make([]string, len(msgs))
	a.context.Messages = msgs
}

// GetContext returns a copy of the current agent context.
func (a *Agent) GetContext() *AgentContext {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.context == nil {
		return nil
	}
	ctx := *a.context
	ctx.Messages = make([]provider.Message, len(a.context.Messages))
	copy(ctx.Messages, a.context.Messages)
	ctx.Tools = make([]provider.ToolDefinition, len(a.context.Tools))
	copy(ctx.Tools, a.context.Tools)
	return &ctx
}

// SetContext replaces the agent context.
func (a *Agent) SetContext(ctx *AgentContext) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.context = ctx
}

// GetContextUsage calculates and returns the current context usage.
func (a *Agent) GetContextUsage() *ctxpkg.ContextUsage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config.Model == nil {
		return nil
	}
	contextWindow := a.config.Model.ContextWindow
	if contextWindow <= 0 {
		return nil
	}

	estimator := ctxpkg.ResolveTokenEstimator(a.config.CompactionSettings, a.config.Model)
	tokens, _ := ctxpkg.EstimateContextTokensWithEstimator(a.messages, estimator)
	percent := float64(tokens) / float64(contextWindow) * 100

	return &ctxpkg.ContextUsage{
		Tokens:        tokens,
		ContextWindow: contextWindow,
		Percent:       &percent,
	}
}

// SetForceCompact marks the agent for forced compaction on the next turn.
func (a *Agent) SetForceCompact() {
	atomic.StoreInt32(&a.forceCompact, 1)
}

func (a *Agent) previousCompactionSummary(messages []provider.Message) string {
	if a.config.Session != nil {
		if compaction, ok := a.config.Session.GetLatestCompaction(); ok {
			return compaction.Summary
		}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].SystemInjected && messages[i].Role == "user" && strings.HasPrefix(messages[i].Content, "## Goal") {
			return messages[i].Content
		}
	}
	return ""
}

// CanCompact reports whether the current conversation has older messages that
// can be summarized while preserving the configured recent context.
func (a *Agent) CanCompact() bool {
	a.mu.RLock()
	model := a.config.Model
	settings := a.config.CompactionSettings
	msgs := make([]provider.Message, len(a.messages))
	copy(msgs, a.messages)
	a.mu.RUnlock()

	if model == nil {
		return false
	}
	return ctxpkg.HasCompactableMessages(msgs, model, settings, a.previousCompactionSummary(msgs))
}

func (a *Agent) canForceCompact() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Model != nil && len(a.messages) > 0
}

func (a *Agent) shouldAutoCompact() bool {
	if !a.config.CompactionSettings.Enabled {
		return false
	}
	if a.config.Model == nil || a.config.Model.ContextWindow <= 0 {
		return false
	}
	sessionContextMsg := a.buildSessionContextMessage()
	messages := a.buildRequestMessages(sessionContextMsg)
	estimator := ctxpkg.ResolveTokenEstimator(a.config.CompactionSettings, a.config.Model)
	tokens := estimateChatRequestTokens(a.frozenSystemPrompt, messages, a.frozenToolDefs, estimator)
	if !ctxpkg.ShouldCompactPercent(tokens, a.config.Model.ContextWindow, defaultAutoCompactionThreshold) {
		return false
	}

	a.mu.RLock()
	msgs := make([]provider.Message, len(a.messages))
	copy(msgs, a.messages)
	model := a.config.Model
	settings := a.config.CompactionSettings
	a.mu.RUnlock()
	return ctxpkg.HasCompactableMessages(msgs, model, settings, a.previousCompactionSummary(msgs))
}

// ShouldCompact checks if compaction should trigger.
// Returns true if context exceeds the threshold OR if forced via SetForceCompact.
func (a *Agent) ShouldCompact() bool {
	if atomic.CompareAndSwapInt32(&a.forceCompact, 1, 0) {
		return a.canForceCompact()
	}
	return a.shouldAutoCompact()
}

func (a *Agent) compactIfNeeded(ctx context.Context, ch chan<- Event) {
	if atomic.CompareAndSwapInt32(&a.forceCompact, 1, 0) {
		if a.canForceCompact() {
			_ = a.CompactForced(ctx, ch)
		}
		return
	}
	if a.shouldAutoCompact() {
		_ = a.Compact(ctx, ch)
	}
}

// Compact performs context compaction using Insert-then-Compress pattern (R4.1-R4.4).
// Uses the SAME system prompt and tools as the main conversation.
func (a *Agent) Compact(ctx context.Context, ch chan<- Event) error {
	return a.compact(ctx, ch, false)
}

// CompactForced performs explicit user-requested compaction. It skips
// preflight compactability checks and allows summary-only checkpoints.
func (a *Agent) CompactForced(ctx context.Context, ch chan<- Event) error {
	return a.compact(ctx, ch, true)
}

func (a *Agent) compact(ctx context.Context, ch chan<- Event, force bool) error {
	if a.config.Model == nil {
		return fmt.Errorf("no model set for compaction")
	}

	compactCtx, cancel := context.WithTimeout(ctx, defaultCompactionTimeout)
	defer cancel()
	go func() {
		select {
		case <-a.abort:
			cancel()
		case <-compactCtx.Done():
		}
	}()

	ch <- Event{Type: EventCompactionStart}

	// Snapshot messages under lock
	a.mu.RLock()
	msgs := make([]provider.Message, len(a.messages))
	copy(msgs, a.messages)
	msgIDs := append([]string(nil), a.messageIDs...)
	a.mu.RUnlock()

	previousSummary := a.previousCompactionSummary(msgs)

	// Use Insert-then-Compress with the SAME system prompt and tools (R4.1)
	result, err := ctxpkg.CompactWithOptions(compactCtx, msgs, a.config.Provider, a.config.Model,
		a.frozenSystemPrompt, a.frozenToolDefs,
		a.config.CompactionSettings, previousSummary,
		ctxpkg.CompactOptions{Force: force})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			ch <- Event{Type: EventCompactionEnd, StatusMessage: "Context compaction canceled", StopReason: "canceled"}
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("compaction timed out after %s: %w", defaultCompactionTimeout, context.DeadlineExceeded)
		}
		ch <- Event{Type: EventCompactionEnd, Error: err}
		return fmt.Errorf("compaction failed: %w", err)
	}

	// Replace messages with summary + kept messages
	// Mark summary as system_injected so cache markers skip it
	firstKeptEntryID := ""
	if result.FirstKeptIndex >= 0 && result.FirstKeptIndex < len(msgIDs) {
		firstKeptEntryID = msgIDs[result.FirstKeptIndex]
	}
	a.mu.Lock()
	summaryMsg := provider.NewSystemInjectedUserMessage(result.Summary)
	keptMessages := cloneMessagesWithoutUsage(msgs[result.FirstKeptIndex:])

	newMessages := make([]provider.Message, 0, 1+len(keptMessages))
	newMessages = append(newMessages, summaryMsg)
	newMessages = append(newMessages, keptMessages...)

	a.messages = newMessages
	a.context.Messages = newMessages

	// Align messageIDs: summary gets empty ID, kept messages keep their IDs
	newIDs := make([]string, 0, 1+len(keptMessages))
	newIDs = append(newIDs, "")
	if result.FirstKeptIndex >= 0 {
		newIDs = append(newIDs, msgIDs[result.FirstKeptIndex:]...)
	}
	a.messageIDs = newIDs
	a.mu.Unlock()

	// Persist compaction to session
	if a.config.Session != nil {
		if _, err := a.config.Session.AppendCompaction(result.Summary, firstKeptEntryID, result.TokensBefore); err != nil {
			// Non-fatal: compaction worked, just couldn't persist the metadata
			ch <- Event{Type: EventStatus, StatusMessage: fmt.Sprintf("Failed to persist compaction: %v", err)}
		}
	}

	ch <- Event{
		Type:          EventCompactionEnd,
		StatusMessage: fmt.Sprintf("Context compacted: %d tokens", result.TokensBefore),
	}

	return nil
}

func (a *Agent) setMessageID(index int, id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index >= 0 && index < len(a.messageIDs) {
		a.messageIDs[index] = id
	}
}
