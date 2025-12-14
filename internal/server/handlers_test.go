package server

import (
	"encoding/json"
	"testing"
)

// Test helper functions

func TestExtractRelatedInstructions(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "no related instructions",
			content:  "mnemonic: MOV\ndescription: test\n",
			expected: nil,
		},
		{
			name: "single related instruction",
			content: `mnemonic: MOV
related_instructions:
  - p2kbPasm2Add
description: test
`,
			expected: []string{"p2kbPasm2Add"},
		},
		{
			name: "multiple related instructions",
			content: `mnemonic: MOV
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Sub
  - p2kbPasm2Loc
description: test
`,
			expected: []string{"p2kbPasm2Add", "p2kbPasm2Sub", "p2kbPasm2Loc"},
		},
		{
			name: "related instructions at end",
			content: `mnemonic: MOV
description: test
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Sub
`,
			expected: []string{"p2kbPasm2Add", "p2kbPasm2Sub"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRelatedInstructions(tt.content)

			if len(result) != len(tt.expected) {
				t.Errorf("got %d items, want %d", len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("item %d = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsNumericID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"2811", true},
		{"OB2811", true},
		{"ob4047", true},
		{"123", true},
		{"led driver", false},
		{"i2c", false},
		{"", false},
		{"12abc", false},
	}

	for _, tt := range tests {
		result := isNumericID(tt.input)
		if result != tt.expected {
			t.Errorf("isNumericID(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Park transformation", "park-transformation"},
		{"WS2812B LED Driver", "ws2812b-led-driver"},
		{"I2C OLED Display (128x64)", "i2c-oled-display-128x64"},
		{"  Spaces & Symbols! @#$  ", "spaces-symbols"},
		{"Simple", "simple"},
		{"CamelCase", "camelcase"},
	}

	for _, tt := range tests {
		result := generateSlug(tt.input)
		if result != tt.expected {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToJSON(t *testing.T) {
	input := map[string]interface{}{
		"key":   "value",
		"count": 42,
	}

	result := toJSON(input)
	if result == "" {
		t.Error("toJSON returned empty string")
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(result), &decoded); err != nil {
		t.Errorf("toJSON produced invalid JSON: %v", err)
	}
}

// Test response helpers

func TestSuccessResponse(t *testing.T) {
	srv := New("1.0.0")
	resp := srv.successResponse(42, map[string]interface{}{"test": "value"})

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %v, want 42", resp.ID)
	}
	if resp.Error != nil {
		t.Error("Error should be nil")
	}
	if resp.Result == nil {
		t.Error("Result should not be nil")
	}
}

func TestErrorResponse(t *testing.T) {
	srv := New("1.0.0")
	resp := srv.errorResponse(42, -32600, "Invalid Request", "details")

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %v, want 42", resp.ID)
	}
	if resp.Result != nil {
		t.Error("Result should be nil")
	}
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code = %d, want -32600", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message = %q, want 'Invalid Request'", resp.Error.Message)
	}
}

// Test p2kb_version

func TestHandleVersion(t *testing.T) {
	srv := New("1.2.3")
	resp := srv.handleVersion(1)

	if resp.Error != nil {
		t.Fatalf("handleVersion returned error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not a []map")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("text is not a string")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to parse text as JSON: %v", err)
	}

	if data["mcp_version"] != "1.2.3" {
		t.Errorf("mcp_version = %v, want 1.2.3", data["mcp_version"])
	}

	// Check for index and obex sections
	if _, ok := data["index"]; !ok {
		t.Error("missing index field")
	}
	if _, ok := data["obex"]; !ok {
		t.Error("missing obex field")
	}
}

// Test p2kb_get

func TestHandleGetMissingQuery(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_get",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for missing query")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleGetInvalidArgs(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_get",
		"arguments": "not an object",
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for invalid arguments")
	}
}

// Test p2kb_find

func TestHandleFindNoParams(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_find",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Should return categories list, not an error
	// (may fail if index not available, but structure should be correct)
	if resp.Error != nil {
		// This is acceptable if index is not available
		t.Log("handleFind returned error (expected if no index):", resp.Error.Message)
	}
}

func TestHandleFindWithTerm(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_find",
		"arguments": map[string]interface{}{
			"term": "mov",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Should search for keys containing "mov"
	if resp.Error != nil {
		t.Log("handleFind with term returned error (expected if no index):", resp.Error.Message)
	}
}

// Test p2kb_obex_get

func TestHandleOBEXGetMissingQuery(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_obex_get",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for missing query")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleOBEXGetWithNumericID(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_get",
		"arguments": map[string]interface{}{
			"query": "2811",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// May fail due to network, but tests the path
	if resp.Error != nil {
		t.Log("handleOBEXGet with ID returned error (expected if no network):", resp.Error.Message)
	}
}

func TestHandleOBEXGetWithSearchTerm(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_get",
		"arguments": map[string]interface{}{
			"query": "led driver",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// May fail due to network, but tests the path
	if resp.Error != nil {
		t.Log("handleOBEXGet with search returned error (expected if no network):", resp.Error.Message)
	}
}

// Test p2kb_obex_find

func TestHandleOBEXFindNoParams(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_obex_find",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Should return overview with categories
	if resp.Error != nil {
		t.Log("handleOBEXFind returned error (expected if no network):", resp.Error.Message)
	}
}

func TestHandleOBEXFindWithCategory(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_find",
		"arguments": map[string]interface{}{
			"category": "drivers",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error != nil {
		t.Log("handleOBEXFind with category returned error (expected if no network):", resp.Error.Message)
	}
}

func TestHandleOBEXFindWithAuthor(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_find",
		"arguments": map[string]interface{}{
			"author": "Jon",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error != nil {
		t.Log("handleOBEXFind with author returned error (expected if no network):", resp.Error.Message)
	}
}

// Test p2kb_refresh

func TestHandleRefresh(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_refresh",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// May fail due to network
	if resp.Error != nil {
		t.Log("handleRefresh returned error (expected if no network):", resp.Error.Message)
	}
}

func TestHandleRefreshWithOBEX(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_refresh",
		"arguments": map[string]interface{}{
			"include_obex": true,
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error != nil {
		t.Log("handleRefresh with OBEX returned error (expected if no network):", resp.Error.Message)
	}
}

// Test unknown tool

func TestHandleToolsCallUnknownTool(t *testing.T) {
	srv := New("1.0.0")
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "unknown_tool",
		"arguments": map[string]interface{}{},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for unknown tool")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Error.Code = %d, want -32601", resp.Error.Code)
	}
}

func TestHandleToolsCallInvalidParams(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`invalid json`),
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for invalid params")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

// Test old API tools return errors (they've been removed)

func TestRemovedToolsReturnError(t *testing.T) {
	srv := New("1.0.0")
	removedTools := []string{
		"p2kb_search",
		"p2kb_browse",
		"p2kb_categories",
		"p2kb_batch_get",
		"p2kb_info",
		"p2kb_stats",
		"p2kb_related",
		"p2kb_help",
		"p2kb_cached",
		"p2kb_index_status",
		"p2kb_obex_search",
		"p2kb_obex_browse",
		"p2kb_obex_authors",
	}

	for _, tool := range removedTools {
		params, _ := json.Marshal(map[string]interface{}{
			"name":      tool,
			"arguments": map[string]interface{}{},
		})

		req := &MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/call",
			Params:  params,
		}

		resp := srv.handleRequest(req)
		if resp.Error == nil {
			t.Errorf("expected error for removed tool %s", tool)
		}
		if resp.Error.Code != -32601 {
			t.Errorf("%s: Error.Code = %d, want -32601", tool, resp.Error.Code)
		}
	}
}
