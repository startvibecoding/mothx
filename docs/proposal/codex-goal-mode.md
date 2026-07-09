# Codex Goal Mode 实现调研报告

> 调研时间：2026-07-09
> 版本：Codex CLI v0.142.5
> 源码路径：`codex-rs/ext/goal/`

---

## 1. 概述

Goal 模式是 Codex CLI 中用于**长期任务跟踪和自动续跑**的机制。用户通过 `/goal` 命令设定一个目标，系统会自动在空闲时续跑、追踪 token 消耗，并在达到预算上限时自动停止。

核心设计目标：
- 将"完成一个任务"从单轮交互提升为**跨轮次的持久化目标**
- 通过 token 预算控制成本
- 自动续跑减少人工干预
- 严格的完成审计防止模型过早标记完成

---

## 2. 架构总览

```
┌─────────────────────────────────────────────────────┐
│                    TUI Layer                         │
│  /goal <obj>  /goal edit  /goal pause  /goal resume │
│  goal_menu.rs  goal_status.rs  goal_files.rs        │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                Extension Layer                       │
│  GoalExtension (extension.rs)                        │
│  ├── ThreadLifecycleContributor                      │
│  ├── TurnLifecycleContributor                        │
│  ├── ToolLifecycleContributor                        │
│  ├── TokenUsageContributor                           │
│  ├── ConfigContributor                               │
│  └── ToolContributor                                 │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Tool Layer                           │
│  GoalToolExecutor (tool.rs)                          │
│  ├── create_goal  (spec.rs)                          │
│  ├── get_goal                                      │
│  └── update_goal                                    │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│               Runtime Layer                          │
│  GoalRuntimeHandle (runtime.rs)                      │
│  ├── continue_if_idle()    ← 自动续跑               │
│  ├── inject_active_turn_steering()  ← 注入提示词    │
│  ├── account_active_goal_progress()  ← 用量追踪     │
│  └── stop_active_goal_for_turn()  ← 异常处理        │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                Storage Layer                          │
│  GoalStore (state/src/runtime/goals.rs)              │
│  SQLite: thread_goals 表                             │
└─────────────────────────────────────────────────────┘
```

---

## 3. 状态机

### 3.1 六种状态

| 状态 | 含义 | 触发方式 |
|------|------|----------|
| `Active` | 目标正在执行 | 用户创建 / resume |
| `Paused` | 用户暂停 | `/goal pause` |
| `Blocked` | 连续3轮受阻 | 模型调用 `update_goal(blocked)` |
| `BudgetLimited` | token 超预算 | 系统自动检测 `tokens_used >= token_budget` |
| `UsageLimited` | 用量限制 | 系统触发 |
| `Complete` | 目标完成 | 模型调用 `update_goal(complete)` |

### 3.2 状态转换图

```
                    ┌─────────────┐
         创建 ─────▶│   Active    │◀──── resume
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
    ┌──────────┐    ┌──────────┐    ┌───────────────┐
    │  Paused  │    │ Blocked  │    │ BudgetLimited │
    └──────────┘    └──────────┘    └───────┬───────┘
                                            │
                                            ▼
                                    ┌──────────────┐
                                    │ UsageLimited │
                                    └──────────────┘

          Active ──(tokens_used >= budget)──▶ BudgetLimited
          Active ──(连续3轮受阻)──▶ Blocked
          Active ──(模型确认完成)──▶ Complete
          Active ──(用户暂停)──▶ Paused
```

### 3.3 关键约束

- **BudgetLimited 优先级最高**：即使用户 resume 或模型标记 complete，只要 `tokens_used >= token_budget`，状态始终为 BudgetLimited
- **Blocked 需连续3轮**：必须在原始触发 + 自动续跑中连续3轮遇到相同阻塞条件
- **Complete 需严格验证**：模型必须提供证据证明每个需求都已满足

---

## 4. 三个工具函数

### 4.1 create_goal

```json
{
  "name": "create_goal",
  "description": "Create a goal only when explicitly requested by the user or system/developer instructions; do not infer goals from ordinary tasks.",
  "parameters": {
    "objective": "Required. The concrete objective to start pursuing.",
    "token_budget": "Positive token budget. Omit unless explicitly requested."
  }
}
```

约束：
- 只在用户明确请求时创建，不从普通任务推断
- 如果已有未完成目标，创建失败
- 创建成功后自动设置线程预览

### 4.2 get_goal

```json
{
  "name": "get_goal",
  "description": "Get the current goal for this thread, including status, budgets, token and elapsed-time usage, and remaining token budget.",
  "parameters": {}
}
```

返回当前目标状态、已用 token、剩余预算等。

### 4.3 update_goal

```json
{
  "name": "update_goal",
  "description": "Update the existing goal. Use this tool only to mark the goal achieved or genuinely blocked.",
  "parameters": {
    "status": "complete | blocked"
  }
}
```

约束：
- 只能改为 `complete` 或 `blocked`，不能暂停/恢复（由用户控制）
- `complete`：目标已实现且无剩余工作
- `blocked`：相同阻塞条件连续出现至少3轮
- 不允许因预算耗尽或停止工作而标记完成

---

## 5. 提示词系统

### 5.1 三层提示词架构

```
Layer 1: Base Instructions (models.json)
    ↓ 注入 personality 模板变量
Layer 2: Goal Steering Prompts (templates/goals/*.md)
    ↓ 作为 user message 注入对话流
Layer 3: Tool Descriptions (spec.rs)
    ↓ 工具调用时的参数约束
```

### 5.2 Base Instructions

存储在 `models-manager/models.json`，为每个模型定义：

```
You are Codex, a coding agent based on GPT-5.

{{ personality }}  ← personality_default / personality_friendly / personality_pragmatic

# General
- Engineering judgment
- Frontend guidance
- Editing constraints
- Autonomy and persistence
- Working with the user (commentary/final 双通道)
- Formatting rules
- Final answer instructions
```

关键行为约束：
- 并行工具调用 `multi_tool_use.parallel`
- `apply_patch` 编辑文件
- dirty worktree 不回退用户改动
- 不用 `git reset --hard`

### 5.3 Goal Steering Prompts

通过 `InternalModelContextFragment` 注入，对模型来说是透明的 user message：

#### continuation.md (空闲续跑)

```markdown
Continue working toward the active thread goal.

<objective>{objective}</objective>

Continuation behavior:
- This goal persists across turns
- Keep the full objective intact
- Temporary rough edges are acceptable

Budget:
- Tokens used: {used}
- Token budget: {budget}
- Tokens remaining: {remaining}

Work from evidence: ...
Completion audit: ...
Blocked audit: ...
```

#### budget_limit.md (预算耗尽)

```markdown
The active thread goal has reached its token budget.
<objective>{objective}</objective>
...
Do not start new substantive work for this goal.
Wrap up this turn soon.
```

#### objective_updated.md (目标被编辑)

```markdown
The active thread goal objective was edited by the user.
<untrusted_objective>{objective}</untrusted_objective>
...
Adjust the current turn to pursue the updated objective.
```

### 5.4 注入机制

```rust
// steering.rs
fn goal_context_input_item(prompt: String) -> ResponseItem {
    ContextualUserFragment::into(InternalModelContextFragment::new(
        InternalContextSource::from_static("goal"),
        prompt,
    ))
}
```

Steering prompt 作为 user message 注入到对话流中，模型看到的是用户消息而非系统指令。

---

## 6. 自动续跑机制

### 6.1 触发条件

```
ThreadIdleEvent → GoalExtension::on_thread_idle()
    → GoalRuntimeHandle::continue_if_idle()
        → 检查线程空闲 + 无待处理用户输入 + 非 Plan 模式
        → thread.try_start_turn_if_idle(vec![continuation_steering_item])
```

### 6.2 续跑流程

```
1. 检查 goals_enabled (Feature Flag)
2. 获取 goal_state_lock (Semaphore, 1 permit)
3. 读取当前 goal 状态
4. 如果 status == Active，构建 continuation_steering_item
5. 调用 try_start_turn_if_idle() 启动新 turn
6. 如果启动成功，标记 turn_goal_active
```

### 6.3 用量追踪

```rust
// 每个 tool finish 后
on_tool_finish() → account_active_goal_progress()
    → 计算 time_delta + token_delta
    → 更新 thread_goals 表
    → 如果 tokens_used >= token_budget → status = BudgetLimited
    → 注入 budget_limit_steering_item
```

### 6.4 异常处理

```rust
on_turn_error() → stop_active_goal_for_turn()
    → UsageLimit → status = UsageLimited
    → 其他错误 → status = Blocked (防止循环消耗)
```

---

## 7. 存储层

### 7.1 SQLite 表结构

```sql
CREATE TABLE thread_goals (
    thread_id TEXT PRIMARY KEY,
    goal_id TEXT NOT NULL,
    objective TEXT NOT NULL,
    status TEXT NOT NULL,  -- active/paused/blocked/budget_limited/usage_limited/complete
    token_budget INTEGER,
    tokens_used INTEGER DEFAULT 0,
    time_used_seconds INTEGER DEFAULT 0,
    created_at_ms INTEGER NOT NULL,
    updated_at_ms INTEGER NOT NULL
)
```

### 7.2 关键 SQL 操作

- `insert_thread_goal`: 只在 status=complete 时允许替换
- `account_thread_goal_usage`: 原子性更新 token/time，自动检测 budget crossing
- `update_thread_goal`: 支持 partial update + optimistic concurrency (expected_goal_id)

---

## 8. 用户交互

### 8.1 Slash 命令

| 命令 | 功能 |
|------|------|
| `/goal <objective>` | 创建目标 |
| `/goal` | 显示当前目标摘要 |
| `/goal edit` | 编辑目标文本 |
| `/goal pause` | 暂停目标 |
| `/goal resume` | 恢复暂停的目标 |
| `/goal clear` | 清除目标 |

### 8.2 TUI 显示

```
Goal
Status: active
Objective: optimize the benchmark
Time used: 2m
Tokens used: 12.5K / 50K

Commands: /goal edit, /goal pause, /goal clear
```

### 8.3 状态栏指示器

```
Active    → ⏱ 2m  或  📊 12.5K / 50K
Paused    → ⏸
Blocked   → 🚫
BudgetLimited → 📉 63.9K / 50K tokens
Complete  → ✅ 40K tokens
```

---

## 9. 设计亮点

### 9.1 严格完成审计

模型必须满足以下条件才能标记 complete：
1. 从 objective 推导出具体需求
2. 为每个需求找到权威证据
3. 检查证据是否证明完成、矛盾、不完整或缺失
4. 审计必须证明完成，而非仅仅未发现明显剩余工作

### 9.2 防过早完成

```rust
// update_goal 工具描述
"Do not mark a goal complete merely because its budget is nearly exhausted 
or because you are stopping work."
```

### 9.3 预算保护

- 创建时立即检测 budget crossing
- 每次 tool finish 后自动更新用量
- BudgetLimited 状态优先级最高，不可覆盖

### 9.4 Optimistic Concurrency

```rust
// 使用 expected_goal_id 防止并发写入冲突
WHERE thread_id = ? AND (? IS NULL OR goal_id = ?)
```

---

## 10. 与其他系统的集成

### 10.1 Feature Flag

通过 `features.goals` 控制是否启用：
- 禁用时：不注入 steering prompt，不自动续跑
- 可通过 `--enable goals` 启用

### 10.2 Plan Mode 互斥

Plan 模式下不自动续跑：
```rust
if matches!(input.collaboration_mode.mode, ModeKind::Plan) {
    accounting.clear_current_turn_goal();
    return;
}
```

### 10.3 SubAgent 不支持

Review 等 SubAgent 线程不支持 goal：
```rust
let tools_available_for_thread = input.persistent_thread_state_available
    && !matches!(input.session_source, SessionSource::SubAgent(SubAgentSource::Review));
```

---

## 11. 总结

Codex Goal Mode 的实现体现了以下设计原则：

1. **持久化**：目标跨 turn 持久化到 SQLite，支持 resume/fork
2. **自动驱动**：空闲时自动续跑，减少人工干预
3. **预算控制**：token 预算硬限制，防止意外消耗
4. **严格验证**：完成需要 evidence-based 审计
5. **Extension 架构**：通过 trait 组合实现，解耦核心逻辑
6. **透明注入**：steering prompt 作为 user message 注入，对模型透明

这套机制使得 Codex 能够处理需要多轮交互的复杂任务，同时保持成本可控和结果可靠。
