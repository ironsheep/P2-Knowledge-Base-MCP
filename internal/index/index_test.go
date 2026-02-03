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

func TestGetIndexStatus(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		indexPath: filepath.Join(tmpDir, "index", "p2kb-index.json"),
		metaPath:  filepath.Join(tmpDir, "index", "p2kb-index.meta"),
		ttl:       DefaultIndexTTL,
	}

	// Test without cached index
	status := m.GetIndexStatus()

	if status.IsCached {
		t.Error("IsCached should be false for non-existent cache")
	}
	if !status.NeedsRefresh {
		t.Error("NeedsRefresh should be true for non-existent cache")
	}
	if status.TTLSeconds != int64(DefaultIndexTTL.Seconds()) {
		t.Errorf("TTLSeconds = %d, want %d", status.TTLSeconds, int64(DefaultIndexTTL.Seconds()))
	}
	if status.CacheFilePath != m.indexPath {
		t.Errorf("CacheFilePath = %q, want %q", status.CacheFilePath, m.indexPath)
	}

	// Create a cached index
	testData := []byte(`{"system":{"version":"2.0.0","total_entries":100,"total_categories":10},"categories":{},"files":{}}`)
	err := m.saveToCache(testData)
	if err != nil {
		t.Fatalf("saveToCache failed: %v", err)
	}

	// Load the index to get version
	m.loadFromCache()

	// Test with cached index
	status = m.GetIndexStatus()

	if !status.IsCached {
		t.Error("IsCached should be true after saving cache")
	}
	if status.NeedsRefresh {
		t.Error("NeedsRefresh should be false for fresh cache")
	}
	if status.AgeSeconds < 0 {
		t.Errorf("AgeSeconds = %d, should be >= 0", status.AgeSeconds)
	}
	if status.Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", status.Version)
	}
}

func TestGetIndexStatusExpired(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		indexPath: filepath.Join(tmpDir, "index", "p2kb-index.json"),
		metaPath:  filepath.Join(tmpDir, "index", "p2kb-index.meta"),
		ttl:       1 * time.Nanosecond, // Very short TTL for testing
	}

	// Create a cached index
	testData := []byte(`{"system":{"version":"1.0.0","total_entries":1,"total_categories":1},"categories":{},"files":{}}`)
	err := m.saveToCache(testData)
	if err != nil {
		t.Fatalf("saveToCache failed: %v", err)
	}

	// Wait a bit for TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Test with expired cache
	status := m.GetIndexStatus()

	if !status.IsCached {
		t.Error("IsCached should be true even when expired")
	}
	if !status.NeedsRefresh {
		t.Error("NeedsRefresh should be true for expired cache")
	}
}

// Tests for natural language query matching

func TestTokenizeKey(t *testing.T) {
	tests := []struct {
		key      string
		expected []string
	}{
		{"p2kbPasm2Mov", []string{"p2kb", "pasm2", "mov"}},
		{"p2kbArchCogMemory", []string{"p2kb", "arch", "cog", "memory"}},
		{"p2kbSpin2Pinwrite", []string{"p2kb", "spin2", "pinwrite"}},
		{"simple", []string{"simple"}},
		{"ABC", []string{"abc"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := tokenizeKey(tt.key)
		if len(result) != len(tt.expected) {
			t.Errorf("tokenizeKey(%q) returned %d tokens, want %d: %v",
				tt.key, len(result), len(tt.expected), result)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("tokenizeKey(%q)[%d] = %q, want %q", tt.key, i, v, tt.expected[i])
			}
		}
	}
}

func TestTokenizeQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected []string
	}{
		{"MOV instruction", []string{"mov", "instruction"}},
		{"spin2 pinwrite method", []string{"spin2", "pinwrite", "method"}},
		{"cog memory", []string{"cog", "memory"}},
		{"  spaces  and  punctuation!  ", []string{"spaces", "and", "punctuation"}},
		{"p2kb", []string{"p2kb"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		result := tokenizeQuery(tt.query)
		if len(result) != len(tt.expected) {
			t.Errorf("tokenizeQuery(%q) returned %d tokens, want %d: %v",
				tt.query, len(result), len(tt.expected), result)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("tokenizeQuery(%q)[%d] = %q, want %q", tt.query, i, v, tt.expected[i])
			}
		}
	}
}

func TestScoreMatch(t *testing.T) {
	tests := []struct {
		queryTokens []string
		keyTokens   []string
		minScore    float64
		maxScore    float64
	}{
		// Exact match should score high
		{[]string{"mov"}, []string{"p2kb", "pasm2", "mov"}, 0.9, 1.0},
		// Multiple token match should score higher
		{[]string{"pasm2", "mov"}, []string{"p2kb", "pasm2", "mov"}, 0.9, 1.1},
		// Partial match
		{[]string{"cog"}, []string{"p2kb", "arch", "cog", "memory"}, 0.9, 1.0},
		// No match
		{[]string{"xyz"}, []string{"p2kb", "pasm2", "mov"}, 0, 0},
		// Empty should score 0
		{[]string{}, []string{"p2kb", "mov"}, 0, 0},
		{[]string{"mov"}, []string{}, 0, 0},
	}

	for i, tt := range tests {
		score := scoreMatch(tt.queryTokens, tt.keyTokens)
		if score < tt.minScore || score > tt.maxScore {
			t.Errorf("test %d: scoreMatch(%v, %v) = %f, want between %f and %f",
				i, tt.queryTokens, tt.keyTokens, score, tt.minScore, tt.maxScore)
		}
	}
}

func TestMatchQuery(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov":      {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Add":      {Path: "pasm2/add.yaml"},
				"p2kbPasm2Movbyts":  {Path: "pasm2/movbyts.yaml"},
				"p2kbSpin2Pinwrite": {Path: "spin2/pinwrite.yaml"},
				"p2kbArchCog":       {Path: "arch/cog.yaml"},
			},
			Categories: map[string][]string{
				"pasm2_math": {"p2kbPasm2Add"},
				"pasm2_data": {"p2kbPasm2Mov", "p2kbPasm2Movbyts"},
				"spin2_pin":  {"p2kbSpin2Pinwrite"},
				"arch":       {"p2kbArchCog"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	tests := []struct {
		query       string
		expectMatch bool
		expectKey   string
	}{
		// Exact key match
		{"p2kbPasm2Mov", true, "p2kbPasm2Mov"},
		// Natural language should find mov
		{"mov instruction", true, ""},
		// Multiple tokens
		{"pasm2 mov", true, ""},
		// Should find pinwrite
		{"spin2 pinwrite", true, "p2kbSpin2Pinwrite"},
		// Should find cog architecture
		{"cog", true, ""},
		// No match
		{"xyz nonexistent", false, ""},
	}

	for _, tt := range tests {
		matches, err := m.MatchQuery(tt.query)
		if tt.expectMatch {
			if err != nil {
				t.Errorf("MatchQuery(%q) error: %v", tt.query, err)
				continue
			}
			if len(matches) == 0 {
				t.Errorf("MatchQuery(%q) returned no matches", tt.query)
				continue
			}
			if tt.expectKey != "" && matches[0].Key != tt.expectKey {
				t.Errorf("MatchQuery(%q) top match = %q, want %q", tt.query, matches[0].Key, tt.expectKey)
			}
		} else {
			if len(matches) > 0 {
				t.Errorf("MatchQuery(%q) expected no matches, got %d", tt.query, len(matches))
			}
		}
	}
}

func TestGetAllKeys(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	keys := m.GetAllKeys()
	if len(keys) != 2 {
		t.Errorf("GetAllKeys returned %d keys, want 2", len(keys))
	}

	// Should be sorted
	if keys[0] != "p2kbPasm2Add" {
		t.Errorf("keys[0] = %q, want p2kbPasm2Add", keys[0])
	}
}

func TestGetFileMtime(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml", Mtime: 1234567890},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	mtime, err := m.GetFileMtime("p2kbPasm2Mov")
	if err != nil {
		t.Fatalf("GetFileMtime error: %v", err)
	}
	if mtime != 1234567890 {
		t.Errorf("mtime = %d, want 1234567890", mtime)
	}

	// Non-existent key
	_, err = m.GetFileMtime("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}
}

func TestGetStaleKeys(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"key1": {Path: "key1.yaml", Mtime: 2000},
				"key2": {Path: "key2.yaml", Mtime: 1000},
				"key3": {Path: "key3.yaml", Mtime: 3000},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	cachedKeys := []string{"key1", "key2", "key3", "removed"}
	getMtime := func(key string) int64 {
		switch key {
		case "key1":
			return 1500 // stale: index has 2000
		case "key2":
			return 1000 // fresh: same as index
		case "key3":
			return 3500 // fresh: newer than index
		default:
			return 0
		}
	}

	stale := m.GetStaleKeys(cachedKeys, getMtime)

	// Should find key1 as stale and "removed" as no longer in index
	if len(stale) != 2 {
		t.Errorf("GetStaleKeys returned %d keys, want 2: %v", len(stale), stale)
	}

	hasKey1 := false
	hasRemoved := false
	for _, k := range stale {
		if k == "key1" {
			hasKey1 = true
		}
		if k == "removed" {
			hasRemoved = true
		}
	}
	if !hasKey1 {
		t.Error("stale keys should include key1")
	}
	if !hasRemoved {
		t.Error("stale keys should include removed")
	}
}

// Tests for alias resolution (v1.3.1+)

func TestResolveKeyDirect(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml"},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
				"MOV": {"p2kbPasm2Mov"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Direct canonical key lookup
	resolution := m.ResolveKey("p2kbPasm2Add")
	if !resolution.Found {
		t.Error("ResolveKey should find canonical key")
	}
	if resolution.CanonicalKey != "p2kbPasm2Add" {
		t.Errorf("CanonicalKey = %q, want p2kbPasm2Add", resolution.CanonicalKey)
	}
	if resolution.ResolvedFrom != "" {
		t.Errorf("ResolvedFrom should be empty for direct match, got %q", resolution.ResolvedFrom)
	}
}

func TestResolveKeyAliasUppercase(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Alias lookup with exact case
	resolution := m.ResolveKey("ADD")
	if !resolution.Found {
		t.Error("ResolveKey should find uppercase alias")
	}
	if resolution.CanonicalKey != "p2kbPasm2Add" {
		t.Errorf("CanonicalKey = %q, want p2kbPasm2Add", resolution.CanonicalKey)
	}
	if resolution.ResolvedFrom != "ADD" {
		t.Errorf("ResolvedFrom = %q, want ADD", resolution.ResolvedFrom)
	}
}

func TestResolveKeyAliasLowercase(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Case-insensitive alias lookup (lowercase input)
	resolution := m.ResolveKey("add")
	if !resolution.Found {
		t.Error("ResolveKey should find alias case-insensitively")
	}
	if resolution.CanonicalKey != "p2kbPasm2Add" {
		t.Errorf("CanonicalKey = %q, want p2kbPasm2Add", resolution.CanonicalKey)
	}
	if resolution.ResolvedFrom != "add" {
		t.Errorf("ResolvedFrom = %q, want add", resolution.ResolvedFrom)
	}
}

func TestResolveKeyAliasMixedCase(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbSpin2Waitms": {Path: "spin2/waitms.yaml"},
			},
			Aliases: map[string][]string{
				"WAITMS": {"p2kbSpin2Waitms"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Mixed case input should resolve
	resolution := m.ResolveKey("WaitMs")
	if !resolution.Found {
		t.Error("ResolveKey should find alias with mixed case input")
	}
	if resolution.CanonicalKey != "p2kbSpin2Waitms" {
		t.Errorf("CanonicalKey = %q, want p2kbSpin2Waitms", resolution.CanonicalKey)
	}
}

func TestResolveKeyNotFound(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
			},
		},
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now(), // Prevent refresh-on-error during test
	}

	// Unknown key
	resolution := m.ResolveKey("NOTAKEY")
	if resolution.Found {
		t.Error("ResolveKey should not find unknown key")
	}
	if resolution.CanonicalKey != "" {
		t.Errorf("CanonicalKey should be empty for unknown key, got %q", resolution.CanonicalKey)
	}
}

func TestResolveKeyNoAliases(t *testing.T) {
	// Test with old-format index (no aliases)
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: nil, // No aliases section
		},
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now(), // Prevent refresh-on-error during test
	}

	// Direct key should still work
	resolution := m.ResolveKey("p2kbPasm2Add")
	if !resolution.Found {
		t.Error("ResolveKey should find canonical key even without aliases")
	}

	// Alias lookup should gracefully fail
	resolution = m.ResolveKey("ADD")
	if resolution.Found {
		t.Error("ResolveKey should not find alias when aliases section is nil")
	}
}

func TestKeyExistsWithAlias(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Canonical key
	if !m.KeyExists("p2kbPasm2Add") {
		t.Error("KeyExists should return true for canonical key")
	}

	// Alias
	if !m.KeyExists("ADD") {
		t.Error("KeyExists should return true for alias")
	}

	// Case-insensitive alias
	if !m.KeyExists("add") {
		t.Error("KeyExists should return true for lowercase alias")
	}

	// Unknown
	if m.KeyExists("NOTAKEY") {
		t.Error("KeyExists should return false for unknown key")
	}
}

func TestGetKeyPathWithAlias(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml", Mtime: 1234567890},
			},
			Aliases: map[string][]string{
				"ADD": {"p2kbPasm2Add"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// Via alias
	path, mtime, err := m.GetKeyPath("ADD")
	if err != nil {
		t.Fatalf("GetKeyPath via alias failed: %v", err)
	}
	if path != "pasm2/add.yaml" {
		t.Errorf("path = %q, want pasm2/add.yaml", path)
	}
	if mtime != 1234567890 {
		t.Errorf("mtime = %d, want 1234567890", mtime)
	}

	// Via lowercase alias
	path, _, err = m.GetKeyPath("add")
	if err != nil {
		t.Fatalf("GetKeyPath via lowercase alias failed: %v", err)
	}
	if path != "pasm2/add.yaml" {
		t.Errorf("path = %q, want pasm2/add.yaml", path)
	}
}

func TestGetFileMtimeWithAlias(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml", Mtime: 9876543210},
			},
			Aliases: map[string][]string{
				"MOV": {"p2kbPasm2Mov"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	mtime, err := m.GetFileMtime("MOV")
	if err != nil {
		t.Fatalf("GetFileMtime via alias failed: %v", err)
	}
	if mtime != 9876543210 {
		t.Errorf("mtime = %d, want 9876543210", mtime)
	}
}

func TestMatchQueryWithAlias(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add":      {Path: "pasm2/add.yaml"},
				"p2kbSpin2Waitms":   {Path: "spin2/waitms.yaml"},
				"p2kbSpin2Pinwrite": {Path: "spin2/pinwrite.yaml"},
			},
			Categories: map[string][]string{
				"pasm2_math": {"p2kbPasm2Add"},
				"spin2_time": {"p2kbSpin2Waitms"},
			},
			Aliases: map[string][]string{
				"ADD":      {"p2kbPasm2Add"},
				"WAITMS":   {"p2kbSpin2Waitms"},
				"PINWRITE": {"p2kbSpin2Pinwrite"},
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	tests := []struct {
		query     string
		expectKey string
	}{
		{"ADD", "p2kbPasm2Add"},
		{"add", "p2kbPasm2Add"},
		{"WAITMS", "p2kbSpin2Waitms"},
		{"waitms", "p2kbSpin2Waitms"},
		{"PINWRITE", "p2kbSpin2Pinwrite"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			matches, err := m.MatchQuery(tt.query)
			if err != nil {
				t.Fatalf("MatchQuery(%q) error: %v", tt.query, err)
			}
			if len(matches) == 0 {
				t.Fatalf("MatchQuery(%q) returned no matches", tt.query)
			}
			if matches[0].Key != tt.expectKey {
				t.Errorf("MatchQuery(%q) top match = %q, want %q", tt.query, matches[0].Key, tt.expectKey)
			}
			if matches[0].Score != 1.0 {
				t.Errorf("MatchQuery(%q) score = %f, want 1.0 for alias match", tt.query, matches[0].Score)
			}
		})
	}
}

func TestGetStatsWithAliases(t *testing.T) {
	m := &Manager{
		index: &Index{
			System: SystemInfo{
				Version:         "3.3.0",
				TotalEntries:    970,
				TotalCategories: 47,
				TotalAliases:    933,
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	stats := m.GetStats()
	if stats.TotalAliases != 933 {
		t.Errorf("TotalAliases = %d, want 933", stats.TotalAliases)
	}
}

func TestResolveKeyAliasConflict(t *testing.T) {
	// Test that first entry wins for conflicting aliases (e.g., ABS exists in both PASM2 and Spin2)
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Abs": {Path: "pasm2/abs.yaml"},
				"p2kbSpin2Abs": {Path: "spin2/abs.yaml"},
			},
			Aliases: map[string][]string{
				"ABS": {"p2kbPasm2Abs", "p2kbSpin2Abs"}, // PASM2 first
			},
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	// First entry (PASM2) should win
	resolution := m.ResolveKey("ABS")
	if !resolution.Found {
		t.Error("ResolveKey should find conflicting alias")
	}
	if resolution.CanonicalKey != "p2kbPasm2Abs" {
		t.Errorf("CanonicalKey = %q, want p2kbPasm2Abs (first entry wins)", resolution.CanonicalKey)
	}
}

// Test case-insensitive canonical key lookups (v1.3.3+)
func TestResolveKeyCaseInsensitiveCanonical(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: nil, // No aliases to ensure we're testing canonical key lookup
		},
		lastRefresh: time.Now(),
		ttl:         DefaultIndexTTL,
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"p2kbPasm2Mov", "p2kbPasm2Mov"},     // Exact case
		{"P2KBPASM2MOV", "p2kbPasm2Mov"},     // All uppercase
		{"p2kbpasm2mov", "p2kbPasm2Mov"},     // All lowercase
		{"P2kbPasm2Mov", "p2kbPasm2Mov"},     // Mixed case
		{"p2kbPASM2mov", "p2kbPasm2Mov"},     // Another mixed case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			resolution := m.ResolveKey(tt.input)
			if !resolution.Found {
				t.Errorf("ResolveKey(%q) should find key case-insensitively", tt.input)
				return
			}
			if resolution.CanonicalKey != tt.expected {
				t.Errorf("ResolveKey(%q) = %q, want %q", tt.input, resolution.CanonicalKey, tt.expected)
			}
		})
	}
}

func TestGetCategoryKeysCaseInsensitive(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Mov": {Path: "pasm2/mov.yaml"},
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Categories: map[string][]string{
				"pasm2_data": {"p2kbPasm2Mov"},
				"pasm2_math": {"p2kbPasm2Add"},
			},
		},
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now(), // Prevent refresh-on-error during test
	}

	tests := []struct {
		category string
		wantLen  int
	}{
		{"pasm2_data", 1},      // Exact case
		{"PASM2_DATA", 1},      // All uppercase
		{"Pasm2_Data", 1},      // Mixed case
		{"PASM2_MATH", 1},      // Another category
		{"nonexistent", 0},     // Should fail
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			keys, err := m.GetCategoryKeys(tt.category)
			if tt.wantLen == 0 {
				if err == nil {
					t.Errorf("GetCategoryKeys(%q) should return error for nonexistent", tt.category)
				}
				return
			}
			if err != nil {
				t.Errorf("GetCategoryKeys(%q) error: %v", tt.category, err)
				return
			}
			if len(keys) != tt.wantLen {
				t.Errorf("GetCategoryKeys(%q) returned %d keys, want %d", tt.category, len(keys), tt.wantLen)
			}
		})
	}
}

// Tests for refresh-on-error feature (v1.3.3+)

func TestTryErrorRefreshCooldown(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
		},
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now(), // Recent error refresh - should be in cooldown
	}

	// Should return false because we're in cooldown
	if m.tryErrorRefresh() {
		t.Error("tryErrorRefresh should return false when in cooldown")
	}
}

func TestTryErrorRefreshCooldownExpired(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
		},
		indexPath:        "/nonexistent/path/index.json", // Ensure refresh will fail (no network in tests)
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now().Add(-10 * time.Minute), // Old error refresh - cooldown expired
	}

	// Should attempt refresh (will fail due to no network, but that's OK)
	// After this, lastErrorRefresh should be updated
	originalTime := m.lastErrorRefresh
	_ = m.tryErrorRefresh()

	// Verify cooldown timestamp was updated (whether refresh succeeded or failed)
	if !m.lastErrorRefresh.After(originalTime) {
		t.Error("tryErrorRefresh should update lastErrorRefresh timestamp")
	}
}

func TestResolveKeyTriggersErrorRefresh(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: nil,
		},
		indexPath:        "/nonexistent/path/index.json", // Ensure refresh will fail
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now().Add(-10 * time.Minute), // Cooldown expired
	}

	originalTime := m.lastErrorRefresh

	// Try to resolve a key that doesn't exist
	resolution := m.ResolveKey("NONEXISTENT")
	if resolution.Found {
		t.Error("Should not find nonexistent key")
	}

	// Verify that an error refresh was attempted (timestamp updated)
	if !m.lastErrorRefresh.After(originalTime) {
		t.Error("ResolveKey should trigger error refresh when key not found and cooldown expired")
	}
}

func TestResolveKeyDoesNotTriggerRefreshInCooldown(t *testing.T) {
	m := &Manager{
		index: &Index{
			Files: map[string]FileEntry{
				"p2kbPasm2Add": {Path: "pasm2/add.yaml"},
			},
			Aliases: nil,
		},
		indexPath:        "/nonexistent/path/index.json",
		lastRefresh:      time.Now(),
		ttl:              DefaultIndexTTL,
		lastErrorRefresh: time.Now(), // Recent - in cooldown
	}

	originalTime := m.lastErrorRefresh

	// Try to resolve a key that doesn't exist
	resolution := m.ResolveKey("NONEXISTENT")
	if resolution.Found {
		t.Error("Should not find nonexistent key")
	}

	// Verify that no refresh was attempted (timestamp unchanged)
	if m.lastErrorRefresh != originalTime {
		t.Error("ResolveKey should NOT trigger error refresh when in cooldown")
	}
}
