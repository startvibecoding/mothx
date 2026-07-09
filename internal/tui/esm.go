package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/startvibecoding/mothx/internal/esm"
	"github.com/startvibecoding/mothx/internal/provider"
)

const (
	esmGetToolName    = "get_esm"
	esmUpdateToolName = "update_esm"
)

func (a *App) ensureESMStore() *esm.Store {
	dir := a.getSessionDir()
	if a.esmStore != nil && a.esmStoreDir == dir {
		return a.esmStore
	}
	if a.esmToolsRegistered && a.registry != nil {
		a.registry.Remove(esmGetToolName)
		a.registry.Remove(esmUpdateToolName)
		a.esmToolsRegistered = false
		a.resetAgent(fmt.Errorf("esm store changed"))
	}
	a.esmStore = esm.NewStore(dir)
	a.esmStoreDir = dir
	return a.esmStore
}

func (a *App) currentSessionID() string {
	if a.session == nil || a.session.GetHeader() == nil {
		return ""
	}
	return a.session.GetHeader().ID
}

func (a *App) loadESMObjective(ctx context.Context) (*esm.Objective, error) {
	store := a.ensureESMStore()
	sessionID := a.currentSessionID()
	if store == nil || sessionID == "" {
		return nil, esm.ErrNotFound
	}
	return store.Get(ctx, sessionID)
}

func (a *App) syncESMTools() error {
	if a.registry == nil {
		return nil
	}
	store := a.ensureESMStore()
	sessionID := a.currentSessionID()
	var obj *esm.Objective
	var err error
	if sessionID != "" {
		obj, err = store.Get(context.Background(), sessionID)
		if errors.Is(err, esm.ErrNotFound) {
			err = nil
		}
		if err != nil {
			return err
		}
	}

	shouldRegister := obj != nil && esm.IsRunnableStatus(obj.Status)
	if shouldRegister && !a.esmToolsRegistered {
		a.registry.Register(esm.NewGetTool(store, a.currentSessionID))
		a.registry.Register(esm.NewUpdateTool(store, a.currentSessionID))
		a.esmToolsRegistered = true
		a.resetAgent(fmt.Errorf("esm tools changed"))
	} else if !shouldRegister && a.esmToolsRegistered {
		a.registry.Remove(esmGetToolName)
		a.registry.Remove(esmUpdateToolName)
		a.esmToolsRegistered = false
		a.resetAgent(fmt.Errorf("esm tools changed"))
	}
	a.setESMFooter(obj)
	return nil
}

func (a *App) setESMFooter(obj *esm.Objective) {
	if obj == nil {
		a.esmFooter = ""
		return
	}
	parts := []string{"ESM", string(obj.Status)}
	tokenPart := formatTokens(int(obj.TokensUsed))
	if obj.TokenBudget != nil {
		tokenPart += "/" + formatTokens(int(*obj.TokenBudget))
	}
	parts = append(parts, tokenPart)
	if obj.TimeUsedMS > 0 {
		parts = append(parts, formatDuration(time.Duration(obj.TimeUsedMS)*time.Millisecond))
	}
	a.esmFooter = strings.Join(parts, " ")
}

func (a *App) handleESMCommand(cmd string) {
	if err := a.ensureSession(); err != nil {
		a.addCommandError(fmt.Sprintf("Error creating session: %v", err))
		return
	}
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "/esm"))
	if raw == "" || raw == "status" {
		a.showESMStatus()
		return
	}
	if a.isThinking {
		a.addCommandError("Cannot change ESM while the agent is running. Press Esc to abort first.")
		return
	}

	ctx := context.Background()
	store := a.ensureESMStore()
	sessionID := a.currentSessionID()
	sub, rest := splitESMSubcommand(raw)
	var (
		obj *esm.Objective
		err error
	)
	switch sub {
	case "edit":
		if strings.TrimSpace(rest) == "" {
			a.addCommandError("Usage: /esm edit <objective>")
			return
		}
		obj, err = store.Edit(ctx, sessionID, rest)
	case "pause":
		if strings.TrimSpace(rest) != "" {
			a.addCommandError("Usage: /esm pause")
			return
		}
		obj, err = store.Pause(ctx, sessionID)
	case "resume":
		if strings.TrimSpace(rest) != "" {
			a.addCommandError("Usage: /esm resume")
			return
		}
		obj, err = store.Resume(ctx, sessionID)
	case "clear":
		if strings.TrimSpace(rest) != "" {
			a.addCommandError("Usage: /esm clear")
			return
		}
		err = store.Clear(ctx, sessionID)
	case "budget":
		obj, err = a.handleESMBudget(ctx, store, sessionID, rest)
	default:
		obj, err = store.Create(ctx, sessionID, raw, nil)
	}
	if err != nil {
		a.addCommandError(formatESMCommandError(err))
		return
	}
	if err := a.syncESMTools(); err != nil {
		a.addCommandError(fmt.Sprintf("ESM updated, but tool sync failed: %v", err))
		return
	}
	if sub != "status" {
		a.resetAgent(fmt.Errorf("esm changed"))
	}
	if sub == "clear" {
		a.addCommandStatus("Enable Supervisor Mode cleared.")
		return
	}
	a.addCommandStatus(formatESMStatus(obj))
}

func splitESMSubcommand(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	idx := strings.IndexAny(raw, " \t")
	if idx < 0 {
		return raw, ""
	}
	return raw[:idx], strings.TrimSpace(raw[idx+1:])
}

func (a *App) handleESMBudget(ctx context.Context, store *esm.Store, sessionID, rest string) (*esm.Objective, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil, fmt.Errorf("Usage: /esm budget <tokens|off>")
	}
	if rest == "off" {
		return store.SetBudget(ctx, sessionID, nil)
	}
	value, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("ESM budget must be a positive integer or off")
	}
	return store.SetBudget(ctx, sessionID, &value)
}

func formatESMCommandError(err error) string {
	switch {
	case errors.Is(err, esm.ErrNotFound):
		return "No ESM objective. Create one with /esm <objective>."
	case errors.Is(err, esm.ErrObjectiveExists):
		return "An unfinished ESM objective already exists. Use /esm edit <objective> or /esm clear."
	case errors.Is(err, esm.ErrInvalidObjective):
		return "ESM objective cannot be empty."
	case errors.Is(err, esm.ErrBudgetStillHit):
		return "ESM is still budget_limited. Raise the budget with /esm budget <tokens> or remove it with /esm budget off, then /esm resume."
	case errors.Is(err, esm.ErrInvalidTransition):
		return "ESM status cannot be changed that way."
	default:
		return err.Error()
	}
}

func (a *App) showESMStatus() {
	obj, err := a.loadESMObjective(context.Background())
	if errors.Is(err, esm.ErrNotFound) {
		a.setESMFooter(nil)
		a.addCommandStatus("Enable Supervisor Mode\nStatus: none\n\nCreate one with /esm <objective>.")
		return
	}
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load ESM status: %v", err))
		return
	}
	a.setESMFooter(obj)
	a.addCommandStatus(formatESMStatus(obj))
}

func formatESMStatus(obj *esm.Objective) string {
	if obj == nil {
		return "Enable Supervisor Mode\nStatus: none"
	}
	var b strings.Builder
	b.WriteString(esm.FormatObjective(obj))
	b.WriteString("\n\nCommands: /esm edit <objective>, /esm pause, /esm resume, /esm clear, /esm budget <tokens|off>")
	return b.String()
}

func (a *App) prepareESMRun() {
	var obj *esm.Objective
	if current, err := a.loadESMObjective(context.Background()); err == nil {
		obj = current
	}
	a.esmMu.Lock()
	a.esmRunSeq++
	a.esmSteeredSeq = 0
	a.esmRunTokens = 0
	a.esmRunSessionID = a.currentSessionID()
	a.esmRunTracked = obj != nil && obj.Status == esm.StatusActive
	a.esmMu.Unlock()
	a.setESMFooter(obj)
}

func (a *App) nextESMSteeringMessages() []provider.Message {
	a.esmMu.Lock()
	seq := a.esmRunSeq
	if seq == 0 || a.esmSteeredSeq == seq {
		a.esmMu.Unlock()
		return nil
	}
	a.esmSteeredSeq = seq
	tracked := a.esmRunTracked
	a.esmMu.Unlock()
	if !tracked {
		return nil
	}
	obj, err := a.loadESMObjective(context.Background())
	if err != nil || obj == nil || obj.Status != esm.StatusActive {
		return nil
	}
	return []provider.Message{esm.SteeringMessage(obj)}
}

func (a *App) recordESMUsage(usage *provider.Usage) {
	if usage == nil {
		return
	}
	total := usage.TotalTokens
	if total <= 0 {
		total = usage.Input + usage.Output
	}
	if total <= 0 {
		return
	}
	a.esmMu.Lock()
	if a.esmRunTracked {
		a.esmRunTokens += int64(total)
	}
	a.esmMu.Unlock()
}

func (a *App) finishESMRun(err error) tea.Cmd {
	a.esmMu.Lock()
	tracked := a.esmRunTracked
	sessionID := a.esmRunSessionID
	tokens := a.esmRunTokens
	a.esmRunTracked = false
	a.esmRunSessionID = ""
	a.esmRunTokens = 0
	a.esmMu.Unlock()

	if !tracked || sessionID == "" {
		return nil
	}
	store := a.ensureESMStore()
	if tokens > 0 || a.lastDuration > 0 {
		if obj, accountErr := store.AccountUsage(context.Background(), sessionID, tokens, int64(a.lastDuration.Milliseconds())); accountErr == nil {
			a.setESMFooter(obj)
		} else if !errors.Is(accountErr, esm.ErrNotFound) {
			a.addCommandError(fmt.Sprintf("Failed to account ESM usage: %v", accountErr))
		}
	}
	if err != nil && esm.IsUsageLimitError(err) {
		if obj, markErr := store.MarkUsageLimited(context.Background(), sessionID); markErr == nil {
			a.setESMFooter(obj)
		}
	}
	if syncErr := a.syncESMTools(); syncErr != nil {
		a.addCommandError(fmt.Sprintf("Failed to sync ESM tools: %v", syncErr))
		return nil
	}
	if err != nil {
		return nil
	}
	return a.startESMContinuationIfIdle()
}

func (a *App) startESMContinuationIfIdle() tea.Cmd {
	if a.mode == "plan" || a.manualCompactionActive || a.waitingForApproval || a.waitingForQuestion || a.hasQueuedInput() {
		return nil
	}
	if strings.TrimSpace(a.input.Value()) != "" {
		return nil
	}
	if err := a.ensureSession(); err != nil {
		a.addCommandError(fmt.Sprintf("Error creating session: %v", err))
		return nil
	}
	obj, err := a.loadESMObjective(context.Background())
	if errors.Is(err, esm.ErrNotFound) || obj == nil || !obj.CanAutoRun() {
		return nil
	}
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load ESM status: %v", err))
		return nil
	}
	if err := a.syncESMTools(); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to sync ESM tools: %v", err))
		return nil
	}
	a.prepareESMRun()
	a.ensureAgent()
	a.registerManagedAgent()
	msg := esm.ContinuationMessage(obj)
	ctx := context.Background()
	return func() tea.Msg {
		return agentStreamStartMsg{
			input:      "",
			eventCh:    a.agent.RunWithUserMessage(ctx, msg),
			compacting: false,
		}
	}
}
