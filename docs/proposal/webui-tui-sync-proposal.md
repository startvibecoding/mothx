# WebUI 同步 TUI 能力方案设计

> 状态: Proposal
> 日期: 2026-07-18
> 目标版本: 待定

## 1. 概述

当前 WebUI 已经复用了统一的 serve / agent / session / tool / sandbox / skills / provider 运行栈，但交互层仍然偏“聊天页 + 设置页 + 事件展示”，与 TUI 的能力存在明显差距，尤其是在以下两类体验上：

1. **mode 语义与切换**：TUI 里用户可以清晰切换 `plan / agent / yolo`，并根据模式得到不同的工具边界、提示和交互反馈；WebUI 目前更多是把 `mode` 当作会话字段和请求参数使用，缺少统一的可视化控制面板。
2. **权限申请与确认**：TUI 有完整的 approval 队列、确认弹层、记忆规则（一次批准 / 拒绝 / 记住命令 / 记住前缀）；WebUI 目前主要依赖消息流和按钮式工具开关，没有把“审批事件”做成一等交互对象。

本提案目标是：让 WebUI 在不改变后端核心执行语义的前提下，逐步同步 TUI 的关键能力，使两端共享同一套运行状态、权限模型与事件协议。这里的“同步”指功能语义和后端协议统一，不要求 WebUI 复刻 TUI 的视觉样式；WebUI 应继续保持现有的 Web 产品形态、布局语言和交互风格。

## 2. 现状梳理

### 2.1 WebUI 当前实现方式

WebUI 位于 `ui/`，是一个 Svelte SPA：

- 入口：`ui/src/App.svelte`
- 聊天页：`ui/src/views/Chat.svelte`
- 设置页：`ui/src/views/Settings.svelte`
- 全局状态：`ui/src/lib/stores.js`
- API 封装：`ui/src/lib/api.js`

它通过以下接口与后端交互：

- `POST /v1/chat/completions`：主聊天入口
- `GET /api/sessions/*`：会话、消息、tool result、sub-agent、run event、capability event
- `GET /api/status`、`/api/channels`、`/api/settings`、`/api/serve/config` 等：全局状态
- `GET /v1/models`：模型列表
- SSE session stream：用于 transcript / run_event / capability_event 的实时同步

WebUI 已经支持一些扩展字段：

- `x_session_id`
- `x_mode`
- `x_working_dir`
- `x_tools`
- `x_transcript`

也就是说，WebUI 目前是“通过扩展请求参数驱动后端”，而不是“前端完整管理运行态”。

### 2.2 后端当前实现方式

WebUI 复用的是 `internal/serve/openaiapi`：

- `handler_chat.go`：处理聊天、模式、工具、运行事件
- `commands.go`：slash command 处理
- `session_mgr.go`：会话状态、能力持久化、消息视图
- `session_stream.go`：SSE 事件流
- `events.go`：run / capability 事件记录

当前后端已经保存并公开了不少运行状态：

- `SessionCapabilities`
- `SessionCapabilityEvent`
- `SessionRunEvent`
- transcript 事件
- sub-agent 信息

这说明 WebUI 和 TUI 的统一基础已经存在，缺的是“同一套交互协议的前端实现”。

### 2.3 TUI 当前做法

TUI 里 mode / approval / delegate / multi-agent 已经是完整状态机；`question` 也是 TUI/ACP 的交互能力，但当前 serve API/channel runtime 不注册也不暴露 `question`，因此 WebUI 同步方案第一版不把 `question` 作为默认目标：

- `internal/tui/commands.go`：`/mode`、`/delegate`、`/browser`、`/alloweditpath`、`/allowautoedit`
- `internal/tui/approval.go`：approval queue、批准/拒绝/记忆规则
- `internal/tui/agent_events.go`：agent 事件驱动 UI 状态变化
- `internal/tui/app.go`：模式、输入、会话、sub-agent、上下文和各种弹层状态

因此，WebUI 最合理的做法不是重新发明一套逻辑，而是把 TUI 的交互语义映射到 WebUI 的组件与状态管理中。映射时只对齐功能和状态语义，不迁移 TUI 的视觉表现；组件形态应服从 WebUI 当前设计语言。

## 3. 设计目标

| 目标 | 说明 |
|------|------|
| 语义一致 | WebUI 的 mode、approval、capability 语义与 TUI 保持一致 |
| 风格独立 | 保持 WebUI 现有视觉风格和 Web 交互习惯，不按 TUI 样式复刻界面 |
| 统一状态 | 运行态状态来源于后端 session / event，不在前端重复造轮子 |
| 交互补齐 | WebUI 能展示并处理与 TUI 等价的模式切换、权限申请、tool 状态 |
| 可渐进落地 | 先补展示，再补交互，再补策略控制，避免一次性大改 |
| 不改 schema | 不修改现有 settings.json / serve.json 的字段语义 |
| 保持复用 | 继续复用 `internal/serve/openaiapi` 的事件、命令与会话体系 |

## 4. 非目标

- 不在第一阶段重写整个 WebUI。
- 不要求 WebUI 完全复刻 TUI 的视觉样式；本方案追求功能、协议和状态语义统一，而不是界面长得一样。
- 不推翻 WebUI 现有的视觉体系、布局风格、组件质感和 Web 交互习惯。新增能力应自然融入当前 WebUI。
- 不改变 agent 核心执行逻辑，只补前端交互与事件协议。
- 不把所有 TUI 专属快捷键原样搬到 WebUI。
- 不引入第二套独立的权限引擎。
- 不在第一版把 TUI 的 `question` 工具直接暴露到 WebUI；如需支持，应单独定义 serve/WebUI 下的安全模型与交互协议。

## 5. 关键差距

### 5.1 mode 差距

TUI 里 mode 是明确的用户操作入口，WebUI 里目前更像是“请求里传一个值”。

需要补齐：

- 当前 session mode 的可见展示
- mode 切换入口
- mode 切换后的运行策略说明
- mode 与能力开关的关系提示

建议 UI 明确显示：

- `plan`：偏只读规划
- `agent`：常规执行
- `yolo`：自动执行

### 5.2 权限申请差距

TUI 里审批是一个独立交互对象，WebUI 目前还没有真正的审批中心。

需要补齐：

- 申请到来的弹窗/面板
- 审批队列
- 一次批准 / 拒绝 / 记忆规则
- 审批历史
- 审批结果回写到 session stream

### 5.3 工具与能力展示差距

TUI 会把当前可用能力、delegate / multi-agent / workflows / browser 等状态体现在运行中；WebUI 目前只有部分 summary。

需要补齐：

- 当前 session capabilities 面板
- 当前工具开关面板
- 运行中 tool 状态条
- sub-agent / delegate / workflow 的状态摘要

## 6. 方案设计

### 6.1 统一“运行态协议”

先把 WebUI 和 TUI 共同关注、且 serve runtime 已适合承载的概念抽成统一协议：

- session runtime config，尤其是 `mode`
- capability snapshot / capability patch
- approval request
- approval response
- tool call / tool result
- run event
- transcript event
- sub-agent status

后端继续负责生成这些事件，前端只负责订阅和渲染。

需要注意：`mode` 不应被简单混同为 capability。`mode` 是 session runtime config 的一部分；capability 是工具和功能可用性的状态。mode 变更可以影响 capability 的最终计算结果，但 WebUI 应分别展示和提交二者，并以服务端返回的 runtime snapshot / capability event 作为最终状态。

落地时需要兼容现状：当前 `internal/serve/openaiapi` 的 capability event 已经会记录 `mode` 变化。第一阶段不要求立刻大改内部事件命名或迁移历史数据；建议新增对外稳定的 runtime snapshot / patch 协议作为 WebUI 标准入口，同时保留现有 capability event 作为兼容增量事件。后续可以再把内部 `capabilitySnapshot` 等命名渐进清理为更准确的 runtime/capability 分层。

### 6.2 WebUI 新增“运行状态与控制入口”

在 Chat 体验中引入轻量 runtime controls，用于承载运行状态展示和必要操作。具体形态应服从 WebUI 现有设计语言，可以落在顶部栏、侧边栏、抽屉、Popover、设置面板或组合布局中，不强制做成固定状态区，也不要求模仿 TUI。

- 当前 session ID
- 当前 mode
- 当前 model
- 当前 workdir
- 当前可用 capability
- 当前工具开关
- 当前 pending approval 数量
- 当前 sub-agent / delegate / workflow 状态

该入口建议覆盖：

- 切换 mode
- 打开/关闭 capability
- 打开审批弹窗
- 查看事件历史

### 6.2.1 Session runtime snapshot / patch API

为了避免 WebUI 首屏依赖 SSE replay 拼状态，后端应提供一个结构化 runtime snapshot 作为单一读取入口：

```http
GET /api/sessions/{id}/runtime
```

返回示例：

```jsonc
{
  "sessionId": "sess-1",
  "mode": "agent",
  "model": "gpt-5.5",
  "workDir": "/home/free/src/startvibecoding/vibecoding",
  "capabilities": {
    "browser": {
      "available": true,
      "enabled": false,
      "effective": false,
      "disabledReason": "disabled for this session"
    },
    "delegate": {
      "available": true,
      "enabled": true,
      "effective": true,
      "disabledReason": ""
    },
    "multiAgent": {
      "available": false,
      "enabled": false,
      "effective": false,
      "disabledReason": "disabled by serve config"
    }
  },
  "pendingApprovals": [],
  "activeRun": {
    "runId": "run_xxx",
    "status": "running"
  }
}
```

WebUI 控件修改 runtime 时使用结构化 patch：

```http
PATCH /api/sessions/{id}/runtime
```

请求示例：

```jsonc
{
  "mode": "yolo",
  "capabilities": {
    "browser": true,
    "delegate": true
  }
}
```

后端返回更新后的 runtime snapshot，并发布对应 runtime / capability 增量事件。首屏与刷新恢复优先使用 `GET /runtime`，SSE 只负责后续增量更新；event replay 主要用于历史展示、审计或断线补偿，不作为 WebUI 主状态恢复路径。

### 6.3 审批中心与详情展示

审批中心建议分两步走：先把审批内容展示清楚，再把 pending / resume 的执行语义接完整。

### 6.3.1 前端详细展示规范

WebUI 的审批弹窗不应只显示“是否批准”，而应把审批内容拆成“摘要层 + 详情层 + 原始数据层”，方便用户快速判断风险。

建议默认展示以下信息：

- 操作类型：`bash` / `write` / `edit` / `delete` / `network` / `delegate`
- 风险等级：`low` / `medium` / `high` / `critical`
- 当前 mode：`plan` / `agent` / `yolo`
- 来源：session ID、agent ID、tool call ID
- 一句话摘要：例如“编辑 internal/serve/openaiapi/handler_chat.go”或“执行 bash: go test ./...”
- 触发原因：例如命中黑名单、超出白名单、需要写文件确认等

展开详情后再显示：

- 工具名
- 完整参数 JSON
- 命中的路径 / 命令 / URL
- 是否命中 allow 规则
- 预计影响范围
- 相关上下文片段
- 可选动作说明

不同工具建议重点展示：

- `bash`：完整命令、工作目录、是否可能修改文件、是否命中黑白名单、危险原因
- `write` / `edit`：目标路径、新建或修改、预估变更范围、是否命中 `alloweditpath`、是否触发 `confirmBeforeWrite`
- `delegate` / `sub-agent`：子任务标题、执行模式、任务范围、预计工具权限、是否阻塞主任务

建议 UI 使用三层折叠：

1. 一眼看懂的摘要
2. 可展开的结构化详情
3. 可复制的 raw JSON

这样既能保证信息密度，又不会让审批界面过于拥挤。

### 6.3.2 审批事件字段设计

为了让前端能完整展示审批内容，后端推送的 `approval_request` 事件建议包含结构化字段，而不是只传 `toolName` 和 `args`。

建议 payload 形态如下：

```jsonc
{
  "type": "approval_request",
  "approvalId": "approval-123",
  "sessionId": "sess-1",
  "agentId": "agent-2",
  "mode": "agent",
  "risk": "high",
  "summary": "Edit internal/serve/openaiapi/handler_chat.go",
  "reason": "file outside whitelist",
  "tool": {
    "name": "edit",
    "label": "Edit file",
    "args": {
      "path": "internal/serve/openaiapi/handler_chat.go",
      "content": "..."
    },
    "details": {
      "path": "internal/serve/openaiapi/handler_chat.go",
      "operation": "modify",
      "matchesAllowPath": false,
      "confirmBeforeWrite": true
    }
  },
  "context": {
    "workDir": "/home/free/src/startvibecoding/vibecoding",
    "preview": "..."
  },
  "actions": ["approve_once", "deny_once", "remember_command", "remember_prefix", "allow_edit_path"]
}
```

回传结果也建议保持结构化，便于后端审计和持久化：

```jsonc
{
  "action": "approve_once",
  "remember": false,
  "rememberType": "command",
  "rememberValue": "go test ./..."
}
```

或者：

```jsonc
{
  "action": "deny_once"
}
```

这样前端可以直接渲染成完整审批卡片，后端也能把它映射回现有的 `HandleApprovalResponse` 和 allow 规则持久化逻辑。

审批 action 必须对齐 TUI 现有语义，不引入 WebUI 专用规则。建议至少支持：一次批准、一次拒绝、记住完整命令、记住命令前缀、允许编辑路径；如果后端已有 deny 规则或 auto edit 规则，也应复用现有配置写入路径。所有“记住”类操作都必须写入现有 settings / allowlist 机制，并保留命令前缀中的有意义空格。

### 6.3.3 审批生命周期与恢复

审批不是单纯的前端弹窗，而是会影响 agent 工具执行的后端状态机。后端需要明确以下语义：

- 生成 `approval_request` 后，对应 tool call / agent execution 必须进入 pending 状态，等待用户响应后再继续或失败退出。
- pending approval 必须有稳定 ID，并能路由回具体 session、agent、tool call 和执行协程。
- 页面刷新或 SSE 断线重连后，WebUI 必须能通过 runtime snapshot 恢复当前 pending approval 队列。
- 多个浏览器客户端同时打开同一 session 时，审批结果以后端首次接受的 response 为准，其他客户端收到最终状态更新。
- 审批 response 需要记录到 session event / audit 视图，便于后续排查“为什么执行或拒绝了某个工具”。
- 需要定义取消、超时、agent 中断时 pending approval 的终态，避免悬挂请求。

### 6.3.4 协议边界与兼容性

WebUI 不直接决定工具是否可执行，也不直接修改 agent 内部权限状态。WebUI 只提交用户意图，例如切换 mode、开关 session capability、响应 approval。最终是否允许执行，由后端根据 mode、settings、serve config、sandbox、allowlist / blacklist 和 session runtime config 统一决策。

所有 runtime / capability / approval / run event 都以后端为单一事实源。前端可以做 optimistic UI，但必须以服务端 snapshot 或 event 修正最终显示。这样可以避免 WebUI、TUI、API 客户端在同一 session 上产生语义分叉。

### 6.4 将 mode 变成一等 UI 控件

建议在顶部栏或 session toolbar 中提供 mode 切换：

- `plan`
- `agent`
- `yolo`

切换时：

- 前端可以先进入 optimistic 展示，但必须等待后端确认后落成最终状态
- 调用后端 session runtime patch，而不是把 mode 写成普通 capability
- 后端记录 runtime / capability event
- 如果 mode 切换影响 agent 实例，则像 TUI 一样触发重建或重置

### 6.5 将能力开关做成显式状态

WebUI 已经有 `sessionToolOptions`，但它更像请求参数。建议升级为可见的运行态：

- webSearch
- browser
- a2aMaster
- delegate
- multiAgent
- workflows

这些能力的来源分成三层：

1. 全局 serve 配置可用性：该能力在当前部署中是否存在。
2. session 级 runtime 开关：用户是否希望在当前 session 启用。
3. 当前策略下的实际可用性：受 `mode`、sandbox、安全策略、allowlist / blacklist、工作目录等影响后，当前是否真的能执行。

WebUI 只允许切换 session 级 runtime 开关，并明确提示全局可用性和实际可用性的边界。建议 capability snapshot 对每个能力使用四个字段：

- `available`：全局是否可用。
- `enabled`：session 是否开启。
- `effective`：当前策略下是否实际生效。
- `disabledReason`：不可用或未生效的原因。

### 6.6 事件流统一到 session stream

当前 `session_stream.go` 已经支持 transcript / run_event / capability_event。建议继续扩展为“全量 snapshot + 增量事件”的模型：

- `runtime_snapshot`：全量运行态，可由 `GET /runtime` 返回，也可在 SSE 建连后立即推送一次。
- `runtime_event`：mode / model / workdir 等 runtime config 变化。
- `capability_event`：capability 的 enabled / effective 状态变化，兼容现有 capability event。
- `approval_request`：后端产生待审批请求。
- `approval_response`：用户提交审批响应。
- `approval_resolved`：后端最终接受、拒绝、过期、取消或被其他客户端处理后的终态。
- `tool_event`：工具调用、运行中、完成、失败等状态。
- `run_event`：agent run 生命周期。
- `transcript_event`：对话 transcript 增量。

这样 WebUI 首屏读取 runtime snapshot，之后通过 SSE 增量实时跟进 agent 运行。`approval_resolved` 很重要，因为用户点击 approve / deny 不代表后端一定接受该响应；例如 approval 已过期、run 已取消、另一个客户端已经处理、remember 规则写入失败，都需要以后端 resolved 结果为准。

`question_request` / `question_response` 暂不纳入第一版 WebUI 协议。当前 serve API/channel runtime 不注册也不暴露 `question`；若未来要支持，需要单独 RFC 明确安全边界、挂起/恢复语义、多客户端行为和与 TUI/ACP 的兼容关系。

### 6.7 前端 store 重构为“事件驱动”

`ui/src/lib/stores.js` 建议新增：

- `sessionCapabilities`
- `pendingApprovals`
- `activeApproval`
- `approvalHistory`
- `toolEvents`

并由 SSE 统一驱动更新。

### 6.8 保持 slash command 兼容

WebUI 输入框中遇到 `/mode`、`/delegate`、`/help` 等命令时，应尽量与 TUI 保持同义。

建议：

- 后端继续用 `commands.go` 或其抽出的 shared function 作为统一命令实现。
- WebUI 只负责发送命令和展示返回结果。
- 不在前端重复实现命令语义。
- Slash command 是兼容入口，不是 WebUI 控件的首选实现方式。
- WebUI 的按钮、开关、菜单应优先调用结构化 runtime API；后端内部复用同一业务函数，保证它与 slash command 语义一致。

## 7. 推荐实施顺序

### Phase 1：只读同步

目标：先看齐“可见状态”，并建立 WebUI 首屏状态恢复的标准入口。

工作项：

- 新增 `GET /api/sessions/{id}/runtime`
- 显示 session mode
- 显示 capability 的 available / enabled / effective 状态
- 显示 tool / sub-agent / run event 摘要
- 显示当前 approval 队列数量

收益：低风险，能快速让 WebUI 具备与 TUI 对齐的状态可见性，同时保持 WebUI 自身风格。

### Phase 2：mode 控制

目标：让 WebUI 可以像 TUI 一样显式切模式。

工作项：

- 新增 `PATCH /api/sessions/{id}/runtime`
- mode 切换控件
- mode 变更写入 session runtime config
- 后端返回 runtime snapshot / capability event
- 运行时策略同步更新

收益：mode 语义统一，也为后续验证审批在 `plan / agent / yolo` 下的行为打基础。

### Phase 3：审批中心

目标：把审批流程拆成可逐步落地的三段。

#### Phase 3A：审批可见化

工作项：

- 后端推送 approval_request / approval_resolved
- 前端新增 approval modal / queue
- 支持查看摘要、详情、raw JSON
- 审批历史进入 audit 视图

#### Phase 3B：审批执行闭环

工作项：

- 后端保存 pending approval 状态并支持刷新恢复
- 审批结果回写后端并路由回对应 tool call / agent execution
- 多客户端审批去重和终态确认

#### Phase 3C：remember 规则持久化

工作项：

- 支持 approve once / deny once / remember
- 记住完整命令、命令前缀、编辑路径等规则
- 复用现有 settings / allowlist 机制

收益：WebUI 开始真正能承接执行型任务，同时保持与 TUI 一致的规则语义。

### Phase 4：高级能力同步

目标：补齐 delegate / multi-agent / workflows 等高级交互。

工作项：

- sub-agent 列表与消息
- delegate / workflows 状态
- 审批历史与审计视图

收益：WebUI 与 TUI 功能基本对齐。

## 8. 建议的后端改造点

### 8.1 `internal/serve/openaiapi`

建议重点改造：

- `types.go`
  - 增加 approval / runtime / capability 的 WebUI 事件类型
- `session_mgr.go` / HTTP route wiring
  - 增加 `GET /api/sessions/{id}/runtime` 和 `PATCH /api/sessions/{id}/runtime`
- `session_stream.go`
  - 扩展 SSE 事件转发
- `events.go`
  - 新增事件记录维度
- `handler_chat.go`
  - 让工具执行过程可广播到 WebUI
- `commands.go`
  - 保持 mode / delegate / approval 相关命令统一实现，或抽出 shared function 供 slash command 与 runtime API 共用

### 8.2 WebUI

建议重点改造，组件命名仅表达职责，不要求采用 TUI 的视觉样式：

- `ui/src/lib/stores.js`
  - 增加运行态 store
- `ui/src/views/Chat.svelte`
  - 增加 mode / capability / approval UI
- 新增组件：
  - `ModeSwitcher.svelte`
  - `ApprovalCenter.svelte`
  - `CapabilityPanel.svelte`

## 9. 风险与约束

| 风险 | 说明 | 缓解 |
|------|------|------|
| 前后端语义分叉 | 前端自己实现一套模式/审批逻辑 | 统一由后端事件和 session 状态驱动 |
| UI 风格漂移 | 为了同步 TUI 能力而复制 TUI 的终端视觉或快捷键习惯，破坏 WebUI 现有风格 | 只同步功能语义和状态协议；新增组件沿用 WebUI 当前设计语言 |
| UI 复杂度过高 | Chat 页塞太多控制面板 | 拆分为独立组件和抽屉/弹窗 |
| 审批状态丢失 | 刷新页面后 pending approval 不见 | 依赖 runtime snapshot 恢复 pending approval，session stream / event replay 只做增量与补偿 |
| 模式切换不一致 | UI 切了但 agent 未重建 | 切换后显式触发后端重建或刷新 |
| `question` 边界不清 | serve runtime 当前不注册/暴露 `question`，直接加入 WebUI 会破坏现有安全边界 | 第一版不支持；未来单独 RFC 定义安全模型与恢复语义 |
| 工具状态不同步 | 前端 toggle 与后端实际能力不一致 | 以 runtime snapshot 和 capability event 为准，前端展示 available / enabled / effective 的最终状态 |

## 10. 验收标准

如果满足以下条件，可认为 WebUI 基本同步了 TUI 核心能力：

- WebUI 可以清楚看到当前 mode
- WebUI 可以切换 mode
- WebUI 可以通过 GET /api/sessions/{id}/runtime 获取并恢复运行态
- WebUI 可以通过 PATCH /api/sessions/{id}/runtime 修改 mode / capability
- WebUI 可以看到并处理审批请求
- WebUI 可以记住审批规则
- WebUI 可以看到当前 session 的能力状态
- WebUI 可以看到运行中的 tool / sub-agent / workflow 状态
- WebUI 的 mode / approval / capability 语义与 TUI 一致
- WebUI 保持现有 Web 风格，不以 TUI 视觉复刻作为验收条件
- 两端共享同一套后端 session / event / command 语义

## 11. 结论

WebUI 同步 TUI 的关键，不是把界面做得像，而是把**运行态、权限态、模式态**做成同一套后端驱动的协议。

建议优先落地“runtime snapshot + 模式控制 + 审批中心 + 事件统一”四件事。这样 WebUI 就能从“聊天界面”升级为“完整运行控制台”，并且后续继续扩展 delegate、workflow、sub-agent 时不会再走偏。
