package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoMigrateProjectDirRenamesLegacyDir(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, LegacyProjectDirName)
	if err := os.MkdirAll(oldDir, 0700); err != nil {
		t.Fatalf("mkdir legacy project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "settings.json"), []byte(`{"defaultMode":"yolo"}`), 0600); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}

	result, ok := AutoMigrateProjectDir(root)
	if !ok {
		t.Fatal("expected project migration result")
	}
	if !result.Migrated || result.Err != nil {
		t.Fatalf("migration result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, ProjectDirName, "settings.json")); err != nil {
		t.Fatalf("new project settings missing: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("legacy project dir still exists or stat failed: %v", err)
	}
}

func TestMigrateLegacyDirSkipsExistingDestination(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "old")
	newDir := filepath.Join(root, "new")
	if err := os.MkdirAll(oldDir, 0700); err != nil {
		t.Fatalf("mkdir old dir: %v", err)
	}
	if err := os.MkdirAll(newDir, 0700); err != nil {
		t.Fatalf("mkdir new dir: %v", err)
	}

	result, ok := migrateLegacyDir("test", oldDir, newDir)
	if !ok {
		t.Fatal("expected migration result")
	}
	if !result.Skipped || result.Migrated || result.Err != nil {
		t.Fatalf("migration result = %#v", result)
	}
	if _, err := os.Stat(oldDir); err != nil {
		t.Fatalf("legacy dir should remain: %v", err)
	}
}

func TestLoadSettingsMigratesLegacyProjectSettings(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Setenv("VIBECODING_DIR", filepath.Join(root, "global"))

	oldDir := filepath.Join(root, LegacyProjectDirName)
	if err := os.MkdirAll(oldDir, 0700); err != nil {
		t.Fatalf("mkdir legacy project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "settings.json"), []byte(`{"defaultMode":"yolo"}`), 0600); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}

	settings, err := LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.DefaultMode != "yolo" {
		t.Fatalf("DefaultMode = %q, want yolo", settings.DefaultMode)
	}
	if _, err := os.Stat(filepath.Join(root, ProjectDirName, "settings.json")); err != nil {
		t.Fatalf("new project settings missing: %v", err)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("legacy project dir still exists or stat failed: %v", err)
	}
}

func TestLoadSettingsWithLegacyDefaultEnvCreatesMothXConfig(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("MOTHX_DIR", "")
	t.Setenv("VIBECODING_DIR", filepath.Join(home, ".vibecoding"))

	if _, err := LoadSettings(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".mothx", "settings.json")); err != nil {
		t.Fatalf("new global settings missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".vibecoding")); !os.IsNotExist(err) {
		t.Fatalf("legacy global dir should not be created, stat err: %v", err)
	}
}
