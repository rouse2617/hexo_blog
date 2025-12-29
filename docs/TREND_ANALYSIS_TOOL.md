# 趋势分析工具使用文档

## 概述

`analyze_trend` 是一个预测性维护工具，基于当前数据分析资源使用趋势，预测未来容量需求。

## 工具名称
```
analyze_trend
```

## 功能特性

### 1. 磁盘增长趋势分析
- 分析各分区当前使用率
- 预测磁盘满的时间
- 评估风险等级（低/中/高/严重/紧急）
- 提供清理建议

### 2. 内存使用趋势分析
- 分析内存使用率和 Swap 使用
- 评估 OOM（内存溢出）风险
- 识别大内存进程
- 提供优化建议

### 3. 日志增长趋势分析
- 统计日志文件大小
- 检查日志轮转配置
- 分析 journalctl 使用情况
- 提供日志管理建议

### 4. 网络流量趋势分析
- 统计网卡流量（RX/TX）
- 显示网络连接统计
- 提供带宽优化建议

## 参数说明

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| host | string | 否 | - | 目标主机名称（单主机分析） |
| hosts | []string | 否 | - | 多个目标主机（批量分析） |
| metric | string | 否 | disk | 指标类型：disk/memory/log/network |
| days | integer | 否 | 7 | 分析天数（用于查找历史数据） |

## 使用示例

### 1. 磁盘趋势分析（单主机）
```json
{
  "host": "server1",
  "metric": "disk",
  "days": 7
}
```

### 2. 内存趋势分析（多主机）
```json
{
  "hosts": ["server1", "server2", "server3"],
  "metric": "memory",
  "days": 7
}
```

### 3. 日志增长分析
```json
{
  "host": "server1",
  "metric": "log",
  "days": 30
}
```

### 4. 网络流量趋势
```json
{
  "host": "server1",
  "metric": "network",
  "days": 7
}
```

## 返回结果示例

### 磁盘分析结果
```json
{
  "host": "server1",
  "current": "Filesystem      Size  Used Avail Use% Mounted on\n/dev/sda1        50G   25G   25G  50% /",
  "analysis": "1.2G\t/var/log\n850M\t/var/lib\n...",
  "inode": "Filesystem      Inodes IUsed IFree IUse% Mounted on\n/dev/sda1      3276800 15234 3261566    1% /",
  "predictions": {
    "predictions": [
      {
        "mount_point": "/",
        "current_usage": 50,
        "risk_level": "中",
        "days_until_full": 60,
        "action_required": "定期监控"
      }
    ],
    "total": 1
  },
  "suggestions": "⚠️  警告: / 使用率 50% (中)\n   定期监控\n...",
  "timestamp": "2025-12-30T10:30:00Z"
}
```

### 内存分析结果
```json
{
  "host": "server1",
  "current": "              total        used        free      shared  buff/cache   available\nMem:           7.6G        5.4G        1.2G        234M        1.0G        1.8G\nSwap:          2.0G        512M        1.5G",
  "processes": "USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND\nuser       1234  2.3  15.6 1234567 123456 ?      Sl   10:00   0:05 java",
  "swap": "NAME      TYPE SIZE USED PRIO\n/dev/sda2 partition 2G 512M -2",
  "meminfo": "MemTotal:        8000000 kB\nMemFree:         1200000 kB\nMemAvailable:    1800000 kB\n...",
  "trend": {
    "memory_total": "7.6G",
    "memory_used": "5.4G",
    "memory_available": "1.8G",
    "memory_percent": 71,
    "swap_total": "2.0G",
    "swap_used": "512M",
    "swap_percent": 25,
    "trend": "上升",
    "oom_risk": "中风险 - 需要关注"
  },
  "suggestions": "⚠️  警告: 中风险 - 需要关注\n...",
  "timestamp": "2025-12-30T10:30:00Z"
}
```

### 日志分析结果
```json
{
  "host": "server1",
  "log_files": "1.2G /var/log/syslog\n850M /var/log/auth.log\n...",
  "rotation": "total 24\ndrwxr-xr-x 2 root root 4096 Dec 30 10:00 .\n...",
  "journal": "Journals take up 850.0M on disk.\nSystem journal is rotating.",
  "growth": {
    "total_size_mb": 2050,
    "avg_size_mb": 102,
    "file_count": 20,
    "risk_level": "严重",
    "largest_files": [
      {"file": "/var/log/syslog", "size": "1.2G"},
      {"file": "/var/log/auth.log", "size": "850M"}
    ]
  },
  "suggestions": "⚠️  警告: 日志风险等级 严重, 总量 2050 MB\n...",
  "timestamp": "2025-12-30T10:30:00Z"
}
```

### 网络分析结果
```json
{
  "host": "server1",
  "network": "eth0 1234567890 987654321\nlo 0 0",
  "connections": "TCP:   1000 (estab 123, closed 877, orphaned 0, synrecv 0)\n...",
  "interfaces": "1: lo: <LOOPBACK,UP,LOWER_UP>\n    inet 127.0.0.1/8 scope host lo\n...",
  "trend": {
    "interfaces": [
      {
        "name": "eth0",
        "rx": "1.1 GB",
        "tx": "941.9 MB",
        "rx_gb": 1.15,
        "tx_gb": 0.92
      }
    ],
    "total_rx": "1.1 GB",
    "total_tx": "941.9 MB",
    "interface_count": 2
  },
  "suggestions": "1. 监控网络流量趋势\n...",
  "timestamp": "2025-12-30T10:30:00Z"
}
```

## 风险等级说明

### 磁盘风险等级
- **低**: 使用率 < 50%，预计 60+ 天满
- **中**: 使用率 50-70%，预计 30-60 天满
- **高**: 使用率 70-85%，预计 7-30 天满
- **严重**: 使用率 85-95%，预计 < 7 天满
- **紧急**: 使用率 > 95%，预计 < 1 天满

### 内存风险等级
- **低风险**: 内存 < 70%，Swap < 10%
- **中风险**: 内存 70-85%，或 Swap 使用 10-30%
- **高风险**: 内存 > 85%，或 Swap 使用 > 30%，OOM 可能性大
- **极高风险**: 内存 > 90%，Swap 使用 > 50%，即将发生 OOM

### 日志风险等级
- **低**: 总量 < 50MB
- **中**: 总量 50-100MB
- **高**: 总量 100-500MB
- **严重**: 总量 500MB-1GB
- **紧急**: 总量 > 1GB

## 常见建议

### 磁盘清理建议
1. 设置日志轮转: 配置 logrotate
2. 定期清理临时文件: 使用 tmpwatch 或 tmpreaper
3. 清理包管理器缓存: yum clean all / apt-get clean
4. 清理旧日志: journalctl --vacuum-time=30d
5. 监控大文件: find / -size +100M -type f
6. 清理 Docker 镜像和容器: docker system prune -a

### 内存优化建议
1. 定期监控内存使用情况
2. 检查是否有内存泄漏: 使用 top/ps/valgrind
3. 分析大内存进程: ps aux --sort=-%mem | head -20
4. 考虑增加物理内存
5. 优化应用内存使用
6. 检查 Swap 使用是否正常

### 日志管理建议
1. 配置 logrotate 自动轮转日志
2. 设置合适的日志保留策略
3. 日志总量较大时，建议立即清理
4. 考虑减少日志级别
5. 使用日志中心集中管理 (ELK/Loki)
6. 清理 journalctl: journalctl --vacuum-size=500M

### 网络优化建议
1. 监控网络流量趋势
2. 设置流量告警阈值
3. 优化网络带宽使用
4. 定期检查网络连接数: ss -s
5. 分析异常连接: ss -tunap

## 注意事项

1. **批量分析**: 使用 `hosts` 参数可以同时分析多个主机的趋势
2. **历史数据**: `days` 参数用于指定查找多少天内的历史数据
3. **SSH 连接**: 工具需要通过 SSH 连接到目标主机执行命令
4. **权限要求**: 某些命令可能需要 root 权限才能获取完整信息
5. **时间戳**: 所有结果都包含 ISO 8601 格式的时间戳

## 适用场景

### 日常巡检
```json
{
  "hosts": ["server1", "server2", "server3"],
  "metric": "disk",
  "days": 7
}
```

### 容量规划
```json
{
  "host": "server1",
  "metric": "memory",
  "days": 30
}
```

### 故障排查
```json
{
  "host": "server1",
  "metric": "log",
  "days": 1
}
```

### 性能优化
```json
{
  "host": "server1",
  "metric": "network",
  "days": 7
}
```

## 技术实现

### 使用的 Linux 命令

#### 磁盘分析
- `df -h`: 查看磁盘使用情况
- `du -sh`: 查看目录大小
- `df -i`: 查看 inode 使用情况
- `find /var/log`: 查找日志文件

#### 内存分析
- `free -h`: 查看内存使用情况
- `ps aux --sort=-%mem`: 按内存使用排序进程
- `swapon --show`: 查看 Swap 使用情况
- `cat /proc/meminfo`: 查看详细内存信息

#### 日志分析
- `find /var/log`: 查找日志文件
- `ls -la /etc/logrotate.d/`: 查看日志轮转配置
- `journalctl --disk-usage`: 查看 journalctl 使用情况

#### 网络分析
- `cat /proc/net/dev`: 查看网卡统计
- `ss -s`: 查看连接统计
- `ip addr show`: 查看网络接口信息

## 开发信息

- **文件路径**: `internal/tool/builtin/trend_analysis.go`
- **包名**: `builtin`
- **结构体**: `TrendAnalysisTool`
- **接口实现**: `tool.Tool`
- **注册函数**: `NewTrendAnalysisTool()`

## 版本历史

- **v1.0** (2025-12-30): 初始版本
  - 实现磁盘、内存、日志、网络趋势分析
  - 支持单主机和批量分析
  - 提供风险等级评估和优化建议
