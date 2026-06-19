package workflow

import (
	"context"
	"strings"
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

func TestNewLispEvaluatorSupportsV002ElispSurface(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "reader syntax",
			source: `(progn
				; line comments are ignored
				(list 'symbol ':prompt "hello\nworld" 3.14 '("read" "grep")))`,
			want: `(symbol :prompt "hello\nworld" 3.14 ("read" "grep"))`,
		},
		{
			name:   "backquote comma and comma splice",
			source: "(let ((items '(b c)) (tail '(d e))) `(a ,@items ,@tail))",
			want:   `(a b c d e)`,
		},
		{
			name: "control special forms",
			source: `(let ((i 0) (sum 0))
				(while (< i 4)
					(setq sum (+ sum i))
					(setq i (+ i 1)))
				(list
					(let ((x 1) (y 2)) (+ x y))
					(let* ((x 1) (y (+ x 2))) y)
					sum
					(if (= sum 6) "if" "bad")
					(when t "when")
					(unless nil "unless")
					(and t "and")
					(or nil "or")
					(cond ((> sum 99) "bad") ((= sum 6) "cond"))
					(catch 'done (throw 'done "caught"))))`,
			want: `(3 3 6 "if" "when" "unless" "and" "or" "cond" "caught")`,
		},
		{
			name: "functions and macros",
			source: "(progn " +
				"(defun add2 (a b) (+ a b)) " +
				"(defmacro twice (x) `(+ ,x ,x)) " +
				"(list (add2 2 3) ((lambda (x) (* x x)) 4) " +
				"(funcall (lambda (x) (- x 1)) 9) (apply 'add2 1 '(6)) " +
				"(twice 5) (macroexpand-1 '(twice 2)) (macroexpand '(twice 3))))",
			want: `(5 16 8 7 10 (+ 2 2) (+ 3 3))`,
		},
		{
			name: "list builtins",
			source: `(list
				(cons 'a '(b c))
				(car '(a b))
				(cdr '(a b c))
				(nth 1 '(a b c))
				(append '(a) '(b c))
				(reverse '(a b c))
				(member 'b '(a b c))
				(assoc 'k '((j 1) (k 2))))`,
			want: `((a b c) a (b c) b (a b c) (c b a) (b c) (k 2))`,
		},
		{
			name: "numeric string and predicate builtins",
			source: `(list
				(+ 1 2 3)
				(- 10 3 2)
				(- 5)
				(* 2 3 4)
				(/ 20 2 5)
				(= 2 2 2)
				(/= 2 3)
				(< 1 2 3)
				(<= 1 1 2)
				(> 3 2 1)
				(>= 3 3 2)
				(eq 'a 'a)
				(equal '(a (b)) '(a (b)))
				(string= "a" "a")
				(string-equal "a" "a")
				(string-lessp "a" "b")
				(string< "a" "b")
				(string-greaterp "b" "a")
				(string> "b" "a")
				(not nil)
				(null '())
				(symbolp 'x)
				(stringp "x")
				(numberp 1)
				(listp '(x))
				(consp '(x))
				(atom 'x))`,
			want: `(6 5 -5 24 2 t t t t t t t t t t t t t t t t t t t t t t)`,
		},
		{
			name: "in memory buffers and markers",
			source: `(progn
				(setq b (get-buffer-create "work"))
				(setq generated (generate-new-buffer "tmp"))
				(setq work-result
					(with-current-buffer b
						(erase-buffer)
						(insert "abcdef")
						(setq point-after-insert (point))
						(setq middle (buffer-substring 2 5))
						(goto-char 3)
						(setq m (point-marker))
						(setq copied (copy-marker m))
						(delete-region 2 4)
						(setq after-delete (buffer-string))
						(setq marker-after-delete (marker-position m))
						(setq marker-buffer-name (buffer-name (marker-buffer copied)))
						(setq detached (set-marker (make-marker) 1))
						(set-marker detached nil)
						(list (buffer-name) (bufferp (current-buffer)) point-after-insert
							(point-min) (point-max) middle after-delete marker-after-delete
							marker-buffer-name (markerp m) (marker-position detached))))
				(setq saved-name (save-current-buffer (set-buffer generated) (buffer-name)))
				(setq after-save (buffer-name))
				(setq generated-name (buffer-name generated))
				(setq fetched (bufferp (get-buffer "work")))
				(kill-buffer b)
				(list work-result saved-name after-save generated-name fetched (get-buffer "work")))`,
			want: `(("work" t 7 1 5 "bcd" "adef" 2 "work" t nil) "tmp" "*scratch*" "tmp" t nil)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewLispEvaluator().EvalString(context.Background(), tt.source)
			if err != nil {
				t.Fatalf("EvalString() error = %v", err)
			}
			if got := elispvm.Stringify(got); got != tt.want {
				t.Fatalf("result = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewLispEvaluatorRejectsVariableLambdaLists(t *testing.T) {
	tests := []string{
		`(lambda (&rest xs) xs)`,
		`(lambda (&optional x) x)`,
		`(lambda (&body body) body)`,
		`(defun bad (&rest xs) xs)`,
		`(defun bad (&optional x) x)`,
		`(defmacro bad (&rest xs) xs)`,
		`(defmacro bad (&optional x) x)`,
		`(defmacro bad (&body body) body)`,
	}
	for _, source := range tests {
		t.Run(source, func(t *testing.T) {
			_, err := NewLispEvaluator().EvalString(context.Background(), source)
			if err == nil {
				t.Fatal("expected fixed-argument-list error")
			}
			if !strings.Contains(err.Error(), "only supports fixed arguments") {
				t.Fatalf("error = %q, want fixed argument list error", err.Error())
			}
		})
	}
}
