#!/bin/bash
#
# Nginx 服务状态检查脚本
# 检查 Nginx 进程、配置语法、访问日志最近错误
#

# 默认参数
CONFIG_PATH="${1:-/etc/nginx/nginx.conf}"
ACCESS_LOG="${2:-/var/log/nginx/access.log}"
ERROR_LOG="${3:-/var/log/nginx/error.log}"

# 输出分隔符
SEPARATOR="----------------------------------------"

echo "=========================================="
echo "Nginx 服务状态检查"
echo "=========================================="
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "配置文件: ${CONFIG_PATH}"
echo "${SEPARATOR}"

# 检查 nginx 是否安装
if ! command -v nginx &> /dev/null; then
    echo "[ERROR] nginx 命令未找到，请先安装 Nginx"
    echo "status: error"
    echo "message: nginx not found"
    exit 1
fi

# 检查进程状态
echo ""
echo "[1] 进程状态"
echo "${SEPARATOR}"
NGINX_PID=$(pgrep -x nginx | head -1)
if [ -n "$NGINX_PID" ]; then
    echo "Nginx 状态: 运行中"
    echo "主进程 PID: ${NGINX_PID}"

    # 获取所有 nginx 进程
    NGINX_PROCESSES=$(pgrep -x nginx | wc -l)
    echo "进程数量: ${NGINX_PROCESSES}"

    # 获取 worker 进程数
    WORKER_COUNT=$((NGINX_PROCESSES - 1))
    echo "Worker 进程数: ${WORKER_COUNT}"

    # 获取进程内存使用
    TOTAL_MEM=0
    for pid in $(pgrep -x nginx); do
        MEM=$(ps -o rss= -p $pid 2>/dev/null | tr -d ' ')
        if [ -n "$MEM" ]; then
            TOTAL_MEM=$((TOTAL_MEM + MEM))
        fi
    done
    TOTAL_MEM_MB=$(echo "scale=2; ${TOTAL_MEM} / 1024" | bc 2>/dev/null || echo "N/A")
    echo "总内存使用: ${TOTAL_MEM_MB} MB"

    PROCESS_STATUS="running"
else
    echo "Nginx 状态: 未运行"
    echo "status: stopped"
    PROCESS_STATUS="stopped"
fi

# 获取 Nginx 版本
echo ""
echo "[2] 版本信息"
echo "${SEPARATOR}"
NGINX_VERSION=$(nginx -v 2>&1 | cut -d'/' -f2)
echo "Nginx 版本: ${NGINX_VERSION}"

# 检查编译参数
NGINX_MODULES=$(nginx -V 2>&1 | grep "configure arguments" | sed 's/configure arguments://')
echo "编译模块: (使用 nginx -V 查看详情)"

# 检查配置语法
echo ""
echo "[3] 配置检查"
echo "${SEPARATOR}"
if [ -f "$CONFIG_PATH" ]; then
    echo "配置文件: 存在"

    CONFIG_TEST=$(nginx -t 2>&1)
    if echo "$CONFIG_TEST" | grep -q "syntax is ok"; then
        echo "语法检查: 通过"
        CONFIG_SYNTAX="ok"
    else
        echo "语法检查: 失败"
        echo "错误信息:"
        echo "$CONFIG_TEST"
        CONFIG_SYNTAX="error"
    fi
else
    echo "配置文件: 不存在 (${CONFIG_PATH})"
    CONFIG_SYNTAX="not_found"
fi

# 检查监听端口
echo ""
echo "[4] 监听端口"
echo "${SEPARATOR}"
if [ "$PROCESS_STATUS" = "running" ]; then
    LISTEN_PORTS=$(ss -tlnp 2>/dev/null | grep nginx | awk '{print $4}' | sed 's/.*://' | sort -u | tr '\n' ' ')
    if [ -n "$LISTEN_PORTS" ]; then
        echo "监听端口: ${LISTEN_PORTS}"
    else
        # 尝试使用 netstat
        LISTEN_PORTS=$(netstat -tlnp 2>/dev/null | grep nginx | awk '{print $4}' | sed 's/.*://' | sort -u | tr '\n' ' ')
        if [ -n "$LISTEN_PORTS" ]; then
            echo "监听端口: ${LISTEN_PORTS}"
        else
            echo "监听端口: 无法获取"
        fi
    fi
else
    echo "监听端口: Nginx 未运行"
fi

# 检查访问日志
echo ""
echo "[5] 访问日志分析"
echo "${SEPARATOR}"
if [ -f "$ACCESS_LOG" ]; then
    echo "访问日志: 存在"

    # 日志文件大小
    LOG_SIZE=$(ls -lh "$ACCESS_LOG" 2>/dev/null | awk '{print $5}')
    echo "日志大小: ${LOG_SIZE}"

    # 最近请求统计（最近100行）
    if [ -r "$ACCESS_LOG" ]; then
        RECENT_REQUESTS=$(tail -100 "$ACCESS_LOG" 2>/dev/null | wc -l)
        echo "最近请求数(100行): ${RECENT_REQUESTS}"

        # 统计 HTTP 状态码
        echo ""
        echo "最近请求状态码分布:"
        tail -1000 "$ACCESS_LOG" 2>/dev/null | awk '{print $9}' | sort | uniq -c | sort -rn | head -10 | while read count code; do
            echo "  ${code}: ${count} 次"
        done
    else
        echo "访问日志: 无读取权限"
    fi
else
    echo "访问日志: 不存在 (${ACCESS_LOG})"
fi

# 检查错误日志
echo ""
echo "[6] 错误日志分析"
echo "${SEPARATOR}"
if [ -f "$ERROR_LOG" ]; then
    echo "错误日志: 存在"

    # 日志文件大小
    ERROR_LOG_SIZE=$(ls -lh "$ERROR_LOG" 2>/dev/null | awk '{print $5}')
    echo "日志大小: ${ERROR_LOG_SIZE}"

    if [ -r "$ERROR_LOG" ]; then
        # 最近错误数量
        RECENT_ERRORS=$(tail -100 "$ERROR_LOG" 2>/dev/null | grep -c "error")
        RECENT_WARNINGS=$(tail -100 "$ERROR_LOG" 2>/dev/null | grep -c "warn")
        echo "最近错误数(100行): ${RECENT_ERRORS}"
        echo "最近警告数(100行): ${RECENT_WARNINGS}"

        # 显示最近的错误
        echo ""
        echo "最近 5 条错误日志:"
        tail -50 "$ERROR_LOG" 2>/dev/null | grep "error" | tail -5 | while read line; do
            echo "  ${line:0:100}..."
        done
    else
        echo "错误日志: 无读取权限"
    fi
else
    echo "错误日志: 不存在 (${ERROR_LOG})"
fi

# 检查上游服务器状态（如果有）
echo ""
echo "[7] 连接统计"
echo "${SEPARATOR}"
if [ "$PROCESS_STATUS" = "running" ]; then
    # 获取当前连接数
    ACTIVE_CONNECTIONS=$(ss -tn 2>/dev/null | grep -E ":80|:443" | wc -l)
    echo "活跃连接数(80/443端口): ${ACTIVE_CONNECTIONS}"

    # 尝试获取 stub_status 信息（如果启用）
    STUB_STATUS=$(curl -s http://127.0.0.1/nginx_status 2>/dev/null)
    if [ -n "$STUB_STATUS" ] && echo "$STUB_STATUS" | grep -q "Active connections"; then
        echo ""
        echo "Nginx Status 模块信息:"
        echo "$STUB_STATUS" | while read line; do
            echo "  ${line}"
        done
    fi
else
    echo "连接统计: Nginx 未运行"
fi

# 汇总状态
echo ""
echo "=========================================="
echo "检查结果汇总"
echo "=========================================="
echo "status: ${PROCESS_STATUS}"
echo "nginx_version: ${NGINX_VERSION}"
echo "config_syntax: ${CONFIG_SYNTAX}"
echo "worker_processes: ${WORKER_COUNT:-0}"
echo "memory_usage_mb: ${TOTAL_MEM_MB:-0}"
echo "check_time: $(date '+%Y-%m-%d %H:%M:%S')"

# 返回状态码
if [ "$PROCESS_STATUS" = "running" ] && [ "$CONFIG_SYNTAX" = "ok" ]; then
    exit 0
else
    exit 1
fi
