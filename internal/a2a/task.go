package a2a

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var fallbackTaskCounter uint64

// TaskState represents the state of an A2A task.
type TaskState string

const (
	TaskStateSubmitted TaskState = "submitted"
	TaskStateWorking   TaskState = "working"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCanceled  TaskState = "canceled"
)

// Task represents an A2A task.
type Task struct {
	ID        string         `json:"id"`
	State     TaskState      `json:"state"`
	Message   *Message       `json:"message,omitempty"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	Error     *TaskError     `json:"error,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Message represents an A2A message (text or structured).
type Message struct {
	Role     string         `json:"role"` // "user" or "agent"
	Parts    []MessagePart  `json:"parts"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MessagePart is a part of a message.
type MessagePart struct {
	Type string `json:"type"` // "text"
	Text string `json:"text,omitempty"`
}

// Artifact represents output produced by an agent task.
type Artifact struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parts       []MessagePart  `json:"parts"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TaskError represents an error in task processing.
type TaskError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TaskStore manages task storage.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

// TaskEvent is sent via SSE for streaming task updates.
type TaskEvent struct {
	TaskID    string     `json:"task_id"`
	State     TaskState  `json:"state"`
	Message   *Message   `json:"message,omitempty"`
	Artifact  *Artifact  `json:"artifact,omitempty"`
	Error     *TaskError `json:"error,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// NewTaskStore creates a new task store.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
	}
}

func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "task_" + hex.EncodeToString(b[:])
	}
	n := atomic.AddUint64(&fallbackTaskCounter, 1)
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), n)
}

// Create creates a new task.
func (s *TaskStore) Create(id string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	task := &Task{
		ID:        id,
		State:     TaskStateSubmitted,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]any),
	}
	s.tasks[id] = task
	return task.Clone()
}

// Get returns a task by ID.
func (s *TaskStore) Get(id string) *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task := s.tasks[id]
	if task == nil {
		return nil
	}
	return task.Clone()
}

// Update updates a task.
func (s *TaskStore) Update(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := task.Clone()
	copy.UpdatedAt = time.Now()
	s.tasks[copy.ID] = copy
}

// SetState updates the task state.
func (s *TaskStore) SetState(id string, state TaskState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task, ok := s.tasks[id]; ok {
		task.State = state
		task.UpdatedAt = time.Now()
	}
}

// Clone returns a deep copy of the task value.
func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}
	copy := *t
	copy.Message = cloneMessage(t.Message)
	if len(t.Artifacts) > 0 {
		copy.Artifacts = make([]Artifact, len(t.Artifacts))
		for i := range t.Artifacts {
			copy.Artifacts[i] = cloneArtifact(t.Artifacts[i])
		}
	}
	if t.Error != nil {
		errCopy := *t.Error
		copy.Error = &errCopy
	}
	copy.Metadata = cloneMap(t.Metadata)
	return &copy
}

func cloneMessage(msg *Message) *Message {
	if msg == nil {
		return nil
	}
	copy := *msg
	copy.Parts = append([]MessagePart(nil), msg.Parts...)
	copy.Metadata = cloneMap(msg.Metadata)
	return &copy
}

func cloneArtifact(artifact Artifact) Artifact {
	copy := artifact
	copy.Parts = append([]MessagePart(nil), artifact.Parts...)
	copy.Metadata = cloneMap(artifact.Metadata)
	return copy
}

func cloneMap(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	copy := make(map[string]any, len(m))
	for k, v := range m {
		copy[k] = v
	}
	return copy
}
