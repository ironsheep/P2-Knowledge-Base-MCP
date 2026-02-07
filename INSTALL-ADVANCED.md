# Advanced Installation: Container-Tools

This guide covers the **container-tools** installation method, designed for users who manage multiple MCP servers and want a unified framework for installing, updating, and organizing them.

> **Looking for the standard install?** See [INSTALL.md](INSTALL.md) for simple platform-specific instructions.

## Overview

The container-tools package installs P2KB MCP into a shared `/opt/container-tools/` directory structure alongside other MCP servers. It provides:

- Unified directory layout for all MCPs
- Automatic backup and rollback on updates
- Hook-based lifecycle management for Claude Code
- Shared configuration (single `mcp.json` for all MCPs)

---

## Installation

1. Go to [Releases](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases)
2. Download the container-tools package: `container-tools-p2kb-mcp-vX.X.X.tar.gz`
3. Extract and install:

```bash
# Extract the package
tar -xzf container-tools-p2kb-mcp-v*.tar.gz
cd p2kb-mcp

# Install (requires sudo for /opt)
sudo ./install.sh

# Verify installation
/opt/container-tools/bin/p2kb-mcp --version
```

### Installer Options

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

---

## What the Installer Does

The installer automatically:

1. **Creates directory structure** (if first-time install):
   - `/opt/container-tools/bin/` — Symlinks to MCP launchers
   - `/opt/container-tools/etc/` — Shared configuration

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
   - Backs up previous installation to `backup/prior/`
   - Backs up mcp.json to `backup/mcp.json-prior`

---

## Installation Layout

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
└── p2kb-mcp/
    ├── bin/
    │   ├── p2kb-mcp                # Universal launcher
    │   └── platforms/              # Platform binaries
    ├── backup/                     # Created during updates
    │   ├── mcp.json-prior          # Backup of mcp.json
    │   └── prior/                  # Prior installation (for rollback)
    ├── README.md
    ├── CHANGELOG.md
    ├── LICENSE
    └── VERSION_MANIFEST.txt
```

---

## MCP Client Configuration

After installing the binary, connect it to your AI tool. The container-tools binary path is `/opt/container-tools/bin/p2kb-mcp`.

### Claude Code CLI

```bash
claude mcp add -s user p2kb-mcp -- /opt/container-tools/bin/p2kb-mcp --mode stdio
```

> **Note:** The container-tools installer may configure this automatically via the hooks system.

### Claude Desktop

Add to your Claude Desktop config file:

| Platform | Config File Location |
|----------|---------------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/claude/claude_desktop_config.json` |

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/container-tools/bin/p2kb-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

### Cursor

Add to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/container-tools/bin/p2kb-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

Or use Cursor Settings → MCP → Add new global MCP server.

### Codex

```bash
codex mcp add p2kb-mcp -- /opt/container-tools/bin/p2kb-mcp --mode stdio
```

Or add to `~/.codex/config.toml`:

```toml
[mcp_servers.p2kb-mcp]
command = "/opt/container-tools/bin/p2kb-mcp"
args = ["--mode", "stdio"]
```

---

## Cache Location

Container-tools installations use a dedicated cache directory:

- **Location:** `/opt/container-tools/var/cache/p2kb-mcp/`

Override with the `P2KB_CACHE_DIR` environment variable.

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

---

## Verification

```bash
# Check version
/opt/container-tools/bin/p2kb-mcp --version

# Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | /opt/container-tools/bin/p2kb-mcp
```

You should see a JSON response listing all available tools.

---

## Updating

The installer handles updates automatically:

```bash
# Download new version
tar -xzf container-tools-p2kb-mcp-vX.X.X.tar.gz
cd p2kb-mcp

# Install (automatically backs up current version)
sudo ./install.sh

# If something goes wrong, rollback
sudo ./install.sh --uninstall
```

The installer:

1. Compares binary MD5 checksums — skips if identical
2. Backs up mcp.json to existing installation's `backup/mcp.json-prior`
3. Moves current installation to temp
4. Installs new version
5. Moves prior installation into new version's `backup/prior/`
6. Preserves other MCPs in mcp.json

---

## Uninstalling

### Using the Installer (Recommended)

```bash
# Navigate to extracted package directory, or download again
cd p2kb-mcp
sudo ./install.sh --uninstall
```

If a prior version exists (from a previous update), uninstall will **rollback** to that version. If no prior exists, it performs a **full removal**.

### Manual Removal

```bash
# Remove the MCP (includes backup/prior/ if exists)
sudo rm -rf /opt/container-tools/p2kb-mcp
sudo rm -f /opt/container-tools/bin/p2kb-mcp
sudo rm -f /opt/container-tools/etc/hooks.d/*/p2kb-mcp.sh

# Edit /opt/container-tools/etc/mcp.json to remove the p2kb-mcp entry
```

---

## Troubleshooting

### Debug Logging

```bash
P2KB_LOG_LEVEL=debug /opt/container-tools/bin/p2kb-mcp
```

### Force Cache Refresh

```bash
# Via MCP tool
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"p2kb_refresh","arguments":{"invalidate_cache":true}}}' | /opt/container-tools/bin/p2kb-mcp

# Or delete cache manually
sudo rm -rf /opt/container-tools/var/cache/p2kb-mcp/*
```

### Network Issues

If behind a proxy:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
```
