package context

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/util"
)

const defaultMaxCompactionSummaryTokens = 4096

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// CompactionSettings holds compaction configuration.
type CompactionSettings struct {
	Enabled          bool   `json:"enabled"`
	ReserveTokens    int    `json:"reserveTokens"`
	KeepRecentTokens int    `json:"keepRecentTokens"`
	Tokenizer        string `json:"tokenizer,omitempty"`
	TokenizerModel   string `json:"tokenizerModel,omitempty"`
	Template         string `json:"template,omitempty"`

	// Idle compression settings (R5.1-R5.5)
	// When enabled, triggers compression during idle periods to maintain cache warmth.
	IdleCompressionEnabled   bool `json:"idleCompressionEnabled,omitempty"`   // R5.1: default off
	IdleTimeoutSeconds       int  `json:"idleTimeoutSeconds,omitempty"`       // seconds of inactivity before triggering (default: 90)
	IdleMinTokensForCompress int  `json:"idleMinTokensForCompress,omitempty"` // minimum tokens to trigger idle compression (default: 150000)
}

// DefaultCompactionSettings returns default compaction settings.
func DefaultCompactionSettings() CompactionSettings {
	return CompactionSettings{
		Enabled:                  true,
		ReserveTokens:            16384,
		KeepRecentTokens:         20000,
		IdleCompressionEnabled:   false,  // R5.1: off by default
		IdleTimeoutSeconds:       90,     // R5.2: 90 seconds
		IdleMinTokensForCompress: 150000, // R5.4: 150k tokens minimum
	}
}

// NormalizeCompactionSettings applies runtime defaults for zero-valued limits.
func NormalizeCompactionSettings(settings CompactionSettings) CompactionSettings {
	defaults := DefaultCompactionSettings()
	if settings.ReserveTokens == 0 {
		settings.ReserveTokens = defaults.ReserveTokens
	}
	if settings.KeepRecentTokens == 0 {
		settings.KeepRecentTokens = defaults.KeepRecentTokens
	}
	if settings.IdleTimeoutSeconds == 0 {
		settings.IdleTimeoutSeconds = defaults.IdleTimeoutSeconds
	}
	if settings.IdleMinTokensForCompress == 0 {
		settings.IdleMinTokensForCompress = defaults.IdleMinTokensForCompress
	}
	return settings
}

func compactionSummaryMaxTokens(settings CompactionSettings, model *provider.Model) int {
	maxTokens := int(float64(settings.ReserveTokens) * 0.8)
	if maxTokens <= 0 || maxTokens > defaultMaxCompactionSummaryTokens {
		maxTokens = defaultMaxCompactionSummaryTokens
	}
	if model != nil && model.MaxTokens > 0 && maxTokens > model.MaxTokens {
		maxTokens = model.MaxTokens
	}
	return maxTokens
}

// CompactionResult holds the result of a compaction operation.
type CompactionResult struct {
	Summary        string
	FirstKeptIndex int
	TokensBefore   int
}

// CutPointResult holds information about where to cut the conversation.
type CutPointResult struct {
	FirstKeptIndex int
	TurnStartIndex int
	IsSplitTurn    bool
}

// FindValidCutPoints finds valid cut points in messages.
// Valid cut points are user, assistant messages (never tool results).
func FindValidCutPoints(messages []provider.Message, startIndex, endIndex int) []int {
	var cutPoints []int
	for i := startIndex; i < endIndex && i < len(messages); i++ {
		msg := messages[i]
		switch msg.Role {
		case "user", "assistant":
			cutPoints = append(cutPoints, i)
		case "toolResult":
			// Never cut at tool results
			continue
		}
	}
	return cutPoints
}

// FindTurnStartIndex finds the user message that starts the turn containing the given index.
func FindTurnStartIndex(messages []provider.Message, entryIndex, startIndex int) int {
	for i := entryIndex; i >= startIndex; i-- {
		if messages[i].Role == "user" {
			return i
		}
	}
	return -1
}

// FindCutPoint finds the cut point that keeps approximately keepRecentTokens.
func FindCutPoint(messages []provider.Message, startIndex, endIndex, keepRecentTokens int) CutPointResult {
	return FindCutPointWithEstimator(messages, startIndex, endIndex, keepRecentTokens, GenericTokenEstimator{})
}

// FindCutPointWithEstimator finds the cut point using the supplied token estimator.
func FindCutPointWithEstimator(messages []provider.Message, startIndex, endIndex, keepRecentTokens int, estimator TokenEstimator) CutPointResult {
	if estimator == nil {
		estimator = GenericTokenEstimator{}
	}
	cutPoints := FindValidCutPoints(messages, startIndex, endIndex)

	if len(cutPoints) == 0 {
		return CutPointResult{FirstKeptIndex: startIndex, TurnStartIndex: -1, IsSplitTurn: false}
	}

	// Walk backwards from newest, accumulating estimated message sizes
	accumulatedTokens := 0
	cutIndex := cutPoints[0] // Default: keep from first message

	for i := endIndex - 1; i >= startIndex; i-- {
		messageTokens := estimator.EstimateTokens(messages[i])
		accumulatedTokens += messageTokens

		if accumulatedTokens >= keepRecentTokens {
			// Find the closest valid cut point to this entry
			bestCut := cutPoints[0]
			bestDist := abs(bestCut - i)
			for _, c := range cutPoints {
				dist := abs(c - i)
				if dist < bestDist {
					bestDist = dist
					bestCut = c
				}
			}
			cutIndex = bestCut
			break
		}
	}

	// Determine if this is a split turn
	isUserMessage := messages[cutIndex].Role == "user"
	turnStartIndex := -1
	if !isUserMessage {
		turnStartIndex = FindTurnStartIndex(messages, cutIndex, startIndex)
	}

	return CutPointResult{
		FirstKeptIndex: cutIndex,
		TurnStartIndex: turnStartIndex,
		IsSplitTurn:    !isUserMessage && turnStartIndex != -1,
	}
}

func messagesToSummarizeForCompaction(messages []provider.Message, settings CompactionSettings, estimator TokenEstimator, previousSummary string) ([]provider.Message, CutPointResult) {
	cutPoint := FindCutPointWithEstimator(messages, 0, len(messages), settings.KeepRecentTokens, estimator)

	messagesToSummarize := messages[:cutPoint.FirstKeptIndex]
	if cutPoint.IsSplitTurn && cutPoint.TurnStartIndex >= 0 {
		messagesToSummarize = messages[:cutPoint.TurnStartIndex]
	}
	messagesToSummarize = stripLeadingPreviousSummary(messagesToSummarize, previousSummary)

	return messagesToSummarize, cutPoint
}

// HasCompactableMessages reports whether compaction would have older messages
// to summarize after preserving the configured recent context.
func HasCompactableMessages(messages []provider.Message, model *provider.Model, settings CompactionSettings, previousSummary string) bool {
	if len(messages) == 0 {
		return false
	}
	estimator := ResolveTokenEstimator(settings, model)
	messagesToSummarize, _ := messagesToSummarizeForCompaction(messages, settings, estimator, previousSummary)
	return len(messagesToSummarize) > 0
}

// SerializeConversation serializes messages to text for summarization.
func SerializeConversation(messages []provider.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		// Skip system-injected messages
		if msg.SystemInjected {
			continue
		}

		switch msg.Role {
		case "user":
			content := msg.Content
			if content == "" {
				content = serializeContentBlocks(msg.Contents)
			}
			sb.WriteString(fmt.Sprintf("User: %s\n\n", content))

		case "assistant":
			sb.WriteString("Assistant: ")
			content := msg.Content
			if content == "" {
				content = serializeTextBlocks(msg.Contents)
			}
			sb.WriteString(content)
			for _, block := range msg.Contents {
				switch block.Type {
				case "thinking":
					sb.WriteString(fmt.Sprintf("[thinking: %s]", block.Thinking))
				case "toolCall":
					if block.ToolCall != nil {
						sb.WriteString(fmt.Sprintf("[tool_call: %s(%s)]", block.ToolCall.Name, string(block.ToolCall.Arguments)))
					}
				}
			}
			sb.WriteString("\n\n")

		case "toolResult":
			content := msg.Content
			if content == "" {
				content = serializeContentBlocks(msg.Contents)
			}
			sb.WriteString(fmt.Sprintf("Tool Result [%s]: %s\n\n", msg.ToolName, truncateString(content, 500)))
		}
	}

	return sb.String()
}

func serializeTextBlocks(blocks []provider.ContentBlock) string {
	var sb strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String()
}

func serializeContentBlocks(blocks []provider.ContentBlock) string {
	var parts []string
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				parts = append(parts, block.Text)
			}
		case "image":
			if block.Image != nil {
				parts = append(parts, fmt.Sprintf("[image: %s]", block.Image.MimeType))
			} else {
				parts = append(parts, "[image]")
			}
		case "thinking":
			parts = append(parts, fmt.Sprintf("[thinking: %s]", block.Thinking))
		case "toolCall":
			if block.ToolCall != nil {
				parts = append(parts, fmt.Sprintf("[tool_call: %s(%s)]", block.ToolCall.Name, string(block.ToolCall.Arguments)))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func truncateString(s string, maxLen int) string {
	return util.TruncateWithSuffix(s, maxLen, "...")
}

// CompressionTemplate contains instructions for initial and update compaction.
type CompressionTemplate struct {
	Name              string
	Instruction       string
	UpdateInstruction string
}

// defaultCompressionInstruction is injected into the conversation for Insert-then-Compress.
// This implements Rule R4.2: the compression instruction is a system_injected message.
const defaultCompressionInstruction = `Please create a structured context checkpoint summary of our conversation so far.

Use this EXACT format:

## Goal
[What is the user trying to accomplish?]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- Or "(none)" if none were mentioned

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- Or "(none)" if not applicable

Keep each section concise. Preserve exact file paths, function names, and error messages.`

// defaultUpdateCompressionInstruction is used when there's an existing summary to update.
const defaultUpdateCompressionInstruction = `Please update the existing summary with new information from our conversation.

<existing-summary>
%s
</existing-summary>

RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use the same EXACT format as the existing summary.`

const codeCompressionInstruction = `Please create a structured coding checkpoint summary of our work so far.

Use this EXACT format:

## Goal
[What coding task is being solved?]

## Constraints & Preferences
- [User requirements, repo conventions, safety constraints]
- Or "(none)" if none were mentioned

## Code Changes
### Done
- [x] [Completed code or docs changes with exact file paths]

### In Progress
- [ ] [Current implementation or review state]

### Not Started
- [ ] [Planned but untouched follow-up work]

## Technical Decisions
- **[Decision]**: [Reason and affected files/functions]

## Verification
- [Commands run and results]
- [Commands not run and why]

## Next Steps
1. [Ordered continuation steps]

## Critical Context
- [Exact file paths, function names, errors, API contracts, or examples needed to continue]
- Or "(none)" if not applicable

Keep each section concise. Preserve exact file paths, function names, commands, and error messages.`

const codeUpdateCompressionInstruction = `Please update the existing coding checkpoint summary with new information from our conversation.

<existing-summary>
%s
</existing-summary>

RULES:
- PRESERVE all still-relevant code changes, file paths, function names, commands, and error messages
- ADD new implementation progress, verification results, and technical decisions
- UPDATE Done/In Progress/Not Started based on what changed
- REMOVE stale next steps only when they are clearly completed or no longer relevant
- Keep the same EXACT format as the existing summary.`

const conversationCompressionInstruction = `Please create a concise conversation checkpoint summary of our conversation so far.

Use this EXACT format:

## Objective
[What the user wants]

## Preferences
- [User preferences or constraints]
- Or "(none)" if none were mentioned

## Discussion So Far
- [Important points and outcomes]

## Decisions
- **[Decision]**: [Brief rationale]

## Open Items
1. [What remains unresolved or should happen next]

## Critical Details
- [Exact names, values, references, or examples needed to continue]
- Or "(none)" if not applicable

Keep it concise and preserve exact identifiers, paths, commands, and error messages.`

const conversationUpdateCompressionInstruction = `Please update the existing conversation checkpoint summary with new information.

<existing-summary>
%s
</existing-summary>

RULES:
- PRESERVE still-relevant objective, preferences, decisions, and critical details
- ADD new outcomes and open items from the new messages
- UPDATE stale open items when completed
- Keep the same EXACT format as the existing summary.`

// ResolveCompressionTemplate returns a built-in compression template.
func ResolveCompressionTemplate(name string) CompressionTemplate {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "code":
		return CompressionTemplate{
			Name:              "code",
			Instruction:       codeCompressionInstruction,
			UpdateInstruction: codeUpdateCompressionInstruction,
		}
	case "conversation":
		return CompressionTemplate{
			Name:              "conversation",
			Instruction:       conversationCompressionInstruction,
			UpdateInstruction: conversationUpdateCompressionInstruction,
		}
	default:
		return CompressionTemplate{
			Name:              "default",
			Instruction:       defaultCompressionInstruction,
			UpdateInstruction: defaultUpdateCompressionInstruction,
		}
	}
}

// GenerateSummaryInsertThenCompress generates a summary using Insert-then-Compress pattern.
// This implements Rule R4.1-R4.2: use the SAME system prompt and tools, not a separate call.
// The compression instruction is injected as a system_injected user message at the end of the conversation.
func GenerateSummaryInsertThenCompress(
	ctx context.Context,
	messages []provider.Message,
	p provider.Provider,
	model *provider.Model,
	systemPrompt string,
	tools []provider.ToolDefinition,
	previousSummary string,
	maxTokens int,
) (string, error) {
	return GenerateSummaryInsertThenCompressWithTemplate(
		ctx, messages, p, model, systemPrompt, tools,
		previousSummary, maxTokens, ResolveCompressionTemplate(""),
	)
}

// GenerateSummaryInsertThenCompressWithTemplate generates a summary using the
// supplied compression template.
func GenerateSummaryInsertThenCompressWithTemplate(
	ctx context.Context,
	messages []provider.Message,
	p provider.Provider,
	model *provider.Model,
	systemPrompt string,
	tools []provider.ToolDefinition,
	previousSummary string,
	maxTokens int,
	template CompressionTemplate,
) (string, error) {
	if template.Instruction == "" || template.UpdateInstruction == "" {
		template = ResolveCompressionTemplate("")
	}

	// Build compression instruction
	var instruction string
	if previousSummary != "" {
		instruction = fmt.Sprintf(template.UpdateInstruction, previousSummary)
	} else {
		instruction = template.Instruction
	}

	// Create the compression instruction message (system_injected)
	compressionMsg := provider.NewSystemInjectedUserMessage(instruction)

	// Build messages: original conversation + compression instruction
	// The LLM sees the full conversation and responds with a summary
	compactionMessages := make([]provider.Message, 0, len(messages)+1)
	compactionMessages = append(compactionMessages, messages...)
	compactionMessages = append(compactionMessages, compressionMsg)

	// Use the SAME system prompt and tools (R4.1: no separate LLM call with different prompt)
	params := provider.ChatParams{
		Messages:     compactionMessages,
		Tools:        tools,
		SystemPrompt: systemPrompt,
		MaxTokens:    maxTokens,
	}
	if model != nil {
		params.ModelID = model.ID
	}

	// Call LLM to generate summary
	streamCh := p.Chat(ctx, params)

	var summary strings.Builder
	for event := range streamCh {
		switch event.Type {
		case provider.StreamTextDelta:
			summary.WriteString(event.TextDelta)
		case provider.StreamError:
			if event.Error != nil {
				if errors.Is(event.Error, context.Canceled) || errors.Is(event.Error, context.DeadlineExceeded) {
					return "", event.Error
				}
				return "", fmt.Errorf("summarization failed: %w", event.Error)
			}
		}
	}

	result := strings.TrimSpace(summary.String())
	if result == "" {
		return "", fmt.Errorf("summarization returned empty result")
	}

	return result, nil
}

// GenerateSummary is the legacy interface that delegates to Insert-then-Compress.
// Kept for backward compatibility but now uses the same system prompt.
// Deprecated: use GenerateSummaryInsertThenCompress directly.
func GenerateSummary(
	ctx context.Context,
	messages []provider.Message,
	p provider.Provider,
	model *provider.Model,
	reserveTokens int,
	previousSummary string,
) (string, error) {
	maxTokens := compactionSummaryMaxTokens(CompactionSettings{ReserveTokens: reserveTokens}, model)

	// Use empty system prompt and tools - this is the legacy path
	// The caller should migrate to GenerateSummaryInsertThenCompress
	return GenerateSummaryInsertThenCompress(ctx, messages, p, model, "", nil, previousSummary, maxTokens)
}

// Compact performs context compaction on the messages using Insert-then-Compress pattern.
// This implements Rule R4.1-R4.4.
func Compact(
	ctx context.Context,
	messages []provider.Message,
	p provider.Provider,
	model *provider.Model,
	systemPrompt string,
	tools []provider.ToolDefinition,
	settings CompactionSettings,
	previousSummary string,
) (*CompactionResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to compact")
	}

	estimator := ResolveTokenEstimator(settings, model)
	tokensBefore := estimator.EstimateMessagesTokens(messages)

	// Find cut point - keep recent messages, summarize older ones
	messagesToSummarize, cutPoint := messagesToSummarizeForCompaction(messages, settings, estimator, previousSummary)

	if len(messagesToSummarize) == 0 {
		return nil, fmt.Errorf("nothing to compact")
	}

	// Calculate max tokens for summary
	maxTokens := compactionSummaryMaxTokens(settings, model)

	// Generate summary using Insert-then-Compress (R4.1-R4.2)
	summary, err := GenerateSummaryInsertThenCompressWithTemplate(
		ctx, messagesToSummarize, p, model,
		systemPrompt, tools,
		previousSummary, maxTokens,
		ResolveCompressionTemplate(settings.Template),
	)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
	}

	// When IsSplitTurn is true, messagesToSummarize was truncated to TurnStartIndex,
	// so FirstKeptIndex must reflect TurnStartIndex to avoid silently dropping messages.
	firstKept := cutPoint.FirstKeptIndex
	if cutPoint.IsSplitTurn && cutPoint.TurnStartIndex >= 0 {
		firstKept = cutPoint.TurnStartIndex
	}

	return &CompactionResult{
		Summary:        summary,
		FirstKeptIndex: firstKept,
		TokensBefore:   tokensBefore,
	}, nil
}

func stripLeadingPreviousSummary(messages []provider.Message, previousSummary string) []provider.Message {
	if previousSummary == "" || len(messages) == 0 {
		return messages
	}
	first := messages[0]
	if first.SystemInjected && first.Role == "user" && first.Content == previousSummary {
		return messages[1:]
	}
	return messages
}

// CompactWithLegacyInterface is a compatibility wrapper that calls the old Compact signature.
// Deprecated: use the new Compact with systemPrompt and tools parameters.
func CompactWithLegacyInterface(
	ctx context.Context,
	messages []provider.Message,
	p provider.Provider,
	model *provider.Model,
	settings CompactionSettings,
	previousSummary string,
) (*CompactionResult, error) {
	return Compact(ctx, messages, p, model, "", nil, settings, previousSummary)
}
