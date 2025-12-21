package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUnifiedStore_MemoryMode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	// Should start in memory mode when no backends are provided
	assert.Equal(t, ModeMemory, store.GetMode())

	// Create test session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages: []Message{
			{
				ID:        uuid.New().String(),
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
		},
	}

	// Save session
	err := store.Save(session)
	require.NoError(t, err)

	// Get session
	retrieved, err := store.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Len(t, retrieved.Messages, 1)
}

func TestUnifiedStore_Update(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	// Create and save session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Original Title",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := store.Save(session)
	require.NoError(t, err)

	// Update session
	session.Title = "Updated Title"
	err = store.Update(session)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
}

func TestUnifiedStore_Delete(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	// Create and save session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := store.Save(session)
	require.NoError(t, err)

	// Delete session
	err = store.Delete(session.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = store.Get(session.ID)
	assert.Error(t, err)
}

func TestUnifiedStore_ListByUser(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	userID := "user-1"

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &Session{
			ID:        uuid.New().String(),
			UserID:    userID,
			Title:     "Test Session",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now().Add(time.Duration(i) * time.Minute),
			ExpiresAt: time.Now().Add(time.Hour),
			Messages:  []Message{},
		}
		err := store.Save(session)
		require.NoError(t, err)
	}

	// List sessions
	sessions, err := store.ListByUser(userID, 3)
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Verify order (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		assert.True(t, sessions[i].UpdatedAt.After(sessions[i+1].UpdatedAt) ||
			sessions[i].UpdatedAt.Equal(sessions[i+1].UpdatedAt))
	}
}

func TestUnifiedStore_DeleteExpired(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	// Create expired session
	expiredSession := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Expired Session",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		Messages:  []Message{},
	}
	err := store.Save(expiredSession)
	require.NoError(t, err)

	// Create active session
	activeSession := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Active Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}
	err = store.Save(activeSession)
	require.NoError(t, err)

	// Delete expired sessions
	count, err := store.DeleteExpired()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify expired session is deleted
	_, err = store.Get(expiredSession.ID)
	assert.Error(t, err)

	// Verify active session still exists
	_, err = store.Get(activeSession.ID)
	assert.NoError(t, err)
}

func TestUnifiedStore_HealthCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := NewUnifiedStore(nil, nil, logger)

	health := store.HealthCheck()

	// Memory should always be healthy
	assert.True(t, health["memory"])

	// PostgreSQL and Redis should not be in health map if not configured
	// (or should be false if configured but not available)
}

func TestUnifiedStore_ModeDegradation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Start with no backends - should be in memory mode
	store := NewUnifiedStore(nil, nil, logger)
	assert.Equal(t, ModeMemory, store.GetMode())

	// Test that operations work in memory mode
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := store.Save(session)
	require.NoError(t, err)

	retrieved, err := store.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
}

func TestSession_ToJSON(t *testing.T) {
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages: []Message{
			{
				ID:        uuid.New().String(),
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
		},
	}

	json, err := session.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, json, session.ID)
	assert.Contains(t, json, session.UserID)
	assert.Contains(t, json, "Hello")
}
