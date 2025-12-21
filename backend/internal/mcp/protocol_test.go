package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtocol_CreateRequest tests basic request creation.
func TestProtocol_CreateRequest(t *testing.T) {
	protocol := NewProtocol()

	req, err := protocol.CreateRequest("test/method", map[string]interface{}{"key": "value"})
	require.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.NotEmpty(t, req.ID)
	assert.Equal(t, "test/method", req.Method)
	assert.NotNil(t, req.Params)
}

// TestProtocol_SerializeRequest_InvalidVersion tests serialization with invalid JSON-RPC version.
func TestProtocol_SerializeRequest_InvalidVersion(t *testing.T) {
	protocol := NewProtocol()

	req := &MCPRequest{
		JSONRPC: "1.0",
		ID:      "test-id",
		Method:  "test/method",
		Params:  nil,
	}

	_, err := protocol.SerializeRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON-RPC version")
}

// TestProtocol_SerializeRequest_MissingID tests serialization with missing ID.
func TestProtocol_SerializeRequest_MissingID(t *testing.T) {
	protocol := NewProtocol()

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "",
		Method:  "test/method",
		Params:  nil,
	}

	_, err := protocol.SerializeRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request ID is required")
}

// TestProtocol_SerializeRequest_MissingMethod tests serialization with missing method.
func TestProtocol_SerializeRequest_MissingMethod(t *testing.T) {
	protocol := NewProtocol()

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "",
		Params:  nil,
	}

	_, err := protocol.SerializeRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request method is required")
}

// TestProtocol_DeserializeRequest_InvalidJSON tests deserialization with invalid JSON.
func TestProtocol_DeserializeRequest_InvalidJSON(t *testing.T) {
	protocol := NewProtocol()

	_, err := protocol.DeserializeRequest([]byte("invalid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize request")
}

// TestProtocol_DeserializeRequest_InvalidVersion tests deserialization with invalid version.
func TestProtocol_DeserializeRequest_InvalidVersion(t *testing.T) {
	protocol := NewProtocol()

	data := []byte(`{"jsonrpc":"1.0","id":"test-id","method":"test/method"}`)
	_, err := protocol.DeserializeRequest(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON-RPC version")
}

// TestProtocol_SerializeResponse_InvalidVersion tests response serialization with invalid version.
func TestProtocol_SerializeResponse_InvalidVersion(t *testing.T) {
	protocol := NewProtocol()

	resp := &MCPResponse{
		JSONRPC: "1.0",
		ID:      "test-id",
		Result:  "test",
	}

	_, err := protocol.SerializeResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON-RPC version")
}

// TestProtocol_SerializeResponse_MissingID tests response serialization with missing ID.
func TestProtocol_SerializeResponse_MissingID(t *testing.T) {
	protocol := NewProtocol()

	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      "",
		Result:  "test",
	}

	_, err := protocol.SerializeResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response ID is required")
}

// TestProtocol_DeserializeResponse_InvalidJSON tests response deserialization with invalid JSON.
func TestProtocol_DeserializeResponse_InvalidJSON(t *testing.T) {
	protocol := NewProtocol()

	_, err := protocol.DeserializeResponse([]byte("invalid json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deserialize response")
}

// TestProtocol_CreateListToolsRequest tests creating a list tools request.
func TestProtocol_CreateListToolsRequest(t *testing.T) {
	protocol := NewProtocol()

	req, err := protocol.CreateListToolsRequest()
	require.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.NotEmpty(t, req.ID)
	assert.Equal(t, "tools/list", req.Method)
}

// TestProtocol_CreateCallToolRequest tests creating a call tool request.
func TestProtocol_CreateCallToolRequest(t *testing.T) {
	protocol := NewProtocol()

	args := map[string]interface{}{
		"node":  "Node-A",
		"limit": 100,
	}

	req, err := protocol.CreateCallToolRequest("get_logs", args)
	require.NoError(t, err)
	assert.Equal(t, "2.0", req.JSONRPC)
	assert.NotEmpty(t, req.ID)
	assert.Equal(t, "tools/call", req.Method)
	assert.NotNil(t, req.Params)
}

// TestProtocol_RoundTrip tests full serialization round-trip.
func TestProtocol_RoundTrip(t *testing.T) {
	protocol := NewProtocol()

	// Create request
	originalReq, err := protocol.CreateCallToolRequest("test_tool", map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	})
	require.NoError(t, err)

	// Serialize
	data, err := protocol.SerializeRequest(originalReq)
	require.NoError(t, err)

	// Deserialize
	deserializedReq, err := protocol.DeserializeRequest(data)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, originalReq.JSONRPC, deserializedReq.JSONRPC)
	assert.Equal(t, originalReq.ID, deserializedReq.ID)
	assert.Equal(t, originalReq.Method, deserializedReq.Method)
}

// TestProtocol_ResponseWithError tests response with error.
func TestProtocol_ResponseWithError(t *testing.T) {
	protocol := NewProtocol()

	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      "test-id",
		Error: &MCPError{
			Code:    -32600,
			Message: "Invalid Request",
			Data:    map[string]interface{}{"detail": "missing parameter"},
		},
	}

	// Serialize
	data, err := protocol.SerializeResponse(resp)
	require.NoError(t, err)

	// Deserialize
	deserializedResp, err := protocol.DeserializeResponse(data)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, resp.JSONRPC, deserializedResp.JSONRPC)
	assert.Equal(t, resp.ID, deserializedResp.ID)
	assert.NotNil(t, deserializedResp.Error)
	assert.Equal(t, -32600, deserializedResp.Error.Code)
	assert.Equal(t, "Invalid Request", deserializedResp.Error.Message)
}

// TestProtocol_ResponseWithResult tests response with result.
func TestProtocol_ResponseWithResult(t *testing.T) {
	protocol := NewProtocol()

	resp := &MCPResponse{
		JSONRPC: "2.0",
		ID:      "test-id",
		Result: map[string]interface{}{
			"status": "success",
			"data":   []string{"item1", "item2"},
		},
	}

	// Serialize
	data, err := protocol.SerializeResponse(resp)
	require.NoError(t, err)

	// Deserialize
	deserializedResp, err := protocol.DeserializeResponse(data)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, resp.JSONRPC, deserializedResp.JSONRPC)
	assert.Equal(t, resp.ID, deserializedResp.ID)
	assert.NotNil(t, deserializedResp.Result)
	assert.Nil(t, deserializedResp.Error)
}
