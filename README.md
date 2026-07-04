<p align="center">
  <img src="docs/assets/mothx.png" alt="MothX" width="128" height="128">
</p>

<h1 align="center">MothX</h1>

<p align="center">
  <strong>🚀 One Binary to Rule Them All — Your AI Coding Assistant in the Terminal</strong>
</p>

<p align="center">
  Stop switching between Claude Code, Codex, Claw, and Hermes.<br>
  MothX packs everything into a single file — providers, tools, sandbox, sessions, skills, and more.
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/mothx-installer"><img src="https://img.shields.io/npm/dm/mothx-installer.svg" alt="npm downloads"></a>
  <a href="https://pypi.org/project/vibecoding-installer/"><img src="https://img.shields.io/pypi/v/vibecoding-installer.svg" alt="PyPI version"></a>
  <a href="https://github.com/startvibecoding/mothx/releases/latest"><img src="https://img.shields.io/github/release/startvibecoding/mothx.svg" alt="GitHub release"></a>
  <a href="https://gitee.com/startvibecoding/mothx/releases/latest"><img src="https://img.shields.io/badge/Gitee-release-blue" alt="Gitee release"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://goreportcard.com/report/github.com/startvibecoding/mothx"><img src="https://goreportcard.com/badge/github.com/startvibecoding/mothx" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/startvibecoding/mothx"><img src="https://pkg.go.dev/badge/github.com/startvibecoding/mothx?status.svg" alt="GoDoc"></a>
  <a href="https://github.com/startvibecoding/mothx/network/dependencies"><img src="https://img.shields.io/librariesio/release/github/startvibecoding/mothx" alt="Dependencies"></a>
</p>

<p align="center">
  <strong>国内镜像: <a href="https://gitee.com/startvibecoding/mothx">Gitee</a></strong>
</p>

> **Rename notice:** MothX was formerly known as VibeCoding. During this transition release, legacy entry points such as the `vibecoding` command, the `vibecoding-installer` npm package, and the existing `.vibecoding` configuration directory are kept for compatibility. New installs and future updates should use `mothx`.

---

## ✨ Why MothX?

**The Problem:** You're juggling multiple AI coding tools — Claude Code for one thing, Codex for another, Claw for something else. Each has its own setup, its own quirks, its own dependencies.

**The Solution:** MothX is the **all-in-one terminal AI coding assistant** that does it all. One binary. One config. Zero hassle.

### 🎯 Key Highlights

| Feature | What It Means for You |
|---------|----------------------|
| **⚙️ Workflow Mode** | Dynamic Elisp workflows with phases, parallel execution, and multi-worker coordination — automate complex development pipelines |
| **🤖 Multi-Provider** | DeepSeek, OpenAI, Anthropic, Volcengine/Doubao, LongCat, Mistral, GitHub Copilot, Cloudflare, Amazon Bedrock, and 20+ vendor adapters — switch models instantly |
| **⚡ Lightning Fast** | SSE streaming, real-time token delivery, cache hit optimization |
| **🧠 Think Mode** | Extended reasoning for complex problems (DeepSeek, o1, Claude, inline `<think>` parsing) |
| **🛡️ Sandboxed** | bwrap process isolation — safe file ops, network control, approval gates |
| **📝 Sessions** | Persistent SQLite-backed history with branching, compaction, and tree structure |
| **🧩 Skills** | Reusable prompt snippets for project conventions — share across teams |
| **💻 IDE Ready** | ACP protocol for VS Code, Zed, JetBrains — native editor integration |
| **🌐 Gateway** | OpenAI-compatible HTTP API — use MothX as a backend service |
| **📱 Messaging** | WeChat, Feishu, WebSocket — deploy as a chatbot |
| **🤝 Multi-Agent** | Async sub-agents with `--multi-agent`, blocking delegation with `--delegate`, and A2A master mode |
| **🎨 Rich TUI** | Markdown rendering, syntax highlighting, thinking display, tool modals, multiline input |
| **📊 Stats Dashboard** | Web-based usage analytics with charts, filtering by time/vendor/protocol, and CLI fallback |
| **🖼️ Multimodal** | Image preprocessing, crop support, browser screenshots, and vision model integration |
| **🔒 Security** | bashBlacklist > whitelist, YOLO mode safety, `--print` fails fast |
| **📦 Pure Go** | No external binary dependencies — uses pure-Go `grep`/`find` SDKs, supports FreeBSD |
| **⚡ Approval V2** | Interactive approval dialog, project-level bash auto-approval rules, and auto-edit whitelists |

---

## 🚀 Get Started in 30 Seconds

```bash
# Install (pick one)
npm install -g mothx-installer               # npm (recommended)
pipx install vibecoding-installer           # PyPI
curl -fsSL https://raw.githubusercontent.com/startvibecoding/mothx/main/install.sh | bash  # Linux/macOS/FreeBSD (GitHub)
curl -fsSL https://gitee.com/startvibecoding/mothx/raw/main/install.sh | bash  # Linux/macOS/FreeBSD (Gitee 国内镜像)

# Set your API key
export DEEPSEEK_API_KEY=sk-...

# Run
mothx
```

That's it. You're coding with AI.

**Supported Platforms:** Linux (x86_64, arm64), macOS (x86_64, arm64), Windows (x86_64), FreeBSD (x86_64, arm64)

**Uninstall:**

```bash
# npm
npm uninstall -g mothx-installer

# Compatibility package, if installed before the rename
npm uninstall -g vibecoding-installer

# PyPI
pipx uninstall vibecoding-installer

# Linux/macOS (one-line install)
curl -fsSL https://gitee.com/startvibecoding/mothx/raw/main/install.sh | bash -s -- --uninstall

# Windows (one-line install)
irm https://gitee.com/startvibecoding/mothx/raw/main/install.ps1 | iex; Uninstall-MothX
```

---

## 🎮 Three Modes for Every Situation

```
🗒️  Plan    → Read-only analysis & planning. Safe, sandboxed, no surprises.
🔧  Agent   → Standard read/write. Bash approval required. (Default)
🚀  YOLO    → Full system access. No restrictions. For the brave.
```

Switch modes anytime with `/mode plan|agent|yolo` or press `Tab`.

---

## 🏗️ Architecture at a Glance

```
mothx/
├── cmd/mothx/        # CLI entry point
├── internal/
│   ├── agent/             # Core agent loop
│   ├── provider/          # LLM provider abstraction (20+ vendors)
│   ├── tools/             # Built-in tools (read, write, bash, grep, find, ...)
│   ├── sandbox/           # bwrap sandbox implementation
│   ├── session/           # SQLite session storage
│   ├── skills/            # Skills system
│   ├── tui/               # Terminal UI (BubbleTea + Lipgloss)
│   ├── gateway/           # OpenAI-compatible HTTP gateway
│   ├── hermes/            # Messaging gateway (WeChat/Feishu/WebSocket)
│   ├── a2a/               # A2A protocol server & master mode
│   ├── acp/               # ACP / MCP integration
│   ├── stats/             # Usage statistics web dashboard
│   ├── workflow/          # Elisp workflow runtime
│   └── memory/            # Persistent memory (memory.md)
└── pkg/sdk/               # Public SDK interface
```

---

## 📚 Documentation

### 🚀 Getting Started
- [Quick Start](docs/en/getting-started.md) — Installation, configuration, first run
- [CLI Reference](docs/en/cli-reference.md) — All commands and flags

### ⚙️ Configuration
- [Configuration Guide](docs/en/configuration.md) — Settings, env vars, authentication
- [Provider Guide](docs/en/provider-guide.md) — Provider/vendor configuration

### 🏗️ Architecture
- [System Architecture](docs/en/architecture.md) — Core components, data flow
- [Tool System](docs/en/tools.md) — Built-in tools guide
- [Skills System](docs/en/skills.md) — Reusable prompt snippets
- [Online Skill Marketplace](docs/en/skillhub.md) — SkillHub / ClawHub integration
- [Dynamic Workflows](docs/en/workflows.md) — Elisp workflow automation

### 🔒 Security
- [Security & Sandbox](docs/en/security.md) — Sandbox modes, permissions, approval

### 💻 IDE Integration
- [ACP Protocol](docs/en/acp.md) — VS Code, Zed, JetBrains integration

### 🌐 Gateway Modes
- [Gateway Mode](docs/en/gateway.md) — OpenAI-compatible HTTP API
- [Hermes Mode](docs/en/hermes.md) — WeChat/Feishu/WebSocket chatbot
- [A2A Protocol](docs/en/a2a.md) — Agent-to-Agent protocol

### 📊 Analytics
- [Stats Dashboard](docs/en/stats.md) — Usage statistics and monitoring

### 📖 Tutorials
- [Scenarios & Walkthroughs](docs/en/scenarios.md) — Practical examples
- [FAQ](docs/en/faq.md) — Common questions answered

### 🇨🇳 中文文档
- [中文文档首页](docs/zh/README.md) — 完整中文文档

---

## 🎯 Use Cases

### 💻 Daily Development
```bash
mothx -P "Refactor this function to use generics"
mothx -P "Write tests for the UserService struct"
mothx -P "Explain what this regex does"
```

### 🔍 Code Review
```bash
mothx --mode plan "Review this PR and suggest improvements"
```

### 🚀 CI/CD Integration
```bash
mothx -p "Generate changelog from git log" > CHANGELOG.md
```

### 🌐 API Server
```bash
mothx gateway  # Start OpenAI-compatible HTTP server
```

### 📱 Chatbot
```bash
mothx hermes   # Deploy as WeChat/Feishu bot
```

### 📊 Usage Analytics
```bash
mothx stats    # Start web dashboard on 127.0.0.1:7878
mothx stats --cli  # Print stats in terminal
```

### 🔄 Dynamic Workflows
```bash
mothx --workflows  # Enable Elisp workflow automation
# Use workflow_run, workflow_status, workflow_cancel tools
```

### 🖼️ Image Analysis
```bash
mothx -P "Describe this screenshot" --image screenshot.png
mothx -P "Extract text from this image" --image document.jpg
```

---

## 🛠️ Built-in Tools

| Tool | Description |
|------|-------------|
| `read` | Read file contents |
| `write` | Create/overwrite files |
| `edit` | Precise text replacement |
| `bash` | Execute shell commands |
| `grep` | Search file contents (powered by pure-Go ripgrep) |
| `find` | Find files by pattern (powered by pure-Go fd) |
| `ls` | List directory contents |
| `plan` | Publish task plans |
| `jobs` | Manage background jobs |
| `kill` | Stop background jobs |
| `skill_ref` | Load skill references |
| `workflow_run` | Execute Elisp workflow DSL |
| `workflow_status` | Check workflow run status |
| `workflow_cancel` | Cancel running workflows |
| `delegate_subagent` | Blocking single sub-agent delegation |
| `subagent_spawn` | Async sub-agent execution |
| `question` | Interactive user prompts (plan/agent modes) |

---

## 🔧 Configuration

### Settings Files

| Location | Platform | Scope |
|----------|----------|-------|
| `~/.vibecoding/settings.json` | Linux/macOS/FreeBSD | Global |
| `%APPDATA%\vibecoding\settings.json` | Windows | Global |
| `.vibe/settings.json` | All | Project (overrides global) |

The `.vibecoding` config directory and `VIBECODING_*` environment variables are retained as compatibility interfaces for this rename.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DEEPSEEK_API_KEY` | DeepSeek API key |
| `VIBECODING_DIR` | Override config directory |
| `VIBECODING_PROVIDER` | Override default provider |
| `VIBECODING_MODEL` | Override default model |
| `VIBECODING_MODE` | Override default mode |
| `VIBECODING_DEBUG` | Enable debug output |
| `VIBECODING_NO_UPDATE_CHECK` | Disable update notifications |
| `VIBECODING_NPM_REGISTRY` | Override npm registry URL |

### Gateway Configuration

Gateway-specific config lives in `gateway.json` (global `~/.config/vibecoding/gateway.json`, project `.vibe/gateway.json`). See [Gateway Mode](docs/en/gateway.md) for details.

### Hermes Configuration

Hermes-specific config lives in `hermes.json` (global `<GLOBAL_DIR>/hermes.json`, project `.vibe/hermes.json`). See [Hermes Mode](docs/en/hermes.md) for details.

---

## 🤝 Contributing

We welcome contributions! See [Development Guide](docs/en/development.md) for details.

```bash
git clone https://github.com/startvibecoding/mothx.git
cd mothx
make build
make test
```

---

## 📄 License

MIT — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <strong>Ready to vibe? ⭐ Star this repo and start coding!</strong>
</p>
