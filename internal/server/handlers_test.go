package server

import (
	"encoding/json"
	"testing"
)

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
}

func TestHandleHelp(t *testing.T) {
	srv := New("1.0.0")
	resp := srv.handleHelp(1)

	if resp.Error != nil {
		t.Fatalf("handleHelp returned error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not a []map")
	}

	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("text is not a string")
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("failed to parse text as JSON: %v", err)
	}

	tools, ok := data["tools"].([]interface{})
	if !ok {
		t.Fatal("tools is not an array")
	}

	if len(tools) != 11 {
		t.Errorf("got %d tools, want 11", len(tools))
	}

	prefixes, ok := data["key_prefixes"].(map[string]interface{})
	if !ok {
		t.Fatal("key_prefixes is not a map")
	}

	expectedPrefixes := []string{"p2kbPasm2*", "p2kbSpin2*", "p2kbArch*", "p2kbGuide*", "p2kbHw*"}
	for _, p := range expectedPrefixes {
		if _, exists := prefixes[p]; !exists {
			t.Errorf("missing prefix: %s", p)
		}
	}
}

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
