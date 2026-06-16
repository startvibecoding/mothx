# 🚀 VibeCoding 5-Minute Quick Start

> Skip the long docs — let's get you coding!

## Step 1: Install (30 seconds)

```bash
# Option 1: npm (recommended, auto-downloads correct binary)
npm install -g vibecoding-installer

# Option 2: One-line install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash

# Option 3: Go install
go install github.com/startvibecoding/vibecoding/cmd/vibecoding@latest
```

**Uninstall:**

```bash
# npm
npm uninstall -g vibecoding-installer

# Linux/macOS (one-line install)
curl -fsSL https://raw.githubusercontent.com/startvibecoding/vibecoding/main/install.sh | bash -s -- --uninstall
```

## Step 2: Configure API Key (30 seconds)

```bash
# Set DeepSeek API key (default provider)
export DEEPSEEK_API_KEY=sk-...

# Or use OpenAI
export OPENAI_API_KEY=sk-...

# Or use Anthropic
export ANTHROPIC_API_KEY=sk-ant-...
```

## Step 3: Run! (10 seconds)

```bash
# Start interactive session
vibecoding

# Or ask directly
vibecoding -P "Hello, VibeCoding!"
```

That's it — you're coding with AI!

---

## 🎮 Three Modes, Switch Anytime

| Mode | Command | Use Case |
|------|---------|----------|
| 🗒️ **Plan** | `vibecoding --mode plan` | Read-only analysis, safe exploration |
| 🔧 **Agent** | `vibecoding --mode agent` | Standard development (default) |
| 🚀 **YOLO** | `vibecoding --mode yolo` | Full freedom, no restrictions |

In interactive mode, press `Tab` or type `/mode plan|agent|yolo` to switch.

---

## 💡 Common Use Cases

### 📝 Code Generation
```bash
vibecoding -P "Write a Go HTTP server with RESTful API"
vibecoding -P "Create a Python scraper for news headlines"
vibecoding -P "Generate a React component with search box and list"
```

### 🔍 Code Understanding
```bash
vibecoding -P "Explain what main.go does"
vibecoding -P "What does this regex mean?"
vibecoding -P "Analyze the architecture of this project"
```

### 🛠️ Code Refactoring
```bash
vibecoding -P "Refactor this function to use generics"
vibecoding -P "Optimize this code for performance"
vibecoding -P "Split this class into smaller modules"
```

### 🧪 Test Generation
```bash
vibecoding -P "Write unit tests for UserService"
vibecoding -P "Generate integration test cases"
vibecoding -P "Write an end-to-end test"
```

### 📚 Documentation
```bash
vibecoding -P "Generate JSDoc comments for this function"
vibecoding -P "Write a README.md"
vibecoding -P "Generate API documentation"
```

---

## ⌨️ Keyboard Shortcuts

| Shortcut | Function |
|----------|----------|
| `Enter` | Submit prompt |
| `Alt+Enter` | New line |
| `Tab` | Switch mode |
| `Esc` | Abort current operation |
| `Ctrl+O` | Open tool details |
| `Ctrl+G` | Toggle compact display |
| `Up/Down` | Browse history |
| `Ctrl+C` | Cancel/clear input |

---

## 🔧 Common Commands

```bash
/mode plan      # Switch to Plan mode
/mode agent     # Switch to Agent mode
/mode yolo      # Switch to YOLO mode
/model          # Show current model
/think          # Toggle thinking level
/clear          # Clear conversation
/help           # Show help
/quit           # Exit
```

---

## 🎯 Real-World Examples

### Example 1: Create a New Project
```bash
vibecoding -P "Create an Express.js project with user auth and database connection"
```

### Example 2: Debug Code
```bash
vibecoding -P "This code throws TypeError: Cannot read property 'map' of undefined, help me fix it"
```

### Example 3: Code Review
```bash
vibecoding --mode plan "Review the code in current directory, find potential issues"
```

### Example 4: Generate Config Files
```bash
vibecoding -P "Generate a Docker Compose file with Node.js, PostgreSQL, and Redis"
```

### Example 5: Write Regex
```bash
vibecoding -P "Write a regex to match Chinese phone numbers"
```

---

## 🛡️ Safety Tips

- **Plan mode**: Read-only, won't modify files — great for exploration
- **Agent mode**: Read/write, but bash commands require approval (configurable whitelist)
- **YOLO mode**: Full freedom — use with caution!

By default, dangerous commands (like `rm -rf`, `sudo`) are blocked by the blacklist.

---

## 📱 Advanced Usage

### IDE Integration
```bash
# VS Code: Add to settings.json
{
  "acp.agents": {
    "vibecoding": {
      "command": "vibecoding",
      "args": ["acp", "--mode", "agent"]
    }
  }
}
```

### API Server
```bash
# Start OpenAI-compatible HTTP server
vibecoding gateway
```

### Chatbot
```bash
# Deploy as WeChat/Feishu bot
vibecoding hermes
```

---

## 🆘 Need Help?

```bash
# Run diagnostics
vibecoding doctor

# Show help
vibecoding --help

# Enable debug mode
vibecoding --debug
```

---

## 📖 More Resources

- [Full Documentation](../../README.md) — All features explained
- [Configuration Guide](configuration.md) — Customize settings
- [Tool Reference](tools.md) — All built-in tools
- [Scenarios & Walkthroughs](scenarios.md) — More real-world examples
- [FAQ](faq.md) — Common questions answered

---

<p align="center">
  <strong>🎉 Congratulations! You've mastered the basics of VibeCoding.</strong><br>
  <strong>Now, start your AI coding journey!</strong>
</p>
