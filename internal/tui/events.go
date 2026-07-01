package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/vibecoding/internal/agent"
)

type agentEventMsg struct{ event agent.Event }
type agentDoneMsg struct {
	err        error
	stopReason string
}
type updateNoticeMsg string

// ShowUpdateNotice displays an update notification from a background check.
func (a *App) ShowUpdateNotice(notice string) {
	if notice == "" || a.program == nil {
		return
	}
	a.program.Send(updateNoticeMsg(notice))
}

func (a *App) listenAgentEvents() tea.Cmd {
	eventCh := a.eventCh
	return func() tea.Msg {
		var next agent.Event
		var lastDone agent.Event
		err := agent.ConsumeEvents(context.Background(), eventCh, agent.EventHandlerFunc(func(_ context.Context, event agent.Event) error {
			next = event
			// Capture the last EventDone/EventError for stop reason
			if event.Type == agent.EventDone || event.Type == agent.EventError {
				lastDone = event
			}
			return context.Canceled
		}))
		if next.Type != 0 || err == context.Canceled {
			return agentEventMsg{event: next}
		}
		return agentDoneMsg{err: err, stopReason: lastDone.StopReason}
	}
}
