package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/agent"
	browserfeature "github.com/startvibecoding/mothx/internal/browser"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/contextfiles"
	"github.com/startvibecoding/mothx/internal/cron"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/systeminit"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/workflow"
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

func (a *App) handleBrowserCommand(parts []string) {
	if len(parts) > 2 {
		a.addCommandError("Usage: /browser [on|off|status]")
		return
	}
	sub := "status"
	if len(parts) == 2 {
		sub = strings.ToLower(parts[1])
	}
	switch sub {
	case "status":
		state := "OFF"
		if a.browserEnabled && browserfeature.IsToolRegistered(a.registry) {
			state = "ON"
		}
		a.addCommandStatus(fmt.Sprintf("Browser tool: %s", state), "Usage: /browser [on|off|status]")
	case "on":
		a.enableBrowserTool()
	case "off":
		a.disableBrowserTool()
	default:
		a.addCommandError("Usage: /browser [on|off|status]")
	}
}

func (a *App) enableBrowserTool() {
	if a.isThinking {
		a.addCommandError("Cannot change browser tool while the agent is running.")
		return
	}
	if a.registry == nil {
		a.addCommandError("Tool registry is not initialized.")
		return
	}
	cwd := a.currentCwd()
	path, created, err := browserfeature.EnsureProjectSkill(cwd)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to create browser skill: %v", err))
		return
	}

	globalSkillsDir := ""
	if a.settings != nil {
		globalSkillsDir = a.settings.GetGlobalSkillsDir()
	}
	a.skillsMgr = skills.NewManagerWithProjectDirs(globalSkillsDir, skills.ProjectSkillDirs(cwd))
	if err := a.skillsMgr.Load(); err != nil {
		a.addCommandError(fmt.Sprintf("Failed to load skills: %v", err))
		return
	}

	a.registry.Register(tools.NewSkillRefTool(a.skillsMgr))
	browserfeature.RegisterTool(a.registry)
	a.browserEnabled = true
	a.browserSkillInBase = false
	a.browserSkillContext = a.skillsMgr.BuildSkillContext(browserfeature.SkillName)
	a.markBuiltinActiveSkills()
	a.rebuildExtraContext()
	a.resetAgent(fmt.Errorf("browser tool changed"))

	action := "Using browser skill"
	if created {
		action = "Created browser skill"
	}
	a.addCommandStatus("Browser tool: ON", fmt.Sprintf("%s: %s", action, path))
}

func (a *App) disableBrowserTool() {
	if a.isThinking {
		a.addCommandError("Cannot change browser tool while the agent is running.")
		return
	}
	browserfeature.RemoveTool(a.registry)
	a.browserEnabled = false
	delete(a.activeSkills, browserfeature.SkillName)
	if a.browserSkillInBase && a.browserSkillContext != "" {
		a.baseExtraContext = strings.Replace(a.baseExtraContext, a.browserSkillContext, "", 1)
	}
	a.browserSkillInBase = false
	a.browserSkillContext = ""
	a.rebuildExtraContext()
	a.resetAgent(fmt.Errorf("browser tool changed"))
	a.addCommandStatus("Browser tool: OFF")
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

func (a *App) handleRuleCommand(parts []string) {
	if a.isThinking {
		a.addCommandError("Cannot change /rule while the agent is running.")
		return
	}
	overwrite, ok := parseRuleForce(parts)
	if !ok {
		a.addCommandError("Usage: /rule [force|--force]")
		return
	}

	path, content, written, err := contextfiles.EnsureRuleFile(a.currentCwd(), overwrite)
	if err != nil {
		a.addCommandError(fmt.Sprintf("Failed to write rule file: %v", err))
		return
	}
	a.ruleContent = content
	a.resetAgent(fmt.Errorf("rule changed"))

	if written {
		action := "Created"
		if overwrite {
			action = "Overwrote"
		}
		a.addCommandStatus(fmt.Sprintf("%s rule file: %s", action, path), "Loaded into the current session.")
		return
	}
	a.addCommandStatus(fmt.Sprintf("Rule file already exists: %s", path), "Not overwritten. Use /rule force to replace it with the default template.", "Loaded existing rule into the current session.")
}

func parseRuleForce(parts []string) (bool, bool) {
	if len(parts) == 1 {
		return false, true
	}
	if len(parts) != 2 {
		return false, false
	}
	switch parts[1] {
	case "force", "--force":
		return true, true
	default:
		return false, false
	}
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
			// Switch model directly
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
			if a.isThinking {
				a.addCommandError("Cannot open /model while the agent is running.")
			} else {
				a.openModelDialog()
			}
		}
	case "/defaultModel":
		scope := "global"
		if len(parts) > 2 {
			a.addCommandError("Usage: /defaultModel [project|global]")
			return nil
		}
		if len(parts) == 2 {
			switch parts[1] {
			case "project", "global":
				scope = parts[1]
			default:
				a.addCommandError("Usage: /defaultModel [project|global]")
				return nil
			}
		}
		if a.isThinking {
			a.addCommandError("Cannot open /defaultModel while the agent is running.")
		} else {
			a.openDefaultModelDialog(scope)
		}
	case "/auth":
		if a.isThinking {
			a.addCommandError("Cannot open /auth while the agent is running.")
		} else {
			a.openAuthDialog()
		}
	case "/settings":
		if a.isThinking {
			a.addCommandError("Cannot open /settings while the agent is running.")
		} else {
			a.openSettingsDialog(parts[1:])
		}
	case "/skills":
		a.listSkills()
	case "/skill":
		if len(parts) > 1 {
			a.activateSkill(parts[1])
		} else {
			a.listSkills()
		}
	case "/paste-image":
		a.handlePasteImageCommand()
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
		a.rebuildExtraContext()
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
	case "/browser":
		a.handleBrowserCommand(parts)
	case "/alloweditpath":
		a.handleAllowEditPathCommand(parts)
	case "/allowautoedit":
		a.handleAllowAutoEditCommand(parts)
	case "/btw":
		return a.handleBtwCommand(cmd)
	case "/systeminit":
		return a.handleSystemInitCommand(cmd)
	case "/rule":
		a.handleRuleCommand(parts)
	case "/reload":
		return a.handleReloadCommand()
	case "/cron":
		a.handleCronCommand(parts)
	case "/workflows":
		a.handleWorkflowsCommand(parts)
	case "/statusline":
		a.handleStatusLineCommand(parts)
	case "/stats":
		return a.handleStatsCommand(parts)
	case "/help":
		a.addCommandStatus(commandHelpText())
	default:
		a.addCommandError(fmt.Sprintf("Unknown: %s", command))
	}

	return nil
}

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
	if a.skillsMgr == nil {
		return
	}
	if a.activeSkills == nil {
		a.activeSkills = make(map[string]string)
	}
	if a.workflows && a.skillsMgr.Get(workflow.SkillName) != nil {
		a.activeSkills[workflow.SkillName] = ""
	}
	if a.browserEnabled && a.skillsMgr.Get(browserfeature.SkillName) != nil {
		if a.browserSkillInBase {
			a.activeSkills[browserfeature.SkillName] = ""
		} else {
			a.activeSkills[browserfeature.SkillName] = a.skillsMgr.BuildSkillContext(browserfeature.SkillName)
		}
	}
}

// getSessionDir returns the session directory path.
