// Package cache manages the local cache of P2KB YAML files.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/logging"
	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

// Manager handles caching of P2KB content.
//
// Every cached body was filtered under the rule in effect, and the cache dir
// carries a stamp naming that rule and the filter engine (agreement §6).
//
// Locking: mu guards the in-memory state and is never held across I/O.
// diskMu serializes every write and removal in the cache dir — body writes,
// discards, Clear and Invalidate* — so a discard can never interleave with a
// body write. Lock order is diskMu, then mu.
type Manager struct {
	fetcher  *fetch.Client
	cacheDir string

	diskMu sync.Mutex

	mu     sync.RWMutex
	memory map[string]cacheEntry
	rule   filter.Rule // rule in effect, as last delivered by SetRule
	stamp  string      // stamp of rule; "" until SetRule
	gen    uint64      // bumped by every SetRule and Clear
}

type cacheEntry struct {
	content string
	mtime   int64
}

// stampFile names the cache stamp inside <cacheDir>/cache.
const stampFile = "filter-stamp"

// NewManager creates a new cache manager that fetches through fetcher. It
// filters under BuiltinRule until SetRule delivers the rule in effect.
func NewManager(fetcher *fetch.Client) *Manager {
	return &Manager{
		fetcher:  fetcher,
		cacheDir: paths.GetCacheDirOrDefault(),
		memory:   make(map[string]cacheEntry),
		rule:     filter.BuiltinRule,
	}
}

// Stamp is the cache stamp for rule: "<rule_id>/<filter.EngineVersion>".
func Stamp(rule filter.Rule) string {
	return rule.RuleID() + "/" + filter.EngineVersion
}

// SetRule makes rule the rule in effect. When the stamp on disk is not
// rule's — a different rule, a different engine, or no stamp at all (a cache
// from before stamping) — every cached body is discarded and the new stamp
// written. Any fetch in flight re-filters under rule before it stores.
func (m *Manager) SetRule(rule filter.Rule) {
	stamp := Stamp(rule)

	m.diskMu.Lock()
	defer m.diskMu.Unlock()

	discard := m.readStamp() != stamp

	m.mu.Lock()
	m.rule = rule
	m.stamp = stamp
	m.gen++
	if discard {
		m.memory = make(map[string]cacheEntry)
	}
	m.mu.Unlock()

	if discard {
		m.wipeDisk(stamp)
	}
}

// Rule returns the rule in effect.
func (m *Manager) Rule() filter.Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rule
}

// wipeDisk removes every cached body and writes stamp ("" writes none). The
// caller holds diskMu.
func (m *Manager) wipeDisk(stamp string) {
	dir := filepath.Join(m.cacheDir, "cache")
	if err := os.RemoveAll(dir); err != nil {
		logging.Warnf("p2kb-mcp: warning: failed to clear cache %s: %v", dir, err)
	}
	if stamp == "" {
		return
	}
	if err := m.writeStamp(stamp); err != nil {
		// The next start finds no matching stamp and discards again.
		logging.Warnf("p2kb-mcp: warning: failed to write cache stamp: %v", err)
	}
}

func (m *Manager) stampPath() string {
	return filepath.Join(m.cacheDir, "cache", stampFile)
}

// readStamp returns the stamp on disk, or "" when there is none.
func (m *Manager) readStamp() string {
	data, err := os.ReadFile(m.stampPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// writeStamp replaces the stamp atomically. The caller holds diskMu.
func (m *Manager) writeStamp(stamp string) error {
	path := m.stampPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(stamp+"\n"), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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
	_, gen := m.snapshot()
	info, err := os.Stat(m.cachePath(key))
	if err != nil || info.ModTime().Unix() < indexMtime {
		return "", false
	}

	content, err := m.readAndHydrate(key, info, gen)
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
	// The fetch belongs to the generation in effect when it starts.
	rule, gen := m.snapshot()

	// Legacy / unverifiable path: a single non-busted fetch, no verification.
	if expectedSHA256 == "" {
		content, err := m.fetchContent(path, false)
		if err != nil {
			return "", err
		}
		return m.filterAndCache(key, content, indexMtime, rule, gen), nil
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
			return m.filterAndCache(key, content, indexMtime, rule, gen), nil
		}
	}

	// Persistent mismatch: do NOT cache. Leave the slot empty for a later retry.
	return "", &VerificationError{Key: key, Expected: expectedSHA256, Actual: actual}
}

// snapshot returns the rule in effect and its generation.
func (m *Manager) snapshot() (filter.Rule, uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rule, m.gen
}

// filterAndCache filters fetched content under rule, the rule of generation
// gen, and stores it in memory and on disk, stamped with indexMtime. Shared by
// the verified and legacy fetch paths.
//
// The filter runs with no lock held. If SetRule or Clear ran since gen was
// taken (the generation moved), the body filtered under the superseded rule
// is dropped and the raw body, still in hand, is filtered again under the new
// one: the caller never receives, and the cache never holds, a superseded
// filtering. The generation is checked under diskMu, so no discard can fall
// between the check and the disk write.
func (m *Manager) filterAndCache(key, rawContent string, indexMtime int64, rule filter.Rule, gen uint64) string {
	for {
		filtered := filter.Apply(rule, rawContent)

		m.diskMu.Lock()
		m.mu.Lock()
		if m.gen != gen {
			rule, gen = m.rule, m.gen
			m.mu.Unlock()
			m.diskMu.Unlock()
			continue
		}
		m.memory[key] = cacheEntry{content: filtered, mtime: indexMtime}
		m.mu.Unlock()

		// Save to disk (best effort), stamping the file mtime to match indexMtime.
		_ = m.saveToDisk(key, filtered, indexMtime)
		m.diskMu.Unlock()
		return filtered
	}
}

// sha256Hex returns the lowercase hex sha256 digest of s, matching the digest
// format the index generator emits for the raw content blob.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Clear clears all cached content (p2kb_refresh flush) and rewrites the
// stamp, so the emptied cache is still stamped for the rule in effect.
func (m *Manager) Clear() {
	m.diskMu.Lock()
	defer m.diskMu.Unlock()

	m.mu.Lock()
	m.memory = make(map[string]cacheEntry)
	m.gen++
	stamp := m.stamp
	m.mu.Unlock()

	m.wipeDisk(stamp)
}

// Invalidate removes a specific key from cache.
func (m *Manager) Invalidate(key string) {
	m.diskMu.Lock()
	defer m.diskMu.Unlock()

	m.mu.Lock()
	delete(m.memory, key)
	m.mu.Unlock()

	_ = os.Remove(m.cachePath(key))
}

// fetchContent fetches the content file at path. bust is passed to the
// fetcher; it is true only to re-fetch after a sha256 mismatch, never on the
// normal path.
func (m *Manager) fetchContent(path string, bust bool) (string, error) {
	data, err := m.fetcher.Fetch(path, bust)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}
	return string(data), nil
}

// cachePath returns the on-disk path for a cached key's content file.
func (m *Manager) cachePath(key string) string {
	return filepath.Join(m.cacheDir, "cache", key+".yaml")
}

// loadFromDisk loads content from disk cache, recovering the stamped mtime.
func (m *Manager) loadFromDisk(key string) (string, error) {
	_, gen := m.snapshot()
	// Stat first so we can recover the stamped mtime
	info, err := os.Stat(m.cachePath(key))
	if err != nil {
		return "", err
	}
	return m.readAndHydrate(key, info, gen)
}

// errSuperseded reports a disk read that overlapped a discard: the body may
// have been filtered under a superseded rule, so it is neither served nor
// hydrated.
var errSuperseded = errors.New("cache discarded during read")

// readAndHydrate reads the cache file for key and stores it in the memory cache,
// preserving the stamped mtime from info. Callers pass the os.FileInfo they
// already statted so the disk read does not stat the file a second time, and
// gen, the generation taken before that stat: a read that overlapped SetRule
// or Clear returns errSuperseded and hydrates nothing.
func (m *Manager) readAndHydrate(key string, info os.FileInfo, gen uint64) (string, error) {
	data, err := os.ReadFile(m.cachePath(key))
	if err != nil {
		return "", err
	}

	// Also store in memory cache for faster access next time, preserving mtime
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gen != gen {
		return "", errSuperseded
	}
	m.memory[key] = cacheEntry{content: string(data), mtime: info.ModTime().Unix()}
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
	MemoryEntries int    `json:"memory_entries"`
	DiskEntries   int    `json:"disk_entries"`
	DiskSizeBytes int64  `json:"disk_size_bytes"`
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

// InvalidateKeys removes multiple keys from cache, returning how many memory
// entries and disk files it removed.
func (m *Manager) InvalidateKeys(keys []string) int {
	m.diskMu.Lock()
	defer m.diskMu.Unlock()

	count := 0
	m.mu.Lock()
	for _, key := range keys {
		if _, ok := m.memory[key]; ok {
			delete(m.memory, key)
			count++
		}
	}
	m.mu.Unlock()

	for _, key := range keys {
		if err := os.Remove(m.cachePath(key)); err == nil {
			count++
		}
	}
	return count
}
