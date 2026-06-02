# 安全与沙箱

VibeCoding 通过 bubblewrap (bwrap) 实现 Linux 进程级沙箱隔离，保护系统安全。

## 沙箱概述

### 什么是沙箱

沙箱是一种安全机制，限制程序的文件系统和网络访问权限。

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

## 沙箱级别

| 级别 | 模式 | 文件系统 | 网络 | 用途 |
|------|------|----------|------|------|
| **none** | YOLO | 完全访问 | 允许 | 系统管理、网络操作 |
| **standard** | Agent | 项目读写，系统只读 | 禁止 | 日常开发 |
| **strict** | Plan | 项目只读，系统只读 | 禁止 | 代码审查、分析 |

### 详细对比

| 功能 | none | standard | strict |
|------|------|----------|--------|
| 读取项目文件 | ✓ | ✓ | ✓ |
| 写入项目文件 | ✓ | ✓ | ✗ |
| 读取系统文件 | ✓ | ✓ | ✓ |
| 写入系统文件 | ✓ | ✗ | ✗ |
| 网络访问 | ✓ | ✗ | ✗ |
| 安装软件包 | ✓ | ✗ | ✗ |
| 修改配置 | ✓ | ✗ | ✗ |

## 模式与沙箱

### Plan 模式 (严格沙箱)

**用途:** 只读分析和规划

```bash
vibecoding --mode plan
vibecoding -M plan
```

**特性:**
- 项目目录: 只读
- 系统目录: 只读
- 网络: 禁止

**适用场景:**
- 代码审查
- 项目分析
- 学习代码库
- 制定开发计划

### Agent 模式 (标准沙箱)

**用途:** 标准开发任务

```bash
vibecoding --mode agent --sandbox
vibecoding -M agent --sandbox
```

**特性:**
- 项目目录: 读写
- 系统目录: 只读
- 网络: 禁止

**适用场景:**
- 编写代码
- 修改文件
- 运行测试
- 构建项目

### YOLO 模式 (无沙箱)

**用途:** 完全访问

```bash
vibecoding --mode yolo
vibecoding -M yolo
```

**特性:**
- 项目目录: 完全访问
- 系统目录: 完全访问
- 网络: 允许

**适用场景:**
- 系统管理
- 安装软件包
- 网络请求
- 跨项目操作

**风险:**
- 可能修改系统文件
- 可能执行危险命令
- 可能泄露敏感信息

## 网络服务加固

Gateway、Hermes 和 A2A 都可能暴露 HTTP/WebSocket 入口。当工具运行在 `agent` 或 `yolo` 模式时，应将这些服务视为远程代码执行入口来保护。

- **Gateway**：对 loopback 以外地址暴露前应启用 `auth.enabled`；当 Gateway 在非 loopback 地址、`yolo` 模式且未认证时，启动会输出警告。
- **A2A**：独立 A2A 默认绑定 `127.0.0.1`。只有明确需要对外暴露时才使用 `--host 0.0.0.0`，并配置 auth token。
- **Hermes WebSocket**：WebSocket 握手时使用 `Authorization: Bearer <token>` 发送 token。Query-string token 仅作为兼容方式保留。
- **工作目录**：使用 `allowedWorkDirs` / `allowed_work_dirs` 限制请求级或平台级工作目录。

## 可信配置中的 Shell 命令

Provider API key 支持通过 `apiKey: "!command"` 从 shell 命令读取，但默认关闭。仅在可信本地配置中启用：

```bash
export VIBECODING_ALLOW_SHELL_CONFIG=1
```

共享配置更推荐使用 `${DEEPSEEK_API_KEY}` 这样的环境变量引用。

## 启用沙箱

### 命令行方式

```bash
# 启用沙箱
vibecoding --sandbox

# 禁用沙箱
vibecoding --no-sandbox

# 结合模式使用
vibecoding --sandbox --mode plan
```

### 配置文件方式

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "allowNetwork": false
  }
}
```

### 环境变量

```bash
export VIBECODING_SANDBOX=true
```

## bubblewrap (bwrap)

### 什么是 bwrap

bubblewrap 是一个轻量级的 Linux 沙箱工具，使用 Linux namespaces 实现进程隔离。

### 安装 bwrap

```bash
# Debian/Ubuntu
sudo apt install bubblewrap

# Fedora/RHEL
sudo dnf install bubblewrap

# Arch Linux
sudo pacman -S bubblewrap

# 验证安装
bwrap --version
```

### bwrap 工作原理

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

### bwrap 命令示例

```bash
# 标准沙箱
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

# 严格沙箱
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

## 沙箱配置详解

### SandboxManager

```go
type Manager struct {
    level       Level
    workdir     string
    allowNetwork bool
}

type Level int

const (
    LevelNone     Level = iota  // 无限制
    LevelStandard               // 标准沙箱
    LevelStrict                 // 严格沙箱
)
```

### 配置选项

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

| 字段 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `enabled` | bool | false | 启用沙箱 |
| `level` | string | "standard" | 沙箱级别 |
| `allowNetwork` | bool | false | 允许网络 |
| `allowDevices` | bool | false | 允许设备访问 |
| `extraBinds` | []string | [] | 额外绑定挂载 |

## 安全最佳实践

### 1. 默认启用沙箱

```bash
# 在 shell 配置中设置别名
alias vibecoding='vibecoding --sandbox'
```

### 2. 根据任务选择模式

| 任务 | 推荐模式 | 原因 |
|------|----------|------|
| 阅读代码 | Plan | 只读，最安全 |
| 修改代码 | Agent | 读写项目，无网络 |
| 运行测试 | Agent | 需要执行权限 |
| 安装依赖 | YOLO | 需要网络 |
| 系统管理 | YOLO | 需要完整权限 |

### 3. 审查高风险操作

对于以下操作，建议人工审查:

- 删除文件
- 修改系统配置
- 执行未知脚本
- 访问敏感数据

### 4. 使用版本控制

```bash
# 定期提交
git add .
git commit -m "save progress"

# 使用分支
git checkout -b feature/new-feature
```

### 5. 限制项目目录

```bash
# 在项目根目录运行
cd /home/user/project
vibecoding --sandbox
```

## 沙箱限制

### 不支持的操作

| 操作 | none | standard | strict |
|------|------|----------|--------|
| 安装系统软件 | ✓ | ✗ | ✗ |
| 访问网络 | ✓ | ✗ | ✗ |
| 修改系统文件 | ✓ | ✗ | ✗ |
| 访问其他项目 | ✓ | ✗ | ✗ |
| Docker 操作 | ✓ | ✗ | ✗ |

### 性能影响

- 启动时间: 增加约 10-50ms
- 内存开销: 可忽略不计
- I/O 性能: 基本无影响

### 平台支持

| 平台 | 支持 |
|------|------|
| Linux | ✓ 完全支持 |
| macOS | ✗ 不支持 |
| Windows | ✗ 不支持 (WSL 可用) |

## 安全风险

### 即使有沙箱仍存在的风险

1. **项目内文件损坏**: Agent 模式下可以修改项目文件
2. **符号链接攻击**: 项目中的符号链接可能指向外部
3. **资源耗尽**: 可能消耗过多 CPU/内存
4. **信息泄露**: 项目文件可能包含敏感信息

### 缓解措施

```bash
# 1. 使用版本控制
git init
git add .
git commit -m "initial"

# 2. 定期备份
cp -r project project.backup

# 3. 限制资源
timeout 300 vibecoding --sandbox

# 4. 检查符号链接
find . -type l -ls
```

## 审批机制

VibeCoding v0.0.4 引入了 Agent 模式审批机制，进一步增强安全性。

### 工作原理

在 Agent 模式下，执行 bash 命令需要用户审批:

1. **白名单检查**: 命令前缀匹配 `bashWhitelist` 中的项 → 自动批准
2. **黑名单检查**: 命令前缀匹配 `bashBlacklist` 中的项 → 始终需要审批
3. **默认行为**: 非白名单命令 → 需要用户输入 y/n 审批

### 配置示例

```json
{
  "approval": {
    "bashWhitelist": ["go ", "make ", "git ", "npm ", "yarn "],
    "bashBlacklist": ["rm -rf", "sudo"]
  }
}
```

### 默认白名单

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

### 审批流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    Approval Flow                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent 请求执行 bash 命令                                    │
│  │                                                           │
│  ▼                                                           │
│  检查模式                                                    │
│  ├─ Plan 模式 → 拒绝 (只读)                                  │
│  ├─ Agent 模式 → 继续检查                                    │
│  └─ YOLO 模式 → 自动批准                                     │
│                                                              │
│  Agent 模式下:                                               │
│  ├─ 非 bash 工具 → 自动批准                                  │
│  ├─ 命令匹配白名单 → 自动批准                                │
│  └─ 其他 → 需要用户审批                                      │
│                                                              │
│  用户审批:                                                   │
│  ├─ 输入 y/yes → 执行命令                                    │
│  └─ 输入 n/no → 拒绝执行                                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 安全建议

1. **保持默认白名单**: 仅允许常见的安全命令
2. **添加黑名单**: 对于危险命令如 `rm -rf`、`sudo` 等添加到黑名单
3. **定期审查**: 检查审批日志，了解 Agent 执行的命令
4. **结合沙箱**: 使用 `--sandbox` 限制文件系统访问

## 审计和日志

### 启用调试日志

```bash
vibecoding --debug
```

### 查看执行的命令

调试模式下，所有执行的命令都会输出到 stderr:

```
[DEBUG] Executing: go build ./...
[DEBUG] Sandbox: bwrap --ro-bind /usr /usr ...
[DEBUG] Exit code: 0
```

### 日志记录

```bash
# 保存日志
vibecoding --debug 2>vibecoding.log

# 分析日志
grep "Executing:" vibecoding.log
```

## 故障排除

### bwrap 未安装

```
错误: bwrap: command not found
```

**解决方案:**

```bash
# 安装 bwrap
sudo apt install bubblewrap

# 或禁用沙箱
vibecoding --no-sandbox
```

### 权限问题

```
错误: bwrap: Permission denied
```

**解决方案:**

1. 检查 bwrap 是否正确安装
2. 确认用户有权限执行 bwrap
3. 检查文件系统权限

### 网络访问失败

```
错误: network is unreachable
```

**解决方案:**

1. 使用 YOLO 模式
2. 或配置 `allowNetwork: true`

### 文件访问失败

```
错误: Read-only file system
```

**解决方案:**

1. 使用 Agent 模式 (standard)
2. 或使用 YOLO 模式 (none)
3. 检查文件权限

## 参考资料

- [bubblewrap GitHub](https://github.com/containers/bubblewrap)
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [seccomp](https://man7.org/linux/man-pages/man2/seccomp.2.html)
- [Security Best Practices](https://owasp.org/www-project-developer-guide/)
