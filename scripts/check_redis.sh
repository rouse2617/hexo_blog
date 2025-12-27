#!/bin/bash
#
# Redis 服务状态检查脚本
# 检查 Redis 连接、内存使用、连接数等信息
#

# 默认参数
HOST="${1:-localhost}"
PORT="${2:-6379}"

# 输出分隔符
SEPARATOR="----------------------------------------"

echo "=========================================="
echo "Redis 服务状态检查"
echo "=========================================="
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "目标主机: ${HOST}:${PORT}"
echo "${SEPARATOR}"

# 检查 redis-cli 是否可用
if ! command -v redis-cli &> /dev/null; then
    echo "[ERROR] redis-cli 命令未找到，请先安装 Redis 客户端"
    echo "status: error"
    echo "message: redis-cli not found"
    exit 1
fi

# 测试连接
echo ""
echo "[1] 连接测试"
echo "${SEPARATOR}"
PING_RESULT=$(redis-cli -h "${HOST}" -p "${PORT}" ping 2>&1)
if [ "$PING_RESULT" = "PONG" ]; then
    echo "连接状态: 成功"
    echo "status: connected"
else
    echo "连接状态: 失败"
    echo "错误信息: ${PING_RESULT}"
    echo "status: disconnected"
    echo "error: ${PING_RESULT}"
    exit 1
fi

# 获取 Redis 信息
echo ""
echo "[2] 服务器信息"
echo "${SEPARATOR}"
INFO=$(redis-cli -h "${HOST}" -p "${PORT}" INFO 2>&1)

# 提取版本信息
REDIS_VERSION=$(echo "$INFO" | grep "redis_version:" | cut -d: -f2 | tr -d '\r')
echo "Redis 版本: ${REDIS_VERSION}"

# 提取运行时间
UPTIME_SECONDS=$(echo "$INFO" | grep "uptime_in_seconds:" | cut -d: -f2 | tr -d '\r')
UPTIME_DAYS=$(echo "$INFO" | grep "uptime_in_days:" | cut -d: -f2 | tr -d '\r')
echo "运行时间: ${UPTIME_DAYS} 天 (${UPTIME_SECONDS} 秒)"

# 内存使用情况
echo ""
echo "[3] 内存使用"
echo "${SEPARATOR}"
USED_MEMORY=$(echo "$INFO" | grep "used_memory:" | head -1 | cut -d: -f2 | tr -d '\r')
USED_MEMORY_HUMAN=$(echo "$INFO" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r')
USED_MEMORY_PEAK_HUMAN=$(echo "$INFO" | grep "used_memory_peak_human:" | cut -d: -f2 | tr -d '\r')
MAXMEMORY=$(echo "$INFO" | grep "maxmemory:" | cut -d: -f2 | tr -d '\r')
MAXMEMORY_HUMAN=$(echo "$INFO" | grep "maxmemory_human:" | cut -d: -f2 | tr -d '\r')

echo "当前内存使用: ${USED_MEMORY_HUMAN}"
echo "内存使用峰值: ${USED_MEMORY_PEAK_HUMAN}"
echo "最大内存限制: ${MAXMEMORY_HUMAN:-未设置}"

# 计算内存使用率
if [ -n "$MAXMEMORY" ] && [ "$MAXMEMORY" != "0" ]; then
    MEMORY_USAGE_PERCENT=$(echo "scale=2; ${USED_MEMORY} * 100 / ${MAXMEMORY}" | bc 2>/dev/null || echo "N/A")
    echo "内存使用率: ${MEMORY_USAGE_PERCENT}%"
fi

# 连接数信息
echo ""
echo "[4] 连接信息"
echo "${SEPARATOR}"
CONNECTED_CLIENTS=$(echo "$INFO" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r')
BLOCKED_CLIENTS=$(echo "$INFO" | grep "blocked_clients:" | cut -d: -f2 | tr -d '\r')
MAXCLIENTS=$(redis-cli -h "${HOST}" -p "${PORT}" CONFIG GET maxclients 2>/dev/null | tail -1)

echo "当前连接数: ${CONNECTED_CLIENTS}"
echo "阻塞客户端数: ${BLOCKED_CLIENTS}"
echo "最大连接数限制: ${MAXCLIENTS:-未知}"

# 键空间信息
echo ""
echo "[5] 键空间统计"
echo "${SEPARATOR}"
KEYSPACE=$(echo "$INFO" | grep "^db" | tr -d '\r')
if [ -n "$KEYSPACE" ]; then
    echo "$KEYSPACE" | while read line; do
        DB_NAME=$(echo "$line" | cut -d: -f1)
        DB_INFO=$(echo "$line" | cut -d: -f2)
        KEYS=$(echo "$DB_INFO" | grep -o "keys=[0-9]*" | cut -d= -f2)
        EXPIRES=$(echo "$DB_INFO" | grep -o "expires=[0-9]*" | cut -d= -f2)
        echo "${DB_NAME}: ${KEYS} 个键, ${EXPIRES} 个过期键"
    done
else
    echo "无数据库键"
fi

# 持久化状态
echo ""
echo "[6] 持久化状态"
echo "${SEPARATOR}"
RDB_LAST_SAVE=$(echo "$INFO" | grep "rdb_last_save_time:" | cut -d: -f2 | tr -d '\r')
RDB_LAST_STATUS=$(echo "$INFO" | grep "rdb_last_bgsave_status:" | cut -d: -f2 | tr -d '\r')
AOF_ENABLED=$(echo "$INFO" | grep "aof_enabled:" | cut -d: -f2 | tr -d '\r')

if [ -n "$RDB_LAST_SAVE" ]; then
    LAST_SAVE_TIME=$(date -d "@${RDB_LAST_SAVE}" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "时间戳: ${RDB_LAST_SAVE}")
    echo "RDB 最后保存时间: ${LAST_SAVE_TIME}"
fi
echo "RDB 最后保存状态: ${RDB_LAST_STATUS:-未知}"
echo "AOF 是否启用: $([ "$AOF_ENABLED" = "1" ] && echo "是" || echo "否")"

# 汇总状态
echo ""
echo "=========================================="
echo "检查结果汇总"
echo "=========================================="
echo "status: ok"
echo "redis_version: ${REDIS_VERSION}"
echo "uptime_days: ${UPTIME_DAYS}"
echo "used_memory: ${USED_MEMORY_HUMAN}"
echo "connected_clients: ${CONNECTED_CLIENTS}"
echo "check_time: $(date '+%Y-%m-%d %H:%M:%S')"

exit 0
