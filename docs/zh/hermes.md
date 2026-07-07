# Hermes 模式

## 概述

Hermes 模式将 MothX 作为**消息平台网关守护进程**运行，支持 WebSocket/HTTP API、微信、飞书和 A2A 协议。它将 MothX 从编码助手扩展为可部署的自主代理。

```bash
mothx hermes start
```

## 快速开始

```bash
# 生成配置模板
mothx hermes config init

# 启动 hermes（前台）
mothx hermes start

# 启动 hermes（后台）
mothx hermes start -d

# 查看状态
mothx hermes status

# 停止 hermes
mothx hermes stop
```

## 架构

```
                    ┌─────────────────────────────────────┐
                    │         Hermes 网关 (:8090)          │
                    │                                     │
                    │  ┌─────────┐  ┌─────────┐  ┌─────┐ │
   微信 ───────────►│  │消息平台 │  │  HTTP   │  │ A2A │ │
   飞书 ───────────►│  │适配器   │  │  REST   │  │     │ │
                    │  └────┬────┘  └────┬────┘  └──┬──┘ │
                    │       │            │          │     │
                    │       └──────┬─────┘──────────┘     │
                    │              ▼                       │
                    │       ┌──────────┐                  │
                    │       │  调度器   │                  │
                    │       └────┬─────┘                  │
                    │            ▼                        │
                    │  ┌──────────────────┐               │
                    │  │   Agent 循环     │               │
                    │  │  (per-user)      │               │
                    │  └──────────────────┘               │
                    └─────────────────────────────────────┘
```

## CLI 命令

### `hermes start`

启动 Hermes 守护进程。

| 标志 | 说明 |
|------|------|
| `-d` | 后台运行 |
| `--port` | 监听端口（默认：配置值或 8090） |
| `--work-dir` | 默认工作目录 |
| `-p`, `--provider` | 覆盖默认 provider |
| `-m`, `--model` | 覆盖默认 model |
| `--multi-agent` | 启用子 Agent 工具 |
| `--sandbox` | 启用 bwrap 沙箱 |
| `--config` | hermes.json 路径 |
| `--verbose` | 详细输出 |
| `--debug` | 调试日志和本地 pprof |

### `hermes stop`

通过 PID 文件 + SIGTERM 停止运行中的 Hermes 守护进程。

### `hermes status`

检查 Hermes 守护进程状态（PID 检查 + HTTP health 查询）。

### `hermes config`

管理 Hermes 配置。

```bash
mothx hermes config init              # 创建全局配置模板
mothx hermes config init --project    # 创建项目配置模板
mothx hermes config show              # 查看生效配置
```

### `hermes wechat`

管理微信 iLink 连接。

```bash
mothx hermes wechat login             # 扫码登录
mothx hermes wechat login --force     # 强制重新登录
mothx hermes wechat status            # 查看连接状态
```

### `hermes feishu`

管理飞书连接。

```bash
mothx hermes feishu setup             # 显示配置指南
mothx hermes feishu status            # 查看连接状态
```

### `hermes webhook`

管理 webhook 路由。

```bash
mothx hermes webhook list             # 列出配置的路由
```

### `hermes memory`

管理持久化记忆。

```bash
mothx hermes memory show              # 查看 memory.md 内容
mothx hermes memory clear             # 重置 memory.md
```

### `hermes sessions`

管理 session。

```bash
mothx hermes sessions list            # 列出活跃 session（查询运行实例）
```

### `hermes cron`

管理定时任务。

```bash
mothx hermes cron list                # 列出所有定时任务
mothx hermes cron add <name> <prompt> # 添加定时任务
mothx hermes cron remove <id>         # 删除定时任务
mothx hermes cron enable <id>         # 启用定时任务
mothx hermes cron disable <id>        # 禁用定时任务
```

## 配置

### `hermes.json`

Hermes 模式的配置文件。支持全局 + 项目级覆盖。

**位置：**
- 全局：`<GLOBAL_DIR>/hermes.json`
- 项目：`.vibe/hermes.json`（覆盖全局）

```jsonc
{
  "server": {
    "port": 8090,
    "host": "0.0.0.0",
    "auth_token": ""
  },
  "default_provider": "",
  "default_model": "",
  "multi_agent": false,
  "sandbox": false,
  "wechat": {
    "enabled": false,
    "cred_path": "",
    "work_dir": "",
    "allowed_users": [],
    "auto_typing": true
  },
  "feishu": {
    "enabled": false,
    "app_id": "",
    "app_secret": "",
    "work_dir": "",
    "allowed_users": []
  },
  "webhooks": {
    "enabled": false,
    "secret": "",
    "routes": [
      {
        "path": "/github",
        "events": ["push", "pull_request"],
        "skill": "code-review",
        "delivery": "feishu",
        "delivery_target": "chat_id"
      }
    ]
  },
  "a2a": {
    "enabled": false,
    "port": 8093
  },
  "cron": {
    "enabled": true
  },
  "memory": {
    "enabled": true,
    "path": ""
  },
  "security": {
    "smart_approvals": true,
    "allowed_work_dirs": []
  },
  "hooks": {
    "pre_tool_call": "",
    "post_tool_call": ""
  },
  "agent": {
    "max_turns": 90,
    "budget_pressure": true,
    "context_pressure": true,
    "budget_pressure_threshold": 0.20,
    "context_pressure_threshold": 0.55
  },
  "work_dir": "."
}
```

### 配置优先级

```
CLI 标志 > hermes.json（项目） > hermes.json（全局） > 默认值
```

### 工作目录优先级

```
平台 work_dir（微信/飞书） > 全局 work_dir > CLI --work-dir > 当前目录
```

## 消息平台

### 微信（iLink 协议）

- 零外部依赖（仅 Go 标准库）
- 扫码登录，凭证保存到 `<GLOBAL_DIR>/wechat-credentials.json`
- 长轮询接收消息（无需公网 IP）
- 过期自动重新登录
- 支持打字指示器

### 飞书

- 官方 SDK：`github.com/larksuite/oapi-sdk-go/v3`
- WebSocket 长连接（无需公网 IP）
- 支持文本消息
- 自动重连

## WebSocket API

### 连接

```
ws://localhost:8090/ws?session=<session_id>
```

配置 `server.auth_token` 后，应在 WebSocket 握手时通过 HTTP header 发送 token：

```http
Authorization: Bearer <auth_token>
```

旧的 `?token=` query 参数仍兼容，但推荐使用 header，避免 token 暴露在 URL 和日志中。

### 客户端 → 服务端消息

```jsonc
// 聊天消息
{"type": "message", "content": "帮我看看这段代码"}

// 斜杠命令
{"type": "command", "content": "/new"}

// 审批响应
{"type": "approval", "approval_id": "ap_xxx", "approved": true}

// 心跳
{"type": "ping"}
```

### 服务端 → 客户端消息

```jsonc
// 连接确认
{"type": "connected", "session_id": "...", "version": "..."}

// 流式文本
{"type": "text_delta", "content": "让我帮你..."}

// 思考过程
{"type": "think_delta", "content": "分析代码..."}

// 工具调用
{"type": "tool_call", "tool": "read", "call_id": "...", "args": {"path": "main.go"}}

// 工具结果
{"type": "tool_result", "tool": "read", "call_id": "...", "result": "..."}

// 文件 diff
{"type": "tool_diff", "call_id": "...", "path": "main.go", "diff": "..."}

// 审批请求（高风险）
{"type": "approval_request", "approval_id": "ap_xxx", "tool": "bash", "args": {...}}

// 用量统计
{"type": "usage", "prompt_tokens": 1200, "completion_tokens": 350}

// 轮次完成
{"type": "done", "stop_reason": "end_turn"}

// 状态消息
{"type": "status", "message": "触发压缩"}

// 命令响应
{"type": "command_result", "command": "/new", "message": "✅ 新 session 已创建"}

// 错误
{"type": "error", "message": "provider error"}

// 心跳响应
{"type": "pong"}
```

## HTTP REST API

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/health` | GET | 否 | 健康检查 |
| `/api/status` | GET | 是 | 服务状态 |
| `/api/sessions` | GET | 是 | 列出活跃 session |
| `/api/sessions/{id}` | GET | 是 | session 详情 |
| `/api/sessions/{id}` | DELETE | 是 | 删除 session |
| `/api/memory` | GET | 是 | 读取 memory.md |
| `/api/memory` | PUT | 是 | 更新 memory.md |
| `/api/platforms` | GET | 是 | 平台状态 |
| `/webhook/*` | POST | Secret | Webhook 入站 |

## 智能审批

工具调用的分级风险分类：

| 风险等级 | WebSocket | 消息平台 |
|----------|-----------|----------|
| Low | 自动批准 | 自动批准 |
| Medium | 自动批准 + 通知 | 自动批准 + 通知 |
| High | `approval_request` → 等待响应（5 分钟超时） | 自动拒绝 + 通知 |

**风险分类：**
- **Low**：`go`、`make`、`npm`、`git status/log/diff`、`ls`、`cat`、`grep`、`find`
- **Medium**：`mv`、`cp -r`、`git push`、`docker`、`curl`、`ssh`
- **High**：`rm -rf`、`sudo`、`shutdown`、`curl | sh`、`eval`、`exec`

## 压力系统

### Context Pressure

当 context 使用率超过阈值（默认 55%）时触发 `EventContextPressure`。

```jsonc
{
  "agent": {
    "context_pressure": true,
    "context_pressure_threshold": 0.55
  }
}
```

### Budget Pressure

当剩余迭代次数达到阈值（默认 20%）时触发 `EventBudgetPressure`。

```jsonc
{
  "agent": {
    "budget_pressure": true,
    "budget_pressure_threshold": 0.20
  }
}
```

两者都是一次性事件：每个阈值越界只触发一次，非每轮触发。

> **注意：** `context_pressure_threshold` 和 `budget_pressure_threshold` 的值范围为 0-1（0.55 表示 55%），而非 0-100。内部比较时会自动转换为百分比。`InitHermesConfig` 生成的项目模板已包含这些默认值，确保即使不手动配置也能正常工作。

## Memory

持久化记忆存储为 `memory.md`（Markdown 格式，人类可读）。

**查找优先级：**
1. `memory.path` 配置 → 显式路径
2. `.vibe/memory.md` → 项目记忆
3. `<GLOBAL_DIR>/memory.md` → 全局记忆

**Section：**
- `## User Profile` — 用户偏好
- `## Working Memory` — 当前上下文
- `## Lessons Learned` — 积累的知识

**默认：** 写入 `.vibe/memory.md`（项目目录）。

## Session 管理

- 每个 `platform:user_id` 一个持久 session
- `/new` 归档当前 session 并创建新 session
- Session 存储在 `<sessionDir>/hermes/<platform>/<user_id>/active.db`
- Context 窗口满时自动压缩

## A2A 协议

详见 [A2A 文档](a2a.md)。

## 安全

- **用户白名单**：per-platform `allowed_users`
- **Auth Token**：HTTP/WebSocket API 的 Bearer token
- **Allowed Work Dirs**：限制工作目录
- **Shell Hooks**：pre/post tool call 外部脚本
- **智能审批**：分级风险分类
