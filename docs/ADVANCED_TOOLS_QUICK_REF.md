# 高级分析工具快速参考

## 工具总览

| 工具名称 | 功能描述 | 主要用途 |
|---------|---------|---------|
| `analyze_performance` | 综合性能分析与瓶颈定位 | 识别系统性能瓶颈，评估健康度 |
| `check_network` | 网络连通性诊断 | 测试网络连通性，分析延迟和丢包 |
| `check_port` | 端口状态检查 | 检查服务端口状态，排查连接问题 |
| `check_inode` | Inode 使用情况检查 | 发现 inode 资源耗尽风险 |

## 快速使用

### 1. 性能分析

```json
{
  "host": "web-server-01",
  "duration": 10,
  "top_n": 15
}
```

**返回**:
- 性能评分 (0-100)
- 瓶颈类型 (CPU/内存/I/O)
- 风险等级 (低/中/高)
- 优化建议列表

### 2. 网络诊断

```json
{
  "host": "web-server-01",
  "target": "www.example.com",
  "count": 5
}
```

**返回**:
- Ping 结果（丢包率、延迟）
- DNS 解析结果
- 路由追踪路径
- 网络状态评估

### 3. 端口检查

```json
{
  "host": "web-server-01",
  "port": 80,
  "check_firewall": true
}
```

**返回**:
- 监听状态
- 服务进程信息
- 连接统计
- 防火墙规则状态

### 4. Inode 检查

```json
{
  "hosts": ["web-server-01", "db-server-01"],
  "threshold": 80,
  "find_large_dirs": true
}
```

**返回**:
- 各分区 inode 使用率
- 高风险分区列表
- inode 消耗大的目录
- 总体评估和建议

## AI 助手使用建议

### 性能问题排查流程

```
用户: "系统很慢，帮我分析一下"

AI:
1. 调用 analyze_performance → 识别瓶颈
2. 如果是 I/O 瓶颈 → 调用 check_disk 查看磁盘详情
3. 如果有网络问题 → 调用 check_network 测试连通性
4. 综合分析 → 提供优化建议
```

### 网络故障排查流程

```
用户: "无法访问外部服务"

AI:
1. 调用 check_network → 测试连通性
2. 调用 check_port → 检查端口监听
3. 查看防火墙规则 → 分析是否被阻止
4. 给出解决方案
```

### 服务故障排查流程

```
用户: "Web 服务启动不了"

AI:
1. 调用 check_port → 检查端口占用
2. 调用 check_process → 查看进程状态
3. 调用 query_log → 查看错误日志
4. 定位问题原因
```

## 告警阈值建议

| 指标 | 正常 | 警告 | 严重 |
|------|------|------|------|
| CPU 使用率 | < 70% | 70-90% | > 90% |
| 内存使用率 | < 80% | 80-90% | > 90% |
| IO 等待 | < 10% | 10-20% | > 20% |
| 性能评分 | > 80 | 60-80 | < 60 |
| Inode 使用率 | < 80% | 80-90% | > 90% |
| 网络丢包率 | 0% | < 5% | > 10% |
| 网络延迟 | < 50ms | 50-100ms | > 200ms |

## 常见问题解决

### Q: 性能评分低，如何优化？

**A**: 根据瓶颈类型采取不同措施：
- **CPU 密集型**: 优化代码、增加 CPU、使用负载均衡
- **内存瓶颈**: 增加内存、检查内存泄漏、优化缓存
- **I/O 密集型**: 使用 SSD、优化数据库、增加缓存

### Q: 网络丢包率高怎么办？

**A**: 检查以下方面：
1. 网络带宽是否充足
2. 网络设备是否过载
3. 网络链路质量
4. 防火墙配置
5. 路由配置

### Q: Inode 不足如何处理？

**A**: 解决方案：
1. 删除大量小文件
2. 清理临时文件和日志
3. 重新格式化磁盘，增加 inode 数量
4. 将小文件打包存储

### Q: 端口未监听如何排查？

**A**: 排查步骤：
1. 检查服务是否启动
2. 查看配置文件中的端口设置
3. 检查端口是否被其他程序占用
4. 查看防火墙规则
5. 查看服务日志

## 最佳实践

### 定期巡检

建议每天执行：
```json
[
  {"tool": "analyze_performance", "hosts": ["all"]},
  {"tool": "check_inode", "hosts": ["all"]},
  {"tool": "check_port", "hosts": ["web-servers"], "port": [80, 443]}
]
```

### 故障响应

故障发生时立即执行：
```json
[
  {"tool": "analyze_performance", "hosts": ["affected-hosts"]},
  {"tool": "check_network", "target": "dependent-services"},
  {"tool": "query_log", "keyword": "error", "time_range": "last-1h"}
]
```

## 技术架构

```
┌─────────────────────────────────────────┐
│           AI Agent Layer                │
├─────────────────────────────────────────┤
│         Tool Registry (工具注册表)        │
├─────────────────────────────────────────┤
│    Advanced Analysis Tools (高级分析)    │
│  ┌────────────┬────────────┬──────────┐ │
│  │ Performance│  Network   │   Port   │ │
│  │  Analysis  │    Check   │   Check  │ │
│  ├────────────┼────────────┼──────────┤ │
│  │   Inode    │            │          │ │
│  │   Check    │            │          │ │
│  └────────────┴────────────┴──────────┘ │
├─────────────────────────────────────────┤
│         SSH Connection Pool              │
├─────────────────────────────────────────┤
│      Target Linux Servers                │
└─────────────────────────────────────────┘
```

## 性能指标

| 工具 | 平均执行时间 | 并发支持 | 超时设置 |
|------|-------------|---------|---------|
| analyze_performance | ~8s | 是 (5协程) | 30s |
| check_network | ~6s | 否 | 30s |
| check_port | ~2s | 否 | 10s |
| check_inode | ~15s | 否 | 60s |

## 代码示例

### 在代码中使用

```go
import "ai-ops/internal/tool/builtin"

// 创建工具实例
perfTool := builtin.NewPerformanceAnalysisTool()

// 准备参数
params := map[string]interface{}{
    "host": "web-server-01",
    "duration": 10,
    "top_n": 15,
}

// 执行
ctx := &tool.Context{
    SSH: sshPool,
}
result, err := perfTool.Execute(ctx, params)
if err != nil {
    // 处理错误
}

// 使用结果
data := result.Data.(map[string]interface{})
score := data["score"].(int)
```

## 相关文档

- [详细设计文档](./ADVANCED_ANALYSIS_TOOLS.md)
- [工具实现指南](../internal/tool/README.md)
- [SSH 执行器文档](../internal/ssh/README.md)

## 版本历史

- **v1.0** (2025-12-30): 初始版本，实现 4 个高级分析工具
