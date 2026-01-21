#!/bin/bash
#
# try-go installer
# A Go port of https://github.com/tobi/try
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/try-go/main/install.sh | bash
#

set -e

REPO="YOUR_USERNAME/try-go"
BINARY_NAME="try"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac
    
    PLATFORM="${OS}_${ARCH}"
    info "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "location:" | sed 's/.*tag\///' | tr -d '\r\n')
    if [ -z "$VERSION" ]; then
        error "Failed to get latest version"
    fi
    info "Latest version: $VERSION"
}

# Download and install
install() {
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/try-go_${VERSION#v}_${PLATFORM}.tar.gz"
    
    info "Downloading from: $DOWNLOAD_URL"
    
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT
    
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/try-go.tar.gz"; then
        error "Failed to download release"
    fi
    
    tar -xzf "$TMP_DIR/try-go.tar.gz" -C "$TMP_DIR"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        warn "Need sudo to install to $INSTALL_DIR"
        sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    info "Installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME"
}

# Verify installation
verify() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        info "Installation successful!"
        echo ""
        $BINARY_NAME --version
        echo ""
        echo "Add to your shell config:"
        echo ""
        echo "  # Bash/Zsh - add to ~/.bashrc or ~/.zshrc"
        echo '  eval "$(try init)"'
        echo ""
        echo "  # Fish - add to ~/.config/fish/config.fish"
        echo '  eval (try init | string collect)'
        echo ""
    else
        warn "Installation completed but 'try' not found in PATH"
        warn "Make sure $INSTALL_DIR is in your PATH"
    fi
}

main() {
    echo ""
    echo "╔═══════════════════════════════════════════════╗"
    echo "║  try-go installer                             ║"
    echo "║  A Go port of github.com/tobi/try             ║"
    echo "╚═══════════════════════════════════════════════╝"
    echo ""
    
    detect_platform
    get_latest_version
    install
    verify
}

main
