// Package websocket provides WebSocket connection management.
package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrConnectionClosed   = errors.New("connection is closed")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrInvalidMessage     = errors.New("invalid message format")
)

// WebSocketConnection implements the Connection interface.
type WebSocketConnection struct {
	id           string
	conn         *websocket.Conn
	handler      MessageHandler
	logger       *zap.Logger
	lastActivity time.Time
	errorCount   atomic.Int32
	closed       atomic.Bool
	writeMu      sync.Mutex
	activityMu   sync.RWMutex
}

// NewWebSocketConnection creates a new WebSocket connection.
func NewWebSocketConnection(conn *websocket.Conn, logger *zap.Logger) *WebSocketConnection {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &WebSocketConnection{
		id:           uuid.New().String(),
		conn:         conn,
		logger:       logger,
		lastActivity: time.Now(),
	}
}

// ID returns the connection ID.
func (c *WebSocketConnection) ID() string {
	return c.id
}

// Send sends a raw message to the client.
func (c *WebSocketConnection) Send(message []byte) error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return err
	}

	return nil
}

// SendJSON sends a JSON-encoded message to the client.
func (c *WebSocketConnection) SendJSON(v interface{}) error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.Send(data)
}

// SendError sends an error message to the client.
func (c *WebSocketConnection) SendError(message string) error {
	return c.SendJSON(map[string]interface{}{
		"type": "error",
		"payload": map[string]string{
			"message": message,
		},
		"timestamp": time.Now().Unix(),
	})
}

// Close closes the connection.
func (c *WebSocketConnection) Close() error {
	if c.closed.Swap(true) {
		return nil // Already closed
	}

	// Only try to close if we have a real connection
	if c.conn == nil {
		return nil
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// Send close message
	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	return c.conn.Close()
}

// ReadMessage reads a message from the client.
func (c *WebSocketConnection) ReadMessage() ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrConnectionClosed
	}

	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	c.updateActivity()
	return message, nil
}

// SetMessageHandler sets the message handler for this connection.
func (c *WebSocketConnection) SetMessageHandler(handler MessageHandler) {
	c.handler = handler
}

// LastActivity returns the time of the last activity.
func (c *WebSocketConnection) LastActivity() time.Time {
	c.activityMu.RLock()
	defer c.activityMu.RUnlock()
	return c.lastActivity
}

// updateActivity updates the last activity time.
func (c *WebSocketConnection) updateActivity() {
	c.activityMu.Lock()
	defer c.activityMu.Unlock()
	c.lastActivity = time.Now()
}

// IncrementErrorCount increments the error count.
func (c *WebSocketConnection) IncrementErrorCount() {
	c.errorCount.Add(1)
}

// ErrorCount returns the error count.
func (c *WebSocketConnection) ErrorCount() int {
	return int(c.errorCount.Load())
}

// IsClosed returns whether the connection is closed.
func (c *WebSocketConnection) IsClosed() bool {
	return c.closed.Load()
}

// sendPing sends a ping message to the client.
func (c *WebSocketConnection) sendPing() error {
	if c.closed.Load() {
		return ErrConnectionClosed
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

// RemoteAddr returns the remote address of the connection.
func (c *WebSocketConnection) RemoteAddr() string {
	if c.conn != nil {
		return c.conn.RemoteAddr().String()
	}
	return ""
}
