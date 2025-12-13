#!/bin/bash
#
# build-standalone.sh - Build standalone platform distribution packages
#
# Creates individual packages for each platform:
#   - Linux (amd64, arm64): .tar.gz with opt-style structure
#   - macOS (amd64, arm64): .tar.gz with opt-style structure
#   - Windows (amd64, arm64): .zip with simple structure
#
# Package structure for Linux/macOS:
#   p2kb-mcp/
#   ├── bin/
#   │   └── p2kb-mcp
#   ├── .cache/              (created at runtime)
#   ├── README.md
#   ├── CHANGELOG.md
#   └── LICENSE
#
# Package structure for Windows:
#   p2kb-mcp/
#   ├── bin/
#   │   └── p2kb-mcp.exe
#   ├── README.md
#   ├── CHANGELOG.md
#   └── LICENSE
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
BUILD_DIR="${REPO_ROOT}/builds/standalone"

# Build metadata
BUILD_DATE=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "=============================================="
echo "Building ${MCP_NAME} Standalone Packages"
echo "=============================================="
echo "Version:    ${VERSION}"
echo "Commit:     ${GIT_COMMIT}"
echo "Build Date: ${BUILD_DATE}"
echo ""

# Clean build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Build function for a single platform
build_platform() {
    local os=$1
    local arch=$2
    local suffix=$3

    local platform_name="${os}-${arch}"
    local binary_name="${MCP_NAME}${suffix}"
    local package_name="${MCP_NAME}-v${VERSION}-${platform_name}"
    local package_dir="${BUILD_DIR}/${package_name}"

    echo ""
    echo "Building ${package_name}..."

    # Create package directory structure
    mkdir -p "${package_dir}/${MCP_NAME}/bin"

    # Build binary
    LDFLAGS="-s -w"
    LDFLAGS="${LDFLAGS} -X 'main.Version=${VERSION}'"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_DATE}'"
    LDFLAGS="${LDFLAGS} -X 'main.GitCommit=${GIT_COMMIT}'"

    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build \
        -ldflags="${LDFLAGS}" \
        -o "${package_dir}/${MCP_NAME}/bin/${binary_name}" \
        "${REPO_ROOT}/cmd/p2kb-mcp"

    # Copy documentation
    cp "${REPO_ROOT}/README.md" "${package_dir}/${MCP_NAME}/"
    cp "${REPO_ROOT}/CHANGELOG.md" "${package_dir}/${MCP_NAME}/"
    cp "${REPO_ROOT}/LICENSE" "${package_dir}/${MCP_NAME}/"

    # Create package
    cd "${BUILD_DIR}"
    if [ "$os" = "windows" ]; then
        # Windows: create zip
        if command -v zip &> /dev/null; then
            zip -r "${package_name}.zip" "${package_name}"
            rm -rf "${package_dir}"
            echo "  Created: ${package_name}.zip"
        else
            # Fallback to tar.gz if zip not available
            echo "  Note: 'zip' not found, creating .tar.gz instead"
            tar -czf "${package_name}.tar.gz" "${package_name}"
            rm -rf "${package_dir}"
            echo "  Created: ${package_name}.tar.gz"
        fi
    else
        # Linux/macOS: create tar.gz
        tar -czf "${package_name}.tar.gz" "${package_name}"
        rm -rf "${package_dir}"
        echo "  Created: ${package_name}.tar.gz"
    fi
}

# Build all platforms
echo "Building platform packages..."

build_platform "linux" "amd64" ""
build_platform "linux" "arm64" ""
build_platform "darwin" "amd64" ""
build_platform "darwin" "arm64" ""
build_platform "windows" "amd64" ".exe"
build_platform "windows" "arm64" ".exe"

echo ""
echo "=============================================="
echo "Build complete!"
echo "=============================================="
echo ""
echo "Packages created in: ${BUILD_DIR}/"
echo ""
ls -la "${BUILD_DIR}/"
echo ""
echo "Installation instructions:"
echo ""
echo "Linux/macOS:"
echo "  tar -xzf ${MCP_NAME}-v${VERSION}-{platform}.tar.gz"
echo "  sudo mv ${MCP_NAME} /opt/"
echo ""
echo "Windows:"
echo "  Extract zip to desired location (e.g., C:\\Program Files\\${MCP_NAME})"
echo ""
echo "Cache location:"
echo "  Linux/macOS: {install}/.cache/ (created automatically)"
echo "  Windows: %LOCALAPPDATA%\\${MCP_NAME}\\cache\\"
