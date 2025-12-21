package mcp

import (
	"encoding/json"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: ops-genius-backend, Property 13: 工具调用请求格式正确性
// Validates: Requirements 4.1
func TestProperty_ToolCallRequestFormat(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("tool call request should have correct JSON-RPC 2.0 format", prop.ForAll(
		func(toolName string, args map[string]interface{}) bool {
			protocol := NewProtocol()

			// Create tool call request
			req, err := protocol.CreateCallToolRequest(toolName, args)
			if err != nil {
				return false
			}

			// Verify JSON-RPC 2.0 format
			if req.JSONRPC != "2.0" {
				return false
			}

			// Verify ID is not empty
			if req.ID == "" {
				return false
			}

			// Verify method is correct
			if req.Method != "tools/call" {
				return false
			}

			// Verify params structure
			if req.Params == nil {
				return false
			}

			// Serialize and deserialize to verify format
			data, err := protocol.SerializeRequest(req)
			if err != nil {
				return false
			}

			// Verify it's valid JSON
			var jsonCheck map[string]interface{}
			if err := json.Unmarshal(data, &jsonCheck); err != nil {
				return false
			}

			// Verify required fields exist
			if jsonCheck["jsonrpc"] != "2.0" {
				return false
			}
			if jsonCheck["id"] == nil || jsonCheck["id"] == "" {
				return false
			}
			if jsonCheck["method"] != "tools/call" {
				return false
			}
			if jsonCheck["params"] == nil {
				return false
			}

			// Deserialize back and verify
			deserializedReq, err := protocol.DeserializeRequest(data)
			if err != nil {
				return false
			}

			if deserializedReq.JSONRPC != "2.0" {
				return false
			}
			if deserializedReq.Method != "tools/call" {
				return false
			}

			return true
		},
		genToolName(),
		genToolArguments(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 13: 工具调用请求格式正确性 (Round-trip)
// Validates: Requirements 4.1
func TestProperty_RequestSerializationRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("request should round-trip through serialization", prop.ForAll(
		func(method string, toolName string, args map[string]interface{}) bool {
			protocol := NewProtocol()

			// Create request
			var req *MCPRequest
			var err error
			if method == "tools/call" {
				req, err = protocol.CreateCallToolRequest(toolName, args)
			} else {
				req, err = protocol.CreateListToolsRequest()
			}
			if err != nil {
				return false
			}

			// Serialize
			data, err := protocol.SerializeRequest(req)
			if err != nil {
				return false
			}

			// Deserialize
			deserializedReq, err := protocol.DeserializeRequest(data)
			if err != nil {
				return false
			}

			// Verify core fields match
			if deserializedReq.JSONRPC != req.JSONRPC {
				return false
			}
			if deserializedReq.ID != req.ID {
				return false
			}
			if deserializedReq.Method != req.Method {
				return false
			}

			return true
		},
		gen.OneConstOf("tools/call", "tools/list"),
		genToolName(),
		genToolArguments(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 13: 工具调用请求格式正确性 (Response)
// Validates: Requirements 4.1
func TestProperty_ResponseSerializationRoundTrip(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("response should round-trip through serialization", prop.ForAll(
		func(hasError bool, resultData string, errorMsg string) bool {
			protocol := NewProtocol()

			// Create response
			resp := &MCPResponse{
				JSONRPC: "2.0",
				ID:      "test-id",
			}

			if hasError {
				resp.Error = &MCPError{
					Code:    -32600,
					Message: errorMsg,
				}
			} else {
				resp.Result = map[string]interface{}{
					"data": resultData,
				}
			}

			// Serialize
			data, err := protocol.SerializeResponse(resp)
			if err != nil {
				return false
			}

			// Deserialize
			deserializedResp, err := protocol.DeserializeResponse(data)
			if err != nil {
				return false
			}

			// Verify core fields match
			if deserializedResp.JSONRPC != resp.JSONRPC {
				return false
			}
			if deserializedResp.ID != resp.ID {
				return false
			}

			// Verify error or result
			if hasError {
				if deserializedResp.Error == nil {
					return false
				}
				if deserializedResp.Error.Message != errorMsg {
					return false
				}
			} else {
				if deserializedResp.Result == nil {
					return false
				}
			}

			return true
		},
		gen.Bool(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Generator for tool names
func genToolName() gopter.Gen {
	return gen.OneConstOf(
		"get_logs",
		"get_metrics",
		"restart_service",
		"get_pods",
		"scale_deployment",
	)
}

// Generator for tool arguments
func genToolArguments() gopter.Gen {
	return gen.SliceOf(gen.AlphaString()).Map(func(keys []string) map[string]interface{} {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key != "" {
				result[key] = "value"
			}
		}
		return result
	})
}
