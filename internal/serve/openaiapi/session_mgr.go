package openaiapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	agentpkg "github.com/startvibecoding/mothx/agent"
	"github.com/startvibecoding/mothx/internal/agent"
	"github.com/startvibecoding/mothx/internal/provider"
	"github.com/startvibecoding/mothx/internal/sandbox"
	"github.com/startvibecoding/mothx/internal/session"
	"github.com/startvibecoding/mothx/internal/skills"
	"github.com/startvibecoding/mothx/internal/tools"
	"github.com/startvibecoding/mothx/internal/util"
)

// APISession holds state for a single API session.
type APISession struct {
	ID           string
	WorkDir      string
	Manager      *session.Manager
	Registry     *tools.Registry
	SandboxMgr   *sandbox.Manager
	AgentMgr     *agent.AgentManager // nil unless sub-agents/delegate/workflows enabled
	SkillsMgr    *skills.Manager
	ActiveSkills map[string]bool
	ExtraContext string
	RuleContent  string
	Mode         string // session-level mode override
	DelegateMode bool   // session-level delegation mode
	Workflows    bool   // session-level workflow mode
	WebSearch    bool   // session-level hosted web search toggle
	Browser      bool   // session-level browser tool toggle
	A2AMaster    bool   // session-level A2A dispatch tool toggle
	MultiAgent   bool   // session-level sub-agent tools toggle
	LastUsed     time.Time
	mu           sync.Mutex // serializes requests within this session
	lastUsedMu   sync.RWMutex
	runMu        sync.RWMutex
	running      bool
	uses         int

	// ForceCompact is a legacy/session flag consumed by the next agent run.
	ForceCompact bool

	approvalMu       sync.Mutex
	pendingApprovals map[string]pendingSessionApproval
	activeRunID      string
	runCancel        context.CancelFunc
}

// pendingSessionApproval retains the agent instance that must be resumed after a WebUI decision.
type pendingSessionApproval struct {
	Request SessionApprovalRequest
	Agent   *agent.Agent
}

// SessionApprovalResponse is a WebUI decision for one pending approval.
type SessionApprovalResponse struct {
	Action string `json:"action"`
}

// SessionApprovalResolution is the server-confirmed terminal approval state.
type SessionApprovalResolution struct {
	ApprovalID string `json:"approvalId"`
	SessionID  string `json:"sessionId"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// ActiveSessionInfo is the management API view of an active API session.
type ActiveSessionInfo struct {
	ID           string    `json:"id"`
	WorkDir      string    `json:"workDir"`
	Mode         string    `json:"mode,omitempty"`
	DelegateMode bool      `json:"delegateMode,omitempty"`
	Workflows    bool      `json:"workflows,omitempty"`
	WebSearch    bool      `json:"webSearch,omitempty"`
	Browser      bool      `json:"browser,omitempty"`
	A2AMaster    bool      `json:"a2aMaster,omitempty"`
	MultiAgent   bool      `json:"multiAgent,omitempty"`
	Active       bool      `json:"active"`
	Running      bool      `json:"running,omitempty"`
	LastUsed     time.Time `json:"lastUsed"`
	MessageCount int       `json:"messageCount"`
	Preview      string    `json:"preview,omitempty"`
	Title        string    `json:"title,omitempty"`
}

// SessionMessageEntry is a simplified message for the WebUI.
type SessionMessageEntry struct {
	ID          string                  `json:"id,omitempty"`
	Seq         int64                   `json:"seq,omitempty"`
	Role        string                  `json:"role"`
	Content     string                  `json:"content,omitempty"`
	Contents    []provider.ContentBlock `json:"contents,omitempty"`
	AgentID     string                  `json:"agentId,omitempty"`
	ToolCallID  string                  `json:"toolCallId,omitempty"`
	ToolName    string                  `json:"toolName,omitempty"`
	Arguments   json.RawMessage         `json:"arguments,omitempty"`
	InvalidArgs string                  `json:"invalidArguments,omitempty"`
	Plan        *SessionTaskPlan        `json:"plan,omitempty"`
	IsError     bool                    `json:"isError,omitempty"`
	Summary     string                  `json:"summary,omitempty"`
	HasDetail   bool                    `json:"hasDetail,omitempty"`
}

// SessionToolResultDetail contains the full persisted result for one tool call.
type SessionToolResultDetail struct {
	ToolCallID string                  `json:"toolCallId"`
	ToolName   string                  `json:"toolName,omitempty"`
	Content    string                  `json:"content,omitempty"`
	Contents   []provider.ContentBlock `json:"contents,omitempty"`
	IsError    bool                    `json:"isError,omitempty"`
}

// SessionSubAgentInfo is the WebUI view of a managed sub-agent.
type SessionSubAgentInfo struct {
	ID           string `json:"id"`
	ParentID     string `json:"parentId,omitempty"`
	Status       string `json:"status"`
	Active       bool   `json:"active"`
	MessageCount int    `json:"messageCount"`
	LastResponse string `json:"lastResponse,omitempty"`
	Error        string `json:"error,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
}

// SessionTaskPlan is the WebUI view of a plan tool call.
type SessionTaskPlan struct {
	Title string            `json:"title,omitempty"`
	Steps []SessionPlanStep `json:"steps,omitempty"`
	Note  string            `json:"note,omitempty"`
}

// SessionPlanStep is one todo item in a plan tool call.
type SessionPlanStep struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

// ErrActiveSessionIDAmbiguous is returned when a session ID matches multiple active workdirs.
var ErrActiveSessionIDAmbiguous = errors.New("active session ID is ambiguous")

// ErrSessionToolResultNotFound is returned when a persisted tool result cannot be found.
var ErrSessionToolResultNotFound = errors.New("session tool result not found")

// ErrSessionNotFound is returned when a session cannot be found in memory or persistence.
var ErrSessionNotFound = errors.New("session not found")

// ErrSubAgentNotFound is returned when a sub-agent cannot be found in an active session.
var ErrSubAgentNotFound = errors.New("sub-agent not found")

// ErrInvalidCapability is returned when a capability patch contains an invalid value.
var ErrInvalidCapability = errors.New("invalid capability value")

// Lock acquires the session lock (one request at a time per session).
func (s *APISession) Lock() { s.mu.Lock() }

// TryLock acquires the session lock without waiting.
func (s *APISession) TryLock() bool { return s != nil && s.mu.TryLock() }

// Unlock releases the session lock.
func (s *APISession) Unlock() { s.mu.Unlock() }

// Touch updates the last-used timestamp.
func (s *APISession) Touch() {
	if s == nil {
		return
	}
	s.lastUsedMu.Lock()
	s.LastUsed = time.Now()
	s.lastUsedMu.Unlock()
}

func (s *APISession) lastUsedAt() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.lastUsedMu.RLock()
	defer s.lastUsedMu.RUnlock()
	return s.LastUsed
}

func (s *APISession) pin() {
	s.runMu.Lock()
	s.uses++
	s.runMu.Unlock()
}

func (s *APISession) unpin() {
	s.runMu.Lock()
	if s.uses > 0 {
		s.uses--
	}
	s.runMu.Unlock()
}

func (s *APISession) isInUse() bool {
	if s == nil {
		return false
	}
	s.runMu.RLock()
	defer s.runMu.RUnlock()
	return s.uses > 0
}

// SetRunning records whether a chat run is currently active for this session.
func (s *APISession) SetRunning(running bool) {
	if s == nil {
		return
	}
	s.runMu.Lock()
	s.running = running
	s.runMu.Unlock()
}

// IsRunning reports whether a chat run is currently active for this session.
func (s *APISession) IsRunning() bool {
	if s == nil {
		return false
	}
	s.runMu.RLock()
	defer s.runMu.RUnlock()
	return s.running
}

// CancelSessionRun requests cancellation of the active run for a session.
func (s *Server) CancelSessionRun(id string) error {
	if id == "" || s == nil || s.pool == nil {
		return ErrSessionNotFound
	}
	sess, err := s.pool.getExact(id)
	if err != nil || sess == nil {
		return ErrSessionNotFound
	}
	sess.approvalMu.Lock()
	cancel := sess.runCancel
	sess.approvalMu.Unlock()
	if cancel == nil || !sess.IsRunning() {
		return ErrSessionNotFound
	}
	cancel()
	return nil
}

// ActiveRunID returns the current run identifier, if any.
func (s *APISession) ActiveRunID() string {
	if s == nil {
		return ""
	}
	s.approvalMu.Lock()
	defer s.approvalMu.Unlock()
	return s.activeRunID
}

// SessionPool manages multiple concurrent API sessions.
type SessionPool struct {
	mu       sync.RWMutex
	sessions map[string]*APISession
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
		sessions: make(map[string]*APISession),
		maxSess:  maxSessions,
		idleTTL:  idleTimeout,
		stopCh:   make(chan struct{}),
	}
	if idleTimeout > 0 {
		go p.cleanupLoop()
	}
	return p
}

func (p *SessionPool) Snapshot() []*APISession {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]*APISession, 0, len(p.sessions))
	for _, sess := range p.sessions {
		result = append(result, sess)
	}
	return result
}

// Get returns an existing session by ID, or nil.
func (p *SessionPool) Get(id string) *APISession {
	return p.GetForWorkDir("", id)
}

// GetForWorkDir returns a session by workDir and ID, or nil.
func (p *SessionPool) GetForWorkDir(workDir, id string) *APISession {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if workDir != "" {
		return p.sessions[sessionPoolKey(workDir, id)]
	}
	var found *APISession
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

// Pin keeps a session resident while a caller is about to use it. The pool
// lock closes the gap between lookup and session locking, so idle eviction
// cannot replace a live session with a second instance.
func (p *SessionPool) Pin(s *APISession) bool {
	if p == nil || s == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sessions[sessionPoolKey(s.WorkDir, s.ID)] != s {
		return false
	}
	s.pin()
	return true
}

// Unpin releases a residency reference acquired by Pin.
func (p *SessionPool) Unpin(s *APISession) {
	if s != nil {
		s.unpin()
	}
}

// Put adds a session to the pool. Returns an error if the pool is at capacity.
func (p *SessionPool) Put(s *APISession) error {
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
func (p *SessionPool) Replace(oldID string, s *APISession) {
	p.ReplaceByWorkDir("", oldID, s)
}

// ReplaceByWorkDir swaps an existing session entry for a new one.
func (p *SessionPool) ReplaceByWorkDir(workDir, oldID string, s *APISession) {
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

func (p *SessionPool) listDetails() []ActiveSessionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sessions := make([]ActiveSessionInfo, 0, len(p.sessions))
	for _, s := range p.sessions {
		lastUsed := s.lastUsedAt()
		messageCount := 0
		if s.Manager != nil {
			messageCount = len(s.Manager.GetMessages())
		}
		sessions = append(sessions, ActiveSessionInfo{
			ID:           s.ID,
			WorkDir:      s.WorkDir,
			Mode:         s.Mode,
			DelegateMode: s.DelegateMode,
			Workflows:    s.Workflows,
			WebSearch:    s.WebSearch,
			Browser:      s.Browser,
			A2AMaster:    s.A2AMaster,
			MultiAgent:   s.MultiAgent,
			Active:       true,
			Running:      s.IsRunning(),
			LastUsed:     lastUsed,
			MessageCount: messageCount,
		})
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].LastUsed.Equal(sessions[j].LastUsed) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].LastUsed.After(sessions[j].LastUsed)
	})
	return sessions
}

func (p *SessionPool) getExact(id string) (*APISession, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var found *APISession
	for _, s := range p.sessions {
		if s.ID != id {
			continue
		}
		if found != nil {
			return nil, ErrActiveSessionIDAmbiguous
		}
		found = s
	}
	return found, nil
}

func (s *Server) findSessionWorkDir(id string) (string, bool, error) {
	if id == "" || s == nil {
		return "", false, nil
	}
	if s.pool != nil {
		sess, err := s.pool.getExact(id)
		if err != nil {
			return "", false, err
		}
		if sess != nil {
			return sess.WorkDir, true, nil
		}
	}
	if s.settings == nil {
		return "", false, nil
	}
	mgr, err := session.OpenByIDExact(s.settings.GetSessionDir(), id)
	if err != nil {
		return "", false, nil
	}
	if header := mgr.GetHeader(); header != nil {
		return header.Cwd, true, nil
	}
	return "", true, nil
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
		if s.isInUse() || s.IsRunning() {
			continue
		}
		if now.Sub(s.lastUsedAt()) > p.idleTTL {
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

// ListActiveSessions returns persisted sessions from sessions.db, merged with
// currently active API runtime state.
func (s *Server) ListActiveSessions() []ActiveSessionInfo {
	if s == nil || s.pool == nil {
		return nil
	}
	active := s.pool.listDetails()
	if s.settings == nil || s.cfg == nil {
		return active
	}
	details, err := session.ListAllDetailed(s.settings.GetSessionDir())
	if err != nil {
		return active
	}
	byID := make(map[string]ActiveSessionInfo, len(active)+len(details))
	for _, item := range details {
		byID[item.ID] = ActiveSessionInfo{
			ID:           item.ID,
			WorkDir:      item.Cwd,
			LastUsed:     item.ModTime,
			MessageCount: item.MessageCount,
			Preview:      item.Preview,
			Title:        item.Name,
		}
	}
	for _, item := range active {
		persisted := byID[item.ID]
		if persisted.ID == "" {
			byID[item.ID] = item
			continue
		}
		item.MessageCount = persisted.MessageCount
		if item.WorkDir == "" {
			item.WorkDir = persisted.WorkDir
		}
		if item.Preview == "" {
			item.Preview = persisted.Preview
		}
		if item.Title == "" {
			item.Title = persisted.Title
		}
		byID[item.ID] = item
	}
	sessions := make([]ActiveSessionInfo, 0, len(byID))
	for _, item := range byID {
		sessions = append(sessions, item)
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].LastUsed.Equal(sessions[j].LastUsed) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].LastUsed.After(sessions[j].LastUsed)
	})
	return sessions
}

// CapabilityOverview returns serve-level capability defaults and availability.
func (s *Server) CapabilityOverview() CapabilityOverview {
	defaults := s.defaultSessionCapabilities("", false, false)
	return CapabilityOverview{
		Modes: []string{"plan", "agent", "yolo"},
		Features: map[string]CapabilityFeature{
			"delegate":   {Available: true, Default: defaults.DelegateMode},
			"multiAgent": {Available: true, Default: defaults.MultiAgent},
			"workflows":  {Available: true, Default: defaults.Workflows},
			"webSearch":  {Available: true, Default: defaults.WebSearch},
			"browser":    {Available: true, Default: defaults.Browser},
			"a2aMaster":  {Available: true, Default: defaults.A2AMaster},
			"sandbox":    {Available: true, Default: s != nil && s.cfg != nil && s.cfg.Sandbox.Enabled},
		},
		Defaults: defaults,
	}
}

// GetSessionCapabilities returns runtime capabilities for an active or persisted session.
func (s *Server) GetSessionCapabilities(id string) (*SessionCapabilities, error) {
	if id == "" {
		return nil, ErrSessionNotFound
	}
	if s != nil && s.pool != nil {
		sess, err := s.pool.getExact(id)
		if err != nil {
			return nil, err
		}
		if sess != nil {
			caps := s.capabilitiesFromSession(sess, true, sess.Manager != nil)
			return &caps, nil
		}
	}
	if s == nil || s.settings == nil {
		return nil, ErrSessionNotFound
	}
	mgr, err := session.OpenByIDExact(s.settings.GetSessionDir(), id)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	workDir := ""
	if header := mgr.GetHeader(); header != nil {
		workDir = header.Cwd
	}
	caps := s.defaultSessionCapabilities(workDir, false, true)
	caps.ID = id
	if stored, ok, err := s.loadStoredCapabilities(id); err != nil {
		return nil, err
	} else if ok {
		applyStoredCapabilitiesToResponse(&caps, stored)
	}
	return &caps, nil
}

func (s *Server) runtimeSnapshotFromCapabilities(caps *SessionCapabilities) *SessionRuntimeSnapshot {
	if caps == nil {
		return &SessionRuntimeSnapshot{Capabilities: map[string]SessionCapabilityState{}}
	}
	snapshot := &SessionRuntimeSnapshot{
		SessionID:        caps.ID,
		Mode:             caps.Mode,
		Model:            caps.Model,
		ThinkingLevel:    caps.ThinkingLevel,
		WorkDir:          caps.WorkDir,
		Capabilities:     make(map[string]SessionCapabilityState, 6),
		PendingApprovals: []SessionApprovalRequest{},
	}
	if snapshot.Mode == "" {
		snapshot.Mode = "yolo"
	}
	state := func(available, enabled bool, unavailableReason string) SessionCapabilityState {
		reason := ""
		if !available {
			reason = unavailableReason
		}
		if available && !enabled {
			if reason == "" {
				reason = "disabled for this session"
			}
		}
		return SessionCapabilityState{
			Available:      available,
			Enabled:        enabled,
			Effective:      available && enabled,
			DisabledReason: reason,
		}
	}
	available := func(name string) bool { return s.runtimeCapabilityAvailable(name) }
	snapshot.Capabilities["browser"] = state(available("browser"), caps.Browser, "disabled by serve config")
	snapshot.Capabilities["delegate"] = state(available("delegate"), caps.DelegateMode, "disabled by serve config")
	snapshot.Capabilities["multiAgent"] = state(available("multiAgent"), caps.MultiAgent, "disabled by serve config")
	snapshot.Capabilities["workflows"] = state(available("workflows"), caps.Workflows, "disabled by serve config")
	snapshot.Capabilities["webSearch"] = state(available("webSearch"), caps.WebSearch, "disabled by serve config")
	snapshot.Capabilities["a2aMaster"] = state(available("a2aMaster"), caps.A2AMaster, "disabled by serve config")
	if s != nil && s.pool != nil && caps.ID != "" {
		if sess, err := s.pool.getExact(caps.ID); err == nil && sess != nil {
			sess.approvalMu.Lock()
			runID := sess.activeRunID
			for _, pending := range sess.pendingApprovals {
				snapshot.PendingApprovals = append(snapshot.PendingApprovals, pending.Request)
			}
			sess.approvalMu.Unlock()
			if sess.IsRunning() {
				snapshot.ActiveRun = &SessionActiveRun{RunID: runID, Status: "running"}
			}
		}
	}
	return snapshot
}

func (s *Server) runtimeCapabilityAvailable(name string) bool {
	if s == nil || s.cfg == nil {
		return false
	}
	switch name {
	case "delegate":
		return s.cfg.EnableDelegate
	case "multiAgent":
		return s.cfg.EnableSubAgents
	case "workflows":
		return s.cfg.EnableWorkflows
	case "webSearch":
		return s.cfg.EnableWebSearch
	case "browser":
		return s.cfg.EnableBrowser
	case "a2aMaster":
		return s.cfg.EnableA2AMaster
	default:
		return false
	}
}

// GetSessionRuntime returns a structured runtime snapshot for WebUI.
func (s *Server) GetSessionRuntime(id string) (*SessionRuntimeSnapshot, error) {
	if id == "" {
		return nil, ErrSessionNotFound
	}
	caps, err := s.GetSessionCapabilities(id)
	if err != nil {
		return nil, err
	}
	return s.runtimeSnapshotFromCapabilities(caps), nil
}

func (s *Server) PatchSessionRuntime(id string, patch SessionRuntimePatch) (*SessionRuntimeSnapshot, error) {
	if id == "" {
		return nil, ErrSessionNotFound
	}
	capPatch := SessionCapabilityPatch{}
	if patch.Mode != nil {
		capPatch.Mode = patch.Mode
	}
	if patch.Tools != nil {
		capPatch.WebSearch = patch.Tools.WebSearch
		capPatch.Browser = patch.Tools.Browser
		capPatch.A2AMaster = patch.Tools.A2AMaster
		capPatch.DelegateMode = patch.Tools.Delegate
		capPatch.MultiAgent = patch.Tools.MultiAgent
		capPatch.Workflows = patch.Tools.Workflows
	}
	for name, enabled := range patch.Capabilities {
		value := enabled
		switch name {
		case "browser":
			capPatch.Browser = &value
		case "delegate":
			capPatch.DelegateMode = &value
		case "multiAgent":
			capPatch.MultiAgent = &value
		case "workflows":
			capPatch.Workflows = &value
		case "webSearch":
			capPatch.WebSearch = &value
		case "a2aMaster":
			capPatch.A2AMaster = &value
		default:
			return nil, ErrInvalidCapability
		}
	}
	updated, err := s.PatchSessionCapabilities(id, capPatch)
	if err != nil {
		return nil, err
	}
	snapshot := s.runtimeSnapshotFromCapabilities(updated)
	s.publishSessionStreamEvent(id, "runtime_event", snapshot)
	return snapshot, nil
}

// PatchSessionCapabilities activates a session if needed and updates mutable runtime capabilities.
func (s *Server) PatchSessionCapabilities(id string, patch SessionCapabilityPatch) (*SessionCapabilities, error) {
	if id == "" {
		return nil, ErrSessionNotFound
	}
	workDir, found, err := s.findSessionWorkDir(id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrSessionNotFound
	}
	sess, err := s.getOrCreateSession(id, workDir)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if !s.pool.Pin(sess) {
		return nil, ErrSessionNotFound
	}
	defer s.pool.Unpin(sess)
	sess.Lock()
	defer sess.Unlock()

	before := capabilitySnapshotFromSession(sess)
	refreshContext := false
	registryChanged := false
	if patch.Mode != nil {
		mode := strings.TrimSpace(*patch.Mode)
		if err := validateCapabilityMode(mode); err != nil {
			return nil, err
		}
		sess.Mode = mode
	}
	if applyBoolOption(&sess.WebSearch, patch.WebSearch) {
		// Web search affects hosted tool injection at next agent construction.
	}
	if applyBoolOption(&sess.Browser, patch.Browser) {
		refreshContext = true
		registryChanged = true
	}
	if applyBoolOption(&sess.A2AMaster, patch.A2AMaster) {
		registryChanged = true
	}
	delegate := patch.DelegateMode
	if delegate == nil {
		delegate = patch.Delegate
	}
	if applyBoolOption(&sess.DelegateMode, delegate) {
		registryChanged = true
	}
	if applyBoolOption(&sess.MultiAgent, patch.MultiAgent) {
		registryChanged = true
	}
	if applyBoolOption(&sess.Workflows, patch.Workflows) {
		refreshContext = true
		registryChanged = true
	}
	// Mode and webSearch only affect agent construction/configuration. They do
	// not own registry state, so a mode-only WebUI PATCH must not re-register
	// session tools (or reinitialize optional integrations such as A2A).
	if registryChanged {
		if err := s.syncSessionTools(sess, refreshContext); err != nil {
			return nil, err
		}
	}
	if err := s.persistSessionCapabilitiesWithEvents(sess, before, "api_patch", "webui", "", map[string]any{
		"source": "session_capabilities_patch",
	}); err != nil {
		return nil, err
	}
	sess.Touch()

	caps := s.capabilitiesFromSession(sess, true, sess.Manager != nil)
	caps.RuntimeOnly = false
	caps.PersistenceNote = ""
	return &caps, nil
}

func (s *Server) loadStoredCapabilities(id string) (*session.SessionCapabilities, bool, error) {
	if s == nil || s.settings == nil || id == "" {
		return nil, false, nil
	}
	return session.LoadSessionCapabilities(s.settings.GetSessionDir(), id)
}

func (s *Server) applyStoredSessionCapabilities(sess *APISession) error {
	if sess == nil {
		return nil
	}
	stored, ok, err := s.loadStoredCapabilities(sess.ID)
	if err != nil || !ok {
		return err
	}
	oldBrowser := sess.Browser
	oldWorkflows := sess.Workflows
	if err := applyStoredCapabilitiesToSession(sess, stored); err != nil {
		return err
	}
	return s.syncSessionTools(sess, oldBrowser != sess.Browser || oldWorkflows != sess.Workflows)
}

func applyStoredCapabilitiesToSession(sess *APISession, stored *session.SessionCapabilities) error {
	if sess == nil || stored == nil {
		return nil
	}
	if err := validateCapabilityMode(stored.Mode); err != nil {
		return err
	}
	sess.Mode = stored.Mode
	sess.DelegateMode = stored.DelegateMode
	sess.MultiAgent = stored.MultiAgent
	sess.Workflows = stored.Workflows
	sess.WebSearch = stored.WebSearch
	sess.Browser = stored.Browser
	sess.A2AMaster = stored.A2AMaster
	return nil
}

func applyStoredCapabilitiesToResponse(caps *SessionCapabilities, stored *session.SessionCapabilities) {
	if caps == nil || stored == nil {
		return
	}
	caps.Mode = stored.Mode
	if caps.Mode == "" {
		caps.Mode = "yolo"
	}
	caps.DelegateMode = stored.DelegateMode
	caps.Delegate = stored.DelegateMode
	caps.MultiAgent = stored.MultiAgent
	caps.Workflows = stored.Workflows
	caps.WebSearch = stored.WebSearch
	caps.Browser = stored.Browser
	caps.A2AMaster = stored.A2AMaster
	caps.RuntimeOnly = false
	caps.PersistenceNote = ""
}

func (s *Server) persistSessionCapabilities(sess *APISession) error {
	if s == nil || s.settings == nil || sess == nil || sess.ID == "" {
		return nil
	}
	return session.SaveSessionCapabilities(s.settings.GetSessionDir(), session.SessionCapabilities{
		SessionID:    sess.ID,
		Mode:         sess.Mode,
		DelegateMode: sess.DelegateMode,
		MultiAgent:   sess.MultiAgent,
		Workflows:    sess.Workflows,
		WebSearch:    sess.WebSearch,
		Browser:      sess.Browser,
		A2AMaster:    sess.A2AMaster,
		UpdatedAt:    time.Now(),
	})
}

func validateCapabilityMode(mode string) error {
	switch mode {
	case "", "plan", "agent", "yolo":
		return nil
	default:
		return fmt.Errorf("%w: mode must be plan, agent, yolo, or empty string", ErrInvalidCapability)
	}
}

func (s *Server) defaultSessionCapabilities(workDir string, active bool, persisted bool) SessionCapabilities {
	mode := ""
	delegateMode := false
	workflows := false
	webSearch := false
	browser := false
	a2aMaster := false
	multiAgent := false
	if s != nil && s.cfg != nil {
		mode = s.cfg.DefaultMode
		delegateMode = s.cfg.EnableDelegate
		workflows = s.cfg.EnableWorkflows
		webSearch = s.cfg.EnableWebSearch
		browser = s.cfg.EnableBrowser
		a2aMaster = s.cfg.EnableA2AMaster
		multiAgent = s.cfg.EnableSubAgents
	}
	if mode == "" {
		mode = "yolo"
	}
	return SessionCapabilities{
		WorkDir:         workDir,
		Active:          active,
		Mode:            mode,
		DelegateMode:    delegateMode,
		Delegate:        delegateMode,
		MultiAgent:      multiAgent,
		Workflows:       workflows,
		WebSearch:       webSearch,
		Browser:         browser,
		A2AMaster:       a2aMaster,
		Model:           s.currentModelID(),
		ThinkingLevel:   s.currentThinkingLevel(),
		Persisted:       persisted,
		RuntimeOnly:     true,
		PersistenceNote: "capability changes are runtime-only until session capability persistence is implemented",
	}
}

func (s *Server) capabilitiesFromSession(sess *APISession, active bool, persisted bool) SessionCapabilities {
	if sess == nil {
		return s.defaultSessionCapabilities("", active, persisted)
	}
	caps := s.defaultSessionCapabilities(sess.WorkDir, active, persisted)
	caps.ID = sess.ID
	if sess.Mode != "" {
		caps.Mode = sess.Mode
	}
	caps.DelegateMode = sess.DelegateMode
	caps.Delegate = sess.DelegateMode
	caps.MultiAgent = sess.MultiAgent
	caps.Workflows = sess.Workflows
	caps.WebSearch = sess.WebSearch
	caps.Browser = sess.Browser
	caps.A2AMaster = sess.A2AMaster
	if _, ok, err := s.loadStoredCapabilities(sess.ID); err == nil && ok {
		caps.RuntimeOnly = false
		caps.PersistenceNote = ""
	}
	return caps
}

func (s *Server) currentModelID() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.model == nil {
		return ""
	}
	return s.model.ID
}

func (s *Server) currentThinkingLevel() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	if s.cfg.DefaultThinkingLevel != "" {
		return s.cfg.DefaultThinkingLevel
	}
	if s.settings != nil {
		return s.settings.DefaultThinkingLevel
	}
	return ""
}

// DeleteActiveSession deletes one active session from persistence and the runtime pool.
func (s *Server) DeleteActiveSession(id string) (bool, error) {
	if s == nil || s.pool == nil {
		return false, nil
	}
	sess, err := s.pool.getExact(id)
	if err != nil {
		return false, err
	}
	if sess == nil {
		if s.settings == nil {
			return false, nil
		}
		mgr, err := session.OpenByIDExact(s.settings.GetSessionDir(), id)
		if err != nil {
			return false, nil
		}
		if err := session.DeleteSession(mgr.GetFile(), s.settings.GetSessionDir()); err != nil {
			return false, err
		}
		return true, nil
	}
	if sess.Manager != nil && sess.Manager.GetFile() != "" && s.settings != nil {
		if err := session.DeleteSession(sess.Manager.GetFile(), s.settings.GetSessionDir()); err != nil {
			return false, err
		}
	}
	s.pool.RemoveByWorkDir(sess.WorkDir, sess.ID)

	s.mu.Lock()
	if s.defaultSessionIDs != nil {
		for workDir, defaultID := range s.defaultSessionIDs {
			if defaultID == sess.ID {
				delete(s.defaultSessionIDs, workDir)
			}
		}
	}
	s.mu.Unlock()

	return true, nil
}

// GetSessionMessages returns the message history for a persisted session.
func (s *Server) GetSessionMessages(id string) ([]SessionMessageEntry, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	if s.settings != nil && id != "" {
		if _, found, err := s.findSessionWorkDir(id); err != nil {
			return nil, err
		} else if found {
			messages, err := session.ListSessionMessagesWithSeq(s.settings.GetSessionDir(), id)
			if err != nil {
				return nil, err
			}
			return sequencedMessagesToEntries(messages), nil
		}
	}
	messages, err := s.sessionMessages(id)
	if err != nil {
		return nil, err
	}
	return sessionMessagesToEntries(messages), nil
}

// GetSessionToolResult returns the full persisted result for a tool call.
func (s *Server) GetSessionToolResult(id, toolCallID string) (*SessionToolResultDetail, error) {
	if s == nil || s.pool == nil {
		return nil, nil
	}
	if toolCallID == "" {
		return nil, ErrSessionToolResultNotFound
	}
	messages, err := s.sessionMessages(id)
	if err != nil {
		return nil, err
	}
	for _, msg := range messages {
		if msg.SystemInjected || msg.Role != "toolResult" || msg.ToolCallID != toolCallID {
			continue
		}
		detail := &SessionToolResultDetail{
			ToolCallID: msg.ToolCallID,
			ToolName:   msg.ToolName,
			Content:    toolResultText(msg),
			IsError:    msg.IsError,
		}
		if len(msg.Contents) > 0 {
			detail.Contents = cloneContentBlocks(msg.Contents)
		}
		return detail, nil
	}
	return nil, ErrSessionToolResultNotFound
}

// GetSessionSubAgents returns sub-agent statuses for an active session.
func (s *Server) GetSessionSubAgents(id string) ([]SessionSubAgentInfo, error) {
	if s == nil || s.pool == nil {
		return nil, ErrSessionNotFound
	}
	sess, err := s.pool.getExact(id)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if sess.AgentMgr == nil {
		return []SessionSubAgentInfo{}, nil
	}

	statuses := sess.AgentMgr.Statuses()
	out := make([]SessionSubAgentInfo, 0, len(statuses))
	for _, st := range statuses {
		if st.ParentID == "" {
			continue
		}
		info := SessionSubAgentInfo{
			ID:           string(st.ID),
			ParentID:     string(st.ParentID),
			Status:       st.State,
			LastResponse: st.Result,
			Error:        st.Error,
		}
		if info.Status == "" {
			info.Status = "unknown"
		}
		if !st.StartedAt.IsZero() {
			info.StartedAt = st.StartedAt.Format(time.RFC3339)
		}
		if !st.UpdatedAt.IsZero() {
			info.UpdatedAt = st.UpdatedAt.Format(time.RFC3339)
		}
		if a, ok := sess.AgentMgr.Get(st.ID); ok {
			info.Active = true
			info.MessageCount = len(a.GetMessages())
		}
		out = append(out, info)
	}
	return out, nil
}

// GetSessionSubAgentMessages returns the in-memory transcript for a sub-agent.
func (s *Server) GetSessionSubAgentMessages(id, agentID string) ([]SessionMessageEntry, error) {
	if s == nil || s.pool == nil {
		return nil, ErrSessionNotFound
	}
	sess, err := s.pool.getExact(id)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if sess.AgentMgr == nil || agentID == "" {
		return nil, ErrSubAgentNotFound
	}
	a, ok := sess.AgentMgr.Get(agentpkg.AgentID(agentID))
	if !ok {
		if _, statusOK := sess.AgentMgr.Status(agentpkg.AgentID(agentID)); statusOK {
			return []SessionMessageEntry{}, nil
		}
		return nil, ErrSubAgentNotFound
	}
	entries := sessionMessagesToEntries(agent.MessagesFromPublic(a.GetMessages()))
	for i := range entries {
		entries[i].AgentID = agentID
		if entries[i].ID == "" {
			entries[i].ID = fmt.Sprintf("%s:%d", agentID, i)
		}
	}
	return entries, nil
}

// GetSessionRunEvents returns persisted run lifecycle events for a session.
func (s *Server) GetSessionRunEvents(id string) ([]SessionRunEventEntry, error) {
	if s == nil || s.settings == nil || id == "" {
		return nil, ErrSessionNotFound
	}
	if _, found, err := s.findSessionWorkDir(id); err != nil {
		return nil, err
	} else if !found {
		return nil, ErrSessionNotFound
	}
	events, err := session.ListSessionRunEventsWithSeq(s.settings.GetSessionDir(), id)
	if err != nil {
		return nil, err
	}
	out := make([]SessionRunEventEntry, 0, len(events))
	for _, item := range events {
		out = append(out, sessionRunEventToEntry(item.Event, item.Seq))
	}
	return out, nil
}

// GetSessionCapabilityEvents returns persisted capability transition events for a session.
func (s *Server) GetSessionCapabilityEvents(id string) ([]SessionCapabilityEventEntry, error) {
	if s == nil || s.settings == nil || id == "" {
		return nil, ErrSessionNotFound
	}
	if _, found, err := s.findSessionWorkDir(id); err != nil {
		return nil, err
	} else if !found {
		return nil, ErrSessionNotFound
	}
	events, err := session.ListSessionCapabilityEventsWithSeq(s.settings.GetSessionDir(), id)
	if err != nil {
		return nil, err
	}
	out := make([]SessionCapabilityEventEntry, 0, len(events))
	for _, item := range events {
		out = append(out, sessionCapabilityEventToEntry(item.Event, item.Seq))
	}
	return out, nil
}

func (s *Server) sessionMessages(id string) ([]provider.Message, error) {
	if id == "" {
		workDir := s.cfg.GetWorkDir()
		s.mu.RLock()
		defaultID := s.defaultSessionIDs[workDir]
		s.mu.RUnlock()
		id = defaultID
	}
	if id == "" {
		return nil, nil
	}
	if s.settings != nil {
		mgr, err := session.OpenByIDExact(s.settings.GetSessionDir(), id)
		if err == nil {
			return mgr.GetMessages(), nil
		}
	}
	sess, err := s.pool.getExact(id)
	if err != nil {
		return nil, err
	}
	if sess == nil || sess.Manager == nil {
		return nil, nil
	}
	return sess.Manager.GetMessages(), nil
}

func formatEventTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func decodeEventData(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil || len(data) == 0 {
		return nil
	}
	return data
}

func sessionRunEventToEntry(ev session.SessionRunEvent, seq int64) SessionRunEventEntry {
	return SessionRunEventEntry{
		Seq:       seq,
		ID:        ev.ID,
		SessionID: ev.SessionID,
		RunID:     ev.RunID,
		EventType: ev.EventType,
		Source:    ev.Source,
		Status:    ev.Status,
		Model:     ev.Model,
		Mode:      ev.Mode,
		Timestamp: formatEventTimestamp(ev.Timestamp),
		Data:      decodeEventData(ev.Data),
	}
}

func sessionCapabilityEventToEntry(ev session.SessionCapabilityEvent, seq int64) SessionCapabilityEventEntry {
	return SessionCapabilityEventEntry{
		Seq:        seq,
		ID:         ev.ID,
		SessionID:  ev.SessionID,
		RunID:      ev.RunID,
		EventType:  ev.EventType,
		Source:     ev.Source,
		Actor:      ev.Actor,
		Capability: ev.Capability,
		OldValue:   ev.OldValue,
		NewValue:   ev.NewValue,
		Timestamp:  formatEventTimestamp(ev.Timestamp),
		Data:       decodeEventData(ev.Data),
	}
}

func sessionMessagesToEntries(msgs []provider.Message) []SessionMessageEntry {
	var entries []SessionMessageEntry
	for _, m := range msgs {
		entries = append(entries, providerMessageToSessionEntries(m, 0, "")...)
	}
	return entries
}

func sequencedMessagesToEntries(msgs []session.SequencedMessage) []SessionMessageEntry {
	var entries []SessionMessageEntry
	for _, item := range msgs {
		entries = append(entries, providerMessageToSessionEntries(item.Message, item.Seq, item.EntryID)...)
	}
	return entries
}

func providerMessageToSessionEntries(m provider.Message, seq int64, entryID string) []SessionMessageEntry {
	var entries []SessionMessageEntry
	if m.SystemInjected {
		return entries
	}
	entryIDFor := func(suffix string) string {
		if entryID == "" {
			return ""
		}
		if suffix == "" {
			return entryID
		}
		return entryID + ":" + suffix
	}
	withCursor := func(entry SessionMessageEntry, suffix string) SessionMessageEntry {
		entry.ID = entryIDFor(suffix)
		entry.Seq = seq
		return entry
	}
	switch m.Role {
	case "user":
		content := messageText(m)
		entry := SessionMessageEntry{Role: m.Role, Content: content}
		if len(m.Contents) > 0 {
			entry.Contents = cloneContentBlocks(m.Contents)
		}
		entries = append(entries, withCursor(entry, ""))
	case "assistant":
		content := messageText(m)
		if content != "" {
			entries = append(entries, withCursor(SessionMessageEntry{Role: m.Role, Content: content}, "assistant"))
		}
		for idx, block := range m.Contents {
			if block.ToolCall == nil {
				continue
			}
			suffix := fmt.Sprintf("tool:%d", idx)
			if block.ToolCall.ID != "" {
				suffix = "tool:" + block.ToolCall.ID
			}
			entries = append(entries, withCursor(SessionMessageEntry{
				Role:        "toolCall",
				ToolCallID:  block.ToolCall.ID,
				ToolName:    block.ToolCall.Name,
				Arguments:   validRawMessage(block.ToolCall.Arguments),
				InvalidArgs: block.ToolCall.InvalidArguments,
				Plan:        planFromToolCall(block.ToolCall.Name, block.ToolCall.Arguments),
			}, suffix))
		}
	case "toolResult":
		suffix := "toolResult"
		if m.ToolCallID != "" {
			suffix += ":" + m.ToolCallID
		}
		entries = append(entries, withCursor(SessionMessageEntry{
			Role:       "toolResult",
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
			IsError:    m.IsError,
			Summary:    summarizeToolResult(m),
			HasDetail:  true,
		}, suffix))
	}
	return entries
}

func messageText(msg provider.Message) string {
	if msg.Content != "" {
		return msg.Content
	}
	var content string
	for _, b := range msg.Contents {
		if b.Type == "text" && b.Text != "" {
			content += b.Text
		}
	}
	return content
}

func toolResultText(msg provider.Message) string {
	text := messageText(msg)
	if text != "" {
		return text
	}
	if len(msg.Contents) > 0 {
		return "(rich tool result)"
	}
	return ""
}

func summarizeToolResult(msg provider.Message) string {
	text := strings.TrimSpace(toolResultText(msg))
	if text == "" {
		text = "(empty result)"
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	return util.TruncateWithSuffix(text, 140, "...")
}

func planFromToolCall(toolName string, args json.RawMessage) *SessionTaskPlan {
	if toolName != "plan" || len(args) == 0 || !json.Valid(args) {
		return nil
	}
	var raw struct {
		Title string `json:"title"`
		Steps []struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"steps"`
		Note string `json:"note"`
	}
	if err := json.Unmarshal(args, &raw); err != nil || len(raw.Steps) == 0 {
		return nil
	}
	plan := &SessionTaskPlan{
		Title: strings.TrimSpace(raw.Title),
		Note:  strings.TrimSpace(raw.Note),
		Steps: make([]SessionPlanStep, 0, len(raw.Steps)),
	}
	for _, step := range raw.Steps {
		title := strings.TrimSpace(step.Title)
		if title == "" {
			continue
		}
		status := normalizeSessionPlanStatus(step.Status)
		if status == "" {
			status = "pending"
		}
		plan.Steps = append(plan.Steps, SessionPlanStep{Title: title, Status: status})
	}
	if len(plan.Steps) == 0 {
		return nil
	}
	return plan
}

func normalizeSessionPlanStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "running", "done", "failed":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return ""
	}
}

func validRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if !json.Valid(raw) {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func cloneContentBlocks(blocks []provider.ContentBlock) []provider.ContentBlock {
	cloned := make([]provider.ContentBlock, len(blocks))
	for i, block := range blocks {
		cloned[i] = block
		if block.Image != nil {
			image := *block.Image
			cloned[i].Image = &image
		}
		if block.ToolCall != nil {
			toolCall := *block.ToolCall
			cloned[i].ToolCall = &toolCall
		}
		if block.CacheControl != nil {
			cacheControl := *block.CacheControl
			cloned[i].CacheControl = &cacheControl
		}
	}
	return cloned
}
