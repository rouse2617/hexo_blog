package builtin

import (
	"ai-ops/internal/tool"
)

// RegisterAll 注册所有内置工具
func RegisterAll(registry *tool.Registry, getHostsFunc func(group string) []HostBasicInfo) error {
	tools := []tool.Tool{
		NewQueryLogTool(),
		NewCheckCPUTool(),
		NewCheckMemoryTool(),
		NewCheckDiskTool(),
		NewCheckProcessTool(),
		NewRunCommandTool(),
		NewListHostsTool(getHostsFunc),
	}

	for _, t := range tools {
		if err := registry.RegisterBuiltin(t); err != nil {
			return err
		}
	}

	return nil
}

// RegisterBasic 注册基础工具（不包含需要外部依赖的工具）
func RegisterBasic(registry *tool.Registry) error {
	tools := []tool.Tool{
		NewQueryLogTool(),
		NewCheckCPUTool(),
		NewCheckMemoryTool(),
		NewCheckDiskTool(),
		NewCheckProcessTool(),
		NewRunCommandTool(),
	}

	for _, t := range tools {
		if err := registry.RegisterBuiltin(t); err != nil {
			return err
		}
	}

	return nil
}
