package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: ops-genius-backend, Property 10: 工具列表获取
// Validates: Requirements 3.2
func TestProperty_ListToolsAfterConnection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("connected client should be able to list tools", prop.ForAll(
		func(tools []MCPTool) bool {
			// Create a mock HTTP server that returns tools
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					return
				}

				// Parse request
				var req MCPRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return tools list
				resp := MCPResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ListToolsResult{
						Tools: tools,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Create client and connect
			client := NewClient(zap.NewNop())
			err := client.Connect(ServerConfig{
				ID:        "test-server",
				Name:      "Test Server",
				Transport: "http",
				URL:       server.URL,
			})
			if err != nil {
				return false
			}
			defer client.Disconnect()

			// List tools
			listedTools, err := client.ListTools()
			if err != nil {
				return false
			}

			// Verify tool count matches
			if len(listedTools) != len(tools) {
				return false
			}

			// Verify each tool
			for i, tool := range tools {
				if listedTools[i].Name != tool.Name {
					return false
				}
				if listedTools[i].Description != tool.Description {
					return false
				}
			}

			return true
		},
		genMCPToolList(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 14: 工具调用结果返回
// Validates: Requirements 4.2
func TestProperty_ToolCallReturnsResult(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tool call should return result on success", prop.ForAll(
		func(toolName string, args map[string]interface{}, resultData string) bool {
			// Create a mock HTTP server that returns tool result
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					return
				}

				// Parse request
				var req MCPRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return success result
				resp := MCPResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: CallToolResult{
						Content: []ContentBlock{
							{
								Type: "text",
								Text: resultData,
							},
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Create client and connect
			client := NewClient(zap.NewNop())
			err := client.Connect(ServerConfig{
				ID:        "test-server",
				Name:      "Test Server",
				Transport: "http",
				URL:       server.URL,
			})
			if err != nil {
				return false
			}
			defer client.Disconnect()

			// Call tool
			ctx := context.Background()
			result, err := client.CallTool(ctx, toolName, args)
			if err != nil {
				return false
			}

			// Verify result
			if !result.Success {
				return false
			}
			if result.Data == nil {
				return false
			}

			return true
		},
		genToolName(),
		genToolArguments(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 14: 工具调用结果返回 (Error case)
// Validates: Requirements 4.2
func TestProperty_ToolCallReturnsError(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tool call should return error on failure", prop.ForAll(
		func(toolName string, args map[string]interface{}, errorMsg string) bool {
			// Skip empty error messages
			if errorMsg == "" {
				return true
			}

			// Create a mock HTTP server that returns error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					return
				}

				// Parse request
				var req MCPRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return error result
				resp := MCPResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &MCPError{
						Code:    -32600,
						Message: errorMsg,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Create client and connect
			client := NewClient(zap.NewNop())
			err := client.Connect(ServerConfig{
				ID:        "test-server",
				Name:      "Test Server",
				Transport: "http",
				URL:       server.URL,
			})
			if err != nil {
				return false
			}
			defer client.Disconnect()

			// Call tool
			ctx := context.Background()
			result, err := client.CallTool(ctx, toolName, args)
			if err != nil {
				return false
			}

			// Verify error result
			if result.Success {
				return false
			}
			if result.Error == "" {
				return false
			}
			if result.Error != errorMsg {
				return false
			}

			return true
		},
		genToolName(),
		genToolArguments(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 10: 工具列表获取 (Empty list)
// Validates: Requirements 3.2
func TestProperty_ListToolsEmptyList(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("client should handle empty tool list", prop.ForAll(
		func() bool {
			// Create a mock HTTP server that returns empty tools
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.WriteHeader(http.StatusOK)
					return
				}

				// Parse request
				var req MCPRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Return empty tools list
				resp := MCPResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: ListToolsResult{
						Tools: []MCPTool{},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			// Create client and connect
			client := NewClient(zap.NewNop())
			err := client.Connect(ServerConfig{
				ID:        "test-server",
				Name:      "Test Server",
				Transport: "http",
				URL:       server.URL,
			})
			if err != nil {
				return false
			}
			defer client.Disconnect()

			// List tools
			listedTools, err := client.ListTools()
			if err != nil {
				return false
			}

			// Verify empty list
			if len(listedTools) != 0 {
				return false
			}

			return true
		},
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generator for MCP tool list
func genMCPToolList() gopter.Gen {
	return gen.SliceOfN(5, genMCPTool())
}

// Generator for MCP tool
func genMCPTool() gopter.Gen {
	return gopter.CombineGens(
		genToolName(),
		gen.AlphaString(),
	).Map(func(values []interface{}) MCPTool {
		return MCPTool{
			Name:        values[0].(string),
			Description: values[1].(string),
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}
	})
}
