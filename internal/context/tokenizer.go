package context

import (
	"strings"

	"github.com/startvibecoding/vibecoding/internal/imageproc"
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

// ModelAwareTokenEstimator preserves the chars/4 text heuristic while using
// model-family image token formulas when image dimensions are available.
type ModelAwareTokenEstimator struct {
	Model *provider.Model
}

func (e ModelAwareTokenEstimator) EstimateTokens(msg provider.Message) int {
	chars := estimateMessageCharsWithImageEstimator(msg, e.estimateImageChars)
	return (chars + 3) / 4
}

func (e ModelAwareTokenEstimator) EstimateMessagesTokens(messages []provider.Message) int {
	total := 0
	for _, msg := range messages {
		total += e.EstimateTokens(msg)
	}
	return total
}

func (e ModelAwareTokenEstimator) estimateImageChars(image *provider.ImageContent) int {
	if tokens := estimateImageTokensForModel(image, e.Model); tokens > 0 {
		return tokens * 4
	}
	return estimateImageChars(image)
}

// ResolveTokenEstimator returns the configured estimator.
// Unsupported tokenizer names intentionally fall back to generic so existing
// configuration remains tolerant while model-specific estimators are added.
func ResolveTokenEstimator(settings CompactionSettings, model *provider.Model) TokenEstimator {
	switch strings.ToLower(strings.TrimSpace(settings.Tokenizer)) {
	case "", "auto", "generic":
		if model != nil {
			return ModelAwareTokenEstimator{Model: model}
		}
		return GenericTokenEstimator{}
	default:
		if model != nil {
			return ModelAwareTokenEstimator{Model: model}
		}
		return GenericTokenEstimator{}
	}
}

func estimateMessageChars(msg provider.Message) int {
	return estimateMessageCharsWithImageEstimator(msg, estimateImageChars)
}

func estimateMessageCharsWithImageEstimator(msg provider.Message, estimateImage func(*provider.ImageContent) int) int {
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
				chars += estimateImage(block.Image)
			}
		}
	} else if msg.Content != "" {
		chars += len(msg.Content)
	}

	return chars
}

func estimateImageChars(image *provider.ImageContent) int {
	if image != nil && image.Width > 0 && image.Height > 0 {
		tokens := estimateGenericImageTokens(image.Width, image.Height)
		return tokens * 4
	}
	// Preserve the existing minimum visual-token cost and payload-size guard.
	// Provider-specific image estimators can replace this through TokenEstimator
	// without changing callers.
	imageChars := 4800
	if image != nil && len(image.Data) > imageChars {
		imageChars = len(image.Data)
	}
	return imageChars
}

func estimateGenericImageTokens(width, height int) int {
	return ceilDiv(width, 512) * ceilDiv(height, 512) * 800
}

func estimateImageTokensForModel(image *provider.ImageContent, model *provider.Model) int {
	if image == nil || image.Width <= 0 || image.Height <= 0 || model == nil {
		return 0
	}
	family := imageproc.InferFamily(imageproc.Hint{
		ProviderID: model.Provider,
		ModelID:    model.ID,
	})
	switch family {
	case imageproc.FamilyAnthropic, imageproc.FamilyAnthropicBedrock:
		return ceilDiv(image.Width, 28) * ceilDiv(image.Height, 28)
	case imageproc.FamilyGemini:
		return estimateGeminiImageTokens(image.Width, image.Height)
	case imageproc.FamilyOpenAI, imageproc.FamilyGrok:
		return estimateOpenAIImageTokens(image)
	default:
		return 0
	}
}

func estimateGeminiImageTokens(width, height int) int {
	if width <= 384 && height <= 384 {
		return 258
	}
	return ceilDiv(width, 768) * ceilDiv(height, 768) * 258
}

func estimateOpenAIImageTokens(image *provider.ImageContent) int {
	switch strings.ToLower(strings.TrimSpace(image.Detail)) {
	case "fast", "low":
		return 85
	default:
		tiles := ceilDiv(image.Width, 512) * ceilDiv(image.Height, 512)
		return 85 + 170*tiles
	}
}

func ceilDiv(n, d int) int {
	if d <= 0 {
		return 0
	}
	return (n + d - 1) / d
}
