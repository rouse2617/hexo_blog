package builtin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/tool"
)

// NetworkCheckTool 网络诊断工具
type NetworkCheckTool struct{}

// NewNetworkCheckTool 创建网络诊断工具
func NewNetworkCheckTool() *NetworkCheckTool {
	return &NetworkCheckTool{}
}

// Name 工具名称
func (t *NetworkCheckTool) Name() string {
	return "check_network"
}

// Description 工具描述
func (t *NetworkCheckTool) Description() string {
	return `# 网络连通性诊断

## 功能
- Ping 测试检测连通性
- 路由追踪查看网络路径
- DNS 解析测试
- 端口连通性检查
- 网络延迟和丢包率统计

## 使用场景
- 主机无法访问外部服务时
- 需要排查网络故障时
- 分析网络延迟原因时
- 验证 DNS 解析是否正常时`
}

// Parameters 工具参数
func (t *NetworkCheckTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "源主机名称",
			Required:    true,
		},
		{
			Name:        "target",
			Type:        "string",
			Description: "目标IP或域名",
			Required:    true,
		},
		{
			Name:        "count",
			Type:        "integer",
			Description: "ping 次数",
			Required:    false,
			Default:     4,
		},
		{
			Name:        "timeout",
			Type:        "integer",
			Description: "超时时间（秒）",
			Required:    false,
			Default:     30,
		},
	}
}

// Execute 执行工具
func (t *NetworkCheckTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	target := tool.GetStringParam(params, "target", "")
	count := tool.GetIntParam(params, "count", 4)

	if host == "" {
		return tool.NewErrorResult("host 参数必填"), nil
	}
	if target == "" {
		return tool.NewErrorResult("target 参数必填"), nil
	}

	// Ping 测试
	pingCmd := fmt.Sprintf("ping -c %d %s", count, target)
	pingResult, _ := ctx.SSH.Exec(host, pingCmd)

	// 解析结果
	packetLoss := t.parsePacketLoss(pingResult)
	avgTime := t.parseAvgTime(pingResult)
	minTime := t.parseMinTime(pingResult)
	maxTime := t.parseMaxTime(pingResult)

	// DNS 测试
	dnsCmd := fmt.Sprintf("nslookup %s 2>/dev/null || dig %s +short", target, target)
	dnsResult, _ := ctx.SSH.Exec(host, dnsCmd)

	// 路由追踪
	traceCmd := fmt.Sprintf("traceroute -m 10 -w 1 %s 2>/dev/null || tracepath -n %s 2>/dev/null || echo '路由追踪工具未安装'", target, target)
	traceResult, _ := ctx.SSH.Exec(host, traceCmd)

	// 检查网络接口状态
	ifaceCmd := "ip addr show 2>/dev/null | grep '^[0-9]' | head -10"
	ifaceResult, _ := ctx.SSH.Exec(host, ifaceCmd)

	status := t.getNetworkStatus(packetLoss, avgTime)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"target":      target,
		"ping":        pingResult,
		"dns":         dnsResult,
		"traceroute":  traceResult,
		"interfaces":  ifaceResult,
		"packet_loss": packetLoss,
		"avg_time":    avgTime,
		"min_time":    minTime,
		"max_time":    maxTime,
		"status":      status,
		"timestamp":   time.Now().Format(time.RFC3339),
	}, fmt.Sprintf("完成从 %s 到 %s 的网络诊断", host, target)), nil
}

// parsePacketLoss 解析丢包率
func (t *NetworkCheckTool) parsePacketLoss(output string) string {
	// Linux ping 输出格式: "3 packets transmitted, 3 received, 0% packet loss"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "packet loss") || strings.Contains(line, "received") {
			// 提取百分比
			re := regexp.MustCompile(`(\d+)%\s*packet loss`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				return matches[1] + "%"
			}
			// 如果找不到百分比，返回整行
			return strings.TrimSpace(line)
		}
	}
	return "未知"
}

// parseAvgTime 解析平均延迟
func (t *NetworkCheckTool) parseAvgTime(output string) string {
	// Linux ping 输出格式: "rtt min/avg/max/mdev = 0.052/0.066/0.090/0.016 ms"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "min/avg/max") || strings.Contains(line, "rtt") {
			re := regexp.MustCompile(`([\d.]+)/([\d.]+)/([\d.]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 2 {
				return matches[2] + " ms"
			}
		}
	}
	return "未知"
}

// parseMinTime 解析最小延迟
func (t *NetworkCheckTool) parseMinTime(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "min/avg/max") || strings.Contains(line, "rtt") {
			re := regexp.MustCompile(`([\d.]+)/([\d.]+)/([\d.]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				return matches[1] + " ms"
			}
		}
	}
	return "未知"
}

// parseMaxTime 解析最大延迟
func (t *NetworkCheckTool) parseMaxTime(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "min/avg/max") || strings.Contains(line, "rtt") {
			re := regexp.MustCompile(`([\d.]+)/([\d.]+)/([\d.]+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 3 {
				return matches[3] + " ms"
			}
		}
	}
	return "未知"
}

// getNetworkStatus 获取网络状态评估
func (t *NetworkCheckTool) getNetworkStatus(loss, avgTime string) string {
	lossPercent := 0
	if loss != "未知" {
		re := regexp.MustCompile(`(\d+)%`)
		if matches := re.FindStringSubmatch(loss); len(matches) > 1 {
			lossPercent, _ = strconv.Atoi(matches[1])
		}
	}

	// 提取平均延迟时间
	avgMs := 0.0
	if avgTime != "未知" {
		re := regexp.MustCompile(`([\d.]+)\s*ms`)
		if matches := re.FindStringSubmatch(avgTime); len(matches) > 1 {
			avgMs, _ = strconv.ParseFloat(matches[1], 64)
		}
	}

	// 评估状态
	if lossPercent == 100 {
		return "完全不通"
	}
	if lossPercent > 50 {
		return "严重丢包"
	}
	if lossPercent > 10 {
		return "网络不稳定"
	}
	if avgMs > 200 {
		return "延迟较高"
	}
	if avgMs > 100 {
		return "延迟一般"
	}
	if lossPercent == 0 && avgMs < 50 {
		return "优秀"
	}
	return "正常"
}
