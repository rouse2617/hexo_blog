// Package models provides data models for the OpsGenius Backend.
package models

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

// Tool represents a tool with additional metadata for internal use.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Readonly    bool                   `json:"readonly"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
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

// MCPServerConfig represents the configuration for an MCP server.
type MCPServerConfig struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Transport string            `json:"transport" yaml:"transport"` // "stdio" | "http"
	Command   string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args      []string          `json:"args,omitempty" yaml:"args,omitempty"`
	URL       string            `json:"url,omitempty" yaml:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Auth      *MCPAuthConfig    `json:"auth,omitempty" yaml:"auth,omitempty"`
}

// MCPAuthConfig represents authentication configuration for an MCP server.
type MCPAuthConfig struct {
	Type  string `json:"type" yaml:"type"`   // "bearer" | "basic"
	Token string `json:"token" yaml:"token"` // For bearer auth
}

// ConnectionStatus represents the connection status of an MCP server.
type ConnectionStatus string

const (
	ConnectionStatusOnline  ConnectionStatus = "online"
	ConnectionStatusOffline ConnectionStatus = "offline"
	ConnectionStatusError   ConnectionStatus = "error"
)

// ServerInfo represents information about an MCP server.
type ServerInfo struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Status ConnectionStatus `json:"status"`
	Tools  []Tool           `json:"tools,omitempty"`
}
