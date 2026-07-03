# 原生支持 ccstatusline 方案

> 日期: 2026-06-26
> 状态: Draft

## 背景

Claude Code 通过 `statusLine` 设置支持自定义底部状态行。它的核心机制是：在需要刷新状态行时执行一个命令，将一份 JSON payload 写入该命令的 stdin，然后把命令 stdout 原样显示为状态行。

对兼容性最关键的行为如下：

- 设置形态为 `statusLine.type = "command"`、`statusLine.command`，并支持可选的 `statusLine.padding` 和 `statusLine.refreshInterval`。
- 命令 stdin 是 JSON 对象，包含 session、cwd、model、workspace、cost、context window、token usage、rate limit、编辑器状态等元数据。
- 命令 stdout 会作为状态行显示，允许多行、ANSI 颜色和 OSC 8 超链接。
- 状态变更会触发刷新；在支持 `refreshInterval` 的版本中，还可以按固定秒数周期刷新。

`ccstatusline` 是基于这套协议实现的状态行渲染器。它在 piped mode 下读取 Claude status JSON，使用宽松 schema 解析，然后按用户配置的 widgets 渲染状态行。它的 TUI installer 会向 Claude settings 写入类似下面的配置：

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccstatusline",
    "padding": 0,
    "refreshInterval": 10
  }
}
```

本地 `/home/free/src/ccstatusline` 的实现对缺失字段比较宽容。因此 MothX 可以直接输出 Claude 兼容 payload 的子集，以纯 `stdin snapshot` 模式驱动 `ccstatusline`，让依赖当前状态快照的 widgets 工作，而不实现 transcript 文件兼容层。

## 目标

- 允许用户把 `ccstatusline` 配置为 MothX TUI footer/status line 渲染器，而不需要把 MothX 包在 Claude Code 里运行。
- 未配置或未启用外部 status line command 时，输入框下方继续显示当前默认 footer，行为和视觉都保持不变。
- 复用 Claude `statusLine` 对象形态，让现有 `ccstatusline` 命令、配置文件、文档和示例尽量少改或无需修改。
- 保持现有 TUI 行为：transcript scrollback、输入框、approval、question prompt、tool modal、`/btw`、multi-agent tab bar、compact mode、context/cache 显示都不应被破坏。
- 功能范围严格限定为交互式 TUI。Gateway、Hermes、ACP、print mode、systeminit 等其他入口都不需要该功能，也不应加载、注册或执行 status line command。

## 非目标

- 不 vendor 或重写 `ccstatusline`。
- 构建期不依赖 Node.js、Bun、`npx` 或 `ccstatusline`。
- 不改变 provider/session 现有配置语义。
- 首阶段不追求所有 Claude 专属 widgets 完全准确。Claude account、Claude rate limits、voice status、Claude hook 派生的 skills 可以为空或 best-effort。
- 不在任何非 TUI 模式中暴露 status line command 执行能力，包括 Hermes/Gateway 这类无人值守入口，以及 ACP/print mode 这类非交互入口。

## 当前项目适配点

本项目 TUI 已经在 `internal/tui/render.go:renderFooter` 集中生成默认 footer。生成 Claude 兼容 payload 所需的大部分运行时数据已经存在：

- provider/model: `App.provider`、`App.model`
- mode/sandbox: `App.mode`、`App.sandboxInfo`
- cwd/session: `App.session.GetHeader()`、`App.session.GetFile()`
- context usage: `App.contextUsage`
- 每轮 usage 和缓存累计值：`internal/tui/agent_events.go` 中的 `EventUsage` 处理
- 请求状态和耗时：`App.isThinking`、`App.timer`、`App.lastDuration`
- 终端宽度：`App.width`

现有 footer 渲染在输入框下方，默认两行：第一行是 mode/model/cwd，第二行是动态提示，并在右侧展示 sandbox/context/cache 信息。外部 status line 只在用户显式启用时替换这个输入框下方的 footer 区域；未启用、无输出或执行失败时继续显示当前默认 footer。它不接管 transcript、输入框或输入框上方的任何区域。

## 用户配置设计

在 `settings.json` 中新增一个可选顶层配置：

```jsonc
{
  "statusLine": {
    "enabled": true,
    "type": "command",
    "command": "ccstatusline",
    "padding": 0,
    "refreshInterval": 10,
    "timeoutMs": 800,
    "fallback": "builtin"
  }
}
```

字段语义：

| 字段 | 含义 |
| --- | --- |
| `enabled` | 是否启用外部 status line 渲染。默认 `false`。 |
| `type` | 首阶段只支持 `"command"`，保留该字段是为了兼容 Claude 配置形态。 |
| `command` | 要执行的 shell 命令。示例：`ccstatusline`、`npx -y ccstatusline@latest`、`bunx -y ccstatusline@latest`、`ccstatusline --config /path/config.json`。 |
| `padding` | 状态行输出下方追加的空行数。默认 `0`；首阶段应支持 `0` 和正整数。 |
| `refreshInterval` | 可选周期刷新秒数。建议 clamp 到 `1..60`；`0` 或缺省表示只按事件触发刷新。 |
| `timeoutMs` | 命令超时时间。默认约 `800ms`；命令失败或卡住不能阻塞 TUI。 |
| `fallback` | `"builtin"` 或 `"empty"`。默认 `"builtin"`，确保命令异常时 TUI 仍可用。 |

这是一个小型 settings schema 增量，不改变现有 provider、model、mode 等字段含义。该配置同时支持全局 settings 和项目 `.vibe/settings.json` 覆盖，沿用现有 settings 加载优先级。该配置只由 TUI 读取和执行；其他模式即使加载了 settings，也必须忽略 `statusLine`。

### 安装与使用体验

首阶段可以只文档化手动配置。后续可增加便捷命令：

- `/statusline`：展示当前命令和最近一次渲染错误。
- `/statusline off`：禁用外部 status line。
- `/statusline ccstatusline`：向项目或全局 settings 写入 `command: "ccstatusline"`。

该命令不应修改 `~/.claude/settings.json`。MothX 应只管理自己的 `~/.vibecoding/settings.json` 和项目 `.vibe/settings.json`。

## 运行时架构

新增一个 TUI 专用的内部包：

```text
internal/tui/statusline/
  config.go      # 配置结构和默认值
  payload.go     # Claude 兼容 payload 结构
  runner.go      # 命令执行、超时、缓存、错误
```

TUI 集成方式：

1. `App.renderFooter()` 继续保留生成内置 footer 的能力。
2. `App.View()` 中输入框下方仍然只保留一个 footer slot。默认情况下该 slot 渲染当前内置 footer。
3. 当 `settings.StatusLine.Enabled` 为 true、command 有效，且 runner 已有成功的非空输出时，该 footer slot 改为渲染外部 status line 输出。
4. Runner 根据状态变化或 interval tick 异步执行命令，并缓存最近一次成功输出。`View()` 绝不能同步 spawn 进程。
5. 命令未配置、未启用、失败、超时、非零退出或输出为空时，footer slot 回退到当前默认 footer。
6. 执行命令时设置 `CCSTATUSLINE_WIDTH=<App.width>`，让 flex separator 和右对齐在 `ccstatusline` 内可靠工作。

显示优先级：

1. `statusLine.enabled != true`：显示当前默认 footer。
2. `statusLine.enabled == true` 但 command 为空或无效：显示当前默认 footer，并记录诊断错误。
3. command 正在首次执行且尚无成功输出：显示当前默认 footer。
4. command 成功返回非空 stdout：用 stdout 替换输入框下方 footer。
5. command 后续失败：默认继续显示当前默认 footer；如未来需要，也可以增加“保留上一帧成功 status line”的策略，但首版不建议这么做，避免展示过期状态。

刷新触发点：

- TUI size change
- model/provider/mode/session 变化
- turn start/end/done/error
- usage/context 更新
- tool call/result 事件，如果状态行需要展示 live activity
- 可选 `refreshInterval`

命令执行规则：

- 使用 `context.WithTimeout`。
- 需要 shell 解析时，优先沿用 settings 中的 `ShellPath`/`ShellCommandPrefix` 语义；否则使用平台默认 shell。
- 将 payload JSON 写入 stdin。
- 分别捕获 stdout/stderr。
- 只裁剪 stdout 尾部换行，保留 ANSI 和 OSC escape sequence。
- 限制 stdout/stderr 最大字节数，避免异常命令占用过多内存。

安全边界：

- Status line command 是用户显式配置的本地命令，风险类似 shell hook，应默认关闭。
- TUI 的 plan/agent/yolo 都属于交互式 TUI 模式。只要用户显式启用 status line，就在这些 TUI mode 中运行；非 TUI 的同名 mode 不运行。不增加 `runInPlanMode` 之类的额外开关。
- payload 中不要传 provider API key 或任何已展开的 secret。
- 后续如果需要更严格控制，可以增加 env allowlist。

## Payload 映射

MothX 输出 Claude 兼容 JSON 对象。`ccstatusline` 的 `StatusJSONSchema` 是宽松解析，缺字段不会直接失败。

示例 payload：

```jsonc
{
  "hook_event_name": "Status",
  "session_id": "abc12345",
  "cwd": "/repo",
  "model": {
    "id": "deepseek-v4-flash",
    "display_name": "DeepSeek-V4-Flash"
  },
  "workspace": {
    "current_dir": "/repo",
    "project_dir": "/repo"
  },
  "version": "v0.1.x",
  "output_style": {
    "name": "default"
  },
  "effort": {
    "level": "medium"
  },
  "cost": {
    "total_cost_usd": 0.0123,
    "total_duration_ms": 45000,
    "total_api_duration_ms": 2300,
    "total_lines_added": 0,
    "total_lines_removed": 0
  },
  "context_window": {
    "context_window_size": 1000000,
    "total_input_tokens": 50113,
    "total_output_tokens": 10462,
    "current_usage": {
      "input_tokens": 8500,
      "output_tokens": 1200,
      "cache_creation_input_tokens": 5000,
      "cache_read_input_tokens": 2000
    },
    "used_percentage": 8,
    "remaining_percentage": 92
  },
  "vim": null,
  "rate_limits": null
}
```

推荐字段映射：

| Claude 字段 | MothX 来源 |
| --- | --- |
| `hook_event_name` | 固定 `"Status"` |
| `session_id` | 优先 session header ID；否则从 `session.GetFile()` 派生短 ID |
| `cwd` | `session.GetHeader().Cwd`，fallback 到进程 cwd |
| `model.id` | `App.model.ID` |
| `model.display_name` | `App.model.Name` |
| `workspace.current_dir` | 同 cwd |
| `workspace.project_dir` | 首版同 session 初始 cwd；未来可映射第一个 workspace root |
| `version` | MothX 版本字符串 |
| `output_style.name` | `"default"` 或当前 TUI theme 名称 |
| `effort.level` | 当前 thinking level；App 未跟踪时省略 |
| `cost.total_cost_usd` | TUI runner 累计 usage cost |
| `cost.total_duration_ms` | session duration 或最近一次请求耗时；有 transcript export 后优先累计 session duration |
| `context_window.context_window_size` | `ContextUsage.ContextWindow` 或 `model.ContextWindow` |
| `context_window.used_percentage` | `ContextUsage.Percent` |
| `context_window.remaining_percentage` | `100 - used_percentage` |
| `context_window.current_usage` | 最近一次 `provider.Usage`，按 Claude 字段名映射 |
| `context_window.total_input_tokens` | 累计 `Usage.TotalInputTokens()` |
| `context_window.total_output_tokens` | 累计 `Usage.Output` |
| `rate_limits` | phase 1 为 `null` |
| `vim` | 除非后续暴露编辑器 mode，否则为 `null` |

`provider.Usage` 到 Claude current usage 的映射：

- `input_tokens`: `Usage.Input`
- `output_tokens`: `Usage.Output`
- `cache_creation_input_tokens`: `Usage.CacheWrite`
- `cache_read_input_tokens`: `Usage.CacheRead`

## ccstatusline 兼容矩阵

基于纯 `stdin snapshot` 模式，phase 1 预计可工作的 widgets：

- Model
- Version
- Current Working Dir
- Context Length / Context Window / Context % / Context Bar
- Cache Hit Rate / Cache Read / Cache Write，基于最近一轮 usage
- Tokens Input / Tokens Output，通过 `context_window` fallback
- Session Cost，如果 TUI 侧累计 cost
- Git 和 JJ 类 widgets，因为它们基于 `cwd` 自行执行本地 VCS 命令
- Terminal Width，通过 `CCSTATUSLINE_WIDTH`
- Custom Text / Custom Symbol / Link / Custom Command

部分支持或受限：

- Tokens Total / cached/session totals：当前只基于 snapshot 和 TUI 累计值，精度不如 transcript 驱动模式。
- Input/Output/Total Speed：不支持，因为需要 transcript 中的时间序列和 usage 行。
- Session Clock 和 Block Timer：不支持 transcript 驱动计算。
- Compaction Counter：不支持，因为需要 transcript 中的 compaction marker。
- Skills：`ccstatusline` 读取以 `session_id` 为 key 的 hook 派生 metrics；MothX 当前不会发这些 hooks。

首阶段不支持：

- Claude Account Email
- Claude usage API widgets 和真实 Claude rate-limit buckets
- Voice Status
- Vim Mode，除非后续增加编辑器 mode metadata

## Snapshot 边界

本方案明确只支持 `stdin snapshot` 模式：

- MothX 每次刷新状态行时，把当前状态 JSON 写入 `ccstatusline` 的 stdin。
- 不输出 `transcript_path`。
- 不生成或导出任何给 `ccstatusline` 消费的 JSONL transcript 文件。
- 所有依赖 transcript 历史扫描的 `ccstatusline` widgets 都视为不支持或精度受限。

## 渲染细节

- 保留命令多行输出。启用外部 status line 时，输入框下方 footer slot 的高度需要根据输出行数和 `padding` 计算；未启用时继续使用当前默认 footer 高度。
- 宽度处理使用 TUI 现有 ANSI-aware 工具，例如 `charmbracelet/x/ansi` 和 Lipgloss width helper。
- 不剥离 ANSI 颜色；`ccstatusline` 会有意输出 ANSI 和 OSC 8。
- 如果命令输出宽于终端，优先让 `ccstatusline` 通过 `CCSTATUSLINE_WIDTH` 自行处理；只有最后兜底时才硬截断。
- multi-agent tab bar 的相对位置应保持一致。当前 TUI 在 footer 后追加 agent tab bar，外部 status line 应保留这个顺序，除非后续进行更大布局调整。

## 实施计划

1. 在 `internal/config.Settings` 增加 `StatusLineSettings`，默认 disabled。
2. 增加 `internal/tui/statusline` 的 payload 和 runner 类型，避免非 TUI 入口依赖或误用。
3. 扩展 TUI `App` 状态：最近输出、最近错误、最近 payload hash、refresh timer、最近 usage counters。
4. 基于现有 App/session/model/context 字段构建 payload。
5. 从现有事件处理器和 window-size update 触发异步刷新。
6. 当存在成功且非空的 cached render 时，用外部输出替换输入框下方 footer；否则显示当前默认 footer。
7. 增加只读 `/statusline` 状态命令；runner 稳定后再考虑 enable/disable 子命令。
8. 不实现 transcript exporter，保持纯 `stdin snapshot` 模式。
9. 实现落地时补充 `docs/en/configuration.md` 和 `docs/zh/configuration.md`；changelog 只在实现完成时更新。

## 测试计划

聚焦测试：

- disabled status line 的配置加载和默认值
- model/session/context/usage 到 payload 的映射
- command runner 成功路径，验证 stdin JSON
- timeout fallback
- 非零退出 fallback，并保留 stderr 诊断
- 多行 ANSI 输出保留
- `CCSTATUSLINE_WIDTH` 环境变量传递
- command disabled、未配置、首次执行中、返回空输出或 failing 时，TUI 输入框下方仍显示当前默认 footer

可选集成测试：

- 使用一个本地小 shell script 作为 status line command，触发 usage/context event 后断言 footer 输出变化。

仓库测试不应依赖 `ccstatusline`、Node.js、Bun、`npx`、GitHub 网络或 Anthropic 凭证。

## 已确认决策

- `statusLine` 配置同时支持全局 settings 和项目 `.vibe/settings.json` 覆盖，沿用现有 settings 优先级。
- 不增加 `runInPlanMode` 开关。该功能已经严格限定为交互式 TUI，且默认关闭；用户显式启用后，在 TUI plan/agent/yolo 中都可运行。
- 保持纯 `stdin snapshot` 模式，不输出 `transcript_path`，不生成 transcript JSONL。
- 不暴露 MothX 自己的 rate-limit/usage buckets 给 Claude usage API widgets；相关字段首阶段保持空值或 `null`。
- 不推动 `ccstatusline` upstream 支持安装到 MothX settings。两个项目保持独立；本项目目标只是原生支持 status line 协议，并让用户可以无痛复用 `ccstatusline` 的渲染能力。
