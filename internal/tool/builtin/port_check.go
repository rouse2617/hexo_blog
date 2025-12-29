package builtin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/tool"
)

// PortCheckTool 端口检查工具
type PortCheckTool struct{}

// NewPortCheckTool 创建端口检查工具
func NewPortCheckTool() *PortCheckTool {
	return &PortCheckTool{}
}

// Name 工具名称
func (t *PortCheckTool) Name() string {
	return "check_port"
}

// Description 工具描述
func (t *PortCheckTool) Description() string {
	return `# 端口状态检查

## 功能
- 检查端口是否在监听
- 显示监听的服务和进程信息
- 检查端口连通性
- 显示连接状态统计
- 检查防火墙规则

## 使用场景
- 服务无法启动时检查端口占用
- 验证服务是否正常监听
- 排查端口连接问题
- 检查防火墙配置`
}

// Parameters 工具参数
func (t *PortCheckTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标主机",
			Required:    true,
		},
		{
			Name:        "port",
			Type:        "integer",
			Description: "端口号",
			Required:    true,
		},
		{
			Name:        "check_firewall",
			Type:        "boolean",
			Description: "是否检查防火墙规则",
			Required:    false,
			Default:     true,
		},
	}
}

// Execute 执行工具
func (t *PortCheckTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	port := tool.GetIntParam(params, "port", 0)
	checkFirewall := tool.GetBoolParam(params, "check_firewall", true)

	if host == "" {
		return tool.NewErrorResult("host 参数必填"), nil
	}

	if port <= 0 || port > 65535 {
		return tool.NewErrorResult("无效的端口号，必须在 1-65535 之间"), nil
	}

	// 检查端口监听
	listenCmd := fmt.Sprintf("ss -tunlp | grep ':%d ' || netstat -tunlp 2>/dev/null | grep ':%d '", port, port)
	listenResult, _ := ctx.SSH.Exec(host, listenCmd)

	// 检查端口连接
	connCmd := fmt.Sprintf("ss -tan | grep ':%d ' || netstat -tan 2>/dev/null | grep ':%d '", port, port)
	connResult, _ := ctx.SSH.Exec(host, connCmd)

	// 检查进程信息
	processCmd := fmt.Sprintf("lsof -i :%d 2>/dev/null || fuser %d/tcp 2>/dev/null", port, port)
	processResult, _ := ctx.SSH.Exec(host, processCmd)

	// 解析状态
	isListening := t.isPortListening(listenResult)
	serviceInfo := t.parseServiceInfo(listenResult)
	connCount := t.countConnections(connResult)
	connDetails := t.parseConnections(connResult)

	var firewallResult string
	var firewallStatus string
	if checkFirewall {
		// 检查 iptables 规则
		iptablesCmd := fmt.Sprintf("iptables -L -n 2>/dev/null | grep %d || echo '无相关 iptables 规则'", port)
		iptablesResult, _ := ctx.SSH.Exec(host, iptablesCmd)

		// 检查 firewalld
		firewalldCmd := fmt.Sprintf("firewall-cmd --list-ports 2>/dev/null | grep %d || echo 'firewalld 未启用或无规则'", port)
		firewalldResult, _ := ctx.SSH.Exec(host, firewalldCmd)

		firewallResult = fmt.Sprintf("=== iptables ===\n%s\n\n=== firewalld ===\n%s", iptablesResult, firewalldResult)
		firewallStatus = t.parseFirewallStatus(iptablesResult, firewalldResult, port)
	}

	status := t.getPortStatus(isListening, connCount)

	return tool.NewResult(map[string]interface{}{
		"host":           host,
		"port":           port,
		"is_listening":   isListening,
		"service_info":   serviceInfo,
		"conn_count":     connCount,
		"conn_details":   connDetails,
		"listen_info":    listenResult,
		"process_info":   processResult,
		"firewall":       firewallResult,
		"firewall_status": firewallStatus,
		"status":         status,
		"timestamp":      time.Now().Format(time.RFC3339),
	}, fmt.Sprintf("端口 %d 检查完成", port)), nil
}

// isPortListening 判断端口是否在监听
func (t *PortCheckTool) isPortListening(output string) bool {
	if strings.TrimSpace(output) == "" {
		return false
	}
	// 检查是否包含 LISTEN 或 LISTEN 字样
	return strings.Contains(strings.ToUpper(output), "LISTEN")
}

// parseServiceInfo 解析服务信息
func (t *PortCheckTool) parseServiceInfo(output string) map[string]interface{} {
	info := make(map[string]interface{})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// 解析进程信息
		// 格式可能: tcp LISTEN 0 128 *:80 *:* users:(("nginx",pid=1234,fd=6))
		if strings.Contains(line, "pid=") {
			re := regexp.MustCompile(`pid=(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				info["pid"] = matches[1]
			}

			// 提取进程名
			re = regexp.MustCompile(`users:\(\("([^"]+)"`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				info["process"] = matches[1]
			}

			// 提取协议
			if strings.HasPrefix(line, "tcp") {
				if strings.Contains(line, "tcp6") {
					info["protocol"] = "tcp6"
				} else {
					info["protocol"] = "tcp"
				}
			} else if strings.HasPrefix(line, "udp") {
				info["protocol"] = "udp"
			}
		}
	}

	// 如果没有找到详细信息，尝试简单解析
	if len(info) == 0 {
		fields := strings.Fields(output)
		if len(fields) > 0 {
			info["raw"] = strings.Join(fields, " ")
		}
	}

	return info
}

// countConnections 统计连接数
func (t *PortCheckTool) countConnections(output string) int {
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" &&
		   !strings.HasPrefix(line, "State") &&
		   !strings.HasPrefix(line, "Proto") {
			count++
		}
	}
	return count
}

// parseConnections 解析连接详情
func (t *PortCheckTool) parseConnections(output string) map[string]interface{} {
	result := make(map[string]interface{})

	lines := strings.Split(output, "\n")
	established := 0
	timeWait := 0
	listen := 0
	otherStates := 0

	for _, line := range lines {
		line = strings.ToUpper(line)
		if strings.Contains(line, "ESTAB") {
			established++
		} else if strings.Contains(line, "TIME-WAIT") || strings.Contains(line, "TIME_WAIT") {
			timeWait++
		} else if strings.Contains(line, "LISTEN") {
			listen++
		} else if strings.TrimSpace(line) != "" &&
		          !strings.HasPrefix(line, "PROTO") &&
		          !strings.HasPrefix(line, "STATE") {
			otherStates++
		}
	}

	result["established"] = established
	result["time_wait"] = timeWait
	result["listen"] = listen
	result["other"] = otherStates

	return result
}

// parseFirewallStatus 解析防火墙状态
func (t *PortCheckTool) parseFirewallStatus(iptables, firewalld string, port int) string {
	// 检查是否有开放规则
	if strings.Contains(iptables, "ACCEPT") {
		return "防火墙已配置允许"
	}
	if strings.Contains(firewalld, strconv.Itoa(port)) {
		return "防火墙已开放端口"
	}
	if strings.Contains(iptables, "无相关") && strings.Contains(firewalld, "未启用") {
		return "防火墙未启用或未配置"
	}
	return "防火墙状态未知"
}

// getPortStatus 获取端口状态
func (t *PortCheckTool) getPortStatus(isListening bool, connCount int) string {
	if !isListening {
		return "未监听"
	}
	if connCount > 1000 {
		return "连接数过多"
	}
	if connCount > 100 {
		return "活跃"
	}
	return "正常"
}
