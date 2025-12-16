# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2025-12-16

### Added

- **Container Tools Integration Guide Compliance**:
  - Full implementation of the Container Tools MCP Integration Guide
  - hooks.d dispatcher pattern for multi-MCP hook coexistence
  - Shared hooks-dispatcher.sh for app-start, compact-start, compact-end hooks
  - Per-MCP hook scripts in `/opt/container-tools/etc/hooks.d/`

- **Enhanced Installer**:
  - `--target DIR` option for custom installation locations
  - `--uninstall` option with intelligent rollback support
  - `--help` option for usage information
  - Skip-if-identical optimization (MD5 checksum comparison)
  - Automatic backup of previous installation to `-prior` suffix
  - Backup of mcp.json before modification
  - Proper mcp.json merging (preserves other MCPs)

- **Rollback Support**:
  - Uninstall restores prior version if available
  - mcp.json entry rollback (merges prior entry, preserves others)
  - Symlink restoration on rollback

- **Improved Launcher**:
  - Symlink resolution for correct binary path detection
  - Works correctly when invoked via `/opt/container-tools/bin/` symlink

### Changed

- Package structure now includes `etc/` directory with:
  - `hooks-dispatcher.sh` - Universal hook dispatcher
  - `hooks.d/app-start/p2kb-mcp.sh` - App start hook
- Backup strategy changed from timestamp suffix to single `-prior` suffix
- mcp.json now includes hooks configuration pointing to dispatcher

### Fixed

- Universal launcher now correctly resolves symlinks to find platform binaries

## [1.1.0] - 2025-12-14

### Breaking Changes

- **API Redesign**: Reduced from 11 tools to 6 tools for better Claude usability
  - Removed: `p2kb_search`, `p2kb_browse`, `p2kb_categories`, `p2kb_batch_get`, `p2kb_info`, `p2kb_stats`, `p2kb_related`, `p2kb_help`, `p2kb_cached`, `p2kb_index_status`
  - Added: `p2kb_find`, `p2kb_obex_get`, `p2kb_obex_find`
  - Changed: `p2kb_get` now accepts natural language queries
  - Changed: `p2kb_version` now includes comprehensive cache/index status

### Added

- **Natural Language Query Support** in `p2kb_get`:
  - Accept queries like "mov instruction" instead of requiring exact keys
  - Token-based scoring matches queries to CamelCase keys
  - Returns suggestions when multiple matches found

- **OBEX (Parallax Object Exchange) Support**:
  - `p2kb_obex_get`: Fetch object by search term or numeric ID
  - `p2kb_obex_find`: Browse categories, search objects, filter by author
  - ~113 community code objects for I2C, SPI, LEDs, motors, etc.
  - Search term expansion (e.g., "i2c" matches iic, twi, two-wire)
  - Download URLs and installation instructions included

- **Smart Cache Invalidation**:
  - Cache-busting headers on index fetch
  - Mtime-based comparison for KB entries
  - TTL-based expiration for OBEX objects (24 hours)
  - `p2kb_refresh` invalidates stale entries based on index timestamps

- **Package Structure Redesign**:
  - Standalone packages use opt-style layout (`p2kb-mcp/bin/p2kb-mcp`)
  - Container-tools package installs to `/opt/container-tools/p2kb-mcp/`
  - All packages include README.md, CHANGELOG.md, and LICENSE

- **Platform-specific cache locations**:
  - Container-tools: `/opt/container-tools/var/cache/p2kb-mcp/`
  - Standalone Linux/macOS: `.cache/` directory next to binary
  - Windows: `%LOCALAPPDATA%\p2kb-mcp\cache\`

- New `internal/obex` package for OBEX object management
- New `internal/paths` package for centralized path resolution
- Standalone package builds for all 6 platforms
- `make standalone` and `make packages` build targets

### Changed

- `p2kb_find` replaces `p2kb_search`, `p2kb_browse`, and `p2kb_categories`
- `p2kb_version` now includes index and cache statistics
- `p2kb_refresh` now performs smart cache invalidation
- Improved error messages with helpful hints

### Migration Guide

See [API.md](DOCs/API.md#migration-from-v10x) for migration details from v1.0.x

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

[Unreleased]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/tag/v1.0.0
