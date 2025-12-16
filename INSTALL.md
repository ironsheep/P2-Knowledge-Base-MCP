# Installation Guide

## Quick Start

### Container-Tools Package (Recommended)

The container-tools package installs P2KB MCP alongside other MCPs in `/opt/container-tools/`:

1. Go to [Releases](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases)
2. Download the container-tools package: `p2kb-mcp-vX.X.X-container-tools.tar.gz`
3. Extract and install:

```bash
# Extract the package
tar -xzf p2kb-mcp-v*-container-tools.tar.gz
cd p2kb-mcp

# Install (requires sudo for /opt)
sudo ./install.sh

# Verify installation
/opt/container-tools/bin/p2kb-mcp --version
```

### Installer Options

The installer supports several options:

```bash
# Default install to /opt/container-tools
sudo ./install.sh

# Install to custom location
./install.sh --target ~/my-container-tools

# Uninstall (or rollback to prior version if available)
sudo ./install.sh --uninstall

# Show help
./install.sh --help
```

### What the Installer Does

The installer automatically:

1. **Creates directory structure** (if first-time install):
   - `/opt/container-tools/bin/` - Symlinks to MCP launchers
   - `/opt/container-tools/etc/` - Shared configuration
   - `/opt/container-tools/var/cache/p2kb-mcp/` - Cache directory

2. **Installs the MCP**:
   - Copies all platform binaries with universal launcher
   - Creates symlink at `/opt/container-tools/bin/p2kb-mcp`

3. **Sets up hooks** (for Claude Code integration):
   - Installs `hooks-dispatcher.sh` (if not present)
   - Installs `hooks.d/app-start/p2kb-mcp.sh`

4. **Updates mcp.json**:
   - Merges entry (preserves other MCPs)
   - Configures hooks dispatcher

5. **Handles updates intelligently**:
   - Skips install if binary is identical (MD5 comparison)
   - Backs up previous installation to `-prior` suffix
   - Backs up mcp.json before modification

### Installation Layout

```
/opt/container-tools/
├── bin/
│   └── p2kb-mcp -> ../p2kb-mcp/bin/p2kb-mcp
├── etc/
│   ├── mcp.json                    # Shared MCP configuration
│   ├── hooks-dispatcher.sh         # Hook dispatcher script
│   └── hooks.d/
│       ├── app-start/
│       │   └── p2kb-mcp.sh         # App start hook
│       ├── compact-start/
│       └── compact-end/
├── var/
│   └── cache/
│       └── p2kb-mcp/               # Cache directory
└── p2kb-mcp/
    ├── bin/
    │   ├── p2kb-mcp                # Universal launcher
    │   └── platforms/              # Platform binaries
    ├── backup/
    │   └── mcp.json-prior          # Backup of mcp.json
    ├── README.md
    ├── CHANGELOG.md
    └── LICENSE
```

### Standalone Package

For single-platform installations without container-tools:

1. Go to [Releases](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases)
2. Download the appropriate package for your platform:

| Platform | File Pattern |
|----------|--------------|
| Linux AMD64 | `p2kb-mcp-vX.X.X-linux-amd64.tar.gz` |
| Linux ARM64 | `p2kb-mcp-vX.X.X-linux-arm64.tar.gz` |
| macOS Intel | `p2kb-mcp-vX.X.X-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `p2kb-mcp-vX.X.X-darwin-arm64.tar.gz` |
| Windows AMD64 | `p2kb-mcp-vX.X.X-windows-amd64.zip` |
| Windows ARM64 | `p2kb-mcp-vX.X.X-windows-arm64.zip` |

3. Extract and install:

```bash
# Linux/macOS
tar -xzf p2kb-mcp-v*-linux-amd64.tar.gz
sudo mv p2kb-mcp /opt/

# Verify
/opt/p2kb-mcp/bin/p2kb-mcp --version
```

```powershell
# Windows - extract zip to desired location
# e.g., C:\Program Files\p2kb-mcp\
```

**Standalone package layout:**
```
p2kb-mcp/
├── bin/
│   └── p2kb-mcp[.exe]
├── .cache/              # Created at runtime (hidden folder)
├── README.md
├── CHANGELOG.md
└── LICENSE
```

### Build from Source

```bash
git clone https://github.com/ironsheep/P2-Knowledge-Base-MCP.git
cd P2-Knowledge-Base-MCP
make build
sudo make install
```

## MCP Client Configuration

### Claude Desktop

Add to `~/.config/claude/claude_desktop_config.json` (Linux/macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

**Container-tools installation:**
```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/container-tools/bin/p2kb-mcp",
      "args": []
    }
  }
}
```

**Standalone installation:**
```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/p2kb-mcp/bin/p2kb-mcp",
      "args": []
    }
  }
}
```

### Claude Code CLI

Add to your MCP configuration using the path where you installed the binary.

## Verification

Test that the MCP server is working:

```bash
# Check version
/opt/container-tools/bin/p2kb-mcp --version

# Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | /opt/container-tools/bin/p2kb-mcp
```

You should see a JSON response listing all available tools.

## Cache Location

P2KB MCP caches the index and YAML files locally. Cache location depends on installation type:

### Container-Tools Installation
- **Location**: `/opt/container-tools/var/cache/p2kb-mcp/`

### Standalone Installation (Linux/macOS)
- **Location**: `.cache/` directory next to the binary (e.g., `/opt/p2kb-mcp/.cache/`)

### Standalone Installation (Windows)
- **Location**: `%LOCALAPPDATA%\p2kb-mcp\cache\`
  (e.g., `C:\Users\{username}\AppData\Local\p2kb-mcp\cache\`)

### Override
Set the `P2KB_CACHE_DIR` environment variable to use a custom location.

**Cache structure:**
```
{cache_dir}/
├── index/
│   ├── p2kb-index.json      # Decompressed index
│   └── p2kb-index.meta      # Index metadata
└── cache/
    ├── p2kbPasm2Mov.yaml    # Cached content files
    └── ...
```

## Troubleshooting

### macOS Gatekeeper Warning

If you see "cannot be opened because the developer cannot be verified":

```bash
xattr -d com.apple.quarantine /path/to/p2kb-mcp
```

Or use signed binaries from releases with macOS signing enabled.

### Network Issues

P2KB MCP fetches data from GitHub. If you're behind a proxy:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
```

### Debug Logging

Enable debug output:

```bash
P2KB_LOG_LEVEL=debug /opt/container-tools/bin/p2kb-mcp
```

### Cache Issues

Force refresh the cache:

```bash
# Via MCP tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"p2kb_refresh","arguments":{"invalidate_cache":true}}}' | /opt/container-tools/bin/p2kb-mcp

# Or delete cache manually
# Container-tools:
sudo rm -rf /opt/container-tools/var/cache/p2kb-mcp/*

# Standalone:
rm -rf /opt/p2kb-mcp/.cache/*
```

## Uninstallation

### Container-Tools Installation

**Using the installer (recommended):**
```bash
# Navigate to extracted package directory, or download again
cd p2kb-mcp
sudo ./install.sh --uninstall
```

If a prior version exists (from a previous update), uninstall will **rollback** to that version.
If no prior exists, it performs a **full removal**.

**Manual removal:**
```bash
# Remove the MCP
sudo rm -rf /opt/container-tools/p2kb-mcp
sudo rm -rf /opt/container-tools/p2kb-mcp-prior  # if exists
sudo rm -f /opt/container-tools/bin/p2kb-mcp
sudo rm -rf /opt/container-tools/var/cache/p2kb-mcp
sudo rm -f /opt/container-tools/etc/hooks.d/*/p2kb-mcp.sh

# Edit /opt/container-tools/etc/mcp.json to remove the p2kb-mcp entry
```

### Standalone Installation

```bash
# Linux/macOS
sudo rm -rf /opt/p2kb-mcp

# Windows
# Delete the installation folder
# Delete %LOCALAPPDATA%\p2kb-mcp
```

## Updating

### Container-Tools Installation

The installer handles updates automatically:

```bash
# Download new version
tar -xzf p2kb-mcp-vX.X.X-container-tools.tar.gz
cd p2kb-mcp

# Install (automatically backs up current version)
sudo ./install.sh

# If something goes wrong, rollback
sudo ./install.sh --uninstall
```

The installer:
1. Compares binary MD5 checksums - skips if identical
2. Backs up current installation to `p2kb-mcp-prior/`
3. Backs up mcp.json to `p2kb-mcp/backup/mcp.json-prior`
4. Installs new version
5. Preserves other MCPs in mcp.json

### Standalone Installation

Replace the installation directory with the new version.
