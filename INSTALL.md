# Installation Guide

## Quick Start

### Container-Tools Package (Recommended)

The container-tools package installs P2KB MCP alongside other MCPs in `/opt/container-tools/`:

```bash
# Download the latest release
curl -LO https://github.com/ironsheep/p2kb-mcp/releases/latest/download/p2kb-mcp-v1.0.0.tar.gz

# Extract
tar -xzf p2kb-mcp-v1.0.0.tar.gz
cd p2kb-mcp-v1.0.0

# Install (requires sudo for /opt)
./install.sh

# Verify installation
/opt/container-tools/opt/p2kb-mcp/bin/p2kb-mcp --version
```

The installer automatically:
- Creates `/opt/container-tools/` directory structure if needed
- Installs all platform binaries
- Creates/updates `/opt/container-tools/etc/mcp.json`

### Standalone Binary

Download the appropriate binary for your platform:

| Platform | Binary |
|----------|--------|
| Linux AMD64 | `p2kb-mcp-v1.0.0-linux-amd64` |
| Linux ARM64 | `p2kb-mcp-v1.0.0-linux-arm64` |
| macOS Intel | `p2kb-mcp-v1.0.0-darwin-amd64` |
| macOS Apple Silicon | `p2kb-mcp-v1.0.0-darwin-arm64` |
| Windows AMD64 | `p2kb-mcp-v1.0.0-windows-amd64.exe` |
| Windows ARM64 | `p2kb-mcp-v1.0.0-windows-arm64.exe` |

```bash
# Linux/macOS
chmod +x p2kb-mcp-v1.0.0-linux-amd64
mv p2kb-mcp-v1.0.0-linux-amd64 /usr/local/bin/p2kb-mcp

# Verify
p2kb-mcp --version
```

### Build from Source

```bash
git clone https://github.com/ironsheep/p2kb-mcp.git
cd p2kb-mcp
make build
sudo make install
```

## MCP Client Configuration

### Claude Desktop

Add to `~/.config/claude/claude_desktop_config.json` (Linux/macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/container-tools/opt/p2kb-mcp/bin/p2kb-mcp",
      "args": []
    }
  }
}
```

### Claude Code CLI

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/usr/local/bin/p2kb-mcp",
      "args": []
    }
  }
}
```

## Verification

Test that the MCP server is working:

```bash
# Check version
p2kb-mcp --version

# Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | p2kb-mcp
```

You should see a JSON response listing all available tools.

## Cache Location

P2KB MCP caches the index and YAML files locally:

- **Default location**: `~/.p2kb-mcp/`
- **Override**: Set `P2KB_CACHE_DIR` environment variable

Cache structure:
```
~/.p2kb-mcp/
├── index/
│   ├── p2kb-index.json      # Decompressed index
│   └── p2kb-index.meta      # Index metadata
├── cache/
│   ├── p2kbPasm2Mov.yaml    # Cached content files
│   └── ...
└── mcp.log                  # Debug log (if enabled)
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
P2KB_LOG_LEVEL=debug p2kb-mcp
```

### Cache Issues

Force refresh the cache:

```bash
# Via MCP tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"p2kb_refresh","arguments":{"invalidate_cache":true}}}' | p2kb-mcp

# Or delete cache manually
rm -rf ~/.p2kb-mcp/
```

## Uninstallation

### Container-Tools Installation

```bash
sudo rm -rf /opt/container-tools/opt/p2kb-mcp

# Edit /opt/container-tools/etc/mcp.json to remove the p2kb-mcp entry
```

### Standalone Installation

```bash
sudo rm /usr/local/bin/p2kb-mcp
rm -rf ~/.p2kb-mcp/
```
