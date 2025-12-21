// Package session provides session management functionality.
package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Helper function to create a test manager with in-memory storage
func newTestManager() *SessionManager {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	return NewManager(store, logger, 1*time.Hour)
}

// Test session creation
func TestManager_Create(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Create("user123")
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user123", session.UserID)
	assert.NotEmpty(t, session.Title)
	assert.Empty(t, session.Messages)
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.UpdatedAt.IsZero())
	assert.False(t, session.ExpiresAt.IsZero())
	assert.True(t, session.ExpiresAt.After(time.Now()))
}

// Test session creation with empty userID
func TestManager_Create_EmptyUserID(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Create("")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "userID cannot be empty")
}

// Test session retrieval
func TestManager_Get(t *testing.T) {
	manager := newTestManager()

	// Create a session
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Get the session
	retrieved, err := manager.Get(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.UserID, retrieved.UserID)
}

// Test session retrieval with invalid ID
func TestManager_Get_InvalidID(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Get("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, session)
}

// Test session retrieval with empty ID
func TestManager_Get_EmptyID(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Get("")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "sessionID cannot be empty")
}

// Test session expiration logic
func TestManager_Get_ExpiredSession(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	// Create manager with very short timeout
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Wait for session to expire
	time.Sleep(150 * time.Millisecond)

	// Try to get expired session
	retrieved, err := manager.Get(session.ID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Contains(t, err.Error(), "session expired")
}

// Test session update
func TestManager_Update(t *testing.T) {
	manager := newTestManager()

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	originalUpdatedAt := session.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Update the session
	session.Title = "Updated Title"
	err = manager.Update(session)
	require.NoError(t, err)

	// Get the updated session
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.True(t, retrieved.UpdatedAt.After(originalUpdatedAt))
}

// Test session update with nil session
func TestManager_Update_NilSession(t *testing.T) {
	manager := newTestManager()

	err := manager.Update(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session cannot be nil")
}

// Test session update with empty ID
func TestManager_Update_EmptyID(t *testing.T) {
	manager := newTestManager()

	session := &Session{
		ID:     "",
		UserID: "user123",
	}

	err := manager.Update(session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session ID cannot be empty")
}

// Test session deletion
func TestManager_Delete(t *testing.T) {
	manager := newTestManager()

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Delete the session
	err = manager.Delete(session.ID)
	require.NoError(t, err)

	// Try to get deleted session
	retrieved, err := manager.Get(session.ID)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

// Test session deletion with empty ID
func TestManager_Delete_EmptyID(t *testing.T) {
	manager := newTestManager()

	err := manager.Delete("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sessionID cannot be empty")
}

// Test listing sessions by user
func TestManager_ListByUser(t *testing.T) {
	manager := newTestManager()

	// Create multiple sessions for the same user
	session1, err := manager.Create("user123")
	require.NoError(t, err)

	session2, err := manager.Create("user123")
	require.NoError(t, err)

	// Create a session for a different user
	_, err = manager.Create("user456")
	require.NoError(t, err)

	// List sessions for user123
	sessions, err := manager.ListByUser("user123", 10)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Verify the sessions belong to the correct user
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		assert.Equal(t, "user123", s.UserID)
		sessionIDs[s.ID] = true
	}

	assert.True(t, sessionIDs[session1.ID])
	assert.True(t, sessionIDs[session2.ID])
}

// Test listing sessions with limit
func TestManager_ListByUser_WithLimit(t *testing.T) {
	manager := newTestManager()

	// Create 5 sessions
	for i := 0; i < 5; i++ {
		_, err := manager.Create("user123")
		require.NoError(t, err)
	}

	// List with limit of 3
	sessions, err := manager.ListByUser("user123", 3)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(sessions), 3)
}

// Test listing sessions with empty userID
func TestManager_ListByUser_EmptyUserID(t *testing.T) {
	manager := newTestManager()

	sessions, err := manager.ListByUser("", 10)
	assert.Error(t, err)
	assert.Nil(t, sessions)
	assert.Contains(t, err.Error(), "userID cannot be empty")
}

// Test adding message to session
func TestManager_AddMessage(t *testing.T) {
	manager := newTestManager()

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Add a message
	err = manager.AddMessage(session.ID, "user", "Hello, world!")
	require.NoError(t, err)

	// Get the session and verify message was added
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Messages, 1)
	assert.Equal(t, "user", retrieved.Messages[0].Role)
	assert.Equal(t, "Hello, world!", retrieved.Messages[0].Content)
	assert.NotEmpty(t, retrieved.Messages[0].ID)
	assert.False(t, retrieved.Messages[0].Timestamp.IsZero())
}

// Test adding multiple messages
func TestManager_AddMessage_Multiple(t *testing.T) {
	manager := newTestManager()

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Add multiple messages
	err = manager.AddMessage(session.ID, "user", "First message")
	require.NoError(t, err)

	err = manager.AddMessage(session.ID, "agent", "Second message")
	require.NoError(t, err)

	err = manager.AddMessage(session.ID, "user", "Third message")
	require.NoError(t, err)

	// Get the session and verify all messages
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.Len(t, retrieved.Messages, 3)
	assert.Equal(t, "First message", retrieved.Messages[0].Content)
	assert.Equal(t, "Second message", retrieved.Messages[1].Content)
	assert.Equal(t, "Third message", retrieved.Messages[2].Content)
}

// Test adding message with empty sessionID
func TestManager_AddMessage_EmptySessionID(t *testing.T) {
	manager := newTestManager()

	err := manager.AddMessage("", "user", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sessionID cannot be empty")
}

// Test adding message with empty role
func TestManager_AddMessage_EmptyRole(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Create("user123")
	require.NoError(t, err)

	err = manager.AddMessage(session.ID, "", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role cannot be empty")
}

// Test adding message with empty content
func TestManager_AddMessage_EmptyContent(t *testing.T) {
	manager := newTestManager()

	session, err := manager.Create("user123")
	require.NoError(t, err)

	err = manager.AddMessage(session.ID, "user", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

// Test checking if session is expired
func TestManager_IsExpired(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Check if not expired
	expired, err := manager.IsExpired(session.ID)
	require.NoError(t, err)
	assert.False(t, expired)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Check if expired
	expired, err = manager.IsExpired(session.ID)
	require.NoError(t, err)
	assert.True(t, expired)
}

// Test cleanup of expired sessions
func TestManager_CleanupExpired(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create multiple sessions
	session1, err := manager.Create("user123")
	require.NoError(t, err)

	session2, err := manager.Create("user456")
	require.NoError(t, err)

	// Wait for sessions to expire
	time.Sleep(150 * time.Millisecond)

	// Create a new session that won't expire
	session3, err := manager.Create("user789")
	require.NoError(t, err)

	// Cleanup expired sessions
	count, err := manager.CleanupExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Verify expired sessions are gone
	_, err = manager.store.Get(session1.ID)
	assert.Error(t, err)

	_, err = manager.store.Get(session2.ID)
	assert.Error(t, err)

	// Verify non-expired session still exists
	_, err = manager.store.Get(session3.ID)
	assert.NoError(t, err)
}

// Test GetOrCreate with new session
func TestManager_GetOrCreate_NewSession(t *testing.T) {
	manager := newTestManager()

	// Get or create with empty sessionID should create new
	session, err := manager.GetOrCreate("", "user123")
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user123", session.UserID)
}

// Test GetOrCreate with existing session
func TestManager_GetOrCreate_ExistingSession(t *testing.T) {
	manager := newTestManager()

	// Create a session
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Get or create with existing sessionID
	session, err := manager.GetOrCreate(created.ID, "user123")
	require.NoError(t, err)
	assert.Equal(t, created.ID, session.ID)
	assert.Equal(t, "user123", session.UserID)
}

// Test GetOrCreate with expired session
func TestManager_GetOrCreate_ExpiredSession(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Get or create should create new session
	session, err := manager.GetOrCreate(created.ID, "user123")
	require.NoError(t, err)
	assert.NotEqual(t, created.ID, session.ID) // Should be a new session
	assert.Equal(t, "user123", session.UserID)
}

// Test GetOrCreate with user mismatch
func TestManager_GetOrCreate_UserMismatch(t *testing.T) {
	manager := newTestManager()

	// Create a session for user123
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Try to get with different userID
	session, err := manager.GetOrCreate(created.ID, "user456")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session does not belong to user")
}

// Test session recovery after expiration refresh
func TestManager_SessionRecovery(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 200*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Wait half the timeout
	time.Sleep(100 * time.Millisecond)

	// Access the session (should refresh expiration)
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Wait another half timeout (original would have expired, but we refreshed)
	time.Sleep(100 * time.Millisecond)

	// Session should still be accessible
	retrieved, err = manager.Get(session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

// Test cleanup routine
func TestManager_StartCleanupRoutine(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	_, err := manager.Create("user123")
	require.NoError(t, err)

	// Start cleanup routine with short interval
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.StartCleanupRoutine(ctx, 200*time.Millisecond)

	// Wait for session to expire and cleanup to run
	time.Sleep(400 * time.Millisecond)

	// Verify cleanup happened (session should be gone)
	sessions, err := manager.ListByUser("user123", 10)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

// ============================================================================
// Additional Unit Tests for Session Expiration Logic (Requirement 8.4)
// ============================================================================

// Test that session expiration time is set correctly on creation
func TestManager_SessionExpirationOnCreation(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	timeout := 30 * time.Minute
	manager := NewManager(store, logger, timeout)

	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Verify expiration time is approximately timeout from now
	expectedExpiry := time.Now().Add(timeout)
	timeDiff := session.ExpiresAt.Sub(expectedExpiry)
	assert.Less(t, timeDiff.Abs(), 1*time.Second, "Expiration time should be close to timeout from creation")
}

// Test that expired sessions are filtered from ListByUser
func TestManager_ListByUser_FiltersExpiredSessions(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create multiple sessions
	session1, err := manager.Create("user123")
	require.NoError(t, err)

	session2, err := manager.Create("user123")
	require.NoError(t, err)

	// Wait for first two sessions to expire
	time.Sleep(150 * time.Millisecond)

	// Create a new session that won't expire
	session3, err := manager.Create("user123")
	require.NoError(t, err)

	// List sessions - should only return the non-expired one
	sessions, err := manager.ListByUser("user123", 10)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session3.ID, sessions[0].ID)

	// Verify expired sessions are not in the list
	for _, s := range sessions {
		assert.NotEqual(t, session1.ID, s.ID)
		assert.NotEqual(t, session2.ID, s.ID)
	}
}

// Test that Update refreshes session expiration
func TestManager_Update_RefreshesExpiration(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 200*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Update the session
	session.Title = "Updated"
	err = manager.Update(session)
	require.NoError(t, err)

	// Get the session and verify expiration was refreshed
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry), "Expiration should be refreshed after update")
}

// Test that AddMessage refreshes session expiration
func TestManager_AddMessage_RefreshesExpiration(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 200*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Add a message (which internally calls Update)
	err = manager.AddMessage(session.ID, "user", "Hello")
	require.NoError(t, err)

	// Get the session and verify expiration was refreshed
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry), "Expiration should be refreshed after adding message")
}

// Test that accessing expired session via AddMessage fails
func TestManager_AddMessage_ExpiredSession(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to add message to expired session
	err = manager.AddMessage(session.ID, "user", "Hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session expired")
}

// Test that IsExpired returns correct status before and after expiration
func TestManager_IsExpired_StatusTransition(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 150*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Check immediately - should not be expired
	expired, err := manager.IsExpired(session.ID)
	require.NoError(t, err)
	assert.False(t, expired, "Session should not be expired immediately after creation")

	// Wait half the timeout
	time.Sleep(75 * time.Millisecond)

	// Check again - should still not be expired
	expired, err = manager.IsExpired(session.ID)
	require.NoError(t, err)
	assert.False(t, expired, "Session should not be expired at half timeout")

	// Wait for full expiration
	time.Sleep(100 * time.Millisecond)

	// Check again - should now be expired
	expired, err = manager.IsExpired(session.ID)
	require.NoError(t, err)
	assert.True(t, expired, "Session should be expired after timeout")
}

// ============================================================================
// Additional Unit Tests for Session Recovery (Requirement 8.1)
// ============================================================================

// Test that Get refreshes expiration on access
func TestManager_Get_RefreshesExpiration(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 200*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Access the session
	retrieved, err := manager.Get(session.ID)
	require.NoError(t, err)

	// Verify expiration was refreshed
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry), "Expiration should be refreshed on access")
}

// Test session recovery with multiple accesses
func TestManager_SessionRecovery_MultipleAccesses(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 150*time.Millisecond)

	// Create a session
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Access the session multiple times, each time before expiration
	for i := 0; i < 5; i++ {
		time.Sleep(80 * time.Millisecond) // Wait most of the timeout

		// Access should refresh and keep session alive
		retrieved, err := manager.Get(session.ID)
		require.NoError(t, err, "Session should be accessible on access %d", i+1)
		assert.Equal(t, session.ID, retrieved.ID)
	}

	// Total time elapsed: 5 * 80ms = 400ms
	// Without refresh, session would have expired after 150ms
	// With refresh, session should still be alive
}

// Test GetOrCreate creates new session when sessionID is empty
func TestManager_GetOrCreate_EmptySessionID(t *testing.T) {
	manager := newTestManager()

	// Call with empty sessionID
	session, err := manager.GetOrCreate("", "user123")
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user123", session.UserID)
}

// Test GetOrCreate recovers existing valid session
func TestManager_GetOrCreate_RecoverValidSession(t *testing.T) {
	manager := newTestManager()

	// Create a session
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Add some messages to the session
	err = manager.AddMessage(created.ID, "user", "First message")
	require.NoError(t, err)

	err = manager.AddMessage(created.ID, "agent", "Second message")
	require.NoError(t, err)

	// Recover the session
	recovered, err := manager.GetOrCreate(created.ID, "user123")
	require.NoError(t, err)

	// Verify it's the same session with all data
	assert.Equal(t, created.ID, recovered.ID)
	assert.Equal(t, "user123", recovered.UserID)
	assert.Len(t, recovered.Messages, 2)
	assert.Equal(t, "First message", recovered.Messages[0].Content)
	assert.Equal(t, "Second message", recovered.Messages[1].Content)
}

// Test GetOrCreate creates new session when old one is expired
func TestManager_GetOrCreate_RecreateAfterExpiration(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager := NewManager(store, logger, 100*time.Millisecond)

	// Create a session
	created, err := manager.Create("user123")
	require.NoError(t, err)

	oldSessionID := created.ID

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Try to recover - should create new session
	recovered, err := manager.GetOrCreate(oldSessionID, "user123")
	require.NoError(t, err)
	assert.NotEqual(t, oldSessionID, recovered.ID, "Should create new session when old one expired")
	assert.Equal(t, "user123", recovered.UserID)
	assert.Empty(t, recovered.Messages, "New session should have no messages")
}

// Test GetOrCreate with nonexistent sessionID creates new session
func TestManager_GetOrCreate_NonexistentSession(t *testing.T) {
	manager := newTestManager()

	// Try to get nonexistent session
	session, err := manager.GetOrCreate("nonexistent-id", "user123")
	require.NoError(t, err)
	assert.NotEqual(t, "nonexistent-id", session.ID, "Should create new session")
	assert.Equal(t, "user123", session.UserID)
}

// Test GetOrCreate rejects session belonging to different user
func TestManager_GetOrCreate_DifferentUser(t *testing.T) {
	manager := newTestManager()

	// Create a session for user123
	created, err := manager.Create("user123")
	require.NoError(t, err)

	// Try to recover with different userID
	session, err := manager.GetOrCreate(created.ID, "user456")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "session does not belong to user")
}

// Test that session recovery maintains all session data
func TestManager_SessionRecovery_MaintainsData(t *testing.T) {
	manager := newTestManager()

	// Create a session with custom title
	session, err := manager.Create("user123")
	require.NoError(t, err)

	// Modify session data
	session.Title = "Custom Title"
	err = manager.Update(session)
	require.NoError(t, err)

	// Add messages
	err = manager.AddMessage(session.ID, "user", "Message 1")
	require.NoError(t, err)

	err = manager.AddMessage(session.ID, "agent", "Message 2")
	require.NoError(t, err)

	// Recover the session
	recovered, err := manager.Get(session.ID)
	require.NoError(t, err)

	// Verify all data is maintained
	assert.Equal(t, session.ID, recovered.ID)
	assert.Equal(t, "user123", recovered.UserID)
	assert.Equal(t, "Custom Title", recovered.Title)
	assert.Len(t, recovered.Messages, 2)
	assert.Equal(t, "Message 1", recovered.Messages[0].Content)
	assert.Equal(t, "Message 2", recovered.Messages[1].Content)
}

// Test session recovery after system restart (simulated by creating new manager)
func TestManager_SessionRecovery_AfterRestart(t *testing.T) {
	logger := zap.NewNop()
	store := NewUnifiedStore(nil, nil, logger)
	manager1 := NewManager(store, logger, 1*time.Hour)

	// Create a session with first manager
	session, err := manager1.Create("user123")
	require.NoError(t, err)

	err = manager1.AddMessage(session.ID, "user", "Test message")
	require.NoError(t, err)

	// Simulate restart by creating new manager with same store
	manager2 := NewManager(store, logger, 1*time.Hour)

	// Recover session with new manager
	recovered, err := manager2.Get(session.ID)
	require.NoError(t, err)

	// Verify session data is intact
	assert.Equal(t, session.ID, recovered.ID)
	assert.Equal(t, "user123", recovered.UserID)
	assert.Len(t, recovered.Messages, 1)
	assert.Equal(t, "Test message", recovered.Messages[0].Content)
}
