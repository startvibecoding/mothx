package tui

import "strings"

type commandOutputKind int

const (
	commandOutputStatus commandOutputKind = iota
	commandOutputError
)

type commandOutput struct {
	kind  commandOutputKind
	lines []string
}

func (a *App) addCommandStatus(lines ...string) {
	a.addCommandOutput(commandOutput{kind: commandOutputStatus, lines: lines})
}

func (a *App) addCommandError(lines ...string) {
	a.addCommandOutput(commandOutput{kind: commandOutputError, lines: lines})
}

func (a *App) addCommandOutput(out commandOutput) {
	text := strings.Join(out.lines, "\n")
	switch out.kind {
	case commandOutputError:
		a.addMessage(errorStyle.Render(text))
	default:
		a.addMessage(statusStyle.Render(text))
	}
}

func commandHelpText() string {
	return strings.Join([]string{
		"Commands:",
		"  /mode [plan|agent|yolo] - Switch or show mode",
		"  /model [model_id]       - Switch or show model",
		"  /skills                 - List available skills",
		"  /skill <name>           - Activate a skill",
		"  /clear                  - Clear conversation",
		"  /compact                - Trigger context compaction",
		"  /sessions               - List sessions for this project",
		"  /sessions ls            - List sessions",
		"  /sessions set <id>      - Switch to session",
		"  /sessions clear         - Create a new session",
		"  /sessions del <id>      - Delete a session",
		"  /init_mcp [target] [template] [--force]",
		"                         - Init mcp.json (target: project|global, template: basic|full)",
		"  /mcps                   - List MCP servers (global/project mcp.json)",
		"  /delegate [on|off|status] - Toggle delegation mode",
		"  /agent list              - List all agents (multi-agent mode)",
		"  /agent switch <id>       - Switch active agent",
		"  /agent destroy <id>      - Destroy a sub-agent",
		"  /cron add <description>  - Add scheduled task (multi-agent mode)",
		"  /cron list               - List scheduled tasks",
		"  /cron enable <id>        - Enable a task",
		"  /cron disable <id>       - Disable a task",
		"  /cron remove <id>        - Remove a task",
		"  /cron run <id>           - Run a task now",
		"  /quit                   - Exit",
		"  /help                   - Show this help",
		"",
		"Keyboard shortcuts:",
		"  Enter             - Submit input",
		"  Alt+Enter/Ctrl+J  - Insert newline in input",
		"  Tab               - Cycle mode (plan/agent/yolo)",
		"  Esc               - Abort current operation",
		"  Ctrl+O            - Open latest tool details",
		"  Ctrl+G            - Toggle compact tool display",
		"  Up/Down           - Move in multiline input; history at boundaries",
		"  PgUp/PgDn         - Page tool details when modal is open",
	}, "\n")
}
