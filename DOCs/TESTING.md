# Testing and Coverage

This document describes the testing strategy and coverage metrics for the P2KB MCP server.

## Running Tests

```bash
# Run all tests
make test

# Run tests with race detection
go test -v -race ./...

# Run fast tests only (no network)
make test-short

# Run tests with coverage report
make test-coverage

# Run live GitHub integration tests
make test-live
```

## Coverage Summary

**Overall Coverage: 54.8%**

| Package | Coverage | Description |
|---------|----------|-------------|
| `internal/filter` | 100.0% | YAML metadata filtering |
| `internal/fetch` | 72.1% | HTTP client for GitHub raw content |
| `internal/index` | 67.3% | KB index management and parsing |
| `internal/cache` | 55.6% | Content caching with TTL support |
| `internal/server` | 43.2% | MCP protocol handler and tools |
| `cmd/p2kb-mcp` | 0.0% | Main entry point (minimal logic) |

## Coverage Details by Package

### internal/filter (100%)

Fully tested. Handles YAML metadata filtering to reduce token usage.

- `FilterMetadata()` - Remove internal tracking fields
- `ShouldFilterLine()` - Line-level filter decisions
- `CountFilteredLines()` - Statistics tracking

### internal/fetch (72.1%)

HTTP client with mock server tests.

**Covered:**
- Client creation and options
- URL fetching with success/error cases
- HEAD requests for existence checks
- Base URL handling

**Not covered:**
- Gzip decompression (requires real gzipped data)
- Network error edge cases

### internal/index (67.3%)

Index management with mock data tests.

**Covered:**
- Manager creation and configuration
- Search functionality
- Category listing and counts
- Key existence checks
- Cache save/load operations

**Not covered:**
- Live network fetching (tested in integration)
- Some error recovery paths

### internal/cache (55.6%)

Memory and disk caching.

**Covered:**
- Memory cache operations
- Disk save/load
- Cache clearing and invalidation

**Not covered:**
- `FetchAndCache()` (requires network)
- Mtime-based freshness checks

### internal/server (43.2%)

MCP protocol handlers.

**Covered:**
- Server initialization
- Request routing
- Tool definitions
- Argument validation errors
- Helper functions (JSON, response builders)

**Not covered:**
- Handlers requiring live index/cache (handleCategories, handleStats, etc.)
- `Run()` stdin/stdout loop
- `getContent()` network path

### cmd/p2kb-mcp (0%)

Main entry point with minimal logic - just flag parsing and server startup. Not unit tested but covered by integration testing.

## Test Categories

### Unit Tests (make test)

Fast, isolated tests using mock data and test servers. No network required.

### Integration Tests (make test-live)

Tests that hit the real GitHub API. Run with:

```bash
go test -v -run "Live" ./...
```

These validate:
- Real index fetching
- Content retrieval
- Cache behavior with real data

### MCP Protocol Tests (make test-mcp)

End-to-end protocol validation:

```bash
make test-mcp
```

Sends JSON-RPC requests to the built binary and validates responses.

## CI Coverage Threshold

The CI pipeline enforces a **50% minimum coverage** threshold. This balances:
- Ensuring core logic is tested
- Acknowledging that network-dependent code is harder to unit test
- Avoiding test complexity that exceeds the value

## Improving Coverage

To improve coverage, focus on:

1. **Mock interfaces** for `indexManager` and `cacheManager` to test handlers
2. **Table-driven tests** for additional edge cases
3. **Error injection** for network failure scenarios

Coverage is tracked in the CHANGELOG for each release.
