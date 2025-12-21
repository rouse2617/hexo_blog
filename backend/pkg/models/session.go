// Package models provides data models for the OpsGenius Backend.
package models

import "time"

// Session represents a user session with conversation history.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Message represents a message in a session.
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "user" | "agent"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// MessageRole constants for message roles.
const (
	MessageRoleUser  = "user"
	MessageRoleAgent = "agent"
)

// SessionRecord represents a session record in the database.
type SessionRecord struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"userId"`
	Title     string    `db:"title" json:"title"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
	ExpiresAt time.Time `db:"expires_at" json:"expiresAt"`
}

// MessageRecord represents a message record in the database.
type MessageRecord struct {
	ID        string    `db:"id" json:"id"`
	SessionID string    `db:"session_id" json:"sessionId"`
	Role      string    `db:"role" json:"role"`
	Content   string    `db:"content" json:"content"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}

// LogRecord represents a log record in the database.
type LogRecord struct {
	ID        string    `db:"id" json:"id"`
	SessionID string    `db:"session_id" json:"sessionId"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	Level     string    `db:"level" json:"level"`
	Message   string    `db:"message" json:"message"`
	AgentRefs string    `db:"agent_refs" json:"agentRefs"` // JSON
	Metadata  string    `db:"metadata" json:"metadata"`    // JSON
}
