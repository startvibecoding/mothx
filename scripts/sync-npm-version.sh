#!/bin/bash

# Sync version from git tag to npm package.json (main + all platform packages)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NPM_DIR="$PROJECT_ROOT/npm"
PACKAGE_JSONS=("$NPM_DIR/package.json" "$NPM_DIR/mothx/package.json")

# Get version from argument or git
if [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION=$(git describe --tags --always 2>/dev/null)
  if [ -z "$VERSION" ]; then
    echo "Error: Could not determine version"
    exit 1
  fi
fi
VERSION="${VERSION#v}"
VERSION="${VERSION%-dirty}"
VERSION="${VERSION%%-[0-9]*-g[0-9a-f]*}"

echo "Syncing npm version to: $VERSION"

# Update main package.json files version + optionalDependencies
for PACKAGE_JSON in "${PACKAGE_JSONS[@]}"; do
  if [ -f "$PACKAGE_JSON" ]; then
    node -e "
    const fs = require('fs');
    const pkgPath = '$PACKAGE_JSON';
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    pkg.version = '$VERSION';
    if (pkg.optionalDependencies) {
      for (const key of Object.keys(pkg.optionalDependencies)) {
        pkg.optionalDependencies[key] = '$VERSION';
      }
    }
    fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
    console.log('Updated: $PACKAGE_JSON');
    "
  fi
done

# Keep mothx-installer aligned with the compatibility package's platform
# optionalDependencies. The compatibility package has been the stable source of
# truth for automatic platform package selection during the rename transition.
node -e "
const fs = require('fs');
const legacyPath = '$NPM_DIR/package.json';
const mothxPath = '$NPM_DIR/mothx/package.json';
if (fs.existsSync(legacyPath) && fs.existsSync(mothxPath)) {
  const legacyPkg = JSON.parse(fs.readFileSync(legacyPath, 'utf8'));
  const mothxPkg = JSON.parse(fs.readFileSync(mothxPath, 'utf8'));
  if (legacyPkg.optionalDependencies) {
    mothxPkg.optionalDependencies = { ...legacyPkg.optionalDependencies };
    fs.writeFileSync(mothxPath, JSON.stringify(mothxPkg, null, 2) + '\n');
    console.log('Synced optionalDependencies from vibecoding-installer to mothx-installer');
  }
}
"

# Update all platform package.json files
PACKAGES_DIR="$NPM_DIR/packages"
if [ -d "$PACKAGES_DIR" ]; then
  for pkgDir in "$PACKAGES_DIR"/*/; do
    if [ -f "$pkgDir/package.json" ]; then
      node -e "
      const fs = require('fs');
      const pkgPath = '${pkgDir}package.json';
      const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
      pkg.version = '$VERSION';
      fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
      console.log('Updated: ' + pkgPath);
      "
    fi
  done
fi

echo "Version sync complete: $VERSION"
