package sandbox

import (
	"os"
	"path/filepath"
	"strings"
)

// protectedGitPaths resolves the project Git metadata directory. A worktree's
// .git is often a file containing `gitdir: <path>` rather than a directory.
func protectedGitPaths(projectDir string) []string {
	gitEntry := filepath.Join(projectDir, ".git")
	paths := []string{gitEntry}
	data, err := os.ReadFile(gitEntry)
	text := strings.TrimSpace(string(data))
	if err != nil || !strings.HasPrefix(strings.ToLower(text), "gitdir:") {
		return paths
	}
	gitDir := strings.TrimSpace(text[len("gitdir:"):])
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectDir, gitDir)
	}
	if canonical, err := canonicalSandboxPath(gitDir); err == nil {
		paths = append(paths, canonical)
	}
	return uniquePaths(paths)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
