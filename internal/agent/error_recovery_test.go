package agent

import (
	"context"
	"testing"
)

// TestErrorRecoveryEngine 测试错误恢复引擎
func TestErrorRecoveryEngine(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	if engine == nil {
		t.Fatal("NewErrorRecoveryEngine returned nil")
	}

	if len(engine.strategies) == 0 {
		t.Error("No recovery strategies loaded")
	}

	t.Logf("Loaded %d recovery strategies", len(engine.strategies))
}

// TestFindStrategy 测试策略查找
func TestFindStrategy(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	tests := []struct {
		name           string
		toolName       string
		errorMsg       string
		shouldFind     bool
		expectedCanRetry bool
	}{
		{
			name:           "SSH connection refused",
			toolName:       "check_cpu",
			errorMsg:       "dial tcp 127.0.0.1:22: connect: connection refused",
			shouldFind:     true,
			expectedCanRetry: true,
		},
		{
			name:           "Log file not found",
			toolName:       "query_log",
			errorMsg:       "open /var/log/syslog: no such file or directory",
			shouldFind:     true,
			expectedCanRetry: false,
		},
		{
			name:           "Permission denied",
			toolName:       "check_disk",
			errorMsg:       "permission denied while trying to connect",
			shouldFind:     true,
			expectedCanRetry: false,
		},
		{
			name:           "Command timeout",
			toolName:       "check_memory",
			errorMsg:       "timeout after 30 seconds",
			shouldFind:     true,
			expectedCanRetry: true,
		},
		{
			name:           "Command not found",
			toolName:       "run_command",
			errorMsg:       "command not found: htop",
			shouldFind:     true,
			expectedCanRetry: false,
		},
		{
			name:           "Generic timeout with wildcard",
			toolName:       "unknown_tool",
			errorMsg:       "timeout after 30 seconds",
			shouldFind:     true,
			expectedCanRetry: true,
		},
		{
			name:           "Unknown error",
			toolName:       "check_cpu",
			errorMsg:       "something weird happened",
			shouldFind:     false,
			expectedCanRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := engine.findStrategy(tt.toolName, tt.errorMsg)

			if tt.shouldFind {
				if strategy == nil {
					t.Errorf("Expected to find strategy for %s: %s", tt.toolName, tt.errorMsg)
					return
				}
				if strategy.CanRetry != tt.expectedCanRetry {
					t.Errorf("Expected CanRetry=%v, got %v", tt.expectedCanRetry, strategy.CanRetry)
				}
				t.Logf("Found strategy: Tool=%s, ErrorCode=%s, CanRetry=%v, Alternatives=%v",
					strategy.ToolName, strategy.ErrorCode, strategy.CanRetry, strategy.AlternativeTools)
			} else {
				if strategy != nil {
					t.Errorf("Expected not to find strategy, but found: %s", strategy.ToolName)
				}
			}
		})
	}
}

// TestBuildRecoveryPrompt 测试恢复提示生成
func TestBuildRecoveryPrompt(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	strategy := &ErrorRecoveryStrategy{
		ToolName:       "check_cpu",
		ErrorCode:      "connection refused",
		RecoveryPrompt: "Check SSH status",
	}

	record := ToolCallRecord{
		Tool:  "check_cpu",
		Error: "connection refused",
	}

	prompt := engine.buildRecoveryPrompt(strategy, record)

	if prompt == "" {
		t.Error("Expected non-empty recovery prompt")
	}

	t.Logf("Generated prompt:\n%s", prompt)
}

// TestConvertParams 测试参数转换
func TestConvertParams(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	tests := []struct {
		name     string
		fromTool string
		toTool   string
		params   map[string]interface{}
		wantHost string
	}{
		{
			name:     "check_cpu to run_command",
			fromTool: "check_cpu",
			toTool:   "run_command",
			params:   map[string]interface{}{"host": "192.168.1.1"},
			wantHost: "192.168.1.1",
		},
		{
			name:     "check_memory to run_command",
			fromTool: "check_memory",
			toTool:   "run_command",
			params:   map[string]interface{}{"host": "test-server"},
			wantHost: "test-server",
		},
		{
			name:     "check_disk to run_command",
			fromTool: "check_disk",
			toTool:   "run_command",
			params:   map[string]interface{}{"host": "db01"},
			wantHost: "db01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.convertParams(tt.fromTool, tt.toTool, tt.params)

			// 简单检查包含 host
			if !contains(result, tt.wantHost) {
				t.Errorf("Expected result to contain host %s, got: %s", tt.wantHost, result)
			}

			t.Logf("Converted params: %s", result)
		})
	}
}

// TestBuildAlternativeToolCalls 测试替代工具调用构建
func TestBuildAlternativeToolCalls(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	strategy := &ErrorRecoveryStrategy{
		ToolName:         "check_cpu",
		AlternativeTools: []string{"run_command"},
	}

	record := ToolCallRecord{
		ID:     "test-123",
		Tool:   "check_cpu",
		Params: map[string]interface{}{"host": "test-server"},
	}

	calls := engine.buildAlternativeToolCalls(strategy, record)

	if len(calls) != 1 {
		t.Fatalf("Expected 1 alternative call, got %d", len(calls))
	}

	if calls[0].Function.Name != "run_command" {
		t.Errorf("Expected tool name 'run_command', got '%s'", calls[0].Function.Name)
	}

	if !contains(calls[0].ID, "alt_") {
		t.Errorf("Expected call ID to contain 'alt_', got '%s'", calls[0].ID)
	}

	t.Logf("Alternative call: %+v", calls[0])
}

// TestRecoverFromError 测试完整的错误恢复流程
func TestRecoverFromError(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	tests := []struct {
		name           string
		toolName       string
		errorMsg       string
		expectRetry    bool
		expectAltTools bool
	}{
		{
			name:           "Retryable error with alternatives",
			toolName:       "check_cpu",
			errorMsg:       "connection refused",
			expectRetry:    true,
			expectAltTools: true,
		},
		{
			name:           "Non-retryable error with alternatives",
			toolName:       "query_log",
			errorMsg:       "no such file or directory",
			expectRetry:    false,
			expectAltTools: false, // query_log 策略的 CanRetry=false，所以不会返回替代工具
		},
		{
			name:           "Non-retryable without alternatives",
			toolName:       "run_command",
			errorMsg:       "command not found",
			expectRetry:    false,
			expectAltTools: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := ToolCallRecord{
				ID:     "test-id",
				Tool:   tt.toolName,
				Error:  tt.errorMsg,
				Params: map[string]interface{}{"host": "test"},
			}

			shouldRetry, newCalls, recoveryPrompt, err := engine.RecoverFromError(
				context.Background(),
				nil, // agent can be nil for this test
				record,
				nil, // messages
			)

			if tt.expectRetry && !shouldRetry {
				t.Errorf("Expected shouldRetry=true")
			}

			if tt.expectAltTools && len(newCalls) == 0 {
				t.Errorf("Expected alternative tool calls")
			}

			if !tt.expectAltTools && len(newCalls) > 0 {
				t.Errorf("Expected no alternative tool calls, got %d", len(newCalls))
			}

			if recoveryPrompt == "" {
				t.Error("Expected recovery prompt to be generated")
			}

			t.Logf("Recovery result: retry=%v, altTools=%d, prompt=%s, err=%v",
				shouldRetry, len(newCalls), recoveryPrompt, err)
		})
	}
}

// TestLogRecovery 测试恢复日志记录
func TestLogRecovery(t *testing.T) {
	engine := NewErrorRecoveryEngine()

	log := RecoveryLog{
		ToolName:    "check_cpu",
		Error:       "connection refused",
		Strategy:    "alternative_tool",
		Success:     true,
		Alternative: "run_command",
	}

	// 这个测试只是确保不会 panic
	engine.LogRecovery(log)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkFindStrategy 性能测试
func BenchmarkFindStrategy(b *testing.B) {
	engine := NewErrorRecoveryEngine()
	errorMsg := "dial tcp 127.0.0.1:22: connect: connection refused"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.findStrategy("check_cpu", errorMsg)
	}
}
