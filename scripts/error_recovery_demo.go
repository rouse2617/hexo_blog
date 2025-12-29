package main

import (
	"fmt"
	"strings"
)

// 简化版的错误恢复引擎演示
type ErrorRecoveryStrategy struct {
	ToolName         string
	ErrorCode        string
	CanRetry         bool
	AlternativeTools []string
	RecoveryPrompt   string
}

type ErrorRecoveryEngine struct {
	strategies []ErrorRecoveryStrategy
}

func NewErrorRecoveryEngine() *ErrorRecoveryEngine {
	return &ErrorRecoveryEngine{
		strategies: []ErrorRecoveryStrategy{
			{
				ToolName:         "check_cpu",
				ErrorCode:        "connection refused",
				CanRetry:         true,
				AlternativeTools: []string{"run_command"},
				RecoveryPrompt:   "SSH 连接被拒绝，尝试使用 run_command 替代",
			},
			{
				ToolName:         "query_log",
				ErrorCode:        "no such file or directory",
				CanRetry:         false,
				AlternativeTools: []string{"run_command"},
				RecoveryPrompt:   "日志文件不存在，正在尝试查找实际日志路径",
			},
			{
				ToolName:         "*",
				ErrorCode:        "timeout",
				CanRetry:         true,
				AlternativeTools: []string{},
				RecoveryPrompt:   "操作超时，正在重试",
			},
		},
	}
}

func (e *ErrorRecoveryEngine) findStrategy(toolName, errorMsg string) *ErrorRecoveryStrategy {
	lowerError := strings.ToLower(errorMsg)

	// 先尝试精确匹配
	for i := range e.strategies {
		strategy := &e.strategies[i]
		if strategy.ToolName == toolName {
			if strings.Contains(lowerError, strings.ToLower(strategy.ErrorCode)) {
				return strategy
			}
		}
	}

	// 尝试通用策略
	for i := range e.strategies {
		strategy := &e.strategies[i]
		if strategy.ToolName == "*" && strings.Contains(lowerError, strings.ToLower(strategy.ErrorCode)) {
			return strategy
		}
	}

	return nil
}

func main() {
	engine := NewErrorRecoveryEngine()

	fmt.Println("=== 智能错误恢复机制演示 ===\n")

	testCases := []struct {
		name     string
		toolName string
		errorMsg string
	}{
		{
			name:     "SSH 连接拒绝",
			toolName: "check_cpu",
			errorMsg: "dial tcp 192.168.1.1:22: connect: connection refused",
		},
		{
			name:     "日志文件不存在",
			toolName: "query_log",
			errorMsg: "open /var/log/app.log: no such file or directory",
		},
		{
			name:     "命令超时",
			toolName: "check_memory",
			errorMsg: "context deadline exceeded after 30s",
		},
		{
			name:     "未知错误",
			toolName: "check_disk",
			errorMsg: "something unexpected happened",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("【测试案例】%s\n", tc.name)
		fmt.Printf("工具: %s\n", tc.toolName)
		fmt.Printf("错误: %s\n", tc.errorMsg)

		strategy := engine.findStrategy(tc.toolName, tc.errorMsg)

		if strategy != nil {
			fmt.Printf("✓ 找到恢复策略:\n")
			fmt.Printf("  - 错误代码: %s\n", strategy.ErrorCode)
			fmt.Printf("  - 可重试: %v\n", strategy.CanRetry)
			fmt.Printf("  - 替代工具: %v\n", strategy.AlternativeTools)
			fmt.Printf("  - 恢复建议: %s\n", strategy.RecoveryPrompt)
		} else {
			fmt.Printf("✗ 未找到匹配的恢复策略\n")
		}
		fmt.Println()
	}

	fmt.Println("=== 恢复策略统计 ===")
	fmt.Printf("已加载 %d 条恢复策略\n", len(engine.strategies))
	fmt.Println("\n策略列表:")
	for i, s := range engine.strategies {
		fmt.Printf("%d. 工具: %s | 错误: %s | 可重试: %v\n",
			i+1, s.ToolName, s.ErrorCode, s.CanRetry)
	}
}
