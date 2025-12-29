package builtin

import (
	"testing"

	"ai-ops/internal/tool"
)

// TestTrendAnalysisTool 测试趋势分析工具
func TestTrendAnalysisTool(t *testing.T) {
	tool := NewTrendAnalysisTool()

	// 测试工具名称
	if tool.Name() != "analyze_trend" {
		t.Errorf("Expected tool name 'analyze_trend', got '%s'", tool.Name())
	}

	// 测试工具描述
	desc := tool.Description()
	if desc == "" {
		t.Error("Tool description should not be empty")
	}

	// 验证描述包含关键词
	keywords := []string{
		"磁盘增长趋势",
		"内存使用趋势",
		"日志增长趋势",
		"网络流量趋势",
		"容量预测",
	}
	for _, keyword := range keywords {
		if !contains(desc, keyword) {
			t.Errorf("Description should contain keyword '%s'", keyword)
		}
	}
}

// TestTrendAnalysisToolParameters 测试参数定义
func TestTrendAnalysisToolParameters(t *testing.T) {
	tool := NewTrendAnalysisTool()
	params := tool.Parameters()

	if len(params) == 0 {
		t.Error("Tool should have parameters")
	}

	// 验证必需参数
	paramMap := make(map[string]tool.Parameter)
	for _, p := range params {
		paramMap[p.Name] = p
	}

	// 检查 host 参数
	if p, ok := paramMap["host"]; !ok {
		t.Error("Tool should have 'host' parameter")
	} else {
		if p.Type != "string" {
			t.Errorf("Expected 'host' parameter type 'string', got '%s'", p.Type)
		}
		if p.Required {
			t.Error("'host' parameter should not be required")
		}
	}

	// 检查 metric 参数
	if p, ok := paramMap["metric"]; !ok {
		t.Error("Tool should have 'metric' parameter")
	} else {
		if p.Type != "string" {
			t.Errorf("Expected 'metric' parameter type 'string', got '%s'", p.Type)
		}
		if p.Default != "disk" {
			t.Errorf("Expected default value 'disk', got '%v'", p.Default)
		}
	}

	// 检查 days 参数
	if p, ok := paramMap["days"]; !ok {
		t.Error("Tool should have 'days' parameter")
	} else {
		if p.Type != "integer" {
			t.Errorf("Expected 'days' parameter type 'integer', got '%s'", p.Type)
		}
		if p.Default != 7 {
			t.Errorf("Expected default value 7, got '%v'", p.Default)
		}
	}
}

// TestTrendAnalysisToolWithoutContext 测试无上下文的执行
func TestTrendAnalysisToolWithoutContext(t *testing.T) {
	tool := NewTrendAnalysisTool()

	// 无上下文应该返回错误
	result, err := tool.Execute(nil, map[string]interface{}{
		"host": "test-host",
	})
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}
	if result.Success {
		t.Error("Result should not be successful without context")
	}
	if result.Error != "SSH 上下文未初始化" {
		t.Errorf("Expected error message 'SSH 上下文未初始化', got '%s'", result.Error)
	}
}

// TestTrendAnalysisToolWithoutHost 测试无主机参数的执行
func TestTrendAnalysisToolWithoutHost(t *testing.T) {
	tool := NewTrendAnalysisTool()

	// 创建一个空的上下文（仅用于测试参数验证）
	ctx := &tool.Context{}

	result, err := tool.Execute(ctx, map[string]interface{}{})
	if err != nil {
		t.Errorf("Execute should not return error, got: %v", err)
	}
	if result.Success {
		t.Error("Result should not be successful without host parameter")
	}
	if result.Error != "请指定 host 或 hosts 参数" {
		t.Errorf("Expected error message '请指定 host 或 hosts 参数', got '%s'", result.Error)
	}
}

// TestParseSize 测试大小解析
func TestParseSize(t *testing.T) {
	tool := NewTrendAnalysisTool()

	tests := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1K", 1024},
		{"1KB", 1024},
		{"1M", 1024 * 1024},
		{"1MB", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"1.5G", int64(1.5 * 1024 * 1024 * 1024)},
		{"512M", 512 * 1024 * 1024},
	}

	for _, tt := range tests {
		result := tool.parseSize(tt.input)
		if result != tt.expected {
			t.Errorf("parseSize(%q) = %d; want %d", tt.input, result, tt.expected)
		}
	}
}

// TestGetMemoryTrend 测试内存趋势判断
func TestGetMemoryTrend(t *testing.T) {
	tool := NewTrendAnalysisTool()

	tests := []struct {
		percent  int
		expected string
	}{
		{40, "稳定"},
		{60, "上升"},
		{75, "快速增长"},
		{90, "危险"},
	}

	for _, tt := range tests {
		result := tool.getMemoryTrend(tt.percent)
		if result != tt.expected {
			t.Errorf("getMemoryTrend(%d) = %s; want %s", tt.percent, result, tt.expected)
		}
	}
}

// TestPredictDiskFull 测试磁盘满预测
func TestPredictDiskFull(t *testing.T) {
	tool := NewTrendAnalysisTool()

	// 模拟 df 输出
	output := `Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        50G   25G   25G  50% /
/dev/sda2       100G   85G   15G  85% /var
/dev/sda3       200G  190G   10G  95% /data
/dev/sda4        30G   29G    1G  98% /home`

	result := tool.predictDiskFull(output)

	predictions, ok := result["predictions"].([]map[string]interface{})
	if !ok {
		t.Fatal("predictions should be a slice of maps")
	}

	// 验证预测数量
	if len(predictions) != 4 {
		t.Errorf("Expected 4 predictions, got %d", len(predictions))
	}

	// 验证预测内容
	for _, pred := range predictions {
		usage := pred["current_usage"].(int)
		risk := pred["risk_level"].(string)

		switch {
		case usage < 50:
			if risk != "低" {
				t.Errorf("Usage %d%% should have risk level '低', got '%s'", usage, risk)
			}
		case usage < 70:
			if risk != "中" {
				t.Errorf("Usage %d%% should have risk level '中', got '%s'", usage, risk)
			}
		case usage < 85:
			if risk != "高" {
				t.Errorf("Usage %d%% should have risk level '高', got '%s'", usage, risk)
			}
		case usage < 95:
			if risk != "严重" {
				t.Errorf("Usage %d%% should have risk level '严重', got '%s'", usage, risk)
			}
		default:
			if risk != "紧急" {
				t.Errorf("Usage %d%% should have risk level '紧急', got '%s'", usage, risk)
			}
		}
	}
}

// TestAssessOOMRisk 测试 OOM 风险评估
func TestAssessOOMRisk(t *testing.T) {
	tool := NewTrendAnalysisTool()

	tests := []struct {
		memPercent  int
		swapPercent int
		expected    string
	}{
		{60, 5, "低风险"},
		{85, 5, "中风险 - 内存压力高"},
		{85, 20, "中风险 - 需要关注"},
		{90, 40, "高风险 - OOM 可能性大"},
		{95, 60, "极高风险 - 即将发生 OOM"},
	}

	for _, tt := range tests {
		result := tool.assessOOMRisk(tt.memPercent, tt.swapPercent)
		if result != tt.expected {
			t.Errorf("assessOOMRisk(%d, %d) = %s; want %s",
				tt.memPercent, tt.swapPercent, result, tt.expected)
		}
	}
}

// TestFormatBytes 测试字节格式化
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s; want %s", tt.bytes, result, tt.expected)
		}
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
