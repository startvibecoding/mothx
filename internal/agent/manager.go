package agent

import (
	"fmt"
	"sync"
	"sync/atomic"

	agentpkg "github.com/startvibecoding/vibecoding/agent"
)

// AgentManager manages the lifecycle of all agent instances.
type AgentManager struct {
	mu       sync.RWMutex
	agents   map[agentpkg.AgentID]agentpkg.Agent
	parentOf map[agentpkg.AgentID]agentpkg.AgentID
	children map[agentpkg.AgentID][]agentpkg.AgentID
	factory  *AgentFactory
	counter  int64
}

// NewAgentManager creates a new agent manager.
func NewAgentManager(factory *AgentFactory) *AgentManager {
	return &AgentManager{
		agents:   make(map[agentpkg.AgentID]agentpkg.Agent),
		parentOf: make(map[agentpkg.AgentID]agentpkg.AgentID),
		children: make(map[agentpkg.AgentID][]agentpkg.AgentID),
		factory:  factory,
	}
}

// Create creates a new agent and registers it.
// If opts.ParentID is set, validates the parent exists and is a top-level agent.
func (m *AgentManager) Create(opts AgentOptions) (agentpkg.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID if not provided
	if opts.ID == "" {
		opts.ID = agentpkg.AgentID(fmt.Sprintf("agent-%d", atomic.AddInt64(&m.counter, 1)))
	}

	// Validate parent
	if opts.ParentID != "" {
		parent, ok := m.agents[opts.ParentID]
		if !ok {
			return nil, fmt.Errorf("parent agent %s not found", opts.ParentID)
		}
		// Decision 5: sub-agents cannot nest (only top-level agents can spawn)
		if parent.ParentID() != "" {
			return nil, fmt.Errorf("parent agent %s is itself a sub-agent; nesting is not allowed", opts.ParentID)
		}
	}

	a := m.factory.Create(opts)
	m.agents[opts.ID] = a
	if opts.ParentID != "" {
		m.parentOf[opts.ID] = opts.ParentID
		m.children[opts.ParentID] = append(m.children[opts.ParentID], opts.ID)
	}

	return a, nil
}

// Get returns an agent by ID.
func (m *AgentManager) Get(id agentpkg.AgentID) (agentpkg.Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.agents[id]
	return a, ok
}

// Destroy stops and removes an agent and all its children.
func (m *AgentManager) Destroy(id agentpkg.AgentID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	a, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent %s not found", id)
	}

	// Recursively destroy children first
	children := m.children[id]
	for _, childID := range children {
		m.destroyLocked(childID)
	}

	// Abort the agent
	a.Abort()

	// Remove from parent's children list
	if parentID, hasParent := m.parentOf[id]; hasParent {
		siblings := m.children[parentID]
		filtered := make([]agentpkg.AgentID, 0, len(siblings))
		for _, sid := range siblings {
			if sid != id {
				filtered = append(filtered, sid)
			}
		}
		m.children[parentID] = filtered
	}

	// Remove self
	delete(m.agents, id)
	delete(m.parentOf, id)
	delete(m.children, id)

	return nil
}

// destroyLocked destroys an agent without locking (caller must hold lock).
func (m *AgentManager) destroyLocked(id agentpkg.AgentID) {
	// Destroy children recursively
	for _, childID := range m.children[id] {
		m.destroyLocked(childID)
	}
	if a, ok := m.agents[id]; ok {
		a.Abort()
	}
	delete(m.agents, id)
	delete(m.parentOf, id)
	delete(m.children, id)
}

// List returns all agent IDs.
func (m *AgentManager) List() []agentpkg.AgentID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]agentpkg.AgentID, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids
}

// Children returns the children of an agent.
func (m *AgentManager) Children(id agentpkg.AgentID) []agentpkg.AgentID {
	m.mu.RLock()
	defer m.mu.RUnlock()
	children := m.children[id]
	if children == nil {
		return nil
	}
	result := make([]agentpkg.AgentID, len(children))
	copy(result, children)
	return result
}

// Parent returns the parent ID of an agent.
func (m *AgentManager) Parent(id agentpkg.AgentID) (agentpkg.AgentID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pid, ok := m.parentOf[id]
	return pid, ok
}

// Count returns the number of active agents.
func (m *AgentManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}
