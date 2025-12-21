// Package websocket provides WebSocket server functionality for real-time communication.
package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/opsgenius/backend/pkg/models"
	"go.uber.org/zap"
)

// Server interface defines the WebSocket server operations.
type Server interface {
	Start(addr string) error
	Stop() error
	Broadcast(message []byte) error
	SendToConnection(connID string, message []byte) error
	GetConnection(connID string) (Connection, bool)
	ConnectionCount() int
}

// Connection interface defines a WebSocket connection.
type Connection interface {
	ID() string
	Send(message []byte) error
	SendJSON(v interface{}) error
	Close() error
	ReadMessage() ([]byte, error)
	SetMessageHandler(handler MessageHandler)
	LastActivity() time.Time
	IncrementErrorCount()
	ErrorCount() int
	IsClosed() bool
}

// MessageHandler interface defines message handling operations.
type MessageHandler interface {
	HandleMessage(conn Connection, message []byte) error
}

// ConnectionStatus represents the status of a connection.
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
)

// WebSocketServer implements the Server interface.
type WebSocketServer struct {
	upgrader    websocket.Upgrader
	connections sync.Map // map[string]*WebSocketConnection
	connCount   atomic.Int32
	handler     MessageHandler
	logger      *zap.Logger
	httpServer  *http.Server
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}

	// Heartbeat configuration
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
}

// ServerOption is a function that configures a WebSocketServer.
type ServerOption func(*WebSocketServer)

// WithLogger sets the logger for the server.
func WithLogger(logger *zap.Logger) ServerOption {
	return func(s *WebSocketServer) {
		s.logger = logger
	}
}

// WithMessageHandler sets the message handler for the server.
func WithMessageHandler(handler MessageHandler) ServerOption {
	return func(s *WebSocketServer) {
		s.handler = handler
	}
}

// WithHeartbeatInterval sets the heartbeat interval.
func WithHeartbeatInterval(interval time.Duration) ServerOption {
	return func(s *WebSocketServer) {
		s.heartbeatInterval = interval
	}
}

// WithHeartbeatTimeout sets the heartbeat timeout.
func WithHeartbeatTimeout(timeout time.Duration) ServerOption {
	return func(s *WebSocketServer) {
		s.heartbeatTimeout = timeout
	}
}

// WithAllowedOrigins sets the allowed origins for CORS.
func WithAllowedOrigins(origins []string) ServerOption {
	return func(s *WebSocketServer) {
		s.upgrader.CheckOrigin = func(r *http.Request) bool {
			if len(origins) == 0 {
				return true
			}
			origin := r.Header.Get("Origin")
			for _, allowed := range origins {
				if allowed == "*" || allowed == origin {
					return true
				}
			}
			return false
		}
	}
}

// NewServer creates a new WebSocket server.
func NewServer(opts ...ServerOption) *WebSocketServer {
	s := &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Default: allow all origins
			},
		},
		stopChan:          make(chan struct{}),
		heartbeatInterval: 30 * time.Second,
		heartbeatTimeout:  5 * time.Minute,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Set default logger if not provided
	if s.logger == nil {
		s.logger, _ = zap.NewProduction()
	}

	return s
}

// Start starts the WebSocket server.
func (s *WebSocketServer) Start(addr string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("Starting WebSocket server", zap.String("addr", addr))

	// Start heartbeat checker
	go s.heartbeatChecker()

	return s.httpServer.ListenAndServe()
}

// Stop stops the WebSocket server.
func (s *WebSocketServer) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	// Signal stop
	close(s.stopChan)

	// Close all connections
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*WebSocketConnection); ok {
			conn.Close()
		}
		return true
	})

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Close()
	}

	return nil
}

// handleWebSocket handles WebSocket upgrade requests.
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create new connection with unique ID
	wsConn := NewWebSocketConnection(conn, s.logger)
	wsConn.SetMessageHandler(s.handler)

	// Register connection
	s.connections.Store(wsConn.ID(), wsConn)
	s.connCount.Add(1)

	s.logger.Info("New WebSocket connection",
		zap.String("connID", wsConn.ID()),
		zap.Int32("totalConnections", s.connCount.Load()))

	// Handle connection in goroutine
	go s.handleConnection(wsConn)
}

// handleConnection handles a WebSocket connection.
func (s *WebSocketServer) handleConnection(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic in connection handler",
				zap.Any("panic", r),
				zap.String("connID", conn.ID()))
		}
		s.removeConnection(conn)
	}()

	for {
		select {
		case <-s.stopChan:
			return
		default:
			message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.logger.Warn("WebSocket read error",
						zap.String("connID", conn.ID()),
						zap.Error(err))
				}
				return
			}

			// Update last activity
			conn.updateActivity()

			// Handle message
			if s.handler != nil {
				if err := s.handler.HandleMessage(conn, message); err != nil {
					s.logger.Error("Failed to handle message",
						zap.String("connID", conn.ID()),
						zap.Error(err))
					conn.IncrementErrorCount()

					// Send error response
					errMsg := models.ServerMessage{
						Type: models.MessageTypeError,
						Payload: models.ErrorPayload{
							Code:    "MESSAGE_HANDLING_ERROR",
							Message: "Failed to process message",
						},
						Timestamp: time.Now().Unix(),
					}
					conn.SendJSON(errMsg)
				}
			}
		}
	}
}

// removeConnection removes a connection from the server.
func (s *WebSocketServer) removeConnection(conn *WebSocketConnection) {
	conn.Close()
	_, loaded := s.connections.LoadAndDelete(conn.ID())
	if loaded {
		s.connCount.Add(-1)
	}

	s.logger.Info("WebSocket connection closed",
		zap.String("connID", conn.ID()),
		zap.Int32("totalConnections", s.connCount.Load()))
}

// handleHealth handles health check requests.
func (s *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"connections": s.connCount.Load(),
	})
}

// heartbeatChecker periodically checks for inactive connections.
func (s *WebSocketServer) heartbeatChecker() {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkHeartbeats()
		}
	}
}

// checkHeartbeats checks all connections for heartbeat timeout.
func (s *WebSocketServer) checkHeartbeats() {
	now := time.Now()
	s.connections.Range(func(key, value interface{}) bool {
		conn, ok := value.(*WebSocketConnection)
		if !ok {
			return true
		}

		// Check if connection has timed out
		if now.Sub(conn.LastActivity()) > s.heartbeatTimeout {
			s.logger.Info("Connection timed out",
				zap.String("connID", conn.ID()),
				zap.Duration("inactive", now.Sub(conn.LastActivity())))
			s.removeConnection(conn)
			return true
		}

		// Send heartbeat ping
		if err := conn.sendPing(); err != nil {
			s.logger.Warn("Failed to send heartbeat",
				zap.String("connID", conn.ID()),
				zap.Error(err))
		}

		return true
	})
}

// Broadcast sends a message to all connected clients.
func (s *WebSocketServer) Broadcast(message []byte) error {
	var lastErr error
	s.connections.Range(func(key, value interface{}) bool {
		conn, ok := value.(*WebSocketConnection)
		if !ok {
			return true
		}
		if err := conn.Send(message); err != nil {
			lastErr = err
			s.logger.Warn("Failed to broadcast to connection",
				zap.String("connID", conn.ID()),
				zap.Error(err))
		}
		return true
	})
	return lastErr
}

// SendToConnection sends a message to a specific connection.
func (s *WebSocketServer) SendToConnection(connID string, message []byte) error {
	value, ok := s.connections.Load(connID)
	if !ok {
		return ErrConnectionNotFound
	}
	conn, ok := value.(*WebSocketConnection)
	if !ok {
		return ErrConnectionNotFound
	}
	return conn.Send(message)
}

// GetConnection returns a connection by ID.
func (s *WebSocketServer) GetConnection(connID string) (Connection, bool) {
	value, ok := s.connections.Load(connID)
	if !ok {
		return nil, false
	}
	conn, ok := value.(*WebSocketConnection)
	return conn, ok
}

// ConnectionCount returns the number of active connections.
func (s *WebSocketServer) ConnectionCount() int {
	return int(s.connCount.Load())
}

// AcceptConnection accepts a new connection (for testing).
func (s *WebSocketServer) AcceptConnection(wsConn *websocket.Conn) *WebSocketConnection {
	conn := NewWebSocketConnection(wsConn, s.logger)
	s.connections.Store(conn.ID(), conn)
	s.connCount.Add(1)
	return conn
}

// GenerateConnectionID generates a unique connection ID.
func GenerateConnectionID() string {
	return uuid.New().String()
}

// ResourceManager manages connection resources and cleanup.
type ResourceManager struct {
	server            *WebSocketServer
	cleanupInterval   time.Duration
	maxIdleTime       time.Duration
	maxErrorCount     int
	stopChan          chan struct{}
	logger            *zap.Logger
	onConnectionClose func(connID string)
}

// ResourceManagerOption is a function that configures a ResourceManager.
type ResourceManagerOption func(*ResourceManager)

// WithCleanupInterval sets the cleanup interval.
func WithCleanupInterval(interval time.Duration) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.cleanupInterval = interval
	}
}

// WithMaxIdleTime sets the maximum idle time before cleanup.
func WithMaxIdleTime(duration time.Duration) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.maxIdleTime = duration
	}
}

// WithMaxErrorCount sets the maximum error count before cleanup.
func WithMaxErrorCount(count int) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.maxErrorCount = count
	}
}

// WithResourceLogger sets the logger for the resource manager.
func WithResourceLogger(logger *zap.Logger) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.logger = logger
	}
}

// WithOnConnectionClose sets the callback for connection close events.
func WithOnConnectionClose(callback func(connID string)) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.onConnectionClose = callback
	}
}

// NewResourceManager creates a new resource manager.
func NewResourceManager(server *WebSocketServer, opts ...ResourceManagerOption) *ResourceManager {
	rm := &ResourceManager{
		server:          server,
		cleanupInterval: 30 * time.Second,
		maxIdleTime:     5 * time.Minute,
		maxErrorCount:   10,
		stopChan:        make(chan struct{}),
	}

	for _, opt := range opts {
		opt(rm)
	}

	if rm.logger == nil {
		rm.logger, _ = zap.NewProduction()
	}

	return rm
}

// Start starts the resource manager.
func (rm *ResourceManager) Start() {
	go rm.cleanupLoop()
}

// Stop stops the resource manager.
func (rm *ResourceManager) Stop() {
	close(rm.stopChan)
}

// cleanupLoop periodically cleans up idle and errored connections.
func (rm *ResourceManager) cleanupLoop() {
	ticker := time.NewTicker(rm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			rm.cleanup()
		}
	}
}

// cleanup cleans up idle and errored connections.
func (rm *ResourceManager) cleanup() {
	now := time.Now()
	var toRemove []string

	rm.server.connections.Range(func(key, value interface{}) bool {
		connID := key.(string)
		conn, ok := value.(*WebSocketConnection)
		if !ok {
			return true
		}

		// Check if connection is closed
		if conn.IsClosed() {
			toRemove = append(toRemove, connID)
			return true
		}

		// Check for idle timeout
		if now.Sub(conn.LastActivity()) > rm.maxIdleTime {
			rm.logger.Info("Cleaning up idle connection",
				zap.String("connID", connID),
				zap.Duration("idleTime", now.Sub(conn.LastActivity())))
			toRemove = append(toRemove, connID)
			return true
		}

		// Check for too many errors
		if conn.ErrorCount() >= rm.maxErrorCount {
			rm.logger.Info("Cleaning up connection with too many errors",
				zap.String("connID", connID),
				zap.Int("errorCount", conn.ErrorCount()))
			toRemove = append(toRemove, connID)
			return true
		}

		return true
	})

	// Remove connections
	for _, connID := range toRemove {
		rm.removeConnection(connID)
	}
}

// removeConnection removes a connection and cleans up resources.
func (rm *ResourceManager) removeConnection(connID string) {
	value, ok := rm.server.connections.LoadAndDelete(connID)
	if !ok {
		return
	}

	conn, ok := value.(*WebSocketConnection)
	if !ok {
		return
	}

	// Close the connection
	conn.Close()

	// Decrement count only if we successfully removed
	rm.server.connCount.Add(-1)

	// Call callback if set
	if rm.onConnectionClose != nil {
		rm.onConnectionClose(connID)
	}

	rm.logger.Info("Connection cleaned up",
		zap.String("connID", connID),
		zap.Int32("remainingConnections", rm.server.connCount.Load()))
}

// GetConnectionStats returns statistics about connections.
func (rm *ResourceManager) GetConnectionStats() ConnectionStats {
	stats := ConnectionStats{
		TotalConnections: int(rm.server.connCount.Load()),
	}

	now := time.Now()
	rm.server.connections.Range(func(key, value interface{}) bool {
		conn, ok := value.(*WebSocketConnection)
		if !ok {
			return true
		}

		idleTime := now.Sub(conn.LastActivity())
		if idleTime > stats.MaxIdleTime {
			stats.MaxIdleTime = idleTime
		}

		if conn.ErrorCount() > 0 {
			stats.ConnectionsWithErrors++
		}

		if conn.IsClosed() {
			stats.ClosedConnections++
		} else {
			stats.ActiveConnections++
		}

		return true
	})

	return stats
}

// ConnectionStats holds statistics about connections.
type ConnectionStats struct {
	TotalConnections      int
	ActiveConnections     int
	ClosedConnections     int
	ConnectionsWithErrors int
	MaxIdleTime           time.Duration
}

// ForceCleanup forces an immediate cleanup of all idle connections.
func (rm *ResourceManager) ForceCleanup() {
	rm.cleanup()
}

// CleanupConnection cleans up a specific connection.
func (rm *ResourceManager) CleanupConnection(connID string) {
	rm.removeConnection(connID)
}
