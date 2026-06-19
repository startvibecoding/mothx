package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeHost struct {
	mu            sync.Mutex
	running       int
	maxRunning    int
	tasks         []AgentTask
	resultsByName map[string]string
}

func (h *fakeHost) RunAgent(ctx context.Context, task AgentTask) (AgentResult, error) {
	h.mu.Lock()
	h.running++
	if h.running > h.maxRunning {
		h.maxRunning = h.running
	}
	h.tasks = append(h.tasks, task)
	h.mu.Unlock()

	select {
	case <-time.After(5 * time.Millisecond):
	case <-ctx.Done():
		return AgentResult{}, ctx.Err()
	}

	h.mu.Lock()
	h.running--
	result := h.resultsByName[task.Name]
	if result == "" {
		result = fmt.Sprintf("%s:%s", task.Name, task.Prompt)
	}
	h.mu.Unlock()

	return AgentResult{Result: result}, nil
}

func TestRunnerExecutesPhasesAndResults(t *testing.T) {
	host := &fakeHost{resultsByName: map[string]string{
		"gateway": "gateway findings",
		"hermes":  "hermes findings",
	}}
	store := &memoryStore{}
	r := &Runner{Host: host, Store: store, Concurrency: 2, Now: fixedClock()}

	state, err := r.Run(context.Background(), `
		(workflow "auth audit"
		  (concurrency 2)
		  (phase "scan"
		    (parallel
		      (agent "gateway"
		        :mode "plan"
		        :tools '("read" "grep")
		        :prompt "scan gateway")
		      (agent "hermes"
		        :mode "plan"
		        :tools '("read" "grep")
		        :prompt "scan hermes")))
		  (phase "verify"
		    (agent "cross-check"
		      :mode "plan"
		      :prompt (concat (result "scan.gateway") "\n" (result "scan.hermes")))))
	`)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if state.Status != StatusDone {
		t.Fatalf("status = %s", state.Status)
	}
	if len(state.Phases) != 2 {
		t.Fatalf("phases = %d, want 2", len(state.Phases))
	}
	if got := state.Results["scan.gateway"].Result; got != "gateway findings" {
		t.Fatalf("scan.gateway result = %q", got)
	}
	gateway := findTask(host.tasks, "gateway")
	if gateway == nil {
		t.Fatal("gateway task not found")
	}
	if gateway.Mode != "plan" {
		t.Fatalf("gateway mode = %q, want plan", gateway.Mode)
	}
	if !equalStrings(gateway.Tools, []string{"read", "grep"}) {
		t.Fatalf("gateway tools = %#v, want read/grep", gateway.Tools)
	}
	verify := findTask(host.tasks, "cross-check")
	if verify == nil {
		t.Fatal("cross-check task not found")
	}
	if !strings.Contains(verify.Prompt, "gateway findings") || !strings.Contains(verify.Prompt, "hermes findings") {
		t.Fatalf("verify prompt did not include prior results: %q", verify.Prompt)
	}
	if host.maxRunning > 2 {
		t.Fatalf("maxRunning = %d, want <= 2", host.maxRunning)
	}
	if store.saved == 0 {
		t.Fatal("expected store saves")
	}
}

func TestRunnerReportsMissingResult(t *testing.T) {
	r := &Runner{Host: &fakeHost{}, Now: fixedClock()}
	state, err := r.Run(context.Background(), `
		(workflow "bad"
		  (phase "verify"
		    (agent "cross-check" :prompt (result "scan.missing"))))
	`)
	if err == nil {
		t.Fatal("expected error")
	}
	if state.Status != StatusError {
		t.Fatalf("status = %s, want error", state.Status)
	}
}

func TestRunnerRequiresLiteralWorkflowPhaseAndAgentNames(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "workflow variable name",
			source: `(let ((name "bad")) (workflow name))`,
			want:   "workflow name: expected string literal",
		},
		{
			name:   "workflow expression name",
			source: `(workflow (concat "bad" "-name"))`,
			want:   "workflow name: expected string literal",
		},
		{
			name: "phase variable name",
			source: `
				(workflow "bad"
				  (let ((phase-name "scan"))
				    (phase phase-name)))`,
			want: "phase name: expected string literal",
		},
		{
			name: "phase expression name",
			source: `
				(workflow "bad"
				  (phase (concat "scan" "-phase")))`,
			want: "phase name: expected string literal",
		},
		{
			name: "agent variable name",
			source: `
				(workflow "bad"
				  (phase "scan"
				    (let ((agent-name "worker"))
				      (agent agent-name :prompt "do work"))))`,
			want: "agent name: expected string literal",
		},
		{
			name: "agent expression name",
			source: `
				(workflow "bad"
				  (phase "scan"
				    (agent (concat "worker" "-a") :prompt "do work")))`,
			want: "agent name: expected string literal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{Host: &fakeHost{}, Now: fixedClock()}
			state, err := r.Run(context.Background(), tt.source)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
			if state == nil {
				t.Fatal("expected error state")
			}
			if state.Status != StatusError {
				t.Fatalf("status = %s, want error", state.Status)
			}
		})
	}
}

func TestRunnerCanBeCanceledByActiveRegistry(t *testing.T) {
	host := &blockingHost{started: make(chan struct{})}
	store := &memoryStore{}
	active := NewActiveRegistry()
	r := &Runner{Host: host, Store: store, Active: active, Now: fixedClock()}

	done := make(chan struct {
		state *RunState
		err   error
	}, 1)
	go func() {
		state, err := r.Run(context.Background(), `
			(workflow "cancel me"
			  (phase "wait"
			    (agent "slow" :prompt "wait until canceled")))
		`)
		done <- struct {
			state *RunState
			err   error
		}{state: state, err: err}
	}()

	<-host.started
	id := waitForWorkflowID(t, store)
	if !active.Cancel(id) {
		t.Fatalf("expected active workflow %s to be cancelable", id)
	}

	result := <-done
	if result.err == nil {
		t.Fatal("expected cancellation error")
	}
	if result.state.Status != StatusCanceled {
		t.Fatalf("status = %s, want canceled", result.state.Status)
	}
	if active.IsActive(id) {
		t.Fatalf("workflow %s should be unregistered after completion", id)
	}
	saved, err := store.Load(context.Background(), id)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Status != StatusCanceled {
		t.Fatalf("saved status = %s, want canceled", saved.Status)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	store := NewFileStore(t.TempDir())
	state := &RunState{
		ID:        "run-1",
		Name:      "Run 1",
		Status:    StatusDone,
		StartedAt: time.Unix(1, 0),
		UpdatedAt: time.Unix(2, 0),
		Results: map[string]AgentResult{
			"scan.one": {Key: "scan.one", Name: "one", Status: StatusDone, Result: "ok"},
		},
	}
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Results["scan.one"].Result != "ok" {
		t.Fatalf("loaded result = %#v", got.Results["scan.one"])
	}
	list, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != "run-1" {
		t.Fatalf("list = %#v", list)
	}
}

type blockingHost struct {
	started chan struct{}
	once    sync.Once
}

func (h *blockingHost) RunAgent(ctx context.Context, task AgentTask) (AgentResult, error) {
	h.once.Do(func() { close(h.started) })
	<-ctx.Done()
	return AgentResult{}, ctx.Err()
}

func findTask(tasks []AgentTask, name string) *AgentTask {
	for i := range tasks {
		if tasks[i].Name == name {
			return &tasks[i]
		}
	}
	return nil
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func fixedClock() func() time.Time {
	var mu sync.Mutex
	t := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		t = t.Add(time.Millisecond)
		return t
	}
}

type memoryStore struct {
	mu     sync.Mutex
	saved  int
	states []*RunState
}

func (s *memoryStore) Save(ctx context.Context, state *RunState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *state
	s.states = append(s.states, &cp)
	s.saved++
	return nil
}

func (s *memoryStore) Load(ctx context.Context, id string) (*RunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.states) - 1; i >= 0; i-- {
		state := s.states[i]
		if state.ID == id {
			cp := *state
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (s *memoryStore) List(ctx context.Context) ([]RunState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RunState, 0, len(s.states))
	for _, state := range s.states {
		out = append(out, *state)
	}
	return out, nil
}

func waitForWorkflowID(t testing.TB, store *memoryStore) string {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		for _, state := range store.states {
			if state.ID != "" {
				id := state.ID
				store.mu.Unlock()
				return id
			}
		}
		store.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for workflow id")
	return ""
}
