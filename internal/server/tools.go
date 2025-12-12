package server

// Tool represents an MCP tool definition with JSON Schema for input validation.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns the complete list of available P2KB tools.
func GetToolDefinitions() []Tool {
	return []Tool{
		// Core Tools (Script Parity)
		{
			Name:        "p2kb_get",
			Description: "Fetch P2 Knowledge Base content by key. Returns YAML documentation for P2 instructions, architecture, smart pins, and Spin2 methods. Use p2kb_search or p2kb_browse to discover valid keys.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The p2kb key to fetch. Examples: 'p2kbPasm2Mov' (MOV instruction), 'p2kbArchCog' (COG architecture), 'p2kbSpin2Pinwrite' (Spin2 pinwrite method). Use p2kb_search('mov') to find keys.",
					},
				},
				"required": []string{"key"},
			},
		},
		{
			Name:        "p2kb_search",
			Description: "Search for P2KB keys matching a term (case-insensitive). Returns a list of matching keys that can be used with p2kb_get.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"term": map[string]interface{}{
						"type":        "string",
						"description": "Search term to match against keys",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 50)",
						"default":     50,
					},
				},
				"required": []string{"term"},
			},
		},
		{
			Name:        "p2kb_browse",
			Description: "List all keys in a specific category. Use p2kb_categories to see available categories.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category name (e.g., 'pasm2_branch', 'pasm2_math', 'spin2_pin'). Use p2kb_categories to list all categories.",
					},
				},
				"required": []string{"category"},
			},
		},
		{
			Name:        "p2kb_categories",
			Description: "List all available categories in the P2 Knowledge Base with entry counts.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "p2kb_version",
			Description: "Get the P2KB MCP server version.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},

		// Enhanced Tools
		{
			Name:        "p2kb_batch_get",
			Description: "Fetch multiple P2KB keys in one call. More efficient than multiple p2kb_get calls for related lookups.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keys": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Array of p2kb keys to fetch",
					},
				},
				"required": []string{"keys"},
			},
		},
		{
			Name:        "p2kb_refresh",
			Description: "Force refresh of the P2KB index and optionally invalidate cached content. Use when you suspect the knowledge base has been updated.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"invalidate_cache": map[string]interface{}{
						"type":        "boolean",
						"description": "Clear all cached YAML files (default: false)",
						"default":     false,
					},
					"prefetch_common": map[string]interface{}{
						"type":        "boolean",
						"description": "Prefetch common guide keys after refresh (default: true)",
						"default":     true,
					},
				},
			},
		},
		{
			Name:        "p2kb_info",
			Description: "Check if a key exists and what categories it belongs to, without fetching the full content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The p2kb key to check",
					},
				},
				"required": []string{"key"},
			},
		},
		{
			Name:        "p2kb_stats",
			Description: "Get P2 Knowledge Base statistics including version, total entries, and category counts.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "p2kb_related",
			Description: "Get related instructions for a key by parsing the related_instructions field from its YAML content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The p2kb key to find related items for",
					},
					"fetch_content": map[string]interface{}{
						"type":        "boolean",
						"description": "Also fetch the content of related items (default: false)",
						"default":     false,
					},
				},
				"required": []string{"key"},
			},
		},
		{
			Name:        "p2kb_help",
			Description: "Get usage information about P2KB MCP tools and key naming conventions.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// handleToolsList returns the list of available tools in MCP format.
func (s *Server) handleToolsList(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": GetToolDefinitions(),
		},
	}
}
