# Serve 作为多 Agent Session 工作台的发展路线

> 状态: 讨论中 (Draft)
> 日期: 2026-07-08
> 目标版本: 待定
> 最近更新: Web UI 已支持 session 级工具勾选；serve 已支持 `--web-search`、`--browser`、`--enable-a2a-master`

## 1. 背景

`mothx serve` 已经不再是旧 HTTP API 模式和消息通道模式的简单拼接。它现在同时承载 OpenAI-compatible API、Web UI、WebSocket、cron、memory、channels、workflow 和 multi-agent 能力。

当前需要重新明确一个核心问题：**serve 的主定位是什么？**

如果仍把 serve 理解为“一个有默认工作目录的 HTTP 服务”，Web UI 就会自然倾向于只展示当前进程/当前 workDir 下的 active session。但 MothX 的 session 本身已经有 `cwd` 元数据，并且所有会话都统一写入 `sessions.db`。从长期看，serve 更合理的定位不是单一工作目录的服务，而是：

> Serve 是 MothX / VibeCoding Agent 核心能力的 GUI 化、Session 化和编排化工作台。

因此，工作目录不应该是 serve 进程的中心概念。**工作目录应该属于 session**，serve 只提供默认值、安全边界和调度入口。

## 2. 核心定位

### 2.1 不是这些

- 不是旧 API server 的 Web 管理页。
- 不是单一 cwd 下的聊天窗口。
- 不是另写一套 agent 逻辑。
- 不是只服务 WeChat/Feishu/WebSocket 的 channel daemon。

### 2.2 应该是这些

- **Session 工作台**：跨工作目录浏览、恢复、分支、删除和管理所有 session。
- **Agent 编排面板**：可视化 multi-agent、delegate、workflow、cron、A2A dispatch 等运行状态。
- **项目上下文入口**：围绕每个 session 的 workDir 展示 AGENTS.md、skills、memory、MCP、approval rules、sandbox 状态。
- **核心能力 GUI 化**：把 TUI/CLI 已有能力用 Web UI 组织起来，而不是创造一套平行能力。
- **核心能力开关中心**：mode、delegate、multi-agent、workflow、thinking、sandbox、webSearch、browser、A2A master 等能力都应能在 serve 中查看和切换。
- **配置管理中心**：settings/serve/mcp/memory/approval 等配置应以结构化 UI 管理，JSON 只作为高级入口。
- **统一服务入口**：OpenAI-compatible API、Web UI、channels、logs、settings、memory、cron 都挂在同一个 serve runtime 上。

## 3. 设计原则

| 原则 | 含义 |
|------|------|
| Session-first | session 是用户工作的主对象；workDir、model、mode、history、branch 都应围绕 session 展示 |
| `sessions.db` 是事实来源 | Web UI 的 session 列表默认来自统一 SQLite 数据库，而不是 active session pool |
| workDir 属于 session | serve config 可以提供默认 workDir，但不应覆盖已存在 session 的 cwd |
| active 只是运行态标记 | active pool 用来标识当前内存中正在运行/可复用的 session，不决定历史列表范围 |
| 不复制 Agent 核心 | serve 调用现有 agent loop、provider factory、tools、sandbox、skills、session，不另写专用逻辑 |
| 编排要可见 | sub-agent、workflow、cron、tool progress、logs 应可追踪、可取消、可回放 |
| 能力开关显式化 | CLI flag、slash command 和请求扩展背后的核心能力要在 Web UI/API 中有结构化状态 |
| Settings 产品化 | Web UI 应提供面向任务的配置界面，而不是简单展示 JSON |
| 安全边界独立于默认路径 | `allowedWorkDirs`、sandbox、auth token 控制权限；默认 workDir 只是新 session 的便利值 |

## 4. WorkDir / Session 语义

### 4.1 配置层

`serve.json` 中的 workDir 应理解为：

- 新建 session 时的默认工作目录。
- Web UI 初次打开时的建议目录。
- API request 未显式提供 `x_working_dir` 且没有 session 时的 fallback。

它不应该表示：

- serve 只能管理这个目录下的 session。
- 旧 session 恢复时要被强行切到这个目录。
- Web UI session 列表只能显示这个目录。

### 4.2 Session 层

每个 session 应有自己的工作目录：

- 真实来源：`sessions.db.sessions.cwd`。
- Web UI 列表展示：每行显示 session 的 workDir。
- 打开 session 后：Chat 页顶部显示该 session 的 workDir。
- 发送后续消息：沿用该 session 的 workDir。

建议规则：

1. 新建 session 时必须明确 workDir，除非 serve 有默认 workDir。
2. 已存在 session 的 workDir 默认不可变。
3. 如果用户想换 workDir，应该创建新 session 或 fork，而不是静默改写原 session。
4. 如果 API 请求同时带 `x_session_id` 和不同的 `x_working_dir`，应返回明确错误或提供显式 fork/change-workdir 动作。

### 4.3 Multi-agent / Workflow 层

父 session 的 workDir 是默认上下文：

- sub-agent 默认继承父 session workDir。
- workflow worker 默认继承 workflow 所属 session workDir。
- 特殊任务可以显式指定 worker workDir，但应该在运行记录中可见。
- child agent 的运行结果应能回到父 session 的编排视图中，而不是变成不可追踪后台状态。

## 5. Session 列表与 `sessions.db`

### 5.1 当前问题

如果 Web UI 的 `/api/sessions` 只来自 active session pool，或只补当前 serve workDir 的历史 session，会出现几个问题：

- 历史 session 不完整。
- 不同项目的 session 在 Web UI 中不可见。
- 打开 serve 后像是“丢了会话”，但其实 session 还在 `sessions.db`。
- serve 被误解为单项目服务，而不是 session 工作台。

### 5.2 期望行为

`GET /api/sessions` 默认返回 `sessions.db` 中的全部 session，并合并 active runtime 状态。

建议返回字段：

```jsonc
{
  "sessions": [
    {
      "id": "session-id",
      "title": "可选标题或首条用户消息摘要",
      "workDir": "/path/to/project",
      "lastUsed": "2026-07-07T12:00:00Z",
      "messageCount": 42,
      "preview": "首条或最近用户消息摘要",
      "active": true,
      "mode": "agent",
      "delegateMode": false,
      "multiAgent": true,
      "workflows": true,
      "webSearch": false,
      "browser": false,
      "a2aMaster": false,
      "parentSession": ""
    }
  ]
}
```

active pool 仍有价值，但它应该只是状态增强：

- `active=true/false`
- 当前 mode/model override
- 是否有 running request
- 当前 tool/workflow/sub-agent 状态
- 当前 session 的 delegate、multi-agent、workflow、webSearch、browser、a2aMaster 等运行态能力值

必要时可以新增：

- `GET /api/sessions?scope=all|active`
- `GET /api/sessions/active`
- `GET /api/workdirs`

## 6. 核心能力开关

serve 应该拥有 TUI/CLI 里的所有核心能力，但这些能力不应该只以启动参数或 slash command 的形式存在。Web UI 需要能查看、切换和持久化这些能力状态，API 也应该提供结构化入口。

### 6.1 开关层级

建议分三层：

| 层级 | 作用 |
|------|------|
| Serve 默认值 | 写在 `serve.json`，作为新 session 的默认能力状态 |
| Session 状态 | 写入 session 元数据或 session 事件，是恢复 session 时的事实来源 |
| Run 覆盖 | 单次请求临时覆盖 mode/model/thinking 等，必须在运行记录中可见 |

也就是说，`mothx serve --delegate --multi-agent` 这类参数可以继续作为启动默认值存在，但它们不应该成为唯一入口。进入 Web UI 后，用户应该能看到当前 session 是否启用了 delegate/multi-agent，并能在策略允许时切换。

### 6.1.1 当前过渡实现状态

截至 2026-07-08，已经落地了一组过渡能力，用来先解决 Web UI 中“每个 session 可勾选工具”的需求：

- `mothx serve` 支持 `--web-search`、`--browser`、`--enable-a2a-master`，这些参数作为新 session 的默认能力状态。
- `serve.json` 的扁平配置支持 `webSearch`、`browser`、`a2aMaster`，并映射到 API runtime 和 channels runtime。
- `/api/status` 的 `features` 返回 `delegate`、`multiAgent`、`webSearch`、`browser`、`a2aMaster` 等默认能力状态，供 Web UI 初始化使用。
- `/api/sessions` 的 active session 信息返回 `delegateMode`、`multiAgent`、`webSearch`、`browser`、`a2aMaster` 等当前运行态值。
- `/api/sessions?scope=all|active` 和 `/api/sessions/active` 已支持区分全量历史视图和 active runtime 视图。
- `GET /api/capabilities` 已返回 serve 默认 capability 状态；`GET/PATCH /api/sessions/{id}/capabilities` 已支持读取和更新 session runtime capabilities。
- session capability 已通过 `sessions.db.session_capabilities` 持久化；PATCH、`x_tools` 同步、`/mode` 和 `/delegate` 会保存当前能力状态，历史 session 恢复时会加载这些状态。
- Chat 输入框下方的 model picker 右侧已经提供 `webSearch`、`browser`、`a2aMaster`、`delegate`、`multi-agent` 勾选项。
- Web UI 已在已有 session 切换工具勾选时调用 session capability PATCH；新建 session 在拿到真实 session ID 前仍使用 localStorage 临时保存勾选状态。
- `/v1/chat/completions` 支持 VibeCoding 扩展字段 `x_tools`，用于把 Web UI 的 session 工具选择发送给后端。
- 后端会把 `x_tools` 应用到当前 `APISession`，并动态注册或移除 browser、A2A dispatch、delegate、sub-agent 工具；webSearch 通过 per-session settings 副本注入 hosted tool。
- `/v1/chat/completions` 对显式 `x_session_id + x_working_dir` 做 cwd 冲突检测；如果请求目录与已存在 session 的真实 cwd 不一致，返回 409。

这不是最终 capability model。当前仍缺少：

- capability change event log。
- capability change 事件、transcript 状态提示和审计记录。
- 对运行中 request 的切换策略和冲突提示。
- 安全 policy 对 browser/web/network/A2A 等能力的细粒度锁定原因。

### 6.2 建议纳入的核心能力

| 能力 | 建议语义 | UI 表达 |
|------|----------|---------|
| Mode | `plan` / `agent` / `yolo` 是 session 级状态，影响 tool policy 和 prompt | Chat 顶部 segmented control |
| Model | session 默认模型，可被单次 run 覆盖 | Chat 顶部 model picker |
| Thinking | 思考强度或格式偏好，属于当前 session 的推理配置 | Chat 顶部下拉或 slider |
| Delegate | 开启后注册 delegate/subtask 相关工具 | Session 开关 + Agent Board 状态 |
| Multi-agent | 开启后注册 sub-agent 管理工具 | Session 开关 + Agent Board |
| Workflow | 开启后注册 workflow 工具并展示 run 状态 | Session 开关 + Workflow 面板 |
| Web Search | 开启后把 provider hosted web search tool 注入当前 run | Chat 工具勾选 + Settings 能力默认值 |
| Browser | 开启后注册 browser tool，并确保 browser skill 进入当前 session 上下文 | Chat 工具勾选 + Project Context 状态 |
| A2A Master | 开启后注册 `a2a_dispatch`，从项目或全局 `a2a-list.json` 加载 remote agents | Chat 工具勾选 + Agent Board |
| Sandbox | 受 serve policy 约束，可显示当前安全状态 | 只读状态或受限开关 |
| Memory | 是否启用项目/全局 memory 输入 | Project 面板开关 |
| MCP | 当前 workDir 可用 MCP server 集合 | Project 面板状态 |
| Channels/Cron | serve 级能力，和 session 运行态关联展示 | Agent Board / Settings |

### 6.3 切换语义

切换 mode、delegate、multi-agent、workflow、webSearch、browser、a2aMaster 这类能力会改变 system prompt、tool registry 或执行策略，因此不能只是改 UI 状态。建议规则：

1. 如果 session 当前有运行中的 request，不允许直接切换，或者要求先取消运行。
2. 切换后重建该 session 的 agent runtime。
3. 写入一条 session 事件，记录旧值、新值、操作者和时间。
4. Chat transcript 中显示一条轻量状态提示，避免用户不清楚上下文为何变化。
5. 已完成的历史消息不重写，只影响后续 run。

### 6.4 结构化 API

Web UI 应优先调用结构化 API，而不是向聊天框注入 slash command。slash command 可以继续保留，作为 API/TUI 的快捷入口。

当前过渡实现已经在 `/v1/chat/completions` 请求体中加入 `x_tools`：

```jsonc
{
  "x_session_id": "session-id",
  "x_tools": {
    "webSearch": true,
    "browser": true,
    "a2aMaster": false,
    "delegate": true,
    "multiAgent": true
  },
  "messages": [
    { "role": "user", "content": "..." }
  ]
}
```

`x_tools` 的语义是“把当前 Web UI 中该 session 的工具选择同步到本次 session runtime”。它适合短期落地，但不应替代正式 capability API，因为它和 chat run 绑定，缺少独立读取、patch、审计和持久化语义。

建议新增：

```text
GET   /api/capabilities
GET   /api/sessions/{id}/capabilities
PATCH /api/sessions/{id}/capabilities
```

`GET /api/capabilities` 返回 serve 当前可用能力和 policy：

```jsonc
{
  "modes": ["plan", "agent", "yolo"],
  "features": {
    "delegate": { "available": true, "default": false },
    "multiAgent": { "available": true, "default": false },
    "workflows": { "available": true, "default": true },
    "sandbox": { "available": true, "locked": true, "value": true }
  }
}
```

`PATCH /api/sessions/{id}/capabilities` 示例：

```jsonc
{
  "mode": "agent",
  "delegateMode": true,
  "multiAgent": true,
  "workflows": true,
  "webSearch": true,
  "browser": true,
  "a2aMaster": false,
  "thinkingLevel": "medium",
  "model": "deepseek-v4-flash"
}
```

### 6.5 持久化建议

建议把能力状态分清楚：

- `serve.json` 保存默认值和安全策略。
- `sessions.db` 保存 session 当前能力状态；后续可以追加 capability change event 用于审计和回放。
- active runtime 只保存内存中的 agent 实例、运行中 request 和 tool/workflow 状态。
- Web UI localStorage 只能作为未持久化 capability API 前的过渡缓存，不能作为长期事实来源。

这样做的好处是：重启 serve 后，历史 session 的 mode/delegate/multi-agent/workflow/webSearch/browser/a2aMaster 状态仍可恢复；同时 serve 的启动参数只影响新 session，不会意外改写已有 session。

### 6.6 安全策略

有些开关不是普通偏好，而是安全能力：

- `yolo` mode
- sandbox off
- 任意 workDir 访问
- browser/web/network 工具
- 外部 channel 自动执行

这些能力应该受 serve policy 控制。Web UI 可以展示禁用开关和原因，例如“被 serve policy 禁止”或“当前 workDir 不在 allowedWorkDirs 内”，而不是静默隐藏。

## 7. Web UI 信息架构

### 7.1 Sessions

目标：像“项目/任务工作台”一样管理所有 session。

能力：

- 全量 session 列表，来自 `sessions.db`。
- 按 workDir、标题、ID、模式、时间过滤。
- 显示 active/running 状态。
- 打开、删除、fork、重命名、打标签。
- 支持树状/分支视图，展示 parent/child session。

### 7.2 Chat

目标：围绕单个 session 的工作流。

能力：

- 打开历史 session 后恢复 transcript。
- 顶部固定展示 workDir、mode、model、thinking、context usage。
- workDir mismatch 时给出明确提示。
- 支持“从当前 session fork 到新 workDir”。
- 工具调用、diff、错误、logs 和 context pressure 可折叠查看。
- 顶部或 composer 工具栏提供 mode/model/thinking/delegate/multi-agent/workflow/webSearch/browser/a2aMaster 等能力开关。
- 当前已在输入框下方的 model picker 右侧提供 `webSearch`、`browser`、`a2aMaster`、`delegate`、`multi-agent` 勾选项，作为正式 capability bar 的第一步。

### 7.3 Agent Board

目标：让 multi-agent 不再只是文本输出，而是可观察的运行图。

能力：

- 当前 session 下的 sub-agents 列表。
- delegate 任务状态。
- workflow run DAG / phase / worker 状态。
- cron job 运行记录。
- A2A dispatch 状态。
- 可取消、可展开日志、可查看最终结果。

### 7.4 Project / WorkDir 面板

目标：围绕 workDir 管理上下文输入。

能力：

- AGENTS.md / CLAUDE.md / CURSOR.md 发现状态。
- 项目 skills。
- 项目 MCP 配置。
- 项目 memory。
- approval whitelist/blacklist。
- sandbox 能力检测。

### 7.5 Settings

目标：把配置变成可理解、可验证、可回滚的 UI，而不是 JSON 编辑器。

原则：

- 默认展示结构化表单、开关、选择器、列表和状态说明。
- JSON 视图只作为高级模式，用于导入、导出、排障和无法覆盖的新字段。
- UI 写配置时必须按配置域 patch，避免保存稀疏对象导致默认值展开或覆盖用户未改字段。
- 全局配置、项目配置和 session capability 要分层展示，不能混在同一个 JSON 面板里。
- 高风险设置必须显示影响范围，例如是否影响已有 session、是否需要重建 runtime、是否需要重启 serve。

建议拆成这些页面：

| 页面 | 管理对象 | 典型控件 |
|------|----------|----------|
| Providers & Models | provider、model、默认模型、compat flags | 表格、model picker、连接测试 |
| Serve Defaults | defaultWorkDir、默认 mode、默认 capabilities、监听地址 | 输入框、目录选择、开关 |
| Security | auth token、allowedWorkDirs、sandbox policy、yolo policy | 列表、受限开关、状态提示 |
| Capabilities | delegate、multi-agent、workflow、webSearch、browser、A2A、channels、cron | 开关、状态、依赖检查 |
| Project Context | AGENTS.md、skills、memory、MCP、approval rules | 文件状态、编辑器、列表 |
| Channels | WebSocket、webhook、Feishu/WeChat 等 channel | 表单、启停、连接状态 |
| Advanced JSON | 原始 JSON 查看/导入/导出 | 只在高级模式显示 |

Settings UI 不应该把配置文件名作为用户的主导航。用户关心的是“模型”“安全”“能力”“项目上下文”，不是 `settings.json` 或 `serve.json` 的内部结构。

## 8. API 演进建议

### 8.1 Session API

```text
GET    /api/sessions
GET    /api/sessions?scope=active
GET    /api/sessions/{id}
GET    /api/sessions/{id}/messages
POST   /api/sessions
PATCH  /api/sessions/{id}
POST   /api/sessions/{id}/fork
DELETE /api/sessions/{id}
```

`POST /api/sessions` 示例：

```jsonc
{
  "workDir": "/path/to/project",
  "mode": "agent",
  "model": "deepseek-v4-flash",
  "title": "修复 serve session 列表"
}
```

### 8.2 Run / Chat API

OpenAI-compatible `/v1/chat/completions` 继续保留，但语义需要清晰：

- `x_session_id` 为空：按 workDir 创建或复用默认 session。
- `x_session_id` 非空：以 session 的 cwd 为准。
- `x_working_dir` 只用于创建新 session，或必须与 session cwd 一致。
- 若 `x_session_id` 与 `x_working_dir` 冲突，优先返回 409，避免静默污染 session。
- mode/model/thinking 等单次覆盖可以通过明确 header 或 request metadata 传入，但需要记录在 run 事件中。
- `x_tools` 当前用于同步 Web UI 的 session 工具选择，字段包括 `webSearch`、`browser`、`a2aMaster`、`delegate`、`multiAgent`；长期应收敛到 session capability API。

### 8.3 Orchestration API

可以在后续阶段抽象：

```text
GET    /api/sessions/{id}/agents
GET    /api/sessions/{id}/workflows
POST   /api/sessions/{id}/workflows
POST   /api/sessions/{id}/workflows/{runID}/cancel
GET    /api/sessions/{id}/tools/events
```

### 8.4 Settings API

Settings UI 需要结构化 API，而不是直接读写完整 JSON 文件。

建议新增：

```text
GET   /api/settings/summary
GET   /api/settings/providers
PATCH /api/settings/providers/{name}
GET   /api/settings/serve
PATCH /api/settings/serve
GET   /api/settings/security
PATCH /api/settings/security
GET   /api/settings/projects/{workDir}
PATCH /api/settings/projects/{workDir}
POST  /api/settings/validate
```

写入规则：

- 只 patch 被用户修改的配置域。
- 返回 validation errors 和 warnings，不静默保存危险配置。
- 明确标记配置来源：global、project、session、runtime。
- 写入后返回是否需要重建 session runtime 或重启 serve。
- Advanced JSON 可以调用同一套 validate/patch 逻辑，不能绕过校验直接覆盖文件。

## 9. 分阶段路线

### Phase 0：语义确认

- 明确 serve 定位为 session workbench。
- 明确 workDir 属于 session。
- 明确 `sessions.db` 是 session 列表事实来源。
- 明确 active pool 只是运行态状态。
- 明确 mode/delegate/multi-agent/workflow/webSearch/browser/a2aMaster 是 session 能力开关，而不是只属于启动参数。
- 已完成：serve flag/config 可以作为新 session 默认值；Web UI 已能在当前 session 上勾选部分工具能力。

### Phase 1：全量 Session Browser

- `/api/sessions` 从 `sessions.db` 返回全部 session。
- 合并 active session 状态。
- 支持 `scope=all|active` 和 `/api/sessions/active`，把历史列表和 active runtime 列表明确分层。
- Web UI 支持跨 workDir 过滤和打开历史 session。
- Chat 打开历史 session 时使用 session 自带 workDir。

### Phase 2：Session 创建 / Fork / WorkDir 不可变

- Web UI 新建 session 显式选择 workDir。
- 已存在 session 不允许静默改 workDir。
- 提供 fork 到新 workDir 的明确动作。
- API 对 `x_session_id + x_working_dir` 冲突返回明确错误。
- 已完成：chat completions 中显式传入冲突的 `x_session_id + x_working_dir` 时返回 409。

### Phase 3：核心能力开关模型

- 新增 capability model，覆盖 mode、model、thinking、delegate、multi-agent、workflow、webSearch、browser、a2aMaster、sandbox 状态。
- Web UI 在 Chat 顶部或 composer 工具栏展示可切换能力。
- 新增结构化 capability API，避免 Web UI 依赖 slash command。
- 切换会影响 tool registry/prompt 的能力时，重建 session runtime 并记录事件。
- 已完成过渡版：`x_tools` 可同步 `webSearch`、`browser`、`a2aMaster`、`delegate`、`multiAgent` 到 active `APISession`。
- 已完成结构化入口：`GET /api/capabilities`、`GET/PATCH /api/sessions/{id}/capabilities`。
- 已完成当前状态持久化：`sessions.db.session_capabilities` 保存每个 session 的最新能力状态。
- 待完成审计版：增加 capability change event log，并补运行中切换策略。

### Phase 4：Session Detail 工作台

- 显示 session 元数据、消息数、preview、labels、branch。
- 显示 context usage、last model、mode、thinking。
- 显示工具事件和错误摘要。

### Phase 5：Multi-agent / Workflow 可视化

- 显示当前 session 的 sub-agent / delegate / workflow run。
- 支持取消运行中的 workflow/sub-agent。
- 支持查看 worker 输出和最终结果。

### Phase 6：Settings UI

- 把 Settings 从 JSON 展示改成结构化页面。
- 优先实现 Providers & Models、Serve Defaults、Security、Capabilities。
- 所有写入使用 patch API，避免覆盖未修改字段。
- Advanced JSON 只作为高级入口，并接入 validate。

### Phase 7：Project Context 面板

- 按 workDir 展示 AGENTS.md、skills、MCP、memory、approval、sandbox。
- 支持在 Web UI 中编辑项目级配置，但避免破坏 `settings.json` 现有语义。

### Phase 8：编排模板与团队工作台

- 保存常用 workflow 模板。
- 为 session 添加标签、负责人、状态。
- 支持团队共享的只读 session 浏览。
- 引入更细粒度的 auth / 权限边界。

## 10. 非目标

- 不在 serve 中重新实现 agent loop。
- 不把 Web UI 变成完整 IDE。
- 不在第一阶段重写 session schema。
- 不把所有 UI 状态写进 `settings.json`。
- 不让 serve config 的默认 workDir 成为全局 session 过滤器。
- 不把 Settings 做成简单 JSON viewer/editor。

## 11. 待讨论问题

1. 已存在 session 的 workDir 是否绝对不可变？还是允许显式迁移？
2. `x_session_id` 与 `x_working_dir` 冲突时，是返回 409，还是忽略 `x_working_dir` 并给 warning？
3. Web UI 默认展示全部 session，是否需要按 `allowedWorkDirs` 过滤？
4. 多用户 serve 场景下，session 列表是否需要 owner / namespace？
5. channels 创建的 per-user session 是否和普通 API/TUI session 在同一列表展示？
6. sub-agent 是否应该拥有独立可见 session，还是只作为父 session 的运行事件？
7. workflow worker 的结果是否需要持久化为独立消息、运行记录，还是只保存在 workflow store？
8. `serve.json` 中是否应该保留 `workDir`，还是改名为 `defaultWorkDir` 来减少误解？
9. Web UI 是否应该提供 `/api/sessions/active`，把 active runtime 与历史 session 明确分层？
10. session title 应该来自首条用户消息、显式重命名，还是由模型摘要生成？
11. mode/delegate/multi-agent/workflow/webSearch/browser/a2aMaster 的当前值是存在 session metadata，还是通过事件流计算？
12. multi-agent、delegate、webSearch、browser、a2aMaster 是 session 级开关，还是 serve 级全局开关加 session opt-in？
13. Web UI 切换能力时，是否允许用 slash command 作为兼容层，还是必须走结构化 API？
14. 运行中 request 存在时，切换 mode 或工具集应该禁止、排队，还是自动取消当前 run？
15. `yolo`、sandbox off、外部 channel 自动执行等高风险能力的 policy 边界放在 `serve.json` 还是独立 security 配置？
16. Settings UI 的 Advanced JSON 是否允许直接保存，还是只能导出/导入并经过 validate？
17. 配置写入时如何展示 global/project/session 三层覆盖关系？
18. 哪些配置变更需要重建 session runtime，哪些需要重启 serve？

## 12. 建议先落地的决策

为了尽快消除当前 Web UI 的认知偏差，建议先做以下小步：

1. `GET /api/sessions` 改为读取 `sessions.db` 全量 session。
2. active pool 只用于补充 `active`、`mode`、`delegateMode`、`workflows`、`webSearch`、`browser`、`a2aMaster` 等运行态字段。
3. Chat 页打开历史 session 后使用该 session 的 workDir 展示和发送。
4. 新建 session 继续要求用户选择 workDir，除非 serve config 提供默认值。
5. 暂不实现 workDir 迁移；需要换目录时先新建或 fork session。
6. 把 mode/delegate/multi-agent/workflow/webSearch/browser/a2aMaster 定义成 session capability model 的第一批字段。
7. 短期保留 `x_tools` 作为 chat run 兼容同步；Web UI 对已有 session 使用结构化 capability API 切换能力，slash command 只作为人类快捷入口保留。
8. serve 启动参数和 `serve.json` 只提供新 session 默认值，不静默改写已有 session。
9. Settings 先做结构化 UI，不把 JSON viewer 当作正式配置界面。
10. Settings 写入必须使用按域 patch 和 validate，避免覆盖用户未修改字段。

这一步完成后，serve 的心智模型会从“当前目录下的 Web chat”切换到“跨项目 session 工作台”，后续 multi-agent/workflow 可视化才有稳定基础。
