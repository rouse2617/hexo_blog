// Package models provides data models for the OpsGenius Backend.
package models

import "time"

// Log represents a log entry for agent activities.
type Log struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"sessionId"`
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Type      LogType                `json:"type"`
	Message   string                 `json:"message"`
	AgentRefs []LogAgentReference    `json:"agentRefs,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogLevel represents the severity level of a log.
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelDebug   LogLevel = "debug"
)

// LogAgentReference represents a reference to an agent in a log.
type LogAgentReference struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}

// LogType represents the type of log entry.
type LogType string

const (
	LogTypeTaskStart  LogType = "task_start"
	LogTypeToolCall   LogType = "tool_call"
	LogTypeToolResult LogType = "tool_result"
	LogTypeAnalysis   LogType = "analysis"
	LogTypeTaskEnd    LogType = "task_end"
	LogTypeError      LogType = "error"
)

// ToLogEntry converts a Log to a LogEntry for WebSocket transmission.
func (l *Log) ToLogEntry() LogEntry {
	agentRefs := make([]AgentReference, len(l.AgentRefs))
	for i, ref := range l.AgentRefs {
		agentRefs[i] = AgentReference{
			ID:          ref.ID,
			DisplayName: ref.DisplayName,
			Type:        ref.Type,
		}
	}
	return LogEntry{
		ID:        l.ID,
		SessionID: l.SessionID,
		Timestamp: l.Timestamp.Unix(),
		Level:     string(l.Level),
		Message:   l.Message,
		AgentRefs: agentRefs,
		Metadata:  l.Metadata,
	}
}
