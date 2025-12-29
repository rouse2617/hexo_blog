package builtin

import (
	"testing"

	"ai-ops/internal/tool"
)

// TestPerformanceAnalysisTool 测试性能分析工具
func TestPerformanceAnalysisTool(t *testing.T) {
	tool := NewPerformanceAnalysisTool()

	// 测试工具基本信息
	if tool.Name() != "analyze_performance" {
		t.Errorf("Expected name 'analyze_performance', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	// 测试参数定义
	params := tool.Parameters()
	if len(params) < 3 {
		t.Errorf("Expected at least 3 parameters, got %d", len(params))
	}

	// 验证必填参数
	foundHost := false
	for _, p := range params {
		if p.Name == "host" {
			foundHost = true
			if !p.Required {
				t.Error("host parameter should be required")
			}
			break
		}
	}
	if !foundHost {
		t.Error("host parameter not found")
	}
}

// TestNetworkCheckTool 测试网络诊断工具
func TestNetworkCheckTool(t *testing.T) {
	tool := NewNetworkCheckTool()

	if tool.Name() != "check_network" {
		t.Errorf("Expected name 'check_network', got '%s'", tool.Name())
	}

	params := tool.Parameters()
	if len(params) < 2 {
		t.Errorf("Expected at least 2 parameters, got %d", len(params))
	}
}

// TestPortCheckTool 测试端口检查工具
func TestPortCheckTool(t *testing.T) {
	tool := NewPortCheckTool()

	if tool.Name() != "check_port" {
		t.Errorf("Expected name 'check_port', got '%s'", tool.Name())
	}

	// 测试端口状态判断
	pct := &PortCheckTool{}

	tests := []struct {
		name     string
		listening bool
		connCount int
		expected string
	}{
		{"未监听", false, 0, "未监听"},
		{"正常", true, 50, "正常"},
		{"活跃", true, 150, "活跃"},
		{"连接数过多", true, 1500, "连接数过多"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pct.getPortStatus(tt.listening, tt.connCount)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestInodeCheckTool 测试 Inode 检查工具
func TestInodeCheckTool(t *testing.T) {
	tool := NewInodeCheckTool()

	if tool.Name() != "check_inode" {
		t.Errorf("Expected name 'check_inode', got '%s'", tool.Name())
	}

	// 测试 inode 信息解析
	ict := &InodeCheckTool{}

	output := `Filesystem      Inodes IUsed   IFree IUse% Mounted on
/dev/sda1      1310720 12345 1298375    1% /
/dev/sda2      524288 500000  24288   96% /var
/dev/sda3      262144 100000 162144   39% /home`

	info := ict.parseInodeInfo(output, 80)

	if critical, ok := info["critical"].(bool); !ok || !critical {
		t.Error("Expected critical to be true")
	}

	partitions, ok := info["partitions"].([]map[string]interface{})
	if !ok || len(partitions) != 3 {
		t.Errorf("Expected 3 partitions, got %d", len(partitions))
	}
}

// TestParameterHelpers 测试参数辅助函数
func TestParameterHelpers(t *testing.T) {
	params := map[string]interface{}{
		"string_param": "test",
		"int_param":    42,
		"bool_param":   true,
		"array_param":  []interface{}{"host1", "host2"},
	}

	// 测试 GetStringParam
	if s := tool.GetStringParam(params, "string_param", ""); s != "test" {
		t.Errorf("Expected 'test', got '%s'", s)
	}

	// 测试 GetIntParam
	if i := tool.GetIntParam(params, "int_param", 0); i != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	// 测试 GetBoolParam
	if b := tool.GetBoolParam(params, "bool_param", false); !b {
		t.Error("Expected true")
	}

	// 测试 GetArrayParam
	arr := tool.GetArrayParam(params, "array_param")
	if arr == nil || len(arr) != 2 {
		t.Error("Expected array with 2 elements")
	}
}

// BenchmarkPerformanceAnalysis 性能分析基准测试
func BenchmarkPerformanceAnalysis(b *testing.B) {
	pat := &PerformanceAnalysisTool{}

	cpuOutput := `%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.5 id,  0.2 wa,  0.0 hi,  0.0 si,  0.0 st`
	memOutput := `Mem:   16384000 total,  8192000 used,  8192000 free,  1024000 buffers`
	ioOutput := ``

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pat.calculateScore(cpuOutput, memOutput, ioOutput)
	}
}

// BenchmarkNetworkParsing 网络解析基准测试
func BenchmarkNetworkParsing(b *testing.B) {
	nct := &NetworkCheckTool{}

	pingOutput := `PING example.com (93.184.216.34): 56 data bytes
64 bytes from 93.184.216.34: icmp_seq=0 ttl=54 time=12.3 ms
64 bytes from 93.184.216.34: icmp_seq=1 ttl=54 time=12.5 ms
--- example.com ping statistics ---
2 packets transmitted, 2 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 12.3/12.4/12.5/0.1 ms`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nct.parsePacketLoss(pingOutput)
		nct.parseAvgTime(pingOutput)
		nct.parseMinTime(pingOutput)
		nct.parseMaxTime(pingOutput)
	}
}
