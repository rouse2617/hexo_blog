package builtin

import (
	"testing"

	"ai-ops/internal/tool"
)

// TestAutoRecoveryTool 测试自动恢复工具
func TestAutoRecoveryTool(t *testing.T) {
	tool := NewAutoRecoveryTool()

	// 测试工具名称
	if tool.Name() != "auto_recover" {
		t.Errorf("期望工具名称为 'auto_recover'，实际为 '%s'", tool.Name())
	}

	// 测试描述
	desc := tool.Description()
	if desc == "" {
		t.Error("工具描述不能为空")
	}

	// 测试参数定义
	params := tool.Parameters()
	if len(params) < 3 {
		t.Errorf("参数定义不完整，期望至少 3 个，实际 %d", len(params))
	}

	// 验证必填参数
	foundHost := false
	foundIssueType := false
	foundConfirm := false
	for _, p := range params {
		switch p.Name {
		case "host":
			foundHost = true
			if !p.Required {
				t.Error("host 参数应该是必填的")
			}
		case "issue_type":
			foundIssueType = true
			if !p.Required {
				t.Error("issue_type 参数应该是必填的")
			}
		case "confirm":
			foundConfirm = true
		}
	}

	if !foundHost {
		t.Error("缺少 host 参数定义")
	}
	if !foundIssueType {
		t.Error("缺少 issue_type 参数定义")
	}
	if !foundConfirm {
		t.Error("缺少 confirm 参数定义")
	}
}

// TestAutoRecoveryTool_Execute_NoConfirm 测试未确认时的错误处理
func TestAutoRecoveryTool_Execute_NoConfirm(t *testing.T) {
	tool := NewAutoRecoveryTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"host":       "test-host",
		"issue_type": "disk_full",
		"confirm":    false,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if result.Success {
		t.Error("未确认时应该返回失败结果")
	}

	if result.Error == "" {
		t.Error("应该包含错误信息")
	}
}

// TestAutoRecoveryTool_Execute_UnknownIssueType 测试未知问题类型
func TestAutoRecoveryTool_Execute_UnknownIssueType(t *testing.T) {
	tool := NewAutoRecoveryTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"host":       "test-host",
		"issue_type": "unknown_type",
		"confirm":    true,
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if result.Success {
		t.Error("未知问题类型应该返回失败结果")
	}
}

// TestAlertManagerTool 测试告警管理工具
func TestAlertManagerTool(t *testing.T) {
	tool := NewAlertManagerTool()

	// 测试工具名称
	if tool.Name() != "alert_manager" {
		t.Errorf("期望工具名称为 'alert_manager'，实际为 '%s'", tool.Name())
	}

	// 测试描述
	desc := tool.Description()
	if desc == "" {
		t.Error("工具描述不能为空")
	}

	// 测试参数定义
	params := tool.Parameters()
	if len(params) < 6 {
		t.Errorf("参数定义不完整，期望至少 6 个，实际 %d", len(params))
	}

	// 验证 action 参数是必填的
	foundAction := false
	for _, p := range params {
		if p.Name == "action" {
			foundAction = true
			if !p.Required {
				t.Error("action 参数应该是必填的")
			}
			break
		}
	}

	if !foundAction {
		t.Error("缺少 action 参数定义")
	}
}

// TestAlertManagerTool_List 测试列出告警
func TestAlertManagerTool_List(t *testing.T) {
	tool := NewAlertManagerTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"action": "list",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if !result.Success {
		t.Errorf("列出告警应该成功: %s", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("返回数据格式错误")
	}

	alerts, ok := data["alerts"].([]map[string]interface{})
	if !ok {
		t.Fatal("返回数据中缺少告警列表")
	}

	if len(alerts) == 0 {
		t.Error("应该返回示例告警数据")
	}
}

// TestAlertManagerTool_Create 测试创建告警
func TestAlertManagerTool_Create(t *testing.T) {
	tool := NewAlertManagerTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"action":    "create",
		"name":      "test-alert",
		"type":      "cpu",
		"threshold": 80.0,
		"level":     "high",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if !result.Success {
		t.Errorf("创建告警应该成功: %s", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("返回数据格式错误")
	}

	alert, ok := data["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("返回数据中缺少告警对象")
	}

	if alert["name"] != "test-alert" {
		t.Error("告警名称不正确")
	}
}

// TestAlertManagerTool_Create_MissingParams 测试缺少必填参数
func TestAlertManagerTool_Create_MissingParams(t *testing.T) {
	tool := NewAlertManagerTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"action": "create",
		// 缺少 name, type, threshold
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if result.Success {
		t.Error("缺少必填参数时应该返回失败")
	}
}

// TestAlertManagerTool_Delete 测试删除告警
func TestAlertManagerTool_Delete(t *testing.T) {
	tool := NewAlertManagerTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"action": "delete",
		"name":   "test-alert",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if !result.Success {
		t.Errorf("删除告警应该成功: %s", result.Error)
	}
}

// TestAlertManagerTool_UnknownAction 测试未知操作
func TestAlertManagerTool_UnknownAction(t *testing.T) {
	tool := NewAlertManagerTool()
	ctx := &tool.Context{}

	params := map[string]interface{}{
		"action": "unknown",
	}

	result, err := tool.Execute(ctx, params)
	if err != nil {
		t.Fatalf("Execute 不应返回错误: %v", err)
	}

	if result.Success {
		t.Error("未知操作应该返回失败")
	}
}
