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
		"  /defaultModel [project|global] - Set default provider/model (default: global)",
		"  /auth                  - Configure provider token, base URL and models",
		"  /settings              - Configure settings.json groups, including providers",
		"  /skills                 - List available skills",
		"  /skill <name>           - Activate a skill",
		"  /paste-image            - Save clipboard image and insert its local file path",
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
		"  /browser [on|off|status] - Toggle browser automation tool",
		"  /stats server|stop-server|tui - Start/stop stats server or show stats in TUI",
		"  /statusline [status|on|off] [project|global] - Inspect or toggle TUI status line",
		"  /statusline command <cmd> [project|global]   - Set the status line command",
		"  /statusline refresh <0-60> [project|global] - Set periodic refresh seconds",
		"  /alloweditpath [add <glob>|remove <glob>|clear] - Auto-edit path whitelist (agent mode)",
		"  /allowautoedit [on|off] [global] - Full auto-edit in agent mode (only bash needs approval)",
		"  /btw <question>          - Ask a side question; inherits context, answer not saved",
		"  /systeminit [guidance]   - Generate/refresh AGENTS.md (asks first; e.g. /systeminit ask me in Chinese, write in English)",
		"  /rule [force]            - Create .vibe/rule.md with safe default project rules",
		"  /reload                  - Restart as a fresh process with a new session",
		"  /workflows [list|show <id>|cancel <id>] - Inspect workflow runs",
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
		"  Ctrl+R            - Preview latest pasted image",
		"  Ctrl+G            - Toggle compact tool display",
		"  Up/Down           - Move in multiline input; history at boundaries",
		"  Left/Right       - Switch detail target when Ctrl+O modal is open",
		"  PgUp/PgDn         - Page details when Ctrl+O modal is open",
	}, "\n")
}
