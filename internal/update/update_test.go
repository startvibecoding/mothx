package update

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.1.50", "1.1.50", 0},
		{"v1.1.50", "1.1.50", 0},
		{"1.1.51", "1.1.50", 1},
		{"1.1.50", "1.1.51", -1},
		{"1.2.0", "1.1.99", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.1.50-pre", "1.1.50", -1},
		{"1.1.50", "1.1.50-pre", 1},
		{"1.1.51-pre", "1.1.50", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsCheckable(t *testing.T) {
	cases := map[string]bool{
		"dev":     false,
		"":        false,
		"v1.1.50": true,
		"1.1.50":  true,
		"unknown": false,
	}
	for v, want := range cases {
		if got := isCheckable(v); got != want {
			t.Errorf("isCheckable(%q)=%v want %v", v, got, want)
		}
	}
}

func TestNormalize(t *testing.T) {
	if got := normalize(" v1.2.3 "); got != "1.2.3" {
		t.Errorf("normalize=%q", got)
	}
}

func TestCheckInBackgroundRespectsDisableFlag(t *testing.T) {
	t.Setenv("VIBECODING_DIR", t.TempDir())
	t.Setenv("VIBECODING_NO_UPDATE_CHECK", "1")

	oldFetch := fetchLatestVersion
	defer func() { fetchLatestVersion = oldFetch }()

	called := false
	fetchLatestVersion = func(context.Context) (string, error) {
		called = true
		return "v1.2.4", nil
	}

	CheckInBackground("v1.2.3", nil)
	if called {
		t.Fatal("expected disabled check not to fetch")
	}
}

func TestCheckInBackgroundRecordsFailureCooldown(t *testing.T) {
	t.Setenv("VIBECODING_DIR", t.TempDir())
	t.Setenv("VIBECODING_NO_UPDATE_CHECK", "")

	oldFetch := fetchLatestVersion
	oldNow := now
	defer func() {
		fetchLatestVersion = oldFetch
		now = oldNow
	}()

	nowValue := time.Unix(1000, 0)
	now = func() time.Time { return nowValue }
	fetchLatestVersion = func(context.Context) (string, error) {
		return "", errors.New("boom")
	}

	refreshCache("v1.2.3", nil)

	c := readCache()
	if c.CheckedAt.IsZero() {
		t.Fatal("expected failed check to record cooldown timestamp")
	}
	if !c.CheckedAt.Equal(nowValue) {
		t.Fatalf("CheckedAt = %v, want %v", c.CheckedAt, nowValue)
	}
}

func TestRefreshCacheNotifiesForNewerSemver(t *testing.T) {
	t.Setenv("VIBECODING_DIR", t.TempDir())
	t.Setenv("VIBECODING_NO_UPDATE_CHECK", "")

	oldFetch := fetchLatestVersion
	oldNow := now
	defer func() {
		fetchLatestVersion = oldFetch
		now = oldNow
	}()

	nowValue := time.Unix(2000, 0)
	now = func() time.Time { return nowValue }
	fetchLatestVersion = func(context.Context) (string, error) {
		return "v1.10.0", nil
	}

	var notice string
	refreshCache("v1.2.3", func(msg string) {
		notice = msg
	})
	if notice == "" {
		t.Fatal("expected semver comparison to notify for newer version")
	}
	c := readCache()
	if !c.CheckedAt.Equal(nowValue) {
		t.Fatalf("CheckedAt = %v, want %v", c.CheckedAt, nowValue)
	}
}

func TestRefreshCacheSkipsNotifyForCurrentOrOlderVersion(t *testing.T) {
	t.Setenv("VIBECODING_DIR", t.TempDir())
	t.Setenv("VIBECODING_NO_UPDATE_CHECK", "")

	oldFetch := fetchLatestVersion
	defer func() { fetchLatestVersion = oldFetch }()

	fetchLatestVersion = func(context.Context) (string, error) {
		return "v1.2.3", nil
	}

	called := false
	refreshCache("v1.2.3", func(string) {
		called = true
	})
	if called {
		t.Fatal("did not expect notification for current version")
	}
}
