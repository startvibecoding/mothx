package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureProjectSkillCreatesWorkflowSkill(t *testing.T) {
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
		"# Workflow Elisp",
		"The first argument of agent must be a string literal.",
		"defun only supports fixed parameter lists.",
		"(workflow \"auth audit\"",
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
	if err := os.WriteFile(path, []byte("custom workflow skill"), 0644); err != nil {
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
	if string(data) != "custom workflow skill" {
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
	if err := os.WriteFile(path, []byte("lowercase workflow skill"), 0644); err != nil {
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
