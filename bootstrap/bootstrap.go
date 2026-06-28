// Package bootstrap wires the internal agent and provider implementations into
// the public agent package. External modules cannot import vibecoding's
// internal packages directly, so they must blank-import this package once to
// enable agent.NewBuilder().Build() and Builder.WithProviderByName(...):
//
//	import _ "github.com/startvibecoding/vibecoding/bootstrap"
//
// Importing this package has no other side effects beyond registering the
// builder and provider resolution hooks at init time.
package bootstrap

import (
	// Registers the internal agent builder (agent.SetBuilderFunc) and, transitively,
	// the provider registry used by Builder.WithProviderByName.
	_ "github.com/startvibecoding/vibecoding/internal/agent"
	_ "github.com/startvibecoding/vibecoding/internal/provider"

	// Register the concrete provider factories in the global provider registry
	// so Builder.WithProviderByName can resolve openai/anthropic/google
	// providers (each subpackage self-registers via its init()).
	_ "github.com/startvibecoding/vibecoding/internal/provider/anthropic"
	_ "github.com/startvibecoding/vibecoding/internal/provider/google"
	_ "github.com/startvibecoding/vibecoding/internal/provider/openai"
)
