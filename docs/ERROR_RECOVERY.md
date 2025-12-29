# 智能错误恢复机制文档

## 概述

智能错误恢复机制是 AI-Ops Agent 的核心组件之一，用于在工具执行失败时自动进行错误分析和恢复。该机制通过预定义的恢复策略，能够：

1. **识别错误类型**：自动分析工具执行失败的原因
2. **提供恢复建议**：根据错误类型提供针对性的解决方案
3. **尝试自动恢复**：执行替代工具或自动恢复命令
4. **记录恢复日志**：跟踪所有恢复尝试的结果

## 架构设计

### 核心组件

```
┌─────────────────────────────────────────────────────────┐
│                  Agent Enhanced                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │         Error Recovery Engine                      │ │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────┐│ │
│  │  │ Strategy    │  │ Parameter    │  │ Auto      ││ │
│  │  │ Finder      │  │ Converter    │  │ Recovery  ││ │
│  │  └─────────────┘  └──────────────┘  └───────────┘│ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### 数据结构

#### ErrorRecoveryStrategy
```go
type ErrorRecoveryStrategy struct {
    ToolName         string   // 目标工具名称
    ErrorCode        string   // 错误代码/关键词
    CanRetry         bool     // 是否可重试
    RetryDelay       int      // 重试延迟（秒）
    MaxRetries       int      // 最大重试次数
    AlternativeTools []string // 替代工具列表
    RecoveryPrompt   string   // 恢复提示信息
    RecoveryCommands []string // 自动执行的恢复命令
}
```

#### ErrorRecoveryEngine
```go
type ErrorRecoveryEngine struct {
    strategies []ErrorRecoveryStrategy // 预定义策略列表
}
```

## 恢复策略

### 内置策略

#### 1. SSH 连接拒绝
- **工具**: `check_cpu`
- **错误**: `connection refused`
- **可重试**: 是（最多 3 次，延迟 2 秒）
- **替代方案**: `run_command`
- **恢复建议**: 检查主机在线状态、SSH 服务状态
- **自动命令**:
  ```bash
  systemctl status sshd
  ping -c 3 {host}
  ```

#### 2. 日志文件不存在
- **工具**: `query_log`
- **错误**: `no such file or directory`
- **可重试**: 否
- **替代方案**: `run_command`
- **恢复建议**: 查找实际日志路径
- **自动命令**:
  ```bash
  ls -la /var/log/ | grep -E '(log|syslog)'
  find /var/log -name '*.log' -type f 2>/dev/null | head -10
  ```

#### 3. 权限不足
- **工具**: `check_disk`
- **错误**: `permission denied`
- **可重试**: 否
- **替代方案**: 无
- **恢复建议**: 使用 sudo 提升权限
- **自动命令**:
  ```bash
  sudo df -h
  ```

#### 4. 命令超时
- **工具**: `check_memory`
- **错误**: `timeout`
- **可重试**: 是（最多 2 次，延迟 1 秒）
- **替代方案**: `run_command`
- **恢复建议**: 使用简化命令
- **自动命令**:
  ```bash
  free -h
  ```

#### 5. 命令不存在
- **工具**: `run_command`
- **错误**: `command not found`
- **可重试**: 否
- **替代方案**: 无
- **恢复建议**: 检查命令是否安装或路径是否正确

#### 6. 进程 ID 无效
- **工具**: `check_process`
- **错误**: `invalid pid`
- **可重试**: 否
- **替代方案**: 无
- **恢复建议**: 进程可能已结束
- **自动命令**:
  ```bash
  ps aux | grep {process_name}
  ```

### 通用策略

除了针对特定工具的策略外，还有通用的 `*` 策略，可以匹配任何工具的常见错误：

- **连接拒绝** (connection refused)
- **超时** (timeout)

## 工作流程

### 1. 错误检测
```go
if err != nil || !result.Success {
    // 记录错误
    record.Error = err.Error()

    // 触发错误恢复
    shouldRetry, newCalls, recoveryPrompt, _ :=
        errorRecovery.RecoverFromError(ctx, agent, record, messages)
}
```

### 2. 策略匹配
```go
strategy := findStrategy(toolName, errorMsg)
// 精确匹配 -> 通用匹配 -> 未找到
```

### 3. 恢复执行
```
┌──────────────┐
│ 工具执行失败  │
└──────┬───────┘
       │
       ▼
┌──────────────┐     否
│ 找到策略?  ─────────► 返回原始错误
└──────┬───────┘
       │ 是
       ▼
┌──────────────┐     否
│ 可重试?    ─────────► 提供恢复建议
└──────┬───────┘
       │ 是
       ▼
┌──────────────┐     有
│ 有替代工具? ────────► 执行替代工具
└──────┬───────┘
       │ 无
       ▼
┌──────────────┐
│ 执行自动恢复  │
│ 命令         │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 返回恢复结果  │
└──────────────┘
```

### 4. 参数转换
当使用替代工具时，需要转换参数：

```go
// check_cpu → run_command
{"host": "192.168.1.1"}
↓
{"host": "192.168.1.1", "command": "top -bn1 | head -20"}

// query_log → run_command
{"host": "server1", "lines": 50}
↓
{"host": "server1", "command": "journalctl -n 50 --no-pager"}
```

## 使用示例

### 场景 1: SSH 连接失败

**原始调用**:
```json
{
  "tool": "check_cpu",
  "params": {"host": "192.168.1.100"}
}
```

**错误信息**:
```
dial tcp 192.168.1.100:22: connect: connection refused
```

**恢复流程**:
1. 识别错误：`connection refused`
2. 匹配策略：check_cpu + connection refused
3. 执行替代工具：`run_command` with `ping -c 3 192.168.1.100`
4. 返回结果：
```
⚠️ 工具 check_cpu 执行失败

错误：dial tcp 192.168.1.100:22: connect: connection refused

建议：SSH 连接被拒绝，尝试使用 run_command 替代

自动恢复尝试:
执行: ping -c 3 192.168.1.100
```

### 场景 2: 日志文件不存在

**原始调用**:
```json
{
  "tool": "query_log",
  "params": {
    "host": "server1",
    "log_type": "application",
    "lines": 100
  }
}
```

**错误信息**:
```
open /var/log/application.log: no such file or directory
```

**恢复流程**:
1. 识别错误：`no such file or directory`
2. 匹配策略：query_log + no such file
3. 执行查找命令：
   - `ls -la /var/log/ | grep -E '(log|syslog)'`
   - `find /var/log -name '*.log' -type f`
4. 建议使用 `run_command` 替代

## 扩展指南

### 添加新的恢复策略

在 `error_recovery.go` 的 `builtInRecoveryStrategies()` 函数中添加：

```go
{
    ToolName:   "your_tool_name",
    ErrorCode:  "error_pattern",
    CanRetry:   true,
    RetryDelay: 2,
    MaxRetries: 3,
    AlternativeTools: []string{"alternative_tool"},
    RecoveryPrompt:   "恢复建议说明",
    RecoveryCommands: []string{
        "auto_recovery_command_1",
        "auto_recovery_command_2",
    },
},
```

### 自定义参数转换

在 `convertParams()` 方法中添加新的转换规则：

```go
case fromTool == "your_tool" && toTool == "run_command":
    param1 := "default"
    if p, ok := params["param1"]; ok {
        param1 = fmt.Sprintf("%v", p)
    }
    return fmt.Sprintf(`{"host": "%s", "command": "your command with %s"}`, host, param1)
```

## 性能考虑

1. **策略查找优化**：使用预编译的正则表达式或哈希表
2. **并行恢复**：多个替代工具可以并行执行
3. **缓存机制**：缓存常见错误的恢复策略
4. **超时控制**：自动恢复命令应该有超时限制

## 最佳实践

1. **明确错误分类**：区分可重试和不可重试的错误
2. **提供有用建议**：恢复提示应该清晰且可操作
3. **避免无限重试**：设置合理的最大重试次数
4. **记录详细日志**：便于问题诊断和策略优化
5. **渐进式恢复**：从简单到复杂逐步尝试恢复方案

## 测试

运行测试：
```bash
go test -v ./internal/agent -run TestErrorRecovery
```

运行基准测试：
```bash
go test -bench=. ./internal/agent -run BenchmarkFindStrategy
```

## 相关文件

- `internal/agent/error_recovery.go` - 错误恢复引擎实现
- `internal/agent/error_recovery_test.go` - 单元测试
- `internal/agent/agent.go` - Agent 主逻辑（集成错误恢复）
- `scripts/error_recovery_demo.go` - 演示程序
