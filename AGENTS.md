# VibeCoding - AI Coding Assistant

## Project Overview

VibeCoding is a terminal-based AI coding assistant written in Go, inspired by [pi.dev](https://pi.dev). It supports multiple LLM providers (OpenAI, Anthropic, and custom providers), sandboxed execution via bubblewrap, and a rich TUI built with BubbleTea.

## Tech Stack

- **Language**: Go 1.24+
- **TUI**: BubbleTea + Lipgloss + Glamour
- **CLI**: Cobra
- **Sandbox**: bubblewrap (bwrap) on Linux

## Project Structure

```
vibecoding/
├── cmd/vibecoding/          # CLI entry point (main.go)
├── docs/                    # Project documentation
│   ├── architecture.md      # Architecture details
│   ├── cli-reference.md     # CLI command reference
│   ├── configuration.md     # Configuration guide
│   ├── development.md       # Development guide
│   ├── security.md          # Security documentation
│   ├── sessions.md          # Session management
│   ├── skills.md            # Skills documentation
│   ├── tools.md             # Tools documentation
│   ├── zh/                  # Chinese documentation
│   └── en/                  # English documentation
├── internal/
│   ├── agent/               # Core agent loop and system prompts
│   ├── config/              # Settings, auth, configuration
│   ├── context/             # Context management and token estimation
│   ├── contextfiles/        # Context file discovery (AGENTS.md, CLAUDE.md, etc.)
│   ├── platform/            # Cross-platform compatibility utilities
│   ├── provider/            # LLM provider abstraction
│   │   ├── anthropic/       # Anthropic Messages API
│   │   └── openai/          # OpenAI Chat Completions API
│   ├── sandbox/             # Sandbox abstraction (bwrap, none)
│   ├── session/             # Session management (JSONL format)
│   ├── skills/              # Skills system
│   ├── tools/               # Tool implementations
│   ├── tui/                 # Terminal UI
│   └── ua/                  # User-Agent string generation
└── pkg/sdk/                 # Public SDK (future)
```

## Architecture

### Provider System
- `provider.Provider` interface: `Chat(ctx, params) <-chan StreamEvent`
- Each provider implements SSE streaming for its API format
- Custom providers configured in `settings.json` with `api` field: `"openai-chat"` or `"anthropic-messages"`

### Agent Loop
1. Build system prompt (mode + tools + context files + skills)
2. Send messages to provider
3. Process stream events (text, thinking, tool calls)
4. Execute tools and append results
5. Repeat until done

### Tools
- `read`: File reading with offset/limit
- `write`: File creation
- `edit`: Precise text replacement
- `bash`: Command execution (through sandbox if enabled)
- `grep`: Content search
- `find`: File search
- `ls`: Directory listing

### Sandbox Levels
- `none`: No restrictions (default)
- `standard`: Project read-write, no network (via --sandbox)
- `strict`: Project read-only, no network (Plan mode)

### Mode Permissions
- `plan`: Read-only tools only (read, grep, find, ls)
- `agent`: Read/write/edit auto-execute; bash requires user approval (with whitelist support)
- `yolo`: All tools auto-execute without approval

### Approval Configuration
In `settings.json`, configure approval whitelist:
```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm ", "yarn "],
    "bashBlacklist": ["rm -rf", "sudo"]
  }
}
```

## Build & Run

```bash
# Build
make build

# Run
./bin/vibecoding

# Install
make install

# Cross-compile for all platforms
make build-all

# Build distribution packages\make dist
```

## Configuration

Config file: `~/.vibecoding/settings.json`

Key settings:
- `providers`: Multi-provider configuration
- `defaultProvider` / `defaultModel`: Default selections
- `defaultMode`: "plan", "agent", or "yolo"
- `defaultThinkingLevel`: "off", "minimal", "low", "medium", "high", "xhigh"
- `maxContextTokens`: Maximum context window size
- `maxOutputTokens`: Maximum output tokens
- `sandbox.enabled`: Enable sandbox (default: false)
- `contextFiles.enabled`: Auto-load context files
- `compaction`: Context compaction settings
- `retry`: Retry settings for API calls
- `theme`: UI theme ("dark" or "light")
- `shellPath`: Custom shell path for bash tool
- `shellCommandPrefix`: Custom command prefix

## Code Conventions

- Use `json.RawMessage` for JSON Schema parameters
- Error handling: return errors, don't panic
- Context propagation: pass `context.Context` through tool execution
- Channel-based streaming: providers return `<-chan StreamEvent`
- Keep tools stateless; registry holds sandbox/workdir references

## Session Format

JSONL files with tree structure:
- `id` / `parentId` for branching
- Entry types: `session`, `message`, `model_change`, `compaction`, `label`
- Stored in `~/.vibecoding/sessions/--<encoded-path>--/`

## Skills System

Skills are reusable prompt snippets stored as SKILL.md files:
- Global skills: `~/.vibecoding/skills/<name>/SKILL.md`
- Project skills: `.skills/<name>/SKILL.md` (overrides global)
- Project skills override global skills with the same name

## Testing

```bash
make test
```

## Git Workflow

- Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`
- Main branch: `main`
- No force push to main
