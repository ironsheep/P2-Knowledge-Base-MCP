// Package cache manages the local cache of P2KB YAML files.
package cache

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

const (
	// BaseContentURL is the base URL for P2KB content files.
	BaseContentURL = "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/"
)

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

// Get retrieves content from cache.
// This method is safe for concurrent access - it releases the read lock before
// any disk I/O to prevent deadlocks when loadFromDisk needs a write lock.
func (m *Manager) Get(key string) (string, bool) {
	// Check memory cache first with read lock
	m.mu.RLock()
	if entry, ok := m.memory[key]; ok {
		content := entry.content
		m.mu.RUnlock()
		return content, true
	}
	m.mu.RUnlock() // Release read lock BEFORE disk I/O

	// Check disk cache - loadFromDisk will acquire its own write lock if needed
	content, err := m.loadFromDisk(key)
	if err != nil {
		return "", false
	}

	return content, true
}

// FetchAndCache fetches content from remote and caches it.
func (m *Manager) FetchAndCache(key, path string, mtime int64) (string, error) {
	// Check if we have a cached version with matching mtime
	m.mu.RLock()
	if entry, ok := m.memory[key]; ok && entry.mtime >= mtime {
		m.mu.RUnlock()
		return entry.content, nil
	}
	m.mu.RUnlock()

	// Fetch from remote
	content, err := m.fetchContent(path)
	if err != nil {
		return "", err
	}

	// Filter metadata
	filtered := filter.FilterMetadata(content)

	// Cache in memory and disk
	m.mu.Lock()
	m.memory[key] = cacheEntry{content: filtered, mtime: mtime}
	m.mu.Unlock()

	// Save to disk (best effort)
	_ = m.saveToDisk(key, filtered)

	return filtered, nil
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
	cachePath := filepath.Join(m.cacheDir, "cache", key+".yaml")
	_ = os.Remove(cachePath)
}

// fetchContent fetches content from the remote URL.
func (m *Manager) fetchContent(path string) (string, error) {
	url := BaseContentURL + path
	resp, err := http.Get(url)
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

// loadFromDisk loads content from disk cache.
func (m *Manager) loadFromDisk(key string) (string, error) {
	cachePath := filepath.Join(m.cacheDir, "cache", key+".yaml")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return "", err
	}

	// Also store in memory cache for faster access next time
	m.mu.Lock()
	m.memory[key] = cacheEntry{content: string(data), mtime: 0}
	m.mu.Unlock()

	return string(data), nil
}

// saveToDisk saves content to disk cache.
func (m *Manager) saveToDisk(key, content string) error {
	cacheDir := filepath.Join(m.cacheDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	cachePath := filepath.Join(cacheDir, key+".yaml")
	return os.WriteFile(cachePath, []byte(content), 0644)
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
func (m *Manager) GetMtime(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.memory[key]; ok {
		return entry.mtime
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
		cachePath := filepath.Join(m.cacheDir, "cache", key+".yaml")
		if err := os.Remove(cachePath); err == nil {
			count++
		}
	}
	return count
}

