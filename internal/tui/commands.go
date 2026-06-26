package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/cron"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/systeminit"
	tuistatusline "github.com/startvibecoding/vibecoding/internal/tui/statusline"
	"github.com/startvibecoding/vibecoding/internal/workflow"
)

// handleAgentCommand handles /agent subcommands (multi-agent mode).
func (a *App) handleAgentCommand(parts []string) {
	if !a.multiAgent {
		a.addCommandError("Multi-agent mode is not enabled. Restart with --multi-agent to enable sub-agent tools.")
		return
	}
	if len(parts) < 2 {
		a.addCommandStatus("Usage: /agent list|switch|destroy")
		return
	}
	switch parts[1] {
	case "list":
		a.listAgents()
	case "switch":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /agent switch <id>")
			return
		}
		a.switchAgent(agentpkg.AgentID(parts[2]))
	case "destroy":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /agent destroy <id>")
			return
		}
		a.destroyAgent(agentpkg.AgentID(parts[2]))
	default:
		a.addCommandError(fmt.Sprintf("Unknown agent command: %s", parts[1]))
	}
}

func (a *App) listAgents() {
	a.addCommandStatus(fmt.Sprintf("Multi-agent mode: ON (active: %s)", a.activeAgent))
	if a.agentMgr == nil {
		a.addCommandStatus("  (AgentManager not initialized)")
		return
	}

	ids := a.agentMgr.List()
	if len(ids) == 0 {
		a.addCommandStatus("  No agents running")
		return
	}

	for _, id := range ids {
		parentID, hasParent := a.agentMgr.Parent(id)
		children := a.agentMgr.Children(id)
		status := "running"
		if id == a.activeAgent {
			status = "active"
		}

		info := fmt.Sprintf("  %s [%s]", id, status)
		if hasParent {
			info += fmt.Sprintf(" parent=%s", parentID)
		}
		if len(children) > 0 {
			info += fmt.Sprintf(" children=%d", len(children))
		}
		a.addCommandStatus(info)
	}
}

func (a *App) switchAgent(id agentpkg.AgentID) {
	if a.agentMgr == nil {
		a.addCommandError("AgentManager not initialized")
		return
	}

	_, ok := a.agentMgr.Get(id)
	if !ok {
		a.addCommandError(fmt.Sprintf("Agent %s not found", id))
		return
	}

	a.activeAgent = id
	a.addCommandStatus(fmt.Sprintf("Focused agent tab: %s", id), "Input still goes to the main agent; use subagent_send for follow-up instructions.")
}

func (a *App) handleDelegateCommand(parts []string) {
	if len(parts) < 2 || parts[1] == "status" {
		state := "OFF"
		if a.delegateMode {
			state = "ON"
		}
		a.addCommandStatus(fmt.Sprintf("Delegation mode: %s", state))
		return
	}
	if a.isThinking {
		a.addCommandError("Cannot change delegation mode while the agent is running.")
		return
	}
	switch parts[1] {
	case "on":
		if a.agentMgr == nil {
			a.addCommandError("AgentManager not initialized")
			return
		}
		agent.RegisterDelegateSubAgentTool(a.registry, a.agentMgr)
		a.delegateMode = true
		a.resetAgent(fmt.Errorf("delegate mode changed"))
		a.addCommandStatus("Delegation mode: ON")
	case "off":
		a.registry.Remove("delegate_subagent")
		a.resetAgent(fmt.Errorf("delegate mode changed"))
		a.delegateMode = false
		a.addCommandStatus("Delegation mode: OFF")
	default:
		a.addCommandError("Usage: /delegate [on|off|status]")
	}
}

// handleAllowEditPathCommand manages the auto-edit path whitelist (allow.json).
func (a *App) handleAllowEditPathCommand(parts []string) {
	if a.allow == nil {
		a.allow = config.LoadAllow()
	}
	if len(parts) < 2 {
		paths := a.allow.EditPathList()
		if len(paths) == 0 {
			a.addCommandStatus("Auto-edit path whitelist is empty. Usage: /alloweditpath add|remove <glob>|clear")
			return
		}
		var sb strings.Builder
		sb.WriteString("Auto-edit path whitelist (agent mode):\n")
		for _, p := range paths {
			sb.WriteString(fmt.Sprintf("  %s\n", p))
		}
		a.addCommandStatus(strings.TrimRight(sb.String(), "\n"))
		return
	}
	switch parts[1] {
	case "add":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /alloweditpath add <glob>")
			return
		}
		glob := strings.Join(parts[2:], " ")
		if !a.allow.AddEditPath(glob) {
			a.addCommandStatus(fmt.Sprintf("Already in whitelist: %s", glob))
			return
		}
		if err := a.allow.SaveProject(); err != nil {
			a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
			return
		}
		a.addCommandStatus(fmt.Sprintf("\u2705 Added to auto-edit whitelist: %s", glob))
	case "remove", "rm":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /alloweditpath remove <glob>")
			return
		}
		glob := strings.Join(parts[2:], " ")
		if !a.allow.RemoveEditPath(glob) {
			a.addCommandStatus(fmt.Sprintf("Not in whitelist: %s", glob))
			return
		}
		if err := a.allow.SaveProject(); err != nil {
			a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
			return
		}
		a.addCommandStatus(fmt.Sprintf("\u2705 Removed from auto-edit whitelist: %s", glob))
	case "clear":
		a.allow.ClearEditPaths()
		if err := a.allow.SaveProject(); err != nil {
			a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
			return
		}
		a.addCommandStatus("\u2705 Auto-edit path whitelist cleared")
	default:
		a.addCommandError("Usage: /alloweditpath [add <glob>|remove <glob>|clear]")
	}
}

// handleAllowAutoEditCommand toggles full auto-edit in agent mode (allow.json).
// With a trailing "global" argument the autoEdit flag is persisted to the
// global allow.json instead of the project one.
func (a *App) handleAllowAutoEditCommand(parts []string) {
	if a.allow == nil {
		a.allow = config.LoadAllow()
	}
	if len(parts) < 2 {
		state := "OFF"
		if a.allow.GetAutoEdit() {
			state = "ON"
		}
		a.addCommandStatus(fmt.Sprintf("Auto-edit (agent mode): %s", state))
		a.addCommandStatus("Usage: /allowautoedit [on|off] [global]")
		return
	}
	globalScope := false
	for _, p := range parts[2:] {
		if p == "global" {
			globalScope = true
		}
	}
	var enable bool
	switch parts[1] {
	case "on":
		enable = true
	case "off":
		enable = false
	default:
		a.addCommandError("Usage: /allowautoedit [on|off] [global]")
		return
	}
	var err error
	scope := "project"
	effective := enable
	if globalScope {
		scope = "global"
		effective = a.allow.SetGlobalAutoEdit(enable)
		err = a.allow.SaveGlobalAutoEditValue(enable)
	} else {
		a.allow.SetProjectAutoEdit(enable)
		err = a.allow.SaveProject()
	}
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to save allow.json: %v", err))
		return
	}
	state := "OFF"
	if enable {
		state = "ON"
	}
	msg := fmt.Sprintf("\u2705 Auto-edit (agent mode): %s [%s]", state, scope)
	if globalScope && effective != enable {
		effectiveState := "OFF"
		if effective {
			effectiveState = "ON"
		}
		msg += fmt.Sprintf(" (effective here: %s due to project override)", effectiveState)
	}
	a.addCommandStatus(msg)
}

func (a *App) destroyAgent(id agentpkg.AgentID) {
	if a.agent != nil && id == a.agent.ID() {
		a.addCommandError("Cannot destroy the main agent")
		return
	}

	if a.agentMgr == nil {
		a.addCommandError("AgentManager not initialized")
		return
	}

	if err := a.agentMgr.Destroy(id); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to destroy agent %s: %v", id, err))
		return
	}

	// If we destroyed the active agent, switch to main
	if a.activeAgent == id {
		a.activeAgent = "main"
	}

	a.addCommandStatus(fmt.Sprintf("Agent %s destroyed", id))
}

// handleCronCommand handles /cron subcommands (multi-agent mode).
func (a *App) handleCronCommand(parts []string) {
	if !a.multiAgent {
		a.addCommandError("Cron commands require multi-agent mode. Restart with --multi-agent to enable.")
		return
	}
	if a.cronStore == nil {
		a.addCommandError("Cron store not initialized.")
		return
	}
	if len(parts) < 2 {
		a.addCommandStatus("Usage: /cron add|list|enable|disable|remove|run")
		return
	}
	switch parts[1] {
	case "add":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /cron add <description>")
			return
		}
		desc := strings.Join(parts[2:], " ")
		job, err := a.cronStore.Create(cron.CronJob{
			Name:    desc,
			Prompt:  desc,
			Enabled: true,
			Mode:    a.mode,
		})
		if err != nil {
			a.addCommandError(fmt.Sprintf("Failed to create cron task: %v", err))
			return
		}
		a.addCommandStatus(fmt.Sprintf("✅ Cron task created: %s (id: %s)", job.Name, job.ID))
	case "list":
		jobs, err := a.cronStore.List()
		if err != nil {
			a.addCommandError(fmt.Sprintf("Failed to list cron tasks: %v", err))
			return
		}
		if len(jobs) == 0 {
			a.addCommandStatus("Cron tasks: (none configured)")
			return
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Cron tasks (%d):\n", len(jobs)))
		for _, j := range jobs {
			status := "✅"
			if !j.Enabled {
				status = "⏸"
			}
			if j.LastStatus == "failed" {
				status = "❌"
			}
			sb.WriteString(fmt.Sprintf("  %s [%s] %s (runs: %d)\n", status, j.ID, j.Name, j.RunCount))
		}
		a.addCommandStatus(sb.String())
	case "enable":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /cron enable <id>")
			return
		}
		job, err := a.cronStore.Get(parts[2])
		if err != nil {
			a.addCommandError(fmt.Sprintf("%v", err))
			return
		}
		job.Enabled = true
		a.cronStore.Update(*job)
		a.addCommandStatus(fmt.Sprintf("✅ Cron task %s enabled", job.ID))
	case "disable":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /cron disable <id>")
			return
		}
		job, err := a.cronStore.Get(parts[2])
		if err != nil {
			a.addCommandError(fmt.Sprintf("%v", err))
			return
		}
		job.Enabled = false
		a.cronStore.Update(*job)
		a.addCommandStatus(fmt.Sprintf("⏸ Cron task %s disabled", job.ID))
	case "remove":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /cron remove <id>")
			return
		}
		if err := a.cronStore.Delete(parts[2]); err != nil {
			a.addCommandError(fmt.Sprintf("%v", err))
			return
		}
		a.addCommandStatus(fmt.Sprintf("🗑 Cron task %s removed", parts[2]))
	case "run":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /cron run <id>")
			return
		}
		job, err := a.cronStore.Get(parts[2])
		if err != nil {
			a.addCommandError(fmt.Sprintf("%v", err))
			return
		}
		if a.scheduler == nil {
			a.addCommandError("Scheduler not running.")
			return
		}
		// Trigger immediate run by resetting LastRun
		job.LastRun = time.Time{}
		a.cronStore.Update(*job)
		a.addCommandStatus(fmt.Sprintf("▶ Cron task %s triggered (will run on next scheduler tick)", job.ID))
	default:
		a.addCommandError(fmt.Sprintf("Unknown cron command: %s", parts[1]))
	}
}

func (a *App) handleWorkflowsCommand(parts []string) {
	store := workflow.DefaultStore()
	sub := "list"
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}
	switch sub {
	case "list", "ls":
		runs, err := store.List(context.Background())
		if err != nil {
			a.addCommandError(fmt.Sprintf("Failed to list workflows: %v", err))
			return
		}
		if len(runs) == 0 {
			a.addCommandStatus("Workflow runs: (none)")
			return
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Workflow runs (%d):\n", len(runs)))
		for _, run := range runs {
			sb.WriteString(fmt.Sprintf("  [%s] %s %s (%s)\n", run.Status, run.ID, run.Name, run.UpdatedAt.Format(time.RFC3339)))
		}
		a.addCommandStatus(strings.TrimRight(sb.String(), "\n"))
	case "show":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /workflows show <id>")
			return
		}
		run, err := store.Load(context.Background(), parts[2])
		if err != nil {
			a.addCommandError(fmt.Sprintf("Failed to load workflow: %v", err))
			return
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Workflow %s: %s\n", run.ID, run.Status))
		if run.Name != "" {
			sb.WriteString(fmt.Sprintf("Name: %s\n", run.Name))
		}
		for _, phase := range run.Phases {
			sb.WriteString(fmt.Sprintf("Phase [%s] %s tasks=%d\n", phase.Status, phase.Name, len(phase.Tasks)))
		}
		for key, result := range run.Results {
			sb.WriteString(fmt.Sprintf("\n%s [%s]\n%s\n", key, result.Status, strings.TrimSpace(result.Result)))
		}
		if run.Error != "" {
			sb.WriteString("\nError: " + run.Error)
		}
		a.addCommandStatus(strings.TrimRight(sb.String(), "\n"))
	case "cancel":
		if len(parts) < 3 {
			a.addCommandStatus("Usage: /workflows cancel <id>")
			return
		}
		id := strings.TrimSpace(parts[2])
		if !workflow.DefaultActiveRegistry().Cancel(id) {
			a.addCommandError(fmt.Sprintf("Workflow run %s is not active.", id))
			return
		}
		a.addCommandStatus(fmt.Sprintf("Workflow run %s cancellation requested.", id))
	default:
		a.addCommandError("Usage: /workflows [list|show <id>|cancel <id>]")
	}
}

// handleSystemInitCommand generates (or refreshes) a project AGENTS.md. In
// plan mode it first switches to agent mode (AGENTS.md needs write access). In
// interactive modes (plan/agent) the agent is told to use the question tool to
// clarify project conventions with the user before writing the file.
func (a *App) handleSystemInitCommand(cmd string) tea.Cmd {
	if a.isThinking {
		a.addCommandError("Cannot run /systeminit while the agent is running.")
		return nil
	}
	if a.manualCompactionActive {
		a.addCommandError("Cannot run /systeminit while context compaction is running.")
		return nil
	}
	extra := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cmd), "/systeminit"))
	if a.mode == "plan" {
		a.mode = "agent"
		a.resetAgent(fmt.Errorf("systeminit requires write access"))
		a.addCommandStatus("Switched to AGENT mode for /systeminit (AGENTS.md needs write access).")
	}
	// The question tool is only available in plan/agent modes, so only request
	// interactive clarification when not in yolo mode.
	interactive := a.mode != "yolo"
	if interactive {
		a.addCommandStatus("\U0001F6E0 /systeminit: analyzing the project; I'll ask a few questions, then write AGENTS.md.")
	} else {
		a.addCommandStatus("\U0001F6E0 /systeminit: analyzing the project and writing AGENTS.md...")
	}
	return a.submitAgentPrompt(systeminit.Prompt(interactive, extra))
}

// handleReloadCommand starts a brand-new session and re-execs the process so
// the next run behaves exactly like a freshly started program (config, context
// files, skills, and MCP are all reloaded).
func (a *App) handleReloadCommand() tea.Cmd {
	if a.isThinking && a.agent != nil {
		a.abortAndResetAgent("reload")
		a.isThinking = false
		a.finishRequestTimer()
	}
	a.reloadRequested = true
	a.addCommandStatus("\u21bb Reloading: starting a fresh process with a new session...")
	return tea.Quit
}

func (a *App) handleCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	if strings.HasPrefix(parts[0], "/skill:") {
		skillName := strings.TrimPrefix(parts[0], "/skill:")
		if skillName != "" {
			a.activateSkill(skillName)
		} else {
			a.listSkills()
		}
		return nil
	}

	command := parts[0]

	switch command {
	case "/mode":
		if len(parts) > 1 {
			switch parts[1] {
			case "plan", "agent", "yolo":
				a.mode = parts[1]
				// If agent is currently running, abort it so the new mode takes effect immediately
				if a.isThinking && a.agent != nil {
					a.pendingAbortReason = "mode change"
					a.abortAndResetAgent("mode changed")
					a.clearQueuedInput()
					a.isThinking = false
					a.finishRequestTimer()
					a.addCommandStatus("⏹ Aborted (mode change)")
				} else {
					a.resetAgent(fmt.Errorf("mode changed"))
				}
				a.addCommandStatus(fmt.Sprintf("Mode: %s", strings.ToUpper(a.mode)))
			default:
				a.addCommandError("Invalid mode")
			}
		} else {
			a.addCommandStatus(fmt.Sprintf("Current mode: %s", strings.ToUpper(a.mode)))
			switch a.mode {
			case "plan":
				a.addCommandStatus("  Permissions: READ only (no modifications)")
			case "agent":
				a.addCommandStatus("  Permissions: READ/WRITE/EDIT auto | BASH requires approval")
			case "yolo":
				a.addCommandStatus("  Permissions: ALL tools auto-execute")
			}
		}
	case "/model":
		if len(parts) > 1 {
			// Switch model
			modelID := parts[1]
			newModel := a.provider.GetModel(modelID)
			if newModel == nil {
				models := a.provider.Models()
				ids := make([]string, len(models))
				for i, m := range models {
					ids[i] = m.ID
				}
				a.addCommandError(fmt.Sprintf("Model %q not found — available: %s", modelID, strings.Join(ids, ", ")))
				return nil
			}
			a.model = newModel
			// Reset agent so next message uses the new model
			a.resetAgent(fmt.Errorf("model changed"))
			a.addCommandStatus(fmt.Sprintf("✅ Model switched to: %s (%s)", newModel.Name, newModel.ID))
		} else {
			// Show current model and available models
			a.addCommandStatus(fmt.Sprintf("Current model: %s (%s)", a.model.Name, a.model.ID))
			models := a.provider.Models()
			if len(models) > 0 {
				var sb strings.Builder
				sb.WriteString("Available models (use /model <id> to switch):\n")
				for _, m := range models {
					marker := " "
					if m.ID == a.model.ID {
						marker = "*"
					}
					sb.WriteString(fmt.Sprintf("  [%s] %s (%s)\n", marker, m.Name, m.ID))
				}
				a.addCommandStatus(sb.String())
			}
		}
	case "/auth":
		if a.isThinking {
			a.addCommandError("Cannot open /auth while the agent is running.")
		} else {
			a.openAuthDialog()
		}
	case "/skills":
		a.listSkills()
	case "/skill":
		if len(parts) > 1 {
			a.activateSkill(parts[1])
		} else {
			a.listSkills()
		}
	case "/compact":
		if a.isThinking {
			a.addCommandError("Cannot compact while the agent is running.")
		} else if a.agent == nil {
			a.addCommandError("Nothing to compact: no active conversation.")
		} else {
			msgs := a.agent.GetMessages()
			if len(msgs) < 2 {
				a.addCommandError("Nothing to compact: conversation is too short.")
			} else if !a.agent.CanCompact() {
				a.addCommandError("Nothing to compact: only recent context is available to keep.")
			} else {
				return a.startManualCompaction()
			}
		}
	case "/clear":
		a.resetTranscriptState()
		a.resetAgent(fmt.Errorf("conversation cleared"))
		a.contextUsage = nil
		a.totalInputTokens = 0
		a.totalCacheRead = 0
		a.totalCacheWrite = 0
		a.pastes = make(map[int]string)
		a.pasteCounter = 0
		a.activeSkills = make(map[string]string)
		a.markBuiltinActiveSkills()
		a.extraContext = a.baseExtraContext
		a.updateViewportContent()
		a.printedMessageIdx = make(map[int]bool)
		a.addCommandStatus("✅ Conversation cleared")

	case "/quit":
		return tea.Quit
	case "/sessions":
		a.handleSessionsCommand(parts)
	case "/init_mcp":
		a.handleInitMCPCommand(parts)
	case "/mcps":
		a.handleMCPsCommand()
	case "/agent":
		a.handleAgentCommand(parts)
	case "/delegate":
		a.handleDelegateCommand(parts)
	case "/alloweditpath":
		a.handleAllowEditPathCommand(parts)
	case "/allowautoedit":
		a.handleAllowAutoEditCommand(parts)
	case "/btw":
		return a.handleBtwCommand(cmd)
	case "/systeminit":
		return a.handleSystemInitCommand(cmd)
	case "/reload":
		return a.handleReloadCommand()
	case "/cron":
		a.handleCronCommand(parts)
	case "/workflows":
		a.handleWorkflowsCommand(parts)
	case "/statusline":
		a.handleStatusLineCommand(parts)
	case "/help":
		a.addCommandStatus(commandHelpText())
	default:
		a.addCommandError(fmt.Sprintf("Unknown: %s", command))
	}

	return nil
}

func (a *App) handleStatusLineCommand(parts []string) {
	sub := "status"
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}
	switch sub {
	case "status":
		a.showStatusLineStatus()
	case "on", "off":
		scope := "project"
		if len(parts) > 2 {
			scope = strings.ToLower(parts[2])
		}
		if scope != "project" && scope != "global" {
			a.addCommandError("Usage: /statusline [status|on|off] [project|global]")
			return
		}
		a.toggleStatusLine(sub == "on", scope)
	case "command":
		a.setStatusLineCommand(parts)
	case "refresh":
		a.setStatusLineRefresh(parts)
	default:
		a.addCommandError("Usage: /statusline [status|on|off|command|refresh] ...")
	}
}

func (a *App) showStatusLineStatus() {
	cfg := a.statusLineConfig()
	if !cfg.Enabled {
		a.addCommandStatus("Status line: OFF", "Footer: builtin")
		return
	}

	var lines []string
	lines = append(lines, "Status line: ON")
	lines = append(lines, fmt.Sprintf("  Type: %s", cfg.Type))
	lines = append(lines, fmt.Sprintf("  Command: %s", strings.TrimSpace(cfg.Command)))
	lines = append(lines, fmt.Sprintf("  Timeout: %dms", tuistatusline.Timeout(cfg).Milliseconds()))
	if cfg.RefreshInterval > 0 {
		lines = append(lines, fmt.Sprintf("  Refresh: %ds", cfg.RefreshInterval))
	} else {
		lines = append(lines, "  Refresh: event-driven")
	}
	if a.statusLineInFlight {
		lines = append(lines, "  Render: running")
	} else if strings.TrimSpace(a.statusLineOutput) != "" {
		lines = append(lines, "  Render: external footer active")
	} else {
		lines = append(lines, "  Render: builtin footer fallback")
	}
	if a.statusLineLastError != "" {
		lines = append(lines, "  Last error: "+a.statusLineLastError)
	}
	a.addCommandStatus(lines...)
}

func (a *App) toggleStatusLine(enabled bool, scope string) {
	s, err := loadStatusLineSettings(scope)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load %s settings: %v", scope, err))
		return
	}

	if enabled {
		if s.StatusLine.Type == "" {
			s.StatusLine.Type = "command"
		}
		if strings.TrimSpace(s.StatusLine.Command) == "" {
			s.StatusLine.Command = "ccstatusline"
		}
		if s.StatusLine.TimeoutMs == 0 {
			s.StatusLine.TimeoutMs = 800
		}
		if s.StatusLine.Fallback == "" {
			s.StatusLine.Fallback = "builtin"
		}
	}
	s.StatusLine.Enabled = enabled

	if err := saveStatusLineSettings(scope, s); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to save %s settings: %v", scope, err))
		return
	}

	if a.settings == nil {
		a.settings = config.DefaultSettings()
	}
	a.settings.StatusLine = s.StatusLine
	if !enabled {
		a.statusLineOutput = ""
		a.statusLineLastError = ""
		a.statusLineLastSuccess = ""
		a.statusLineLastAttempt = ""
		a.statusLinePending = nil
		a.statusLineInFlight = false
	} else if a.ready && a.width > 0 {
		a.requestStatusLineRefresh(true)
	}
	a.scheduleRender()

	state := "OFF"
	if enabled {
		state = "ON"
	}
	a.addCommandStatus(fmt.Sprintf("Status line: %s (%s settings)", state, scope))
}

func (a *App) setStatusLineCommand(parts []string) {
	if len(parts) < 3 {
		a.addCommandError("Usage: /statusline command <cmd> [project|global]")
		return
	}
	scope := "project"
	end := len(parts)
	last := strings.ToLower(parts[len(parts)-1])
	if last == "project" || last == "global" {
		scope = last
		end--
	}
	cmd := strings.TrimSpace(strings.Join(parts[2:end], " "))
	if cmd == "" {
		a.addCommandError("Usage: /statusline command <cmd> [project|global]")
		return
	}
	s, err := loadStatusLineSettings(scope)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load %s settings: %v", scope, err))
		return
	}
	s.StatusLine.Type = "command"
	s.StatusLine.Command = cmd
	if s.StatusLine.TimeoutMs == 0 {
		s.StatusLine.TimeoutMs = 800
	}
	if s.StatusLine.Fallback == "" {
		s.StatusLine.Fallback = "builtin"
	}
	if err := saveStatusLineSettings(scope, s); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to save %s settings: %v", scope, err))
		return
	}
	if a.settings == nil {
		a.settings = config.DefaultSettings()
	}
	a.settings.StatusLine = s.StatusLine
	if a.settings.StatusLine.Enabled && a.ready && a.width > 0 {
		a.requestStatusLineRefresh(true)
	}
	a.scheduleRender()
	a.addCommandStatus(fmt.Sprintf("Status line command updated (%s settings): %s", scope, cmd))
}

func (a *App) setStatusLineRefresh(parts []string) {
	if len(parts) < 3 {
		a.addCommandError("Usage: /statusline refresh <sec> [project|global]")
		return
	}
	scope := "project"
	valueIdx := 2
	if len(parts) > 3 {
		last := strings.ToLower(parts[len(parts)-1])
		if last == "project" || last == "global" {
			scope = last
		}
	}
	refreshStr := parts[valueIdx]
	refresh, err := strconv.Atoi(refreshStr)
	if err != nil || refresh < 0 || refresh > 60 {
		a.addCommandError("Usage: /statusline refresh <0-60> [project|global]")
		return
	}
	s, err := loadStatusLineSettings(scope)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load %s settings: %v", scope, err))
		return
	}
	s.StatusLine.RefreshInterval = refresh
	if err := saveStatusLineSettings(scope, s); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to save %s settings: %v", scope, err))
		return
	}
	if a.settings == nil {
		a.settings = config.DefaultSettings()
	}
	a.settings.StatusLine = s.StatusLine
	a.statusLineIntervalInit = false
	if a.statusLineEnabled() && refresh > 0 && a.program != nil {
		a.program.Send(statusLineTickMsg(time.Now()))
	}
	a.scheduleRender()
	if refresh == 0 {
		a.addCommandStatus(fmt.Sprintf("Status line refresh updated (%s settings): event-driven", scope))
		return
	}
	a.addCommandStatus(fmt.Sprintf("Status line refresh updated (%s settings): %ds", scope, refresh))
}

func loadStatusLineSettings(scope string) (*config.Settings, error) {
	switch scope {
	case "global":
		return config.LoadGlobalSettingsSparse()
	default:
		return config.LoadProjectSettingsSparse()
	}
}

func saveStatusLineSettings(scope string, s *config.Settings) error {
	switch scope {
	case "global":
		return config.SaveGlobalSettings(s)
	default:
		return config.SaveProjectSettings(s)
	}
}

// listSkills displays all available skills.
func (a *App) listSkills() {
	if a.skillsMgr == nil {
		a.addCommandStatus("No skills manager available.")
		return
	}
	skillList := a.skillsMgr.List()
	if len(skillList) == 0 {
		a.addCommandStatus("No skills found.")
		return
	}

	var sb strings.Builder
	sb.WriteString("Available skills:\n")
	for _, s := range skillList {
		marker := " "
		if _, ok := a.activeSkills[s.Name]; ok {
			marker = "*"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s (%s): %s\n", marker, s.Name, s.Source, s.Description))
	}
	sb.WriteString("\nUse /skill <name> or /skill:<name> to activate a skill.")
	a.addCommandStatus(sb.String())
}

// activateSkill loads a skill's content into the extra context.
func (a *App) activateSkill(name string) {
	if a.skillsMgr == nil {
		a.addCommandError("No skills manager available.")
		return
	}
	skill := a.skillsMgr.Get(name)
	if skill == nil {
		a.addCommandError(fmt.Sprintf("Skill not found: %s", name))
		return
	}

	// Check if already active
	if _, ok := a.activeSkills[name]; ok {
		a.addCommandStatus(fmt.Sprintf("Skill '%s' is already active.", name))
		return
	}

	// Add skill content to active skills
	skillCtx := a.skillsMgr.BuildSkillContext(name)
	a.activeSkills[name] = skillCtx

	// Rebuild extraContext from base + all active skills
	a.rebuildExtraContext()

	// Reset agent so next message uses the updated context
	a.resetAgent(fmt.Errorf("skill activated"))

	a.addCommandStatus(fmt.Sprintf("✅ Skill '%s' activated (%s): %s", name, skill.Source, skill.Description))
}

// rebuildExtraContext rebuilds extraContext from base context + all active skills.
func (a *App) rebuildExtraContext() {
	sb := strings.Builder{}
	sb.WriteString(a.baseExtraContext)
	for _, ctx := range a.activeSkills {
		sb.WriteString(ctx)
	}
	a.extraContext = sb.String()
}

func (a *App) markBuiltinActiveSkills() {
	if !a.workflows || a.skillsMgr == nil {
		return
	}
	if a.skillsMgr.Get(workflow.SkillName) == nil {
		return
	}
	if a.activeSkills == nil {
		a.activeSkills = make(map[string]string)
	}
	if _, ok := a.activeSkills[workflow.SkillName]; !ok {
		a.activeSkills[workflow.SkillName] = ""
	}
}

// getSessionDir returns the session directory path.
func (a *App) getSessionDir() string {
	if a.settings != nil {
		return a.settings.GetSessionDir()
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".vibecoding", "sessions")
}

// getCurrentSessionID returns the current session's short ID (first 8 chars).
func (a *App) getCurrentSessionID() string {
	if a.session == nil {
		return ""
	}
	file := a.session.GetFile()
	if file == "" {
		return ""
	}
	base := filepath.Base(file)
	base = strings.TrimSuffix(base, ".db")
	if idx := strings.Index(base, "_"); idx >= 0 {
		return base[idx+1:]
	}
	return ""
}

// handleSessionsCommand handles the /sessions command and its subcommands.
func (a *App) handleSessionsCommand(parts []string) {
	sub := "ls"
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}

	switch sub {
	case "ls", "list":
		a.sessionsList()
	case "set", "switch", "use":
		if len(parts) < 3 {
			a.addCommandError("Usage: /sessions set <id>")
			return
		}
		a.sessionsSet(parts[2])
	case "clear", "new":
		a.sessionsClear()
	case "del", "delete", "rm":
		if len(parts) < 3 {
			a.addCommandError("Usage: /sessions del <id>")
			return
		}
		a.sessionsDel(parts[2])
	default:
		a.addCommandError(fmt.Sprintf("Unknown subcommand: %s. Use ls, set, clear, del.", sub))
	}
}

// sessionsList lists all sessions for the current project directory.
func (a *App) sessionsList() {
	cwd := ""
	if a.session != nil && a.session.GetHeader() != nil {
		cwd = a.session.GetHeader().Cwd
	}
	if cwd == "" {
		if w, err := os.Getwd(); err == nil {
			cwd = w
		}
	}

	sessionDir := a.getSessionDir()
	details, err := session.ListForDirDetailed(cwd, sessionDir)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Error listing sessions: %v", err))
		return
	}

	if len(details) == 0 {
		a.addCommandStatus("No sessions found for this project.")
		return
	}

	currentID := a.getCurrentSessionID()

	var sb strings.Builder
	sb.WriteString("Sessions for this project:\n\n")
	for _, d := range details {
		marker := " "
		if d.ID == currentID {
			marker = "*"
		}
		age := formatAge(d.ModTime)
		preview := ""
		if d.Preview != "" {
			preview = " - " + d.Preview
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s  %d msgs  %s%s\n",
			marker, d.ID, d.MessageCount, age, preview))
	}
	sb.WriteString("\nUse /sessions set <id> to switch. * = current session.")
	a.addCommandStatus(sb.String())
}

// sessionsSet switches to a different session by ID prefix.
func (a *App) sessionsSet(id string) {
	cwd := ""
	if a.session != nil && a.session.GetHeader() != nil {
		cwd = a.session.GetHeader().Cwd
	}
	if cwd == "" {
		if w, err := os.Getwd(); err == nil {
			cwd = w
		}
	}

	// Don't switch to the same session
	if id == a.getCurrentSessionID() {
		a.addCommandStatus("Already on this session.")
		return
	}

	sessionDir := a.getSessionDir()
	details, err := session.ListForDirDetailed(cwd, sessionDir)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Error: %v", err))
		return
	}

	// Find matching session by ID prefix
	var match *session.SessionDetail
	for i, d := range details {
		if strings.HasPrefix(d.ID, id) {
			if match != nil {
				a.addCommandError(fmt.Sprintf("Ambiguous ID '%s'. Be more specific.", id))
				return
			}
			match = &details[i]
		}
	}

	if match == nil {
		a.addCommandError(fmt.Sprintf("No session found matching '%s'.", id))
		return
	}

	// Open the session
	newSess, err := session.OpenByID(cwd, sessionDir, match.ID)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Error opening session: %v", err))
		return
	}

	// Switch session
	a.session = newSess
	a.historyLoaded = false

	// Reset agent and UI state
	a.resetAgent(fmt.Errorf("session changed"))
	a.resetTranscriptState()
	a.contextUsage = nil
	a.totalInputTokens = 0
	a.totalCacheRead = 0
	a.totalCacheWrite = 0

	// Load history messages from the new session
	a.LoadHistoryMessages()
	// Recompute context usage so the status bar reflects the switched session
	// immediately instead of waiting for the next agent turn.
	a.contextUsage = a.computeContextUsage()
	a.updateViewportContent()
	for idx := range a.messages {
		a.printMessageOnce(idx)
	}

	a.addCommandStatus(fmt.Sprintf("✅ Switched to session %s (%d msgs)",
		match.ID, match.MessageCount))
}

func (a *App) handleInitMCPCommand(parts []string) {
	target := "project"
	template := "full"
	force := false

	for _, p := range parts[1:] {
		switch strings.ToLower(p) {
		case "project", "global":
			target = strings.ToLower(p)
		case "basic", "full":
			template = strings.ToLower(p)
		case "--force":
			force = true
		default:
			a.addCommandError("Usage: /init_mcp [project|global] [basic|full] [--force]")
			return
		}
	}

	path := config.ProjectMCPPath()
	if target == "global" {
		path = config.GlobalMCPPath()
	}

	if !force {
		if _, err := os.Stat(path); err == nil {
			a.addCommandStatus(fmt.Sprintf("MCP config already exists: %s (use --force to overwrite)", path))
			return
		}
	}

	var cfg *config.MCPConfig
	if template == "basic" {
		cfg = config.DefaultMCPConfig()
	} else {
		cfg = config.FullMCPConfigTemplate()
	}

	if err := config.SaveMCPConfig(path, cfg); err != nil {
		a.addCommandError(fmt.Sprintf("Init MCP config failed: %v", err))
		return
	}
	a.addCommandStatus(fmt.Sprintf("✅ Created MCP config: %s", path))
	a.addCommandStatus(fmt.Sprintf("Template: %s | Target: %s", template, target))
}

func (a *App) handleMCPsCommand() {
	type sourceInfo struct {
		label string
		path  string
	}
	sources := []sourceInfo{
		{label: "Global", path: config.GlobalMCPPath()},
		{label: "Project", path: config.ProjectMCPPath()},
	}

	var sb strings.Builder
	sb.WriteString("MCP servers:\n")
	foundAny := false

	for _, src := range sources {
		sb.WriteString(fmt.Sprintf("\n%s (%s):\n", src.label, src.path))
		cfg, err := config.LoadMCPConfig(src.path)
		if err != nil {
			if os.IsNotExist(err) {
				sb.WriteString("  (not configured)\n")
				continue
			}
			sb.WriteString(fmt.Sprintf("  (invalid: %v)\n", err))
			continue
		}
		config.NormalizeMCPConfig(cfg)
		if len(cfg.MCPServers) == 0 {
			sb.WriteString("  (empty)\n")
			continue
		}
		for _, srv := range cfg.MCPServers {
			foundAny = true
			target := srv.Command
			if target == "" {
				target = srv.URL
			}
			if target == "" {
				target = "-"
			}
			sb.WriteString(fmt.Sprintf("  - %s [%s] %s\n", srv.Name, srv.Type, target))
		}
	}

	if !foundAny {
		sb.WriteString("\nUse /init_mcp to create project mcp.json.")
	}
	a.addCommandStatus(sb.String())
}

// sessionsClear creates a new session, starting fresh.
func (a *App) sessionsClear() {
	cwd := ""
	if a.session != nil && a.session.GetHeader() != nil {
		cwd = a.session.GetHeader().Cwd
	}
	if cwd == "" {
		if w, err := os.Getwd(); err == nil {
			cwd = w
		}
	}

	sessionDir := a.getSessionDir()
	newSess := session.New(cwd, sessionDir)
	if err := newSess.Init(); err != nil {
		a.addCommandError(fmt.Sprintf("Error creating session: %v", err))
		return
	}

	a.session = newSess
	a.historyLoaded = false

	// Reset agent and UI state
	a.resetAgent(fmt.Errorf("session changed"))
	a.resetTranscriptState()
	a.contextUsage = nil
	a.totalInputTokens = 0
	a.totalCacheRead = 0
	a.totalCacheWrite = 0
	a.updateViewportContent()

	a.addCommandStatus("✅ New session created.")
}

// sessionsDel deletes a session by ID prefix.
func (a *App) sessionsDel(id string) {
	cwd := ""
	if a.session != nil && a.session.GetHeader() != nil {
		cwd = a.session.GetHeader().Cwd
	}
	if cwd == "" {
		if w, err := os.Getwd(); err == nil {
			cwd = w
		}
	}

	// Don't delete the current session
	if id == a.getCurrentSessionID() {
		a.addCommandError("Cannot delete the current session. Switch to another session first, or use /sessions clear to start fresh.")
		return
	}

	sessionDir := a.getSessionDir()
	details, err := session.ListForDirDetailed(cwd, sessionDir)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Error: %v", err))
		return
	}

	// Find matching session by ID prefix
	var match *session.SessionDetail
	for i, d := range details {
		if strings.HasPrefix(d.ID, id) {
			if match != nil {
				a.addCommandError(fmt.Sprintf("Ambiguous ID '%s'. Be more specific.", id))
				return
			}
			match = &details[i]
		}
	}

	if match == nil {
		a.addCommandError(fmt.Sprintf("No session found matching '%s'.", id))
		return
	}

	if err := session.DeleteSession(match.Path, a.settings.GetSessionDir()); err != nil {
		a.addCommandError(fmt.Sprintf("Error deleting session: %v", err))
		return
	}

	a.addCommandStatus(fmt.Sprintf("✅ Deleted session %s.", match.ID))
}

// formatAge returns a human-readable age string for a time.
func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}
