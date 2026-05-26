# VibeCoding Documentation

<p align="center">
  <img src="assets/logo.svg" alt="VibeCoding" width="128" height="128">
</p>

<p align="center">
  <strong>AI-Powered Terminal Coding Assistant</strong>
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

## Features

- Multi-provider AI coding assistant for DeepSeek, OpenAI, Anthropic, and compatible custom APIs
- Rich terminal UI with sessions, context management, skills, and sandboxed tool execution
- ACP support: run VibeCoding as an Agent Client Protocol stdio agent for editor integrations and compatible clients, including VS Code, Zed, and JetBrains IDEs such as IntelliJ IDEA/WebStorm via ACP-compatible plugins
- Safer approval handling: `bashBlacklist` now overrides whitelist entries, including in YOLO mode, and `--print` exits early when approval would be required
- Unified cache metrics across TUI and print mode for cache hit rate and token totals
- Consistent provider debug output via `--debug`, including ACP mode

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
