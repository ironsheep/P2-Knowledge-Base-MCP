// Package server implements the MCP protocol server for P2KB.
package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ironsheep/p2kb-mcp/internal/cache"
	"github.com/ironsheep/p2kb-mcp/internal/index"
)

// Server handles MCP protocol communication over stdio.
type Server struct {
	version      string
	indexManager *index.Manager
	cacheManager *cache.Manager
}

// MCPRequest represents an incoming JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an outgoing JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC 2.0 error object.
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// New creates and initializes a new MCP server instance.
func New(version string) *Server {
	return &Server{
		version:      version,
		indexManager: index.NewManager(),
		cacheManager: cache.NewManager(),
	}
}

// Run starts the MCP server's main loop, processing requests from stdin.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large requests
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req MCPRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			continue
		}

		resp := s.handleRequest(&req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				log.Printf("Failed to encode response: %v", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest routes JSON-RPC requests to the appropriate handler method.
func (s *Server) handleRequest(req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// handleInitialize responds to the MCP initialize request.
func (s *Server) handleInitialize(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "p2kb-mcp",
				"version": s.version,
			},
		},
	}
}
