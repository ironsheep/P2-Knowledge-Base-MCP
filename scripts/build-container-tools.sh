#!/bin/bash
#
# build-container-tools.sh - Build the container-tools distribution package
#
# Creates a tarball that installs to /opt/container-tools/p2kb-mcp/
# with cache at /opt/container-tools/var/cache/p2kb-mcp/
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Get version from VERSION file
VERSION=$(cat "${REPO_ROOT}/VERSION" | tr -d '\n')
if [ -z "$VERSION" ]; then
    echo "ERROR: VERSION file is empty or missing"
    exit 1
fi

MCP_NAME="p2kb-mcp"
PACKAGE_NAME="${MCP_NAME}-v${VERSION}-container-tools"
BUILD_DIR="${REPO_ROOT}/builds/container-tools"
PACKAGE_DIR="${BUILD_DIR}/${PACKAGE_NAME}"

# Build metadata
BUILD_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "=============================================="
echo "Building ${MCP_NAME} Container Tools Package"
echo "=============================================="
echo "Version:    ${VERSION}"
echo "Commit:     ${GIT_COMMIT}"
echo "Build Date: ${BUILD_DATE}"
echo ""

# Clean and create build directory
# Package structure:
#   p2kb-mcp/
#   ├── bin/
#   │   ├── p2kb-mcp           (universal launcher)
#   │   └── platforms/         (platform binaries)
#   ├── README.md
#   ├── CHANGELOG.md
#   ├── LICENSE
#   └── install.sh
rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}/${MCP_NAME}/bin/platforms"

# Build function
build_binary() {
    local os=$1
    local arch=$2
    local suffix=$3
    local output_name="${MCP_NAME}-v${VERSION}-${os}-${arch}${suffix}"

    echo "Building ${output_name}..."

    LDFLAGS="-s -w"
    LDFLAGS="${LDFLAGS} -X 'main.Version=${VERSION}'"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_DATE}'"
    LDFLAGS="${LDFLAGS} -X 'main.GitCommit=${GIT_COMMIT}'"

    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
        -ldflags="${LDFLAGS}" \
        -o "${PACKAGE_DIR}/${MCP_NAME}/bin/platforms/${output_name}" \
        "${REPO_ROOT}/cmd/p2kb-mcp"
}

# Build all platforms
echo "Building platform binaries..."

build_binary "darwin" "amd64" ""
build_binary "darwin" "arm64" ""
build_binary "linux" "amd64" ""
build_binary "linux" "arm64" ""
build_binary "windows" "amd64" ".exe"
build_binary "windows" "arm64" ".exe"

echo ""
echo "Creating universal launcher..."

# Create universal launcher script
cat > "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}" << 'LAUNCHER'
#!/usr/bin/env bash
#
# Universal launcher for p2kb-mcp
# Auto-detects OS, architecture, and container environment
#
set -e

LAUNCHER_VERSION="__VERSION__"
BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${BIN_DIR}/platforms"
MCP_NAME="p2kb-mcp"

# Debug mode
debug() {
    if [ -n "${P2KB_MCP_DEBUG}" ]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Container detection
is_container() {
    [ -f /.dockerenv ] && return 0
    [ -n "${CONTAINER}" ] && return 0
    [ -f /proc/1/cgroup ] && grep -qE 'docker|containerd|podman|kubernetes' /proc/1/cgroup 2>/dev/null && return 0
    return 1
}

# Architecture detection
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

# OS detection
detect_os() {
    case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
        darwin) echo "darwin" ;;
        linux) echo "linux" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

# Binary selection
select_binary() {
    local os=$(detect_os)
    local arch=$(detect_arch)
    local suffix=""

    # Always use Linux binary in containers
    if is_container; then
        debug "Container detected, using Linux binary"
        os="linux"
    fi

    if [ "$os" = "windows" ]; then
        suffix=".exe"
    fi

    local binary="${PLATFORMS_DIR}/${MCP_NAME}-v${LAUNCHER_VERSION}-${os}-${arch}${suffix}"

    debug "OS: ${os}, Arch: ${arch}, Container: $(is_container && echo yes || echo no)"
    debug "Selected binary: ${binary}"

    if [ ! -f "$binary" ]; then
        echo "ERROR: Binary not found: ${binary}" >&2
        echo "Available binaries:" >&2
        ls -1 "${PLATFORMS_DIR}/" >&2
        exit 1
    fi

    echo "$binary"
}

# Execute
exec "$(select_binary)" "$@"
LAUNCHER

# Replace version placeholder
sed -i.bak "s/__VERSION__/${VERSION}/g" "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}"
rm -f "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}.bak"
chmod +x "${PACKAGE_DIR}/${MCP_NAME}/bin/${MCP_NAME}"

echo "Copying documentation files..."

# Copy README, CHANGELOG, LICENSE from repo root
cp "${REPO_ROOT}/README.md" "${PACKAGE_DIR}/${MCP_NAME}/"
cp "${REPO_ROOT}/CHANGELOG.md" "${PACKAGE_DIR}/${MCP_NAME}/"
cp "${REPO_ROOT}/LICENSE" "${PACKAGE_DIR}/${MCP_NAME}/"

echo "Creating install script..."

# Create install.sh
cat > "${PACKAGE_DIR}/${MCP_NAME}/install.sh" << 'INSTALL'
#!/bin/bash
#
# Install p2kb-mcp into /opt/container-tools/
# Creates proper directory structure for container-tools ecosystem
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MCP_NAME="p2kb-mcp"
INSTALL_ROOT="/opt/container-tools"

echo "Installing ${MCP_NAME} to ${INSTALL_ROOT}..."
echo ""

# Check for root/sudo
if [ "$EUID" -ne 0 ] && [ ! -w "$INSTALL_ROOT" ] 2>/dev/null; then
    echo "This script requires sudo to install to ${INSTALL_ROOT}"
    echo "Re-running with sudo..."
    exec sudo "$0" "$@"
fi

# Create container-tools directory structure
echo "Creating directory structure..."
mkdir -p "${INSTALL_ROOT}/bin"
mkdir -p "${INSTALL_ROOT}/etc"
mkdir -p "${INSTALL_ROOT}/var/cache/${MCP_NAME}"

# Backup existing installation if present
if [ -d "${INSTALL_ROOT}/${MCP_NAME}" ]; then
    echo "Backing up existing ${MCP_NAME}..."
    mv "${INSTALL_ROOT}/${MCP_NAME}" "${INSTALL_ROOT}/${MCP_NAME}.backup.$(date +%Y%m%d%H%M%S)"
fi

# Copy MCP files
echo "Copying ${MCP_NAME} files..."
cp -r "${SCRIPT_DIR}" "${INSTALL_ROOT}/${MCP_NAME}"

# Remove install.sh from installed location (not needed after install)
rm -f "${INSTALL_ROOT}/${MCP_NAME}/install.sh"

# Make binaries executable
chmod +x "${INSTALL_ROOT}/${MCP_NAME}/bin/${MCP_NAME}"
chmod +x "${INSTALL_ROOT}/${MCP_NAME}/bin/platforms"/*

# Create symlink in bin directory
echo "Creating symlink..."
ln -sf "../${MCP_NAME}/bin/${MCP_NAME}" "${INSTALL_ROOT}/bin/${MCP_NAME}"

# Set up cache directory permissions
chmod 755 "${INSTALL_ROOT}/var/cache/${MCP_NAME}"

# Update mcp.json
MCP_JSON="${INSTALL_ROOT}/etc/mcp.json"
echo "Updating MCP configuration..."

if [ -f "$MCP_JSON" ]; then
    # Check if jq is available
    if command -v jq &> /dev/null; then
        # Merge our entry into existing config
        TEMP_JSON=$(mktemp)
        jq --arg cmd "${INSTALL_ROOT}/bin/${MCP_NAME}" \
           '.mcpServers["p2kb-mcp"] = {"command": $cmd, "args": []}' \
           "$MCP_JSON" > "$TEMP_JSON"
        mv "$TEMP_JSON" "$MCP_JSON"
        echo "  Merged into existing mcp.json"
    else
        echo ""
        echo "WARNING: jq not installed. Please manually add to ${MCP_JSON}:"
        echo ""
        echo '  "p2kb-mcp": {'
        echo "    \"command\": \"${INSTALL_ROOT}/bin/${MCP_NAME}\","
        echo '    "args": []'
        echo '  }'
        echo ""
    fi
else
    # Create new mcp.json
    cat > "$MCP_JSON" << MCPJSON
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "${INSTALL_ROOT}/bin/${MCP_NAME}",
      "args": []
    }
  }
}
MCPJSON
    echo "  Created new mcp.json"
fi

echo ""
echo "=============================================="
echo "Installation complete!"
echo "=============================================="
echo ""
echo "Installed to: ${INSTALL_ROOT}/${MCP_NAME}/"
echo "Symlink:      ${INSTALL_ROOT}/bin/${MCP_NAME}"
echo "Cache:        ${INSTALL_ROOT}/var/cache/${MCP_NAME}/"
echo "Config:       ${INSTALL_ROOT}/etc/mcp.json"
echo ""
echo "Test the installation:"
echo "  ${INSTALL_ROOT}/bin/${MCP_NAME} --version"
INSTALL

chmod +x "${PACKAGE_DIR}/${MCP_NAME}/install.sh"

echo "Creating VERSION_MANIFEST.txt..."

# Create VERSION_MANIFEST.txt
cat > "${PACKAGE_DIR}/VERSION_MANIFEST.txt" << MANIFEST
Package: ${MCP_NAME}
Version: ${VERSION}
Git Commit: ${GIT_COMMIT}
Build Date: ${BUILD_DATE}
Build Host: $(hostname 2>/dev/null || echo "unknown")

Package Type: container-tools
Install Location: /opt/container-tools/${MCP_NAME}/
Cache Location: /opt/container-tools/var/cache/${MCP_NAME}/

Platforms:
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64
  - windows-amd64
  - windows-arm64
MANIFEST

echo ""
echo "Creating tarball..."

# Create tarball - package contains p2kb-mcp/ directory
cd "${BUILD_DIR}"
tar -czf "${PACKAGE_NAME}.tar.gz" -C "${PACKAGE_NAME}" "${MCP_NAME}" VERSION_MANIFEST.txt

echo ""
echo "=============================================="
echo "Build complete!"
echo "=============================================="
echo ""
echo "Package: ${BUILD_DIR}/${PACKAGE_NAME}.tar.gz"
echo ""
echo "Package contents:"
tar -tzf "${PACKAGE_NAME}.tar.gz" | sed 's/^/  /'
echo ""
echo "To install:"
echo "  tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "  cd ${MCP_NAME}"
echo "  sudo ./install.sh"
