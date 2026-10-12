package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
	"github.com/ironsheep/p2kb-mcp/internal/obex"
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
			"category": "Protocol",
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

// findOBEX runs p2kb_obex_find with the given arguments and decodes the result.
func findOBEX(t *testing.T, srv *Server, args string) map[string]interface{} {
	t.Helper()
	resp := srv.handleOBEXFind(1, json.RawMessage(args))
	if resp.Error != nil {
		t.Fatalf("handleOBEXFind(%s): %v", args, resp.Error)
	}
	return extractResultMap(t, resp)
}

// findIDs returns the sorted object_id of each object in a find result.
func findIDs(t *testing.T, res map[string]interface{}) []string {
	t.Helper()
	objs, _ := res["objects"].([]interface{})
	ids := make([]string, 0, len(objs))
	for _, o := range objs {
		entry, _ := o.(map[string]interface{})
		id, _ := entry["object_id"].(string)
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// sortedKeys returns the keys of m, sorted.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestOBEXFindOverviewCountsEachCategoryElement: a list category contributes one
// count per element; a string category contributes its one name.
func TestOBEXFindOverviewCountsEachCategoryElement(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)   // [Math, Motor Control, Protocol, Snippet]
	r.AddOBEX(kbtest.OBEXStringCategoryID) // Protocol
	r.AddOBEX(kbtest.OBEXNoDownloadID)     // [Display, Other, Protocol]

	got := findOBEX(t, srv, `{}`)
	if got["type"] != "overview" {
		t.Fatalf("type = %v, want overview", got["type"])
	}
	want := map[string]interface{}{
		"Math": float64(1), "Motor Control": float64(1), "Protocol": float64(3),
		"Snippet": float64(1), "Display": float64(1), "Other": float64(1),
	}
	if !reflect.DeepEqual(got["categories"], want) {
		t.Errorf("categories = %v, want %v", got["categories"], want)
	}
}

// TestOBEXFindCategoryMatchesWhenAnElementIsTheName: category Protocol returns
// every object whose category is or includes Protocol. The string-category
// object 2842 alone does not prove this; 2811 and 4581 hold Protocol as one
// element of a list, so a whole-string comparison ("Math, Motor Control,
// Protocol, Snippet" equal to "Protocol") would drop them and fail this test.
func TestOBEXFindCategoryMatchesWhenAnElementIsTheName(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.AddOBEX(kbtest.OBEXStringCategoryID)
	r.AddOBEX(kbtest.OBEXNoDownloadID)

	wantProtocol := []string{"2811", "2842", "4581"}
	for _, args := range []string{`{"category":"Protocol"}`, `{"category":"protocol"}`} {
		got := findOBEX(t, srv, args)
		if got["type"] != "objects" || got["count"] != float64(3) {
			t.Errorf("%s: type = %v, count = %v, want objects, 3", args, got["type"], got["count"])
		}
		if ids := findIDs(t, got); !reflect.DeepEqual(ids, wantProtocol) {
			t.Errorf("%s: object ids = %v, want %v", args, ids, wantProtocol)
		}
	}

	// An element that only the list object carries.
	if ids := findIDs(t, findOBEX(t, srv, `{"category":"Motor Control"}`)); !reflect.DeepEqual(ids, []string{"2811"}) {
		t.Errorf("Motor Control object ids = %v, want [2811]", ids)
	}

	// A category no object has: an empty result, not an error.
	got := findOBEX(t, srv, `{"category":"Nonexistent"}`)
	if got["type"] != "objects" || got["count"] != float64(0) {
		t.Errorf("Nonexistent: type = %v, count = %v, want objects, 0", got["type"], got["count"])
	}
	if objs, ok := got["objects"].([]interface{}); !ok || len(objs) != 0 {
		t.Errorf("Nonexistent objects = %v, want an empty list", got["objects"])
	}
}

// TestOBEXFindTermResultShape: term with limit keeps the 1.5.0 key sets exactly.
func TestOBEXFindTermResultShape(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXStringCategoryID) // Full Duplex Serial
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.AddOBEX(kbtest.OBEXNoDownloadID)

	got := findOBEX(t, srv, `{"term":"serial","limit":2}`)
	wantTop := []string{"count", "objects", "term", "type"}
	if keys := sortedKeys(got); !reflect.DeepEqual(keys, wantTop) {
		t.Errorf("top-level keys = %v, want %v", keys, wantTop)
	}
	if got["term"] != "serial" || got["type"] != "objects" {
		t.Errorf("term = %v, type = %v, want serial, objects", got["term"], got["type"])
	}
	objs, _ := got["objects"].([]interface{})
	if len(objs) == 0 || len(objs) > 2 || got["count"] != float64(len(objs)) {
		t.Fatalf("objects = %d entries, count = %v, want 1 to 2 entries with a matching count", len(objs), got["count"])
	}
	wantObject := []string{"author", "category", "description", "match_type", "object_id", "title"}
	for _, o := range objs {
		entry, _ := o.(map[string]interface{})
		if keys := sortedKeys(entry); !reflect.DeepEqual(keys, wantObject) {
			t.Errorf("object keys = %v, want %v", keys, wantObject)
		}
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

// downloadResponse calls p2kb_obex_download and returns the response.
func downloadResponse(t *testing.T, srv *Server, args map[string]interface{}) *MCPResponse {
	t.Helper()
	params, _ := json.Marshal(map[string]interface{}{"name": "p2kb_obex_download", "arguments": args})
	return srv.handleRequest(&MCPRequest{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: params})
}

func TestHandleOBEXDownloadNoDownload(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXNoDownloadID)
	r.Put("wp-admin/admin-ajax.php", "must not be fetched")

	resp := downloadResponse(t, srv, map[string]interface{}{"object_id": "OB4581"})
	if resp.Error != nil {
		t.Fatalf("no download must be a success result, got error %+v", resp.Error)
	}
	got := extractResultMap(t, resp)
	if got["type"] != "no_download" {
		t.Errorf("type = %v, want no_download", got["type"])
	}
	if got["object_id"] != "4581" || got["title"] != "ISP P2 RPi IoT Gateway" ||
		got["obex_page"] != "https://obex.parallax.com/obex/isp-p2-rpi-iot-gateway/" {
		t.Errorf("result = %v", got)
	}
	want := "OBEX offers no download for object 4581; see its page https://obex.parallax.com/obex/isp-p2-rpi-iot-gateway/ — the description says where the code is."
	if got["message"] != want {
		t.Errorf("message = %q, want %q", got["message"], want)
	}
	if n := r.Hits("wp-admin/admin-ajax.php"); n != 0 {
		t.Errorf("download host saw %d requests, want 0", n)
	}
	if _, err := os.Stat("OBEX"); !os.IsNotExist(err) {
		t.Errorf("OBEX directory exists after no-download (stat err %v)", err)
	}
}

func TestHandleOBEXDownloadFromDownloadDirect(t *testing.T) {
	srv, r := newTestServer(t)
	body := strings.Replace(kbtest.OBEXObject(kbtest.OBEXListCategoryID),
		"https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB2811",
		r.URL()+"/dl/park.zip", 1)
	r.AddFile("p2kbCommunity2811", kbtest.OBEXObjectsPath+"/2811.yaml", 1700000000, body)
	zbuf := new(bytes.Buffer)
	zw := zip.NewWriter(zbuf)
	w, _ := zw.Create("park.spin2")
	_, _ = w.Write([]byte("' park"))
	_ = zw.Close()
	r.Put("dl/park.zip", zbuf.String())

	resp := downloadResponse(t, srv, map[string]interface{}{"object_id": "2811"})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	got := extractResultMap(t, resp)
	if got["type"] != "download_complete" || got["file_count"] != float64(1) {
		t.Errorf("result = %v", got)
	}
	if n := r.Hits("dl/park.zip"); n != 1 {
		t.Errorf("download host saw %d requests, want 1", n)
	}
}

func TestHandleOBEXDownloadFailuresStayErrors(t *testing.T) {
	srv, r := newTestServer(t)
	body := strings.Replace(kbtest.OBEXObject(kbtest.OBEXListCategoryID),
		"https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB2811",
		r.URL()+"/dl/missing.zip", 1)
	r.AddFile("p2kbCommunity2811", kbtest.OBEXObjectsPath+"/2811.yaml", 1700000000, body)

	for _, id := range []string{"9999", "2811"} { // unknown id; host answers 404
		resp := downloadResponse(t, srv, map[string]interface{}{"object_id": id})
		if resp.Error == nil || resp.Error.Code != -32000 {
			t.Errorf("id %s: want -32000 error, got %+v", id, resp)
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
	r.AddOBEX(kbtest.OBEXListCategoryID)

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

// TestOBEXGetResultShape pins p2kb_obex_get's result fields (captured from
// the 1.4 server against the live KB before OBEX moved onto the main index).
func TestOBEXGetResultShape(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.AddOBEX(kbtest.OBEXStringCategoryID)
	r.AddOBEX(kbtest.OBEXNoDownloadID)

	get := func(query string) map[string]interface{} {
		t.Helper()
		resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"`+query+`"}`))
		if resp.Error != nil {
			t.Fatalf("handleOBEXGet(%s): %v", query, resp.Error)
		}
		return extractResultMap(t, resp)
	}
	keysOf := func(m map[string]interface{}) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}

	// With a download (2842): the whole key set, download pair included.
	got := get("2842")
	want := []string{"author", "category", "description", "download_instructions", "download_url", "full_metadata",
		"languages", "metadata", "obex_page", "object_id", "tags", "title", "type"}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("2842 result fields = %v, want %v", keys, want)
	}
	if got["type"] != "obex_object" || got["object_id"] != "2842" || got["title"] != "Full Duplex Serial" {
		t.Errorf("2842 result = %v", got)
	}
	if !reflect.DeepEqual(got["metadata"], map[string]interface{}{"created_date": "2022-01-04"}) {
		t.Errorf("2842 metadata = %v, want exactly created_date", got["metadata"])
	}
	if wantURL := "https://obex.parallax.com/wp-admin/admin-ajax.php?action=download_obex_zip&popcorn=salty&obuid=OB2842"; got["download_url"] != wantURL {
		t.Errorf("2842 download_url = %v, want the fixture's download_direct", got["download_url"])
	}
	full, _ := got["full_metadata"].(map[string]interface{})
	wantFull := []string{"author", "functionality", "metadata", "object_id", "technical_details", "title", "urls"}
	if keys := keysOf(full); !reflect.DeepEqual(keys, wantFull) {
		t.Errorf("2842 full_metadata keys = %v, want %v", keys, wantFull)
	}
	// No quality, version or file_size anywhere in the result, at any depth.
	encoded, _ := json.Marshal(got)
	for _, banned := range []string{`"quality"`, `"version"`, `"file_size"`} {
		if strings.Contains(string(encoded), banned) {
			t.Errorf("2842 result contains %s", banned)
		}
	}

	// No download_direct (4581): neither download_url nor download_instructions.
	got = get("4581")
	want = []string{"author", "category", "description", "full_metadata", "languages", "metadata", "obex_page",
		"object_id", "tags", "title", "type"}
	if keys := keysOf(got); !reflect.DeepEqual(keys, want) {
		t.Errorf("4581 result fields = %v, want %v", keys, want)
	}
	for _, absent := range []string{"download_url", "download_instructions"} {
		if v, present := got[absent]; present {
			t.Errorf("4581 result has %s = %v, want the key absent", absent, v)
		}
	}

	// 2811's category is a list in the KB; the result keeps that form.
	got = get("2811")
	if cats, ok := got["category"].([]interface{}); !ok || len(cats) != 4 || cats[0] != "Math" {
		t.Errorf("2811 category = %#v, want the 4-element list", got["category"])
	}
	if !reflect.DeepEqual(got["metadata"], map[string]interface{}{"created_date": "2020-05-09"}) {
		t.Errorf("2811 metadata = %v, want created_date only", got["metadata"])
	}
}

// TestOBEXGetOmitsMetadataWithoutCreatedDate: no created_date, no metadata key.
func TestOBEXGetOmitsMetadataWithoutCreatedDate(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddFile("p2kbCommunity9002", obex.OBEXPath+"/9002.yaml", 1700000000,
		"object_metadata:\n  object_id: \"9002\"\n  title: Dateless\n  metadata:\n    source: x\n")

	resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"9002"}`))
	if resp.Error != nil {
		t.Fatalf("handleOBEXGet: %v", resp.Error)
	}
	got := extractResultMap(t, resp)
	if v, present := got["metadata"]; present {
		t.Errorf("metadata = %v, want the key omitted", v)
	}
	if _, present := got["full_metadata"]; !present {
		t.Error("full_metadata missing")
	}
}

// TestOBEXGetUnparseableObjectIsAnErrorNamingIt: a body that does not parse is
// an error naming the object, not object_not_found.
func TestOBEXGetUnparseableObjectIsAnErrorNamingIt(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXMalformedID)

	resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"`+kbtest.OBEXMalformedID+`"}`))
	if resp.Error == nil {
		t.Fatalf("handleOBEXGet result = %v, want an error", resp.Result)
	}
	if !strings.Contains(resp.Error.Data.(string), kbtest.OBEXMalformedID) {
		t.Errorf("error = %+v, want it to name the object", resp.Error)
	}
}

// TestOBEXGetIndexUnavailableIsAnError: an unreachable index is reported as
// an error, never as "object not found".
func TestOBEXGetIndexUnavailableIsAnError(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.Close()

	resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"2811"}`))
	if resp.Error == nil {
		t.Errorf("handleOBEXGet result = %v, want an error", resp.Result)
	}
}

// TestVersionUnreadableCountUnknownIsNotZero: with no readable index the
// unreadable count is unknown, so p2kb_version reports null and the error.
func TestVersionUnreadableCountUnknownIsNotZero(t *testing.T) {
	srv, r := newTestServer(t)
	r.Close()

	obexSection, ok := versionResult(t, srv)["obex"].(map[string]interface{})
	if !ok {
		t.Fatal("p2kb_version has no obex section")
	}
	if got, present := obexSection["unreadable_objects"]; !present || got != nil {
		t.Errorf("unreadable_objects = %v (present %v), want null", got, present)
	}
	if obexSection["unreadable_objects_error"] == nil {
		t.Error("unreadable_objects_error missing when the index cannot be read")
	}
}

// obexResults runs every OBEX handler that can list objects, returning each
// decoded result by name. Every call must succeed.
func obexResults(t *testing.T, srv *Server) map[string]map[string]interface{} {
	t.Helper()
	calls := map[string]*MCPResponse{
		"overview":    srv.handleOBEXFind(1, json.RawMessage(`{}`)),
		"search":      srv.handleOBEXFind(1, json.RawMessage(`{"term":"a"}`)),
		"browse":      srv.handleOBEXFind(1, json.RawMessage(`{"category":"protocol"}`)),
		"author":      srv.handleOBEXFind(1, json.RawMessage(`{"author":"a"}`)),
		"suggestions": srv.handleOBEXGet(1, json.RawMessage(`{"query":"a"}`)),
		"no_matches":  srv.handleOBEXGet(1, json.RawMessage(`{"query":"zzzqqq"}`)),
	}
	out := make(map[string]map[string]interface{}, len(calls))
	for name, resp := range calls {
		if resp.Error != nil {
			t.Fatalf("%s: %v", name, resp.Error)
		}
		out[name] = extractResultMap(t, resp)
	}
	if out["suggestions"]["type"] != "suggestions" || out["no_matches"]["type"] != "no_matches" {
		t.Fatalf("obex_get types = %v, %v; want suggestions, no_matches", out["suggestions"]["type"], out["no_matches"]["type"])
	}
	return out
}

// TestOBEXUnreadableObjectsAreReported: a healthy KB has no unreadable_objects
// key anywhere and p2kb_version counts 0; with the malformed 9990 among good
// objects every result lists it, the good objects still come back, and
// p2kb_version counts 1.
func TestOBEXUnreadableObjectsAreReported(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.AddOBEX(kbtest.OBEXStringCategoryID)

	healthy := obexResults(t, srv)
	for name, res := range healthy {
		if _, present := res["unreadable_objects"]; present {
			t.Errorf("healthy %s result has an unreadable_objects key: %v", name, res)
		}
	}
	// O5: suggestions stay a short list, with no metadata or full_metadata.
	suggestions, _ := healthy["suggestions"]["suggestions"].([]interface{})
	if len(suggestions) == 0 {
		t.Fatalf("suggestions = %v, want entries", healthy["suggestions"])
	}
	for _, s := range suggestions {
		entry, _ := s.(map[string]interface{})
		for _, k := range []string{"metadata", "full_metadata"} {
			if _, present := entry[k]; present {
				t.Errorf("suggestion %v carries %s", entry["object_id"], k)
			}
		}
	}
	obexSection := func() map[string]interface{} {
		section, _ := versionResult(t, srv)["obex"].(map[string]interface{})
		return section
	}
	if got := obexSection()["unreadable_objects"]; got != float64(0) {
		t.Errorf("healthy p2kb_version unreadable_objects = %v, want 0", got)
	}

	r.AddOBEX(kbtest.OBEXMalformedID)
	if err := srv.indexManager.Refresh(); err != nil { // the healthy calls above loaded the index
		t.Fatal(err)
	}

	results := obexResults(t, srv)
	for name, res := range results {
		bad, ok := res["unreadable_objects"].([]interface{})
		if !ok || len(bad) != 1 {
			t.Errorf("%s: unreadable_objects = %v, want one entry for 9990", name, res["unreadable_objects"])
			continue
		}
		entry, _ := bad[0].(map[string]interface{})
		if entry["object_id"] != "9990" || entry["error"] == nil || entry["error"] == "" {
			t.Errorf("%s: unreadable entry = %v, want object_id 9990 and an error", name, entry)
		}
	}
	if got := results["search"]["count"]; got != float64(2) {
		t.Errorf("search count = %v, want the 2 good objects", got)
	}
	if cats, _ := results["overview"]["categories"].(map[string]interface{}); cats["Protocol"] != float64(2) {
		t.Errorf("overview categories = %v, want 9990 excluded (Protocol 2)", cats)
	}
	if got := obexSection()["unreadable_objects"]; got != float64(1) {
		t.Errorf("p2kb_version unreadable_objects = %v, want 1", got)
	}
}

// TestOBEXGetUnparseableObjectIsAnError: p2kb_obex_get on the malformed object
// returns an error naming it, not the object_not_found result.
func TestOBEXGetUnparseableObjectIsAnError(t *testing.T) {
	srv, r := newTestServer(t)
	r.AddOBEX(kbtest.OBEXListCategoryID)
	r.AddOBEX(kbtest.OBEXMalformedID)

	resp := srv.handleOBEXGet(1, json.RawMessage(`{"query":"9990"}`))
	if resp.Error == nil {
		t.Fatalf("handleOBEXGet(9990) = %v, want an error", resp.Result)
	}
	if resp.Result != nil {
		t.Errorf("handleOBEXGet(9990) has a result (object_not_found shape?): %v", resp.Result)
	}
	if text := fmt.Sprint(resp.Error.Message, " ", resp.Error.Data); !strings.Contains(text, "9990") {
		t.Errorf("error %q does not name 9990", text)
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
