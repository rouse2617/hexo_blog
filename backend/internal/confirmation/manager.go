// Package confirmation provides confirmation management for high-risk operations.
package confirmation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Action represents the user's response to a confirmation request.
type Action string

const (
	ActionConfirmed Action = "confirmed"
	ActionCancelled Action = "cancelled"
	ActionTimeout   Action = "timeout"
)

// RiskLevel represents the risk level of an operation.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// Manager interface defines confirmation management operations.
type Manager interface {
	CreateRequest(ctx context.Context, operation Operation) (*Request, error)
	WaitForResponse(ctx context.Context, requestID string) (*Response, error)
	Confirm(requestID string) error
	Cancel(requestID string) error
}

// Request represents a confirmation request.
type Request struct {
	ID          string
	Operation   Operation
	Description string
	RiskLevel   RiskLevel
	Timeout     time.Duration
	CreatedAt   time.Time
}

// Operation represents an operation that requires confirmation.
type Operation struct {
	Type   string // restart, modify_config, delete
	Target string
	Params map[string]interface{}
}

// Response represents a confirmation response.
type Response struct {
	RequestID string
	Action    Action
	Timestamp time.Time
}

// DefaultManager implements the Manager interface.
type DefaultManager struct {
	mu              sync.RWMutex
	requests        map[string]*Request
	responseChan    map[string]chan *Response
	defaultTimeout  time.Duration
	sendRequestFunc func(*Request) error // Function to send request to frontend
}

// NewManager creates a new confirmation manager.
func NewManager(defaultTimeout time.Duration, sendRequestFunc func(*Request) error) *DefaultManager {
	if defaultTimeout == 0 {
		defaultTimeout = 30 * time.Second
	}
	return &DefaultManager{
		requests:        make(map[string]*Request),
		responseChan:    make(map[string]chan *Response),
		defaultTimeout:  defaultTimeout,
		sendRequestFunc: sendRequestFunc,
	}
}

// CreateRequest creates a new confirmation request and sends it to the frontend.
func (m *DefaultManager) CreateRequest(ctx context.Context, operation Operation) (*Request, error) {
	if operation.Type == "" {
		return nil, errors.New("operation type is required")
	}
	if operation.Target == "" {
		return nil, errors.New("operation target is required")
	}

	request := &Request{
		ID:          uuid.New().String(),
		Operation:   operation,
		Description: fmt.Sprintf("%s on %s", operation.Type, operation.Target),
		RiskLevel:   determineRiskLevel(operation.Type),
		Timeout:     m.defaultTimeout,
		CreatedAt:   time.Now(),
	}

	m.mu.Lock()
	m.requests[request.ID] = request
	m.responseChan[request.ID] = make(chan *Response, 1)
	m.mu.Unlock()

	// Send request to frontend
	if m.sendRequestFunc != nil {
		if err := m.sendRequestFunc(request); err != nil {
			m.mu.Lock()
			delete(m.requests, request.ID)
			delete(m.responseChan, request.ID)
			m.mu.Unlock()
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
	}

	// Start timeout timer
	go m.handleTimeout(request.ID, request.Timeout)

	return request, nil
}

// WaitForResponse waits for a user response to a confirmation request.
func (m *DefaultManager) WaitForResponse(ctx context.Context, requestID string) (*Response, error) {
	m.mu.RLock()
	respChan, exists := m.responseChan[requestID]
	m.mu.RUnlock()

	if !exists {
		return nil, errors.New("request not found")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		return resp, nil
	}
}

// Confirm confirms a request.
func (m *DefaultManager) Confirm(requestID string) error {
	return m.respond(requestID, ActionConfirmed)
}

// Cancel cancels a request.
func (m *DefaultManager) Cancel(requestID string) error {
	return m.respond(requestID, ActionCancelled)
}

// respond sends a response for a request.
func (m *DefaultManager) respond(requestID string, action Action) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	respChan, exists := m.responseChan[requestID]
	if !exists {
		return errors.New("request not found")
	}

	response := &Response{
		RequestID: requestID,
		Action:    action,
		Timestamp: time.Now(),
	}

	// Send response (non-blocking)
	select {
	case respChan <- response:
		// Clean up
		delete(m.requests, requestID)
		delete(m.responseChan, requestID)
		return nil
	default:
		return errors.New("response channel full")
	}
}

// handleTimeout handles request timeout.
func (m *DefaultManager) handleTimeout(requestID string, timeout time.Duration) {
	time.Sleep(timeout)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if request still exists (not already responded)
	respChan, exists := m.responseChan[requestID]
	if !exists {
		return
	}

	response := &Response{
		RequestID: requestID,
		Action:    ActionTimeout,
		Timestamp: time.Now(),
	}

	// Send timeout response (non-blocking)
	select {
	case respChan <- response:
		// Clean up
		delete(m.requests, requestID)
		delete(m.responseChan, requestID)
	default:
		// Channel already has a response
	}
}

// determineRiskLevel determines the risk level based on operation type.
func determineRiskLevel(operationType string) RiskLevel {
	switch operationType {
	case "delete", "destroy", "terminate":
		return RiskLevelHigh
	case "restart", "modify_config", "update":
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}
