package workflow

import (
	"context"
	"time"
)

const (
	StatusRunning  = "running"
	StatusDone     = "done"
	StatusError    = "error"
	StatusCanceled = "canceled"
)

// AgentTask describes one workflow worker-agent invocation.
type AgentTask struct {
	Name              string   `json:"name"`
	Phase             string   `json:"phase,omitempty"`
	InstanceKey       string   `json:"instanceKey,omitempty"`
	Prompt            string   `json:"prompt"`
	Mode              string   `json:"mode,omitempty"`
	WorkDir           string   `json:"workDir,omitempty"`
	Tools             []string `json:"tools,omitempty"`
	MaxIterations     int      `json:"maxIterations,omitempty"`
	SystemPromptExtra string   `json:"systemPromptExtra,omitempty"`
}

// AgentResult captures the completed worker-agent output.
type AgentResult struct {
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Phase       string    `json:"phase,omitempty"`
	InstanceKey string    `json:"instanceKey,omitempty"`
	Status      string    `json:"status"`
	Result      string    `json:"result,omitempty"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"startedAt"`
	FinishedAt  time.Time `json:"finishedAt,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// PhaseState captures runtime state for a workflow phase.
type PhaseState struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Tasks      []string  `json:"tasks,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// WorkflowLog is a timestamped log entry emitted by workflow DSL.
type WorkflowLog struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

// RunState is the persisted workflow run state.
type RunState struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	StartedAt  time.Time              `json:"startedAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	FinishedAt time.Time              `json:"finishedAt,omitempty"`
	Phases     []PhaseState           `json:"phases,omitempty"`
	Results    map[string]AgentResult `json:"results,omitempty"`
	Logs       []WorkflowLog          `json:"logs,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// ProgressEvent captures a lightweight workflow lifecycle update.
type ProgressEvent struct {
	RunID   string
	Name    string
	Phase   string
	Task    string
	Status  string
	Message string
	Time    time.Time
}

// Host runs workflow worker-agent tasks.
type Host interface {
	RunAgent(ctx context.Context, task AgentTask) (AgentResult, error)
}

// Store persists workflow run state.
type Store interface {
	Save(ctx context.Context, state *RunState) error
	Load(ctx context.Context, id string) (*RunState, error)
	List(ctx context.Context) ([]RunState, error)
}
