package gateway

import (
	"sync"
	"time"

	"github.com/startvibecoding/vibecoding/internal/agent"
	"github.com/startvibecoding/vibecoding/internal/session"
	"github.com/startvibecoding/vibecoding/internal/skills"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

// GatewaySession holds state for a single gateway session.
type GatewaySession struct {
	ID           string
	WorkDir      string
	Manager      *session.Manager
	Registry     *tools.Registry
	AgentMgr     *agent.AgentManager // nil unless sub-agents/delegate/workflows enabled
	SkillsMgr    *skills.Manager
	ExtraContext string
	Mode         string // session-level mode override
	DelegateMode bool   // session-level delegation mode
	Workflows    bool   // session-level workflow mode
	LastUsed     time.Time
	mu           sync.Mutex // serializes requests within this session

	// ForceCompact is set by /compact command and consumed by the next agent run.
	ForceCompact bool
}

// Lock acquires the session lock (one request at a time per session).
func (s *GatewaySession) Lock() { s.mu.Lock() }

// Unlock releases the session lock.
func (s *GatewaySession) Unlock() { s.mu.Unlock() }

// Touch updates the last-used timestamp.
func (s *GatewaySession) Touch() { s.LastUsed = time.Now() }

// SessionPool manages multiple concurrent gateway sessions.
type SessionPool struct {
	mu       sync.RWMutex
	sessions map[string]*GatewaySession
	maxSess  int
	idleTTL  time.Duration
	stopCh   chan struct{}
}

func sessionPoolKey(workDir, id string) string {
	return workDir + "\x00" + id
}

// NewSessionPool creates a session pool.
func NewSessionPool(maxSessions int, idleTimeout time.Duration) *SessionPool {
	p := &SessionPool{
		sessions: make(map[string]*GatewaySession),
		maxSess:  maxSessions,
		idleTTL:  idleTimeout,
		stopCh:   make(chan struct{}),
	}
	if idleTimeout > 0 {
		go p.cleanupLoop()
	}
	return p
}

// Get returns an existing session by ID, or nil.
func (p *SessionPool) Get(id string) *GatewaySession {
	return p.GetForWorkDir("", id)
}

// GetForWorkDir returns a session by workDir and ID, or nil.
func (p *SessionPool) GetForWorkDir(workDir, id string) *GatewaySession {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if workDir != "" {
		return p.sessions[sessionPoolKey(workDir, id)]
	}
	var found *GatewaySession
	for _, s := range p.sessions {
		if s.ID != id {
			continue
		}
		if found != nil {
			return nil
		}
		found = s
	}
	return found
}

// Put adds a session to the pool. Returns an error if the pool is at capacity.
func (p *SessionPool) Put(s *GatewaySession) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := sessionPoolKey(s.WorkDir, s.ID)
	if p.maxSess > 0 && len(p.sessions) >= p.maxSess {
		// Check if we have an existing entry (replace is OK)
		if _, exists := p.sessions[key]; !exists {
			return &PoolFullError{Max: p.maxSess}
		}
	}
	s.Touch()
	p.sessions[key] = s
	return nil
}

// Remove removes a session by ID.
func (p *SessionPool) Remove(id string) {
	p.RemoveByWorkDir("", id)
}

// RemoveByWorkDir removes a session by workDir and ID.
func (p *SessionPool) RemoveByWorkDir(workDir, id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if workDir != "" {
		delete(p.sessions, sessionPoolKey(workDir, id))
		return
	}
	var key string
	var found bool
	for k, s := range p.sessions {
		if s.ID != id {
			continue
		}
		if found {
			return
		}
		key = k
		found = true
	}
	if found {
		delete(p.sessions, key)
	}
}

// Replace swaps an existing session entry for a new one.
func (p *SessionPool) Replace(oldID string, s *GatewaySession) {
	p.ReplaceByWorkDir("", oldID, s)
}

// ReplaceByWorkDir swaps an existing session entry for a new one.
func (p *SessionPool) ReplaceByWorkDir(workDir, oldID string, s *GatewaySession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if oldID != "" {
		if workDir != "" {
			delete(p.sessions, sessionPoolKey(workDir, oldID))
		} else {
			for k, sess := range p.sessions {
				if sess.ID == oldID {
					delete(p.sessions, k)
					break
				}
			}
		}
	}
	if s != nil {
		s.Touch()
		key := sessionPoolKey(s.WorkDir, s.ID)
		if _, exists := p.sessions[key]; !exists && p.maxSess > 0 && len(p.sessions) >= p.maxSess {
			for k, sess := range p.sessions {
				if sess.ID == s.ID && sess.WorkDir == s.WorkDir {
					key = k
					break
				}
			}
		}
		p.sessions[key] = s
	}
}

// Count returns the number of active sessions.
func (p *SessionPool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.sessions)
}

// List returns all session IDs.
func (p *SessionPool) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0, len(p.sessions))
	for id := range p.sessions {
		ids = append(ids, id)
	}
	return ids
}

// ListForWorkDir returns all session IDs for a specific workDir.
func (p *SessionPool) ListForWorkDir(workDir string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0)
	for _, s := range p.sessions {
		if s.WorkDir == workDir {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

// Stop shuts down the cleanup goroutine.
func (p *SessionPool) Stop() {
	close(p.stopCh)
}

// cleanupLoop periodically removes idle sessions.
func (p *SessionPool) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.evictIdle()
		}
	}
}

func (p *SessionPool) evictIdle() {
	if p.idleTTL <= 0 {
		return
	}
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, s := range p.sessions {
		if now.Sub(s.LastUsed) > p.idleTTL {
			delete(p.sessions, id)
		}
	}
}

// PoolFullError is returned when the session pool is at capacity.
type PoolFullError struct {
	Max int
}

func (e *PoolFullError) Error() string {
	return "session pool is at capacity"
}
