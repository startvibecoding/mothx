package mcp

import "testing"

func FuzzSanitizeToolName(f *testing.F) {
	for _, seed := range []string{"read_file", "MCP tool/1", "  ", "\x00name", "hello-world"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, name string) {
		got := SanitizeToolName(name)
		if got == "" {
			t.Fatal("SanitizeToolName returned an empty name")
		}
		for _, r := range got {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				t.Fatalf("SanitizeToolName(%q) = %q contains invalid character %q", name, got, r)
			}
		}
	})
}
