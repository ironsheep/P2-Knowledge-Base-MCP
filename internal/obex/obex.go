// Package obex manages OBEX (Parallax Object Exchange) metadata.
package obex

import (
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

// Manager handles OBEX operations.
type Manager struct {
	mu          sync.RWMutex
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
func (m *Manager) EnsureIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have a fresh index
	if len(m.objectIDs) > 0 && time.Since(m.lastRefresh) < m.ttl {
		return nil
	}

	// Try to load from cache
	if m.loadIndexFromCache() {
		return nil
	}

	// Fetch from GitHub API
	return m.fetchIndex()
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

// Refresh forces a refresh of the OBEX index and clears stale objects.
func (m *Manager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear memory cache
	m.objects = make(map[string]*OBEXObject)

	// Fetch fresh index
	return m.fetchIndex()
}

// ClearCache clears all cached OBEX data.
func (m *Manager) ClearCache() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.objects)
	m.objects = make(map[string]*OBEXObject)
	m.objectIDs = nil
	m.lastRefresh = time.Time{}

	// Clear disk cache
	cacheDir := filepath.Join(m.cacheDir, "obex")
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

func (m *Manager) fetchIndex() error {
	url := fmt.Sprintf("%s/%s", GitHubAPIBase, OBEXPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "p2kb-mcp")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch OBEX index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch OBEX index: HTTP %d", resp.StatusCode)
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("failed to parse OBEX index: %w", err)
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

	// Save to cache
	m.saveIndexToCache(objectIDs)

	m.objectIDs = objectIDs
	m.lastRefresh = time.Now()
	return nil
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
