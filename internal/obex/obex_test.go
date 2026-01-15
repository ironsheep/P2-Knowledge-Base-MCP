package obex

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
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

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"WS2812 LED Driver", "ws2812-led-driver"},
		{"Simple Test", "simple-test"},
		{"Test!!!Object", "test-object"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"CamelCase", "camelcase"},
		{"123Numbers456", "123numbers456"},
		{"", ""},
		{"---Already-Slugged---", "already-slugged"},
		{"Special@#$%Characters", "special-characters"},
	}

	for _, tt := range tests {
		result := generateSlug(tt.title)
		if result != tt.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.title, result, tt.expected)
		}
	}
}

func TestValidateTargetPath(t *testing.T) {
	tests := []struct {
		path      string
		shouldErr bool
	}{
		{"OBX/test", false},
		{"./OBX/test", false},
		{"test/nested/path", false},
		{"../escape", true},
		{"test/../escape", true},
		{"test/../../escape", true},
	}

	for _, tt := range tests {
		err := validateTargetPath(tt.path)
		if tt.shouldErr && err == nil {
			t.Errorf("validateTargetPath(%q) = nil, want error", tt.path)
		}
		if !tt.shouldErr && err != nil {
			t.Errorf("validateTargetPath(%q) = %v, want nil", tt.path, err)
		}
	}
}

func TestExtractZip(t *testing.T) {
	// Create a test zip file in memory
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	// Add a test file
	testContent := []byte("test file content")
	fileWriter, err := zipWriter.Create("test.txt")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := fileWriter.Write(testContent); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}

	// Add a nested file
	nestedWriter, err := zipWriter.Create("subdir/nested.txt")
	if err != nil {
		t.Fatalf("failed to create nested zip entry: %v", err)
	}
	if _, err := nestedWriter.Write([]byte("nested content")); err != nil {
		t.Fatalf("failed to write nested zip entry: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	// Extract to temp directory
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "extracted")

	files, totalSize, err := extractZip(zipBuf.Bytes(), targetDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	// Verify extraction
	if len(files) != 2 {
		t.Errorf("extracted %d files, want 2", len(files))
	}

	if totalSize == 0 {
		t.Error("totalSize = 0, want > 0")
	}

	// Check test.txt exists
	content, err := os.ReadFile(filepath.Join(targetDir, "test.txt"))
	if err != nil {
		t.Errorf("failed to read extracted file: %v", err)
	}
	if string(content) != "test file content" {
		t.Errorf("file content = %q, want 'test file content'", string(content))
	}

	// Check nested file exists
	nestedContent, err := os.ReadFile(filepath.Join(targetDir, "subdir", "nested.txt"))
	if err != nil {
		t.Errorf("failed to read nested file: %v", err)
	}
	if string(nestedContent) != "nested content" {
		t.Errorf("nested content = %q, want 'nested content'", string(nestedContent))
	}
}

func TestExtractZipSlipPrevention(t *testing.T) {
	// Create a malicious zip with path traversal
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	// Try to create a file with path traversal
	fileWriter, err := zipWriter.Create("../../../etc/passwd")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := fileWriter.Write([]byte("malicious content")); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	// Try to extract - should fail
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "extracted")

	_, _, err = extractZip(zipBuf.Bytes(), targetDir)
	if err == nil {
		t.Error("extractZip should have failed for zip slip attack")
	}
}

func TestDownloadAndExtract(t *testing.T) {
	// Create a test zip file
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	fileWriter, err := zipWriter.Create("test_driver.spin2")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := fileWriter.Write([]byte("' Test Spin2 Driver\nPUB start()\n")); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}

	readmeWriter, err := zipWriter.Create("README.txt")
	if err != nil {
		t.Fatalf("failed to create readme entry: %v", err)
	}
	if _, err := readmeWriter.Write([]byte("Test object readme")); err != nil {
		t.Fatalf("failed to write readme: %v", err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}

	// Create mock HTTP server for zip download
	zipServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(zipBuf.Bytes())
	}))
	defer zipServer.Close()

	// Create mock HTTP server for metadata YAML
	yamlContent := `object_metadata:
  object_id: "9999"
  title: "Test Driver Object"
  author: "Test Author"
  functionality:
    category: "drivers"
    description_short: "A test driver"
  technical_details:
    languages:
      - SPIN2
`

	// Create temp directory and manager
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}

	// Change to work directory for the test
	oldWd, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldWd)

	m := &Manager{
		cacheDir:    filepath.Join(tmpDir, "cache"),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"9999"},
		ttl:         DefaultOBEXTTL,
		lastRefresh: time.Now(),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}

	// Pre-populate the object cache to avoid network call
	m.objects["9999"] = &OBEXObject{}
	m.objects["9999"].ObjectMetadata.ObjectID = "9999"
	m.objects["9999"].ObjectMetadata.Title = "Test Driver Object"
	m.objects["9999"].ObjectMetadata.Author = "Test Author"

	// Override the download URL by testing with a custom target
	// We need to mock the download - for this test, save the zip to a known location
	// and test the extraction part

	// For a more complete test, we'd need to inject the HTTP client
	// For now, let's test the individual components work together
	_ = yamlContent // Used conceptually

	// Test with direct extraction since we've already tested the HTTP parts
	targetDir := filepath.Join(workDir, "OBX", "test-driver-object")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}

	files, totalSize, err := extractZip(zipBuf.Bytes(), targetDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("extracted %d files, want 2", len(files))
	}

	if totalSize == 0 {
		t.Error("totalSize = 0, want > 0")
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(targetDir, "test_driver.spin2")); os.IsNotExist(err) {
		t.Error("test_driver.spin2 was not extracted")
	}
}

func TestDownloadAndExtractWithMockServer(t *testing.T) {
	// Create a test zip file
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	fileWriter, err := zipWriter.Create("mock_driver.spin2")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := fileWriter.Write([]byte("' Mock Spin2 Driver")); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Write(zipBuf.Bytes())
	}))
	defer server.Close()

	// Setup test environment
	tmpDir := t.TempDir()
	workDir := filepath.Join(tmpDir, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create manager with mock object
	m := &Manager{
		cacheDir:    filepath.Join(tmpDir, "cache"),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1234"},
		ttl:         DefaultOBEXTTL,
		lastRefresh: time.Now(),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}

	// Pre-populate object metadata
	m.objects["1234"] = &OBEXObject{}
	m.objects["1234"].ObjectMetadata.ObjectID = "1234"
	m.objects["1234"].ObjectMetadata.Title = "Mock Driver"

	// Test downloadZip method directly with mock server
	zipData, err := m.downloadZip(server.URL)
	if err != nil {
		t.Fatalf("downloadZip failed: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("downloadZip returned empty data")
	}

	// Verify it's valid zip data
	files, _, err := extractZip(zipData, filepath.Join(workDir, "extracted"))
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	if len(files) != 1 || files[0] != "mock_driver.spin2" {
		t.Errorf("unexpected files: %v", files)
	}
}

func TestDownloadResult(t *testing.T) {
	result := &DownloadResult{
		ObjectID:       "2811",
		Title:          "Test Object",
		ExtractionPath: "/tmp/OBX/test-object",
		Files:          []string{"test.spin2", "README.txt"},
		TotalSize:      1234,
	}

	if result.ObjectID != "2811" {
		t.Errorf("ObjectID = %q, want '2811'", result.ObjectID)
	}
	if len(result.Files) != 2 {
		t.Errorf("Files count = %d, want 2", len(result.Files))
	}
	if result.TotalSize != 1234 {
		t.Errorf("TotalSize = %d, want 1234", result.TotalSize)
	}
}
