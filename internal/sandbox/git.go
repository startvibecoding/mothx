package sandbox

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
)

type gitAccessContextKey struct{}

func ContextWithGitAccess(ctx context.Context, allowed bool) context.Context {
	return context.WithValue(ctx, gitAccessContextKey{}, allowed)
}

func GitAccessFromContext(ctx context.Context) bool {
	allowed, _ := ctx.Value(gitAccessContextKey{}).(bool)
	return allowed
}

var gitPathPattern = regexp.MustCompile(`(?i)(^|[^[:alnum:]_])(git([[:space:]]|$)|\.git([/\\\\]|$)|--git-dir([=:]|[[:space:]])|GIT_DIR=)`)

// GitAccessRequired conservatively identifies commands that may access Git
// metadata. It is only an approval hint; the sandbox deny rule remains the
// actual enforcement boundary.
func GitAccessRequired(command, workDir string) bool {
	if !gitPathPattern.MatchString(command) {
		return false
	}
	if workDir == "" {
		return true
	}
	// A command mentioning a .git path under the working tree is protected.
	gitPath := filepath.Join(filepath.Clean(workDir), ".git")
	return strings.Contains(strings.ToLower(command), strings.ToLower(gitPath)) || strings.Contains(strings.ToLower(command), ".git")
}

func IsGitDeniedPath(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return strings.HasSuffix(clean, "/.git") || strings.Contains(clean, "/.git/")
}
