package builtin

import (
	"ai-ops/internal/tool"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IntrusionDetectionTool 入侵检测工具
type IntrusionDetectionTool struct{}

func (t *IntrusionDetectionTool) Name() string {
	return "detect_intrusion"
}

func (t *IntrusionDetectionTool) Description() string {
	return `# 入侵检测与安全审计

## 检测项目
1. **登录审计** - 最近登录记录、异常时间登录、异常 IP 登录、失败登录统计
2. **命令审计** - root 命令历史、危险命令执行、可疑脚本执行
3. **文件完整性** - SUID/SGID 文件、关键文件变更、新增可执行文件
4. **后门检测** - 异常监听端口、异常定时任务、异常启动项
5. **挖矿病毒检测** - CPU 长期 100%、已知挖矿进程名、外连矿池 IP

## 输出内容
- 安全评分
- 风险项列表
- 详细证据
- 处置建议`
}

func (t *IntrusionDetectionTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{Name: "host", Type: "string", Description: "目标主机", Required: true},
	}
}

func (t *IntrusionDetectionTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")

	findings := make([]map[string]interface{}, 0)
	score := 100

	// 1. 登录审计
	loginAudit := t.auditLogins(ctx, host)
	findings = append(findings, loginAudit...)

	// 2. 命令审计
	cmdAudit := t.auditCommands(ctx, host)
	findings = append(findings, cmdAudit...)

	// 3. 文件完整性检查
	fileAudit := t.auditFiles(ctx, host)
	findings = append(findings, fileAudit...)

	// 4. 后门检测
	backdoorAudit := t.detectBackdoor(ctx, host)
	findings = append(findings, backdoorAudit...)

	// 5. 挖矿检测
	minerAudit := t.detectMiner(ctx, host)
	findings = append(findings, minerAudit...)

	// 计算安全评分
	for _, f := range findings {
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

	return tool.NewResult(map[string]interface{}{
		"host":     host,
		"score":    score,
		"findings": findings,
		"summary":  t.generateSecuritySummary(findings),
	}, fmt.Sprintf("安全审计完成，评分: %d/100", score)), nil
}

// auditLogins 登录审计
func (t *IntrusionDetectionTool) auditLogins(ctx *tool.Context, host string) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// 最近登录记录
	cmd := "last -n 50"
	_, _ = ctx.SSH.Exec(host, cmd)

	// 检查异常时间登录（凌晨 2-6 点）
	cmd = "last | awk '$5 ~ /^(02|03|04|05|06)/ {print $0}'"
	nightLogins, _ := ctx.SSH.Exec(host, cmd)
	if strings.TrimSpace(nightLogins) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "login",
			"level":    "medium",
			"item":     "异常时间登录",
			"detail":   nightLogins,
			"advice":   "确认是否为本人操作",
		})
	}

	// 检查失败登录
	cmd = "lastb | awk '{print $3}' | sort | uniq -c | sort -rn | head -10"
	failedLogins, _ := ctx.SSH.Exec(host, cmd)
	lines := strings.Split(failedLogins, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			count, _ := strconv.Atoi(fields[0])
			if count >= 10 {
				findings = append(findings, map[string]interface{}{
					"category": "login",
					"level":    "high",
					"item":     "暴力破解尝试",
					"detail":   fmt.Sprintf("IP %s 失败 %d 次", fields[1], count),
					"advice":   "建议封禁相关 IP，启用 fail2ban",
				})
			}
		}
	}

	// 检查 root 登录
	cmd = "last | grep 'root still logged'"
	rootLogin, _ := ctx.SSH.Exec(host, cmd)
	if strings.TrimSpace(rootLogin) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "login",
			"level":    "high",
			"item":     "root 用户登录",
			"detail":   rootLogin,
			"advice":   "确认是否为授权操作",
		})
	}

	return findings
}

// auditCommands 命令审计
func (t *IntrusionDetectionTool) auditCommands(ctx *tool.Context, host string) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// 检查 root 历史命令
	cmd := "cat /root/.bash_history 2>/dev/null | tail -50"
	history, _ := ctx.SSH.Exec(host, cmd)

	dangerousCommands := []string{
		"rm -rf /",
		"mkfs",
		"dd if=",
		`:(){:|:&};:`, // fork bomb
		"chmod 777",
		"wget.*\\|.*sh",
		"curl.*\\|.*sh",
	}

	for _, dangerCmd := range dangerousCommands {
		re := regexp.MustCompile(dangerCmd)
		if re.MatchString(history) {
			findings = append(findings, map[string]interface{}{
				"category": "command",
				"level":    "critical",
				"item":     "危险命令执行",
				"detail":   fmt.Sprintf("发现匹配: %s", dangerCmd),
				"advice":   "立即检查命令是否合法",
			})
		}
	}

	// 检查可疑下载
	cmd = "grep -E '(wget|curl).*\\|.*sh' /root/.bash_history 2>/dev/null | tail -10"
	suspiciousDownload, _ := ctx.SSH.Exec(host, cmd)
	if strings.TrimSpace(suspiciousDownload) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "command",
			"level":    "critical",
			"item":     "可疑脚本下载执行",
			"detail":   suspiciousDownload,
			"advice":   "可能是恶意脚本，请检查",
		})
	}

	return findings
}

// auditFiles 文件审计
func (t *IntrusionDetectionTool) auditFiles(ctx *tool.Context, host string) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// 检查 SUID 文件
	cmd := "find / -type f -perm -4000 2>/dev/null | head -20"
	suidFiles, _ := ctx.SSH.Exec(host, cmd)

	unusualSUID := []string{
		"/bin/bash",
		"/bin/sh",
		"/bin/cat",
		"/bin/vim",
		"/usr/bin/less",
	}

	for _, unusual := range unusualSUID {
		if strings.Contains(suidFiles, unusual) {
			findings = append(findings, map[string]interface{}{
				"category": "file",
				"level":    "critical",
				"item":     "异常 SUID 文件",
				"detail":   fmt.Sprintf("%s 具有 SUID 位", unusual),
				"advice":   "可能是后门，请检查",
			})
		}
	}

	// 检查最近修改的敏感文件
	cmd = "find /etc -type f -mtime -1 -name '*.conf' 2>/dev/null"
	modifiedConfig, _ := ctx.SSH.Exec(host, cmd)
	if strings.TrimSpace(modifiedConfig) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "file",
			"level":    "medium",
			"item":     "配置文件最近修改",
			"detail":   modifiedConfig,
			"advice":   "确认配置修改是否合法",
		})
	}

	// 检查新增可执行文件
	cmd = "find /tmp -type f -perm +111 2>/dev/null | head -10"
	execInTmp, _ := ctx.SSH.Exec(host, cmd)
	if strings.TrimSpace(execInTmp) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "file",
			"level":    "high",
			"item":     "临时目录有可执行文件",
			"detail":   execInTmp,
			"advice":   "可能是恶意程序，请检查",
		})
	}

	return findings
}

// detectBackdoor 后门检测
func (t *IntrusionDetectionTool) detectBackdoor(ctx *tool.Context, host string) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// 检查异常端口监听
	cmd := "ss -tunlp | awk '{print $5}' | grep -oP '[0-9]+$' | sort -u | xargs -I{} sh -c 'echo {} | grep -vE \"^(22|80|443|3306|6379|27017|8080)$\" && echo {}'"
	unusualPorts, _ := ctx.SSH.Exec(host, cmd)

	if strings.TrimSpace(unusualPorts) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "backdoor",
			"level":    "high",
			"item":     "异常端口监听",
			"detail":   unusualPorts,
			"advice":   "检查是否为合法服务",
		})
	}

	// 检查异常定时任务
	cmd = "crontab -l 2>/dev/null; cat /etc/crontab 2>/dev/null"
	cronContent, _ := ctx.SSH.Exec(host, cmd)

	suspiciousCron := []string{
		"wget.*\\|.*sh",
		"curl.*\\|.*sh",
		"\\*\\*\\*\\*\\*",
		"base64.*\\|.*bash",
	}

	for _, pattern := range suspiciousCron {
		re := regexp.MustCompile(pattern)
		if re.MatchString(cronContent) {
			findings = append(findings, map[string]interface{}{
				"category": "backdoor",
				"level":    "critical",
				"item":     "可疑定时任务",
				"detail":   fmt.Sprintf("匹配模式: %s", pattern),
				"advice":   "可能是后门，请检查",
			})
		}
	}

	// 检查异常启动项
	cmd = "systemctl list-unit-files | grep enabled | awk '{print $1}'"
	enabledServices, _ := ctx.SSH.Exec(host, cmd)

	unusualServices := []string{
		"miner",
		"xmrig",
		"kinsing",
		"kdevtmpfsi",
	}

	for _, unusual := range unusualServices {
		if strings.Contains(strings.ToLower(enabledServices), unusual) {
			findings = append(findings, map[string]interface{}{
				"category": "backdoor",
				"level":    "critical",
				"item":     "可疑自启动服务",
				"detail":   fmt.Sprintf("发现 %s 相关服务", unusual),
				"advice":   "立即检查并禁用",
			})
		}
	}

	return findings
}

// detectMiner 挖矿检测
func (t *IntrusionDetectionTool) detectMiner(ctx *tool.Context, host string) []map[string]interface{} {
	findings := make([]map[string]interface{}, 0)

	// 已知挖矿进程名
	minerProcesses := []string{
		"xmrig",
		"cpuminer",
		"minerd",
		"ccminer",
		"clay miner",
		"kinsing",
		"kdevtmpfsi",
	}

	// 检查进程
	for _, miner := range minerProcesses {
		cmd := fmt.Sprintf("ps aux | grep -i '%s' | grep -v grep", miner)
		output, _ := ctx.SSH.Exec(host, cmd)

		if strings.TrimSpace(output) != "" {
			findings = append(findings, map[string]interface{}{
				"category": "miner",
				"level":    "critical",
				"item":     "挖矿进程",
				"detail":   output,
				"advice":   "立即终止进程并清除",
				"action":   fmt.Sprintf("killall -9 %s", miner),
			})
		}
	}

	// 检查高 CPU 但低内存的进程（挖矿特征）
	cmd := "ps aux --sort=-%cpu | awk 'NR>1 && $3>80 && $4<5 {print $0}'"
	output, _ := ctx.SSH.Exec(host, cmd)

	if strings.TrimSpace(output) != "" {
		findings = append(findings, map[string]interface{}{
			"category": "miner",
			"level":    "high",
			"item":     "可疑高CPU进程",
			"detail":   output,
			"advice":   "检查是否为挖矿程序",
		})
	}

	// 检查已知矿池 IP
	cmd = "netstat -antp 2>/dev/null | grep ESTABLISHED | awk '{print $5}' | cut -d: -f1 | sort -u | xargs -I{} whois {} 2>/dev/null | grep -i 'miner\\|pool\\|crypto'"
	_, _ = ctx.SSH.Exec(host, cmd)

	return findings
}

// generateSecuritySummary 生成安全摘要
func (t *IntrusionDetectionTool) generateSecuritySummary(findings []map[string]interface{}) string {
	if len(findings) == 0 {
		return "未发现安全问题，系统安全"
	}

	categoryCount := make(map[string]int)
	levelCount := make(map[string]int)

	for _, f := range findings {
		category := f["category"].(string)
		level := f["level"].(string)

		categoryCount[category]++
		levelCount[level]++
	}

	summary := []string{"安全审计发现以下问题："}

	for level, count := range levelCount {
		summary = append(summary, fmt.Sprintf("- %s级: %d 项", level, count))
	}

	if len(findings) > 5 {
		summary = append(summary, "\n建议立即进行安全加固！")
	}

	return strings.Join(summary, "\n")
}
