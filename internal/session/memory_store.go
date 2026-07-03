package session

import (
	"sync"
	"time"

	"github.com/startvibecoding/mothx/internal/provider"
)

// MemoryStore is an in-memory implementation of Store for testing.
// It does not persist data to disk.
type MemoryStore struct {
	mu      sync.RWMutex
	header  *Header
	entries []interface{}
	leafID  *string
	file    string
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	m.header = &Header{
		Type:      EntrySession,
		Version:   CurrentVersion,
		ID:        id,
		Timestamp: time.Now(),
	}
	return nil
}

func (m *MemoryStore) InitWithID(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		id = GenerateID()
	}
	m.header = &Header{
		Type:      EntrySession,
		Version:   CurrentVersion,
		ID:        id,
		Timestamp: time.Now(),
	}
	return nil
}

func (m *MemoryStore) AppendMessage(msg provider.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	entry := MessageEntry{
		EntryBase: EntryBase{
			Type:      EntryMessage,
			ID:        id,
			ParentID:  m.leafID,
			Timestamp: time.Now(),
		},
		Message: msg,
	}
	m.entries = append(m.entries, entry)
	m.leafID = &id
	return id, nil
}

func (m *MemoryStore) AppendCompaction(summary, firstKeptEntryID string, tokensBefore int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	entry := CompactionEntry{
		EntryBase: EntryBase{
			Type:      EntryCompaction,
			ID:        id,
			ParentID:  m.leafID,
			Timestamp: time.Now(),
		},
		Summary:        summary,
		FirstKeptEntry: firstKeptEntryID,
		TokensBefore:   tokensBefore,
	}
	m.entries = append(m.entries, entry)
	m.leafID = &id
	return id, nil
}

func (m *MemoryStore) AppendModelChange(providerName, modelID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	entry := ModelChangeEntry{
		EntryBase: EntryBase{
			Type:      EntryModelChange,
			ID:        id,
			ParentID:  m.leafID,
			Timestamp: time.Now(),
		},
		Provider: providerName,
		ModelID:  modelID,
	}
	m.entries = append(m.entries, entry)
	m.leafID = &id
	return id, nil
}

func (m *MemoryStore) AppendThinkingLevelChange(level string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	entry := ThinkingLevelChangeEntry{
		EntryBase: EntryBase{
			Type:      EntryThinkingChange,
			ID:        id,
			ParentID:  m.leafID,
			Timestamp: time.Now(),
		},
		ThinkingLevel: level,
	}
	m.entries = append(m.entries, entry)
	m.leafID = &id
	return id, nil
}

func (m *MemoryStore) AppendSessionInfo(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := GenerateID()
	entry := SessionInfoEntry{
		EntryBase: EntryBase{
			Type:      EntrySessionInfo,
			ID:        id,
			ParentID:  m.leafID,
			Timestamp: time.Now(),
		},
		Name: name,
	}
	m.entries = append(m.entries, entry)
	m.leafID = &id
	return id, nil
}

func (m *MemoryStore) GetMessages() []provider.Message {
	state := m.GetReplayState()
	return state.Messages
}

func (m *MemoryStore) GetReplayState() ReplayState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state := buildReplayState(m.entries)
	return ReplayState{
		Messages: state.messages,
		EntryIDs: state.entryIDs,
	}
}

func (m *MemoryStore) GetLeafID() *string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.leafID
}

func (m *MemoryStore) GetLatestCompaction() (CompactionEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return latestCompactionLocked(m.entries)
}

func (m *MemoryStore) GetFile() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.file
}

func (m *MemoryStore) GetHeader() *Header {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.header
}

// Compile-time check that MemoryStore implements Store.
var _ Store = (*MemoryStore)(nil)
