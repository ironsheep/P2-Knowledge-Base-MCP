#!/bin/bash
#
# build-container-tools.sh - Build the container-tools distribution package
#
# Creates a tarball that installs to /opt/container-tools/p2kb-mcp/
# following the Container Tools MCP Integration Guide
#
# Package structure:
#   p2kb-mcp/
#   ├── bin/
#   │   ├── p2kb-mcp           (universal launcher)
#   │   └── platforms/         (platform binaries)
#   ├── README.md
#   ├── CHANGELOG.md
#   ├── LICENSE
#   ├── VERSION_MANIFEST.txt
#   └── install.sh
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
PACKAGE_NAME="container-tools-${MCP_NAME}-v${VERSION}"
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
# Resolve symlinks to get the actual script location
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -L "$SCRIPT_PATH" ]; do
    SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
    SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"
    [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$SCRIPT_DIR/$SCRIPT_PATH"
done
BIN_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
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
# p2kb-mcp installer for container-tools
#
# Usage:
#   ./install.sh [OPTIONS] [target-dir]
#
# Options:
#   --target DIR    Install to DIR (default: /opt/container-tools)
#   --uninstall     Remove/rollback p2kb-mcp from container-tools
#   --help          Show this help
#
# Default target: /opt/container-tools
#

set -e

YOUR_MCP="p2kb-mcp"
VERSION="__VERSION__"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Parse arguments
UNINSTALL=false
TARGET="/opt/container-tools"

while [[ $# -gt 0 ]]; do
    case $1 in
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --target)
            TARGET="$2"
            shift 2
            ;;
        --help|-h)
            head -20 "$0" | tail -15
            exit 0
            ;;
        *)
            TARGET="$1"
            shift
            ;;
    esac
done

# Check for sudo if needed
need_sudo() {
    if [ -w "$TARGET" ] 2>/dev/null || [ -w "$(dirname "$TARGET")" ] 2>/dev/null; then
        echo ""
    else
        echo "sudo"
    fi
}
SUDO=$(need_sudo)

# Get script directory (where the package was extracted)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

#
# PLATFORM DETECTION
#
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) arch="unknown" ;;
    esac
    echo "${os}-${arch}"
}

#
# LEGACY CLEANUP (remove old hooks-dispatcher pattern if present)
#
cleanup_legacy() {
    # Remove our hook from old hooks.d location
    $SUDO rm -f "$TARGET/etc/hooks.d/app-start/$YOUR_MCP.sh" 2>/dev/null || true

    # Check if hooks-dispatcher is still needed by other MCPs
    local hooks_d_count=0
    if [ -d "$TARGET/etc/hooks.d" ]; then
        hooks_d_count=$(find "$TARGET/etc/hooks.d" -name "*.sh" -type f 2>/dev/null | wc -l)
    fi

    if [ "$hooks_d_count" -eq 0 ]; then
        # No other hooks, safe to remove dispatcher infrastructure
        $SUDO rm -f "$TARGET/etc/hooks-dispatcher.sh" 2>/dev/null || true
        $SUDO rm -rf "$TARGET/etc/hooks.d" 2>/dev/null || true
    fi

    # Remove old hook format from mcp.json if present
    if [ -f "$TARGET/etc/mcp.json" ] && command -v jq &> /dev/null; then
        if jq -e '.hooks' "$TARGET/etc/mcp.json" > /dev/null 2>&1; then
            $SUDO jq 'del(.hooks)' "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
            $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
            info "Cleaned up legacy hooks from mcp.json"
        fi
    fi
}

#
# UNINSTALL / ROLLBACK
#
if [ "$UNINSTALL" = true ]; then
    info "Uninstalling $YOUR_MCP from $TARGET..."

    # Check if we have a prior installation to roll back to
    if [ -d "$TARGET/$YOUR_MCP/backup/prior" ]; then
        info "Prior installation found - performing rollback..."

        # 1. Move prior installation to temp
        $SUDO mv "$TARGET/$YOUR_MCP/backup/prior" "/tmp/$YOUR_MCP-restore"

        # 2. Remove current installation
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # 3. Restore prior installation
        $SUDO mv "/tmp/$YOUR_MCP-restore" "$TARGET/$YOUR_MCP"
        info "Restored prior installation"

        # 4. Rollback mcp.json entry
        PRIOR_MCP_JSON="$TARGET/$YOUR_MCP/backup/mcp.json-prior"
        CURRENT_MCP_JSON="$TARGET/etc/mcp.json"

        if [ -f "$PRIOR_MCP_JSON" ] && command -v jq &> /dev/null; then
            PRIOR_ENTRY=$($SUDO cat "$PRIOR_MCP_JSON" | jq ".mcpServers[\"$YOUR_MCP\"]")
            if [ "$PRIOR_ENTRY" != "null" ]; then
                $SUDO jq --argjson entry "$PRIOR_ENTRY" \
                   ".mcpServers[\"$YOUR_MCP\"] = \$entry" \
                   "$CURRENT_MCP_JSON" > "/tmp/mcp.json.tmp"
                $SUDO mv "/tmp/mcp.json.tmp" "$CURRENT_MCP_JSON"
                info "Rolled back mcp.json entry"
            fi
        fi

        # 5. Update symlink
        case "$OSTYPE" in
            msys*|cygwin*|win32*) ;;
            *)
                $SUDO rm -f "$TARGET/bin/$YOUR_MCP"
                $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
                ;;
        esac

        info "Rollback complete - restored prior version"
    else
        info "No prior installation - performing full removal..."

        # Remove our directory
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # Remove our symlink
        $SUDO rm -f "$TARGET/bin/$YOUR_MCP"

        # Remove our entry from mcp.json
        if command -v jq &> /dev/null && [ -f "$TARGET/etc/mcp.json" ]; then
            $SUDO jq "del(.mcpServers[\"$YOUR_MCP\"])" \
               "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
            $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
            info "Removed $YOUR_MCP from mcp.json"
        fi

        info "Uninstall complete"
    fi
    exit 0
fi

#
# INSTALL
#

# Check if already up to date (skip-if-identical optimization)
PLATFORM=$(detect_platform)
SOURCE_BIN=$(find "$SCRIPT_DIR/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)
DEST_BIN=$(find "$TARGET/$YOUR_MCP/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)

if [ -n "$SOURCE_BIN" ] && [ -n "$DEST_BIN" ]; then
    SOURCE_MD5=$(md5sum "$SOURCE_BIN" 2>/dev/null | awk '{print $1}')
    DEST_MD5=$(md5sum "$DEST_BIN" 2>/dev/null | awk '{print $1}')
    if [ -n "$SOURCE_MD5" ] && [ "$SOURCE_MD5" = "$DEST_MD5" ]; then
        info "Already up to date (${PLATFORM} binary unchanged)"
        # Still run legacy cleanup in case upgrading from old version
        cleanup_legacy
        exit 0
    fi
fi

echo ""
echo "========================================="
echo -e "${BLUE}Installing $YOUR_MCP v${VERSION}${NC}"
echo "========================================="
echo ""
info "Target: $TARGET"

# 1. Create directory structure if first-time install
if [ ! -d "$TARGET" ]; then
    info "Creating container-tools directory structure..."
    $SUDO mkdir -p "$TARGET/bin"
    $SUDO mkdir -p "$TARGET/etc"
fi
$SUDO mkdir -p "$TARGET/bin"
$SUDO mkdir -p "$TARGET/etc"

# 2. Backup existing installation
PRIOR_TEMP=""
HAS_PRIOR_MCP_JSON=false
if [ -d "$TARGET/$YOUR_MCP" ]; then
    info "Backing up previous installation..."

    # Remember if mcp.json exists (we'll copy it after installing new version)
    [ -f "$TARGET/etc/mcp.json" ] && HAS_PRIOR_MCP_JSON=true

    # Move existing installation to temp
    $SUDO rm -rf "/tmp/$YOUR_MCP-prior"
    $SUDO mv "$TARGET/$YOUR_MCP" "/tmp/$YOUR_MCP-prior"
    PRIOR_TEMP="/tmp/$YOUR_MCP-prior"
fi

# 3. Install MCP directory
info "Installing $YOUR_MCP..."
$SUDO cp -r "$SCRIPT_DIR" "$TARGET/$YOUR_MCP"

# 4. Move prior installation into backup/prior/ and save mcp.json backup
if [ -n "$PRIOR_TEMP" ] && [ -d "$PRIOR_TEMP" ]; then
    $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
    $SUDO rm -rf "$TARGET/$YOUR_MCP/backup/prior"
    $SUDO mv "$PRIOR_TEMP" "$TARGET/$YOUR_MCP/backup/prior"
    info "Prior installation saved to $YOUR_MCP/backup/prior/"

    # Now copy mcp.json backup to the new installation's backup directory
    if [ "$HAS_PRIOR_MCP_JSON" = true ] && [ -f "$TARGET/etc/mcp.json" ]; then
        $SUDO cp "$TARGET/etc/mcp.json" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"
    fi
fi

# 5. Ensure binaries are executable
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP"
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/platforms"/* 2>/dev/null || true

# 6. Create symlink (Linux/macOS only)
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        warn "Windows detected - skipping symlink"
        warn "Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
        info "Created symlink: $TARGET/bin/$YOUR_MCP"
        ;;
esac

# 7. Update mcp.json
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"

if [ ! -f "$MCP_JSON" ]; then
    info "Creating mcp.json..."
    $SUDO tee "$MCP_JSON" > /dev/null << MCPEOF
{
  "mcpServers": {
    "$YOUR_MCP": {
      "command": "$YOUR_COMMAND",
      "args": ["--mode", "stdio"]
    }
  }
}
MCPEOF
elif command -v jq &> /dev/null; then
    info "Merging into mcp.json..."
    $SUDO jq --arg name "$YOUR_MCP" \
       --arg cmd "$YOUR_COMMAND" \
       '.mcpServers[$name] = {"command": $cmd, "args": ["--mode", "stdio"]}' \
       "$MCP_JSON" > "/tmp/mcp.json.tmp"
    $SUDO mv "/tmp/mcp.json.tmp" "$MCP_JSON"
else
    warn "jq not found - please manually add $YOUR_MCP to mcp.json"
fi

# 8. Cleanup legacy hooks-dispatcher pattern (migration from older versions)
cleanup_legacy

# 9. Verify installation
echo ""
info "Verifying installation..."
if [ -x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP" ]; then
    VERSION_OUTPUT=$("$TARGET/$YOUR_MCP/bin/$YOUR_MCP" --version 2>&1 | head -1)
    info "Installed: $VERSION_OUTPUT"
else
    error "Installation verification failed - launcher not executable"
fi

# Summary
echo ""
echo "========================================="
echo -e "${GREEN}Installation complete!${NC}"
echo "========================================="
echo ""
echo "Installed to: $TARGET/$YOUR_MCP/"
echo ""
echo -e "${BLUE}Next steps:${NC}"
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        echo "  1. Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        echo "  1. Add $TARGET/bin to your PATH (if not already)"
        ;;
esac
echo "  2. Restart Claude Code to load the new MCP"
echo ""
echo -e "${BLUE}Test:${NC}"
echo "  $TARGET/$YOUR_MCP/bin/$YOUR_MCP --version"
echo ""
echo -e "${BLUE}Rollback (if needed):${NC}"
echo "  ./install.sh --uninstall"
echo ""
INSTALL

# Replace version placeholder in install.sh
sed -i.bak "s/__VERSION__/${VERSION}/g" "${PACKAGE_DIR}/${MCP_NAME}/install.sh"
rm -f "${PACKAGE_DIR}/${MCP_NAME}/install.sh.bak"
chmod +x "${PACKAGE_DIR}/${MCP_NAME}/install.sh"

echo "Creating VERSION_MANIFEST.txt..."

# Create VERSION_MANIFEST.txt
cat > "${PACKAGE_DIR}/${MCP_NAME}/VERSION_MANIFEST.txt" << MANIFEST
Package: ${MCP_NAME}
Version: ${VERSION}
Git Commit: ${GIT_COMMIT}
Build Date: ${BUILD_DATE}
Build Host: $(hostname 2>/dev/null || echo "unknown")

Package Type: container-tools
Install Location: /opt/container-tools/${MCP_NAME}/

Package Structure:
  container-tools-${MCP_NAME}-v${VERSION}/
  └── ${MCP_NAME}/
      ├── bin/
      │   ├── ${MCP_NAME}           (universal launcher)
      │   └── platforms/            (platform binaries)
      ├── CHANGELOG.md
      ├── LICENSE
      ├── README.md
      ├── VERSION_MANIFEST.txt
      └── install.sh

Runtime Directories (created during installation/updates):
  ${MCP_NAME}/backup/
  ├── mcp.json-prior            (mcp.json before modification)
  └── prior/                    (prior installation for rollback)

Platforms:
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64
  - windows-amd64
  - windows-arm64

Installer Features:
  - Skip-if-identical (MD5 comparison)
  - Rollback on --uninstall (restores backup/prior/)
  - mcp.json merge (preserves other MCPs)
  - Legacy hooks-dispatcher cleanup (migration from v1.x)
  - All backups inside ${MCP_NAME}/backup/
MANIFEST

echo ""
echo "Creating tarball..."

# Create tarball - versioned directory as root for side-by-side installs
cd "${BUILD_DIR}"
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"

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
echo "  cd ${PACKAGE_NAME}/${MCP_NAME}"
echo "  sudo ./install.sh"
echo ""
echo "To install to custom location:"
echo "  sudo ./install.sh --target /custom/path"
echo ""
echo "To uninstall/rollback:"
echo "  sudo ./install.sh --uninstall"
echo ""
