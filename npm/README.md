> **Rename notice:** VibeCoding is now **MothX**. This legacy npm install name is `vibecoding-installer`; use `mothx-installer` (`npm install -g mothx-installer`) for new installs and future updates.

# vibecoding-installer compatibility package

This package is kept for a smooth transition to **MothX**.

> `vibecoding-installer` is a compatibility package for this release. Future updates should use `mothx-installer`.

A terminal-based AI coding assistant. One binary, multiple providers, sandbox, sessions, skills, and more.

## Installation

```bash
npm install -g vibecoding-installer
# Future updates:
npm install -g mothx-installer
```

## Quick Start

```bash
# Set API key
export DEEPSEEK_API_KEY=sk-...

# Start interactive mode
vibecoding

# Non-interactive (print mode)
vibecoding -P "Write a hello world in Go"

# Specify provider and model
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# Change mode
vibecoding --mode plan    # Read-only planning
vibecoding --mode agent   # Standard (default)
vibecoding --mode yolo    # Full access
```

## Features

- ⚙️ **Workflow Mode**: Dynamic Elisp workflows with phases, parallel execution, and multi-worker coordination
- 🤖 **Multi-Provider**: DeepSeek, OpenAI, Anthropic, Volcengine/Doubao, 20+ vendor adapters
- ⚡ **SSE Streaming**: Real-time token delivery
- 🧠 **Think Mode**: Extended reasoning for complex problems
- 🛡️ **Sandbox**: bwrap process isolation (Linux)
- 📝 **Sessions**: Persistent SQLite-backed history with branching
- 🧩 **Skills**: Reusable prompt snippets
- 💻 **IDE Integration**: ACP protocol for VS Code, JetBrains
- 🌐 **Gateway**: OpenAI-compatible HTTP API
- 📱 **Messaging**: WeChat, Feishu, WebSocket chatbot
- 🤝 **Multi-Agent**: Delegate subtasks, A2A protocol

## Configuration

```bash
# DeepSeek (default)
export DEEPSEEK_API_KEY=sk-...

# OpenAI
export OPENAI_API_KEY=sk-...

# Anthropic
export ANTHROPIC_API_KEY=sk-ant-...
```

Or configure in `~/.mothx/settings.json`:

```json
{
  "defaultProvider": "deepseek-openai",
  "defaultModel": "deepseek-v4-flash"
}
```

## More Information

- **GitHub**: [github.com/startvibecoding/mothx](https://github.com/startvibecoding/mothx)
- **Gitee (国内)**: [gitee.com/startvibecoding/mothx](https://gitee.com/startvibecoding/mothx)
- **Documentation**: [docs](https://github.com/startvibecoding/mothx/tree/main/docs)

## Uninstall

```bash
# npm
npm uninstall -g vibecoding-installer

# Linux/macOS (one-line install)
curl -fsSL https://gitee.com/startvibecoding/mothx/raw/main/install.sh | bash -s -- --uninstall

# Windows (one-line install)
irm https://gitee.com/startvibecoding/mothx/raw/main/install.ps1 | iex; Uninstall-VibeCoding
```

## License

MIT
