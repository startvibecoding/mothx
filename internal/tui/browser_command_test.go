package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	browserfeature "github.com/startvibecoding/vibecoding/internal/browser"
	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/provider"
	"github.com/startvibecoding/vibecoding/internal/tools"
)

func TestBrowserCommandOnCreatesSkillAndRegistersTool(t *testing.T) {
	tmpDir := t.TempDir()
	registry := tools.NewRegistry(tmpDir, nil)
	app := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, registry, "", "", nil, "agent", false, false, nil, nil, nil)
	app.cwd = tmpDir

	app.handleCommand("/browser on")

	if !browserfeature.IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be registered")
	}
	skillPath := filepath.Join(tmpDir, ".skills", browserfeature.SkillName, "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("expected browser skill to be created: %v", err)
	}
	if !app.browserEnabled {
		t.Fatal("browserEnabled is false")
	}
	if app.browserSkillInBase {
		t.Fatal("dynamic /browser on should not mark browser skill as base context")
	}
	if !strings.Contains(app.extraContext, "Active Skill: "+browserfeature.SkillName) {
		t.Fatalf("extraContext missing active browser skill:\n%s", app.extraContext)
	}
}

func TestBrowserCommandPreservesExistingSkillAndOffRemovesTool(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, ".skills", browserfeature.SkillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte("# Custom Browser\n\nuser edited"), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	registry := tools.NewRegistry(tmpDir, nil)
	app := NewApp(nil, &provider.Model{Name: "test"}, config.DefaultSettings(), nil, registry, "", "", nil, "agent", false, false, nil, nil, nil)
	app.cwd = tmpDir

	app.handleCommand("/browser on")

	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read custom skill: %v", err)
	}
	if string(data) != "# Custom Browser\n\nuser edited" {
		t.Fatalf("custom skill was overwritten: %q", string(data))
	}
	if !strings.Contains(app.extraContext, "# Custom Browser") {
		t.Fatalf("extraContext missing custom browser skill:\n%s", app.extraContext)
	}

	app.handleCommand("/browser off")

	if browserfeature.IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be removed")
	}
	if app.browserEnabled {
		t.Fatal("browserEnabled is true after off")
	}
	if strings.Contains(app.extraContext, "# Custom Browser") {
		t.Fatalf("extraContext still contains browser skill after off:\n%s", app.extraContext)
	}
}
