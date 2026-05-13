// Package ua provides User-Agent string generation for vibecoding.
package ua

import (
	"fmt"
	"os"
	"runtime"
)

// Version is set at build time via ldflags.
var Version = "dev"

// DefaultUserAgent is the default User-Agent prefix.
const DefaultUserAgent = "Vibecoding Client"

// UserAgent returns the User-Agent string for vibecoding.
// Can be overridden by VIBECODING_USER_AGENT environment variable.
func UserAgent() string {
	// Check for environment variable override
	if ua := os.Getenv("VIBECODING_USER_AGENT"); ua != "" {
		return ua
	}

	return fmt.Sprintf("%s/%s (%s; %s; %s)",
		DefaultUserAgent,
		Version,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
	)
}

// ProviderUserAgent returns the User-Agent string for provider API calls.
func ProviderUserAgent() string {
	return UserAgent()
}
