package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fuckvibecoding/vibecoding/internal/agent"
	"github.com/fuckvibecoding/vibecoding/internal/config"
	"github.com/fuckvibecoding/vibecoding/internal/provider"
	"github.com/fuckvibecoding/vibecoding/internal/session"
	"github.com/fuckvibecoding/vibecoding/internal/skills"
	"github.com/fuckvibecoding/vibecoding/internal/tools"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	thinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	modePlanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	modeAgentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	modeYOLOStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

// App is the main TUI application.
type App struct {
	provider    provider.Provider
	model       *provider.Model
	settings    *config.Settings
	session     *session.Manager
	registry    *tools.Registry
	sandboxInfo string
	mode        string
	extraContext string
	skillsMgr   *skills.Manager

	// State
	viewport   viewport.Model
	editor     textarea.Model
	messages   []displayMessage
	isThinking bool
	agent      *agent.Agent
	eventCh    <-chan agent.Event
	width      int
	height     int
	ready      bool
	showThink  bool
}

type displayMessage struct {
	role     string // "user", "assistant", "tool", "error", "system"
	content  string
	toolName string
}

// NewApp creates a new TUI application.
func NewApp(p provider.Provider, model *provider.Model, settings *config.Settings, sess *session.Manager, registry *tools.Registry, sandboxInfo string, extraContext string, skillsMgr *skills.Manager) *App {
	editor := textarea.New()
	editor.Placeholder = "Type a message... (@ for files, / for commands, Tab to switch mode)"
	editor.Focus()
	editor.SetHeight(3)
	editor.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	return &App{
		provider:     p,
		model:        model,
		settings:     settings,
		session:      sess,
		registry:     registry,
		sandboxInfo:  sandboxInfo,
		mode:         settings.DefaultMode,
		extraContext: extraContext,
		skillsMgr:    skillsMgr,
		editor:       editor,
		viewport:     vp,
		showThink:    true,
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		editorCmd tea.Cmd
		vpCmd     tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

		headerHeight := 1
		footerHeight := 2
		inputHeight := 5
		available := msg.Height - headerHeight - footerHeight - inputHeight
		if available < 5 {
			available = 5
		}

		a.viewport.Width = msg.Width
		a.viewport.Height = available
		a.viewport.SetContent(a.renderMessages())
		a.editor.SetWidth(msg.Width - 2)

		return a, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			// Ctrl+C always quits
			return a, tea.Quit

		case "esc":
			// Escape: abort current thinking/action if running
			if a.isThinking {
				if a.agent != nil {
					a.agent.Abort()
				}
				a.isThinking = false
				a.addMessage("system", "⏹ Aborted")
				a.viewport.SetContent(a.renderMessages())
				a.viewport.GotoBottom()
				return a, nil
			}
			// If not thinking, clear input
			a.editor.Reset()
			return a, nil

		case "tab":
			// Tab: cycle Plan → Agent → YOLO
			a.cycleMode()
			return a, nil

		case "enter":
			// Check for special commands
			input := strings.TrimSpace(a.editor.Value())
			if input != "" {
				a.editor.Reset()
				return a, a.processInput(input)
			}
			return a, nil

		case "ctrl+t":
			// Toggle thinking block display
			a.showThink = !a.showThink
			a.viewport.SetContent(a.renderMessages())
			return a, nil
		}

	case agentDoneMsg:
		a.isThinking = false
		if msg.err != nil {
			a.addMessage("error", msg.err.Error())
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()
		return a, nil

	case agentEventMsg:
		return a, a.handleAgentEvent(msg.event)

	case agentStartMsg:
		a.isThinking = true
		a.addMessage("user", msg.input)
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()
		return a, nil
	}

	a.editor, editorCmd = a.editor.Update(msg)
	a.viewport, vpCmd = a.viewport.Update(msg)

	return a, tea.Batch(editorCmd, vpCmd)
}

// View implements tea.Model.
func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	header := a.renderHeader()
	footer := a.renderFooter()
	editorView := inputStyle.Render(a.editor.View())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		a.viewport.View(),
		editorView,
		footer,
	)
}

func (a *App) renderHeader() string {
	modelName := "unknown"
	if a.model != nil {
		modelName = a.model.Name
	}

	thinking := a.settings.DefaultThinkingLevel
	if thinking == "" {
		thinking = "off"
	}

	// Mode display with color
	var modeDisplay string
	switch a.mode {
	case "plan":
		modeDisplay = modePlanStyle.Render("🗒 PLAN")
	case "agent":
		modeDisplay = modeAgentStyle.Render("🔧 AGENT")
	case "yolo":
		modeDisplay = modeYOLOStyle.Render("🚀 YOLO")
	default:
		modeDisplay = strings.ToUpper(a.mode)
	}

	header := fmt.Sprintf("  VibeCoding | %s | %s | Think: %s",
		modelName, modeDisplay, thinking)

	return titleStyle.Width(a.width).Render(header)
}

func (a *App) renderFooter() string {
	cwd := "."
	if a.session != nil && a.session.GetHeader() != nil {
		cwd = a.session.GetHeader().Cwd
	}

	status := fmt.Sprintf("  📁 %s  |  %s", cwd, a.sandboxInfo)

	if a.isThinking {
		status += "  |  ⏳ Processing... (Esc to abort)"
	} else {
		status += "  |  Tab: switch mode"
	}

	return statusStyle.Width(a.width).Render(status)
}

func (a *App) renderMessages() string {
	var sb strings.Builder

	for _, m := range a.messages {
		switch m.role {
		case "user":
			sb.WriteString(userStyle.Render("You: "))
			sb.WriteString(m.content)
			sb.WriteString("\n\n")

		case "assistant":
			sb.WriteString(assistantStyle.Render("Assistant: "))
			sb.WriteString(m.content)
			sb.WriteString("\n\n")

		case "thinking":
			if a.showThink {
				sb.WriteString(thinkStyle.Render("💭 Thinking: " + m.content))
				sb.WriteString("\n\n")
			}

		case "tool":
			sb.WriteString(toolStyle.Render(fmt.Sprintf("🔧 [%s] %s", m.toolName, m.content)))
			sb.WriteString("\n\n")

		case "error":
			sb.WriteString(errorStyle.Render("❌ Error: " + m.content))
			sb.WriteString("\n\n")

		case "system":
			sb.WriteString(toolStyle.Render("ℹ️ " + m.content))
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (a *App) addMessage(role, content string) {
	a.messages = append(a.messages, displayMessage{role: role, content: content})
}

func (a *App) addToolMessage(toolName, content string) {
	a.messages = append(a.messages, displayMessage{role: "tool", toolName: toolName, content: content})
}

// cycleMode cycles through Plan → Agent → YOLO modes.
func (a *App) cycleMode() {
	modes := []string{"plan", "agent", "yolo"}
	current := 0
	for i, m := range modes {
		if m == a.mode {
			current = i
			break
		}
	}
	next := (current + 1) % len(modes)
	a.mode = modes[next]

	// Update sandbox for the new mode
	sbMgr := a.registry.GetSandbox()

	var modeLabel string
	switch a.mode {
	case "plan":
		modeLabel = "🗒️  PLAN - Read-only analysis and planning"
		if sbMgr != nil {
			// Plan uses strict sandbox
		}
	case "agent":
		modeLabel = "🔧 AGENT - Standard read/write access"
		if sbMgr != nil {
			// Agent uses standard sandbox
		}
	case "yolo":
		modeLabel = "🚀 YOLO - Full system access, no restrictions"
		if sbMgr != nil {
			// YOLO uses no sandbox
		}
	}

	// Reset agent to pick up new mode on next message
	a.agent = nil

	a.addMessage("system", fmt.Sprintf("Mode: %s", modeLabel))
	a.viewport.SetContent(a.renderMessages())
	a.viewport.GotoBottom()
}

func (a *App) processInput(input string) tea.Cmd {
	// Handle commands
	if strings.HasPrefix(input, "/") {
		return a.handleCommand(input)
	}

	// Normal message: send to agent
	a.addMessage("user", input)
	a.viewport.SetContent(a.renderMessages())
	a.viewport.GotoBottom()

	// Create agent if not exists
	if a.agent == nil {
		agentCfg := agent.Config{
			Provider:      a.provider,
			Model:         a.model,
			Mode:          a.mode,
			ThinkingLevel: provider.ThinkingLevel(a.settings.DefaultThinkingLevel),
			MaxTokens:     a.settings.MaxOutputTokens,
			Settings:      a.settings,
			Session:       a.session,
			ExtraContext:  a.extraContext,
		}
		a.agent = agent.New(agentCfg, a.registry)
	}

	ctx := context.Background()
	a.eventCh = a.agent.Run(ctx, input)

	// Start a goroutine to forward events to the TUI
	return tea.Batch(
		func() tea.Msg { return agentStartMsg{input: input} },
		listenEvents(a.eventCh),
	)
}

func (a *App) handleCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	command := parts[0]

	switch command {
	case "/mode":
		if len(parts) > 1 {
			switch parts[1] {
			case "plan", "agent", "yolo":
				a.mode = parts[1]
				a.agent = nil
				a.addMessage("system", fmt.Sprintf("Mode switched to: %s", strings.ToUpper(a.mode)))
			default:
				a.addMessage("error", "Invalid mode. Use: plan, agent, yolo")
			}
		} else {
			a.addMessage("system", fmt.Sprintf("Current mode: %s\nUse /mode [plan|agent|yolo] or press Tab to switch.", strings.ToUpper(a.mode)))
		}
	case "/model":
		a.addMessage("system", fmt.Sprintf("Model: %s (%s)", a.model.Name, a.model.Provider))
	case "/think":
		a.addMessage("system", fmt.Sprintf("Thinking level: %s\nChange in ~/.vibecoding/settings.json or with --thinking flag.", a.settings.DefaultThinkingLevel))
	case "/help":
		a.addMessage("system", `Commands:
  /mode [plan|agent|yolo]  - Switch mode
  /model                   - Show current model
  /think                   - Show thinking level
  /clear                   - Clear conversation
  /quit                    - Exit

Keys:
  Tab       - Switch mode (Plan → Agent → YOLO)
  Enter     - Send message
  Escape    - Abort current action / Clear input
  Ctrl+C    - Quit
  Ctrl+T    - Toggle thinking display`)
	case "/clear":
		a.messages = nil
		a.agent = nil
		a.addMessage("system", "Conversation cleared.")
	case "/quit":
		return tea.Quit
	default:
		a.addMessage("error", fmt.Sprintf("Unknown command: %s. Type /help for available commands.", command))
	}

	a.viewport.SetContent(a.renderMessages())
	a.viewport.GotoBottom()
	return nil
}

func (a *App) handleAgentEvent(event agent.Event) tea.Cmd {
	switch event.Type {
	case agent.EventTextDelta:
		// Update last assistant message or create new one
		found := false
		for i := len(a.messages) - 1; i >= 0; i-- {
			if a.messages[i].role == "assistant" {
				a.messages[i].content += event.TextDelta
				found = true
				break
			}
		}
		if !found {
			a.messages = append(a.messages, displayMessage{role: "assistant", content: event.TextDelta})
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventThinkDelta:
		found := false
		for i := len(a.messages) - 1; i >= 0; i-- {
			if a.messages[i].role == "thinking" {
				a.messages[i].content += event.ThinkDelta
				found = true
				break
			}
		}
		if !found {
			a.messages = append(a.messages, displayMessage{role: "thinking", content: event.ThinkDelta})
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventToolCall:
		if event.ToolCall != nil {
			a.addToolMessage(event.ToolCall.Name, fmt.Sprintf("Calling %s...", event.ToolCall.Name))
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventToolStart:
		// Already handled by EventToolCall

	case agent.EventToolResult:
		// Find and update the tool message
		for i := len(a.messages) - 1; i >= 0; i-- {
			if a.messages[i].role == "tool" && a.messages[i].toolName == event.ToolName {
				a.messages[i].content = event.ToolResult
				break
			}
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventDone:
		a.isThinking = false
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventError:
		a.isThinking = false
		if event.Error != nil {
			a.addMessage("error", event.Error.Error())
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()

	case agent.EventUsage:
		if event.Usage != nil {
			costStr := fmt.Sprintf("Tokens: %d in / %d out | Cost: $%.4f",
				event.Usage.Input, event.Usage.Output, event.Usage.Cost.Total)
			a.addMessage("system", costStr)
		}
		a.viewport.SetContent(a.renderMessages())
		a.viewport.GotoBottom()
	}

	return listenEvents(a.eventCh)
}

// Message types for BubbleTea
type agentStartMsg struct {
	input string
}

type agentEventMsg struct {
	event agent.Event
}

type agentDoneMsg struct {
	err error
}

// listenEvents creates a tea.Cmd that listens for agent events and converts them to tea.Msg.
func listenEvents(eventCh <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-eventCh
		if !ok {
			return agentDoneMsg{}
		}
		if event.Type == agent.EventError {
			return agentDoneMsg{err: event.Error}
		}
		if event.Type == agent.EventDone {
			return agentDoneMsg{}
		}
		return agentEventMsg{event: event}
	}
}
