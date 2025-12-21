// Package session provides session management functionality.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides Redis-backed session caching.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCache creates a new Redis session cache.
func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

// ConnectRedis creates a new Redis connection.
func ConnectRedis(host string, port int, password string, db int, poolSize int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
		PoolSize: poolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// Set stores a session in the cache.
func (c *Cache) Set(ctx context.Context, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := c.sessionKey(session.ID)
	err = c.client.Set(ctx, key, data, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set session in cache: %w", err)
	}

	// Also add to user's session list
	userKey := c.userSessionsKey(session.UserID)
	err = c.client.ZAdd(ctx, userKey, redis.Z{
		Score:  float64(session.UpdatedAt.Unix()),
		Member: session.ID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add session to user list: %w", err)
	}

	// Set expiration on user's session list
	c.client.Expire(ctx, userKey, c.ttl)

	return nil
}

// Get retrieves a session from the cache.
func (c *Cache) Get(ctx context.Context, sessionID string) (*Session, error) {
	key := c.sessionKey(sessionID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found in cache: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session from cache: %w", err)
	}

	var session Session
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete removes a session from the cache.
func (c *Cache) Delete(ctx context.Context, sessionID string) error {
	// Get session to find user ID
	session, err := c.Get(ctx, sessionID)
	if err != nil {
		// If not found, still try to delete the key
		key := c.sessionKey(sessionID)
		c.client.Del(ctx, key)
		return nil
	}

	// Delete session key
	key := c.sessionKey(sessionID)
	err = c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session from cache: %w", err)
	}

	// Remove from user's session list
	userKey := c.userSessionsKey(session.UserID)
	err = c.client.ZRem(ctx, userKey, sessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove session from user list: %w", err)
	}

	return nil
}

// ListByUser retrieves session IDs for a specific user.
func (c *Cache) ListByUser(ctx context.Context, userID string, limit int) ([]string, error) {
	userKey := c.userSessionsKey(userID)

	// Get session IDs sorted by update time (descending)
	sessionIDs, err := c.client.ZRevRange(ctx, userKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list user sessions: %w", err)
	}

	return sessionIDs, nil
}

// Exists checks if a session exists in the cache.
func (c *Cache) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := c.sessionKey(sessionID)
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

// Refresh updates the TTL of a session in the cache.
func (c *Cache) Refresh(ctx context.Context, sessionID string) error {
	key := c.sessionKey(sessionID)
	err := c.client.Expire(ctx, key, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to refresh session TTL: %w", err)
	}

	return nil
}

// DeleteExpired removes expired sessions from user lists.
// Note: Redis automatically expires keys, but we need to clean up the sorted sets.
func (c *Cache) DeleteExpired(ctx context.Context, userID string) error {
	userKey := c.userSessionsKey(userID)
	now := time.Now().Unix()

	// Remove sessions older than TTL from the sorted set
	expiredBefore := now - int64(c.ttl.Seconds())
	err := c.client.ZRemRangeByScore(ctx, userKey, "-inf", fmt.Sprintf("%d", expiredBefore)).Err()
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return nil
}

// Clear removes all sessions from the cache (for testing).
func (c *Cache) Clear(ctx context.Context) error {
	// This is a dangerous operation, only use in tests
	return c.client.FlushDB(ctx).Err()
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// Helper functions for key generation
func (c *Cache) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func (c *Cache) userSessionsKey(userID string) string {
	return fmt.Sprintf("user:%s:sessions", userID)
}

// Ping checks if Redis is reachable.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
