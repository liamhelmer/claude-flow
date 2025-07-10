#!/usr/bin/env bash

# Copyright 2024 The Swarm Authors.
# Licensed under the Apache License, Version 2.0

set -e

# Configuration
REPO="claude-flow/kubectl-swarm"
BINARY_NAME="kubectl-swarm"
INSTALL_DIR="${KUBECTL_PLUGINS_PATH:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
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

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    case $OS in
        linux|darwin)
            PLATFORM="${OS}-${ARCH}"
            ;;
        mingw*|cygwin*|msys*)
            OS="windows"
            PLATFORM="${OS}-${ARCH}"
            BINARY_NAME="${BINARY_NAME}.exe"
            ;;
        *)
            error "Unsupported operating system: $OS"
            ;;
    esac

    info "Detected platform: $PLATFORM"
}

# Get the latest release version
get_latest_version() {
    info "Fetching latest release version..."
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        error "Failed to fetch latest version"
    fi
    
    info "Latest version: $VERSION"
}

# Download the binary
download_binary() {
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}.tar.gz"
    local tmp_dir=$(mktemp -d)
    local archive="${tmp_dir}/${BINARY_NAME}.tar.gz"
    
    info "Downloading ${BINARY_NAME} ${VERSION} for ${PLATFORM}..."
    
    if ! curl -fsL "$url" -o "$archive"; then
        error "Failed to download from $url"
    fi
    
    info "Extracting archive..."
    tar -xzf "$archive" -C "$tmp_dir"
    
    if [ ! -f "${tmp_dir}/${BINARY_NAME}" ]; then
        error "Binary not found in archive"
    fi
    
    echo "$tmp_dir"
}

# Install the binary
install_binary() {
    local src_dir=$1
    
    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"
    
    info "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."
    
    # Copy binary
    cp "${src_dir}/${BINARY_NAME}" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    # Clean up
    rm -rf "$src_dir"
}

# Verify installation
verify_installation() {
    if ! command -v kubectl >/dev/null 2>&1; then
        warn "kubectl is not installed. Please install kubectl first."
    fi
    
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        warn "${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "Add the following to your shell configuration file:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi
    
    if "${INSTALL_DIR}/${BINARY_NAME}" version >/dev/null 2>&1; then
        info "Installation successful!"
        info "Run 'kubectl swarm --help' to get started"
    else
        error "Installation verification failed"
    fi
}

# Main installation flow
main() {
    echo "🐝 Installing kubectl-swarm..."
    echo ""
    
    detect_platform
    get_latest_version
    
    # Allow version override
    if [ -n "$1" ]; then
        VERSION=$1
        info "Using specified version: $VERSION"
    fi
    
    tmp_dir=$(download_binary)
    install_binary "$tmp_dir"
    verify_installation
    
    echo ""
    echo "✅ kubectl-swarm has been installed successfully!"
    echo ""
    echo "To get started, run:"
    echo "  kubectl swarm --help"
    echo ""
    echo "For more information, visit:"
    echo "  https://github.com/${REPO}"
}

# Run main function
main "$@"