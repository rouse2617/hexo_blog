package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opsgenius/backend/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockMessageHandler is a mock implementation of MessageHandler for testing.
type mockMessageHandler struct {
	messages [][]byte
	err      error
}

func (m *mockMessageHandler) HandleMessage(conn Connection, message []byte) error {
	m.messages = append(m.messages, message)
	return m.err
}

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	server := NewServer()
	assert.NotNil(t, server)
	assert.Equal(t, 0, server.ConnectionCount())
}

// TestNewServerWithOptions tests server creation with options.
func TestNewServerWithOptions(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &mockMessageHandler{}

	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(handler),
		WithHeartbeatInterval(10*time.Second),
		WithHeartbeatTimeout(1*time.Minute),
		WithAllowedOrigins([]string{"http://localhost:3000"}),
	)

	assert.NotNil(t, server)
	assert.Equal(t, logger, server.logger)
	assert.Equal(t, handler, server.handler)
	assert.Equal(t, 10*time.Second, server.heartbeatInterval)
	assert.Equal(t, 1*time.Minute, server.heartbeatTimeout)
}

// TestWebSocketConnection tests WebSocket connection establishment.
func TestWebSocketConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Verify connection count
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestWebSocketConnectionDisconnect tests WebSocket connection disconnection.
func TestWebSocketConnectionDisconnect(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Wait for connection to be registered
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())

	// Close connection
	ws.Close()

	// Wait for disconnection to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify connection count
	assert.Equal(t, 0, server.ConnectionCount())
}

// TestWebSocketMessageParsing tests message parsing.
func TestWebSocketMessageParsing(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &mockMessageHandler{}
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(handler),
	)

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Send a valid message
	msg := models.ClientMessage{
		Type: models.MessageTypeUserMessage,
		Payload: json.RawMessage(`{"sessionId":"test-session","content":"Hello"}`),
		Timestamp: time.Now().Unix(),
	}
	msgBytes, _ := json.Marshal(msg)
	err = ws.WriteMessage(websocket.TextMessage, msgBytes)
	require.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify message was received
	assert.Len(t, handler.messages, 1)
}

// TestWebSocketInvalidMessageHandling tests handling of invalid messages.
func TestWebSocketInvalidMessageHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &mockMessageHandler{}
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(handler),
	)

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Send an invalid message (not JSON)
	err = ws.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	require.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)

	// Connection should still be open (not closed due to invalid message)
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestWebSocketBroadcast tests broadcasting messages to all connections.
func TestWebSocketBroadcast(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Wait for connections
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, server.ConnectionCount())

	// Broadcast a message
	broadcastMsg := []byte(`{"type":"broadcast","payload":"hello"}`)
	err = server.Broadcast(broadcastMsg)
	require.NoError(t, err)

	// Read messages from both clients
	ws1.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg1, err := ws1.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, broadcastMsg, msg1)

	ws2.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg2, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, broadcastMsg, msg2)
}

// TestWebSocketSendToConnection tests sending messages to specific connections.
func TestWebSocketSendToConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Get connection ID
	var connID string
	server.connections.Range(func(key, value interface{}) bool {
		connID = key.(string)
		return false
	})
	require.NotEmpty(t, connID)

	// Send message to specific connection
	testMsg := []byte(`{"type":"test","payload":"hello"}`)
	err = server.SendToConnection(connID, testMsg)
	require.NoError(t, err)

	// Read message
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := ws.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, msg)
}

// TestWebSocketSendToNonExistentConnection tests sending to non-existent connection.
func TestWebSocketSendToNonExistentConnection(t *testing.T) {
	server := NewServer()

	err := server.SendToConnection("non-existent-id", []byte("test"))
	assert.Equal(t, ErrConnectionNotFound, err)
}

// TestWebSocketGetConnection tests getting a connection by ID.
func TestWebSocketGetConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Get connection ID
	var connID string
	server.connections.Range(func(key, value interface{}) bool {
		connID = key.(string)
		return false
	})

	// Get connection
	conn, ok := server.GetConnection(connID)
	assert.True(t, ok)
	assert.NotNil(t, conn)
	assert.Equal(t, connID, conn.ID())

	// Try to get non-existent connection
	_, ok = server.GetConnection("non-existent")
	assert.False(t, ok)
}

// TestHealthEndpoint tests the health check endpoint.
func TestHealthEndpoint(t *testing.T) {
	server := NewServer()

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Make health check request
	resp, err := http.Get(testServer.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "healthy", result["status"])
	assert.Equal(t, float64(0), result["connections"])
}

// TestGenerateConnectionID tests connection ID generation.
func TestGenerateConnectionID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate multiple IDs and verify uniqueness
	for i := 0; i < 1000; i++ {
		id := GenerateConnectionID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "Duplicate ID generated: %s", id)
		ids[id] = true
	}
}


// TestResourceManager tests the resource manager.
func TestResourceManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	rm := NewResourceManager(server,
		WithResourceLogger(logger),
		WithCleanupInterval(100*time.Millisecond),
		WithMaxIdleTime(200*time.Millisecond),
		WithMaxErrorCount(5),
	)

	assert.NotNil(t, rm)
}

// TestResourceManagerCleanupIdleConnections tests cleanup of idle connections.
func TestResourceManagerCleanupIdleConnections(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Track cleaned up connections
	cleanedUp := make([]string, 0)
	rm := NewResourceManager(server,
		WithResourceLogger(logger),
		WithCleanupInterval(50*time.Millisecond),
		WithMaxIdleTime(100*time.Millisecond),
		WithOnConnectionClose(func(connID string) {
			cleanedUp = append(cleanedUp, connID)
		}),
	)

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())

	// Start resource manager
	rm.Start()
	defer rm.Stop()

	// Wait for idle timeout and cleanup
	time.Sleep(300 * time.Millisecond)

	// Connection should be cleaned up
	assert.Equal(t, 0, server.ConnectionCount())
	assert.Len(t, cleanedUp, 1)

	ws.Close()
}

// TestResourceManagerGetConnectionStats tests getting connection statistics.
func TestResourceManagerGetConnectionStats(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	rm := NewResourceManager(server, WithResourceLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Wait for connections
	time.Sleep(100 * time.Millisecond)

	// Get stats
	stats := rm.GetConnectionStats()
	assert.Equal(t, 2, stats.TotalConnections)
	assert.Equal(t, 2, stats.ActiveConnections)
	assert.Equal(t, 0, stats.ClosedConnections)
	assert.Equal(t, 0, stats.ConnectionsWithErrors)
}

// TestResourceManagerForceCleanup tests force cleanup.
func TestResourceManagerForceCleanup(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	rm := NewResourceManager(server,
		WithResourceLogger(logger),
		WithMaxIdleTime(1*time.Millisecond),
	)

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())

	// Wait for idle timeout
	time.Sleep(50 * time.Millisecond)

	// Force cleanup
	rm.ForceCleanup()

	// Connection should be cleaned up
	assert.Equal(t, 0, server.ConnectionCount())

	ws.Close()
}

// TestHeartbeatMechanism tests the heartbeat mechanism.
func TestHeartbeatMechanism(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(
		WithLogger(logger),
		WithHeartbeatInterval(100*time.Millisecond),
		WithHeartbeatTimeout(5*time.Minute),
	)

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect to WebSocket with pong handler
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Set up pong handler to track heartbeats
	pongReceived := make(chan struct{}, 10)
	ws.SetPongHandler(func(appData string) error {
		pongReceived <- struct{}{}
		return nil
	})

	// Wait for connection
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())

	// The connection should remain active as long as we respond to pings
	// Read messages in a goroutine to handle pings
	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Wait and verify connection is still active
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestMultipleConcurrentConnections tests that multiple concurrent connections get unique IDs.
// Requirements: 1.1 - Each connection should have a unique ID
func TestMultipleConcurrentConnections(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	// Create test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect multiple clients concurrently
	numClients := 10
	clients := make([]*websocket.Conn, numClients)
	
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = ws
	}
	defer func() {
		for _, ws := range clients {
			ws.Close()
		}
	}()

	// Wait for all connections to be registered
	time.Sleep(200 * time.Millisecond)

	// Verify connection count
	assert.Equal(t, numClients, server.ConnectionCount())

	// Verify all connection IDs are unique
	ids := make(map[string]bool)
	server.connections.Range(func(key, value interface{}) bool {
		connID := key.(string)
		assert.False(t, ids[connID], "Duplicate connection ID found: %s", connID)
		ids[connID] = true
		return true
	})
	assert.Equal(t, numClients, len(ids))
}

// TestConnectionErrorCountTracking tests that error counts are tracked correctly.
// Requirements: 1.4 - Connection resource management
func TestConnectionErrorCountTracking(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	
	// Create a handler that returns errors
	errorHandler := &mockMessageHandler{err: assert.AnError}
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(errorHandler),
	)

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Send multiple messages that will cause handler errors
	for i := 0; i < 3; i++ {
		msg := models.ClientMessage{
			Type:      models.MessageTypeUserMessage,
			Payload:   json.RawMessage(`{"sessionId":"test","content":"test"}`),
			Timestamp: time.Now().Unix(),
		}
		msgBytes, _ := json.Marshal(msg)
		err = ws.WriteMessage(websocket.TextMessage, msgBytes)
		require.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// Verify error count was incremented
	var conn *WebSocketConnection
	server.connections.Range(func(key, value interface{}) bool {
		conn = value.(*WebSocketConnection)
		return false
	})
	require.NotNil(t, conn)
	assert.Equal(t, 3, conn.ErrorCount())
}

// TestMessageParsingEmptyPayload tests handling of messages with empty payload.
// Requirements: 1.1 - Message parsing and routing
func TestMessageParsingEmptyPayload(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	router := NewRouter(WithRouterLogger(logger))
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(router),
	)

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Send message with empty type
	msg := `{"type":"","payload":{},"timestamp":123}`
	err = ws.WriteMessage(websocket.TextMessage, []byte(msg))
	require.NoError(t, err)

	// Read error response
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, respBytes, err := ws.ReadMessage()
	require.NoError(t, err)

	var resp models.ServerMessage
	err = json.Unmarshal(respBytes, &resp)
	require.NoError(t, err)
	assert.Equal(t, models.MessageTypeError, resp.Type)

	// Connection should still be open
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestMessageParsingUnknownType tests handling of messages with unknown type.
// Requirements: 1.1 - Message parsing and routing
func TestMessageParsingUnknownType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	router := NewRouter(WithRouterLogger(logger))
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(router),
	)

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Send message with unknown type
	msg := models.ClientMessage{
		Type:      "unknown_message_type",
		Payload:   json.RawMessage(`{}`),
		Timestamp: time.Now().Unix(),
	}
	msgBytes, _ := json.Marshal(msg)
	err = ws.WriteMessage(websocket.TextMessage, msgBytes)
	require.NoError(t, err)

	// Read error response
	ws.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, respBytes, err := ws.ReadMessage()
	require.NoError(t, err)

	var resp models.ServerMessage
	err = json.Unmarshal(respBytes, &resp)
	require.NoError(t, err)
	assert.Equal(t, models.MessageTypeError, resp.Type)

	// Verify error payload contains appropriate message
	payloadBytes, _ := json.Marshal(resp.Payload)
	var errPayload models.ErrorPayload
	json.Unmarshal(payloadBytes, &errPayload)
	assert.Equal(t, "UNKNOWN_MESSAGE_TYPE", errPayload.Code)

	// Connection should still be open
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestMessageParsingMalformedJSON tests handling of malformed JSON messages.
// Requirements: 1.1 - Message parsing error handling
func TestMessageParsingMalformedJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	router := NewRouter(WithRouterLogger(logger))
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(router),
	)

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Send malformed JSON
	malformedMessages := []string{
		`{invalid json}`,
		`{"type": "user_message", "payload": }`,
		`not json at all`,
		`{"type": 123}`, // type should be string
	}

	for _, msg := range malformedMessages {
		err = ws.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)

		// Read error response
		ws.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, respBytes, err := ws.ReadMessage()
		require.NoError(t, err)

		var resp models.ServerMessage
		err = json.Unmarshal(respBytes, &resp)
		require.NoError(t, err)
		assert.Equal(t, models.MessageTypeError, resp.Type)
	}

	// Connection should still be open after all malformed messages
	assert.Equal(t, 1, server.ConnectionCount())
}

// TestConnectionDisconnectResourceCleanup tests that resources are cleaned up on disconnect.
// Requirements: 1.4 - Connection disconnect resource cleanup
func TestConnectionDisconnectResourceCleanup(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	// Connect
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, server.ConnectionCount())

	// Get connection ID before disconnect
	var connID string
	server.connections.Range(func(key, value interface{}) bool {
		connID = key.(string)
		return false
	})
	require.NotEmpty(t, connID)

	// Disconnect
	ws.Close()

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Verify connection is removed
	assert.Equal(t, 0, server.ConnectionCount())

	// Verify connection is no longer accessible
	_, ok := server.GetConnection(connID)
	assert.False(t, ok)
}

// TestConnectionCloseIdempotent tests that closing a connection multiple times is safe.
// Requirements: 1.4 - Connection resource management
func TestConnectionCloseIdempotent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Get the connection
	var conn *WebSocketConnection
	server.connections.Range(func(key, value interface{}) bool {
		conn = value.(*WebSocketConnection)
		return false
	})
	require.NotNil(t, conn)

	// Close multiple times - should not panic
	err = conn.Close()
	assert.NoError(t, err)

	err = conn.Close()
	assert.NoError(t, err)

	// Verify connection is marked as closed
	assert.True(t, conn.IsClosed())

	ws.Close()
}

// TestSendToClosedConnection tests sending to a closed connection.
// Requirements: 1.4 - Connection state management
func TestSendToClosedConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := NewServer(WithLogger(logger))

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Get the connection
	var conn *WebSocketConnection
	server.connections.Range(func(key, value interface{}) bool {
		conn = value.(*WebSocketConnection)
		return false
	})
	require.NotNil(t, conn)

	// Close the connection
	conn.Close()

	// Try to send - should return error
	err = conn.Send([]byte("test"))
	assert.Equal(t, ErrConnectionClosed, err)

	err = conn.SendJSON(map[string]string{"test": "data"})
	assert.Equal(t, ErrConnectionClosed, err)

	ws.Close()
}

// TestConnectionLastActivityUpdate tests that last activity is updated on message receive.
// Requirements: 1.4 - Connection activity tracking
func TestConnectionLastActivityUpdate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	handler := &mockMessageHandler{}
	server := NewServer(
		WithLogger(logger),
		WithMessageHandler(handler),
	)

	testServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer testServer.Close()

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	// Get the connection and initial activity time
	var conn *WebSocketConnection
	server.connections.Range(func(key, value interface{}) bool {
		conn = value.(*WebSocketConnection)
		return false
	})
	require.NotNil(t, conn)

	initialActivity := conn.LastActivity()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Send a message
	msg := models.ClientMessage{
		Type:      models.MessageTypeHeartbeat,
		Payload:   json.RawMessage(`{}`),
		Timestamp: time.Now().Unix(),
	}
	msgBytes, _ := json.Marshal(msg)
	err = ws.WriteMessage(websocket.TextMessage, msgBytes)
	require.NoError(t, err)

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)

	// Verify last activity was updated
	newActivity := conn.LastActivity()
	assert.True(t, newActivity.After(initialActivity), "Last activity should be updated after receiving message")
}
