package agent

import (
	"context"

	"github.com/startvibecoding/vibecoding/internal/config"
	internalprovider "github.com/startvibecoding/vibecoding/internal/provider"
	_ "github.com/startvibecoding/vibecoding/internal/provider/anthropic"
	_ "github.com/startvibecoding/vibecoding/internal/provider/google"
	_ "github.com/startvibecoding/vibecoding/internal/provider/openai"
)

type providerAdapter struct {
	inner internalprovider.Provider
}

func (a *providerAdapter) Chat(ctx context.Context, params ChatParams) <-chan StreamEvent {
	internalParams := internalprovider.ChatParams{
		Messages:      make([]internalprovider.Message, len(params.Messages)),
		Tools:         make([]internalprovider.ToolDefinition, len(params.Tools)),
		SystemPrompt:  params.SystemPrompt,
		ThinkingLevel: internalprovider.ThinkingLevel(params.ThinkingLevel),
		MaxTokens:     params.MaxTokens,
		ModelID:       params.ModelID,
		Abort:         params.Abort,
	}
	for i, m := range params.Messages {
		internalParams.Messages[i] = internalprovider.Message{
			Role:           string(m.Role),
			Content:        m.Content,
			Contents:       make([]internalprovider.ContentBlock, len(m.Contents)),
			IsError:        m.IsError,
			SystemInjected: m.SystemInjected,
			ToolCallID:     m.ToolCallID,
			ToolName:       m.ToolName,
		}
		for j, cb := range m.Contents {
			internalParams.Messages[i].Contents[j] = internalprovider.ContentBlock{
				Type:      cb.Type,
				Text:      cb.Text,
				Thinking:  cb.Thinking,
				Signature: cb.Signature,
			}
			if cb.Image != nil {
				internalParams.Messages[i].Contents[j].Image = &internalprovider.ImageContent{
					MimeType:       cb.Image.MimeType,
					Data:           cb.Image.Data,
					Width:          cb.Image.Width,
					Height:         cb.Image.Height,
					Bytes:          cb.Image.Bytes,
					OriginalWidth:  cb.Image.OriginalWidth,
					OriginalHeight: cb.Image.OriginalHeight,
					OriginalBytes:  cb.Image.OriginalBytes,
					Detail:         cb.Image.Detail,
					Scale:          cb.Image.Scale,
					Cropped:        cb.Image.Cropped,
					CropX:          cb.Image.CropX,
					CropY:          cb.Image.CropY,
					CropWidth:      cb.Image.CropWidth,
					CropHeight:     cb.Image.CropHeight,
				}
			}
			if cb.ToolCall != nil {
				internalParams.Messages[i].Contents[j].ToolCall = &internalprovider.ToolCallBlock{
					ID:               cb.ToolCall.ID,
					Name:             cb.ToolCall.Name,
					Arguments:        cb.ToolCall.Arguments,
					InvalidArguments: cb.ToolCall.InvalidArguments,
					ThoughtSignature: cb.ToolCall.ThoughtSignature,
				}
			}
			if cb.CacheControl != nil {
				internalParams.Messages[i].Contents[j].CacheControl = &internalprovider.CacheControl{Type: cb.CacheControl.Type}
			}
		}
	}
	for i, t := range params.Tools {
		internalParams.Tools[i] = internalprovider.ToolDefinition{
			Name:         t.Name,
			Description:  t.Description,
			Parameters:   t.Parameters,
			Kind:         t.Kind,
			Provider:     t.Provider,
			ProviderType: t.ProviderType,
			Model:        t.Model,
		}
	}

	ch := make(chan StreamEvent, 100)
	go func() {
		defer close(ch)
		for ev := range a.inner.Chat(ctx, internalParams) {
			ch <- StreamEvent{
				Type:       streamEventTypeToPublic(ev.Type),
				TextDelta:  ev.TextDelta,
				ThinkDelta: ev.ThinkDelta,
				ToolCall:   toolCallToPublic(ev.ToolCall),
				Usage:      usageToPublic(ev.Usage),
				Error:      ev.Error,
				StopReason: ev.StopReason,
			}
		}
	}()
	return ch
}

func streamEventTypeToPublic(t internalprovider.StreamEventType) StreamEventType {
	switch t {
	case internalprovider.StreamStart:
		return StreamStart
	case internalprovider.StreamTextDelta:
		return StreamTextDelta
	case internalprovider.StreamThinkDelta, internalprovider.StreamThinkSignature:
		return StreamThinkDelta
	case internalprovider.StreamToolCall:
		return StreamToolCall
	case internalprovider.StreamUsage:
		return StreamUsage
	case internalprovider.StreamDone:
		return StreamDone
	case internalprovider.StreamError, internalprovider.StreamRetry:
		return StreamError
	default:
		return StreamError
	}
}

func (a *providerAdapter) Name() string { return a.inner.Name() }

func (a *providerAdapter) Models() []ModelInfo {
	models := a.inner.Models()
	result := make([]ModelInfo, len(models))
	for i, m := range models {
		result[i] = ModelInfo{
			ID:            m.ID,
			Name:          m.Name,
			Provider:      m.Provider,
			Reasoning:     m.Reasoning,
			Input:         append([]string(nil), m.Input...),
			ContextWindow: m.ContextWindow,
			MaxTokens:     m.MaxTokens,
		}
	}
	return result
}

func (a *providerAdapter) GetModel(id string) *ModelInfo {
	m := a.inner.GetModel(id)
	if m == nil {
		return nil
	}
	pub := ModelInfo{
		ID:            m.ID,
		Name:          m.Name,
		Provider:      m.Provider,
		Reasoning:     m.Reasoning,
		Input:         append([]string(nil), m.Input...),
		ContextWindow: m.ContextWindow,
		MaxTokens:     m.MaxTokens,
	}
	return &pub
}

func toolCallToPublic(tc *internalprovider.ToolCallBlock) *ToolCallBlock {
	if tc == nil {
		return nil
	}
	return &ToolCallBlock{
		ID:               tc.ID,
		Name:             tc.Name,
		Arguments:        tc.Arguments,
		InvalidArguments: tc.InvalidArguments,
		ThoughtSignature: tc.ThoughtSignature,
	}
}

func usageToPublic(u *internalprovider.Usage) *Usage {
	if u == nil {
		return nil
	}
	return &Usage{
		InputTokens:  u.Input,
		OutputTokens: u.Output,
		CacheRead:    u.CacheRead,
		CacheWrite:   u.CacheWrite,
		TotalTokens:  u.TotalTokens,
	}
}

func init() {
	SetResolveProviderFunc(func(vendor, baseURL, api, apiKey string) (Provider, error) {
		p, err := internalprovider.ResolveProvider(&config.ProviderConfig{
			Vendor:  vendor,
			BaseURL: baseURL,
			API:     api,
			APIKey:  apiKey,
		})
		if err != nil {
			return nil, err
		}
		return &providerAdapter{inner: p}, nil
	})
}
