// Package index manages the P2KB index file.
package index

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

const (
	// IndexURL is the URL of the compressed index file.
	IndexURL = "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main/deliverables/ai/p2kb-index.json.gz"

	// DefaultIndexTTL is the default time-to-live for the cached index.
	DefaultIndexTTL = 24 * time.Hour
)

// Index represents the P2KB index structure.
type Index struct {
	System     SystemInfo            `json:"system"`
	Categories map[string][]string   `json:"categories"`
	Files      map[string]FileEntry  `json:"files"`
}

// SystemInfo contains metadata about the index.
type SystemInfo struct {
	Version        string `json:"version"`
	Generated      string `json:"generated"`
	TotalEntries   int    `json:"total_entries"`
	TotalCategories int   `json:"total_categories"`
}

// FileEntry represents a single file in the index.
type FileEntry struct {
	Path  string `json:"path"`
	Mtime int64  `json:"mtime"`
}

// Stats contains index statistics.
type Stats struct {
	Version         string
	TotalEntries    int
	TotalCategories int
}

// Manager handles index operations.
type Manager struct {
	mu          sync.RWMutex
	index       *Index
	indexPath   string
	metaPath    string
	lastRefresh time.Time
	ttl         time.Duration
}

// NewManager creates a new index manager.
func NewManager() *Manager {
	cacheDir := paths.GetCacheDirOrDefault()
	return &Manager{
		indexPath: filepath.Join(cacheDir, "index", "p2kb-index.json"),
		metaPath:  filepath.Join(cacheDir, "index", "p2kb-index.meta"),
		ttl:       getIndexTTL(),
	}
}

// EnsureIndex ensures the index is loaded and fresh.
func (m *Manager) EnsureIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have a fresh cached index
	if m.index != nil && time.Since(m.lastRefresh) < m.ttl {
		return nil
	}

	// Try to load from cache
	if m.loadFromCache() {
		return nil
	}

	// Fetch from remote
	return m.fetchIndex()
}

// Refresh forces a refresh of the index.
func (m *Manager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fetchIndex()
}

// GetKeyPath returns the path and mtime for a key.
func (m *Manager) GetKeyPath(key string) (string, int64, error) {
	if err := m.EnsureIndex(); err != nil {
		return "", 0, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.index.Files[key]
	if !ok {
		return "", 0, fmt.Errorf("key not found: %s", key)
	}

	return entry.Path, entry.Mtime, nil
}

// KeyExists checks if a key exists in the index.
func (m *Manager) KeyExists(key string) bool {
	if err := m.EnsureIndex(); err != nil {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.index.Files[key]
	return ok
}

// Search searches for keys matching a term.
func (m *Manager) Search(term string, limit int) []string {
	if term == "" {
		return nil
	}

	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	term = strings.ToLower(term)
	var matches []string

	for key := range m.index.Files {
		if strings.Contains(strings.ToLower(key), term) {
			matches = append(matches, key)
		}
	}

	sort.Strings(matches)
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

// FindSimilarKeys finds keys similar to the given key.
func (m *Manager) FindSimilarKeys(key string, limit int) []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	keyLower := strings.ToLower(key)
	var matches []string

	// Try different substrings
	for k := range m.index.Files {
		kLower := strings.ToLower(k)
		// Check if key is a substring or contains common parts
		if strings.Contains(kLower, keyLower) || strings.Contains(keyLower, kLower) {
			matches = append(matches, k)
		} else {
			// Check for partial matches (e.g., "mov" in "p2kbPasm2Mov")
			for i := 3; i <= len(keyLower); i++ {
				if strings.Contains(kLower, keyLower[:i]) {
					matches = append(matches, k)
					break
				}
			}
		}
	}

	sort.Strings(matches)
	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	return matches
}

// GetCategories returns all category names.
func (m *Manager) GetCategories() []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	categories := make([]string, 0, len(m.index.Categories))
	for cat := range m.index.Categories {
		categories = append(categories, cat)
	}
	sort.Strings(categories)
	return categories
}

// GetCategoriesWithCounts returns categories with their entry counts.
func (m *Manager) GetCategoriesWithCounts() map[string]int {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]int)
	for cat, keys := range m.index.Categories {
		counts[cat] = len(keys)
	}
	return counts
}

// GetCategoryKeys returns all keys in a category.
func (m *Manager) GetCategoryKeys(category string) ([]string, error) {
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	keys, ok := m.index.Categories[category]
	if !ok {
		return nil, fmt.Errorf("category not found: %s", category)
	}

	result := make([]string, len(keys))
	copy(result, keys)
	sort.Strings(result)
	return result, nil
}

// GetKeyCategories returns the categories a key belongs to.
func (m *Manager) GetKeyCategories(key string) []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var categories []string
	for cat, keys := range m.index.Categories {
		for _, k := range keys {
			if k == key {
				categories = append(categories, cat)
				break
			}
		}
	}
	sort.Strings(categories)
	return categories
}

// GetStats returns index statistics.
func (m *Manager) GetStats() Stats {
	if err := m.EnsureIndex(); err != nil {
		return Stats{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		Version:         m.index.System.Version,
		TotalEntries:    m.index.System.TotalEntries,
		TotalCategories: m.index.System.TotalCategories,
	}
}

// loadFromCache attempts to load the index from the local cache.
func (m *Manager) loadFromCache() bool {
	// Check if cache exists and is fresh
	info, err := os.Stat(m.indexPath)
	if err != nil {
		return false
	}

	// Check TTL
	if time.Since(info.ModTime()) > m.ttl {
		return false
	}

	// Load the index
	data, err := os.ReadFile(m.indexPath)
	if err != nil {
		return false
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return false
	}

	m.index = &idx
	m.lastRefresh = info.ModTime()
	return true
}

// fetchIndex fetches the index from the remote URL.
func (m *Manager) fetchIndex() error {
	resp, err := http.Get(IndexURL)
	if err != nil {
		return fmt.Errorf("failed to fetch index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch index: HTTP %d", resp.StatusCode)
	}

	// Decompress gzip
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress index: %w", err)
	}
	defer gr.Close()

	data, err := io.ReadAll(gr)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("failed to parse index: %w", err)
	}

	// Save to cache
	if err := m.saveToCache(data); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to cache index: %v\n", err)
	}

	m.index = &idx
	m.lastRefresh = time.Now()
	return nil
}

// saveToCache saves the index to the local cache.
func (m *Manager) saveToCache(data []byte) error {
	dir := filepath.Dir(m.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.indexPath, data, 0644)
}


// getIndexTTL returns the index TTL from environment or default.
func getIndexTTL() time.Duration {
	if ttl := os.Getenv("P2KB_INDEX_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl + "s"); err == nil {
			return d
		}
	}
	return DefaultIndexTTL
}
