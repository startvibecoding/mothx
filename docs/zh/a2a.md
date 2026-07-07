# A2A 协议（Agent-to-Agent）

## 概述

A2A（Agent-to-Agent）协议使不同的 AI Agent 能够互相发现、通信和协作。MothX 将 A2A 协议实现为独立服务器。

## 快速开始

```bash
# 独立模式
mothx a2a start

# 查看状态
mothx a2a status

# 查看 Agent Card
mothx a2a card

# 向其他 A2A 服务器发送任务
mothx a2a send "列出所有 Go 文件" --target http://remote:8093

# 发现远程 Agent Card
mothx a2a discover http://remote:8093

# 停止
mothx a2a stop
```

## 运行模式

在单独的端口运行专用的 A2A HTTP 服务器（默认：`127.0.0.1:8093`）。

```bash
mothx a2a start --port 8093 --work-dir /path/to/project
```

只有在明确需要对外暴露 A2A 服务时才使用 `--host 0.0.0.0`，并为对外部署配置 auth token。

## 协议细节

- **传输**：JSON-RPC 2.0 over HTTP
- **流式**：SSE（Server-Sent Events）实时推送
- **Task 生命周期**：`submitted` → `working` → `completed`/`failed`/`canceled`

## Agent Card

Agent Card 描述 Agent 的能力，在 `/.well-known/agent.json` 提供。

```json
{
  "name": "MothX",
  "description": "AI coding assistant with file editing, terminal, and search capabilities",
  "url": "http://localhost:8093/a2a",
  "version": "0.1.31",
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

## JSON-RPC 方法

### `message/send`

发送消息以创建或继续任务。

**请求：**
```json
{
  "jsonrpc": "2.0",
  "method": "message/send",
  "params": {
    "task_id": "task_123",  // 可选，省略则创建新任务
    "message": {
      "role": "user",
      "parts": [
        {"type": "text", "text": "帮我重构 main.go"}
      ]
    }
  },
  "id": 1
}
```

**响应（同步）：**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "id": "task_123",
    "state": "completed",
    "artifacts": [
      {
        "name": "response",
        "parts": [{"type": "text", "text": "我已经分析了 main.go..."}]
      }
    ]
  },
  "id": 1
}
```

**SSE 流式（添加 `Accept: text/event-stream` 头）：**
```
data: {"task_id":"task_123","state":"working","message":{"role":"agent","parts":[{"type":"text","text":"让我"}]}}

data: {"task_id":"task_123","state":"working","message":{"role":"agent","parts":[{"type":"text","text":"分析代码..."}]}}

data: {"task_id":"task_123","state":"completed","artifact":{"name":"response","parts":[{"type":"text","text":"这是重构后的版本..."}]}}
```

### `task/get`

获取任务当前状态。

**请求：**
```json
{
  "jsonrpc": "2.0",
  "method": "task/get",
  "params": {
    "task_id": "task_123"
  },
  "id": 2
}
```

### `task/cancel`

取消运行中的任务。

**请求：**
```json
{
  "jsonrpc": "2.0",
  "method": "task/cancel",
  "params": {
    "task_id": "task_123"
  },
  "id": 3
}
```

## REST 端点

为简化集成，也提供 REST 风格的端点：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/.well-known/agent.json` | GET | Agent Card |
| `/a2a` | POST | JSON-RPC 2.0 端点 |
| `/a2a/send` | POST | 提交任务（同步或 SSE） |
| `/a2a/task?task_id=xxx` | GET | 获取任务状态 |
| `/a2a/task/cancel` | POST | 取消任务 |
| `/a2a/events?task_id=xxx` | GET | SSE 事件流 |

## Task 状态

```
submitted ─► working ─► completed
                    ─► failed
                    ─► canceled
```

## 示例

### 提交任务（curl）

```bash
# 同步响应
curl -X POST http://localhost:8093/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "列出项目中的所有 Go 文件"}]
      }
    },
    "id": 1
  }'

# SSE 流式
curl -X POST http://localhost:8093/a2a \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "解释项目结构"}]
      }
    },
    "id": 1
  }'
```

### REST API

```bash
# 提交任务
curl -X POST http://localhost:8093/a2a/send \
  -H "Content-Type: application/json" \
  -d '{"message": {"role": "user", "parts": [{"type": "text", "text": "你好"}]}}'

# 获取任务
curl http://localhost:8093/a2a/task?task_id=task_123

# 取消任务
curl -X POST http://localhost:8093/a2a/task/cancel \
  -H "Content-Type: application/json" \
  -d '{"task_id": "task_123"}'
```

## 安全

- **Auth Token**：来自 `a2a.json` 或 `--auth-token` 标志的 Bearer token 认证
- **Agent Card**：公开访问（无需认证）
- **受保护端点**：配置 `auth_token` 后，`/a2a`、REST A2A 路由和 `/a2a/events` 都需要认证

配置认证后，客户端需要发送：

```bash
Authorization: Bearer <auth_token>
```

## A2A Client

向其他 A2A 服务器发送任务。

```bash
# 发送任务
mothx a2a send "解释项目结构" --target http://remote:8093

# 带认证发送
mothx a2a send "运行测试" --target http://remote:8093 --auth-token xxx

# 发现服务器能力
mothx a2a discover http://remote:8093
```

## A2A 调度

定时任务可以向 A2A 服务器发送任务，而不是运行本地 Agent。

```bash
# 调度每日任务到远程 A2A 服务器
mothx serve cron add "daily-review" "review recent changes" \
  --schedule "@daily" \
  --a2a-target http://review-agent:8093

# 带认证的调度
mothx serve cron add "ci-check" "run CI tests" \
  --schedule "@every 1h" \
  --a2a-target http://ci-agent:8093 \
  --a2a-token ${CI_TOKEN}
```

调度器会将 prompt 发送到 A2A 服务器，而不是启动本地 Agent。

## A2A Master 模式

A2A Master 模式让你可以在一个 MothX 实例中管理多个远程 A2A Agent，通过 `a2a_dispatch` tool 向它们分发任务。

### 快速开始

```bash
# 1. 生成示例配置
mothx --init-a2a-master-config

# 2. 编辑 a2a-list.json，填入实际的远程 agent 信息
#    位置：~/.vibecoding/a2a-list.json 或 .vibe/a2a-list.json

# 3. 启用 master 模式
mothx --enable-a2a-master
```

### 配置文件

`a2a-list.json` 结构如下：

```json
{
  "agents": [
    {
      "name": "code-reviewer",
      "url": "http://localhost:8093"
    },
    {
      "name": "ci-agent",
      "url": "http://ci-server:8093",
      "auth_token": "your-secret-token"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | Agent 名称（唯一标识，用于 tool 调用） |
| `url` | string | A2A 服务器地址 |
| `auth_token` | string | Bearer Token（可选） |

配置文件位置（优先级从低到高）：
- `~/.vibecoding/a2a-list.json`（全局）
- `.vibe/a2a-list.json`（项目级，覆盖全局）

### a2a_dispatch Tool

启用后，LLM 会多出一个 `a2a_dispatch` tool，可以向注册的远程 agent 发送任务：

**参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| `agent_name` | string | 目标 agent 名称（从配置中自动枚举） |
| `message` | string | 任务消息 |

**示例：**
```
a2a_dispatch(agent_name="code-reviewer", message="review main.go for bugs")
a2a_dispatch(agent_name="ci-agent", message="run all unit tests")
```

### CLI 参数

| 参数 | 说明 |
|------|------|
| `--enable-a2a-master` | 启用 A2A Master 模式（默认关闭） |
| `--init-a2a-master-config` | 生成示例 `a2a-list.json` |
| `--force` | 覆盖已存在的配置文件 |
