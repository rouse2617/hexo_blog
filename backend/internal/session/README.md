# Session Management Implementation

This package provides a comprehensive session management system with automatic degradation support.

## Features

### 1. PostgreSQL Store (`store.go`)
- Full CRUD operations for sessions
- Message persistence with foreign key relationships
- Automatic schema initialization
- Expired session cleanup
- Transaction support for data consistency

### 2. Redis Cache (`cache.go`)
- Fast session caching with TTL
- User session list management using sorted sets
- Session existence checks
- TTL refresh capabilities
- Automatic expiration handling

### 3. Unified Store with Degradation (`unified_store.go`)
- Automatic fallback: PostgreSQL → Redis → Memory
- Health checking for all storage layers
- Mode recovery attempts
- Transparent operation across all modes
- Thread-safe memory storage

## Architecture

```
┌─────────────────────────────────────────┐
│         Unified Store                   │
│  (Automatic Degradation Logic)          │
└─────────────────────────────────────────┘
           │         │         │
           ▼         ▼         ▼
    ┌──────────┐ ┌──────┐ ┌────────┐
    │PostgreSQL│ │ Redis│ │ Memory │
    │  Store   │ │Cache │ │  Map   │
    └──────────┘ └──────┘ └────────┘
```

## Usage

### Basic Setup

```go
import (
    "github.com/opsgenius/backend/internal/session"
    "go.uber.org/zap"
)

// Connect to PostgreSQL
pgDB, err := session.ConnectPostgres(
    "localhost", 5432, "opsgenius", "user", "password", 100)
if err != nil {
    log.Fatal(err)
}

pgStore := session.NewStore(pgDB)
err = pgStore.InitSchema()
if err != nil {
    log.Fatal(err)
}

// Connect to Redis
redisClient, err := session.ConnectRedis(
    "localhost", 6379, "", 0, 50)
if err != nil {
    log.Fatal(err)
}

cache := session.NewCache(redisClient, time.Hour)

// Create unified store
logger, _ := zap.NewProduction()
store := session.NewUnifiedStore(pgStore, cache, logger)
```

### Session Operations

```go
// Create a new session
session := &session.Session{
    ID:        uuid.New().String(),
    UserID:    "user-123",
    Title:     "My Session",
    CreatedAt: time.Now(),
    UpdatedAt: time.Now(),
    ExpiresAt: time.Now().Add(time.Hour),
    Messages:  []session.Message{},
}

// Save session
err := store.Save(session)

// Get session
retrieved, err := store.Get(session.ID)

// Update session
session.Title = "Updated Title"
err = store.Update(session)

// Delete session
err = store.Delete(session.ID)

// List user sessions
sessions, err := store.ListByUser("user-123", 10)

// Delete expired sessions
count, err := store.DeleteExpired()
```

### Health Monitoring

```go
// Check health of all storage layers
health := store.HealthCheck()
fmt.Printf("PostgreSQL: %v\n", health["postgres"])
fmt.Printf("Redis: %v\n", health["redis"])
fmt.Printf("Memory: %v\n", health["memory"])

// Get current storage mode
mode := store.GetMode()
fmt.Printf("Current mode: %s\n", mode)

// Attempt to recover to higher mode
store.RecoverMode()
```

## Degradation Strategy

The unified store automatically degrades when storage layers fail:

1. **Normal Operation**: Uses PostgreSQL for persistence, Redis for caching
2. **PostgreSQL Failure**: Falls back to Redis-only mode
3. **Redis Failure**: Falls back to in-memory mode
4. **Recovery**: Periodically attempts to recover to higher modes

### Example Degradation Flow

```
PostgreSQL fails → Log warning → Switch to Redis mode
Redis fails → Log warning → Switch to Memory mode
PostgreSQL recovers → Detect in health check → Switch back to PostgreSQL mode
```

## Database Schema

### Sessions Table
```sql
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
```

### Messages Table
```sql
CREATE TABLE messages (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
```

## Redis Keys

- Session data: `session:{sessionID}`
- User session list: `user:{userID}:sessions` (sorted set by update time)

## Testing

### Unit Tests
```bash
go test ./internal/session/...
```

### Property Tests
```bash
go test -v -run TestProperty ./internal/session/...
```

The property tests verify:
- Session round-trip consistency through memory store
- Session ID uniqueness
- Unified store round-trip in memory mode

## Configuration

### PostgreSQL Configuration
```yaml
database:
  host: "localhost"
  port: 5432
  database: "opsgenius"
  user: "postgres"
  password: "${DB_PASSWORD}"
  max_connections: 100
```

### Redis Configuration
```yaml
redis:
  host: "localhost"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 50
```

## Performance Considerations

1. **Connection Pooling**: Both PostgreSQL and Redis use connection pools
2. **Caching Strategy**: Redis cache reduces database load for frequent reads
3. **Batch Operations**: Use transactions for multiple related operations
4. **Index Usage**: Indexes on user_id, expires_at, and session_id for fast queries
5. **Memory Mode**: Suitable for development and testing, not for production

## Error Handling

All operations return errors that should be checked:

```go
session, err := store.Get(sessionID)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        // Handle not found case
    } else {
        // Handle other errors
    }
}
```

## Thread Safety

- **UnifiedStore**: Thread-safe with mutex protection for memory map
- **PostgreSQL Store**: Thread-safe via connection pool
- **Redis Cache**: Thread-safe via Redis client

## Migration

To initialize the database schema:

```bash
psql -U postgres -d opsgenius -f migrations/001_init_schema.sql
```

Or use the programmatic initialization:

```go
store := session.NewStore(db)
err := store.InitSchema()
```

## Requirements Validation

This implementation satisfies the following requirements:

- **8.1**: Session creation and recovery
- **8.2**: Message addition to session history
- **8.3**: Response addition to session history
- **8.4**: Session expiration after 1 hour of inactivity
- **8.5**: Session persistence to database
- **10.4**: Degradation strategy (PostgreSQL → Redis → Memory)

## Property Testing

The implementation includes property-based tests that verify:

- **Property 28**: Session data persistence round-trip consistency
  - For any session, saving and loading should produce an equivalent session
  - Tested with 100+ random sessions
  - Validates message content, timestamps, and metadata preservation
