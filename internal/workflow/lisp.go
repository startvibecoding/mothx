package workflow

import elispvm "github.com/startvibecoding/vibeEmacsLispVm"

// NewLispEvaluator creates an evaluator for workflow DSL execution.
//
// Workflow-specific functions such as workflow, phase, parallel, agent, result,
// and log must be registered by the workflow runner. The VM itself remains a
// generic Elisp subset implementation.
func NewLispEvaluator() *elispvm.Evaluator {
	return elispvm.New()
}
