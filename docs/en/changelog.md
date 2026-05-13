# Changelog

## v0.0.4

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
- Added v0.0.4 release notes

---

## v0.0.3

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

---

## v0.0.2

### ✨ Features

- **One-line Installation Scripts**
  - `install.sh` for Linux/macOS - downloads from GitHub Releases automatically
  - `install.ps1` for Windows PowerShell - supports custom install directory via `VIBECODING_INSTALL_DIR`
  - Both scripts detect platform/architecture, verify checksums, and configure PATH

- **Documentation Redesign**
  - Redesigned with Google Material Design style
  - Default language changed to English
  - Added hash routing for easy document sharing (e.g., `#/en/README`, `#/zh/configuration`)
  - Added logo to header and README

- **Brand Assets**
  - Added `docs/assets/icon.svg` (512×512) for packaging
  - Added `docs/assets/logo.svg` (128×128) for README and small displays
  - Minimal, professional design with slate color palette

- **Build System**
  - Added `make build-windows` target (amd64 + arm64)
  - Added `make build-linux` and `make build-darwin` targets
  - Updated `make build-all` to use platform-specific targets

- **Documentation**
  - Added `docs/en/skills.md` for Skills system
  - Updated installation instructions in README and getting-started guides

### 🐛 Bug Fixes

- Moved assets to `docs/assets/` for proper GitHub Pages deployment

---

**Full Changelog**: https://github.com/fuckvibecoding/vibecoding/compare/v0.0.1...v0.0.4
