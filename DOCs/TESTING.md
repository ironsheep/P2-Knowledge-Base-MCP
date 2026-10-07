# Testing and Coverage

This document describes how the P2KB MCP server is tested and how to run the tests.

## Running Tests

```bash
# Run all tests: the Go suite with the race detector, then the installer hygiene test
make test

# The Go suite alone
CGO_ENABLED=1 go test -race ./...

# Coverage report (coverage.out, coverage.html)
make test-coverage

# Send an initialize and a tools/list request to the built binary
make test-mcp

# Installer cache/backup hygiene test (shell, no Go)
make test-installer
```

## The suite needs no network

No test reaches the internet. Every package that fetches serves the index and content from
`internal/kbtest`, an in-memory stand-in for the KB's raw-content host: it serves a gzipped index
built from a Go value, content files at their index paths, per-path sequences of bodies, per-path hit
counts and recorded requests, and can hold a path's response until released (for race tests). The
fetcher is pointed at it the way `P2KB_BASE_URL` points the server at any host.

Server tests are built by one constructor, `newTestServer`, which also gives each test a temporary
cache directory and working directory, so a test run never touches a developer's real cache and never
writes into the source tree.

Measured 2026-10-07: the whole suite passes with `HTTPS_PROXY`/`HTTP_PROXY` pointed at an unreachable
proxy, and a recording proxy sees no connection attempt from any package.

## Coverage

Measured 2026-10-07 with `go test -coverprofile` (statements). **Total: 79.1%.**

| Package | Coverage | What its tests pin |
|---------|----------|--------------------|
| `internal/logging` | 100.0% | Level parsing, the default level, which lines each level writes |
| `internal/filter` | 96.7% | The format-1 delivery filter: the published `rule_id`, every refused rule block, and an engine table whose cases each fail on the pre-1.5.0 filter |
| `internal/kbtest` | 94.3% | The test KB host itself: index round trip, 404s, hit counts, body sequences, the response gate |
| `internal/fetch` | 90.7% | Base URL resolution, cache-busting only when asked, errors naming the URL, one debug line per request |
| `internal/index` | 87.0% | Index loading and refresh, alias resolution, search, and the delivery-filter rule in effect (index, last-good, built-in, refused) |
| `internal/cache` | 84.3% | The mtime- and sha256-aware cache tiers, the rule stamp and its discards, and the race guard that keeps a superseded filtering out of the cache |
| `internal/paths` | 75.0% | Cache directory resolution for each install layout |
| `internal/obex` | 73.2% | OBEX objects read through the main index and the cache, search, categories, authors, and download extraction |
| `internal/server` | 66.9% | Tool routing and argument errors, `p2kb_get`, `p2kb_version`, `p2kb_refresh`, and the OBEX tool results |
| `cmd/p2kb-mcp` | 0.0% | Entry point: flag handling and startup only |

Uncovered code is mostly error paths that need a failing filesystem, and the stdin/stdout loop in
`Server.Run`, which `make test-mcp` exercises against the built binary.

## CI Coverage Threshold

The CI pipeline (`make test-ci`) fails below **50%** total coverage.

## Writing Tests

- Serve KB data with `kbtest.NewRemote(t)`: `AddFile` lists a file in the index with its sha256 and
  serves it; `SetIndex` shapes the whole index, including a `delivery_filter` block; `Client()` returns
  a fetcher pointed at it.
- Build server tests with `newTestServer(t)`; pass a seed function to model what a previous run left in
  the cache directory.
- Concurrent code is tested under `-race`; a test of a race should fail when the guard it covers is
  removed.
