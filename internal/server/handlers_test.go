package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ironsheep/p2kb-mcp/internal/index"
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

// Test p2kb_obex_download

func TestHandleOBEXDownloadMissingObjectID(t *testing.T) {
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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
	srv := New("1.0.0")
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

// makeMinimalGzippedIndex builds a minimal valid p2kb index gzip payload for tests.
func makeMinimalGzippedIndex(t *testing.T) []byte {
	t.Helper()
	idx := map[string]interface{}{
		"system": map[string]interface{}{
			"version":           "test-1.0",
			"generated":         "2024-01-01T00:00:00Z",
			"total_entries":     0,
			"total_categories":  0,
			"total_aliases":     0,
		},
		"categories": map[string]interface{}{},
		"files":      map[string]interface{}{},
		"aliases":    map[string]interface{}{},
	}
	raw, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// newServerWithLocalIndex creates a server whose index manager points at a local
// httptest server (no live network required) and whose cache dir is isolated in
// a temp directory.  It returns the server and a cleanup function.
func newServerWithLocalIndex(t *testing.T) (*Server, func()) {
	t.Helper()

	// Build a tiny httptest server that serves the gzipped index payload.
	payload := makeMinimalGzippedIndex(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))

	// Redirect the package-level IndexURL var to our local server.
	origURL := index.IndexURL
	index.IndexURL = ts.URL

	// Redirect cache to an isolated temp dir.
	tmpDir := t.TempDir()
	if err := os.Setenv("P2KB_CACHE_DIR", tmpDir); err != nil {
		t.Fatalf("setenv P2KB_CACHE_DIR: %v", err)
	}

	srv := New("1.0.0")

	cleanup := func() {
		ts.Close()
		index.IndexURL = origURL
		os.Unsetenv("P2KB_CACHE_DIR")
	}
	return srv, cleanup
}

// seedDiskCache writes fake YAML files into the server's cache dir so
// GetCachedKeys() reports them.  It returns the cache dir path and the keys
// written.
func seedDiskCache(t *testing.T) (cacheDir string, keys []string) {
	t.Helper()
	cacheDir = os.Getenv("P2KB_CACHE_DIR")
	if cacheDir == "" {
		t.Fatal("P2KB_CACHE_DIR not set; call newServerWithLocalIndex first")
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
	srv, cleanup := newServerWithLocalIndex(t)
	defer cleanup()

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
	srv, cleanup := newServerWithLocalIndex(t)
	defer cleanup()

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
