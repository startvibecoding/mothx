# Security & Sandbox

MothX implements Linux process-level sandbox isolation through bubblewrap (bwrap) to protect system security.

## Sandbox Overview

### What is a Sandbox

A sandbox is a security mechanism that restricts a program's file system and network access permissions.

```
┌─────────────────────────────────────────────────────────────┐
│                     Sandbox Overview                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                  Sandboxed Process                      ││
│  │                                                          ││
│  │   Project Directory                                      ││
│  │   ┌─────────────────────────────────────────────────┐   ││
│  │   │  RW/RO access                                   │   ││
│  │   └─────────────────────────────────────────────────┘   ││
│  │                                                          ││
│  │   System Directories                                     ││
│  │   ┌─────────────────────────────────────────────────┐   ││
│  │   │  RO access only                                 │   ││
│  │   └─────────────────────────────────────────────────┘   ││
│  │                                                          ││
│  │   Network                                                ││
│  │   ┌─────────────────────────────────────────────────┐   ││
│  │   │  Blocked (standard/strict)                      │   ││
│  │   └─────────────────────────────────────────────────┘   ││
│  │                                                          ││
│  └─────────────────────────────────────────────────────────┘│
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Sandbox Levels

| Level | Mode | File System | Network | Use Case |
|-------|------|------------|---------|----------|
| **none** | YOLO | Full access | Allowed | System administration, network operations |
| **standard** | Agent | Project R/W, System R/O | Disabled | Daily development |
| **strict** | Plan | Project R/O, System R/O | Disabled | Code review, analysis |

### Detailed Comparison

| Feature | none | standard | strict |
|---------|------|----------|--------|
| Read project files | ✓ | ✓ | ✓ |
| Write project files | ✓ | ✓ | ✗ |
| Read system files | ✓ | ✓ | ✓ |
| Write system files | ✓ | ✗ | ✗ |
| Network access | ✓ | ✗ | ✗ |
| Install packages | ✓ | ✗ | ✗ |
| Modify configuration | ✓ | ✗ | ✗ |

## Modes and Sandboxes

### Plan Mode (Strict Sandbox)

**Purpose:** Read-only analysis and planning

```bash
mothx --mode plan
mothx -M plan
```

**Features:**
- Project directory: Read-only
- System directory: Read-only
- Network: Disabled

**Use Cases:**
- Code review
- Project analysis
- Learning codebase
- Creating development plans

### Agent Mode (Standard Sandbox)

**Purpose:** Standard development tasks

```bash
mothx --mode agent --sandbox
mothx -M agent --sandbox
```

**Features:**
- Project directory: Read/write
- System directory: Read-only
- Network: Disabled

**Use Cases:**
- Writing code
- Modifying files
- Running tests
- Building projects

### YOLO Mode (No Sandbox)

**Purpose:** Full access

```bash
mothx --mode yolo
mothx -M yolo
```

**Features:**
- Project directory: Full access
- System directory: Full access
- Network: Allowed

**Use Cases:**
- System administration
- Installing packages
- Network requests
- Cross-project operations

**Risks:**
- May modify system files
- May execute dangerous commands
- May expose sensitive information

## Network Service Hardening

Gateway, Hermes, and A2A can expose HTTP/WebSocket entry points. Treat these services as remote code-execution surfaces whenever tools can run in `agent` or `yolo` mode.

- **Gateway**: enable `auth.enabled` before exposing beyond loopback; startup warns when Gateway listens beyond loopback in `yolo` mode without authentication.
- **A2A**: standalone A2A binds to `127.0.0.1` by default. Use `--host 0.0.0.0` only for intentional exposure, and configure an auth token.
- **Hermes WebSocket**: send tokens with `Authorization: Bearer <token>` during the WebSocket handshake. Query-string tokens are accepted only for compatibility.
- **Working directories**: use `allowedWorkDirs` / `allowed_work_dirs` to restrict per-request or per-platform working directories.

## Trusted Config Shell Commands

Provider API keys can be loaded from shell commands with `apiKey: "!command"`, but this is disabled by default. Enable it only for trusted local config:

```bash
export VIBECODING_ALLOW_SHELL_CONFIG=1
```

Prefer environment-variable references such as `${DEEPSEEK_API_KEY}` for shared configs.

## Enabling Sandbox

### Command Line

```bash
# Enable sandbox
mothx --sandbox

# Disable sandbox
mothx --no-sandbox

# Combine with mode
mothx --sandbox --mode plan
```

### Configuration File

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "allowNetwork": false
  }
}
```

### Environment Variable

```bash
export VIBECODING_SANDBOX=true
```

## bubblewrap (bwrap)

### What is bwrap

bubblewrap is a lightweight Linux sandbox tool that uses Linux namespaces for process isolation.

### Installing bwrap

```bash
# Debian/Ubuntu
sudo apt install bubblewrap

# Fedora/RHEL
sudo dnf install bubblewrap

# Arch Linux
sudo pacman -S bubblewrap

# Verify installation
bwrap --version
```

### How bwrap Works

```
┌─────────────────────────────────────────────────────────────┐
│                    bwrap Namespaces                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Mount Namespace                       ││
│  │  /project    → RW/RO (depending on level)              ││
│  │  /usr        → RO                                       ││
│  │  /lib        → RO                                       ││
│  │  /etc        → RO                                       ││
│  └─────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Network Namespace                     ││
│  │  No network access (standard/strict)                    ││
│  └─────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    PID Namespace                         ││
│  │  Process isolation                                      ││
│  └─────────────────────────────────────────────────────────┘│
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### bwrap Command Examples

```bash
# Standard sandbox
bwrap \
  --ro-bind /usr /usr \
  --ro-bind /lib /lib \
  --ro-bind /lib64 /lib64 \
  --ro-bind /bin /bin \
  --ro-bind /etc /etc \
  --bind /home/user/project /home/user/project \
  --tmpdir /tmp \
  --proc /proc \
  --dev /dev \
  --chdir /home/user/project \
  --unshare-net \
  --unshare-pid \
  --die-with-parent \
  -- sh -c "go build ./..."

# Strict sandbox
bwrap \
  --ro-bind /usr /usr \
  --ro-bind /lib /lib \
  --ro-bind /home/user/project /home/user/project \
  --tmpdir /tmp \
  --proc /proc \
  --dev /dev \
  --chdir /home/user/project \
  --unshare-net \
  --unshare-pid \
  --die-with-parent \
  -- sh -c "go vet ./..."
```

## Sandbox Configuration Details

### SandboxManager

```go
type Manager struct {
    level       Level
    workdir     string
    allowNetwork bool
}

type Level int

const (
    LevelNone     Level = iota  // Unrestricted
    LevelStandard               // Standard sandbox
    LevelStrict                 // Strict sandbox
)
```

### Configuration Options

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "allowNetwork": false,
    "allowDevices": false,
    "extraBinds": [
      "/path/to/extra:/mnt/extra"
    ]
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable sandbox |
| `level` | string | "standard" | Sandbox level |
| `allowNetwork` | bool | false | Allow network |
| `allowDevices` | bool | false | Allow device access |
| `extraBinds` | []string | [] | Extra bind mounts |

## Security Best Practices

### 1. Enable Sandbox by Default

```bash
# Set alias in shell configuration
alias mothx='mothx --sandbox'
```

### 2. Choose Mode Based on Task

| Task | Recommended Mode | Reason |
|------|------------------|--------|
| Reading code | Plan | Read-only, safest |
| Modifying code | Agent | Project R/W, no network |
| Running tests | Agent | Needs execution permission |
| Installing dependencies | YOLO | Needs network |
| System administration | YOLO | Needs full permissions |

### 3. Review High-Risk Operations

The following operations should be reviewed by humans:

- Deleting files
- Modifying system configuration
- Executing unknown scripts
- Accessing sensitive data

### 4. Use Version Control

```bash
# Commit regularly
git add .
git commit -m "save progress"

# Use branches
git checkout -b feature/new-feature
```

### 5. Limit Project Directory

```bash
# Run from project root directory
cd /home/user/project
mothx --sandbox
```

## Sandbox Limitations

### Unsupported Operations

| Operation | none | standard | strict |
|-----------|------|----------|--------|
| Install system software | ✓ | ✗ | ✗ |
| Access network | ✓ | ✗ | ✗ |
| Modify system files | ✓ | ✗ | ✗ |
| Access other projects | ✓ | ✗ | ✗ |
| Docker operations | ✓ | ✗ | ✗ |

### Performance Impact

- Startup time: ~10-50ms increase
- Memory overhead: Negligible
- I/O performance: Minimal impact

### Platform Support

| Platform | Support |
|----------|---------|
| Linux | ✓ Full support |
| macOS | ✗ Not supported |
| Windows | ✗ Not supported (WSL usable) |

## Security Risks

### Risks That Still Exist Even With Sandbox

1. **Project file corruption**: Agent mode can modify project files
2. **Symbolic link attacks**: Symbolic links in project may point outside
3. **Resource exhaustion**: May consume excessive CPU/memory
4. **Information leakage**: Project files may contain sensitive information

### Mitigation Measures

```bash
# 1. Use version control
git init
git add .
git commit -m "initial"

# 2. Regular backups
cp -r project project.backup

# 3. Limit resources
timeout 300 mothx --sandbox

# 4. Check symbolic links
find . -type l -ls
```

## Approval Mechanism

MothX uses a multi-layer approval system for Agent mode.

### How It Works

In Agent mode, executing bash commands goes through the following checks:

1. **Blacklist check (highest priority)**: Command prefix matches `bashBlacklist` → Always require approval, even if project allow rules match.
2. **Project allow rules**: Command matches `bashCommands` (exact) or `bashPrefixes` (prefix) in `.vibe/allow.json` → Auto-approve.
3. **Settings whitelist**: Command prefix matches `bashWhitelist` in `settings.json` → Auto-approve.
4. **Default behavior**: Non-whitelisted commands → Require user approval via the interactive dialog.

### Interactive Approval Dialog

When a command requires approval, the TUI displays a dialog with:

- **Approve Once** — Run the command this time only.
- **Deny** — Reject the command.
- **Always Allow Exact Command** — Persist the exact command to `.vibe/allow.json` and approve.
- **Always Allow Command Prefix** — Persist the command prefix (e.g. `go test `) to `.vibe/allow.json` and approve.

Navigation: ↑/↓ to move, Enter to select, y to approve, n to deny, Esc to abort.

### Project-Level Allow Rules (`allow.json`)

Project allow rules are stored in `.vibe/allow.json` and support:

- `bashCommands`: Exact command strings that auto-approve.
- `bashPrefixes`: Command prefixes that auto-approve. Trailing spaces are significant.
- `autoEdit`: Auto-approve write/edit tools (defaults to `true` when no file exists).
- `editPaths`: Glob patterns for auto-approve paths.

```json
{
  "bashCommands": ["make test"],
  "bashPrefixes": ["go test ", "go build "]
}
```

### Configuration Example

```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm ", "yarn "],
    "bashBlacklist": ["rm -rf", "sudo"]
  }
}
```

### Default Whitelist

```json
[
  "go ",
  "make ",
  "git ",
  "npm ",
  "yarn ",
  "node ",
  "python ",
  "pip "
]
```

### Approval Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Approval Flow                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent requests bash command execution                       │
│  │                                                           │
│  ▼                                                           │
│  Check mode                                                  │
│  ├─ Plan mode → Deny (read-only)                             │
│  ├─ Agent mode → Continue checking                           │
│  └─ YOLO mode → Auto-approve unless blacklisted              │
│                                                              │
│  Blacklist check (highest priority):                         │
│  ├─ Matches blacklist → Require approval                     │
│  └─ Otherwise continue                                       │
│                                                              │
│  In Agent mode:                                              │
│  ├─ Project allow.json matches → Auto-approve                │
│  ├─ Matches settings whitelist → Auto-approve                │
│  └─ Otherwise → Interactive approval dialog                  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Security Recommendations

1. **Keep default whitelist**: Only allow common safe commands
2. **Add blacklist**: Add dangerous commands like `rm -rf`, `sudo` to blacklist
3. **Use project allow rules**: Prefer `.vibe/allow.json` over global whitelist for project-specific commands
4. **Regular review**: Check approval logs to understand Agent-executed commands
5. **Combine with sandbox**: Use `--sandbox` to limit file system access

## Audit and Logging

### Enable Debug Logging

```bash
mothx --debug
```

### View Executed Commands

In debug mode, all executed commands are output to stderr:

```
[DEBUG] Executing: go build ./...
[DEBUG] Sandbox: bwrap --ro-bind /usr /usr ...
[DEBUG] Exit code: 0
```

### Log Recording

```bash
# Save logs
mothx --debug 2>mothx.log

# Analyze logs
grep "Executing:" mothx.log
```

## Troubleshooting

### bwrap Not Installed

```
Error: bwrap: command not found
```

**Solution:**

```bash
# Install bwrap
sudo apt install bubblewrap

# Or disable sandbox
mothx --no-sandbox
```

### Permission Issues

```
Error: bwrap: Permission denied
```

**Solution:**

1. Check if bwrap is correctly installed
2. Confirm user has permission to execute bwrap
3. Check file system permissions

### Network Access Failure

```
Error: network is unreachable
```

**Solution:**

1. Use YOLO mode
2. Or configure `allowNetwork: true`

### File Access Failure

```
Error: Read-only file system
```

**Solution:**

1. Use Agent mode (standard)
2. Or use YOLO mode (none)
3. Check file permissions

## References

- [bubblewrap GitHub](https://github.com/containers/bubblewrap)
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [seccomp](https://man7.org/linux/man-pages/man2/seccomp.2.html)
- [Security Best Practices](https://owasp.org/www-project-developer-guide/)
