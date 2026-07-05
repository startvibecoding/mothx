# Unified Serve Mode — 融合方案

> 状态: Draft
> 日期: 2026-07-05
> 目标版本: v0.2.0

---

## 1. 问题

当前有三种服务端运行模式，各自独立：

| 模式 | 命令 | 配置文件 | 能力 |
|------|------|----------|------|
| Gateway | `mothx gateway` | `gateway.json` | OpenAI API、认证、session 管理 |
| Hermes | `mothx hermes` | `hermes.json` | 微信/飞书通道、cron、memory、webhook |
| Serve | `mothx serve` | `serve.json` | 把上面两个硬拼在一起 |

用户痛点：
- 不知道该用哪个命令
- 配置文件碎片化：`serve.json` 内部嵌套了 `gateway`、`channels`、`cron`、`memory` 等来自不同模块的字段
- Gateway 没有微信/飞书，Hermes 没有 Web UI 和 OpenAI API，功能割裂
- 现有 `mothx serve` 只是物理拼装，没有统一模型

## 2. 目标

- **一个命令**: `mothx serve`
- **一个配置文件**: `serve.json`
- **一个心智模型**: 用户不需要知道底层有 gateway 和 hermes
- **零配置即跑**: 启动后 Web UI + OpenAI API 可用，通道按需开启
- **CLI/TUI 不受影响**: `mothx` 交互式模式、`mothx acp` 等保持不变

补充约束：

- **先统一配置和编排，再统一内部包边界**
- **Phase 1 不要求立即删除 `gateway` / `hermes` 现有包和子命令**
- **文档中出现的“目标架构”不等于当前实现边界**

## 3. 统一配置: `serve.json`

```
~/.mothx/serve.json          # 全局
<project>/.mothx/serve.json  # 项目级覆盖
```

### 3.1 Schema

```jsonc
{
  // ── 服务基础 ──
  "listen": ":8080",
  "provider": "openai",
  "model": "gpt-4o",
  "mode": "yolo",
  "workDir": ".",

  // ── 认证 ──
  "auth": {
    "enabled": false,
    "tokens": []
  },

  // ── 功能开关 ──
  "features": {
    "webUI": true,
    "openaiAPI": true,
    "wechat": false,
    "feishu": false,
    "websocket": false,
    "multiAgent": false,
    "cron": false,
    "memory": true
  },

  // ── 安全 ──
  "sandbox": {
    "enabled": false,
    "level": "standard"
  },
  "allowedWorkDirs": null,

  // ── 通道配置（仅 features 中启用的生效）──
  "channels": {
    "wechat": { "autoTyping": true },
    "feishu": { "appId": "", "appSecret": "" }
  },

  // ── 高级配置 ──
  "session": {
    "idleTimeoutSeconds": 1800,
    "maxSessions": 100
  },
  "toolVisibility": { "mode": "content", "detail": "collapsed" },
  "thinking": "medium",
  "systemPromptMode": "append",
  "requestTimeoutSeconds": 1800,
  "maxConcurrentRequests": 10,
  "agent": { "maxTurns": 50, "maxTokens": 8192 }
}
```

### 3.2 与旧配置的关系

`serve.json` 是唯一推荐入口，但 `gateway.json` 和 `hermes.json` 不在 Phase 1 立即删除。

迁移策略：

- **Phase 1**: `serve.json` 成为主配置；`gateway.json` / `hermes.json` 继续被旧命令使用
- **Phase 2**: `mothx gateway` / `mothx hermes` 输出弃用提示，文档默认只介绍 `mothx serve`
- **Phase 3**: 视迁移完成度决定是否删除旧命令与旧配置读取逻辑

用户升级路径：

- 新用户：直接创建 `serve.json`
- 旧用户：允许继续使用 `gateway.json` / `hermes.json` 一段兼容窗口
- 如需自动迁移，可后续补充 `serve.json` 初始化或导入工具，但不作为 Phase 1 前置条件

## 4. 启动

```bash
mothx serve                  # 默认启动
mothx serve --port 9090      # 指定端口
mothx serve --provider deepseek --model deepseek-v4
mothx serve --sandbox
mothx serve --lobster        # yolo + 无 sandbox + 子代理
mothx --init-serve           # 生成 serve.json 模板（当前已存在）
```

启动输出：

```
  MothX Serve v0.2.0

  Web UI        http://127.0.0.1:8080/
  OpenAI API    http://127.0.0.1:8080/v1
  WeChat        ✗ disabled
  Feishu        ✗ disabled

  Config: ~/.mothx/serve.json
```

## 5. Web UI

从调试面板升级为管理控制台：

```
┌─────────────────────────────────────────────────────┐
│  MothX Serve                           [状态: 运行中] │
├──────────┬──────────────────────────────────────────┤
│  Chat    │  流式对话 + 工具调用展示                    │
│  Sessions│  会话列表、统计、删除                       │
│  Channels│  WeChat/Feishu 连接状态                    │
│  Settings│  可视化编辑 serve.json                     │
│  Logs    │  实时日志（WebSocket）                      │
└──────────┴──────────────────────────────────────────┘
```

Chat 页面核心：
- SSE 流式输出
- 工具调用 collapsed 展示，点击展开
- Session selector（新建/切换/删除）
- Markdown 渲染
- `Enter` 发送、`Shift+Enter` 换行

技术方向：Svelte 5 + Vite，输出到 `ui/dist/`，由 `mothx serve` 提供静态资源。

实现约束：

- Phase 1 只要求 Chat 页面可用
- Settings 页面允许先用 JSON 编辑器或现有 API 直连
- 不要求在第一阶段完成完整控制台和所有管理面板
- 嵌入静态资源、构建链路和 CI 集成需要单独设计，不在本提案中默认视为已解决

## 6. 代码架构

### 6.1 原则

- **不变的内核**: `internal/agent/` — agent loop、system prompt、工具执行、流式事件
- **先抽共享运行时，再谈目录重命名**
- **serve 作为唯一用户入口，但内部重构允许分阶段进行**
- **任何包拆分都要以当前代码边界为起点，不按文档一次性搬家**

### 6.2 当前实现现状

当前 `serve` 不是统一运行时，而是已有模块的编排层：

- `internal/serve/run.go` 负责加载 `serve.json`
- `serve` 会先构造 Hermes dispatcher 和消息平台
- `serve` 会把 `cfg.Gateway` 写入临时文件，再调用 `gateway.Run(...)`
- Web UI、`/api/serve/config`、`/api/settings`、`/api/channels` 已通过 `gateway.ExtraRoutes` 方式挂载

这意味着：

- 当前 `serve.json` 仍然是旧配置模型的组合，不是最终扁平 schema 的原生运行时
- `gateway` 仍同时承担 HTTP server、OpenAI API、provider/model 构建、sandbox、session pool、skills/context 初始化
- `hermes` 的 dispatcher 仍同时承担消息分发、session 生命周期、security、hooks、cron、MCP、agent 构建

### 6.3 目标态模块拆分

```
internal/
├── agent/              # 核心 agent loop（不变）
├── httpserver/         # ← 从 gateway 拆出：HTTP 基础设施
│   ├── server.go       # http.Server 启动、优雅关闭
│   ├── middleware.go   # CORS、认证中间件
│   └── static.go       # 静态文件服务（Web UI）
├── api/                # ← 从 gateway 拆出：OpenAI 兼容 API
│   ├── chat.go         # POST /v1/chat/completions
│   ├── models.go       # GET /v1/models
│   └── tool_format.go  # 工具调用输出格式化
├── runtime/            # ← 新增：共享运行时装配
│   ├── builder.go      # provider/model/sandbox/context/skills 装配
│   ├── agent_factory.go
│   └── tool_registry.go
├── sessionruntime/     # ← 从 gateway + hermes 拆出：会话运行时
│   ├── pool.go         # 会话池（idle timeout、max sessions）
│   └── session.go      # Gateway/Hermes 共享会话包装
├── channels/           # ← 从 hermes + messaging 合并
│   ├── platform.go     # Platform 接口
│   ├── wechat/         # 微信 bot
│   ├── feishu/         # 飞书 bot
│   └── websocket/      # WebSocket 通道
├── dispatch/           # ← 从 hermes 拆出：消息分发
│   └── dispatcher.go   # 平台消息 → agent loop → 回复
├── cron/               # ← 从 hermes 拆出：定时调度
├── memory/             # 保留：持久化记忆
├── security/           # ← 从 hermes 拆出：安全策略
├── provider/           # 不变
├── tools/              # 不变
├── session/            # 保留：SQLite 持久化和 schema migration
├── context/            # 不变
├── skills/             # 不变
└── serve/              # 编排层（重写）
    ├── config.go       # serve.json 加载/保存
    ├── run.go          # 启动编排
    └── routes.go       # 管理 API 路由
```

说明：

- 这里刻意不用 `sessions/` 命名，避免与现有 `internal/session/` 混淆
- `internal/session/` 继续负责 SQLite 存储、schema migration、`sessions.db`
- 新增层只负责“运行时会话对象”和缓存，不替代底层持久化包

### 6.4 模块职责

| 模块 | 来源 | 职责 | 接口 |
|------|------|------|------|
| `agent` | 保留 | agent loop、工具执行 | `Agent.Run(ctx, messages) → events` |
| `httpserver` | gateway | HTTP 服务器 | `Server.Run(handler)` |
| `api` | gateway | OpenAI 兼容路由 | 注册到 mux |
| `runtime` | 新增 | provider/model/sandbox/context/skills 共享装配 | `BuildRuntime(cfg) → Runtime` |
| `sessionruntime` | 新增 | 会话缓存、默认 session、共享会话包装 | `Pool.GetOrCreate(id) → Session` |
| `channels` | hermes + messaging | 平台 bot 适配 | `Platform.Start/Stop/IsConnected` |
| `dispatch` | hermes | 消息分发 | `Dispatcher.HandleMessage(msg)` |
| `cron` | hermes | 定时任务 | `Scheduler.Add/Remove` |
| `memory` | 保留 | 持久化记忆 | `Store.Read/Write` |
| `security` | hermes | bash 执行策略 | `Guard.Check(cmd) → bool` |
| `session` | 保留 | SQLite 会话存储和 migration | `ApplyMigrations/Open/New` |
| `serve` | 重写 | 编排所有模块 | `Run(opts, version) → error` |

### 6.5 过渡态架构

Phase 1 的现实目标不是一次性把 `gateway/`、`hermes/` 删除，而是先把重复运行时逻辑收束起来。

过渡态建议：

1. 保留现有 `internal/gateway`、`internal/hermes` 包和 `mothx gateway`、`mothx hermes` 命令
2. 新增共享的 `runtime` 和 `sessionruntime` 层
3. 让 `gateway.Run` 和 `hermes.NewDispatcher` 优先依赖共享装配逻辑
4. `serve.Run` 先停止“写临时 gateway.json 再启动”的做法，改为直接传 struct
5. Web UI 和管理 API 继续由 `serve` 编排，但底层尽量复用共享 runtime

也就是说，**先消除重复运行时，再重命名目录**。

### 6.6 serve 编排流程（目标态）

```go
func Run(opts RunOptions, version string) error {
    cfg, path, err := loadConfig(opts.ConfigPath)
    if err != nil { return err }
    applyOverrides(cfg, opts)

    // 初始化模块
    sessionPool := sessionruntime.NewPool(...)
    agent := agent.New(...)
    dispatcher := dispatch.New(agent, sessionPool, ...)

    // 按 features 启动通道
    if cfg.Features.Wechat { channels.NewWechat(...) }
    if cfg.Features.Feishu { channels.NewFeishu(...) }

    // 组装 HTTP 路由
    mux := http.NewServeMux()
    if cfg.Features.OpenaiAPI { api.RegisterChatRoutes(mux, ...) }
    if cfg.Features.WebUI    { httpserver.RegisterStatic(mux, ...) }
    routes.RegisterManageRoutes(mux, cfg, ...)

    return httpserver.Run(mux, cfg.Listen)
}
```

### 6.7 Phase 1 编排流程（建议实现）

```go
func Run(opts RunOptions, version string) error {
    cfg, path, err := loadServeConfig(opts.ConfigPath)
    if err != nil { return err }
    applyOverrides(cfg, opts)

    settings, err := config.LoadSettings()
    if err != nil { return err }

    rt, err := runtime.BuildFromServeConfig(cfg, settings, version)
    if err != nil { return err }

    channelRuntime, err := serve.StartChannels(rt, cfg)
    if err != nil { return err }

    mux := http.NewServeMux()
    serve.RegisterManageRoutes(mux, cfg, path, channelRuntime)
    gateway.RegisterOpenAIRoutes(mux, rt.GatewayAdapter())
    serve.RegisterStatic(mux, cfg.WebUI)

    return serve.RunHTTP(rt, mux)
}
```

这个阶段允许内部仍然调用部分 `gateway` / `hermes` 代码，但不再依赖临时配置文件，也不要求目录已经完成最终拆分。

### 6.8 gateway/hermes 去向（目标态）

| 原文件 | 去向 |
|--------|------|
| `gateway/gateway.go` | → `httpserver/server.go` |
| `gateway/handler_chat.go` | → `api/chat.go` |
| `gateway/tool_format.go` | → `api/tool_format.go` |
| `gateway/session_mgr.go` | → `sessionruntime/pool.go` |
| `gateway/auth.go` | → `httpserver/middleware.go` |
| `gateway/config.go` | Phase 2 后视兼容窗口决定 |
| `hermes/dispatcher.go` | → `dispatch/dispatcher.go` |
| `hermes/config.go` | Phase 2 后视兼容窗口决定 |
| `hermes/security.go` | → `security/guard.go` |
| `hermes/hooks/` | → `hooks/` |
| `hermes/cron/` | → `cron/` |
| `messaging/` | → `channels/` |
| `hermes/ws/` | → `channels/websocket/` 或 `httpserver/`，待定 |
| `hermes/` | Phase 3 评估后再删 |
| `gateway/` | Phase 3 评估后再删 |

### 6.9 不变的模块

```
internal/agent/        agent loop
internal/provider/     provider 抽象
internal/tools/        内置工具
internal/session/      SQLite session 存储
internal/context/      context window
internal/contextfiles/ AGENTS.md 发现
internal/skills/       skills 加载
internal/memory/       持久化记忆
internal/config/       settings.json
internal/stats/        使用统计
```

## 7. API

### OpenAI 兼容

| 端点 | 说明 |
|------|------|
| `POST /v1/chat/completions` | 核心对话接口 |
| `GET /v1/models` | 可用模型列表 |

### 管理 API

| 端点 | 说明 |
|------|------|
| `GET /health` | 健康检查 |
| `GET/PUT /api/serve/config` | 读写 serve.json |
| `GET /api/sessions` | 会话列表 |
| `DELETE /api/sessions/:id` | 删除会话 |
| `GET /api/channels` | 通道连接状态 |
| `GET/PUT /api/settings` | 读写 settings.json |
| `WS /ws/logs` | 实时日志流 |

备注：

- `GET/PUT /api/settings` 是现有方向，文档中不再写成单独的 `GET`
- `/api/sessions` 在 Phase 1 可先只覆盖 Web UI / WebSocket / 当前 serve 运行时可见会话
- 如果需要列出完整历史会话，应单独定义是否查询 `sessions.db`

## 8. 不影响的部分

- **CLI/TUI**: `mothx` 交互模式、`mothx acp`、`mothx stats`、`mothx doctor` 等完全不变
- **ACP 服务器**: 不受影响
- **A2A 协议**: 可选集成到 serve（通过 features.a2a 开关）
- **settings.json**: 保持不变，serve.json 是独立配置
- **sessions.db**: 共享同一个 SQLite 存储

## 9. 命令与兼容策略

短期保持现状：

- `mothx serve` 继续作为统一入口
- `mothx gateway` 继续保留
- `mothx hermes` 继续保留，包括 `start/stop/status/client/wechat/...` 子命令
- `mothx --init-serve` 继续作为创建模板的已实现入口

弃用策略：

- 不在 Phase 1 删除 `mothx gateway` / `mothx hermes`
- 当 `serve` 覆盖 Gateway 和 Hermes 的主要能力后，再给旧命令加 deprecated 提示
- `hermes` 独有的运维子命令若仍有价值，可迁移为 `mothx serve channels ...` 或保留独立命令

## 10. 实施计划

| 阶段 | 内容 | 周期 |
|------|------|------|
| Phase 1 | 扁平 `serve.json` schema + 兼容映射层 + 去除临时 `gateway.json` 文件写入 + Web UI Chat 页面 | 2 周 |
| Phase 2 | 共享 runtime/sessionruntime 抽取 + Sessions/Channels/Settings 管理页面 + WebSocket 日志 | 1-2 周 |
| Phase 3 | gateway/hermes 旧命令弃用、评估删除旧配置读取逻辑和旧目录 | 1 周 |

Phase 1 明确不做：

- 不要求立即删除 `internal/gateway` / `internal/hermes`
- 不要求一次性完成完整控制台
- 不要求自动迁移旧配置
- 不要求在第一阶段统一 Gateway 与 Hermes 的全部会话语义

## 11. 风险

| 风险 | 缓解 |
|------|------|
| `serve` 继续依赖旧 runtime，文档与代码脱节 | 先写过渡态架构，按阶段实施 |
| Gateway/Hermes 都有自己的 session 包装和缓存 | 先抽共享 runtime/sessionruntime，不急于删旧包 |
| `internal/session` 与新会话层命名混淆 | 新层避免使用 `sessions/` 命名 |
| Web UI 前端资源与构建链路增加复杂度 | Phase 1 只做 Chat，其余 JSON 编辑器兜底 |
| Hermes dispatcher 仍承载 security/hooks/cron/MCP | 先拆共享 builder，再拆 dispatcher |
| 旧用户升级配置丢失 | `mothx --init-serve` 模板 + 兼容窗口 + 后续迁移工具 |

## 12. 总结

目标仍然是一个命令 `mothx serve`、一个配置文件 `serve.json`、一个 Web UI 管理控制台。但实现顺序需要调整：先统一配置和运行时装配，再推进目录拆分和旧命令退场。`agent`、`provider`、`tools`、`session`、`context`、`skills` 等核心模块保持稳定，`serve` 逐步成为唯一的用户入口，而不是通过一次性大搬家强行完成统一。
