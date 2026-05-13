## What's New in v0.0.4

### ✨ Features

- **Agent Mode Approval Mechanism**
  - Bash commands in Agent mode now require user approval
  - Configurable `bashWhitelist` for auto-approved command prefixes
  - Configurable `bashBlacklist` for commands always requiring approval
  - TUI displays approval prompt; user responds with `y`/`yes` or `n`/`no`
  - Approval requests can be cancelled via `abort`

- **Mode Permission Matrix**
  - Plan mode: Read-only tools (read, grep, find, ls)
  - Agent mode: Read/write auto-execute, bash requires approval
  - YOLO mode: All tools auto-execute
  - Updated system prompts with explicit permission matrix

### 🛠 Improvements

- **Default Approval Whitelist**
  - Default whitelist: `go`, `make`, `git`, `npm`, `yarn`, `node`, `python`, `pip`
  - Customizable in `settings.json`

- **Mode Switch Feedback**
  - Mode switching now shows detailed permission descriptions
  - `/mode` command displays full permission list for current mode

### 📖 Documentation

- Added approval configuration section
- Updated security docs with approval mechanism details

### 📦 Installation

**Linux/macOS:**
```bash
curl -fsSL https://raw.githubusercontent.com/fuckvibecoding/vibecoding/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/fuckvibecoding/vibecoding/main/install.ps1 | iex
```

**Go Install:**
```bash
go install github.com/fuckvibecoding/vibecoding/cmd/vibecoding@v0.0.4
```

---

**Full Changelog**: https://github.com/fuckvibecoding/vibecoding/compare/v0.0.3...v0.0.4
