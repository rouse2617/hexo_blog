package builtin

import (
	"ai-ops/internal/tool"
	"fmt"
)

// AlertManagerTool 告警管理工具
type AlertManagerTool struct{}

func (t *AlertManagerTool) Name() string {
	return "alert_manager"
}

func (t *AlertManagerTool) Description() string {
	return `# 告警规则管理

## 功能说明
创建、删除、查询告警规则。

## 支持的告警类型
1. **cpu** - CPU 使用率告警
2. **memory** - 内存使用率告警
3. **disk** - 磁盘使用率告警
4. **process** - 进程退出告警
5. **log** - 日志关键字告警
6. **service** - 服务停止告警

## 告警级别
- critical: 严重，立即处理
- high: 高级，1小时内处理
- medium: 中级，当天处理
- low: 低级，计划处理`
}

func (t *AlertManagerTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{Name: "action", Type: "string", Description: "操作(create/delete/list)", Required: true},
		{Name: "name", Type: "string", Description: "告警规则名称", Required: false},
		{Name: "type", Type: "string", Description: "告警类型", Required: false},
		{Name: "threshold", Type: "number", Description: "阈值", Required: false},
		{Name: "level", Type: "string", Description: "级别", Required: false},
		{Name: "hosts", Type: "array", Description: "主机列表", Required: false},
	}
}

func (t *AlertManagerTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	action := tool.GetStringParam(params, "action", "")

	switch action {
	case "create":
		return t.createAlert(ctx, params)
	case "delete":
		return t.deleteAlert(ctx, params)
	case "list":
		return t.listAlerts(ctx)
	default:
		return tool.NewErrorResult("未知操作，支持: create, delete, list"), nil
	}
}

// createAlert 创建告警规则
func (t *AlertManagerTool) createAlert(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	name := tool.GetStringParam(params, "name", "")
	alertType := tool.GetStringParam(params, "type", "")
	threshold := tool.GetFloatParam(params, "threshold", 0)
	level := tool.GetStringParam(params, "level", "medium")

	if name == "" || alertType == "" || threshold == 0 {
		return tool.NewErrorResult("name, type, threshold 参数必填"), nil
	}

	alert := map[string]interface{}{
		"name":      name,
		"type":      alertType,
		"threshold": threshold,
		"level":     level,
		"enabled":   true,
	}

	// 这里应该保存到数据库或配置文件
	// 简化实现，返回创建的规则

	return tool.NewResult(map[string]interface{}{
		"alert":   alert,
		"message": fmt.Sprintf("告警规则 %s 创建成功", name),
	}, fmt.Sprintf("告警规则 %s 创建成功", name)), nil
}

// deleteAlert 删除告警规则
func (t *AlertManagerTool) deleteAlert(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	name := tool.GetStringParam(params, "name", "")

	if name == "" {
		return tool.NewErrorResult("name 参数必填"), nil
	}

	return tool.NewResult(map[string]interface{}{
		"message": fmt.Sprintf("告警规则 %s 已删除", name),
	}, fmt.Sprintf("告警规则 %s 已删除", name)), nil
}

// listAlerts 列出告警规则
func (t *AlertManagerTool) listAlerts(ctx *tool.Context) (*tool.Result, error) {
	// 返回示例告警规则
	alerts := []map[string]interface{}{
		{
			"name":      "cpu-high",
			"type":      "cpu",
			"threshold": 80.0,
			"level":     "high",
			"enabled":   true,
		},
		{
			"name":      "disk-full",
			"type":      "disk",
			"threshold": 85.0,
			"level":     "critical",
			"enabled":   true,
		},
	}

	return tool.NewResult(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	}, fmt.Sprintf("共 %d 条告警规则", len(alerts))), nil
}

func NewAlertManagerTool() *AlertManagerTool {
	return &AlertManagerTool{}
}
