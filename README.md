# P2KB MCP Server

[![CI](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/ci.yml)
[![Release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml/badge.svg)](https://github.com/ironsheep/P2-Knowledge-Base-MCP/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Coverage: 54.8%](https://img.shields.io/badge/Coverage-54.8%25-yellow.svg)](DOCs/TESTING.md)

MCP (Model Context Protocol) server providing Claude AI with structured access to the [Propeller 2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base).

## What It Does

Gives Claude direct access to comprehensive Propeller 2 documentation:

- **PASM2 Instructions** - Assembly language reference with syntax, encoding, and examples
- **Spin2 Methods** - High-level language built-in methods
- **Architecture** - COG, HUB, Smart Pins, and hardware documentation
- **Guides** - Quick reference and getting started guides

## Quick Start

```bash
# Download and install (see INSTALL.md for all options)
curl -LO https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/latest/download/p2kb-mcp-linux-amd64
chmod +x p2kb-mcp-linux-amd64
./p2kb-mcp-linux-amd64 --version
```

Add to your MCP configuration:

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

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

## Available Tools

| Tool | Description |
|------|-------------|
| `p2kb_get` | Fetch content by key (e.g., `p2kbPasm2Mov`) |
| `p2kb_search` | Search for keys matching a term |
| `p2kb_browse` | List all keys in a category |
| `p2kb_categories` | List all available categories |
| `p2kb_batch_get` | Fetch multiple keys in one call |
| `p2kb_related` | Get related instructions for a key |
| `p2kb_stats` | Knowledge base statistics |
| `p2kb_refresh` | Force refresh of index and cache |
| `p2kb_info` | Check if a key exists |
| `p2kb_version` | Get MCP server version |
| `p2kb_help` | Usage information |

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
- [Propeller 2](https://www.parallax.com/propeller-2/) - The microcontroller
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification

---

*Iron Sheep Productions, LLC*
