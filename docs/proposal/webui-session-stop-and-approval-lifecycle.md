# WebUI 会话终止与审批生命周期方案

> 状态：Proposal
> 日期：2026-07-22
> 目标版本：待定

## 1. 背景

WebUI 的一次聊天请求可能在页面刷新、切换会话或浏览器网络断开后，仍继续在服务端执行。前端原有的 `AbortController` 只能终止当前浏览器持有的 HTTP/SSE 请求，不能可靠终止服务端仍在运行的 session run。

尤其在 agent 模式等待工具审批时，审批等待由 Agent 内部状态控制；仅取消 HTTP request context 不足以解除等待。若会话仍被标记为 running，用户重新进入该 session 后无法开始新的对话。

本提案定义 WebUI 的**会话级停止按钮**、服务端取消语义，以及审批请求在一次运行终止时的最终状态和 SQLite 审计要求。

## 2. 目标

1. WebUI 根据 session 的运行状态显示停止按钮；停止是界面操作，不要求用户输入聊天指令 `/stop`。
2. 用户刷新页面后重新进入一个仍在运行的 session，仍可停止该 session 当前运行。
3. 停止针对指定 session 的当前 run，不影响其他 session 或其他 run。
4. 一次 run 因用户停止、超时、客户端断开、Agent 结束或异常失败而终结时，所有未决审批必须结束，不能遗留为 pending。
5. 每个审批请求及其最终决议（批准、拒绝、取消/废弃）必须作为 session 审计日志持久化到 `session.db`。
6. run 终止完成后，session 不再是 running，用户可以立即开始下一轮对话。

## 3. 非目标

- 不将 WebUI 的停止行为实现为用户输入的 `/stop` 聊天命令。
- 不改变 plan / agent / yolo 的权限和审批策略。
- 不支持跨 session 查看、响应、取消或复用审批。
- 不恢复已经被取消或废弃的审批；终态审批不可再次决议。
- 本提案不要求在运行中切换 mode。运行时 mode 修改是否支持，应由独立提案定义。

## 4. 核心概念与归属

### 4.1 Session、Run 与 Approval

- **Session**：持久化对话身份，使用 `sessionID` 标识。
- **Run**：session 内的一次服务端 Agent 执行，使用 `runID` 标识；同一 session 同一时间最多只有一个 active run。
- **Approval**：Agent 因受保护工具调用发出的审批请求，使用 `approvalID` 标识。

审批的完整归属键是：

```text
sessionID + runID + approvalID
```

`approvalID` 不得被视为全局身份。即使不同 session 或不同 run 产生相同的 approval ID，也必须彼此隔离。

### 4.2 隔离约束

- 审批查询只返回目标 session 当前 active run 的 pending approvals。
- 审批响应接口必须以 sessionID 定位 session，并只在该 session 当前 run 内查找 approvalID。
- Session A 的 stop、审批响应或审批清理不得读取、修改或解除 Session B 的审批。
- 一个旧 run 的审批在新 run 启动后仍不得被响应；旧 run 的审批必须已写入终态审计记录。

## 5. WebUI 用户体验

### 5.1 输入区操作

WebUI 应根据服务端 session runtime snapshot 决定输入区状态：

| Session 状态 | 发送按钮 | 停止按钮 | 输入行为 |
|---|---|---|---|
| 无 active run | 可用 | 不显示或禁用 | 可发送新消息 |
| active run 为 `running` / `cancelling` | 禁用 | 可见；`cancelling` 时禁用并显示处理中 | 不允许并发发送 |
| stop 成功且运行已终结 | 恢复可用 | 隐藏 | 可立即发送下一轮 |

停止按钮应放置在输入框下方、发送按钮旁边。它是 WebUI 的显式控制，不在消息内容中插入 `/stop`，也不将停止动作写为用户聊天消息。

### 5.2 刷新后恢复

页面初始化、选择 session、以及 reconnect SSE 时，WebUI 必须请求该 session 的 runtime snapshot。

当 snapshot 表示该 session 有 active run 时：

1. UI 恢复 busy 状态；
2. 输入区显示停止按钮；
3. 如有 pending approvals，只展示属于该 session + active run 的审批；
4. 用户点击停止后调用服务端 session 停止接口；
5. 不依赖刷新前浏览器中存在的 `AbortController`。

### 5.3 停止请求

建议沿用管理 API，而非聊天 slash command：

```http
POST /api/sessions/{sessionID}/stop
```

接口成功仅表示服务端已接受并开始终止该 session 的 active run。客户端随后以 runtime event / run event / 再次拉取 snapshot 确认终态，避免将“请求已发送”错误地当作“run 已完全结束”。

## 6. Run 状态机

建议将 run 的生命周期明确为：

```text
running -- user stop ----------------------> cancelling
running -- timeout/disconnect/agent end ---> terminalizing
cancelling --------------------------------> cancelled
terminalizing -----------------------------> completed | failed | cancelled
```

约束：

- 进入 `cancelling` 或 `terminalizing` 后，run 不得产生新的可决策 pending approval。
- 终态为 `completed`、`failed` 或 `cancelled` 后，清除 active run 引用、取消函数和内存 pending approval；session `running=false`。
- 新 run 只能在旧 run 已到终态且 session 解锁后创建。

## 7. 审批生命周期与终止语义

### 7.1 正常审批

```text
Agent 发出审批请求
  -> 服务端绑定 sessionID + runID 并登记
  -> 写入 approval_requested 审计事件
  -> WebUI 显示该 session 的 pending approval
  -> 用户批准或拒绝
  -> 解除等待
  -> 写入 approval_resolved 审计事件（终态）
```

### 7.2 Run 终止时的审批处理

当一个 run 进入取消或任何终结流程时，属于该 run 的每一个 pending approval 必须：

1. 从 WebUI 可决策队列移除；
2. 向等待中的 Agent 返回拒绝/不可继续结果，确保工具调用不被执行；
3. 持久化终态审计日志；
4. 发布 `approval_resolved` 或等价 runtime event，供已连接 WebUI 清除界面状态；
5. 后续收到该 approvalID 的响应时返回“该审批已不再 pending”的错误，绝不恢复执行。

建议的终态字段如下：

| 场景 | `status` | `action` | `message` |
|---|---|---|---|
| 用户点击 WebUI 停止 | `cancelled` | `deny_once` | `run cancelled by user` |
| 请求超时 | `cancelled` | `deny_once` | `run timed out before approval was resolved` |
| 客户端断开导致服务端 run 取消 | `cancelled` | `deny_once` | `run ended before approval was resolved` |
| Agent/服务端异常结束 | `cancelled` | `deny_once` | `run ended before approval was resolved` |
| 用户显式拒绝 | `resolved` | `deny_once` | `approval denied` |
| 用户显式批准 | `resolved` | 对应批准动作 | `approval accepted` |

这里的 `deny_once` 表示审批对应的工具调用不可继续执行；`status=cancelled` 用于保留“并非用户正常点击拒绝，而是 run 已失效”的审计语义。

### 7.3 并发竞态：审批产生与 stop 同时发生

必须避免以下窗口：Agent 已发出 approval event，但服务端尚未将审批登记到 session 的 pending 集合；这时用户点击 stop，普通的 context cancel 不会解除 `RequestApproval()` 等待。

建议实现以下原子语义：

1. session 保存当前 active run 的 Agent 引用和 run lifecycle 状态；
2. stop 在同一生命周期保护下先将 run 标为 `cancelling`；
3. stop 无条件对当前 run 的 Agent 调用 `Abort()`，再调用 run context 的 cancel；
4. stop 清理已经登记的 pending approvals，并为每项写入 `cancelled` 审计事件；
5. `registerSessionApproval()` 在登记前检查 runID 和 lifecycle：
   - 仅当该 run 仍为 `running` 时，登记为 pending 并对 WebUI 可见；
   - 若 run 已 `cancelling` / `terminalizing` / 终态，则立即拒绝该审批、写入 `cancelled` 审计事件，不加入 pending 队列；
6. run 的 defer/finalizer 必须复用同一终结路径，保证任何退出原因都会收敛 pending approvals。

这样可保证“审批请求出现”与“session stop”无论顺序如何，都不会产生悬挂等待或遗漏审计日志。

## 8. 持久化与审计

### 8.1 存储要求

审批日志必须保存在与 session 相同的根 `sessions.db` 中，遵循现有 `internal/session/migrations.go` 迁移机制。不得仅保存在 `APISession.pendingApprovals` 内存 map 中。

如果现有 session run event 存储已经能保存结构化 `approval_request` 和 `approval_resolved` 数据，应复用该机制；否则新增 migration 与明确的审批事件表/字段。新 schema 必须通过向 `migrations` slice 追加迁移建立，而非在业务路径直接执行 `CREATE TABLE IF NOT EXISTS`。

### 8.2 最小审计字段

每个审批请求和终态记录至少应保留：

```text
session_id
run_id
approval_id
event_type                 # approval_requested / approval_resolved
status                     # pending / resolved / cancelled
requested_at / resolved_at
request                    # 工具、参数、风险、原因、mode、workDir 等结构化快照
resolution.action
resolution.message
```

`runID` 是恢复历史与隔离旧 run 的关键字段，不得在记录终态时丢失或错误归属到后续 run。

### 8.3 恢复规则

- WebUI 刷新后，pending 列表以服务端当前 runtime 为准，不从旧日志推断为 pending。
- 已持久化为 `resolved` 或 `cancelled` 的审批只进入历史/审计展示，绝不重新显示为待处理。
- 服务端启动恢复时，若发现一个不再有活动运行实体的旧 run 留有 pending 审批，应通过受控恢复流程将其标为 `cancelled`，而不是继续向用户暴露可批准操作。

## 9. 服务端实现边界

建议关注以下现有代码区域：

- `internal/serve/openaiapi/session_mgr.go`
  - `APISession` 运行态、`CancelSessionRun()`、session runtime snapshot。
- `internal/serve/openaiapi/handler_chat.go`
  - run 创建、Agent 创建、context cancel、run defer/finalizer、stream/non-stream event 消费。
- `internal/serve/openaiapi/approval.go`
  - approval 登记、响应、清理、审计事件写入。
- `internal/agent/agent_approval.go`
  - `RequestApproval()` 对 `Abort()` 和审批响应的等待语义。
- `ui/src/views/Chat.svelte`
  - session runtime 拉取、SSE runtime event、审批中心、输入区发送/停止控制。

实现时应将“终结 run”收敛到一个幂等的服务端操作，供用户 stop、timeout、stream 结束、错误退出和 defer/finalizer 共用，避免不同退出路径只清理内存或只写日志的一致性缺陷。

## 10. 验收标准与测试

### 10.1 后端测试

1. **刷新后停止**：创建运行中的 session，模拟没有前端 controller；调用 session stop 接口后 run 进入 `cancelled`，`running=false`，随后可创建新 run。
2. **跨 session 隔离**：Session A 和 B 同时拥有待审批项；停止 A 后，B 的 Agent、pending approval 和日志均不变。
3. **停止已登记审批**：stop 后 Agent 的审批等待退出；该审批从 pending 移除，并在 `session.db` 中存在 `cancelled` 的终态事件。
4. **停止/登记竞态**：模拟 approval event 已发出但尚未完成登记时 stop；Agent 不得悬挂，审批必须产生 `cancelled` 审计记录，且不得出现在 pending 列表。
5. **终态不可响应**：run 已取消后，对其旧 approvalID 发出批准请求必须失败，且工具不得执行。
6. **其他结束原因**：超时、客户端 request context 结束、Agent 错误结束均会终结 pending approvals 并写入终态日志。
7. **run 归属**：同一 session 的旧 run 审批不能被新 run 的操作响应或覆盖；审计事件保留正确 runID。

### 10.2 WebUI 测试

1. runtime snapshot 显示 active run 时，发送旁显示停止按钮，发送操作不可用。
2. 页面刷新并重新选择 running session 后，停止按钮仍可见并可调用 session stop API。
3. 点击停止后，界面显示 cancelling/停止中，收到终态 runtime event 后恢复输入与发送。
4. 停止后审批中心不再显示该 run 的 pending approval；审计历史显示取消原因。
5. Session A 的 stop 不改变当前展示的 Session B 审批或运行状态。

## 11. 开放问题

- stop API 应同步等待 run 完全终结，还是立即返回 `cancelling` 并由 runtime event 确认终态？建议立即返回，避免 HTTP 请求被长时间阻塞。
- 对非 WebUI 的 OpenAI-compatible API client 断开连接，是否默认取消 run，还是仅停止 SSE 输出而允许后台继续？无论选择何种策略，只要 run 最终结束，未决审批都必须终结并持久化。
- 审批历史在 WebUI 中应展示完整的请求快照，还是默认仅展示摘要、按需展开原始 JSON？这不影响本提案的持久化与终态约束。
