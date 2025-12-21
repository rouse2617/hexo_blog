package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	// Use in-memory SQLite for testing (requires sqlite driver)
	// For now, skip if no test database is available
	t.Skip("Skipping PostgreSQL tests - requires test database")
	return nil
}

func TestStore_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	store := NewStore(db)
	require.NoError(t, store.InitSchema())

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
			{
				ID:        uuid.New().String(),
				Role:      "agent",
				Content:   "Hi there!",
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
	assert.Equal(t, session.Title, retrieved.Title)
	assert.Len(t, retrieved.Messages, 2)
}

func TestStore_Update(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	store := NewStore(db)
	require.NoError(t, store.InitSchema())

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
	session.UpdatedAt = time.Now()
	err = store.Update(session)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
}

func TestStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	store := NewStore(db)
	require.NoError(t, store.InitSchema())

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

func TestStore_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	store := NewStore(db)
	require.NoError(t, store.InitSchema())

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

func TestStore_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	store := NewStore(db)
	require.NoError(t, store.InitSchema())

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
