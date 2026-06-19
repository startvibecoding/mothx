package workflow

import (
	"fmt"
	"os"
	"path/filepath"
)

const SkillName = "workflow-elisp"

const defaultSkillContent = `# Workflow Elisp

Use this skill when authoring workflow_run source for dynamic workflow mode.

The workflow_run source must be one complete raw Elisp form. Do not wrap it in
Markdown fences.

## Progressive References

Start with the loaded core rules. Load scenario files only when the task matches
that pattern.

### 1. Core Rules and Skeletons (references/00-core-rules.md) [已加载]
### 2. Research and Investigation Workflows (references/01-research.md) [待按需加载]
### 3. Serial and Parallel Composition (references/02-serial-parallel.md) [待按需加载]
### 4. Decision Routing and Branching (references/03-decision-routing.md) [待按需加载]
### 5. Continuous Loops and Iterative Tasks (references/04-continuous-loops.md) [待按需加载]
### 6. Horizontal Multi-Agent Collaboration (references/05-horizontal-collaboration.md) [待按需加载]
### 7. Master-Slave Small Team Workflows (references/06-master-slave-team.md) [待按需加载]
### 8. Evaluator-Optimizer and Critic Loops (references/07-evaluator-optimizer.md) [待按需加载]
### 9. Governance and Human Checkpoints (references/08-governance-checkpoints.md) [待按需加载]

## Pattern Selection

- Simple ordered task: load serial and parallel composition.
- Broad research or audit: load research workflows.
- Distinct input classes or risk levels: load decision routing.
- Repeat-until-good-enough work: load continuous loops and evaluator-optimizer.
- Several peer experts checking one problem: load horizontal collaboration.
- One coordinator decomposes work for specialists: load master-slave team.
- High-impact or user-sensitive decisions: load governance checkpoints.

## Non-Negotiable Constraints

- workflow, phase, and agent names must be string literals.
- defun and defmacro only support fixed parameter lists. Do not use &optional,
  &rest, &body, or any argument marker beginning with &.
- Tool lists must be quoted string lists: '("read" "grep" "find").
- Every agent needs a bounded prompt, expected output, and stop condition.
`

var defaultReferenceFiles = map[string]string{
	"references/00-core-rules.md": `# Core Rules and Skeletons

## Hard Syntax Constraints

- The first argument of workflow must be a string literal.
- The first argument of phase must be a string literal.
- The first argument of agent must be a string literal.
- Do not generate workflow, phase, or agent names with variables, function calls,
  concat, format, let bindings, or other expressions.
- defun only supports fixed parameter lists. Do not use &optional, &rest, &key,
  &body, &allow-other-keys, or any parameter marker beginning with &.
- defmacro only supports fixed parameter lists. Do not use &optional, &rest,
  &key, &body, &allow-other-keys, or any parameter marker beginning with &.
- Use keyword/value pairs for agent options. Every agent option key must be an
  unquoted keyword symbol such as :prompt or :tools.
- The :tools value must be a quoted list of string literals, for example:
  '("read" "grep" "find")

Invalid examples:

    (let ((name "scan")) (phase name ...))
    (agent (concat "worker-" suffix) :prompt "...")
    (defun join (&rest parts) ...)
    (defmacro with-worker (&body body) ...)

## Supported Workflow Forms

- (workflow "name" body...) defines one workflow run.
- (concurrency n) sets the maximum number of concurrent worker agents.
- (phase "name" body...) groups sequential phases and records phase state.
- (parallel expr...) evaluates independent branches concurrently.
- (series expr...) evaluates branches sequentially.
- (agent "name" :prompt "..." [:mode "plan|agent|yolo"] [:work-dir "..."]
  [:tools '("read" "grep")] [:max-iterations n]
  [:system-prompt-extra "..."]) runs one worker agent and returns its final text.
- (result "phase.agent") returns one prior worker result.
- (results "phase") returns all results from a prior phase as text.
- (log "message" ...) appends a workflow log entry.

## Minimal Valid Skeleton

    (workflow "auth audit"
      (concurrency 2)
      (phase "scan"
        (parallel
          (agent "gateway"
            :mode "plan"
            :tools '("read" "grep" "find")
            :prompt "Audit internal/gateway authentication risks. Return file:line evidence.")
          (agent "hermes"
            :mode "plan"
            :tools '("read" "grep" "find")
            :prompt "Audit internal/hermes authentication risks. Return file:line evidence.")))
      (phase "verify"
        (agent "cross-check"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (results "scan")
            "\nVerify the evidence, reject weak claims, and list final risks."))))

## Generation Checklist

- Source starts with (workflow "literal-name" ...).
- Every phase and agent has a literal string name.
- Parentheses and strings are balanced.
- Every agent option is a keyword/value pair.
- Every agent has :prompt.
- :tools uses quoted list syntax exactly like '("read" "grep").
- Prior outputs are referenced with (result "phase.agent") or (results "phase").
`,
	"references/01-research.md": `# Research and Investigation Workflows

Use this when the task is discovery-heavy: architecture review, risk audit,
competitive research, incident investigation, or "find all places where..."

The pattern is: split independent research lanes, collect evidence, then run a
verification phase that rejects weak claims.

## Codebase Audit

    (workflow "security research"
      (concurrency 4)
      (phase "scan"
        (parallel
          (agent "entrypoints"
            :mode "plan"
            :tools '("read" "grep" "find")
            :prompt "Find public entrypoints and request validation paths. Return file:line evidence only.")
          (agent "storage"
            :mode "plan"
            :tools '("read" "grep" "find")
            :prompt "Inspect persistence and session storage for trust boundary risks. Return file:line evidence.")
          (agent "tools"
            :mode "plan"
            :tools '("read" "grep" "find")
            :prompt "Inspect tool execution paths for sandbox, approval, and path validation risks.")))
      (phase "verify"
        (agent "cross-check"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (results "scan")
            "\nVerify each claim against source. Drop speculative findings. Return prioritized issues."))))

## External Topic Research

For web or document research, split by source class or question, not by arbitrary
page count.

    (workflow "market research"
      (concurrency 3)
      (phase "research"
        (parallel
          (agent "primary-sources"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Review provided primary source files. Extract factual claims and citations.")
          (agent "competitors"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Review competitor notes. Extract positioning, pricing, and gaps.")
          (agent "risks"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Identify legal, operational, and implementation risks from the provided docs.")))
      (phase "synthesis"
        (agent "brief"
          :mode "plan"
          :tools '("read")
          :prompt (concat
            (results "research")
            "\nWrite a concise brief with source-grounded conclusions and unresolved questions."))))
`,
	"references/02-serial-parallel.md": `# Serial and Parallel Composition

Use serial phases when later work depends on earlier output. Use parallel inside
a phase when branches are independent and can be reconciled later.

## Prompt Chaining / Serial Pipeline

    (workflow "design then implement"
      (phase "design"
        (agent "designer"
          :mode "plan"
          :tools '("read" "grep" "find")
          :prompt "Design the minimal change. Return files, behavior, risks, and tests. Do not edit."))
      (phase "implement"
        (agent "builder"
          :mode "agent"
          :tools '("read" "grep" "edit" "write")
          :prompt (concat
            "Implement this plan exactly. Keep edits scoped.\n\n"
            (result "design.designer"))))
      (phase "verify"
        (agent "verifier"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            "Review the implementation against the design. Report issues only.\n\n"
            (results "implement")))))

## Parallel Sectioning

    (workflow "parallel review"
      (concurrency 3)
      (phase "review"
        (parallel
          (agent "api"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Review API compatibility. Return concrete regressions.")
          (agent "tests"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Review test coverage gaps. Return missing cases.")
          (agent "docs"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Review docs and user-facing behavior mismatch.")))
      (phase "merge"
        (agent "triage"
          :mode "plan"
          :prompt (concat (results "review") "\nDeduplicate and prioritize findings."))))
`,
	"references/03-decision-routing.md": `# Decision Routing and Branching

Use routing when distinct classes of input need different tools, modes, or
prompts. The current workflow DSL supports Elisp if/cond, but workflow, phase,
and agent names inside branches must still be string literals.

## Risk-Based Route

    (workflow "risk routed task"
      (phase "classify"
        (agent "classifier"
          :mode "plan"
          :tools '("read" "grep")
          :prompt "Classify the request as LOW, MEDIUM, or HIGH risk. Return one label and rationale."))
      (phase "route"
        (if (string= (result "classify.classifier") "HIGH")
            (agent "high-risk-review"
              :mode "plan"
              :tools '("read" "grep" "find")
              :prompt "Perform conservative high-risk analysis. Require evidence and list approval checkpoints.")
          (agent "standard-review"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Perform standard bounded review and return direct recommendations."))))

## Multi-Route with cond

    (workflow "request router"
      (phase "classify"
        (agent "classifier"
          :mode "plan"
          :prompt "Return exactly one token: BUG, DOCS, REFACTOR, or UNKNOWN."))
      (phase "handle"
        (cond
          ((string= (result "classify.classifier") "BUG")
            (agent "bug-handler"
              :mode "agent"
              :tools '("read" "grep" "edit")
              :prompt "Investigate and fix the bug with minimal edits."))
          ((string= (result "classify.classifier") "DOCS")
            (agent "docs-handler"
              :mode "agent"
              :tools '("read" "grep" "edit")
              :prompt "Update docs for the requested behavior."))
          (t
            (agent "fallback"
              :mode "plan"
              :tools '("read" "grep")
              :prompt "Clarify unknown route and recommend next steps.")))))

Prefer routing labels that are exact strings. If classifier output may include
rationale, ask it to put the label on the first line and route conservatively.
`,
	"references/04-continuous-loops.md": `# Continuous Loops and Iterative Tasks

Use loops for bounded repeated work: test-fix cycles, repeated search until a
coverage threshold, or periodic audit batches. Always include a hard iteration
limit and a clear stop condition.

## Bounded Test-Fix Loop

    (workflow "bounded fix loop"
      (concurrency 1)
      (let ((i 0)
            (status "NEEDS_WORK"))
        (while (and (< i 3) (not (string= status "DONE")))
          (phase "iteration"
            (agent "worker"
              :mode "agent"
              :tools '("read" "grep" "edit")
              :prompt (concat
                "Iteration " (format "%s" i) ". Fix only the highest-confidence issue. "
                "Return DONE if no issue remains, otherwise NEEDS_WORK plus evidence."))
            (agent "checker"
              :mode "plan"
              :tools '("read" "grep")
              :prompt (concat
                (result "iteration.worker")
                "\nCheck whether the objective is complete. Return exactly DONE or NEEDS_WORK.")))
          (setq status (result "iteration.checker"))
          (setq i (+ i 1)))
        (phase "final"
          (agent "summary"
            :mode "plan"
            :prompt (concat
              "Loop stopped after bounded iterations. Final checker status: "
              status
              "\nSummarize changes, evidence, and residual risk.")))))

Note: repeated phase and agent literal names overwrite result keys from prior
iterations. Use loop logs or final summaries when you only need the latest
iteration. If you need all iteration outputs, ask each worker to append a compact
history into its final response or split into explicit phase names.

## Persistent Monitoring Batch

For unattended or cron-triggered workflows, make each run finite:

    (workflow "daily regression audit"
      (phase "collect"
        (agent "collector"
          :mode "plan"
          :tools '("read" "grep" "find")
          :prompt "Inspect today's changed files and list likely regression areas."))
      (phase "audit"
        (agent "auditor"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (result "collect.collector")
            "\nAudit the listed areas. Return only actionable findings."))))
`,
	"references/05-horizontal-collaboration.md": `# Horizontal Multi-Agent Collaboration

Use this when peer specialists should independently analyze the same problem and
then reconcile. It is useful for architecture decisions, security reviews,
product tradeoffs, and adversarial checks.

## Peer Expert Panel

    (workflow "expert panel"
      (concurrency 4)
      (phase "positions"
        (parallel
          (agent "security"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Analyze the proposal from a security perspective. Return must-fix risks and acceptable tradeoffs.")
          (agent "maintainability"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Analyze maintainability, ownership boundaries, and future migration cost.")
          (agent "performance"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Analyze runtime, memory, and scaling implications.")
          (agent "product"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Analyze user-facing behavior, migration risk, and support burden.")))
      (phase "reconcile"
        (agent "moderator"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (results "positions")
            "\nFind agreements, contradictions, and a final recommendation with confidence."))))

## Voting Variant

Run the same review prompt through independent agents when diversity matters:

    (workflow "three reviewer vote"
      (concurrency 3)
      (phase "vote"
        (parallel
          (agent "reviewer-a" :mode "plan" :tools '("read" "grep") :prompt "Review for correctness. Return PASS or FAIL with evidence.")
          (agent "reviewer-b" :mode "plan" :tools '("read" "grep") :prompt "Review for correctness. Return PASS or FAIL with evidence.")
          (agent "reviewer-c" :mode "plan" :tools '("read" "grep") :prompt "Review for correctness. Return PASS or FAIL with evidence.")))
      (phase "decision"
        (agent "judge"
          :mode "plan"
          :prompt (concat (results "vote") "\nDecide PASS only if evidence supports it."))))
`,
	"references/06-master-slave-team.md": `# Master-Slave Small Team Workflows

Use this when a coordinator should decompose work, then specialists execute
bounded assignments. The parent workflow is the real master; worker prompts
should not ask workers to spawn or manage further sub-agents.

## Planner Assigns Specialist Tasks

    (workflow "small team change"
      (concurrency 3)
      (phase "plan"
        (agent "master"
          :mode "plan"
          :tools '("read" "grep" "find")
          :prompt "Decompose the request into API, storage, UI, and test tasks. Return scoped instructions for each role."))
      (phase "execute"
        (parallel
          (agent "api-worker"
            :mode "agent"
            :tools '("read" "grep" "edit")
            :prompt (concat
              "You own API/server code only. Do not edit UI or docs.\n\n"
              (result "plan.master")))
          (agent "storage-worker"
            :mode "agent"
            :tools '("read" "grep" "edit")
            :prompt (concat
              "You own persistence/config/session code only. Do not edit UI.\n\n"
              (result "plan.master")))
          (agent "test-worker"
            :mode "agent"
            :tools '("read" "grep" "edit")
            :prompt (concat
              "You own tests only unless a tiny fixture change is required.\n\n"
              (result "plan.master")))))
      (phase "integrate"
        (agent "master-review"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (results "execute")
            "\nReview integration boundaries, conflicts, missing tests, and final risks."))))

Rules:

- Give every worker ownership boundaries.
- Tell workers they are not alone in the codebase.
- Prefer narrow tools for each worker.
- Add a final master-review phase before reporting success.
`,
	"references/07-evaluator-optimizer.md": `# Evaluator-Optimizer and Critic Loops

Use this when quality improves through explicit critique: writing, migration
plans, design docs, policy analysis, or complex search.

## Generate, Critique, Revise

    (workflow "proposal refinement"
      (phase "draft"
        (agent "writer"
          :mode "plan"
          :tools '("read" "grep")
          :prompt "Draft the proposal. Include assumptions, tradeoffs, and open questions."))
      (phase "critique"
        (agent "critic"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            (result "draft.writer")
            "\nCritique against correctness, completeness, operational risk, and testability.")))
      (phase "revise"
        (agent "reviser"
          :mode "plan"
          :tools '("read" "grep")
          :prompt (concat
            "Revise the draft using this critique. Preserve strong parts; fix weak claims.\n\nDRAFT:\n"
            (result "draft.writer")
            "\n\nCRITIQUE:\n"
            (result "critique.critic")))))

## Bounded Optimizer Loop

Use a loop only if each critique has objective criteria. Keep max iterations
small and put final acceptance in a separate phase.
`,
	"references/08-governance-checkpoints.md": `# Governance and Human Checkpoints

Workflow workers cannot directly ask the user mid-run. For high-impact tasks,
split the workflow so it produces a decision packet, then the parent agent asks
the user before running a second workflow that edits or executes.

## Decision Packet First

    (workflow "migration decision packet"
      (concurrency 3)
      (phase "assess"
        (parallel
          (agent "benefits"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "List concrete benefits with evidence.")
          (agent "risks"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "List operational, compatibility, security, and rollback risks.")
          (agent "costs"
            :mode "plan"
            :tools '("read" "grep")
            :prompt "Estimate implementation, testing, migration, and support cost.")))
      (phase "packet"
        (agent "decision-packet"
          :mode "plan"
          :prompt (concat
            (results "assess")
            "\nProduce: recommendation, alternatives, explicit approval question, and rollback plan."))))

After this workflow returns, ask the user for approval in the main conversation.
Only then run an implementation workflow.

## Governance Checklist

- Prefer plan mode before edits.
- Make approval points explicit.
- Separate decision workflows from execution workflows.
- Include rollback and observability requirements.
- Record unresolved assumptions in the final packet.
`,
}

// EnsureProjectSkill creates the project-local workflow skill if it does not
// already exist. Existing SKILL.md or skill.md files are never overwritten so
// users can customize the workflow authoring instructions.
func EnsureProjectSkill(projectRoot string) (path string, created bool, err error) {
	if projectRoot == "" {
		return "", false, fmt.Errorf("project root is required")
	}
	skillDir := filepath.Join(projectRoot, ".skills", SkillName)
	upperPath := filepath.Join(skillDir, "SKILL.md")
	lowerPath := filepath.Join(skillDir, "skill.md")

	if _, err := os.Stat(upperPath); err == nil {
		if err := ensureReferenceFiles(skillDir); err != nil {
			return "", false, err
		}
		return upperPath, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if _, err := os.Stat(lowerPath); err == nil {
		if err := ensureReferenceFiles(skillDir); err != nil {
			return "", false, err
		}
		return lowerPath, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", false, err
	}
	f, err := os.OpenFile(upperPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return upperPath, false, nil
		}
		return "", false, err
	}
	if _, err := f.WriteString(defaultSkillContent); err != nil {
		_ = f.Close()
		return "", false, err
	}
	if err := f.Close(); err != nil {
		return "", false, err
	}
	if err := ensureReferenceFiles(skillDir); err != nil {
		return "", false, err
	}
	return upperPath, true, nil
}

func ensureReferenceFiles(skillDir string) error {
	for relPath, content := range defaultReferenceFiles {
		path := filepath.Join(skillDir, relPath)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
		if _, err := f.WriteString(content); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
