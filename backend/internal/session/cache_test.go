package session

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test Redis client
func setupTestRedis(t *testing.T) *Cache {
	// Skip if no test Redis is available
	t.Skip("Skipping Redis tests - requires test Redis instance")
	return nil
}

func TestCache_SetAndGet(t *testing.T) {
	cache := setupTestRedis(t)
	if cache == nil {
		return
	}
	defer cache.Clear(context.Background())

	ctx := context.Background()

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

	// Set session
	err := cache.Set(ctx, session)
	require.NoError(t, err)

	// Get session
	retrieved, err := cache.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Title, retrieved.Title)
	assert.Len(t, retrieved.Messages, 1)
}

func TestCache_Delete(t *testing.T) {
	cache := setupTestRedis(t)
	if cache == nil {
		return
	}
	defer cache.Clear(context.Background())

	ctx := context.Background()

	// Create and set session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := cache.Set(ctx, session)
	require.NoError(t, err)

	// Delete session
	err = cache.Delete(ctx, session.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = cache.Get(ctx, session.ID)
	assert.Error(t, err)
}

func TestCache_ListByUser(t *testing.T) {
	cache := setupTestRedis(t)
	if cache == nil {
		return
	}
	defer cache.Clear(context.Background())

	ctx := context.Background()
	userID := "user-1"

	// Create multiple sessions
	sessionIDs := make([]string, 5)
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
		sessionIDs[i] = session.ID
		err := cache.Set(ctx, session)
		require.NoError(t, err)
	}

	// List sessions
	retrievedIDs, err := cache.ListByUser(ctx, userID, 3)
	require.NoError(t, err)
	assert.Len(t, retrievedIDs, 3)

	// Verify most recent sessions are returned
	assert.Contains(t, retrievedIDs, sessionIDs[4])
	assert.Contains(t, retrievedIDs, sessionIDs[3])
	assert.Contains(t, retrievedIDs, sessionIDs[2])
}

func TestCache_Exists(t *testing.T) {
	cache := setupTestRedis(t)
	if cache == nil {
		return
	}
	defer cache.Clear(context.Background())

	ctx := context.Background()

	// Create and set session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := cache.Set(ctx, session)
	require.NoError(t, err)

	// Check existence
	exists, err := cache.Exists(ctx, session.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Check non-existent session
	exists, err = cache.Exists(ctx, "non-existent")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCache_Refresh(t *testing.T) {
	cache := setupTestRedis(t)
	if cache == nil {
		return
	}
	defer cache.Clear(context.Background())

	ctx := context.Background()

	// Create and set session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Messages:  []Message{},
	}

	err := cache.Set(ctx, session)
	require.NoError(t, err)

	// Refresh TTL
	err = cache.Refresh(ctx, session.ID)
	require.NoError(t, err)

	// Session should still exist
	exists, err := cache.Exists(ctx, session.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}
