package cache

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentGet tests the fix for the RLock upgrade deadlock.
// This reproduces the exact scenario that caused the original crash:
// multiple parallel Get() calls that all need to load from disk.
func TestConcurrentGet(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Create disk cache entries (not in memory)
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Create 10 cache files on disk
	keys := []string{
		"p2kbPasm2Rep",
		"p2kbPasm2Altgb",
		"p2kbPasm2Getbyte",
		"p2kbPasm2Setbyte",
		"p2kbPasm2Waitx",
		"p2kbPasm2Nop",
		"p2kbPasm2Mov",
		"p2kbPasm2Add",
		"p2kbPasm2Sub",
		"p2kbPasm2And",
	}

	for _, key := range keys {
		cachePath := filepath.Join(cacheDir, key+".yaml")
		content := "test content for " + key
		if err := os.WriteFile(cachePath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create cache file: %v", err)
		}
	}

	// Launch 10 concurrent Get() calls - this is the exact scenario that caused the deadlock
	var wg sync.WaitGroup
	errors := make(chan error, len(keys))

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			content, found := m.Get(k)
			if !found {
				errors <- nil // Some may not be found, that's ok
				return
			}
			if content == "" {
				t.Errorf("Get(%s) returned empty content", k)
			}
		}(key)
	}

	// Wait for all goroutines to complete (with timeout via test framework)
	wg.Wait()
	close(errors)

	// If we get here without deadlock, the test passes!
	t.Log("All concurrent Get() calls completed without deadlock")
}

// TestConcurrentGetMemoryAndDisk tests mixed memory/disk access patterns.
func TestConcurrentGetMemoryAndDisk(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Pre-populate some keys in memory
	m.memory["key1"] = cacheEntry{content: "memory content 1", mtime: 1000}
	m.memory["key2"] = cacheEntry{content: "memory content 2", mtime: 1000}
	m.memory["key3"] = cacheEntry{content: "memory content 3", mtime: 1000}

	// Create disk cache entries for other keys
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	for i := 4; i <= 6; i++ {
		key := "key" + string(rune('0'+i))
		cachePath := filepath.Join(cacheDir, key+".yaml")
		if err := os.WriteFile(cachePath, []byte("disk content "+key), 0644); err != nil {
			t.Fatalf("failed to create cache file: %v", err)
		}
	}

	// Launch concurrent access to both memory and disk keys
	var wg sync.WaitGroup
	keys := []string{"key1", "key2", "key3", "key4", "key5", "key6"}

	// Run multiple rounds to increase chance of race detection
	for round := 0; round < 10; round++ {
		for _, key := range keys {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				m.Get(k)
			}(key)
		}
	}

	wg.Wait()
	t.Log("Mixed memory/disk concurrent access completed without deadlock")
}

// TestConcurrentGetCachedKeys tests concurrent access to GetCachedKeys.
func TestConcurrentGetCachedKeys(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir: tmpDir,
		memory:   make(map[string]cacheEntry),
	}

	// Pre-populate memory
	for i := 0; i < 5; i++ {
		key := "memkey" + string(rune('0'+i))
		m.memory[key] = cacheEntry{content: "content", mtime: 1000}
	}

	// Create disk entries
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	for i := 0; i < 5; i++ {
		key := "diskkey" + string(rune('0'+i))
		cachePath := filepath.Join(cacheDir, key+".yaml")
		if err := os.WriteFile(cachePath, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create cache file: %v", err)
		}
	}

	// Launch concurrent GetCachedKeys calls
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			keys := m.GetCachedKeys()
			if len(keys) == 0 {
				t.Error("GetCachedKeys returned empty")
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent GetCachedKeys completed without deadlock")
}
