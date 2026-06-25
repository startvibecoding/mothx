package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/config"
	ctxpkg "github.com/startvibecoding/vibecoding/internal/context"
	"github.com/startvibecoding/vibecoding/internal/skills"
	"github.com/startvibecoding/vibecoding/internal/workflow"
)

// CommandResult holds the output of a slash command.
type CommandResult struct {
	Message string
	Error   bool
}

// handleCommand processes a /xxx slash command.
// Returns nil if the input is not a command (should go to agent).
func (s *Server) handleCommand(sess *GatewaySession, input string) *CommandResult {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	switch cmd {
	case "/clear":
		return s.cmdClear(sess)
	case "/mode":
		return s.cmdMode(sess, parts)
	case "/model":
		return s.cmdModel(parts)
	case "/models":
		return s.cmdModels()
	case "/sessions":
		return s.cmdSessions(parts)
	case "/status":
		return s.cmdStatus(sess)
	case "/compact":
		return s.cmdCompact(sess)
	case "/delegate":
		return s.cmdDelegate(sess, parts)
	case "/alloweditpath":
		return s.cmdAllowEditPath(parts)
	case "/allowautoedit":
		return s.cmdAllowAutoEdit(parts)
	case "/workflows":
		return s.cmdWorkflows(parts)
	case "/skill":
		return s.cmdSkill(sess, parts)
	case "/skills":
		return s.cmdSkills(sess)
	case "/help":
		return s.cmdHelp()
	default:
		return &CommandResult{Message: fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd), Error: true}
	}
}

func (s *Server) getAllow() *config.AllowConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allow == nil {
		s.allow = config.LoadAllow()
	}
	return s.allow
}

func (s *Server) cmdClear(sess *GatewaySession) *CommandResult {
	if sess == nil {
		return &CommandResult{Message: "No active session to clear.", Error: true}
	}
	// The session manager keeps persisted SQLite state, but we reset the in-memory state.
	// The caller will set agent=nil so the next request builds a fresh agent.
	return &CommandResult{Message: "✅ Conversation cleared"}
}

func (s *Server) cmdMode(sess *GatewaySession, parts []string) *CommandResult {
	if len(parts) > 1 {
		switch parts[1] {
		case "plan", "agent", "yolo":
			if sess != nil {
				sess.Mode = parts[1]
			}
			return &CommandResult{Message: fmt.Sprintf("Mode: %s", strings.ToUpper(parts[1]))}
		default:
			return &CommandResult{Message: "Invalid mode. Use: plan, agent, yolo", Error: true}
		}
	}
	mode := s.cfg.DefaultMode
	if sess != nil && sess.Mode != "" {
		mode = sess.Mode
	}
	return &CommandResult{Message: fmt.Sprintf("Current mode: %s", strings.ToUpper(mode))}
}

func (s *Server) cmdModel(parts []string) *CommandResult {
	if len(parts) > 1 {
		modelID := parts[1]
		newModel := s.provider.GetModel(modelID)
		if newModel == nil {
			return &CommandResult{Message: fmt.Sprintf("Model %q not found — available: %s", modelID, modelIDs(s.provider.Models())), Error: true}
		}
		s.mu.Lock()
		s.model = newModel
		s.mu.Unlock()
		return &CommandResult{Message: fmt.Sprintf("✅ Model switched to: %s (%s)", newModel.Name, newModel.ID)}
	}
	s.mu.RLock()
	m := s.model
	s.mu.RUnlock()
	return &CommandResult{Message: fmt.Sprintf("Current model: %s (%s)", m.Name, m.ID)}
}

func (s *Server) cmdModels() *CommandResult {
	models := s.provider.Models()
	if len(models) == 0 {
		return &CommandResult{Message: "No models available."}
	}
	var sb strings.Builder
	sb.WriteString("Available models:\n")
	s.mu.RLock()
	currentID := s.model.ID
	s.mu.RUnlock()
	for _, m := range models {
		marker := " "
		if m.ID == currentID {
			marker = "*"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s (%s)\n", marker, m.Name, m.ID))
	}
	return &CommandResult{Message: sb.String()}
}

func (s *Server) cmdSessions(parts []string) *CommandResult {
	sub := "ls"
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}
	switch sub {
	case "ls", "list":
		ids := s.pool.List()
		if len(ids) == 0 {
			return &CommandResult{Message: "No active sessions."}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Active sessions (%d):\n", len(ids)))
		for _, id := range ids {
			sb.WriteString(fmt.Sprintf("  - %s\n", id))
		}
		return &CommandResult{Message: sb.String()}
	case "clear", "new":
		return &CommandResult{Message: "✅ Use a new x_session_id to start a fresh session."}
	case "del", "delete", "rm":
		if len(parts) < 3 {
			return &CommandResult{Message: "Usage: /sessions del <id>", Error: true}
		}
		id := parts[2]
		if s.pool.Get(id) == nil {
			return &CommandResult{Message: fmt.Sprintf("Session not found: %s", id), Error: true}
		}
		s.pool.Remove(id)
		return &CommandResult{Message: fmt.Sprintf("✅ Session %s deleted.", id)}
	default:
		return &CommandResult{Message: "Usage: /sessions [ls|clear|del <id>]", Error: true}
	}
}

func (s *Server) cmdStatus(sess *GatewaySession) *CommandResult {
	if sess == nil {
		return &CommandResult{Message: "No active session.", Error: true}
	}
	mode := s.cfg.DefaultMode
	if sess.Mode != "" {
		mode = sess.Mode
	}
	s.mu.RLock()
	modelID := s.model.ID
	s.mu.RUnlock()
	msgCount := 0
	if sess.Manager != nil {
		msgCount = len(sess.Manager.GetMessages())
	}
	msg := fmt.Sprintf("Session: %s\nMode: %s\nModel: %s\nMessages: %d\nWorkDir: %s",
		sess.ID, strings.ToUpper(mode), modelID, msgCount, sess.WorkDir)
	return &CommandResult{Message: msg}
}

func (s *Server) cmdDelegate(sess *GatewaySession, parts []string) *CommandResult {
	if sess == nil {
		return &CommandResult{Message: "No active session.", Error: true}
	}
	if len(parts) < 2 || parts[1] == "status" {
		state := "OFF"
		if sess.DelegateMode {
			state = "ON"
		}
		return &CommandResult{Message: fmt.Sprintf("Delegation mode: %s", state)}
	}
	switch parts[1] {
	case "on":
		if sess.AgentMgr == nil {
			compactionSettings := ctxpkg.CompactionSettings{
				Enabled:          s.settings.Compaction.Enabled,
				ReserveTokens:    s.settings.Compaction.ReserveTokens,
				KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
				Tokenizer:        s.settings.Compaction.Tokenizer,
				TokenizerModel:   s.settings.Compaction.TokenizerModel,
				Template:         s.settings.Compaction.Template,
			}
			extraContext := sess.ExtraContext
			if extraContext == "" {
				extraContext = s.extraContext
			}
			skillsMgr := sess.SkillsMgr
			if skillsMgr == nil {
				skillsMgr = s.skillsMgr
			}
			factory := agent.NewAgentFactoryWithOptions(s.provider, s.model, s.settings, s.sandboxMgr, extraContext, skillsMgr, compactionSettings, nil, agent.AgentFactoryOptions{
				MultiAgentEnabled: true,
				DelegateEnabled:   true,
				Allow:             s.getAllow(),
			})
			sess.AgentMgr = agent.NewAgentManager(factory)
		}
		agent.RegisterDelegateSubAgentTool(sess.Registry, sess.AgentMgr)
		sess.DelegateMode = true
		return &CommandResult{Message: "Delegation mode: ON"}
	case "off":
		sess.Registry.Remove("delegate_subagent")
		sess.DelegateMode = false
		return &CommandResult{Message: "Delegation mode: OFF"}
	default:
		return &CommandResult{Message: "Usage: /delegate [on|off|status]", Error: true}
	}
}

func (s *Server) cmdAllowEditPath(parts []string) *CommandResult {
	allow := s.getAllow()
	if len(parts) < 2 {
		paths := allow.EditPathList()
		if len(paths) == 0 {
			return &CommandResult{Message: "Auto-edit path whitelist is empty. Usage: /alloweditpath add|remove <glob>|clear"}
		}
		var sb strings.Builder
		sb.WriteString("Auto-edit path whitelist (agent mode):\n")
		for _, p := range paths {
			sb.WriteString(fmt.Sprintf("  %s\n", p))
		}
		return &CommandResult{Message: strings.TrimRight(sb.String(), "\n")}
	}
	switch parts[1] {
	case "add":
		if len(parts) < 3 {
			return &CommandResult{Message: "Usage: /alloweditpath add <glob>", Error: true}
		}
		glob := strings.Join(parts[2:], " ")
		if !allow.AddEditPath(glob) {
			return &CommandResult{Message: fmt.Sprintf("Already in whitelist: %s", glob)}
		}
		if err := allow.SaveProject(); err != nil {
			return &CommandResult{Message: fmt.Sprintf("Failed to save allow.json: %v", err), Error: true}
		}
		return &CommandResult{Message: fmt.Sprintf("✅ Added to auto-edit whitelist: %s", glob)}
	case "remove", "rm":
		if len(parts) < 3 {
			return &CommandResult{Message: "Usage: /alloweditpath remove <glob>", Error: true}
		}
		glob := strings.Join(parts[2:], " ")
		if !allow.RemoveEditPath(glob) {
			return &CommandResult{Message: fmt.Sprintf("Not in whitelist: %s", glob)}
		}
		if err := allow.SaveProject(); err != nil {
			return &CommandResult{Message: fmt.Sprintf("Failed to save allow.json: %v", err), Error: true}
		}
		return &CommandResult{Message: fmt.Sprintf("✅ Removed from auto-edit whitelist: %s", glob)}
	case "clear":
		allow.ClearEditPaths()
		if err := allow.SaveProject(); err != nil {
			return &CommandResult{Message: fmt.Sprintf("Failed to save allow.json: %v", err), Error: true}
		}
		return &CommandResult{Message: "✅ Auto-edit path whitelist cleared"}
	default:
		return &CommandResult{Message: "Usage: /alloweditpath [add <glob>|remove <glob>|clear]", Error: true}
	}
}

func (s *Server) cmdAllowAutoEdit(parts []string) *CommandResult {
	allow := s.getAllow()
	if len(parts) < 2 {
		state := "OFF"
		if allow.GetAutoEdit() {
			state = "ON"
		}
		return &CommandResult{Message: fmt.Sprintf("Auto-edit (agent mode): %s\nUsage: /allowautoedit [on|off] [global]", state)}
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
		return &CommandResult{Message: "Usage: /allowautoedit [on|off] [global]", Error: true}
	}
	var err error
	scope := "project"
	effective := enable
	if globalScope {
		scope = "global"
		effective = allow.SetGlobalAutoEdit(enable)
		err = allow.SaveGlobalAutoEditValue(enable)
	} else {
		allow.SetProjectAutoEdit(enable)
		err = allow.SaveProject()
	}
	if err != nil {
		return &CommandResult{Message: fmt.Sprintf("Failed to save allow.json: %v", err), Error: true}
	}
	state := "OFF"
	if enable {
		state = "ON"
	}
	msg := fmt.Sprintf("✅ Auto-edit (agent mode): %s [%s]", state, scope)
	if globalScope && effective != enable {
		effectiveState := "OFF"
		if effective {
			effectiveState = "ON"
		}
		msg += fmt.Sprintf(" (effective here: %s due to project override)", effectiveState)
	}
	return &CommandResult{Message: msg}
}

func (s *Server) cmdCompact(sess *GatewaySession) *CommandResult {
	if sess == nil {
		return &CommandResult{Message: "No active session.", Error: true}
	}

	// Check if there are enough messages to compact
	if sess.Manager == nil || len(sess.Manager.GetMessages()) < 2 {
		return &CommandResult{Message: "Nothing to compact: conversation is too short.", Error: true}
	}
	previousSummary := ""
	if compaction, ok := sess.Manager.GetLatestCompaction(); ok {
		previousSummary = compaction.Summary
	}
	compactionSettings := ctxpkg.NormalizeCompactionSettings(ctxpkg.CompactionSettings{
		Enabled:          s.settings.Compaction.Enabled,
		ReserveTokens:    s.settings.Compaction.ReserveTokens,
		KeepRecentTokens: s.settings.Compaction.KeepRecentTokens,
		Tokenizer:        s.settings.Compaction.Tokenizer,
		TokenizerModel:   s.settings.Compaction.TokenizerModel,
		Template:         s.settings.Compaction.Template,
	})
	if !ctxpkg.HasCompactableMessages(sess.Manager.GetReplayState().Messages, s.model, compactionSettings, previousSummary) {
		return &CommandResult{Message: "Nothing to compact: only recent context is available to keep.", Error: true}
	}

	// Set the force flag so the next agent run triggers compaction
	sess.ForceCompact = true
	return &CommandResult{Message: "✅ Context compaction will be triggered on the next request."}
}

func (s *Server) cmdSkill(sess *GatewaySession, parts []string) *CommandResult {
	skillsMgr := s.sessionSkills(sess)
	if skillsMgr == nil {
		return &CommandResult{Message: "No skills available.", Error: true}
	}
	if len(parts) < 2 {
		return s.cmdSkills(sess)
	}
	name := parts[1]
	skill := skillsMgr.Get(name)
	if skill == nil {
		return &CommandResult{Message: fmt.Sprintf("Skill not found: %s", name), Error: true}
	}
	return &CommandResult{Message: fmt.Sprintf("✅ Skill '%s' activated: %s", name, skill.Description)}
}

func (s *Server) cmdSkills(sess *GatewaySession) *CommandResult {
	skillsMgr := s.sessionSkills(sess)
	if skillsMgr == nil {
		return &CommandResult{Message: "No skills available."}
	}
	skillList := skillsMgr.List()
	if len(skillList) == 0 {
		return &CommandResult{Message: "No skills found."}
	}
	var sb strings.Builder
	sb.WriteString("Available skills:\n")
	for _, sk := range skillList {
		sb.WriteString(fmt.Sprintf("  - %s (%s): %s\n", sk.Name, sk.Source, sk.Description))
	}
	return &CommandResult{Message: sb.String()}
}

func (s *Server) sessionSkills(sess *GatewaySession) *skills.Manager {
	if sess != nil && sess.SkillsMgr != nil {
		return sess.SkillsMgr
	}
	return s.skillsMgr
}

func (s *Server) cmdWorkflows(parts []string) *CommandResult {
	store := workflow.DefaultStore()
	sub := "list"
	if len(parts) > 1 {
		sub = strings.ToLower(parts[1])
	}
	switch sub {
	case "list", "ls":
		runs, err := store.List(context.Background())
		if err != nil {
			return &CommandResult{Message: fmt.Sprintf("Failed to list workflows: %v", err), Error: true}
		}
		if len(runs) == 0 {
			return &CommandResult{Message: "Workflow runs: (none)"}
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Workflow runs (%d):\n", len(runs)))
		for _, run := range runs {
			sb.WriteString(fmt.Sprintf("  [%s] %s %s\n", run.Status, run.ID, run.Name))
		}
		return &CommandResult{Message: strings.TrimRight(sb.String(), "\n")}
	case "show":
		if len(parts) < 3 {
			return &CommandResult{Message: "Usage: /workflows show <id>", Error: true}
		}
		run, err := store.Load(context.Background(), parts[2])
		if err != nil {
			return &CommandResult{Message: fmt.Sprintf("Failed to load workflow: %v", err), Error: true}
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
		return &CommandResult{Message: strings.TrimRight(sb.String(), "\n")}
	case "cancel":
		if len(parts) < 3 {
			return &CommandResult{Message: "Usage: /workflows cancel <id>", Error: true}
		}
		id := strings.TrimSpace(parts[2])
		if !workflow.DefaultActiveRegistry().Cancel(id) {
			return &CommandResult{Message: fmt.Sprintf("Workflow run %s is not active.", id), Error: true}
		}
		return &CommandResult{Message: fmt.Sprintf("Workflow run %s cancellation requested.", id)}
	default:
		return &CommandResult{Message: "Usage: /workflows [list|show <id>|cancel <id>]", Error: true}
	}
}

func (s *Server) cmdHelp() *CommandResult {
	help := `Available commands:
  /clear                  - Clear conversation context
  /mode [plan|agent|yolo] - Show or switch mode
  /model [model_id]       - Show or switch model
  /models                 - List available models
  /sessions               - List active sessions
  /sessions del <id>      - Delete a session
  /compact                - Trigger context compaction
  /delegate [on|off|status] - Toggle delegation mode
  /alloweditpath [add <glob>|remove <glob>|clear] - Auto-edit path whitelist
  /allowautoedit [on|off] [global] - Toggle full auto-edit in agent mode
  /workflows [list|show <id>|cancel <id>] - Inspect workflow runs
  /status                 - Show session status
  /skill <name>           - Activate a skill
  /skills                 - List available skills
  /help                   - Show this help`
	return &CommandResult{Message: help}
}
