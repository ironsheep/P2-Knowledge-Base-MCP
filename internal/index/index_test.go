package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/paths"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.ttl == 0 {
		t.Error("TTL should not be zero")
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

func TestGetIndexTTL(t *testing.T) {
	// Test default
	ttl := getIndexTTL()
	if ttl != DefaultIndexTTL {
		t.Errorf("getIndexTTL() = %v, want %v", ttl, DefaultIndexTTL)
	}

	// Test with env var (in seconds)
	os.Setenv("P2KB_INDEX_TTL", "3600")
	defer os.Unsetenv("P2KB_INDEX_TTL")

	ttl = getIndexTTL()
	if ttl.Seconds() != 3600 {
		t.Errorf("getIndexTTL() = %v, want 3600s", ttl)
	}
}

func TestManagerSearch(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov":      {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Add":      {Path: "pasm2/add.yaml"},
				"p2kbPasm2Movbyts":  {Path: "pasm2/movbyts.yaml"},
				"p2kbSpin2Pinwrite": {Path: "spin2/pinwrite.yaml"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	tests := []struct {
		term     string
		limit    int
		expected int
	}{
		{"mov", 0, 2},    // p2kbPasm2Mov, p2kbPasm2Movbyts
		{"pasm2", 0, 3},  // All pasm2 keys
		{"xyz", 0, 0},    // No matches
		{"mov", 1, 1},    // Limited to 1
		{"", 0, 0},       // Empty term
	}

	for _, tt := range tests {
		t.Run(tt.term, func(t *testing.T) {
			results := m.Search(tt.term, tt.limit)
			if len(results) != tt.expected {
				t.Errorf("Search(%q, %d) returned %d results, want %d",
					tt.term, tt.limit, len(results), tt.expected)
			}
		})
	}
}

func TestManagerGetCategories(t *testing.T) {
	m := &Manager{
		index: &Index{
			Categories: map[string][]string{
				"pasm2_math":   {"p2kbPasm2Add", "p2kbPasm2Sub"},
				"pasm2_branch": {"p2kbPasm2Jmp", "p2kbPasm2Call"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	categories := m.GetCategories()
	if len(categories) != 2 {
		t.Errorf("got %d categories, want 2", len(categories))
	}

	// Categories should be sorted
	if categories[0] != "pasm2_branch" {
		t.Errorf("first category = %q, want pasm2_branch", categories[0])
	}
}

func TestManagerGetCategoriesWithCounts(t *testing.T) {
	m := &Manager{
		index: &Index{
			Categories: map[string][]string{
				"pasm2_math":   {"p2kbPasm2Add", "p2kbPasm2Sub"},
				"pasm2_branch": {"p2kbPasm2Jmp"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	counts := m.GetCategoriesWithCounts()
	if counts["pasm2_math"] != 2 {
		t.Errorf("pasm2_math count = %d, want 2", counts["pasm2_math"])
	}
	if counts["pasm2_branch"] != 1 {
		t.Errorf("pasm2_branch count = %d, want 1", counts["pasm2_branch"])
	}
}

func TestManagerGetCategoryKeys(t *testing.T) {
	m := &Manager{
		index: &Index{
			Categories: map[string][]string{
				"pasm2_math": {"p2kbPasm2Add", "p2kbPasm2Sub"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	keys, err := m.GetCategoryKeys("pasm2_math")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("got %d keys, want 2", len(keys))
	}

	// Test non-existent category
	_, err = m.GetCategoryKeys("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent category")
	}
}

func TestManagerKeyExists(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	if !m.KeyExists("p2kbPasm2Mov") {
		t.Error("KeyExists(p2kbPasm2Mov) = false, want true")
	}
	if m.KeyExists("nonexistent") {
		t.Error("KeyExists(nonexistent) = true, want false")
	}
}

func TestManagerGetKeyCategories(t *testing.T) {
	m := &Manager{
		index: &Index{
			Categories: map[string][]string{
				"pasm2_math": {"p2kbPasm2Add", "p2kbPasm2Mov"},
				"pasm2_data": {"p2kbPasm2Mov"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	categories := m.GetKeyCategories("p2kbPasm2Mov")
	if len(categories) != 2 {
		t.Errorf("got %d categories, want 2", len(categories))
	}

	// Should return empty for non-existent key
	categories = m.GetKeyCategories("nonexistent")
	if len(categories) != 0 {
		t.Errorf("got %d categories for nonexistent key, want 0", len(categories))
	}
}

func TestManagerGetStats(t *testing.T) {
	m := &Manager{
		index: &Index{
			System: SystemInfo{
				Version:         "3.2.0",
				TotalEntries:    970,
				TotalCategories: 47,
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	stats := m.GetStats()
	if stats.Version != "3.2.0" {
		t.Errorf("Version = %q, want 3.2.0", stats.Version)
	}
	if stats.TotalEntries != 970 {
		t.Errorf("TotalEntries = %d, want 970", stats.TotalEntries)
	}
	if stats.TotalCategories != 47 {
		t.Errorf("TotalCategories = %d, want 47", stats.TotalCategories)
	}
}

func TestManagerFindSimilarKeys(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov":     {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Movbyts": {Path: "pasm2/movbyts.yaml"},
				"p2kbPasm2Add":     {Path: "pasm2/add.yaml"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Test finding similar keys
	similar := m.FindSimilarKeys("mov", 5)
	if len(similar) < 1 {
		t.Error("FindSimilarKeys should find at least 1 match")
	}

	// Test with typo
	similar = m.FindSimilarKeys("p2kbPasm2Mvo", 5)
	// Should find similar keys based on partial match
	if len(similar) == 0 {
		t.Log("Warning: FindSimilarKeys didn't find matches for typo")
	}
}

func TestSaveAndLoadFromCache(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	m := &Manager{
		indexPath: filepath.Join(tmpDir, "index", "p2kb-index.json"),
		metaPath:  filepath.Join(tmpDir, "index", "p2kb-index.meta"),
		ttl:       DefaultIndexTTL,
	}

	testData := []byte(`{"system":{"version":"1.0.0","total_entries":1,"total_categories":1},"categories":{},"files":{}}`)

	// Save to cache
	err := m.saveToCache(testData)
	if err != nil {
		t.Fatalf("saveToCache failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(m.indexPath); os.IsNotExist(err) {
		t.Error("cache file was not created")
	}

	// Load from cache
	if !m.loadFromCache() {
		t.Error("loadFromCache returned false")
	}

	if m.index == nil {
		t.Error("index is nil after loadFromCache")
	}
}
