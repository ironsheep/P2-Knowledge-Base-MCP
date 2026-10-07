package obex

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/cache"
	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/index"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
)

// objectYAML is an OBEX object file, in the KB's shape.
const objectYAML = `# OBEX object
object_metadata:
  object_id: "2811"
  title: "Park transformation"
  author: "ManAtWork"
  technical_details:
    languages:
      - SPIN2
      - PASM2
  functionality:
    category: "motors"
    description_short: "The Park or d/q-transformation"
    tags:
      - servo
      - motor
`

// secondYAML is a second object, in another category.
const secondYAML = `object_metadata:
  object_id: "4047"
  title: "I2C driver"
  author: "Jon McPhalen (jonnymac)"
  functionality:
    category: "drivers"
    description_short: "Two-wire bus driver"
    tags:
      - i2c
`

// newTestManager builds the OBEX manager the way server.New does: index and
// cache managers over a temp cache dir and a kbtest.Remote.
func newTestManager(t *testing.T) (*Manager, *kbtest.Remote) {
	t.Helper()
	t.Setenv("P2KB_CACHE_DIR", t.TempDir())
	r := kbtest.NewRemote(t)
	idx := index.NewManager(r.Client())
	c := cache.NewManager(r.Client())
	idx.OnRuleChange(c.SetRule)
	idx.ResolveStartupRule()
	return NewManager(idx, c), r
}

// objectPath is where the KB keeps object id.
func objectPath(id string) string { return OBEXPath + "/" + id + ".yaml" }

// addObject lists object id in the index and serves body for it.
func addObject(r *kbtest.Remote, id, body string) {
	r.AddFile("p2kbCommunity"+id, objectPath(id), 1700000000, body)
}

// addTypicalObjects serves two objects, the template and a non-OBEX file.
func addTypicalObjects(r *kbtest.Remote) {
	addObject(r, "2811", objectYAML)
	addObject(r, "4047", secondYAML)
	r.AddFile("p2kbCommunityTemplate", OBEXPath+"/_template.yaml", 1700000000, "object_metadata:\n  title: TEMPLATE\n")
	r.AddFile("p2kbPasm2Mov", "deliverables/ai/P2/pasm2/mov.yaml", 1700000000, "mnemonic: MOV\n")
}

func TestNewManagerRemovesRetiredOBEXCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("P2KB_CACHE_DIR", dir)
	legacy := filepath.Join(dir, "obex", "objects")
	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "2811.yaml"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	r := kbtest.NewRemote(t)
	NewManager(index.NewManager(r.Client()), cache.NewManager(r.Client()))
	if _, err := os.Stat(filepath.Join(dir, "obex")); !os.IsNotExist(err) {
		t.Errorf("retired OBEX cache still present (stat err %v)", err)
	}
}

func TestObjectListComesFromMainIndex(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)

	if got, want := m.GetObjectIDs(), []string{"2811", "4047"}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetObjectIDs = %v, want %v (template and non-OBEX files excluded)", got, want)
	}
	if got := m.GetTotalObjects(); got != 2 {
		t.Errorf("GetTotalObjects = %d, want 2", got)
	}
	if got := r.Hits(fetch.IndexPath); got != 1 {
		t.Errorf("index fetches = %d, want 1", got)
	}
}

func TestGetObjectReadsThroughKBContentPath(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)

	for _, id := range []string{"2811", "OB2811", "ob2811", " OB2811 "} {
		obj, err := m.GetObject(id)
		if err != nil {
			t.Fatalf("GetObject(%q): %v", id, err)
		}
		meta := obj.ObjectMetadata
		if meta.ObjectID != "2811" || meta.Title != "Park transformation" || meta.Author != "ManAtWork" ||
			meta.Functionality.Category != "motors" ||
			!reflect.DeepEqual(meta.TechnicalDetails.Languages, []string{"SPIN2", "PASM2"}) {
			t.Errorf("GetObject(%q) = %+v", id, meta)
		}
	}
	if got := r.Hits(objectPath("2811")); got != 1 {
		t.Errorf("object fetches = %d, want 1 (cached and parsed once)", got)
	}
}

func TestGetObjectTemplateIsNotAnObject(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)
	m.lastErrorRefresh = time.Now() // keep the not-found path from refreshing
	if _, err := m.GetObject("_template"); err == nil {
		t.Error("the template resolved as an object")
	}
}

func TestGetObjectReparsesOnlyWhenFileChanges(t *testing.T) {
	m, r := newTestManager(t)
	addObject(r, "2811", objectYAML)

	first, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	again, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Error("unchanged file was parsed again")
	}

	changed := `object_metadata:
  object_id: "2811"
  title: "Park transformation v2"
`
	r.AddFile("p2kbCommunity2811", objectPath("2811"), 1700000100, changed)
	if err := m.index.Refresh(); err != nil {
		t.Fatal(err)
	}
	updated, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	if updated == first || updated.ObjectMetadata.Title != "Park transformation v2" {
		t.Errorf("changed sha256 was not re-parsed: title %q", updated.ObjectMetadata.Title)
	}
}

func TestObjectBodyIsFilteredBeforeParse(t *testing.T) {
	m, r := newTestManager(t)
	addObject(r, "2811", objectYAML)
	idx := r.Index()
	// A rule that removes a field the object parser reads proves the body is
	// filtered before it is parsed.
	idx.DeliveryFilter = json.RawMessage(`{"format":1,"remove_fields":["author"],"remove_column0_comments":true}`)
	r.SetIndex(idx)

	obj, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	if obj.ObjectMetadata.Author != "" {
		t.Errorf("author = %q, want it removed by the rule before parsing", obj.ObjectMetadata.Author)
	}
	if obj.ObjectMetadata.Title != "Park transformation" {
		t.Errorf("title = %q, want it kept", obj.ObjectMetadata.Title)
	}
}

func TestRuleChangeReparsesObject(t *testing.T) {
	m, r := newTestManager(t)
	addObject(r, "2811", objectYAML)
	before, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	if before.ObjectMetadata.Author == "" {
		t.Fatal("author missing under the built-in rule")
	}

	idx := r.Index()
	idx.DeliveryFilter = json.RawMessage(`{"format":1,"remove_fields":["author"],"remove_column0_comments":true}`)
	r.SetIndex(idx)
	if err := m.index.Refresh(); err != nil {
		t.Fatal(err)
	}

	after, err := m.GetObject("2811")
	if err != nil {
		t.Fatal(err)
	}
	if after.ObjectMetadata.Author != "" {
		t.Errorf("author = %q after the rule changed, want removed", after.ObjectMetadata.Author)
	}
}

func TestGetObjectHashMismatchIsVerificationError(t *testing.T) {
	m, r := newTestManager(t)
	addObject(r, "2811", objectYAML)
	r.Put(objectPath("2811"), "object_metadata:\n  title: tampered\n")

	_, err := m.GetObject("2811")
	var verr *cache.VerificationError
	if !errors.As(err, &verr) {
		t.Fatalf("GetObject err = %v, want *cache.VerificationError", err)
	}
}

func TestSearchCategoriesAuthorsAcrossAllObjects(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)

	results, err := m.Search("servo", "", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ObjectID != "2811" || results[0].MatchType != "tag" {
		t.Errorf("Search(servo) = %+v", results)
	}
	if results, _ := m.Search("i2c", "", "", 10); len(results) != 1 || results[0].ObjectID != "4047" {
		t.Errorf("Search(i2c) = %+v", results)
	}

	cats, err := m.GetCategories()
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]int{"motors": 1, "drivers": 1}; !reflect.DeepEqual(cats, want) {
		t.Errorf("GetCategories = %v, want %v", cats, want)
	}

	browse, err := m.BrowseCategory("drivers")
	if err != nil || len(browse) != 1 || browse[0].ObjectID != "4047" {
		t.Errorf("BrowseCategory(drivers) = %+v, %v", browse, err)
	}

	authors, err := m.GetAuthors()
	if err != nil || len(authors) != 2 {
		t.Errorf("GetAuthors = %+v, %v", authors, err)
	}
}

func TestGetAuthorsOrdersTiesByName(t *testing.T) {
	m, r := newTestManager(t)
	for i, author := range []string{"zed", "amy", "kim", "amy"} {
		id := string(rune('1' + i))
		addObject(r, id, "object_metadata:\n  object_id: \""+id+"\"\n  author: \""+author+"\"\n")
	}
	want := []AuthorStats{{"amy", 2}, {"kim", 1}, {"zed", 1}}
	for i := 0; i < 20; i++ { // map order varies run to run; the result must not
		got, err := m.GetAuthors()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("GetAuthors = %+v, want %+v", got, want)
		}
	}
}

func TestClearCacheDropsParsedObjects(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)
	if _, err := m.GetObject("2811"); err != nil {
		t.Fatal(err)
	}
	parsed, cached := m.GetCacheStats()
	if parsed != 1 || cached != 1 {
		t.Errorf("GetCacheStats = %d parsed, %d cached; want 1, 1", parsed, cached)
	}
	if n := m.ClearCache(); n != 1 {
		t.Errorf("ClearCache = %d, want 1", n)
	}
	if parsed, _ := m.GetCacheStats(); parsed != 0 {
		t.Errorf("parsed after ClearCache = %d, want 0", parsed)
	}
}

func TestNotFoundInCooldownDoesNotRefresh(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)
	m.GetObjectIDs() // load the index
	m.lastErrorRefresh = time.Now()
	originalTime := m.lastErrorRefresh

	if _, err := m.GetObject("9999"); err == nil {
		t.Error("GetObject(9999) found a missing object")
	}
	if got := r.Hits(fetch.IndexPath); got != 1 {
		t.Errorf("index fetches = %d, want 1 (no refresh in cooldown)", got)
	}
	if m.lastErrorRefresh != originalTime {
		t.Error("GetObject should NOT trigger error refresh when in cooldown")
	}
}

func TestNotFoundAfterCooldownRefreshesAndFinds(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)
	m.GetObjectIDs()
	m.lastErrorRefresh = time.Now().Add(-10 * time.Minute)
	originalTime := m.lastErrorRefresh

	addObject(r, "5274", "object_metadata:\n  object_id: \"5274\"\n  title: New object\n")
	obj, err := m.GetObject("5274")
	if err != nil {
		t.Fatalf("GetObject after refresh: %v", err)
	}
	if obj.ObjectMetadata.Title != "New object" {
		t.Errorf("title = %q", obj.ObjectMetadata.Title)
	}
	if got := r.Hits(fetch.IndexPath); got != 2 {
		t.Errorf("index fetches = %d, want 2 (one refresh-on-error)", got)
	}
	if !m.lastErrorRefresh.After(originalTime) {
		t.Error("tryErrorRefresh should update lastErrorRefresh timestamp")
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
		{"Ob2811", "2811"}, // Mixed case - first letter uppercase
		{"oB2811", "2811"}, // Mixed case - second letter uppercase
		{" 2811 ", "2811"},
		{"OB 2811", "2811"},  // Space after OB prefix (trimmed from result)
		{" OB2811 ", "2811"}, // Spaces around the whole thing
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
	m := &Manager{}

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

func TestMatchObject(t *testing.T) {
	m := &Manager{}

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

func TestDownloadZipWithMockServer(t *testing.T) {
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
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer server.Close()

	m := &Manager{httpClient: &http.Client{Timeout: 30 * time.Second}}

	// Test downloadZip method directly with mock server
	zipData, err := m.downloadZip(server.URL)
	if err != nil {
		t.Fatalf("downloadZip failed: %v", err)
	}

	if len(zipData) == 0 {
		t.Error("downloadZip returned empty data")
	}

	// Verify it's valid zip data
	files, _, err := extractZip(zipData, filepath.Join(t.TempDir(), "extracted"))
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
