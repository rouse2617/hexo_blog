package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// RootCauseDiagnosisTool 根因分析工具
type RootCauseDiagnosisTool struct{}

func (t *RootCauseDiagnosisTool) Name() string {
	return "diagnose_root_cause"
}

func (t *RootCauseDiagnosisTool) Description() string {
	return `# 智能根因分析

## 功能说明
根据症状自动诊断问题根因，执行诊断流程并给出结论。

## 支持的症状类型
1. **service_down** - 服务不可用
2. **slow_system** - 系统变慢
3. **disk_full** - 磁盘空间不足
4. **oom** - 内存不足
5. **network_issue** - 网络问题
6. **high_cpu** - CPU 过高

## 输出内容
- 问题分类
- 诊断步骤
- 根因分析
- 解决建议

适用场景：服务异常、性能下降、资源耗尽等故障诊断。`
}

func (t *RootCauseDiagnosisTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标主机",
			Required:    true,
		},
		{
			Name:        "symptom",
			Type:        "string",
			Description: "症状类型：service_down, slow_system, disk_full, oom, network_issue, high_cpu",
			Required:    true,
			Enum:        []interface{}{"service_down", "slow_system", "disk_full", "oom", "network_issue", "high_cpu"},
		},
		{
			Name:        "service_name",
			Type:        "string",
			Description: "服务名（可选）",
			Required:    false,
		},
		{
			Name:        "port",
			Type:        "integer",
			Description: "端口号（可选）",
			Required:    false,
		},
	}
}

func (t *RootCauseDiagnosisTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	symptom := tool.GetStringParam(params, "symptom", "")

	if host == "" {
		return tool.NewErrorResult("参数 host 不能为空"), nil
	}

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	switch symptom {
	case "service_down":
		return t.diagnoseServiceDown(ctx, host, params)
	case "slow_system":
		return t.diagnoseSlowSystem(ctx, host, params)
	case "disk_full":
		return t.diagnoseDiskFull(ctx, host, params)
	case "oom":
		return t.diagnoseOOM(ctx, host, params)
	case "network_issue":
		return t.diagnoseNetworkIssue(ctx, host, params)
	case "high_cpu":
		return t.diagnoseHighCPU(ctx, host, params)
	default:
		return tool.NewErrorResult("未知的症状类型: " + symptom), nil
	}
}

// diagnoseServiceDown 服务不可用诊断
func (t *RootCauseDiagnosisTool) diagnoseServiceDown(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	serviceName := tool.GetStringParam(params, "service_name", "")
	port := tool.GetIntParam(params, "port", 0)

	steps := make([]map[string]interface{}, 0)
	conclusions := make([]string, 0)

	// Step 1: 检查服务状态
	if serviceName != "" {
		statusCmd := fmt.Sprintf("systemctl status %s --no-pager", serviceName)
		statusOutput, err := ctx.SSH.Exec(host, statusCmd)

		stepStatus := "passed"
		stepConclusion := ""

		if err != nil {
			stepStatus = "failed"
			stepConclusion = fmt.Sprintf("服务 %s 未运行", serviceName)
			conclusions = append(conclusions, stepConclusion)
		} else if strings.Contains(statusOutput, "active (running)") {
			stepConclusion = fmt.Sprintf("服务 %s 正在运行", serviceName)
			conclusions = append(conclusions, stepConclusion)
		} else if strings.Contains(statusOutput, "inactive") {
			stepStatus = "failed"
			stepConclusion = fmt.Sprintf("服务 %s 未运行", serviceName)
			conclusions = append(conclusions, stepConclusion)
		}

		steps = append(steps, map[string]interface{}{
			"step":       "检查服务状态",
			"cmd":        statusCmd,
			"status":     stepStatus,
			"output":     truncateOutput(statusOutput, 500),
			"conclusion": stepConclusion,
		})
	}

	// Step 2: 检查端口监听
	if port > 0 {
		portCmd := fmt.Sprintf("ss -tunlp | grep :%d", port)
		portOutput, _ := ctx.SSH.Exec(host, portCmd)

		stepStatus := "passed"
		stepConclusion := ""

		if strings.Contains(portOutput, fmt.Sprintf(":%d", port)) {
			stepConclusion = fmt.Sprintf("端口 %d 正在监听", port)
		} else {
			stepStatus = "failed"
			stepConclusion = fmt.Sprintf("端口 %d 未监听", port)
			conclusions = append(conclusions, stepConclusion)
		}

		steps = append(steps, map[string]interface{}{
			"step":       "检查端口监听",
			"cmd":        portCmd,
			"status":     stepStatus,
			"output":     truncateOutput(portOutput, 200),
			"conclusion": stepConclusion,
		})
	}

	// Step 3: 检查防火墙
	firewallCmd := "iptables -L -n | head -20"
	firewallOutput, _ := ctx.SSH.Exec(host, firewallCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查防火墙规则",
		"cmd":    firewallCmd,
		"output": truncateOutput(firewallOutput, 500),
	})

	// Step 4: 查看错误日志
	if serviceName != "" {
		logCmd := fmt.Sprintf("journalctl -u %s --since '10 minutes ago' -p err --no-pager", serviceName)
		logOutput, _ := ctx.SSH.Exec(host, logCmd)

		hasErrors := !strings.Contains(logOutput, "-- No entries")

		steps = append(steps, map[string]interface{}{
			"step":       "查看错误日志",
			"cmd":        logCmd,
			"output":     truncateOutput(logOutput, 1000),
			"has_errors": hasErrors,
		})

		if hasErrors {
			conclusions = append(conclusions, "发现错误日志，请检查")
		}
	}

	// 生成建议
	suggestion := t.generateServiceDownSuggestion(conclusions, serviceName, port)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"service":     serviceName,
		"steps":       steps,
		"conclusions": conclusions,
		"suggestion":  suggestion,
	}, "服务不可用诊断完成"), nil
}

// diagnoseSlowSystem 系统变慢诊断
func (t *RootCauseDiagnosisTool) diagnoseSlowSystem(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	steps := make([]map[string]interface{}, 0)
	issues := make([]string, 0)

	// 检查 CPU
	cpuCmd := "top -bn1 | grep 'Cpu(s)'"
	cpuOutput, _ := ctx.SSH.Exec(host, cpuCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查 CPU",
		"cmd":    cpuCmd,
		"output": cpuOutput,
	})

	// 检查 I/O wait
	if strings.Contains(strings.ToLower(cpuOutput), "iowait") {
		issues = append(issues, "I/O wait 较高，可能是磁盘瓶颈")
	}

	// 检查内存
	memCmd := "free -h"
	memOutput, _ := ctx.SSH.Exec(host, memCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查内存",
		"cmd":    memCmd,
		"output": memOutput,
	})

	// 检查 Swap
	if strings.Contains(strings.ToLower(memOutput), "swap") {
		lines := strings.Split(memOutput, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Swap:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 && fields[2] != "0B" {
					issues = append(issues, "Swap 被使用，可能存在内存压力")
				}
			}
		}
	}

	// 检查磁盘 I/O
	ioCmd := "iostat -x 1 2 2>/dev/null || vmstat 1 2"
	ioOutput, _ := ctx.SSH.Exec(host, ioCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查磁盘 I/O",
		"cmd":    ioCmd,
		"output": truncateOutput(ioOutput, 800),
	})

	// 检查高 CPU 进程
	processCmd := "ps aux --sort=-%cpu | head -10"
	processOutput, _ := ctx.SSH.Exec(host, processCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查高 CPU 进程",
		"cmd":    processCmd,
		"output": processOutput,
	})

	suggestion := ""
	if len(issues) > 0 {
		suggestion = "发现以下问题：\n" + strings.Join(issues, "\n")
	} else {
		suggestion = "系统资源使用正常，可能是应用层问题"
	}

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"steps":      steps,
		"issues":     issues,
		"suggestion": suggestion,
	}, "系统变慢诊断完成"), nil
}

// diagnoseDiskFull 磁盘满诊断
func (t *RootCauseDiagnosisTool) diagnoseDiskFull(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	steps := make([]map[string]interface{}, 0)

	// 检查磁盘使用
	diskCmd := "df -h | grep -v tmpfs | grep -v loop"
	diskOutput, _ := ctx.SSH.Exec(host, diskCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查磁盘使用",
		"cmd":    diskCmd,
		"output": diskOutput,
	})

	// 查找大目录
	largeDirCmd := "du -sh /var/* 2>/dev/null | sort -hr | head -10"
	largeDirOutput, _ := ctx.SSH.Exec(host, largeDirCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "查找大目录",
		"cmd":    largeDirCmd,
		"output": largeDirOutput,
	})

	// 查找可清理文件
	cleanupCmd := "find /tmp -type f -size +10M 2>/dev/null | head -10"
	cleanupOutput, _ := ctx.SSH.Exec(host, cleanupCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "查找可清理文件",
		"cmd":    cleanupCmd,
		"output": cleanupOutput,
	})

	suggestion := []string{
		"1. 清理日志文件: journalctl --vacuum-time=7d",
		"2. 清理临时文件: find /tmp -type f -mtime +7 -delete",
		"3. 清理包管理器缓存: yum clean all 或 apt-get clean",
	}

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"steps":      steps,
		"suggestion": strings.Join(suggestion, "\n"),
	}, "磁盘满诊断完成"), nil
}

// diagnoseOOM 内存不足诊断
func (t *RootCauseDiagnosisTool) diagnoseOOM(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	steps := make([]map[string]interface{}, 0)

	// 检查内存使用
	memCmd := "free -h"
	memOutput, _ := ctx.SSH.Exec(host, memCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查内存使用",
		"cmd":    memCmd,
		"output": memOutput,
	})

	// 检查 OOM 日志
	oomCmd := "dmesg | grep -i 'out of memory' | tail -10"
	oomOutput, _ := ctx.SSH.Exec(host, oomCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查 OOM 日志",
		"cmd":    oomCmd,
		"output": oomOutput,
	})

	// 检查大内存进程
	processCmd := "ps aux --sort=-%mem | head -10"
	processOutput, _ := ctx.SSH.Exec(host, processCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查大内存进程",
		"cmd":    processCmd,
		"output": processOutput,
	})

	suggestion := []string{
		"1. 识别并终止占用内存过多的进程",
		"2. 清理 page cache: sync && echo 1 > /proc/sys/vm/drop_caches",
		"3. 增加物理内存或 Swap",
		"4. 检查是否有内存泄漏",
	}

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"steps":      steps,
		"suggestion": strings.Join(suggestion, "\n"),
	}, "OOM 诊断完成"), nil
}

// diagnoseNetworkIssue 网络问题诊断
func (t *RootCauseDiagnosisTool) diagnoseNetworkIssue(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	steps := make([]map[string]interface{}, 0)
	issues := make([]string, 0)

	// 检查网络接口
	ifaceCmd := "ip addr show"
	ifaceOutput, _ := ctx.SSH.Exec(host, ifaceCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查网络接口",
		"cmd":    ifaceCmd,
		"output": truncateOutput(ifaceOutput, 500),
	})

	// 检查路由表
	routeCmd := "ip route show"
	routeOutput, _ := ctx.SSH.Exec(host, routeCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查路由表",
		"cmd":    routeCmd,
		"output": routeOutput,
	})

	// 检查 DNS
	dnsCmd := "cat /etc/resolv.conf"
	dnsOutput, _ := ctx.SSH.Exec(host, dnsCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查 DNS 配置",
		"cmd":    dnsCmd,
		"output": dnsOutput,
	})

	// 测试网络连通性
	pingCmd := "ping -c 3 8.8.8.8"
	pingOutput, err := ctx.SSH.Exec(host, pingCmd)
	if err != nil || strings.Contains(pingOutput, "100% packet loss") {
		issues = append(issues, "无法连接到外网（8.8.8.8）")
	}
	steps = append(steps, map[string]interface{}{
		"step":   "测试外网连通性",
		"cmd":    pingCmd,
		"output": pingOutput,
	})

	suggestion := "建议检查网络配置、防火墙规则和网关连接"

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"steps":      steps,
		"issues":     issues,
		"suggestion": suggestion,
	}, "网络问题诊断完成"), nil
}

// diagnoseHighCPU CPU 过高诊断
func (t *RootCauseDiagnosisTool) diagnoseHighCPU(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	steps := make([]map[string]interface{}, 0)

	// 检查 CPU 使用率
	cpuCmd := "top -bn1 | head -20"
	cpuOutput, _ := ctx.SSH.Exec(host, cpuCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查 CPU 使用率",
		"cmd":    cpuCmd,
		"output": cpuOutput,
	})

	// 检查高 CPU 进程
	processCmd := "ps aux --sort=-%cpu | head -15"
	processOutput, _ := ctx.SSH.Exec(host, processCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查高 CPU 进程",
		"cmd":    processCmd,
		"output": processOutput,
	})

	// 检查 CPU 负载
	loadCmd := "uptime"
	loadOutput, _ := ctx.SSH.Exec(host, loadCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查系统负载",
		"cmd":    loadCmd,
		"output": loadOutput,
	})

	// 检查 CPU 核心数
	coreCmd := "nproc"
	coreOutput, _ := ctx.SSH.Exec(host, coreCmd)
	steps = append(steps, map[string]interface{}{
		"step":   "检查 CPU 核心数",
		"cmd":    coreCmd,
		"output": coreOutput,
	})

	suggestion := []string{
		"1. 查看高 CPU 进程是否正常业务进程",
		"2. 检查是否有死循环或异常计算",
		"3. 考虑进行应用性能优化",
		"4. 必要时进行水平扩展",
	}

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"steps":      steps,
		"suggestion": strings.Join(suggestion, "\n"),
	}, "CPU 过高诊断完成"), nil
}

// generateServiceDownSuggestion 生成服务故障建议
func (t *RootCauseDiagnosisTool) generateServiceDownSuggestion(conclusions []string, serviceName string, port int) string {
	suggestions := []string{
		"1. 检查服务配置文件是否正确",
	}

	if serviceName != "" {
		suggestions = append(suggestions, fmt.Sprintf("2. 尝试重启服务: systemctl restart %s", serviceName))
	}

	if port > 0 {
		suggestions = append(suggestions, fmt.Sprintf("3. 检查端口 %d 是否被占用", port))
	}

	suggestions = append(suggestions, "4. 查看服务日志获取详细错误信息")

	return strings.Join(suggestions, "\n")
}

func NewRootCauseDiagnosisTool() *RootCauseDiagnosisTool {
	return &RootCauseDiagnosisTool{}
}
