#!/usr/bin/env bash

set -e

GITHUB_REPO="logandonley/font-manager"
BINARY_NAME="fm"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

info() {
    echo -e "${BLUE}$1${NC}"
}

success() {
    echo -e "${GREEN}$1${NC}"
}

# Check for required commands
command -v curl >/dev/null 2>&1 || error "curl is required but not installed"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in
    x86_64|amd64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) error "Unsupported architecture: $(uname -m)" ;;
esac

# Only support macOS and Linux
case $OS in
    darwin|linux) ;;
    *) error "Unsupported operating system: $OS" ;;
esac

# Get the latest release version
info "Finding latest release..."
LATEST_RELEASE=$(curl -sL -H 'Accept: application/json' "https://api.github.com/repos/${GITHUB_REPO}/releases/latest")
VERSION=$(echo "$LATEST_RELEASE" | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)

if [ -z "$VERSION" ]; then
    error "Could not determine latest version"
fi

info "Latest version: $VERSION"

# Generate download URL
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${BINARY_NAME}_${OS}_${ARCH}"

# Create temporary directory
TMP_DIR=$(mktemp -d)
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

# Download binary
info "Downloading $DOWNLOAD_URL..."
curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/fm"

# Install binary
info "Installing to $INSTALL_DIR..."
if [ ! -w "$INSTALL_DIR" ]; then
    sudo mv "$TMP_DIR/fm" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/fm"
else
    mv "$TMP_DIR/fm" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/fm"
fi

# Verify installation
if command -v fm >/dev/null 2>&1; then
    success "Successfully installed fm $VERSION"
    success "Run 'fm --help' to get started"
else
    error "Installation failed. Please check if $INSTALL_DIR is in your PATH"
fi
