package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
)

// Test helper functions

// movBody is a content file that lists related instructions and carries
// provenance the built-in rule removes.
const movBody = `# maintainer note
mnemonic: MOV
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Sub
source: manual p.12
description: Move data
`

// serveMov makes r serve movBody as p2kbPasm2Mov, reachable by alias "MOV".
func serveMov(r *kbtest.Remote) {
	r.AddFile("p2kbPasm2Mov", "deliverables/ai/P2/pasm2/mov.yaml", 1700000000, movBody)
	idx := r.Index()
	idx.Aliases = map[string][]string{"MOV": {"p2kbPasm2Mov"}}
	idx.Categories = map[string][]string{"pasm2_data": {"p2kbPasm2Mov"}}
	r.SetIndex(idx)
}

func TestGetResultCarriesNoRelatedField(t *testing.T) {
	srv, r := newTestServer(t)
	serveMov(r)

	resp := srv.contentResponse(1, "p2kbPasm2Mov", "")
	if resp.Error != nil {
		t.Fatalf("contentResponse: %v", resp.Error)
	}
	got := extractResultMap(t, resp)

	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if want := []string{"categories", "content", "key", "type"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("result fields = %v, want %v", keys, want)
	}
	if _, ok := got["related"]; ok {
		t.Error("p2kb_get result still carries `related`")
	}
	// The related keys are still delivered, inside the content.
	want := filter.Apply(filter.BuiltinRule, movBody)
	if got["content"] != want {
		t.Errorf("content =\n%q\nwant\n%q", got["content"], want)
	}
}

func TestGetByAliasSetsResolvedFrom(t *testing.T) {
	srv, r := newTestServer(t)
	serveMov(r)

	resp := srv.handleGet(1, json.RawMessage(`{"query":"MOV"}`))
	if resp.Error != nil {
		t.Fatalf("handleGet: %v", resp.Error)
	}
	got := extractResultMap(t, resp)
	if got["resolved_from"] != "MOV" || got["key"] != "p2kbPasm2Mov" {
		t.Errorf("resolved_from = %v, key = %v; want MOV, p2kbPasm2Mov", got["resolved_from"], got["key"])
	}
	if _, ok := got["related"]; ok {
		t.Error("alias lookup result carries `related`")
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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

// versionResult calls handleVersion and returns its decoded result.
func versionResult(t *testing.T, srv *Server) map[string]interface{} {
	t.Helper()
	resp := srv.handleVersion(1)
	if resp.Error != nil {
		t.Fatalf("handleVersion: %v", resp.Error)
	}
	return extractResultMap(t, resp)
}

func TestVersionReportsBuiltinRule(t *testing.T) {
	srv, _ := newTestServer(t)
	got := versionResult(t, srv)
	if got["filter_rule_id"] != "b431af2a9f515c11fe5fd982642c636c6f8843bf89ed2c3003ee0e28c04877ee" {
		t.Errorf("filter_rule_id = %v", got["filter_rule_id"])
	}
	if got["filter_rule_source"] != "built-in" {
		t.Errorf("filter_rule_source = %v, want built-in", got["filter_rule_source"])
	}
	if got["filter_engine_version"] != "1" {
		t.Errorf("filter_engine_version = %v, want 1", got["filter_engine_version"])
	}
	if _, ok := got["filter_rule_refused"]; ok {
		t.Error("filter_rule_refused present without a refused block")
	}
}

func TestVersionReportsIndexRule(t *testing.T) {
	srv, r := newTestServer(t)
	idx := r.Index()
	idx.DeliveryFilter = json.RawMessage(testRuleBlock)
	r.SetIndex(idx)
	if err := srv.indexManager.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	rule, _ := filter.ParseRule(json.RawMessage(testRuleBlock))

	got := versionResult(t, srv)
	if got["filter_rule_id"] != rule.RuleID() || got["filter_rule_source"] != "index" {
		t.Errorf("filter_rule_id, source = %v, %v; want %s, index", got["filter_rule_id"], got["filter_rule_source"], rule.RuleID())
	}
	if _, ok := got["filter_rule_refused"]; ok {
		t.Error("filter_rule_refused present for an accepted block")
	}
}

func TestVersionReportsLastGoodRule(t *testing.T) {
	srv, _ := newTestServer(t, func(cacheDir string) {
		dir := filepath.Join(cacheDir, "index")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "delivery-filter.json"), []byte(testRuleBlock), 0644); err != nil {
			t.Fatal(err)
		}
	})
	rule, _ := filter.ParseRule(json.RawMessage(testRuleBlock))

	got := versionResult(t, srv)
	if got["filter_rule_id"] != rule.RuleID() || got["filter_rule_source"] != "last-good" {
		t.Errorf("filter_rule_id, source = %v, %v; want %s, last-good", got["filter_rule_id"], got["filter_rule_source"], rule.RuleID())
	}
}

func TestVersionReportsRefusedBlock(t *testing.T) {
	srv, r := newTestServer(t)
	idx := r.Index()
	idx.DeliveryFilter = json.RawMessage(`{"format":2,"remove_fields":["source"],"remove_column0_comments":true}`)
	r.SetIndex(idx)
	if err := srv.indexManager.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}

	got := versionResult(t, srv)
	refused, ok := got["filter_rule_refused"].(map[string]interface{})
	if !ok {
		t.Fatalf("filter_rule_refused = %v, want an object", got["filter_rule_refused"])
	}
	if refused["format"] != float64(2) {
		t.Errorf("filter_rule_refused.format = %v, want 2", refused["format"])
	}
	if reason, _ := refused["reason"].(string); reason == "" {
		t.Error("filter_rule_refused.reason is empty")
	}
	if got["filter_rule_source"] != "built-in" {
		t.Errorf("filter_rule_source = %v, want built-in (refused block, no last-good)", got["filter_rule_source"])
	}
}

func TestHandleVersion(t *testing.T) {
	srv, _ := newTestServer(t)
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

	if data["mcp_version"] != testVersion {
		t.Errorf("mcp_version = %v, want %s", data["mcp_version"], testVersion)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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
	srv, _ := newTestServer(t)
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

// Test p2kb_obex_download

func TestHandleOBEXDownloadMissingObjectID(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_obex_download",
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
		t.Error("expected error for missing object_id")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleOBEXDownloadEmptyObjectID(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_download",
		"arguments": map[string]interface{}{
			"object_id": "",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	if resp.Error == nil {
		t.Error("expected error for empty object_id")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleOBEXDownloadInvalidArgs(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name":      "p2kb_obex_download",
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
	if resp.Error.Code != -32602 {
		t.Errorf("Error.Code = %d, want -32602", resp.Error.Code)
	}
}

func TestHandleOBEXDownloadWithObjectID(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_download",
		"arguments": map[string]interface{}{
			"object_id": "2811",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Will fail due to network, but tests the path
	if resp.Error != nil {
		t.Log("handleOBEXDownload returned error (expected if no network):", resp.Error.Message)
		// Verify it's a -32000 error (operation failed) not -32602 (invalid params)
		if resp.Error.Code == -32602 {
			t.Error("should not be invalid params error")
		}
	}
}

func TestHandleOBEXDownloadWithTargetDir(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_download",
		"arguments": map[string]interface{}{
			"object_id":  "2811",
			"target_dir": "custom/output/path",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Will fail due to network, but tests the path
	if resp.Error != nil {
		t.Log("handleOBEXDownload with target_dir returned error (expected if no network):", resp.Error.Message)
	}
}

func TestHandleOBEXDownloadWithPathTraversal(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_download",
		"arguments": map[string]interface{}{
			"object_id":  "2811",
			"target_dir": "../../../etc/passwd",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Should fail due to path traversal attempt (or network, but the path check comes later)
	if resp.Error != nil {
		t.Log("handleOBEXDownload with path traversal returned error:", resp.Error.Message)
	}
}

func TestHandleOBEXDownloadWithOBPrefix(t *testing.T) {
	srv, _ := newTestServer(t)
	params, _ := json.Marshal(map[string]interface{}{
		"name": "p2kb_obex_download",
		"arguments": map[string]interface{}{
			"object_id": "OB2811",
		},
	})

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	resp := srv.handleRequest(req)
	// Will fail due to network, but tests that OB prefix is handled
	if resp.Error != nil {
		t.Log("handleOBEXDownload with OB prefix returned error (expected if no network):", resp.Error.Message)
		// Should NOT be a "missing parameter" error
		if resp.Error.Code == -32602 {
			t.Error("OB prefix should be accepted")
		}
	}
}

// testVersion is the version every test server reports.
const testVersion = "1.0.0"

// newTestServer creates the server every server test uses, isolated from the
// developer's machine: the cache dir is a temp dir, the index and content come
// from a fresh kbtest.Remote (returned so the test can shape what it serves),
// and the working directory is a temp dir so downloads land there. Each seed
// runs on the cache dir before New, to model what a previous run left there.
// It is the only caller of New in the package's tests.
func newTestServer(t *testing.T, seeds ...func(cacheDir string)) (*Server, *kbtest.Remote) {
	t.Helper()
	r := kbtest.NewRemote(t)
	cacheDir := t.TempDir()
	t.Setenv(fetch.BaseURLEnv, r.URL())
	t.Setenv("P2KB_CACHE_DIR", cacheDir)
	chdirTemp(t)
	for _, seed := range seeds {
		seed(cacheDir)
	}
	return New(testVersion), r
}

// chdirTemp moves the test into a temp working directory and moves it back on
// cleanup. (testing.T.Chdir arrives in Go 1.24.)
func chdirTemp(t *testing.T) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// seedDiskCache writes fake YAML files into the server's cache dir so
// GetCachedKeys() reports them.  It returns the cache dir path and the keys
// written.
func seedDiskCache(t *testing.T) (cacheDir string, keys []string) {
	t.Helper()
	cacheDir = os.Getenv("P2KB_CACHE_DIR")
	if cacheDir == "" {
		t.Fatal("P2KB_CACHE_DIR not set; call newTestServer first")
	}
	subDir := filepath.Join(cacheDir, "cache")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir cache subdir: %v", err)
	}
	keys = []string{"testKey1", "testKey2"}
	for _, k := range keys {
		path := filepath.Join(subDir, k+".yaml")
		if err := os.WriteFile(path, []byte("mnemonic: "+k), 0644); err != nil {
			t.Fatalf("write seed file %s: %v", path, err)
		}
	}
	return cacheDir, keys
}

// TestHandleRefreshFlushEmptiesCache verifies that flush:true wipes both the
// memory and disk cache entirely.  Index refresh is served by a local httptest
// server so no live network is required.
func TestHandleRefreshFlushEmptiesCache(t *testing.T) {
	srv, _ := newTestServer(t)

	// Seed the disk cache with two fake entries.
	_, seededKeys := seedDiskCache(t)
	if got := len(srv.cacheManager.GetCachedKeys()); got != len(seededKeys) {
		t.Fatalf("pre-flush: expected %d cached keys, got %d", len(seededKeys), got)
	}

	// Invoke p2kb_refresh with flush:true
	args, _ := json.Marshal(map[string]interface{}{"flush": true})
	resp := srv.handleRefresh(1, args)

	if resp.Error != nil {
		t.Fatalf("handleRefresh(flush:true) returned error: %v", resp.Error)
	}

	// Cache must be empty after flush.
	remaining := srv.cacheManager.GetCachedKeys()
	if len(remaining) != 0 {
		t.Errorf("post-flush: expected 0 cached keys, got %d: %v", len(remaining), remaining)
	}

	// GetStats must report 0 memory + 0 disk entries.
	stats := srv.cacheManager.GetStats()
	if stats.MemoryEntries != 0 {
		t.Errorf("post-flush: MemoryEntries = %d, want 0", stats.MemoryEntries)
	}
	if stats.DiskEntries != 0 {
		t.Errorf("post-flush: DiskEntries = %d, want 0", stats.DiskEntries)
	}

	// Result must carry flushed:true
	resultMap := extractResultMap(t, resp)
	if resultMap["flushed"] != true {
		t.Errorf("result[flushed] = %v, want true", resultMap["flushed"])
	}
	if resultMap["refreshed"] != true {
		t.Errorf("result[refreshed] = %v, want true", resultMap["refreshed"])
	}
}

// TestHandleRefreshSelectivePathUnchanged confirms that the default (flush:false)
// selective invalidation path still runs and returns expected fields.
func TestHandleRefreshSelectivePathUnchanged(t *testing.T) {
	srv, _ := newTestServer(t)

	args, _ := json.Marshal(map[string]interface{}{})
	resp := srv.handleRefresh(1, args)

	if resp.Error != nil {
		t.Fatalf("handleRefresh(default) returned error: %v", resp.Error)
	}

	resultMap := extractResultMap(t, resp)

	if resultMap["refreshed"] != true {
		t.Errorf("result[refreshed] = %v, want true", resultMap["refreshed"])
	}
	if resultMap["flushed"] != false {
		t.Errorf("result[flushed] = %v, want false", resultMap["flushed"])
	}
	if _, ok := resultMap["stale_keys_found"]; !ok {
		t.Error("result missing stale_keys_found on selective path")
	}
	if _, ok := resultMap["cache_entries_invalidated"]; !ok {
		t.Error("result missing cache_entries_invalidated on selective path")
	}
}

// sha256HexT computes the lowercase-hex sha256 of s, mirroring the cache layer's
// transport-verification digest so tests can build matching/mismatching indexes.
func sha256HexT(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestGetContentVerificationFailureMapsTo32001 covers the user-facing half of the
// hash-validation feature: when a downloaded file's sha256 never matches the
// index, the p2kb_get path must surface the distinct -32001 "temporarily
// unavailable" error carrying expected/actual digests — NOT a generic failure.
func TestGetContentVerificationFailureMapsTo32001(t *testing.T) {
	const correct = "real: yaml content\n"
	served := "TAMPERED: not the real content\n"
	srv, r := newTestServer(t)
	r.SetIndex(kbtest.Index{Files: map[string]kbtest.FileEntry{
		"p2kbVerifyMe": {Path: "verify/me.yaml", Mtime: 1700000000, SHA256: sha256HexT(correct)},
	}})
	r.Put("verify/me.yaml", served) // never matches the index sha256

	resp := srv.contentResponse(1, "p2kbVerifyMe", "")
	if resp.Error == nil {
		t.Fatal("expected an error for sha256 mismatch, got success")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("error code = %d, want -32001 (verification failure)", resp.Error.Code)
	}
	data, ok := resp.Error.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("error data is not a map: %T", resp.Error.Data)
	}
	if data["expected_sha256"] != sha256HexT(correct) {
		t.Errorf("expected_sha256 = %v, want %s", data["expected_sha256"], sha256HexT(correct))
	}
	if data["actual_sha256"] != sha256HexT(served) {
		t.Errorf("actual_sha256 = %v, want %s (hash of served bytes)", data["actual_sha256"], sha256HexT(served))
	}
}

// TestGetContentNetworkErrorMapsTo32000 is the distinctness half: a non-
// verification failure (here HTTP 404 on a file with no sha256, so no
// verification) must map to the generic -32000, NOT -32001. This guards the
// errors.As discrimination from collapsing the two error classes.
func TestGetContentNetworkErrorMapsTo32000(t *testing.T) {
	srv, r := newTestServer(t)
	r.SetIndex(kbtest.Index{Files: map[string]kbtest.FileEntry{
		// no sha256 -> legacy path, verification skipped; no body served -> 404
		"p2kbPlainFail": {Path: "plain/fail.yaml", Mtime: 1700000000},
	}})

	resp := srv.contentResponse(1, "p2kbPlainFail", "")
	if resp.Error == nil {
		t.Fatal("expected an error for HTTP 404, got success")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("error code = %d, want -32000 (generic, distinct from -32001)", resp.Error.Code)
	}
}

// TestHandleRefreshFlushClearsObexCache covers the flush + include_obex path:
// it must clear the OBEX disk cache, not just the content cache.
func TestHandleRefreshFlushClearsObexCache(t *testing.T) {
	srv, r := newTestServer(t)
	serveOBEXObject(r)

	// Read one object, so it is parsed and its body cached.
	if _, err := srv.obexManager.GetObject("2811"); err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if parsed, cached := srv.obexManager.GetCacheStats(); parsed != 1 || cached != 1 {
		t.Fatalf("pre-flush: %d parsed, %d cached; want 1, 1", parsed, cached)
	}

	args, _ := json.Marshal(map[string]interface{}{"flush": true, "include_obex": true})
	resp := srv.handleRefresh(1, args)
	if resp.Error != nil {
		t.Fatalf("handleRefresh(flush+obex) returned error: %v", resp.Error)
	}

	if parsed, cached := srv.obexManager.GetCacheStats(); parsed != 0 || cached != 0 {
		t.Errorf("post-flush: %d parsed, %d cached; want 0, 0", parsed, cached)
	}
	resultMap := extractResultMap(t, resp)
	if resultMap["obex_refreshed"] != true {
		t.Errorf("result[obex_refreshed] = %v, want true", resultMap["obex_refreshed"])
	}
	if resultMap["obex_cache_entries_cleared"] != float64(1) {
		t.Errorf("result[obex_cache_entries_cleared] = %v, want 1", resultMap["obex_cache_entries_cleared"])
	}
}

// obexObjectYAML is one OBEX object file, in the KB's shape.
const obexObjectYAML = `object_metadata:
  object_id: "2811"
  title: "Park transformation"
  author: "ManAtWork"
  urls:
    obex_page: "https://obex.parallax.com/obex/park-transformation/"
  technical_details:
    languages:
      - SPIN2
      - PASM2
    file_size: "16 B"
  functionality:
    category: "motors"
    description_short: "The Park or d/q-transformation"
    tags:
      - servo
      - motor
  metadata:
    quality_score: 5
    created_date: "2020-05-09 12:00:00"
`

// serveOBEXObject lists object 2811 in r's index and serves it.
func serveOBEXObject(r *kbtest.Remote) {
	r.AddFile("p2kbCommunity2811", "deliverables/ai/P2/community/obex/objects/2811.yaml", 1700000000, obexObjectYAML)
}

// TestOBEXGetResultShape pins p2kb_obex_get's result fields (captured from
// the 1.4 server against the live KB before OBEX moved onto the main index).
func TestOBEXGetResultShape(t *testing.T) {
	srv, r := newTestServer(t)
	serveOBEXObject(r)

	resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"2811"}`))
	if resp.Error != nil {
		t.Fatalf("handleOBEXGet: %v", resp.Error)
	}
	got := extractResultMap(t, resp)

	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"author", "category", "description", "download_instructions", "download_url",
		"languages", "metadata", "obex_page", "object_id", "tags", "title", "type"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("result fields = %v, want %v", keys, want)
	}
	if got["type"] != "obex_object" || got["object_id"] != "2811" || got["title"] != "Park transformation" {
		t.Errorf("result = %v", got)
	}
}

// extractResultMap pulls the JSON result map out of a successResponse for assertions.
func extractResultMap(t *testing.T, resp *MCPResponse) map[string]interface{} {
	t.Helper()
	outer, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("resp.Result is not a map[string]interface{}")
	}
	content, ok := outer["content"].([]map[string]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("resp.Result[content] is missing or empty")
	}
	text, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("content[0][text] is not a string")
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(text), &resultMap); err != nil {
		t.Fatalf("failed to parse result text as JSON: %v", err)
	}
	return resultMap
}
