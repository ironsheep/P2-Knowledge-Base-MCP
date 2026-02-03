# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.3] - 2026-02-03

### Added

- **Refresh-on-error**: Automatic index refresh when key lookup fails
  - If a key is not found and the 5-minute cooldown has passed, automatically refreshes the index and retries
  - Helps when the remote index is updated but local cache hasn't expired yet
  - Cooldown prevents refresh storms when keys are genuinely not found
  - Works for both KB index (`ResolveKey`) and OBEX index (`GetObject`)

### Changed

- **Case-insensitive lookups**: All search and lookup operations are now fully case-insensitive
  - Canonical key lookups: `p2kbpasm2mov`, `P2KBPASM2MOV`, `p2kbPasm2Mov` all resolve to the same key
  - Category lookups: `PASM2_DATA`, `pasm2_data`, `Pasm2_Data` all work
  - OBEX object ID prefix: `OB2811`, `ob2811`, `Ob2811`, `oB2811` all normalize correctly
  - OBEX object existence checks are now case-insensitive

## [1.3.2] - 2026-01-17

### Changed

- **Friendlier error handling**: "Not found" responses are now success results instead of JSON-RPC errors
  - `no_matches` - query worked but found no results (was error -32000)
  - `category_not_found` - category lookup found no match (includes available categories)
  - `object_not_found` - OBEX object ID not found
  - Actual errors (network failures, invalid params) remain as JSON-RPC errors
  - This prevents potential MCP connection issues and provides friendlier responses for AI assistants

## [1.3.1] - 2026-01-17

### Added

- **Alias Resolution**: Query using natural names instead of canonical keys
  - Instruction mnemonics: `ADD`, `MOV`, `JMP` → `p2kbPasm2Add`, etc.
  - Method names: `WAITMS`, `PINWRITE` → `p2kbSpin2Waitms`, etc.
  - Pattern IDs: `motor_controller`, `state_machine`
  - Symbol names: `_CLKFREQ`, `_CLKMODE`
  - Case-insensitive matching (ADD, add, Add all work)

- **Resolution Metadata**: Responses include `resolved_from` field when an alias was used
  - Helps AI assistants understand the mapping between user query and canonical key

- **Index Statistics**: `p2kb_version` now reports `total_aliases` count

### Changed

- `p2kb_get` now accepts aliases as the query parameter
- `KeyExists`, `GetKeyPath`, `GetFileMtime` all support alias resolution
- `MatchQuery` checks aliases before falling back to token-based matching

### Technical

- Requires index version 3.3.0+ with `aliases` section
- Older indexes (without aliases) continue to work; alias lookups simply return no match

## [1.3.0] - 2026-01-15

### Added

- **New Tool: `p2kb_obex_download`** - Download and extract OBEX objects directly to your project
  - AI assistants can now complete the full OBEX workflow: search → get info → download
  - Automatically extracts zip files to `./OBEX/{objectID}-{slug}/` (e.g., `OBEX/2811-ws2812-led-driver/`)
  - Returns list of extracted files, paths, and total size
  - Optional `target_dir` parameter to override default extraction location

- **Security protections** for file operations:
  - Path traversal prevention (blocks `..` in paths)
  - Zip slip attack protection (validates all extracted file paths)
  - Per-file size limits to prevent decompression bombs

### Changed

- OBEX workflow is now complete end-to-end through MCP tools
  - Previously: AI could only provide download URLs and manual instructions
  - Now: AI can search, inspect, and download OBEX objects autonomously

## [1.2.3] - 2026-01-07

### Fixed

- **Critical: Fixed RLock upgrade deadlock in cache manager** that caused server crashes on parallel exact-key lookups
  - The server would crash with MCP error -32000 "Connection closed" when 6 parallel exact key queries were made
  - Root cause: `Get()` held RLock while calling `loadFromDisk()` which tried to acquire Lock - a classic deadlock pattern
  - Symptom: Fuzzy search worked, but fetching multiple exact keys in parallel caused immediate crash
  - Fix: RLock is now released before calling `loadFromDisk()` to prevent lock upgrade deadlock

- **Critical: Fixed lock-during-network-I/O in OBEX manager** (same bug pattern as index manager fixed in v1.2.2)
  - `EnsureIndex()` and `Refresh()` now use the same safe pattern as the index manager
  - Added separate `fetchMu` mutex to prevent concurrent fetches without blocking readers
  - Network I/O now happens outside the data lock

- **Moderate: Fixed disk I/O under locks in multiple functions**
  - `cache.GetCachedKeys()`: RLock released before `os.ReadDir()` call
  - `obex.ClearCache()`: Lock released before `os.RemoveAll()` call
  - `index.GetIndexStatus()`: RLock released before `os.Stat()` call

### Added

- Concurrent stress tests for cache and OBEX managers
  - `TestConcurrentGet`: Reproduces the exact 6-parallel-exact-key-lookup crash scenario
  - `TestConcurrentGetMemoryAndDisk`: Mixed memory/disk access patterns
  - `TestConcurrentGetCachedKeys`: Parallel GetCachedKeys calls
  - `TestConcurrentEnsureIndex`: Parallel EnsureIndex calls
  - All tests pass with Go's race detector enabled

## [1.2.2] - 2025-12-27

### Fixed

- **Critical: Fixed mutex deadlock in index manager** that caused server crashes
  - The server would hang and eventually crash with MCP error -32000 "Connection closed"
  - Root cause: Write lock was held during network I/O (up to 30 seconds)
  - Symptom: Fuzzy search worked, but fetching content by exact key caused crash
  - Fix: Network I/O now happens outside the lock using a separate fetch mutex

### Changed

- **Improved error diagnostics** for easier bug reporting
  - Error responses now include structured hints and report instructions
  - Set `P2KB_LOG_LEVEL=debug` or `P2KB_LOG_LEVEL=info` to see errors on stderr
  - Error messages include key, underlying error, and troubleshooting hints

## [1.2.1] - 2025-12-22

### Changed

- **Simplified Installer**: Removed hooks functionality from container-tools package
  - p2kb-mcp is stateless and doesn't require session hooks
  - MCP tools are self-describing through the MCP protocol
  - Removed `hooks/` directory from package
  - Removed all `~/.claude/settings.json` management code
  - Reduced install script from ~400 to ~300 lines

- **Updated Integration Guide**: Documentation updated to v2.0 pattern
  - Hooks now go in `~/.claude/settings.json` (not `mcp.json`)
  - Added native Claude Code hook event types documentation
  - Added jq patterns for MCPs that need hooks
  - Added migration guide from deprecated hooks-dispatcher pattern

### Fixed

- Fixed mcp.json-prior backup location (now correctly in `backup/` not nested in `backup/prior/`)

### Removed

- Removed `hooks/session-start.sh` from package (was a no-op placeholder)
- Removed jq as a hard dependency for installation

### Migration

- Legacy cleanup still included: automatically removes old hooks-dispatcher.sh and hooks.d/ artifacts when upgrading from v1.2.0

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

### Deprecated

- hooks-dispatcher pattern deprecated in favor of native Claude Code hooks in settings.json

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

[Unreleased]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.3.2...HEAD
[1.3.2]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.2.3...v1.3.0
[1.2.3]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/tag/v1.0.0
