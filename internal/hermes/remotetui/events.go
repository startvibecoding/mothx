package remotetui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/websocket"

	"github.com/startvibecoding/mothx/internal/agent"
)

type agentEventMsg struct{ event agent.Event }
type agentDoneMsg struct{ err error }
type wsEventMsg struct{ event wsEvent }

func (a *App) listenAgentEvents() tea.Cmd {
	return func() tea.Msg {
		if a.ws == nil {
			return agentDoneMsg{err: fmt.Errorf("websocket is not connected")}
		}
		var ev wsEvent
		if err := websocket.JSON.Receive(a.ws, &ev); err != nil {
			if err == io.EOF {
				return agentDoneMsg{err: fmt.Errorf("connection closed")}
			}
			return agentDoneMsg{err: err}
		}
		return wsEventMsg{event: ev}
	}
}
