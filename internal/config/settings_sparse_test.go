package config

import (
	"os"
	"path/filepath"
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
