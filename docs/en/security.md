# Security & Sandbox

VibeCoding implements Linux process-level sandbox isolation through bubblewrap (bwrap) to protect system security.

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
vibecoding --mode plan
vibecoding -M plan
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
vibecoding --mode agent --sandbox
vibecoding -M agent --sandbox
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
vibecoding --mode yolo
vibecoding -M yolo
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

## Enabling Sandbox

### Command Line

```bash
# Enable sandbox
vibecoding --sandbox

# Disable sandbox
vibecoding --no-sandbox

# Combine with mode
vibecoding --sandbox --mode plan
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
alias vibecoding='vibecoding --sandbox'
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
vibecoding --sandbox
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
timeout 300 vibecoding --sandbox

# 4. Check symbolic links
find . -type l -ls
```

## Audit and Logging

### Enable Debug Logging

```bash
vibecoding --debug
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
vibecoding --debug 2>vibecoding.log

# Analyze logs
grep "Executing:" vibecoding.log
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
vibecoding --no-sandbox
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