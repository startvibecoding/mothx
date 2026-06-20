package workflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	elispvm "github.com/startvibecoding/vibeEmacsLispVm"
)

// Runner evaluates Elisp workflow DSL and delegates agent tasks to a Host.
type Runner struct {
	Host        Host
	Store       Store
	Active      *ActiveRegistry
	Concurrency int
	Now         func() time.Time
	Progress    func(ProgressEvent)
}

// Run evaluates a workflow source string.
func (r *Runner) Run(ctx context.Context, source string) (*RunState, error) {
	if r.Host == nil {
		return nil, fmt.Errorf("workflow host is required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	now := r.now()
	rt := &runtime{
		runner:      r,
		state:       &RunState{Status: StatusRunning, StartedAt: now, UpdatedAt: now, Results: make(map[string]AgentResult)},
		phaseIndex:  -1,
		concurrency: r.Concurrency,
		cancel:      cancel,
	}
	defer rt.unregisterActive()
	if rt.concurrency <= 0 {
		rt.concurrency = 5
	}
	e := NewLispEvaluator()
	rt.register(e)
	if _, err := e.EvalString(runCtx, source); err != nil {
		rt.markError(err)
		_ = rt.save(context.WithoutCancel(ctx))
		return rt.snapshot(), err
	}
	rt.finish(StatusDone, "")
	_ = rt.save(context.WithoutCancel(ctx))
	return rt.snapshot(), nil
}

func (r *Runner) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

type runtime struct {
	mu          sync.Mutex
	runner      *Runner
	state       *RunState
	activeID    string
	cancel      context.CancelFunc
	phase       string
	phaseIndex  int
	concurrency int
	sem         chan struct{}
}

func (rt *runtime) register(e *elispvm.Evaluator) {
	e.RegisterSpecial("workflow", rt.specialWorkflow)
	e.RegisterSpecial("phase", rt.specialPhase)
	e.RegisterSpecial("parallel", rt.specialParallel)
	e.RegisterSpecial("series", rt.specialSeries)
	e.RegisterSpecial("agent", rt.specialAgent)
	e.RegisterSpecial("result", rt.specialResult)
	e.RegisterFunc("concurrency", rt.fnConcurrency)
	e.RegisterFunc("result-key", rt.fnResultKey)
	e.RegisterFunc("result-latest", rt.fnResultLatest)
	e.RegisterFunc("results", rt.fnResults)
	e.RegisterFunc("log", rt.fnLog)
}

func (rt *runtime) specialWorkflow(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("workflow expects a name and body")
	}
	name, err := literalString(args[0])
	if err != nil {
		return nil, fmt.Errorf("workflow name: %w", err)
	}
	rt.mu.Lock()
	rt.state.Name = name
	if rt.state.ID == "" {
		rt.state.ID = makeRunID(name, rt.runner.now())
	}
	rt.state.UpdatedAt = rt.runner.now()
	rt.mu.Unlock()
	if err := rt.save(ctx.Context); err != nil {
		return nil, err
	}
	rt.emitProgress(ProgressEvent{
		RunID:   rt.state.ID,
		Name:    name,
		Status:  StatusRunning,
		Message: fmt.Sprintf("workflow %q started", name),
	})
	if err := rt.registerActive(); err != nil {
		return nil, err
	}
	if len(args) == 1 {
		return elispvm.String(rt.state.ID), nil
	}
	if _, err := ctx.EvalAll(args[1:]); err != nil {
		return nil, err
	}
	return elispvm.String(rt.state.ID), nil
}

func (rt *runtime) specialPhase(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("phase expects a name and body")
	}
	name, err := literalString(args[0])
	if err != nil {
		return nil, fmt.Errorf("phase name: %w", err)
	}
	idx := rt.startPhase(name)
	prevPhase := rt.phase
	prevIndex := rt.phaseIndex
	rt.phase = name
	rt.phaseIndex = idx
	_, evalErr := ctx.EvalAll(args[1:])
	rt.phase = prevPhase
	rt.phaseIndex = prevIndex
	if evalErr != nil {
		rt.finishPhase(idx, statusForError(evalErr), evalErr.Error())
		return nil, evalErr
	}
	rt.finishPhase(idx, StatusDone, "")
	return elispvm.String(name), nil
}

func (rt *runtime) specialSeries(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	return ctx.EvalAll(args)
}

func (rt *runtime) specialParallel(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	if len(args) == 0 {
		return elispvm.Nil, nil
	}
	type item struct {
		i int
		v elispvm.Value
		e error
	}
	results := make([]elispvm.Value, len(args))
	ch := make(chan item, len(args))
	for i, expr := range args {
		i, expr := i, expr
		go func() {
			child := ctx.Child()
			v, err := child.Eval(expr)
			ch <- item{i: i, v: v, e: err}
		}()
	}
	var firstErr error
	for range args {
		item := <-ch
		if item.e != nil && firstErr == nil {
			firstErr = item.e
		}
		results[item.i] = item.v
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return elispvm.List(results), nil
}

func (rt *runtime) specialAgent(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("agent expects a name and keyword arguments")
	}
	name, err := literalString(args[0])
	if err != nil {
		return nil, fmt.Errorf("agent name: %w", err)
	}
	task := AgentTask{Name: name, Phase: rt.phase}
	if len(args[1:])%2 != 0 {
		return nil, fmt.Errorf("agent keyword arguments must be pairs")
	}
	for i := 1; i < len(args); i += 2 {
		key, ok := args[i].(elispvm.Symbol)
		if !ok || !strings.HasPrefix(string(key), ":") {
			return nil, fmt.Errorf("agent argument %d must be a keyword symbol", i)
		}
		value, err := ctx.Eval(args[i+1])
		if err != nil {
			return nil, err
		}
		if err := applyAgentOption(&task, string(key), value); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(task.Prompt) == "" {
		return nil, fmt.Errorf("agent %q requires :prompt", name)
	}
	result, err := rt.runAgent(ctx.Context, task)
	if err != nil {
		return nil, err
	}
	return elispvm.String(result.Result), nil
}

func (rt *runtime) fnConcurrency(_ *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("concurrency expects 1 argument")
	}
	n, ok := args[0].(elispvm.Number)
	if !ok {
		return nil, fmt.Errorf("concurrency expects a number")
	}
	limit := int(float64(n))
	if limit <= 0 {
		return nil, fmt.Errorf("concurrency must be greater than 0")
	}
	rt.mu.Lock()
	rt.concurrency = limit
	rt.sem = nil
	rt.state.UpdatedAt = rt.runner.now()
	rt.mu.Unlock()
	return elispvm.Number(limit), nil
}

func (rt *runtime) specialResult(ctx *elispvm.EvalContext, args []elispvm.Expr) (elispvm.Value, error) {
	if len(args) != 1 && len(args) != 3 {
		return nil, fmt.Errorf("result expects 1 argument or :key with a value")
	}
	keyValue, err := ctx.Eval(args[0])
	if err != nil {
		return nil, err
	}
	key, ok := keyValue.(elispvm.String)
	if !ok {
		return nil, fmt.Errorf("result expects a string key")
	}
	instanceKey := ""
	if len(args) == 3 {
		option, ok := args[1].(elispvm.Symbol)
		if !ok || string(option) != ":key" {
			return nil, fmt.Errorf("result optional argument must be :key")
		}
		value, err := ctx.Eval(args[2])
		if err != nil {
			return nil, err
		}
		keyValue, ok := value.(elispvm.String)
		if !ok {
			return nil, fmt.Errorf("result :key expects a string")
		}
		instanceKey = string(keyValue)
		if err := validateInstanceKey(instanceKey); err != nil {
			return nil, fmt.Errorf("result :key %w", err)
		}
	}
	result, ok := rt.lookupResult(string(key), instanceKey)
	if !ok {
		if instanceKey != "" {
			return nil, fmt.Errorf("workflow result %q with key %q not found", string(key), instanceKey)
		}
		return nil, fmt.Errorf("workflow result %q not found", string(key))
	}
	return elispvm.String(result.Result), nil
}

func (rt *runtime) fnResultKey(_ *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("result-key expects a result key and instance key")
	}
	key, ok := args[0].(elispvm.String)
	if !ok {
		return nil, fmt.Errorf("result-key expects a string result key")
	}
	instanceKey, ok := args[1].(elispvm.String)
	if !ok {
		return nil, fmt.Errorf("result-key expects a string instance key")
	}
	if err := validateInstanceKey(string(instanceKey)); err != nil {
		return nil, fmt.Errorf("result-key instance key %w", err)
	}
	result, ok := rt.lookupResult(string(key), string(instanceKey))
	if !ok {
		return nil, fmt.Errorf("workflow result %q with key %q not found", string(key), string(instanceKey))
	}
	return elispvm.String(result.Result), nil
}

func (rt *runtime) fnResultLatest(_ *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("result-latest expects 1 argument")
	}
	key, ok := args[0].(elispvm.String)
	if !ok {
		return nil, fmt.Errorf("result-latest expects a string key")
	}
	result, ok := rt.latestResultForBase(string(key))
	if !ok {
		return nil, fmt.Errorf("workflow result %q not found", string(key))
	}
	return elispvm.String(result.Result), nil
}

func (rt *runtime) lookupResult(baseKey string, instanceKey string) (AgentResult, bool) {
	if instanceKey != "" {
		return rt.resultByStorageKey(resultStorageKey(baseKey, instanceKey))
	}
	if result, ok := rt.resultByStorageKey(baseKey); ok {
		return result, true
	}
	return rt.latestResultForBase(baseKey)
}

func (rt *runtime) resultByStorageKey(key string) (AgentResult, bool) {
	rt.mu.Lock()
	result, ok := rt.state.Results[key]
	rt.mu.Unlock()
	return result, ok
}

func (rt *runtime) latestResultForBase(baseKey string) (AgentResult, bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	var latest AgentResult
	found := false
	for _, result := range rt.state.Results {
		if !resultMatchesBase(result, baseKey) {
			continue
		}
		if !found || result.StartedAt.After(latest.StartedAt) || result.FinishedAt.After(latest.FinishedAt) {
			latest = result
			found = true
		}
	}
	return latest, found
}

func (rt *runtime) fnResults(_ *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("results expects 1 argument")
	}
	phase, ok := args[0].(elispvm.String)
	if !ok {
		return nil, fmt.Errorf("results expects a phase name string")
	}
	query := string(phase)
	prefix := query + "."
	rt.mu.Lock()
	matches := make([]AgentResult, 0)
	for _, res := range rt.state.Results {
		if res.Phase == query || strings.HasPrefix(res.Key, prefix) || resultMatchesBase(res, query) {
			matches = append(matches, res)
		}
	}
	rt.mu.Unlock()
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].StartedAt.Equal(matches[j].StartedAt) {
			return matches[i].Key < matches[j].Key
		}
		return matches[i].StartedAt.Before(matches[j].StartedAt)
	})
	var out strings.Builder
	for _, res := range matches {
		if out.Len() > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(res.Key)
		out.WriteString(":\n")
		out.WriteString(res.Result)
	}
	return elispvm.String(out.String()), nil
}

func (rt *runtime) fnLog(_ *elispvm.EvalContext, args []elispvm.Value) (elispvm.Value, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, rawString(arg))
	}
	msg := strings.Join(parts, " ")
	rt.mu.Lock()
	rt.state.Logs = append(rt.state.Logs, WorkflowLog{Time: rt.runner.now(), Message: msg})
	rt.state.UpdatedAt = rt.runner.now()
	rt.mu.Unlock()
	rt.emitProgress(ProgressEvent{Status: StatusRunning, Message: msg})
	return elispvm.String(msg), nil
}

func (rt *runtime) runAgent(ctx context.Context, task AgentTask) (AgentResult, error) {
	if err := validateInstanceKey(task.InstanceKey); err != nil {
		return AgentResult{}, fmt.Errorf("agent %q :key: %w", task.Name, err)
	}
	sem := rt.semaphore()
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return AgentResult{}, ctx.Err()
	}
	defer func() { <-sem }()

	key := taskStorageKey(task.Phase, task.Name, task.InstanceKey)
	started := rt.runner.now()
	rt.recordTaskStart(key)
	rt.emitProgress(ProgressEvent{
		Phase:   task.Phase,
		Task:    task.Name,
		Status:  StatusRunning,
		Message: fmt.Sprintf("task %s started", key),
	})
	result, err := rt.runner.Host.RunAgent(ctx, task)
	finished := rt.runner.now()
	if result.Key == "" {
		result.Key = key
	}
	result.Name = task.Name
	result.Phase = task.Phase
	result.InstanceKey = task.InstanceKey
	if result.StartedAt.IsZero() {
		result.StartedAt = started
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = finished
	}
	result.Duration = result.FinishedAt.Sub(result.StartedAt).Round(time.Millisecond).String()
	if err != nil {
		result.Status = statusForError(err)
		result.Error = err.Error()
	} else if result.Status == "" {
		result.Status = StatusDone
	}
	rt.recordResult(result)
	rt.emitProgress(ProgressEvent{
		Phase:   task.Phase,
		Task:    task.Name,
		Status:  result.Status,
		Message: fmt.Sprintf("task %s %s", key, result.Status),
	})
	if err := rt.save(ctx); err != nil {
		return result, err
	}
	if err != nil {
		return result, err
	}
	return result, nil
}

func (rt *runtime) semaphore() chan struct{} {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.sem == nil {
		rt.sem = make(chan struct{}, rt.concurrency)
	}
	return rt.sem
}

func (rt *runtime) startPhase(name string) int {
	now := rt.runner.now()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state.Phases = append(rt.state.Phases, PhaseState{Name: name, Status: StatusRunning, StartedAt: now})
	rt.state.UpdatedAt = now
	idx := len(rt.state.Phases) - 1
	rt.emitProgressLocked(ProgressEvent{
		Phase:   name,
		Status:  StatusRunning,
		Message: fmt.Sprintf("phase %q started", name),
	})
	return idx
}

func (rt *runtime) finishPhase(idx int, status string, msg string) {
	now := rt.runner.now()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if idx >= 0 && idx < len(rt.state.Phases) {
		rt.state.Phases[idx].Status = status
		rt.state.Phases[idx].FinishedAt = now
		rt.state.Phases[idx].Error = msg
		rt.emitProgressLocked(ProgressEvent{
			Phase:   rt.state.Phases[idx].Name,
			Status:  status,
			Message: fmt.Sprintf("phase %q %s", rt.state.Phases[idx].Name, status),
		})
	}
	rt.state.UpdatedAt = now
}

func (rt *runtime) recordTaskStart(key string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.phaseIndex >= 0 && rt.phaseIndex < len(rt.state.Phases) {
		rt.state.Phases[rt.phaseIndex].Tasks = append(rt.state.Phases[rt.phaseIndex].Tasks, key)
	}
	rt.state.UpdatedAt = rt.runner.now()
}

func (rt *runtime) recordResult(result AgentResult) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state.Results[result.Key] = result
	rt.state.UpdatedAt = rt.runner.now()
}

func (rt *runtime) markError(err error) {
	rt.finish(statusForError(err), err.Error())
}

func (rt *runtime) finish(status string, msg string) {
	now := rt.runner.now()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.state.Status = status
	rt.state.Error = msg
	rt.state.UpdatedAt = now
	rt.state.FinishedAt = now
	message := fmt.Sprintf("workflow %s", status)
	if msg != "" {
		message += ": " + msg
	}
	rt.emitProgressLocked(ProgressEvent{Status: status, Message: message})
}

func (rt *runtime) emitProgress(ev ProgressEvent) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.emitProgressLocked(ev)
}

func (rt *runtime) emitProgressLocked(ev ProgressEvent) {
	if rt.runner == nil || rt.runner.Progress == nil {
		return
	}
	if ev.RunID == "" {
		ev.RunID = rt.state.ID
	}
	if ev.Name == "" {
		ev.Name = rt.state.Name
	}
	if ev.Phase == "" {
		ev.Phase = rt.phase
	}
	if ev.Time.IsZero() {
		ev.Time = rt.runner.now()
	}
	rt.runner.Progress(ev)
}

func (rt *runtime) snapshot() *RunState {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	cp := *rt.state
	cp.Phases = append([]PhaseState(nil), rt.state.Phases...)
	cp.Logs = append([]WorkflowLog(nil), rt.state.Logs...)
	cp.Results = make(map[string]AgentResult, len(rt.state.Results))
	for k, v := range rt.state.Results {
		cp.Results[k] = v
	}
	return &cp
}

func (rt *runtime) save(ctx context.Context) error {
	if rt.runner.Store == nil {
		return nil
	}
	return rt.runner.Store.Save(ctx, rt.snapshot())
}

func (rt *runtime) activeRegistry() *ActiveRegistry {
	if rt.runner.Active != nil {
		return rt.runner.Active
	}
	return DefaultActiveRegistry()
}

func (rt *runtime) registerActive() error {
	rt.mu.Lock()
	id := rt.state.ID
	if rt.activeID == id {
		rt.mu.Unlock()
		return nil
	}
	rt.activeID = id
	rt.mu.Unlock()
	return rt.activeRegistry().Register(id, rt.cancel)
}

func (rt *runtime) unregisterActive() {
	rt.mu.Lock()
	id := rt.activeID
	rt.activeID = ""
	rt.mu.Unlock()
	rt.activeRegistry().Unregister(id)
}

func literalString(expr elispvm.Expr) (string, error) {
	s, ok := expr.(elispvm.String)
	if !ok {
		return "", fmt.Errorf("expected string literal")
	}
	return string(s), nil
}

func applyAgentOption(task *AgentTask, key string, value elispvm.Value) error {
	switch key {
	case ":prompt":
		v, ok := value.(elispvm.String)
		if !ok {
			return fmt.Errorf(":prompt expects a string")
		}
		task.Prompt = string(v)
	case ":mode":
		v, ok := value.(elispvm.String)
		if !ok {
			return fmt.Errorf(":mode expects a string")
		}
		task.Mode = string(v)
	case ":work-dir":
		v, ok := value.(elispvm.String)
		if !ok {
			return fmt.Errorf(":work-dir expects a string")
		}
		task.WorkDir = string(v)
	case ":tools":
		tools, err := stringList(value)
		if err != nil {
			return fmt.Errorf(":tools expects a string list: %w", err)
		}
		task.Tools = tools
	case ":max-iterations":
		v, ok := value.(elispvm.Number)
		if !ok {
			return fmt.Errorf(":max-iterations expects a number")
		}
		task.MaxIterations = int(float64(v))
	case ":key":
		v, ok := value.(elispvm.String)
		if !ok {
			return fmt.Errorf(":key expects a string")
		}
		if err := validateInstanceKey(string(v)); err != nil {
			return fmt.Errorf(":key %w", err)
		}
		task.InstanceKey = string(v)
	case ":system-prompt-extra":
		v, ok := value.(elispvm.String)
		if !ok {
			return fmt.Errorf(":system-prompt-extra expects a string")
		}
		task.SystemPromptExtra = string(v)
	default:
		return fmt.Errorf("unknown agent option %s", key)
	}
	return nil
}

func taskStorageKey(phase string, name string, instanceKey string) string {
	base := name
	if phase != "" {
		base = phase + "." + name
	}
	return resultStorageKey(base, instanceKey)
}

func resultStorageKey(baseKey string, instanceKey string) string {
	if instanceKey == "" {
		return baseKey
	}
	return baseKey + "[" + instanceKey + "]"
}

func resultBaseKey(result AgentResult) string {
	if result.Phase == "" {
		return result.Name
	}
	return result.Phase + "." + result.Name
}

func resultMatchesBase(result AgentResult, baseKey string) bool {
	return resultBaseKey(result) == baseKey || result.Key == baseKey
}

func validateInstanceKey(key string) error {
	if key == "" {
		return nil
	}
	if strings.TrimSpace(key) != key {
		return fmt.Errorf("must not have leading or trailing whitespace")
	}
	if strings.ContainsAny(key, "[]\n\r\t") {
		return fmt.Errorf("must not contain brackets or control whitespace")
	}
	return nil
}

func stringList(v elispvm.Value) ([]string, error) {
	list, ok := v.(elispvm.List)
	if !ok {
		return nil, fmt.Errorf("got %s", elispvm.Stringify(v))
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(elispvm.String)
		if !ok {
			return nil, fmt.Errorf("item %s is not a string", elispvm.Stringify(item))
		}
		out = append(out, string(s))
	}
	return out, nil
}

func rawString(v elispvm.Value) string {
	switch x := v.(type) {
	case elispvm.String:
		return string(x)
	case elispvm.Symbol:
		return string(x)
	default:
		return elispvm.Stringify(v)
	}
}

func statusForError(err error) string {
	if errors.Is(err, context.Canceled) {
		return StatusCanceled
	}
	return StatusError
}

func makeRunID(name string, t time.Time) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case b.Len() > 0 && b.String()[b.Len()-1] != '-':
			b.WriteByte('-')
		}
	}
	slug = strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "workflow"
	}
	return fmt.Sprintf("%s-%s", slug, t.UTC().Format("20060102T150405.000000000"))
}
