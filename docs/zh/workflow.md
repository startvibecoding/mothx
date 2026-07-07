# Workflow 模式

Workflow 模式让你可以用 Elisp DSL 编排多个 worker agent，自动执行分阶段、并行、带验证的复杂任务。它适合代码审计、架构调研、多角色评审、生成-评审循环等需要多智能体协作的场景。

> **💡 开箱即用的编排能力**
> 启用 `--workflows` 后，项目目录会自动创建 `workflow-elisp` skill，包含完整的语法规则、模式骨架和最佳实践。AI 会自动加载这个 skill，无需你手动编写 Elisp——你只需要用自然语言描述任务，AI 会自动生成对应的 workflow 编排脚本。

---

## 快速开始

### 启用 workflow 模式

CLI 下使用 `--workflows` 参数启用：

```bash
mothx --workflows
```

Serve 和 ACP 也支持 workflow 模式，在对应配置中开启即可。

### 第一个 workflow

启动后，你可以直接让 AI 运行 workflow。下面是一个最小的代码审计 workflow：

```elisp
(workflow "quick audit"
  (phase "scan"
    (parallel
      (agent "api"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/serve/openaiapi for auth risks. Return file:line evidence.")
      (agent "channels"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/serve/channels for auth risks. Return file:line evidence.")))
  (phase "verify"
    (agent "cross-check"
      :mode "plan"
      :tools '("read" "grep")
      :prompt (concat
        (results "scan")
        "\nVerify each claim against source. Drop weak findings. Return prioritized issues."))))
```

这个 workflow 有两个阶段：
1. **scan**：两个 agent 并行审计不同模块
2. **verify**：一个 agent 交叉验证前一阶段的结果

---

## 核心概念

### Workflow、Phase、Agent

- **Workflow**：一次完整的编排运行，有唯一 ID、名称、状态
- **Phase**：阶段，按顺序执行，每个阶段包含一个或多个 agent 任务
- **Agent**：worker agent，执行具体任务，有自己的 mode、tools、prompt

### 执行模型

- `phase` 内的代码按顺序执行
- `parallel` 内的分支并发执行
- `series` 内的分支串行执行（通常不需要，默认就是串行）
- agent 数量受 `concurrency` 限制（默认 5）

### 结果传递

- `(result "phase.agent")`：获取某个 agent 的结果
- `(result "phase.agent" :key "r0")` 或 `(result-key "phase.agent" "r0")`：获取重复 agent 的指定实例结果
- `(result-latest "phase.agent")`：获取某个逻辑 agent 的最新结果
- `(results "phase")`：获取某个阶段所有 agent 的结果，拼接为带标题的文本
- `(results "phase.agent")`：获取某个逻辑 agent 的所有 keyed 历史结果

---

## 语法参考

### 必须是字符串字面量的地方

为了可分析性和安全性，以下名称必须是直接的字符串字面量，不能用变量、函数调用或表达式生成：

- `workflow` 的第一个参数（workflow 名称）
- `phase` 的第一个参数（阶段名称）
- `agent` 的第一个参数（agent 名称）

✅ 正确：

```elisp
(workflow "audit"
  (phase "scan"
    (agent "worker" :prompt "...")))
```

❌ 错误：

```elisp
(let ((name "scan")) (phase name ...))   ; 变量不行
(phase (concat "scan" "-phase"))         ; 函数调用不行
```

### agent 选项

| 选项 | 类型 | 说明 |
|------|------|------|
| `:prompt` | string | **必填**，任务描述 |
| `:key` | string | 重复逻辑 agent 的可选实例 key，尤其用于 `while` 循环。可以是字符串表达式，例如 `(format "r%s" i)`。结果 key 会保存为 `phase.agent[key]` |
| `:mode` | string | 运行模式：`plan` / `agent` / `yolo`，默认继承父 agent；不可用时默认 `agent` |
| `:tools` | string list | 可用工具列表，用 `'("read" "grep")` 语法。省略时使用当前 mode 的默认工具集，但 worker 不能 spawn 子 agent、delegate 或启动嵌套 workflow |
| `:work-dir` | string | 工作目录，默认是当前进程工作目录 |
| `:max-iterations` | number | worker-agent 最大循环次数。省略、`0` 或负数时默认 50 |
| `:system-prompt-extra` | string | 额外的 system prompt 内容 |

### concurrency

设置最大并发 agent 数：

```elisp
(concurrency 4)
```

省略时默认并发数为 5。它限制的是同时运行的 worker agent 数量，不是 worker 总数。

### log

记录工作流日志：

```elisp
(log "Starting scan phase with" (number-to-string 3) "workers")
```

### defun / defmacro

支持自定义函数和宏，但**只支持固定参数列表**，不能用 `&optional`、`&rest`、`&body` 等。

✅ 正确：

```elisp
(defun wrap-prompt (prefix suffix)
  (concat prefix "\n\n" suffix))
```

❌ 错误：

```elisp
(defun join (&rest parts) ...)           ; &rest 不支持
(defmacro with-worker (&body body) ...)  ; &body 不支持
```

---

## 常用模式

### 1. 研究与审计

并行扫描多个区域，然后统一验证。

```elisp
(workflow "security audit"
  (concurrency 4)
  (phase "scan"
    (parallel
      (agent "entrypoints" :mode "plan" :tools '("read" "grep" "find")
        :prompt "Find public entrypoints and validation paths.")
      (agent "storage" :mode "plan" :tools '("read" "grep" "find")
        :prompt "Inspect persistence for trust boundary risks.")
      (agent "tools" :mode "plan" :tools '("read" "grep" "find")
        :prompt "Inspect tool execution paths for sandbox risks.")))
  (phase "verify"
    (agent "cross-check" :mode "plan" :tools '("read" "grep")
      :prompt (concat
        (results "scan")
        "\nVerify each claim. Drop speculative findings. Return prioritized issues."))))
```

### 2. 串行流水线

设计 → 实现 → 验证，每一步依赖前一步输出。

```elisp
(workflow "design then implement"
  (phase "design"
    (agent "designer" :mode "plan" :tools '("read" "grep" "find")
      :prompt "Design the minimal change. Return files, behavior, risks, and tests."))
  (phase "implement"
    (agent "builder" :mode "agent" :tools '("read" "grep" "edit" "write")
      :prompt (concat "Implement this plan exactly.\n\n" (result "design.designer"))))
  (phase "verify"
    (agent "verifier" :mode "plan" :tools '("read" "grep")
      :prompt (concat "Review implementation against the design. Report issues only.\n\n" (results "implement")))))
```

### 3. 决策路由

先分类，再根据分类走不同路径。

```elisp
(workflow "risk routed task"
  (phase "classify"
    (agent "classifier" :mode "plan" :tools '("read" "grep")
      :prompt "Classify the request as LOW, MEDIUM, or HIGH risk. Return one label."))
  (phase "route"
    (if (string= (result "classify.classifier") "HIGH")
        (agent "high-risk-review" :mode "plan" :tools '("read" "grep" "find")
          :prompt "Perform conservative high-risk analysis. List approval checkpoints.")
      (agent "standard-review" :mode "plan" :tools '("read" "grep")
        :prompt "Perform standard bounded review and return direct recommendations."))))
```

### 4. 有界循环

测试-修复循环，有最大迭代次数。

```elisp
(workflow "bounded fix loop"
  (concurrency 1)
  (let ((i 0)
        (status "NEEDS_WORK")
        (last-worker ""))
    (while (and (< i 3) (not (string= status "DONE")))
      (phase "iteration"
        (agent "worker" :mode "agent" :tools '("read" "grep" "edit")
          :key (format "r%s" i)
          :prompt (concat "Iteration " (format "%s" i) ". Fix the highest-confidence issue."))
        (setq last-worker (result-latest "iteration.worker"))
        (agent "checker" :mode "plan" :tools '("read" "grep")
          :key (format "r%s" i)
          :prompt (concat last-worker "\nCheck if complete. Return DONE or NEEDS_WORK.")))
      (setq status (result-latest "iteration.checker"))
      (setq i (+ i 1)))
    (phase "final"
      (agent "summary" :mode "plan"
        :prompt (concat "Final status: " status "\nLast worker result:\n" last-worker
          "\nSummarize changes and residual risk.")))))
```

### 5. 专家评审团

多个 peer agent 独立分析，然后汇总。

```elisp
(workflow "expert panel"
  (concurrency 4)
  (phase "positions"
    (parallel
      (agent "security" :mode "plan" :tools '("read" "grep")
        :prompt "Analyze from a security perspective.")
      (agent "maintainability" :mode "plan" :tools '("read" "grep")
        :prompt "Analyze maintainability and ownership boundaries.")
      (agent "performance" :mode "plan" :tools '("read" "grep")
        :prompt "Analyze runtime and scaling implications.")
      (agent "product" :mode "plan" :tools '("read" "grep")
        :prompt "Analyze user-facing behavior and support burden.")))
  (phase "reconcile"
    (agent "moderator" :mode "plan" :tools '("read" "grep")
      :prompt (concat (results "positions")
        "\nFind agreements, contradictions, and a final recommendation."))))
```

### 6. 主从小团队

一个协调者拆解任务，多个专家并行执行。

```elisp
(workflow "small team change"
  (concurrency 3)
  (phase "plan"
    (agent "master" :mode "plan" :tools '("read" "grep" "find")
      :prompt "Decompose the request into API, storage, UI, and test tasks."))
  (phase "execute"
    (parallel
      (agent "api-worker" :mode "agent" :tools '("read" "grep" "edit")
        :prompt (concat "You own API/server code only.\n\n" (result "plan.master")))
      (agent "storage-worker" :mode "agent" :tools '("read" "grep" "edit")
        :prompt (concat "You own persistence code only.\n\n" (result "plan.master")))
      (agent "test-worker" :mode "agent" :tools '("read" "grep" "edit")
        :prompt (concat "You own tests only.\n\n" (result "plan.master")))))
  (phase "integrate"
    (agent "master-review" :mode "plan" :tools '("read" "grep")
      :prompt (concat (results "execute")
        "\nReview integration boundaries, conflicts, missing tests, and final risks."))))
```

---

## 最佳实践

### 1. 优先使用 plan 模式

审计、调研、评审类任务优先用 `:mode "plan"` 配合只读工具（`read`、`grep`、`find`）。只有明确需要修改代码时才用 `agent` 或 `yolo` 模式。

### 2. 给每个 agent 明确的边界

- 明确告诉 agent 它负责哪些文件/模块
- 明确输出格式（比如"只返回 file:line 证据"）
- 明确停止条件

### 3. 用窄工具集

只给 agent 完成任务必需的工具。只读任务就不要给 `edit` 和 `bash`。

### 4. 结果 key 是 "phase.agent"

`result` 和 `results` 用的是**字面量名称**，不是变量。阶段和 agent 的名字要稳定、有意义。

### 5. 验证阶段不可少

多 agent 并行的结果一定要有一个汇总/验证阶段。Worker 的结果是"证据"，不是"结论"。

### 6. 控制并发度

默认 5 个并发。如果你的 agent 都调用同一个 LLM provider，注意 rate limit，可以调低到 2-3。

### 7. 重要默认值要显式设置

对于安全敏感或长时间运行的 worker，不要依赖继承默认值，显式设置 `:mode`、`:tools` 和 `:max-iterations`。`:mode` 会继承父 agent 模式，`:tools` 省略时会使用当前 mode 的默认工具集，`:max-iterations` 会回退到 50 次 worker-agent 循环。

`workflow_run` 的 `timeoutSeconds` 是工具参数，不是 `(agent ...)` DSL 选项。长 workflow 应设置它；worker agent 没有 per-worker `:timeout` 选项。

### 8. 小任务不要用 workflow

简单的顺序任务、单 agent 就能搞定的事，直接用普通对话更便宜更快。Workflow 的 overhead 不小。

---

## 避坑指南

### ❌ 不要用变量生成名称

```elisp
;; 错误 — phase 名称必须是字符串字面量
(let ((phase-name "scan"))
  (phase phase-name ...))

;; 正确
(phase "scan" ...)
```

对于重复 worker，保持 `agent` 名称为字面量，把轮次身份放到 `:key`：

```elisp
(agent "worker"
  :key (format "r%s" i)
  :prompt "...")
```

### ❌ 不要在 :tools 里用未加引号的列表

```elisp
;; 错误 — 会被当作函数调用
:tools ("read" "grep")

;; 正确
:tools '("read" "grep")
```

### ❌ 不要用 &rest / &optional / &body

```elisp
;; 错误 — 只支持固定参数列表
(defun join (&rest parts) ...)

;; 正确
(defun join3 (a b c) (concat a b c))
```

### ❌ 不要期望 worker 自己 spawn 子 agent

Workflow 的 worker agent 默认禁用了 multi-agent、delegate 和 workflows。如果需要多层编排，在 workflow DSL 层面做。

### ❌ 不要使用不存在的 worker 选项

Workflow DSL 不支持 per-worker `:model`、`:thinking-level`、`:max-tokens`、`:tool-execution-mode` 或 `:timeout`。模型、thinking level、输出 token 上限、sandbox 行为和 session 行为来自外层配置。模型在同一轮发起多个 tool call 时，worker 默认并行执行这些工具调用。

### ❌ 不要写无界循环

循环一定要有最大迭代次数和明确的退出条件。

---

## 相关命令

CLI 和 Serve 中都可以用 `/workflows` 命令查看和管理 workflow 运行：

- `/workflows` — 列出最近的 workflow 运行
- `/workflows <id>` — 查看某个 workflow 的详细状态
- `/workflows cancel <id>` — 取消正在运行的 workflow

也可以通过工具调用：

- `workflow_run` — 运行一个 workflow
- `workflow_status` — 查看状态
- `workflow_cancel` — 取消运行

`workflow_run` 支持可选的 `timeoutSeconds` 参数。省略时使用默认工具超时；有明确上限的长任务可以设置正数秒数；只有确实需要持续运行的 workflow 才设置为 `0`，表示不加 agent 级 deadline。它和 worker 的 `:max-iterations` 是两套独立限制，也不是 `(agent ...)` 选项。Serve 模式下，`requestTimeoutSeconds` 仍然是外层 HTTP 请求 deadline，长时间请求也需要同步调大这个配置。

---

## 进一步学习

启用 workflow 模式后，项目目录下会自动创建 `.skills/workflow-elisp/` 目录，里面有完整的 skill 文档和 8 个模式参考文件。你可以直接修改这些文件来自定义 workflow 编写指引。
