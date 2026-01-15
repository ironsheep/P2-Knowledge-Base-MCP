// Package obex manages OBEX (Parallax Object Exchange) metadata.
package obex

import (
	"archive/zip"
	"bytes"
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
	"gopkg.in/yaml.v3"
)

const (
	// GitHubAPIBase is the base URL for GitHub API requests.
	GitHubAPIBase = "https://api.github.com/repos/ironsheep/P2-Knowledge-Base/contents"

	// GitHubRawBase is the base URL for raw content.
	GitHubRawBase = "https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main"

	// OBEXPath is the path to OBEX objects in the repository.
	OBEXPath = "deliverables/ai/P2/community/obex/objects"

	// OBEXDownloadBase is the base URL for OBEX downloads.
	OBEXDownloadBase = "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB"

	// DefaultOBEXTTL is the default time-to-live for the OBEX index.
	DefaultOBEXTTL = 24 * time.Hour
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
type Manager struct {
	mu          sync.RWMutex
	fetchMu     sync.Mutex                 // Prevents concurrent index fetches, separate from data lock
	cacheDir    string
	objectIDs   []string                   // List of all object IDs
	objects     map[string]*OBEXObject     // Cached objects by ID
	lastRefresh time.Time
	ttl         time.Duration
	httpClient  *http.Client
}

// NewManager creates a new OBEX manager.
func NewManager() *Manager {
	return &Manager{
		cacheDir:   paths.GetCacheDirOrDefault(),
		objects:    make(map[string]*OBEXObject),
		ttl:        DefaultOBEXTTL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// EnsureIndex ensures the OBEX object list is loaded.
// This method is safe for concurrent access and does NOT hold locks during network I/O.
func (m *Manager) EnsureIndex() error {
	// Fast path: check with read lock if we have a fresh index
	m.mu.RLock()
	if len(m.objectIDs) > 0 && time.Since(m.lastRefresh) < m.ttl {
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
	if len(m.objectIDs) > 0 && time.Since(m.lastRefresh) < m.ttl {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// Try to load from cache (quick file I/O, safe to hold write lock briefly)
	m.mu.Lock()
	if m.loadIndexFromCache() {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Fetch from GitHub API WITHOUT holding the data lock
	// This is the critical fix: network I/O happens outside the lock
	objectIDs, err := m.fetchIndexData()
	if err != nil {
		return fmt.Errorf("OBEX index fetch failed: %w", err)
	}

	// Update the index under write lock (quick operation)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Save to cache
	m.saveIndexToCache(objectIDs)

	m.objectIDs = objectIDs
	m.lastRefresh = time.Now()
	return nil
}

// GetObjectIDs returns all OBEX object IDs.
func (m *Manager) GetObjectIDs() []string {
	if err := m.EnsureIndex(); err != nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.objectIDs))
	copy(result, m.objectIDs)
	return result
}

// GetObject retrieves an OBEX object by ID.
func (m *Manager) GetObject(objectID string) (*OBEXObject, error) {
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	// Normalize ID (remove OB prefix if present)
	objectID = normalizeObjectID(objectID)

	m.mu.RLock()
	if obj, ok := m.objects[objectID]; ok {
		m.mu.RUnlock()
		return obj, nil
	}
	m.mu.RUnlock()

	// Check if ID exists in index
	if !m.objectExists(objectID) {
		return nil, fmt.Errorf("OBEX object not found: %s", objectID)
	}

	// Fetch from remote or cache
	return m.fetchObject(objectID)
}

// Search searches OBEX objects by term.
func (m *Manager) Search(term string, category string, language string, limit int) ([]SearchResult, error) {
	if err := m.EnsureIndex(); err != nil {
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
	objectIDs := m.GetObjectIDs()

	for _, objID := range objectIDs {
		obj, err := m.GetObject(objID)
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
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	categories := make(map[string]int)
	objectIDs := m.GetObjectIDs()

	for _, objID := range objectIDs {
		obj, err := m.GetObject(objID)
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
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	var results []SearchResult
	objectIDs := m.GetObjectIDs()

	for _, objID := range objectIDs {
		obj, err := m.GetObject(objID)
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
	if err := m.EnsureIndex(); err != nil {
		return nil, err
	}

	authorCounts := make(map[string]int)
	objectIDs := m.GetObjectIDs()

	for _, objID := range objectIDs {
		obj, err := m.GetObject(objID)
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

	sort.Slice(authors, func(i, j int) bool {
		return authors[i].ObjectCount > authors[j].ObjectCount
	})

	return authors, nil
}

// GetTotalObjects returns the total number of OBEX objects.
func (m *Manager) GetTotalObjects() int {
	if err := m.EnsureIndex(); err != nil {
		return 0
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objectIDs)
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

// Refresh forces a refresh of the OBEX index and clears stale objects.
// This method fetches fresh data from remote without holding locks during network I/O.
func (m *Manager) Refresh() error {
	// Use fetchMu to prevent concurrent fetches
	m.fetchMu.Lock()
	defer m.fetchMu.Unlock()

	// Fetch from GitHub API WITHOUT holding the data lock
	objectIDs, err := m.fetchIndexData()
	if err != nil {
		return fmt.Errorf("OBEX index refresh failed: %w", err)
	}

	// Update the index under write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear memory cache
	m.objects = make(map[string]*OBEXObject)

	// Save to cache
	m.saveIndexToCache(objectIDs)

	m.objectIDs = objectIDs
	m.lastRefresh = time.Now()
	return nil
}

// ClearCache clears all cached OBEX data.
// This method releases the lock before disk I/O for better concurrency.
func (m *Manager) ClearCache() int {
	// Clear memory cache under lock
	m.mu.Lock()
	count := len(m.objects)
	m.objects = make(map[string]*OBEXObject)
	m.objectIDs = nil
	m.lastRefresh = time.Time{}
	cacheDir := filepath.Join(m.cacheDir, "obex")
	m.mu.Unlock() // Release lock BEFORE disk I/O

	// Clear disk cache - no lock needed for this operation
	_ = os.RemoveAll(cacheDir)

	return count
}

// GetCacheStats returns OBEX cache statistics.
func (m *Manager) GetCacheStats() (memoryCount, diskCount int, staleCount int) {
	m.mu.RLock()
	memoryCount = len(m.objects)
	m.mu.RUnlock()

	objectsDir := filepath.Join(m.cacheDir, "obex", "objects")
	entries, err := os.ReadDir(objectsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				diskCount++

				// Check if stale
				info, err := entry.Info()
				if err == nil && time.Since(info.ModTime()) > m.ttl {
					staleCount++
				}
			}
		}
	}

	return
}

// Private methods

func (m *Manager) objectExists(objectID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, id := range m.objectIDs {
		if id == objectID {
			return true
		}
	}
	return false
}

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

func (m *Manager) loadIndexFromCache() bool {
	indexPath := filepath.Join(m.cacheDir, "obex", "index.json")

	info, err := os.Stat(indexPath)
	if err != nil {
		return false
	}

	if time.Since(info.ModTime()) > m.ttl {
		return false
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return false
	}

	var objectIDs []string
	if err := json.Unmarshal(data, &objectIDs); err != nil {
		return false
	}

	m.objectIDs = objectIDs
	m.lastRefresh = info.ModTime()
	return true
}

// fetchIndexData fetches the OBEX index from GitHub API and returns the object IDs.
// This method does NOT modify any state - it only performs network I/O and parsing.
// Caller is responsible for updating the index under appropriate locks.
func (m *Manager) fetchIndexData() ([]string, error) {
	url := fmt.Sprintf("%s/%s", GitHubAPIBase, OBEXPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "p2kb-mcp")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OBEX index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch OBEX index: HTTP %d", resp.StatusCode)
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse OBEX index: %w", err)
	}

	var objectIDs []string
	for _, entry := range entries {
		if entry.Type == "file" && strings.HasSuffix(entry.Name, ".yaml") {
			// Skip template
			if entry.Name == "_template.yaml" {
				continue
			}
			// Extract object ID from filename
			objectID := strings.TrimSuffix(entry.Name, ".yaml")
			objectIDs = append(objectIDs, objectID)
		}
	}

	sort.Strings(objectIDs)
	return objectIDs, nil
}

func (m *Manager) saveIndexToCache(objectIDs []string) {
	indexDir := filepath.Join(m.cacheDir, "obex")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return
	}

	data, err := json.Marshal(objectIDs)
	if err != nil {
		return
	}

	indexPath := filepath.Join(indexDir, "index.json")
	_ = os.WriteFile(indexPath, data, 0644)
}

func (m *Manager) fetchObject(objectID string) (*OBEXObject, error) {
	// Try to load from disk cache first
	obj, err := m.loadObjectFromCache(objectID)
	if err == nil {
		m.mu.Lock()
		m.objects[objectID] = obj
		m.mu.Unlock()
		return obj, nil
	}

	// Fetch from GitHub
	url := fmt.Sprintf("%s/%s/%s.yaml", GitHubRawBase, OBEXPath, objectID)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OBEX object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OBEX object not found: %s (HTTP %d)", objectID, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OBEX object: %w", err)
	}

	obj = &OBEXObject{}
	if err := yaml.Unmarshal(data, obj); err != nil {
		return nil, fmt.Errorf("failed to parse OBEX object: %w", err)
	}

	// Cache to memory and disk
	m.mu.Lock()
	m.objects[objectID] = obj
	m.mu.Unlock()

	m.saveObjectToCache(objectID, data)

	return obj, nil
}

func (m *Manager) loadObjectFromCache(objectID string) (*OBEXObject, error) {
	cachePath := filepath.Join(m.cacheDir, "obex", "objects", objectID+".yaml")

	// Check file age against TTL
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}

	// If file is older than TTL, treat as cache miss
	if time.Since(info.ModTime()) > m.ttl {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	obj := &OBEXObject{}
	if err := yaml.Unmarshal(data, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (m *Manager) saveObjectToCache(objectID string, data []byte) {
	cacheDir := filepath.Join(m.cacheDir, "obex", "objects")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return
	}

	cachePath := filepath.Join(cacheDir, objectID+".yaml")
	_ = os.WriteFile(cachePath, data, 0644)
}

func normalizeObjectID(objectID string) string {
	// Remove "OB" prefix if present
	objectID = strings.TrimPrefix(objectID, "OB")
	objectID = strings.TrimPrefix(objectID, "ob")
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
