# MothX

MothX was formerly known as VibeCoding. This PyPI package is kept as a legacy compatibility distribution: each wheel includes a platform-native `vibecoding` binary, while new installs and future updates should prefer `npm install -g mothx`.

## Installation

```bash
pip install vibecoding-installer
```

Or with an isolated environment:

```bash
pipx install vibecoding-installer
```

After installation, the legacy `vibecoding` command is available on your `PATH`.

## Quick Start

```bash
# Set your API key
export DEEPSEEK_API_KEY=sk-...

# Start an interactive session
vibecoding

# Or ask a question directly
vibecoding -P "Write a Go HTTP server"
```

## Choose Your Mode

Every session runs in one of three modes, switchable anytime with `Tab` or `/mode`:

| Mode | Files | Network | Best For |
|------|-------|---------|----------|
| **Plan** | Read-only | ✗ | Analysis, code review, safe exploration |
| **Agent** | Read/Write | ✗ | Daily development (default) |
| **YOLO** | Full | ✓ | System administration, automation |

```bash
vibecoding --mode plan     # read-only
vibecoding --mode agent    # standard (default)
vibecoding --mode yolo     # full access
```

## Multi-Provider Support

Switch providers with a single flag:

```bash
# DeepSeek (default)
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# OpenAI
vibecoding --provider openai --model gpt-4o

# Anthropic
vibecoding --provider anthropic --model claude-3-5-sonnet-20241022

# 20+ vendor adapters supported: Gemini, Kimi, Volcengine, Mistral,
# OpenRouter, Groq, Bedrock, GitHub Copilot, and more
```

## Session Management

MothX persists conversation history in SQLite with branching support:

```bash
vibecoding                    # start new session
vibecoding --continue         # resume last session
vibecoding --resume <id>      # resume a specific session
vibecoding --session <file>   # open a specific session file
```

Sessions live in `~/.vibecoding/sessions/` (Linux/macOS) or `%APPDATA%\vibecoding\sessions\` (Windows).

## Print / CI Mode

Use MothX in non-interactive mode for scripting and CI/CD:

```bash
vibecoding -P "Write unit tests for auth.go"
vibecoding -P "Generate a CHANGELOG from git log" > CHANGELOG.md
echo "Explain this function" | vibecoding -P
```

Use `--mode plan` for read-only tasks and pair with `--print` to fail fast when approval is required.

## Think Mode

Enable deeper reasoning for complex problems:

```bash
vibecoding --thinking high "Solve this architecture problem"
```

Levels: `off`, `minimal`, `low`, `medium`, `high`, `xhigh` — toggle in TUI with `/think`.

## Security & Sandboxing

MothX uses [bubblewrap](https://github.com/containers/bubblewrap) for process-level isolation. Three layers of protection:

- **Blacklist** — dangerous commands (`rm -rf`, `sudo`) are always blocked, even in YOLO mode
- **Approval gates** — Agent mode prompts before executing bash commands
- **Whitelist** — pre-approve safe commands in `settings.json`:

```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm "],
    "bashBlacklist": ["rm -rf", "sudo"],
    "confirmBeforeWrite": true
  }
}
```

## Skills System

Skills are reusable prompt snippets for project conventions:

```bash
# List skills
> /skills

# Activate a skill
> /skill my-conventions
```

Place `SKILL.md` files in:
- **Global** — `~/.vibecoding/skills/<name>/SKILL.md`
- **Project** — `.skills/<name>/SKILL.md` (overrides global)

Compatible with [SkillHub](https://github.com/startvibecoding/skillhub) for discovering and sharing community skills.

## IDE Integration

MothX speaks [Agent Client Protocol (ACP)](https://github.com/startvibecoding/mothx/blob/main/docs/en/acp.md), integrating with VS Code, Zed, and JetBrains IDEs.

### VS Code

Add to `.vscode/settings.json`:

```json
{
  "acp.agents": {
    "vibecoding": {
      "command": "vibecoding",
      "args": ["acp", "--mode", "agent"]
    }
  }
}
```

### JetBrains (IntelliJ, WebStorm, etc.)

Go to `Settings → Tools → ACP Agents` and add:
- Name: `MothX`
- Command: `vibecoding`
- Arguments: `acp --mode agent`

## Gateway Mode

Run MothX as an OpenAI-compatible HTTP server:

```bash
vibecoding gateway
```

Config in `~/.vibecoding/gateway.json` or `.vibe/gateway.json`. Ideal for team sharing, CI/CD pipelines, and embedding into your own tools.

## Multi-Agent Mode

Enable sub-agent delegation and cron workflows:

```bash
vibecoding --multi-agent
```

Sub-agent tools: `subagent_spawn`, `subagent_status`, `subagent_send`, `subagent_destroy`.

For a single blocking investigation:

```bash
vibecoding --delegate
```

## Hermes / Messaging Mode

Deploy MothX as a chatbot on WeChat, Feishu, or WebSocket:

```bash
vibecoding hermes
```

Config in `~/.vibecoding/hermes.json`. Each user gets an independent, persistent session.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Enter` | Submit prompt |
| `Alt+Enter` / `Ctrl+J` | Insert newline |
| `Tab` | Cycle mode (plan → agent → yolo) |
| `Esc` | Abort current operation |
| `Ctrl+O` | Open tool details |
| `Ctrl+G` | Toggle compact display |
| `Ctrl+T` | Toggle thinking display |

## Supported Platforms

- Linux x86_64, arm64, ppc64le, s390x
- Linux musl x86_64, arm64
- macOS x86_64, arm64
- Windows x64, arm64

Other platforms (LoongArch64, RISC-V64, *BSD) are available via `npm install -g mothx` or the one-line installer.

## Diagnostics

```bash
vibecoding doctor
```

Checks config files, API connections, sandbox, MCP servers, sessions, and skills.

## Uninstall

```bash
pip uninstall vibecoding-installer

# or if you used pipx:
pipx uninstall vibecoding-installer
```

## Links

- **Homepage** — <https://github.com/startvibecoding/mothx>
- **Documentation** — <https://github.com/startvibecoding/mothx/tree/main/docs>
- **Changelog** — <https://github.com/startvibecoding/mothx/blob/main/docs/en/changelog.md>
- **Issues** — <https://github.com/startvibecoding/mothx/issues>
- **License** — MIT
