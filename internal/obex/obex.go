// Package obex manages OBEX (Parallax Object Exchange) metadata.
//
// OBEX object YAMLs are KB files listed in the main index. They are read
// through the same path as every other KB file — the index for the list and
// each file's sha256, the cache for the hash-checked, filtered, rule-stamped
// body — and parsed here.
package obex

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/cache"
	"github.com/ironsheep/p2kb-mcp/internal/index"
	"github.com/ironsheep/p2kb-mcp/internal/logging"
	"github.com/ironsheep/p2kb-mcp/internal/paths"
	"gopkg.in/yaml.v3"
)

const (
	// OBEXPath is the path to OBEX objects in the repository.
	OBEXPath = "deliverables/ai/P2/community/obex/objects"

	// templateFile is the object template, listed in the index but not an object.
	templateFile = "_template.yaml"

	// OBEXDownloadBase is the base URL for OBEX downloads.
	OBEXDownloadBase = "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB"

	// ErrorRefreshCooldown is the minimum time between refresh-on-error attempts.
	// This prevents excessive refresh attempts when objects are genuinely not found.
	ErrorRefreshCooldown = 5 * time.Minute
)

// ObjectMetadata represents the object_metadata section of an OBEX YAML file.
type ObjectMetadata struct {
	ObjectID       string `yaml:"object_id"`
	Title          string `yaml:"title"`
	Author         string `yaml:"author"`
	AuthorUsername string `yaml:"author_username"`

	URLs struct {
		OBEXPage        string `yaml:"obex_page"`
		DownloadDirect  string `yaml:"download_direct"`
		ForumDiscussion string `yaml:"forum_discussion"`
		GithubRepo      string `yaml:"github_repo"`
		Documentation   string `yaml:"documentation"`
	} `yaml:"urls"`

	TechnicalDetails struct {
		Languages       []string `yaml:"languages"`
		Microcontroller []string `yaml:"microcontroller"`
		Version         string   `yaml:"version"`
		FileFormat      string   `yaml:"file_format"`
		FileSize        string   `yaml:"file_size"`
	} `yaml:"technical_details"`

	Functionality struct {
		Category         string   `yaml:"category"`
		Subcategory      string   `yaml:"subcategory"`
		DescriptionShort string   `yaml:"description_short"`
		DescriptionFull  string   `yaml:"description_full"`
		Tags             []string `yaml:"tags"`
		HardwareSupport  []string `yaml:"hardware_support"`
		Peripherals      []string `yaml:"peripherals"`
	} `yaml:"functionality"`

	Metadata struct {
		DiscoveryDate    string `yaml:"discovery_date"`
		LastVerified     string `yaml:"last_verified"`
		ExtractionStatus string `yaml:"extraction_status"`
		QualityScore     int    `yaml:"quality_score"`
		CreatedDate      string `yaml:"created_date"`
	} `yaml:"metadata"`
}

// OBEXObject represents a complete OBEX object from a YAML file.
type OBEXObject struct {
	ObjectMetadata ObjectMetadata `yaml:"object_metadata"`
}

// SearchResult represents a search match.
type SearchResult struct {
	ObjectID         string `json:"object_id"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Category         string `json:"category"`
	DescriptionShort string `json:"description_short"`
	MatchType        string `json:"match_type"`
}

// AuthorStats tracks objects per author.
type AuthorStats struct {
	Name        string `json:"name"`
	ObjectCount int    `json:"object_count"`
}

// DownloadResult contains the result of downloading and extracting an OBEX object.
type DownloadResult struct {
	ObjectID       string   `json:"object_id"`
	Title          string   `json:"title"`
	ExtractionPath string   `json:"extraction_path"`
	Files          []string `json:"files"`
	TotalSize      int64    `json:"total_size"`
}

// Manager handles OBEX operations.
//
// It holds no lock while it calls the index or cache manager.
type Manager struct {
	index      *index.Manager
	cache      *cache.Manager
	httpClient *http.Client // downloads from obex.parallax.com

	mu               sync.RWMutex
	parsed           map[string]parsedObject // by object ID
	lastErrorRefresh time.Time               // Tracks last refresh-on-error attempt to prevent refresh storms
}

// parsedObject is a parsed object and the version of its file it was parsed from.
type parsedObject struct {
	version string
	obj     *OBEXObject
}

// objectFile locates an object's YAML in the main index.
type objectFile struct {
	key  string
	file index.FileEntry
}

// NewManager creates a new OBEX manager reading through idx and c. It removes
// the OBEX disk cache that servers before 1.5.0 kept, once.
func NewManager(idx *index.Manager, c *cache.Manager) *Manager {
	legacy := filepath.Join(paths.GetCacheDirOrDefault(), "obex")
	if err := os.RemoveAll(legacy); err != nil {
		logging.Warnf("p2kb-mcp: warning: failed to remove retired OBEX cache %s: %v", legacy, err)
	}
	return &Manager{
		index:      idx,
		cache:      c,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		parsed:     make(map[string]parsedObject),
	}
}

// objectFiles lists the objects in the main index by object ID: every file
// under OBEXPath except the template, its ID the file name without .yaml.
func (m *Manager) objectFiles() (map[string]objectFile, error) {
	files, err := m.index.FilesUnder(OBEXPath)
	if err != nil {
		return nil, fmt.Errorf("OBEX index unavailable: %w", err)
	}
	objects := make(map[string]objectFile, len(files))
	for key, f := range files {
		name := path.Base(f.Path)
		if name == templateFile || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		objects[strings.TrimSuffix(name, ".yaml")] = objectFile{key: key, file: f}
	}
	return objects, nil
}

// GetObjectIDs returns all OBEX object IDs, sorted.
func (m *Manager) GetObjectIDs() []string {
	objects, err := m.objectFiles()
	if err != nil {
		return nil
	}
	return sortedIDs(objects)
}

func sortedIDs(objects map[string]objectFile) []string {
	ids := make([]string, 0, len(objects))
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetObject retrieves an OBEX object by ID.
// If object is not found and cooldown has passed, attempts one refresh before giving up.
func (m *Manager) GetObject(objectID string) (*OBEXObject, error) {
	// Normalize ID (remove OB prefix if present)
	objectID = normalizeObjectID(objectID)

	objects, err := m.objectFiles()
	if err != nil {
		return nil, err
	}
	of, ok := lookupObject(objects, objectID)
	if !ok {
		// Object not found - try refresh-on-error if cooldown has passed
		if m.tryErrorRefresh() {
			// Retry lookup after refresh
			if objects, err = m.objectFiles(); err == nil {
				of, ok = lookupObject(objects, objectID)
			}
		}
		if !ok {
			return nil, fmt.Errorf("OBEX object not found: %s", objectID)
		}
	}
	return m.getObject(of)
}

// lookupObject finds objectID, ignoring case.
func lookupObject(objects map[string]objectFile, objectID string) (objectFile, bool) {
	if of, ok := objects[objectID]; ok {
		return of, true
	}
	for id, of := range objects {
		if strings.EqualFold(id, objectID) {
			return of, true
		}
	}
	return objectFile{}, false
}

// getObject returns the parsed object for of, reading its body through the
// cache (hash-checked, filtered under the rule in effect, stamped) and
// parsing it again only when the file or the rule changed.
func (m *Manager) getObject(of objectFile) (*OBEXObject, error) {
	id := strings.TrimSuffix(path.Base(of.file.Path), ".yaml")
	version := fmt.Sprintf("%s|%d|%s", of.file.SHA256, of.file.Mtime, cache.Stamp(m.cache.Rule()))

	m.mu.RLock()
	p, ok := m.parsed[id]
	m.mu.RUnlock()
	if ok && p.version == version {
		return p.obj, nil
	}

	content, err := m.cache.GetOrFetch(of.key, of.file.Path, of.file.SHA256, of.file.Mtime)
	if err != nil {
		return nil, err
	}
	obj := &OBEXObject{}
	if err := yaml.Unmarshal([]byte(content), obj); err != nil {
		return nil, fmt.Errorf("failed to parse OBEX object %s: %w", id, err)
	}

	m.mu.Lock()
	m.parsed[id] = parsedObject{version: version, obj: obj}
	m.mu.Unlock()
	return obj, nil
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
	if err := m.index.Refresh(); err != nil {
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

// Search searches OBEX objects by term.
func (m *Manager) Search(term string, category string, language string, limit int) ([]SearchResult, error) {
	objects, err := m.objectFiles()
	if err != nil {
		return nil, err
	}

	if term == "" {
		return nil, fmt.Errorf("search term required")
	}

	if limit <= 0 {
		limit = 20
	}

	// Expand search terms
	searchTerms := expandSearchTerms(strings.ToLower(term))

	var results []SearchResult
	for _, objID := range sortedIDs(objects) {
		obj, err := m.getObject(objects[objID])
		if err != nil {
			continue
		}

		// Apply filters
		if category != "" && !strings.EqualFold(obj.ObjectMetadata.Functionality.Category, category) {
			continue
		}

		if language != "" {
			hasLanguage := false
			for _, lang := range obj.ObjectMetadata.TechnicalDetails.Languages {
				if strings.EqualFold(lang, language) {
					hasLanguage = true
					break
				}
			}
			if !hasLanguage {
				continue
			}
		}

		// Check for matches
		matchType := m.matchObject(obj, searchTerms)
		if matchType != "" {
			results = append(results, SearchResult{
				ObjectID:         obj.ObjectMetadata.ObjectID,
				Title:            obj.ObjectMetadata.Title,
				Author:           obj.ObjectMetadata.Author,
				Category:         obj.ObjectMetadata.Functionality.Category,
				DescriptionShort: obj.ObjectMetadata.Functionality.DescriptionShort,
				MatchType:        matchType,
			})

			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// GetCategories returns OBEX categories with counts.
func (m *Manager) GetCategories() (map[string]int, error) {
	objects, err := m.objectFiles()
	if err != nil {
		return nil, err
	}

	categories := make(map[string]int)
	for _, objID := range sortedIDs(objects) {
		obj, err := m.getObject(objects[objID])
		if err != nil {
			continue
		}

		cat := obj.ObjectMetadata.Functionality.Category
		if cat == "" {
			cat = "uncategorized"
		}
		categories[cat]++
	}

	return categories, nil
}

// BrowseCategory returns objects in a category.
func (m *Manager) BrowseCategory(category string) ([]SearchResult, error) {
	objects, err := m.objectFiles()
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, objID := range sortedIDs(objects) {
		obj, err := m.getObject(objects[objID])
		if err != nil {
			continue
		}

		if category != "" && !strings.EqualFold(obj.ObjectMetadata.Functionality.Category, category) {
			continue
		}

		results = append(results, SearchResult{
			ObjectID:         obj.ObjectMetadata.ObjectID,
			Title:            obj.ObjectMetadata.Title,
			Author:           obj.ObjectMetadata.Author,
			Category:         obj.ObjectMetadata.Functionality.Category,
			DescriptionShort: obj.ObjectMetadata.Functionality.DescriptionShort,
		})
	}

	return results, nil
}

// GetAuthors returns authors sorted by object count.
func (m *Manager) GetAuthors() ([]AuthorStats, error) {
	objects, err := m.objectFiles()
	if err != nil {
		return nil, err
	}

	authorCounts := make(map[string]int)
	for _, objID := range sortedIDs(objects) {
		obj, err := m.getObject(objects[objID])
		if err != nil {
			continue
		}

		author := obj.ObjectMetadata.Author
		if author == "" {
			author = "Unknown"
		}
		authorCounts[author]++
	}

	// Convert to slice and sort
	authors := make([]AuthorStats, 0, len(authorCounts))
	for name, count := range authorCounts {
		authors = append(authors, AuthorStats{
			Name:        name,
			ObjectCount: count,
		})
	}

	// Most objects first; ties by name, so the order never depends on map order.
	sort.Slice(authors, func(i, j int) bool {
		if authors[i].ObjectCount != authors[j].ObjectCount {
			return authors[i].ObjectCount > authors[j].ObjectCount
		}
		return authors[i].Name < authors[j].Name
	})

	return authors, nil
}

// GetTotalObjects returns the total number of OBEX objects.
func (m *Manager) GetTotalObjects() int {
	objects, err := m.objectFiles()
	if err != nil {
		return 0
	}
	return len(objects)
}

// GetDownloadURL returns the download URL for an object.
func (m *Manager) GetDownloadURL(objectID string) string {
	objectID = normalizeObjectID(objectID)
	return OBEXDownloadBase + objectID
}

// DownloadAndExtract downloads an OBEX object zip and extracts it to the target directory.
// If targetDir is empty, it defaults to "./OBX/{object-slug}/".
// Returns information about the extracted files.
func (m *Manager) DownloadAndExtract(objectID, targetDir string) (*DownloadResult, error) {
	objectID = normalizeObjectID(objectID)

	// Get object metadata to determine title/slug
	obj, err := m.GetObject(objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Generate slug from title
	slug := generateSlug(obj.ObjectMetadata.Title)

	// Determine target directory: OBEX/{objID}-{slug}/
	if targetDir == "" {
		if slug != "" {
			targetDir = filepath.Join("OBEX", objectID+"-"+slug)
		} else {
			targetDir = filepath.Join("OBEX", objectID)
		}
	}

	// Validate target path (security check)
	if err := validateTargetPath(targetDir); err != nil {
		return nil, err
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download the zip file
	downloadURL := m.GetDownloadURL(objectID)
	zipData, err := m.downloadZip(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download zip: %w", err)
	}

	// Extract zip to target directory
	files, totalSize, err := extractZip(zipData, targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract zip: %w", err)
	}

	// Get absolute path for result
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		absPath = targetDir
	}

	return &DownloadResult{
		ObjectID:       objectID,
		Title:          obj.ObjectMetadata.Title,
		ExtractionPath: absPath,
		Files:          files,
		TotalSize:      totalSize,
	}, nil
}

// downloadZip downloads a zip file from the given URL and returns its contents.
func (m *Manager) downloadZip(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "p2kb-mcp")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// extractZip extracts a zip archive to the target directory.
// Returns the list of extracted files and total size.
func extractZip(zipData []byte, targetDir string) ([]string, int64, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, 0, fmt.Errorf("invalid zip file: %w", err)
	}

	var files []string
	var totalSize int64

	for _, file := range reader.File {
		// Security: prevent zip slip attack
		destPath := filepath.Join(targetDir, file.Name)
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(targetDir)) {
			return nil, 0, fmt.Errorf("invalid file path in zip: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return nil, 0, fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, 0, fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Extract file
		if err := extractFile(file, destPath); err != nil {
			return nil, 0, err
		}

		files = append(files, file.Name)
		totalSize += int64(file.UncompressedSize64)
	}

	return files, totalSize, nil
}

// extractFile extracts a single file from a zip archive.
func extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip entry: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Limit copy size to prevent decompression bombs (100MB max per file)
	_, err = io.Copy(outFile, io.LimitReader(rc, 100*1024*1024))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// validateTargetPath ensures the target path is safe (no directory traversal).
func validateTargetPath(targetDir string) error {
	// Check for ".." in original path BEFORE cleaning (Clean will normalize it away)
	if strings.Contains(targetDir, "..") {
		return fmt.Errorf("target directory cannot contain '..'")
	}

	// Clean the path
	cleaned := filepath.Clean(targetDir)

	// Disallow absolute paths that go outside current directory
	if filepath.IsAbs(cleaned) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		if !strings.HasPrefix(cleaned, cwd) {
			return fmt.Errorf("target directory must be within working directory")
		}
	}

	// Also verify the cleaned path doesn't start with ".." (e.g., "../foo")
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("target directory cannot escape working directory")
	}

	return nil
}

// generateSlug creates a filesystem-safe slug from a title.
func generateSlug(title string) string {
	slug := strings.ToLower(title)

	var result strings.Builder
	lastWasHyphen := false

	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			lastWasHyphen = false
		} else if !lastWasHyphen {
			result.WriteRune('-')
			lastWasHyphen = true
		}
	}

	return strings.Trim(result.String(), "-")
}

// Refresh drops every parsed object so each is parsed again on next use. The
// object list and bodies refresh with the main index and the content cache.
func (m *Manager) Refresh() error {
	m.ClearCache()
	return nil
}

// ClearCache drops every parsed object, returning how many there were.
func (m *Manager) ClearCache() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := len(m.parsed)
	m.parsed = make(map[string]parsedObject)
	return count
}

// GetCacheStats returns how many objects are parsed in memory and how many
// object bodies the content cache holds.
func (m *Manager) GetCacheStats() (parsedCount, cachedCount int) {
	m.mu.RLock()
	parsedCount = len(m.parsed)
	m.mu.RUnlock()

	objects, err := m.objectFiles()
	if err != nil {
		return parsedCount, 0
	}
	keys := make(map[string]bool, len(objects))
	for _, of := range objects {
		keys[of.key] = true
	}
	for _, k := range m.cache.GetCachedKeys() {
		if keys[k] {
			cachedCount++
		}
	}
	return parsedCount, cachedCount
}

// Private methods

func (m *Manager) matchObject(obj *OBEXObject, searchTerms []string) string {
	titleLower := strings.ToLower(obj.ObjectMetadata.Title)
	descShortLower := strings.ToLower(obj.ObjectMetadata.Functionality.DescriptionShort)
	descFullLower := strings.ToLower(obj.ObjectMetadata.Functionality.DescriptionFull)

	// Check tags
	tagsLower := make([]string, len(obj.ObjectMetadata.Functionality.Tags))
	for i, tag := range obj.ObjectMetadata.Functionality.Tags {
		tagsLower[i] = strings.ToLower(tag)
	}

	for _, term := range searchTerms {
		// Title match (highest priority)
		if strings.Contains(titleLower, term) {
			return "title"
		}
	}

	for _, term := range searchTerms {
		// Tag match
		for _, tag := range tagsLower {
			if strings.Contains(tag, term) || strings.Contains(term, tag) {
				return "tag"
			}
		}
	}

	for _, term := range searchTerms {
		// Description match
		if strings.Contains(descShortLower, term) || strings.Contains(descFullLower, term) {
			return "description"
		}
	}

	return ""
}

func normalizeObjectID(objectID string) string {
	// Remove "OB" prefix if present (case-insensitive)
	objectID = strings.TrimSpace(objectID)
	upper := strings.ToUpper(objectID)
	if strings.HasPrefix(upper, "OB") {
		objectID = objectID[2:]
	}
	return strings.TrimSpace(objectID)
}

// expandSearchTerms expands a search term to include related terms.
func expandSearchTerms(term string) []string {
	expansions := map[string][]string{
		"i2c":     {"i2c", "iic", "twi", "two-wire"},
		"spi":     {"spi", "serial peripheral", "shift"},
		"uart":    {"uart", "serial", "rs232", "rs485"},
		"led":     {"led", "pixel", "ws2812", "rgb", "neopixel", "strip"},
		"motor":   {"motor", "servo", "stepper", "pwm", "drive"},
		"sensor":  {"sensor", "detector", "measure", "monitor"},
		"display": {"display", "lcd", "oled", "screen", "graphics"},
		"audio":   {"audio", "sound", "speaker", "wav", "music"},
		"video":   {"video", "vga", "hdmi", "graphics"},
		"usb":     {"usb", "hid", "cdc"},
		"sd":      {"sd", "sdcard", "fat", "filesystem"},
		"wifi":    {"wifi", "wireless", "esp", "network"},
	}

	terms := []string{term}

	// Check if term matches any expansion key
	for key, expanded := range expansions {
		if strings.Contains(term, key) {
			terms = append(terms, expanded...)
		}
	}

	// Also check if any expansion key is in the term
	for key, expanded := range expansions {
		for _, exp := range expanded {
			if strings.Contains(term, exp) && exp != term {
				terms = append(terms, key)
				terms = append(terms, expanded...)
				break
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(terms))
	for _, t := range terms {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}

	return unique
}
