# P2KB MCP Server

[![CI](https://github.com/ironsheep/p2kb-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/ironsheep/p2kb-mcp/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

MCP (Model Context Protocol) server providing Claude AI access to the [Propeller 2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base).

## Overview

P2KB MCP gives Claude structured access to comprehensive Propeller 2 documentation:

- **PASM2 Instructions** - Assembly language reference with syntax, encoding, and examples
- **Spin2 Methods** - High-level language built-in methods
- **Architecture** - COG, HUB, Smart Pins, and hardware documentation
- **Guides** - Quick reference and getting started guides

## Features

- **11 MCP tools** for searching, browsing, and fetching P2 documentation
- **Automatic caching** - Index and content cached locally for performance
- **Metadata filtering** - Removes internal tracking data to save tokens
- **Cross-platform** - Binaries for Linux, macOS, and Windows (AMD64 and ARM64)
- **Container-ready** - Easy installation into existing container setups

## Installation

### Option 1: Container-Tools Package (Recommended)

Download the latest release and run the installer:

```bash
tar -xzf p2kb-mcp-v1.0.0.tar.gz
cd p2kb-mcp-v1.0.0
./install.sh
```

This installs to `/opt/container-tools/` and configures `mcp.json` automatically.

### Option 2: Standalone Binary

Download the appropriate binary for your platform from the [Releases](https://github.com/ironsheep/p2kb-mcp/releases) page.

```bash
# Linux/macOS
chmod +x p2kb-mcp-v1.0.0-linux-amd64
./p2kb-mcp-v1.0.0-linux-amd64 --version
```

### Option 3: Build from Source

```bash
git clone https://github.com/ironsheep/p2kb-mcp.git
cd p2kb-mcp
make build
./p2kb-mcp --version
```

## Configuration

Add to your Claude MCP configuration:

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/path/to/p2kb-mcp",
      "args": []
    }
  }
}
```

## Available Tools

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content by key (e.g., `p2kbPasm2Mov`) |
| `p2kb_search` | Search for keys matching a term |
| `p2kb_browse` | List all keys in a category |
| `p2kb_categories` | List all available categories |
| `p2kb_version` | Get MCP server version |
| `p2kb_batch_get` | Fetch multiple keys in one call |
| `p2kb_refresh` | Force refresh of index and cache |
| `p2kb_info` | Check if a key exists and its categories |
| `p2kb_stats` | Knowledge base statistics |
| `p2kb_related` | Get related instructions for a key |
| `p2kb_help` | Usage information and key prefixes |

## Usage Examples

```
# Search for MOV instruction
p2kb_search("mov")

# Get specific instruction documentation
p2kb_get("p2kbPasm2Mov")

# Browse all branch instructions
p2kb_browse("pasm2_branch")

# Get multiple instructions at once
p2kb_batch_get(["p2kbPasm2Mov", "p2kbPasm2Add", "p2kbPasm2Sub"])
```

## Key Naming Convention

| Prefix | Content Type |
|--------|--------------|
| `p2kbPasm2*` | PASM2 assembly instructions |
| `p2kbSpin2*` | Spin2 methods |
| `p2kbArch*` | Architecture documentation |
| `p2kbGuide*` | Guides and quick references |
| `p2kbHw*` | Hardware specifications |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `P2KB_CACHE_DIR` | `~/.p2kb-mcp` | Cache directory location |
| `P2KB_INDEX_TTL` | `86400` | Index TTL in seconds (24 hours) |
| `P2KB_LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |

## Development

### Prerequisites

- Go 1.22+
- Make

### Building

```bash
make build        # Build for current platform
make test         # Run tests
make lint         # Run linters
make dist         # Build for all platforms
```

### Testing

```bash
make test-short   # Fast tests (no network)
make test-live    # Live GitHub integration tests
make test-coverage # Generate coverage report
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [P2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base) - The documentation source
- [Propeller 2](https://www.parallax.com/propeller-2/) - The Parallax Propeller 2 microcontroller

## Author

Iron Sheep Productions, LLC

---

*Built with the [Model Context Protocol](https://modelcontextprotocol.io/)*
