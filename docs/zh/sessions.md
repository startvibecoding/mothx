# 会话管理

VibeCoding 使用 SQLite 存储会话数据，支持树状结构、分支、压缩、标签和快速会话查找。

## 会话存储

### 存储架构

VibeCoding 的会话设计在不同运行模式下有所区别：

1. **CLI / TUI / Gateway 模式（单一数据库 + 虚拟句柄）**
   所有会话元数据（如会话列表、会话ID、CWD、时间戳）和所有历史消息/条目均统一保存在单个 SQLite 数据库文件 `sessions.db` 中。
   在该模式下，**不需要且不会在磁盘上生成工作目录子目录或物理会话文件**。CLI/TUI 中展示的 `.db` 路径（如 `~/.vibecoding/sessions/20260625-120000_abcd1234.db`）是根据数据库元数据动态计算出的**虚拟路径**（句柄），无需真实存在即可被程序识别、定位和删除。

2. **Hermes 模式（单一数据库 + 物理句柄）**
   Hermes 作为无人值守的聊天机器人网关，在单一 `sessions.db` 的基础上，还会为每个用户在磁盘上创建编码后的工作目录物理子目录，并在其中写入包含对应会话 ID 的真实物理句柄文件（如 `20260625-120000_abcd1234.db`），以便进行特定平台的会话关联与生命周期管理。

### 存储位置布局（以全局 session 目录为例）

```text
~/.vibecoding/sessions/
├── sessions.db                         # 统一存储所有会话和消息条目的唯一 SQLite 数据库
└── hermes/                             # (仅在 Hermes 模式下存在)
    └── wechat/user_123/active.db       # Hermes 平台特定的物理会话句柄
```

### 路径编码

在需要编码工作目录以进行隔离的场景下（例如 Hermes），工作目录路径使用 URL 安全的 base64 编码，避免冲突和文件系统兼容性问题。

示例：
- `/home/user/project` → `--L2hvbWUvdXNlci9wcm9qZWN0--`
- `/home/user/my.app` → 另一个互不冲突的编码目录名

## SQLite 结构

会话状态存储在 `sessions.db` 的两张核心表中：

| 表 | 用途 |
|----|------|
| `sessions` | 会话元数据：ID、工作目录、时间戳、父会话、版本 |
| `entries` | 有序事件日志：消息、模型切换、压缩、标签和元数据条目 |

条目会保留稳定的 ID 和父 ID，因此对话可以按树状结构重放并安全压缩。

### 条目类型

| 类型 | 描述 |
|------|------|
| `session` | 会话元数据 |
| `message` | 用户/助手/工具消息 |
| `model_change` | 模型切换记录 |
| `thinking_level_change` | 思考等级切换记录 |
| `compaction` | 上下文压缩检查点 |
| `session_info` | 会话名等展示元数据 |
| `branch_summary` | 分支切换摘要 |
| `label` | 用户自定义标签 |

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
vibecoding --continue
vibecoding -c
```

```go
sess, err := session.ContinueRecent(cwd, sessionDir)
```

### 恢复特定会话

```bash
# 通过 session ID 或唯一前缀
vibecoding --resume abcd1234

# 通过会话句柄路径
vibecoding --resume ~/.vibecoding/sessions/--encoded-working-directory--/20260625-120000_abcd1234.db
```

```go
sess, err := session.OpenByPathOrID(cwd, sessionDir, "abcd1234")
```

### 添加消息

```go
_, err := sess.AppendMessage(provider.NewUserMessage("Hello"))
```

## 树状结构

会话条目形成父子链接的树：

```text
session-abcd1234
├── msg-001 (user: "Hello")
│   └── msg-002 (assistant: "Hi!")
│       └── msg-003 (user: "Tell me more")
└── msg-004 (user: "Different question")  # 分支点
    └── msg-005 (assistant: "...")
```

这支持探索不同方向、回到之前的节点重新开始，并保留多个解决方案。

## 会话压缩

VibeCoding 会把压缩检查点记录到 SQLite。压缩后，重放状态会保留每条消息的 Entry ID 和压缩边界（`firstKeptEntryID`）。当会话被重新加载时：

- 消息会被裁剪到正确的压缩边界
- 摘要消息会自动前置
- 后续消息保持原始 Entry ID

压缩配置示例：

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000
  }
}
```

## 最佳实践

### 定期清理

删除单个会话时，优先使用内置 `/sessions delete <id>` 命令，因为它会同时删除句柄文件和 SQLite 记录：

```bash
/sessions list
/sessions delete abcd1234
```

如果手动清理，只删除编码工作目录子目录下按时间命名的单会话句柄文件。不要删除根目录的 `sessions.db`，除非你确实要移除所有持久化会话数据。

```bash
# 示例：先查看旧句柄文件，再谨慎删除
find ~/.vibecoding/sessions -path '*/--*--/*.db' -mtime +30 -print
```

### 备份重要会话

备份 session 根目录，尤其是 `sessions.db` 和编码后的工作目录句柄目录：

```bash
cp -a ~/.vibecoding/sessions ~/backups/
```

## 故障排除

### 会话数据库错误

```text
Error: session "..." not registered in DB
```

可能原因：
- 会话句柄文件存在，但 SQLite 记录已被删除
- `sessions.db` 被删除，或从较旧备份恢复
- session 根目录只被部分复制

解决方案：
1. 从备份恢复完整的 session 根目录
2. 使用 `/sessions list` 显示的有效 session ID 恢复
3. 如果 SQLite 记录已不存在，删除失效的句柄文件

### 会话丢失

可能原因：
- 工作目录发生变化
- 会话句柄文件或数据库被删除
- 编码后的工作目录不同

解决方案：
1. 检查 `~/.vibecoding/sessions/` 或 `%APPDATA%\vibecoding\sessions\`
2. 在原工作目录下使用 `--resume <session-id>`
3. 确认 `sessionDir` 配置正确
