package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileStore persists workflow state as JSON files.
type FileStore struct {
	dir string
}

// NewFileStore creates a file-backed workflow store rooted at dir.
func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

func (s *FileStore) Save(ctx context.Context, state *RunState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("workflow state is nil")
	}
	if state.ID == "" {
		return fmt.Errorf("workflow state id is required")
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(state.ID)
	tmp, err := os.CreateTemp(s.dir, ".tmp-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

func (s *FileStore) Load(ctx context.Context, id string) (*RunState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("workflow run id is required")
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var state RunState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *FileStore) List(ctx context.Context) ([]RunState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var states []RunState
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var state RunState
		if err := json.Unmarshal(data, &state); err == nil {
			states = append(states, state)
		}
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].StartedAt.After(states[j].StartedAt)
	})
	return states, nil
}

func (s *FileStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}
