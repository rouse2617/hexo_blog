// Package agent provides the AI agent engine for intent parsing and task orchestration.
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/opsgenius/backend/internal/confirmation"
	"github.com/opsgenius/backend/internal/mcp"
	"github.com/opsgenius/backend/internal/session"
	"github.com/opsgenius/backend/pkg/models"
	"go.uber.org/zap"
)

// Engine interface defines the Agent engine operations.
type Engine interface {
	ProcessMessage(ctx context.Context, sessionID string, message string) (<-chan Response, error)
	Stop() error
}

// IntentParser interface defines intent parsing operations.
type IntentParser interface {
	Parse(ctx context.Context, message string, history []Message) (*Intent, error)
}

// TaskOrchestrator interface defines task orchestration operations.
type TaskOrchestrator interface {
	Execute(ctx context.Context, intent *Intent) (<-chan TaskResult, error)
}

// IntentType represents the type of user intent.
type IntentType string

const (
	IntentTypeQuery   IntentType = "query"
	IntentTypeAnalyze IntentType = "analyze"
	IntentTypeOperate IntentType = "operate"
)

// Intent represents a parsed user intent.
type Intent struct {
	Type     IntentType             // query, analyze, operate
	Entities map[string]interface{} // node, service, timeRange, etc.
	Missing  []string               // missing slots
}

// ResponseType represents the type of agent response.
type ResponseType string

const (
	ResponseTypeText  ResponseType = "text"
	ResponseTypeChart ResponseType = "chart"
	ResponseTypeLog   ResponseType = "log"
	ResponseTypeError ResponseType = "error"
	ResponseTypeDone  ResponseType = "done"
)

// Response represents an agent response.
type Response struct {
	Type    ResponseType
	Content interface{}
}

// Message represents a chat message.
type Message struct {
	ID        string
	Role      string // user, agent
	Content   string
	Timestamp int64
}

// TaskResult represents the result of a task execution.
type TaskResult struct {
	Success bool
	Data    interface{}
	Error   string
}

// AgentEngine implements the Engine interface.
type AgentEngine struct {
	sessionManager      session.Manager
	mcpManager          mcp.Manager
	confirmationManager confirmation.Manager
	intentParser        IntentParser
	taskOrchestrator    TaskOrchestrator
	logger              *zap.Logger
	
	mu      sync.RWMutex
	stopped bool
}

// EngineConfig holds configuration for the agent engine.
type EngineConfig struct {
	SessionManager      session.Manager
	MCPManager          mcp.Manager
	ConfirmationManager confirmation.Manager
	IntentParser        IntentParser
	TaskOrchestrator    TaskOrchestrator
	Logger              *zap.Logger
}

// NewEngine creates a new agent engine.
func NewEngine(config EngineConfig) (*AgentEngine, error) {
	if config.SessionManager == nil {
		return nil, fmt.Errorf("session manager is required")
	}
	if config.MCPManager == nil {
		return nil, fmt.Errorf("MCP manager is required")
	}
	if config.ConfirmationManager == nil {
		return nil, fmt.Errorf("confirmation manager is required")
	}
	
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	
	// Intent parser and task orchestrator are optional for now (can be added later)
	// For now, we'll use simple implementations
	
	return &AgentEngine{
		sessionManager:      config.SessionManager,
		mcpManager:          config.MCPManager,
		confirmationManager: config.ConfirmationManager,
		intentParser:        config.IntentParser,
		taskOrchestrator:    config.TaskOrchestrator,
		logger:              config.Logger,
		stopped:             false,
	}, nil
}

// ProcessMessage processes a user message and returns a channel of streaming responses.
// Requirements: 6.1, 6.2, 6.3, 6.4
func (e *AgentEngine) ProcessMessage(ctx context.Context, sessionID string, message string) (<-chan Response, error) {
	e.mu.RLock()
	if e.stopped {
		e.mu.RUnlock()
		return nil, fmt.Errorf("engine is stopped")
	}
	e.mu.RUnlock()
	
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}
	
	// Create response channel
	responseChan := make(chan Response, 10)
	
	// Start processing in a goroutine
	go func() {
		defer close(responseChan)
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("Panic in ProcessMessage",
					zap.Any("panic", r),
					zap.String("sessionID", sessionID))
				
				// Send error response
				select {
				case responseChan <- Response{
					Type:    ResponseTypeError,
					Content: fmt.Sprintf("Internal error: %v", r),
				}:
				case <-ctx.Done():
				}
			}
		}()
		
		// Add user message to session
		if err := e.sessionManager.AddMessage(sessionID, "user", message); err != nil {
			e.logger.Error("Failed to add user message to session",
				zap.String("sessionID", sessionID),
				zap.Error(err))
			
			select {
			case responseChan <- Response{
				Type:    ResponseTypeError,
				Content: "Failed to save message",
			}:
			case <-ctx.Done():
				return
			}
			return
		}
		
		// Process the message and generate streaming responses
		if err := e.processAndStream(ctx, sessionID, message, responseChan); err != nil {
			e.logger.Error("Failed to process message",
				zap.String("sessionID", sessionID),
				zap.Error(err))
			
			select {
			case responseChan <- Response{
				Type:    ResponseTypeError,
				Content: err.Error(),
			}:
			case <-ctx.Done():
			}
		}
		
		// Always send done marker as final response
		select {
		case responseChan <- Response{
			Type:    ResponseTypeDone,
			Content: nil,
		}:
		case <-ctx.Done():
		}
	}()
	
	return responseChan, nil
}

// processAndStream processes the message and streams responses.
func (e *AgentEngine) processAndStream(ctx context.Context, sessionID string, message string, responseChan chan<- Response) error {
	// Get session for history
	sess, err := e.sessionManager.Get(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	
	// Convert session messages to agent messages
	history := make([]Message, len(sess.Messages))
	for i, msg := range sess.Messages {
		history[i] = Message{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp.Unix(),
		}
	}
	
	// Parse intent (if parser is available)
	var intent *Intent
	if e.intentParser != nil {
		intent, err = e.intentParser.Parse(ctx, message, history)
		if err != nil {
			e.logger.Warn("Failed to parse intent, using default",
				zap.Error(err))
			// Use default intent
			intent = &Intent{
				Type:     IntentTypeQuery,
				Entities: make(map[string]interface{}),
				Missing:  []string{},
			}
		}
	} else {
		// Default intent if no parser
		intent = &Intent{
			Type:     IntentTypeQuery,
			Entities: make(map[string]interface{}),
			Missing:  []string{},
		}
	}
	
	// Check for missing slots
	if len(intent.Missing) > 0 {
		// Generate clarification question
		clarification := fmt.Sprintf("I need more information: %v", intent.Missing)
		
		select {
		case responseChan <- Response{
			Type:    ResponseTypeText,
			Content: clarification,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
		
		return nil
	}
	
	// Execute task orchestration (if orchestrator is available)
	if e.taskOrchestrator != nil {
		taskResultChan, err := e.taskOrchestrator.Execute(ctx, intent)
		if err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}
		
		// Stream task results
		for taskResult := range taskResultChan {
			if err := e.handleTaskResult(ctx, sessionID, taskResult, responseChan); err != nil {
				return err
			}
		}
	} else {
		// Simple response generation without orchestrator
		if err := e.generateSimpleResponse(ctx, sessionID, message, intent, responseChan); err != nil {
			return err
		}
	}
	
	return nil
}

// handleTaskResult processes a task result and sends appropriate responses.
func (e *AgentEngine) handleTaskResult(ctx context.Context, sessionID string, result TaskResult, responseChan chan<- Response) error {
	if !result.Success {
		select {
		case responseChan <- Response{
			Type:    ResponseTypeError,
			Content: result.Error,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}
	
	// Check if result contains tool call data
	if toolData, ok := result.Data.(map[string]interface{}); ok {
		if toolName, hasToolName := toolData["tool"]; hasToolName {
			// This is a tool result - embed it in the response
			select {
			case responseChan <- Response{
				Type: ResponseTypeText,
				Content: map[string]interface{}{
					"text":       fmt.Sprintf("Tool %v executed successfully", toolName),
					"toolResult": toolData,
				},
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}
		
		// Check if result contains chart data
		if chartData, hasChart := toolData["chart"]; hasChart {
			if chart, ok := chartData.(models.ChartData); ok {
				// Send chart as separate response
				select {
				case responseChan <- Response{
					Type:    ResponseTypeChart,
					Content: chart,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}
		}
	}
	
	// Regular text response
	select {
	case responseChan <- Response{
		Type:    ResponseTypeText,
		Content: fmt.Sprintf("%v", result.Data),
	}:
	case <-ctx.Done():
		return ctx.Err()
	}
	
	return nil
}

// generateSimpleResponse generates a simple response without task orchestration.
func (e *AgentEngine) generateSimpleResponse(ctx context.Context, sessionID string, message string, intent *Intent, responseChan chan<- Response) error {
	// Generate a simple streaming response
	messageID := uuid.New().String()
	
	// Simulate streaming by sending response in chunks
	responseText := fmt.Sprintf("Processing your %s request", intent.Type)
	
	// Send first chunk
	select {
	case responseChan <- Response{
		Type:    ResponseTypeText,
		Content: responseText[:len(responseText)/2],
	}:
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Small delay to simulate streaming
	time.Sleep(10 * time.Millisecond)
	
	// Send second chunk
	select {
	case responseChan <- Response{
		Type:    ResponseTypeText,
		Content: responseText[len(responseText)/2:],
	}:
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Save agent response to session
	fullResponse := responseText
	if err := e.sessionManager.AddMessage(sessionID, "agent", fullResponse); err != nil {
		e.logger.Error("Failed to add agent message to session",
			zap.String("sessionID", sessionID),
			zap.String("messageID", messageID),
			zap.Error(err))
	}
	
	return nil
}

// Stop stops the agent engine.
func (e *AgentEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.stopped {
		return nil
	}
	
	e.stopped = true
	e.logger.Info("Agent engine stopped")
	
	return nil
}
