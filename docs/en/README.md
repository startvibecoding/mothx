# VibeCoding Documentation

<p align="center">
  <img src="assets/logo.svg" alt="VibeCoding" width="128" height="128">
</p>

<p align="center">
  <strong>AI-Powered Terminal Coding Assistant</strong>
</p>

<p align="center">
  <a href="https://github.com/startvibecoding/vibecoding"><img src="https://img.shields.io/github/v/release/startvibecoding/vibecoding" alt="GitHub Release"></a>
  <a href="https://github.com/startvibecoding/vibecoding/blob/main/LICENSE"><img src="https://img.shields.io/github/license/startvibecoding/vibecoding" alt="License"></a>
  <a href="https://github.com/startvibecoding/vibecoding"><img src="https://img.shields.io/github/stars/startvibecoding/vibecoding?style=social" alt="GitHub Stars"></a>
</p>

---

Welcome to the VibeCoding Documentation Center!

<<<<<<< HEAD
## Features

- Multi-provider AI coding assistant for DeepSeek, OpenAI, Anthropic, and compatible custom APIs
- Rich terminal UI with sessions, context management, skills, and sandboxed tool execution
- ACP support: run VibeCoding as an Agent Client Protocol stdio agent for editor integrations and compatible clients, including VS Code, Zed, and JetBrains IDEs such as IntelliJ IDEA/WebStorm via ACP-compatible plugins
=======
## What is VibeCoding?

VibeCoding is a terminal-based AI coding assistant that helps you write, debug, refactor, and understand code. It supports multiple LLM providers including DeepSeek (default), OpenAI, Anthropic, and any OpenAI/Anthropic-compatible API.

### Key Features

- 🤖 **Multi-Provider Support** — DeepSeek, OpenAI, Anthropic, and custom providers
- 🔧 **7 Built-in Tools** — File operations, code search, command execution
- 🛡️ **Sandbox Security** — Process-level isolation via bubblewrap
- 📝 **Session Management** — Persistent conversation history with branching
- 🎯 **3 Operation Modes** — Plan (read-only), Agent (standard), YOLO (full access)
- 🧩 **Skills System** — Reusable prompt snippets for project conventions
- 💻 **IDE Integration** — ACP protocol for VS Code and JetBrains
- 🖼️ **Image Support** — Read and analyze image files
- ⚡ **Prompt Cache** — Reduces API costs by caching repeated prefixes
- 🎨 **Rich TUI** — Markdown rendering, syntax highlighting, thinking display
>>>>>>> a543dfae0c5b47e8da257cbdcc346cb763d2c930

## Table of Contents

### Getting Started
- [Quick Start](getting-started.md) — Installation, configuration, and first run
- [CLI Reference](cli-reference.md) — Complete CLI parameter reference

### Configuration
- [Configuration Guide](configuration.md) — Settings files, environment variables, authentication

### Architecture
- [System Architecture](architecture.md) — Project structure, core components, data flow
- [Tool System](tools.md) — Built-in tools usage guide
- [Skills System](skills.md) — Reusable prompt snippets
- [Session Management](sessions.md) — Session storage and management

### Security
- [Security & Sandbox](security.md) — Sandbox modes, permission control, approval mechanism

### IDE Integration
- [ACP Protocol](acp.md) — Agent Client Protocol for VS Code and JetBrains

### Development
- [Development Guide](development.md) — Contributing code, testing, building

### Reference
- [FAQ](faq.md) — Frequently asked questions
- [Changelog](changelog.md) — Version history and release notes

## Quick Links

| Topic | Description |
|-------|-------------|
| [Quick Start](getting-started.md) | Get started with VibeCoding in 5 minutes |
| [Configuration](configuration.md) | Customize providers, models, and behavior |
| [Tool Reference](tools.md) | Learn about all 7 built-in tools |
| [Security Model](security.md) | Understand sandbox, modes, and permissions |
| [ACP Protocol](acp.md) | IDE integration via Agent Client Protocol |
| [Session Management](sessions.md) | Conversation history and branching |
| [Skills System](skills.md) | Create reusable prompt snippets |
| [Changelog](changelog.md) | See what's new in each release |

## Supported LLMs

| Provider | Models | API Format |
|----------|--------|------------|
| **DeepSeek** (default) | deepseek-v4-flash, deepseek-v4-pro | OpenAI Chat / Anthropic Messages |
| **OpenAI** | GPT-4o, o1, etc. | OpenAI Chat |
| **Anthropic** | Claude Sonnet, Opus, etc. | Anthropic Messages |
| **Custom** | Any compatible model | OpenAI Chat or Anthropic Messages |

## Quick Install

```bash
# npm (recommended)
npm install -g vibecoding-installer

# One-line install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash

# Go install
go install github.com/startvibecoding/vibecoding/cmd/vibecoding@latest
```

## Getting Help

- Use the `/help` command for interactive help
- Check the [CLI Reference](cli-reference.md) for all commands
- Read the [FAQ](faq.md) for common questions
<<<<<<< HEAD
=======
- Visit [GitHub Issues](https://github.com/startvibecoding/vibecoding/issues) to report bugs
>>>>>>> a543dfae0c5b47e8da257cbdcc346cb763d2c930
