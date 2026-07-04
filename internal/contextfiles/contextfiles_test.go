package contextfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContextFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	globalDir := filepath.Join(tmpDir, "global")

	os.MkdirAll(projectDir, 0755)
	os.MkdirAll(globalDir, 0755)

	// Create test files
	os.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte("# Project Agent"), 0644)
	os.WriteFile(filepath.Join(globalDir, "AGENTS.md"), []byte("# Global Config"), 0644)

	// Load context files
	result := LoadContextFiles(projectDir, globalDir, nil)

	// Check results
	if len(result.ProjectFiles) != 1 {
		t.Errorf("expected 1 project file, got %d", len(result.ProjectFiles))
	}

	if len(result.GlobalFiles) != 1 {
		t.Errorf("expected 1 global file, got %d", len(result.GlobalFiles))
	}

	// Check file names
	if result.ProjectFiles[0].Name != "AGENTS.md" {
		t.Errorf("expected 'AGENTS.md', got '%s'", result.ProjectFiles[0].Name)
	}
}

func TestBuildContextString(t *testing.T) {
	result := &LoadResult{
		ProjectFiles: []FileContent{
			{Name: "AGENTS.md", Path: "/test/AGENTS.md", Content: "# Test Content"},
		},
	}

	context := BuildContextString(result)

	if context == "" {
		t.Fatal("expected non-empty context string")
	}

	if !contains(context, "AGENTS.md") {
		t.Error("expected context to contain 'AGENTS.md'")
	}

	if !contains(context, "# Test Content") {
		t.Error("expected context to contain file content")
	}
}

func TestBuildContextStringEmpty(t *testing.T) {
	result := &LoadResult{}
	context := BuildContextString(result)

	if context != "" {
		t.Errorf("expected empty context string, got '%s'", context)
	}
}

func TestExtraFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create extra file
	os.WriteFile(filepath.Join(tmpDir, "CUSTOM.md"), []byte("# Custom"), 0644)

	extraFiles := []string{"CUSTOM.md"}
	result := LoadContextFiles(tmpDir, "", extraFiles)

	if len(result.ProjectFiles) != 1 {
		t.Errorf("expected 1 project file, got %d", len(result.ProjectFiles))
	}

	if result.ProjectFiles[0].Name != "CUSTOM.md" {
		t.Errorf("expected 'CUSTOM.md', got '%s'", result.ProjectFiles[0].Name)
	}
}

func TestExtraFilesCannotEscapeBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectDir, 0755)

	os.WriteFile(filepath.Join(tmpDir, "SECRET.md"), []byte("# Secret"), 0644)
	os.WriteFile(filepath.Join(projectDir, "SAFE.md"), []byte("# Safe"), 0644)

	result := LoadContextFiles(projectDir, "", []string{"../SECRET.md", filepath.Join(tmpDir, "SECRET.md"), "SAFE.md"})

	if len(result.ProjectFiles) != 1 {
		t.Fatalf("expected 1 project file, got %d", len(result.ProjectFiles))
	}
	if result.ProjectFiles[0].Name != "SAFE.md" {
		t.Fatalf("loaded %q, want SAFE.md", result.ProjectFiles[0].Name)
	}
}

func TestLoadRuleFileMissingDoesNotCreateFile(t *testing.T) {
	tmpDir := t.TempDir()

	if got := LoadRuleFile(tmpDir); got != "" {
		t.Fatalf("LoadRuleFile() = %q, want empty string", got)
	}

	rulePath := filepath.Join(tmpDir, RuleFile)
	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Fatalf("LoadRuleFile created %s, stat err = %v", rulePath, err)
	}
}

func TestLoadRuleFileReadsProjectRule(t *testing.T) {
	tmpDir := t.TempDir()
	rulePath := filepath.Join(tmpDir, RuleFile)
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte("follow local rules\n"), 0644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	if got := LoadRuleFile(tmpDir); got != "follow local rules\n" {
		t.Fatalf("LoadRuleFile() = %q", got)
	}
}

func TestEnsureRuleFileCreatesDefault(t *testing.T) {
	tmpDir := t.TempDir()

	path, content, written, err := EnsureRuleFile(tmpDir, false)
	if err != nil {
		t.Fatalf("EnsureRuleFile() error = %v", err)
	}
	if !written {
		t.Fatal("EnsureRuleFile() did not report written")
	}
	if path != filepath.Join(tmpDir, RuleFile) {
		t.Fatalf("path = %q", path)
	}
	if content != DefaultRuleContent {
		t.Fatal("content != DefaultRuleContent")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created rule: %v", err)
	}
	if string(data) != DefaultRuleContent {
		t.Fatal("created rule content mismatch")
	}
}

func TestEnsureRuleFilePreservesExistingUnlessForced(t *testing.T) {
	tmpDir := t.TempDir()
	rulePath := filepath.Join(tmpDir, RuleFile)
	if err := os.MkdirAll(filepath.Dir(rulePath), 0755); err != nil {
		t.Fatalf("mkdir rule dir: %v", err)
	}
	if err := os.WriteFile(rulePath, []byte("custom rule"), 0644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	_, content, written, err := EnsureRuleFile(tmpDir, false)
	if err != nil {
		t.Fatalf("EnsureRuleFile() error = %v", err)
	}
	if written {
		t.Fatal("EnsureRuleFile() overwrote without force")
	}
	if content != "custom rule" {
		t.Fatalf("content = %q", content)
	}

	_, content, written, err = EnsureRuleFile(tmpDir, true)
	if err != nil {
		t.Fatalf("EnsureRuleFile(force) error = %v", err)
	}
	if !written {
		t.Fatal("EnsureRuleFile(force) did not report written")
	}
	if content != DefaultRuleContent {
		t.Fatal("forced content != DefaultRuleContent")
	}
}

func TestParentFiles(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(parentDir, "child")

	os.MkdirAll(childDir, 0755)

	// Create file in parent directory
	os.WriteFile(filepath.Join(parentDir, "AGENTS.md"), []byte("# Parent Config"), 0644)

	// Load from child directory
	result := LoadContextFiles(childDir, "", nil)

	if len(result.ParentFiles) != 1 {
		t.Errorf("expected 1 parent file, got %d", len(result.ParentFiles))
	}

	if result.ParentFiles[0].Name != "AGENTS.md" {
		t.Errorf("expected 'AGENTS.md', got '%s'", result.ParentFiles[0].Name)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
