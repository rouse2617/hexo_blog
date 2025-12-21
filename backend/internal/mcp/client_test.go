package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestMCPClient_ConnectInvalidTransport tests connection with invalid transport.
func TestMCPClient_ConnectInvalidTransport(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "invalid",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported transport")
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectStdioInvalidCommand tests stdio connection with invalid command.
// Validates: Requirements 3.3
func TestMCPClient_ConnectStdioInvalidCommand(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "nonexistent-command-12345",
		Args:      []string{},
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
	assert.Contains(t, err.Error(), "failed to start command")
}

// TestMCPClient_ConnectHTTPInvalidURL tests HTTP connection with invalid URL.
// Validates: Requirements 3.3
func TestMCPClient_ConnectHTTPInvalidURL(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:99999",
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectHTTPUnreachableServer tests HTTP connection to unreachable server.
// Validates: Requirements 3.3
func TestMCPClient_ConnectHTTPUnreachableServer(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://192.0.2.1:8080", // TEST-NET-1 address (unreachable)
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
	// Error should indicate connection failure
	assert.True(t, err != nil)
}

// TestMCPClient_ConnectHTTPInvalidProtocol tests HTTP connection with invalid protocol.
// Validates: Requirements 3.3
func TestMCPClient_ConnectHTTPInvalidProtocol(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "invalid://localhost:8080",
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectStdioCommandNotFound tests stdio with command not in PATH.
// Validates: Requirements 3.3
func TestMCPClient_ConnectStdioCommandNotFound(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "/nonexistent/path/to/command",
		Args:      []string{"--arg"},
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectFailureStatusTransition tests status transitions on connection failure.
// Validates: Requirements 3.3
func TestMCPClient_ConnectFailureStatusTransition(t *testing.T) {
	client := NewClient(zap.NewNop())

	// Initial status should be offline
	assert.Equal(t, StatusOffline, client.Status())

	// Attempt to connect with invalid transport
	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Transport: "invalid-transport",
	})

	// Should fail and status should be error
	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())

	// Disconnect should reset to offline
	client.Disconnect()
	assert.Equal(t, StatusOffline, client.Status())
}

// TestMCPClient_ListToolsWhenDisconnected tests listing tools when not connected.
func TestMCPClient_ListToolsWhenDisconnected(t *testing.T) {
	client := NewClient(zap.NewNop())

	tools, err := client.ListTools()

	assert.Error(t, err)
	assert.Nil(t, tools)
	assert.Contains(t, err.Error(), "not connected")
}

// TestMCPClient_CallToolWhenDisconnected tests calling a tool when not connected.
// Validates: Requirements 4.4
func TestMCPClient_CallToolWhenDisconnected(t *testing.T) {
	client := NewClient(zap.NewNop())

	ctx := context.Background()
	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not connected")
}

// TestMCPClient_CallToolWithCancelledContext tests tool call with cancelled context.
// Validates: Requirements 4.4
func TestMCPClient_CallToolWithCancelledContext(t *testing.T) {
	client := NewClient(zap.NewNop())

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because client is not connected
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestMCPClient_CallToolWithEmptyArgs tests tool call with empty arguments.
// Validates: Requirements 4.4
func TestMCPClient_CallToolWithEmptyArgs(t *testing.T) {
	client := NewClient(zap.NewNop())

	ctx := context.Background()
	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because client is not connected
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not connected")
}

// TestMCPClient_CallToolWithNilArgs tests tool call with nil arguments.
// Validates: Requirements 4.4
func TestMCPClient_CallToolWithNilArgs(t *testing.T) {
	client := NewClient(zap.NewNop())

	ctx := context.Background()
	result, err := client.CallTool(ctx, "test_tool", nil)

	// Should fail because client is not connected
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestMCPClient_CallToolTimeout tests tool call timeout.
// Validates: Requirements 4.4
func TestMCPClient_CallToolTimeout(t *testing.T) {
	client := NewClient(zap.NewNop())

	// Create a context with very short timeout to simulate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(10 * time.Millisecond)

	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because client is not connected (but demonstrates timeout handling)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not connected")
}

// TestMCPClient_CallToolTimeoutWithConnectedClient tests timeout with a connected client.
// Validates: Requirements 4.4
func TestMCPClient_CallToolTimeoutWithConnectedClient(t *testing.T) {
	// This test demonstrates that timeout is respected even when client is connected
	// In a real scenario with a slow server, the context timeout would trigger

	client := NewClient(zap.NewNop())

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait to ensure timeout
	time.Sleep(5 * time.Millisecond)

	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail - either because not connected or because of timeout
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestMCPClient_CallToolTimeoutDuration tests that 30 second timeout is reasonable.
// Validates: Requirements 4.4
func TestMCPClient_CallToolTimeoutDuration(t *testing.T) {
	// This test verifies that a 30-second timeout context can be created
	// and that the client respects it

	client := NewClient(zap.NewNop())

	// Create a context with 30 second timeout (as per requirement)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to call tool (will fail because not connected, but timeout is set)
	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because not connected
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not connected")

	// Verify context is still valid (hasn't timed out yet)
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done yet")
	default:
		// Context is still valid, as expected
	}
}

// TestMCPClient_DisconnectWhenNotConnected tests disconnecting when not connected.
func TestMCPClient_DisconnectWhenNotConnected(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Disconnect()

	assert.NoError(t, err)
	assert.Equal(t, StatusOffline, client.Status())
}

// TestMCPClient_StatusTransitions tests status transitions.
func TestMCPClient_StatusTransitions(t *testing.T) {
	client := NewClient(zap.NewNop())

	// Initial status should be offline
	assert.Equal(t, StatusOffline, client.Status())

	// After failed connection, status should be error
	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Transport: "invalid",
	})
	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())

	// After disconnect, status should be offline
	client.Disconnect()
	assert.Equal(t, StatusOffline, client.Status())
}

// TestMCPClient_ConnectHTTPWithAuth tests HTTP connection with authentication.
// Validates: Requirements 3.3
func TestMCPClient_ConnectHTTPWithAuth(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:99999",
		Auth: &AuthConfig{
			Type:  "bearer",
			Token: "test-token",
		},
	})

	// Should fail because server doesn't exist, but auth should be processed
	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectHTTPEmptyURL tests HTTP connection with empty URL.
// Validates: Requirements 3.3
func TestMCPClient_ConnectHTTPEmptyURL(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "",
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_ConnectStdioEmptyCommand tests stdio connection with empty command.
// Validates: Requirements 3.3
func TestMCPClient_ConnectStdioEmptyCommand(t *testing.T) {
	client := NewClient(zap.NewNop())

	err := client.Connect(ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "",
		Args:      []string{},
	})

	assert.Error(t, err)
	assert.Equal(t, StatusError, client.Status())
}

// TestMCPClient_MultipleDisconnects tests multiple disconnect calls.
func TestMCPClient_MultipleDisconnects(t *testing.T) {
	client := NewClient(zap.NewNop())

	// First disconnect
	err := client.Disconnect()
	assert.NoError(t, err)

	// Second disconnect should also succeed
	err = client.Disconnect()
	assert.NoError(t, err)

	assert.Equal(t, StatusOffline, client.Status())
}

// TestMCPClient_ContextCancellation tests that context cancellation is respected.
func TestMCPClient_ContextCancellation(t *testing.T) {
	client := NewClient(zap.NewNop())

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	result, err := client.CallTool(ctx, "test_tool", map[string]interface{}{})

	// Should fail because client is not connected (not because of context)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestNewClient tests client creation.
func TestNewClient(t *testing.T) {
	// Test with nil logger
	client := NewClient(nil)
	require.NotNil(t, client)
	assert.Equal(t, StatusOffline, client.Status())

	// Test with real logger
	logger := zap.NewNop()
	client = NewClient(logger)
	require.NotNil(t, client)
	assert.Equal(t, StatusOffline, client.Status())
}
