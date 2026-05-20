#!/bin/bash
# Deprecated helper kept temporarily to avoid silent breakage for local stale workflows.
# fd/rg are now embedded into the vibecoding binary via go:embed.
# Use ./scripts/prepare-vendored.sh to prepare internal/vendored/bin/ for builds.

echo "scripts/extract-vendored-tool.sh has been removed from the build flow." >&2
echo "Use ./scripts/prepare-vendored.sh instead." >&2
exit 1
