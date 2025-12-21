// Package models provides data models for the OpsGenius Backend.
package models

import (
	"encoding/json"
	"time"
)

// Message type constants for client messages.
const (
	MessageTypeUserMessage          = "user_message"
	MessageTypeConfirmationResponse = "confirmation_response"
	MessageTypeHeartbeat            = "heartbeat"
)

// Message type constants for server messages.
const (
	MessageTypeAgentStream          = "agent_stream"
	MessageTypeAgentLog             = "agent_log"
	MessageTypeMCPStatus            = "mcp_status"
	MessageTypeConfirmationRequest  = "confirmation_request"
	MessageTypeChartData            = "chart_data"
	MessageTypeError                = "error"
)

// ClientMessage represents a message from the client.
type ClientMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// ServerMessage represents a message from the server.
type ServerMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp int64       `json:"timestamp"`
}

// UserMessagePayload represents the payload for a user message.
type UserMessagePayload struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

// ConfirmationResponsePayload represents the payload for a confirmation response.
type ConfirmationResponsePayload struct {
	RequestID string `json:"requestId"`
	Action    string `json:"action"` // "confirm" | "cancel"
}

// ConfirmationAction constants for confirmation actions.
const (
	ConfirmationActionConfirm = "confirm"
	ConfirmationActionCancel  = "cancel"
)

// HeartbeatPayload represents the payload for a heartbeat message.
type HeartbeatPayload struct{}

// AgentStreamPayload represents the payload for an agent stream message.
type AgentStreamPayload struct {
	SessionID string `json:"sessionId"`
	MessageID string `json:"messageId"`
	Content   string `json:"content"`
	Done      bool   `json:"done"`
}

// AgentLogPayload represents the payload for an agent log message.
type AgentLogPayload struct {
	Log LogEntry `json:"log"`
}

// MCPStatusPayload represents the payload for an MCP status message.
type MCPStatusPayload struct {
	ServerID string `json:"serverId"`
	Status   string `json:"status"` // "online" | "offline" | "error"
}

// ConfirmationRequestPayload represents the payload for a confirmation request.
type ConfirmationRequestPayload struct {
	Request ConfirmationRequest `json:"request"`
}

// ConfirmationRequest represents a confirmation request for high-risk operations.
type ConfirmationRequest struct {
	ID          string           `json:"id"`
	Operation   Operation        `json:"operation"`
	Description string           `json:"description"`
	RiskLevel   RiskLevel        `json:"riskLevel"`
	Timeout     time.Duration    `json:"timeout"`
	CreatedAt   time.Time        `json:"createdAt"`
}

// Operation represents an operation that requires confirmation.
type Operation struct {
	Type   string                 `json:"type"` // "restart" | "modify_config" | "delete"
	Target string                 `json:"target"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// RiskLevel represents the risk level of an operation.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// ConfirmationResponse represents a response to a confirmation request.
type ConfirmationResponse struct {
	RequestID string    `json:"requestId"`
	Action    Action    `json:"action"` // "confirmed" | "cancelled" | "timeout"
	Timestamp time.Time `json:"timestamp"`
}

// Action represents the action taken on a confirmation request.
type Action string

const (
	ActionConfirmed Action = "confirmed"
	ActionCancelled Action = "cancelled"
	ActionTimeout   Action = "timeout"
)

// ChartDataPayload represents the payload for chart data.
type ChartDataPayload struct {
	SessionID string    `json:"sessionId"`
	MessageID string    `json:"messageId"`
	ChartType string    `json:"chartType"` // "line" | "bar" | "pie"
	Data      ChartData `json:"data"`
}

// ChartType constants for chart types.
const (
	ChartTypeLine = "line"
	ChartTypeBar  = "bar"
	ChartTypePie  = "pie"
)

// ChartData represents chart data.
type ChartData struct {
	Labels   []string       `json:"labels"`
	Datasets []ChartDataset `json:"datasets"`
}

// ChartDataset represents a dataset in a chart.
type ChartDataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

// ErrorPayload represents the payload for an error message.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// LogEntry represents a log entry (used in AgentLogPayload).
type LogEntry struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"sessionId"`
	Timestamp int64                  `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	AgentRefs []AgentReference       `json:"agentRefs,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AgentReference represents a reference to an agent.
type AgentReference struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}
