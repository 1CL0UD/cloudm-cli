#!/usr/bin/env bash
set -e

# cloudm-cli installer script
# Usage: curl -sSL https://raw.githubusercontent.com/1CL0UD/cloudm-cli/main/install.sh | bash

REPO="1CL0UD/cloudm-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="cloudm-cli"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Temp file tracking
DOWNLOADED_FILE=""
TMP_DIR=""

print_info() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}==>${NC} $1"
}

print_error() {
    echo -e "${RED}Error:${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

# Cleanup on exit
cleanup() {
    if [ -n "$DOWNLOADED_FILE" ] && [ -f "$DOWNLOADED_FILE" ]; then
        rm -f "$DOWNLOADED_FILE"
    fi
    if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# Detect OS and architecture
detect_platform() {
    local os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local arch="$(uname -m)"

    case "$os" in
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
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
}

# Get latest release version
get_latest_version() {
    print_info "Fetching latest version..."
    
    local api_response
    if command -v curl >/dev/null 2>&1; then
        api_response=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || echo "")
    elif command -v wget >/dev/null 2>&1; then
        api_response=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || echo "")
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    VERSION=$(echo "$api_response" | grep '"tag_name"' | cut -d'"' -f4)

    if [ -z "$VERSION" ]; then
        print_warning "No releases found. Will try to build from source."
        return 1
    fi

    print_info "Latest version: $VERSION"
    return 0
}

# Download binary
download_binary() {
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}-${PLATFORM}"
    DOWNLOADED_FILE="/tmp/${BINARY_NAME}-$$"

    print_info "Downloading ${BINARY_NAME}-${PLATFORM}..."

    local download_success=false
    if command -v curl >/dev/null 2>&1; then
        if curl -fsSL "$url" -o "$DOWNLOADED_FILE" 2>/dev/null; then
            download_success=true
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q "$url" -O "$DOWNLOADED_FILE" 2>/dev/null; then
            download_success=true
        fi
    fi

    if [ "$download_success" = false ]; then
        print_warning "Download failed. Will try to build from source."
        rm -f "$DOWNLOADED_FILE"
        return 1
    fi

    chmod +x "$DOWNLOADED_FILE"
    
    # Verify binary works
    if ! "$DOWNLOADED_FILE" version >/dev/null 2>&1; then
        print_warning "Downloaded binary is not valid. Will try to build from source."
        rm -f "$DOWNLOADED_FILE"
        return 1
    fi

    return 0
}

# Build from source (fallback)
build_from_source() {
    print_info "Building from source..."

    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is required to build from source. Please install Go 1.21+ or wait for a release."
        exit 1
    fi

    if ! command -v git >/dev/null 2>&1; then
        print_error "Git is required to build from source."
        exit 1
    fi

    TMP_DIR=$(mktemp -d)
    local src_dir="${TMP_DIR}/src"

    print_info "Cloning repository..."
    git clone --depth 1 "https://github.com/${REPO}.git" "$src_dir" >/dev/null 2>&1

    cd "$src_dir"
    
    print_info "Compiling..."
    DOWNLOADED_FILE="${TMP_DIR}/${BINARY_NAME}"
    go build -ldflags "-X main.Version=source -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=$(git rev-parse --short HEAD)" -o "$DOWNLOADED_FILE" main.go

    chmod +x "$DOWNLOADED_FILE"
    print_success "Built from source successfully"
}

# Install binary
install_binary() {
    print_info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

    # Ensure install directory exists
    if [ ! -d "$INSTALL_DIR" ]; then
        if [ -w "$(dirname "$INSTALL_DIR")" ]; then
            mkdir -p "$INSTALL_DIR"
        else
            sudo mkdir -p "$INSTALL_DIR"
        fi
    fi

    # Install binary
    if [ -w "$INSTALL_DIR" ]; then
        cp "$DOWNLOADED_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        print_warning "Need sudo to install to ${INSTALL_DIR}"
        sudo cp "$DOWNLOADED_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        print_info "Verifying installation..."
        echo ""
        "$BINARY_NAME" version
        echo ""
        print_success "Successfully installed ${BINARY_NAME}!"
        print_info "Run '${BINARY_NAME} --help' to get started"
    else
        print_warning "Installation completed but ${BINARY_NAME} not found in PATH"
        print_info "You may need to add ${INSTALL_DIR} to your PATH:"
        echo ""
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi
}

# Main installation flow
main() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║       cloudm-cli Installer            ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    detect_platform
    print_info "Detected platform: ${PLATFORM}"
    
    # Try to get latest version and download, fallback to source
    if get_latest_version && download_binary; then
        print_success "Downloaded release successfully"
    else
        build_from_source
    fi

    install_binary
    verify_installation

    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║     Installation Complete! 🎉         ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""
}

# Run main installation
main
