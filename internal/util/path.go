package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePathWithExistingSymlinks returns an absolute path after resolving all
// existing components. Missing descendants are retained beneath the resolved
// nearest existing ancestor.
func ResolvePathWithExistingSymlinks(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}

	missing := make([]string, 0)
	for current := abs; ; current = filepath.Dir(current) {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			return filepath.Join(append([]string{resolved}, missing...)...), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing ancestor")
		}
		missing = append([]string{filepath.Base(current)}, missing...)
	}
}

// IsWithinPath reports whether candidate is equal to or below parent after
// resolving all existing symlinks in both paths.
func IsWithinPath(parent, candidate string) (bool, error) {
	resolvedParent, err := ResolvePathWithExistingSymlinks(parent)
	if err != nil {
		return false, err
	}
	resolvedCandidate, err := ResolvePathWithExistingSymlinks(candidate)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(resolvedParent, resolvedCandidate)
	if err != nil {
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}
