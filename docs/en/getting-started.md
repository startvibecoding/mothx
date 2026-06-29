# Quick Start

This guide helps you get started with VibeCoding in 5 minutes.

## System Requirements

- **Operating System**: Linux, macOS, Windows (WSL)
- **Go**: 1.24+ (when building from source)
- **Optional**: bubblewrap (for sandbox functionality)

## Installation

### Method 1: npm (Recommended)

```bash
npm install -g vibecoding-installer
```

This will automatically download the correct binary for your platform.

### Method 2: PyPI

```bash
pipx install vibecoding-installer
```

Or without pipx:

```bash
python -m pip install --user vibecoding-installer
```

This installs the platform-specific wheel for your machine.

### Method 3: One-line Install

**Linux/macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.ps1 | iex
```

Or with custom install directory:

```bash
# Linux/macOS
INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash

# Windows
$env:VIBECODING_INSTALL_DIR="C:\Tools\vibecoding"; irm https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.ps1 | iex
```

**Uninstall:**

```bash
# npm
npm uninstall -g vibecoding-installer

# PyPI
pipx uninstall vibecoding-installer

# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash -s -- --uninstall

# Windows
irm https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.ps1 | iex; Uninstall-VibeCoding
```

This will automatically download the latest release from GitHub and install the binary. Default install locations:
- Linux/macOS: `/usr/local/bin`
- Windows: `%LOCALAPPDATA%\vibecoding`

### Method 4: Go Install

```bash
go install github.com/startvibecoding/vibecoding/cmd/vibecoding@latest
```

### Method 5: Build from Source

```bash
# Clone repository
git clone https://github.com/startvibecoding/vibecoding.git
cd vibecoding

# Build
make build

# Binary is located at bin/vibecoding
```

### Method 6: Install to System

```bash
# After building from source
make install
```

## Configure API Keys

### Option 1: Environment Variables

```bash
# DeepSeek
export DEEPSEEK_API_KEY=sk-...
```

### Option 2: Settings File

Or add keys directly to your settings.json:

```json
{
  "providers": {
    "deepseek-openai": {
      "vendor": "deepseek",
      "api": "openai-chat",
      "baseUrl": "https://api.deepseek.com",
      "apiKey": "sk-..."
    }
  }
}
```

The optional `vendor` field selects a vendor adapter. If it is omitted, VibeCoding detects the vendor from `baseUrl` when possible and otherwise falls back to the generic provider selected by `api`. See the [Configuration Guide](configuration.md) for details.

## First Run

### Interactive Mode

```bash
# Start interactive session
vibecoding
```

### Non-Interactive Mode

```bash
# Single question
vibecoding -p "Explain what this code does"

# Read from stdin
echo "Write a Hello World" | vibecoding -P
```

### Specify Model

```bash
# Use DeepSeek-V4-Flash
vibecoding --provider deepseek-openai --model deepseek-v4-flash

# Use DeepSeek-V4-Pro
vibecoding --provider deepseek-openai --model deepseek-v4-pro
```

### Multi-Agent Mode

```bash
# Enable sub-agent tools and multi-agent commands
vibecoding --multi-agent

# ACP sessions can opt in too
vibecoding acp --multi-agent
```

Multi-agent mode registers `subagent_*` tools for delegated work. Cron command entry points are available in TUI multi-agent workflows.

### Delegate Mode

```bash
# Enable blocking single sub-agent delegation
vibecoding --delegate

# Toggle it at runtime in TUI
/delegate on
/delegate status
/delegate off
```

Delegate mode registers `delegate_subagent`, a synchronous tool for one bounded subtask at a time. It is useful for multi-step investigations or focused verification where the main agent only needs a summarized result.

### A2A Master Mode

```bash
# Generate sample config
vibecoding --init-a2a-master-config

# Enable master mode
vibecoding --enable-a2a-master
```

A2A Master mode lets you manage multiple remote A2A agents, with the LLM automatically dispatching tasks via the `a2a_dispatch` tool. See [A2A Protocol](a2a.md) for details.

## Choose Mode

VibeCoding provides three modes:

```bash
# Plan mode - read-only analysis
vibecoding --mode plan

# Agent mode - standard read/write (default)
vibecoding --mode agent

# YOLO mode - full access
vibecoding --mode yolo
```

| Mode | File System | Network | Use Case |
|------|------------|---------|----------|
| **Plan** | Read-only | ✗ | Analysis, planning |
| **Agent** | Read/Write | ✗ | Daily development |
| **YOLO** | Full | ✓ | System-level operations |

## Basic Interaction

### Common Commands

```bash
/mode plan      # Switch to Plan mode
/mode agent     # Switch to Agent mode
/model          # View current model
/think          # Toggle thinking level
/clear          # Clear conversation
/help           # Show help
/quit           # Exit
```

### Keyboard Shortcuts

| Shortcut | Function |
|----------|----------|
| `Enter` | Submit the current prompt |
| `Alt+Enter` / `Ctrl+J` | Insert a newline in the prompt editor |
| `Tab` | Cycle mode (`plan` → `agent` → `yolo`) |
| `Esc` | Abort the current operation, approval, or question prompt |
| `Ctrl+O` | Open the latest tool/details modal; press again, `Esc`, or `q` to close |
| `Ctrl+G` | Toggle compact tool display mode |
| `Up` / `Down` | Move within multiline input; browse prompt history at the first/last input line; scroll tool details when the modal is open |
| `PgUp` / `PgDn` | Page through tool details when the modal is open |
| `Home` / `End` | Move to the start/end of the current input line; jump to top/bottom when the tool modal is open |

Completed conversation blocks are printed to the terminal's native scrollback, so use your terminal's normal mouse wheel, trackpad, or scrollbar to review older transcript output.

## Usage Examples

### Code Explanation

```bash
vibecoding -P "Explain the purpose of main.go"
```

### Code Generation

```bash
vibecoding -P "Write a Go HTTP server"
```

### File Operations

```bash
vibecoding -P "Create a README.md in the current directory"
```

### Continue Session

```bash
# Continue most recent session
vibecoding --continue

# Resume specific session
vibecoding --resume <session-id>
```

## Skills System

Skills are reusable prompt snippets that help enforce project conventions:

```bash
# List available skills
> /skills

# Activate a skill
> /skill my-conventions
```

Create skills by adding `SKILL.md` files:
- **Global**: `~/.vibecoding/skills/<name>/SKILL.md` (available for all projects)
- **Project**: `.skills/<name>/SKILL.md` (project-specific, overrides global)

See the [Skills System](skills.md) documentation for details.

## IDE Integration

VibeCoding can be integrated into your IDE via the Agent Client Protocol (ACP):

### VS Code

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

### JetBrains IDEs

Navigate to `Settings → Tools → ACP Agents` and add:
- **Name**: VibeCoding
- **Command**: `vibecoding`
- **Arguments**: `acp --mode agent`

See the [ACP Protocol](acp.md) documentation for details.

## Troubleshooting

If something doesn't work, run the built-in diagnostics:

```bash
vibecoding doctor
```

This checks your environment, config files, providers, sandbox, MCP, sessions, skills, and context files, and reports any issues.

## Next Steps

- Read the [Configuration Guide](configuration.md) to customize settings
- Check the [Tool Reference](tools.md) to learn about available tools
- Try [multi-agent mode](cli-reference.md#multi-agent-mode) for delegated investigation and cron command entry points
- Try [delegate mode](cli-reference.md#delegate-mode) for blocking single sub-agent investigation tasks
- Understand the [Security Model](security.md) to protect your system
- Explore the [Skills System](skills.md) to create reusable prompt snippets
- Set up [IDE Integration](acp.md) with VS Code or JetBrains
- Check out [Scenarios & Walkthroughs](scenarios.md) for practical usage examples
