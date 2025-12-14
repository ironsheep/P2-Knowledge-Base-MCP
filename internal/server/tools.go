package server

// Tool represents an MCP tool definition with JSON Schema for input validation.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns the complete list of available P2KB tools.
// The API is intentionally minimal to reduce cognitive load for Claude.
func GetToolDefinitions() []Tool {
	return []Tool{
		// Primary content access - natural language query
		{
			Name: "p2kb_get",
			Description: `Fetch P2 Knowledge Base content using natural language or exact key.
Accepts natural language queries like "mov instruction", "cog architecture", "spin2 pinwrite".
Also accepts exact keys like "p2kbPasm2Mov" for direct lookup.
Returns the content along with related items for exploration.
If query is ambiguous, returns matching suggestions.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
						"description": `Natural language query or exact key.
Examples: "mov instruction", "pasm2 add", "spin2 pinwrite", "cog memory", "smart pin", "p2kbPasm2Mov"`,
					},
				},
				"required": []string{"query"},
			},
		},

		// Discovery/exploration tool
		{
			Name: "p2kb_find",
			Description: `Explore and discover P2KB documentation. Use to find what's available.
With no parameters: lists all categories with counts.
With term: searches for matching keys.
With category: lists keys in that category.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"term": map[string]interface{}{
						"type":        "string",
						"description": "Search term to find matching keys (optional)",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category to browse (e.g., 'pasm2_math', 'spin2_pin'). Use without term to list all keys in category.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results (default: 50)",
						"default":     50,
					},
				},
			},
		},

		// OBEX code retrieval - natural language query with download
		{
			Name: "p2kb_obex_get",
			Description: `Get OBEX (Parallax Object Exchange) code object by search or ID.
Searches OBEX objects using natural language (e.g., "i2c sensor", "led driver") or retrieves by numeric ID.
Search terms are automatically expanded (i2c matches twi, iic; led matches ws2812, neopixel).
Returns object metadata with download URL and instructions.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
						"description": `Natural language search or numeric object ID.
Examples: "i2c sensor", "led driver", "servo motor", "2811", "4047"`,
					},
				},
				"required": []string{"query"},
			},
		},

		// OBEX discovery/exploration
		{
			Name: "p2kb_obex_find",
			Description: `Explore OBEX objects. Lists categories, searches, or browses by category/author.
With no parameters: lists all categories with counts.
With term: searches across all objects.
With category: lists objects in that category.
With author: lists objects by that author.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"term": map[string]interface{}{
						"type":        "string",
						"description": "Search term (optional)",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category filter: drivers, misc, display, demos, audio, motors, communication, sensors, tools",
					},
					"author": map[string]interface{}{
						"type":        "string",
						"description": "Filter by author name",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results (default: 20)",
						"default":     20,
					},
				},
			},
		},

		// Version diagnostic
		{
			Name:        "p2kb_version",
			Description: "Get P2KB MCP server version and index info. Useful for debugging.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},

		// User-triggered refresh
		{
			Name: "p2kb_refresh",
			Description: `Force refresh of P2KB index and invalidate stale cache entries.
Use when the knowledge base has been updated and you need fresh content.
Automatically detects and removes stale cached items based on index timestamps.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"include_obex": map[string]interface{}{
						"type":        "boolean",
						"description": "Also refresh OBEX index and clear stale OBEX cache (default: false)",
						"default":     false,
					},
				},
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
