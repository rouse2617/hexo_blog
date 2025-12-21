package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/opsgenius/backend/pkg/models"
)

// Feature: ops-genius-backend, Property 2: 消息类型路由正确性
// *对于任何*客户端消息，消息应该根据其类型被路由到对应的处理器。
// **Validates: Requirements 1.2**
func TestProperty_MessageTypeRouting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Test that registered handlers are called for their message types
	properties.Property("registered handlers should be called for their message types", prop.ForAll(
		func(messageType string) bool {
			router := NewRouter()

			// Track if handler was called
			handlerCalled := false
			calledWithType := ""

			// Register a custom handler
			router.RegisterHandler(messageType, func(conn Connection, payload json.RawMessage) error {
				handlerCalled = true
				calledWithType = messageType
				return nil
			})

			// Create a mock connection
			mockConn := &mockConnection{id: "test-conn"}

			// Create a message with the registered type
			msg := models.ClientMessage{
				Type:      messageType,
				Payload:   json.RawMessage(`{}`),
				Timestamp: time.Now().Unix(),
			}
			msgBytes, _ := json.Marshal(msg)

			// Handle the message
			err := router.HandleMessage(mockConn, msgBytes)

			// Verify handler was called
			return err == nil && handlerCalled && calledWithType == messageType
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Test that unregistered message types return error
	properties.Property("unregistered message types should return error", prop.ForAll(
		func(messageType string) bool {
			router := NewRouter()

			// Unregister all default handlers
			router.UnregisterHandler(models.MessageTypeUserMessage)
			router.UnregisterHandler(models.MessageTypeConfirmationResponse)
			router.UnregisterHandler(models.MessageTypeHeartbeat)

			// Create a mock connection
			mockConn := &mockConnection{id: "test-conn"}

			// Create a message with an unregistered type
			msg := models.ClientMessage{
				Type:      messageType,
				Payload:   json.RawMessage(`{}`),
				Timestamp: time.Now().Unix(),
			}
			msgBytes, _ := json.Marshal(msg)

			// Handle the message - should send error but not return error
			err := router.HandleMessage(mockConn, msgBytes)

			// The router sends an error message but doesn't return an error
			// Check that an error message was sent
			return err == nil && len(mockConn.sentMessages) > 0
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Test that each default message type routes to correct handler
	properties.Property("default message types should route to correct handlers", prop.ForAll(
		func(typeIdx int) bool {
			messageTypes := []string{
				models.MessageTypeUserMessage,
				models.MessageTypeConfirmationResponse,
				models.MessageTypeHeartbeat,
			}
			messageType := messageTypes[typeIdx%len(messageTypes)]

			router := NewRouter()

			// Track which handler was called
			handlerCalled := ""

			// Override handlers to track calls
			router.RegisterHandler(models.MessageTypeUserMessage, func(conn Connection, payload json.RawMessage) error {
				handlerCalled = models.MessageTypeUserMessage
				return nil
			})
			router.RegisterHandler(models.MessageTypeConfirmationResponse, func(conn Connection, payload json.RawMessage) error {
				handlerCalled = models.MessageTypeConfirmationResponse
				return nil
			})
			router.RegisterHandler(models.MessageTypeHeartbeat, func(conn Connection, payload json.RawMessage) error {
				handlerCalled = models.MessageTypeHeartbeat
				return nil
			})

			// Create a mock connection
			mockConn := &mockConnection{id: "test-conn"}

			// Create appropriate payload based on message type
			var payload json.RawMessage
			switch messageType {
			case models.MessageTypeUserMessage:
				payload = json.RawMessage(`{"sessionId":"test","content":"hello"}`)
			case models.MessageTypeConfirmationResponse:
				payload = json.RawMessage(`{"requestId":"test","action":"confirm"}`)
			case models.MessageTypeHeartbeat:
				payload = json.RawMessage(`{}`)
			}

			// Create a message
			msg := models.ClientMessage{
				Type:      messageType,
				Payload:   payload,
				Timestamp: time.Now().Unix(),
			}
			msgBytes, _ := json.Marshal(msg)

			// Handle the message
			err := router.HandleMessage(mockConn, msgBytes)

			// Verify correct handler was called
			return err == nil && handlerCalled == messageType
		},
		gen.IntRange(0, 2),
	))

	// Test that GetHandler returns correct handler
	properties.Property("GetHandler should return registered handler", prop.ForAll(
		func(messageType string) bool {
			router := NewRouter()

			// Register a handler
			expectedHandler := func(conn Connection, payload json.RawMessage) error {
				return nil
			}
			router.RegisterHandler(messageType, expectedHandler)

			// Get the handler
			handler, ok := router.GetHandler(messageType)

			return ok && handler != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Test that UnregisterHandler removes handler
	properties.Property("UnregisterHandler should remove handler", prop.ForAll(
		func(messageType string) bool {
			router := NewRouter()

			// Register a handler
			router.RegisterHandler(messageType, func(conn Connection, payload json.RawMessage) error {
				return nil
			})

			// Verify it exists
			_, existsBefore := router.GetHandler(messageType)
			if !existsBefore {
				return false
			}

			// Unregister
			router.UnregisterHandler(messageType)

			// Verify it's gone
			_, existsAfter := router.GetHandler(messageType)
			return !existsAfter
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// mockConnection is a mock implementation of Connection for testing.
type mockConnection struct {
	id           string
	sentMessages [][]byte
	closed       bool
}

func (m *mockConnection) ID() string {
	return m.id
}

func (m *mockConnection) Send(message []byte) error {
	m.sentMessages = append(m.sentMessages, message)
	return nil
}

func (m *mockConnection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return m.Send(data)
}

func (m *mockConnection) Close() error {
	m.closed = true
	return nil
}

func (m *mockConnection) ReadMessage() ([]byte, error) {
	return nil, nil
}

func (m *mockConnection) SetMessageHandler(handler MessageHandler) {}

func (m *mockConnection) LastActivity() time.Time {
	return time.Now()
}

func (m *mockConnection) IncrementErrorCount() {}

func (m *mockConnection) ErrorCount() int {
	return 0
}

func (m *mockConnection) IsClosed() bool {
	return m.closed
}
