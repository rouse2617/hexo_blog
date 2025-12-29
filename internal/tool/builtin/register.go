package builtin

import (
	"ai-ops/internal/tool"
)

// RegisterAll 注册所有内置工具（使用增强版描述）
func RegisterAll(registry *tool.Registry, getHostsFunc func(group string) []HostBasicInfo) error {
	return RegisterAllEnhanced(registry, getHostsFunc)
}

// RegisterAllEnhanced 注册所有增强版工具
func RegisterAllEnhanced(registry *tool.Registry, getHostsFunc func(group string) []HostBasicInfo) error {
	tools := []tool.Tool{
		NewQueryLogToolEnhanced(),
		NewCheckCPUToolEnhanced(),
		NewCheckMemoryToolEnhanced(),
		NewCheckDiskToolEnhanced(),
		NewCheckProcessToolEnhanced(),
		NewRunCommandToolEnhanced(),
		NewListHostsToolEnhanced(getHostsFunc),
		&IntrusionDetectionTool{},
		NewTrendAnalysisTool(),
		NewRootCauseDiagnosisTool(),
		NewAnomalyDetectionTool(),
		NewAutoRecoveryTool(),
		NewAlertManagerTool(),
		// 高级分析工具
		NewPerformanceAnalysisTool(),
		NewNetworkCheckTool(),
		NewPortCheckTool(),
		NewInodeCheckTool(),
	}

	for _, t := range tools {
		if err := registry.RegisterBuiltin(t); err != nil {
			return err
		}
	}

	return nil
}

// RegisterAllStandard 注册标准版工具（向后兼容）
func RegisterAllStandard(registry *tool.Registry, getHostsFunc func(group string) []HostBasicInfo) error {
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

// RegisterBasic 注册基础工具（不包含需要外部依赖的工具，使用增强版）
func RegisterBasic(registry *tool.Registry) error {
	return RegisterBasicEnhanced(registry)
}

// RegisterBasicEnhanced 注册增强版基础工具
func RegisterBasicEnhanced(registry *tool.Registry) error {
	tools := []tool.Tool{
		NewQueryLogToolEnhanced(),
		NewCheckCPUToolEnhanced(),
		NewCheckMemoryToolEnhanced(),
		NewCheckDiskToolEnhanced(),
		NewCheckProcessToolEnhanced(),
		NewRunCommandToolEnhanced(),
		&IntrusionDetectionTool{},
		// 高级分析工具
		NewPerformanceAnalysisTool(),
		NewNetworkCheckTool(),
		NewPortCheckTool(),
		NewInodeCheckTool(),
		NewTrendAnalysisTool(),
		NewRootCauseDiagnosisTool(),
		NewAnomalyDetectionTool(),
		NewAutoRecoveryTool(),
		NewAlertManagerTool(),
		// 存储专家工具
		NewStorageCheckTool(),
		NewRAIDCheckTool(),
	}

	for _, t := range tools {
		if err := registry.RegisterBuiltin(t); err != nil {
			return err
		}
	}

	return nil
}

// RegisterBasicStandard 注册标准版基础工具（向后兼容）
func RegisterBasicStandard(registry *tool.Registry) error {
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
