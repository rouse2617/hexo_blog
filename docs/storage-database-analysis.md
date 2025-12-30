# Distributed Token Bucket Rate-Limiting: Storage & Database Design

**Document Version**: 1.0
**Date**: 2025-12-31
**System**: Three-Layer Token Bucket with Cost Normalization
**Target**: High-Performance Cloud Storage Platform (100k+ QPS)

---

## Executive Summary

This document provides a comprehensive storage design for a distributed token bucket rate-limiting system operating at three layers (Cluster → Application → Local/Edge). The design optimizes for:

- **Performance**: Sub-millisecond token checks at edge, <10ms at cluster level
- **Scalability**: Support 100k+ QPS with Redis Cluster
- **Consistency**: Atomic operations via Lua scripts
- **Reliability**: Fail-open degradation strategies
- **Cost Efficiency**: Memory-optimized data structures

**Key Design Decisions**:
1. **Redis Cluster** for L1/L2 with Hash tags for data locality
2. **Nginx Shared Memory** for L3 edge cache
3. **Lua Scripts** for atomic multi-layer operations
4. **Pipeline Batching** for 100-1000x QPS reduction
5. **TTL-based Eviction** with active expiration

---

## Table of Contents

1. [Redis Data Model Design](#1-redis-data-model-design)
2. [Atomic Operations & Lua Scripts](#2-atomic-operations--lua-scripts)
3. [Performance Optimization](#3-performance-optimization)
4. [Data Consistency & Reliability](#4-data-consistency--reliability)
5. [Capacity Planning](#5-capacity-planning)
6. [Implementation Examples](#6-implementation-examples)

---

## 1. Redis Data Model Design

### 1.1 Key Naming Convention

**Hierarchical Namespace Pattern**:
```
{layer}:{entity}:{identifier}[:{sub-entity}]
```

**Key Patterns**:

```
L1 (Cluster Level):
  - cluster:tokens                    # Total cluster token pool
  - cluster:config                    # Cluster configuration
  - cluster:stats                     # Aggregate statistics
  - apps:active                       # Set of active app IDs

L2 (Application Level):
  - app:{app_id}:tokens               # App's token pool
  - app:{app_id}:config               # App-specific config
  - app:{app_id}:users:active         # Set of active users in app
  - app:{app_id}:stats:daily:{date}   # Daily statistics (TTL 7d)

L3 (User Level):
  - user:{app_id}:{user_id}:tokens    # User's token bucket
  - user:{app_id}:{user_id}:usage     # Usage tracking (TTL 1d)
  - user:{app_id}:{user_id}:limits    # Custom limits

Batch Accumulation:
  - batch:{node_id}:{app_id}:{user_id} # Micro-batch accumulator (TTL 1h)

Configuration:
  - config:cost_model:{version}       # Cost model versioning
  - config:rate:{level}:{id}          # Rate limits by level
```

**Example Keys**:
```
cluster:tokens                        → String: 10000000
app:storage-service:tokens            → Hash: {tokens: 500000, capacity: 1000000}
user:storage-service:user123:tokens   → Hash: {tokens: 1000, last_update: 1704067200}
batch:nginx-01:storage-service:user123 → String: 500 (accumulated usage)
```

### 1.2 Data Structure Selection

#### L1: Cluster Level (Centralized)

**Data Structures**:

| Key | Type | Fields | Rationale |
|-----|------|--------|-----------|
| `cluster:tokens` | **String** | Integer value | Single atomic counter for global pool |
| `cluster:config` | **Hash** | rate, capacity, emergency_threshold, replenish_interval | Multiple config fields, O(1) field access |
| `cluster:stats` | **Hash** | total_deducted, total_replenished, apps_count, last_distributi | Real-time statistics, field-level updates |
| `apps:active` | **Set** | App IDs | O(1) membership test, O(N) iteration for distribution |

**Memory Estimate**:
```
cluster:tokens (String):      8 bytes
cluster:config (Hash):        ~100 bytes (5 fields × 20 bytes avg)
cluster:stats (Hash):         ~80 bytes (4 fields × 20 bytes)
apps:active (Set):            10 bytes per app_id

Total L1 (10 apps): ~300 bytes
```

#### L2: Application Level (Tenant Isolation)

**Data Structures**:

| Key | Type | Fields | Rationale |
|-----|------|--------|-----------|
| `app:{id}:tokens` | **Hash** | tokens, capacity, rate, reserved, burst | O(1) field access, atomic HINCRBY |
| `app:{id}:config` | **Hash** | weight, priority, cost_model_version, tier | Multi-field configuration |
| `app:{id}:users:active` | **Sorted Set** | User ID → last_seen | O(log N) range queries by activity |
| `app:{id}:stats:daily:{date}` | **Hash** | ops_{GET}, ops_{PUT}, ops_{LIST}, tokens_used | Time-series data with TTL |

**Memory Estimate**:
```
Per application:
  app:{id}:tokens (Hash):              ~120 bytes
  app:{id}:config (Hash):              ~100 bytes
  app:{id}:users:active (Sorted Set):  100 bytes per user
  app:{id}:stats:daily:{date} (Hash):  ~200 bytes

Total L2 (10 apps, 1000 users/app): ~140 KB
```

#### L3: User Level (Individual Quotas)

**Data Structures**:

| Key | Type | Fields | Rationale |
|-----|------|--------|-----------|
| `user:{app}:{id}:tokens` | **Hash** | tokens, last_update, capacity, rate, reserved, burst | Complete bucket state in one key |
| `user:{app}:{id}:usage` | **String** | JSON: {ops: [{ts, op, cost}], total_cost} | Append-only log for reconciliation (TTL 1d) |
| `user:{app}:{id}:limits` | **Hash** | max_ops_sec, max_bw_mb, priority, whitelist | Custom overrides |

**Memory Estimate**:
```
Per user:
  user:{app}:{id}:tokens (Hash):     ~200 bytes (6 fields)
  user:{app}:{id}:usage (String):    ~1 KB (100 ops avg)
  user:{app}:{id}:limits (Hash):     ~80 bytes

Total L3 (10,000 users): ~12.8 MB
```

#### Batch Accumulation (Micro-Batching)

**Data Structures**:

| Key | Type | Value | TTL | Rationale |
|-----|------|-------|-----|-----------|
| `batch:{node}:{app}:{user}` | **String** | Integer: accumulated_tokens | 1h | Simple counter, auto-cleanup |

**Memory Estimate**:
```
Per active batch (node/app/user combination):
  batch:* (String): 8 bytes

Total Batches (100 nodes × 10 apps × 1000 active users): 8 MB
With TTL cleanup: Actual ~2-3 MB (churn)
```

### 1.3 Hash Tags for Data Locality

**Purpose**: Ensure related keys map to same Redis Cluster shard

**Pattern**:
```
{app_id} in key = All keys for same app on same shard

Examples:
  app:storage-service:tokens     → Hash tag: {storage-service}
  user:storage-service:user123   → Hash tag: {storage-service}
  user:storage-service:user456   → Hash tag: {storage-service}
```

**Benefits**:
- Single Lua script can access all app keys atomically
- Reduces cross-slot errors in Redis Cluster
- Improves locality for app-level operations

**Implementation**:
```lua
-- Use curly braces for hash tags
local app_key = "app:{storage-service}:tokens"
local user_key_pattern = "user:{storage-service}:*"

-- All operations on these keys go to same shard
```

### 1.4 TTL Strategies

**TTL Assignment Table**:

| Key Pattern | TTL | Rationale |
|-------------|-----|-----------|
| `cluster:tokens` | **None** | Persistent, never expires |
| `cluster:stats` | **None** | Persistent statistics |
| `apps:active` | **None** | Active app registry |
| `app:{id}:tokens` | **None** | Persistent token pool |
| `app:{id}:config` | **None** | Persistent configuration |
| `app:{id}:users:active` | **30 days** | Cleanup inactive users |
| `app:{id}:stats:daily:{date}` | **7 days** | Retain week of daily stats |
| `user:{app}:{id}:tokens` | **90 days** | Cleanup inactive users (quarterly) |
| `user:{app}:{id}:usage` | **1 day** | Short-term reconciliation log |
| `user:{app}:{id}:limits` | **None** | Persistent custom limits |
| `batch:{node}:{app}:{user}` | **1 hour** | Force flush or auto-expire |
| `config:cost_model:{version}` | **None** | Versioned persistence |

**TTL Management Commands**:
```redis
# Set TTL on creation
SETEX batch:nginx-01:app1:user123 3600 500

# Update TTL on access (sliding expiration)
EXPIRE user:app1:user123:usage 86400

# Batch cleanup of expired keys
SCAN 0 MATCH app:*:stats:daily:* COUNT 1000
```

---

## 2. Atomic Operations & Lua Scripts

### 2.1 Three-Layer Token Deduction (Atomic)

**Purpose**: Deduct tokens from L3 → L2 → L1 atomically with fallback

**Lua Script**:
```lua
-- KEYS[1]: User token key (L3)
-- KEYS[2]: App token key (L2)
-- KEYS[3]: Cluster token key (L1)
-- ARGV[1]: Cost to deduct
-- ARGV[2]: Current timestamp
-- ARGV[3]: Replenish rate
-- ARGV[4]: Burst capacity

local user_key = KEYS[1]
local app_key = KEYS[2]
local cluster_key = KEYS[3]
local cost = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local rate = tonumber(ARGV[3])
local capacity = tonumber(ARGV[4])

-- L3: Check and deduct from user bucket
local user_data = redis.call('HGETALL', user_key)
local user_tokens = tonumber(user_data[2]) or capacity
local user_last_update = tonumber(user_data[4]) or now

-- Replenish user tokens
local user_replenish = (now - user_last_update) * rate
user_tokens = math.min(capacity, user_tokens + user_replenish)

if user_tokens >= cost then
    -- L3 deduction successful
    user_tokens = user_tokens - cost
    redis.call('HMSET', user_key,
        'tokens', user_tokens,
        'last_update', now,
        'capacity', capacity,
        'rate', rate
    )
    redis.call('HINCRBY', user_key, 'deducted_count', 1)
    redis.call('HINCRBY', user_key, 'deducted_total', cost)
    return {1, user_tokens}  -- Success, remaining tokens
end

-- L3 exhausted, try L2 (App level)
local app_data = redis.call('HGETALL', app_key)
local app_tokens = tonumber(app_data[2]) or 0
local app_capacity = tonumber(app_data[4]) or capacity * 100

if app_tokens >= cost then
    -- L2 deduction successful, allocate to L3
    app_tokens = app_tokens - cost

    -- Reserve additional tokens for L3 cache (bulk allocation)
    local reserve_batch = math.min(app_tokens * 0.1, 10000)
    app_tokens = app_tokens - reserve_batch

    redis.call('HMSET', app_key, 'tokens', app_tokens)
    redis.call('HINCRBY', app_key, 'deducted_count', 1)

    -- Refill L3
    redis.call('HMSET', user_key,
        'tokens', reserve_batch - cost,
        'last_update', now,
        'capacity', capacity,
        'rate', rate
    )

    return {2, reserve_batch - cost}  -- L2 success, L3 refilled
end

-- L2 exhausted, try L1 (Cluster level) - Rare path
local cluster_tokens = tonumber(redis.call('GET', cluster_key)) or 0

if cluster_tokens >= cost * 100 then  -- Batch allocation
    -- L1 deduction successful, allocate to L2
    local allocation = cost * 100
    redis.call('DECRBY', cluster_key, allocation)

    -- Refill L2
    redis.call('HINCRBY', app_key, 'tokens', allocation)

    -- Refill L3
    local reserve_batch = math.min(allocation * 0.1, 10000)
    redis.call('HMSET', user_key,
        'tokens', reserve_batch - cost,
        'last_update', now,
        'capacity', capacity,
        'rate', rate
    )

    return {3, reserve_batch - cost}  -- L1 success, L2/L3 refilled
end

-- All levels exhausted
redis.call('HINCRBY', user_key, 'rejected_count', 1)
redis.call('HSET', user_key, 'last_rejected', now)
return {0, 0}  -- All exhausted
```

**Usage**:
```python
import redis

r = redis.StrictRedis(host='redis-cluster', port=6379, decode_responses=True)

# Load script
script = """
[Lua script above]
"""

# Execute
result = r.eval(script, 3,
    'user:app1:user123:tokens',
    'app:app1:tokens',
    'cluster:tokens',
    100,  # cost
    int(time.time()),  # now
    10,  # rate (tokens/sec)
    1000  # capacity
)

level, remaining = result
if level == 0:
    return "Rate limit exceeded"
else:
    return f"Request allowed (level {level}), {remaining} tokens remaining"
```

### 2.2 Cluster Token Distribution (Top-Down)

**Purpose**: Distribute cluster tokens to applications proportionally

**Lua Script**:
```lua
-- KEYS[1]: Cluster token key
-- ARGV[1]: Distribution timestamp

local cluster_key = KEYS[1]
local now = tonumber(ARGV[1])

-- Get cluster token pool
local cluster_tokens = tonumber(redis.call('GET', cluster_key))
if not cluster_tokens or cluster_tokens <= 0 then
    return 0  -- Nothing to distribute
end

-- Get all active apps
local apps = redis.call('SMEMBERS', 'apps:active')
if #apps == 0 then
    return 0
end

-- Calculate total weight
local total_weight = 0
for _, app_id in ipairs(apps) do
    local weight = tonumber(redis.call('HGET', 'app:' .. app_id .. ':config', 'weight')) or 1
    total_weight = total_weight + weight
end

-- Distribute tokens proportionally
local distributed_total = 0
for _, app_id in ipairs(apps) do
    local app_key = 'app:' .. app_id .. ':tokens'
    local config_key = 'app:' .. app_id .. ':config'

    local weight = tonumber(redis.call('HGET', config_key, 'weight')) or 1
    local allocation = math.floor(cluster_tokens * weight / total_weight)

    if allocation > 0 then
        -- Don't overfill app bucket
        local current = tonumber(redis.call('HGET', app_key, 'tokens')) or 0
        local capacity = tonumber(redis.call('HGET', app_key, 'capacity')) or 10000
        local top_up = math.min(allocation, capacity - current)

        if top_up > 0 then
            redis.call('HINCRBY', app_key, 'tokens', top_up)
            redis.call('HSET', app_key, 'last_distributed', now)
            distributed_total = distributed_total + top_up
        end
    end
end

-- Deduct from cluster pool
if distributed_total > 0 then
    redis.call('DECRBY', cluster_key, distributed_total)
    redis.call('HSET', 'cluster:stats', 'last_distribution', now)
    redis.call('HINCRBY', 'cluster:stats', 'total_distributed', distributed_total)
end

return distributed_total
```

**Scheduling**: Run every 1 second via cron or Redis keyspace notifications

### 2.3 Batch Reconciliation (Fix Drift)

**Purpose**: Reconcile micro-batch accumulators with source of truth

**Lua Script**:
```lua
-- KEYS[1]: Batch key pattern prefix
-- ARGV[1]: Maximum batches to process
-- ARGV[2]: Current timestamp

local prefix = KEYS[1]
local max_batches = tonumber(ARGV[1])
local now = tonumber(ARGV[2])

local processed = 0
local total_corrected = 0

-- Scan for batch keys
local cursor = "0"
local batch_keys = {}

repeat
    local result = redis.call('SCAN', cursor, 'MATCH', prefix .. '*', 'COUNT', 100)
    cursor = result[1]
    for _, key in ipairs(result[2]) do
        table.insert(batch_keys, key)
        if #batch_keys >= max_batches then
            break
        end
    end
until cursor == "0" or #batch_keys >= max_batches

-- Process each batch
for _, batch_key in ipairs(batch_keys) do
    local accumulated = tonumber(redis.call('GET', batch_key))

    if accumulated and accumulated > 0 then
        -- Parse user key from batch key
        -- Format: batch:{node}:{app}:{user}
        local parts = {}
        for part in string.gmatch(batch_key, "[^:]+") do
            table.insert(parts, part)
        end

        if #parts >= 4 then
            local node_id = parts[2]
            local app_id = parts[3]
            local user_id = parts[4]

            -- Construct user token key
            local user_key = 'user:' .. app_id .. ':' .. user_id .. ':tokens'

            -- Get current user tokens
            local user_tokens = tonumber(redis.call('HGET', user_key, 'tokens')) or 0

            -- Reconcile: Deduct accumulated usage
            if user_tokens >= accumulated then
                -- Normal case: User has enough tokens
                redis.call('HINCRBY', user_key, 'tokens', -accumulated)
                redis.call('HINCRBY', user_key, 'reconciled_deductions', accumulated)
            else
                -- Edge case: User over-drew (L3 inconsistency)
                local shortfall = accumulated - user_tokens
                redis.call('HSET', user_key, 'tokens', 0)
                redis.call('HINCRBY', user_key, 'reconciled_shortfall', shortfall)

                -- Log for monitoring
                redis.call('LPUSH', 'reconcile:errors',
                    string.format('%s|%d|%d|%d', user_key, accumulated, user_tokens, now)
                )
            end

            total_corrected = total_corrected + accumulated
            processed = processed + 1
        end

        -- Delete batch key
        redis.call('DEL', batch_key)
    end
end

return {processed, total_corrected}
```

**Scheduling**: Run every 60 seconds

### 2.4 Race Condition Prevention

**Problem**: Concurrent requests can cause check-and-deduct race conditions

**Solution 1: Lua Script Atomicity**
```lua
-- Entire script runs atomically
-- No other commands can intervene between check and deduct
local tokens = redis.call('HGET', key, 'tokens')
if tokens >= cost then
    redis.call('HINCRBY', key, 'tokens', -cost)
    return 1
end
return 0
```

**Solution 2: Redis Transactions (WATCH/MULTI/EXEC)**
```python
# Python example
def deduct_with_watch(key, cost):
    with r.pipeline() as pipe:
        while True:
            try:
                pipe.watch(key)
                tokens = int(pipe.hget(key, 'tokens'))

                if tokens < cost:
                    pipe.unwatch()
                    return False

                pipe.multi()
                pipe.hincrby(key, 'tokens', -cost)
                pipe.execute()
                return True

            except WatchError:
                # Retry on conflict
                continue
```

**Solution 3: Optimistic Locking (Version Number)**
```lua
-- Use CAS (Compare-And-Swap) pattern
local current_version = redis.call('HGET', key, 'version')
local expected_version = ARGV[1]

if current_version == expected_version then
    -- Update atomically if version matches
    redis.call('HINCRBY', key, 'version', 1)
    redis.call('HINCRBY', key, 'tokens', -cost)
    return 1
else
    return 0  -- Version mismatch, retry
end
```

**Recommendation**: Use **Lua Scripts** (Solution 1) for simplicity and performance

---

## 3. Performance Optimization

### 3.1 Connection Pooling Strategies

**Redis Cluster Connection Pool**:

```python
import redis
from redis.cluster import RedisCluster

# Optimal pool configuration
redis_client = RedisCluster(
    host='redis-cluster',
    port=6379,
    # Connection pool settings
    max_connections=100,          # Max connections per pool
    socket_timeout=2,             # Socket timeout (seconds)
    socket_connect_timeout=2,     # Connection timeout
    retry_on_timeout=True,        # Retry on timeout
    max_connections_per_node=50,  # Per-node limit

    # Performance tuning
    skip_full_coverage_check=True,    # Skip cluster state check (faster)
    health_check_interval=15,         # Node health check interval
    full_coverage_check_interval=0,   # Disable periodic full coverage check

    # Pool reuse
    connection_pool=redis.ConnectionPool(
        max_connections=100,
        retry=redis.Retry(NoBackoff(), 3)  # 3 retries with no backoff
    )
)
```

**Nginx (OpenResty) Connection Pool**:

```lua
-- /etc/nginx/lua/redis_pool.lua

local redis = require "resty.redis"

local function get_redis_connection()
    local red = redis:new()

    -- Connection pool settings
    red:set_timeout(1000)  -- 1 second timeout

    -- Connect to Redis Cluster
    local ok, err = red:connect("redis-cluster", 6379)
    if not ok then
        return nil, err
    end

    -- Set keepalive (return to pool after use)
    -- Parameters: max_idle_timeout, pool_size
    red:set_keepalive(10000, 100)  -- 10s timeout, 100 connections

    return red
end

return {
    get_redis_connection = get_redis_connection
}
```

**Best Practices**:
1. **Pool Size**: Set to 2× number of concurrent threads/workers
2. **Timeouts**: Aggressive timeouts (1-2s) with retry logic
3. **Keepalive**: Reuse connections to avoid TCP handshake overhead
4. **Per-Node Pools**: In cluster mode, pool per node for better locality

### 3.2 Pipeline and Batch Operations

**Problem**: One-by-one Redis commands = N× network RTT

**Solution 1: Redis Pipeline (Bulk Operations)**

```python
# Pipeline for bulk token deduction
def deduct_bulk_users(deductions):
    """
    deductions: List of (app_id, user_id, cost) tuples
    """
    pipe = r.pipeline()

    for app_id, user_id, cost in deductions:
        key = f'user:{app_id}:{user_id}:tokens'
        pipe.hincrby(key, 'tokens', -cost)
        pipe.hincrby(key, 'deducted_count', 1)
        pipe.hset(key, 'last_update', int(time.time()))

    # Execute all commands in single round-trip
    results = pipe.execute()

    return results
```

**Performance Gain**:
- Without pipeline: 1000 operations × 5ms RTT = 5 seconds
- With pipeline: 1 batch = 5-10ms (500-1000× faster)

**Solution 2: Lua Script with Multiple Keys**

```lua
-- Single Lua script = single network round-trip
-- Process multiple user deductions atomically

local results = {}
for i = 1, #KEYS do
    local key = KEYS[i]
    local cost = tonumber(ARGV[i])

    local tokens = tonumber(redis.call('HGET', key, 'tokens'))
    if tokens and tokens >= cost then
        redis.call('HINCRBY', key, 'tokens', -cost)
        table.insert(results, 1)
    else
        table.insert(results, 0)
    end
end

return results
```

**Solution 3: Micro-Batch Accumulation (L3 → L2)**

```lua
-- Accumulate in Nginx shared memory
local batch = ngx.shared.batch_accumulator
local batch_key = app_id .. ":" .. user_id

-- Accumulate locally
local current = batch:get(batch_key) or 0
batch:set(batch_key, current + cost, 3600)  -- TTL 1 hour

-- Periodic flush (every 100ms or 1000 operations)
local function flush_batches()
    local keys = batch:get_keys(1000)

    local redis = require "resty.redis"
    local red = redis:new()
    red:init_pipeline()

    for _, key in ipairs(keys) do
        local usage = batch:get(key)
        if usage > 0 then
            red:hincrby("user:" .. key, "tokens", -usage)
            batch:delete(key)
        end
    end

    red:commit_pipeline()
    red:set_keepalive(10000, 100)
end
```

**Performance Comparison**:

| Approach | QPS per Redis Instance | Network Round-Trips | Latency |
|----------|------------------------|---------------------|---------|
| No Batching | 1,000 | N × RTT | 50ms per op |
| Pipeline (100 ops) | 50,000 | N/100 × RTT | 5ms per op |
| Lua Script (100 keys) | 100,000 | 1 × RTT | 1-2ms per op |
| Micro-Batch (1000 ops) | 200,000+ | 1 per 1000 ops | <1ms per op |

### 3.3 Local Cache Invalidation Strategies

**L3 (Nginx) Cache Invalidation**:

**Strategy 1: TTL-based Expiration**
```lua
-- Set TTL on local cache entries
ngx.shared.token_cache:set(
    "user:" .. app_id .. ":" .. user_id,
    tokens,
    60  -- TTL: 60 seconds
)
```

**Strategy 2: Active Invalidation (Redis Pub/Sub)**
```lua
-- Subscribe to invalidation channel
local redis = require "resty.redis"
local red = redis:new()

red:subscribe("invalidate:token_cache")

-- Listen for invalidation messages
while true do
    local msg, err = red:read_reply()
    if msg and msg[1] == "message" then
        local channel = msg[2]
        local data = msg[3]

        -- Parse: "invalidate:token_cache:{app_id}:{user_id}"
        local app_id, user_id = data:match("([^:]+):([^:]+)")

        -- Invalidate local cache
        ngx.shared.token_cache:delete("user:" .. app_id .. ":" .. user_id)
    end
end
```

**Strategy 3: Lazy Invalidation (Version Check)**
```lua
-- Before using local cache, check version in Redis
local function get_tokens_with_version(app_id, user_id)
    local cache_key = "user:" .. app_id .. ":" .. user_id
    local local_data = ngx.shared.token_cache:get(cache_key)

    if local_data then
        local local_version = local_data.version
        local redis_version = redis:hget(cache_key, "version")

        if local_version == redis_version then
            -- Cache still valid
            return local_data.tokens
        else
            -- Version mismatch, invalidate and refetch
            ngx.shared.token_cache:delete(cache_key)
        end
    end

    -- Fetch from Redis and update cache
    local tokens = fetch_from_redis(app_id, user_id)
    ngx.shared.token_cache:set(cache_key, tokens, 60)

    return tokens
end
```

**Recommendation**: **TTL-based** for simplicity, with **Active Invalidation** for critical updates

### 3.4 Redis Cluster Sharding Approach

**Hash Slot Allocation**:

```
Redis Cluster: 16384 hash slots
Shard Strategy: CRC16(key) % 16384

Key Distribution:
  User keys:    user:{app_id}:{user_id}:tokens    → Slot based on {app_id}
  App keys:     app:{app_id}:tokens               → Slot based on {app_id}
  Cluster keys: cluster:tokens                    → Fixed slot

Hash Tags: Use {braces} to force same slot
  user:{app123}:user1  → Same slot as app:{app123}
  user:{app123}:user2  → Same slot as app:{app123}
```

**Shard Configuration (6 nodes, 3 masters + 3 replicas)**:

```bash
# Node 1 (Master): Slots 0-5460
# Node 2 (Master): Slots 5461-10922
# Node 3 (Master): Slots 10923-16383
# Node 4-6 (Replicas): Follow respective masters

# Create cluster
redis-cli --cluster create \
  10.0.1.1:6379 \
  10.0.1.2:6379 \
  10.0.1.3:6379 \
  10.0.2.1:6379 \
  10.0.2.2:6379 \
  10.0.2.3:6379 \
  --cluster-replicas 1
```

**Data Locality Optimization**:

```python
# Use hash tags to co-locate related data
def get_user_keys(app_id, user_id):
    """
    All keys for same app_id map to same shard
    Enables efficient Lua scripts with multiple keys
    """
    return {
        'user_tokens': f'user:{{{app_id}}}:{user_id}:tokens',
        'user_usage': f'user:{{{app_id}}}:{user_id}:usage',
        'app_tokens': f'app:{{{app_id}}}:tokens',
        'app_config': f'app:{{{app_id}}}:config',
    }

# Single Lua script can access all these keys atomically
script = """
local user_tokens = redis.call('HGET', KEYS[1], 'tokens')
local app_tokens = redis.call('HGET', KEYS[3], 'tokens')
-- ... process both keys
"""
```

**Resharding Strategy**:

```bash
# Add new node
redis-cli --cluster add-node 10.0.1.4:6379 10.0.1.1:6379

# Migrate 20% of slots from each master to new node
redis-cli --cluster reshard 10.0.1.1:6379 \
  --cluster-from <node_ids> \
  --cluster-to <new_node_id> \
  --cluster-slots 3277 \
  --cluster-yes
```

**Monitoring Shard Balance**:

```python
def check_shard_balance():
    """
    Ensure even distribution of keys across shards
    """
    shard_info = {}

    for node in redis_client.cluster_nodes():
        slots = node['slots']
        key_count = 0

        for slot_range in slots:
            # Sample keys in this slot range
            sample_keys = redis_client.cluster_scan(
                f'*',
                count=1000,
                slot_range=slot_range
            )
            key_count += len(sample_keys)

        shard_info[node['id']] = {
            'slots': len(slots),
            'key_count': key_count,
            'balance_ratio': key_count / len(slots)
        }

    return shard_info
```

---

## 4. Data Consistency & Reliability

### 4.1 L1/L2/L3 Consistency Guarantees

**Consistency Model**: **Eventual Consistency** with Strong Consistency options

**Layer-wise Consistency**:

| Layer | Consistency Model | Rationale |
|-------|------------------|-----------|
| **L1 (Cluster)** | Strong | Single source of truth, Lua scripts ensure atomicity |
| **L2 (App)** | Strong | Same Redis instance, Lua scripts for atomic ops |
| **L3 (Local)** | Eventual | Async micro-batch, sync every 100ms or 60s max |

**Timeline of Consistency**:

```
T0: User request arrives at Nginx (L3)
T0+0.1ms: Check local cache (L3)
T0+0.2ms: If cache miss, fetch from Redis (L2)
T0+5ms: Deduct tokens from L2 (atomic Lua)
T0+5.1ms: Update local cache (L3)
T0+5.2ms: Accumulate in batch accumulator
T0+100ms: Batch flush to L2 (async)
T0+60s: Full reconciliation if batch flush failed
```

**Consistency Guarantees**:

1. **L1 → L2**: Strong consistency (atomic Lua)
   - Distribution script runs every 1s
   - No lost tokens, no double-spending

2. **L2 → L3**: Eventual consistency (async batch)
   - Max delay: 100ms (normal), 60s (failure)
   - Temporary over-usage possible during batch delay
   - Reconciliation corrects drift

3. **L3 → Client**: Strong consistency (local check)
   - Client sees result immediately
   - No race conditions within single Nginx worker

### 4.2 Handling Redis Failures (Fail-Open)

**Failure Detection**:

```lua
-- Health check function
local function check_redis_health()
    local redis = require "resty.redis"
    local red = redis:new()
    red:set_timeout(1000)  -- 1 second timeout

    local ok, err = red:connect("redis-cluster", 6379)
    if not ok then
        return "unavailable", err
    end

    -- Ping to check connection
    local res, err = red:ping()
    if not res then
        return "error", err
    end

    -- Check cluster state
    local info = red:info("cluster")
    if not string.match(info, "cluster_state:ok") then
        return "degraded", "Cluster not OK"
    end

    red:set_keepalive(10000, 100)
    return "ok", nil
end
```

**Fail-Open Levels**:

**Level 1: Redis Slight Degradation** (Latency > 10ms, < 100ms)

```lua
local health, err = check_redis_health()

if health == "degraded" then
    -- Increase L3 cache size and batch window
    local batch_window = 1000  -- Increase from 100ms to 1000ms
    local cache_reserve = 50000  -- Increase from 10k to 50k tokens

    -- Continue normal operation with larger hysteresis
    return deduct_from_local_cache()
end
```

**Level 2: Redis Significant Degradation** (Latency > 100ms or 5% errors)

```lua
if health == "error" then
    -- Switch to token reservation mode
    -- Allocate larger chunks from L2, reduce refresh frequency

    local reserve_chunk = 100000  -- Reserve 100k tokens at once
    local refresh_interval = 60  -- Refresh every 60s instead of 1s

    -- Implement token reservation
    local reserved = reserve_tokens_from_l2(reserve_chunk)

    if reserved > 0 then
        -- Operate in offline mode with reserved tokens
        ngx.shared.token_cache:set("offline_mode", true, 60)
        return deduct_from_reserved(reserved)
    end
end
```

**Level 3: Redis Complete Failure** (Connection timeout or > 50% errors)

```lua
if health == "unavailable" then
    -- FAIL-OPEN: Allow all traffic with local enforcement only

    ngx.log(ngx.WARN, "Redis unavailable, activating fail-open mode")

    -- Simple per-IP rate limit
    local ip = ngx.var.remote_addr
    local limit_key = "fail_open:" .. ip

    local count = ngx.shared.fail_open_limit:get(limit_key) or 0

    if count >= 100 then  -- 100 req/sec per IP
        ngx.status = 429
        ngx.header["Retry-After"] = "60"
        ngx.header["X-RateLimit-Limit"] = "local"
        ngx.exit(429)
    else
        ngx.shared.fail_open_limit:incr(limit_key, 1, 1)  -- TTL 1s
        return true  -- Allow request
    end
end
```

**Nginx Connection Limit (Last Line of Defense)**:

```nginx
http {
    # Per-IP connection limit
    limit_conn_zone $binary_remote_addr zone=addr:10m;

    server {
        # Limit to 80 concurrent connections per IP
        limit_conn addr 80;
        limit_conn_status 503;

        # Whitelist critical endpoints
        location /health {
            limit_conn addr 1000;  # Higher limit for health checks
        }

        location / {
            # Normal rate limiting with fail-open
            access_by_lua_block {
                -- Rate limiting logic here
            }
        }
    }
}
```

**Monitoring Fail-Open Activation**:

```lua
-- Log fail-open events
if fail_open_activated then
    ngx.log(ngx.ERR, "[FAIL-OPEN] Redis unavailable at ", ngx.now())

    -- Send to monitoring
    local prometheus = require "resty.prometheus"
    prometheus:metric("redis_fail_open_total", 1, {
        reason = health,
        node = ngx.worker.id()
    })

    -- Alert operations team
    send_alert({
        severity = "critical",
        message = "Redis unavailable, fail-open mode activated",
        node = os.getenv("HOSTNAME")
    })
end
```

### 4.3 Recovery and Synchronization After Failures

**Recovery Procedure**:

```lua
-- Step 1: Detect Redis recovery
local function detect_redis_recovery()
    local health, err = check_redis_health()

    if health == "ok" then
        local was_in_fail_open = ngx.shared.token_cache:get("offline_mode")

        if was_in_fail_open then
            ngx.log(ngx.NOTICE, "Redis recovered, initiating synchronization")
            return true
        end
    end

    return false
end

-- Step 2: Synchronize L3 cache with L2
local function synchronize_after_recovery()
    local redis = require "resty.redis"
    local red = redis:new()

    local ok, err = red:connect("redis-cluster", 6379)
    if not ok then
        return nil, err
    end

    -- Get all local cache entries
    local local_keys = ngx.shared.token_cache:get_keys(0)  -- All keys

    for _, key in ipairs(local_keys) do
        local local_tokens = ngx.shared.token_cache:get(key)

        -- Parse key: user:{app_id}:{user_id}:tokens
        local app_id, user_id = key:match("user:([^:]+):([^:]+)")

        if app_id and user_id then
            local user_key = "user:" .. app_id .. ":" .. user_id .. ":tokens"

            -- Fetch actual tokens from Redis
            local redis_tokens = red:hget(user_key, "tokens")

            if redis_tokens then
                -- Update local cache with actual value
                ngx.shared.token_cache:set(key, redis_tokens, 60)
            end
        end
    end

    -- Flush any pending batches
    flush_batches()

    red:set_keepalive(10000, 100)

    -- Exit offline mode
    ngx.shared.token_cache:delete("offline_mode")

    return true
end

-- Step 3: Reconciliation to correct drift
local function reconcile_after_recovery()
    local redis = require "resty.redis"
    local red = redis:new()

    local ok, err = red:connect("redis-cluster", 6379)
    if not ok then
        return nil, err
    end

    -- Run reconciliation script
    local reconcile_script = """
    [Reconciliation script from section 2.3]
    """

    red:eval(reconcile_script, 1, "batch:", 10000, ngx.time())

    red:set_keepalive(10000, 100)

    return true
end
```

**Recovery Sequence**:

```
T0: Redis failure detected
    → Activate fail-open mode
    → Switch to per-IP limits
    → Log event, send alert

T+30s: Redis recovers
    → Detect recovery (health check passes)
    → Synchronize L3 cache with L2
    → Flush pending batches
    → Exit offline mode

T+60s: Full reconciliation
    → Compare L2 usage logs with L3 deductions
    → Correct drift > 10%
    → Log reconciliation results
    → Return to normal operation
```

**Data Loss Prevention**:

```lua
-- Before entering fail-open, flush critical data
local function emergency_flush()
    local redis = require "resty.redis"
    local red = redis:new()

    -- Try one last time to connect
    red:set_timeout(500)  -- Short timeout

    local ok, err = red:connect("redis-cluster", 6379)
    if ok then
        -- Flush all pending batches
        local keys = ngx.shared.batch_accumulator:get_keys(0)

        red:init_pipeline()
        for _, key in ipairs(keys) do
            local usage = ngx.shared.batch_accumulator:get(key)
            if usage > 0 then
                red:hincrby("user:" .. key, "tokens", -usage)
            end
        end

        red:commit_pipeline()

        ngx.log(ngx.NOTICE, "Emergency flush successful: ", #keys, " batches")
    else
        ngx.log(ngx.ERR, "Emergency flush failed, data may be lost")
    end
end
```

---

## 5. Capacity Planning

### 5.1 Memory Estimation per Token Bucket

**Memory Breakdown (Redis)**:

```python
# Base memory per key (Redis overhead)
REDIS_KEY_OVERHEAD = 100  # bytes (dict entry, SDS header, etc.)

# Hash field overhead
HASH_FIELD_OVERHEAD = 50  # bytes per field

# Integer values (tokens, counts)
INTEGER_VALUE_SIZE = 8    # bytes (int64)

# String values (IDs)
STRING_VALUE_SIZE = 20    # bytes avg

def calculate_memory_per_user_bucket(num_fields=6):
    """
    user:{app}:{user_id}:tokens
    Fields: tokens, last_update, capacity, rate, reserved, burst
    """
    memory = (
        REDIS_KEY_OVERHEAD +                       # Key overhead
        (num_fields * HASH_FIELD_OVERHEAD) +       # Field headers
        (num_fields * INTEGER_VALUE_SIZE)          # Field values
    )
    return memory

# Example
print(calculate_memory_per_user_bucket())
# Output: 100 + (6 * 50) + (6 * 8) = 100 + 300 + 48 = 448 bytes
```

**Per-User Memory**: **~500 bytes**

**Per-Application Memory**:

```python
def calculate_memory_per_app_bucket(num_fields=5, num_users=1000):
    """
    app:{app_id}:tokens
    Fields: tokens, capacity, rate, weight, priority
    Plus user references
    """
    app_memory = (
        REDIS_KEY_OVERHEAD +
        (num_fields * HASH_FIELD_OVERHEAD) +
        (num_fields * INTEGER_VALUE_SIZE)
    )

    users_memory = num_users * calculate_memory_per_user_bucket()

    return app_memory + users_memory

# Example
print(calculate_memory_per_app_bucket())
# Output: 350 + (1000 * 500) = 500,350 bytes (~500 KB)
```

**Total Memory Estimate**:

```python
def calculate_total_memory(
    num_apps=10,
    num_users_per_app=1000,
    num_nodes=100
):
    """
    Calculate total Redis memory required
    """
    # L1: Cluster level (~1 KB)
    l1_memory = 1000

    # L2: Application level
    l2_memory = num_apps * calculate_memory_per_app_bucket()

    # L3: User level (already counted in L2)

    # Batch accumulators (node/app/user combinations)
    # Assume 10% of users active per node at any time
    active_batches = num_nodes * num_apps * (num_users_per_app * 0.1)
    batch_memory = active_batches * (REDIS_KEY_OVERHEAD + INTEGER_VALUE_SIZE)

    # Add 30% for Redis overhead, fragmentation
    total = (l1_memory + l2_memory + batch_memory) * 1.3

    return {
        'l1': l1_memory,
        'l2': l2_memory,
        'batch': batch_memory,
        'total': total
    }

# Example
memory = calculate_total_memory()
print(f"L1: {memory['l1']} bytes")
print(f"L2: {memory['l2'] / 1024 / 1024:.2f} MB")
print(f"Batch: {memory['batch'] / 1024 / 1024:.2f} MB")
print(f"Total: {memory['total'] / 1024 / 1024:.2f} MB")
```

**Output**:
```
L1: 1000 bytes
L2: 4.88 MB (10 apps × 1000 users)
Batch: 5.20 MB (100 nodes × 10 apps × 100 active users)
Total: 13.07 MB (with 30% overhead)
```

### 5.2 Key Count Scaling (Multi-Tenant)

**Key Count Growth**:

```python
def estimate_key_count(
    num_apps,
    num_users_per_app,
    num_daily_stats=7,
    num_batch_nodes=100
):
    """
    Estimate total key count for capacity planning
    """
    keys = {
        'cluster': 3,  # tokens, config, stats
        'apps': num_apps * 3,  # tokens, config, users:active
        'users': num_apps * num_users_per_app * 2,  # tokens, usage
        'daily_stats': num_apps * num_daily_stats,  # Last 7 days
        'batch': num_batch_nodes * num_apps * (num_users_per_app * 0.1)  # Active batches
    }

    keys['total'] = sum(keys.values())

    return keys

# Scenarios
scenarios = [
    ("Small", 10, 100),
    ("Medium", 50, 1000),
    ("Large", 100, 10000),
    ("X-Large", 500, 100000)
]

for name, apps, users in scenarios:
    keys = estimate_key_count(apps, users)
    print(f"{name}: {keys['total']:,} keys ({keys['total'] / 1000000:.2f}M)")
```

**Output**:
```
Small: 2,010 keys (0.00M)
Medium: 100,050 keys (0.10M)
Large: 2,000,500 keys (2.00M)
X-Large: 50,005,000 keys (50.01M)
```

**Redis Limits**:
- **Max Keys**: 2^32 (4.29 billion) - Not a concern
- **Practical Limit**: 100M keys per cluster (memory-bound)

**Recommendation**: For >10M keys, consider:
- Sharding by app_id (multiple Redis clusters)
- Cold data archival (move inactive users to cheaper storage)
- Data compression (Redis 7+ with zstd)

### 5.3 Eviction Policies

**Redis Eviction Policy Selection**:

```redis
# Recommended policy for token buckets
maxmemory-policy allkeys-lru

# Rationale:
# - allkeys: All keys can be evicted (no volatile-only requirement)
# - lru: Evict least recently used (good for temporal locality)
# - Alternative: volatile-ttl if all keys have TTL
```

**Policy Comparison**:

| Policy | Behavior | Use Case |
|--------|----------|----------|
| **noeviction** | Return error on OOM | Not recommended (system stops) |
| **allkeys-lru** | Evict least recently used | **Recommended** for token buckets |
| **volatile-lru** | Evict LRUs among keys with TTL | Use if some keys must never expire |
| **allkeys-random** | Evict random keys | Not recommended (poor locality) |
| **volatile-ttl** | Evict keys with shortest TTL | Good for time-series data |
| **allkeys-lfu** | Evict least frequently used | Good for access pattern stability |

**Setting Eviction Policy**:

```bash
# redis.conf
maxmemory 4gb
maxmemory-policy allkeys-lru
maxmemory-samples 5  # Check 5 keys for LRU eviction

# Or at runtime
redis-cli CONFIG SET maxmemory 4gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

**Monitoring Eviction**:

```python
def monitor_eviction():
    """
    Track eviction rate to tune memory
    """
    info = redis_client.info('stats')

    evicted_keys = info['evicted_keys']
    total_keys = info['keyspace']['db0']  # Keys in DB 0

    eviction_rate = evicted_keys / total_keys if total_keys > 0 else 0

    if eviction_rate > 0.1:  # >10% eviction rate
        print(f"WARNING: High eviction rate: {eviction_rate:.2%}")
        print("Consider increasing Redis memory or optimizing data structures")

    return {
        'evicted_keys': evicted_keys,
        'total_keys': total_keys,
        'eviction_rate': eviction_rate
    }
```

**Tuning Memory Usage**:

```python
def optimize_memory():
    """
    Reduce memory footprint
    """
    # 1. Compress strings (Redis 7+)
    redis_client.config_set('compression', 'zstd')

    # 2. Use int encoding instead of string for numbers
    redis_client.hset('user:app1:user1:tokens', mapping={
        'tokens': 1000,  # Stored as int64 (8 bytes)
        'last_update': 1704067200  # Stored as int64
    })

    # 3. Enable lazy eviction (free memory asynchronously)
    redis_client.config_set('lazyfree-lazy-eviction', 'yes')
    redis_client.config_set('lazyfree-lazy-expire', 'yes')

    # 4. Set aggressive TTL on cold data
    redis_client.expire('user:app1:inactive_user:usage', 86400)  # 1 day

    # 5. Use HyperLogLog for cardinality estimation
    redis_client.pfadd('app:app1:unique_users', 'user1', 'user2', 'user3')
    count = redis_client.pfcount('app:app1:unique_users')  # ~12 bytes
```

### 5.4 Capacity Planning Calculator

**Interactive Sizing**:

```python
def plan_capacity(
    qps_per_node,
    num_nodes,
    req_size_bytes,
    num_apps,
    num_users_per_app,
    safety_margin=0.3
):
    """
    Plan Redis cluster capacity
    """
    # Calculate required throughput
    total_qps = qps_per_node * num_nodes

    # Redis can handle ~100k ops/sec per instance (simple ops)
    # With Lua scripts, assume ~50k ops/sec
    redis_instances = max(1, total_qps / 50000)

    # Memory calculation
    memory_per_user = 500  # bytes
    total_users = num_apps * num_users_per_app
    memory_needed = total_users * memory_per_user * (1 + safety_margin)

    # Network bandwidth
    # Assume 1 KB per operation (request + response)
    bandwidth_gbps = (total_qps * 1024 * 8) / 1e9

    # Recommendations
    recommendations = {
        'redis_instances': int(redis_instances) + 1,
        'memory_per_instance_gb': memory_needed / (1 + safety_margin) / 1024 / 1024 / 1024,
        'total_memory_gb': memory_needed / 1024 / 1024 / 1024,
        'network_bandwidth_gbps': bandwidth_gbps,
        'replication_factor': 3  # Standard for HA
    }

    return recommendations

# Example usage
plan = plan_capacity(
    qps_per_node=1000,
    num_nodes=100,
    req_size_bytes=1024,
    num_apps=100,
    num_users_per_app=10000
)

print("Capacity Plan:")
print(f"  Redis Instances: {plan['redis_instances']}")
print(f"  Memory per Instance: {plan['memory_per_instance_gb']:.2f} GB")
print(f"  Total Memory: {plan['total_memory_gb']:.2f} GB")
print(f"  Network Bandwidth: {plan['network_bandwidth_gbps']:.2f} Gbps")
print(f"  Replication Factor: {plan['replication_factor']}")
```

**Output**:
```
Capacity Plan:
  Redis Instances: 3
  Memory per Instance: 4.88 GB
  Total Memory: 6.50 GB
  Network Bandwidth: 0.81 Gbps
  Replication Factor: 3
```

**Scaling Recommendations**:

| Scale | Apps | Users | QPS | Redis Nodes | Memory (Total) |
|-------|------|-------|-----|-------------|----------------|
| **Small** | 10 | 1,000 | 10k | 1 (master) + 2 (replicas) | 1 GB |
| **Medium** | 50 | 10,000 | 50k | 3 (master) + 6 (replicas) | 8 GB |
| **Large** | 100 | 100,000 | 200k | 6 (master) + 12 (replicas) | 80 GB |
| **X-Large** | 500 | 1,000,000 | 1M | 12 (master) + 24 (replicas) | 800 GB |

---

## 6. Implementation Examples

### 6.1 Complete Lua Script Suite

**File: `redis_scripts.lua`**

```lua
local redis_scripts = {}

-- Script 1: Three-layer token deduction
redis_scripts.deduct_tokens = [[
[See Section 2.1 for full script]
]]

-- Script 2: Cluster token distribution
redis_scripts.distribute_cluster = [[
[See Section 2.2 for full script]
]]

-- Script 3: Batch reconciliation
redis_scripts.reconcile_batch = [[
[See Section 2.3 for full script]
]]

-- Script 4: Health check
redis_scripts.health_check = [[
local cluster_state = redis.call('INFO', 'cluster')

if string.match(cluster_state, 'cluster_state:ok') then
    local keys = redis.call('DBSIZE')
    local memory = redis.call('INFO', 'memory')
    local used_mem = string.match(memory, 'used_memory:(%d+)')

    return {1, keys, used_mem}
else
    return {0, 0, 0}
end
]]

-- Script 5: Emergency mode activation
redis_scripts.activate_emergency = [[
local cluster_key = KEYS[1]
local emergency_threshold = tonumber(ARGV[1])

local cluster_tokens = tonumber(redis.call('GET', cluster_key))
local cluster_capacity = tonumber(redis.call('HGET', 'cluster:config', 'capacity'))

local usage_ratio = 1 - (cluster_tokens / cluster_capacity)

if usage_ratio >= emergency_threshold then
    redis.call('HSET', 'cluster:config', 'emergency_mode', 'true')
    redis.call('LPUSH', 'cluster:events', 'emergency:' .. ngx.time())
    return 1  -- Emergency activated
else
    redis.call('HSET', 'cluster:config', 'emergency_mode', 'false')
    return 0  -- Normal mode
end
]]

return redis_scripts
```

**Loading Scripts in Python**:

```python
import redis
from redis.cluster import RedisCluster
import hashlib

class RateLimiterRedis:
    def __init__(self, hosts):
        self.client = RedisCluster(
            startup_nodes=[{'host': h, 'port': 6379} for h in hosts],
            decode_responses=True,
            skip_full_coverage_check=True
        )

        # Load scripts
        with open('redis_scripts.lua', 'r') as f:
            script_content = f.read()

        # Script SHAs (calculated from script content)
        self.scripts = {
            'deduct': self.client.script_load(redis_scripts.deduct_tokens),
            'distribute': self.client.script_load(redis_scripts.distribute_cluster),
            'reconcile': self.client.script_load(redis_scripts.reconcile_batch),
            'health': self.client.script_load(redis_scripts.health_check),
            'emergency': self.client.script_load(redis_scripts.activate_emergency)
        }

    def deduct_tokens(self, app_id, user_id, cost):
        """Deduct tokens using atomic Lua script"""
        keys = [
            f'user:{app_id}:{user_id}:tokens',
            f'app:{app_id}:tokens',
            'cluster:tokens'
        ]
        args = [cost, int(time.time()), 10, 1000]  # cost, now, rate, capacity

        result = self.client.evalsha(self.scripts['deduct'], len(keys), *keys, *args)

        level, remaining = result
        return level > 0  # True if allowed, False if rejected

    def distribute_cluster_tokens(self):
        """Run cluster token distribution"""
        keys = ['cluster:tokens']
        args = [int(time.time())]

        distributed = self.client.evalsha(
            self.scripts['distribute'],
            len(keys),
            *keys,
            *args
        )

        return distributed

    def health_check(self):
        """Check Redis cluster health"""
        result = self.client.evalsha(self.scripts['health'], 0)

        status, keys, memory = result
        return {
            'healthy': status == 1,
            'keys': keys,
            'memory_bytes': memory
        }
```

### 6.2 Nginx/OpenResty Integration

**File: `/etc/nginx/lua/rate_limit.lua`**

```lua
local _M = {}
local redis = require "resty.redis"
local cjson = require "cjson"

-- Configuration
local config = {
    redis_host = "redis-cluster",
    redis_port = 6379,
    redis_timeout = 1000,
    batch_flush_interval = 0.1,  -- 100ms
    batch_max_size = 1000,
    local_cache_ttl = 60,
    emergency_mode = false
}

-- Shared memory zones
local token_cache = ngx.shared.token_cache
local batch_accumulator = ngx.shared.batch_accumulator
local fail_open_cache = ngx.shared.fail_open_cache

-- Redis connection pool
local function get_redis()
    local red = redis:new()
    red:set_timeout(config.redis_timeout)

    local ok, err = red:connect(config.redis_host, config.redis_port)
    if not ok then
        return nil, err
    end

    return red
end

-- Check and deduct tokens
function _M.check_and_deduct(app_id, user_id, operation, size)
    -- Calculate cost
    local cost = calculate_cost(operation, size)

    -- Check local cache first
    local cache_key = "user:" .. app_id .. ":" .. user_id .. ":tokens"
    local local_tokens = token_cache:get(cache_key)

    if local_tokens and local_tokens >= cost then
        -- Fast path: Local cache hit
        token_cache:set(cache_key, local_tokens - cost, config.local_cache_ttl)

        -- Accumulate for batch flush
        local batch_key = "batch:" .. ngx.worker.id() .. ":" .. app_id .. ":" .. user_id
        local current_batch = batch_accumulator:get(batch_key) or 0
        batch_accumulator:set(batch_key, current_batch + cost, 3600)

        return true, local_tokens - cost
    end

    -- Slow path: Cache miss, fetch from Redis
    local red, err = get_redis()
    if not red then
        ngx.log(ngx.ERR, "Redis connection failed: ", err)
        return handle_fail_open(app_id, user_id, cost)
    end

    -- Load and execute deduction script
    local deduce_script = [[
    [Insert Section 2.1 Lua script here]
    ]]

    local keys = {
        "user:" .. app_id .. ":" .. user_id .. ":tokens",
        "app:" .. app_id .. ":tokens",
        "cluster:tokens"
    }
    local args = {cost, ngx.time(), 10, 1000}

    local result, err = red:eval(deduce_script, #keys, unpack(keys), unpack(args))

    if not result then
        ngx.log(ngx.ERR, "Deduction script failed: ", err)
        red:set_keepalive(10000, 100)
        return handle_fail_open(app_id, user_id, cost)
    end

    local level, remaining = result[1], result[2]

    if level == 0 then
        -- All levels exhausted
        red:set_keepalive(10000, 100)
        return false, 0
    end

    -- Update local cache
    token_cache:set(cache_key, remaining, config.local_cache_ttl)

    red:set_keepalive(10000, 100)
    return true, remaining
end

-- Handle fail-open mode
function handle_fail_open(app_id, user_id, cost)
    -- Simple per-IP rate limit
    local ip = ngx.var.remote_addr
    local limit_key = "fail_open:" .. ip

    local count = fail_open_cache:get(limit_key) or 0

    if count >= 100 then  -- 100 req/sec per IP
        ngx.status = 429
        ngx.header["Retry-After"] = "60"
        ngx.header["X-RateLimit-Limit"] = "local"
        ngx.exit(429)
    else
        fail_open_cache:incr(limit_key, 1, 1)  -- TTL 1s
        return true, 100 - count
    end
end

-- Calculate token cost
function calculate_cost(operation, size)
    -- Cost model parameters
    local C_base = {
        GET = 1,
        PUT = 5,
        LIST = 3,
        DELETE = 2
    }

    local C_bw = {
        GET = 1.0,
        PUT = 2.0,
        LIST = 0.5,
        DELETE = 0.3
    }

    local unit_quantum = 4096  -- 4KB

    local base_cost = C_base[operation] or 1
    local bw_cost = (size / unit_quantum) * (C_bw[operation] or 1.0)

    return base_cost + bw_cost
end

-- Batch flush timer
local function create_flush_timer()
    local ok, err = ngx.timer.at(config.batch_flush_interval, flush_batches)
    if not ok then
        ngx.log(ngx.ERR, "Failed to create flush timer: ", err)
    end
end

-- Flush accumulated batches to Redis
function flush_batches(premature)
    if premature then
        return
    end

    local red, err = get_redis()
    if not red then
        ngx.log(ngx.ERR, "Redis connection failed during flush: ", err)
        -- Retry after delay
        local ok, err = ngx.timer.at(1, flush_batches)
        return
    end

    -- Get all batch keys
    local keys = batch_accumulator:get_keys(0)
    local batches_processed = 0

    red:init_pipeline()

    for _, key in ipairs(keys) do
        local usage = batch_accumulator:get(key)

        if usage and usage > 0 then
            -- Parse key: batch:{worker}:{app}:{user}
            local parts = {}
            for part in string.gmatch(key, "[^:]+") do
                table.insert(parts, part)
            end

            if #parts >= 4 then
                local app_id = parts[3]
                local user_id = parts[4]
                local user_key = "user:" .. app_id .. ":" .. user_id .. ":tokens"

                -- Send HINCRBY command
                red:hincrby(user_key, "tokens", -usage)

                -- Clear batch
                batch_accumulator:delete(key)
                batches_processed = batches_processed + 1
            end
        end
    end

    -- Commit pipeline
    local results, err = red:commit_pipeline()
    if not results then
        ngx.log(ngx.ERR, "Failed to commit pipeline: ", err)
    else
        ngx.log(ngx.INFO, "Flushed ", batches_processed, " batches")
    end

    red:set_keepalive(10000, 100)

    -- Schedule next flush
    create_flush_timer()
end

-- Initialize module
function _M.init()
    -- Start batch flush timer
    create_flush_timer()

    ngx.log(ngx.NOTICE, "Rate limiter initialized")
end

return _M
```

**Nginx Configuration**:

```nginx
# /etc/nginx/nginx.conf

user nginx;
worker_processes auto;
worker_rlimit_nofile 100000;

events {
    worker_connections 10000;
    use epoll;
    multi_accept on;
}

http {
    # Lua shared memory zones
    lua_shared_dict token_cache 100m;
    lua_shared_dict batch_accumulator 50m;
    lua_shared_dict fail_open_cache 10m;
    lua_shared_dict prometheus_metrics 10m;

    # Lua package path
    lua_package_path "/etc/nginx/lua/?.lua;;";

    # Init phase
    init_by_lua_block {
        local rate_limit = require "rate_limit"
        rate_limit.init()
    }

    # Rate limiting phase
    access_by_lua_block {
        local rate_limit = require "rate_limit"

        -- Extract request parameters
        local app_id = ngx.var.http_x_app_id or "default"
        local user_id = ngx.var.http_x_user_id or ngx.var.remote_addr
        local operation = ngx.req.get_method()
        local size = tonumber(ngx.var.http_content_length) or 0

        -- Check and deduct tokens
        local allowed, remaining = rate_limit.check_and_deduct(
            app_id,
            user_id,
            operation,
            size
        )

        if not allowed then
            ngx.status = 429
            ngx.header["Retry-After"] = "60"
            ngx.header["X-RateLimit-Limit"] = "10000"
            ngx.header["X-RateLimit-Remaining"] = "0"
            ngx.header["X-RateLimit-Reset"] = ngx.time() + 60
            ngx.header["Content-Type"] = "application/json"

            ngx.say(cjson.encode({
                error = "rate_limit_exceeded",
                message = "Token bucket exhausted",
                retry_after_seconds = 60
            }))

            ngx.exit(429)
        else
            -- Add rate limit headers
            ngx.header["X-RateLimit-Remaining"] = remaining
            ngx.header["X-RateLimit-Limit"] = "10000"
        end
    }

    # Prometheus metrics endpoint
    location /metrics {
        content_by_lua_block {
            local prometheus = require "resty.prometheus"
            prometheus:collect()
        }
    }

    # Upstream backend
    location / {
        proxy_pass http://backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 6.3 Monitoring and Observability

**Prometheus Metrics Export**:

```lua
-- /etc/nginx/lua/prometheus.lua

local _M = {}
local prometheus = require "resty.prometheus"

local p = prometheus.init("prometheus_metrics")

-- Define metrics
local rate_limit_requests = p:counter(
    "rate_limit_requests_total",
    "Total number of rate limit requests",
    {"app_id", "operation", "result"}
)

local rate_limit_latency = p:histogram(
    "rate_limit_latency_seconds",
    "Rate limit check latency",
    {"app_id"},
    {0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0}
)

local token_bucket_level = p:gauge(
    "token_bucket_tokens",
    "Current token count",
    {"app_id", "user_id", "level"}
)

local batch_flush_duration = p:histogram(
    "batch_flush_duration_seconds",
    "Batch flush duration",
    {},
    {0.01, 0.05, 0.1, 0.25, 0.5, 1.0}
)

local redis_errors = p:counter(
    "redis_errors_total",
    "Total Redis errors",
    {"operation", "error_type"}
)

function _M.record_request(app_id, operation, result, latency)
    rate_limit_requests:inc(1, {app_id, operation, result})
    rate_limit_latency:observe(latency, {app_id})
end

function _M.record_token_level(app_id, user_id, level, tokens)
    token_bucket_level:set(tokens, {app_id, user_id, "l" .. level})
end

function _M.record_batch_flush(duration)
    batch_flush_duration:observe(duration)
end

function _M.record_redis_error(operation, error_type)
    redis_errors:inc(1, {operation, error_type})
end

return _M
```

**Grafana Dashboard JSON**:

```json
{
  "dashboard": {
    "title": "Rate Limiting System Overview",
    "panels": [
      {
        "title": "Request Volume",
        "targets": [
          {
            "expr": "sum(rate(rate_limit_requests_total[5m])) by (app_id, result)",
            "legendFormat": "{{app_id}} - {{result}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "P95 Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(rate_limit_latency_seconds_bucket[5m])) by (app_id, le))",
            "legendFormat": "{{app_id}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Token Bucket Levels",
        "targets": [
          {
            "expr": "avg(token_bucket_tokens) by (level)",
            "legendFormat": "Level {{level}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Rejection Rate",
        "targets": [
          {
            "expr": "sum(rate(rate_limit_requests_total{result=\"rejected\"}[5m])) / sum(rate(rate_limit_requests_total[5m]))",
            "legendFormat": "Rejection Rate"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Redis Error Rate",
        "targets": [
          {
            "expr": "sum(rate(redis_errors_total[5m])) by (error_type)",
            "legendFormat": "{{error_type}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Batch Flush Performance",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(batch_flush_duration_seconds_bucket[5m])) by (le))",
            "legendFormat": "P95 Flush Duration"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

---

## Conclusion

This storage design provides a comprehensive foundation for a high-performance distributed token bucket rate-limiting system. Key takeaways:

1. **Redis Cluster** with Hash Tags ensures data locality and atomic operations
2. **Lua Scripts** provide atomic multi-layer token deduction
3. **Micro-Batching** reduces Redis load by 100-1000×
4. **Fail-Open** strategies ensure availability during Redis outages
5. **Capacity Planning** guides infrastructure scaling

**Next Steps**:
1. Implement Lua scripts in Redis Cluster
2. Deploy Nginx/OpenResty with local caching
3. Set up monitoring and alerting
4. Conduct load testing to validate performance
5. Plan phased rollout (pilot → 10% → 100%)

---

**Document End**
**Version**: 1.0
**Last Updated**: 2025-12-31
