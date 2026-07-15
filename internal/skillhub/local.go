package skillhub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const metadataFileName = ".mothx-skillhub.json"

type InstallMetadata struct {
	Market      Market    `json:"market"`
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installedAt"`
	SourceURL   string    `json:"sourceURL"`
}

type LocalIndex struct{ entries map[string]InstalledState }

func NewLocalIndex(globalDir string, projectDirs []string) (*LocalIndex, error) {
	index := &LocalIndex{entries: make(map[string]InstalledState)}
	if err := index.scan(globalDir, "global"); err != nil {
		return nil, err
	}
	// First project directory has the highest priority.
	for _, dir := range projectDirs {
		if err := index.scan(dir, "project"); err != nil {
			return nil, err
		}
	}
	return index, nil
}

func (i *LocalIndex) scan(root, scope string) error {
	if root == "" {
		return nil
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		metadata, err := readMetadata(dir)
		if err != nil {
			continue
		}
		key := installedKey(metadata.Market, metadata.ID)
		if _, exists := i.entries[key]; !exists {
			i.entries[key] = InstalledState{Installed: true, Scope: scope, Dir: dir, Version: metadata.Version}
		}
	}
	return nil
}

func (i *LocalIndex) State(market Market, id string) *InstalledState {
	state, ok := i.entries[installedKey(market, id)]
	if !ok {
		return nil
	}
	copy := state
	return &copy
}
func (i *LocalIndex) Apply(items []SkillSummary) {
	for n := range items {
		state := i.State(items[n].Market, items[n].ID)
		if state != nil {
			state.UpdateAvailable = versionsDiffer(state.Version, items[n].Version)
		}
		items[n].Installed = state
	}
}
func (i *LocalIndex) List() []InstalledState {
	out := make([]InstalledState, 0, len(i.entries))
	for _, state := range i.entries {
		out = append(out, state)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Dir < out[b].Dir })
	return out
}
func installedKey(market Market, id string) string { return string(market) + "\x00" + id }
func readMetadata(dir string) (InstallMetadata, error) {
	var metadata InstallMetadata
	data, err := os.ReadFile(filepath.Join(dir, metadataFileName))
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, err
	}
	if metadata.Market == "" || strings.TrimSpace(metadata.ID) == "" {
		return metadata, os.ErrInvalid
	}
	return metadata, nil
}
