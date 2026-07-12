package util

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzTruncateString(f *testing.F) {
	for _, seed := range []struct {
		input string
		limit int
	}{
		{"", 0},
		{"hello", 3},
		{"你好世界", 5},
		{"🙂🙂", 5},
		{"a", -1},
	} {
		f.Add(seed.input, seed.limit)
	}

	f.Fuzz(func(t *testing.T, input string, limit int) {
		got := TruncateString(input, limit)
		if limit <= 0 && got != "" {
			t.Fatalf("TruncateString(%q, %d) = %q, want empty string", input, limit, got)
		}
		if limit > 0 && len(got) > limit {
			t.Fatalf("TruncateString(%q, %d) = %q exceeds byte limit", input, limit, got)
		}
		if !strings.HasPrefix(input, got) {
			t.Fatalf("TruncateString(%q, %d) = %q is not a prefix", input, limit, got)
		}
		if utf8.ValidString(input) && !utf8.ValidString(got) {
			t.Fatalf("TruncateString(%q, %d) returned invalid UTF-8 %q", input, limit, got)
		}
	})
}
