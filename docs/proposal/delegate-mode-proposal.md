# Delegate Mode 委派模式方案设计

> 状态: Proposal
> 日期: 2026-06-15
> 目标版本: v0.1.40

## 1. 概述

Delegate Mode（委派模式）是在现有 multi-agent 架构上的轻量增强：主 Agent 可以自行判断某个子任务是否适合委派给一个独立 sub-agent 执行。委派执行是**阻塞式**的：主 Agent 调用委派工具后等待 sub-agent 完整完成，再只接收最终总结结果。

核心目标是节约主 Agent 上下文占用：sub-agent 的中间推理、工具调用过程、探索路径不会进入主 Agent 的 conversation history；主 Agent 只看到压缩后的结果、证据、变更和风险。

## 2. 设计目标

| 目标 | 说明 |
|------|------|
| AI 自动判断 | 委派模式开启后，由主 Agent 根据系统提示决定是否使用委派工具 |
| 阻塞等待 | 委派工具同步等待 sub-agent 完成后返回结果 |
| 上下文节省 | 主 Agent 只接收最终摘要，不接收 sub-agent 完整过程 |
| 单 sub-agent 限制 | 同一时间最多运行一个委派 sub-agent |
| 独立上下文 | sub-agent 拥有独立 message history、registry、session 和 system context |
| 非嵌套 | sub-agent 不能再创建 sub-agent，也不能再次委派 |
| 可运行时切换 | 支持 `/delegate on|off|status` 动态控制 |
| 启动时开启 | 支持 `--delegate` 设置初始状态 |

## 3. 非目标

- 不替代现有 `--multi-agent` 的异步多代理模式。
- 不暴露 `spawn/status/send/destroy` 工作流给委派模式使用。
- 不支持多个委派 sub-agent 并行执行。
- 不要求 sub-agent 中间输出实时进入主 Agent 上下文。
- 第一阶段不强制支持 Hermes / ACP，除非后续明确需要。

## 4. 与现有 Multi-Agent 模式的关系

当前已有 multi-agent 模式：

- CLI flag: `--multi-agent`
- 工具：
  - `subagent_spawn`
  - `subagent_status`
  - `subagent_send`
  - `subagent_destroy`
- 运行方式：异步、handle 驱动、主 Agent 需要轮询和汇总。

Delegate Mode 新增一个单独工具：

- CLI flag: `--delegate`
- Slash command: `/delegate on|off|status`
- 工具：`delegate_subagent`
- 运行方式：阻塞式、单任务、自动等待完成、返回最终总结。

二者可以共存：

| 模式 | 工具 | 执行模型 | 适用场景 |
|------|------|----------|----------|
| `--multi-agent` | `subagent_*` | 异步多代理 | 多个并行调查、长期子任务、手动编排 |
| `--delegate` | `delegate_subagent` | 阻塞单代理 | 单个独立子任务、节省上下文、简单编排 |

如果用户同时开启 `--multi-agent --delegate`，主 Agent 可同时看到两组能力，但系统提示需要清楚说明两者差异。

## 5. 用户体验

### 5.1 启动时开启

```bash
vibecoding --delegate
vibecoding --delegate "分析这个 bug 并修复"
vibecoding --print --delegate "检查当前项目测试失败原因"
```

Gateway 可选支持：

```bash
vibecoding gateway --delegate
```

### 5.2 会话内切换

```text
/delegate status
/delegate on
/delegate off
```

行为：

- `/delegate status`：显示当前委派模式是否开启。
- `/delegate on`：注册 `delegate_subagent` 工具，更新 system prompt，重置当前 Agent 实例。
- `/delegate off`：移除 `delegate_subagent` 工具，更新 system prompt，重置当前 Agent 实例。

注意：Agent 创建时会冻结工具列表和 system prompt，因此切换后必须重建 Agent，下一条用户消息开始生效。

## 6. 工具设计

### 6.1 工具名称

```text
delegate_subagent
```

### 6.2 工具描述

```text
Delegate one bounded independent subtask to a blocking sub-agent. The tool waits until completion and returns a summarized result.
```

### 6.3 参数 Schema

```jsonc
{
  "type": "object",
  "properties": {
    "task": {
      "type": "string",
      "description": "Focused delegated task, including scope, relevant paths/context, expected output, and stop conditions"
    },
    "mode": {
      "type": "string",
      "enum": ["plan", "agent", "yolo"],
      "default": "agent",
      "description": "Sub-agent execution mode"
    },
    "work_dir": {
      "type": "string",
      "description": "Working directory for the sub-agent, defaults to current work directory"
    },
    "tools": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Allowed tools for the sub-agent. Empty means all tools except nested sub-agent/delegate tools."
    },
    "max_iterations": {
      "type": "integer",
      "default": 50,
      "description": "Maximum sub-agent loop iterations"
    },
    "system_prompt_extra": {
      "type": "string",
      "description": "Additional instructions for the delegated worker"
    }
  },
  "required": ["task"]
}
```

### 6.4 返回格式

工具结果建议返回 JSON 文本：

```jsonc
{
  "status": "done",
  "result": "Direct answer or completed artifact summary",
  "evidence": "Files inspected, commands run, checks performed",
  "changes": "Files modified, if any",
  "risks": "Assumptions, uncertainty, follow-up needed",
  "duration": "12.3s"
}
```

失败时：

```jsonc
{
  "status": "error",
  "error": "sub-agent failed or timed out",
  "partial_result": "last assistant response if available",
  "duration": "5m0s"
}
```

## 7. 委派判断准则

系统提示应指导主 Agent：

适合委派：

- 可独立描述、有明确边界的子任务。
- 代码库检索、定位文件、调查失败原因、验证假设。
- 需要较多工具调用但最终只需摘要反馈的工作。
- 与主对话强相关性较低、可以隔离上下文的工作。

不适合委派：

- 很小的任务，直接执行成本更低。
- 需要持续和用户澄清的问题。
- 强依赖主 Agent 当前完整上下文的状态化任务。
- 风险较高、需要主 Agent 实时审查每一步的操作。

主 Agent 必须：

- 对 sub-agent 输出保持审查态度。
- 重要结论需要自行验证或明确标记来源。
- 最终用户回答仍由主 Agent 负责组织。

## 8. 系统提示设计

在 `DelegateMode` 开启时追加：

```md
## Delegation Mode

You may delegate one bounded independent subtask at a time using `delegate_subagent`.

Use delegation when:
- the task can be isolated from the main conversation
- the subtask needs codebase inspection, focused research, or verification
- the result can be summarized back to the main agent

Do not delegate:
- tiny tasks
- tasks requiring continuous back-and-forth with the user
- highly stateful tasks that require the full main conversation

The `delegate_subagent` tool blocks until completion and returns only a summarized result.
Only one delegated sub-agent can run at a time.
Review the returned result before making final decisions.
```

Sub-agent prompt 继续使用现有 `BuildSubAgentContext()`，并额外保证：

- 不包含 multi-agent orchestrator 指南。
- 不包含 delegation mode 指南。
- registry 中移除所有 subagent/delegate 工具。

## 9. 架构改造

### 9.1 `internal/agent/agent.go`

`Config` 新增字段：

```go
type Config struct {
    ...
    MultiAgent   bool
    DelegateMode bool
}
```

`BuildSystemPrompt(...)` 调用需要传递 `DelegateMode`。

### 9.2 `internal/agent/system_prompt.go`

修改签名：

```go
func BuildSystemPrompt(
    mode string,
    toolNames []string,
    cwd string,
    extraContext string,
    toolSnippets map[string]string,
    toolGuidelines []string,
    multiAgent bool,
    delegateMode bool,
) string
```

追加 Delegate Mode section。

测试需要覆盖：

- delegate off 时不包含 `Delegation Mode`。
- delegate on 时包含 `delegate_subagent` 指南。
- sub-agent prompt 不包含 delegate 指南。

### 9.3 `internal/agent/subagent.go`

新增：

```go
type DelegateSubAgentTool struct {
    manager *AgentManager
    busy    atomic.Bool // or mutex-protected bool
}
```

执行流程：

1. 校验 `task`。
2. 使用 busy 标记拒绝并发委派。
3. 从 context 获取：
   - parent agent id
   - parent event channel
   - parent run context
4. 使用 `AgentManager.Create(AgentOptions{ParentID: parentID, ...})` 创建子 Agent。
5. `MarkRunning`。
6. 在当前 goroutine 执行 `a.Run(runCtx, buildSubAgentTask(task))`。
7. 消费事件直到完成：
   - 转发 approval request 给父事件通道。
   - 记录 `EventDone` / `EventError`。
8. 读取 `lastAssistantResponse(a)`。
9. `MarkDone` / `MarkError`。
10. 清理子 Agent：建议 `Destroy` 或新增 manager 方法释放活跃 agent 但保留最终状态。
11. 返回最终 JSON。

### 9.4 `internal/agent/subagent_tools.go`

新增注册函数：

```go
func RegisterDelegateSubAgentTool(registry *tools.Registry, manager *AgentManager) {
    if registry == nil || manager == nil {
        return
    }
    registry.Register(NewDelegateSubAgentTool(manager))
}
```

现有 `RegisterSubAgentTools(...)` 保持不变。

### 9.5 `internal/agent/factory.go`

确保子 Agent 不具备 nested sub-agent / delegate 能力：

```go
if opts.ParentID != "" {
    registry.Remove("subagent_spawn")
    registry.Remove("subagent_status")
    registry.Remove("subagent_send")
    registry.Remove("subagent_destroy")
    registry.Remove("delegate_subagent")
}
```

`AgentOptions` 可增加可选覆盖：

```go
DelegateMode *bool
```

规则：

```go
delegateMode := f.delegateEnabled && opts.ParentID == ""
if opts.DelegateMode != nil {
    delegateMode = *opts.DelegateMode
}
if opts.ParentID != "" {
    delegateMode = false
}
```

## 10. CLI / TUI 集成

### 10.1 `cmd/vibecoding/main.go`

新增 flag：

```go
flags.BoolVar(&flagDelegate, "delegate", false, "Enable delegation mode (blocking single sub-agent tool)")
```

`runOptions` 新增：

```go
delegate bool
```

AgentManager 初始化建议调整为始终可用于 TUI 动态开启：

```go
factory := agent.NewAgentFactory(...)
agentMgr := agent.NewAgentManager(factory)
```

但工具注册按模式控制：

```go
if opts.multiAgent {
    agent.RegisterSubAgentTools(registry, agentMgr)
}
if opts.delegate {
    agent.RegisterDelegateSubAgentTool(registry, agentMgr)
}
```

Cron 仍只在 `opts.multiAgent` 时启用。

### 10.2 `cmd/vibecoding/main_util.go`

`runPrint(...)` 增加 `delegate bool` 参数。

构造 `agent.Config`：

```go
DelegateMode: delegate,
```

print 模式下，如果 delegate 开启，也要注册主 Agent 到 `AgentManager`，让子 Agent 创建时能找到 parent。

### 10.3 `internal/tui/app.go`

`App` 新增字段：

```go
delegateMode bool
```

`NewApp(...)` 增加参数并保存。

### 10.4 `internal/tui/input.go`

创建 agent 时传递：

```go
DelegateMode: a.delegateMode,
```

### 10.5 `internal/tui/commands.go`

新增命令：

```go
case "/delegate":
    a.handleDelegateCommand(parts)
```

行为：

```text
/delegate              # same as status
/delegate status       # show ON/OFF
/delegate on           # register delegate_subagent, enable prompt, reset agent
/delegate off          # remove delegate_subagent, disable prompt, reset agent
```

切换时必须：

```go
a.agent = nil
a.agentHistoryLoaded = false
```

如果当前 agent 正在运行，应拒绝切换或中止当前运行。建议第一版拒绝切换：

```text
Cannot change delegation mode while the agent is running.
```

### 10.6 `internal/tui/command_render.go`

帮助文案新增：

```text
/delegate [on|off|status] - Toggle delegation mode
```

## 11. Gateway 集成

为保持 slash command parity，Gateway 也应支持 `/delegate`。

### 11.1 `internal/gateway/gateway.go`

`RunOptions` 增加：

```go
Delegate bool
```

### 11.2 `cmd/vibecoding/main.go`

Gateway flags 增加：

```go
gatewayFlags.BoolVar(&flagDelegate, "delegate", false, "Enable delegation mode")
```

传给 `gateway.Run(...)`。

### 11.3 Gateway session

`GatewaySession` 增加：

```go
DelegateMode bool
```

每个 session 独立维护委派状态。

### 11.4 `internal/gateway/handler_chat.go`

构建 agent 前根据 session 状态注册/移除工具：

```go
if sess.DelegateMode && sess.AgentMgr != nil {
    agent.RegisterDelegateSubAgentTool(sess.Registry, sess.AgentMgr)
} else {
    sess.Registry.Remove("delegate_subagent")
}
```

构造 `agent.Config`：

```go
DelegateMode: sess.DelegateMode,
```

### 11.5 `internal/gateway/commands.go`

新增：

```go
case "/delegate":
    return s.cmdDelegate(sess, parts)
```

帮助文本同步新增 `/delegate [on|off|status]`。

## 12. ACP / Hermes 集成

第一阶段可不做 ACP / Hermes，以降低改动面。

后续如果需要：

- ACP：增加 `RunOptions.Delegate`、`--delegate` flag、注册工具、传递 `DelegateMode`。
- Hermes：在 `hermes.json` 增加配置或启动参数，按 session 注册 `delegate_subagent`。

## 13. 超时与取消

现状：`executeSingleToolCall` 对所有工具统一设置 5 分钟 timeout：

```go
toolCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
```

而 `DefaultSubAgentPolicy().TimeoutPerAgent` 是 10 分钟。

第一版建议：

- 接受外层工具 5 分钟实际限制。
- `delegate_subagent` 内部仍使用 policy timeout，但不会超过 parent/tool context。
- 文档中注明委派工具受工具执行超时约束。

后续可优化：

- 给 tool 增加可选 timeout interface。
- 或在 `executeSingleToolCall` 中对 `delegate_subagent` 使用 sub-agent policy timeout。

## 14. 并发控制

由于 Agent 默认 `ToolExecutionMode` 是 `parallel`，模型可能在一轮响应中同时调用多个工具，甚至多个 `delegate_subagent`。

必须在 `DelegateSubAgentTool` 内部做全局/实例级并发限制：

```go
if !busy.CompareAndSwap(false, true) {
    return tools.ToolResult{}, fmt.Errorf("a delegated sub-agent is already running")
}
defer busy.Store(false)
```

并发限制范围建议绑定到 tool instance。由于主 Agent registry 内只有一个 `delegate_subagent` 实例，可以满足“每次只能开启一个 subagent”。

如果 Gateway 多 session 共享 registry，需要确保每个 session registry 有独立 tool instance；否则并发限制会跨 session 生效，可能过严。

## 15. 状态清理

委派执行完成后需要释放 active child，避免：

- manager 中残留 agent。
- parent children 数量不断增长，触发 `MaxChildren` 限制。
- `/agent list` 中出现已完成但不再需要的委派 agent。

可选方案：

### 方案 A：执行后直接 `Destroy`

优点：简单，释放彻底。

缺点：如果 `Destroy` 删除状态，则无法通过 manager 查询历史委派状态。但委派模式本来只返回最终结果，不要求后续查询。

### 方案 B：新增 `ReleaseChild`

释放 active agent，但保留 `ManagedAgentStatus`。

优点：更适合调试和审计。

缺点：需要改 `AgentManager`，改动稍大。

第一版建议采用方案 A。

## 16. 安全与权限

- sub-agent 模式默认使用 `agent`，保持现有 approval / sandbox 规则。
- `mode` 参数仍经过 `DefaultSubAgentPolicy().Validate`。
- 默认 policy 当前只允许 sub-agent 使用 `agent` mode；如果传入 `plan` 或 `yolo` 会被拒绝。
- 如需允许 `plan`，应单独调整 policy，而不是委派工具绕过。
- approval request 继续转发给 parent event channel，由 TUI/CLI/Gateway 统一处理。

## 17. 测试计划

### 17.1 `internal/agent/subagent_test.go`

新增：

- `TestDelegateSubAgentTool`
  - 执行 `delegate_subagent`
  - 返回 `status=done`
  - 包含最终 result
- `TestDelegateSubAgentToolMissingTask`
  - 缺少 `task` 报错
- `TestDelegateSubAgentToolRejectsConcurrentRun`
  - 同时调用两个 delegate，第二个返回 busy 错误
- `TestDelegateSubAgentToolNoNestedDelegation`
  - 子 Agent registry 不包含 `delegate_subagent`
- `TestDelegateSubAgentToolCleansUpChild`
  - 执行完成后 parent children 不残留活跃 child

### 17.2 `internal/agent/agent_test.go`

新增/扩展：

- `TestBuildSystemPromptDelegateModeGated`
- `TestBuildSystemPromptDelegateAndMultiAgentCanCoexist`
- `TestSubAgentPromptOmitsDelegateGuidance`

### 17.3 TUI command tests

如现有测试结构支持，覆盖：

- `/delegate status`
- `/delegate on`
- `/delegate off`
- running 状态下拒绝切换
- on/off 后 agent 被 reset

### 17.4 Gateway command tests

覆盖：

- `/delegate status`
- `/delegate on`
- `/delegate off`
- `/help` 包含 delegate

## 18. 实施顺序

1. Agent core
   - `Config.DelegateMode`
   - system prompt support
   - factory registry filtering
2. Blocking delegate tool
   - `DelegateSubAgentTool`
   - `RegisterDelegateSubAgentTool`
   - unit tests
3. CLI / print mode
   - `--delegate`
   - `runOptions.delegate`
   - `runPrint` support
4. TUI
   - App field
   - `/delegate` command
   - help text
5. Gateway
   - flag plumbing
   - session state
   - `/delegate` command
6. Focused validation
   - `go test ./internal/agent`
   - `go test ./internal/tui`
   - `go test ./internal/gateway`

## 19. 开放问题

1. 委派工具实际 timeout 是否应保持 5 分钟，还是提升到 10 分钟？
2. 委派完成后是否需要保留 manager status 供调试？
3. 是否允许 delegate sub-agent 使用 `plan` mode？当前 policy 默认只允许 `agent`。
4. Gateway 的 `/delegate on` 是否应该影响默认 session，还是仅当前 `x_session_id`？建议仅当前 session。
5. ACP / Hermes 是否需要第一阶段同步支持？建议暂缓。

## 20. 验收标准

- `vibecoding --delegate` 启动后，主 Agent system prompt 包含委派指南。
- `/delegate on|off|status` 可在 TUI 中正常工作。
- 开启后 registry 包含 `delegate_subagent`，关闭后移除。
- 主 Agent 能调用 `delegate_subagent`，并阻塞等待 sub-agent 完成。
- 主 Agent 只收到最终总结结果，不收到 sub-agent 完整中间过程。
- 同一时间第二个委派请求会被拒绝。
- sub-agent 不能创建 nested sub-agent，也不能再次委派。
- 现有 `--multi-agent` 行为不回归。
