// Package update provides non-blocking version update detection based on the
// npm registry. It never blocks the user: the foreground only reads a local
// cache, while network checks run in the background and refresh that cache for
// the next run.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/startvibecoding/vibecoding/internal/config"
	"github.com/startvibecoding/vibecoding/internal/ua"
)

// PackageName is the npm package used for update detection.
const PackageName = "vibecoding-installer"

// checkInterval is the minimum time between background network checks.
const checkInterval = 24 * time.Hour

// registryURL returns the npm registry endpoint for the latest dist-tag.
func registryURL() string {
	if u := os.Getenv("VIBECODING_NPM_REGISTRY"); u != "" {
		return strings.TrimRight(u, "/") + "/" + PackageName + "/latest"
	}
	return "https://registry.npmjs.org/" + PackageName + "/latest"
}

// cacheEntry is the on-disk cache for update checks.
type cacheEntry struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

func cachePath() string {
	return filepath.Join(config.ConfigDir(), "update-check.json")
}

func readCache() cacheEntry {
	var c cacheEntry
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

func writeCache(c cacheEntry) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(cachePath())
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(cachePath(), data, 0o644)
}

// CachedNotice returns a reminder string if the cached latest version is newer
// than current, otherwise an empty string. It performs no network I/O.
func CachedNotice(current string) string {
	if !isCheckable(current) {
		return ""
	}
	c := readCache()
	if c.LatestVersion == "" {
		return ""
	}
	if compareVersions(c.LatestVersion, current) <= 0 {
		return ""
	}
	return fmt.Sprintf(
		"✨ Update available: %s → %s\n   Run: npm install -g %s@latest",
		normalize(current), normalize(c.LatestVersion), PackageName,
	)
}

// CheckInBackground refreshes the cache in a background goroutine if it is
// stale. It returns immediately and never blocks the caller. Disable with the
// VIBECODING_NO_UPDATE_CHECK environment variable.
func CheckInBackground(current string) {
	if !isCheckable(current) {
		return
	}
	if os.Getenv("VIBECODING_NO_UPDATE_CHECK") != "" {
		return
	}
	c := readCache()
	if !c.CheckedAt.IsZero() && time.Since(c.CheckedAt) < checkInterval {
		return
	}
	go func() {
		latest, err := fetchLatest(context.Background())
		if err != nil {
			return
		}
		writeCache(cacheEntry{LatestVersion: latest, CheckedAt: time.Now()})
	}()
}

// fetchLatest queries the npm registry for the latest published version.
func fetchLatest(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", ua.UserAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry: status %d", resp.StatusCode)
	}

	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Version == "" {
		return "", fmt.Errorf("npm registry: empty version")
	}
	return payload.Version, nil
}

// isCheckable reports whether current is a real release version worth checking.
func isCheckable(current string) bool {
	v := normalize(current)
	if v == "" || v == "dev" {
		return false
	}
	// Must start with a digit to be a real semver release.
	return v[0] >= '0' && v[0] <= '9'
}

// normalize strips a leading "v" and surrounding whitespace.
func normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// compareVersions compares two semver-ish strings. It returns -1 if a < b,
// 0 if equal, and 1 if a > b. Pre-release suffixes (e.g. "-pre") make a version
// lower than the same version without a suffix. Unparseable parts compare as 0.
func compareVersions(a, b string) int {
	ac, ap := splitPre(normalize(a))
	bc, bp := splitPre(normalize(b))

	an := parseParts(ac)
	bn := parseParts(bc)
	for i := 0; i < 3; i++ {
		if an[i] != bn[i] {
			if an[i] < bn[i] {
				return -1
			}
			return 1
		}
	}
	// Core versions equal: a release outranks a pre-release.
	switch {
	case ap == "" && bp == "":
		return 0
	case ap == "" && bp != "":
		return 1
	case ap != "" && bp == "":
		return -1
	default:
		return strings.Compare(ap, bp)
	}
}

func splitPre(v string) (core, pre string) {
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func parseParts(core string) [3]int {
	var out [3]int
	parts := strings.Split(core, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return out
		}
		out[i] = n
	}
	return out
}
