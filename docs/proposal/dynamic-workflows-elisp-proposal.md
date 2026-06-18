# Dynamic Workflows with Embedded Elisp Proposal

> 状态: 讨论中 (Draft)
> 日期: 2026-06-18
> 目标版本: v0.1.44

## 1. 背景

Claude Code 的 dynamic workflows 把复杂任务拆成一个可执行编排脚本，由 runtime 调度大量
sub-agents，并把中间状态留在脚本运行时，而不是塞进主对话上下文。它的核心价值不是新增一种
prompt，而是把 fan-out / fan-in、阶段依赖、结果聚合、失败处理变成可复跑的程序。

VibeCoding 已经具备大部分底座:

- `AgentManager` 管理 agent 生命周期。
- `subagent_spawn/status/send/destroy` 支持异步子 agent。
- `delegate_subagent` 支持同步单子 agent。
- `AgentFactory` 已支持 `WorkDir`、`Tools`、`SystemPromptExtra`、`MaxIterations` 等 per-agent 参数。
- TUI 和 Gateway 都已有 slash command 框架。
- Cron 已验证后台 agent 调度和持久化状态的基本模式。

因此，最小实现不需要重写 agent runtime。重点是新增一个 workflow runtime 和一个足够小的
Elisp DSL 解释器。

## 2. 当前决策

| 决策 | 结论 |
|------|------|
| DSL | 使用 Elisp 语法的最小子集 |
| Elisp 实现 | 独立 Go library `github.com/startvibecoding/vibeEmacsLispVm v0.0.1`，已通过 `go.mod` 接入 |
| 扩展机制 | 只通过 Go 侧注册函数 / special form，不扩展或改变 Elisp 语法 |
| 第一版目标 | 支持可执行 workflow、阶段执行、并发子 agent、结果引用和状态查询 |
| 第一版非目标 | 不实现完整 Emacs Lisp，不加载 `.el` 包，不支持宏系统，不支持任意文件/进程 API |
| 集成方式 | 复用现有 `AgentManager` / `AgentFactory` / session / tools / sandbox |
| 启用方式 | 使用独立 `--workflows` 启动参数开启；不挂在 `--multi-agent` 上 |

明确拒绝 JSON DSL。Workflow 的唯一 DSL 是 Elisp subset；不提供 JSON DSL 用户入口，
也不以 JSON 结构作为内部 workflow 编排表达。JSON 仅可用于 run state 持久化等普通数据存储，
不能用于描述或执行 workflow。

Elisp subset 必须保持语法稳定: 不新增 reader 语法、不新增非 Elisp 风格的 token、不把 workflow
能力做成特殊语法糖。所有 workflow 能力都以普通函数或少量必要 special form 的形式注册进解释器。

Elisp VM 不在 `vibecoding` 仓库内实现。`vibeEmacsLispVm` 已作为独立 Go module 发布到 GitHub，
提供通用 Elisp subset parser/evaluator 和 Go 侧函数注册 API。`vibecoding` 通过 `go.mod`
依赖 `github.com/startvibecoding/vibeEmacsLispVm v0.0.1`，只负责把 workflow builtins 注册进去。

Workflow 与 multi-agent 的上层能力必须隔离: workflow 可以在底层复用 `AgentManager`、
`AgentFactory`、session、sandbox、tool registry 等基础设施，但不能因为启用 `--workflows`
就暴露 `subagent_*` 工具或注入 multi-agent orchestration prompt；也不能因为启用
`--multi-agent` 就暴露 workflow tools 或注入 workflow prompt。

## 3. MVP 用户体验

启动方式:

```bash
vibecoding --workflows
vibecoding --workflows --mode agent
vibecoding gateway --workflows
vibecoding acp --workflows
```

用户可以让主 agent 生成并执行一个 workflow:

```text
用 workflow 并行审计 internal/gateway 和 internal/hermes 的认证风险，最后交叉验证结论。
```

模型可调用 `workflow_run` 工具并传入 Elisp DSL:

```elisp
(workflow "auth-risk-audit"
  (concurrency 4)
  (phase "scan"
    (parallel
      (agent "gateway"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/gateway for authentication and workdir validation risks. Return findings with file paths and line numbers.")
      (agent "hermes"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/hermes for authentication and webhook security risks. Return findings with file paths and line numbers.")))
  (phase "verify"
    (agent "cross-check"
      :mode "plan"
      :tools '("read" "grep" "find")
      :prompt (concat
        "Cross-check these findings. Reject weak claims and produce final prioritized risks.\n\n"
        (result "scan.gateway")
        "\n\n"
        (result "scan.hermes")))))
```

用户可以查询:

```text
/workflows
/workflows show <run-id>
/workflows cancel <run-id>
```

Gateway 也提供同名 slash commands，便于 HTTP 客户端查询 workflow 状态。

## 4. Elisp Subset 设计

这不是完整 Emacs Lisp，而是一个受限的 Elisp 最小子集。语法保持 Elisp S-expression 形态，
方便模型生成和人类阅读，但只实现 workflow 编排需要的最小集合。

### 4.1 词法和数据类型

支持:

- list: `(foo bar)`
- symbol: `workflow`、`agent`、`:prompt`
- keyword symbol: `:mode`、`:tools`
- string: `"hello"`
- integer / float
- bool: `t` / `nil`
- quote: `'("read" "grep")`
- line comment: `; comment`

暂不支持:

- vector
- char literal
- backquote / comma
- reader macro
- package namespace
- buffer / marker / process 等 Emacs runtime 对象

### 4.2 Special Forms

MVP 只需要:

| Form | 用途 |
|------|------|
| `quote` / `'` | 字面 list |
| `progn` | 顺序执行 |
| `let` | 局部绑定 |
| `setq` | 设置变量 |
| `if` | 条件分支 |
| `when` / `unless` | 简化条件 |

### 4.3 Builtins

通用内置函数:

| 函数 | 用途 |
|------|------|
| `concat` | 拼接字符串 |
| `format` | 最小格式化，第一版可只支持 `%s` |
| `list` | 构造列表 |
| `length` | 返回列表或字符串长度 |
| `=` / `<` / `>` | 数值比较 |
| `string=` | 字符串比较 |
| `not` / `and` / `or` | 布尔逻辑 |

Workflow 内置函数:

| 函数 | 用途 |
|------|------|
| `workflow` | 定义并执行一个 workflow |
| `concurrency` | 设置最大并发子 agent 数 |
| `phase` | 定义顺序执行阶段 |
| `parallel` | 并行执行多个 `agent` |
| `series` | 顺序执行多个 `agent` 或子 form |
| `agent` | 启动一个子 agent 并等待结果 |
| `result` | 读取之前任务结果，如 `"scan.gateway"` |
| `results` | 读取某个 phase 的全部结果 |
| `log` | 写入 workflow run 日志 |

第一版 `agent` 是阻塞语义: 启动子 agent，等待完成，把结果写入 run state。`parallel` 内的多个
`agent` 并发执行，但受 workflow `concurrency` 限制。

### 4.4 函数注册机制

解释器提供一个最小注册表，Go 侧可以注册普通函数和必要 special form:

```go
type BuiltinFunc func(ctx *EvalContext, args []Value) (Value, error)
type SpecialForm func(ctx *EvalContext, unevaluated []Expr) (Value, error)

func (e *Evaluator) RegisterFunc(name string, fn BuiltinFunc)
func (e *Evaluator) RegisterSpecial(name string, fn SpecialForm)
```

注册约束:

- 注册名必须是普通 symbol，例如 `agent`、`parallel`、`result`。
- 不允许注册 reader macro 或改变 tokenizer/parser 行为。
- 不允许通过注册函数获得文件、shell、网络等直接副作用能力。
- Workflow 能力通过注册函数提供，而不是通过语法扩展提供。
- MVP 中只有 `quote`、`let`、`setq`、`if`、`progn`、`when`、`unless` 这类求值控制需要 special form；`workflow`、`phase`、`parallel`、`agent` 优先实现为注册函数，只有在必须控制参数求值时才注册为 special form。

## 5. 执行模型

### 5.1 Runtime Flow

1. `workflow_run` 接收 Elisp DSL 字符串。
2. 独立库 `vibeEmacsLispVm` 解析为 AST。
3. `vibeEmacsLispVm` evaluator 执行 AST，并通过 `vibecoding` 注册进去的 workflow builtins 调用 `WorkflowHost`。
4. `WorkflowHost.RunAgent` 使用 `AgentManager.Create(agent.AgentOptions{...})` 创建子 agent。
5. 子 agent 的最终响应写入 `RunState.Results["phase.agent"]`。
6. `workflow_status` 和 slash command 从 store 读取 run state。

### 5.2 Prompt Cache 约束

Workflow 实现必须遵守 `docs/proposal/cache-optimization.md` 的缓存准则，尤其是冻结 system prompt、
session context 注入、双标记滚动缓冲和 Insert-then-Compress。

具体要求:

- 不允许在 workflow 运行过程中修改已创建 agent 的 system prompt 或 tool definitions。
- Workflow 工具注册必须发生在 agent 构造前；启用/禁用 workflow 后必须通过创建新 agent 生效。
- Workflow 指导文本可以加入 `BuildSystemPrompt`，但只能作为 frozen system prompt 的一部分，在 agent 构造时一次性确定。
- run id、phase name、task status、result 摘要、当前时间、动态并发状态等 runtime 信息不能写入 system prompt。
- 传给子 agent 的具体任务、phase 上下文、前序结果引用应作为普通 user message 或 system-injected message 进入消息流，而不是拼进 system prompt。
- 如果需要注入 workflow runtime context，应使用 `provider.NewSystemInjectedUserMessage(...)`，并确保 `SystemInjected=true`，让 cache marker 选择逻辑跳过它。
- 子 agent 压缩仍使用该子 agent 自己的 frozen system prompt 和 frozen tools，沿用 Insert-then-Compress，不为 workflow 单独创建不同的压缩 prompt。
- 不把完整 workflow run state 反复注入主 agent 上下文；主 agent 只接收 compact summary，完整状态由 workflow store 查询。
- `system_prompt_extra` 在 workflow 中应谨慎使用: 它会参与子 agent 构造期的 frozen system prompt。MVP 优先把动态约束放入任务 prompt，只有稳定、可复用、低变化的 worker contract 才能放入 `SystemPromptExtra`。

缓存友好的子 agent 创建策略:

1. 对同一 workflow run，优先复用稳定的 tools/mode/worker contract 组合，减少 system prompt 变体。
2. 把每个任务的具体 prompt 放入 `a.Run(ctx, taskPrompt)`，不要为每个 task 生成不同 system prompt。
3. 前序结果通过 `(result ...)` 拼入下一阶段的 task prompt；不要写入 `extraContext`。
4. Workflow status 查询只读 store，不触发 LLM 调用。

### 5.3 上层能力隔离

Workflow 和 multi-agent 共享底层 agent 执行能力，但上层开关、工具和 prompt 指令独立。

启动参数语义:

| 参数组合 | 暴露给主 agent 的工具 | System prompt 指导 |
|----------|----------------------|--------------------|
| 无 | 默认工具 | 无 sub-agent / workflow 指导 |
| `--multi-agent` | `subagent_spawn/status/send/destroy` | 只注入 Sub-Agent Tools 指导 |
| `--workflows` | `workflow_run/status/cancel` | 只注入 Workflow 指导 |
| `--multi-agent --workflows` | 两组工具都暴露 | 两组指导分别注入，但文本分区清晰，互不引用为前置条件 |

实现要求:

- `--workflows` 需要初始化 `AgentManager`，但该 manager 只作为 workflow runner 的内部依赖。
- 单独启用 `--workflows` 时，主 agent registry 不注册 `subagent_*`，只注册 `workflow_*`。
- 单独启用 `--multi-agent` 时，主 agent registry 不注册 `workflow_*`。
- `BuildSystemPrompt` 应增加独立的 `workflowsEnabled bool` 参数，不能复用 `multiAgent bool` 判断 workflow 指导。
- Workflow runner 创建的 worker agent 默认不注册 `subagent_*` 或 `workflow_*`，避免 workflow 内部递归暴露上层编排工具。
- 如果未来允许 nested workflow，必须作为独立设计重新讨论，不能由工具注册泄漏自然发生。

### 5.4 状态持久化

建议路径:

```text
.vibe/workflows/runs/<run-id>.json
```

MVP `RunState` 字段:

```go
type RunState struct {
    ID        string
    Name      string
    Status    string // running, done, error, canceled
    StartedAt time.Time
    UpdatedAt time.Time
    Phases    []PhaseState
    Results   map[string]AgentResult
    Logs      []WorkflowLog
    Error     string
}
```

状态文件是 runtime 状态，不替代 session JSONL。子 agent 仍按现有 session 体系保存自己的消息。

### 5.5 安全边界

Elisp evaluator 不提供:

- 文件读写函数
- shell/process 函数
- 网络函数
- 动态加载函数
- 反射 Go runtime 的能力

所有外部副作用必须通过 `agent` 函数间接发生，而子 agent 仍受现有 mode、tool filter、sandbox、
approval 机制约束。

## 6. 代码落点

新增:

```text
internal/workflow/
  types.go          # Workflow state, result, host interfaces
  runner.go         # workflow runtime and cancellation
  store.go          # file-backed run state store
  tools.go          # workflow_run/status/cancel tools
```

已完成:

- `go.mod`: 已引入 `github.com/startvibecoding/vibeEmacsLispVm v0.0.1`。
- `internal/workflow/lisp.go`: 已新增最小 VM 接入层，返回 `vibeEmacsLispVm` evaluator。
- `internal/workflow/types.go` / `runner.go` / `store.go`: 已新增 workflow 状态、Elisp builtins runner 和 JSON store。
- `internal/workflow/agent_host.go` / `tools.go`: 已新增真实 AgentManager host 和 `workflow_run/status/cancel` tools。
- `cmd/vibecoding/main.go`: 已增加 `--workflows` 启动参数；启用时注册 workflow tools。
- `internal/agent/system_prompt.go`: 已增加独立 `workflows` 判断，启用 workflow 时追加 workflow 使用指导、自研 Elisp VM 子集说明和 workflow DSL 参考；未启用 workflow 时不注入这些内容。
- 已补充 CLI flag、AgentFactory worker prompt 隔离、workflow tool 注册隔离和 Gateway help 的回归测试。
- `docs/en/tools.md` / `docs/zh/tools.md`: 已补充 workflow tools 说明，明确 DSL 只接受 Elisp，`workflow_cancel` 仅取消当前进程内 active run。

待修改:

- 无。

## 7. 最小实现步骤

1. [x] 在 `/home/free/src/vibeEmacsLispVm` 创建独立 Go module，实现 lexer/parser/evaluator 和注册 API。
2. [x] 在独立库中完成 Elisp subset 单元测试，覆盖 quote、let、setq、if、progn、函数注册、错误定位。
3. [x] 发布 `vibeEmacsLispVm` 到 GitHub，并在 `vibecoding` 通过 `go.mod` 引入 `v0.0.1`。
4. [x] 新增 `internal/workflow/lisp.go` 最小接入层和依赖验证测试。
5. [x] 在 `vibecoding` 中实现 `WorkflowHost` 接口和 fake host 测试，验证 `parallel`、`series`、`result`。
6. [x] 接入真实 `AgentManager`，实现 `workflow_run` / `workflow_status` / `workflow_cancel` tools。
7. [x] 增加 `--workflows` 启动参数，在 CLI / ACP / Gateway 路径独立注册 workflow tools。
8. [x] 增加 TUI `/workflows` 和 Gateway `/workflows` 查询入口。
9. [x] 增加 prompt 指导，让模型优先为多阶段并行任务生成 workflow DSL。
10. [x] 增加 cache 相关测试，确认 workflow 不在运行时污染 worker frozen system prompt / tools，动态 workflow context 使用 worker task prompt。
11. [x] 用一个只读审计 workflow 做端到端验证。

## 8. MVP 验收标准

- [x] Elisp VM 在 `/home/free/src/vibeEmacsLispVm` 作为独立 Go library 开发。
- [x] `vibecoding` 不内嵌 Elisp parser/evaluator；已通过 `go.mod` 依赖该库。
- [x] 不依赖现成第三方 Elisp 库；只依赖我们自己的 `vibeEmacsLispVm`。
- [x] 能解析并执行基础 Elisp subset。
- [x] `workflow_run` 能并发运行至少 2 个子 agent。
- [x] `concurrency` 能限制同时运行的子 agent 数。
- [x] 后续 phase 能通过 `(result "phase.agent")` 引用前序结果。
- [x] run state 能持久化到 `.vibe/workflows/runs/`。
- [x] `/workflows` 能列出运行记录。
- [x] `/workflows show <id>` 能显示 phase/task 状态和结果摘要。
- [x] `/workflows cancel <id>` 能取消当前进程内正在运行的 workflow。
- [x] 子 agent 仍遵守现有 mode、tool filter、sandbox、approval 规则。
- [x] `--workflows` 不依赖 `--multi-agent` 开启。
- [x] 仅启用 `--workflows` 时，主 agent 不暴露 `subagent_*` 工具，也不注入 Sub-Agent Tools 指导。
- [x] 仅启用 `--multi-agent` 时，主 agent 不暴露 `workflow_*` 工具，也不注入 Workflow 指导。
- [x] Workflow worker agent 默认不暴露 `subagent_*` 或 `workflow_*` 编排工具。
- [x] Workflow 工具注册发生在 agent 构造前；运行中不修改 frozen system prompt / frozen tools。
- [x] Workflow runtime context 不进入 system prompt；当前 MVP 通过 worker task prompt 作为普通 user message 传递动态上下文。
- [x] 子 agent compaction 仍沿用自己的 frozen system prompt 和 frozen tools。

## 9. 讨论记录

### 2026-06-18

- 调研 Claude Code dynamic workflows 后，初始最小方案曾考虑 JSON DSL + Go runner。
- 用户明确希望使用 Elisp 作为 DSL。
- 决策更新: 直接拒绝 JSON DSL。主方案为最小 Elisp subset，不依赖现成第三方 Elisp 库；JSON 仅允许作为 run state 等普通持久化格式，不能作为 workflow DSL 或内部编排表示。
- 决策更新: Elisp 实现必须是最小子集 + Go 侧自定义函数注册机制。不能改变 Elisp 语法，不能新增 reader/parser 语法扩展，workflow 能力只能通过注册普通函数或必要 special form 提供。
- 决策更新: Workflow 必须遵守 `cache-optimization.md`。不得在 workflow 运行时修改 frozen system prompt / tools；run state、phase、结果等动态信息不能写入 system prompt，应通过普通 user message、`SystemInjected` message 或 workflow store 管理。
- 决策更新: Workflow 使用独立 `--workflows` 启动参数开启，不挂在 `--multi-agent` 上。底层可复用 agent manager/factory，但上层工具注册和 system prompt 指令必须隔离，避免 workflow 与 multi-agent 互相污染。
- 决策更新: Elisp VM 在 `/home/free/src/vibeEmacsLispVm` 作为独立 Go library 开发，不放进 `vibecoding` 仓库。发布到 GitHub 后，`vibecoding` 已把它作为第三方依赖引入。
- 实施记录: `/home/free/src/vibeEmacsLispVm` 已创建独立 Go module，完成最小 Elisp subset parser/evaluator、Go 侧 `RegisterFunc` / `RegisterSpecial` API、核心 special forms/builtins 和单元测试。
- 实施记录: `vibeEmacsLispVm` 已发布到 GitHub，`vibecoding` 已通过 `go.mod` 引入 `github.com/startvibecoding/vibeEmacsLispVm v0.0.1`，并新增 `internal/workflow` 的最小 VM 接入测试。
- 实施记录: `vibecoding` 已新增 workflow runner/store/tools/AgentHost，接入 `--workflows` CLI/ACP/Gateway 开关，补充 TUI/Gateway `/workflows` 状态查询命令。
- 实施记录: 已补充 `--workflows` 与 `--multi-agent` 的 CLI flag 隔离测试、workflow tools 不注册 `subagent_*` 的测试，以及 worker agent 不继承 workflow/multi-agent/delegate prompt 的测试。`AgentHost` 已显式关闭 worker 的 workflow 编排开关，并将工具调用 ctx 与父 run ctx 组合，任一取消都会中断 worker。
- 实施记录: 已补充 workflow DSL 参数传递测试，确认 `:mode`、`:tools` 等 worker 选项会传入 host，并继续交由现有 AgentManager/AgentFactory 执行 sandbox、approval、tool filter 等规则。
- 实施记录: 已实现进程内 active workflow registry。`workflow_run` 运行时注册 cancel func，完成后自动移除；`workflow_cancel` 和 TUI/Gateway `/workflows cancel <id>` 可取消当前进程内 active run，并将 run state 保存为 `canceled`。
- 实施记录: 已更新中英文工具文档，记录 `workflow_run` / `workflow_status` / `workflow_cancel` 的可用范围、参数和 active run 限制。
- 实施记录: 已新增只读审计 workflow 端到端测试，覆盖 `workflow_run` tool、真实 `AgentManager`/`AgentFactory`、worker agent 调度、结果汇总和 store 查询。
- 实施记录: Workflow worker 是普通 child agent，compaction 继续走 `Agent.Compact` 的 frozen system prompt / frozen tools 路径；worker prompt 隔离测试已覆盖 workflow 指导不会进入 child frozen prompt。
- 实施记录: 已优化 workflow-only system prompt，明确 VM 是我们自研最小 Elisp 子集，列出支持语法、special forms、builtins、不支持项、host-defined workflow forms 和推荐模板；测试确认默认非 workflow prompt 不包含 Elisp/workflow DSL 参考。
