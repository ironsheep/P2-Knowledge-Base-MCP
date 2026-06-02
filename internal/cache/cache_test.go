package cache

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

// knownMtime is a whole-second Unix timestamp used across mtime tests.
// os.Chtimes is second-resolution on most filesystems, so we use a clean value.
const knownMtime int64 = 1700000000

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.memory == nil {
		t.Error("memory map is nil")
	}
}

func TestGetCacheDir(t *testing.T) {
	// Test with env var - using a writable temp directory
	tmpDir := t.TempDir()
	os.Setenv("P2KB_CACHE_DIR", tmpDir)
	defer os.Unsetenv("P2KB_CACHE_DIR")

	dir := paths.GetCacheDirOrDefault()
	if dir != tmpDir {
		t.Errorf("GetCacheDirOrDefault() = %q, want %q", dir, tmpDir)
	}
}

// TestGetOrFetchMemoryTier verifies the memory tier of GetOrFetch: a fresh
// memory entry whose disk file exists is served without touching the network.
func TestGetOrFetchMemoryTier(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Prime memory + disk so the disk-existence authority check passes.
	primeCache(t, m, "test-key", "test content", knownMtime)

	// Path is deliberately bogus: if GetOrFetch falls through to the remote
	// tier it will fail, proving the memory tier did not serve.
	content, err := m.GetOrFetch("test-key", "bogus/should-not-fetch.yaml", "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch should serve from memory without fetching: %v", err)
	}
	if content != "test content" {
		t.Errorf("content = %q, want 'test content'", content)
	}
}

func TestManagerClear(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Add some data
	m.memory["key1"] = cacheEntry{content: "content1"}
	m.memory["key2"] = cacheEntry{content: "content2"}

	// Create a cache file
	cacheDir := filepath.Join(tmpDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.WriteFile(filepath.Join(cacheDir, "test.yaml"), []byte("test"), 0644)

	// Clear
	m.Clear()

	// Memory should be empty
	if len(m.memory) != 0 {
		t.Errorf("memory has %d items after Clear, want 0", len(m.memory))
	}

	// Cache directory should be removed
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("cache directory should be removed after Clear")
	}
}

func TestManagerInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Add some data
	m.memory["key1"] = cacheEntry{content: "content1"}
	m.memory["key2"] = cacheEntry{content: "content2"}

	// Create cache files
	cacheDir := filepath.Join(tmpDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.WriteFile(filepath.Join(cacheDir, "key1.yaml"), []byte("test"), 0644)

	// Invalidate key1
	m.Invalidate("key1")

	// key1 should be gone
	if _, ok := m.memory["key1"]; ok {
		t.Error("key1 should be removed from memory")
	}

	// key2 should still exist
	if _, ok := m.memory["key2"]; !ok {
		t.Error("key2 should still exist in memory")
	}
}

func TestManagerSaveAndLoadFromDisk(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Save to disk (mtime 0 is fine for the basic content round-trip test)
	err := m.saveToDisk("test-key", "test content", 0)
	if err != nil {
		t.Fatalf("saveToDisk failed: %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tmpDir, "cache", "test-key.yaml")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache file was not created")
	}

	// Load from disk
	content, err := m.loadFromDisk("test-key")
	if err != nil {
		t.Fatalf("loadFromDisk failed: %v", err)
	}
	if content != "test content" {
		t.Errorf("content = %q, want 'test content'", content)
	}

	// Should also be in memory cache now
	if _, ok := m.memory["test-key"]; !ok {
		t.Error("loadFromDisk should add to memory cache")
	}
}

func TestCacheEntryMtime(t *testing.T) {
	m := NewManager()

	// Add entry with mtime
	m.mu.Lock()
	m.memory["key1"] = cacheEntry{content: "old content", mtime: 1000}
	m.mu.Unlock()

	// Entry with older mtime shouldn't be returned as fresh
	m.mu.RLock()
	entry := m.memory["key1"]
	m.mu.RUnlock()

	if entry.mtime != 1000 {
		t.Errorf("mtime = %d, want 1000", entry.mtime)
	}
}

func TestBaseContentURL(t *testing.T) {
	expected := "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/"
	if BaseContentURL != expected {
		t.Errorf("BaseContentURL = %q, want %q", BaseContentURL, expected)
	}
}

func TestGetCachedKeys(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Add entries to memory
	m.memory["memkey1"] = cacheEntry{content: "content1"}
	m.memory["memkey2"] = cacheEntry{content: "content2"}

	// Add files to disk cache
	cacheDir := filepath.Join(tmpDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.WriteFile(filepath.Join(cacheDir, "diskkey1.yaml"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(cacheDir, "memkey1.yaml"), []byte("test"), 0644) // overlap

	keys := m.GetCachedKeys()

	// Should have 3 unique keys: memkey1, memkey2, diskkey1
	if len(keys) != 3 {
		t.Errorf("GetCachedKeys() returned %d keys, want 3", len(keys))
	}

	// Verify all expected keys are present
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	expected := []string{"memkey1", "memkey2", "diskkey1"}
	for _, k := range expected {
		if !keySet[k] {
			t.Errorf("missing key: %s", k)
		}
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Add entries to memory
	m.memory["key1"] = cacheEntry{content: "content1"}
	m.memory["key2"] = cacheEntry{content: "content2"}

	// Add files to disk cache
	cacheDir := filepath.Join(tmpDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.WriteFile(filepath.Join(cacheDir, "diskkey1.yaml"), []byte("test content here"), 0644)

	stats := m.GetStats()

	if stats.MemoryEntries != 2 {
		t.Errorf("MemoryEntries = %d, want 2", stats.MemoryEntries)
	}
	if stats.DiskEntries != 1 {
		t.Errorf("DiskEntries = %d, want 1", stats.DiskEntries)
	}
	if stats.DiskSizeBytes == 0 {
		t.Error("DiskSizeBytes should be > 0")
	}
	if stats.CacheDir != tmpDir {
		t.Errorf("CacheDir = %q, want %q", stats.CacheDir, tmpDir)
	}
}

func TestGetCachedKeysEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	keys := m.GetCachedKeys()
	if len(keys) != 0 {
		t.Errorf("GetCachedKeys() returned %d keys for empty cache, want 0", len(keys))
	}
}

func TestGetStatsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	stats := m.GetStats()

	if stats.MemoryEntries != 0 {
		t.Errorf("MemoryEntries = %d, want 0", stats.MemoryEntries)
	}
	if stats.DiskEntries != 0 {
		t.Errorf("DiskEntries = %d, want 0", stats.DiskEntries)
	}
	if stats.DiskSizeBytes != 0 {
		t.Errorf("DiskSizeBytes = %d, want 0", stats.DiskSizeBytes)
	}
}

// TestMtimeRoundTrip verifies that a mtime written via saveToDisk is recovered
// exactly (within filesystem second-resolution) by loadFromDisk, even after the
// in-memory entry is evicted.
func TestMtimeRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	const key = "mtime-roundtrip-key"

	// Seed disk state with a known whole-second mtime.
	if err := m.saveToDisk(key, "some yaml content", knownMtime); err != nil {
		t.Fatalf("saveToDisk failed: %v", err)
	}

	// Evict the memory entry so loadFromDisk must re-hydrate from disk.
	m.mu.Lock()
	delete(m.memory, key)
	m.mu.Unlock()

	// loadFromDisk should recover the mtime from the filesystem stamp.
	content, err := m.loadFromDisk(key)
	if err != nil {
		t.Fatalf("loadFromDisk failed: %v", err)
	}
	if content != "some yaml content" {
		t.Errorf("content = %q, want %q", content, "some yaml content")
	}

	// The in-memory entry now populated by loadFromDisk must carry the mtime.
	m.mu.RLock()
	entry, ok := m.memory[key]
	m.mu.RUnlock()

	if !ok {
		t.Fatal("loadFromDisk did not populate memory cache")
	}
	if entry.mtime != knownMtime {
		t.Errorf("mtime after loadFromDisk = %d, want %d", entry.mtime, knownMtime)
	}
}

// TestGetMtimeDiskFallback verifies that GetMtime returns the stamped mtime for
// a key that exists only on disk (not in the memory map), rather than 0.
func TestGetMtimeDiskFallback(t *testing.T) {
	tmpDir := t.TempDir()
	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	const key = "disk-only-key"

	// Write the cache file with a known mtime. saveToDisk does not touch the
	// memory map, so the key remains disk-only.
	if err := m.saveToDisk(key, "disk only content", knownMtime); err != nil {
		t.Fatalf("saveToDisk failed: %v", err)
	}

	// GetMtime must fall back to os.Stat and return the stamped mtime.
	got := m.GetMtime(key)
	if got != knownMtime {
		t.Errorf("GetMtime (disk fallback) = %d, want %d", got, knownMtime)
	}

	// Sanity: a non-existent key must still return 0.
	if zero := m.GetMtime("nonexistent-key"); zero != 0 {
		t.Errorf("GetMtime for missing key = %d, want 0", zero)
	}
}

// stubRemoteSeq points the remote tier (BaseContentURL) at a local httptest
// server that serves bodies[i] on the i-th request (clamping to the last entry
// for any further requests) and counts how many times it is hit. BaseContentURL
// is a global, so tests using this helper must not call t.Parallel. The returned
// pointer is the live hit counter; cleanup restores BaseContentURL and closes
// the server.
func stubRemoteSeq(t *testing.T, bodies ...string) *int32 {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := int(atomic.AddInt32(&hits, 1)) - 1
		if n >= len(bodies) {
			n = len(bodies) - 1
		}
		_, _ = io.WriteString(w, bodies[n])
	}))
	prev := BaseContentURL
	BaseContentURL = srv.URL + "/"
	t.Cleanup(func() {
		BaseContentURL = prev
		srv.Close()
	})
	return &hits
}

// stubRemote is the single-body case of stubRemoteSeq.
func stubRemote(t *testing.T, body string) *int32 {
	t.Helper()
	return stubRemoteSeq(t, body)
}

// shrinkBackoff lowers the cache-busting retry backoff to keep mismatch tests
// fast, restoring it on cleanup.
func shrinkBackoff(t *testing.T) {
	t.Helper()
	prev := contentRetryBackoff
	contentRetryBackoff = time.Millisecond
	t.Cleanup(func() { contentRetryBackoff = prev })
}

// primeCache seeds both the disk and memory tiers for key at mtime.
func primeCache(t *testing.T, m *Manager, key, content string, mtime int64) {
	t.Helper()
	if err := m.saveToDisk(key, content, mtime); err != nil {
		t.Fatalf("saveToDisk failed: %v", err)
	}
	m.mu.Lock()
	m.memory[key] = cacheEntry{content: content, mtime: mtime}
	m.mu.Unlock()
}

// TestGetOrFetchRefetchesOnNewerIndex (invariant a): when the index mtime
// advances past the cached entry, GetOrFetch must re-fetch from remote.
func TestGetOrFetchRefetchesOnNewerIndex(t *testing.T) {
	hits := stubRemote(t, "fresh remote content")
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	primeCache(t, m, key, "stale content", knownMtime)

	content, err := m.GetOrFetch(key, "any/path.yaml", "", knownMtime+100)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("remote fetches = %d, want 1 (newer index must refetch)", got)
	}
	if want := filter.FilterMetadata("fresh remote content"); content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
}

// TestGetOrFetchRefetchesWhenDiskFileDeleted (invariant b): a fresh memory
// entry must NOT be served once its backing disk file is gone — disk presence
// is authoritative, so a deleted file forces a re-fetch.
func TestGetOrFetchRefetchesWhenDiskFileDeleted(t *testing.T) {
	hits := stubRemote(t, "remote after delete")
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	primeCache(t, m, key, "cached content", knownMtime)

	// Remove the disk file but leave the (still fresh) memory entry in place.
	if err := os.Remove(m.cachePath(key)); err != nil {
		t.Fatalf("removing disk file: %v", err)
	}

	content, err := m.GetOrFetch(key, "any/path.yaml", "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("remote fetches = %d, want 1 (deleted disk file must force refetch)", got)
	}
	if want := filter.FilterMetadata("remote after delete"); content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
}

// TestGetOrFetchServesFreshCacheWithoutNetwork (invariant c): a cache that is
// at least as new as the index is served with zero network calls.
func TestGetOrFetchServesFreshCacheWithoutNetwork(t *testing.T) {
	hits := stubRemote(t, "should-not-be-fetched")
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	primeCache(t, m, key, "cached content", knownMtime)

	content, err := m.GetOrFetch(key, "any/path.yaml", "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Errorf("remote fetches = %d, want 0 (fresh cache must not hit network)", got)
	}
	if content != "cached content" {
		t.Errorf("content = %q, want %q", content, "cached content")
	}
}

// TestGetOrFetchServesFreshDiskWithoutNetwork covers the disk tier: with the
// memory map empty but a fresh disk file present, GetOrFetch serves from disk
// (and re-hydrates memory) without touching the network.
func TestGetOrFetchServesFreshDiskWithoutNetwork(t *testing.T) {
	hits := stubRemote(t, "should-not-be-fetched")
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	// Disk only — memory map left empty.
	if err := m.saveToDisk(key, "disk content", knownMtime); err != nil {
		t.Fatalf("saveToDisk failed: %v", err)
	}

	content, err := m.GetOrFetch(key, "any/path.yaml", "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Errorf("remote fetches = %d, want 0 (fresh disk must not hit network)", got)
	}
	if content != "disk content" {
		t.Errorf("content = %q, want %q", content, "disk content")
	}
	// Memory should now be hydrated from disk.
	m.mu.RLock()
	_, ok := m.memory[key]
	m.mu.RUnlock()
	if !ok {
		t.Error("disk-tier hit did not hydrate memory cache")
	}
}

// TestSHA256HexFormat sanity-checks the digest helper: lowercase, 64 hex chars,
// matching crypto/sha256 for a known input.
func TestSHA256HexFormat(t *testing.T) {
	// echo -n "abc" | sha256sum
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := sha256Hex("abc"); got != want {
		t.Errorf("sha256Hex(\"abc\") = %q, want %q", got, want)
	}
}

// TestGetOrFetchVerifiedMatchCachesAndServes: a download whose sha256 matches
// the index digest is fetched exactly once, then filtered, cached, and served.
func TestGetOrFetchVerifiedMatchCachesAndServes(t *testing.T) {
	const body = "verified content"
	hits := stubRemote(t, body)
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	content, err := m.GetOrFetch(key, "any/path.yaml", sha256Hex(body), knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("remote fetches = %d, want 1 (verified match should not retry)", got)
	}
	if want := filter.FilterMetadata(body); content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
	// Cached to memory and disk.
	m.mu.RLock()
	_, inMem := m.memory[key]
	m.mu.RUnlock()
	if !inMem {
		t.Error("verified content was not cached in memory")
	}
	if !m.diskFileExists(key) {
		t.Error("verified content was not written to disk")
	}
}

// TestGetOrFetchVerifiedMismatchReturnsUnavailable: a download whose sha256
// never matches busts the CDN (multiple fetches) and ultimately returns a
// *VerificationError carrying expected/actual, caching nothing.
func TestGetOrFetchVerifiedMismatchReturnsUnavailable(t *testing.T) {
	shrinkBackoff(t)
	const served = "tampered or stale bytes"
	hits := stubRemote(t, served)
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	expected := sha256Hex("the real content")
	_, err := m.GetOrFetch(key, "any/path.yaml", expected, knownMtime)
	if err == nil {
		t.Fatal("expected a verification error, got nil")
	}

	var verr *VerificationError
	if !errors.As(err, &verr) {
		t.Fatalf("error type = %T, want *VerificationError: %v", err, err)
	}
	if verr.Expected != expected {
		t.Errorf("verr.Expected = %q, want %q", verr.Expected, expected)
	}
	if verr.Actual != sha256Hex(served) {
		t.Errorf("verr.Actual = %q, want %q", verr.Actual, sha256Hex(served))
	}

	// It must have cache-busted: more than the single non-busted attempt.
	if got := atomic.LoadInt32(hits); int(got) != contentFetchAttempts {
		t.Errorf("remote fetches = %d, want %d (one normal + busted retries)", got, contentFetchAttempts)
	}

	// Nothing cached — the slot stays empty so the next request retries.
	m.mu.RLock()
	_, inMem := m.memory[key]
	m.mu.RUnlock()
	if inMem {
		t.Error("mismatched content must not be cached in memory")
	}
	if m.diskFileExists(key) {
		t.Error("mismatched content must not be written to disk")
	}
}

// TestGetOrFetchVerifiedBustRecovers: the first (edge) fetch returns stale
// bytes, but a cache-busting retry gets the correct bytes — the content is then
// served, proving bust-on-mismatch recovers from CDN propagation lag.
func TestGetOrFetchVerifiedBustRecovers(t *testing.T) {
	shrinkBackoff(t)
	const good = "the real content"
	hits := stubRemoteSeq(t, "stale edge bytes", good)
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	content, err := m.GetOrFetch(key, "any/path.yaml", sha256Hex(good), knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 2 {
		t.Errorf("remote fetches = %d, want 2 (one stale, one busted recovery)", got)
	}
	if want := filter.FilterMetadata(good); content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
	if !m.diskFileExists(key) {
		t.Error("recovered content was not cached to disk")
	}
}

// TestGetOrFetchEmptySHASkipsVerification: with no index digest (pre-3.5.0),
// the content is fetched once and served without any verification, even though
// it would not match an arbitrary hash.
func TestGetOrFetchEmptySHASkipsVerification(t *testing.T) {
	hits := stubRemote(t, "unverified legacy content")
	m := &Manager{cacheDir: t.TempDir(), memory: make(map[string]cacheEntry)}
	const key = "k"

	content, err := m.GetOrFetch(key, "any/path.yaml", "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("remote fetches = %d, want 1 (legacy path: single non-busted fetch)", got)
	}
	if want := filter.FilterMetadata("unverified legacy content"); content != want {
		t.Errorf("content = %q, want %q", content, want)
	}
}
