package server

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	srv := New("1.0.0")
	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.version != "1.0.0" {
		t.Errorf("version = %q, want %q", srv.version, "1.0.0")
	}
	if srv.indexManager == nil {
		t.Error("indexManager is nil")
	}
	if srv.cacheManager == nil {
		t.Error("cacheManager is nil")
	}
}

func TestHandleInitialize(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := srv.handleRequest(req)
	if resp == nil {
		t.Fatal("handleInitialize returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("handleInitialize returned error: %v", resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("ID = %v, want 1", resp.ID)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// With no client-supplied protocolVersion, server returns its max.
	if result["protocolVersion"] != serverMaxVersion {
		t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], serverMaxVersion)
	}

	// Result must contain only the spec-allowed top-level keys.
	allowed := map[string]bool{"protocolVersion": true, "capabilities": true, "serverInfo": true, "instructions": true, "_meta": true}
	for k := range result {
		if !allowed[k] {
			t.Errorf("non-spec field in initialize result: %q", k)
		}
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("serverInfo is not a map")
	}
	if serverInfo["name"] != "p2kb-mcp" {
		t.Errorf("serverInfo.name = %v, want p2kb-mcp", serverInfo["name"])
	}
}

func TestHandleInitializeProtocolNegotiation(t *testing.T) {
	srv := New("1.0.0")

	cases := []struct {
		name        string
		clientVer   string
		wantVersion string
	}{
		{"supported 2024-11-05 echoed", "2024-11-05", "2024-11-05"},
		{"supported 2025-03-26 echoed", "2025-03-26", "2025-03-26"},
		{"supported 2025-06-18 echoed", "2025-06-18", "2025-06-18"},
		{"future version falls back to server max", "2099-01-01", serverMaxVersion},
		{"unknown version falls back to server max", "garbage", serverMaxVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, _ := json.Marshal(map[string]interface{}{"protocolVersion": tc.clientVer})
			req := &MCPRequest{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: params}
			resp := srv.handleRequest(req)
			if resp == nil || resp.Error != nil {
				t.Fatalf("unexpected error: %+v", resp)
			}
			result := resp.Result.(map[string]interface{})
			if result["protocolVersion"] != tc.wantVersion {
				t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], tc.wantVersion)
			}
		})
	}
}

func TestHandleNotificationsAreFiltered(t *testing.T) {
	srv := New("1.0.0")

	// Any method without an id is a notification and must not produce a response,
	// including notifications the server doesn't recognize.
	cases := []string{
		"notifications/initialized",
		"notifications/cancelled",
		"notifications/progress",
		"some/unknown/notification",
	}
	for _, method := range cases {
		t.Run(method, func(t *testing.T) {
			req := &MCPRequest{JSONRPC: "2.0", ID: nil, Method: method}
			if resp := srv.handleRequest(req); resp != nil {
				t.Errorf("notification %q produced a response: %+v", method, resp)
			}
		})
	}
}

func TestHandlePing(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "ping",
	}

	resp := srv.handleRequest(req)
	if resp == nil {
		t.Fatal("handlePing returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("handlePing returned error: %v", resp.Error)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %v, want 42", resp.ID)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	resp := srv.handleRequest(req)
	if resp == nil {
		t.Fatal("handleRequest returned nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestHandleToolsList(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := srv.handleRequest(req)
	if resp == nil {
		t.Fatal("handleToolsList returned nil")
	}
	if resp.Error != nil {
		t.Fatalf("handleToolsList returned error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	tools, ok := result["tools"].([]Tool)
	if !ok {
		t.Fatal("tools is not a []Tool")
	}

	// Check we have all 7 tools
	if len(tools) != 7 {
		t.Errorf("got %d tools, want 7", len(tools))
	}

	// Check for specific tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"p2kb_get", "p2kb_find", "p2kb_obex_get", "p2kb_obex_find",
		"p2kb_obex_download", "p2kb_version", "p2kb_refresh",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestHandleNotificationsInitialized(t *testing.T) {
	srv := New("1.0.0")
	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      nil,
		Method:  "notifications/initialized",
	}

	resp := srv.handleRequest(req)
	if resp != nil {
		t.Error("notifications/initialized should return nil")
	}
}

func TestMCPRequestUnmarshal(t *testing.T) {
	jsonStr := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	var req MCPRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", req.JSONRPC)
	}
	if req.Method != "tools/list" {
		t.Errorf("Method = %q, want tools/list", req.Method)
	}
}

func TestMCPResponseMarshal(t *testing.T) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"test": "value"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MCPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", decoded.JSONRPC)
	}
}

func TestMCPErrorResponse(t *testing.T) {
	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &MCPError{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    "test data",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MCPResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Error is nil")
	}
	if decoded.Error.Code != -32600 {
		t.Errorf("Error.Code = %d, want -32600", decoded.Error.Code)
	}
}
