# ✨ VibeCoding Core Features Explained

> One tool to meet all your AI coding needs.

---

## 🤖 Multi-Provider Support

### Supported Providers

| Provider | Models | API Format |
|----------|--------|------------|
| **DeepSeek** (default) | deepseek-v4-flash, deepseek-v4-pro | OpenAI Chat / Anthropic Messages |
| **OpenAI** | GPT-4o, o1, etc. | OpenAI Chat |
| **Anthropic** | Claude Sonnet, Opus, etc. | Anthropic Messages |
| **Vendor Adapters** | Google Gemini, Google Vertex, Xiaomi, Kimi, MiniMax, Seed, Qianfan, Bailian, Gitee, OpenRouter, Together, Groq, Fireworks, and more | OpenAI Chat or Anthropic Messages |
| **Custom** | Any compatible model | Generic OpenAI Chat or Anthropic Messages fallback |

### Quick Switch

```bash
# Use DeepSeek
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# Use OpenAI
vibecoding --provider openai --model gpt-4o

# Use Anthropic
vibecoding --provider anthropic --model claude-3-5-sonnet-20241022
```

### Vendor Adapters

VibeCoding supports 15+ vendor adapters, including:

- **China**: Xiaomi, Kimi, MiniMax, Seed, Qianfan, Bailian, Gitee
- **International**: Google Gemini, Google Vertex, OpenRouter, Together, Groq, Fireworks

No additional configuration needed — just set the API key and start using.

---

## ⚡ Lightning Fast

### SSE Streaming

VibeCoding uses SSE (Server-Sent Events) streaming to return tokens in real-time, no need to wait for complete responses.

### Cache Hit Optimization

- **Real-time cache hit rate display**: Shows cache hit percentage in the TUI footer
- **Per-turn cache statistics**: Detailed cache hit info for each conversation turn
- **Unified cache metrics**: Consistent statistics across TUI and print modes

### Performance Comparison

| Feature | VibeCoding | Other Tools |
|---------|-----------|-------------|
| Streaming | ✅ SSE | Partial support |
| Cache hit rate | ✅ Real-time display | ❌ Not displayed |
| Token statistics | ✅ Unified metrics | ❌ Inconsistent |

---

## 🧠 Think Mode

### Supported Models

- **DeepSeek**: deepseek-v4-pro (supports thinking mode)
- **OpenAI**: o1 series (supports thinking mode)
- **Anthropic**: Claude series (supports thinking mode)

### Thinking Levels

```bash
# Set thinking level
vibecoding --thinking off      # No thinking
vibecoding --thinking minimal  # Minimal thinking
vibecoding --thinking low      # Low thinking
vibecoding --thinking medium   # Medium thinking
vibecoding --thinking high     # High thinking
vibecoding --thinking xhigh    # Ultra high thinking
```

### Use Cases

- **Complex problems**: Use `high` or `xhigh` for deeper reasoning
- **Simple problems**: Use `off` or `minimal` for faster responses
- **Debugging code**: Use `medium` for balanced speed and quality

---

## 🛡️ Sandboxed

### bwrap Process Isolation

VibeCoding uses [bubblewrap](https://github.com/containers/bubblewrap) for process-level isolation, ensuring secure execution.

### Three Modes

| Mode | File System | Network | Use Case |
|------|------------|---------|----------|
| **Plan** | Read-only | ❌ | Analysis, planning |
| **Agent** | Read/Write | ❌ | Daily development |
| **YOLO** | Full | ✅ | System-level operations |

### Security Features

- **bashBlacklist priority**: Blacklisted commands (like `rm -rf`, `sudo`) are always blocked
- **Approval gates**: In Agent mode, bash commands require approval
- **`--print` fast fail**: Exits immediately when approval is needed, won't hang

### Installing bwrap

```bash
# Debian/Ubuntu
sudo apt install bubblewrap

# Fedora
sudo dnf install bubblewrap

# Arch
sudo pacman -S bubblewrap
```

---

## 📝 Session Management

### JSONL Persistence

VibeCoding uses JSONL format for persistent session history, supporting:

- **Branching**: Create new branches from any node
- **Compaction**: Automatically compact old sessions to save space
- **Tree structure**: Visualize session trees

### Session Commands

```bash
# Continue most recent session
vibecoding --continue

# Resume specific session
vibecoding --resume <session-id>

# Use specific session file
vibecoding --session <session-file>
```

### Session Storage Location

| Platform | Location |
|----------|----------|
| Linux/macOS | `~/.vibecoding/sessions/` |
| Windows | `%APPDATA%\vibecoding\sessions\` |

---

## 🧩 Skills System

### What Are Skills?

Skills are reusable prompt snippets for:

- **Project conventions**: Coding standards, naming rules, architecture patterns
- **Team sharing**: Unify team development style
- **Quick onboarding**: New members quickly understand project conventions

### Creating Skills

```bash
# Global skills (available for all projects)
~/.vibecoding/skills/<name>/SKILL.md

# Project skills (override global)
.skills/<name>/SKILL.md
```

### Using Skills

```bash
# List available skills
> /skills

# Activate a skill
> /skill my-conventions

# Load skill reference
> /skill_ref my-references
```

### Online Skill Marketplace

VibeCoding is compatible with SkillHub / ClawHub, supporting:

- **Skill installation**: One-click install online skills
- **Cron infrastructure**: Scheduled skill execution
- **Community sharing**: Share your skills

---

## 💻 IDE Integration

### ACP Protocol

VibeCoding supports Agent Client Protocol (ACP), integrating with:

- **VS Code**
- **Zed**
- **JetBrains IDEs** (IntelliJ IDEA, WebStorm, etc.)

### VS Code Integration

Add to `settings.json`:

```json
{
  "acp.agents": {
    "vibecoding": {
      "command": "vibecoding",
      "args": ["acp", "--mode", "agent", "--multi-agent"]
    }
  }
}
```

### JetBrains Integration

Navigate to `Settings → Tools → ACP Agents` and add:

- **Name**: VibeCoding
- **Command**: `vibecoding`
- **Arguments**: `acp --mode agent`

---

## 🌐 Gateway Mode

### OpenAI-Compatible HTTP API

VibeCoding can be used as an OpenAI-compatible HTTP server:

```bash
# Start gateway
vibecoding gateway

# Configuration files
~/.vibecoding/gateway.json
.vibe/gateway.json
```

### Use Cases

- **Team sharing**: Team members share AI coding assistant
- **CI/CD integration**: Use AI in CI/CD pipelines
- **API service**: Provide AI coding API service

---

## 📱 Messaging Platforms

### WeChat/Feishu/WebSocket

VibeCoding can be deployed as a chatbot:

```bash
# Start messaging gateway
vibecoding hermes

# Configuration files
~/.vibecoding/hermes.json
.vibe/hermes.json
```

### Features

- **Multi-user support**: Independent sessions per user
- **Persistent sessions**: Session history automatically saved
- **Progress callbacks**: Real-time tool execution progress
- **Thinking display**: Show AI's thinking process

---

## 🤝 Multi-Agent

### Sub-Agent Delegation

```bash
# Enable multi-agent mode
vibecoding --multi-agent

# Sub-agent tools
subagent_spawn    # Create sub-agent
subagent_status   # Check sub-agent status
subagent_send     # Send message to sub-agent
subagent_destroy  # Destroy sub-agent
```

### Delegate Mode

```bash
# Enable one blocking delegated sub-agent
vibecoding --delegate

# Toggle in TUI or Gateway sessions
/delegate on
/delegate status
/delegate off
```

`--delegate` registers `delegate_subagent`, a synchronous tool for one bounded investigation at a time. Use it when the main agent only needs a summarized result from a focused multi-step task.

### A2A Protocol

```bash
# Enable A2A Master mode
vibecoding --enable-a2a-master

# Generate sample config
vibecoding --init-a2a-master-config
```

### Dynamic Workflows

```bash
# Enable Workflow mode
vibecoding --workflows

# Workflow tools
workflow_run      # Run an Elisp workflow
workflow_status   # Check workflow status
workflow_cancel   # Cancel a workflow run
```

Orchestrate multiple worker agents using an Elisp DSL, with phase grouping, parallel execution, concurrency control, result fan-in, and run logs. Ideal for code audits, architecture research, multi-role reviews, generate-critic loops, and other complex multi-agent scenarios.

See the [Workflow Mode](workflow.md) documentation for full syntax and best practices.

### Use Cases

- **Large tasks**: Split large tasks into multiple subtasks
- **Parallel execution**: Multiple sub-agents working in parallel
- **Remote collaboration**: Manage remote agents via A2A protocol

---

## 🎨 Rich TUI

### Features

- **Markdown rendering**: Real-time Markdown content rendering
- **Syntax highlighting**: Code block syntax highlighting
- **Thinking display**: Show AI's thinking process
- **Tool modals**: View tool execution details
- **Status bar**: Show cache hit rate, token statistics, etc.

### Keyboard Shortcuts

| Shortcut | Function |
|----------|----------|
| `Ctrl+O` | Open tool details |
| `Ctrl+G` | Toggle compact display |
| `Ctrl+T` | Toggle thinking display |
| `Tab` | Switch mode |
| `Esc` | Abort current operation |

---

## 🔒 Security Control

### Approval Mechanism

```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm "],
    "bashBlacklist": ["rm -rf", "sudo"],
    "confirmBeforeWrite": true
  }
}
```

### Security Features

- **bashBlacklist priority**: Blacklisted commands are always blocked
- **YOLO mode safety**: Even in YOLO mode, blacklist still applies
- **`--print` fast fail**: Exits immediately when approval is needed

---

## 📊 Unified Cache Metrics

### Real-time Display

- **Cache hit rate**: Real-time display in TUI footer
- **Token statistics**: Detailed token usage per conversation turn
- **Unified metrics**: Consistent statistics across TUI and print modes

### Use Cases

- **Cost optimization**: Monitor cache hit rate to optimize API call costs
- **Performance monitoring**: Monitor token usage to optimize response speed

---

## 🐞 Consistent Debug Output

### `--debug` Mode

```bash
# Enable debug mode
vibecoding --debug

# Debug output includes
- Provider request/response
- ACP mode debugging
- Tool execution details
```

### Use Cases

- **Troubleshooting**: Quickly locate API call issues
- **Development debugging**: Debug custom tools or skills
- **Performance analysis**: Analyze API call performance

---

## 🩺 Environment Diagnostics

### `doctor` Subcommand

```bash
# Run environment diagnostics
vibecoding doctor
```

### Check Items

- **Configuration files**: Check settings.json, gateway.json, hermes.json
- **Provider**: Check API keys and connections
- **Sandbox**: Check bwrap installation and configuration
- **MCP**: Check MCP server configuration
- **Sessions**: Check session storage
- **Skills**: Check skill loading
- **Context files**: Check AGENTS.md, CLAUDE.md, etc.

---

## 🎯 Summary

VibeCoding is an **all-in-one terminal AI coding assistant** with these core advantages:

1. **One binary does it all**: No need to install multiple tools
2. **Multi-provider support**: 15+ vendor adapters, one-click switching
3. **Security control**: Sandbox isolation, approval gates, blacklist protection
4. **Rich features**: Session management, skills system, IDE integration
5. **High performance**: SSE streaming, cache hit optimization

**Ready to get started?** [5-Minute Quick Start](quick-start-tutorial.md)
