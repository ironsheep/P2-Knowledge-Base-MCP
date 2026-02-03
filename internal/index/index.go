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

	// ErrorRefreshCooldown is the minimum time between refresh-on-error attempts.
	// This prevents excessive refresh attempts when keys are genuinely not found.
	ErrorRefreshCooldown = 5 * time.Minute
)

// Index represents the P2KB index structure.
type Index struct {
	System     SystemInfo            `json:"system"`
	Categories map[string][]string   `json:"categories"`
	Files      map[string]FileEntry  `json:"files"`
	Aliases    map[string][]string   `json:"aliases"` // alias -> []canonical keys (first wins)
}

// SystemInfo contains metadata about the index.
type SystemInfo struct {
	Version         string `json:"version"`
	Generated       string `json:"generated"`
	TotalEntries    int    `json:"total_entries"`
	TotalCategories int    `json:"total_categories"`
	TotalAliases    int    `json:"total_aliases"`
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
	TotalAliases    int
}

// IndexStatus contains index freshness information.
type IndexStatus struct {
	LastUpdated   time.Time `json:"last_updated"`
	AgeSeconds    int64     `json:"age_seconds"`
	NeedsRefresh  bool      `json:"needs_refresh"`
	TTLSeconds    int64     `json:"ttl_seconds"`
	Version       string    `json:"version"`
	CacheFilePath string    `json:"cache_file_path"`
	IsCached      bool      `json:"is_cached"`
}

// Manager handles index operations.
type Manager struct {
	mu               sync.RWMutex
	fetchMu          sync.Mutex // Prevents concurrent fetches, separate from data lock
	index            *Index
	indexPath        string
	metaPath         string
	lastRefresh      time.Time
	ttl              time.Duration
	lastErrorRefresh time.Time // Tracks last refresh-on-error attempt to prevent refresh storms
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
// This method is safe for concurrent access and does NOT hold locks during network I/O.
func (m *Manager) EnsureIndex() error {
	// Fast path: check with read lock if we have a fresh index
	m.mu.RLock()
	if m.index != nil && time.Since(m.lastRefresh) < m.ttl {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// Slow path: need to load or refresh
	// Use fetchMu to prevent concurrent fetches (separate from data lock)
	m.fetchMu.Lock()
	defer m.fetchMu.Unlock()

	// Double-check after acquiring fetch lock (another goroutine may have loaded it)
	m.mu.RLock()
	if m.index != nil && time.Since(m.lastRefresh) < m.ttl {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// Try to load from cache (quick file I/O, safe to hold write lock briefly)
	m.mu.Lock()
	if m.loadFromCache() {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Fetch from remote WITHOUT holding the data lock
	// This is the critical fix: network I/O happens outside the lock
	idx, data, err := m.fetchIndexData()
	if err != nil {
		return fmt.Errorf("index fetch failed: %w", err)
	}

	// Update the index under write lock (quick operation)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Save to cache
	if err := m.saveToCache(data); err != nil {
		fmt.Fprintf(os.Stderr, "p2kb-mcp: warning: failed to cache index: %v\n", err)
	}

	m.index = idx
	m.lastRefresh = time.Now()
	return nil
}

// Refresh forces a refresh of the index.
// This method fetches fresh data from remote without holding locks during network I/O.
func (m *Manager) Refresh() error {
	// Use fetchMu to prevent concurrent fetches
	m.fetchMu.Lock()
	defer m.fetchMu.Unlock()

	// Fetch from remote WITHOUT holding the data lock
	idx, data, err := m.fetchIndexData()
	if err != nil {
		return fmt.Errorf("index refresh failed: %w", err)
	}

	// Update the index under write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.saveToCache(data); err != nil {
		fmt.Fprintf(os.Stderr, "p2kb-mcp: warning: failed to cache index: %v\n", err)
	}

	m.index = idx
	m.lastRefresh = time.Now()
	return nil
}

// KeyResolution contains the result of resolving a key through aliases.
type KeyResolution struct {
	CanonicalKey string // The resolved canonical key (e.g., "p2kbPasm2Add")
	ResolvedFrom string // The original query if it was an alias (e.g., "ADD"), empty if direct match
	Found        bool   // Whether the key was found
}

// ResolveKey resolves a key that might be a canonical key or an alias.
// Returns the canonical key if found, along with resolution metadata.
// Case-insensitive alias lookup: tries uppercase first (PASM2 mnemonics),
// then lowercase (method names, pattern IDs).
// If key is not found and cooldown has passed, attempts one refresh before giving up.
func (m *Manager) ResolveKey(key string) KeyResolution {
	if err := m.EnsureIndex(); err != nil {
		return KeyResolution{}
	}

	m.mu.RLock()
	resolution := m.resolveKeyLocked(key)
	m.mu.RUnlock()

	if resolution.Found {
		return resolution
	}

	// Key not found - try refresh-on-error if cooldown has passed
	if m.tryErrorRefresh() {
		// Retry lookup after refresh
		m.mu.RLock()
		resolution = m.resolveKeyLocked(key)
		m.mu.RUnlock()
	}

	return resolution
}

// tryErrorRefresh attempts to refresh the index if the error cooldown has passed.
// Returns true if a refresh was attempted, false if still in cooldown.
// This method is safe for concurrent access.
func (m *Manager) tryErrorRefresh() bool {
	m.mu.RLock()
	lastError := m.lastErrorRefresh
	m.mu.RUnlock()

	if time.Since(lastError) < ErrorRefreshCooldown {
		return false // Still in cooldown
	}

	// Attempt refresh
	if err := m.Refresh(); err != nil {
		// Refresh failed, but still update timestamp to prevent retry storm
		m.mu.Lock()
		m.lastErrorRefresh = time.Now()
		m.mu.Unlock()
		return false
	}

	// Refresh succeeded, update timestamp
	m.mu.Lock()
	m.lastErrorRefresh = time.Now()
	m.mu.Unlock()
	return true
}

// resolveKeyLocked performs key resolution while holding the read lock.
// This is an internal helper for use by other methods that already hold the lock.
func (m *Manager) resolveKeyLocked(key string) KeyResolution {
	// Direct lookup first (case-insensitive)
	if _, ok := m.index.Files[key]; ok {
		return KeyResolution{CanonicalKey: key, Found: true}
	}
	// Try case-insensitive match on canonical keys
	keyLower := strings.ToLower(key)
	for canonicalKey := range m.index.Files {
		if strings.ToLower(canonicalKey) == keyLower {
			return KeyResolution{CanonicalKey: canonicalKey, Found: true}
		}
	}

	// Try alias lookup (case-insensitive)
	if m.index.Aliases != nil {
		// Try uppercase first (PASM2 mnemonics like ADD, MOV)
		if canonicals, ok := m.index.Aliases[strings.ToUpper(key)]; ok && len(canonicals) > 0 {
			// First entry wins for conflicts (per spec)
			if _, exists := m.index.Files[canonicals[0]]; exists {
				return KeyResolution{
					CanonicalKey: canonicals[0],
					ResolvedFrom: key,
					Found:        true,
				}
			}
		}
		// Try lowercase (method names, pattern IDs)
		if canonicals, ok := m.index.Aliases[strings.ToLower(key)]; ok && len(canonicals) > 0 {
			if _, exists := m.index.Files[canonicals[0]]; exists {
				return KeyResolution{
					CanonicalKey: canonicals[0],
					ResolvedFrom: key,
					Found:        true,
				}
			}
		}
		// Try exact case (as stored in index)
		if canonicals, ok := m.index.Aliases[key]; ok && len(canonicals) > 0 {
			if _, exists := m.index.Files[canonicals[0]]; exists {
				return KeyResolution{
					CanonicalKey: canonicals[0],
					ResolvedFrom: key,
					Found:        true,
				}
			}
		}
	}

	return KeyResolution{}
}

// GetKeyPath returns the path and mtime for a key.
// Supports both canonical keys and aliases.
func (m *Manager) GetKeyPath(key string) (string, int64, error) {
	if err := m.EnsureIndex(); err != nil {
		return "", 0, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	resolution := m.resolveKeyLocked(key)
	if !resolution.Found {
		return "", 0, fmt.Errorf("key not found: %s", key)
	}

	entry := m.index.Files[resolution.CanonicalKey]
	return entry.Path, entry.Mtime, nil
}

// KeyExists checks if a key exists in the index.
// Supports both canonical keys and aliases.
func (m *Manager) KeyExists(key string) bool {
	if err := m.EnsureIndex(); err != nil {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	resolution := m.resolveKeyLocked(key)
	return resolution.Found
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

// QueryMatch represents a key that matches a natural language query.
type QueryMatch struct {
	Key      string  `json:"key"`
	Score    float64 `json:"score"`
	Category string  `json:"category,omitempty"`
}

// MatchQuery finds keys matching a natural language query.
// Returns exact match if query is a valid key or alias, otherwise finds best matches.
// Query examples: "mov instruction", "pasm2 add", "spin2 pinwrite", "cog architecture"
// Also supports aliases: "ADD", "WAITMS", "motor_controller"
func (m *Manager) MatchQuery(query string) ([]QueryMatch, error) {
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for exact key or alias match first
	resolution := m.resolveKeyLocked(query)
	if resolution.Found {
		cat := m.getKeyCategory(resolution.CanonicalKey)
		return []QueryMatch{{Key: resolution.CanonicalKey, Score: 1.0, Category: cat}}, nil
	}

	// Tokenize the query
	queryTokens := tokenizeQuery(query)
	if len(queryTokens) == 0 {
		return nil, fmt.Errorf("empty query")
	}

	// Score each key
	var matches []QueryMatch
	for key := range m.index.Files {
		keyTokens := tokenizeKey(key)
		score := scoreMatch(queryTokens, keyTokens)
		if score > 0 {
			cat := m.getKeyCategory(key)
			matches = append(matches, QueryMatch{Key: key, Score: score, Category: cat})
		}
	}

	// Sort by score descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > 20 {
		matches = matches[:20]
	}

	return matches, nil
}

// getKeyCategory returns the first category a key belongs to (internal, no lock).
func (m *Manager) getKeyCategory(key string) string {
	for cat, keys := range m.index.Categories {
		for _, k := range keys {
			if k == key {
				return cat
			}
		}
	}
	return ""
}

// tokenizeKey splits a CamelCase key into lowercase tokens.
// "p2kbPasm2Mov" -> ["p2kb", "pasm2", "mov"]
// "p2kbArchCogMemory" -> ["p2kb", "arch", "cog", "memory"]
func tokenizeKey(key string) []string {
	var tokens []string
	var current strings.Builder

	for i, r := range key {
		// Start new token on uppercase letter
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(key[i-1])
			// Split on: lowercase->uppercase or digit->uppercase transitions
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
				if current.Len() > 0 {
					tokens = append(tokens, strings.ToLower(current.String()))
					current.Reset()
				}
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		tokens = append(tokens, strings.ToLower(current.String()))
	}

	return tokens
}

// tokenizeQuery splits a natural language query into lowercase tokens.
// "MOV instruction" -> ["mov", "instruction"]
// "spin2 pinwrite method" -> ["spin2", "pinwrite", "method"]
func tokenizeQuery(query string) []string {
	query = strings.ToLower(query)
	// Split on non-alphanumeric characters
	var tokens []string
	var current strings.Builder

	for _, r := range query {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// scoreMatch calculates how well query tokens match key tokens.
// Returns a score from 0 to 1, where 1 is a perfect match.
func scoreMatch(queryTokens, keyTokens []string) float64 {
	if len(queryTokens) == 0 || len(keyTokens) == 0 {
		return 0
	}

	// Skip common prefix "p2kb" in scoring
	filteredKeyTokens := keyTokens
	if len(keyTokens) > 0 && keyTokens[0] == "p2kb" {
		filteredKeyTokens = keyTokens[1:]
	}

	// Count matching tokens
	matches := 0
	for _, qt := range queryTokens {
		for _, kt := range filteredKeyTokens {
			// Exact match
			if qt == kt {
				matches++
				break
			}
			// Substring match (for partial words like "mov" matching "move")
			if len(qt) >= 3 && strings.Contains(kt, qt) {
				matches++
				break
			}
			// Prefix match
			if len(qt) >= 3 && strings.HasPrefix(kt, qt) {
				matches++
				break
			}
		}
	}

	if matches == 0 {
		return 0
	}

	// Score: ratio of matched query tokens, with bonus for more specific matches
	score := float64(matches) / float64(len(queryTokens))

	// Bonus for matching more key tokens
	if matches == len(queryTokens) && len(queryTokens) > 1 {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// GetAllKeys returns all keys in the index.
func (m *Manager) GetAllKeys() []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.index.Files))
	for k := range m.index.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetFileMtime returns the modification time for a key.
// Supports both canonical keys and aliases.
func (m *Manager) GetFileMtime(key string) (int64, error) {
	if err := m.EnsureIndex(); err != nil {
		return 0, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	resolution := m.resolveKeyLocked(key)
	if !resolution.Found {
		return 0, fmt.Errorf("key not found: %s", key)
	}

	entry := m.index.Files[resolution.CanonicalKey]
	return entry.Mtime, nil
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
// Category lookup is case-insensitive.
func (m *Manager) GetCategoryKeys(category string) ([]string, error) {
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try exact match first
	keys, ok := m.index.Categories[category]
	if !ok {
		// Try case-insensitive match
		categoryLower := strings.ToLower(category)
		for cat, catKeys := range m.index.Categories {
			if strings.ToLower(cat) == categoryLower {
				keys = catKeys
				ok = true
				break
			}
		}
	}
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
		TotalAliases:    m.index.System.TotalAliases,
	}
}

// GetIndexStatus returns index freshness information.
// This method releases the read lock before disk I/O for better concurrency.
func (m *Manager) GetIndexStatus() IndexStatus {
	// Get fields that require lock
	m.mu.RLock()
	ttl := m.ttl
	indexPath := m.indexPath
	var version string
	if m.index != nil {
		version = m.index.System.Version
	}
	m.mu.RUnlock() // Release lock BEFORE disk I/O

	status := IndexStatus{
		TTLSeconds:    int64(ttl.Seconds()),
		CacheFilePath: indexPath,
		Version:       version,
	}

	// Check disk cache - no lock needed for this operation
	info, err := os.Stat(indexPath)
	if err == nil {
		status.IsCached = true
		status.LastUpdated = info.ModTime()
		status.AgeSeconds = int64(time.Since(info.ModTime()).Seconds())
		status.NeedsRefresh = time.Since(info.ModTime()) > ttl
	} else {
		status.IsCached = false
		status.NeedsRefresh = true
	}

	return status
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

// fetchIndexData fetches the index from the remote URL and returns the parsed index and raw data.
// This method does NOT modify any state - it only performs network I/O and parsing.
// Caller is responsible for updating the index under appropriate locks.
func (m *Manager) fetchIndexData() (*Index, []byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", IndexURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Cache-busting headers to get fresh content
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("network error fetching index from %s: %w", IndexURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d fetching index from %s", resp.StatusCode, IndexURL)
	}

	// Decompress gzip
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decompress index: %w", err)
	}
	defer gr.Close()

	data, err := io.ReadAll(gr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read index data: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, nil, fmt.Errorf("failed to parse index JSON: %w", err)
	}

	return &idx, data, nil
}

// GetStaleKeys returns keys whose cached content is older than the index mtime.
// This allows smart cache invalidation based on actual file changes.
func (m *Manager) GetStaleKeys(cachedKeys []string, getCacheMtime func(key string) int64) []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var stale []string
	for _, key := range cachedKeys {
		entry, ok := m.index.Files[key]
		if !ok {
			// Key no longer in index, consider stale
			stale = append(stale, key)
			continue
		}

		cacheMtime := getCacheMtime(key)
		if cacheMtime > 0 && entry.Mtime > cacheMtime {
			// Index entry is newer than cache
			stale = append(stale, key)
		}
	}

	return stale
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
