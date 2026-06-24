package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// AllowConfig holds runtime auto-approval settings that are persisted separately
// from settings.json in allow.json.
//
//   - AutoEdit: when true, write/edit tools auto-approve in agent mode.
//   - EditPaths: glob whitelist of paths whose write/edit auto-approve in agent
//     mode. Supports "**" (cross-directory) and "*" (single segment).
//
// Loading follows the same global->project override order as settings.json:
// AutoEdit is taken from the project file when present, otherwise the global
// file. EditPaths is project-level only (never loaded from the global file).
type AllowConfig struct {
	AutoEdit  bool     `json:"autoEdit,omitempty"`
	EditPaths []string `json:"editPaths,omitempty"`

	mu                 sync.RWMutex `json:"-"`
	projectAutoEditSet bool         `json:"-"`
}

// GlobalAllowPath returns the global allow.json path.
func GlobalAllowPath() string {
	return filepath.Join(ConfigDir(), "allow.json")
}

// ProjectAllowPath returns the project-level allow.json path.
func ProjectAllowPath() string {
	return filepath.Join(".vibe", "allow.json")
}

// LoadAllow loads allow configuration with global->project override semantics.
// AutoEdit follows the override order; EditPaths is project-level only.
// A missing file is not an error; it yields a zero-value (non-permissive) config.
func LoadAllow() *AllowConfig {
	c := &AllowConfig{}

	// Global: only autoEdit is honored.
	if data, err := os.ReadFile(GlobalAllowPath()); err == nil {
		if v, ok := readAllowAutoEdit(data); ok {
			c.AutoEdit = v
		}
	}

	// Project: overrides autoEdit and is the sole source of editPaths.
	if data, err := os.ReadFile(ProjectAllowPath()); err == nil {
		var p AllowConfig
		if json.Unmarshal(data, &p) == nil {
			if v, ok := readAllowAutoEdit(data); ok {
				c.AutoEdit = v
				c.projectAutoEditSet = true
			}
			c.EditPaths = p.EditPaths
		}
	}

	return c
}

// SetAutoEdit updates the in-memory AutoEdit flag.
func (c *AllowConfig) SetAutoEdit(v bool) {
	c.mu.Lock()
	c.AutoEdit = v
	c.mu.Unlock()
}

// SetProjectAutoEdit updates the AutoEdit flag and marks it as explicitly set
// at project scope, so SaveProject can persist false as an intentional override.
func (c *AllowConfig) SetProjectAutoEdit(v bool) {
	c.mu.Lock()
	c.AutoEdit = v
	c.projectAutoEditSet = true
	c.mu.Unlock()
}

// SetGlobalAutoEdit updates the effective AutoEdit flag only when the project
// file does not explicitly override it. It returns the current effective value.
func (c *AllowConfig) SetGlobalAutoEdit(v bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.projectAutoEditSet {
		c.AutoEdit = v
	}
	return c.AutoEdit
}

// GetAutoEdit reports the current AutoEdit flag.
func (c *AllowConfig) GetAutoEdit() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoEdit
}

// EditPathList returns a copy of the current edit-path whitelist.
func (c *AllowConfig) EditPathList() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, len(c.EditPaths))
	copy(out, c.EditPaths)
	return out
}

// AddEditPath appends a glob to the whitelist if not already present.
// Returns true when the list changed.
func (c *AllowConfig) AddEditPath(glob string) bool {
	glob = strings.TrimSpace(glob)
	if glob == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, g := range c.EditPaths {
		if g == glob {
			return false
		}
	}
	c.EditPaths = append(c.EditPaths, glob)
	return true
}

// RemoveEditPath removes a glob from the whitelist. Returns true when removed.
func (c *AllowConfig) RemoveEditPath(glob string) bool {
	glob = strings.TrimSpace(glob)
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, g := range c.EditPaths {
		if g == glob {
			c.EditPaths = append(c.EditPaths[:i], c.EditPaths[i+1:]...)
			return true
		}
	}
	return false
}

// ClearEditPaths empties the whitelist.
func (c *AllowConfig) ClearEditPaths() {
	c.mu.Lock()
	c.EditPaths = nil
	c.mu.Unlock()
}

// MatchEditPath reports whether path matches any whitelist glob.
func (c *AllowConfig) MatchEditPath(path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.EditPaths) == 0 {
		return false
	}
	clean := normalizeMatchPath(path)
	for _, g := range c.EditPaths {
		if matchGlob(normalizeMatchPath(g), clean) {
			return true
		}
	}
	return false
}

// SaveProject persists the project config. The project autoEdit key is written
// only when it was explicitly set at project scope; inherited global state is
// never copied into .vibe/allow.json as a side effect of editing path rules.
func (c *AllowConfig) SaveProject() error {
	c.mu.RLock()
	autoEdit := c.AutoEdit
	autoEditSet := c.projectAutoEditSet
	editPaths := append([]string(nil), c.EditPaths...)
	c.mu.RUnlock()
	return writeProjectAllowFile(ProjectAllowPath(), autoEdit, autoEditSet, editPaths)
}

// SaveGlobalAutoEdit persists only autoEdit to the global file, preserving any
// other keys that may exist there.
func (c *AllowConfig) SaveGlobalAutoEdit() error {
	c.mu.RLock()
	v := c.AutoEdit
	c.mu.RUnlock()
	return writeGlobalAllowAutoEdit(v)
}

// SaveGlobalAutoEditValue persists the provided global autoEdit value without
// changing project-scoped effective state in memory.
func (c *AllowConfig) SaveGlobalAutoEditValue(v bool) error {
	return writeGlobalAllowAutoEdit(v)
}

func writeGlobalAllowAutoEdit(v bool) error {
	existing := map[string]json.RawMessage{}
	if data, err := os.ReadFile(GlobalAllowPath()); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	autoEditJSON, err := json.Marshal(v)
	if err != nil {
		return err
	}
	existing["autoEdit"] = autoEditJSON
	delete(existing, "editPaths") // editPaths are project-only.
	return writeJSONFile(GlobalAllowPath(), existing)
}

func readAllowAutoEdit(data []byte) (bool, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, false
	}
	v, ok := raw["autoEdit"]
	if !ok {
		return false, false
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		return false, false
	}
	return b, true
}

func writeProjectAllowFile(path string, autoEdit bool, autoEditSet bool, editPaths []string) error {
	out := map[string]any{}
	if autoEditSet {
		out["autoEdit"] = autoEdit
	}
	if len(editPaths) > 0 {
		out["editPaths"] = editPaths
	}
	return writeJSONFile(path, out)
}

func writeJSONFile(path string, v any) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// normalizeMatchPath cleans a path for matching: forward slashes, trimmed
// leading "./".
func normalizeMatchPath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	return p
}

// matchGlob matches a glob pattern against a path. It supports:
//   - "*"  matches any run of characters except "/"
//   - "**" matches any run of characters including "/"
//   - "?"  matches a single non-"/" character
func matchGlob(pattern, name string) bool {
	return globMatch(pattern, name)
}

// globMatch implements a recursive glob matcher with ** support.
func globMatch(pattern, name string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Check for "**".
			if len(pattern) >= 2 && pattern[1] == '*' {
				// Collapse consecutive stars.
				rest := pattern[2:]
				for len(rest) > 0 && rest[0] == '*' {
					rest = rest[1:]
				}
				// "**/" should also match zero directories.
				if strings.HasPrefix(rest, "/") {
					if globMatch(rest[1:], name) {
						return true
					}
				}
				if rest == "" {
					return true
				}
				// Try to match rest at every position (including across "/").
				for i := 0; i <= len(name); i++ {
					if globMatch(rest, name[i:]) {
						return true
					}
				}
				return false
			}
			// Single "*": match any run not containing "/".
			rest := pattern[1:]
			for i := 0; i <= len(name); i++ {
				if i > 0 && name[i-1] == '/' {
					break
				}
				if globMatch(rest, name[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(name) == 0 || name[0] == '/' {
				return false
			}
			pattern = pattern[1:]
			name = name[1:]
		default:
			if len(name) == 0 || pattern[0] != name[0] {
				return false
			}
			pattern = pattern[1:]
			name = name[1:]
		}
	}
	return len(name) == 0
}
