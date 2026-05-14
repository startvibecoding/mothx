## What's New in v0.0.5

### ✨ Features

- **Non-root Installation**
  - `install.sh` now supports installation without root or sudo
  - Auto-detects writable install directory: uses `/usr/local/bin` if writable, otherwise falls back to `~/.vibecoding/bin`
  - Removes all `sudo` calls — user-level installation never requires elevated privileges

- **Automatic PATH Setup**
  - Auto-detects user's shell (bash, zsh, fish) and configures PATH in the appropriate config file
  - Supports `.bashrc`, `.bash_profile`, `.zshrc`, `.zshenv`, `config.fish`, and `.profile`
  - Skips configuration if PATH entry already exists (no duplicates)
  - Fish shell uses `set -gx PATH` syntax; bash/zsh use `export PATH=...`

### 🛠 Improvements

- **Environment Variables**
  - `INSTALL_DIR` — override the install directory (unchanged)
  - `AUTO_SETUP_PATH=0` — disable automatic PATH configuration
  - Better error messages for permission issues

- **Install Experience**
  - Shows install directory and PATH auto-setup status at the start
  - Cleaner output with colored status messages

### 📖 Documentation

- Added v0.0.5 release notes

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
go install github.com/fuckvibecoding/vibecoding/cmd/vibecoding@v0.0.5
```

---

**Full Changelog**: https://github.com/fuckvibecoding/vibecoding/compare/v0.0.4...v0.0.5
