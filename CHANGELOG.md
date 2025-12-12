# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2025-12-12

### Added

- Initial release of P2KB MCP Server
- **11 MCP tools** for accessing the Propeller 2 Knowledge Base:
  - **Core Tools**: `p2kb_get`, `p2kb_search`, `p2kb_browse`, `p2kb_categories`, `p2kb_version`
  - **Enhanced Tools**: `p2kb_batch_get`, `p2kb_refresh`, `p2kb_info`, `p2kb_stats`, `p2kb_related`, `p2kb_help`

- **6-platform binary releases**:
  - Linux AMD64
  - Linux ARM64
  - macOS AMD64 (Intel)
  - macOS ARM64 (Apple Silicon)
  - Windows AMD64
  - Windows ARM64

- **Container deployment package** (`container-tools-*.tar.gz`) for easy integration with existing container setups

- **macOS code signing and notarization** support (when signing is enabled)

- Automatic index caching with 24-hour TTL
- Content caching with mtime-based invalidation
- Metadata filtering for reduced token usage
- Helpful error messages with key suggestions

### Technical Details

- Built with Go 1.22+
- MCP protocol version 2024-11-05
- JSON-RPC 2.0 over stdio
- No external dependencies required at runtime

### Test Coverage

**Overall: 54.8%**

| Package | Coverage |
|---------|----------|
| `internal/filter` | 100.0% |
| `internal/fetch` | 72.1% |
| `internal/index` | 67.3% |
| `internal/cache` | 55.6% |
| `internal/server` | 43.2% |

See [TESTING.md](DOCs/TESTING.md) for detailed coverage information.

### Data Source

All documentation fetched from the [P2 Knowledge Base](https://github.com/ironsheep/P2-Knowledge-Base):
- 970+ entries across 47 categories
- PASM2 instructions, Spin2 methods, architecture documentation
- Smart pin configurations, hardware specifications

[Unreleased]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/tag/v1.0.0
