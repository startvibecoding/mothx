# VibeCoding Agent Guide

This file is for AI agents working in this repository. Keep changes aligned with the current codebase and prefer concise, minimal edits.

## Project Snapshot

- Language: Go
- UI: Bubble Tea + Lipgloss
- CLI: Cobra
- Default working style: terminal-first, tool-driven
- Main purpose: a terminal AI coding assistant with provider abstraction, sessions, tools, sandboxing, context files, skills, and unified serve mode

## Important Directories

- `cmd/mothx/` — CLI entry
- `internal/agent/` — agent loop, events, system prompt
- `internal/config/` — settings and defaults
- `internal/context/` — context window and compaction
- `internal/contextfiles/` — `AGENTS.md` / `CLAUDE.md` discovery
- `internal/esm/` — Enable Supervisor Mode state, prompts, reports, and tools
- `internal/memory/` — persistent memory (memory.md)
- `internal/messaging/` — messaging platform abstraction (wechat, feishu)
- `internal/provider/` — provider abstraction and implementations
- `internal/provider/factory/` — shared provider/model construction from config
- `internal/provider/vendor*.go` — vendor adapter registry and per-vendor defaults
- `internal/provider/anthropic/` — full Anthropic provider implementation
- `internal/provider/google/` — full Google Gemini provider implementation
- `internal/provider/openai/` — full OpenAI provider implementation
- `internal/sandbox/` — sandbox backends
- `internal/session/` — SQLite session storage, schema migrations
- `internal/skills/` — skills loading
- `internal/stats/` — usage stats web dashboard
- `internal/tools/` — built-in tools
- `internal/tui/` — terminal UI
- `internal/acp/` — ACP / MCP related integration
- `internal/a2a/` — A2A (Agent-to-Agent) protocol server and master mode
- `internal/mcp/` — MCP (Model Context Protocol) server integration
- `internal/cron/` — scheduled task management and cron tool
- `internal/browser/` — browser automation tool (vibe-browser)
- `internal/workflow/` — workflow engine (phases, tasks, skill integration)
- `internal/platform/` — cross-platform compatibility (OS detection, busybox)
- `internal/imageproc/` — image preprocessing for tool results
- `internal/systeminit/` — shared prompt for /systeminit command
- `internal/update/` — non-blocking version update detection
- `internal/ua/` — user-agent string generation
- `internal/util/` — utility functions
- `internal/debugpprof/` — debug profiling HTTP server (pprof)
- `internal/serve/` — unified server mode: OpenAI-compatible API, Web UI, channels, cron, memory, settings APIs
- `internal/serve/openaiapi/` — OpenAI-compatible HTTP API runtime used by serve
- `internal/serve/channels/` — WeChat/Feishu/WebSocket channel dispatcher used by serve
- `internal/serve/ws/` — serve WebSocket channel runtime
- `internal/serve/webhook/` — inbound webhook routing for serve channels
- `internal/serve/hooks/` — hooks execution
- `ui/` — Svelte 5 + Vite frontend for serve Web UI; embedded into the Go binary via `ui/embed.go`
- `docs/` — documentation

## Architecture Notes

- Providers stream responses through the provider abstraction.
- Provider creation should go through `internal/provider/factory` so CLI and ACP keep the same behavior.
- Vendor-specific behavior belongs in `internal/provider/vendor*.go` adapters and model `compat` flags, not in CLI/ACP wiring.
- Each vendor that needs detection or defaults should have a separate `internal/provider/vendor_<name>.go` file.
- Full provider implementations live in `internal/provider/anthropic/`, `internal/provider/google/`, and `internal/provider/openai/` subdirectories, containing substantial provider-specific logic (streaming, thinking, tool use).
- Vendors without special behavior should fall back to the generic OpenAI-compatible or Anthropic-compatible provider based on `api` / base URL detection.
- Do not change the settings JSON schema or the expected meaning of existing provider config fields when adding vendor support.
- The agent loop builds a system prompt, sends messages, handles stream events, executes tools, and continues until completion.
- Tools should stay stateless when possible; shared execution state belongs in registries/managers.
- AgentManager creates future agents from its AgentFactory. When runtime provider/model/settings change in the TUI, update the manager runtime too so sub-agents and ESM workers use the same provider/model as the main app.
- Context files and skills are first-class prompt inputs.
- Enable Supervisor Mode state and report parsing live in `internal/esm`; TUI orchestration owns worker/critic/audit sub-agent scheduling and status transitions.
- Sessions are stored in SQLite with parent/child relationships. CLI and serve API sessions use a single root `sessions.db` database with dynamically computed virtual `.db` paths for listing/switching; serve channels additionally write physical handle files in per-user channel directories on disk.
- Schema migrations are managed via `internal/session/migrations.go`. A `schema_migrations` table tracks which migrations have been applied. `ApplyMigrations(db)` runs any pending migrations and is called on every DB open from both `session.withDB()` and `stats.Open()`. To add a schema change, append a new entry to the `migrations` slice — do not use `CREATE TABLE IF NOT EXISTS` directly in new code.

### Settings Configuration

- `settings.json` schema lives in `internal/config/settings.go`. Do not change existing field meanings when adding UI or provider behavior.
- TUI `/settings` is the central editor for top-level `settings.json` groups. Provider/model configuration is one branch under that menu, not the whole command.
- When writing a global top-level setting from the TUI, prefer `config.SaveGlobalSettingsPatch()` so only the affected JSON key is updated. Do not save a sparse `Settings` object with `SaveGlobalSettings()` for top-level edits, because non-`omitempty` struct fields can expand defaults and accidentally override unset config.
- `/settings` provider edits should not change `defaultProvider` / `defaultModel` by default. Use the Defaults picker or an explicit "Set as Default" path for default model changes.
- Approval bash whitelist/blacklist entries are command prefixes; trailing spaces can be meaningful (for example `go `). Preserve them and avoid comma-based trimming when editing those lists.

### API Mode

- `internal/serve/openaiapi/` implements the OpenAI-compatible Chat Completions API used by `mothx serve`.
- The API runtime reuses the same agent loop, provider factory, session, tools, sandbox, and skills as CLI/ACP — no separate agent logic.
- Configuration lives in `serve.json` (global `~/.mothx/serve.json`, project `.mothx/serve.json`), separate from `settings.json`.
- Project-level `.mothx/serve.json` overrides global, same pattern as `.mothx/settings.json`.
- The API runtime supports slash commands processed at the HTTP layer without invoking the LLM: `/clear`, `/mode`, `/model`, `/defaultModel`, `/models`, `/sessions`, `/status`, `/compact`, `/delegate`, `/alloweditpath`, `/allowautoedit`, `/workflows`, `/skill`, `/skills`, `/rule`, `/help`.
- Tool output visibility (`toolVisibility.mode` + `toolVisibility.detail`) is configurable: collapsed (default, one-line summary) or expanded (full code fences).
- `edit`/`write` diffs and errors always show in full regardless of detail level.
- When `x_session_id` is empty, the API runtime reuses a default session so consecutive requests share context.
- Security: three independent layers — Bearer token auth, `allowedWorkDirs` whitelist, sandbox (bwrap).
- No external HTTP framework; uses `net/http` standard library.

### Channels Mode

- `internal/serve/channels/` implements WeChat/Feishu/WebSocket channels with persistent agent sessions.
- Channels reuse the same agent loop, provider factory, session, tools, sandbox, skills, and MCP as CLI/ACP.
- Configuration lives in `serve.json`; top-level fields are `api`, `features`, `channels`, `webUI`, `lobsterMode`, `cron`, `memory`, `security`, `hooks`, and `agent`.
- Per-user channel sessions are stored under `<sessionDir>/channels/<platform>/<user_id>/`.
- Default mode is `yolo` (not `agent`) — messaging platforms are unattended by nature.
- `provider` / `model` in `serve.json` override settings.json; CLI `-p`/`-m` override `serve.json`.
- `features.multiAgent` enables sub-agent tools (spawn/status/send/destroy).
- `sandbox.enabled` enables bwrap sandbox (default off).
- MCP servers from global/project `mcp.json` are loaded per-session and auto-closed on removal.
- memory.md defaults to project directory (`.mothx/memory.md`); only uses global when `memory.path` is explicitly set.
- Progress events (tool execution + thinking) are sent to messaging platforms via `InboundMessage.ProgressFunc`.
- The `messaging.InboundMessage.ProgressFunc` callback is set by each platform bot; nil means no progress updates.
- `formatToolProgress` in `dispatcher.go` formats tool events as `[tool]: args ✅/❌`.
- Think deltas are accumulated and flushed as `💭 ...` (truncated to 500 chars) before tool/text events.

### Stats Dashboard Mode

- `internal/stats/` implements a web server that displays usage statistics (tokens, requests, duration) with charts.
- The `vibecoding stats` CLI subcommand starts the dashboard server (default `127.0.0.1:7878`).
- Flags: `--addr` (listen address), `--db` (path to sessions.db, defaults to `~/.mothx/sessions/sessions.db`).
- Stats are recorded automatically by the agent loop after every LLM call via `session.RecordUsageFromProviderUsage()`.
- The dashboard is pure HTML/CSS/JS — no external JS/CSS libraries. Charts are drawn on `<canvas>`.
- API endpoints: `/api/summary`, `/api/timeseries`, `/api/by-provider`, `/api/by-model`, `/api/recent`.
- All stats queries go through the shared `sessions.db`. The stats server calls `session.ApplyMigrations()` on open to ensure the `request_stats` table exists.
- The dashboard supports filtering by time range (today/week/month/all), provider, and model.

## Working Rules

- Read before editing.
- Prefer small, targeted changes that deliver the best maintainable implementation; do not choose a merely minimal or quick workaround when a more robust design is appropriate.
- Keep behavior consistent with existing patterns.
- Do not introduce broad refactors unless requested.
- Do not add license headers unless the repository already uses them.
- Do not auto-commit. Commit only when the user explicitly asks.

## Go Conventions

- Return errors; do not panic for normal control flow.
- Pass `context.Context` through request/execution paths.
- Keep interfaces and structs consistent with nearby code.
- Follow existing naming and file layout before introducing new abstractions.
- Add tests when changing behavior or fixing bugs if there is an obvious test location.

## Tooling Notes

Built-in tools include:
- `read`, `write`, `edit`
- `bash`, `jobs`, `kill`
- `grep`, `find`, `ls`
- `plan`, `question` (TUI plan/agent modes only)
- `skill_ref`
- `a2a_dispatch` (dynamically registered when A2A is enabled)
- `cron` (dynamically registered for scheduled task management)
- `browser` (loaded as a skill via `vibe-browser`)

`grep` and `find` are backed by pure-Go SDKs (`github.com/startvibecoding/go-ripgrep` and `github.com/startvibecoding/go-fd`). They work on all Go-supported architectures without external binary dependencies.

`bash` defaults to a short synchronous timeout (45s); use `async=true` for dev servers and other long-running processes, and treat `timeout=0` as an explicit no tool-level deadline.

## Modes and Safety

- `plan`: read-only tools + `question` (interactive, TUI only)
- `agent`: file edits allowed; `bash` usually requires approval; `question` available (interactive, TUI only)
- `yolo`: all tools auto-execute (no `question`)

The `question` tool is registered for interactive TUI sessions (not print mode) and for the ACP server, and exposed in `plan` and `agent` modes via `Registry.ModeTools` (excluded in `yolo`). It uses the `QuestionHandler` optional interface (type assertion) to avoid polluting the public `Agent` interface. TUI shows it inline; ACP surfaces questions through the `session/request_permission` channel. Serve API/channel runtimes never register or expose it.

The `/systeminit` command (TUI, ACP, and the `vibecoding systeminit` CLI subcommand) generates or refreshes a project `AGENTS.md`. In interactive surfaces (TUI/ACP) the agent is told to use the `question` tool to clarify conventions first; on the CLI it runs non-interactively in yolo+print. The shared instruction prompt lives in `internal/systeminit`.

The TUI `/reload` command re-execs the process with session-continuation flags stripped, giving a fresh process with a new session.

When changing code, prefer the least risky approach that satisfies the request, but optimize for the most correct, maintainable implementation rather than the smallest possible diff.

## Serve-Specific Notes

- Serve-only config belongs in `internal/serve/config.go`, not in `internal/config/settings.go`.
- Tool output formatting (collapsed/expanded, markdown code fences) belongs in `internal/serve/openaiapi/tool_format.go`.
- Slash command handlers belong in `internal/serve/openaiapi/commands.go`, kept separate from TUI commands (different dependencies).
- The `resolveToolEvent()` helper in `handler_chat.go` handles the fact that `EventToolCall` carries tool name in `ev.ToolCall.Name` (not `ev.ToolName`).
- When adding new slash commands, add to both API `commands.go` and TUI `commands.go` to keep feature parity.

## WebUI-Specific Notes

- The serve Web UI is a Svelte 5 + Vite SPA. Source lives in `ui/src/`, built output in `ui/dist/`, embedded into Go via `ui/embed.go` (`webui.DistFS()`).
- Build: `cd ui && npm run build`. The Go binary serves the embedded `dist/` at runtime; no separate frontend server needed in production.
- Dev server: `cd ui && npm run dev` (Vite, `127.0.0.1:5173`). Point the serve API at it via `webUI.dir` or the Vite proxy.
- Entry: `ui/src/App.svelte` renders a two-pane layout (`Sidebar` + `workbench`) inside `.app-shell` (CSS grid).
- Shared reactive stores live in `ui/src/lib/stores.js`. Views subscribe to stores; `refreshAll()` reloads everything after server-state changes.
- Routing is hash-based (`ui/src/lib/router.js`): `#/chat`, `#/sessions`, `#/stats`, `#/cron`, `#/skills`, `#/settings`.
- Translations (i18n) live in `ui/src/lib/preferences.js` as `zh` and `en` flat key maps. Use the `$t('key')` store. Add new keys to both language maps.
- Styling is a single global `ui/src/style.css` (no CSS modules, no Tailwind). Follow existing class naming and CSS variable conventions (`--bg`, `--border`, `--text-muted`, etc.).
- Mobile responsiveness uses Svelte-native patterns, not CSS media queries for interactive behavior:
  - `isMobile` is a `readable` store backed by `window.matchMedia('(max-width: 900px)')` in `stores.js`. Use `{#if $isMobile}` to conditionally render mobile-specific UI (e.g., hamburger button, drawer).
  - The sidebar drawer on mobile uses `{#if $sidebarOpen}` + Svelte `transition:fly` / `transition:fade`. The overlay and drawer are only in the DOM when open, so they cannot block clicks when closed.
  - `sidebarOpen` store defaults to `false` (collapsed). Navigation clicks and Escape key close the drawer.
  - CSS media queries in `style.css` are reserved for non-interactive layout adjustments (grid columns, padding, form layouts). Do not use media queries to toggle `display` or `pointer-events` for interactive elements — use Svelte `{#if}` instead.
- Svelte 5 snippets (`{#snippet name()}...{/snippet}` + `{@render name()}`) are used to share markup between mobile and desktop branches without duplication.
- When adding new views, register them in `App.svelte` and add a nav entry in `Sidebar.svelte` (`primaryNav` or `secondaryNav`).
- When adding translation keys, update both `zh` and `en` maps in `preferences.js`.

## TUI-Specific Notes

- Completed transcript blocks must be printed to native terminal scrollback with Bubble Tea `Program.Println`.
- Do not route completed transcript scrollback output through `tea.Println(...)` sent via `Program.Send`; that puts printing back into the update loop and can swallow or delay transcript output.
- Keep only active streaming content in the managed Bubble Tea view. Completed user/assistant/tool/status blocks should leave the live view after they are printed so mouse scrolling and terminal selection use the terminal's own scrollback.
- Show a visible tool "running" line before the final result line, rather than overwriting a single tool entry in place.
- In auth/settings dialogs, clear stale `ParamField` / `ParamFieldKey` when changing views. Menu navigation and toggle fields must not leave input mode active for the next view.
- In `/auth` model lists, `Backspace` / `Delete` on a model row deletes that model. Keep deletion scoped to model rows only; action rows like `+ Add Model` and `Done` must not delete anything.
- `/defaultModel`, `/model`, and `/auth` provider saves must keep `App.provider`, `App.providerName`, `App.model`, `App.settings`, and `AgentManager` runtime config in sync; otherwise ESM and sub-agents can continue using stale provider/model instances.

## ESM-Specific Notes

- ESM completion is a worker candidate followed by independent critic and audit review. Only a passing audit can mark the objective complete.
- Critic/audit `fail`, malformed reports, missing pass evidence, missing `requirements_checked`, or all-failed inspection tools must reject the completion candidate and return the objective to active.
- Worker `complete_candidate` requires successful tool-backed inspection evidence. If all worker tools failed, keep ESM active and record the completion review.
- If the token budget is reached during worker/critic/audit execution, do not mark the objective complete.
- Preserve `completion_review` in the next worker prompt so later runs can address previous audit failures.
- `blocked_candidate` must follow the repeated-blocker audit rules in `internal/esm.Store`; do not treat a single blocked report as terminal.

## Provider and Model Reference

- Full provider/model list with context windows, max tokens, thinking format, API type, and pricing is documented in `docs/provider-model-list.md`.
- When adding or updating a provider or model, update both `internal/config/settings.go` (`defaultProviderConfigs`) and `docs/provider-model-list.md`.
- Vendor adapters live in `internal/provider/vendor_*.go`; run `go test ./internal/provider/...` after adding or changing vendor detection.

## Docs and Release Notes

- Put changelog updates only in:
  - `docs/en/changelog.md`
  - `docs/zh/changelog.md`
- Do not create separate release note files.
- Update README files only for user-visible major changes.

### Online Install Scripts

- The public site is deployed by Cloudflare from the `docs/` directory. Public installer URLs map directly to `docs/install.sh` and `docs/install.bat`:
  - `https://mothx.net/install.sh`
  - `https://mothx.net/install.bat`
- Keep the root `install.sh` / `install.bat` copies synchronized with the corresponding files under `docs/` when changing installer behavior.
- Online installers must prefer an existing Node.js installation. If Node.js is missing, install the latest supported Node.js LTS release for the current OS/architecture, then run `npm install -g mothx-installer`.
- User-facing documentation should use the `mothx.net` installer URLs rather than GitHub raw or Gitee raw URLs.

## Validation

When appropriate, verify with the smallest useful scope first.
Examples:
- focused package tests
- targeted grep/find checks
- full test suite only when necessary

## Build / Test

Common commands:
- `make build`
- `make test`
