package agent

import "context"

// EventHandler receives agent events from a running request.
type EventHandler interface {
	HandleAgentEvent(context.Context, Event) error
}

// EventHandlerFunc adapts a function to EventHandler.
type EventHandlerFunc func(context.Context, Event) error

// HandleAgentEvent implements EventHandler.
func (f EventHandlerFunc) HandleAgentEvent(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// ConsumeEvents forwards every event from eventCh to handler until the stream
// closes, the context is canceled, or the handler returns an error.
func ConsumeEvents(ctx context.Context, eventCh <-chan Event, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}
			if err := handler.HandleAgentEvent(ctx, event); err != nil {
				return err
			}
		}
	}
}
