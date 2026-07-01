package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/startvibecoding/GoStreamingMarkdown/gsm"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/cron"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/skills"
	"github.com/startvibecoding/vibecoding/internal/tools"
	"github.com/startvibecoding/vibecoding/internal/tui/components/editor"
	"github.com/startvibecoding/vibecoding/internal/tui/components/suggest"
	"github.com/startvibecoding/vibecoding/internal/ua"
)

var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	toolModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	thinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			BorderTop(true).
			BorderForeground(lipgloss.Color("240"))

	footerSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))

	footerTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	pasteMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)

// InputEvent represents a queued input event
type InputEvent struct {
	msg     tea.Msg
	arrived time.Time
}

type toolResultStatus string

const (
	toolResultStatusRunning   toolResultStatus = "running"
	toolResultStatusCompleted toolResultStatus = "completed"
)

// toolResult stores tool result information
type toolResult struct {
	toolCallID  string // Unique tool call ID for precise matching
	toolName    string
	toolArgs    map[string]any // Tool call arguments
	status      toolResultStatus
	summary     string // Short summary for collapsed view
	fullContent string // Full content for expanded view
	diff        *tools.FileDiff
	msgIndex    int // Index in a.messages where this tool message lives
	expanded    string
}

// App is the main TUI application.
type App struct {
	provider     provider.Provider
	providerName string
	model        *provider.Model
	settings     *config.Settings
	allow        *config.AllowConfig
	session      *session.Manager
	registry     *tools.Registry
	sandboxInfo  string
	cwd          string
	mode         string
	extraContext string
	skillsMgr    *skills.Manager

	// Skills state: base extraContext (without skills) and active skill names
	baseExtraContext string            // extraContext without skill content
	activeSkills     map[string]string // skill name -> skill context string

	// UI Components
	input      editor.Model
	authInput  editor.Model
	modelInput editor.Model
	suggest    suggest.Model
	timer      stopwatch.Model

	// State
	messages               []string
	auth                   authDialogState
	modelDialog            modelDialogState
	defaultModelDialog     defaultModelDialogState
	sessionsDialog         sessionsDialogState
	toolResults            []toolResult // Store tool results for expansion
	isThinking             bool
	manualCompactionActive bool
	pendingAbortReason     string
	agent                  *agent.Agent
	eventCh                <-chan agent.Event
	width                  int
	height                 int
	ready                  bool

	// Paste markers storage
	pasteCounter int
	pastes       map[int]string

	// Input queue for batching
	inputQueue     []InputEvent
	inputQueueMu   sync.Mutex
	lastInputTime  time.Time
	inputBatchSize int
	inputDelay     time.Duration

	// Completed transcript blocks are printed above the live Bubble Tea view so
	// the terminal owns scrollback, mouse scrolling, and text selection.
	liveContent string
	printMu     sync.Mutex
	printCond   *sync.Cond
	printQueue  []string
	printOnce   sync.Once

	// Initial message to display
	initialMessage string

	// Tool output modal
	toolModalOpen         bool
	toolModalOffset       int
	toolModalPinnedBottom bool
	toolModalActive       int
	toolModalCacheValid   bool
	toolModalCacheActive  int
	toolModalCacheVersion int
	toolModalVersion      int
	toolModalCacheLines   []string

	// /btw side-question floating layer
	btwOpen     bool
	btwActive   bool   // a /btw sub-agent is currently running
	btwQuestion string // the side question being answered
	btwAnswer   string // accumulated answer text (raw)
	btwThink    string // latest think summary (raw, tail-truncated)
	btwErr      error  // error if the side query failed
	btwScroll   int    // scroll offset within the overlay
	btwEventCh  <-chan agent.Event
	btwCancel   context.CancelFunc

	// /btw streaming render optimization (mirrors main-agent assistant rendering):
	// accumulate deltas with a Builder, render markdown via a dedicated streaming
	// renderer, and cache the wrapped output so View does not re-wrap every frame.
	btwAnswerBuilder *strings.Builder
	btwThinkBuilder  *strings.Builder
	btwRenderer      *gsm.Stream
	btwRendered      string // cached rendered+wrapped answer body
	btwRenderWidth   int    // width the cache was built for
	btwAnswerDirty   bool   // answer changed since last render

	// Compact tool display mode
	compactMode bool

	// Context usage
	contextUsage *ctxpkg.ContextUsage
	currentPlan  *tools.TaskPlan

	// Cache usage tracking (cumulative)
	totalInputTokens int
	totalCacheRead   int
	totalCacheWrite  int
	totalCostUSD     float64
	latestUsage      *provider.Usage

	// Spinner state
	spinnerIndex int
	requestStart time.Time
	lastDuration time.Duration

	// Session history
	sessionMu          sync.Mutex
	historyLoaded      bool
	agentHistoryLoaded bool

	// Prompt input history
	inputHistory         []string
	inputHistoryBrowsing bool
	inputHistoryIndex    int
	inputHistoryDraft    string

	// Render throttling
	lastRender     time.Time
	renderPending  bool
	renderMu       sync.Mutex
	renderInterval time.Duration

	// Approval state
	waitingForApproval bool
	pendingApprovalID  string
	currentApproval    pendingApproval
	currentApprovalIdx int
	approvalQueue      []pendingApproval

	// Question state
	waitingForQuestion bool
	pendingQuestionID  string
	currentQuestion    pendingQuestion
	questionQueue      []pendingQuestion

	// Multi-agent / delegate state
	multiAgent         bool
	delegateMode       bool
	workflows          bool
	activeAgent        agentpkg.AgentID
	agentMgr           *agent.AgentManager
	agentActivities    map[agentpkg.AgentID]*agentActivity
	agentActivityOrder []agentpkg.AgentID

	// Cron state
	cronStore cron.CronStore
	scheduler *cron.Scheduler

	// Current streaming message indices (-1 = none)
	currentAssistantIdx int
	currentThinkIdx     int
	printedMessageIdx   map[int]bool
	thinkRaw            map[int]string // message index -> raw thinking content

	// Markdown rendering for assistant messages
	mdRenderer        *gsm.Stream
	assistantRaw      map[int]string // message index -> raw markdown content
	assistantBuilders map[int]*strings.Builder
	assistantRendered map[int]string // message index -> gsm-rendered content
	assistantDirty    map[int]bool   // message index -> needs re-render
	thinkBuilders     map[int]*strings.Builder

	// Bubble Tea program used to marshal deferred renders back onto the UI goroutine.
	program *tea.Program

	// External TUI-only status line state.
	statusLineOutput       string
	statusLineLastError    string
	statusLineInFlight     bool
	statusLineLastSuccess  string
	statusLineLastAttempt  string
	statusLinePending      *statusLineRequest
	statusLineIntervalInit bool

	// reloadRequested is set by /reload to ask main to re-exec the process after
	// the TUI exits, giving a clean "fresh process" with a brand-new session.
	reloadRequested bool
}

// ReloadRequested reports whether the user asked to reload via /reload.
func (a *App) ReloadRequested() bool { return a.reloadRequested }

// pendingApproval holds a queued approval request.
type pendingApproval struct {
	agentID    agentpkg.AgentID
	approvalID string
	toolName   string
	args       map[string]any
}

// pendingQuestion holds a queued question request.
type pendingQuestion struct {
	questionID string
	question   string
	options    []string
	context    string
}

type statusLineRequest struct {
	hash    string
	force   bool
	payload []byte
	width   int
}

// NewApp creates a new TUI application.
func NewApp(p provider.Provider, model *provider.Model, settings *config.Settings, sess *session.Manager, registry *tools.Registry, sandboxInfo string, extraContext string, skillsMgr *skills.Manager, initialMode string, multiAgent bool, delegateMode bool, agentMgr *agent.AgentManager, cronStore cron.CronStore, scheduler *cron.Scheduler) *App {
	return NewAppWithWorkflows(p, model, settings, sess, registry, sandboxInfo, extraContext, skillsMgr, initialMode, multiAgent, delegateMode, false, agentMgr, cronStore, scheduler)
}

func NewAppWithWorkflows(p provider.Provider, model *provider.Model, settings *config.Settings, sess *session.Manager, registry *tools.Registry, sandboxInfo string, extraContext string, skillsMgr *skills.Manager, initialMode string, multiAgent bool, delegateMode bool, workflows bool, agentMgr *agent.AgentManager, cronStore cron.CronStore, scheduler *cron.Scheduler) *App {
	return NewAppWithWorkflowsAndAllow(p, model, settings, sess, registry, sandboxInfo, extraContext, skillsMgr, initialMode, multiAgent, delegateMode, workflows, agentMgr, cronStore, scheduler, "", config.LoadAllow())
}

// NewAppWithWorkflowsAndAllow creates a new TUI application. providerKey is the
// user-configured settings.json provider key (e.g. "xiaomi", "doubao"); when
// empty the resolved vendor name from p.Name() is used as a fallback.
func NewAppWithWorkflowsAndAllow(p provider.Provider, model *provider.Model, settings *config.Settings, sess *session.Manager, registry *tools.Registry, sandboxInfo string, extraContext string, skillsMgr *skills.Manager, initialMode string, multiAgent bool, delegateMode bool, workflows bool, agentMgr *agent.AgentManager, cronStore cron.CronStore, scheduler *cron.Scheduler, providerKey string, allow *config.AllowConfig) *App {
	input := editor.New(80).SetPlaceholder("Type a message...").SetMaxLines(5)

	// Determine initial mode: use provided mode, fall back to settings default
	mode := initialMode
	if mode == "" {
		mode = settings.DefaultMode
	}
	if mode == "" {
		mode = "agent"
	}
	if allow == nil {
		allow = config.LoadAllow()
	}

	providerName := providerKey
	if providerName == "" {
		providerName = safeProviderName(p)
	}

	app := &App{
		provider:            p,
		providerName:        providerName,
		model:               model,
		settings:            settings,
		allow:               allow,
		session:             sess,
		registry:            registry,
		sandboxInfo:         sandboxInfo,
		cwd:                 currentWorkingDir(sess),
		mode:                mode,
		extraContext:        extraContext,
		baseExtraContext:    extraContext,
		activeSkills:        make(map[string]string),
		skillsMgr:           skillsMgr,
		input:               input,
		authInput:           editor.New(80).SetPlaceholder("").SetMaxLines(3),
		suggest:             suggest.New(80).SetItems(commandSuggestionItems()),
		timer:               stopwatch.NewWithInterval(time.Second),
		printQueue:          make([]string, 0, 256),
		pastes:              make(map[int]string),
		inputQueue:          make([]InputEvent, 0, 100),
		inputBatchSize:      10,
		inputDelay:          16 * time.Millisecond, // ~60fps
		renderInterval:      16 * time.Millisecond, // ~60fps
		currentAssistantIdx: -1,
		currentThinkIdx:     -1,
		currentApprovalIdx:  -1,
		printedMessageIdx:   make(map[int]bool),
		thinkRaw:            make(map[int]string),
		thinkBuilders:       make(map[int]*strings.Builder),
		assistantRaw:        make(map[int]string),
		assistantBuilders:   make(map[int]*strings.Builder),
		assistantRendered:   make(map[int]string),
		assistantDirty:      make(map[int]bool),
		multiAgent:          multiAgent,
		delegateMode:        delegateMode,
		workflows:           workflows,
		agentMgr:            agentMgr,
		agentActivities:     make(map[agentpkg.AgentID]*agentActivity),
		cronStore:           cronStore,
		scheduler:           scheduler,
	}
	app.markBuiltinActiveSkills()

	app.configureMarkdownRenderer()

	return app
}

// SetInitialMessage sets an initial message to display when the TUI starts.
func (a *App) SetInitialMessage(msg string) {
	a.initialMessage = msg
}

// SetProgram stores the Bubble Tea program used for deferred UI updates.
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
	if a.printCond == nil {
		a.printCond = sync.NewCond(&a.printMu)
	}
	a.printOnce.Do(func() {
		go func() {
			for {
				a.printMu.Lock()
				for len(a.printQueue) == 0 {
					a.printCond.Wait()
				}
				line := a.printQueue[0]
				a.printQueue[0] = ""
				a.printQueue = a.printQueue[1:]
				program := a.program
				a.printMu.Unlock()

				if program != nil {
					// Print directly to native terminal scrollback; do not route
					// completed transcript output back through the update loop.
					program.Println(line)
				}
			}
		}()
	})
}

// LoadHistoryMessages loads messages from session history into TUI display.
func (a *App) LoadHistoryMessages() {
	a.sessionMu.Lock()
	defer a.sessionMu.Unlock()

	if a.session == nil {
		return
	}

	historyMessages := a.session.GetMessages()
	if len(historyMessages) == 0 {
		return
	}

	a.historyLoaded = true

	// Display history messages in TUI
	for _, msg := range historyMessages {
		switch msg.Role {
		case "user":
			a.messages = append(a.messages, userStyle.Render("You: ")+msg.Content)
		case "assistant":
			// Extract text content from assistant message
			var textContent string
			if msg.Content != "" {
				textContent = msg.Content
			} else if len(msg.Contents) > 0 {
				for _, block := range msg.Contents {
					if block.Type == "text" && block.Text != "" {
						textContent += block.Text
					}
				}
			}
			if textContent != "" {
				a.messages = append(a.messages, assistantStyle.Render(textContent))
			}
		}
	}
}

// computeContextUsage estimates context usage from the current session's
// messages and the active model. Returns nil when usage cannot be computed
// (no model, no context window, or no session/messages).
func (a *App) computeContextUsage() *ctxpkg.ContextUsage {
	if a.model == nil || a.model.ContextWindow <= 0 || a.session == nil {
		return nil
	}
	messages := a.session.GetMessages()
	if len(messages) == 0 {
		return nil
	}
	compactionSettings := ctxpkg.CompactionSettings{
		Enabled:          a.settings.Compaction.Enabled,
		ReserveTokens:    a.settings.Compaction.ReserveTokens,
		KeepRecentTokens: a.settings.Compaction.KeepRecentTokens,
		Tokenizer:        a.settings.Compaction.Tokenizer,
		TokenizerModel:   a.settings.Compaction.TokenizerModel,
		Template:         a.settings.Compaction.Template,
	}
	estimator := ctxpkg.ResolveTokenEstimator(compactionSettings, a.model)
	tokens, _ := ctxpkg.EstimateContextTokensWithEstimator(messages, estimator)
	percent := float64(tokens) / float64(a.model.ContextWindow) * 100
	return &ctxpkg.ContextUsage{
		Tokens:        tokens,
		ContextWindow: a.model.ContextWindow,
		Percent:       &percent,
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Render header as first message (use width 80 as default before WindowSizeMsg)
	headerW := a.width
	if headerW <= 0 {
		headerW = 80
	}
	providerName := ""
	modelName := "unknown"
	if a.model != nil {
		providerName = a.model.Provider
		modelName = a.model.Name
	}
	cwd := a.currentCwd()
	header := renderHeader(headerW, ua.Version, providerName, modelName, cwd)
	if header != "" {
		a.messages = append(a.messages, header)
	}

	// Show initial message if set
	if a.initialMessage != "" {
		a.messages = append(a.messages, statusStyle.Render(a.initialMessage))
	}

	// Load history messages from session
	a.LoadHistoryMessages()
	a.updateViewportContent()
	for idx := range a.messages {
		a.printMessageOnce(idx)
	}

	if inputCmd := a.input.Init(); inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}
	cmds = append(cmds, a.processInputQueue())
	return tea.Batch(cmds...)
}

func currentWorkingDir(sess *session.Manager) string {
	if sess != nil && sess.GetHeader() != nil && sess.GetHeader().Cwd != "" {
		return sess.GetHeader().Cwd
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		return cwd
	}
	return "."
}

func (a *App) currentCwd() string {
	if a.session != nil && a.session.GetHeader() != nil && a.session.GetHeader().Cwd != "" {
		return a.session.GetHeader().Cwd
	}
	if a.cwd != "" {
		return a.cwd
	}
	return "."
}

// processInputQueue returns a command that processes queued input events
func (a *App) processInputQueue() tea.Cmd {
	return tea.Tick(a.inputDelay, func(t time.Time) tea.Msg {
		return inputQueueTickMsg(t)
	})
}

// inputQueueTickMsg is sent when the input queue should be processed
type inputQueueTickMsg time.Time

// spinnerTickMsg is sent to update the spinner animation
type spinnerTickMsg time.Time

// Spinner characters for the thinking animation
var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const spinnerInterval = 100 * time.Millisecond
const mouseWheelScrollLines = 3

// tickSpinner returns a command that updates the spinner
func (a *App) tickSpinner() tea.Cmd {
	return tea.Tick(spinnerInterval, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		oldWidth := a.width
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

		a.input = a.input.SetWidth(msg.Width)
		a.authInput = a.authInput.SetWidth(msg.Width - 8)
		a.suggest = a.suggest.SetWidth(msg.Width - 2)
		if oldWidth != a.width {
			a.configureMarkdownRenderer()
			a.markAssistantRenderedDirty()
			a.invalidateToolModalCache()
		}

		a.updateViewportContent()
		if a.statusLineEnabled() {
			if !a.statusLineIntervalInit {
				a.statusLineIntervalInit = true
				if cmd := a.tickStatusLine(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			a.requestStatusLineRefresh(true)
		}
		return a, tea.Batch(cmds...)

	case inputQueueTickMsg:
		// Process queued input events
		cmd := a.flushInputQueue()
		cmds = append(cmds, cmd)
		// Schedule next tick
		cmds = append(cmds, a.processInputQueue())
		return a, tea.Batch(cmds...)

	case spinnerTickMsg:
		// Update timed UI affordances while the agent is active or user input is pending.
		if a.isThinking || a.waitingForApproval {
			a.spinnerIndex = (a.spinnerIndex + 1) % len(spinnerChars)
			cmds = append(cmds, a.tickSpinner())
		}
		return a, tea.Batch(cmds...)

	case stopwatch.TickMsg, stopwatch.StartStopMsg, stopwatch.ResetMsg:
		var timerCmd tea.Cmd
		a.timer, timerCmd = a.timer.Update(msg)
		if timerCmd != nil {
			cmds = append(cmds, timerCmd)
		}
		return a, tea.Batch(cmds...)

	case renderRequestMsg:
		a.updateViewportContent()
		return a, nil

	case statusLineRenderedMsg:
		return a, a.handleStatusLineRendered(msg)

	case statusLineTickMsg:
		if a.statusLineEnabled() {
			a.requestStatusLineRefresh(true)
			if cmd := a.tickStatusLine(); cmd != nil {
				return a, cmd
			}
		}
		return a, nil

	case tea.MouseMsg:
		return a, a.handleMouse(msg)

	case tea.KeyMsg:
		if handled, cmd := a.handleSessionsKey(msg); handled {
			return a, cmd
		}
		if handled, cmd := a.handleDefaultModelKey(msg); handled {
			return a, cmd
		}
		if handled, cmd := a.handleModelKey(msg); handled {
			return a, cmd
		}
		if handled, cmd := a.handleAuthKey(msg); handled {
			return a, cmd
		}
		if a.btwOpen {
			switch {
			case msg.Type == tea.KeyEsc || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
				a.closeBtw()
				a.scheduleRender()
				return a, nil
			case msg.Type == tea.KeyUp:
				a.scrollBtw(-1)
				return a, nil
			case msg.Type == tea.KeyDown:
				a.scrollBtw(1)
				return a, nil
			case msg.Type == tea.KeyPgUp:
				a.scrollBtw(-10)
				return a, nil
			case msg.Type == tea.KeyPgDown:
				a.scrollBtw(10)
				return a, nil
			}
			return a, nil
		}
		if a.toolModalOpen {
			switch {
			case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlO || (msg.Type == tea.KeyRunes && string(msg.Runes) == "q"):
				a.closeToolModal()
				return a, nil
			case msg.Type == tea.KeyUp:
				a.scrollToolModal(-1)
				return a, nil
			case msg.Type == tea.KeyDown:
				a.scrollToolModal(1)
				return a, nil
			case msg.Type == tea.KeyLeft:
				a.switchToolModalTarget(-1)
				return a, nil
			case msg.Type == tea.KeyRight:
				a.switchToolModalTarget(1)
				return a, nil
			case msg.Type == tea.KeyPgUp:
				a.scrollToolModal(-a.toolModalPageSize())
				return a, nil
			case msg.Type == tea.KeyPgDown:
				a.scrollToolModal(a.toolModalPageSize())
				return a, nil
			case msg.Type == tea.KeyHome:
				a.toolModalOffset = 0
				a.toolModalPinnedBottom = false
				return a, nil
			case msg.Type == tea.KeyEnd:
				a.toolModalOffset = a.maxToolModalOffset()
				a.toolModalPinnedBottom = true
				return a, nil
			}
			return a, nil
		}

		// Special keys are processed immediately; regular text input is batched.
		switch msg.Type {
		case tea.KeyCtrlC:
			return a, tea.Quit
		case tea.KeyEsc:
			if a.isThinking || a.waitingForApproval || a.waitingForQuestion {
				a.pendingAbortReason = "user pressed Esc"
				if a.agent != nil {
					a.abortAndResetAgent("aborted")
				}
				a.clearApprovalState()
				a.clearQuestionState()
				a.clearQueuedInput()
				a.input.Reset()
				a.resetInputHistoryNavigation()
				a.isThinking = false
				a.manualCompactionActive = false
				a.finishRequestTimer()
				a.addMessage(statusStyle.Render("⏹ Aborted"))
				return a, a.timer.Stop()
			} else {
				a.input.Reset()
				a.resetInputHistoryNavigation()
			}
			return a, nil
		case tea.KeyEnter:
			flushCmd := a.flushInputQueue()
			if a.commandSuggestionsVisible() && a.applySelectedCommandSuggestion() {
				return a, flushCmd
			}
			if msg.Alt {
				a.input, _ = a.input.Update(msg)
				a.scheduleRender()
				return a, flushCmd
			}

			// Process enter immediately
			input := strings.TrimSpace(a.input.Value())

			// Check if waiting for approval
			if a.waitingForApproval {
				approved := strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
				approvalIdx := a.currentApprovalIdx
				if approvalIdx >= 0 {
					a.printMessageOnce(approvalIdx)
				}
				a.handleApprovalResponse(a.pendingApprovalID, approved)
				if approved {
					a.addMessage(statusStyle.Render("✅ Approved"))
				} else {
					a.addMessage(statusStyle.Render("❌ Denied"))
				}
				// Show next queued approval or clear waiting state
				if len(a.approvalQueue) > 0 {
					a.showNextApproval()
				} else {
					a.waitingForApproval = false
					a.pendingApprovalID = ""
					a.currentApproval = pendingApproval{}
					a.currentApprovalIdx = -1
				}
				a.input.Reset()
				a.resetInputHistoryNavigation()
				a.scheduleRender()
				return a, nil
			}

			// Check if waiting for a question
			if a.waitingForQuestion {
				if a.agent != nil {
					answer := strings.TrimSpace(input)
					if answer == "" {
						// Empty input — re-prompt
						a.input.Reset()
						a.resetInputHistoryNavigation()
						a.scheduleRender()
						return a, nil
					}

					// Resolve numbered selections to the actual option text so the
					// question tool result is meaningful to the model.
					var num int
					if _, err := fmt.Sscanf(answer, "%d", &num); err == nil && num > 0 && num <= len(a.currentQuestion.options) {
						answer = a.currentQuestion.options[num-1]
						a.agent.HandleQuestionResponse(a.pendingQuestionID, answer)
						a.addMessage(statusStyle.Render(fmt.Sprintf("✅ Selected: %s", answer)))
					} else {
						// Custom text input, including out-of-range numbers.
						a.agent.HandleQuestionResponse(a.pendingQuestionID, answer)
						a.addMessage(statusStyle.Render(fmt.Sprintf("✅ Answer: %s", answer)))
					}
				}
				// Show next queued question or clear waiting state
				if len(a.questionQueue) > 0 {
					a.showNextQuestion()
				} else {
					a.waitingForQuestion = false
					a.pendingQuestionID = ""
					a.currentQuestion = pendingQuestion{}
				}
				a.input.Reset()
				a.resetInputHistoryNavigation()
				a.scheduleRender()
				return a, nil
			}

			if input != "" {
				if a.manualCompactionActive {
					a.addCommandError("Cannot send input while context compaction is running.")
					return a, nil
				}
				a.input.Reset()
				a.recordInputHistory(input)
				expandedInput := a.expandPasteMarkers(input)
				return a, a.processInput(expandedInput)
			}
			return a, nil
		case tea.KeyTab:
			flushCmd := a.flushInputQueue()
			if a.commandSuggestionsVisible() && a.applySelectedCommandSuggestion() {
				return a, flushCmd
			}
			a.cycleMode()
			return a, flushCmd
		case tea.KeyUp:
			flushCmd := a.flushInputQueue()
			if a.handleCommandSuggestionKey(msg) {
				return a, flushCmd
			}
			if a.inputHistoryBrowsing || a.input.AtFirstLine() {
				if a.navigateInputHistory(-1) {
					return a, flushCmd
				}
			}
			if !a.input.AtFirstLine() {
				a.input, _ = a.input.Update(msg)
				a.scheduleRender()
				return a, flushCmd
			}
		case tea.KeyDown:
			flushCmd := a.flushInputQueue()
			if a.handleCommandSuggestionKey(msg) {
				return a, flushCmd
			}
			if a.inputHistoryBrowsing {
				if a.navigateInputHistory(1) {
					return a, flushCmd
				}
			}
			if !a.input.AtLastLine() {
				a.input, _ = a.input.Update(msg)
				a.scheduleRender()
				return a, flushCmd
			}
		case tea.KeyCtrlO:
			if a.openLatestToolModal() {
				return a, nil
			}
			a.addMessage(statusStyle.Render("No conversation details yet."))
			return a, nil
		case tea.KeyCtrlG:
			a.compactMode = !a.compactMode
			if a.compactMode {
				a.addMessage(statusStyle.Render("Compact tool display: ON"))
			} else {
				a.addMessage(statusStyle.Render("Compact tool display: OFF"))
			}
			a.updateViewportContent()
			return a, nil
		case tea.KeyCtrlP:
			a.addCommandStatus("Multi-agent mode is configured at startup. Run with --multi-agent to enable sub-agent tools.")
			return a, nil
		}

		// Check for paste (multi-line input in a single key event)
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			input := string(msg.Runes)
			if msg.Paste || strings.Contains(input, "\n") {
				a.handlePaste(input)
				return a, nil
			}
		}

		a.queueInput(msg)
		a.resetInputHistoryNavigation()
		return a, nil

	case agentStreamStartMsg:
		a.eventCh = msg.eventCh
		a.isThinking = true
		a.manualCompactionActive = msg.compacting
		a.pendingAbortReason = ""
		a.spinnerIndex = 0
		a.requestStart = time.Now()
		a.lastDuration = 0
		if !msg.compacting && msg.input != "" {
			a.addMessage(userStyle.Render("You: ") + msg.input)
		}
		return a, tea.Batch(a.listenAgentEvents(), a.tickSpinner(), a.timer.Reset(), a.timer.Start())

	case agentEventMsg:
		return a, a.handleAgentEvent(msg.event)

	case btwStreamStartMsg:
		a.btwEventCh = msg.eventCh
		return a, a.listenBtwEvents()

	case btwEventMsg:
		return a, a.handleBtwEvent(msg.event)

	case btwDoneMsg:
		a.btwActive = false
		a.btwEventCh = nil
		if msg.err != nil {
			a.btwErr = msg.err
		}
		a.scheduleRender()
		return a, nil

	case agentDoneMsg:
		a.isThinking = false
		a.manualCompactionActive = false
		a.finishRequestTimer()
		if msg.err != nil {
			a.addMessage(errorStyle.Render("Error: ") + msg.err.Error())
		} else if msg.stopReason != "" {
			a.addMessage(statusStyle.Render("Session ended: ") + msg.stopReason)
		}
		return a, a.timer.Stop()
	}

	// Update components
	var inputCmd tea.Cmd
	a.input, inputCmd = a.input.Update(msg)

	if inputCmd != nil {
		cmds = append(cmds, inputCmd)
	}

	return a, tea.Batch(cmds...)
}

// queueInput adds an input event to the queue
func (a *App) queueInput(msg tea.Msg) {
	a.inputQueueMu.Lock()
	defer a.inputQueueMu.Unlock()

	a.inputQueue = append(a.inputQueue, InputEvent{
		msg:     msg,
		arrived: time.Now(),
	})
	a.lastInputTime = time.Now()
}

// flushInputQueue processes all queued input events
func (a *App) flushInputQueue() tea.Cmd {
	a.inputQueueMu.Lock()
	events := make([]InputEvent, len(a.inputQueue))
	copy(events, a.inputQueue)
	a.inputQueue = a.inputQueue[:0]
	a.inputQueueMu.Unlock()

	if len(events) == 0 {
		return nil
	}

	// Process events in batch
	var cmds []tea.Cmd
	for _, event := range events {
		// Update input component
		if keyMsg, ok := event.msg.(tea.KeyMsg); ok {
			var cmd tea.Cmd
			a.input, cmd = a.input.Update(keyMsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Schedule render
	a.updateCommandSuggestions()
	a.scheduleRender()

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// scheduleRender schedules a render update with throttling.
// If called inside the throttle window, a delayed render is scheduled
// so the pending update is not lost.
func (a *App) scheduleRender() {
	a.renderMu.Lock()
	defer a.renderMu.Unlock()

	a.requestStatusLineRefresh(false)

	now := time.Now()
	if now.Sub(a.lastRender) < a.renderInterval {
		if !a.renderPending {
			a.renderPending = true
			remaining := a.renderInterval - now.Sub(a.lastRender)
			time.AfterFunc(remaining, func() {
				a.renderMu.Lock()
				wasPending := a.renderPending
				if wasPending {
					a.lastRender = time.Now()
					a.renderPending = false
				}
				a.renderMu.Unlock()
				if wasPending {
					if a.program != nil {
						a.program.Send(renderRequestMsg{})
					}
				}
			})
		}
		return
	}

	// Render now
	a.lastRender = now
	a.renderPending = false
	a.updateViewportContent()
}

// View implements tea.Model.
func (a *App) View() string {
	if !a.ready {
		return "\n  Loading...\n"
	}

	footer := a.renderFooter()
	if a.toolModalOpen {
		return a.renderFixedHeight(lipgloss.JoinVertical(lipgloss.Left, a.renderToolModal(), footer))
	}
	if a.btwOpen {
		return a.renderFixedHeight(lipgloss.JoinVertical(lipgloss.Left, a.renderBtwOverlay(), footer))
	}

	var parts []string

	// 1. Live transcript (streaming content only when unmanaged output is active)
	if live := a.renderLiveTranscriptContent(); live != "" {
		parts = append(parts, live)
	}

	// 2. Sticky todo list (non-done tasks)
	maxTodoVisible := 5
	if a.height > 0 {
		maxTodoVisible = a.height / 8
		if maxTodoVisible < 3 {
			maxTodoVisible = 3
		}
	}
	if todoList := renderStickyTodoList(a.currentPlan, a.width, maxTodoVisible); todoList != "" {
		parts = append(parts, todoList)
	}

	// 3. Loading indicator (spinner + timer + tokens when thinking)
	if loading := renderLoadingIndicator(a.isThinking, a.spinnerIndex, a.timer.Elapsed(), 0, a.width); loading != "" {
		parts = append(parts, loading)
	}
	if activity := a.renderActivitySummary(a.width); activity != "" {
		parts = append(parts, activity)
	}

	// 4. Dialog or input field
	if a.sessionsDialog.Open {
		parts = append(parts, a.renderSessionsDialog())
	} else if a.defaultModelDialog.Open {
		parts = append(parts, a.renderDefaultModelDialog())
	} else if a.modelDialog.Open {
		parts = append(parts, a.renderModelDialog())
	} else if a.auth.Open {
		parts = append(parts, a.renderAuthDialog())
	} else {
		inputGap := lipgloss.NewStyle().
			Foreground(lipgloss.Color("236")).
			Render(strings.Repeat("▄", a.width))
		parts = append(parts, inputGap)
		parts = append(parts, a.input.View())
		if suggestions := a.suggest.View(); suggestions != "" {
			parts = append(parts, suggestions)
		}
	}

	// 5. Footer (status bar)
	parts = append(parts, footer)

	// 6. Agent tab bar (multi-agent mode)
	if tabBar := renderAgentTabBar(a.agentMgr, string(a.activeAgent), a.width); tabBar != "" {
		parts = append(parts, tabBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// handlePaste handles large pastes by creating markers
func (a *App) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if a.toolModalOpen {
		switch {
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp:
			a.scrollToolModal(-mouseWheelScrollLines)
		case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown:
			a.scrollToolModal(mouseWheelScrollLines)
		}
		return nil
	}

	return nil
}

func (a *App) markAssistantRenderedDirty() {
	if a.assistantDirty == nil {
		a.assistantDirty = make(map[int]bool)
	}
	for idx := range a.assistantRendered {
		a.assistantDirty[idx] = true
	}
}

// Message types
type agentStreamStartMsg struct {
	input      string
	eventCh    <-chan agent.Event
	compacting bool
}
type renderRequestMsg struct{}

func safeProviderName(p provider.Provider) string {
	if p == nil {
		return ""
	}
	return p.Name()
}
