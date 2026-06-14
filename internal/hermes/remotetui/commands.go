package remotetui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/quit", "/exit":
		a.closeWS()
		return tea.Quit
	case "/help":
		a.showHelp()
		return nil
	case "/mode":
		if len(parts) > 1 {
			switch parts[1] {
			case "plan", "agent", "yolo":
				a.mode = parts[1]
			default:
				a.addMessage(errorStyle.Render("Invalid mode"))
				return nil
			}
		}
	case "/clear":
		a.resetTranscript()
	}

	if err := a.sendWS(wsMessage{Type: "command", Content: cmd}); err != nil {
		a.addMessage(errorStyle.Render("Error: ") + err.Error())
	}
	return nil
}

func (a *App) showHelp() {
	a.addMessage(statusStyle.Render("Commands:"))
	a.addMessage(statusStyle.Render("  /mode [plan|agent|yolo] - Switch or show mode"))
	a.addMessage(statusStyle.Render("  /clear                  - Clear conversation"))
	a.addMessage(statusStyle.Render("  /compact                - Trigger context compaction"))
	a.addMessage(statusStyle.Render("  /status                 - Show Hermes session status"))
	a.addMessage(statusStyle.Render("  /sessions               - List active Hermes sessions"))
	a.addMessage(statusStyle.Render("  /new                    - Start a new Hermes session"))
	a.addMessage(statusStyle.Render("  /quit                   - Exit"))
	a.addMessage(statusStyle.Render("  /help                   - Show this help"))
	a.addMessage(statusStyle.Render(""))
	a.addMessage(statusStyle.Render("Keyboard shortcuts:"))
	a.addMessage(statusStyle.Render("  Tab       - Cycle mode (plan/agent/yolo)"))
	a.addMessage(statusStyle.Render("  Esc       - Abort current local wait state"))
	a.addMessage(statusStyle.Render("  Ctrl+O    - Open expanded transcript"))
	a.addMessage(statusStyle.Render("  PgUp/PgDn - Page transcript details when open"))
}

func (a *App) resetTranscript() {
	a.messages = nil
	a.toolResults = nil
	a.contextUsage = nil
	a.currentPlan = nil
	a.totalInputTokens = 0
	a.totalCacheRead = 0
	a.totalCacheWrite = 0
	a.pastes = make(map[int]string)
	a.pasteCounter = 0
	a.assistantRaw = make(map[int]string)
	a.assistantRendered = make(map[int]string)
	a.assistantDirty = make(map[int]bool)
	a.thinkRaw = make(map[int]string)
	a.printedMessageIdx = make(map[int]bool)
	a.currentAssistantIdx = -1
	a.currentThinkIdx = -1
	a.liveContent = ""
	a.viewport.SetContent("")
	a.viewport.GotoBottom()
	a.addMessage(statusStyle.Render("✅ Conversation cleared"))
}

func (a *App) toggleMultiAgent() {
	if err := a.sendWS(wsMessage{Type: "command", Content: "/agent"}); err != nil {
		a.addMessage(errorStyle.Render("Error: ") + err.Error())
		return
	}
	a.addMessage(statusStyle.Render("Sent: /agent"))
}

func (a *App) handleAgentCommand(parts []string) {
	if len(parts) == 0 {
		return
	}
	cmd := strings.Join(parts, " ")
	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}
	if err := a.sendWS(wsMessage{Type: "command", Content: cmd}); err != nil {
		a.addMessage(errorStyle.Render(fmt.Sprintf("Error: %v", err)))
	}
}
