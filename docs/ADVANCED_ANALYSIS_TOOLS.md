# 高级分析工具实现总结

## 概述

已成功实现 4 个高级分析工具，这些工具将显著增强 AI-Ops 系统的诊断和分析能力。

## 新增工具

### 1. 性能分析工具 (`analyze_performance`)

**文件位置**: `internal/tool/builtin/performance_analysis.go`

**功能特性**:
- 综合性能分析，覆盖 CPU、内存、I/O、网络四大维度
- 自动识别性能瓶颈类型（CPU密集型、内存瓶颈、I/O密集型）
- 智能性能评分系统（0-100 分）
- 风险等级评估（低/中/高）
- 针对瓶颈类型的优化建议生成
- 并发采集指标数据，提高效率
- Top N 进程分析（可配置）

**参数**:
- `host` (必填): 目标主机名称
- `duration` (可选): 采集时长（秒），默认 5 秒
- `top_n` (可选): 显示 Top N 进程，默认 10

**输出内容**:
```json
{
  "host": "服务器名称",
  "score": 75,
  "bottleneck": "内存瓶颈",
  "risk_level": "中风险",
  "suggestions": [
    "增加物理内存",
    "优化应用内存使用",
    "检查是否有内存泄漏"
  ],
  "cpu": "...",
  "memory": "...",
  "io": "...",
  "network": "...",
  "processes": "..."
}
```

### 2. 网络诊断工具 (`check_network`)

**文件位置**: `internal/tool/builtin/network_check.go`

**功能特性**:
- Ping 测试检测网络连通性
- 丢包率和延迟分析（最小/平均/最大）
- DNS 解析测试
- 路由追踪（支持 traceroute 和 tracepath）
- 网络接口状态检查
- 智能网络状态评估（优秀/正常/一般/不通/严重丢包）

**参数**:
- `host` (必填): 源主机名称
- `target` (必填): 目标 IP 或域名
- `count` (可选): ping 次数，默认 4
- `timeout` (可选): 超时时间（秒），默认 30

**输出内容**:
```json
{
  "host": "源主机",
  "target": "目标地址",
  "ping": "ping 输出",
  "dns": "DNS 解析结果",
  "traceroute": "路由追踪结果",
  "interfaces": "网络接口信息",
  "packet_loss": "0%",
  "avg_time": "12.4 ms",
  "min_time": "12.3 ms",
  "max_time": "12.5 ms",
  "status": "优秀"
}
```

### 3. 端口检查工具 (`check_port`)

**文件位置**: `internal/tool/builtin/port_check.go`

**功能特性**:
- 端口监听状态检测
- 服务和进程信息解析（PID、进程名、协议类型）
- 连接统计（ESTABLISHED、TIME-WAIT、LISTEN 等）
- 防火墙规则检查（iptables 和 firewalld）
- 连接状态评估（正常/活跃/连接数过多/未监听）

**参数**:
- `host` (必填): 目标主机
- `port` (必填): 端口号（1-65535）
- `check_firewall` (可选): 是否检查防火墙规则，默认 true

**输出内容**:
```json
{
  "host": "主机名",
  "port": 80,
  "is_listening": true,
  "service_info": {
    "pid": "1234",
    "process": "nginx",
    "protocol": "tcp"
  },
  "conn_count": 150,
  "conn_details": {
    "established": 100,
    "time_wait": 30,
    "listen": 1,
    "other": 19
  },
  "firewall_status": "防火墙已配置允许",
  "status": "活跃"
}
```

### 4. Inode 检查工具 (`check_inode`)

**文件位置**: `internal/tool/builtin/inode_check.go`

**功能特性**:
- 批量检查多主机的 inode 使用情况
- 智能识别高危分区（使用率超过阈值）
- 自动查找 inode 消耗最大的目录
- 简单统计和详细分析两种模式
- 总体风险评估和建议生成

**参数**:
- `hosts` (必填): 主机列表（数组）
- `threshold` (可选): 告警阈值（百分比），默认 80
- `find_large_dirs` (可选): 是否查找 inode 使用高的目录，默认 true

**输出内容**:
```json
{
  "results": [
    {
      "host": "server1",
      "inode_info": {
        "partitions": [...],
        "critical_partitions": [...],
        "critical": true,
        "warning": false
      },
      "status": "告警",
      "risk_level": "高"
    }
  ],
  "summary": {
    "total_hosts": 5,
    "normal_count": 3,
    "warning_count": 1,
    "alert_count": 1,
    "overall_status": "告警",
    "suggestions": [...]
  },
  "threshold": 80
}
```

## 技术实现亮点

### 1. 并发性能优化
- 性能分析工具使用 `sync.WaitGroup` 并发采集 CPU、内存、I/O、网络、进程数据
- 显著减少总体采集时间

### 2. 智能解析
- 使用正则表达式精确解析系统命令输出
- 容错处理，兼容不同 Linux 发行版的命令输出差异

### 3. 风险评估
- 性能评分算法综合考虑多个指标
- 多级风险等级（低/中/高）
- 针对性的优化建议

### 4. Go 语言最佳实践
- 符合 Go 惯用写法（Effective Go）
- 错误处理完整
- 代码结构清晰，易于维护
- 使用 Table-Driven 测试模式

## 集成方式

所有工具已自动注册到工具注册表：

```go
// 在 RegisterAllEnhanced 和 RegisterBasicEnhanced 中
tools := []tool.Tool{
    // ... 其他工具
    NewPerformanceAnalysisTool(),
    NewNetworkCheckTool(),
    NewPortCheckTool(),
    NewInodeCheckTool(),
}
```

## 使用示例

### AI 调用示例

```markdown
用户: "帮我分析一下 web-server 的性能"

AI: "我来为您分析 web-server 的性能情况"

工具调用: analyze_performance
参数: {
  "host": "web-server",
  "duration": 10,
  "top_n": 15
}
```

```markdown
用户: "检查 web-server 到 www.example.com 的网络连接"

AI: "正在测试网络连通性..."

工具调用: check_network
参数: {
  "host": "web-server",
  "target": "www.example.com",
  "count": 5
}
```

## 测试验证

已创建验证程序：`cmd/verify-tools/main.go`

运行验证：
```bash
go run ./cmd/verify-tools/main.go
```

测试文件：`internal/tool/builtin/advanced_tools_test.go`

包含单元测试和基准测试。

## 依赖的系统命令

### 性能分析工具
- `top` - CPU 和进程信息
- `free` - 内存使用情况
- `iostat` - I/O 统计（需要 sysstat 包）
- `ps` - 进程列表
- `cat /proc/net/dev` - 网络统计

### 网络诊断工具
- `ping` - ICMP 测试
- `nslookup` / `dig` - DNS 查询
- `traceroute` / `tracepath` - 路由追踪
- `ip addr show` - 网络接口

### 端口检查工具
- `ss` - Socket 统计
- `netstat` - 网络统计（备选）
- `lsof` - 打开文件列表
- `fuser` - 文件/端口占用
- `iptables` - 防火墙规则
- `firewall-cmd` - Firewalld 配置

### Inode 检查工具
- `df -i` - inode 使用情况
- `find` - 查找文件

## 扩展建议

### 短期优化
1. 添加缓存机制，避免短时间内重复采集
2. 支持历史数据对比，生成趋势图
3. 添加更多性能指标（如上下文切换、系统负载等）

### 长期规划
1. 集成 Prometheus/Grafana 指标
2. 实现自动告警规则
3. 机器学习驱动的异常检测
4. 容器环境支持（Docker/Kubernetes）

## 文件清单

```
internal/tool/builtin/
├── performance_analysis.go    (9.5 KB)
├── network_check.go           (6.1 KB)
├── port_check.go              (7.1 KB)
├── inode_check.go             (9.6 KB)
├── advanced_tools_test.go     (新增测试文件)
└── register.go                (已更新)

internal/tool/
└── tool.go                    (已添加 GetArrayParam 函数)

cmd/verify-tools/
└── main.go                    (验证程序)

docs/
└── ADVANCED_ANALYSIS_TOOLS.md (本文档)
```

## 总结

成功实现了 4 个高级分析工具，代码质量符合 Go 语言最佳实践：

- ✓ 完整的错误处理
- ✓ 并发安全
- ✓ 清晰的代码结构
- ✓ 充分的测试覆盖
- ✓ 详细的文档说明
- ✓ 通过编译验证

这些工具将显著提升 AI-Ops 系统的智能运维能力，为用户提供更精准、更全面的分析服务。
