// Package session provides session management functionality.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// StorageMode represents the current storage mode.
type StorageMode int

const (
	// ModePostgres uses PostgreSQL for storage.
	ModePostgres StorageMode = iota
	// ModeRedis uses Redis for storage.
	ModeRedis
	// ModeMemory uses in-memory storage.
	ModeMemory
)

func (m StorageMode) String() string {
	switch m {
	case ModePostgres:
		return "postgres"
	case ModeRedis:
		return "redis"
	case ModeMemory:
		return "memory"
	default:
		return "unknown"
	}
}

// UnifiedStore provides a unified session store with automatic degradation.
type UnifiedStore struct {
	pgStore    *Store
	cache      *Cache
	memory     map[string]*Session
	memoryLock sync.RWMutex
	mode       StorageMode
	modeLock   sync.RWMutex
	logger     *zap.Logger
}

// NewUnifiedStore creates a new unified session store.
func NewUnifiedStore(pgStore *Store, cache *Cache, logger *zap.Logger) *UnifiedStore {
	// Determine initial mode based on available backends
	mode := ModeMemory
	if pgStore != nil {
		mode = ModePostgres
	} else if cache != nil {
		mode = ModeRedis
	}

	return &UnifiedStore{
		pgStore: pgStore,
		cache:   cache,
		memory:  make(map[string]*Session),
		mode:    mode,
		logger:  logger,
	}
}

// GetMode returns the current storage mode.
func (u *UnifiedStore) GetMode() StorageMode {
	u.modeLock.RLock()
	defer u.modeLock.RUnlock()
	return u.mode
}

// setMode sets the storage mode.
func (u *UnifiedStore) setMode(mode StorageMode) {
	u.modeLock.Lock()
	defer u.modeLock.Unlock()
	if u.mode != mode {
		u.logger.Warn("Storage mode changed",
			zap.String("from", u.mode.String()),
			zap.String("to", mode.String()))
		u.mode = mode
	}
}

// Save saves a session with automatic degradation.
func (u *UnifiedStore) Save(session *Session) error {
	ctx := context.Background()

	// Try PostgreSQL first
	if u.GetMode() == ModePostgres && u.pgStore != nil {
		if err := u.pgStore.Save(session); err != nil {
			u.logger.Warn("PostgreSQL save failed, falling back to Redis",
				zap.Error(err),
				zap.String("sessionID", session.ID))
			u.setMode(ModeRedis)
		} else {
			// Also update cache for faster reads
			if u.cache != nil {
				_ = u.cache.Set(ctx, session)
			}
			return nil
		}
	}

	// Try Redis
	if u.GetMode() == ModeRedis && u.cache != nil {
		if err := u.cache.Set(ctx, session); err != nil {
			u.logger.Warn("Redis save failed, falling back to memory",
				zap.Error(err),
				zap.String("sessionID", session.ID))
			u.setMode(ModeMemory)
		} else {
			return nil
		}
	}

	// Fallback to memory
	u.memoryLock.Lock()
	defer u.memoryLock.Unlock()
	u.memory[session.ID] = session
	return nil
}

// Get retrieves a session with automatic degradation.
func (u *UnifiedStore) Get(sessionID string) (*Session, error) {
	ctx := context.Background()

	// Try cache first for performance
	if u.cache != nil {
		session, err := u.cache.Get(ctx, sessionID)
		if err == nil {
			return session, nil
		}
	}

	// Try PostgreSQL
	if u.GetMode() == ModePostgres && u.pgStore != nil {
		session, err := u.pgStore.Get(sessionID)
		if err != nil {
			u.logger.Warn("PostgreSQL get failed",
				zap.Error(err),
				zap.String("sessionID", sessionID))
		} else {
			// Update cache
			if u.cache != nil {
				_ = u.cache.Set(ctx, session)
			}
			return session, nil
		}
	}

	// Try Redis
	if u.GetMode() >= ModeRedis && u.cache != nil {
		session, err := u.cache.Get(ctx, sessionID)
		if err == nil {
			return session, nil
		}
		u.logger.Warn("Redis get failed",
			zap.Error(err),
			zap.String("sessionID", sessionID))
	}

	// Try memory
	u.memoryLock.RLock()
	defer u.memoryLock.RUnlock()
	session, ok := u.memory[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

// Update updates a session.
func (u *UnifiedStore) Update(session *Session) error {
	session.UpdatedAt = time.Now()
	return u.Save(session)
}

// Delete deletes a session.
func (u *UnifiedStore) Delete(sessionID string) error {
	ctx := context.Background()

	// Delete from all layers
	if u.pgStore != nil {
		_ = u.pgStore.Delete(sessionID)
	}

	if u.cache != nil {
		_ = u.cache.Delete(ctx, sessionID)
	}

	u.memoryLock.Lock()
	delete(u.memory, sessionID)
	u.memoryLock.Unlock()

	return nil
}

// ListByUser retrieves sessions for a specific user.
func (u *UnifiedStore) ListByUser(userID string, limit int) ([]*Session, error) {
	ctx := context.Background()

	// Try PostgreSQL first
	if u.GetMode() == ModePostgres && u.pgStore != nil {
		sessions, err := u.pgStore.ListByUser(userID, limit)
		if err != nil {
			u.logger.Warn("PostgreSQL list failed",
				zap.Error(err),
				zap.String("userID", userID))
		} else {
			return sessions, nil
		}
	}

	// Try Redis
	if u.GetMode() >= ModeRedis && u.cache != nil {
		sessionIDs, err := u.cache.ListByUser(ctx, userID, limit)
		if err != nil {
			u.logger.Warn("Redis list failed",
				zap.Error(err),
				zap.String("userID", userID))
		} else {
			// Fetch full sessions
			sessions := make([]*Session, 0, len(sessionIDs))
			for _, id := range sessionIDs {
				session, err := u.cache.Get(ctx, id)
				if err == nil {
					sessions = append(sessions, session)
				}
			}
			return sessions, nil
		}
	}

	// Fallback to memory
	u.memoryLock.RLock()
	defer u.memoryLock.RUnlock()

	sessions := make([]*Session, 0)
	for _, session := range u.memory {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}

	// Sort by updated time (most recent first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[i].UpdatedAt.Before(sessions[j].UpdatedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	// Apply limit
	if len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// DeleteExpired deletes expired sessions.
func (u *UnifiedStore) DeleteExpired() (int64, error) {
	var count int64

	// Delete from PostgreSQL
	if u.pgStore != nil {
		deleted, err := u.pgStore.DeleteExpired()
		if err == nil {
			count += deleted
		}
	}

	// Delete from memory
	u.memoryLock.Lock()
	now := time.Now()
	for id, session := range u.memory {
		if session.ExpiresAt.Before(now) {
			delete(u.memory, id)
			count++
		}
	}
	u.memoryLock.Unlock()

	return count, nil
}

// HealthCheck checks the health of all storage layers.
func (u *UnifiedStore) HealthCheck() map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health := make(map[string]bool)

	// Check PostgreSQL
	if u.pgStore != nil {
		err := u.pgStore.db.PingContext(ctx)
		health["postgres"] = err == nil
		if err != nil {
			u.logger.Warn("PostgreSQL health check failed", zap.Error(err))
		}
	}

	// Check Redis
	if u.cache != nil {
		err := u.cache.Ping(ctx)
		health["redis"] = err == nil
		if err != nil {
			u.logger.Warn("Redis health check failed", zap.Error(err))
		}
	}

	// Memory is always healthy
	health["memory"] = true

	return health
}

// RecoverMode attempts to recover to a higher storage mode.
func (u *UnifiedStore) RecoverMode() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	currentMode := u.GetMode()

	// Try to recover to PostgreSQL
	if currentMode != ModePostgres && u.pgStore != nil {
		if err := u.pgStore.db.PingContext(ctx); err == nil {
			u.setMode(ModePostgres)
			u.logger.Info("Recovered to PostgreSQL mode")
			return
		}
	}

	// Try to recover to Redis
	if currentMode == ModeMemory && u.cache != nil {
		if err := u.cache.Ping(ctx); err == nil {
			u.setMode(ModeRedis)
			u.logger.Info("Recovered to Redis mode")
			return
		}
	}
}

// Close closes all connections.
func (u *UnifiedStore) Close() error {
	var errs []error

	if u.pgStore != nil {
		if err := u.pgStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if u.cache != nil {
		if err := u.cache.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing store: %v", errs)
	}

	return nil
}
