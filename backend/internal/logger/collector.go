// Package logger provides log collection and reporting functionality.
package logger

import "time"

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// Collector interface defines log collection operations.
type Collector interface {
	Log(entry LogEntry) error
	GetLogs(sessionID string, limit int) ([]LogEntry, error)
	Subscribe(sessionID string) (<-chan LogEntry, error)
}

// LogEntry represents a log entry.
type LogEntry struct {
	ID        string
	SessionID string
	Timestamp time.Time
	Level     LogLevel
	Message   string
	AgentRefs []AgentReference
	Metadata  map[string]interface{}
}

// AgentReference represents a reference to an agent in a log entry.
type AgentReference struct {
	ID          string
	DisplayName string
	Type        string
}
