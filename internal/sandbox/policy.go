package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// NormalizeOptions resolves sandbox path rules against projectDir and rejects
// ambiguous allow/deny overlaps before a backend constructs any mounts.
func NormalizeOptions(projectDir string, opts Options) (Options, error) {
	base, err := canonicalSandboxPath(projectDir)
	if err != nil {
		return Options{}, fmt.Errorf("normalize project directory: %w", err)
	}
	if base == "" {
		return Options{}, fmt.Errorf("sandbox project directory is required")
	}

	var normalize func([]string, string) ([]string, error)
	normalize = func(paths []string, field string) ([]string, error) {
		out := make([]string, 0, len(paths))
		seen := make(map[string]struct{}, len(paths))
		for _, path := range paths {
			if !filepath.IsAbs(path) {
				path = filepath.Join(base, path)
			}
			canonical, err := canonicalSandboxPath(path)
			if err != nil {
				return nil, fmt.Errorf("normalize sandbox.%s path %q: %w", field, path, err)
			}
			if _, ok := seen[canonical]; !ok {
				seen[canonical] = struct{}{}
				out = append(out, canonical)
			}
		}
		return out, nil
	}

	if opts.AllowedRead, err = normalize(opts.AllowedRead, "allowedRead"); err != nil {
		return Options{}, err
	}
	if opts.AllowedWrite, err = normalize(opts.AllowedWrite, "allowedWrite"); err != nil {
		return Options{}, err
	}
	if opts.DeniedPaths, err = normalize(opts.DeniedPaths, "deniedPaths"); err != nil {
		return Options{}, err
	}
	if opts.TmpSize, err = normalizeTmpSize(opts.TmpSize); err != nil {
		return Options{}, err
	}
	if runtime.GOOS == "linux" {
		filtered := opts.DeniedPaths[:0]
		for _, deny := range opts.DeniedPaths {
			// Bubblewrap already replaces the real user home with an isolated
			// tmpfs before mounting the project. The historical default /home
			// deny therefore conflicts with the normal Linux layout where all
			// development projects live below /home; treat it as redundant.
			if deny == "/home" && pathContains(deny, base) {
				continue
			}
			filtered = append(filtered, deny)
		}
		opts.DeniedPaths = filtered
	}
	for _, deny := range opts.DeniedPaths {
		if pathContains(deny, base) {
			return Options{}, fmt.Errorf("sandbox denied path %q contains project directory %q", deny, base)
		}
		for _, allow := range append(append([]string{}, opts.AllowedRead...), opts.AllowedWrite...) {
			if pathsOverlap(deny, allow) {
				return Options{}, fmt.Errorf("sandbox denied path %q overlaps allowed path %q", deny, allow)
			}
		}
	}
	return opts, nil
}

func normalizeTmpSize(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "100000000", nil
	}
	bytes, err := parseTmpSize(value)
	if err != nil || bytes == 0 {
		if err == nil {
			err = fmt.Errorf("size must be greater than zero")
		}
		return "", fmt.Errorf("invalid sandbox tmpSize %q: %w", value, err)
	}
	return strconv.FormatUint(bytes, 10), nil
}

func parseTmpSize(value string) (uint64, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	multiplier := uint64(1)
	for suffix, factor := range map[string]uint64{"k": 1024, "kb": 1024, "m": 1024 * 1024, "mb": 1024 * 1024, "g": 1024 * 1024 * 1024, "gb": 1024 * 1024 * 1024} {
		if strings.HasSuffix(value, suffix) {
			value = strings.TrimSpace(strings.TrimSuffix(value, suffix))
			multiplier = factor
			break
		}
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if n > ^uint64(0)/multiplier {
		return 0, fmt.Errorf("size overflows bytes")
	}
	return n * multiplier, nil
}
func canonicalSandboxPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	// Canonicalize the longest existing parent, then append the missing suffix.
	var suffix []string
	parent := path
	for {
		if _, err := os.Lstat(parent); err == nil {
			resolved, err := filepath.EvalSymlinks(parent)
			if err != nil {
				return "", err
			}
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return resolved, nil
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("no existing parent")
		}
		suffix = append(suffix, filepath.Base(parent))
		parent = next
	}
}

func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+string(os.PathSeparator)) || strings.HasPrefix(b, a+string(os.PathSeparator))
}
