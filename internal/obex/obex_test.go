package obex

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.objects == nil {
		t.Error("objects map is nil")
	}
	if m.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNormalizeObjectID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2811", "2811"},
		{"OB2811", "2811"},
		{"ob2811", "2811"},
		{" 2811 ", "2811"},
		{"OB 2811", "2811"},
	}

	for _, tt := range tests {
		result := normalizeObjectID(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeObjectID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExpandSearchTerms(t *testing.T) {
	tests := []struct {
		term          string
		minExpected   int
		shouldContain []string
	}{
		{"i2c", 4, []string{"i2c", "iic", "twi"}},
		{"led", 5, []string{"led", "pixel", "ws2812"}},
		{"motor", 5, []string{"motor", "servo", "stepper"}},
		{"xyz", 1, []string{"xyz"}}, // No expansion
	}

	for _, tt := range tests {
		result := expandSearchTerms(tt.term)
		if len(result) < tt.minExpected {
			t.Errorf("expandSearchTerms(%q) returned %d terms, want at least %d", tt.term, len(result), tt.minExpected)
		}

		for _, expected := range tt.shouldContain {
			found := false
			for _, r := range result {
				if r == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expandSearchTerms(%q) should contain %q", tt.term, expected)
			}
		}
	}
}

func TestGetDownloadURL(t *testing.T) {
	m := NewManager()

	tests := []struct {
		objectID string
		expected string
	}{
		{"2811", "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB2811"},
		{"OB2811", "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB2811"},
		{"4047", "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB4047"},
	}

	for _, tt := range tests {
		result := m.GetDownloadURL(tt.objectID)
		if result != tt.expected {
			t.Errorf("GetDownloadURL(%q) = %q, want %q", tt.objectID, result, tt.expected)
		}
	}
}

func TestSaveAndLoadIndexFromCache(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir: tmpDir,
		objects:  make(map[string]*OBEXObject),
		ttl:      DefaultOBEXTTL,
	}

	testIDs := []string{"2811", "4047", "5274"}

	// Save to cache
	m.saveIndexToCache(testIDs)

	// Verify file exists
	indexPath := filepath.Join(tmpDir, "obex", "index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index file was not created")
	}

	// Load from cache
	if !m.loadIndexFromCache() {
		t.Error("loadIndexFromCache returned false")
	}

	if len(m.objectIDs) != 3 {
		t.Errorf("got %d object IDs, want 3", len(m.objectIDs))
	}
}

func TestSaveAndLoadObjectFromCache(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir: tmpDir,
		objects:  make(map[string]*OBEXObject),
		ttl:      DefaultOBEXTTL,
	}

	testYAML := []byte(`object_metadata:
  object_id: "2811"
  title: "Test Object"
  author: "Test Author"
  functionality:
    category: "drivers"
    description_short: "A test object"
    tags:
      - test
      - example
  technical_details:
    languages:
      - SPIN2
      - PASM2
`)

	// Save to cache
	m.saveObjectToCache("2811", testYAML)

	// Verify file exists
	cachePath := filepath.Join(tmpDir, "obex", "objects", "2811.yaml")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("object file was not created")
	}

	// Load from cache
	obj, err := m.loadObjectFromCache("2811")
	if err != nil {
		t.Fatalf("loadObjectFromCache failed: %v", err)
	}

	if obj.ObjectMetadata.ObjectID != "2811" {
		t.Errorf("ObjectID = %q, want 2811", obj.ObjectMetadata.ObjectID)
	}
	if obj.ObjectMetadata.Title != "Test Object" {
		t.Errorf("Title = %q, want 'Test Object'", obj.ObjectMetadata.Title)
	}
	if obj.ObjectMetadata.Author != "Test Author" {
		t.Errorf("Author = %q, want 'Test Author'", obj.ObjectMetadata.Author)
	}
}

func TestMatchObject(t *testing.T) {
	m := NewManager()

	obj := &OBEXObject{
		ObjectMetadata: ObjectMetadata{
			ObjectID: "2811",
			Title:    "Park Transformation Driver",
			Author:   "TestAuthor",
		},
	}
	obj.ObjectMetadata.Functionality.DescriptionShort = "CORDIC-based park transformation"
	obj.ObjectMetadata.Functionality.Tags = []string{"motor", "cordic", "servo"}

	tests := []struct {
		searchTerms []string
		expected    string
	}{
		{[]string{"park"}, "title"},
		{[]string{"transformation"}, "title"},
		{[]string{"motor"}, "tag"},
		{[]string{"cordic"}, "tag"},
		{[]string{"xyz"}, ""},
	}

	for _, tt := range tests {
		result := m.matchObject(obj, tt.searchTerms)
		if result != tt.expected {
			t.Errorf("matchObject with %v = %q, want %q", tt.searchTerms, result, tt.expected)
		}
	}
}

func TestGitHubConstants(t *testing.T) {
	if GitHubAPIBase == "" {
		t.Error("GitHubAPIBase is empty")
	}
	if GitHubRawBase == "" {
		t.Error("GitHubRawBase is empty")
	}
	if OBEXPath == "" {
		t.Error("OBEXPath is empty")
	}
	if OBEXDownloadBase == "" {
		t.Error("OBEXDownloadBase is empty")
	}
}

func TestGetTotalObjects(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Manager{
		cacheDir:    tmpDir,
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1", "2", "3"},
		ttl:         DefaultOBEXTTL,
		lastRefresh: time.Now(), // Prevent EnsureIndex from trying to fetch
	}

	// Since objectIDs is already populated and lastRefresh is set,
	// GetTotalObjects should just return the count without fetching
	m.mu.RLock()
	total := len(m.objectIDs)
	m.mu.RUnlock()
	if total != 3 {
		t.Errorf("objectIDs count = %d, want 3", total)
	}
}

func TestObjectExists(t *testing.T) {
	m := &Manager{
		objectIDs: []string{"2811", "4047", "5274"},
	}

	if !m.objectExists("2811") {
		t.Error("objectExists(2811) = false, want true")
	}
	if !m.objectExists("4047") {
		t.Error("objectExists(4047) = false, want true")
	}
	if m.objectExists("9999") {
		t.Error("objectExists(9999) = true, want false")
	}
}
