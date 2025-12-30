package agent

import "time"

// ToolTimeoutConfig 工具超时配置
var ToolTimeoutConfig = map[string]time.Duration{
	// 快速检查类（10秒）
	"check_cpu":    10 * time.Second,
	"check_memory": 10 * time.Second,
	"check_load":   10 * time.Second,
	"check_uptime": 10 * time.Second,
	"list_hosts":   5 * time.Second,

	// 中等耗时类（30秒）
	"check_disk":     30 * time.Second,
	"check_network":  30 * time.Second,
	"check_process":  30 * time.Second,
	"check_inode":    30 * time.Second,
	"check_io":       30 * time.Second,
	"check_users":    15 * time.Second,
	"check_services": 30 * time.Second,

	// 可能耗时较长类（60秒）
	"query_log":       60 * time.Second,
	"intrusion_check": 60 * time.Second,

	// 自定义命令（可能很长）
	"run_command": 120 * time.Second,
}

// DefaultToolTimeout 默认工具超时
const DefaultToolTimeout = 30 * time.Second

// GetToolTimeout 获取工具超时时间
func GetToolTimeout(toolName string) time.Duration {
	if timeout, ok := ToolTimeoutConfig[toolName]; ok {
		return timeout
	}
	return DefaultToolTimeout
}

// SetToolTimeout 动态设置工具超时（运行时调整）
func SetToolTimeout(toolName string, timeout time.Duration) {
	ToolTimeoutConfig[toolName] = timeout
}
