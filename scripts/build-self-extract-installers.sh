#!/bin/bash
set -euo pipefail

# MothX self-extracting installer builder.
# Usage: ./scripts/build-self-extract-installers.sh [version]

BINARY_NAME="mothx"
PACKAGE_NAME="mothx"
VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "0.0.1")}"
VERSION="${VERSION#v}"
VERSION="${VERSION%-dirty}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist/installers"
BUILD_DIR="${ROOT_DIR}/.tmp/self-extract"
WIN_TEMPLATE="${ROOT_DIR}/packaging/windows/self-extract/install.bat.tmpl"
WIN_UNINSTALL_TEMPLATE="${ROOT_DIR}/packaging/windows/self-extract/uninstall.bat.tmpl"
WIN_README="${ROOT_DIR}/packaging/windows/self-extract/README.txt"
MAC_TEMPLATE="${ROOT_DIR}/packaging/macos/self-extract/install.sh.tmpl"
MAC_UNINSTALL_TEMPLATE="${ROOT_DIR}/packaging/macos/self-extract/uninstall-mothx.sh.tmpl"
MAC_README="${ROOT_DIR}/packaging/macos/self-extract/README.txt"

require_file() {
    local path=$1
    if [[ ! -f "$path" ]]; then
        echo "Error: required file not found: $path" >&2
        exit 1
    fi
}

require_cmd() {
    local cmd=$1
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Error: required command not found: $cmd" >&2
        exit 1
    fi
}

sha256_file() {
    sha256sum "$1" | awk '{print $1}'
}

render_template() {
    local template=$1
    local output=$2
    local version=$3
    local platform=$4
    local expected_machine=$5
    local payload_sha=$6

    sed \
        -e "s|{{VERSION}}|${version}|g" \
        -e "s|{{PLATFORM}}|${platform}|g" \
        -e "s|{{EXPECTED_MACHINE}}|${expected_machine}|g" \
        -e "s|{{PAYLOAD_SHA256}}|${payload_sha}|g" \
        "$template" > "$output"
}

append_payload() {
    local payload=$1
    local installer=$2

    # Keep wrapped output for text-editor friendliness. Installers strip whitespace.
    base64 -w 76 "$payload" >> "$installer"
}

build_windows_installer() {
    local arch="amd64"
    local platform="windows-x64"
    local binary="${ROOT_DIR}/bin/${BINARY_NAME}-windows-${arch}.exe"
    local payload_dir="${BUILD_DIR}/payload/${platform}"
    local payload_zip="${BUILD_DIR}/${PACKAGE_NAME}-${VERSION}-${platform}-payload.zip"
    local installer="${DIST_DIR}/${PACKAGE_NAME}-${VERSION}-${platform}-install.bat"
    local roundtrip_dir="${BUILD_DIR}/roundtrip/${platform}"

    require_file "$binary"
    require_file "$WIN_TEMPLATE"
    require_file "$WIN_UNINSTALL_TEMPLATE"
    require_file "$WIN_README"

    echo "Building ${PACKAGE_NAME}-${VERSION}-${platform}-install.bat..."
    rm -rf "$payload_dir" "$roundtrip_dir"
    mkdir -p "$payload_dir" "$roundtrip_dir"

    cp "$binary" "${payload_dir}/${BINARY_NAME}.exe"
    cp "$WIN_README" "${payload_dir}/README.txt"
    sed -e "s|{{VERSION}}|${VERSION}|g" "$WIN_UNINSTALL_TEMPLATE" > "${payload_dir}/uninstall.bat"

    rm -f "$payload_zip"
    (
        cd "$payload_dir"
        zip -q -9 -r "$payload_zip" .
    )

    local payload_sha
    payload_sha=$(sha256_file "$payload_zip")

    render_template "$WIN_TEMPLATE" "$installer" "$VERSION" "$platform" "" "$payload_sha"
    append_payload "$payload_zip" "$installer"

    verify_windows_payload "$installer" "$payload_sha" "$roundtrip_dir"
    echo "  Created: ${installer#${ROOT_DIR}/}"
}

verify_windows_payload() {
    local installer=$1
    local expected_sha=$2
    local out_dir=$3
    local payload_b64="${out_dir}/payload.b64"
    local payload_zip="${out_dir}/payload.zip"

    extract_last_payload "$installer" "$payload_b64"

    base64 -d "$payload_b64" > "$payload_zip"
    local actual_sha
    actual_sha=$(sha256_file "$payload_zip")
    if [[ "$actual_sha" != "$expected_sha" ]]; then
        echo "Error: Windows payload round-trip checksum mismatch." >&2
        echo "Expected: $expected_sha" >&2
        echo "Actual:   $actual_sha" >&2
        exit 1
    fi

    unzip -q -t "$payload_zip" >/dev/null
}

build_macos_installer() {
    local goarch=$1
    local platform=$2
    local expected_machine=$3
    local binary="${ROOT_DIR}/bin/${BINARY_NAME}-darwin-${goarch}"
    local payload_dir="${BUILD_DIR}/payload/${platform}"
    local payload_tgz="${BUILD_DIR}/${PACKAGE_NAME}-${VERSION}-${platform}-payload.tar.gz"
    local installer="${DIST_DIR}/${PACKAGE_NAME}-${VERSION}-${platform}-install.sh"
    local roundtrip_dir="${BUILD_DIR}/roundtrip/${platform}"

    require_file "$binary"
    require_file "$MAC_TEMPLATE"
    require_file "$MAC_UNINSTALL_TEMPLATE"
    require_file "$MAC_README"

    echo "Building ${PACKAGE_NAME}-${VERSION}-${platform}-install.sh..."
    rm -rf "$payload_dir" "$roundtrip_dir"
    mkdir -p "$payload_dir" "$roundtrip_dir"

    cp "$binary" "${payload_dir}/${BINARY_NAME}"
    chmod 755 "${payload_dir}/${BINARY_NAME}"
    cp "$MAC_README" "${payload_dir}/README.txt"
    sed -e "s|{{VERSION}}|${VERSION}|g" "$MAC_UNINSTALL_TEMPLATE" > "${payload_dir}/uninstall-mothx.sh"
    chmod 755 "${payload_dir}/uninstall-mothx.sh"

    rm -f "$payload_tgz"
    tar -C "$payload_dir" -czf "$payload_tgz" .

    local payload_sha
    payload_sha=$(sha256_file "$payload_tgz")

    render_template "$MAC_TEMPLATE" "$installer" "$VERSION" "$platform" "$expected_machine" "$payload_sha"
    append_payload "$payload_tgz" "$installer"
    chmod 755 "$installer"

    verify_macos_payload "$installer" "$payload_sha" "$roundtrip_dir"
    echo "  Created: ${installer#${ROOT_DIR}/}"
}

verify_macos_payload() {
    local installer=$1
    local expected_sha=$2
    local out_dir=$3
    local payload_b64="${out_dir}/payload.b64"
    local payload_tgz="${out_dir}/payload.tar.gz"

    extract_last_payload "$installer" "$payload_b64"

    base64 -d "$payload_b64" > "$payload_tgz"
    local actual_sha
    actual_sha=$(sha256_file "$payload_tgz")
    if [[ "$actual_sha" != "$expected_sha" ]]; then
        echo "Error: macOS payload round-trip checksum mismatch." >&2
        echo "Expected: $expected_sha" >&2
        echo "Actual:   $actual_sha" >&2
        exit 1
    fi

    tar -tzf "$payload_tgz" >/dev/null
}

extract_last_payload() {
    local installer=$1
    local output=$2

    awk '
        {
            lines[NR] = $0
            stripped = $0
            sub(/\r$/, "", stripped)
            if (stripped == "__MOTHX_PAYLOAD_BELOW__") {
                marker = NR
            }
        }
        END {
            if (marker < 1) {
                exit 1
            }
            for (i = marker + 1; i <= NR; i++) {
                print lines[i]
            }
        }
    ' "$installer" > "$output"
}

write_checksums() {
    local checksum_file="${DIST_DIR}/installer-checksums.txt"

    (
        cd "$DIST_DIR"
        find . -maxdepth 1 -type f \( -name "*.bat" -o -name "*.sh" \) | sort | while read -r file; do
            sha256sum "$file"
        done
    ) > "$checksum_file"

    echo "Checksums written to ${checksum_file#${ROOT_DIR}/}"
}

main() {
    require_cmd base64
    require_cmd sha256sum
    require_cmd sed
    require_cmd awk
    require_cmd tar
    require_cmd zip
    require_cmd unzip

    rm -rf "$BUILD_DIR"
    mkdir -p "$DIST_DIR" "$BUILD_DIR"

    build_windows_installer
    build_macos_installer "amd64" "macos-x64" "x86_64"
    build_macos_installer "arm64" "macos-arm64" "arm64"
    write_checksums
    rm -rf "$BUILD_DIR"

    echo "Self-extracting installers complete."
}

main "$@"
