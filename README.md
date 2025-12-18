# P2KB MCP Server

[![CI](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml)
[![Release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

MCP (Model Context Protocol) server providing Claude AI with structured access to the [Propeller 2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base) and [OBEX (Parallax Object Exchange)](https://obex.parallax.com/).

## What It Does

Gives Claude direct access to comprehensive Propeller 2 documentation and community code:

- **PASM2 Instructions** - Assembly language reference with syntax, encoding, and examples
- **Spin2 Methods** - High-level language built-in methods
- **Architecture** - COG, HUB, Smart Pins, and hardware documentation
- **Guides** - Quick reference and getting started guides
- **OBEX Objects** - ~113 community code objects for I2C, SPI, LEDs, motors, and more

## Quick Start

### Option 1: Container-Tools Package (Recommended)

For installations in `/opt/container-tools/` (works alongside other MCPs):

```bash
# Download and extract
tar -xzf container-tools-p2kb-mcp-vX.X.X.tar.gz
cd p2kb-mcp

# Install (or update existing installation)
sudo ./install.sh

# Verify
/opt/container-tools/bin/p2kb-mcp --version

# Rollback or uninstall
sudo ./install.sh --uninstall
```

### Option 2: Standalone Package

For single-platform installations:

```bash
# Download and extract
tar -xzf p2kb-mcp-vX.X.X-linux-amd64.tar.gz

# Move to /opt (or your preferred location)
sudo mv p2kb-mcp /opt/

# Verify
/opt/p2kb-mcp/bin/p2kb-mcp --version
```

Add to your MCP configuration:

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

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

## Available Tools (v1.1.0+)

The API is intentionally minimal (6 tools) to reduce cognitive load for Claude.

### Documentation Access

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content using natural language ("mov instruction") or exact key (`p2kbPasm2Mov`). Returns content with related items. |
| `p2kb_find` | Explore documentation: list categories, search by term, or browse a category. |

### OBEX (Community Code)

| Tool | Description |
|------|-------------|
| `p2kb_obex_get` | Get OBEX object by search ("i2c sensor") or ID ("2811"). Returns download URL and instructions. |
| `p2kb_obex_find` | Explore OBEX: list categories, search objects, or browse by category/author. |

### System

| Tool | Description |
|------|-------------|
| `p2kb_version` | Server version and index/cache status (for debugging). |
| `p2kb_refresh` | Force refresh of index and invalidate stale cache entries. |

## Usage Examples

### Natural Language Queries

Claude can use natural language to find documentation:

```
p2kb_get("mov instruction")     → Returns PASM2 MOV documentation
p2kb_get("spin2 pinwrite")      → Returns Spin2 pinwrite method
p2kb_get("cog memory")          → Returns COG architecture docs
```

### OBEX Code Objects

Find and download community code:

```
p2kb_obex_get("led driver")     → Finds WS2812B, NeoPixel objects
p2kb_obex_get("2811")           → Gets specific object by ID
p2kb_obex_find(category="drivers") → Lists all driver objects
```

## Key Prefixes

| Prefix | Content |
|--------|---------|
| `p2kbPasm2*` | PASM2 assembly instructions |
| `p2kbSpin2*` | Spin2 methods |
| `p2kbArch*` | Architecture docs |
| `p2kbGuide*` | Guides |
| `p2kbHw*` | Hardware specs |

## Documentation

- [Installation Guide](INSTALL.md) - Detailed setup instructions
- [API Reference](DOCs/API.md) - Tool specifications
- [Testing & Coverage](DOCs/TESTING.md) - Test strategy and metrics
- [Changelog](CHANGELOG.md) - Version history

## Development

```bash
make build    # Build for current platform
make test     # Run tests
make lint     # Run linters
```

See [TESTING.md](DOCs/TESTING.md) for coverage details.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related

- [P2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base) - Documentation source
- [OBEX](https://obex.parallax.com/) - Parallax Object Exchange
- [Propeller 2](https://www.parallax.com/propeller-2/) - The microcontroller
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification

---

*Iron Sheep Productions, LLC*
