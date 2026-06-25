# VibeCoding Documentation

<p align="center">
  <img src="assets/logo.svg" alt="VibeCoding" width="128" height="128">
</p>

<p align="center">
  <strong>🚀 One Binary to Rule Them All — Your AI Coding Assistant in the Terminal</strong>
</p>

<p align="center">
  Stop switching between Claude Code, Codex, Claw, and Hermes.<br>
  VibeCoding packs everything into a single file — providers, tools, sandbox, sessions, skills, and more.
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/vibecoding-installer"><img src="https://img.shields.io/npm/dm/vibecoding-installer.svg" alt="npm downloads"></a>
  <a href="https://github.com/startvibecoding/vibecoding/releases/latest"><img src="https://img.shields.io/github/release/startvibecoding/vibecoding.svg" alt="GitHub release"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://goreportcard.com/report/github.com/startvibecoding/vibecoding"><img src="https://goreportcard.com/badge/github.com/startvibecoding/vibecoding" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/startvibecoding/vibecoding"><img src="https://pkg.go.dev/badge/github.com/startvibecoding/vibecoding?status.svg" alt="GoDoc"></a>
  <a href="https://github.com/startvibecoding/vibecoding/network/dependencies"><img src="https://img.shields.io/librariesio/release/github/startvibecoding/vibecoding" alt="Dependencies"></a>
</p>

---

Welcome to the VibeCoding Documentation Center!

## ✨ Why VibeCoding?

**The Problem:** You're juggling multiple AI coding tools — Claude Code for one thing, Codex for another, Claw for something else. Each has its own setup, its own quirks, its own dependencies.

**The Solution:** VibeCoding is the **all-in-one terminal AI coding assistant** that does it all. One binary. One config. Zero hassle.

### 🎯 Key Highlights

| Feature | What It Means for You |
|---------|----------------------|
| **🤖 Multi-Provider** | DeepSeek, OpenAI, Anthropic, Volcengine/Doubao, and 20+ vendor adapters — switch models instantly |
| **⚡ Lightning Fast** | SSE streaming, real-time token delivery, cache hit optimization |
| **🧠 Think Mode** | Extended reasoning for complex problems (DeepSeek, o1, Claude) |
| **🛡️ Sandboxed** | bwrap process isolation — safe file ops, network control, approval gates |
| **📝 Sessions** | Persistent SQLite-backed history with branching, compaction, and tree structure |
| **🧩 Skills** | Reusable prompt snippets for project conventions — share across teams |
| **💻 IDE Ready** | ACP protocol for VS Code, Zed, JetBrains — native editor integration |
| **🌐 Gateway** | OpenAI-compatible HTTP API — use VibeCoding as a backend service |
| **📱 Messaging** | WeChat, Feishu, WebSocket — deploy as a chatbot |
| **🤝 Multi-Agent** | Async sub-agents with `--multi-agent`, blocking delegation with `--delegate`, and A2A master mode |
| **🎨 Rich TUI** | Markdown rendering, syntax highlighting, thinking display, tool modals |
| **🔒 Security** | bashBlacklist > whitelist, YOLO mode safety, `--print` fails fast |

---

## 📚 Documentation

### 🚀 Getting Started
- [5-Minute Quick Start](quick-start-tutorial.md) — Skip the long docs, just start coding!
- [Quick Start](getting-started.md) — Installation, configuration, first run
- [CLI Reference](cli-reference.md) — All commands and flags

### ✨ Core Features
- [Features Overview](features-overview.md) — Multi-provider, sandbox, sessions, skills, and more

### 🎯 Use Cases
- [Use Cases & Real-World Examples](use-cases.md) — From daily development to enterprise deployment

### ⚙️ Configuration
- [Configuration Guide](configuration.md) — Settings, env vars, authentication

### 🏗️ Architecture
- [System Architecture](architecture.md) — Core components, data flow
- [Tool System](tools.md) — Built-in tools guide
- [Skills System](skills.md) — Reusable prompt snippets
- [Online Skill Marketplace](skillhub.md) — SkillHub / ClawHub integration
- [Session Management](sessions.md) — Session storage and management
- [SDK Integration](sdk.md) — Embed VibeCoding agent in your Go applications

### 🔒 Security
- [Security & Sandbox](security.md) — Sandbox modes, permissions, approval

### 💻 IDE Integration
- [ACP Protocol](acp.md) — VS Code, Zed, JetBrains integration

### 🌐 Gateway Modes
- [Gateway Mode](gateway.md) — OpenAI-compatible HTTP API
- [Hermes Mode](hermes.md) — WeChat/Feishu/WebSocket chatbot
- [A2A Protocol](a2a.md) — Agent-to-Agent protocol

### 📖 Tutorials
- [Scenarios & Walkthroughs](scenarios.md) — Practical examples
- [FAQ](faq.md) — Common questions answered

### 🇨🇳 中文文档
- [中文文档首页](../zh/README.md) — 完整中文文档

---

## 🚀 Quick Install

```bash
# npm (recommended)
npm install -g vibecoding-installer

# One-line install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash

# Go install
go install github.com/startvibecoding/vibecoding/cmd/vibecoding@latest
```

## 🗑️ Uninstall

```bash
# npm
npm uninstall -g vibecoding-installer

# Linux/macOS (one-line install)
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash -s -- --uninstall

# Windows (one-line install)
irm https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.ps1 | iex; Uninstall-VibeCoding
```

## 🎮 Three Modes for Every Situation

```
🗒️  Plan    → Read-only analysis & planning. Safe, sandboxed, no surprises.
🔧  Agent   → Standard read/write. Bash approval required. (Default)
🚀  YOLO    → Full system access. No restrictions. For the brave.
```

Switch modes anytime with `/mode plan|agent|yolo` or press `Tab`.

## 🎯 Use Cases

### 💻 Daily Development
```bash
vibecoding -P "Refactor this function to use generics"
vibecoding -P "Write tests for the UserService struct"
vibecoding -P "Explain what this regex does"
```

### 🔍 Code Review
```bash
vibecoding --mode plan "Review this PR and suggest improvements"
```

### 🚀 CI/CD Integration
```bash
vibecoding -p "Generate changelog from git log" > CHANGELOG.md
```

### 🌐 API Server
```bash
vibecoding gateway  # Start OpenAI-compatible HTTP server
```

### 📱 Chatbot
```bash
vibecoding hermes   # Deploy as WeChat/Feishu bot
```

## Getting Help

- Use the `/help` command for interactive help
- Check the [CLI Reference](cli-reference.md) for all commands
- Read the [FAQ](faq.md) for common questions
