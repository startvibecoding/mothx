# Session Management

VibeCoding uses JSONL format for session storage, supporting tree structure and branching.

## Session Storage

### Storage Location

```
~/.vibecoding/sessions/
└── --home-user-projects-myapp--/    # Encoded working directory path
    ├── session-abc123.jsonl
    ├── session-def456.jsonl
    └── ...
```

### Path Encoding

Working directory paths are encoded to filesystem-safe format:
- `/` → `-`
- `.` → removed or replaced

Examples:
- `/home/user/project` → `--home-user-project--`
- `/home/user/my.app` → `--home-user-myapp--`

## Session File Format

### JSONL Structure

One JSON object per line:

```jsonl
{"id":"session-abc123","type":"session","timestamp":"2024-01-01T00:00:00Z","cwd":"/home/user/project","provider":"anthropic","model":"claude-sonnet-4-20250514"}
{"id":"msg-001","parentId":"session-abc123","type":"message","role":"user","content":"Hello"}
{"id":"msg-002","parentId":"msg-001","type":"message","role":"assistant","content":"Hi! How can I help you?"}
{"id":"msg-003","parentId":"msg-002","type":"message","role":"user","content":"Explain this code"}
{"id":"msg-004","parentId":"msg-003","type":"message","role":"assistant","content":"This code does..."}
```

### Field Description

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `parentId` | string | Parent node ID (for tree structure) |
| `type` | string | Entry type |
| `timestamp` | string | ISO 8601 timestamp |
| `role` | string | Message role (user/assistant/system/tool) |
| `content` | string | Message content |

### Entry Types

| Type | Description |
|------|-------------|
| `session` | Session metadata |
| `message` | User/assistant messages |
| `model_change` | Model change record |
| `compaction` | Context compression record |
| `label` | Session label |

## Session Operations

### Create New Session

```go
sess := session.New(cwd, sessionDir)
if err := sess.Init(); err != nil {
    return err
}
```

### Continue Recent Session

```bash
# Command line
vibecoding --continue
vibecoding -c

# Code
sess, err := session.ContinueRecent(cwd, sessionDir)
```

### Resume Specific Session

```bash
# By ID
vibecoding --resume session-abc123

# By file path
vibecoding --resume ~/.vibecoding/sessions/my-session.jsonl

# Code
sess, err := session.Open(sessionID)
```

### Add Message

```go
sess.AddMessage(session.Message{
    Role:    "user",
    Content: "Hello",
})
```

## Tree Structure

### Session Branching

Tree structure allows creating branches:

```
session-abc123
├── msg-001 (user: "Hello")
│   └── msg-002 (assistant: "Hi!")
│       └── msg-003 (user: "Tell me more")
│           └── msg-004 (assistant: "Sure!")
│               └── msg-005 (user: "What about...")
│                   └── msg-006 (assistant: "...")
└── msg-007 (user: "Different question")  # Branch point
    └── msg-008 (assistant: "...")
```

### Branch Uses

- Explore different directions
- Go back to a point and start over
- Preserve multiple solutions

## Session Compression

### Automatic Compression

Automatically compresses when context approaches limits:

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

### Compression Process

1. Calculate current token usage
2. If exceeds threshold (80%), trigger compression
3. Keep recent messages
4. Compress old messages into summary
5. Mark compression point in session

### Manual Compression

```bash
# Interactive
/clear

# Start new session
vibecoding
```

## Session Labels

### Add Label

```bash
# Interactive
/label "Refactoring session"
```

### Label Uses

- Mark important sessions
- Quickly identify session content
- Organize sessions

## Best Practices

### 1. Regular Cleanup

```bash
# Delete sessions older than 30 days
find ~/.vibecoding/sessions -mtime +30 -delete
```

### 2. Use Labels

Mark important sessions for easy retrieval later.

### 3. Configure Compression

Adjust compression parameters as needed:

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

### 4. Backup Important Sessions

```bash
cp ~/.vibecoding/sessions/important.jsonl ~/backups/
```

## Troubleshooting

### Session File Corruption

```
Error: invalid JSON in session file
```

**Solution:**

1. Check JSONL file format
2. Manually fix or delete corrupted lines
3. Restore from backup

### Session Lost

**Possible Causes:**
- Working directory changed
- Session file deleted
- Path encoding issue

**Solution:**

1. Check `~/.vibecoding/sessions/` directory
2. Use `--resume` to specify session ID
3. Confirm working directory is correct

### Compression Issues

**Possible Causes:**
- Token estimation inaccurate
- Compression threshold set incorrectly

**Solution:**

1. Adjust `maxContextTokens`
2. Adjust `compaction.reserveTokens`
3. Use `/clear` to manually clear

## API Reference

### Session Manager

```go
type Manager struct {
    ID        string
    Path      string
    CWD       string
    Provider  string
    Model     string
    CreatedAt time.Time
}

func New(cwd, sessionDir string) *Manager
func Open(path string) (*Manager, error)
func ContinueRecent(cwd, sessionDir string) (*Manager, error)

func (m *Manager) Init() error
func (m *Manager) AddMessage(msg Message) error
func (m *Manager) Messages() ([]Message, error)
func (m *Manager) Close() error
```

### Message

```go
type Message struct {
    ID        string
    ParentID  string
    Role      string    // "user", "assistant", "system", "tool"
    Content   string
    Timestamp time.Time
}
```