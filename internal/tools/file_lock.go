package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FileLockManager coordinates in-process writes to individual files.
// It deliberately supports acquiring only one file at a time; write-like tools
// should not hold one file lock while waiting for another.
type FileLockManager struct {
	mu    sync.Mutex
	locks map[string]*fileLock
}

type fileLock struct {
	done       chan struct{}
	owner      string
	acquiredAt time.Time
}

var defaultFileLockManager = NewFileLockManager()

// NewFileLockManager creates an empty in-memory file lock manager.
func NewFileLockManager() *FileLockManager {
	return &FileLockManager{
		locks: make(map[string]*fileLock),
	}
}

// DefaultFileLockManager returns the process-wide file lock manager used by
// default registries. It coordinates parent and sub-agent registries in the
// same process.
func DefaultFileLockManager() *FileLockManager {
	return defaultFileLockManager
}

// Acquire waits for exclusive access to path and returns a release function.
// Waiting is cancellable through ctx.
func (m *FileLockManager) Acquire(ctx context.Context, path, owner string) (func(), error) {
	if m == nil {
		return func() {}, nil
	}
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if owner == "" {
		owner = "unknown"
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		m.mu.Lock()
		if current := m.locks[path]; current == nil {
			entry := &fileLock{
				done:       make(chan struct{}),
				owner:      owner,
				acquiredAt: time.Now(),
			}
			m.locks[path] = entry
			m.mu.Unlock()
			return func() {
				m.release(path, entry)
			}, nil
		} else {
			done := current.done
			currentOwner := current.owner
			acquiredAt := current.acquiredAt
			m.mu.Unlock()

			select {
			case <-done:
				continue
			case <-ctx.Done():
				return nil, fmt.Errorf("wait for file lock %s held by %s since %s: %w",
					path, currentOwner, acquiredAt.Format(time.RFC3339), ctx.Err())
			}
		}
	}
}

func (m *FileLockManager) release(path string, entry *fileLock) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current := m.locks[path]; current == entry {
		delete(m.locks, path)
		close(entry.done)
	}
}
