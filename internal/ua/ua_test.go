package ua

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestUserAgent(t *testing.T) {
	// Clean up env var after test
	defer os.Unsetenv("VIBECODING_USER_AGENT")

	ua := UserAgent()

	if ua == "" {
		t.Fatal("expected non-empty user agent")
	}

	// Should contain default prefix
	if !strings.Contains(ua, "Vibecoding Client") {
		t.Error("expected user agent to contain 'Vibecoding Client'")
	}

	// Should contain version
	if !strings.Contains(ua, Version) {
		t.Errorf("expected user agent to contain '%s'", Version)
	}

	// Should contain OS
	if !strings.Contains(ua, runtime.GOOS) {
		t.Errorf("expected user agent to contain '%s'", runtime.GOOS)
	}

	// Should contain architecture
	if !strings.Contains(ua, runtime.GOARCH) {
		t.Errorf("expected user agent to contain '%s'", runtime.GOARCH)
	}
}

func TestUserAgentWithEnvOverride(t *testing.T) {
	// Set custom UA
	customUA := "MyCustomAgent/1.0"
	os.Setenv("VIBECODING_USER_AGENT", customUA)
	defer os.Unsetenv("VIBECODING_USER_AGENT")

	ua := UserAgent()

	if ua != customUA {
		t.Errorf("expected '%s', got '%s'", customUA, ua)
	}
}

func TestProviderUserAgent(t *testing.T) {
	ua := ProviderUserAgent()

	if ua == "" {
		t.Fatal("expected non-empty provider user agent")
	}

	// Should be same as UserAgent
	if ua != UserAgent() {
		t.Errorf("expected provider user agent to be same as user agent")
	}
}

func TestVersion(t *testing.T) {
	// Default version should be "dev"
	if Version != "dev" {
		t.Errorf("expected default version to be 'dev', got '%s'", Version)
	}
}

func TestDefaultUserAgent(t *testing.T) {
	if DefaultUserAgent != "Vibecoding Client" {
		t.Errorf("expected default user agent to be 'Vibecoding Client', got '%s'", DefaultUserAgent)
	}
}
