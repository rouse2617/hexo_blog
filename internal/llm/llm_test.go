package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewOpenAIClient(t *testing.T) {
	// 测试默认配置
	client := NewOpenAIClient(OpenAIConfig{})

	if client.endpoint != "http://localhost:11434/v1" {
		t.Errorf("默认 endpoint 错误: %s", client.endpoint)
	}
	if client.model != "qwen2.5:14b" {
		t.Errorf("默认 model 错误: %s", client.model)
	}
	if client.timeout != 60*time.Second {
		t.Errorf("默认 timeout 错误: %v", client.timeout)
	}
	if client.maxTokens != 4096 {
		t.Errorf("默认 maxTokens 错误: %d", client.maxTokens)
	}

	// 测试自定义配置
	client2 := NewOpenAIClient(OpenAIConfig{
		Endpoint:  "http://custom:8080/v1/",
		Model:     "gpt-4",
		APIKey:    "test-key",
		Timeout:   30 * time.Second,
		MaxTokens: 2048,
	})

	if client2.endpoint != "http://custom:8080/v1" {
		t.Errorf("自定义 endpoint 错误: %s", client2.endpoint)
	}
	if client2.model != "gpt-4" {
		t.Errorf("自定义 model 错误: %s", client2.model)
	}
	if client2.apiKey != "test-key" {
		t.Errorf("自定义 apiKey 错误: %s", client2.apiKey)
	}
}

func TestChat(t *testing.T) {
	// 创建 mock 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求
		if r.Method != "POST" {
			t.Errorf("期望 POST 方法, 实际 %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("期望路径 /chat/completions, 实际 %s", r.URL.Path)
		}

		// 返回 mock 响应
		resp := openAIResponse{
			ID:    "test-id",
			Model: "test-model",
			Choices: []struct {
				Index        int     `json:"index"`
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Index:        0,
					Message:      Message{Role: RoleAssistant, Content: "Hello!"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIConfig{
		Endpoint: server.URL,
		Model:    "test-model",
	})

	messages := []Message{
		NewUserMessage("Hi"),
	}

	resp, err := client.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat 失败: %v", err)
	}

	if resp.Message.Content != "Hello!" {
		t.Errorf("响应内容错误: %s", resp.Message.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason 错误: %s", resp.FinishReason)
	}
}

func TestChatWithTools(t *testing.T) {
	// 创建 mock 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析请求
		var req openAIRequest
		json.NewDecoder(r.Body).Decode(&req)

		// 验证工具定义
		if len(req.Tools) == 0 {
			t.Error("请求中应该包含工具定义")
		}

		// 返回工具调用响应
		resp := openAIResponse{
			ID:    "test-id",
			Model: "test-model",
			Choices: []struct {
				Index        int     `json:"index"`
				Message      Message `json:"message"`
				FinishReason string  `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: Message{
						Role: RoleAssistant,
						ToolCalls: []ToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: FunctionCall{
									Name:      "check_cpu",
									Arguments: `{"host": "node-1"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIConfig{
		Endpoint: server.URL,
	})

	messages := []Message{
		NewUserMessage("检查 node-1 的 CPU"),
	}

	tools := []ToolDef{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "check_cpu",
				Description: "检查 CPU 使用率",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host": map[string]interface{}{
							"type":        "string",
							"description": "主机名",
						},
					},
					"required": []string{"host"},
				},
			},
		},
	}

	resp, err := client.ChatWithTools(context.Background(), messages, tools)
	if err != nil {
		t.Fatalf("ChatWithTools 失败: %v", err)
	}

	if !resp.HasToolCalls() {
		t.Error("响应应该包含工具调用")
	}

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("期望 1 个工具调用, 实际 %d", len(resp.Message.ToolCalls))
	}

	tc := resp.Message.ToolCalls[0]
	if tc.Function.Name != "check_cpu" {
		t.Errorf("工具名错误: %s", tc.Function.Name)
	}
}

func TestChatAPIError(t *testing.T) {
	// 创建返回错误的 mock 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewOpenAIClient(OpenAIConfig{
		Endpoint: server.URL,
	})

	_, err := client.Chat(context.Background(), []Message{NewUserMessage("Hi")})
	if err == nil {
		t.Error("期望返回错误")
	}
}

func TestMessageHelpers(t *testing.T) {
	// 测试消息创建辅助函数
	sysMsg := NewSystemMessage("You are a helpful assistant")
	if sysMsg.Role != RoleSystem || sysMsg.Content != "You are a helpful assistant" {
		t.Error("NewSystemMessage 错误")
	}

	userMsg := NewUserMessage("Hello")
	if userMsg.Role != RoleUser || userMsg.Content != "Hello" {
		t.Error("NewUserMessage 错误")
	}

	assistantMsg := NewAssistantMessage("Hi there!")
	if assistantMsg.Role != RoleAssistant || assistantMsg.Content != "Hi there!" {
		t.Error("NewAssistantMessage 错误")
	}

	toolCalls := []ToolCall{
		{ID: "1", Type: "function", Function: FunctionCall{Name: "test"}},
	}
	toolCallMsg := NewAssistantToolCallMessage(toolCalls)
	if toolCallMsg.Role != RoleAssistant || len(toolCallMsg.ToolCalls) != 1 {
		t.Error("NewAssistantToolCallMessage 错误")
	}

	toolMsg := NewToolMessage("call_1", "test_tool", "result")
	if toolMsg.Role != RoleTool || toolMsg.ToolCallID != "call_1" || toolMsg.Name != "test_tool" {
		t.Error("NewToolMessage 错误")
	}
}

func TestHasToolCalls(t *testing.T) {
	resp1 := &ChatResponse{
		Message: Message{
			Role:    RoleAssistant,
			Content: "Hello",
		},
	}
	if resp1.HasToolCalls() {
		t.Error("没有工具调用时应返回 false")
	}

	resp2 := &ChatResponse{
		Message: Message{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{ID: "1"},
			},
		},
	}
	if !resp2.HasToolCalls() {
		t.Error("有工具调用时应返回 true")
	}
}

func TestGetSetModel(t *testing.T) {
	client := NewOpenAIClient(OpenAIConfig{
		Model: "model-1",
	})

	if client.GetModel() != "model-1" {
		t.Errorf("GetModel 错误: %s", client.GetModel())
	}

	client.SetModel("model-2")
	if client.GetModel() != "model-2" {
		t.Errorf("SetModel 后 GetModel 错误: %s", client.GetModel())
	}
}
