package tool

import (
	"strings"
	"testing"
)

// MockTool 测试用 Mock 工具
type MockTool struct {
	name        string
	description string
	parameters  []Parameter
	executeFunc func(ctx *Context, params map[string]interface{}) (*Result, error)
}

func (t *MockTool) Name() string        { return t.name }
func (t *MockTool) Description() string { return t.description }
func (t *MockTool) Parameters() []Parameter {
	if t.parameters == nil {
		return []Parameter{}
	}
	return t.parameters
}
func (t *MockTool) Execute(ctx *Context, params map[string]interface{}) (*Result, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, params)
	}
	return NewResult("mock result", "执行成功"), nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Error("NewRegistry 返回 nil")
	}
	if registry.Count() != 0 {
		t.Errorf("新建注册中心应该为空, 实际 %d", registry.Count())
	}
}

func TestRegister(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "测试工具",
	}

	// 注册工具
	err := registry.RegisterBuiltin(tool)
	if err != nil {
		t.Fatalf("注册工具失败: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("注册后工具数期望 1, 实际 %d", registry.Count())
	}

	// 重复注册应该失败
	err = registry.RegisterBuiltin(tool)
	if err == nil {
		t.Error("重复注册应该返回错误")
	}
}

func TestGet(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "测试工具",
	}
	registry.RegisterBuiltin(tool)

	// 获取存在的工具
	got, ok := registry.Get("test_tool")
	if !ok {
		t.Error("获取工具失败")
	}
	if got.Name() != "test_tool" {
		t.Errorf("工具名期望 test_tool, 实际 %s", got.Name())
	}

	// 获取不存在的工具
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("获取不存在的工具应该返回 false")
	}
}

func TestHas(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{name: "test_tool"}
	registry.RegisterBuiltin(tool)

	if !registry.Has("test_tool") {
		t.Error("Has 应该返回 true")
	}
	if registry.Has("nonexistent") {
		t.Error("Has 应该返回 false")
	}
}

func TestUnregister(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{name: "test_tool"}
	registry.RegisterBuiltin(tool)

	registry.Unregister("test_tool")

	if registry.Has("test_tool") {
		t.Error("注销后工具应该不存在")
	}
	if registry.Count() != 0 {
		t.Errorf("注销后工具数期望 0, 实际 %d", registry.Count())
	}
}

func TestList(t *testing.T) {
	registry := NewRegistry()

	tools := []*MockTool{
		{name: "tool1"},
		{name: "tool2"},
		{name: "tool3"},
	}

	for _, tool := range tools {
		registry.RegisterBuiltin(tool)
	}

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("列表长度期望 3, 实际 %d", len(list))
	}
}

func TestListInfo(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "测试工具描述",
		parameters: []Parameter{
			{Name: "host", Type: "string", Required: true},
		},
	}
	registry.RegisterBuiltin(tool)

	infos := registry.ListInfo()
	if len(infos) != 1 {
		t.Fatalf("信息列表长度期望 1, 实际 %d", len(infos))
	}

	info := infos[0]
	if info.Name != "test_tool" {
		t.Errorf("Name 期望 test_tool, 实际 %s", info.Name)
	}
	if info.Type != "builtin" {
		t.Errorf("Type 期望 builtin, 实际 %s", info.Type)
	}
	if len(info.Parameters) != 1 {
		t.Errorf("Parameters 长度期望 1, 实际 %d", len(info.Parameters))
	}
}

func TestExecute(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name: "test_tool",
		executeFunc: func(ctx *Context, params map[string]interface{}) (*Result, error) {
			host := GetStringParam(params, "host", "")
			return NewResult(host, "执行成功"), nil
		},
	}
	registry.RegisterBuiltin(tool)

	// 执行存在的工具
	result, err := registry.Execute(nil, "test_tool", map[string]interface{}{
		"host": "192.168.1.1",
	})
	if err != nil {
		t.Fatalf("执行工具失败: %v", err)
	}
	if !result.Success {
		t.Error("执行结果应该成功")
	}
	if result.Data != "192.168.1.1" {
		t.Errorf("Data 期望 192.168.1.1, 实际 %v", result.Data)
	}

	// 执行不存在的工具
	_, err = registry.Execute(nil, "nonexistent", nil)
	if err == nil {
		t.Error("执行不存在的工具应该返回错误")
	}
}

func TestExecute_HostInjection_Single(t *testing.T) {
	registry := NewRegistry()

	tt := &MockTool{
		name: "test_tool",
		parameters: []Parameter{
			{Name: "host", Type: "string", Required: true},
		},
		executeFunc: func(ctx *Context, params map[string]interface{}) (*Result, error) {
			host := GetStringParam(params, "host", "")
			return NewResult(host, "ok"), nil
		},
	}
	registry.RegisterBuiltin(tt)

	ctx := &Context{Hosts: []string{"node3"}}
	res, err := registry.Execute(ctx, "test_tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("执行工具失败: %v", err)
	}
	if res.Data != "node3" {
		t.Fatalf("期望注入 host=node3, 实际 %v", res.Data)
	}
}

func TestExecute_HostInjection_Multi(t *testing.T) {
	registry := NewRegistry()

	tt := &MockTool{
		name: "test_tool",
		parameters: []Parameter{
			{Name: "hosts", Type: "[]string", Required: true},
		},
		executeFunc: func(ctx *Context, params map[string]interface{}) (*Result, error) {
			hosts := GetStringSliceParam(params, "hosts")
			return NewResult(hosts, "ok"), nil
		},
	}
	registry.RegisterBuiltin(tt)

	ctx := &Context{Hosts: []string{"node2", "node3"}}
	res, err := registry.Execute(ctx, "test_tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("执行工具失败: %v", err)
	}

	got, ok := res.Data.([]string)
	if !ok {
		t.Fatalf("期望返回 []string, 实际 %T", res.Data)
	}
	if len(got) != 2 || got[0] != "node2" || got[1] != "node3" {
		t.Fatalf("期望注入 hosts=[node2 node3], 实际 %v", got)
	}
}

func TestGeneratePrompt(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name:        "query_log",
		description: "查询日志文件",
		parameters: []Parameter{
			{Name: "host", Type: "string", Description: "目标主机", Required: true},
			{Name: "lines", Type: "int", Description: "行数", Required: false, Default: 100},
			{Name: "log_type", Type: "string", Description: "日志类型", Enum: []interface{}{"nginx", "app", "system"}},
		},
	}
	registry.RegisterBuiltin(tool)

	prompt := registry.GeneratePrompt()

	// 验证 prompt 包含关键信息
	if !strings.Contains(prompt, "query_log") {
		t.Error("Prompt 应该包含工具名")
	}
	if !strings.Contains(prompt, "查询日志文件") {
		t.Error("Prompt 应该包含描述")
	}
	if !strings.Contains(prompt, "host") {
		t.Error("Prompt 应该包含参数名")
	}
	if !strings.Contains(prompt, "必填") {
		t.Error("Prompt 应该标注必填参数")
	}
	if !strings.Contains(prompt, "默认: 100") {
		t.Error("Prompt 应该包含默认值")
	}
}

func TestGenerateJSONSchema(t *testing.T) {
	registry := NewRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "测试工具",
		parameters: []Parameter{
			{Name: "host", Type: "string", Description: "主机", Required: true},
			{Name: "count", Type: "int", Description: "数量", Required: false},
		},
	}
	registry.RegisterBuiltin(tool)

	schemas := registry.GenerateJSONSchema()
	if len(schemas) != 1 {
		t.Fatalf("Schema 数量期望 1, 实际 %d", len(schemas))
	}

	schema := schemas[0]
	if schema["type"] != "function" {
		t.Errorf("type 期望 function, 实际 %v", schema["type"])
	}

	fn := schema["function"].(map[string]interface{})
	if fn["name"] != "test_tool" {
		t.Errorf("name 期望 test_tool, 实际 %v", fn["name"])
	}
}

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"host": "192.168.1.1",
		"port": 22,
	}

	if GetStringParam(params, "host", "") != "192.168.1.1" {
		t.Error("GetStringParam 获取失败")
	}
	if GetStringParam(params, "nonexistent", "default") != "default" {
		t.Error("GetStringParam 默认值失败")
	}
}

func TestGetIntParam(t *testing.T) {
	params := map[string]interface{}{
		"count":  100,
		"float":  100.5,
		"int64":  int64(200),
		"string": "not int",
	}

	if GetIntParam(params, "count", 0) != 100 {
		t.Error("GetIntParam int 获取失败")
	}
	if GetIntParam(params, "float", 0) != 100 {
		t.Error("GetIntParam float64 获取失败")
	}
	if GetIntParam(params, "int64", 0) != 200 {
		t.Error("GetIntParam int64 获取失败")
	}
	if GetIntParam(params, "nonexistent", 50) != 50 {
		t.Error("GetIntParam 默认值失败")
	}
}

func TestGetBoolParam(t *testing.T) {
	params := map[string]interface{}{
		"enabled": true,
		"string":  "not bool",
	}

	if !GetBoolParam(params, "enabled", false) {
		t.Error("GetBoolParam 获取失败")
	}
	if GetBoolParam(params, "nonexistent", true) != true {
		t.Error("GetBoolParam 默认值失败")
	}
}

func TestGetStringSliceParam(t *testing.T) {
	params := map[string]interface{}{
		"hosts":      []string{"host1", "host2"},
		"interfaces": []interface{}{"eth0", "eth1"},
	}

	hosts := GetStringSliceParam(params, "hosts")
	if len(hosts) != 2 || hosts[0] != "host1" {
		t.Error("GetStringSliceParam []string 获取失败")
	}

	interfaces := GetStringSliceParam(params, "interfaces")
	if len(interfaces) != 2 || interfaces[0] != "eth0" {
		t.Error("GetStringSliceParam []interface{} 获取失败")
	}

	empty := GetStringSliceParam(params, "nonexistent")
	if empty != nil {
		t.Error("GetStringSliceParam 不存在应返回 nil")
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult("data", "成功")
	if !result.Success {
		t.Error("NewResult Success 应该为 true")
	}
	if result.Data != "data" {
		t.Error("NewResult Data 错误")
	}
	if result.Message != "成功" {
		t.Error("NewResult Message 错误")
	}
}

func TestNewErrorResult(t *testing.T) {
	result := NewErrorResult("出错了")
	if result.Success {
		t.Error("NewErrorResult Success 应该为 false")
	}
	if result.Error != "出错了" {
		t.Error("NewErrorResult Error 错误")
	}
}
