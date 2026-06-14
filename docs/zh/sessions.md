# 会话管理

VibeCoding 使用 JSONL 格式存储会话，支持树状结构和分支。

## 会话存储

### 存储位置

```
Linux/macOS: ~/.vibecoding/sessions/
Windows:     %APPDATA%\vibecoding\sessions\

└── --home-user-projects-myapp--/    # 编码后的工作目录路径
    ├── session-abc123.jsonl
    ├── session-def456.jsonl
    └── ...
```

### 路径编码

工作目录路径会被编码为文件系统安全的格式:
- `/` → `-`
- `.` → 被移除或替换

示例:
- `/home/user/project` → `--home-user-project--`
- `/home/user/my.app` → `--home-user-myapp--`

## 会话文件格式

### JSONL 结构

每行一个 JSON 对象:

```jsonl
{"id":"session-abc123","type":"session","timestamp":"2024-01-01T00:00:00Z","cwd":"/home/user/project","provider":"anthropic","model":"claude-sonnet-4-20250514"}
{"id":"msg-001","parentId":"session-abc123","type":"message","role":"user","content":"Hello"}
{"id":"msg-002","parentId":"msg-001","type":"message","role":"assistant","content":"Hi! How can I help you?"}
{"id":"msg-003","parentId":"msg-002","type":"message","role":"user","content":"Explain this code"}
{"id":"msg-004","parentId":"msg-003","type":"message","role":"assistant","content":"This code does..."}
```

### 字段说明

| 字段 | 类型 | 描述 |
|------|------|------|
| `id` | string | 唯一标识符 |
| `parentId` | string | 父节点 ID (用于树状结构) |
| `type` | string | 条目类型 |
| `timestamp` | string | ISO 8601 时间戳 |
| `role` | string | 消息角色 (user/assistant/system/tool) |
| `content` | string | 消息内容 |

### 条目类型

| 类型 | 描述 |
|------|------|
| `session` | 会话元数据 |
| `message` | 用户/助手消息 |
| `model_change` | 模型切换记录 |
| `compaction` | 上下文压缩记录 |
| `label` | 会话标签 |

## 会话操作

### 创建新会话

```go
sess := session.New(cwd, sessionDir)
if err := sess.Init(); err != nil {
    return err
}
```

### 继续最近会话

```bash
# 命令行
vibecoding --continue
vibecoding -c

# 代码
sess, err := session.ContinueRecent(cwd, sessionDir)
```

### 恢复特定会话

```bash
# 通过 ID
vibecoding --resume session-abc123

# 通过文件路径 (Linux/macOS)
vibecoding --resume ~/.vibecoding/sessions/my-session.jsonl
# Windows: %APPDATA%\vibecoding\sessions\my-session.jsonl

# 代码
sess, err := session.Open(sessionID)
```

### 添加消息

```go
sess.AddMessage(session.Message{
    Role:    "user",
    Content: "Hello",
})
```

## 树状结构

### 会话分支

树状结构允许创建分支:

```
session-abc123
├── msg-001 (user: "Hello")
│   └── msg-002 (assistant: "Hi!")
│       └── msg-003 (user: "Tell me more")
│           └── msg-004 (assistant: "Sure!")
│               └── msg-005 (user: "What about...")
│                   └── msg-006 (assistant: "...")
└── msg-007 (user: "Different question")  # 分支点
    └── msg-008 (assistant: "...")
```

### 分支用途

- 探索不同方向
- 回退到某个点重新开始
- 保留多个解决方案

## 会话压缩

### 自动压缩

当上下文接近限制时自动压缩:

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

### 压缩过程

1. 计算当前 token 使用量
2. 如果超过阈值 (80%)，触发压缩
3. 保留最近的消息
4. 将旧消息压缩为摘要
5. 在会话中标记压缩点

### 重放状态持久化

压缩后，会话会持久化一个**重放状态**，记录每条消息的 Entry ID 和压缩边界（`firstKeptEntryID`）。当会话被重新加载时（例如重启后），重放状态确保：

- 消息被正确裁剪到压缩边界
- 摘要消息被正确前置
- 后续消息保持其原始的 Session Entry ID

这一切都是自动处理的，无需用户配置。

### 手动压缩

```bash
# 交互式
/clear

# 开始新会话
vibecoding
```

## 会话标签

### 添加标签

```bash
# 交互式
/label "Refactoring session"
```

### 标签用途

- 标记重要会话
- 快速识别会话内容
- 组织会话

## 最佳实践

### 1. 定期清理

```bash
# 删除 30 天前的会话
# Linux/macOS:
find ~/.vibecoding/sessions -mtime +30 -delete
# Windows (PowerShell):
Get-ChildItem "$env:APPDATA\vibecoding\sessions" -Recurse | Where-Object { $_.LastWriteTime -lt (Get-Date).AddDays(-30) } | Remove-Item
```

### 2. 使用标签

标记重要会话以便后续查找。

### 3. 配置压缩

根据需要调整压缩参数:

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

### 4. 备份重要会话

```bash
# Linux/macOS:
cp ~/.vibecoding/sessions/important.jsonl ~/backups/
# Windows:
Copy-Item "$env:APPDATA\vibecoding\sessions\important.jsonl" "$env:USERPROFILE\backups\"
```

## 故障排除

### 会话文件损坏

```
错误: invalid JSON in session file
```

**解决方案:**

1. 检查 JSONL 文件格式
2. 手动修复或删除损坏的行
3. 使用备份恢复

### 会话丢失

**可能原因:**
- 工作目录变更
- 会话文件被删除
- 路径编码问题

**解决方案:**

1. 检查会话目录:
   - Linux/macOS: `~/.vibecoding/sessions/`
   - Windows: `%APPDATA%\vibecoding\sessions\`
2. 使用 `--resume` 指定会话 ID
3. 确认工作目录正确

### 压缩问题

**可能原因:**
- Token 估算不准确
- 压缩阈值设置不当

**解决方案:**

1. 调整 `maxContextTokens`
2. 调整 `compaction.reserveTokens`
3. 使用 `/clear` 手动清空

## API 参考

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
