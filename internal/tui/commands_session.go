package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/startvibecoding/mothx/internal/config"
	"github.com/startvibecoding/mothx/internal/platform"
	"github.com/startvibecoding/mothx/internal/session"
)

type sessionsDialogState struct {
	Open    bool
	Cursor  int
	Items   []session.SessionDetail
	Cwd     string
	Error   string
	Message string
}

var sessionsDialogStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(1, 2)

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
	if a.session.GetHeader() != nil {
		return a.session.GetHeader().ID
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
	if len(parts) == 1 {
		a.openSessionsDialog()
		return
	}

	sub := strings.ToLower(parts[1])
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
		a.addCommandError(fmt.Sprintf("Unknown subcommand: %s. Use /sessions, or ls, set, clear, del.", sub))
	}
}

func (a *App) sessionsCwd() string {
	if a.session != nil && a.session.GetHeader() != nil && a.session.GetHeader().Cwd != "" {
		return a.session.GetHeader().Cwd
	}
	if a.cwd != "" {
		return a.cwd
	}
	if w, err := os.Getwd(); err == nil {
		return w
	}
	return "."
}

func (a *App) openSessionsDialog() {
	if a.isThinking {
		a.addCommandError("Cannot switch sessions while the agent is running.")
		return
	}
	cwd := a.sessionsCwd()
	details, err := session.ListForDirDetailed(cwd, a.getSessionDir())
	state := sessionsDialogState{
		Open:  true,
		Items: details,
		Cwd:   cwd,
	}
	if err != nil {
		state.Error = fmt.Sprintf("Error listing sessions: %v", err)
	}
	currentID := a.getCurrentSessionID()
	for i, d := range details {
		if d.ID == currentID {
			state.Cursor = i
			break
		}
	}
	a.sessionsDialog = state
	a.input = a.input.Blur()
	a.scheduleRender()
}

func (a *App) closeSessionsDialog() {
	a.sessionsDialog = sessionsDialogState{}
	a.input = a.input.Focus()
	a.scheduleRender()
}

func (a *App) moveSessionsCursor(delta int) {
	if len(a.sessionsDialog.Items) == 0 {
		return
	}
	a.sessionsDialog.Cursor += delta
	if a.sessionsDialog.Cursor < 0 {
		a.sessionsDialog.Cursor = len(a.sessionsDialog.Items) - 1
	}
	if a.sessionsDialog.Cursor >= len(a.sessionsDialog.Items) {
		a.sessionsDialog.Cursor = 0
	}
	a.scheduleRender()
}

func (a *App) confirmSessionsDialog() {
	if len(a.sessionsDialog.Items) == 0 {
		return
	}
	item := a.sessionsDialog.Items[a.sessionsDialog.Cursor]
	if item.ID == a.getCurrentSessionID() {
		a.closeSessionsDialog()
		a.addCommandStatus("Already on this session.")
		return
	}
	if err := a.switchToSession(item); err != nil {
		a.sessionsDialog.Error = err.Error()
		a.scheduleRender()
		return
	}
	a.closeSessionsDialog()
	a.addCommandStatus(fmt.Sprintf("✅ Switched to session %s (%d msgs)", item.ID, item.MessageCount))
}

func (a *App) deleteSelectedSessionDialog() {
	if len(a.sessionsDialog.Items) == 0 {
		return
	}
	item := a.sessionsDialog.Items[a.sessionsDialog.Cursor]
	if item.ID == a.getCurrentSessionID() {
		a.sessionsDialog.Error = "Cannot delete the current session. Switch to another session first."
		a.scheduleRender()
		return
	}
	if err := session.DeleteSession(item.Path, a.getSessionDir()); err != nil {
		a.sessionsDialog.Error = fmt.Sprintf("Error deleting session: %v", err)
		a.scheduleRender()
		return
	}
	a.sessionsDialog.Items = append(a.sessionsDialog.Items[:a.sessionsDialog.Cursor], a.sessionsDialog.Items[a.sessionsDialog.Cursor+1:]...)
	if a.sessionsDialog.Cursor >= len(a.sessionsDialog.Items) {
		a.sessionsDialog.Cursor = max(0, len(a.sessionsDialog.Items)-1)
	}
	a.sessionsDialog.Message = fmt.Sprintf("Deleted session %s.", item.ID)
	a.sessionsDialog.Error = ""
	a.scheduleRender()
}

func (a *App) handleSessionsKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !a.sessionsDialog.Open {
		return false, nil
	}
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		a.closeSessionsDialog()
		return true, nil
	case tea.KeyUp:
		a.moveSessionsCursor(-1)
		return true, nil
	case tea.KeyDown:
		a.moveSessionsCursor(1)
		return true, nil
	case tea.KeyEnter:
		a.confirmSessionsDialog()
		return true, nil
	case tea.KeyRunes:
		switch strings.ToLower(string(msg.Runes)) {
		case "q":
			a.closeSessionsDialog()
			return true, nil
		case "d":
			a.deleteSelectedSessionDialog()
			return true, nil
		case "n":
			a.closeSessionsDialog()
			a.sessionsClear()
			return true, nil
		}
	}
	return true, nil
}

// sessionsList lists all sessions for the current project directory.
func (a *App) sessionsList() {
	cwd := a.sessionsCwd()
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
	cwd := a.sessionsCwd()

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

	if err := a.switchToSession(*match); err != nil {
		a.addCommandError(err.Error())
		return
	}

	a.addCommandStatus(fmt.Sprintf("✅ Switched to session %s (%d msgs)",
		match.ID, match.MessageCount))
}

func (a *App) switchToSession(detail session.SessionDetail) error {
	newSess, err := session.OpenByID(a.sessionsCwd(), a.getSessionDir(), detail.ID)
	if err != nil {
		return fmt.Errorf("Error opening session: %v", err)
	}

	a.session = newSess
	a.cwd = newSess.GetHeader().Cwd
	a.historyLoaded = false
	a.agentHistoryLoaded = false

	a.resetAgent(fmt.Errorf("session changed"))
	a.resetTranscriptState()
	a.contextUsage = nil
	a.totalInputTokens = 0
	a.totalCacheRead = 0
	a.totalCacheWrite = 0

	a.LoadHistoryMessages()
	a.contextUsage = a.computeContextUsage()
	a.updateViewportContent()
	for idx := range a.messages {
		a.printMessageOnce(idx)
	}
	return nil
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
	a.session = nil
	a.historyLoaded = false
	a.agentHistoryLoaded = false

	// Reset agent and UI state
	a.resetAgent(fmt.Errorf("session changed"))
	a.resetTranscriptState()
	a.contextUsage = nil
	a.totalInputTokens = 0
	a.totalCacheRead = 0
	a.totalCacheWrite = 0
	a.updateViewportContent()

	a.addCommandStatus("✅ New session will be created when you send the next message.")
}

// sessionsDel deletes a session by ID prefix.
func (a *App) sessionsDel(id string) {
	cwd := a.sessionsCwd()

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

func (a *App) renderSessionsDialog() string {
	if !a.sessionsDialog.Open {
		return ""
	}
	width := a.width - 4
	if width < 60 {
		width = 60
	}
	if width > 110 {
		width = 110
	}

	var lines []string
	lines = append(lines, "Sessions")
	path := a.sessionsDialog.Cwd
	if xansi.StringWidth(path) > width-4 {
		path = xansi.Truncate(path, width-4, "…")
	}
	lines = append(lines, statusStyle.Render(path), "")

	if len(a.sessionsDialog.Items) == 0 {
		lines = append(lines, "No sessions found for this project.")
	} else {
		limit := 10
		if a.height > 0 && a.height-10 < limit {
			limit = a.height - 10
			if limit < 4 {
				limit = 4
			}
		}
		start, end := visibleRange(a.sessionsDialog.Cursor, len(a.sessionsDialog.Items), limit)
		currentID := a.getCurrentSessionID()
		for i := start; i < end; i++ {
			d := a.sessionsDialog.Items[i]
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == a.sessionsDialog.Cursor {
				cursor = "› "
				style = style.Foreground(lipgloss.Color("86")).Bold(true)
			}
			marker := "  "
			if d.ID == currentID {
				marker = "* "
			}
			preview := d.Preview
			if preview != "" {
				preview = " - " + strings.ReplaceAll(preview, "\n", " ")
			}
			line := fmt.Sprintf("%s%s%s  %d msgs  %s%s", cursor, marker, d.ID, d.MessageCount, formatAge(d.ModTime), preview)
			if xansi.StringWidth(line) > width-4 {
				line = xansi.Truncate(line, width-4, "…")
			}
			lines = append(lines, style.Render(line))
		}
		if len(a.sessionsDialog.Items) > limit {
			lines = append(lines, "", statusStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(a.sessionsDialog.Items))))
		}
	}

	if a.sessionsDialog.Message != "" {
		lines = append(lines, "", statusStyle.Render(a.sessionsDialog.Message))
	}
	if a.sessionsDialog.Error != "" {
		lines = append(lines, "", errorStyle.Render(a.sessionsDialog.Error))
	}
	lines = append(lines, "", "Enter switch  Up/Down select  n new  d delete  Esc close")

	return sessionsDialogStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func visibleRange(cursor, total, limit int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if limit <= 0 || limit >= total {
		return 0, total
	}
	start := cursor - limit/2
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > total {
		end = total
		start = end - limit
	}
	return start, end
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
