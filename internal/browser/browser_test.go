package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/startvibecoding/vibecoding/internal/tools"
)

func TestEnsureProjectSkillCreatesBrowserSkill(t *testing.T) {
	root := t.TempDir()

	path, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if !created {
		t.Fatal("expected skill to be created")
	}
	wantPath := filepath.Join(root, ".skills", SkillName, "SKILL.md")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"# Vibe Browser",
		"`browser` tool",
		"`snapshot`",
		"`screenshot`",
		"Never claim a UI state changed until you verify it",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("skill content missing %q", want)
		}
	}
}

func TestEnsureProjectSkillDoesNotOverwriteExistingSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".skills", SkillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte("custom browser skill"), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	gotPath, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if created {
		t.Fatal("did not expect existing skill to be recreated")
	}
	if gotPath != path {
		t.Fatalf("path = %q, want %q", gotPath, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read custom skill: %v", err)
	}
	if string(data) != "custom browser skill" {
		t.Fatalf("skill was overwritten: %q", string(data))
	}
}

func TestEnsureProjectSkillRespectsLowercaseSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, ".skills", SkillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(skillDir, "skill.md")
	if err := os.WriteFile(path, []byte("lowercase browser skill"), 0644); err != nil {
		t.Fatalf("write lowercase skill: %v", err)
	}

	gotPath, created, err := EnsureProjectSkill(root)
	if err != nil {
		t.Fatalf("EnsureProjectSkill() error = %v", err)
	}
	if created {
		t.Fatal("did not expect lowercase skill to be recreated")
	}
	if gotPath != path {
		t.Fatalf("path = %q, want %q", gotPath, path)
	}
}

func TestRegisterAndRemoveBrowserTool(t *testing.T) {
	registry := tools.NewRegistry(t.TempDir(), nil)

	RegisterTool(registry)
	if !IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be registered")
	}

	RemoveTool(registry)
	if IsToolRegistered(registry) {
		t.Fatal("expected browser tool to be removed")
	}
}

func TestClientOptionsDefaultLaunchViewport(t *testing.T) {
	opts := clientOptions(map[string]any{})
	if opts.Launch == nil {
		t.Fatal("Launch options are nil")
	}
	if opts.Launch.ViewportWidth != defaultViewportWidth {
		t.Fatalf("ViewportWidth = %d, want %d", opts.Launch.ViewportWidth, defaultViewportWidth)
	}
	if opts.Launch.ViewportHeight != defaultViewportHeight {
		t.Fatalf("ViewportHeight = %d, want %d", opts.Launch.ViewportHeight, defaultViewportHeight)
	}
	if !opts.Launch.Headless {
		t.Fatal("default launch should remain headless")
	}
}

func TestClientOptionsAllowsViewportAndHeadlessOverride(t *testing.T) {
	opts := clientOptions(map[string]any{
		"viewportWidth":  float64(1366),
		"viewportHeight": float64(768),
		"headless":       false,
	})
	if opts.Launch == nil {
		t.Fatal("Launch options are nil")
	}
	if opts.Launch.ViewportWidth != 1366 {
		t.Fatalf("ViewportWidth = %d, want 1366", opts.Launch.ViewportWidth)
	}
	if opts.Launch.ViewportHeight != 768 {
		t.Fatalf("ViewportHeight = %d, want 768", opts.Launch.ViewportHeight)
	}
	if opts.Launch.Headless {
		t.Fatal("headless override was not honored")
	}
}
