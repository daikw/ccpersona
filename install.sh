#!/bin/sh
# ccpersona installer script
# Usage: curl -sSL https://raw.githubusercontent.com/daikw/ccpersona/main/install.sh | sh
#
# Environment variables:
#   CCPERSONA_VERSION  - Specific version to install (default: latest)
#   CCPERSONA_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#
# Security note: Always review scripts before piping to shell.
# You can download and inspect this script first:
#   curl -sL https://raw.githubusercontent.com/daikw/ccpersona/main/install.sh > install.sh
#   # Review the script, then run: sh install.sh

set -e

REPO="daikw/ccpersona"
BINARY_NAME="ccpersona"
DEFAULT_INSTALL_DIR="/usr/local/bin"
GITHUB_BASE_URL="https://github.com/${REPO}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    printf "${GREEN}[INFO]${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1"
}

error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "Windows" ;;
        *)       error "Unsupported OS: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l)        echo "armv7" ;;
        armv6l)        echo "armv6" ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest release version from GitHub (with jq fallback)
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    local response
    local version

    if command -v curl >/dev/null 2>&1; then
        response=$(curl -sL "$api_url")
    elif command -v wget >/dev/null 2>&1; then
        response=$(wget -qO- "$api_url")
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    # Try jq first if available, fallback to grep/sed if jq fails
    # (jq can fail if response contains control characters)
    if command -v jq >/dev/null 2>&1; then
        version=$(echo "$response" | jq -r '.tag_name // empty' 2>/dev/null) || version=""
    fi

    # Fallback to grep/sed if jq failed or not available
    if [ -z "$version" ]; then
        version=$(echo "$response" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    fi

    echo "$version"
}

# Validate version format
validate_version() {
    local version="$1"
    case "$version" in
        v[0-9]*) return 0 ;;
        *) return 1 ;;
    esac
}

# Validate download URL
validate_url() {
    local url="$1"
    case "$url" in
        "${GITHUB_BASE_URL}/releases/download/"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Download file with error checking
download() {
    local url="$1"
    local output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -sL --fail -o "$output" "$url" || return 1
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$output" "$url" || return 1
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    # Verify file was created and has content
    [ -s "$output" ] || return 1
}

# Check required dependencies
check_dependencies() {
    local os="$1"

    if [ "$os" = "Windows" ]; then
        command -v unzip >/dev/null 2>&1 || error "unzip is required but not installed"
    else
        command -v tar >/dev/null 2>&1 || error "tar is required but not installed"
    fi
}

# Main installation
main() {
    info "Installing ccpersona..."

    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected: ${OS} ${ARCH}"

    # Check dependencies
    check_dependencies "$OS"

    # Get version
    VERSION="${CCPERSONA_VERSION:-$(get_latest_version)}"
    if [ -z "$VERSION" ]; then
        error "Could not determine version to install"
    fi

    # Validate version format
    if ! validate_version "$VERSION"; then
        error "Invalid version format: $VERSION (expected: v*.*.*)"
    fi
    info "Version: ${VERSION}"

    # Determine archive extension
    if [ "$OS" = "Windows" ]; then
        EXT="zip"
    else
        EXT="tar.gz"
    fi

    # Build download URL
    ARCHIVE_NAME="${BINARY_NAME}_${OS}_${ARCH}.${EXT}"
    DOWNLOAD_URL="${GITHUB_BASE_URL}/releases/download/${VERSION}/${ARCHIVE_NAME}"

    # Validate URL
    if ! validate_url "$DOWNLOAD_URL"; then
        error "Invalid download URL: $DOWNLOAD_URL"
    fi

    info "Downloading from: ${DOWNLOAD_URL}"

    # Create temp directory with restricted permissions
    TMP_DIR=$(mktemp -d)
    chmod 700 "$TMP_DIR"
    trap "rm -rf $TMP_DIR" EXIT

    # Download archive
    ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"
    if ! download "$DOWNLOAD_URL" "$ARCHIVE_PATH"; then
        error "Download failed. Check your network connection or verify the version exists."
    fi

    # Extract
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$EXT" = "zip" ]; then
        unzip -q "$ARCHIVE_PATH" || error "Failed to extract ZIP archive"
    else
        tar -xzf "$ARCHIVE_PATH" || error "Failed to extract tar archive"
    fi

    # Find binary
    BINARY_PATH=$(find "$TMP_DIR" -name "$BINARY_NAME" -type f | head -1)
    if [ -z "$BINARY_PATH" ]; then
        error "Binary not found in archive"
    fi

    # Verify binary is executable
    if [ ! -x "$BINARY_PATH" ]; then
        chmod +x "$BINARY_PATH" || error "Failed to make binary executable"
    fi

    # Install
    INSTALL_DIR="${CCPERSONA_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        SUDO=""
    else
        if command -v sudo >/dev/null 2>&1; then
            SUDO="sudo"
            info "Installing to ${INSTALL_DIR} (requires sudo)"
        else
            error "Cannot write to ${INSTALL_DIR} and sudo is not available. Set CCPERSONA_INSTALL_DIR to a writable directory."
        fi
    fi

    # Create directory if needed
    $SUDO mkdir -p "$INSTALL_DIR" || error "Failed to create installation directory"

    # Copy binary
    $SUDO cp "$BINARY_PATH" "${INSTALL_DIR}/${BINARY_NAME}" || error "Failed to copy binary"
    $SUDO chmod +x "${INSTALL_DIR}/${BINARY_NAME}" || error "Failed to set executable permissions"

    info "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

    # Verify installation
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        INSTALLED_VERSION=$("$BINARY_NAME" --version 2>&1 | head -1)
        info "Successfully installed: ${INSTALLED_VERSION}"
    else
        warn "Installed successfully, but ${BINARY_NAME} is not in PATH"
        warn "Add ${INSTALL_DIR} to your PATH, or run: export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi

    echo ""
    info "Quick start:"
    echo "  1. Start AivisSpeech or VOICEVOX"
    echo "  2. Run: ccpersona setup"
    echo "  3. Start a new Claude Code session"
    echo ""
    info "Documentation: https://github.com/${REPO}"
}

main "$@"
