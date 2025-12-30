# Distributed Token Bucket Rate-Limiting System - Comprehensive SRE & Reliability Analysis

**Document Version**: 1.0
**Analysis Date**: 2025-12-31
**Author**: SRE Reliability Assessment
**Classifications**: Internal Use - SRE/Operations

---

## Executive Summary

This document provides a comprehensive SRE and reliability analysis of the distributed three-tier token bucket rate-limiting system designed for cloud storage platforms. The analysis covers failure modes, degradation strategies, disaster recovery, observability, capacity planning, incident response, and chaos engineering.

**Overall Reliability Maturity**: 7.5/10

**Critical Findings**:
- Strong fail-open design prioritizes availability
- Multiple degradation layers provide good fault tolerance
- Redis dependency remains primary single point of failure
- Operational complexity requires significant SRE investment
- Monitoring and alerting gaps identified in edge scenarios

---

## 1. Failure Mode Analysis (FMEA)

### 1.1 FMEA Table

| Component | Failure Mode | Failure Effect | Severity (1-10) | Occurrence (1-10) | Detection (1-10) | RPN | Mitigations |
|-----------|--------------|----------------|-----------------|-------------------|------------------|-----|-------------|
| **Redis Cluster** | Complete outage | Fail-open to local limits, potential overage | 7 | 3 | 1 | 21 | Redis Cluster multi-AZ, fail-open design, connection limits |
| **Redis Cluster** | Single node failure | 5-10% key slots unavailable, some request failures | 4 | 5 | 2 | 40 | Automatic failover, client retry logic, health checks |
| **Redis Cluster** | High latency (>100ms) | Degraded performance, increased L3 cache pressure | 5 | 4 | 3 | 60 | Dynamic rate adjustment, increased batch sizes, fail-open |
| **Redis Cluster** | Network partition | Split-brain risk, majority/minority partition behavior | 8 | 2 | 4 | 64 | Redis Cluster quorum, fail-open when partition detected |
| **L3 Cache** | Nginx crash before flush | Token loss, temporary quota drift | 6 | 2 | 5 | 60 | Periodic reconciliation every 60s, crash recovery scripts |
| **L3 Cache** | Memory exhaustion | Cache eviction, increased Redis load | 5 | 4 | 4 | 80 | Memory limits, LRU eviction, max batch size caps |
| **L3 Cache** | Flush failures (Redis down) | Over-limit usage, quota violations | 7 | 3 | 5 | 105 | Retry with exponential backoff, fail-open at 10% failure rate |
| **L1 Token Pool** | Exhaustion (≥95%) | Emergency mode, request throttling | 6 | 4 | 2 | 48 | Predictive scaling, priority queues, emergency mode procedures |
| **L2 Token Pool** | Individual app exhaustion | App-level throttling, no cross-app impact | 4 | 6 | 3 | 72 | Token borrowing, overflow pools, dynamic reallocation |
| **Nginx Gateway** | Worker connection exhaustion (80%) | 503 errors, new connections rejected | 8 | 3 | 2 | 48 | Connection limits, graceful degradation, auto-scaling |
| **Nginx Gateway** | CPU/memory exhaustion | Latency spikes, request queueing | 7 | 4 | 3 | 84 | Resource limits, horizontal pod autoscaling, load shedding |
| **Control Center** | Unavailability | No quota adjustments, stale policies | 5 | 3 | 6 | 90 | Cached policies, versioned configurations, automated rollback |
| **Network** | Inter-AZ outage | Cross-AZ Redis latency spikes | 6 | 2 | 4 | 48 | Local Redis replicas, client-side retry, fail-open |
| **Network** | Gateway to Redis partition | Complete Redis unavailability | 8 | 2 | 3 | 48 | Local-only mode, connection limits, monitoring alerts |

**RPN = Risk Priority Number = Severity × Occurrence × Detection**

### 1.2 Top Critical Failure Modes (RPN > 60)

#### 1. L3 Cache Flush Failures (RPN: 105)
**Scenario**: Redis becomes unavailable during batch flush operations

**Impact**:
- Unflushed token deductions lost
- Users may exceed quota (temporary)
- Reconciliation latency increases

**Detection**:
- Redis error monitoring
- Flush failure rate metrics
- Token drift alerts

**Mitigation Strategies**:
```lua
-- Flush failure detection and fail-open
local function flush_batch_with_retry()
    local max_retries = 3
    local base_delay = 0.1  -- 100ms

    for attempt = 1, max_retries do
        local ok, err = red:commit_pipeline()

        if ok then
            return true  -- Success
        end

        -- Track failure rate
        track_flush_failure(err)

        if attempt == max_retries then
            -- Fail-open: Enable local-only mode
            if get_flush_failure_rate() > 0.1 then
                activate_fail_open_mode("flush_failure")
            end
            return false
        end

        -- Exponential backoff with jitter
        local delay = base_delay * (2 ^ (attempt - 1))
        ngx.sleep(delay + math.random() * 0.05)
    end
end
```

**Runbook**: `runbooks/redis-flush-failure.md`

#### 2. Nginx Gateway CPU/Memory Exhaustion (RPN: 84)
**Scenario**: Request volume exceeds gateway capacity

**Impact**:
- Increased latency
- Request queueing
- Potential cascading failure

**Detection**:
- CPU/memory monitoring
- Request queue depth
- Latency percentiles (P95, P99)

**Mitigation Strategies**:
```yaml
# Kubernetes HPA for Nginx gateways
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: nginx-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: nginx-gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 30
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
```

**Runbook**: `runbooks/gateway-overload.md`

#### 3. L2 Token Pool App Exhaustion (RPN: 72)
**Scenario**: Individual application exhausts quota while cluster has capacity

**Impact**:
- App-level throttling
- Poor user experience for that app
- Potential customer complaints

**Detection**:
- Per-app token utilization alerts
- Rejection rate spike by app_id
- Customer-reported issues

**Mitigation Strategies**:
```lua
-- Token borrowing mechanism
function borrow_from_cluster_pool(app_id, amount)
    local cluster_available = redis.call("GET", "cluster:available")
    local app_tokens = redis.call("HGET", "app:" .. app_id, "tokens")

    if cluster_available >= amount and app_tokens < amount then
        -- Borrow with 20% interest rate
        local loan = amount
        local repayment = amount * 1.2

        -- Atomic borrowing
        redis.call("DECRBY", "cluster:available", loan)
        redis.call("HINCRBY", "app:" .. app_id, "tokens", loan)
        redis.call("HINCRBY", "app:" .. app_id, "debt", repayment)

        log_event("token_borrow", {
            app_id = app_id,
            loan = loan,
            repayment = repayment,
            timestamp = ngx.time()
        })

        return true
    end

    return false
end
```

**Runbook**: `runbooks/app-quota-exhaustion.md`

---

## 2. Degradation Strategies Evaluation

### 2.1 Redis Fail-Open Analysis

#### Level 1: Slight Degradation (Latency 10-100ms, <5% errors)

**Trigger**:
```lua
function check_redis_health()
    local start = ngx.now()
    local ok, err = red:ping()
    local latency = (ngx.now() - start) * 1000  -- ms

    local error_rate = get_error_rate()

    if latency > 10 and latency < 100 and error_rate < 0.05 then
        return "slight_degradation"
    end
end
```

**Response Strategy**:
- Increase L3 cache allocation from 10k to 20k tokens
- Extend batch flush interval from 100ms to 500ms
- Reduce Redis connection pool timeout from 1s to 500ms
- Continue normal operation with monitoring

**Effectiveness**: HIGH
- Pros: Maintains availability, reduces Redis load
- Cons: Increased token drift risk
- Recovery: Automatic when Redis recovers

**Exit Criteria**: Redis latency <10ms and error rate <1% for 60 seconds

#### Level 2: Significant Degradation (Latency >100ms or ≥5% errors)

**Trigger**:
```lua
if latency > 100 or error_rate >= 0.05 then
    return "significant_degradation"
end
```

**Response Strategy**:
- Switch to token reservation mode (allocate 50k tokens in bulk)
- Reduce Redis refresh frequency to 5 seconds
- Enable local-only enforcement with 2x limits
- Alert on-call engineer

**Effectiveness**: MEDIUM
- Pros: Reduces Redis dependency significantly
- Cons: Risk of over-allocation, potential quota violations
- Recovery: Gradual return to normal over 5 minutes

**Exit Criteria**: Redis latency <20ms and error rate <2% for 120 seconds

#### Level 3: Complete Failure (Timeout or >50% errors)

**Trigger**:
```lua
if not ok or error_rate >= 0.50 then
    return "complete_failure"
end
```

**Response Strategy**:
```nginx
# Complete fail-open configuration
http {
    # Local-only rate limiting
    lua_shared_dict local_limit 10m;

    limit_conn_zone $binary_remote_addr zone=addr:10m;
    limit_conn addr 80;  # 80% of worker_connections

    # Bypass Redis entirely
    init_by_lua_block {
        _ENV.FAIL_OPEN_MODE = true
        _ENV.LOCAL_LIMIT_QPS = 100
    }
}
```

**Effectiveness**: HIGH (for availability)
- Pros: Maximum availability, simple operation
- Cons: No quota enforcement, all users share same limits
- Recovery: Manual verification required before returning to normal

**Exit Criteria**:
1. Redis fully operational for 5 minutes
2. Manual SRE approval
3. Gradual traffic ramp-up (10% → 50% → 100%)

### 2.2 Connection Limit Strategy (80% Threshold)

**Configuration**:
```nginx
worker_processes auto;
worker_connections 1024;  # Per worker
worker_rlimit_nofile 65535;

http {
    # Global connection limit
    limit_conn_zone $server_name zone=server:10m;
    limit_conn server 818;  # 80% of 1024

    # Per-IP connection limit
    limit_conn_zone $binary_remote_addr zone=addr:10m;
    limit_conn addr 100;  # Prevent abuse

    # Whitelist critical endpoints
    map $uri $is_critical {
        default 0;
        /health 1;
        /metrics 1;
        /api/v1/health 1;
    }

    server {
        # Critical endpoints bypass limits
        if ($is_critical) {
            set $limit_connection "";
        }
    }
}
```

**Effectiveness Analysis**:

| Metric | Target | Actual | Assessment |
|--------|--------|--------|------------|
| Connection rejection rate | <0.1% | Measured during peak | TBD |
| Gateway CPU at limit | <70% | Measured under load | TBD |
| Health check success | 100% | Always | PASS |
| Customer impact | Minimal | Reports | TBD |

**Why 80%?**:
- Reserve 20% for health checks and critical operations
- Prevent cascading failures from resource exhaustion
- Industry standard for safety margins
- Allows headroom for sudden traffic spikes

**Potential Issues**:
1. **Too Conservative**: May reject valid connections prematurely
   - Mitigation: Auto-scale Nginx pods before hitting 80%

2. **Too Aggressive**: 20% may be insufficient for health check storms
   - Mitigation: Dedicated health check endpoints with separate limits

**Recommendations**:
- Implement predictive scaling to hit 70% before 80%
- Add per-service connection limits (different services have different patterns)
- Monitor and tune based on production metrics

### 2.3 Emergency Mode (95% Cluster Usage)

**Trigger**:
```lua
function check_emergency_mode()
    local cluster_usage = redis.call("GET", "cluster:usage_ratio")
    local iops_usage = get_prometheus_metric("storage_iops_usage_ratio")
    local bw_usage = get_prometheus_metric("storage_bandwidth_usage_ratio")

    local max_usage = math.max(cluster_usage, iops_usage, bw_usage)

    if max_usage >= 0.95 then
        return true
    end
    return false
end
```

**Response Strategy**:

```lua
function activate_emergency_mode()
    -- 1. Stop new quota allocations
    redis.call("SET", "cluster:allocations_enabled", "false")

    -- 2. Aggressive token reclamation
    local apps = redis.call("SMEMBERS", "apps:active")
    for _, app_id in ipairs(apps) do
        local last_used = redis.call("HGET", "app:" .. app_id, "last_deduction")
        local idle_time = ngx.time() - tonumber(last_used)

        -- Reclaim from apps idle > 5 minutes
        if idle_time > 300 then
            local tokens = redis.call("HGET", "app:" .. app_id, "tokens")
            redis.call("HSET", "app:" .. app_id, "tokens", 0)
            redis.call("INCRBY", "cluster:reclaimed", tokens)
        end
    end

    -- 3. Prioritize operations
    set_operation_priority("GET", 1)     -- High priority
    set_operation_priority("LIST", 2)    -- Medium priority
    set_operation_priority("PUT", 3)     -- Low priority
    set_operation_priority("DELETE", 3)  -- Low priority

    -- 4. Return 503 with Retry-After
    redis.call("SET", "cluster:retry_after", "60")

    -- 5. Alert operations
    send_alert("emergency_mode", {
        usage = max_usage,
        timestamp = ngx.time()
    })
end
```

**Priority Queue Implementation**:
```lua
-- Priority-based token cost multiplier
PRIORITY_MULTIPLIER = {
    CRITICAL = 0.1,   -- 10% of normal cost
    HIGH = 0.5,       -- 50% of normal cost
    NORMAL = 1.0,     -- Normal cost
    LOW = 2.0         -- 2x normal cost
}

function calculate_priority_cost(operation, user_tier)
    local base_cost = C_base[operation]
    local priority = get_operation_priority(operation)

    -- Critical infrastructure (health checks) always succeed
    if operation == "HEALTH_CHECK" then
        return 0
    end

    -- Enterprise users get priority
    if user_tier == "enterprise" then
        priority = PRIORITY_MULTIPLIER.HIGH
    elseif user_tier == "free" then
        priority = PRIORITY_MULTIPLIER.LOW
    end

    return base_cost * priority
end
```

**Exit Emergency Mode**:
- Trigger: Usage drops below 85% (hysteresis prevents oscillation)
- Actions:
  1. Gradual restoration of normal operation (10% → 50% → 100%)
  2. Restore reclaimed tokens to original apps
  3. Normalize operation priorities
  4. Remove Retry-After headers

**Effectiveness**: MEDIUM-HIGH
- Pros: Prevents cluster overload, prioritizes critical operations
- Cons: Poor user experience during emergency, potential revenue impact
- Recovery: Gradual over 10-15 minutes

---

## 3. Disaster Recovery

### 3.1 Redis Cluster Recovery

#### Scenario 1: Single Node Failure

**Detection**:
```bash
# Redis cluster health check
redis-cli --cluster check <cluster_host>:<port>
```

**Automatic Recovery** (Redis Cluster):
1. Replica detects master failure (down-after-milliseconds)
2. Replica election (via Raft consensus)
3. Replica promoted to master (automated)
4. Cluster reshards to assign new replicas

**RTO (Recovery Time Objective)**: < 30 seconds
**RPO (Recovery Point Objective)**: 0 seconds (synchronous replication)

**Token Reconstruction**:
```lua
-- Token buckets already replicated to new master
-- No reconstruction needed for synchronous replicas
-- For asynchronous replicas, minimal data loss possible

-- Verify token integrity after failover
function verify_token_integrity()
    local apps = redis.call("SMEMBERS", "apps:active")

    for _, app_id in ipairs(apps) do
        -- Check for negative token counts (data corruption)
        local tokens = redis.call("HGET", "app:" .. app_id, "tokens")

        if tokens < 0 then
            -- Correct to zero
            redis.call("HSET", "app:" .. app_id, "tokens", 0)
            log_correction(app_id, tokens, 0)
        end
    end
end
```

#### Scenario 2: Complete Redis Cluster Outage

**Detection**: All Redis nodes unreachable > 30 seconds

**Recovery Strategy**: Sync vs Reset

##### Option A: Sync Strategy (Preferred)

**Approach**: Reconstruct token state from L3 caches and application logs

**Process**:
1. Restore Redis cluster from backup (last consistent snapshot)
2. Aggregate L3 cache state from all Nginx gateways
3. Cross-reference with application audit logs
4. Reconstruct token buckets to best-known state
5. Reconcile differences over time

```lua
-- Token reconstruction from L3 caches
function reconstruct_tokens_from_l3()
    local gateways = get_all_nginx_gateways()

    for _, gateway in ipairs(gateways) do
        -- Query L3 cache state via admin API
        local l3_state = http.get(gateway .. "/rate_limit/local_state")

        for app_id, user_tokens in pairs(l3_state) do
            for user_id, tokens in pairs(user_tokens) do
                -- Aggregate token usage
                local key = "user:" .. app_id .. ":" .. user_id

                -- Calculate expected token count
                local quota = redis.call("HGET", key, "quota") or 10000
                local used = tokens.deducted
                local expected = quota - used

                -- Reconstruct
                redis.call("HSET", key, "tokens", expected)
            end
        end
    end
end
```

**RTO**: 5-15 minutes
**RPO**: 5 minutes (batch flush interval)
**Complexity**: HIGH

##### Option B: Reset Strategy (Fallback)

**Approach**: Reset all token buckets to capacity, accept temporary overages

**Process**:
1. Restore Redis cluster from backup
2. Reset all token buckets to full capacity
3. Enable "high water mark" mode (track overages)
4. Gradual enforcement restoration

```lua
-- Reset all token buckets to capacity
function reset_token_buckets()
    local apps = redis.call("SMEMBERS", "apps:active")

    for _, app_id in ipairs(apps) do
        local capacity = redis.call("HGET", "app:" .. app_id, "capacity")

        -- Reset to full capacity
        redis.call("HSET", "app:" .. app_id, "tokens", capacity)

        -- Enable high water mark tracking
        redis.call("HSET", "app:" .. app_id, "high_water_mark", 0)
        redis.call("HSET", "app:" .. app_id, "overage", 0)
    end
end
```

**RTO**: 2-5 minutes
**RPO**: Data loss (usage during outage not tracked)
**Complexity**: LOW

**Recommendation**: Start with Option B, upgrade to Option A for critical applications

#### Scenario 3: Redis Cluster Corruption

**Detection**: Redis checksum errors, inconsistent data across replicas

**Recovery Strategy**:
1. Immediate fail-open activation
2. Redis cluster rebuild from scratch
3. Seed from latest consistent backup
4. Run reconciliation for 24 hours
5. Manual verification before returning to normal

```lua
-- Corruption detection
function detect_corruption()
    -- Compare replicas
    local master_value = redis.call("GET", "cluster:tokens")
    local replica_values = {}

    for _, replica in ipairs(replicas) do
        local replica_value = replica:GET("cluster:tokens")
        table.insert(replica_values, replica_value)
    end

    -- Check for consistency
    for _, value in ipairs(replica_values) do
        if value ~= master_value then
            activate_fail_open_mode("corruption_detected")
            send_critical_alert("Redis corruption detected")
            return true
        end
    end

    return false
end
```

### 3.2 Multi-Region Deployment

#### Architecture

```
                    Global Control Plane
                           |
        +----------+-------+-------+----------+
        |          |               |          |
   Region A    Region B      Region C    Region D
        |          |               |          |
   Redis A    Redis B       Redis C    Redis D
   Nginx A    Nginx B       Nginx C    Nginx D
```

#### Token Distribution Strategy

**Option 1: Independent Regional Pools** (Recommended)
```lua
-- Each region has independent token pool
REGIONAL_POOLS = {
    ["us-east-1"] = { capacity = 100000, rate = 1000 },
    ["us-west-2"] = { capacity = 80000, rate = 800 },
    ["eu-central-1"] = { capacity = 60000, rate = 600 },
    ["ap-southeast-1"] = { capacity = 40000, rate = 400 }
}

function deduct_regional(app_id, user_id, region, cost)
    local pool = REGIONAL_POOLS[region]

    -- Check regional pool
    local tokens = redis.call("HGET", "region:" .. region .. ":app:" .. app_id, "tokens")

    if tokens >= cost then
        redis.call("HINCRBY", "region:" .. region .. ":app:" .. app_id, "tokens", -cost)
        return true
    else
        -- Borrow from other regions (cross-region penalty)
        return borrow_cross_region(app_id, user_id, region, cost)
    end
end
```

**Pros**:
- No cross-region latency
- Region isolation (failure doesn't spread)
- Simple to operate

**Cons**:
- Underutilization in one region can't help another
- Requires capacity planning per region

**Option 2: Global Pool with Regional Caches**
```lua
-- Single global pool, regional L3 caches
function deduct_global_cached(app_id, user_id, region, cost)
    -- Try local cache first
    local local_tokens = get_local_cache(app_id, user_id)

    if local_tokens >= cost then
        deduct_local(app_id, user_id, cost)
        return true
    end

    -- Fall back to global Redis (higher latency)
    local global_tokens = redis.call("HGET", "global:app:" .. app_id, "tokens")

    if global_tokens >= cost then
        redis.call("HINCRBY", "global:app:" .. app_id, "tokens", -cost)

        -- Replenish local cache (larger batch for cross-region)
        set_local_cache(app_id, user_id, cost * 10)
        return true
    end

    return false
end
```

**Pros**:
- Better global resource utilization
- No region starvation
- Centralized control

**Cons**:
- Cross-region latency (50-200ms)
- Global Redis becomes bottleneck
- Cross-region failure propagation

**Recommendation**: Option 1 (Independent Pools) for latency-sensitive workloads

### 3.3 Backup and Restore Procedures

#### Redis Cluster Backup Strategy

**Automated Backups**:
```bash
#!/bin/bash
# redis_backup.sh - Daily automated backup

BACKUP_DIR="/data/redis/backups"
DATE=$(date +%Y%m%d_%H%M%S)
REDIS_HOST="redis-cluster.example.com"
S3_BUCKET="s3://redis-backups/example-com"

# Create RDB snapshot
redis-cli -h $REDIS_HOST BGSAVE

# Wait for BGSAVE to complete
while [ $(redis-cli -h $REDIS_HOST LASTSAVE) -eq $LASTSAVE ]; do
    sleep 1
done

# Copy RDB file to backup directory
cp /var/lib/redis/dump.rdb $BACKUP_DIR/dump_$DATE.rdb

# Upload to S3 with encryption
aws s3 cp $BACKUP_DIR/dump_$DATE.rdb $S3_BUCKET/dump_$DATE.rdb \
    --storage-class STANDARD_IA \
    --server-side-encryption AES256

# Retention: Keep daily for 30 days, weekly for 12 weeks
find $BACKUP_DIR -name "dump_*.rdb" -mtime +30 -delete

echo "Backup completed: dump_$DATE.rdb"
```

**Restore Procedure**:
```bash
#!/bin/bash
# redis_restore.sh - Restore from backup

BACKUP_FILE=$1  # e.g., dump_20251231_120000.rdb
REDIS_HOST="redis-cluster.example.com"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# 1. Stop Redis Cluster
redis-cli -h $REDIS_HOST CLUSTER FAILOVER FORCE
redis-cli -h $REDIS_HOST SHUTDOWN NOSAVE

# 2. Download backup from S3
aws s3 cp s3://redis-backups/example-com/$BACKUP_FILE /tmp/$BACKUP_FILE

# 3. Verify backup integrity
redis-check-rdb /tmp/$BACKUP_FILE
if [ $? -ne 0 ]; then
    echo "Backup corrupted!"
    exit 1
fi

# 4. Replace current RDB file
cp /tmp/$BACKUP_FILE /var/lib/redis/dump.rdb

# 5. Start Redis Cluster
redis-server /etc/redis/redis.conf

# 6. Verify cluster health
redis-cli --cluster check $REDIS_HOST

echo "Restore completed: $BACKUP_FILE"
```

#### RTO and RPO Targets

| Scenario | RTO | RPO | Backup Method |
|----------|-----|-----|---------------|
| Single node failure | <30s | 0s | Automatic replica promotion |
| Complete cluster failure | 15min | 5min | RDB restore + token reconstruction |
| Region failure | 30min | 5min | Failover to another region |
| Data corruption | 1hour | 1hour | Point-in-time recovery |

---

## 4. Observability

### 4.1 Key Metrics

#### L1: Cluster Level

```promql
# Cluster token utilization
cluster_token_utilization =
    (cluster_tokens_total - cluster_tokens_available) / cluster_tokens_total

# Cluster replenishment rate
cluster_replenish_rate = rate(cluster_tokens_replenished_total[5m])

# Emergency mode status
cluster_emergency_mode = cluster_emergency_mode_enabled * 100

# Cross-app token borrowing rate
cross_app_borrow_rate = rate(token_borrow_total[5m])

# Token reclamation rate
token_reclamation_rate = rate(token_reclaimed_total[5m])
```

**Dashboard Queries**:
```promql
# Cluster health overview
sum(cluster_tokens_available) by (cluster_id) /
sum(cluster_tokens_total) by (cluster_id)

# Apps approaching quota
topk(10,
    sum by (app_id) (
        app_tokens_deducted_total / app_tokens_quota
    )
)
```

#### L2: Application Level

```promql
# Per-app token usage
app_token_usage_ratio{app_id="$app_id"} =
    app_tokens_deducted{app_id="$app_id"} / app_tokens_quota{app_id="$app_id"}

# Per-app rejection rate
app_rejection_rate{app_id="$app_id"} =
    rate(app_tokens_rejected_total{app_id="$app_id"}[5m]) /
    rate(app_requests_total{app_id="$app_id"}[5m])

# Per-operation cost breakdown
app_cost_by_operation{app_id="$app_id"} =
    sum by (operation) (
        rate(app_tokens_deducted_total{app_id="$app_id"}[5m])
    )

# User-level quota utilization
user_quota_utilization{app_id="$app_id"} =
    user_tokens_deducted{app_id="$app_id", user_id="$user_id"} /
    user_tokens_quota{app_id="$app_id", user_id="$user_id"}
```

#### L3: Local Cache Level

```promql
# L3 cache hit rate
l3_cache_hit_rate =
    rate(l3_cache_hits_total[5m]) /
    (rate(l3_cache_hits_total[5m]) + rate(l3_cache_misses_total[5m]))

# L3 batch flush latency
l3_flush_latency_p95 =
    histogram_quantile(0.95,
        sum(rate(l3_flush_latency_seconds_bucket[5m])) by (le)
    )

# L3 batch size distribution
l3_batch_size =
    avg(l3_batch_operations_count)

# Redis fallback rate (requests hitting Redis directly)
redis_fallback_rate =
    rate(redis_requests_total[5m]) /
    rate(total_requests_total[5m])
```

#### Redis Health

```promql
# Redis connection health
redis_connected_clients
redis_rejected_connections
redis_blocked_clients

# Redis latency
redis_command_latency_p95{command="HINCRBY"}
redis_command_latency_p99{command="HGET"}

# Redis cluster health
redis_cluster_state_ok
redis_cluster_slots_assigned
redis_cluster_slots_ok

# Redis error rate
redis_errors_total =
    rate(redis_connection_errors_total[5m]) +
    rate(redis_timeout_errors_total[5m])
```

#### End-to-End Metrics

```promql
# Token check latency (full request path)
token_check_latency_p95 =
    histogram_quantile(0.95,
        sum(rate(token_check_latency_seconds_bucket[5m])) by (le)
    )

# Overall rejection rate (all layers)
overall_rejection_rate =
    sum(rate(requests_rejected_total[5m])) /
    sum(rate(requests_total[5m]))

# Rejection by layer
rejections_by_layer =
    sum by (layer) (
        rate(requests_rejected_total[5m])
    )

# Cost efficiency (tokens per resource unit)
cost_efficiency =
    sum(tokens_consumed_total) /
    sum(storage_iops_total + storage_bandwidth_bytes_total / 1e6)
```

### 4.2 Distributed Tracing

#### Trace Context Propagation

```nginx
# Inject trace context into Lua
http {
    init_by_lua_block {
        -- Initialize OpenTelemetry
        local tracer = require "opentelemetry".tracer
        _GLOBAL_TRACER = tracer("rate-limiter")
    }

    access_by_lua_block {
        local tracer = _GLOBAL_TRACER
        local span = tracer:start_span("token_check")

        -- Inject context for downstream services
        ngx.var.trace_id = span:context().trace_id
        ngx.var.span_id = span:context().span_id

        -- L3 cache check
        local l3_span = span:start_span("l3_cache_check")
        local l3_hit = check_l3_cache()
        l3_span:set_attribute("l3.cache.hit", l3_hit)
        l3_span:finish()

        if not l3_hit then
            -- Redis check
            local redis_span = span:start_span("redis_check")
            redis_span:set_attribute("redis.operation", "HGET")

            local redis_ok, redis_latency = check_redis()
            redis_span:set_attribute("redis.latency_ms", redis_latency)
            redis_span:finish()
        end

        span:finish()
    }
}
```

#### Trace Spans

```
[HTTP Request] (root)
├── [Extract Auth Token] (~50μs)
├── [Calculate Cost] (~100μs)
│   ├── Parse operation
│   ├── Calculate base cost
│   └── Calculate bandwidth cost
├── [L3 Cache Check] (~200μs)
│   ├── Get local cache
│   └── Check sufficient tokens
├── [Redis Check] (on cache miss, ~5-20ms)
│   ├── Connect to Redis
│   ├── Execute HGET
│   └── Parse response
├── [Token Deduction] (~500μs)
│   └── Update local cache
├── [Batch Accumulation] (~100μs)
│   └── Add to batch buffer
└── [Request Forwarding] (~50μs)
```

### 4.3 Alerting

#### Alert Rules

```yaml
groups:
  - name: rate_limiter_critical
    interval: 30s
    rules:
      # Redis cluster down
      - alert: RedisClusterDown
        expr: redis_cluster_state_ok == 0
        for: 1m
        labels:
          severity: critical
          team: platform
        annotations:
          summary: "Redis cluster is down"
          description: "Redis cluster state: {{ $value }}. Fail-open mode activated."
          runbook: "https://runbooks.example.com/redis-down"

      # High rejection rate
      - alert: HighRejectionRate
        expr: |
          sum(rate(requests_rejected_total[5m])) /
          sum(rate(requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
          team: sre
        annotations:
          summary: "High rejection rate: {{ $value | humanizePercentage }}"
          description: "More than 5% of requests are being rejected"
          runbook: "https://runbooks.example.com/high-rejection-rate"

      # Token drift too high
      - alert: TokenDriftExceeded
        expr: |
          abs(token_expected - token_actual) / token_expected > 0.10
        for: 10m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Token drift: {{ $value | humanizePercentage }}"
          description: "Token drift exceeds 10% threshold"
          runbook: "https://runbooks.example.com/token-drift"

  - name: rate_limiter_warning
    interval: 1m
    rules:
      # L3 cache miss rate high
      - alert: HighL3CacheMissRate
        expr: |
          rate(l3_cache_misses_total[5m]) /
          (rate(l3_cache_hits_total[5m]) + rate(l3_cache_misses_total[5m])) > 0.50
        for: 10m
        labels:
          severity: info
          team: platform
        annotations:
          summary: "L3 cache miss rate: {{ $value | humanizePercentage }}"
          description: "Consider increasing L3 cache size"

      # Redis latency high
      - alert: HighRedisLatency
        expr: |
          histogram_quantile(0.95,
            sum(rate(redis_command_latency_seconds_bucket{command="HINCRBY"}[5m])) by (le)
          ) > 0.05
        for: 5m
        labels:
          severity: warning
          team: platform
        annotations:
          summary: "Redis P95 latency: {{ $value }}s"
          description: "Redis latency exceeds 50ms threshold"

      # Emergency mode activated
      - alert: EmergencyModeActivated
        expr: cluster_emergency_mode_enabled == 1
        for: 0m
        labels:
          severity: critical
          team: sre
        annotations:
          summary: "Emergency mode activated"
          description: "Cluster usage ≥95%. Request throttling active."
          runbook: "https://runbooks.example.com/emergency-mode"
```

#### Escalation Policy

```
P1 (Critical) → Page on-call immediately → Escalate to manager after 15min
P2 (High)     → Page on-call immediately → Escalate to manager after 30min
P3 (Medium)   -> Create ticket, notify Slack → Escalate after 1 hour
P4 (Low)      -> Create ticket, daily digest → No escalation
```

---

## 5. Capacity Planning

### 5.1 Cluster Quota Sizing

**Methodology**: Token-based resource modeling

**Step 1: Measure Resource Constraints**
```promql
# Current storage cluster IOPS capacity
storage_iops_capacity = 100000  # ops/sec
storage_bandwidth_capacity = 10  # Gbps

# Current utilization (peak)
storage_iops_peak = rate(storage_iops_total[7d]) * 1.2  # 20% growth margin
storage_bandwidth_peak = rate(storage_bandwidth_bytes_total[7d]) * 1.2
```

**Step 2: Calculate Token Capacity**
```
# Token = Normalized resource unit
1 Token = 1 IOPS operation OR 4KB bandwidth

Total cluster token capacity = min(
    storage_iops_capacity,
    storage_bandwidth_capacity * (1e9 / 4096)  # Convert Gbps to tokens/sec
)

# Example:
# IOPS: 100,000 ops/sec
# Bandwidth: 10 Gbps = 10 * 1e9 / 4096 = 2,441,406 tokens/sec

# IOPS is bottleneck
cluster_token_capacity = 100,000 tokens/sec
cluster_token_burst = 200,000 tokens (2x rate for 1 second)
```

**Step 3: Reserve for Overhead**
```
# Reserve 20% for system operations (health checks, replication, GC)
cluster_quota_usable = cluster_token_capacity * 0.80
cluster_quota_system = cluster_token_capacity * 0.20
```

**Step 4: Set Emergency Threshold**
```
# Emergency mode at 95% usage
cluster_emergency_threshold = cluster_quota_usable * 0.95

# Exit emergency mode at 85% (hysteresis)
cluster_normal_threshold = cluster_quota_usable * 0.85
```

### 5.2 App Quota Allocation Strategy

#### Strategy 1: Weighted Fair Queueing (WFQ)

```lua
-- Allocate based on app priority weight
APPS = {
    { id = "app_a", weight = 10, min_quota = 10000, max_quota = 50000 },
    { id = "app_b", weight = 5,  min_quota = 5000,  max_quota = 25000 },
    { id = "app_c", weight = 2,  min_quota = 2000,  max_quota = 10000 },
}

function distribute_quota_weighted()
    local total_weight = 0
    for _, app in ipairs(APPS) do
        total_weight = total_weight + app.weight
    end

    local available = cluster_quota_usable

    for _, app in ipairs(APPS) do
        -- Calculate proportional allocation
        local allocation = math.floor(available * app.weight / total_weight)

        -- Enforce min/max bounds
        allocation = math.max(app.min_quota, math.min(app.max_quota, allocation))

        redis.call("HSET", "app:" .. app.id, "quota", allocation)
        redis.call("HSET", "app:" .. app.id, "tokens", allocation)
    end
end
```

#### Strategy 2: Usage-Based Dynamic Allocation

```lua
-- Allocate based on historical usage (80th percentile)
function allocate_based_on_usage()
    local apps = redis.call("SMEMBERS", "apps:active")

    for _, app_id in ipairs(apps) do
        -- Get historical usage (last 7 days)
        local usage_p80 = get_prometheus_metric(
            "histogram_quantile(0.80, sum(rate(app_tokens_consumed_total{app_id='" .. app_id .. "'}[7d])))"
        )

        -- Add 20% growth margin
        local quota = math.ceil(usage_p80 * 1.2)

        -- Enforce cluster-wide limits
        quota = math.min(quota, cluster_quota_usable * 0.50)  -- Max 50% per app

        redis.call("HSET", "app:" .. app_id, "quota", quota)
    end
end
```

#### Strategy 3: Tiered Service Levels

```lua
-- Different service tiers have different quota guarantees
TIERS = {
    platinum = {
        min_quota = 50000,
        max_quota = 100000,
        priority = 1,
        burst_multiplier = 3
    },
    gold = {
        min_quota = 10000,
        max_quota = 50000,
        priority = 2,
        burst_multiplier = 2
    },
    silver = {
        min_quota = 2000,
        max_quota = 10000,
        priority = 3,
        burst_multiplier = 1.5
    },
    bronze = {
        min_quota = 500,
        max_quota = 2000,
        priority = 4,
        burst_multiplier = 1
    }
}

function allocate_by_tier(app_id, tier)
    local config = TIERS[tier]

    -- Get current cluster load
    local cluster_load = redis.call("GET", "cluster:load_ratio")

    -- Adjust allocation based on cluster load
    local multiplier = 1.0
    if cluster_load > 0.90 then
        multiplier = 0.5  -- Reduce allocations during high load
    elseif cluster_load > 0.80 then
        multiplier = 0.8
    end

    local quota = config.min_quota * multiplier
    quota = math.min(quota, config.max_quota)

    redis.call("HSET", "app:" .. app_id, "quota", quota)
    redis.call("HSET", "app:" .. app_id, "priority", config.priority)
    redis.call("HSET", "app:" .. app_id, "burst_capacity", quota * config.burst_multiplier)
end
```

**Recommendation**: Start with Strategy 1 (Weighted Fair Queueing), evolve to Strategy 3 (Tiered Service Levels)

### 5.3 Burst Capacity Planning

**Burst Duration**: Allow burst for 60 seconds

**Burst Size Calculation**:
```
# Normal rate: 1000 tokens/sec
# Burst rate: Allow 10x normal rate for 60 seconds

burst_capacity = (normal_rate * 10 - normal_rate) * 60
              = (10000 - 1000) * 60
              = 540,000 tokens

# Bucket capacity
bucket_capacity = normal_rate + burst_capacity
                = 1000 + 540000
                = 541,000 tokens
```

**Validation**:
```lua
-- Simulate burst scenario
function simulate_burst()
    local rate = 1000  -- tokens/sec
    local capacity = 541000

    -- Burst at 10x for 60 seconds
    for t = 1, 60 do
        local deduction = rate * 10
        capacity = capacity - deduction

        if capacity < 0 then
            print("Burst capacity exhausted at t=" .. t)
            return
        end
    end

    print("Burst simulation passed: remaining capacity=" .. capacity)
end
```

### 5.4 Growth Forecasting

**Method**: Linear regression on historical data

```python
import pandas as pd
from sklearn.linear_model import LinearRegression
from datetime import datetime, timedelta

# Load historical token consumption data
df = pd.read_csv('token_consumption_history.csv',
                 parse_dates=['timestamp'])

# Aggregate daily consumption
daily = df.groupby(df['timestamp'].dt.date)['tokens_consumed'].sum().reset_index()
daily['day_num'] = (daily['timestamp'] - daily['timestamp'].min()).dt.days

# Train linear regression model
X = daily[['day_num']]
y = daily['tokens_consumed']
model = LinearRegression().fit(X, y)

# Forecast next 90 days
forecast_days = 90
last_day_num = daily['day_num'].max()
future_days = [[last_day_num + i] for i in range(1, forecast_days + 1)]
predictions = model.predict(future_days)

# Calculate required capacity (with 50% headroom)
required_capacity = predictions.max() * 1.5

print(f"Current daily consumption: {daily['tokens_consumed'].iloc[-1]:,.0f}")
print(f"Forecasted consumption (90 days): {predictions.max():,.0f}")
print(f"Recommended capacity: {required_capacity:,.0f}")
```

**Seasonality Adjustment**:
```python
# Add seasonal decomposition for weekly/monthly patterns
from statsmodels.tsa.seasonal import seasonal_decompose

# Resample to daily data
daily_ts = df.set_index('timestamp').resample('D')['tokens_consumed'].sum()

# Decompose into trend + seasonal + residual
result = seasonal_decompose(daily_ts, model='additive', period=7)  # Weekly seasonality

# Forecast using trend + seasonal component
trend_forecast = model.predict(future_days)
seasonal_component = result.seasonal[:forecast_days]
final_forecast = trend_forecast + seasonal_component.values
```

---

## 6. Incident Response

### 6.1 Runbooks

#### Runbook: Redis Cluster Degraded Performance

**Title**: Redis Cluster High Latency (>100ms)
**Severity**: P2 (High)
**Owner**: Platform Team

**Symptoms**:
- API latency P95 > 200ms
- Increased Redis errors in logs
- Dashboard alert: HighRedisLatency

**Diagnosis**:
```bash
# 1. Check Redis cluster health
redis-cli --cluster check redis-cluster.example.com

# 2. Check Redis latency
redis-cli --latency-history -h redis-cluster.example.com -i 10

# 3. Check Redis slow log
redis-cli --latency-dist -h redis-cluster.example.com

# 4. Check Redis INFO
redis-cli INFO stats
redis-cli INFO commandstats

# 5. Check network latency
ping -c 10 redis-cluster.example.com
mtr -r -c 10 redis-cluster.example.com
```

**Triage**:

| Symptom | Possible Cause | Action |
|---------|----------------|--------|
| Latency >100ms, error rate <5% | Redis CPU high | Check Redis CPU, scale if needed |
| Latency >100ms, error rate 5-20% | Network issues | Check network, verify routing |
| Latency >500ms, error rate >20% | Redis overload | Activate fail-open |
| Latency spikes every 5 seconds | Lua script slow | Optimize Lua scripts, add batching |

**Resolution Steps**:

1. **Immediate (0-5 min)**:
   ```bash
   # Activate Level 2 degradation
   curl -X POST http://nginx-gateway/admin/degradation/level2

   # Verify activation
   curl http://nginx-gateway/admin/degradation/status
   ```

2. **Short-term (5-15 min)**:
   ```bash
   # Check Redis CPU usage
   top -p $(pgrep redis-server)

   # If CPU >80%, scale Redis cluster
   kubectl scale statefulset redis-cluster --replicas=6

   # Monitor cluster rebalancing
   redis-cli --cluster check redis-cluster.example.com
   ```

3. **Medium-term (15-60 min)**:
   ```bash
   # Check for slow Lua scripts
   redis-cli SLOWLOG GET 20

   # If scripts slow, optimize or add batching
   # Example: Batch HINCRBY operations
   ```

4. **Long-term (>60 min)**:
   - Review Redis capacity planning
   - Implement auto-scaling based on CPU/memory
   - Add Redis read replicas for offloading

**Verification**:
```promql
# Check latency returned to normal
histogram_quantile(0.95, sum(rate(redis_command_latency_seconds_bucket[5m])) by (le)) < 0.01

# Check error rate reduced
rate(redis_errors_total[5m]) / rate(redis_commands_total[5m]) < 0.01
```

**Escape**: If latency >500ms for >10 min, activate complete fail-open:
```bash
curl -X POST http://nginx-gateway/admin/fail-open/activate
```

---

#### Runbook: Token Exhaustion

**Title**: Application or User Token Bucket Exhausted
**Severity**: P3 (Medium)
**Owner**: SRE Team

**Symptoms**:
- Increased 429 responses
- Customer reports: "rate limit exceeded"
- Dashboard alert: HighRejectionRate for specific app_id

**Diagnosis**:
```bash
# 1. Check token exhaustion by app
redis-cli HGET app:app123 tokens
redis-cli HGET app:app123 quota
redis-cli HGET app:app123 last_replenish

# 2. Check cluster-wide status
redis-cli GET cluster:usage_ratio
redis-cli GET cluster:available

# 3. Check recent requests for app
redis-cli --scan --pattern "user:app123:*" | head -20

# 4. Check for abnormal traffic patterns
kubectl logs -l app=nginx-gateway --tail=1000 | grep app123
```

**Triage**:

| Scenario | Cause | Action |
|----------|-------|--------|
| Single app exhausted, cluster has capacity | Legitimate traffic spike | Approve token borrowing |
| Single app exhausted, cluster at 95% | Cluster-wide overload | Emergency mode, no borrowing |
| Multiple users in same app exhausted | App-specific attack or bug | Investigate app behavior |
| All apps exhausted simultaneously | Storage cluster overload | Emergency mode, scale storage |

**Resolution Steps**:

1. **Single App Exhaustion (Cluster has capacity)**:
   ```lua
   -- Approve token borrowing
   function approve_emergency_loan(app_id, amount)
       local cluster_available = redis.call("GET", "cluster:available")

       if cluster_available >= amount then
           -- Approve loan with 50% interest (emergency rate)
           local repayment = amount * 1.5

           redis.call("DECRBY", "cluster:available", amount)
           redis.call("HINCRBY", "app:" .. app_id, "tokens", amount)
           redis.call("HINCRBY", "app:" .. app_id, "debt", repayment)

           -- Log for audit
           log_emergency_loan(app_id, amount, repayment)

           return true
       end

       return false
   end
   ```

2. **Cluster-Wide Exhaustion**:
   ```bash
   # Activate emergency mode
   redis-cli SET cluster:emergency_mode "true"
   redis-cli SET cluster:retry_after "60"

   # Notify all gateways
   for gateway in $(kubectl get pods -l app=nginx-gateway -o name); do
       kubectl exec $gateway -- curl -X POST http://localhost/admin/emergency
   done
   ```

3. **Investigate Anomalous Traffic**:
   ```bash
   # Check for suspicious patterns (e.g., single user consuming all tokens)
   redis-cli --scan --pattern "user:app123:*" | \
     xargs -I {} redis-cli HGETALL {} | \
     jq -s 'sort_by(.deducted_total) | reverse | .[0:10]'

   # If single user responsible, consider rate limiting that user
   ```

**Verification**:
```promql
# Check rejection rate decreased
sum by (app_id) (rate(requests_rejected_total{app_id="app123"}[5m])) < 10

# Check tokens available
redis-cli HGET app:app123 tokens > 1000
```

**Post-Incident**:
- Review why quota was insufficient
- Consider increasing app quota for next cycle
- Add alert for earlier detection (80% threshold)
- Document in postmortem

---

#### Runbook: Connection Limit Hit

**Title**: Nginx Worker Connections at 80% Limit
**Severity**: P2 (High)
**Owner**: Platform Team

**Symptoms**:
- Increased 503 errors
- Nginx logs: "worker_connections are not enough"
- Dashboard alert: NginxConnectionLimit

**Diagnosis**:
```bash
# 1. Check current connections
curl http://nginx-gateway/nginx_status

# 2. Check worker connection usage
kubectl exec -it nginx-gateway-pod -- ash -c "cat /var/run/nginx.pid"
worker_pid=$(kubectl exec nginx-gateway-pod -- cat /var/run/nginx.pid)
kubectl exec nginx-gateway-pod -- ls -l /proc/$worker_pid/fd | wc -l

# 3. Check connection limit
kubectl exec nginx-gateway-pod -- nginx -T 2>&1 | grep worker_connections

# 4. Identify top connection consumers
kubectl exec nginx-gateway-pod -- netstat -an | \
  grep ESTABLISHED | \
  awk '{print $5}' | \
  cut -d: -f1 | \
  sort | uniq -c | sort -rn | head -20
```

**Triage**:

| Symptom | Cause | Action |
|---------|-------|--------|
| Connections steady at 80% | Normal high traffic | Scale Nginx pods |
| Connections spike suddenly | DDoS or traffic surge | Enable rate limiting, scale |
| Connections growing slowly | Connection leak | Check for idle connections |
| Per-IP limit hit | Abusive client | Ban offending IP |

**Resolution Steps**:

1. **Immediate (0-5 min)**:
   ```bash
   # Scale Nginx horizontally
   kubectl scale deployment nginx-gateway --replicas=10

   # Verify new pods ready
   kubectl rollout status deployment nginx-gateway
   ```

2. **Short-term (5-15 min)**:
   ```bash
   # Check for connection leaks (idle connections)
   kubectl exec nginx-gateway-pod -- netstat -an | \
     grep ESTABLISHED | \
     awk '{if ($6 < 1000) print}' | wc -l

   # If many idle connections, reduce keepalive timeout
   kubectl exec nginx-gateway-pod -- sed -i 's/keepalive_timeout 65;/keepalive_timeout 10;/' /etc/nginx/nginx.conf
   kubectl exec nginx-gateway-pod -- nginx -s reload
   ```

3. **Medium-term (15-60 min)**:
   ```bash
   # Enable aggressive connection cleanup
   # Add to nginx.conf:
   # proxy_read_timeout 30s;
   # proxy_send_timeout 30s;
   # send_timeout 10s;

   # Identify and ban abusive IPs (if DDoS)
   kubectl logs -l app=nginx-gateway --tail=10000 | \
     awk '{print $1}' | sort | uniq -c | sort -rn | head -10 > top_ips.txt

   # Ban IPs with >1000 connections
   while read count ip; do
     if [ $count -gt 1000 ]; then
       kubectl exec nginx-gateway-pod -- iptables -A INPUT -s $ip -j DROP
     fi
   done < top_ips.txt
   ```

4. **Long-term (>60 min)**:
   - Implement auto-scaling based on connection count
   - Add per-IP connection limits (lower than global limit)
   - Consider upgrading instance types (more CPU/memory)

**Verification**:
```promql
# Check connection count decreased
nginx_connections_current < nginx_connections_limit * 0.7

# Check 503 error rate decreased
rate(nginx_503_errors_total[5m]) < 1
```

**Prevention**:
- Set up auto-scaling:
  ```yaml
  apiVersion: autoscaling/v2
  kind: HorizontalPodAutoscaler
  metadata:
    name: nginx-gateway-hpa
  spec:
    scaleTargetRef:
      apiVersion: apps/v1
      kind: Deployment
      name: nginx-gateway
    minReplicas: 3
    maxReplicas: 20
    metrics:
    - type: Pods
      pods:
        metric:
          name: connections
        target:
          type: AverageValue
          averageValue: "500"  # Scale at 500 connections per pod
  ```

---

#### Runbook: Emergency Mode Triggered

**Title**: Cluster Usage ≥95%, Emergency Mode Active
**Severity**: P1 (Critical)
**Owner**: SRE Team + Storage Team

**Symptoms**:
- Dashboard alert: EmergencyModeActivated
- Increased 503 responses with Retry-After header
- Storage cluster metrics show IOPS or bandwidth saturation

**Diagnosis**:
```bash
# 1. Check cluster utilization
redis-cli GET cluster:usage_ratio
redis-cli GET cluster:emergency_mode

# 2. Check storage metrics
curl http://prometheus/api/v1/query?query=storage_iops_usage_ratio
curl http://prometheus/api/v1/query?query=storage_bandwidth_usage_ratio

# 3. Identify top consumers
redis-cli --scan --pattern "app:*" | \
  xargs -I {} sh -c 'echo "{} $(redis-cli HGET {} tokens) $(redis-cli HGET {} quota)"' | \
  awk '$2 < $3 * 0.05' | \
  sort -t' ' -k2 -n

# 4. Check for abnormal operations (e.g., large PUTs)
kubectl logs -l app=nginx-gateway --tail=1000 | grep "PUT" | grep "size=[0-9]\{8,\}"
```

**Triage**:

| Scenario | Cause | Action |
|----------|-------|--------|
| Storage IOPS at 100% | Small object access spike | Throttle PUT/LIST, prioritize GET |
| Storage bandwidth at 100% | Large file transfers | Throttle large PUTs, reduce bandwidth cost weight |
| Single app consuming 50%+ | Abusive or buggy app | Revoke tokens, investigate app |
| All apps consuming equally | Legitimate growth | Scale storage cluster |

**Resolution Steps**:

1. **Immediate (0-5 min)**:
   ```bash
   # Verify emergency mode active
   redis-cli GET cluster:emergency_mode  # Should return "true"

   # Check operation priority settings
   redis-cli HGETALL operation_priority

   # Verify Retry-After header set
   redis-cli GET cluster:retry_after  # Should return "60"

   # Notify teams
   send_alert("#sre", "Emergency mode activated: cluster usage ≥95%")
   send_alert("#storage", "Storage cluster saturated, please investigate")
   ```

2. **Short-term (5-15 min)**:
   ```bash
   # Identify top consumers
   for app in $(redis-cli --scan --pattern "app:*"); do
     tokens=$(redis-cli HGET $app tokens)
     quota=$(redis-cli HGET $app quota)
     used_ratio=$(echo "scale=2; ($quota - $tokens) / $quota" | bc)
     echo "$app $used_ratio" >> /tmp/app_usage.txt
   done

   sort -k2 -rn /tmp/app_usage.txt | head -10

   # If single app dominating, revoke tokens
   redis-cli HSET app:offending_app tokens 0
   redis-cli HSET app:offending_app revoked "true"
   ```

3. **Medium-term (15-60 min)**:
   ```bash
   # Scale storage cluster (if possible)
   kubectl scale statefulset storage-node --replicas=20

   # If bandwidth saturation, adjust cost model
   redis-cli HSET cost_model:PUT C_bw "3.0"  # Increase bandwidth weight

   # If IOPS saturation, adjust base costs
   redis-cli HSET cost_model:PUT C_base "10"  # Increase IOPS cost
   redis-cli HSET cost_model:LIST C_base "5"
   ```

4. **Long-term (>60 min)**:
   - Review storage capacity planning
   - Consider adding more storage nodes
   - Implement traffic shaping at application level
   - Conduct postmortem

**Exit Emergency Mode**:
```bash
# Check if usage dropped below 85%
usage=$(redis-cli GET cluster:usage_ratio)
if (( $(echo "$usage < 0.85" | bc -l) )); then
    # Deactivate emergency mode
    redis-cli SET cluster:emergency_mode "false"
    redis-cli DEL cluster:retry_after

    # Restore normal operation priorities
    redis-cli HDEL operation_priority

    # Notify teams
    send_alert("#sre", "Emergency mode deactivated: usage dropped to ${usage}")
fi
```

**Verification**:
```promql
# Check cluster usage decreased
cluster_token_utilization < 0.85

# Check 503 rate decreased
sum(rate(requests_503_total[5m])) < 10
```

**Post-Incident**:
- Conduct postmortem within 24 hours
- Identify root cause (capacity vs abnormal traffic)
- Update capacity plan if growth-related
- Implement preventive measures if avoidable

---

### 6.2 Post-Incident Review Process

#### Postmortem Template

```markdown
# Postmortem: Emergency Mode Activation - [Date]

## Summary
[Brief description of what happened]

## Impact
- Duration: [Start time] - [End time] ([X] minutes)
- Affected services: [List]
- Customer impact: [Internal/External/Both]
- Error rate: [X%]
- Request rejection rate: [X%]

## Root Cause
[What caused the incident? Use 5 Whys if needed]

### Timeline
| Time | Event | Duration |
|------|-------|----------|
| 14:30 | Cluster usage hit 95% | - |
| 14:31 | Emergency mode activated | 1min |
| 14:32 | On-call notified | 1min |
| 14:35 | Investigation started | 3min |
| 14:40 | Top consumer identified | 5min |
| 14:42 | Revoked tokens from offending app | 2min |
| 14:45 | Usage dropped below 85% | 3min |
| 15:00 | Emergency mode deactivated | 15min |
| 15:15 | Monitoring verified normal | 15min |

**Total Incident Duration**: 45 minutes

## What Went Well
- Emergency mode activated automatically
- On-call responded within 1 minute
- Quick identification of top consumer

## What Went Wrong
- No early warning at 80% threshold
- Manual investigation took too long
- Offending app continued consuming tokens after revocation (race condition)

## Action Items
### Preventive
1. [ ] Add alert at 80% cluster usage (Owner: SRE, Due: [Date])
2. [ ] Implement auto-revocation for abusive apps (Owner: Platform, Due: [Date])
3. [ ] Review and increase app quota limits (Owner: Storage, Due: [Date])

### Detective
1. [ ] Add dashboard panel for top 10 consumers (Owner: SRE, Due: [Date])
2. [ ] Implement automated RCA analysis (Owner: Platform, Due: [Date])

### Corrective
1. [ ] Fix race condition in token revocation (Owner: Platform, Due: [Date])
2. [ ] Add app-level rate limiting as fallback (Owner: Platform, Due: [Date])

## Follow-Up Meeting
- Date: [Date]
- Attendees: SRE, Platform, Storage
- Agenda: Review action items, capacity planning update
```

---

## 7. Chaos Engineering

### 7.1 Fault Injection Test Scenarios

#### Scenario 1: Redis Node Failure

**Objective**: Verify automatic failover and token reconstruction

**Prerequisites**:
- Redis Cluster with 3 masters + 3 replicas
- Nginx gateways configured with retry logic
- Monitoring dashboard active

**Test Steps**:
```bash
# 1. Record baseline metrics
baseline_latency=$(curl -s http://prometheus/api/v1/query?query='rate(requests_total[5m])' | jq '.data.result[0].value[1]')

# 2. Kill one Redis master pod
kubectl delete pod redis-cluster-0 -n redis

# 3. Observe behavior
# Expected:
# - Replica promoted to master within 30s
# - Some requests see "MOVED" errors
# - Nginx retries automatically
# - Overall request success rate >99%

# 4. Monitor metrics
watch -n 1 'redis-cli --cluster check redis-cluster.example.com'

# 5. Verify after 5 minutes
success_rate=$(curl -s http://prometheus/api/v1/query?query='sum(rate(requests_success_total[5m])) / sum(rate(requests_total[5m]))' | jq '.data.result[0].value[1]')
echo "Success rate: $success_rate"
```

**Expected Results**:
- Failover time: < 30 seconds
- Request success rate: > 99%
- No manual intervention required
- No token corruption

**Cleanup**:
```bash
# Verify cluster healthy
redis-cli --cluster check redis-cluster.example.com
```

---

#### Scenario 2: Network Partition

**Objective**: Verify fail-open activation and graceful degradation

**Prerequisites**:
- iptables access on Nginx gateways
- Redis Cluster in multi-AZ deployment
- Alert configuration verified

**Test Steps**:
```bash
# 1. Block Redis access from one Nginx gateway
kubectl exec nginx-gateway-pod-1 -- iptables -A OUTPUT -p tcp --dport 6379 -j DROP

# 2. Monitor gateway behavior
kubectl logs -f nginx-gateway-pod-1 --tail=100

# Expected log messages:
# "Redis connection timeout, activating fail-open"
# "Switching to local-only rate limiting"

# 3. Verify requests still succeed
curl -I http://nginx-gateway-pod-1/api/v1/storage/bucket/test
# Expected: 200 OK (not 503)

# 4. Check metrics
curl http://nginx-gateway-pod-1/metrics | grep fail_open_mode
# Expected: fail_open_mode{status="active"} 1

# 5. Restore network access after 5 minutes
kubectl exec nginx-gateway-pod-1 -- iptables -D OUTPUT -p tcp --dport 6379 -j DROP

# 6. Verify gradual return to normal
# Expected:
# - Gateway reconnects to Redis
# - Syncs local state
# - Resumes normal operation
```

**Expected Results**:
- Fail-open activates within 10 seconds
- Requests continue with local-only enforcement
- No 503 errors during partition
- Automatic recovery when network restored

**Cleanup**:
```bash
# Verify iptables rules removed
kubectl exec nginx-gateway-pod-1 -- iptables -L OUTPUT
```

---

#### Scenario 3: Sudden Traffic Spike

**Objective**: Verify auto-scaling and connection limits

**Prerequisites**:
- Kubernetes HPA configured for Nginx
- Load testing tool (e.g., Locust, Vegeta)
- Monitoring active

**Test Steps**:
```bash
# 1. Record baseline pod count
baseline_pods=$(kubectl get deployment nginx-gateway -o jsonpath='{.spec.replicas}')

# 2. Start load test (10x normal traffic)
vegeta attack -targets=targets.txt -rate=10000 -duration=5m | vegeta report

# 3. Monitor auto-scaling
watch -n 5 'kubectl get pods -l app=nginx-gateway'

# Expected:
# - HPA detects CPU/connection increase
# - New pods spin up (up to maxReplicas)
# - Connection limits prevent overload

# 4. Verify connection limits working
kubectl exec nginx-gateway-pod-1 -- nginx -T 2>&1 | grep limit_conn

# 5. Check 503 rate
curl -s http://prometheus/api/v1/query?query='sum(rate(nginx_503_total[5m]))' | jq

# Expected: <1% of requests get 503

# 6. Stop load test
# Verify pods scale down after stabilization period (10 minutes)
```

**Expected Results**:
- HPA scales pods from 3 to 20 within 5 minutes
- Connection limits prevent resource exhaustion
- Request success rate >99%
- Gradual scale-down after traffic normalizes

**Cleanup**:
```bash
# Verify pods returned to baseline
kubectl get pods -l app=nginx-gateway
```

---

#### Scenario 4: Token Bucket Exhaustion

**Objective**: Verify emergency mode activation

**Prerequisites**:
- Test application with known quota
- Ability to consume tokens rapidly
- Monitoring dashboard active

**Test Steps**:
```bash
# 1. Get test app quota
redis-cli HGET app:test_app quota

# 2. Rapidly consume tokens (simulate spike)
for i in {1..1000}; do
    curl -X POST http://nginx-gateway/api/v1/storage/put \
        -H "X-App-ID: test_app" \
        -d '{"size": 1048576}' &  # 1MB PUT
done
wait

# 3. Monitor token exhaustion
watch -n 1 'redis-cli HGET app:test_app tokens'

# Expected: Tokens drop to 0

# 4. Verify 429 responses
curl -X POST http://nginx-gateway/api/v1/storage/put \
    -H "X-App-ID: test_app" \
    -d '{"size": 1048576}'

# Expected: 429 Too Many Requests
# Headers:
# Retry-After: 60
# X-RateLimit-Limit: 10000
# X-RateLimit-Remaining: 0

# 5. Check cluster-wide impact
redis-cli GET cluster:usage_ratio

# If cluster usage >95%, emergency mode should activate

# 6. Verify emergency mode
redis-cli GET cluster:emergency_mode  # Should return "true"
```

**Expected Results**:
- Test app receives 429 responses
- Emergency mode activates if cluster at 95%
- Other apps unaffected (if cluster not at emergency)
- Automatic recovery when tokens replenish

**Cleanup**:
```bash
# Replenish tokens for test app
redis-cli HSET app:test_app tokens 10000

# Deactivate emergency mode
redis-cli SET cluster:emergency_mode "false"
```

---

### 7.2 Gradual Rollback Strategy

**Scenario**: New cost model causes high rejection rate

**Rollback Procedure**:

1. **Detect Issue**:
   ```promql
   # Alert triggered
   rate(requests_rejected_total[5m]) / rate(requests_total[5m]) > 0.10
   ```

2. **Verify Root Cause**:
   ```bash
   # Check recent changes
   git log --since="1 hour ago" -- config/cost_model.yaml

   # Verify cost model deployed
   redis-cli HGETALL cost_model
   ```

3. **Gradual Rollback** (Canary-style):
   ```bash
   # Step 1: Rollback 10% of traffic to old cost model
   redis-cli SET cost_model:version:old:traffic_percentage 10

   # Step 2: Monitor for 5 minutes
   # Check if rejection rate improved

   # Step 3: If improved, increase to 50%
   redis-cli SET cost_model:version:old:traffic_percentage 50

   # Step 4: Monitor for 5 minutes

   # Step 5: If still improved, rollback 100%
   redis-cli SET cost_model:active_version "old"
   redis-cli DEL cost_model:version:old:traffic_percentage
   ```

4. **Verify Rollback**:
   ```promql
   # Check rejection rate returned to baseline
   rate(requests_rejected_total[5m]) / rate(requests_total[5m]) < 0.02
   ```

5. **Post-Rollback Actions**:
   - Conduct incident review
   - Fix cost model in development environment
   - Add shadow mode testing (see recommendations)
   - Plan next deployment with more gradual rollout

---

### 7.3 Canary Deployment Approach

**Objective**: Deploy new cost model to small percentage of traffic

**Implementation**:

```lua
-- Canary deployment logic
function get_cost_model_version(user_id)
    -- Hash user_id to determine cohort (0-99)
    local hash = tonumber(string.sub(sha1(user_id), 1, 8), 16)
    local cohort = hash % 100

    -- Get canary traffic percentage
    local canary_pct = tonumber(redis.call("GET", "cost_model:canary_percentage")) or 0

    if cohort < canary_pct then
        return "new"  -- Canary
    else
        return "old"  -- Control
    end
end

function calculate_cost(operation, size, user_id)
    local version = get_cost_model_version(user_id)

    if version == "new" then
        -- New cost model
        local C_base = NEW_COST_BASE[operation]
        local C_bw = NEW_COST_BW[operation]
        local cost = C_base + (size / UNIT_QUANTUM) * C_bw

        -- Log for comparison
        redis.call("LPUSH", "cost_model:new:observations", json.encode({
            operation = operation,
            size = size,
            cost = cost,
            user_id = user_id,
            timestamp = ngx.time()
        }))

        return cost
    else
        -- Old cost model (current)
        local C_base = OLD_COST_BASE[operation]
        local C_bw = OLD_COST_BW[operation]
        return C_base + (size / UNIT_QUANTUM) * C_bw
    end
end
```

**Canary Stages**:

| Stage | Traffic Percentage | Duration | Success Criteria | Rollback Trigger |
|-------|-------------------|----------|------------------|------------------|
| 1 | 1% | 30 min | Rejection rate < baseline + 0.5% | Rejection rate > baseline + 2% |
| 2 | 5% | 1 hour | P95 latency unchanged | Latency increase > 10% |
| 3 | 25% | 2 hours | No customer complaints | Customer complaints > 5 |
| 4 | 50% | 4 hours | All metrics stable | Any metric degrades > 5% |
| 5 | 100% | 24 hours | Continuous monitoring | Rollback to previous stage |

**Monitoring During Canary**:

```promql
# Compare new vs old cost model
# Rejection rate by version
sum by (version) (
    rate(requests_rejected_total{version=~"old|new"}[5m]) /
    rate(requests_total{version=~"old|new"}[5m])
)

# Latency by version
histogram_quantile(0.95,
    sum(rate(token_check_latency_seconds_bucket{version=~"old|new"}[5m])) by (le, version)
)

# Cost difference (how much more/less expensive is new model)
avg(cost_model_cost{version="new"}) -
avg(cost_model_cost{version="old"})
```

**Automated Rollback** (if triggers hit):
```lua
-- Check rollback conditions every 30 seconds
local function check_canary_health()
    local new_rejection = redis.call(
        "HGET", "metrics:canary:new", "rejection_rate"
    )
    local old_rejection = redis.call(
        "HGET", "metrics:canary:old", "rejection_rate"
    )

    -- Rollback if new rejection rate > 2x baseline
    if tonumber(new_rejection) > tonumber(old_rejection) * 2 then
        redis.call("SET", "cost_model:canary_percentage", "0")
        redis.call("SET", "cost_model:active_version", "old")

        send_alert("canary_failed", {
            new_rejection = new_rejection,
            old_rejection = old_rejection,
            reason = "Rejection rate doubled"
        })

        return false
    end

    return true
end
```

---

## 8. Recommendations Summary

### 8.1 High Priority (Implement Immediately)

1. **Add Shadow Mode for Cost Model Changes**
   - Reduces production risk when tuning C_base and C_bw
   - Allows validation without impacting customers
   - Implementation effort: 2-3 days

2. **Implement Comprehensive Monitoring Dashboard**
   - Required for operational maturity
   - Should include all metrics from Section 4.1
   - Implementation effort: 1 week

3. **Create Runbooks for All Scenarios**
   - Document all failure modes and responses
   - Train on-call engineers
   - Implementation effort: 1 week

4. **Add Automated Token Reconciliation**
   - Prevents token drift over time
   - Critical for accuracy
   - Implementation effort: 3-5 days

5. **Implement Canary Deployment for All Changes**
   - Reduces risk of bad deployments
   - Enables gradual rollouts
   - Implementation effort: 1 week

### 8.2 Medium Priority (Implement in 3-6 Months)

1. **Dynamic Rate Adjustment**
   - Automatically adjust token rates based on cluster utilization
   - Prevents overload before critical
   - Implementation effort: 2 weeks

2. **Token Borrowing Between Apps**
   - Better resource utilization
   - Apps can handle unexpected spikes
   - Implementation effort: 1 week

3. **Priority Queues**
   - Differentiate critical vs normal operations
   - Better handling of emergencies
   - Implementation effort: 1 week

4. **Auto-Scaling Based on Connections**
   - Scale Nginx gateways before hitting 80% limit
   - Prevents connection exhaustion
   - Implementation effort: 2-3 days

5. **Multi-Region Token Pools**
   - For global deployments
   - Reduces cross-region latency
   - Implementation effort: 2 weeks

### 8.3 Low Priority (Consider in 6-12 Months)

1. **Token Marketplace**
   - Internal capacity trading between apps
   - Innovative but complex
   - Implementation effort: 1 month

2. **Machine Learning for Cost Optimization**
   - Automated tuning of cost model
   - Reduces manual effort
   - Implementation effort: 2-3 months

3. **Predictive Auto-Scaling**
   - Forecast demand and pre-scale
   - Proactive vs reactive
   - Implementation effort: 1 month

---

## 9. Conclusion

The distributed token bucket rate-limiting system demonstrates strong architectural design with multiple layers of fault tolerance and degradation strategies. The fail-open approach prioritizes availability, which is appropriate for cloud storage platforms.

**Key Strengths**:
- Multi-dimensional cost normalization (genuinely innovative)
- Three-tier hierarchical architecture
- Comprehensive failure handling (Redis, Nginx, network)
- Edge caching for performance optimization

**Key Risks**:
- Operational complexity (high SRE burden)
- Redis dependency (despite clustering)
- Eventual consistency between L3 and L2/L1
- Cost model tuning challenges

**Overall Assessment**: 8.5/10 for architectural design, 7/10 for operational readiness

**Success Factors**:
1. Invest heavily in monitoring and observability (non-negotiable)
2. Document all failure modes and runbooks before production deployment
3. Implement canary deployments for all changes
4. Conduct regular game days to practice failure scenarios
5. Start with conservative cost estimates and tune based on production data

With proper operational maturity and the recommended improvements implemented, this system should provide reliable, scalable rate limiting for cloud storage platforms with heterogeneous resource constraints.

---

**Document End**

For questions or clarifications, contact:
- SRE Team: sre@example.com
- Platform Team: platform@example.com
- On-Call: https://oncall.example.com
