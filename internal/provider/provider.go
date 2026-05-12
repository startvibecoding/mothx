package provider

import "context"

// Provider is the interface that all LLM providers must implement.
type Provider interface {
	// Chat sends a chat request and returns a channel of streaming events.
	Chat(ctx context.Context, params ChatParams) <-chan StreamEvent

	// Name returns the provider's name (e.g. "openai", "anthropic").
	Name() string

	// Models returns the list of available models.
	Models() []*Model

	// GetModel returns a model by ID, or nil if not found.
	GetModel(id string) *Model
}

// BaseProvider provides common functionality for provider implementations.
type BaseProvider struct {
	name   string
	models []*Model
}

func NewBaseProvider(name string, models []*Model) BaseProvider {
	return BaseProvider{name: name, models: models}
}

func (p *BaseProvider) Name() string {
	return p.name
}

func (p *BaseProvider) Models() []*Model {
	return p.models
}

func (p *BaseProvider) GetModel(id string) *Model {
	for _, m := range p.models {
		if m.ID == id {
			return m
		}
	}
	return nil
}
