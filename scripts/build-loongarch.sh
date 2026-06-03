#!/bin/bash
set -e

# Build and package the Linux LoongArch64 (GOARCH=loong64) release.
# Usage: ./scripts/build-loongarch.sh [version]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"

VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "0.0.1")}"

cd "${PROJECT_ROOT}"

echo "Building Linux LoongArch64 binary..."
make build-linux-loong64 VERSION="${VERSION}"

echo ""
echo "Packaging Linux LoongArch64 tarball..."
"${SCRIPT_DIR}/build-tarball.sh" linux loong64 "${VERSION}"

echo ""
echo "Packaging Linux LoongArch64 Debian package..."
"${SCRIPT_DIR}/build-deb.sh" loong64 "${VERSION}"

echo ""
echo "LoongArch64 packages created under dist/"
