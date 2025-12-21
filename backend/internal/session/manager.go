// Package session provides session management functionality.
package session

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Manager interface defines session management operations.
type Manager interface {
	Create(userID string) (*Session, error)
	Get(sessionID string) (*Session, error)
	Update(session *Session) error
	Delete(sessionID string) error
	ListByUser(userID string, limit int) ([]*Session, error)
	AddMessage(sessionID string, role string, content string) error
	IsExpired(sessionID string) (bool, error)
	CleanupExpired() (int64, error)
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Title     string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

// Message represents a chat message in a session.
type Message struct {
	ID        string
	Role      string // user, agent
	Content   string
	Timestamp time.Time
}

// SessionManager implements the Manager interface.
type SessionManager struct {
	store          *UnifiedStore
	logger         *zap.Logger
	sessionTimeout time.Duration
}

// NewManager creates a new session manager.
func NewManager(store *UnifiedStore, logger *zap.Logger, sessionTimeout time.Duration) *SessionManager {
	if sessionTimeout == 0 {
		sessionTimeout = 1 * time.Hour // Default 1 hour as per requirements
	}

	return &SessionManager{
		store:          store,
		logger:         logger,
		sessionTimeout: sessionTimeout,
	}
}

// Create creates a new session for a user.
func (m *SessionManager) Create(userID string) (*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	now := time.Now()
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     fmt.Sprintf("Session %s", now.Format("2006-01-02 15:04:05")),
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(m.sessionTimeout),
	}

	if err := m.store.Save(session); err != nil {
		m.logger.Error("Failed to create session",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.logger.Info("Session created",
		zap.String("sessionID", session.ID),
		zap.String("userID", userID))

	return session, nil
}

// Get retrieves a session by ID.
func (m *SessionManager) Get(sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	session, err := m.store.Get(sessionID)
	if err != nil {
		m.logger.Warn("Failed to get session",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		m.logger.Warn("Session expired",
			zap.String("sessionID", sessionID),
			zap.Time("expiresAt", session.ExpiresAt))
		return nil, fmt.Errorf("session expired: %s", sessionID)
	}

	// Refresh expiration time on access
	session.ExpiresAt = time.Now().Add(m.sessionTimeout)
	if err := m.store.Update(session); err != nil {
		m.logger.Warn("Failed to refresh session expiration",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		// Don't fail the Get operation if refresh fails
	}

	return session, nil
}

// Update updates an existing session.
func (m *SessionManager) Update(session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Update the UpdatedAt timestamp
	session.UpdatedAt = time.Now()

	// Refresh expiration time
	session.ExpiresAt = time.Now().Add(m.sessionTimeout)

	if err := m.store.Update(session); err != nil {
		m.logger.Error("Failed to update session",
			zap.String("sessionID", session.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update session: %w", err)
	}

	m.logger.Debug("Session updated",
		zap.String("sessionID", session.ID))

	return nil
}

// Delete deletes a session.
func (m *SessionManager) Delete(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	if err := m.store.Delete(sessionID); err != nil {
		m.logger.Error("Failed to delete session",
			zap.String("sessionID", sessionID),
			zap.Error(err))
		return fmt.Errorf("failed to delete session: %w", err)
	}

	m.logger.Info("Session deleted",
		zap.String("sessionID", sessionID))

	return nil
}

// ListByUser retrieves sessions for a specific user.
func (m *SessionManager) ListByUser(userID string, limit int) ([]*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	if limit <= 0 {
		limit = 10 // Default limit
	}

	sessions, err := m.store.ListByUser(userID, limit)
	if err != nil {
		m.logger.Error("Failed to list sessions",
			zap.String("userID", userID),
			zap.Int("limit", limit),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Filter out expired sessions
	validSessions := make([]*Session, 0, len(sessions))
	now := time.Now()
	for _, session := range sessions {
		if now.Before(session.ExpiresAt) {
			validSessions = append(validSessions, session)
		}
	}

	return validSessions, nil
}

// AddMessage adds a message to a session.
func (m *SessionManager) AddMessage(sessionID string, role string, content string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	if role == "" {
		return fmt.Errorf("role cannot be empty")
	}

	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Get the session
	session, err := m.Get(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Create new message
	message := Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add message to session
	session.Messages = append(session.Messages, message)

	// Update session
	if err := m.Update(session); err != nil {
		return fmt.Errorf("failed to update session with new message: %w", err)
	}

	m.logger.Debug("Message added to session",
		zap.String("sessionID", sessionID),
		zap.String("messageID", message.ID),
		zap.String("role", role))

	return nil
}

// IsExpired checks if a session is expired.
func (m *SessionManager) IsExpired(sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("sessionID cannot be empty")
	}

	session, err := m.store.Get(sessionID)
	if err != nil {
		return false, fmt.Errorf("failed to get session: %w", err)
	}

	return time.Now().After(session.ExpiresAt), nil
}

// CleanupExpired removes all expired sessions.
func (m *SessionManager) CleanupExpired() (int64, error) {
	count, err := m.store.DeleteExpired()
	if err != nil {
		m.logger.Error("Failed to cleanup expired sessions", zap.Error(err))
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	if count > 0 {
		m.logger.Info("Cleaned up expired sessions", zap.Int64("count", count))
	}

	return count, nil
}

// GetOrCreate gets an existing session or creates a new one if it doesn't exist or is expired.
func (m *SessionManager) GetOrCreate(sessionID string, userID string) (*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// If no sessionID provided, create new session
	if sessionID == "" {
		return m.Create(userID)
	}

	// Try to get existing session
	session, err := m.Get(sessionID)
	if err != nil {
		// If session not found or expired, create new one
		m.logger.Info("Session not found or expired, creating new session",
			zap.String("sessionID", sessionID),
			zap.String("userID", userID))
		return m.Create(userID)
	}

	// Verify the session belongs to the user
	if session.UserID != userID {
		m.logger.Warn("Session user mismatch",
			zap.String("sessionID", sessionID),
			zap.String("expectedUserID", userID),
			zap.String("actualUserID", session.UserID))
		return nil, fmt.Errorf("session does not belong to user")
	}

	return session, nil
}

// StartCleanupRoutine starts a background goroutine to periodically cleanup expired sessions.
func (m *SessionManager) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 10 * time.Minute // Default cleanup interval
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				m.logger.Info("Stopping session cleanup routine")
				return
			case <-ticker.C:
				count, err := m.CleanupExpired()
				if err != nil {
					m.logger.Error("Cleanup routine failed", zap.Error(err))
				} else if count > 0 {
					m.logger.Info("Cleanup routine completed", zap.Int64("deleted", count))
				}
			}
		}
	}()

	m.logger.Info("Session cleanup routine started", zap.Duration("interval", interval))
}
