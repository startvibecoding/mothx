package update

import "testing"

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
		"dev":      false,
		"":         false,
		"v1.1.50":  true,
		"1.1.50":   true,
		"unknown":  false,
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
