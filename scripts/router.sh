#!/usr/bin/env bash
#
# router.sh - Universal binary router for p2kb-mcp
#
# This script detects the current OS and architecture and executes
# the appropriate platform-specific binary.
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORMS_DIR="${SCRIPT_DIR}/platforms"
MCP_NAME="p2kb-mcp"

# Read version from adjacent VERSION file or use placeholder
if [ -f "${SCRIPT_DIR}/../VERSION" ]; then
    VERSION=$(cat "${SCRIPT_DIR}/../VERSION" | tr -d '\n')
else
    VERSION="dev"
fi

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
        *)
            echo "ERROR: Unsupported architecture: $(uname -m)" >&2
            exit 1
            ;;
    esac
}

# OS detection
detect_os() {
    case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
        darwin) echo "darwin" ;;
        linux) echo "linux" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *)
            echo "ERROR: Unsupported OS: $(uname -s)" >&2
            exit 1
            ;;
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

    local binary="${PLATFORMS_DIR}/${MCP_NAME}-v${VERSION}-${os}-${arch}${suffix}"

    debug "OS: ${os}, Arch: ${arch}, Container: $(is_container && echo yes || echo no)"
    debug "Selected binary: ${binary}"

    if [ ! -f "$binary" ]; then
        echo "ERROR: Binary not found: ${binary}" >&2
        echo "Available binaries:" >&2
        ls -1 "${PLATFORMS_DIR}/" >&2 2>/dev/null || echo "  (platforms directory not found)" >&2
        exit 1
    fi

    echo "$binary"
}

# Execute the appropriate binary
exec "$(select_binary)" "$@"
