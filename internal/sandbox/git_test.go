package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedGitPathsResolvesWorktreeGitFile(t *testing.T) {
	project := t.TempDir()
	realGit := filepath.Join(t.TempDir(), "worktree-git")
	if err := os.Mkdir(realGit, 0o755); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(project, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: "+realGit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := protectedGitPaths(project)
	if len(paths) != 2 || paths[0] != gitFile || paths[1] != realGit {
		t.Fatalf("protected paths = %#v", paths)
	}
}

func TestProtectedGitPathsKeepsMissingGitDirectory(t *testing.T) {
	project := t.TempDir()
	paths := protectedGitPaths(project)
	if len(paths) != 1 || paths[0] != filepath.Join(project, ".git") {
		t.Fatalf("protected paths = %#v", paths)
	}
}

func TestGitAccessRequired(t *testing.T) {
	if !GitAccessRequired("git status", "") {
		t.Fatal("git command should require Git access approval")
	}
	if !GitAccessRequired("cat .git/config", t.TempDir()) {
		t.Fatal(".git path should require Git access approval")
	}
	if GitAccessRequired("go test ./...", t.TempDir()) {
		t.Fatal("ordinary command should not require Git access approval")
	}
}

func TestGitAccessContext(t *testing.T) {
	ctx := ContextWithGitAccess(context.Background(), true)
	if !GitAccessFromContext(ctx) {
		t.Fatal("expected one-shot Git access context")
	}
	if GitAccessFromContext(context.Background()) {
		t.Fatal("unexpected Git access on empty context")
	}
}

func TestGitAccessCommandRemovesOnlyGitDeny(t *testing.T) {
	project := t.TempDir()
	gitDir := filepath.Join(project, ".git")
	other := filepath.Join(project, "secret")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewBwrapSandboxWithOptions(project, LevelStandard, Options{
		DeniedPaths: []string{other},
		ProtectGit:  true,
	})
	cmd := s.WrapCommandWithGitAccess(context.Background(), "/bin/sh", "true", ExecOpts{WorkDir: project})
	if cmd == nil {
		t.Fatal("expected command")
	}
	args := cmd.Args
	if indexArgs(args, "--tmpfs", gitDir) >= 0 {
		t.Fatalf("git deny should be removed: %#v", args)
	}
	if indexArgs(args, "--tmpfs", other) < 0 {
		t.Fatalf("non-Git deny should remain: %#v", args)
	}
}
