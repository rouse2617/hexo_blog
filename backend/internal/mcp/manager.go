// Package mcp provides MCP (Model Context Protocol) client functionality.
package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// mcpManager implements the Manager interface.
type mcpManager struct {
	clients  map[string]Client
	configs  map[string]ServerConfig
	logger   *zap.Logger
	mu       sync.RWMutex
	
	// Status broadcast callback
	statusCallback func(serverID string, status ConnectionStatus)
	
	// Reconnection control
	reconnectInterval time.Duration
	stopReconnect     map[string]chan struct{}
}

// NewManager creates a new MCP connection manager.
func NewManager(logger *zap.Logger) Manager {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &mcpManager{
		clients:           make(map[string]Client),
		configs:           make(map[string]ServerConfig),
		logger:            logger,
		reconnectInterval: 30 * time.Second,
		stopReconnect:     make(map[string]chan struct{}),
	}
}

// SetStatusCallback sets the callback function for status updates.
func (m *mcpManager) SetStatusCallback(callback func(serverID string, status ConnectionStatus)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCallback = callback
}

// AddServer adds a new MCP server and attempts to connect.
func (m *mcpManager) AddServer(config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("server ID is required")
	}

	// Check if server already exists
	if _, exists := m.clients[config.ID]; exists {
		return fmt.Errorf("server %s already exists", config.ID)
	}

	// Create new client
	client := NewClient(m.logger)
	
	// Store config and client
	m.configs[config.ID] = config
	m.clients[config.ID] = client

	// Attempt to connect
	if err := client.Connect(config); err != nil {
		m.logger.Warn("Failed to connect to MCP server on add",
			zap.String("server", config.ID),
			zap.Error(err))
		
		// Start reconnection loop
		m.startReconnectLoop(config.ID)
		
		// Broadcast offline status
		if m.statusCallback != nil {
			m.statusCallback(config.ID, StatusOffline)
		}
		
		return nil // Don't return error, just start reconnecting
	}

	m.logger.Info("Successfully added and connected to MCP server",
		zap.String("server", config.ID))

	// Broadcast online status
	if m.statusCallback != nil {
		m.statusCallback(config.ID, StatusOnline)
	}

	return nil
}

// RemoveServer removes an MCP server and disconnects.
func (m *mcpManager) RemoveServer(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[serverID]
	if !exists {
		return fmt.Errorf("server %s not found", serverID)
	}

	// Stop reconnection loop if running
	if stopChan, exists := m.stopReconnect[serverID]; exists {
		close(stopChan)
		delete(m.stopReconnect, serverID)
	}

	// Disconnect client
	if err := client.Disconnect(); err != nil {
		m.logger.Warn("Error disconnecting from MCP server",
			zap.String("server", serverID),
			zap.Error(err))
	}

	// Remove from maps
	delete(m.clients, serverID)
	delete(m.configs, serverID)

	m.logger.Info("Removed MCP server", zap.String("server", serverID))

	// Broadcast offline status
	if m.statusCallback != nil {
		m.statusCallback(serverID, StatusOffline)
	}

	return nil
}

// GetClient returns the client for a specific server.
func (m *mcpManager) GetClient(serverID string) (Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[serverID]
	if !exists {
		return nil, fmt.Errorf("server %s not found", serverID)
	}

	return client, nil
}

// ListServers returns information about all managed servers.
func (m *mcpManager) ListServers() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]ServerInfo, 0, len(m.clients))
	for id, client := range m.clients {
		config := m.configs[id]
		tools, _ := client.ListTools() // Ignore error, tools may be empty

		servers = append(servers, ServerInfo{
			ID:     id,
			Name:   config.Name,
			Status: client.Status(),
			Tools:  tools,
		})
	}

	return servers
}

// BroadcastStatus broadcasts the status of all servers.
func (m *mcpManager) BroadcastStatus() error {
	m.mu.RLock()
	callback := m.statusCallback
	clients := make(map[string]Client, len(m.clients))
	for id, client := range m.clients {
		clients[id] = client
	}
	m.mu.RUnlock()

	if callback == nil {
		return fmt.Errorf("status callback not set")
	}

	for id, client := range clients {
		callback(id, client.Status())
	}

	return nil
}

// CallTool calls a tool on the appropriate MCP server.
func (m *mcpManager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find which server provides this tool
	for serverID, client := range m.clients {
		if client.Status() != StatusOnline {
			continue
		}

		tools, err := client.ListTools()
		if err != nil {
			continue
		}

		for _, tool := range tools {
			if tool.Name == toolName {
				m.logger.Info("Routing tool call to server",
					zap.String("tool", toolName),
					zap.String("server", serverID))
				
				return client.CallTool(ctx, toolName, args)
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found on any connected server", toolName)
}

// startReconnectLoop starts a goroutine that attempts to reconnect to a server.
func (m *mcpManager) startReconnectLoop(serverID string) {
	stopChan := make(chan struct{})
	m.stopReconnect[serverID] = stopChan

	go func() {
		ticker := time.NewTicker(m.reconnectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				m.logger.Info("Stopping reconnect loop", zap.String("server", serverID))
				return
			case <-ticker.C:
				m.attemptReconnect(serverID)
			}
		}
	}()
}

// attemptReconnect attempts to reconnect to a server.
func (m *mcpManager) attemptReconnect(serverID string) {
	m.mu.RLock()
	client, exists := m.clients[serverID]
	config, configExists := m.configs[serverID]
	m.mu.RUnlock()

	if !exists || !configExists {
		return
	}

	// Check current status
	if client.Status() == StatusOnline {
		// Already connected, stop reconnecting
		m.mu.Lock()
		if stopChan, exists := m.stopReconnect[serverID]; exists {
			close(stopChan)
			delete(m.stopReconnect, serverID)
		}
		m.mu.Unlock()
		return
	}

	m.logger.Info("Attempting to reconnect to MCP server", zap.String("server", serverID))

	// Attempt to connect
	if err := client.Connect(config); err != nil {
		m.logger.Warn("Reconnection attempt failed",
			zap.String("server", serverID),
			zap.Error(err))
		
		// Broadcast error status
		m.mu.RLock()
		callback := m.statusCallback
		m.mu.RUnlock()
		
		if callback != nil {
			callback(serverID, StatusError)
		}
		return
	}

	m.logger.Info("Successfully reconnected to MCP server", zap.String("server", serverID))

	// Stop reconnection loop
	m.mu.Lock()
	if stopChan, exists := m.stopReconnect[serverID]; exists {
		close(stopChan)
		delete(m.stopReconnect, serverID)
	}
	callback := m.statusCallback
	m.mu.Unlock()

	// Broadcast online status
	if callback != nil {
		callback(serverID, StatusOnline)
	}
}

// Shutdown gracefully shuts down the manager and all connections.
func (m *mcpManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Shutting down MCP manager")

	// Stop all reconnection loops
	for serverID, stopChan := range m.stopReconnect {
		close(stopChan)
		delete(m.stopReconnect, serverID)
	}

	// Disconnect all clients
	for serverID, client := range m.clients {
		if err := client.Disconnect(); err != nil {
			m.logger.Warn("Error disconnecting from MCP server during shutdown",
				zap.String("server", serverID),
				zap.Error(err))
		}
	}

	// Clear maps
	m.clients = make(map[string]Client)
	m.configs = make(map[string]ServerConfig)

	return nil
}
