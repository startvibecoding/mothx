package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGlobalSettingsSparseDoesNotExpandDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VIBECODING_DIR", tmpDir)
	data := []byte(`{
		"providers": {
			"xiaomi": {"api": "openai-chat", "baseUrl": "https://x.test", "models": [{"id": "m"}]}
		},
		"defaultProvider": "xiaomi",
		"defaultModel": "m"
	}`)
	if err := os.WriteFile(filepath.Join(tmpDir, "settings.json"), data, 0600); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	s, err := LoadGlobalSettingsSparse()
	if err != nil {
		t.Fatalf("load sparse: %v", err)
	}
	if len(s.Providers) != 1 || s.Providers["xiaomi"] == nil {
		t.Fatalf("providers = %#v, want only xiaomi", s.Providers)
	}
	if s.Providers["openai"] != nil || s.Providers["deepseek-openai"] != nil {
		t.Fatalf("defaults expanded into sparse settings: %#v", s.Providers)
	}
}

func TestSaveGlobalSettingsPatchPreservesSparseFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VIBECODING_DIR", tmpDir)
	data := []byte(`{
		"providers": {
			"xiaomi": {"api": "openai-chat", "baseUrl": "https://x.test", "models": [{"id": "m"}]}
		},
		"defaultProvider": "xiaomi"
	}`)
	path := filepath.Join(tmpDir, "settings.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	if err := SaveGlobalSettingsPatch(map[string]any{"defaultMode": "yolo"}); err != nil {
		t.Fatalf("save patch: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	text := string(out)
	for _, want := range []string{`"defaultMode": "yolo"`, `"defaultProvider": "xiaomi"`, `"xiaomi"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("settings missing %s:\n%s", want, text)
		}
	}
	for _, unexpected := range []string{`"deepseek-openai"`, `"statusLine"`, `"contextFiles"`, `"compaction"`, `"sandbox"`} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("settings patch expanded defaults with %s:\n%s", unexpected, text)
		}
	}
}

func TestLoadSettingsWithMetaCreatesSparseDefaultFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0700); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	configDir := filepath.Join(tmpDir, "config")
	t.Setenv("VIBECODING_DIR", configDir)

	s, meta, err := LoadSettingsWithMeta()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if !meta.CreatedGlobalConfig {
		t.Fatal("expected global config to be created")
	}
	if s.Providers["deepseek-openai"] == nil {
		t.Fatal("runtime defaults should still include built-in providers")
	}

	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	text := string(data)
	for _, unexpected := range []string{`"providers"`, `"anthropic"`, `"google-gemini"`, `"xiaomi"`} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("created settings expanded defaults with %s:\n%s", unexpected, text)
		}
	}
	for _, want := range []string{
		`"defaultProvider": "deepseek-openai"`,
		`"defaultModel": "deepseek-v4-flash"`,
		`"defaultMode": "agent"`,
		`"statusLine"`,
		`"webSearch"`,
		`"contextFiles"`,
		`"compaction"`,
		`"sandbox"`,
		`"sessionDir"`,
		`"theme": "dark"`,
		`"retry"`,
		`"maxRetries": 5`,
		`"baseDelayMs": 3000`,
		`"approval"`,
		`"confirmBeforeWrite": true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("created settings missing %s:\n%s", want, text)
		}
	}
}
