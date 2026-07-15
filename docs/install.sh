#!/usr/bin/env bash
set -euo pipefail

PACKAGE="mothx-installer"
NODE_HOME="${MOTHX_NODE_HOME:-$HOME/.mothx/node}"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { printf "%b[INFO]%b %s\n" "$BLUE" "$NC" "$*"; }
success() { printf "%b[SUCCESS]%b %s\n" "$GREEN" "$NC" "$*"; }
warn() { printf "%b[WARN]%b %s\n" "$YELLOW" "$NC" "$*"; }
fail() { printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$*" >&2; exit 1; }

show_help() {
    cat <<'EOF'
MothX online installer

Usage:
  curl -fsSL https://mothx.net/install.sh | bash
  curl -fsSL https://mothx.net/install.sh | bash -s -- --uninstall

The installer uses an existing Node.js installation when available. Otherwise it
installs the latest Node.js LTS release for the current system, then runs:
  npm install -g mothx-installer
EOF
}

download() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1" -o "$2"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        fail "curl or wget is required to install Node.js."
    fi
}

shell_config() {
    case "$(basename "${SHELL:-sh}")" in
        zsh) printf '%s\n' "$HOME/.zshrc" ;;
        bash) printf '%s\n' "$HOME/.bashrc" ;;
        fish) printf '%s\n' "$HOME/.config/fish/config.fish" ;;
        *) printf '%s\n' "$HOME/.profile" ;;
    esac
}

persist_node_path() {
    local config marker path_line
    config=$(shell_config)
    marker="# MothX Node.js"
    mkdir -p "$(dirname "$config")"
    touch "$config"
    grep -Fq "$marker" "$config" 2>/dev/null && return

    if [ "$(basename "${SHELL:-sh}")" = "fish" ]; then
        path_line="set -gx PATH $NODE_HOME/bin \$PATH"
    else
        path_line="export PATH=\"$NODE_HOME/bin:\$PATH\""
    fi
    printf '\n%s\n%s\n' "$marker" "$path_line" >> "$config"
    info "Added Node.js to PATH in $config"
}

install_node_archive() {
    local os arch node_arch version archive url tmp extracted
    case "$(uname -s)" in
        Linux) os="linux" ;;
        Darwin) os="darwin" ;;
        *) return 1 ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) node_arch="x64" ;;
        arm64|aarch64) node_arch="arm64" ;;
        armv7l) node_arch="armv7l" ;;
        ppc64le) node_arch="ppc64le" ;;
        s390x) node_arch="s390x" ;;
        *) return 1 ;;
    esac

    tmp=$(mktemp -d)
    trap "rm -rf '$tmp'" EXIT
    download "https://nodejs.org/dist/index.tab" "$tmp/index.tab"
    version=$(awk -F '\t' 'NR > 1 && $10 != "-" { print $1; exit }' "$tmp/index.tab")
    [ -n "$version" ] || fail "Could not determine the latest Node.js LTS version."

    archive="node-${version}-${os}-${node_arch}.tar.gz"
    url="https://nodejs.org/dist/${version}/${archive}"
    info "Installing Node.js ${version} for ${os}/${node_arch}..."
    download "$url" "$tmp/$archive"
    tar -xzf "$tmp/$archive" -C "$tmp"
    extracted="$tmp/${archive%.tar.gz}"

    rm -rf "$NODE_HOME"
    mkdir -p "$(dirname "$NODE_HOME")"
    mv "$extracted" "$NODE_HOME"
    export PATH="$NODE_HOME/bin:$PATH"
    persist_node_path
}

install_node_package_manager() {
    info "Installing Node.js with the system package manager..."
    if command -v pkg >/dev/null 2>&1; then
        sudo pkg install -y node npm
    elif command -v pkgin >/dev/null 2>&1; then
        sudo pkgin -y install nodejs npm
    elif command -v pkg_add >/dev/null 2>&1; then
        doas pkg_add node npm
    else
        return 1
    fi
}

install_node() {
    info "Node.js was not found. Installing the latest LTS release..."
    install_node_archive || install_node_package_manager || \
        fail "This system is not supported for automatic Node.js installation. Install Node.js LTS and run: npm install -g $PACKAGE"
    command -v node >/dev/null 2>&1 || fail "Node.js installation completed, but node is not available in PATH."
}

install_mothx() {
    if command -v node >/dev/null 2>&1; then
        info "Using Node.js $(node --version)."
    else
        install_node
    fi

    command -v npm >/dev/null 2>&1 || fail "npm was not found alongside Node.js."
    info "Installing the latest MothX release..."
    if npm install -g "$PACKAGE"; then
        :
    elif command -v sudo >/dev/null 2>&1; then
        warn "Global npm directory is not writable; retrying with sudo."
        sudo npm install -g "$PACKAGE"
    else
        fail "npm global installation failed. Check npm permissions and retry."
    fi

    success "MothX installed successfully."
    if command -v mothx >/dev/null 2>&1; then
        mothx --version || true
    else
        warn "Open a new terminal if the mothx command is not yet in PATH."
    fi
}

case "${1:-}" in
    -h|--help) show_help ;;
    -u|--uninstall)
        command -v npm >/dev/null 2>&1 || fail "npm is required to uninstall MothX."
        npm uninstall -g "$PACKAGE"
        success "MothX uninstalled."
        ;;
    "") install_mothx ;;
    *) fail "Unknown option: $1" ;;
esac
