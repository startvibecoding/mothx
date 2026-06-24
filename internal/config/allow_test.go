package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "dir/main.go", false},
		{"**/*.go", "dir/main.go", true},
		{"**/*.go", "main.go", true},
		{"internal/**", "internal/agent/agent.go", true},
		{"internal/**", "internal", false},
		{"internal/**", "cmd/main.go", false},
		{"docs/**/*.md", "docs/en/changelog.md", true},
		{"docs/**/*.md", "docs/changelog.md", true},
		{"docs/**/*.md", "docs/en/sub/x.md", true},
		{"src/*", "src/a", true},
		{"src/*", "src/a/b", false},
		{"a?c", "abc", true},
		{"a?c", "a/c", false},
		{"./*.go", "main.go", true},
	}
	for _, c := range cases {
		got := matchGlob(normalizeMatchPath(c.pattern), normalizeMatchPath(c.path))
		if got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestAllowEditPathOps(t *testing.T) {
	c := &AllowConfig{}
	if !c.AddEditPath("internal/**") {
		t.Fatal("AddEditPath should return true on new entry")
	}
	if c.AddEditPath("internal/**") {
		t.Fatal("AddEditPath should return false on duplicate")
	}
	if !c.MatchEditPath("internal/agent/agent.go") {
		t.Fatal("expected match for whitelisted path")
	}
	if c.MatchEditPath("cmd/main.go") {
		t.Fatal("unexpected match for non-whitelisted path")
	}
	if !c.RemoveEditPath("internal/**") {
		t.Fatal("RemoveEditPath should return true when present")
	}
	if c.MatchEditPath("internal/agent/agent.go") {
		t.Fatal("should not match after removal")
	}
}

func TestAllowAutoEditFlag(t *testing.T) {
	c := &AllowConfig{}
	if c.GetAutoEdit() {
		t.Fatal("default AutoEdit should be false")
	}
	c.SetAutoEdit(true)
	if !c.GetAutoEdit() {
		t.Fatal("AutoEdit should be true after SetAutoEdit(true)")
	}
}

func TestAllowProjectEditPathsDoNotPersistInheritedGlobalAutoEdit(t *testing.T) {
	withTempAllowPaths(t)
	if err := os.MkdirAll(filepath.Dir(GlobalAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GlobalAllowPath(), []byte(`{"autoEdit":true}`), 0600); err != nil {
		t.Fatal(err)
	}

	c := LoadAllow()
	if !c.GetAutoEdit() {
		t.Fatal("expected global autoEdit to load")
	}
	if !c.AddEditPath("internal/**") {
		t.Fatal("expected edit path to be added")
	}
	if err := c.SaveProject(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ProjectAllowPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "autoEdit") {
		t.Fatalf("project allow.json should not persist inherited global autoEdit: %s", string(data))
	}

	c.SetAutoEdit(false)
	if err := c.SaveGlobalAutoEdit(); err != nil {
		t.Fatal(err)
	}
	reloaded := LoadAllow()
	if reloaded.GetAutoEdit() {
		t.Fatal("project editPaths-only allow.json should not override global autoEdit=false")
	}
}

func TestAllowProjectExplicitFalseOverridesGlobalAutoEdit(t *testing.T) {
	withTempAllowPaths(t)
	if err := os.MkdirAll(filepath.Dir(GlobalAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ProjectAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GlobalAllowPath(), []byte(`{"autoEdit":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProjectAllowPath(), []byte(`{"autoEdit":false,"editPaths":["internal/**"]}`), 0600); err != nil {
		t.Fatal(err)
	}

	c := LoadAllow()
	if c.GetAutoEdit() {
		t.Fatal("project autoEdit=false should override global autoEdit=true")
	}
	if !c.MatchEditPath("internal/agent/agent.go") {
		t.Fatal("project editPaths should still load")
	}
}

func TestAllowSaveProjectPersistsExplicitFalse(t *testing.T) {
	withTempAllowPaths(t)
	c := &AllowConfig{}
	c.SetProjectAutoEdit(false)
	if err := c.SaveProject(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ProjectAllowPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"autoEdit": false`) {
		t.Fatalf("expected explicit project autoEdit=false to be persisted: %s", string(data))
	}
}

func TestAllowGlobalAutoEditDoesNotOverrideProjectEffectiveState(t *testing.T) {
	withTempAllowPaths(t)
	if err := os.MkdirAll(filepath.Dir(GlobalAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ProjectAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GlobalAllowPath(), []byte(`{"autoEdit":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProjectAllowPath(), []byte(`{"autoEdit":true}`), 0600); err != nil {
		t.Fatal(err)
	}

	c := LoadAllow()
	effective := c.SetGlobalAutoEdit(false)
	if !effective || !c.GetAutoEdit() {
		t.Fatal("project autoEdit=true should remain effective after changing global autoEdit")
	}
	if err := c.SaveGlobalAutoEditValue(false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(GlobalAllowPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"autoEdit": false`) {
		t.Fatalf("expected global autoEdit=false to be saved: %s", string(data))
	}
}

func withTempAllowPaths(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	t.Setenv("VIBECODING_DIR", filepath.Join(tmp, "global"))
}
