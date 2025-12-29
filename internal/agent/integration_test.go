package agent

import (
	"context"
	"testing"

	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
)

// TestAgentWithErrorRecovery 测试 Agent 集成错误恢复
func TestAgentWithErrorRecovery(t *testing.T) {
	// 创建 SSH 连接池
	sshPool := ssh.NewPool()

	// 创建工具注册表
	toolRegistry := tool.NewRegistry()
	toolRegistry.RegisterBuiltinTools()

	// 创建 mock LLM 客户端
	llmClient := &mockLLMClient{}

	// 创建 Agent（会自动初始化错误恢复引擎）
	cfg := Config{
		MaxLoops:       5,
		Timeout:        defaultTimeout,
		PromptVersion:  "enhanced",
		EnableThinking: true,
	}

	agent := NewAgent(llmClient, toolRegistry, sshPool, cfg)

	// 验证错误恢复引擎已初始化
	if agent.errorRecovery == nil {
		t.Fatal("Error recovery engine not initialized")
	}

	t.Log("✓ Agent with error recovery engine created successfully")
}

// TestErrorRecoveryIntegration 测试错误恢复集成
func TestErrorRecoveryIntegration(t *testing.T) {
	// 准备测试环境
	sshPool := ssh.NewPool()
	toolRegistry := tool.NewRegistry()
	toolRegistry.RegisterBuiltinTools()
	llmClient := &mockLLMClient{}

	cfg := Config{
		MaxLoops:      5,
		Timeout:       defaultTimeout,
		PromptVersion: "enhanced",
	}

	agent := NewAgent(llmClient, toolRegistry, sshPool, cfg)

	// 测试用例：工具调用失败场景
	testCases := []struct {
		name        string
		toolCall    llm.ToolCall
		expectRetry bool
	}{
		{
			name: "SSH connection failure - should have alternative",
			toolCall: llm.ToolCall{
				ID: "test-1",
				Function: llm.FunctionCall{
					Name:      "check_cpu",
					Arguments: `{"host": "192.168.1.1"}`,
				},
			},
			expectRetry: true,
		},
		{
			name: "Log file not found - should have alternative",
			toolCall: llm.ToolCall{
				ID: "test-2",
				Function: llm.FunctionCall{
					Name:      "query_log",
					Arguments: `{"host": "server1", "log_type": "app"}`,
				},
			},
			expectRetry: true, // 有替代工具
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 模拟工具调用失败
			record := ToolCallRecord{
				ID:     tc.toolCall.ID,
				Tool:   tc.toolCall.Function.Name,
				Params: map[string]interface{}{"host": "test-server"},
				Error:  "connection refused",
			}

			// 调用错误恢复
			shouldRetry, newCalls, recoveryPrompt, err := agent.errorRecovery.RecoverFromError(
				context.Background(),
				agent,
				record,
				nil,
			)

			if err != nil {
				t.Logf("Recovery error (expected): %v", err)
			}

			// 验证恢复提示
			if recoveryPrompt == "" {
				t.Error("Expected recovery prompt to be generated")
			} else {
				t.Logf("Recovery prompt: %s", recoveryPrompt)
			}

			// 验证是否找到策略
			if tc.expectRetry && !shouldRetry && len(newCalls) == 0 {
				t.Error("Expected retry or alternative tools")
			}

			if len(newCalls) > 0 {
				t.Logf("Generated %d alternative tool calls", len(newCalls))
				for i, call := range newCalls {
					t.Logf("  %d. %s: %s", i+1, call.Function.Name, call.Function.Arguments)
				}
			}
		})
	}
}

// mockLLMClient 用于测试的 mock LLM 客户端
type mockLLMClient struct{}

func (m *mockLLMClient) ChatWithTools(ctx context.Context, messages []llm.Message, tools []llm.ToolDef) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: "Mock response",
		},
	}, nil
}

func (m *mockLLMClient) ChatStreamWithTools(ctx context.Context, messages []llm.Message, tools []llm.ToolDef, callback func(chunk llm.StreamChunk)) error {
	callback(llm.StreamChunk{
		Type:    "content",
		Content: "Mock",
	})
	callback(llm.StreamChunk{Type: "done"})
	return nil
}

func (m *mockLLMClient) Chat(ctx context.Context, messages []llm.Message) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: "Mock response",
		},
	}, nil
}

// TestErrorRecoveryLogging 测试错误恢复日志
func TestErrorRecoveryLogging(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	log := RecoveryLog{
		Timestamp:   testTime(),
		ToolName:    "check_cpu",
		Error:       "connection refused",
		Strategy:    "alternative_tool",
		Success:     true,
		Alternative: "run_command",
	}

	// 这个测试只是确保不会 panic
	engine.LogRecovery(log)
	t.Log("✓ Recovery logging works")
}

func testTime() interface{} {
	// 简化的时间返回
	return interface{}(nil)
}

// BenchmarkErrorRecoveryInAgent 性能测试
func BenchmarkErrorRecoveryInAgent(b *testing.B) {
	sshPool := ssh.NewPool()
	toolRegistry := tool.NewRegistry()
	toolRegistry.RegisterBuiltinTools()
	llmClient := &mockLLMClient{}

	cfg := Config{
		MaxLoops:      5,
		Timeout:       defaultTimeout,
		PromptVersion: "enhanced",
	}

	agent := NewAgent(llmClient, toolRegistry, sshPool, cfg)

	record := ToolCallRecord{
		ID:     "bench-test",
		Tool:   "check_cpu",
		Params: map[string]interface{}{"host": "test"},
		Error:  "connection refused",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = agent.errorRecovery.RecoverFromError(
			context.Background(),
			agent,
			record,
			nil,
		)
	}
}
