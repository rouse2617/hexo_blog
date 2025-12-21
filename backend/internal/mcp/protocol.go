// Package mcp provides MCP (Model Context Protocol) client functionality.
package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Protocol implements JSON-RPC 2.0 protocol for MCP.
type Protocol struct{}

// NewProtocol creates a new Protocol instance.
func NewProtocol() *Protocol {
	return &Protocol{}
}

// CreateRequest creates a JSON-RPC 2.0 request.
func (p *Protocol) CreateRequest(method string, params interface{}) (*MCPRequest, error) {
	return &MCPRequest{
		JSONRPC: "2.0",
		ID:      uuid.New().String(),
		Method:  method,
		Params:  params,
	}, nil
}

// SerializeRequest serializes an MCP request to JSON bytes.
func (p *Protocol) SerializeRequest(req *MCPRequest) ([]byte, error) {
	if req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", req.JSONRPC)
	}
	if req.ID == "" {
		return nil, fmt.Errorf("request ID is required")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("request method is required")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}
	return data, nil
}

// DeserializeRequest deserializes JSON bytes to an MCP request.
func (p *Protocol) DeserializeRequest(data []byte) (*MCPRequest, error) {
	var req MCPRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to deserialize request: %w", err)
	}

	if req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", req.JSONRPC)
	}
	if req.ID == "" {
		return nil, fmt.Errorf("request ID is required")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("request method is required")
	}

	return &req, nil
}

// SerializeResponse serializes an MCP response to JSON bytes.
func (p *Protocol) SerializeResponse(resp *MCPResponse) ([]byte, error) {
	if resp.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", resp.JSONRPC)
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("response ID is required")
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize response: %w", err)
	}
	return data, nil
}

// DeserializeResponse deserializes JSON bytes to an MCP response.
func (p *Protocol) DeserializeResponse(data []byte) (*MCPResponse, error) {
	var resp MCPResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	if resp.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid JSON-RPC version: %s", resp.JSONRPC)
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("response ID is required")
	}

	return &resp, nil
}

// CreateListToolsRequest creates a request to list tools.
func (p *Protocol) CreateListToolsRequest() (*MCPRequest, error) {
	return p.CreateRequest("tools/list", ListToolsParams{})
}

// CreateCallToolRequest creates a request to call a tool.
func (p *Protocol) CreateCallToolRequest(toolName string, args map[string]interface{}) (*MCPRequest, error) {
	params := CallToolParams{
		Name:      toolName,
		Arguments: args,
	}
	return p.CreateRequest("tools/call", params)
}

// MCPRequest represents a JSON-RPC 2.0 request for MCP.
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// MCPResponse represents a JSON-RPC 2.0 response for MCP.
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in an MCP response.
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ListToolsParams represents parameters for listing tools.
type ListToolsParams struct{}

// ListToolsResult represents the result of listing tools.
type ListToolsResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPTool represents an MCP tool definition from the server.
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// CallToolParams represents parameters for calling a tool.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// CallToolResult represents the result of calling a tool.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block in a tool result.
type ContentBlock struct {
	Type string `json:"type"` // "text" | "image" | "resource"
	Text string `json:"text,omitempty"`
}
