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
		if err == nil {
			key := installedKey(metadata.Market, metadata.ID)
			if _, exists := i.entries[key]; !exists {
				i.entries[key] = InstalledState{Installed: true, Scope: scope, Dir: dir, Market: metadata.Market, ID: metadata.ID, Name: entry.Name(), Version: metadata.Version}
			}
			continue
		}
		if !hasSkillFile(dir) {
			continue
		}
		key := localInstalledKey(dir)
		if _, exists := i.entries[key]; !exists {
			i.entries[key] = InstalledState{Installed: true, Scope: scope, Dir: dir, Name: entry.Name(), Local: true}
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
func (i *LocalIndex) FindDir(market Market, id string) (InstalledState, bool) {
	state := i.State(market, id)
	return func() (InstalledState, bool) {
		if state == nil {
			return InstalledState{}, false
		}
		return *state, true
	}()
}
func (i *LocalIndex) Apply(items []SkillSummary) {
	for n := range items {
		state := i.State(items[n].Market, items[n].ID)
		if state == nil {
			state = i.localState(items[n])
		}
		if state != nil {
			state.UpdateAvailable = !state.Local && versionsDiffer(state.Version, items[n].Version)
		}
		items[n].Installed = state
	}
}
func (i *LocalIndex) localState(item SkillSummary) *InstalledState {
	for _, state := range i.entries {
		if state.Local && (state.Name == item.Slug || state.Name == item.Name || state.Name == item.DisplayName) {
			copy := state
			return &copy
		}
	}
	return nil
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
func localInstalledKey(dir string) string          { return "local\x00" + filepath.Clean(dir) }
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
