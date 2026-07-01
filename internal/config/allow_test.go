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

func TestAllowBashCommandOps(t *testing.T) {
	c := &AllowConfig{}
	if !c.AddBashCommand("go test ./internal/tui") {
		t.Fatal("AddBashCommand should return true on new command")
	}
	if c.AddBashCommand("go test ./internal/tui") {
		t.Fatal("AddBashCommand should return false on duplicate")
	}
	if !c.MatchBashCommand("go test ./internal/tui") {
		t.Fatal("expected exact bash command match")
	}
	if c.MatchBashCommand("go test ./internal/config") {
		t.Fatal("unexpected match before prefix is added")
	}
	if !c.AddBashPrefix("go test ") {
		t.Fatal("AddBashPrefix should return true on new prefix")
	}
	if c.AddBashPrefix("go test ") {
		t.Fatal("AddBashPrefix should return false on duplicate")
	}
	if !c.MatchBashCommand("go test ./internal/config") {
		t.Fatal("expected bash command prefix match")
	}
	if c.MatchBashCommand("go env") {
		t.Fatal("unexpected match for non-prefix command")
	}
}

func TestAllowAutoEditFlag(t *testing.T) {
	c := &AllowConfig{AutoEdit: true}
	if !c.GetAutoEdit() {
		t.Fatal("default AutoEdit should be true")
	}
	c.SetAutoEdit(false)
	if c.GetAutoEdit() {
		t.Fatal("AutoEdit should be false after SetAutoEdit(false)")
	}
	c.SetAutoEdit(true)
	if !c.GetAutoEdit() {
		t.Fatal("AutoEdit should be true after SetAutoEdit(true)")
	}
}

func TestLoadAllowDefaultsAutoEditOn(t *testing.T) {
	withTempAllowPaths(t)

	c := LoadAllow()
	if !c.GetAutoEdit() {
		t.Fatal("missing allow.json should default autoEdit to true")
	}
}

func TestAllowGlobalExplicitFalseOverridesDefaultAutoEdit(t *testing.T) {
	withTempAllowPaths(t)
	if err := os.MkdirAll(filepath.Dir(GlobalAllowPath()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GlobalAllowPath(), []byte(`{"autoEdit":false}`), 0600); err != nil {
		t.Fatal(err)
	}

	c := LoadAllow()
	if c.GetAutoEdit() {
		t.Fatal("global autoEdit=false should override default autoEdit=true")
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

func TestAllowProjectBashRulesPersistAndReload(t *testing.T) {
	withTempAllowPaths(t)

	c := &AllowConfig{}
	if !c.AddBashCommand("make test") {
		t.Fatal("expected bash command to be added")
	}
	if !c.AddBashPrefix("go test ") {
		t.Fatal("expected bash prefix to be added")
	}
	if err := c.SaveProject(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ProjectAllowPath())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"bashCommands"`) || !strings.Contains(text, `"make test"`) {
		t.Fatalf("project allow.json missing bashCommands: %s", text)
	}
	if !strings.Contains(text, `"bashPrefixes"`) || !strings.Contains(text, `"go test "`) {
		t.Fatalf("project allow.json missing bashPrefixes: %s", text)
	}

	reloaded := LoadAllow()
	if !reloaded.MatchBashCommand("make test") {
		t.Fatal("reloaded allow should match exact bash command")
	}
	if !reloaded.MatchBashCommand("go test ./internal/tui") {
		t.Fatal("reloaded allow should match bash command prefix")
	}
	if strings.Contains(text, "autoEdit") {
		t.Fatalf("project bash rules should not persist inherited autoEdit: %s", text)
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
