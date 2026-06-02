// Package cache manages the local cache of P2KB YAML files.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

// BaseContentURL is the base URL for P2KB content files. It is a var (not a
// const) so tests can point the remote tier at a local httptest server.
var BaseContentURL = "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/"

// Manager handles caching of P2KB content.
type Manager struct {
	mu       sync.RWMutex
	cacheDir string
	memory   map[string]cacheEntry
}

type cacheEntry struct {
	content string
	mtime   int64
}

// NewManager creates a new cache manager.
func NewManager() *Manager {
	return &Manager{
		cacheDir: paths.GetCacheDirOrDefault(),
		memory:   make(map[string]cacheEntry),
	}
}

// GetOrFetch resolves content for a key using an mtime-aware three-tier lookup.
// The caller supplies the authoritative index mtime (git commit time, from
// GetKeyPath) so a newer index always invalidates older cached content on read.
// Disk presence is authoritative: a cached memory entry is served only while its
// backing disk file still exists, so deleting the disk file forces a re-fetch.
//
// Tiers, in order:
//  1. Memory — served iff entry.mtime >= indexMtime AND the disk file exists.
//  2. Disk   — loaded (and memory re-hydrated) iff the stamped file mtime >= indexMtime.
//  3. Remote — fetched, filtered, and cached to memory + disk stamped with indexMtime.
//
// This is the single content entry point: callers never bypass the index mtime.
// expectedSHA256 is the index's content digest; when non-empty the remote tier
// verifies the raw download against it (see fetchAndStore). Concurrency follows
// CLAUDE.md — read locks are released before any disk or network I/O, and no
// data lock is held across a fetch.
func (m *Manager) GetOrFetch(key, path, expectedSHA256 string, indexMtime int64) (string, error) {
	// Tier 1: memory — only serve a fresh entry whose disk file still exists.
	m.mu.RLock()
	entry, ok := m.memory[key]
	m.mu.RUnlock() // release BEFORE disk I/O

	if ok && entry.mtime >= indexMtime && m.diskFileExists(key) {
		return entry.content, nil
	}

	// Tier 2: disk — serve if present and at least as new as the index.
	if content, fresh := m.loadFromDiskIfFresh(key, indexMtime); fresh {
		return content, nil
	}

	// Tier 3: remote.
	return m.fetchAndStore(key, path, expectedSHA256, indexMtime)
}

// diskFileExists reports whether the cached file for key is present on disk.
// Statted on every memory-tier read; negligible cost, and it enforces the
// "removed disk file => not served from memory" invariant.
func (m *Manager) diskFileExists(key string) bool {
	_, err := os.Stat(m.cachePath(key))
	return err == nil
}

// loadFromDiskIfFresh loads the disk-cached content for key only when its
// stamped mtime is at least indexMtime, hydrating the memory cache on a hit.
// Returns (content, true) on a fresh hit, ("", false) when absent or stale.
// It reuses the os.Stat result for hydration so the disk tier stats once.
func (m *Manager) loadFromDiskIfFresh(key string, indexMtime int64) (string, bool) {
	info, err := os.Stat(m.cachePath(key))
	if err != nil || info.ModTime().Unix() < indexMtime {
		return "", false
	}

	content, err := m.readAndHydrate(key, info)
	if err != nil {
		return "", false
	}
	return content, true
}

// contentFetchAttempts bounds how many times the remote tier fetches a file
// whose sha256 fails to verify. The first attempt rides the CDN edge; the rest
// cache-bust to defeat propagation lag after a KB push.
const contentFetchAttempts = 3

// contentRetryBackoff is the pause before each cache-busting retry, giving the
// CDN a moment to propagate fresh bytes. It is a var so tests can shrink it.
var contentRetryBackoff = 250 * time.Millisecond

// VerificationError reports that a downloaded file's sha256 did not match the
// index digest after exhausting cache-busting retries. It is distinct from
// not-found and network errors so the caller can surface a "temporarily
// unavailable — verification failed" result rather than conflating the cases.
type VerificationError struct {
	Key      string
	Expected string
	Actual   string
}

func (e *VerificationError) Error() string {
	return fmt.Sprintf("content verification failed for %q: expected sha256 %s, got %s",
		e.Key, e.Expected, e.Actual)
}

// fetchAndStore is the remote tier of GetOrFetch: it fetches the content and,
// when expectedSHA256 is set, verifies the RAW download against it BEFORE any
// filtering (the generator hashes the raw blob, so the digest is filter-
// agnostic). On a verified fetch — or when no digest is available (pre-3.5.0
// index, graceful degrade) — the content is filtered and cached. On a
// persistent mismatch nothing is cached and a *VerificationError is returned;
// the slot stays empty so the next natural request retries.
func (m *Manager) fetchAndStore(key, path, expectedSHA256 string, indexMtime int64) (string, error) {
	// Legacy / unverifiable path: a single non-busted fetch, no verification.
	if expectedSHA256 == "" {
		content, err := m.fetchContent(path, false)
		if err != nil {
			return "", err
		}
		return m.filterAndCache(key, content, indexMtime), nil
	}

	// Verified path. Attempt 0 rides the CDN edge; later attempts cache-bust
	// and back off briefly to ride out CDN propagation lag after a push.
	var actual string
	for attempt := 0; attempt < contentFetchAttempts; attempt++ {
		bust := attempt > 0
		if bust {
			time.Sleep(contentRetryBackoff)
		}

		content, err := m.fetchContent(path, bust)
		if err != nil {
			return "", err
		}

		actual = sha256Hex(content)
		if actual == expectedSHA256 {
			return m.filterAndCache(key, content, indexMtime), nil
		}
	}

	// Persistent mismatch: do NOT cache. Leave the slot empty for a later retry.
	return "", &VerificationError{Key: key, Expected: expectedSHA256, Actual: actual}
}

// filterAndCache filters fetched content and stores it in memory and on disk,
// stamped with indexMtime. Shared by the verified and legacy fetch paths.
func (m *Manager) filterAndCache(key, rawContent string, indexMtime int64) string {
	filtered := filter.FilterMetadata(rawContent)

	m.mu.Lock()
	m.memory[key] = cacheEntry{content: filtered, mtime: indexMtime}
	m.mu.Unlock()

	// Save to disk (best effort), stamping the file mtime to match indexMtime.
	_ = m.saveToDisk(key, filtered, indexMtime)

	return filtered
}

// sha256Hex returns the lowercase hex sha256 digest of s, matching the digest
// format the index generator emits for the raw content blob.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Clear clears all cached content.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear memory cache
	m.memory = make(map[string]cacheEntry)

	// Clear disk cache
	cacheDir := filepath.Join(m.cacheDir, "cache")
	_ = os.RemoveAll(cacheDir)
}

// Invalidate removes a specific key from cache.
func (m *Manager) Invalidate(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.memory, key)
	cachePath := m.cachePath(key)
	_ = os.Remove(cachePath)
}

// fetchContent fetches content from the remote URL. When bust is true it adds a
// cache-busting query parameter and no-cache headers to bypass the GitHub CDN
// (Fastly), mirroring the index fetch — used only to re-fetch after a sha256
// mismatch, never on the normal path.
func (m *Manager) fetchContent(path string, bust bool) (string, error) {
	url := BaseContentURL + path
	if bust {
		// Fastly keys on the query string but ignores client Cache-Control;
		// the unique ?t= is what actually forces a fresh origin fetch.
		url = fmt.Sprintf("%s?t=%d", url, time.Now().UnixNano())
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create content request: %w", err)
	}
	if bust {
		req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		req.Header.Set("Pragma", "no-cache")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch content: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	return string(data), nil
}

// cachePath returns the on-disk path for a cached key's content file.
func (m *Manager) cachePath(key string) string {
	return filepath.Join(m.cacheDir, "cache", key+".yaml")
}

// loadFromDisk loads content from disk cache, recovering the stamped mtime.
func (m *Manager) loadFromDisk(key string) (string, error) {
	// Stat first so we can recover the stamped mtime
	info, err := os.Stat(m.cachePath(key))
	if err != nil {
		return "", err
	}
	return m.readAndHydrate(key, info)
}

// readAndHydrate reads the cache file for key and stores it in the memory cache,
// preserving the stamped mtime from info. Callers pass the os.FileInfo they
// already statted so the disk read does not stat the file a second time.
func (m *Manager) readAndHydrate(key string, info os.FileInfo) (string, error) {
	data, err := os.ReadFile(m.cachePath(key))
	if err != nil {
		return "", err
	}

	// Also store in memory cache for faster access next time, preserving mtime
	m.mu.Lock()
	m.memory[key] = cacheEntry{content: string(data), mtime: info.ModTime().Unix()}
	m.mu.Unlock()

	return string(data), nil
}

// saveToDisk saves content to disk cache and stamps the file mtime to match
// the YAML mtime so that staleness detection survives process restarts.
func (m *Manager) saveToDisk(key, content string, mtime int64) error {
	cacheDir := filepath.Join(m.cacheDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cachePath := m.cachePath(key)
	if err := os.WriteFile(cachePath, []byte(content), 0644); err != nil {
		return err
	}

	// Stamp the filesystem mtime so it survives process restarts.
	// os.Chtimes is second-resolution on most filesystems.
	t := time.Unix(mtime, 0)
	return os.Chtimes(cachePath, t, t)
}

// CacheStats contains statistics about the cache.
type CacheStats struct {
	MemoryEntries int   `json:"memory_entries"`
	DiskEntries   int   `json:"disk_entries"`
	DiskSizeBytes int64 `json:"disk_size_bytes"`
	CacheDir      string `json:"cache_dir"`
}

// GetCachedKeys returns a list of all cached keys (memory + disk).
// This method releases the read lock before disk I/O for better concurrency.
func (m *Manager) GetCachedKeys() []string {
	// Use a map to deduplicate
	keySet := make(map[string]struct{})

	// Add memory keys under read lock
	m.mu.RLock()
	for key := range m.memory {
		keySet[key] = struct{}{}
	}
	m.mu.RUnlock() // Release read lock BEFORE disk I/O

	// Add disk keys - no lock needed for reading directory
	cacheDir := filepath.Join(m.cacheDir, "cache")
	entries, err := os.ReadDir(cacheDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				if len(name) > 5 && name[len(name)-5:] == ".yaml" {
					key := name[:len(name)-5]
					keySet[key] = struct{}{}
				}
			}
		}
	}

	// Convert to slice
	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	return keys
}

// GetStats returns cache statistics.
func (m *Manager) GetStats() CacheStats {
	m.mu.RLock()
	memoryCount := len(m.memory)
	m.mu.RUnlock()

	var diskCount int
	var diskSize int64

	cacheDir := filepath.Join(m.cacheDir, "cache")
	entries, err := os.ReadDir(cacheDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				if len(name) > 5 && name[len(name)-5:] == ".yaml" {
					diskCount++
					if info, err := entry.Info(); err == nil {
						diskSize += info.Size()
					}
				}
			}
		}
	}

	return CacheStats{
		MemoryEntries: memoryCount,
		DiskEntries:   diskCount,
		DiskSizeBytes: diskSize,
		CacheDir:      m.cacheDir,
	}
}

// GetMtime returns the cached mtime for a key, or 0 if not cached.
// When the key is absent from the memory map it falls back to os.Stat on the
// cache file so that disk-only entries (after a process restart) still report
// their stamped mtime rather than 0.
// Per the CLAUDE.md concurrency rule "release locks before I/O", the read lock
// is released before any disk I/O.
func (m *Manager) GetMtime(key string) int64 {
	// Fast path: check memory under read lock
	m.mu.RLock()
	if entry, ok := m.memory[key]; ok {
		mtime := entry.mtime
		m.mu.RUnlock()
		return mtime
	}
	m.mu.RUnlock() // release BEFORE disk I/O

	// Slow path: fall back to the stamped filesystem mtime
	cachePath := m.cachePath(key)
	if info, err := os.Stat(cachePath); err == nil {
		return info.ModTime().Unix()
	}
	return 0
}

// InvalidateKeys removes multiple keys from cache.
func (m *Manager) InvalidateKeys(keys []string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, key := range keys {
		if _, ok := m.memory[key]; ok {
			delete(m.memory, key)
			count++
		}
		cachePath := m.cachePath(key)
		if err := os.Remove(cachePath); err == nil {
			count++
		}
	}
	return count
}

