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
