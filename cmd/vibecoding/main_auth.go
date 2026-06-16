package main

import (
	"strings"

	"github.com/startvibecoding/vibecoding/internal/config"
)

func settingsHasResolvedDefaultToken(settings *config.Settings) bool {
	if settings == nil || settings.DefaultProvider == "" {
		return false
	}
	key := strings.TrimSpace(settings.ResolveKey(settings.DefaultProvider))
	if key == "" {
		return false
	}
	return !(strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}"))
}
