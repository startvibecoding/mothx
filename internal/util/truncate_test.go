package util

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateStringKeepsValidUTF8(t *testing.T) {
	got := TruncateString("你好世界", 5)
	if !utf8.ValidString(got) {
		t.Fatalf("invalid UTF-8: %q", got)
	}
	if got != "你" {
		t.Fatalf("got %q, want 你", got)
	}
}

func TestTruncateWithSuffix(t *testing.T) {
	got := TruncateWithSuffix("hello world", 5, "...")
	if got != "hello..." {
		t.Fatalf("got %q, want hello...", got)
	}
	if strings.ContainsRune(TruncateWithSuffix("🙂🙂", 5, "..."), utf8.RuneError) {
		t.Fatal("truncated string contains replacement rune")
	}
}
