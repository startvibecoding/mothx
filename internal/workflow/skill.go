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

## Authoring Guidance

- Prefer :mode "plan" with read-only tools for audits and exploration.
- Give every worker a bounded prompt with relevant paths, expected output, and
  stop conditions.
- Use stable, explicit phase and agent names because result keys are
  "phase.agent".
- Use concat with string literals when building prompts from prior results.
- Treat worker results as evidence to reconcile and verify, not as final truth.

## Minimal Valid Example

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
`

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
		return upperPath, false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if _, err := os.Stat(lowerPath); err == nil {
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
	defer f.Close()
	if _, err := f.WriteString(defaultSkillContent); err != nil {
		return "", false, err
	}
	return upperPath, true, nil
}
