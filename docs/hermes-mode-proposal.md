# v0.1.27 Hermes 模式 — 研发计划

> **日期**: 2026-05-28
> **目标版本**: v0.1.27
> **状态**: ✅ 决策已确认，待进入开发

---

## 1. 概述

VibeCoding 当前提供三种运行模式：**CLI (TUI)**、**ACP (编辑器集成)**、**Gateway (HTTP API)**。

本提案引入第四种运行模式 **`hermes`** — 通过 `vibecoding hermes` 子命令启动，提供**消息平台网关 + 自动化调度 + 持久化记忆**等能力，让 VibeCoding 从"编码助手"扩展为"可部署的自主代理"。

### 设计哲学

- **渐进式采纳**：Hermes 模式是对现有 CLI/Gateway 的增强，不是替代
- **复用优先**：尽量复用已有的 agent loop、provider、tools、session、sandbox 基础设施
- **Go 原生**：VibeCoding 是 Go 项目，不移植 Python 生态，只借鉴架构思路
- **缓存友好**：memory 等动态内容通过 tool call 按需加载（同 `skill_ref`），不注入 system prompt，保护 prompt cache 命中率

---

## 2. 已确认的决策

| 决策项 | 结论 | 备注 |
|--------|------|------|
| 消息平台 v0.1.27 | **微信 (iLink) + 飞书** | 微信参考 iLink 协议自行实现；飞书用官方 SDK 长连接 |
| 消息平台 v0.1.28+ | Telegram → Discord | 延后 |
| 企业微信 | **不做** | 用个人微信 iLink 协议 |
| Web 搜索工具 | **不做** | 用户通过第三方 skill 自行扩展 |
| 记忆存储 | **memory.md** | Markdown 文件，人类可读 |
| 记忆注入方式 | **通过 `memory` 工具按需读取**，同 `skill_ref` 模式 | 不注入 system prompt，保护缓存命中 |
| 配置文件 | **hermes.json** — 独立配置文件 | 同 gateway.json 模式 |
| Shell Hooks | **外部脚本** — JSON stdin/stdout 通信 | 语言无关 |
| Checkpoints/Rollback | **不做** — 推迟到后续版本 | 降低 v0.1.27 范围 |
| A2A 协议 | **采纳** — Hermes 作为 A2A Server 暴露 | 官方 Go SDK `github.com/a2aproject/a2a-go/v2` |

---

## 3. 能力清单

### 🟢 v0.1.27 采纳

| # | 能力 | 工作量 | 实现思路 |
|---|------|--------|---------|
| 1 | **微信 Bot (iLink 协议)** | 大 | `internal/messaging/wechat/` — 参考 iLink 协议自行实现，纯标准库零外部依赖，QR 登录 + 长轮询 |
| 2 | **飞书 Bot** | 大 | `internal/messaging/feishu/` — 官方 SDK `github.com/larksuite/oapi-sdk-go/v3`，**长连接**接收事件 |
| 3 | **消息 Session 隔离** | 中 | 每个 user_id 独立 session |
| 4 | **用户白名单** | 小 | `hermes.json` 中 `allowed_users` |
| 5 | **Cron 完善** | 中 | 补齐 CLI 管理命令，关联 hermes 网关 |
| 6 | **持久化记忆 (memory.md)** | 中 | `internal/memory/` — Markdown 文件存储，通过 `memory` 工具按需读写 |
| 7 | **User Profile** | 小 | memory.md 中的 `## User Profile` section |
| 8 | **Budget Pressure** | 小 | 在 tool result 中注入迭代预算警告 |
| 9 | **Context Pressure** | 小 | 接近 compaction 阈值时发出警告 |
| 10 | **Smart Approvals** | 中 | 命令危险性分类 + 审批流 |
| 11 | **Shell Hooks** | 中 | pre/post tool call 外部脚本 |
| 12 | **Webhook 入站** | 中 | HTTP endpoint 接收事件，驱动 agent 任务 |
| 13 | **A2A 协议 (Server)** | 中 | Hermes 作为 A2A Server，其他 Agent 可通过标准协议发送任务 |

### 🟡 延后（v0.1.28+）

| 能力 | 原因 |
|------|------|
| Telegram Bot | 排在微信/飞书之后 |
| Discord Bot | 排在 Telegram 之后 |
| Slack Bot | 排在 Discord 之后 |
| 浏览器自动化 | CDP 复杂度高 |
| Vision (图像分析) | 多模态非核心 |
| Session Search | FTS5 搜索 |
| Kanban 看板 | 大工程 |
| Persistent Goals | 配合 Kanban |
| Git Worktree | 并行代理场景 |
| Checkpoints / Rollback | 已确认推迟 |
| 其他消息平台 | Email, Matrix, Mattermost 等 |

### 🔴 不做

| 能力 | 原因 |
|------|------|
| **Web 搜索** | 用户通过第三方 skill 自行扩展 |
| **企业微信** | 用个人微信 iLink 协议代替 |
| WhatsApp / Signal / SMS | 外部依赖重 |
| Python Plugins | Go 项目 |
| 图片生成 / Voice Mode | 非核心 |
| RL Training / Batch | Python 生态 |

---

## 4. 消息平台技术方案

### 4.1 微信 iLink（优先级 #1）

**实现方式**: 根据 iLink 协议规范自行实现（参考 `/home/free/src/wechatbot/golang` 中的协议实现），**不引入外部依赖**。协议层约 1600 行纯标准库代码，直接写入 `internal/messaging/wechat/`

| 维度 | 方案 |
|------|------|
| **认证** | QR 码扫码登录，凭证持久化到 `~/.config/vibecoding/wechat-credentials.json` |
| **消息接收** | **长轮询** (`getupdates`)，无需公网 IP |
| **消息发送** | `sendmessage` API，支持文本/图片/文件/视频 |
| **Typing 指示** | 支持（`getconfig` → `sendtyping`） |
| **CDN 媒体** | AES-128-ECB 加密上传/下载 |
| **会话恢复** | `context_token` 自动管理；session 过期（errcode -14）自动重新登录 |
| **优势** | 无需公网暴露；个人微信即可；长轮询天然可靠 |

**代码结构**（参考 iLink 协议，VibeCoding 内部包自行实现）：

```
internal/messaging/wechat/
├── wechat.go      # Bot 主体 + 消息处理（实现 messaging.Platform）
├── types.go       # iLink 协议类型定义
├── protocol.go    # iLink HTTP API 调用（getupdates/sendmessage/getconfig 等）
├── auth.go        # QR 码登录 + 凭证持久化
└── crypto.go      # AES-128-ECB CDN 加密/解密
```

全部使用 Go 标准库（`crypto/aes`、`net/http`、`encoding/json`），**零外部依赖**。

**核心 API 端点**（来自 iLink 协议）：

| 端点 | 作用 |
|------|------|
| `GET /ilink/bot/get_bot_qrcode` | 获取 QR 码 |
| `GET /ilink/bot/get_qrcode_status` | 轮询扫码状态 |
| `POST /ilink/bot/getupdates` | 长轮询接收消息 |
| `POST /ilink/bot/sendmessage` | 发送消息 |
| `POST /ilink/bot/getconfig` | 获取 typing ticket |
| `POST /ilink/bot/sendtyping` | 发送/取消打字指示 |

### 4.2 飞书（优先级 #2）

**依赖**: `github.com/larksuite/oapi-sdk-go/v3` — 飞书官方 Go SDK

参考文档: https://open.feishu.cn/document/server-side-sdk/golang-sdk-guide/preparations

| 维度 | 方案 |
|------|------|
| **SDK** | 飞书官方 Go SDK v3 |
| **消息接收** | **长连接** (WebSocket)，无需公网 IP |
| **消息发送** | REST API (飞书 IM 接口) |
| **认证** | App ID + App Secret |
| **消息类型** | 文本、富文本、Markdown、卡片消息 |
| **创建步骤** | 飞书开放平台 → 创建应用 → 开启机器人能力 → 配置事件订阅 |
| **优势** | WebSocket 无需公网；官方 SDK 维护有保障；卡片消息表现力强 |

**飞书长连接模式关键点**：
- 使用 `larkws` 包建立 WebSocket 长连接
- 订阅 `im.message.receive_v1` 事件接收消息
- 无需配置回调 URL，适合内网/开发环境
- 自动断线重连

### 4.3 A2A 协议 (Agent-to-Agent)

**依赖**: `github.com/a2aproject/a2a-go/v2` — Google A2A 官方 Go SDK

**A2A 是什么**：Google 主导的开放协议，让不同框架、不同厂商的 AI Agent 能够互相发现、通信和协作，在不暴露内部状态的前提下完成复杂任务。

VibeCoding Hermes 作为 **A2A Server** 运行，其他 Agent 可通过标准 A2A 协议向 VibeCoding 发送任务。

| 维度 | 方案 |
|------|------|
| **角色** | A2A Server（接收外部 Agent 的任务请求） |
| **传输** | JSON-RPC 2.0 over HTTP（同步 + SSE 流式） |
| **Agent Card** | `/.well-known/agent.json` 发布能力描述 |
| **Task 生命周期** | submitted → working → completed/failed |
| **认证** | Bearer token（复用 Gateway 的认证机制） |
| **流式响应** | SSE 实时推送 Task 状态和 Artifact 更新 |

**与现有协议的关系**：

| 协议 | 角色 | 关系 |
|------|------|------|
| **ACP** (Agent Client Protocol) | 编辑器 ↔ Agent | 已有，用于 IDE 集成 |
| **MCP** (Model Context Protocol) | Agent ↔ 工具服务 | 已有，让 Agent 调用外部工具 |
| **A2A** (Agent-to-Agent) | Agent ↔ Agent | **新增**，Agent 间对等协作 |
| **Gateway** (OpenAI 兼容) | 应用 ↔ LLM API | 已有，应用调 VibeCoding 当 LLM |

**A2A Server 暴露的能力 (Agent Card)**：

```json
{
  "name": "VibeCoding",
  "description": "AI coding assistant with file editing, terminal, and search capabilities",
  "url": "http://localhost:8093/a2a",
  "version": "0.1.27",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false
  },
  "skills": [
    {
      "id": "code-edit",
      "name": "Code Editing",
      "description": "Read, write, and edit code files with precise text replacement"
    },
    {
      "id": "terminal",
      "name": "Terminal Execution",
      "description": "Execute shell commands, run tests, build projects"
    },
    {
      "id": "code-search",
      "name": "Code Search",
      "description": "Search codebases with ripgrep and fd"
    }
  ]
}
```

**实现方式**：外部 Agent 通过 A2A SendMessage 发送任务 → Hermes dispatcher 创建 agent loop 处理 → 通过 SSE 流式返回结果。复用与消息平台相同的 agent 基础设施。

---

## 5. memory.md 设计

### 5.1 核心原则：不破坏缓存命中

**关键设计决策**：memory.md 的内容 **不注入 system prompt**。

原因：system prompt 是 prompt cache 的主要命中区域。如果每次都把变化的 memory 内容注入 system prompt，会导致缓存失效，增加成本和延迟。

**实现方式**：memory 通过 `memory` 工具按需读写，与 `skill_ref` 工具的设计模式一致。Agent 在需要时主动调用 `memory(action="read")` 获取记忆，而不是被动接收注入。

### 5.2 文件位置

`~/.config/vibecoding/memory.md`

### 5.3 格式

```markdown
# Agent Memory

## User Profile

- 用户偏好使用中文交流
- Go 为主要开发语言
- 项目使用 Cobra + Bubble Tea 技术栈
- 编辑器偏好: VSCode + Vim 键位

## Working Memory

- vibecoding 项目版本当前为 v0.1.26，下一个版本 v0.1.27
- 用户对消息平台的优先级：微信 > 飞书 > Telegram > Discord
- settings.json 中 provider 配置不要随意改动 schema

## Lessons Learned

- edit 工具的 oldText 必须在文件中唯一匹配，不要用太大的上下文
- 用户不喜欢过多的确认提示，yolo 模式下直接执行
- 中文文档要和英文文档同步更新
```

### 5.4 memory 工具设计

```
memory(action="read")
  → 返回 memory.md 全文（Agent 按需调用）

memory(action="read", section="User Profile")
  → 返回指定 section 内容

memory(action="add", section="Working Memory", content="新的记忆条目")
  → 在指定 section 末尾追加条目

memory(action="update", section="Working Memory", old="旧内容", new="新内容")
  → 更新指定条目

memory(action="delete", section="Working Memory", content="要删除的条目")
  → 删除指定条目
```

### 5.5 System Prompt 中的提示（轻量级，不含数据）

在 system prompt 的 Guidelines 中添加一行静态提示（不影响缓存）：

```
- A persistent memory file (memory.md) is available via the `memory` tool. Read it at the start of complex tasks to recall user preferences and prior context. Update it when you learn important facts about the user or project.
```

这行提示是**静态**的，不包含 memory.md 的实际内容，所以不影响 prompt cache。

---

## 6. 子命令设计

### 6.1 命令树

```
vibecoding hermes
├── start             # 启动 hermes 守护进程（前台运行）
├── start -d          # 后台启动
├── stop              # 停止守护进程
├── status            # 查看运行状态（各平台连接状态）
│
├── config
│   ├── init          # 创建 hermes.json 配置模板
│   └── show          # 查看当前配置
│
├── wechat
│   ├── login         # 微信扫码登录
│   └── status        # 查看微信连接状态
│
├── feishu
│   ├── setup         # 交互式配置飞书（AppID/AppSecret）
│   └── status        # 查看飞书连接状态
│
├── webhook
│   ├── list          # 列出 webhook 路由
│   ├── add           # 添加路由
│   └── test          # 测试
│
├── cron
│   ├── list          # 列出定时任务
│   ├── add           # 添加
│   ├── edit          # 编辑
│   ├── delete        # 删除
│   ├── enable        # 启用
│   ├── disable       # 禁用
│   └── run           # 立即执行
│
├── memory
│   ├── show          # 查看 memory.md 内容
│   ├── search        # 搜索记忆
│   ├── clear         # 清空
│   └── edit          # 打开编辑器编辑 memory.md
│
└── sessions
    ├── list          # 列出活跃 session
    └── kill          # 终止 session
```

### 6.2 配置文件 `hermes.json`

位置: `~/.config/vibecoding/hermes.json`（全局）或 `.vibe/hermes.json`（项目级覆盖）

```jsonc
{
  // === 微信 (iLink) ===
  
  "wechat": {
    "enabled": true,
    "cred_path": "",       // 空 = 默认 ~/.config/vibecoding/wechat-credentials.json
    "allowed_users": [],   // 空 = 允许所有人（危险！）
    "auto_typing": true    // 自动显示"正在输入"
  },
  
  // === 飞书 ===
  
  "feishu": {
    "enabled": false,
    "app_id": "${FEISHU_APP_ID}",
    "app_secret": "${FEISHU_APP_SECRET}",
    "allowed_users": []
  },
  
  // === Webhook 入站 ===
  
  "webhooks": {
    "enabled": false,
    "port": 8092,
    "secret": "${WEBHOOK_SECRET}",
    "routes": [
      {
        "path": "/github",
        "events": ["push", "pull_request"],
        "skill": "code-review",
        "delivery": "wechat"
      }
    ]
  },
  
  // === A2A Server ===
  
  "a2a": {
    "enabled": false,
    "port": 8093,
    "auth_token": "${A2A_AUTH_TOKEN}"  // 空 = 无认证
  },
  
  // === Cron ===
  
  "cron": {
    "enabled": true
  },
  
  // === 记忆 ===
  
  "memory": {
    "enabled": true,
    "path": ""  // 空 = 默认 ~/.config/vibecoding/memory.md
  },

  // === 安全 ===
  
  "security": {
    "smart_approvals": true,
    "allowed_work_dirs": []
  },
  
  // === Shell Hooks ===
  
  "hooks": {
    "pre_tool_call": "",   // 外部脚本路径
    "post_tool_call": ""
  },
  
  // === Agent ===
  
  "agent": {
    "max_turns": 90,
    "budget_pressure": true,
    "context_pressure": true
  },
  
  // === 工作目录 ===
  
  "work_dir": "."
}
```

---

## 7. 架构设计

### 7.1 新增包结构

```
internal/
├── messaging/                   # 消息平台层（抽象 + 各平台实现）
│   ├── platform.go              # Platform 接口 + InboundMessage 等公共类型
│   ├── wechat/                  # 微信 iLink 适配器（自行实现，零外部依赖）
│   │   ├── wechat.go            # Bot 主体，实现 messaging.Platform
│   │   ├── types.go             # iLink 协议类型定义
│   │   ├── protocol.go          # iLink HTTP API 调用
│   │   ├── auth.go              # QR 登录 + 凭证持久化
│   │   └── crypto.go            # AES-128-ECB CDN 加解密
│   └── feishu/                  # 飞书适配器
│       ├── feishu.go            # 飞书 SDK 封装（长连接），实现 messaging.Platform
│       └── session.go           # per-user Session 管理
│
├── hermes/                      # Hermes 模式编排层
│   ├── server.go                # 守护进程主循环（组装 messaging + webhook + a2a + cron）
│   ├── config.go                # hermes.json 配置加载
│   ├── dispatcher.go            # 消息 → Agent 转发调度器
│   ├── a2a/                     # A2A 协议 Server
│   │   ├── server.go            # A2A JSON-RPC handler（基于 a2a-go SDK）
│   │   ├── agent_card.go        # Agent Card 生成 (/.well-known/agent.json)
│   │   └── executor.go          # AgentExecutor 实现（A2A Task → agent loop）
│   ├── webhook/                 # Webhook 入站
│   │   ├── server.go            # HTTP 服务
│   │   └── router.go            # 路由分发 → Agent
│   └── hooks/                   # Shell Hooks
│       └── hooks.go             # 外部脚本调用
│
├── memory/                      # 持久化记忆
│   ├── store.go                 # memory.md 读写
│   └── tool.go                  # memory 工具定义（同 skill_ref 模式）
│
└── (existing packages unchanged)
```

> **为什么分 `messaging/` 和 `hermes/`？**
> - `internal/messaging/` 是消息平台的**抽象 + 实现**层，纯粹关注"接收消息、发送消息"。每个子包（`wechat/`、`feishu/`、未来的 `telegram/`、`discord/`）是独立适配器，实现同一个 `messaging.Platform` 接口。
> - `internal/hermes/` 是 Hermes 模式的**编排层**，负责把 messaging、webhook、cron、agent loop 组装到一起运行。
> - 这个分层使得消息平台适配器可以被其他模式复用（例如 Gateway 模式未来也可能需要消息推送），且新增平台只需在 `messaging/` 下加子包，无需改动编排层。

### 7.2 消息平台抽象

```go
// internal/messaging/platform.go
package messaging

type Platform interface {
    // Name returns the platform identifier (e.g. "wechat", "feishu").
    Name() string
    // Start begins receiving messages. Blocks until ctx is cancelled or Stop is called.
    Start(ctx context.Context, handler MessageHandler) error
    // Stop gracefully shuts down the platform connection.
    Stop() error
    // SendMessage sends a text message to a specific chat.
    SendMessage(ctx context.Context, chatID string, text string) error
}

// MessageHandler is called for each incoming message. Returns the response text.
type MessageHandler func(ctx context.Context, msg InboundMessage) (string, error)

type InboundMessage struct {
    Platform  string    // "wechat", "feishu", etc.
    ChatID    string    // 会话标识
    UserID    string    // 发送者 ID
    UserName  string    // 发送者名称
    Text      string    // 消息文本
    Timestamp time.Time
}
```

### 7.3 复用关系

```
hermes server (internal/hermes/)
  │
  ├─ 完全复用 ──────────────────────────────
  │   ├── agent.Agent          (agent loop)
  │   ├── provider.*           (OpenAI/Anthropic)
  │   ├── tools.Registry       (所有内置工具)
  │   ├── session.Store        (JSONL 持久化)
  │   ├── sandbox              (bwrap)
  │   ├── skills               (SKILL.md)
  │   ├── context compaction   (压缩)
  │   └── context files        (AGENTS.md)
  │
  ├─ 新增 ──────────────────────────────────
  │   ├── memory tool          (memory.md 按需读写，不注入 system prompt)
  │   ├── messaging.Platform   (WeChat iLink / Feishu)
  │   ├── hermes/a2a           (A2A Server — Agent 间协作)
  │   ├── hermes.Webhook       (入站 webhook)
  │   ├── hermes.Hooks         (shell hooks)
  │   ├── budget pressure      (agent loop 注入)
  │   ├── context pressure     (compaction 层注入)
  │   └── smart approvals      (tools 层拦截)
  │
  └─ 增强 ──────────────────────────────────
      └── cron                 (管理 CLI 补齐)
```

### 7.4 Shell Hooks 协议

外部脚本通过 JSON stdin/stdout 通信：

**pre_tool_call — stdin:**
```json
{
  "hook": "pre_tool_call",
  "tool": "bash",
  "args": {"command": "rm -rf /tmp/test"},
  "platform": "wechat",
  "user_id": "wxid_12345"
}
```

**stdout:**
```json
{"action": "allow"}
```
或
```json
{"action": "block", "reason": "destructive command blocked"}
```

---

## 8. 实施阶段

### Phase 1: 骨架 & 配置（1 天）

- [ ] `internal/messaging/platform.go` — Platform 接口定义
- [ ] `internal/hermes/` 编排层骨架
- [ ] `hermes.json` 配置结构定义与加载
- [ ] `vibecoding hermes` 子命令注册（start/stop/status/config）
- [ ] Hermes server 主循环框架

### Phase 2: memory 工具 & 压力系统（1 天）

- [ ] `internal/memory/store.go` — memory.md 读写
- [ ] `internal/memory/tool.go` — memory 工具（read/add/update/delete）
- [ ] System prompt guidelines 添加静态 memory 提示
- [ ] Budget Pressure — tool result 注入预算警告
- [ ] Context Pressure — compaction 阈值警告

### Phase 3: 安全层（1 天）

- [ ] Smart Approvals — 命令危险性分类
- [ ] Shell Hooks — 外部脚本调用框架
- [ ] 用户白名单验证

### Phase 4: 微信网关（2 天）

- [ ] `internal/messaging/wechat/types.go` — iLink 协议类型定义
- [ ] `internal/messaging/wechat/protocol.go` — iLink HTTP API 调用
- [ ] `internal/messaging/wechat/auth.go` — QR 登录 + 凭证持久化
- [ ] `internal/messaging/wechat/crypto.go` — AES-128-ECB CDN 加解密
- [ ] `internal/messaging/wechat/wechat.go` — 实现 `messaging.Platform`
- [ ] `internal/hermes/dispatcher.go` — 消息 → Agent 转发
- [ ] `vibecoding hermes wechat login` — QR 码登录
- [ ] 消息平台命令（/new /clear /mode 等）

### Phase 5: 飞书网关（2 天）

- [ ] `go get github.com/larksuite/oapi-sdk-go/v3`
- [ ] `internal/messaging/feishu/feishu.go` — 实现 `messaging.Platform`（长连接）
- [ ] `internal/messaging/feishu/session.go` — per-user Session 隔离
- [ ] `vibecoding hermes feishu setup` — 交互式配置

### Phase 6: A2A Server + Webhook + Cron（1 天）

- [ ] `go get github.com/a2aproject/a2a-go/v2`
- [ ] `internal/hermes/a2a/server.go` — A2A JSON-RPC handler
- [ ] `internal/hermes/a2a/agent_card.go` — Agent Card 生成
- [ ] `internal/hermes/a2a/executor.go` — AgentExecutor 实现（A2A Task → agent loop）
- [ ] SSE 流式响应支持
- [ ] `internal/hermes/webhook/` — HTTP 入站 webhook
- [ ] Webhook 路由 → Agent 任务
- [ ] Cron 管理 CLI 命令完善

### Phase 7: 文档 & 测试（1 天）

- [ ] hermes 子命令使用文档
- [ ] hermes.json 配置文档
- [ ] 微信 iLink / 飞书 Bot 设置指南
- [ ] A2A Server 接入文档
- [ ] 单元测试
- [ ] 集成测试

**预计总工期：约 10 天**

---

## 9. 与现有模式的关系

| 维度 | CLI (TUI) | ACP | Gateway | **Hermes (新增)** |
|------|-----------|-----|---------|-------------------|
| **入口** | 终端 stdin | Editor stdio | HTTP API | 消息平台 (微信/飞书) + Webhook + **A2A** |
| **使用者** | 开发者本人 | 编辑器 | 其他应用 | **终端用户 (Bot)** |
| **Session** | 手动管理 | 编辑器管理 | 客户端指定 | **服务端自动管理 (per-user)** |
| **认证** | 无 | 无 | Bearer token | **平台用户白名单** |
| **常驻** | 否 | 否 | 是 | **是** |
| **Cron** | 无 | 无 | 无 | **内置调度器** |
| **记忆** | 无 | 无 | 无 | **memory.md (tool 按需读写)** |
| **配置** | settings.json | settings.json | gateway.json | **hermes.json** |
| **A2A** | 无 | 无 | 无 | **A2A Server (Agent 间协作)** |

---

## 10. 供应链安全原则

| 组件 | 策略 | 说明 |
|------|------|------|
| 微信 iLink | **自行实现** | 参考 iLink 协议规范实现为 internal 包，零外部依赖 |
| 飞书 SDK | **官方 SDK** | `larksuite/oapi-sdk-go` 飞书官方维护，可接受 |
| A2A SDK | **官方 SDK** | `a2aproject/a2a-go` Google/Linux Foundation 维护，可接受 |
| CDN 加密 | **标准库** | `crypto/aes` Go 标准库，无外部依赖 |
| HTTP 调用 | **标准库** | `net/http` Go 标准库 |

> **原则**：能用标准库实现的不引入外部包；必须引入的只用官方/基金会维护的 SDK。

---

## 11. 非目标

1. **Web 搜索** — 用户通过第三方 skill 扩展
2. **Checkpoints / Rollback** — 推迟
3. **企业微信** — 用个人微信 iLink 代替
4. **Memory 注入 system prompt** — 破坏缓存命中，改用 tool 按需读写
5. **Telegram / Discord** — v0.1.28
6. **Python 插件 / RL Training / Voice** — 不做

---

*决策已确认。可以开始开发。*
