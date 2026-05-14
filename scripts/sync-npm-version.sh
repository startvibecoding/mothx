#!/bin/bash

# Sync version from git tag to npm/package.json

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PACKAGE_JSON="$PROJECT_ROOT/npm/package.json"

# Get version from argument or git
if [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION=$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
  if [ -z "$VERSION" ]; then
    echo "Error: Could not determine version"
    exit 1
  fi
fi

echo "Syncing npm version to: $VERSION"

# Update package.json version
if command -v jq &> /dev/null; then
  # Use jq if available
  jq --arg version "$VERSION" '.version = $version' "$PACKAGE_JSON" > "$PACKAGE_JSON.tmp"
  mv "$PACKAGE_JSON.tmp" "$PACKAGE_JSON"
else
  # Fallback to sed
  sed -i.bak "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" "$PACKAGE_JSON"
  rm -f "$PACKAGE_JSON.bak"
fi

echo "Updated $PACKAGE_JSON"
