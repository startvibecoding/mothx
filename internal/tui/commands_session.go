package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/platform"
	"github.com/startvibecoding/vibecoding/internal/session"
)

func (a *App) getSessionDir() string {
	if a.settings != nil {
		return a.settings.GetSessionDir()
	}
	return platform.SessionDir()
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
