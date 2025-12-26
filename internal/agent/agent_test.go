package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
)

// MockTool 测试用 Mock 工具
type MockTool struct {
	name        string
	description string
	params      []tool.Parameter
	result      *tool.Result
}

func (t *MockTool) Name() string                  { return t.name }
func (t *MockTool) Description() string           { return t.description }
func (t *MockTool) Parameters() []tool.Parameter  { return t.params }
func (t *MockTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	if t.result != nil {
		return t.result, nil
	}
	return tool.NewResult("mock result", "执行成功"), nil
}

func TestBuildSystemPrompt(t *testing.T) {
	registry := tool.NewRegistry()
	registry.RegisterBuiltin(&MockTool{
		name:        "test_tool",
		description: "测试工具",
		params: []tool.Parameter{
			{Name: "host", Type: "string", Required: true},
		},
	})

	prompt := BuildSystemPrompt(registry)

	// 验证 prompt 包含关键内容
	if !strings.Contains(prompt, "AI-Ops") {
		t.Error("Prompt 应该包含 AI-Ops")
	}
	if !strings.Contains(prompt, "test_tool") {
		t.Error("Prompt 应该包含工具名")
	}
	if !strings.Contains(prompt, "可用工具") {
		t.Error("Prompt 应该包含工具列表标题")
	}
}

func TestBuildSystemPromptWithHosts(t *testing.T) {
	registry := tool.NewRegistry()

	// 无主机
	prompt1 := BuildSystemPromptWithHosts(registry, nil)
	if strings.Contains(prompt1, "当前会话关联的主机") {
		t.Error("无主机时不应该包含主机信息")
	}

	// 有主机
	prompt2 := BuildSystemPromptWithHosts(registry, []string{"node-1", "node-2"})
	if !strings.Contains(prompt2, "当前会话关联的主机") {
		t.Error("有主机时应该包含主机信息")
	}
	if !strings.Contains(prompt2, "node-1") {
		t.Error("应该包含主机名 node-1")
	}
}

func TestBuildTaskPrompt(t *testing.T) {
	prompt := BuildTaskPrompt("检查所有节点的磁盘使用率")

	if !strings.Contains(prompt, "检查所有节点的磁盘使用率") {
		t.Error("任务提示词应该包含任务内容")
	}
	if !strings.Contains(prompt, "当前任务") {
		t.Error("任务提示词应该包含标题")
	}
}

func TestBuildErrorRecoveryPrompt(t *testing.T) {
	prompt := BuildErrorRecoveryPrompt("连接超时")

	if !strings.Contains(prompt, "连接超时") {
		t.Error("错误恢复提示词应该包含错误信息")
	}
}

func TestPromptBuilder(t *testing.T) {
	registry := tool.NewRegistry()
	builder := NewPromptBuilder(registry)

	// 测试链式调用
	builder.SetHosts([]string{"host1", "host2"})

	system := builder.BuildSystem()
	if !strings.Contains(system, "host1") {
		t.Error("系统提示词应该包含主机")
	}

	task := builder.BuildTask("test task")
	if !strings.Contains(task, "test task") {
		t.Error("任务提示词应该包含任务")
	}

	errPrompt := builder.BuildErrorRecovery("test error")
	if !strings.Contains(errPrompt, "test error") {
		t.Error("错误提示词应该包含错误")
	}

	summary := builder.BuildSummary()
	if summary == "" {
		t.Error("总结提示词不应为空")
	}
}

func TestNewAgent(t *testing.T) {
	// 测试默认配置
	agent := NewAgent(nil, nil, nil, Config{})

	if agent.maxLoops != 10 {
		t.Errorf("默认 maxLoops 期望 10, 实际 %d", agent.maxLoops)
	}
	if agent.timeout != 5*time.Minute {
		t.Errorf("默认 timeout 期望 5m, 实际 %v", agent.timeout)
	}

	// 测试自定义配置
	agent2 := NewAgent(nil, nil, nil, Config{
		MaxLoops: 5,
		Timeout:  2 * time.Minute,
	})

	if agent2.maxLoops != 5 {
		t.Errorf("自定义 maxLoops 期望 5, 实际 %d", agent2.maxLoops)
	}
}

func TestAgentChat(t *testing.T) {
	// 创建 mock LLM 服务器
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// 第一次调用返回工具调用
		if callCount == 1 {
			resp := map[string]interface{}{
				"id":    "test-1",
				"model": "test",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "mock_tool",
										"arguments": `{"param": "value"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		// 第二次调用返回最终回复
		resp := map[string]interface{}{
			"id":    "test-2",
			"model": "test",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "任务完成！",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 创建 LLM 客户端
	llmClient := llm.NewOpenAIClient(llm.OpenAIConfig{
		Endpoint: server.URL,
		Model:    "test",
	})

	// 创建工具注册中心
	registry := tool.NewRegistry()
	registry.RegisterBuiltin(&MockTool{
		name:        "mock_tool",
		description: "Mock 工具",
		params: []tool.Parameter{
			{Name: "param", Type: "string"},
		},
		result: tool.NewResult("mock data", "成功"),
	})

	// 创建 SSH 池
	sshPool := ssh.NewPool(ssh.Config{})

	// 创建 Agent
	agent := NewAgent(llmClient, registry, sshPool, Config{
		MaxLoops: 5,
		Timeout:  30 * time.Second,
	})

	// 执行对话
	resp, err := agent.Chat(context.Background(), ChatRequest{
		SessionID: "test-session",
		Message:   "执行测试",
	})

	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if resp.Reply != "任务完成！" {
		t.Errorf("回复内容错误: %s", resp.Reply)
	}

	if len(resp.ToolCalls) != 1 {
		t.Errorf("期望 1 个工具调用, 实际 %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].Tool != "mock_tool" {
		t.Errorf("工具名错误: %s", resp.ToolCalls[0].Tool)
	}
}

func TestAgentChatNoToolCalls(t *testing.T) {
	// 创建直接返回回复的 mock 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":    "test-1",
			"model": "test",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "你好！有什么可以帮助你的？",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llmClient := llm.NewOpenAIClient(llm.OpenAIConfig{
		Endpoint: server.URL,
	})

	registry := tool.NewRegistry()
	agent := NewAgent(llmClient, registry, nil, Config{})

	resp, err := agent.Chat(context.Background(), ChatRequest{
		Message: "你好",
	})

	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if resp.Reply != "你好！有什么可以帮助你的？" {
		t.Errorf("回复内容错误: %s", resp.Reply)
	}

	if len(resp.ToolCalls) != 0 {
		t.Errorf("不应该有工具调用, 实际 %d", len(resp.ToolCalls))
	}
}

func TestToolCallRecord(t *testing.T) {
	record := ToolCallRecord{
		Tool:   "test_tool",
		Params: map[string]interface{}{"key": "value"},
		Result: "success",
	}

	if record.Tool != "test_tool" {
		t.Error("Tool 错误")
	}
	if record.Params["key"] != "value" {
		t.Error("Params 错误")
	}
	if record.Result != "success" {
		t.Error("Result 错误")
	}
}

func TestStreamChunk(t *testing.T) {
	// 测试内容块
	chunk1 := StreamChunk{
		Type:    "content",
		Content: "Hello",
	}
	if chunk1.Type != "content" || chunk1.Content != "Hello" {
		t.Error("内容块错误")
	}

	// 测试工具调用块
	tc := &llm.ToolCall{
		ID:   "1",
		Type: "function",
	}
	chunk2 := StreamChunk{
		Type:     "tool_call",
		ToolCall: tc,
	}
	if chunk2.Type != "tool_call" || chunk2.ToolCall == nil {
		t.Error("工具调用块错误")
	}

	// 测试完成块
	chunk3 := StreamChunk{Type: "done"}
	if chunk3.Type != "done" {
		t.Error("完成块错误")
	}
}
