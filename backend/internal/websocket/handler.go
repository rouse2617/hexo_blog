// Package websocket provides message handling for WebSocket connections.
package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/opsgenius/backend/pkg/models"
	"go.uber.org/zap"
)

// MessageType represents the type of a message.
type MessageType string

// HandlerFunc is a function that handles a specific message type.
type HandlerFunc func(conn Connection, payload json.RawMessage) error

// Router routes messages to appropriate handlers based on message type.
type Router struct {
	handlers map[string]HandlerFunc
	logger   *zap.Logger
	mu       sync.RWMutex

	// Callbacks for different message types
	onUserMessage          func(conn Connection, payload *models.UserMessagePayload) error
	onConfirmationResponse func(conn Connection, payload *models.ConfirmationResponsePayload) error
	onHeartbeat            func(conn Connection) error
}

// RouterOption is a function that configures a Router.
type RouterOption func(*Router)

// WithRouterLogger sets the logger for the router.
func WithRouterLogger(logger *zap.Logger) RouterOption {
	return func(r *Router) {
		r.logger = logger
	}
}

// WithUserMessageHandler sets the handler for user messages.
func WithUserMessageHandler(handler func(conn Connection, payload *models.UserMessagePayload) error) RouterOption {
	return func(r *Router) {
		r.onUserMessage = handler
	}
}

// WithConfirmationResponseHandler sets the handler for confirmation responses.
func WithConfirmationResponseHandler(handler func(conn Connection, payload *models.ConfirmationResponsePayload) error) RouterOption {
	return func(r *Router) {
		r.onConfirmationResponse = handler
	}
}

// WithHeartbeatHandler sets the handler for heartbeat messages.
func WithHeartbeatHandler(handler func(conn Connection) error) RouterOption {
	return func(r *Router) {
		r.onHeartbeat = handler
	}
}

// NewRouter creates a new message router.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{
		handlers: make(map[string]HandlerFunc),
	}

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	// Set default logger if not provided
	if r.logger == nil {
		r.logger, _ = zap.NewProduction()
	}

	// Register default handlers
	r.registerDefaultHandlers()

	return r
}

// registerDefaultHandlers registers the default message handlers.
func (r *Router) registerDefaultHandlers() {
	r.RegisterHandler(models.MessageTypeUserMessage, r.handleUserMessage)
	r.RegisterHandler(models.MessageTypeConfirmationResponse, r.handleConfirmationResponse)
	r.RegisterHandler(models.MessageTypeHeartbeat, r.handleHeartbeat)
}

// RegisterHandler registers a handler for a specific message type.
func (r *Router) RegisterHandler(messageType string, handler HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[messageType] = handler
}

// UnregisterHandler removes a handler for a specific message type.
func (r *Router) UnregisterHandler(messageType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, messageType)
}

// GetHandler returns the handler for a specific message type.
func (r *Router) GetHandler(messageType string) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[messageType]
	return handler, ok
}

// HandleMessage implements the MessageHandler interface.
func (r *Router) HandleMessage(conn Connection, message []byte) error {
	// Parse the message
	var clientMsg models.ClientMessage
	if err := json.Unmarshal(message, &clientMsg); err != nil {
		r.logger.Warn("Failed to parse client message",
			zap.String("connID", conn.ID()),
			zap.Error(err),
			zap.String("raw", string(message)))
		return r.sendError(conn, "INVALID_MESSAGE_FORMAT", "Failed to parse message")
	}

	// Validate message type
	if clientMsg.Type == "" {
		r.logger.Warn("Message type is empty",
			zap.String("connID", conn.ID()))
		return r.sendError(conn, "MISSING_MESSAGE_TYPE", "Message type is required")
	}

	// Get handler for message type
	handler, ok := r.GetHandler(clientMsg.Type)
	if !ok {
		r.logger.Warn("Unknown message type",
			zap.String("connID", conn.ID()),
			zap.String("type", clientMsg.Type))
		return r.sendError(conn, "UNKNOWN_MESSAGE_TYPE", fmt.Sprintf("Unknown message type: %s", clientMsg.Type))
	}

	// Execute handler
	if err := handler(conn, clientMsg.Payload); err != nil {
		r.logger.Error("Handler error",
			zap.String("connID", conn.ID()),
			zap.String("type", clientMsg.Type),
			zap.Error(err))
		return r.sendError(conn, "HANDLER_ERROR", err.Error())
	}

	return nil
}

// handleUserMessage handles user message payloads.
func (r *Router) handleUserMessage(conn Connection, payload json.RawMessage) error {
	var userMsg models.UserMessagePayload
	if err := json.Unmarshal(payload, &userMsg); err != nil {
		return fmt.Errorf("failed to parse user message payload: %w", err)
	}

	// Validate payload
	if userMsg.SessionID == "" {
		return fmt.Errorf("sessionId is required")
	}
	if userMsg.Content == "" {
		return fmt.Errorf("content is required")
	}

	r.logger.Debug("Received user message",
		zap.String("connID", conn.ID()),
		zap.String("sessionID", userMsg.SessionID),
		zap.Int("contentLength", len(userMsg.Content)))

	// Call custom handler if set
	if r.onUserMessage != nil {
		return r.onUserMessage(conn, &userMsg)
	}

	return nil
}

// handleConfirmationResponse handles confirmation response payloads.
func (r *Router) handleConfirmationResponse(conn Connection, payload json.RawMessage) error {
	var confirmResp models.ConfirmationResponsePayload
	if err := json.Unmarshal(payload, &confirmResp); err != nil {
		return fmt.Errorf("failed to parse confirmation response payload: %w", err)
	}

	// Validate payload
	if confirmResp.RequestID == "" {
		return fmt.Errorf("requestId is required")
	}
	if confirmResp.Action != models.ConfirmationActionConfirm && confirmResp.Action != models.ConfirmationActionCancel {
		return fmt.Errorf("action must be 'confirm' or 'cancel'")
	}

	r.logger.Debug("Received confirmation response",
		zap.String("connID", conn.ID()),
		zap.String("requestID", confirmResp.RequestID),
		zap.String("action", confirmResp.Action))

	// Call custom handler if set
	if r.onConfirmationResponse != nil {
		return r.onConfirmationResponse(conn, &confirmResp)
	}

	return nil
}

// handleHeartbeat handles heartbeat payloads.
func (r *Router) handleHeartbeat(conn Connection, payload json.RawMessage) error {
	r.logger.Debug("Received heartbeat",
		zap.String("connID", conn.ID()))

	// Call custom handler if set
	if r.onHeartbeat != nil {
		return r.onHeartbeat(conn)
	}

	// Send heartbeat response
	return conn.SendJSON(models.ServerMessage{
		Type:      models.MessageTypeHeartbeat,
		Payload:   models.HeartbeatPayload{},
		Timestamp: time.Now().Unix(),
	})
}

// sendError sends an error message to the client.
func (r *Router) sendError(conn Connection, code, message string) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeError,
		Payload: models.ErrorPayload{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
	})
}

// SendAgentStream sends an agent stream message to a connection.
func SendAgentStream(conn Connection, sessionID, messageID, content string, done bool) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeAgentStream,
		Payload: models.AgentStreamPayload{
			SessionID: sessionID,
			MessageID: messageID,
			Content:   content,
			Done:      done,
		},
		Timestamp: time.Now().Unix(),
	})
}

// SendMCPStatus sends an MCP status message to a connection.
func SendMCPStatus(conn Connection, serverID, status string) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeMCPStatus,
		Payload: models.MCPStatusPayload{
			ServerID: serverID,
			Status:   status,
		},
		Timestamp: time.Now().Unix(),
	})
}

// SendConfirmationRequest sends a confirmation request to a connection.
func SendConfirmationRequest(conn Connection, request models.ConfirmationRequest) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeConfirmationRequest,
		Payload: models.ConfirmationRequestPayload{
			Request: request,
		},
		Timestamp: time.Now().Unix(),
	})
}

// SendChartData sends chart data to a connection.
func SendChartData(conn Connection, sessionID, messageID, chartType string, data models.ChartData) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeChartData,
		Payload: models.ChartDataPayload{
			SessionID: sessionID,
			MessageID: messageID,
			ChartType: chartType,
			Data:      data,
		},
		Timestamp: time.Now().Unix(),
	})
}

// SendAgentLog sends an agent log message to a connection.
func SendAgentLog(conn Connection, log models.LogEntry) error {
	return conn.SendJSON(models.ServerMessage{
		Type: models.MessageTypeAgentLog,
		Payload: models.AgentLogPayload{
			Log: log,
		},
		Timestamp: time.Now().Unix(),
	})
}

// ListRegisteredHandlers returns a list of registered message types.
func (r *Router) ListRegisteredHandlers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}
