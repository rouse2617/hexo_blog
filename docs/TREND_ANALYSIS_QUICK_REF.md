# 趋势分析工具 - 快速参考

## 工具信息
- **名称**: `analyze_trend`
- **类型**: 预测性维护工具
- **文件**: `internal/tool/builtin/trend_analysis.go`
- **代码量**: 726 行

## 快速使用

### 单主机分析
```json
{
  "host": "server1",
  "metric": "disk",
  "days": 7
}
```

### 批量分析
```json
{
  "hosts": ["server1", "server2", "server3"],
  "metric": "memory"
}
```

## 指标类型

| 指标 | 说明 | 主要命令 |
|------|------|----------|
| disk | 磁盘趋势分析 | df, du, find |
| memory | 内存趋势分析 | free, ps, swapon |
| log | 日志增长分析 | find, ls, journalctl |
| network | 网络流量分析 | /proc/net/dev, ss, ip |

## 风险等级

### 磁盘风险
- 🟢 低: < 50%
- 🟡 中: 50-70%
- 🟠 高: 70-85%
- 🔴 严重: 85-95%
- 🚨 紧急: > 95%

### 内存风险
- 🟢 低风险: Mem < 70%, Swap < 10%
- 🟡 中风险: Mem 70-85% 或 Swap 10-30%
- 🟠 高风险: Mem > 85% 或 Swap > 30%
- 🔴 极高: Mem > 90%, Swap > 50%

## 输出结构

### 磁盘分析
```json
{
  "current": "当前磁盘使用情况",
  "analysis": "大目录分析",
  "inode": "inode 使用情况",
  "predictions": {
    "predictions": [...],
    "total": 分区数量
  },
  "suggestions": "优化建议"
}
```

### 内存分析
```json
{
  "current": "当前内存使用",
  "processes": "大内存进程",
  "swap": "Swap 使用情况",
  "meminfo": "详细内存信息",
  "trend": {
    "memory_percent": 使用率,
    "swap_percent": Swap使用率,
    "trend": "趋势描述",
    "oom_risk": "OOM风险评估"
  },
  "suggestions": "优化建议"
}
```

### 日志分析
```json
{
  "log_files": "日志文件列表",
  "rotation": "轮转配置",
  "journal": "journalctl 使用情况",
  "growth": {
    "total_size_mb": 总大小MB,
    "file_count": 文件数量,
    "risk_level": "风险等级"
  },
  "suggestions": "管理建议"
}
```

### 网络分析
```json
{
  "network": "网卡流量统计",
  "connections": "连接统计",
  "interfaces": "网络接口信息",
  "trend": {
    "interfaces": [接口列表],
    "total_rx": "总接收",
    "total_tx": "总发送"
  },
  "suggestions": "优化建议"
}
```

## 常见清理命令

### 磁盘清理
```bash
# 日志清理
journalctl --vacuum-time=30d
journalctl --vacuum-size=500M

# 包管理器缓存
yum clean all          # CentOS/RHEL
apt-get clean          # Debian/Ubuntu

# Docker 清理
docker system prune -a

# 查找大文件
find / -size +100M -type f
```

### 内存优化
```bash
# 查看大内存进程
ps aux --sort=-%mem | head -20

# 清理 page cache
sync; echo 3 > /proc/sys/vm/drop_caches

# 查看 Swap 使用
swapon --show
free -h
```

### 日志管理
```bash
# 配置 logrotate
vim /etc/logrotate.d/<app>

# 手动轮转
logrotate -f /etc/logrotate.conf

# journalctl 管理
journalctl --disk-usage
journalctl --vacuum-time=7d
journalctl --vacuum-size=100M
```

## AI 使用场景

### 容量规划
> "帮我分析所有服务器的磁盘使用趋势，预测哪些需要扩容"

### 故障预防
> "检查生产环境内存趋势，评估 OOM 风险"

### 日志审计
> "分析日志增长情况，给出日志轮转建议"

### 性能优化
> "分析网络流量趋势，识别带宽瓶颈"

## 技术要点

### Go 语言特性
- ✅ 接口实现: tool.Tool
- ✅ 错误处理: 完整的错误处理
- ✅ 类型安全: 强类型参数解析
- ✅ 并发支持: 支持 BatchExec 批量操作
- ✅ 代码规范: go fmt 通过
- ✅ 静态检查: go vet 通过

### 设计模式
- **策略模式**: 不同指标使用不同分析策略
- **工厂模式**: NewTrendAnalysisTool() 构造函数
- **组合模式**: 支持 host/hosts 两种调用方式

### 代码质量
- ✅ 单一职责: 每个分析方法专注于一个指标
- ✅ 开闭原则: 易于扩展新的指标类型
- ✅ 依赖注入: 通过 Context 注入 SSH 连接池
- ✅ 测试覆盖: 包含完整的单元测试

## 扩展建议

### 可添加的功能
1. **历史数据存储**: 将分析结果存入数据库，生成历史趋势图
2. **告警规则**: 配置告警阈值，自动触发通知
3. **预测算法**: 使用机器学习算法提高预测准确性
4. **可视化**: 生成趋势图表和仪表板
5. **定期巡检**: 结合 cron 定期自动执行分析

### 可优化的点
1. **缓存机制**: 缓存 SSH 执行结果，减少重复查询
2. **并发控制**: 限制并发 SSH 连接数
3. **超时管理**: 为每个命令设置合理的超时时间
4. **重试机制**: 失败命令自动重试
5. **结果压缩**: 大量结果压缩后返回

## 相关文档
- 完整文档: `docs/TREND_ANALYSIS_TOOL.md`
- 测试文件: `internal/tool/builtin/trend_analysis_test.go`
- 工具注册: `internal/tool/builtin/register.go`
