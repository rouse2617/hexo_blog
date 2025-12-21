package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestManager_AddServer(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	config := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:9999",
	}

	// Adding server should not return error even if connection fails
	err := manager.AddServer(config)
	assert.NoError(t, err)

	// Server should be in the list
	servers := manager.ListServers()
	assert.Len(t, servers, 1)
	assert.Equal(t, "test-server", servers[0].ID)
	assert.Equal(t, "Test Server", servers[0].Name)
}

func TestManager_AddServer_DuplicateID(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	config := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:9999",
	}

	// Add first server
	err := manager.AddServer(config)
	assert.NoError(t, err)

	// Try to add duplicate
	err = manager.AddServer(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_RemoveServer(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	config := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:9999",
	}

	// Add server
	err := manager.AddServer(config)
	require.NoError(t, err)

	// Remove server
	err = manager.RemoveServer("test-server")
	assert.NoError(t, err)

	// Server should not be in the list
	servers := manager.ListServers()
	assert.Len(t, servers, 0)
}

func TestManager_RemoveServer_NotFound(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	err := manager.RemoveServer("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_GetClient(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	config := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:9999",
	}

	// Add server
	err := manager.AddServer(config)
	require.NoError(t, err)

	// Get client
	client, err := manager.GetClient("test-server")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestManager_GetClient_NotFound(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	client, err := manager.GetClient("non-existent")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_ListServers(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger)

	// Initially empty
	servers := manager.ListServers()
	assert.Len(t, servers, 0)

	// Add multiple servers
	configs := []ServerConfig{
		{
			ID:        "server-1",
			Name:      "Server 1",
			Transport: "http",
			URL:       "http://localhost:9001",
		},
		{
			ID:        "server-2",
			Name:      "Server 2",
			Transport: "http",
			URL:       "http://localhost:9002",
		},
	}

	for _, config := range configs {
		err := manager.AddServer(config)
		require.NoError(t, err)
	}

	// List should contain all servers
	servers = manager.ListServers()
	assert.Len(t, servers, 2)

	// Verify server IDs
	ids := make(map[string]bool)
	for _, server := range servers {
		ids[server.ID] = true
	}
	assert.True(t, ids["server-1"])
	assert.True(t, ids["server-2"])
}

func TestManager_BroadcastStatus(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger).(*mcpManager)

	// Without callback, should return error
	err := manager.BroadcastStatus()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "callback not set")

	// Set callback
	callCount := 0
	manager.SetStatusCallback(func(serverID string, status ConnectionStatus) {
		callCount++
	})

	// Add servers
	configs := []ServerConfig{
		{
			ID:        "server-1",
			Name:      "Server 1",
			Transport: "http",
			URL:       "http://localhost:9001",
		},
		{
			ID:        "server-2",
			Name:      "Server 2",
			Transport: "http",
			URL:       "http://localhost:9002",
		},
	}

	for _, config := range configs {
		err := manager.AddServer(config)
		require.NoError(t, err)
	}

	// Reset call count (AddServer calls callback)
	callCount = 0

	// Broadcast status
	err = manager.BroadcastStatus()
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestManager_Shutdown(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger).(*mcpManager)

	// Add servers
	configs := []ServerConfig{
		{
			ID:        "server-1",
			Name:      "Server 1",
			Transport: "http",
			URL:       "http://localhost:9001",
		},
		{
			ID:        "server-2",
			Name:      "Server 2",
			Transport: "http",
			URL:       "http://localhost:9002",
		},
	}

	for _, config := range configs {
		err := manager.AddServer(config)
		require.NoError(t, err)
	}

	// Shutdown
	err := manager.Shutdown()
	assert.NoError(t, err)

	// All servers should be removed
	servers := manager.ListServers()
	assert.Len(t, servers, 0)
}

func TestManager_ReconnectLoop(t *testing.T) {
	logger := zap.NewNop()
	manager := NewManager(logger).(*mcpManager)

	// Set short reconnect interval for testing
	manager.reconnectInterval = 100 * time.Millisecond

	// Track status callbacks
	statusUpdates := make([]ConnectionStatus, 0)
	manager.SetStatusCallback(func(serverID string, status ConnectionStatus) {
		statusUpdates = append(statusUpdates, status)
	})

	config := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "http",
		URL:       "http://localhost:9999", // Invalid URL
	}

	// Add server (will fail to connect)
	err := manager.AddServer(config)
	require.NoError(t, err)

	// Wait for at least one reconnection attempt
	time.Sleep(250 * time.Millisecond)

	// Should have received offline status
	assert.Greater(t, len(statusUpdates), 0)

	// Cleanup
	manager.Shutdown()
}
