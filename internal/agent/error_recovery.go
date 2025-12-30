package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// ErrorRecoveryStrategy 错误恢复策略
type ErrorRecoveryStrategy struct {
	ToolName         string   `json:"tool_name"`
	ErrorCode        string   `json:"error_code"`
	CanRetry         bool     `json:"can_retry"`
	RetryDelay       int      `json:"retry_delay"`       // 秒
	MaxRetries       int      `json:"max_retries"`
	AlternativeTools []string `json:"alternative_tools"`
	RecoveryPrompt   string   `json:"recovery_prompt"`
	RecoveryCommands []string `json:"recovery_commands"` // 自动执行的恢复命令
}

// ErrorRecoveryEngine 错误恢复引擎
type ErrorRecoveryEngine struct {
	strategies []ErrorRecoveryStrategy
}

// NewErrorRecoveryEngine 创建错误恢复引擎
func NewErrorRecoveryEngine() *ErrorRecoveryEngine {
	return &ErrorRecoveryEngine{
		strategies: builtInRecoveryStrategies(),
	}
}

// builtInRecoveryStrategies 内置恢复策略
func builtInRecoveryStrategies() []ErrorRecoveryStrategy {
	return []ErrorRecoveryStrategy{
		{
			ToolName:   "check_cpu",
			ErrorCode:  "connection refused",
			CanRetry:   true,
			RetryDelay: 2,
			MaxRetries: 3,
			AlternativeTools: []string{"run_command"},
			RecoveryPrompt:   "SSH 连接被拒绝，建议：1) 检查主机是否在线 2) 检查 SSH 服务状态 3) 尝试其他主机",
			RecoveryCommands: []string{
				"systemctl status sshd",
				"ping -c 3 {host}",
			},
		},
		{
			ToolName:   "query_log",
			ErrorCode:  "no such file or directory",
			CanRetry:   false,
			AlternativeTools: []string{"run_command"},
			RecoveryPrompt:   "日志文件不存在，正在尝试查找实际日志路径...",
			RecoveryCommands: []string{
				"ls -la /var/log/ | grep -E '(log|syslog)'",
				"find /var/log -name '*.log' -type f 2>/dev/null | head -10",
			},
		},
		{
			ToolName:   "check_disk",
			ErrorCode:  "permission denied",
			CanRetry:   false,
			RecoveryPrompt: "权限不足，建议检查用户权限或使用 sudo",
			RecoveryCommands: []string{
				"sudo df -h",
			},
		},
		{
			ToolName:   "check_memory",
			ErrorCode:  "timeout",
			CanRetry:   true,
			RetryDelay: 1,
			MaxRetries: 2,
			AlternativeTools: []string{"run_command"},
			RecoveryPrompt:   "命令执行超时，尝试使用简化命令...",
			RecoveryCommands: []string{
				"free -h",
			},
		},
		{
			ToolName:   "run_command",
			ErrorCode:  "command not found",
			CanRetry:   false,
			RecoveryPrompt: "命令不存在，请检查命令是否安装或路径是否正确",
		},
		{
			ToolName:   "check_process",
			ErrorCode:  "invalid pid",
			CanRetry:   false,
			RecoveryPrompt: "进程 ID 无效，可能进程已结束",
			RecoveryCommands: []string{
				"ps aux | grep {process_name}",
			},
		},
		{
			ToolName:         "*",
			ErrorCode:        "connection refused",
			CanRetry:         true,
			RetryDelay:       2,
			MaxRetries:       2,
			RecoveryPrompt:   "连接被拒绝，请检查主机状态和网络连接",
			RecoveryCommands: []string{"ping -c 3 {host}"},
		},
		{
			ToolName:         "*",
			ErrorCode:        "timeout",
			CanRetry:         true,
			RetryDelay:       1,
			MaxRetries:       2,
			RecoveryPrompt:   "操作超时，正在重试...",
		},
	}
}

// RecoverFromError 尝试从错误中恢复
func (e *ErrorRecoveryEngine) RecoverFromError(
	ctx context.Context,
	agent *Agent,
	record ToolCallRecord,
	messages []llm.Message,
) (shouldRetry bool, newToolCalls []llm.ToolCall, recoveryPrompt string, err error) {

	// 1. 分析错误类型
	strategy := e.findStrategy(record.Tool, record.Error)
	if strategy == nil {
		return false, nil, "", fmt.Errorf("无匹配的恢复策略")
	}

	logger.Info("找到错误恢复策略",
		zap.String("tool", record.Tool),
		zap.String("error_code", strategy.ErrorCode),
		zap.Bool("can_retry", strategy.CanRetry),
	)

	// 2. 生成恢复提示
	recoveryPrompt = e.buildRecoveryPrompt(strategy, record)

	// 3. 判断是否可以重试
	if !strategy.CanRetry {
		return false, nil, recoveryPrompt, nil
	}

	// 4. 尝试自动恢复命令
	if len(strategy.RecoveryCommands) > 0 {
		recoveryResult := e.tryAutoRecovery(ctx, agent, record, strategy)
		if recoveryResult != "" {
			recoveryPrompt += "\n\n自动恢复尝试:\n" + recoveryResult
		}
	}

	// 5. 尝试替代工具
	if len(strategy.AlternativeTools) > 0 {
		newToolCalls = e.buildAlternativeToolCalls(strategy, record)
		if len(newToolCalls) > 0 {
			return true, newToolCalls, recoveryPrompt, nil
		}
	}

	// 6. 简单重试
	return true, nil, recoveryPrompt, nil
}

// findStrategy 查找匹配的策略
func (e *ErrorRecoveryEngine) findStrategy(toolName string, errorMsg string) *ErrorRecoveryStrategy {
	lowerError := strings.ToLower(errorMsg)

	for i := range e.strategies {
		strategy := &e.strategies[i]
		if strategy.ToolName == toolName {
			// 精确匹配错误代码
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

// buildRecoveryPrompt 构建恢复提示
func (e *ErrorRecoveryEngine) buildRecoveryPrompt(strategy *ErrorRecoveryStrategy, record ToolCallRecord) string {
	return fmt.Sprintf("⚠️ 工具 %s 执行失败\n\n错误：%s\n\n建议：%s",
		strategy.ToolName,
		record.Error,
		strategy.RecoveryPrompt,
	)
}

// buildAlternativeToolCalls 构建替代工具调用
func (e *ErrorRecoveryEngine) buildAlternativeToolCalls(strategy *ErrorRecoveryStrategy, failedRecord ToolCallRecord) []llm.ToolCall {
	var newCalls []llm.ToolCall

	for _, altTool := range strategy.AlternativeTools {
		newCall := llm.ToolCall{
			ID: failedRecord.ID + "_alt_" + altTool,
			Function: llm.FunctionCall{
				Name:      altTool,
				Arguments: e.convertParams(failedRecord.Tool, altTool, failedRecord.Params),
			},
		}
		newCalls = append(newCalls, newCall)
	}

	return newCalls
}

// convertParams 参数转换
func (e *ErrorRecoveryEngine) convertParams(fromTool, toTool string, params map[string]interface{}) string {
	// 提取 host 参数
	var host string
	if h, ok := params["host"]; ok {
		host = fmt.Sprintf("%v", h)
	}

	// 根据工具类型转换
	switch {
	case fromTool == "check_cpu" && toTool == "run_command":
		return fmt.Sprintf(`{"host": "%s", "command": "top -bn1 | head -20"}`, host)

	case fromTool == "check_memory" && toTool == "run_command":
		return fmt.Sprintf(`{"host": "%s", "command": "free -h"}`, host)

	case fromTool == "check_disk" && toTool == "run_command":
		return fmt.Sprintf(`{"host": "%s", "command": "df -h"}`, host)

	case fromTool == "query_log" && toTool == "run_command":
		lines := 10
		if l, ok := params["lines"]; ok {
			lines = int(l.(float64))
		}
		return fmt.Sprintf(`{"host": "%s", "command": "journalctl -n %d --no-pager"}`, host, lines)

	default:
		// 尝试直接转换参数
		return fmt.Sprintf(`{"host": "%s"}`, host)
	}
}

// tryAutoRecovery 尝试自动恢复
func (e *ErrorRecoveryEngine) tryAutoRecovery(
	ctx context.Context,
	agent *Agent,
	record ToolCallRecord,
	strategy *ErrorRecoveryStrategy,
) string {
	if agent == nil || agent.sshPool == nil {
		return ""
	}

	var results []string

	// 获取主机
	var host string
	if h, ok := record.Params["host"]; ok {
		host = fmt.Sprintf("%v", h)
	}
	if host == "" {
		return ""
	}

	for _, cmd := range strategy.RecoveryCommands {
		// 替换占位符
		cmd = strings.ReplaceAll(cmd, "{host}", host)
		if processName, ok := record.Params["process_name"]; ok {
			cmd = strings.ReplaceAll(cmd, "{process_name}", fmt.Sprintf("%v", processName))
		}

		logger.Debug("执行自动恢复命令", zap.String("host", host), zap.String("command", cmd))

		// 实际执行 SSH 命令
		output, err := agent.sshPool.Exec(host, cmd)
		if err != nil {
			results = append(results, fmt.Sprintf("执行 %s 失败: %v", cmd, err))
		} else {
			// 截断过长的输出
			if len(output) > 500 {
				output = output[:500] + "...(truncated)"
			}
			results = append(results, fmt.Sprintf("执行 %s:\n%s", cmd, output))
		}
	}

	if len(results) > 0 {
		return strings.Join(results, "\n")
	}
	return ""
}

// RecoveryLog 恢复日志
type RecoveryLog struct {
	Timestamp   time.Time `json:"timestamp"`
	ToolName    string    `json:"tool_name"`
	Error       string    `json:"error"`
	Strategy    string    `json:"strategy"`
	Success     bool      `json:"success"`
	Alternative string    `json:"alternative,omitempty"`
}

// LogRecovery 记录恢复尝试
func (e *ErrorRecoveryEngine) LogRecovery(log RecoveryLog) {
	logger.Info("错误恢复尝试",
		zap.Time("timestamp", log.Timestamp),
		zap.String("tool", log.ToolName),
		zap.String("error", log.Error),
		zap.String("strategy", log.Strategy),
		zap.Bool("success", log.Success),
		zap.String("alternative", log.Alternative),
	)
}
