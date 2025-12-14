package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

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

func TestManagerMemoryCache(t *testing.T) {
	m := NewManager()

	// Initially empty
	_, found := m.Get("test-key")
	if found {
		t.Error("Get should return false for non-existent key")
	}

	// Add to memory cache directly
	m.mu.Lock()
	m.memory["test-key"] = cacheEntry{content: "test content", mtime: 12345}
	m.mu.Unlock()

	// Should find in memory
	content, found := m.Get("test-key")
	if !found {
		t.Error("Get should return true for cached key")
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

	// Save to disk
	err := m.saveToDisk("test-key", "test content")
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
