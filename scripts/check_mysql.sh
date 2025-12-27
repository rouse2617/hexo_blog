#!/bin/bash
#
# MySQL 服务状态检查脚本
# 检查 MySQL 连接、运行状态、连接数等信息
#

# 默认参数
HOST="${1:-localhost}"
PORT="${2:-3306}"
USER="${3:-root}"
PASSWORD="${4:-}"

# 输出分隔符
SEPARATOR="----------------------------------------"

echo "=========================================="
echo "MySQL 服务状态检查"
echo "=========================================="
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "目标主机: ${HOST}:${PORT}"
echo "连接用户: ${USER}"
echo "${SEPARATOR}"

# 检查 mysql 客户端是否可用
if ! command -v mysql &> /dev/null; then
    echo "[ERROR] mysql 命令未找到，请先安装 MySQL 客户端"
    echo "status: error"
    echo "message: mysql client not found"
    exit 1
fi

# 构建 MySQL 连接命令
MYSQL_CMD="mysql -h ${HOST} -P ${PORT} -u ${USER}"
if [ -n "$PASSWORD" ]; then
    MYSQL_CMD="${MYSQL_CMD} -p${PASSWORD}"
fi
MYSQL_CMD="${MYSQL_CMD} --connect-timeout=5"

# 测试连接
echo ""
echo "[1] 连接测试"
echo "${SEPARATOR}"
CONNECT_TEST=$(${MYSQL_CMD} -e "SELECT 1" 2>&1)
if [ $? -eq 0 ]; then
    echo "连接状态: 成功"
    echo "status: connected"
else
    echo "连接状态: 失败"
    echo "错误信息: ${CONNECT_TEST}"
    echo "status: disconnected"
    echo "error: ${CONNECT_TEST}"
    exit 1
fi

# 获取 MySQL 版本信息
echo ""
echo "[2] 服务器信息"
echo "${SEPARATOR}"
VERSION=$(${MYSQL_CMD} -N -e "SELECT VERSION()" 2>/dev/null)
echo "MySQL 版本: ${VERSION}"

# 获取运行时间
UPTIME=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Uptime'" 2>/dev/null | awk '{print $2}')
if [ -n "$UPTIME" ]; then
    UPTIME_DAYS=$((UPTIME / 86400))
    UPTIME_HOURS=$(((UPTIME % 86400) / 3600))
    UPTIME_MINUTES=$(((UPTIME % 3600) / 60))
    echo "运行时间: ${UPTIME_DAYS} 天 ${UPTIME_HOURS} 小时 ${UPTIME_MINUTES} 分钟"
fi

# 获取服务器 ID
SERVER_ID=$(${MYSQL_CMD} -N -e "SELECT @@server_id" 2>/dev/null)
echo "服务器 ID: ${SERVER_ID}"

# 连接数信息
echo ""
echo "[3] 连接信息"
echo "${SEPARATOR}"
MAX_CONNECTIONS=$(${MYSQL_CMD} -N -e "SHOW VARIABLES LIKE 'max_connections'" 2>/dev/null | awk '{print $2}')
CURRENT_CONNECTIONS=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Threads_connected'" 2>/dev/null | awk '{print $2}')
THREADS_RUNNING=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Threads_running'" 2>/dev/null | awk '{print $2}')
MAX_USED_CONNECTIONS=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Max_used_connections'" 2>/dev/null | awk '{print $2}')

echo "最大连接数: ${MAX_CONNECTIONS}"
echo "当前连接数: ${CURRENT_CONNECTIONS}"
echo "活跃线程数: ${THREADS_RUNNING}"
echo "历史最大连接数: ${MAX_USED_CONNECTIONS}"

# 计算连接使用率
if [ -n "$MAX_CONNECTIONS" ] && [ "$MAX_CONNECTIONS" != "0" ]; then
    CONNECTION_USAGE=$(echo "scale=2; ${CURRENT_CONNECTIONS} * 100 / ${MAX_CONNECTIONS}" | bc 2>/dev/null || echo "N/A")
    echo "连接使用率: ${CONNECTION_USAGE}%"
fi

# 数据库列表
echo ""
echo "[4] 数据库列表"
echo "${SEPARATOR}"
DATABASES=$(${MYSQL_CMD} -N -e "SHOW DATABASES" 2>/dev/null)
DB_COUNT=$(echo "$DATABASES" | wc -l)
echo "数据库数量: ${DB_COUNT}"
echo "数据库列表:"
echo "$DATABASES" | while read db; do
    echo "  - ${db}"
done

# 查询统计
echo ""
echo "[5] 查询统计"
echo "${SEPARATOR}"
QUESTIONS=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Questions'" 2>/dev/null | awk '{print $2}')
SELECT_COUNT=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Com_select'" 2>/dev/null | awk '{print $2}')
INSERT_COUNT=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Com_insert'" 2>/dev/null | awk '{print $2}')
UPDATE_COUNT=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Com_update'" 2>/dev/null | awk '{print $2}')
DELETE_COUNT=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Com_delete'" 2>/dev/null | awk '{print $2}')

echo "总查询数: ${QUESTIONS}"
echo "SELECT 查询: ${SELECT_COUNT}"
echo "INSERT 查询: ${INSERT_COUNT}"
echo "UPDATE 查询: ${UPDATE_COUNT}"
echo "DELETE 查询: ${DELETE_COUNT}"

# 计算 QPS
if [ -n "$QUESTIONS" ] && [ -n "$UPTIME" ] && [ "$UPTIME" != "0" ]; then
    QPS=$(echo "scale=2; ${QUESTIONS} / ${UPTIME}" | bc 2>/dev/null || echo "N/A")
    echo "平均 QPS: ${QPS}"
fi

# InnoDB 状态
echo ""
echo "[6] InnoDB 状态"
echo "${SEPARATOR}"
INNODB_BUFFER_POOL_SIZE=$(${MYSQL_CMD} -N -e "SHOW VARIABLES LIKE 'innodb_buffer_pool_size'" 2>/dev/null | awk '{print $2}')
INNODB_BUFFER_POOL_SIZE_MB=$((INNODB_BUFFER_POOL_SIZE / 1024 / 1024))
INNODB_BUFFER_POOL_READ_REQUESTS=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool_read_requests'" 2>/dev/null | awk '{print $2}')
INNODB_BUFFER_POOL_READS=$(${MYSQL_CMD} -N -e "SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool_reads'" 2>/dev/null | awk '{print $2}')

echo "InnoDB 缓冲池大小: ${INNODB_BUFFER_POOL_SIZE_MB} MB"
echo "缓冲池读请求: ${INNODB_BUFFER_POOL_READ_REQUESTS}"
echo "磁盘读取次数: ${INNODB_BUFFER_POOL_READS}"

# 计算缓冲池命中率
if [ -n "$INNODB_BUFFER_POOL_READ_REQUESTS" ] && [ "$INNODB_BUFFER_POOL_READ_REQUESTS" != "0" ]; then
    HIT_RATE=$(echo "scale=4; (${INNODB_BUFFER_POOL_READ_REQUESTS} - ${INNODB_BUFFER_POOL_READS}) * 100 / ${INNODB_BUFFER_POOL_READ_REQUESTS}" | bc 2>/dev/null || echo "N/A")
    echo "缓冲池命中率: ${HIT_RATE}%"
fi

# 复制状态（如果是从库）
echo ""
echo "[7] 复制状态"
echo "${SEPARATOR}"
SLAVE_STATUS=$(${MYSQL_CMD} -e "SHOW SLAVE STATUS\G" 2>/dev/null)
if [ -n "$SLAVE_STATUS" ] && echo "$SLAVE_STATUS" | grep -q "Slave_IO_Running"; then
    SLAVE_IO=$(echo "$SLAVE_STATUS" | grep "Slave_IO_Running:" | awk '{print $2}')
    SLAVE_SQL=$(echo "$SLAVE_STATUS" | grep "Slave_SQL_Running:" | awk '{print $2}')
    SECONDS_BEHIND=$(echo "$SLAVE_STATUS" | grep "Seconds_Behind_Master:" | awk '{print $2}')
    echo "从库 IO 线程: ${SLAVE_IO}"
    echo "从库 SQL 线程: ${SLAVE_SQL}"
    echo "复制延迟(秒): ${SECONDS_BEHIND}"
else
    echo "当前服务器不是从库或未配置复制"
fi

# 汇总状态
echo ""
echo "=========================================="
echo "检查结果汇总"
echo "=========================================="
echo "status: ok"
echo "mysql_version: ${VERSION}"
echo "uptime_days: ${UPTIME_DAYS}"
echo "current_connections: ${CURRENT_CONNECTIONS}"
echo "max_connections: ${MAX_CONNECTIONS}"
echo "threads_running: ${THREADS_RUNNING}"
echo "check_time: $(date '+%Y-%m-%d %H:%M:%S')"

exit 0
