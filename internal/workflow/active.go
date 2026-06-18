package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ActiveRegistry tracks workflow runs that can be canceled in this process.
type ActiveRegistry struct {
	mu      sync.RWMutex
	cancels map[string]context.CancelFunc
}

var defaultActiveRegistry = NewActiveRegistry()

func NewActiveRegistry() *ActiveRegistry {
	return &ActiveRegistry{cancels: make(map[string]context.CancelFunc)}
}

func DefaultActiveRegistry() *ActiveRegistry {
	return defaultActiveRegistry
}

func (r *ActiveRegistry) Register(id string, cancel context.CancelFunc) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("workflow run id is required")
	}
	if cancel == nil {
		return fmt.Errorf("workflow cancel function is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancels[id] = cancel
	return nil
}

func (r *ActiveRegistry) Cancel(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	r.mu.RLock()
	cancel, ok := r.cancels[id]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

func (r *ActiveRegistry) Unregister(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancels, id)
}

func (r *ActiveRegistry) IsActive(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.cancels[id]
	return ok
}
