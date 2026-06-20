# Workflow Mode

Workflow mode lets you orchestrate multiple worker agents using an Elisp DSL, automatically executing multi-phase, parallel, verified complex tasks. It is ideal for code audits, architecture research, multi-role reviews, generate-review loops, and other scenarios requiring multi-agent collaboration.

> **💡 Out-of-the-box orchestration**
> When you enable `--workflows`, a `workflow-elisp` skill is automatically created in your project directory with complete syntax rules, pattern skeletons, and best practices. The AI loads this skill automatically — you don't need to write Elisp manually. Just describe the task in natural language, and the AI will generate the corresponding workflow orchestration script for you.

---

## Quick Start

### Enable Workflow Mode

Use `--workflows` flag in CLI:

```bash
vibecoding --workflows
```

Gateway and ACP also support workflow mode — enable it in their respective configurations.

### Your First Workflow

After launching, you can ask the AI to run a workflow directly. Here is a minimal code audit workflow:

```elisp
(workflow "quick audit"
  (phase "scan"
    (parallel
      (agent "gateway"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/gateway for auth risks. Return file:line evidence.")
      (agent "hermes"
        :mode "plan"
        :tools '("read" "grep" "find")
        :prompt "Audit internal/hermes for auth risks. Return file:line evidence.")))
  (phase "verify"
    (agent "cross-check"
      :mode "plan"
      :tools '("read" "grep")
      :prompt (concat
        (results "scan")
        "\nVerify each claim against source. Drop weak findings. Return prioritized issues."))))
```

This workflow has two phases:
1. **scan**: two agents audit different modules in parallel
2. **verify**: one agent cross-validates results from the previous phase

---

## Core Concepts

### Workflow, Phase, Agent

- **Workflow**: one complete orchestration run, with unique ID, name, and status
- **Phase**: a stage executed in order, containing one or more agent tasks
- **Agent**: a worker agent that performs a concrete task, with its own mode, tools, and prompt

### Execution Model

- Code inside `phase` executes sequentially
- Branches inside `parallel` execute concurrently
- Branches inside `series` execute sequentially (usually unnecessary — default is sequential)
- Agent count is limited by `concurrency` (default 5)

### Result Passing

- `(result "phase.agent")`: get one agent's result
- `(result "phase.agent" :key "r0")` or `(result-key "phase.agent" "r0")`: get a keyed repeated agent result
- `(result-latest "phase.agent")`: get the newest result for a logical agent
- `(results "phase")`: get all results from a phase, concatenated as titled text
- `(results "phase.agent")`: get all keyed results for one logical agent

---

## Syntax Reference

### Where String Literals Are Required

For analyzability and safety, the following names must be direct string literals — not variables, function calls, or generated expressions:

- The first argument of `workflow` (workflow name)
- The first argument of `phase` (phase name)
- The first argument of `agent` (agent name)

✅ Correct:

```elisp
(workflow "audit"
  (phase "scan"
    (agent "worker" :prompt "...")))
```

❌ Wrong:

```elisp
(let ((name "scan")) (phase name ...))   ; variables not allowed
(phase (concat "scan" "-phase"))         ; function calls not allowed
```

### Agent Options

| Option | Type | Description |
|--------|------|-------------|
| `:prompt` | string | **Required**. Task description. |
| `:key` | string | Optional instance key for repeated logical agents, especially inside `while` loops. May be a string expression such as `(format "r%s" i)`. Stored result keys become `phase.agent[key]`. |
| `:mode` | string | Run mode: `plan` / `agent` / `yolo`. Defaults to the parent agent's mode; if unavailable, defaults to `agent`. |
| `:tools` | string list | Available tools. Use `'("read" "grep")` syntax. If omitted, the worker gets the default tool set for its mode, but cannot spawn subagents, delegate, or start nested workflows. |
| `:work-dir` | string | Working directory. Defaults to the current process working directory. |
| `:max-iterations` | number | Maximum worker-agent loop iterations. Omitted, `0`, or negative values default to 50. |
| `:system-prompt-extra` | string | Extra system prompt content. |

### concurrency

Set maximum concurrent agent count:

```elisp
(concurrency 4)
```

If omitted, workflow concurrency defaults to 5. This limits the number of worker agents running at the same time, not the total number of workers.

### log

Append a workflow log entry:

```elisp
(log "Starting scan phase with" (number-to-string 3) "workers")
```

### defun / defmacro

Custom functions and macros are supported, but **only with fixed parameter lists**. You cannot use `&optional`, `&rest`, `&body`, etc.

✅ Correct:

```elisp
(defun wrap-prompt (prefix suffix)
  (concat prefix "\n\n" suffix))
```

❌ Wrong:

```elisp
(defun join (&rest parts) ...)           ; &rest not supported
(defmacro with-worker (&body body) ...)  ; &body not supported
```

---

## Common Patterns

### 1. Research and Audit

Scan multiple areas in parallel, then verify collectively.

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

### 2. Serial Pipeline

Design → Implement → Verify. Each step depends on the previous step's output.

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

### 3. Decision Routing

Classify first, then take different paths based on the classification.

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

### 4. Bounded Loop

Test-fix loop with a maximum iteration count.

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

### 5. Expert Panel

Multiple peer agents analyze independently, then reconcile.

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

### 6. Master-Slave Small Team

One coordinator decomposes the work, multiple specialists execute in parallel.

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

## Best Practices

### 1. Prefer Plan Mode

For audit, research, and review tasks, prefer `:mode "plan"` with read-only tools (`read`, `grep`, `find`). Use `agent` or `yolo` only when you explicitly need code edits.

### 2. Give Each Agent Clear Boundaries

- Tell the agent exactly which files/modules it is responsible for
- Specify output format (e.g. "return only file:line evidence")
- Specify stop conditions

### 3. Use Narrow Tool Sets

Give each agent only the tools it needs to complete the task. Don't give `edit` and `bash` to read-only tasks.

### 4. Result Keys Are "phase.agent"

`result` and `results` use **literal names**, not variables. Phase and agent names should be stable and meaningful.

### 5. Always Have a Verification Phase

Multi-agent parallel results always need a summary/verification phase. Worker results are "evidence," not "conclusions."

### 6. Control Concurrency

Default is 5 concurrent agents. If all agents call the same LLM provider, watch out for rate limits — you may want to lower it to 2-3.

### 7. Make Defaults Explicit When They Matter

For safety-sensitive or long-running workers, set `:mode`, `:tools`, and `:max-iterations` explicitly instead of relying on inherited defaults. `:mode` inherits the parent mode, `:tools` omission grants the default tool set for the mode, and `:max-iterations` falls back to 50 worker-agent loop iterations.

`workflow_run` `timeoutSeconds` is a tool parameter, not an `(agent ...)` DSL option. Use it for long workflows; worker agents do not have a per-worker `:timeout` option.

### 8. Don't Use Workflows for Small Tasks

For simple sequential tasks or anything a single agent can handle, regular conversation is cheaper and faster. Workflow overhead is non-trivial.

---

## Pitfalls to Avoid

### ❌ Don't Generate Names with Variables

```elisp
;; WRONG — phase name must be a string literal
(let ((phase-name "scan"))
  (phase phase-name ...))

;; CORRECT
(phase "scan" ...)
```

For repeated workers, keep the `agent` name literal and put the round identity in `:key`:

```elisp
(agent "worker"
  :key (format "r%s" i)
  :prompt "...")
```

### ❌ Don't Use Unquoted Lists in :tools

```elisp
;; WRONG — will be evaluated as a function call
:tools ("read" "grep")

;; CORRECT
:tools '("read" "grep")
```

### ❌ Don't Use &rest / &optional / &body

```elisp
;; WRONG — only fixed argument lists are supported
(defun join (&rest parts) ...)

;; CORRECT
(defun join3 (a b c) (concat a b c))
```

### ❌ Don't Expect Workers to Spawn Sub-Agents

Workflow worker agents have multi-agent, delegate, and workflows disabled by default. If you need multi-level orchestration, do it at the workflow DSL level.

### ❌ Don't Use Nonexistent Worker Options

The workflow DSL does not support per-worker `:model`, `:thinking-level`, `:max-tokens`, `:tool-execution-mode`, or `:timeout`. Model, thinking level, output token limit, sandbox behavior, and session behavior come from the surrounding configuration. Worker tool calls execute in parallel by default when the model emits multiple tool calls in the same turn.

### ❌ Don't Write Unbounded Loops

Loops must always have a maximum iteration count and a clear exit condition.

---

## Related Commands

Both CLI and Gateway support `/workflows` commands to view and manage workflow runs:

- `/workflows` — list recent workflow runs
- `/workflows <id>` — view detailed status of a workflow
- `/workflows cancel <id>` — cancel a running workflow

You can also use tool calls:

- `workflow_run` — run a workflow
- `workflow_status` — check status
- `workflow_cancel` — cancel a run

`workflow_run` accepts an optional `timeoutSeconds` parameter. Omit it to use the default tool timeout, set a positive value for long bounded workflows, or set `0` only for intentional continuous workflows that should not be interrupted by the agent-level deadline. This is separate from worker `:max-iterations` and is not an `(agent ...)` option. In Gateway mode, `requestTimeoutSeconds` is still an outer HTTP request deadline and must also be sized for long-running requests.

---

## Further Learning

When you enable workflow mode, a `.skills/workflow-elisp/` directory is automatically created under your project root, containing the full skill documentation and 8 pattern reference files. You can edit these files directly to customize your workflow authoring guidance.
