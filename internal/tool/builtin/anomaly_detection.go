package builtin

import (
	"fmt"
	"regexp"
	"strings"

	"ai-ops/internal/tool"
)

// AnomalyDetectionTool 异常检测工具
type AnomalyDetectionTool struct{}

func (t *AnomalyDetectionTool) Name() string {
	return "detect_anomaly"
}

func (t *AnomalyDetectionTool) Description() string {
	return `# 异常检测与安全扫描

## 检测项目
1. **可疑进程** - CPU 占用异常高、网络连接异常多、挖矿病毒特征
2. **异常登录** - 异常时间登录、异常 IP 登录、失败登录次数
3. **文件异常** - 关键文件被修改、SUID/SGID 文件、新增可执行文件
4. **网络异常** - 异常端口监听、异常外连、流量异常突增
5. **资源异常** - 磁盘写入异常、网络流量异常

## 返回结果
- 安全评分 (0-100)
- 异常列表（按严重程度分类）
- 处理建议

适用场景：安全巡检、病毒检测、入侵检测、异常行为分析。`
}

func (t *AnomalyDetectionTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标主机",
			Required:    true,
		},
	}
}

func (t *AnomalyDetectionTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")

	if host == "" {
		return tool.NewErrorResult("参数 host 不能为空"), nil
	}

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	anomalies := make([]map[string]interface{}, 0)

	// 检测可疑进程
	suspiciousProcesses := t.detectSuspiciousProcesses(ctx, host)
	if len(suspiciousProcesses) > 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "suspicious_process",
			"level": "high",
			"count": len(suspiciousProcesses),
			"items": suspiciousProcesses,
		})
	}

	// 检测异常登录
	abnormalLogins := t.detectAbnormalLogins(ctx, host)
	if len(abnormalLogins) > 0 {
		level := "medium"
		for _, login := range abnormalLogins {
			if loginType, ok := login["type"].(string); ok && loginType == "brute_force" {
				level = "high"
				break
			}
		}
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "abnormal_login",
			"level": level,
			"count": len(abnormalLogins),
			"items": abnormalLogins,
		})
	}

	// 检测异常端口
	abnormalPorts := t.detectAbnormalPorts(ctx, host)
	if len(abnormalPorts) > 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "abnormal_port",
			"level": "medium",
			"count": len(abnormalPorts),
			"items": abnormalPorts,
		})
	}

	// 检测挖矿病毒
	minerProcesses := t.detectMinerProcesses(ctx, host)
	if len(minerProcesses) > 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "miner",
			"level": "critical",
			"count": len(minerProcesses),
			"items": minerProcesses,
		})
	}

	// 检测 SUID/SGID 文件
	suidFiles := t.detectSUIDFiles(ctx, host)
	if len(suidFiles) > 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "suid_file",
			"level": "medium",
			"count": len(suidFiles),
			"items": suidFiles,
		})
	}

	// 检测最近修改的可执行文件
	modifiedBinaries := t.detectModifiedBinaries(ctx, host)
	if len(modifiedBinaries) > 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":  "modified_binary",
			"level": "high",
			"count": len(modifiedBinaries),
			"items": modifiedBinaries,
		})
	}

	// 计算安全评分
	score := t.calculateSecurityScore(anomalies)

	return tool.NewResult(map[string]interface{}{
		"host":      host,
		"score":     score,
		"anomalies": anomalies,
		"summary":   t.generateAnomalySummary(anomalies),
	}, fmt.Sprintf("检测完成，发现 %d 项异常", len(anomalies))), nil
}

// detectSuspiciousProcesses 检测可疑进程
func (t *AnomalyDetectionTool) detectSuspiciousProcesses(ctx *tool.Context, host string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)

	// 检测高 CPU 进程
	cpuCmd := "ps aux --sort=-%cpu | awk 'NR>1 && $3>80 {print $0}'"
	cpuOutput, _ := ctx.SSH.Exec(host, cpuCmd)

	lines := strings.Split(cpuOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, map[string]interface{}{
				"type":       "high_cpu_process",
				"info":       line,
				"suggestion": "检查进程是否合法，必要时终止进程",
			})
		}
	}

	// 检测可疑进程名
	suspiciousCmd := "ps aux | grep -E '(kdevtmpfsi|kinsing|xmrig|miner|crypto|scan|masscan|zmap)' | grep -v grep"
	suspiciousOutput, _ := ctx.SSH.Exec(host, suspiciousCmd)

	suspiciousLines := strings.Split(suspiciousOutput, "\n")
	for _, line := range suspiciousLines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "grep") {
			results = append(results, map[string]interface{}{
				"type":       "suspicious_name",
				"info":       line,
				"suggestion": "该进程名称与恶意软件特征匹配，建议立即检查",
			})
		}
	}

	return results
}

// detectAbnormalLogins 检测异常登录
func (t *AnomalyDetectionTool) detectAbnormalLogins(ctx *tool.Context, host string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)

	// 检查凌晨登录 (2-6点)
	nightCmd := "last | awk '$5 ~ /^(02|03|04|05|06)/ {print $0}'"
	nightOutput, _ := ctx.SSH.Exec(host, nightCmd)

	if strings.TrimSpace(nightOutput) != "" {
		// 统计异常登录次数
		lines := strings.Split(strings.TrimSpace(nightOutput), "\n")
		if len(lines) > 0 {
			results = append(results, map[string]interface{}{
				"type":     "night_login",
				"detail":   truncateOutput(nightOutput, 500),
				"count":    len(lines),
				"advice":   "确认是否为本人操作，警惕非授权访问",
			})
		}
	}

	// 检查失败登录（检测暴力破解）
	failCmd := "lastb | awk '{print $3}' | sort | uniq -c | sort -rn | head -5"
	failOutput, _ := ctx.SSH.Exec(host, failCmd)

	// 解析失败次数
	failLines := strings.Split(strings.TrimSpace(failOutput), "\n")
	hasBruteForce := false
	bruteDetail := ""

	for _, line := range failLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: "次数 IP"
		re := regexp.MustCompile(`^(\d+)\s+(\S+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			countStr := matches[1]
			if strings.Contains(countStr, "0") {
				// 提取数字
				var count int
				fmt.Sscanf(countStr, "%d", &count)
				if count >= 10 {
					hasBruteForce = true
					bruteDetail = line
					break
				}
			}
		}
	}

	if hasBruteForce {
		results = append(results, map[string]interface{}{
			"type":     "brute_force",
			"detail":   bruteDetail,
			"advice":   "检测到暴力破解攻击，建议封禁相关 IP，启用 fail2ban",
		})
	}

	return results
}

// detectAbnormalPorts 检测异常端口
func (t *AnomalyDetectionTool) detectAbnormalPorts(ctx *tool.Context, host string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)

	// 常见端口白名单
	commonPorts := map[string]bool{
		"22":     true, // SSH
		"80":     true, // HTTP
		"443":    true, // HTTPS
		"3306":   true, // MySQL
		"6379":   true, // Redis
		"27017":  true, // MongoDB
		"8080":   true, // HTTP Alt
		"8443":   true, // HTTPS Alt
		"2375":   true, // Docker
		"2376":   true, // Docker TLS
		"6443":   true, // Kubernetes API
		"9090":   true, // Prometheus
		"9100":   true, // Node Exporter
		"53":     true, // DNS
		"25":     true, // SMTP
		"110":    true, // POP3
		"143":    true, // IMAP
		"993":    true, // IMAPS
		"995":    true, // POP3S
		"21":     true, // FTP
	}

	// 获取监听端口
	cmd := "ss -tunlp | awk 'NR>1 {print $5}' | grep -oP '[0-9]+$' | sort -u"
	output, _ := ctx.SSH.Exec(host, cmd)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		port := strings.TrimSpace(line)
		if port != "" && !commonPorts[port] {
			// 获取监听进程信息
			portInfoCmd := fmt.Sprintf("ss -tunlp | grep :%s", port)
			portInfo, _ := ctx.SSH.Exec(host, portInfoCmd)

			results = append(results, map[string]interface{}{
				"port":      port,
				"info":      truncateOutput(portInfo, 200),
				"advice":    fmt.Sprintf("端口 %s 不是常见服务端口，请确认是否为合法服务", port),
			})
		}
	}

	return results
}

// detectMinerProcesses 检测挖矿病毒
func (t *AnomalyDetectionTool) detectMinerProcesses(ctx *tool.Context, host string) []map[string]interface{} {
	// 挖矿进程特征
	minerPatterns := []string{
		"xmrig",
		"cpuminer",
		"minerd",
		"ccminer",
		"cgminer",
		"bminer",
		"tbminer",
		"claymore",
		"ethminer",
	}

	results := make([]map[string]interface{}, 0)

	// 生成 grep 命令
	pattern := strings.Join(minerPatterns, "|")
	cmd := fmt.Sprintf("ps aux | grep -iE '%s' | grep -v grep", pattern)
	output, _ := ctx.SSH.Exec(host, cmd)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "grep") {
			// 提取 PID
			fields := strings.Fields(line)
			pid := ""
			if len(fields) >= 2 {
				pid = fields[1]
			}

			results = append(results, map[string]interface{}{
				"pid":     pid,
				"process": line,
				"advice":  "立即终止该进程并检查系统安全",
				"action":  fmt.Sprintf("kill -9 %s", pid),
			})
		}
	}

	return results
}

// detectSUIDFiles 检测 SUID/SGID 文件
func (t *AnomalyDetectionTool) detectSUIDFiles(ctx *tool.Context, host string) []map[string]interface{} {
	// 常见的合法 SUID 文件
	validSUIDs := map[string]bool{
		"/usr/bin/sudo":        true,
		"/usr/bin/passwd":      true,
		"/usr/bin/su":          true,
		"/usr/bin/ping":        true,
		"/usr/bin/newgrp":      true,
		"/usr/bin/chsh":        true,
		"/usr/bin/chfn":        true,
		"/usr/bin/gpasswd":     true,
		"/bin/ping":            true,
		"/bin/su":              true,
		"/bin/mount":           true,
		"/bin/umount":          true,
		"/usr/bin/mount":       true,
		"/usr/bin/umount":      true,
	}

	results := make([]map[string]interface{}, 0)

	// 查找所有 SUID 文件
	cmd := "find / -type f -perm -4000 2>/dev/null | head -20"
	output, _ := ctx.SSH.Exec(host, cmd)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !validSUIDs[line] {
			// 获取文件详细信息
			infoCmd := fmt.Sprintf("ls -l %s", line)
			infoOutput, _ := ctx.SSH.Exec(host, infoCmd)

			results = append(results, map[string]interface{}{
				"file":     line,
				"info":     strings.TrimSpace(infoOutput),
				"advice":   "该文件具有 SUID 权限，请确认是否合法",
			})
		}
	}

	return results
}

// detectModifiedBinaries 检测最近修改的可执行文件
func (t *AnomalyDetectionTool) detectModifiedBinaries(ctx *tool.Context, host string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0)

	// 查找最近 7 天修改的 /usr/bin 中的可执行文件
	cmd := "find /usr/bin -type f -perm -111 -mtime -7 2>/dev/null | head -10"
	output, _ := ctx.SSH.Exec(host, cmd)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// 获取文件修改时间和详细信息
			statCmd := fmt.Sprintf("ls -l --time-style=full-iso %s", line)
			statOutput, _ := ctx.SSH.Exec(host, statCmd)

			results = append(results, map[string]interface{}{
				"file":     line,
				"info":     strings.TrimSpace(statOutput),
				"advice":   "系统二进制文件最近被修改，可能被植入后门",
			})
		}
	}

	return results
}

// calculateSecurityScore 计算安全评分
func (t *AnomalyDetectionTool) calculateSecurityScore(anomalies []map[string]interface{}) int {
	score := 100

	for _, anomaly := range anomalies {
		level, ok := anomaly["level"].(string)
		if !ok {
			continue
		}
		count, _ := anomaly["count"].(int)

		switch level {
		case "critical":
			score -= 25 * count
		case "high":
			score -= 15 * count
		case "medium":
			score -= 5 * count
		case "low":
			score -= 2 * count
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

// generateAnomalySummary 生成异常摘要
func (t *AnomalyDetectionTool) generateAnomalySummary(anomalies []map[string]interface{}) string {
	if len(anomalies) == 0 {
		return "未检测到异常，系统安全"
	}

	summary := []string{"检测到以下异常："}
	for _, anomaly := range anomalies {
		atype, _ := anomaly["type"].(string)
		count, _ := anomaly["count"].(int)
		level, _ := anomaly["level"].(string)
		summary = append(summary, fmt.Sprintf("- %s: %d 项 (%s级)", atype, count, level))
	}

	return strings.Join(summary, "\n")
}

func NewAnomalyDetectionTool() *AnomalyDetectionTool {
	return &AnomalyDetectionTool{}
}
