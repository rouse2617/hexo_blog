package builtin

import (
	"strings"
	"testing"

	"ai-ops/internal/tool"
)

func TestQueryLogTool(t *testing.T) {
	qt := NewQueryLogTool()

	// 测试基本信息
	if qt.Name() != "query_log" {
		t.Errorf("Name 期望 query_log, 实际 %s", qt.Name())
	}
	if qt.Description() == "" {
		t.Error("Description 不应为空")
	}
	if len(qt.Parameters()) == 0 {
		t.Error("Parameters 不应为空")
	}

	// 测试参数验证 - 缺少 host
	result, _ := qt.Execute(nil, map[string]interface{}{
		"log_type": "nginx",
	})
	if result.Success {
		t.Error("缺少 host 应该失败")
	}

	// 测试参数验证 - 缺少 log_type
	result, _ = qt.Execute(nil, map[string]interface{}{
		"host": "test-host",
	})
	if result.Success {
		t.Error("缺少 log_type 应该失败")
	}

	// 测试 SSH 上下文未初始化
	result, _ = qt.Execute(nil, map[string]interface{}{
		"host":     "test-host",
		"log_type": "nginx",
	})
	if result.Success {
		t.Error("SSH 未初始化应该失败")
	}
}

func TestQueryLogToolGetLogPath(t *testing.T) {
	qt := NewQueryLogTool()

	tests := []struct {
		logType    string
		customPath string
		expected   string
	}{
		{"nginx", "", "/var/log/nginx/error.log"},
		{"app", "", "/var/log/app/app.log"},
		{"system", "", "/var/log/syslog"},
		{"custom", "/var/log/custom.log", "/var/log/custom.log"},
		{"unknown", "", ""},
	}

	for _, tt := range tests {
		result := qt.getLogPath(tt.logType, tt.customPath)
		if result != tt.expected {
			t.Errorf("getLogPath(%s, %s) = %s, 期望 %s", tt.logType, tt.customPath, result, tt.expected)
		}
	}
}

func TestQueryLogToolBuildCommand(t *testing.T) {
	qt := NewQueryLogTool()

	// 无过滤
	cmd := qt.buildCommand("/var/log/test.log", 100, "", "")
	if !strings.Contains(cmd, "tail -100") {
		t.Errorf("无过滤命令应包含 tail, 实际: %s", cmd)
	}

	// 有关键词过滤
	cmd = qt.buildCommand("/var/log/test.log", 50, "error", "")
	if !strings.Contains(cmd, "grep") {
		t.Errorf("有关键词应包含 grep, 实际: %s", cmd)
	}
	if !strings.Contains(cmd, "error") {
		t.Errorf("应包含关键词 error, 实际: %s", cmd)
	}

	// 有级别过滤
	cmd = qt.buildCommand("/var/log/test.log", 50, "", "error")
	if !strings.Contains(cmd, "grep") {
		t.Errorf("有级别过滤应包含 grep, 实际: %s", cmd)
	}
}

func TestCheckCPUTool(t *testing.T) {
	ct := NewCheckCPUTool()

	if ct.Name() != "check_cpu" {
		t.Errorf("Name 期望 check_cpu, 实际 %s", ct.Name())
	}

	// 测试缺少参数
	result, _ := ct.Execute(&tool.Context{}, map[string]interface{}{})
	if result.Success {
		t.Error("缺少 host/hosts 应该失败")
	}
}

func TestCheckMemoryTool(t *testing.T) {
	mt := NewCheckMemoryTool()

	if mt.Name() != "check_memory" {
		t.Errorf("Name 期望 check_memory, 实际 %s", mt.Name())
	}

	// 测试缺少参数
	result, _ := mt.Execute(&tool.Context{}, map[string]interface{}{})
	if result.Success {
		t.Error("缺少 host/hosts 应该失败")
	}
}

func TestCheckDiskTool(t *testing.T) {
	dt := NewCheckDiskTool()

	if dt.Name() != "check_disk" {
		t.Errorf("Name 期望 check_disk, 实际 %s", dt.Name())
	}

	// 测试缺少参数
	result, _ := dt.Execute(&tool.Context{}, map[string]interface{}{})
	if result.Success {
		t.Error("缺少 host/hosts 应该失败")
	}
}

func TestCheckProcessTool(t *testing.T) {
	pt := NewCheckProcessTool()

	if pt.Name() != "check_process" {
		t.Errorf("Name 期望 check_process, 实际 %s", pt.Name())
	}

	// 测试缺少 host
	result, _ := pt.Execute(nil, map[string]interface{}{})
	if result.Success {
		t.Error("缺少 host 应该失败")
	}
}

func TestRunCommandTool(t *testing.T) {
	rt := NewRunCommandTool()

	if rt.Name() != "run_command" {
		t.Errorf("Name 期望 run_command, 实际 %s", rt.Name())
	}

	// 测试缺少参数
	result, _ := rt.Execute(nil, map[string]interface{}{})
	if result.Success {
		t.Error("缺少参数应该失败")
	}

	result, _ = rt.Execute(nil, map[string]interface{}{
		"host": "test",
	})
	if result.Success {
		t.Error("缺少 command 应该失败")
	}
}

func TestRunCommandToolDangerousCommand(t *testing.T) {
	rt := NewRunCommandTool()

	dangerousCmds := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs.ext4 /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
		"shutdown -h now",
		"reboot",
		"init 0",
		":(){ :|:& };:",
	}

	for _, cmd := range dangerousCmds {
		err := rt.checkDangerousCommand(cmd)
		if err == nil {
			t.Errorf("危险命令 '%s' 应该被拦截", cmd)
		}
	}

	// 安全命令
	safeCmds := []string{
		"ls -la",
		"cat /var/log/syslog",
		"ps aux",
		"df -h",
		"free -m",
	}

	for _, cmd := range safeCmds {
		err := rt.checkDangerousCommand(cmd)
		if err != nil {
			t.Errorf("安全命令 '%s' 不应该被拦截: %v", cmd, err)
		}
	}
}

func TestListHostsTool(t *testing.T) {
	// 模拟主机列表
	mockHosts := []HostBasicInfo{
		{Name: "web-01", Host: "192.168.1.1", Group: "web"},
		{Name: "web-02", Host: "192.168.1.2", Group: "web"},
		{Name: "db-01", Host: "192.168.1.10", Group: "db"},
	}

	getHostsFunc := func(group string) []HostBasicInfo {
		if group == "" {
			return mockHosts
		}
		var filtered []HostBasicInfo
		for _, h := range mockHosts {
			if h.Group == group {
				filtered = append(filtered, h)
			}
		}
		return filtered
	}

	lt := NewListHostsTool(getHostsFunc)

	if lt.Name() != "list_hosts" {
		t.Errorf("Name 期望 list_hosts, 实际 %s", lt.Name())
	}

	// 测试获取所有主机
	result, _ := lt.Execute(nil, map[string]interface{}{})
	if !result.Success {
		t.Error("获取所有主机应该成功")
	}
	hosts := result.Data.([]HostBasicInfo)
	if len(hosts) != 3 {
		t.Errorf("期望 3 个主机, 实际 %d", len(hosts))
	}

	// 测试按分组过滤
	result, _ = lt.Execute(nil, map[string]interface{}{
		"group": "web",
	})
	if !result.Success {
		t.Error("按分组过滤应该成功")
	}
	hosts = result.Data.([]HostBasicInfo)
	if len(hosts) != 2 {
		t.Errorf("期望 2 个 web 主机, 实际 %d", len(hosts))
	}

	// 测试空分组
	result, _ = lt.Execute(nil, map[string]interface{}{
		"group": "nonexistent",
	})
	if !result.Success {
		t.Error("空结果也应该成功")
	}
}

func TestListHostsToolNoFunc(t *testing.T) {
	lt := NewListHostsTool(nil)

	result, _ := lt.Execute(nil, map[string]interface{}{})
	if result.Success {
		t.Error("未配置 GetHostsFunc 应该失败")
	}
}

func TestRegisterBasic(t *testing.T) {
	registry := tool.NewRegistry()

	err := RegisterBasic(registry)
	if err != nil {
		t.Fatalf("注册基础工具失败: %v", err)
	}

	// 验证工具已注册
	expectedTools := []string{
		"query_log",
		"check_cpu",
		"check_memory",
		"check_disk",
		"check_process",
		"run_command",
	}

	for _, name := range expectedTools {
		if !registry.Has(name) {
			t.Errorf("工具 %s 未注册", name)
		}
	}

	if registry.Count() != len(expectedTools) {
		t.Errorf("工具数量期望 %d, 实际 %d", len(expectedTools), registry.Count())
	}
}

func TestRegisterAll(t *testing.T) {
	registry := tool.NewRegistry()

	getHostsFunc := func(group string) []HostBasicInfo {
		return []HostBasicInfo{}
	}

	err := RegisterAll(registry, getHostsFunc)
	if err != nil {
		t.Fatalf("注册所有工具失败: %v", err)
	}

	// 验证 list_hosts 也已注册
	if !registry.Has("list_hosts") {
		t.Error("list_hosts 未注册")
	}

	if registry.Count() != 7 {
		t.Errorf("工具数量期望 7, 实际 %d", registry.Count())
	}
}
