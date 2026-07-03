#!/bin/bash

# Build platform-specific PyPI wheels.
# Each wheel contains exactly one native vibecoding binary, and pip selects the
# correct artifact using standard wheel platform tags.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PYPI_DIR="$PROJECT_ROOT/pypi"
BUILD_DIR="$PROJECT_ROOT/bin"
DIST_DIR="$PYPI_DIR/dist"
WORK_ROOT="$PYPI_DIR/build/wheel-work"
PYTHON="${PYTHON:-python3}"

if ! command -v "$PYTHON" >/dev/null 2>&1; then
  echo "Error: $PYTHON not found"
  exit 1
fi

"$PYTHON" - <<'PY'
import sys

try:
    import build  # noqa: F401
    from setuptools import Distribution, setup  # noqa: F401
    from setuptools.command.bdist_wheel import bdist_wheel  # noqa: F401
except Exception as exc:
    sys.stderr.write(f"Error: Python wheel build dependencies are not ready: {exc}\n")
    sys.stderr.write('Install them with: python -m pip install "setuptools>=77.0.0" build twine\n')
    raise SystemExit(1)
PY

if [ ! -d "$BUILD_DIR" ]; then
  echo "Error: Build directory not found. Run 'make build-all' first."
  exit 1
fi

VERSION=$("$PYTHON" - "$PYPI_DIR/pyproject.toml" <<'PY'
from pathlib import Path
import re
import sys

text = Path(sys.argv[1]).read_text()
match = re.search(r'(?m)^version = "([^"]+)"', text)
if not match:
    raise SystemExit("version field not found in pyproject.toml")
print(match.group(1))
PY
)

rm -rf "$DIST_DIR" "$WORK_ROOT"
mkdir -p "$DIST_DIR" "$WORK_ROOT"

PLATFORMS=(
  "linux-x64|mothx-linux-amd64|manylinux_2_17_x86_64"
  "linux-arm64|mothx-linux-arm64|manylinux_2_17_aarch64"
  "linux-ppc64le|mothx-linux-ppc64le|manylinux_2_17_ppc64le"
  "linux-s390x|mothx-linux-s390x|manylinux_2_17_s390x"
  "linux-musl-x64|mothx-linux-musl-amd64|musllinux_1_2_x86_64"
  "linux-musl-arm64|mothx-linux-musl-arm64|musllinux_1_2_aarch64"
  "darwin-x64|mothx-darwin-amd64|macosx_10_15_x86_64"
  "darwin-arm64|mothx-darwin-arm64|macosx_11_0_arm64"
  "win32-x64|mothx-windows-amd64.exe|win_amd64"
  "win32-arm64|mothx-windows-arm64.exe|win_arm64"
)

echo "Building PyPI wheels for version $VERSION..."
echo ""

BUILT=0
for row in "${PLATFORMS[@]}"; do
  IFS='|' read -r PLATFORM_KEY BINARY_NAME PLAT_TAG <<< "$row"
  BINARY_PATH="$BUILD_DIR/$BINARY_NAME"
  WORK_DIR="$WORK_ROOT/$PLATFORM_KEY"

  if [ ! -f "$BINARY_PATH" ]; then
    echo "Warning: Binary not found: $BINARY_PATH, skipping $PLATFORM_KEY"
    continue
  fi

  mkdir -p "$WORK_DIR/src"
  cp "$PYPI_DIR/pyproject.toml" "$PYPI_DIR/setup.py" "$PYPI_DIR/README.md" "$WORK_DIR/"
  cp -R "$PYPI_DIR/src/vibecoding_installer" "$WORK_DIR/src/"
  rm -rf "$WORK_DIR/src/vibecoding_installer/bin"
  mkdir -p "$WORK_DIR/src/vibecoding_installer/bin"

  if [[ "$PLATFORM_KEY" == win32-* ]]; then
    INNER_BINARY="vibecoding.exe"
  else
    INNER_BINARY="vibecoding"
  fi

  cp "$BINARY_PATH" "$WORK_DIR/src/vibecoding_installer/bin/$INNER_BINARY"
  chmod +x "$WORK_DIR/src/vibecoding_installer/bin/$INNER_BINARY" 2>/dev/null || true

  (
    cd "$WORK_DIR"
    "$PYTHON" -m build --wheel --no-isolation --outdir "$DIST_DIR" -C--build-option=--plat-name="$PLAT_TAG"
  )

  SIZE=$(du -sh "$DIST_DIR"/*"$PLAT_TAG"*.whl | cut -f1 | tail -n1)
  echo "  Built: $PLATFORM_KEY ($PLAT_TAG) - $SIZE"
  BUILT=$((BUILT + 1))
done

echo ""
echo "Built $BUILT PyPI wheels in $DIST_DIR"
echo ""
echo "Skipped by design: linux-loong64, linux-riscv64, freebsd, openbsd, netbsd"
echo "PyPI/pip platform tags for those targets are not broadly portable; use npm or release archives for them."
echo ""
ls -lh "$DIST_DIR"/*.whl 2>/dev/null || true
