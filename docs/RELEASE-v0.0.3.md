## What's New in v0.0.3

### ✨ Features

- **Session History Loading**
  - Display session info (file path and message count) when continuing or opening sessions
  - Load and display historical messages from previous sessions in TUI
  - Load history messages into agent context for continuity
  - Reset agent on abort to ensure clean state for next request

### 🛠 Improvements

- **Build & Distribution System**
  - Restructured Makefile with clear per-platform build and dist targets
  - Added `dist-linux`, `dist-darwin`, `dist-windows` targets
  - Added `build-zip.sh` for Windows zip packages
  - Added `checksums` target for release verification
  - Updated `build-deb.sh` and `build-tarball.sh` to support all platforms

### 📖 Documentation

- Added GitHub repository button in documentation site header
- Added v0.0.2 release notes

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
go install github.com/fuckvibecoding/vibecoding/cmd/vibecoding@v0.0.3
```

---

**Full Changelog**: https://github.com/fuckvibecoding/vibecoding/compare/v0.0.2...v0.0.3
