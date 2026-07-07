# Mode 边界增强方案：/btw 旁路问答 + 可编辑路径白名单 + 全自动编辑

> 状态: Proposal
> 日期: 2026-06-24
> 目标版本: 待定

## 1. 概述

本提案针对当前 mode（plan / agent / yolo）权限边界的三个痛点，新增三条交互指令及其配置落盘机制：

1. `/btw`（by the way）—— 在**不中断主任务**的前提下，继承主任务上下文历史，快速启动一个**一次性 sub-agent** 回答一个临时问题。结果显示在**临时浮动层**，**不写回主任务 session**，**不增加主任务上下文窗口占用与 token 消耗**。
2. `/alloweditpath` —— 维护一份**可编辑路径白名单**（支持通配符），让 agent 模式下命中白名单的 `write`/`edit` 无需逐次申请、自动放行。
3. `/allowautoedit` —— 打开 agent 模式下的**全自动编辑**开关：所有 `write`/`edit` 自动放行，相当于「只有 bash 需要申请权限」。

后两者的状态都会**落盘**到项目目录 `.vibe/allow.json`，启动时自动载入；`/allowautoedit` 同时支持全局配置。优先级为**项目目录优先、全局次之**。

## 2. 设计目标

| 目标 | 说明 |
|------|------|
| 主任务零打扰 | `/btw` 执行期间主任务的 message history、context usage、token 统计完全不受影响 |
| 上下文继承 | `/btw` 的 sub-agent 以主任务当前对话历史为只读上下文，能够「就刚才的内容」回答 |
| 结果不回写 | `/btw` 的问答只在浮动层展示，结束后不进入主 session，不进入 SQLite 持久化 |
| 渐进放权 | `/alloweditpath` 允许细粒度地按路径放行编辑，避免一次性全开 |
| 一键全自动 | `/allowautoedit` 让 agent 模式编辑无需申请，仅保留 bash 审批 |
| 状态可落盘 | 路径白名单与全自动开关持久化，重启后自动恢复 |
| 双层配置 | 项目级 `.vibe/allow.json` 覆盖全局配置，优先级清晰 |
| 与现有边界一致 | 复用既有 `NeedsApproval` 决策路径，不破坏 plan/yolo 语义 |

## 3. 非目标

- 不改变 `settings.json` 既有 schema 与字段含义（白名单单独放 `allow.json`）。
- 不改变 plan 模式只读语义、不改变 yolo 模式全自动语义。
- `/btw` 第一阶段只面向 TUI；Serve / ACP 暂不暴露（理由见 §7）。
- `/btw` 不支持多轮追问浮动层内连续对话（第一阶段为单问单答）。
- 不引入外部依赖；复用现有 multi-agent / delegate 的 sub-agent 基础设施。

## 4. 现状梳理（代码锚点）

### 4.1 审批决策

权限边界集中在 `internal/agent/agent.go`：

```go
// internal/agent/agent.go:1705
func (a *Agent) NeedsApproval(toolName string, args map[string]any) bool {
    if (toolName == "write" || toolName == "edit") && a.config.Mode == "agent" {
        return a.config.Settings != nil &&
            a.config.Settings.Approval.ConfirmBeforeWrite != nil &&
            *a.config.Settings.Approval.ConfirmBeforeWrite
    }
    if toolName != "bash" {
        return false
    }
    if a.isBashBlacklisted(args) {
        return true
    }
    switch a.config.Mode {
    case "plan":
        return false
    case "agent":
        return !a.isBashWhitelisted(args)
    case "yolo":
        return false
    default:
        return false
    }
}
```

当前 `write`/`edit` 在 agent 模式下的放行只看全局开关 `ConfirmBeforeWrite`，没有**按路径**的细粒度控制。这正是 `/alloweditpath` / `/allowautoedit` 的插入点。

### 4.2 命令分发

TUI 命令在 `internal/tui/commands.go` 的 `handleCommand`（约 line 334）中以 `switch command` 分发，已有 `/mode`、`/model`、`/delegate`、`/agent` 等。新增指令在此追加 case。

### 4.3 Sub-agent 基础设施

- `internal/agent/manager.go`：`AgentManager.Create(opts)` 创建受管 agent。
- `internal/agent/subagent.go`：`delegate_subagent` 为**阻塞式单任务**模型，最贴近 `/btw` 需求。
- `internal/agent/agent.go`：`GetMessages()` / `LoadHistoryMessages()` 提供上下文导入导出能力（经 `bridge.go:610` 暴露）。

`/btw` 的实现思路：复用 delegate 的阻塞式 sub-agent，但用主 agent 的 `GetMessages()` 作为只读历史注入，结果只回浮动层。

### 4.4 配置加载

`internal/config/settings.go`：`GlobalSettingsPath()` = `ConfigDir()/settings.json`，`ProjectSettingsPath()` = `.vibe/settings.json`；加载顺序为「全局 → 项目覆盖」（line 471 `LoadSettingsWithMeta`）。`allow.json` 复用同一目录约定与覆盖顺序。

## 5. 详细设计

### 5.1 `/btw` 旁路问答

#### 5.1.1 用户交互

```
/btw <问题文本>
```

- 输入后**不阻塞**主任务输入框的后续排队，但同一时刻只允许一个 `/btw` 在执行（避免并发浮层）。
- sub-agent 以「主任务当前完整对话历史 + 该问题」为输入运行。
- 运行中浮动层显示流式回答与简单的思考/工具进度。
- 结束后：
  - 浮动层保留答案，按 `Esc` / `q` 关闭。
  - **不**向主 session 追加任何 user/assistant 消息。
  - **不**改变主任务 `contextUsage`、`totalInputTokens` 等统计。

#### 5.1.2 上下文继承（关键：不污染主任务）

```
主 Agent.messages (只读快照)
        │  clone（深拷贝，剥离 usage）
        ▼
临时 Sub-Agent（独立实例）
   - LoadHistoryMessages(snapshot)
   - 追加一条 user: <问题>
   - 独立 registry（只读工具集，见 5.1.4）
   - 独立 session：内存态 / 临时文件，不落入主 session 目录
        │  流式事件 → 浮动层（非主 transcript）
        ▼
结束即销毁，snapshot 与 sub-agent 一并丢弃
```

要点：

- 用 `cloneMessagesWithoutUsage`（已存在于 `agent.go:226`）拷贝主历史，确保 sub-agent 的 token 统计独立、不回流。
- sub-agent 的 system prompt 复用主任务 system prompt，并追加一句「你是旁路助手：只回答用户的临时问题，不要修改任何文件，不要假设你的回答会被主任务记住」。
- sub-agent 的 message history、工具调用、最终回答全部**不**写回主 agent，也**不**写入主 session SQLite state。

#### 5.1.3 浮动层（Floating Layer）

当前 TUI 没有通用 overlay，只有 `toolModal`（`internal/tui/tool_modal.go`）这类居中弹层与 `render.go:308` 的 `toolModalOpen` 合成逻辑。方案：

- 新增 `btwModalOpen bool` 与 `btwModal` 状态（仿照 tool modal 的状态/渲染/关闭三件套：`invalidateToolModalCache` 风格）。
- 在 `render.go` 的合成阶段，当 `btwModalOpen` 时，在主视图之上叠加浮动层，展示：
  - 顶部：`💬 /btw: <问题>`
  - 主体：流式答案（可滚动，复用既有滚动组件 `components/vscroll`）
  - 底部：`Esc 关闭 · ↑/↓ 滚动`
- 浮动层内容**不**进入终端 scrollback（区别于已完成 transcript 块），关闭即清除。

#### 5.1.4 工具集限制

`/btw` 的 sub-agent 默认只读：注册 `read` / `grep` / `find` / `ls` / `skill_ref`，不注册 `write` / `edit` / `bash`。这样旁路问答天然安全，且无需走审批。

#### 5.1.6 旁路开销控制（主历史过长）

注入快照前对历史做体量控制，避免旁路 sub-agent 自身 token 暴涨：

- 估算注入快照 token（复用 `estimateChatRequestTokens` / 既有 estimator）。
- 超过阈值（默认取 sub-agent 模型 `ContextWindow` 的一定比例，如 50%）时，对快照做**保留首尾、裁剪中段**的截断：保留最早的系统/任务设定与最近若干轮对话，中段以一条占位说明替代（`[... 省略 N 条较早消息 ...]`）。
- 截断只作用于旁路 sub-agent 的输入快照，**不影响主任务历史**。
- 阈值可后续做成 `allow.json` / settings 可配置项；第一阶段用内置默认值。

#### 5.1.5 生命周期与中断

- 主任务运行中也可发起 `/btw`（二者独立 goroutine / 独立事件通道）。
- 关闭浮动层 = 取消 sub-agent 的 `context`（调用 `AgentManager.SetCancel` 注册的 cancel）。
- 主任务被 `/clear`、退出时，若有活动 `/btw`，一并取消。

### 5.2 `/alloweditpath` 可编辑路径白名单

#### 5.2.1 用户交互

```
/alloweditpath                      # 列出当前白名单
/alloweditpath add <glob>           # 新增一条（支持通配符）
/alloweditpath remove <glob>        # 删除一条
/alloweditpath clear                # 清空
```

示例 glob：`src/**`、`internal/agent/*.go`、`docs/**/*.md`、`*.json`。

#### 5.2.2 匹配语义

- 仅在 **agent 模式**对 `write` / `edit` 生效（plan 模式仍只读；yolo 模式本就全放行）。
- 命中白名单 → 跳过审批；未命中 → 走原有 `ConfirmBeforeWrite` 逻辑。
- 路径取自工具参数 `args["path"]`，统一转为相对工作目录的清洗路径后匹配。
- 通配符语义：采用 `**` 跨目录、`*` 单层的 doublestar 风格（用现有依赖或最小实现，详见 §6）。

#### 5.2.3 与 `allowedWorkDirs` / sandbox 的关系

`/alloweditpath` 只影响**是否需要人工审批**，不放宽 sandbox / allowedWorkDirs 的物理边界。即使路径在白名单里，仍受 sandbox 与工作目录限制约束。两者是「审批层」与「隔离层」的正交关系。

### 5.3 `/allowautoedit` 全自动编辑开关

#### 5.3.1 用户交互

```
/allowautoedit            # 显示当前状态 ON/OFF
/allowautoedit on         # 打开：agent 模式下所有 write/edit 自动放行
/allowautoedit off        # 关闭
```

#### 5.3.2 语义

- ON 时，agent 模式下 `write` / `edit` 一律跳过审批，等价于「只有 bash 还需审批」。
- 与 `/alloweditpath` 的关系：`/allowautoedit on` 是「全量放行」，覆盖路径白名单（路径白名单是子集）。
- 不影响 plan（仍只读）与 yolo（本就全开）。
- bash 的黑/白名单逻辑保持不变。

### 5.4 审批决策改造

在 `NeedsApproval` 的 write/edit 分支插入新规则（优先级从高到低）：

```go
if (toolName == "write" || toolName == "edit") && a.config.Mode == "agent" {
    // 1) 全自动编辑：直接放行
    if a.allow.AutoEdit {
        return false
    }
    // 2) 路径白名单：命中放行
    if p, ok := args["path"].(string); ok && a.allow.MatchEditPath(p) {
        return false
    }
    // 3) 回退到原有逻辑
    return a.config.Settings != nil &&
        a.config.Settings.Approval.ConfirmBeforeWrite != nil &&
        *a.config.Settings.Approval.ConfirmBeforeWrite
}
```

`a.allow` 为新增的运行时配置载体（见 §6.2），由 TUI 在执行 `/alloweditpath` / `/allowautoedit` 后同步更新并落盘。

## 6. 配置落盘：`.vibe/allow.json`

### 6.1 文件位置与优先级

| 层级 | 路径 | 适用项 |
|------|------|--------|
| 项目级 | `.vibe/allow.json` | `editPaths` + `autoEdit` |
| 全局级 | `<ConfigDir>/allow.json` | `autoEdit`（以及可选的全局 `editPaths`） |

- 加载顺序：**全局 → 项目覆盖**（与 settings 一致）。
- `autoEdit`：项目级显式设置时覆盖全局；项目未设置则用全局。
- `editPaths`：默认**项目级独立**（每个项目自己的白名单）。如同时存在全局 `editPaths`，采用「合并」还是「覆盖」需在评审中确认（见 §10 待决项）。本提案默认**项目优先、不合并**（与 settings 覆盖语义一致），保持简单。

### 6.2 数据结构

```go
// internal/config/allow.go (新增)
type AllowConfig struct {
    AutoEdit  bool     `json:"autoEdit,omitempty"`
    EditPaths []string `json:"editPaths,omitempty"`
}

func GlobalAllowPath() string  { return filepath.Join(ConfigDir(), "allow.json") }
func ProjectAllowPath() string { return filepath.Join(".vibe", "allow.json") }

func LoadAllow() (*AllowConfig, error)          // 全局→项目覆盖
func (c *AllowConfig) SaveProject() error       // 写 .vibe/allow.json
func (c *AllowConfig) SaveGlobalAutoEdit() error // 仅落 autoEdit 到全局
func (c *AllowConfig) MatchEditPath(path string) bool
```

示例 `.vibe/allow.json`：

```json
{
  "autoEdit": false,
  "editPaths": [
    "internal/**",
    "docs/**/*.md",
    "*.go"
  ]
}
```

示例全局 `<ConfigDir>/allow.json`：

```json
{
  "autoEdit": true
}
```

### 6.3 落盘时机

- `/alloweditpath add|remove|clear` → 更新内存 + 写 `.vibe/allow.json`。
- `/allowautoedit on|off`：
  - 默认写**项目级** `.vibe/allow.json`。
  - 新增 `global` 子参数写全局（如 `/allowautoedit on global`）。
- 启动时：`LoadAllow()` 注入 agent 的 `a.allow`，TUI 持有同一引用以便指令热更新。

## 7. 各运行模式的适用范围

| 能力 | TUI | Serve | Serve channels | ACP |
|------|-----|---------|--------|-----|
| `/btw` | ✅ 第一阶段 | ⛔（无浮层 UI 语义） | ⛔ | ⛔ |
| `/alloweditpath` | ✅ | 可选（slash 已支持） | 可选 | ⛔ |
| `/allowautoedit` | ✅ | 可选 | 可选 | ⛔ |
| `allow.json` 自动载入 | ✅ | ✅（复用 config 层） | ✅ | ✅ |

`/btw` 依赖浮动层 UI，仅 TUI 有意义。审批边界改造（§5.4）属于 agent 层，对所有 runtime 生效，因此 `allow.json` 在所有 runtime 自动载入即可统一行为。

## 8. 安全考量

- **`/btw` 只读**：sub-agent 不注册 write/edit/bash，旁路问答不可能改文件。
- **不回写**：旁路问答不进入主 session SQLite state，不污染审计轨迹（如需审计，可单独记录到 `btw.log`，待评审）。
- **白名单仅放宽审批层**：sandbox / allowedWorkDirs 物理边界不变。
- **全自动编辑显式开启**：`/allowautoedit on` 需用户主动执行并落盘，重启后状态可见、可关闭。
- **plan / yolo 语义不变**：新规则只在 agent 模式的 write/edit 分支生效。

## 9. 实施步骤

1. `internal/config/allow.go`：新增 `AllowConfig`、加载/保存/匹配（含通配符）。
2. `internal/agent`：在 agent 配置中持有 `*AllowConfig`，改造 `NeedsApproval`（§5.4），补单测（仿 `approval_test.go`）。
3. `internal/tui/commands.go`：新增 `/alloweditpath`、`/allowautoedit` 两个 case 与处理函数；指令修改后热更新 `a.allow` 并落盘。
4. `internal/tui`：新增 `/btw` case；实现旁路 sub-agent（复用 `AgentManager` + 只读 registry + `LoadHistoryMessages`）。
5. `internal/tui`：新增浮动层状态与渲染（仿 tool_modal），接入流式事件与 Esc 关闭/取消。
6. 启动装配：CLI 装配处 `LoadAllow()` 注入 agent 与 App。
7. 文档：`docs/en/changelog.md`、`docs/zh/changelog.md`；`/help` 文案补充新指令。
8. 测试：`make test` 与针对 `NeedsApproval`、`MatchEditPath` 的聚焦测试。

## 10. 已决问题（评审结论）

1. **指令大小写**：全部小写，`/btw`、`/alloweditpath`、`/allowautoedit`（更优雅，符合既有风格）。
2. **全局 `editPaths`**：**不落全局**，仅项目级 `.vibe/allow.json`。
3. **`/allowautoedit` 落盘层级**：默认落项目级；新增 `global` 子参数显式落全局（如 `/allowautoedit on global`）。
4. **`/btw` 上下文体量**：**需要控制旁路开销**。主历史超过阈值时对注入快照做截断/裁剪（见 §5.1.6），避免旁路 sub-agent token 暴涨。
5. **通配符库**：使用现有依赖中的 doublestar 实现（若不可用则最小自实现 `**`/`*`）。

### 命令最终形态

```
/btw <问题>
/alloweditpath [add <glob> | remove <glob> | clear]
/allowautoedit [on [global] | off [global]]
```

## 11. 兼容性

- 不修改 `settings.json` schema，`allow.json` 为独立新文件，缺失时行为与现状完全一致（无白名单、非全自动）。
- 现有 plan / agent / yolo 行为在不存在 `allow.json` 时保持不变。
- `/btw` 为纯增量 TUI 能力，不影响既有命令与会话格式。
