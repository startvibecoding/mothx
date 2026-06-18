package workflow

import (
	"context"
	"testing"

	elispvm "github.com/startvibecoding/vibeEmacsLispVm"
)

func TestNewLispEvaluatorUsesPublishedDependency(t *testing.T) {
	e := NewLispEvaluator()
	e.RegisterFunc("workflow-test-join", func(ctx *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
		return elispvm.String(string(args[0].(elispvm.String)) + "/" + string(args[1].(elispvm.String))), nil
	})

	got, err := e.EvalString(context.Background(), `(workflow-test-join "phase" "agent")`)
	if err != nil {
		t.Fatalf("EvalString() error = %v", err)
	}
	if got := string(got.(elispvm.String)); got != "phase/agent" {
		t.Fatalf("result = %q", got)
	}
}
