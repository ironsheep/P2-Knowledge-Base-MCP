# Claude Code Instructions for P2KB-MCP

## Project Overview

P2KB-MCP is an MCP (Model Context Protocol) server that provides access to the Propeller 2 Knowledge Base for AI assistants. It's written in Go and supports PASM2 instructions, Spin2 methods, and OBEX community objects.

## Release Process

When releasing a new version:

1. **Update VERSION file** - This is checked by the CI workflow
   ```bash
   echo "X.Y.Z" > VERSION
   ```

2. **Update CHANGELOG.md** - Add new version section with changes

3. **Commit all changes** including VERSION and CHANGELOG.md

4. **Create and push tag**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z - Brief description"
   git push origin main
   git push origin vX.Y.Z
   ```

The release workflow validates that VERSION file matches the git tag before building.

## Concurrency Patterns

This codebase uses `sync.RWMutex` extensively. Follow these rules to avoid deadlocks:

1. **Never upgrade locks** - Don't call a function that needs `Lock()` while holding `RLock()`
2. **Release locks before I/O** - Network and disk I/O should happen outside locks
3. **Use separate fetch mutexes** - For operations that do network I/O, use a separate mutex (like `fetchMu`) to prevent concurrent fetches without blocking readers

Example of the correct pattern (from `index.go`):
```go
// Fast path with read lock
m.mu.RLock()
if m.data != nil && fresh {
    m.mu.RUnlock()
    return nil
}
m.mu.RUnlock()

// Slow path with fetch mutex (separate from data lock)
m.fetchMu.Lock()
defer m.fetchMu.Unlock()

// Network I/O happens here - NO data lock held
data, err := m.fetchData()

// Update under write lock (quick operation)
m.mu.Lock()
defer m.mu.Unlock()
m.data = data
```

## Testing

Run tests with race detector when modifying concurrent code:
```bash
CGO_ENABLED=1 go test ./... -race
```

## Key Directories

- `internal/cache/` - Content caching with disk persistence
- `internal/index/` - KB index management
- `internal/obex/` - OBEX (Parallax Object Exchange) support
- `internal/server/` - MCP protocol handler
- `internal/filter/` - YAML metadata filtering
- `internal/fetch/` - HTTP client utilities
