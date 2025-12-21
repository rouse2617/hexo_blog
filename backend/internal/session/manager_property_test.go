// Package session provides session management functionality.
package session

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: ops-genius-backend, Property 26: 会话创建或恢复
// Validates: Requirements 8.1
func TestProperty_SessionCreateOrRecover(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any userID and optional sessionID, GetOrCreate should return a valid session", prop.ForAll(
		func(userID string, hasSessionID bool) bool {
			if userID == "" {
				return true // Skip empty userIDs
			}

			manager := newTestManager()

			var sessionID string
			if hasSessionID {
				// Create an existing session first
				existing, err := manager.Create(userID)
				if err != nil {
					return false
				}
				sessionID = existing.ID
			}

			// GetOrCreate should always return a valid session
			session, err := manager.GetOrCreate(sessionID, userID)
			if err != nil {
				return false
			}

			// Verify session properties
			if session == nil {
				return false
			}

			if session.ID == "" {
				return false
			}

			if session.UserID != userID {
				return false
			}

			if session.CreatedAt.IsZero() {
				return false
			}

			if session.UpdatedAt.IsZero() {
				return false
			}

			if session.ExpiresAt.IsZero() {
				return false
			}

			// ExpiresAt should be in the future
			if !session.ExpiresAt.After(time.Now()) {
				return false
			}

			// If we provided a sessionID, verify it matches (unless expired)
			if hasSessionID && sessionID != "" {
				// The returned session should either match the provided ID
				// or be a new session if the old one expired
				retrieved, err := manager.store.Get(sessionID)
				if err == nil && !time.Now().After(retrieved.ExpiresAt) {
					// If old session is still valid, IDs should match
					if session.ID != sessionID {
						return false
					}
				}
			}

			return true
		},
		gen.Identifier(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 27: 消息追加到会话历史
// Validates: Requirements 8.2, 8.3
func TestProperty_MessageAppendToHistory(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any session, adding messages should append them to history in order", prop.ForAll(
		func(userID string, messageCount uint8) bool {
			if userID == "" {
				return true // Skip empty userIDs
			}

			// Limit message count to reasonable number
			if messageCount > 20 {
				messageCount = 20
			}

			manager := newTestManager()

			// Create a session
			session, err := manager.Create(userID)
			if err != nil {
				return false
			}

			// Add messages
			messages := make([]struct {
				role    string
				content string
			}, messageCount)

			for i := uint8(0); i < messageCount; i++ {
				role := "user"
				if i%2 == 1 {
					role = "agent"
				}
				contentSample, ok := gen.Identifier().Sample()
				if !ok {
					return false
				}
				content := contentSample.(string)

				messages[i] = struct {
					role    string
					content string
				}{role, content}

				err := manager.AddMessage(session.ID, role, content)
				if err != nil {
					return false
				}
			}

			// Retrieve session and verify messages
			retrieved, err := manager.Get(session.ID)
			if err != nil {
				return false
			}

			// Verify message count
			if len(retrieved.Messages) != int(messageCount) {
				return false
			}

			// Verify messages are in order and match
			for i := 0; i < int(messageCount); i++ {
				if retrieved.Messages[i].Role != messages[i].role {
					return false
				}
				if retrieved.Messages[i].Content != messages[i].content {
					return false
				}
				if retrieved.Messages[i].ID == "" {
					return false
				}
				if retrieved.Messages[i].Timestamp.IsZero() {
					return false
				}

				// Verify timestamps are in order
				if i > 0 {
					if retrieved.Messages[i].Timestamp.Before(retrieved.Messages[i-1].Timestamp) {
						return false
					}
				}
			}

			return true
		},
		gen.Identifier(),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Session expiration is consistent
func TestProperty_SessionExpirationConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any session, expiration check should be consistent with ExpiresAt", prop.ForAll(
		func(userID string) bool {
			if userID == "" {
				return true
			}

			manager := newTestManager()

			// Create a session
			session, err := manager.Create(userID)
			if err != nil {
				return false
			}

			// Check expiration
			expired, err := manager.IsExpired(session.ID)
			if err != nil {
				return false
			}

			// Should not be expired immediately after creation
			if expired {
				return false
			}

			// Verify ExpiresAt is in the future
			if !session.ExpiresAt.After(time.Now()) {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Session updates preserve ID and UserID
func TestProperty_SessionUpdatePreservesIdentity(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any session, updates should preserve ID and UserID", prop.ForAll(
		func(userID string, newTitle string) bool {
			if userID == "" {
				return true
			}

			manager := newTestManager()

			// Create a session
			session, err := manager.Create(userID)
			if err != nil {
				return false
			}

			originalID := session.ID
			originalUserID := session.UserID

			// Update the session
			session.Title = newTitle
			err = manager.Update(session)
			if err != nil {
				return false
			}

			// Retrieve and verify
			retrieved, err := manager.Get(session.ID)
			if err != nil {
				return false
			}

			// ID and UserID should be unchanged
			if retrieved.ID != originalID {
				return false
			}

			if retrieved.UserID != originalUserID {
				return false
			}

			// Title should be updated
			if retrieved.Title != newTitle {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: ListByUser returns only sessions for that user
func TestProperty_ListByUserFiltersCorrectly(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any user, ListByUser should return only their sessions", prop.ForAll(
		func(userID string, otherUserID string, sessionCount uint8) bool {
			if userID == "" || otherUserID == "" || userID == otherUserID {
				return true
			}

			// Limit session count
			if sessionCount > 10 {
				sessionCount = 10
			}

			manager := newTestManager()

			// Create sessions for target user
			targetSessions := make(map[string]bool)
			for i := uint8(0); i < sessionCount; i++ {
				session, err := manager.Create(userID)
				if err != nil {
					return false
				}
				targetSessions[session.ID] = true
			}

			// Create sessions for other user
			for i := uint8(0); i < sessionCount; i++ {
				_, err := manager.Create(otherUserID)
				if err != nil {
					return false
				}
			}

			// List sessions for target user
			sessions, err := manager.ListByUser(userID, 100)
			if err != nil {
				return false
			}

			// Verify all returned sessions belong to target user
			for _, session := range sessions {
				if session.UserID != userID {
					return false
				}

				// Verify it's one of the sessions we created
				if !targetSessions[session.ID] {
					return false
				}
			}

			// Verify we got all the sessions we created
			if len(sessions) != int(sessionCount) {
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Identifier(),
		gen.UInt8Range(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Delete removes session completely
func TestProperty_DeleteRemovesSession(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any session, delete should make it inaccessible", prop.ForAll(
		func(userID string) bool {
			if userID == "" {
				return true
			}

			manager := newTestManager()

			// Create a session
			session, err := manager.Create(userID)
			if err != nil {
				return false
			}

			sessionID := session.ID

			// Verify session exists
			_, err = manager.Get(sessionID)
			if err != nil {
				return false
			}

			// Delete the session
			err = manager.Delete(sessionID)
			if err != nil {
				return false
			}

			// Verify session is gone
			_, err = manager.Get(sessionID)
			if err == nil {
				return false // Should return error
			}

			return true
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Session refresh extends expiration
func TestProperty_SessionRefreshExtendsExpiration(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any session, accessing it should extend expiration", prop.ForAll(
		func(userID string) bool {
			if userID == "" {
				return true
			}

			logger := zap.NewNop()
			store := NewUnifiedStore(nil, nil, logger)
			manager := NewManager(store, logger, 500*time.Millisecond)

			// Create a session
			session, err := manager.Create(userID)
			if err != nil {
				return false
			}

			originalExpiry := session.ExpiresAt

			// Wait a bit
			time.Sleep(100 * time.Millisecond)

			// Access the session (should refresh expiration)
			retrieved, err := manager.Get(session.ID)
			if err != nil {
				return false
			}

			// New expiration should be later than original
			if !retrieved.ExpiresAt.After(originalExpiry) {
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
