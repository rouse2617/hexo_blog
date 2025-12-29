# 错误恢复机制快速参考

## 文件清单

✅ **已创建的文件**：
- `internal/agent/error_recovery.go` - 核心实现
- `internal/agent/error_recovery_test.go` - 单元测试
- `internal/agent/agent.go` - 已修改，集成错误恢复
- `scripts/error_recovery_demo.go` - 演示程序
- `docs/ERROR_RECOVERY.md` - 完整文档

## 核心功能

### 1. 错误恢复引擎

```go
// 创建引擎
engine := NewErrorRecoveryEngine()

// 恢复错误
shouldRetry, newCalls, recoveryPrompt, err := engine.RecoverFromError(
    ctx, agent, record, messages,
)
```

### 2. 策略匹配

- **精确匹配**：工具名 + 错误代码
- **通用匹配**：`*` + 错误代码
- **优先级**：精确匹配 > 通用匹配

### 3. 恢复方式

1. **重试机制**：根据策略自动重试
2. **替代工具**：使用备用工具执行相同任务
3. **自动命令**：执行诊断/修复命令
4. **恢复提示**：向用户提供操作建议

## 支持的错误类型

| 工具 | 错误 | 可重试 | 替代方案 |
|------|------|--------|----------|
| check_cpu | connection refused | ✓ (3次) | run_command |
| query_log | no such file | ✗ | run_command |
| check_disk | permission denied | ✗ | - |
| check_memory | timeout | ✓ (2次) | run_command |
| run_command | command not found | ✗ | - |
| check_process | invalid pid | ✗ | - |
| * | timeout | ✓ (2次) | - |
| * | connection refused | ✓ (2次) | - |

## 代码示例

### 基本使用

```go
// 工具执行失败后
if err != nil {
    // 尝试恢复
    shouldRetry, newCalls, recoveryPrompt, _ :=
        a.errorRecovery.RecoverFromError(ctx, a, record, nil)

    // 使用恢复提示
    if recoveryPrompt != "" {
        logger.Info("恢复提示", zap.String("prompt", recoveryPrompt))
    }

    // 执行替代工具
    if shouldRetry && len(newCalls) > 0 {
        for _, call := range newCalls {
            // 执行替代工具...
        }
    }
}
```

### 添加自定义策略

```go
func builtInRecoveryStrategies() []ErrorRecoveryStrategy {
    return []ErrorRecoveryStrategy{
        {
            ToolName:         "my_tool",
            ErrorCode:        "my_error",
            CanRetry:         true,
            RetryDelay:       1,
            MaxRetries:       3,
            AlternativeTools: []string{"backup_tool"},
            RecoveryPrompt:   "检测到错误，尝试备用方案",
            RecoveryCommands: []string{"diag_command_1", "diag_command_2"},
        },
    }
}
```

## 测试

### 运行单元测试
```bash
go test -v ./internal/agent -run TestErrorRecovery
```

### 运行演示程序
```bash
go run scripts/error_recovery_demo.go
```

### 性能测试
```bash
go test -bench=. ./internal/agent -run BenchmarkFindStrategy
```

## 关键数据结构

```go
type ErrorRecoveryStrategy struct {
    ToolName         string   // 目标工具（"*" 表示通用）
    ErrorCode        string   // 错误关键词
    CanRetry         bool     // 是否可重试
    RetryDelay       int      // 重试延迟（秒）
    MaxRetries       int      // 最大重试次数
    AlternativeTools []string // 替代工具列表
    RecoveryPrompt   string   // 恢复建议
    RecoveryCommands []string // 自动执行命令
}

type ErrorRecoveryEngine struct {
    strategies []ErrorRecoveryStrategy
}
```

## 集成要点

在 `Agent` 结构体中：
```go
type Agent struct {
    // ...
    errorRecovery *ErrorRecoveryEngine
}
```

在 `NewAgent` 中初始化：
```go
return &Agent{
    // ...
    errorRecovery: NewErrorRecoveryEngine(),
}
```

在 `executeToolCalls` 中调用：
```go
if err != nil || !result.Success {
    // 尝试错误恢复
    shouldRetry, newCalls, recoveryPrompt, _ :=
        a.errorRecovery.RecoverFromError(ctx, a, record, nil)
    // ...
}
```

## 下一步优化

1. **自动恢复命令执行**：实现真正的 SSH 命令执行
2. **重试机制**：实现延时重试逻辑
3. **策略学习**：基于历史数据优化策略
4. **并发恢复**：多个替代工具并行执行
5. **恢复统计**：收集成功率数据

## 相关文档

- 完整文档：`docs/ERROR_RECOVERY.md`
- 演示代码：`scripts/error_recovery_demo.go`
- 测试代码：`internal/agent/error_recovery_test.go`
