package server

import (
	"encoding/json"
	"fmt"
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
	case "p2kb_search":
		return s.handleSearch(req.ID, params.Arguments)
	case "p2kb_browse":
		return s.handleBrowse(req.ID, params.Arguments)
	case "p2kb_categories":
		return s.handleCategories(req.ID)
	case "p2kb_version":
		return s.handleVersion(req.ID)
	case "p2kb_batch_get":
		return s.handleBatchGet(req.ID, params.Arguments)
	case "p2kb_refresh":
		return s.handleRefresh(req.ID, params.Arguments)
	case "p2kb_info":
		return s.handleInfo(req.ID, params.Arguments)
	case "p2kb_stats":
		return s.handleStats(req.ID)
	case "p2kb_related":
		return s.handleRelated(req.ID, params.Arguments)
	case "p2kb_help":
		return s.handleHelp(req.ID)
	default:
		return s.errorResponse(req.ID, -32601, "Unknown tool", params.Name)
	}
}

// handleGet implements p2kb_get tool.
func (s *Server) handleGet(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Key == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "key")
	}

	content, err := s.getContent(params.Key)
	if err != nil {
		// Try to provide suggestions
		suggestions := s.indexManager.FindSimilarKeys(params.Key, 5)
		return s.errorResponse(id, -32000, fmt.Sprintf("Key '%s' not found", params.Key),
			map[string]interface{}{
				"suggestions": suggestions,
				"hint":        "Use p2kb_search to find valid keys",
			})
	}

	return s.successResponse(id, map[string]interface{}{
		"content": content,
	})
}

// handleSearch implements p2kb_search tool.
func (s *Server) handleSearch(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Term  string `json:"term"`
		Limit int    `json:"limit"`
	}
	params.Limit = 50 // default

	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Term == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "term")
	}

	keys := s.indexManager.Search(params.Term, params.Limit)

	return s.successResponse(id, map[string]interface{}{
		"keys":  keys,
		"count": len(keys),
		"term":  params.Term,
	})
}

// handleBrowse implements p2kb_browse tool.
func (s *Server) handleBrowse(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Category string `json:"category"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Category == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "category")
	}

	keys, err := s.indexManager.GetCategoryKeys(params.Category)
	if err != nil {
		categories := s.indexManager.GetCategories()
		return s.errorResponse(id, -32000, fmt.Sprintf("Category '%s' not found", params.Category),
			map[string]interface{}{
				"available_categories": categories,
			})
	}

	return s.successResponse(id, map[string]interface{}{
		"category": params.Category,
		"keys":     keys,
		"count":    len(keys),
	})
}

// handleCategories implements p2kb_categories tool.
func (s *Server) handleCategories(id interface{}) *MCPResponse {
	categories := s.indexManager.GetCategoriesWithCounts()
	stats := s.indexManager.GetStats()

	return s.successResponse(id, map[string]interface{}{
		"categories":       categories,
		"total_categories": stats.TotalCategories,
		"total_entries":    stats.TotalEntries,
	})
}

// handleVersion implements p2kb_version tool.
func (s *Server) handleVersion(id interface{}) *MCPResponse {
	return s.successResponse(id, map[string]interface{}{
		"mcp_version": s.version,
	})
}

// handleBatchGet implements p2kb_batch_get tool.
func (s *Server) handleBatchGet(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if len(params.Keys) == 0 {
		return s.errorResponse(id, -32602, "Missing required parameter", "keys")
	}

	results := make(map[string]interface{})
	successCount := 0
	errorCount := 0

	for _, key := range params.Keys {
		content, err := s.getContent(key)
		if err != nil {
			results[key] = map[string]interface{}{"error": err.Error()}
			errorCount++
		} else {
			results[key] = map[string]interface{}{"content": content}
			successCount++
		}
	}

	return s.successResponse(id, map[string]interface{}{
		"results": results,
		"success": successCount,
		"errors":  errorCount,
	})
}

// handleRefresh implements p2kb_refresh tool.
func (s *Server) handleRefresh(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		InvalidateCache bool `json:"invalidate_cache"`
		PrefetchCommon  bool `json:"prefetch_common"`
	}
	params.PrefetchCommon = true // default

	if len(args) > 0 {
		if err := json.Unmarshal(args, &params); err != nil {
			return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
		}
	}

	// Refresh index
	if err := s.indexManager.Refresh(); err != nil {
		return s.errorResponse(id, -32000, "Failed to refresh index", err.Error())
	}

	// Optionally invalidate cache
	if params.InvalidateCache {
		s.cacheManager.Clear()
	}

	// Prefetch common keys
	if params.PrefetchCommon {
		commonKeys := []string{
			"p2kbGuideQuickQueries",
			"p2kbGuideSpin2GettingStarted",
			"p2kbGuidePasm2GettingStarted",
		}
		for _, key := range commonKeys {
			_, _ = s.getContent(key) // Ignore errors, best effort
		}
	}

	stats := s.indexManager.GetStats()
	return s.successResponse(id, map[string]interface{}{
		"refreshed":     true,
		"version":       stats.Version,
		"total_entries": stats.TotalEntries,
	})
}

// handleInfo implements p2kb_info tool.
func (s *Server) handleInfo(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Key == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "key")
	}

	exists := s.indexManager.KeyExists(params.Key)
	categories := s.indexManager.GetKeyCategories(params.Key)

	return s.successResponse(id, map[string]interface{}{
		"key":        params.Key,
		"exists":     exists,
		"categories": categories,
	})
}

// handleStats implements p2kb_stats tool.
func (s *Server) handleStats(id interface{}) *MCPResponse {
	stats := s.indexManager.GetStats()

	return s.successResponse(id, map[string]interface{}{
		"version":          stats.Version,
		"total_entries":    stats.TotalEntries,
		"total_categories": stats.TotalCategories,
	})
}

// handleRelated implements p2kb_related tool.
func (s *Server) handleRelated(id interface{}, args json.RawMessage) *MCPResponse {
	var params struct {
		Key          string `json:"key"`
		FetchContent bool   `json:"fetch_content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return s.errorResponse(id, -32602, "Invalid arguments", err.Error())
	}

	if params.Key == "" {
		return s.errorResponse(id, -32602, "Missing required parameter", "key")
	}

	// Get the content and extract related instructions
	content, err := s.getContent(params.Key)
	if err != nil {
		return s.errorResponse(id, -32000, fmt.Sprintf("Key '%s' not found", params.Key), nil)
	}

	// Parse related_instructions from YAML content
	related := extractRelatedInstructions(content)

	result := map[string]interface{}{
		"key":     params.Key,
		"related": related,
	}

	// Optionally fetch content for related items
	if params.FetchContent && len(related) > 0 {
		relatedContent := make(map[string]string)
		for _, relKey := range related {
			if c, err := s.getContent(relKey); err == nil {
				relatedContent[relKey] = c
			}
		}
		result["content"] = relatedContent
	}

	return s.successResponse(id, result)
}

// handleHelp implements p2kb_help tool.
func (s *Server) handleHelp(id interface{}) *MCPResponse {
	tools := []string{
		"p2kb_get", "p2kb_search", "p2kb_browse", "p2kb_categories",
		"p2kb_version", "p2kb_batch_get", "p2kb_refresh", "p2kb_info",
		"p2kb_stats", "p2kb_related", "p2kb_help",
	}

	keyPrefixes := map[string]string{
		"p2kbPasm2*": "PASM2 assembly instructions",
		"p2kbSpin2*": "Spin2 methods",
		"p2kbArch*":  "Architecture documentation",
		"p2kbGuide*": "Guides and quick references",
		"p2kbHw*":    "Hardware specifications",
	}

	return s.successResponse(id, map[string]interface{}{
		"tools":        tools,
		"key_prefixes": keyPrefixes,
	})
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
