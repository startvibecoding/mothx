package context

import (
	"strings"

	"github.com/startvibecoding/vibecoding/internal/provider"
)

// TokenEstimator estimates the context footprint of provider messages.
type TokenEstimator interface {
	EstimateTokens(msg provider.Message) int
	EstimateMessagesTokens(messages []provider.Message) int
}

// GenericTokenEstimator preserves the existing chars/4 heuristic.
type GenericTokenEstimator struct{}

// EstimateTokens estimates token count for a message using a chars/4 heuristic.
func (GenericTokenEstimator) EstimateTokens(msg provider.Message) int {
	chars := estimateMessageChars(msg)
	return (chars + 3) / 4 // ceil(chars/4)
}

// EstimateMessagesTokens estimates the total token count for messages.
func (e GenericTokenEstimator) EstimateMessagesTokens(messages []provider.Message) int {
	total := 0
	for _, msg := range messages {
		total += e.EstimateTokens(msg)
	}
	return total
}

// ResolveTokenEstimator returns the configured estimator.
// Unsupported tokenizer names intentionally fall back to generic so existing
// configuration remains tolerant while model-specific estimators are added.
func ResolveTokenEstimator(settings CompactionSettings, model *provider.Model) TokenEstimator {
	switch strings.ToLower(strings.TrimSpace(settings.Tokenizer)) {
	case "", "auto", "generic":
		return GenericTokenEstimator{}
	default:
		return GenericTokenEstimator{}
	}
}

func estimateMessageChars(msg provider.Message) int {
	chars := 0

	if len(msg.Contents) > 0 {
		// Rich content blocks take precedence; avoid double-counting with Content.
		for _, block := range msg.Contents {
			switch block.Type {
			case "text":
				chars += len(block.Text)
			case "thinking":
				chars += len(block.Thinking)
			case "toolCall":
				if block.ToolCall != nil {
					chars += len(block.ToolCall.Name)
					chars += len(block.ToolCall.Arguments)
				}
			case "image":
				chars += estimateImageChars(block.Image)
			}
		}
	} else if msg.Content != "" {
		chars += len(msg.Content)
	}

	return chars
}

func estimateImageChars(image *provider.ImageContent) int {
	// Preserve the existing minimum visual-token cost and payload-size guard.
	// Provider-specific image estimators can replace this through TokenEstimator
	// without changing callers.
	imageChars := 4800
	if image != nil && len(image.Data) > imageChars {
		imageChars = len(image.Data)
	}
	return imageChars
}
