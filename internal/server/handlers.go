package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ToolCallParams represents the params for a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// handleToolsCall dispatches tool calls to their implementations.
func (s *Server) handleToolsCall(req *MCPRequest) *MCPResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	switch params.Name {
	case "p2kb_get":
		return s.handleGet(req.ID, params.Arguments)
	case "p2kb_find":
		return s.handleFind(req.ID, params.Arguments)
	case "p2kb_obex_get":
		return s.handleOBEXGet(req.ID, params.Arguments)
	case "p2kb_obex_find":
		return s.handleOBEXFind(req.ID, params.Arguments)
	case "p2kb_version":
		return s.handleVersion(req.ID)
	case "p2kb_refresh":
		return s.handleRefresh(req.ID, params.Arguments)
	default:
		return s.errorResponse(req.ID, -32601, "Unknown tool", params.Name)
	}
}

// handleGet implements p2kb_get - natural language or key-based content retrieval.
func (s *Server) handleGet(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Query == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "query")
	}

	// Try exact key match first
	if s.indexManager.KeyExists(params.Query) {
		return s.getContentWithRelated(id, params.Query)
	}

	// Use natural language matching
	matches, err := s.indexManager.MatchQuery(params.Query)
	if err != nil {
		return s.errorResponse(id, -32000, "Query failed", err.Error())
	}

	if len(matches) == 0 {
		// Try substring search as fallback
		keys := s.indexManager.Search(params.Query, 10)
		if len(keys) > 0 {
			return s.successResponse(id, map[string]interface{}{
				"type":        "suggestions",
				"query":       params.Query,
				"message":     "No exact matches found. Did you mean one of these?",
				"suggestions": keys,
			})
		}
		return s.errorResponse(id, -32000, "No matches found",
			map[string]interface{}{
				"query": params.Query,
				"hint":  "Try using p2kb_find to explore available documentation",
			})
	}

	// If single high-confidence match, return content
	if len(matches) == 1 || matches[0].Score > 0.9 {
		return s.getContentWithRelated(id, matches[0].Key)
	}

	// If top match is significantly better, return it
	if len(matches) >= 2 && matches[0].Score > matches[1].Score+0.2 {
		return s.getContentWithRelated(id, matches[0].Key)
	}

	// Multiple matches - return suggestions
	suggestions := make([]map[string]interface{}, 0, len(matches))
	for _, m := range matches {
		suggestions = append(suggestions, map[string]interface{}{
			"key":      m.Key,
			"score":    m.Score,
			"category": m.Category,
		})
	}

	return s.successResponse(id, map[string]interface{}{
		"type":        "suggestions",
		"query":       params.Query,
		"message":     "Multiple matches found. Please be more specific or use an exact key.",
		"suggestions": suggestions,
	})
}

// getContentWithRelated fetches content and extracts related items.
func (s *Server) getContentWithRelated(id interface{}, key string) *MCPResponse {
	content, err := s.getContent(key)
	if err != nil {
		return s.errorResponse(id, -32000, fmt.Sprintf("Failed to fetch content for '%s'", key), err.Error())
	}

	// Extract related instructions
	related := extractRelatedInstructions(content)
	categories := s.indexManager.GetKeyCategories(key)

	result := map[string]interface{}{
		"type":       "content",
		"key":        key,
		"content":    content,
		"categories": categories,
	}

	if len(related) > 0 {
		result["related"] = related
	}

	return s.successResponse(id, result)
}

// handleFind implements p2kb_find - explore/discover documentation.
func (s *Server) handleFind(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Term     string `json:"term"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	params.Limit = 50 // default

	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
		}
	}

	// No parameters - list categories
	if params.Term == "" && params.Category == "" {
		categories := s.indexManager.GetCategoriesWithCounts()
		stats := s.indexManager.GetStats()

		return s.successResponse(id, map[string]interface{}{
			"type":             "categories",
			"categories":       categories,
			"total_categories": stats.TotalCategories,
			"total_entries":    stats.TotalEntries,
		})
	}

	// Category only - list keys in category
	if params.Term == "" && params.Category != "" {
		keys, err := s.indexManager.GetCategoryKeys(params.Category)
		if err != nil {
			categories := s.indexManager.GetCategories()
			return s.errorResponse(id, -32000, fmt.Sprintf("Category '%s' not found", params.Category),
				map[string]interface{}{
					"available_categories": categories,
				})
		}

		if params.Limit > 0 && len(keys) > params.Limit {
			keys = keys[:params.Limit]
		}

		return s.successResponse(id, map[string]interface{}{
			"type":     "keys",
			"category": params.Category,
			"keys":     keys,
			"count":    len(keys),
		})
	}

	// Search by term
	keys := s.indexManager.Search(params.Term, params.Limit)

	// If category specified, filter results
	if params.Category != "" {
		categoryKeys, err := s.indexManager.GetCategoryKeys(params.Category)
		if err == nil {
			categorySet := make(map[string]bool)
			for _, k := range categoryKeys {
				categorySet[k] = true
			}

			filtered := make([]string, 0)
			for _, k := range keys {
				if categorySet[k] {
					filtered = append(filtered, k)
				}
			}
			keys = filtered
		}
	}

	return s.successResponse(id, map[string]interface{}{
		"type":  "keys",
		"term":  params.Term,
		"keys":  keys,
		"count": len(keys),
	})
}

// handleOBEXGet implements p2kb_obex_get - OBEX object retrieval.
func (s *Server) handleOBEXGet(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Query == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "query")
	}

	// Check if query is a numeric ID
	if isNumericID(params.Query) {
		return s.getOBEXObject(id, params.Query)
	}

	// Search for matching objects
	results, err := s.obexManager.Search(params.Query, "", "", 10)
	if err != nil {
		return s.errorResponse(id, -32000, "OBEX search failed", err.Error())
	}

	if len(results) == 0 {
		return s.errorResponse(id, -32000, "No OBEX objects found",
			map[string]interface{}{
				"query": params.Query,
				"hint":  "Try using p2kb_obex_find to explore available objects",
			})
	}

	// Single result - return full object info
	if len(results) == 1 {
		return s.getOBEXObject(id, results[0].ObjectID)
	}

	// Multiple results - return as suggestions
	suggestions := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		suggestions = append(suggestions, map[string]interface{}{
			"object_id":   r.ObjectID,
			"title":       r.Title,
			"author":      r.Author,
			"category":    r.Category,
			"description": r.DescriptionShort,
			"match_type":  r.MatchType,
		})
	}

	return s.successResponse(id, map[string]interface{}{
		"type":        "suggestions",
		"query":       params.Query,
		"message":     "Multiple OBEX objects found. Specify an object_id or refine your search.",
		"suggestions": suggestions,
	})
}

// getOBEXObject returns full OBEX object info with download instructions.
func (s *Server) getOBEXObject(id interface{}, objectID string) *MCPResponse {
	obj, err := s.obexManager.GetObject(objectID)
	if err != nil {
		return s.errorResponse(id, -32000, fmt.Sprintf("OBEX object not found: %s", objectID),
			map[string]interface{}{
				"hint": "Use p2kb_obex_find to search for objects",
			})
	}

	meta := obj.ObjectMetadata
	downloadURL := s.obexManager.GetDownloadURL(meta.ObjectID)

	// Generate slug for directory naming
	slug := generateSlug(meta.Title)

	return s.successResponse(id, map[string]interface{}{
		"type":         "obex_object",
		"object_id":    meta.ObjectID,
		"title":        meta.Title,
		"author":       meta.Author,
		"category":     meta.Functionality.Category,
		"description":  meta.Functionality.DescriptionShort,
		"languages":    meta.TechnicalDetails.Languages,
		"tags":         meta.Functionality.Tags,
		"download_url": downloadURL,
		"obex_page":    meta.URLs.OBEXPage,
		"download_instructions": map[string]interface{}{
			"suggested_directory": fmt.Sprintf("OBEX/%s", slug),
			"filename":            fmt.Sprintf("OB%s.zip", meta.ObjectID),
			"command":             fmt.Sprintf("curl -L -o OB%s.zip '%s'", meta.ObjectID, downloadURL),
		},
		"metadata": map[string]interface{}{
			"version":      meta.TechnicalDetails.Version,
			"file_size":    meta.TechnicalDetails.FileSize,
			"quality":      meta.Metadata.QualityScore,
			"created_date": meta.Metadata.CreatedDate,
		},
	})
}

// handleOBEXFind implements p2kb_obex_find - explore OBEX objects.
func (s *Server) handleOBEXFind(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Term     string `json:"term"`
		Category string `json:"category"`
		Author   string `json:"author"`
		Limit    int    `json:"limit"`
	}
	params.Limit = 20 // default

	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
		}
	}

	// No parameters - list categories
	if params.Term == "" && params.Category == "" && params.Author == "" {
		categories, err := s.obexManager.GetCategories()
		if err != nil {
			return s.errorResponse(id, -32000, "Failed to get OBEX categories", err.Error())
		}

		// Also get top authors
		authors, _ := s.obexManager.GetAuthors()
		topAuthors := authors
		if len(topAuthors) > 5 {
			topAuthors = topAuthors[:5]
		}

		return s.successResponse(id, map[string]interface{}{
			"type":          "overview",
			"categories":    categories,
			"total_objects": s.obexManager.GetTotalObjects(),
			"top_authors":   topAuthors,
		})
	}

	// Author filter
	if params.Author != "" && params.Term == "" && params.Category == "" {
		// Get all objects and filter by author
		objects, err := s.obexManager.BrowseCategory("")
		if err != nil {
			return s.errorResponse(id, -32000, "Failed to browse OBEX", err.Error())
		}

		filtered := make([]map[string]interface{}, 0)
		for _, obj := range objects {
			if strings.Contains(strings.ToLower(obj.Author), strings.ToLower(params.Author)) {
				filtered = append(filtered, map[string]interface{}{
					"object_id":   obj.ObjectID,
					"title":       obj.Title,
					"author":      obj.Author,
					"category":    obj.Category,
					"description": obj.DescriptionShort,
				})
				if len(filtered) >= params.Limit {
					break
				}
			}
		}

		return s.successResponse(id, map[string]interface{}{
			"type":    "objects",
			"author":  params.Author,
			"objects": filtered,
			"count":   len(filtered),
		})
	}

	// Search or browse
	if params.Term != "" {
		results, err := s.obexManager.Search(params.Term, params.Category, "", params.Limit)
		if err != nil {
			return s.errorResponse(id, -32000, "OBEX search failed", err.Error())
		}

		objects := make([]map[string]interface{}, 0, len(results))
		for _, r := range results {
			objects = append(objects, map[string]interface{}{
				"object_id":   r.ObjectID,
				"title":       r.Title,
				"author":      r.Author,
				"category":    r.Category,
				"description": r.DescriptionShort,
				"match_type":  r.MatchType,
			})
		}

		return s.successResponse(id, map[string]interface{}{
			"type":    "objects",
			"term":    params.Term,
			"objects": objects,
			"count":   len(objects),
		})
	}

	// Category browse
	if params.Category != "" {
		objects, err := s.obexManager.BrowseCategory(params.Category)
		if err != nil {
			return s.errorResponse(id, -32000, "Failed to browse category", err.Error())
		}

		if len(objects) > params.Limit {
			objects = objects[:params.Limit]
		}

		result := make([]map[string]interface{}, 0, len(objects))
		for _, obj := range objects {
			result = append(result, map[string]interface{}{
				"object_id":   obj.ObjectID,
				"title":       obj.Title,
				"author":      obj.Author,
				"description": obj.DescriptionShort,
			})
		}

		return s.successResponse(id, map[string]interface{}{
			"type":     "objects",
			"category": params.Category,
			"objects":  result,
			"count":    len(result),
		})
	}

	// Shouldn't reach here
	return s.errorResponse(id, -32602, "Invalid parameters", nil)
}

// handleVersion implements p2kb_version.
func (s *Server) handleVersion(id interface{}) *MCPResponse {
	stats := s.indexManager.GetStats()
	indexStatus := s.indexManager.GetIndexStatus()
	obexMem, obexDisk, obexStale := s.obexManager.GetCacheStats()

	return s.successResponse(id, map[string]interface{}{
		"mcp_version":  s.version,
		"index_version": stats.Version,
		"index": map[string]interface{}{
			"total_entries":    stats.TotalEntries,
			"total_categories": stats.TotalCategories,
			"is_cached":        indexStatus.IsCached,
			"age_seconds":      indexStatus.AgeSeconds,
			"needs_refresh":    indexStatus.NeedsRefresh,
		},
		"obex": map[string]interface{}{
			"total_objects":       s.obexManager.GetTotalObjects(),
			"cached_memory":       obexMem,
			"cached_disk":         obexDisk,
			"stale_cache_entries": obexStale,
		},
	})
}

// handleRefresh implements p2kb_refresh - smart cache refresh.
func (s *Server) handleRefresh(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		IncludeOBEX bool `json:"include_obex"`
	}

	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
		}
	}

	// Refresh index
	if err := s.indexManager.Refresh(); err != nil {
		return s.errorResponse(id, -32000, "Failed to refresh index", err.Error())
	}

	// Detect and invalidate stale cache entries
	cachedKeys := s.cacheManager.GetCachedKeys()
	staleKeys := s.indexManager.GetStaleKeys(cachedKeys, s.cacheManager.GetMtime)
	invalidatedCount := s.cacheManager.InvalidateKeys(staleKeys)

	result := map[string]interface{}{
		"refreshed":            true,
		"stale_keys_found":     len(staleKeys),
		"cache_entries_invalidated": invalidatedCount,
	}

	// Optionally refresh OBEX
	if params.IncludeOBEX {
		if err := s.obexManager.Refresh(); err != nil {
			result["obex_error"] = err.Error()
		} else {
			result["obex_refreshed"] = true
		}
	}

	stats := s.indexManager.GetStats()
	result["index_version"] = stats.Version
	result["total_entries"] = stats.TotalEntries

	return s.successResponse(id, result)
}

// Helper methods

func (s *Server) getContent(key string) (string, error) {
	// Check cache first
	if content, found := s.cacheManager.Get(key); found {
		return content, nil
	}

	// Get path from index
	path, mtime, err := s.indexManager.GetKeyPath(key)
	if err != nil {
		return "", err
	}

	// Fetch from remote
	content, err := s.cacheManager.FetchAndCache(key, path, mtime)
	if err != nil {
		return "", err
	}

	return content, nil
}

func (s *Server) successResponse(id interface{}, result interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": toJSON(result),
				},
			},
		},
	}
}

func (s *Server) errorResponse(id interface{}, code int, message string, data interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func toJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// extractRelatedInstructions parses the related_instructions field from YAML content.
func extractRelatedInstructions(content string) []string {
	var related []string
	lines := strings.Split(content, "\n")
	inRelated := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "related_instructions:") {
			inRelated = true
			continue
		}
		if inRelated {
			if strings.HasPrefix(trimmed, "- ") {
				key := strings.TrimPrefix(trimmed, "- ")
				key = strings.TrimSpace(key)
				if key != "" {
					related = append(related, key)
				}
			} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
				// End of related_instructions block
				break
			}
		}
	}

	return related
}

// isNumericID checks if the query is a numeric object ID.
func isNumericID(query string) bool {
	query = strings.TrimSpace(query)
	query = strings.TrimPrefix(query, "OB")
	query = strings.TrimPrefix(query, "ob")

	matched, _ := regexp.MatchString(`^\d+$`, query)
	return matched
}

// generateSlug creates a filesystem-safe slug from a title.
func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)

	// Replace non-alphanumeric with hyphens
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

	// Trim leading/trailing hyphens
	return strings.Trim(result.String(), "-")
}
