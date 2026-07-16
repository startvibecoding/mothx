//go:build darwin

package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestMacSandboxProfileUsesOptions(t *testing.T) {
	project := t.TempDir()
	denied := filepath.Join(project, "secret")
	s := newMacSandboxWithOptions(project, LevelStandard, Options{
		AllowNetwork: true,
		AllowedRead:  []string{"/opt/tool"},
		AllowedWrite: []string{"/tmp/work"},
		DeniedPaths:  []string{denied},
	})
	profile := s.buildProfile(ExecOpts{WorkDir: project})
	for _, want := range []string{"/opt/tool", "/tmp/work", denied} {
		if !strings.Contains(profile, want) {
			t.Fatalf("profile missing %q: %s", want, profile)
		}
	}
	if strings.Contains(profile, "(deny network*)") {
		t.Fatal("network deny present despite AllowNetwork")
	}
}
