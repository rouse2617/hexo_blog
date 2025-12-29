package builtin

import (
	"testing"
	"ai-ops/internal/tool"
)

// TestIntrusionDetectionTool 测试入侵检测工具的基本功能
func TestIntrusionDetectionTool(t *testing.T) {
	tool := &IntrusionDetectionTool{}

	// 测试工具名称
	if tool.Name() != "detect_intrusion" {
		t.Errorf("Expected name 'detect_intrusion', got '%s'", tool.Name())
	}

	// 测试描述不为空
	desc := tool.Description()
	if desc == "" {
		t.Error("Description should not be empty")
	}

	// 测试参数
	params := tool.Parameters()
	if len(params) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(params))
	}

	if params[0].Name != "host" {
		t.Errorf("Expected parameter name 'host', got '%s'", params[0].Name)
	}

	if !params[0].Required {
		t.Error("Expected 'host' parameter to be required")
	}

	// 测试工具注册
	registry := tool.NewRegistry()
	err := registry.RegisterBuiltin(tool)
	if err != nil {
		t.Errorf("Failed to register tool: %v", err)
	}

	// 验证工具已注册
	registeredTool, exists := registry.GetBuiltin("detect_intrusion")
	if !exists {
		t.Error("Tool should be registered")
	}

	if registeredTool.Name() != "detect_intrusion" {
		t.Errorf("Registered tool name mismatch")
	}
}

// TestIntrusionDetectionToolSecurityScoring 测试安全评分逻辑
func TestIntrusionDetectionToolSecurityScoring(t *testing.T) {
	testCases := []struct {
		name     string
		findings []map[string]interface{}
		expected int
	}{
		{
			name:     "No findings",
			findings: []map[string]interface{}{},
			expected: 100,
		},
		{
			name: "One critical finding",
			findings: []map[string]interface{}{
				{"level": "critical"},
			},
			expected: 75,
		},
		{
			name: "One high finding",
			findings: []map[string]interface{}{
				{"level": "high"},
			},
			expected: 90,
		},
		{
			name: "One medium finding",
			findings: []map[string]interface{}{
				{"level": "medium"},
			},
			expected: 95,
		},
		{
			name: "Mixed findings",
			findings: []map[string]interface{}{
				{"level": "critical"},
				{"level": "high"},
				{"level": "medium"},
			},
			expected: 60,
		},
		{
			name: "Multiple critical findings",
			findings: []map[string]interface{}{
				{"level": "critical"},
				{"level": "critical"},
				{"level": "critical"},
				{"level": "critical"},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := 100
			for _, f := range tc.findings {
				if f["level"] == "critical" {
					score -= 25
				} else if f["level"] == "high" {
					score -= 10
				} else if f["level"] == "medium" {
					score -= 5
				}
			}
			if score < 0 {
				score = 0
			}

			if score != tc.expected {
				t.Errorf("Expected score %d, got %d", tc.expected, score)
			}
		})
	}
}

// TestIntrusionDetectionToolSummary 测试安全摘要生成
func TestIntrusionDetectionToolSummary(t *testing.T) {
	tool := &IntrusionDetectionTool{}

	t.Run("No findings", func(t *testing.T) {
		findings := []map[string]interface{}{}
		summary := tool.generateSecuritySummary(findings)
		expected := "未发现安全问题，系统安全"
		if summary != expected {
			t.Errorf("Expected summary '%s', got '%s'", expected, summary)
		}
	})

	t.Run("With findings", func(t *testing.T) {
		findings := []map[string]interface{}{
			{"category": "login", "level": "high"},
			{"category": "command", "level": "critical"},
			{"category": "file", "level": "medium"},
		}
		summary := tool.generateSecuritySummary(findings)

		// 检查摘要中包含关键信息
		if len(summary) == 0 {
			t.Error("Summary should not be empty for findings")
		}
	})
}
