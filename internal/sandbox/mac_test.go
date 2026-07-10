//go:build darwin

package sandbox

import (
	"context"
	"os"
	"testing"
)

func TestMacSandboxCleansUpCommandProfile(t *testing.T) {
	s := newMacSandbox(t.TempDir(), LevelStandard)
	cmd := s.WrapCommand(context.Background(), "/bin/sh", "true", ExecOpts{WorkDir: t.TempDir()})

	s.profileMu.Lock()
	profilePath := s.profiles[cmd]
	s.profileMu.Unlock()
	if profilePath == "" {
		t.Fatal("expected a tracked sandbox profile")
	}
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("stat profile: %v", err)
	}

	s.CleanupCommand(cmd)
	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		t.Fatalf("profile still exists after cleanup: %v", err)
	}
}
