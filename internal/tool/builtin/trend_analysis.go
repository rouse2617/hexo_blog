package builtin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/tool"
)

// TrendAnalysisTool 趋势分析工具
type TrendAnalysisTool struct{}

func (t *TrendAnalysisTool) Name() string {
	return "analyze_trend"
}

func (t *TrendAnalysisTool) Description() string {
	return `# 资源趋势分析与容量预测

## 功能说明
基于当前数据分析资源使用趋势，预测未来容量需求。

## 分析维度
1. **磁盘增长趋势** - 按目录分析增长速度，预测磁盘满时间
2. **内存使用趋势** - 内存增长趋势，预测 OOM 风险
3. **日志增长趋势** - 日志文件增长速度，给出日志轮转建议
4. **网络流量趋势** - 流量峰值统计，带宽使用率

## 输出内容
- 当前使用情况
- 增长速度分析
- 容量预测
- 扩容建议

## 使用示例
- 单主机磁盘趋势分析: metric="disk", host="server1"
- 多主机内存趋势: metric="memory", hosts=["server1", "server2"]
- 日志增长分析: metric="log", days=30
- 网络流量趋势: metric="network", days=7

## 判断标准
- **正常**: 增长速度 < 5%/天, 容量可用 > 30天
- **警告**: 增长速度 5-10%/天, 容量可用 7-30天
- **危险**: 增长速度 > 10%/天, 容量可用 < 7天
- **紧急**: 容量可用 < 3天, 需要立即处理`
}

func (t *TrendAnalysisTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标主机名称",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标主机（批量分析）",
			Required:    false,
		},
		{
			Name:        "metric",
			Type:        "string",
			Description: "指标类型: disk(磁盘)/memory(内存)/log(日志)/network(网络)",
			Required:    false,
			Default:     "disk",
			Enum:        []interface{}{"disk", "memory", "log", "network"},
		},
		{
			Name:        "days",
			Type:        "integer",
			Description: "分析天数（用于查找历史数据）",
			Required:    false,
			Default:     7,
		},
	}
}

func (t *TrendAnalysisTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	metric := tool.GetStringParam(params, "metric", "disk")
	days := tool.GetIntParam(params, "days", 7)

	// 单主机分析
	if host != "" {
		return t.analyzeSingleHost(ctx, host, metric, days)
	}

	// 多主机批量分析
	if len(hosts) > 0 {
		return t.analyzeMultipleHosts(ctx, hosts, metric, days)
	}

	return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
}

// analyzeSingleHost 分析单个主机
func (t *TrendAnalysisTool) analyzeSingleHost(ctx *tool.Context, host string, metric string, days int) (*tool.Result, error) {
	switch metric {
	case "disk":
		return t.analyzeDiskTrend(ctx, host, days)
	case "memory":
		return t.analyzeMemoryTrend(ctx, host, days)
	case "log":
		return t.analyzeLogTrend(ctx, host, days)
	case "network":
		return t.analyzeNetworkTrend(ctx, host, days)
	default:
		return t.analyzeDiskTrend(ctx, host, days)
	}
}

// analyzeMultipleHosts 分析多个主机
func (t *TrendAnalysisTool) analyzeMultipleHosts(ctx *tool.Context, hosts []string, metric string, days int) (*tool.Result, error) {
	results := make([]map[string]interface{}, 0, len(hosts))

	for _, host := range hosts {
		result, err := t.analyzeSingleHost(ctx, host, metric, days)
		if err != nil {
			results = append(results, map[string]interface{}{
				"host":  host,
				"error": err.Error(),
			})
			continue
		}

		results = append(results, map[string]interface{}{
			"host":   host,
			"metric": metric,
			"data":   result.Data,
		})
	}

	return tool.NewResult(results, fmt.Sprintf("成功分析 %d 个主机的 %s 趋势", len(hosts), metric)), nil
}

// analyzeDiskTrend 分析磁盘趋势
func (t *TrendAnalysisTool) analyzeDiskTrend(ctx *tool.Context, host string, days int) (*tool.Result, error) {
	// 获取当前磁盘使用
	currentCmd := "df -h | grep -v tmpfs | grep -v loop"
	currentOutput, err := ctx.SSH.Exec(host, currentCmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("获取磁盘信息失败: %v", err)), nil
	}

	// 分析大目录增长
	analyzeCmd := fmt.Sprintf(`
		# 检查大目录
		du -sh /var/* 2>/dev/null | sort -hr | head -10

		# 检查日志目录
		find /var/log -type f -name "*.log" -mtime -%d -exec ls -lh {} \; 2>/dev/null | head -20
	`, days)

	analyzeOutput, _ := ctx.SSH.Exec(host, analyzeCmd)

	// 分析 inode 使用
	inodeCmd := "df -i | grep -v tmpfs | grep -v loop"
	inodeOutput, _ := ctx.SSH.Exec(host, inodeCmd)

	// 计算预测
	predictions := t.predictDiskFull(currentOutput)

	// 生成建议
	suggestions := t.generateDiskSuggestions(predictions)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"current":     strings.TrimSpace(currentOutput),
		"analysis":    strings.TrimSpace(analyzeOutput),
		"inode":       strings.TrimSpace(inodeOutput),
		"predictions": predictions,
		"suggestions": strings.Join(suggestions, "\n"),
		"timestamp":   time.Now().Format(time.RFC3339),
	}, "磁盘趋势分析完成"), nil
}

// analyzeMemoryTrend 分析内存趋势
func (t *TrendAnalysisTool) analyzeMemoryTrend(ctx *tool.Context, host string, days int) (*tool.Result, error) {
	// 当前内存状态
	currentCmd := "free -h"
	currentOutput, err := ctx.SSH.Exec(host, currentCmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("获取内存信息失败: %v", err)), nil
	}

	// 检查大内存进程
	processCmd := "ps aux --sort=-%mem | head -20"
	processOutput, _ := ctx.SSH.Exec(host, processCmd)

	// 检查 Swap 使用
	swapCmd := "swapon --show"
	swapOutput, _ := ctx.SSH.Exec(host, swapCmd)

	// 检查内存统计
	memInfoCmd := "cat /proc/meminfo | grep -E 'MemTotal|MemFree|MemAvailable|Buffers|Cached|SwapTotal|SwapFree'"
	memInfoOutput, _ := ctx.SSH.Exec(host, memInfoCmd)

	// 分析内存趋势
	trendAnalysis := t.analyzeMemoryTrendData(currentOutput)

	// 生成建议
	suggestions := t.generateMemorySuggestions(trendAnalysis)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"current":     strings.TrimSpace(currentOutput),
		"processes":   strings.TrimSpace(processOutput),
		"swap":        strings.TrimSpace(swapOutput),
		"meminfo":     strings.TrimSpace(memInfoOutput),
		"trend":       trendAnalysis,
		"suggestions": strings.Join(suggestions, "\n"),
		"timestamp":   time.Now().Format(time.RFC3339),
	}, "内存趋势分析完成"), nil
}

// analyzeLogTrend 分析日志趋势
func (t *TrendAnalysisTool) analyzeLogTrend(ctx *tool.Context, host string, days int) (*tool.Result, error) {
	// 检查日志文件大小
	logCmd := fmt.Sprintf(`
		find /var/log -type f -name "*.log" -mtime -%d -exec ls -lh {} \; 2>/dev/null | \
		awk '{print $5, $9}' | sort -hr | head -20
	`, days)
	logOutput, err := ctx.SSH.Exec(host, logCmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("获取日志信息失败: %v", err)), nil
	}

	// 检查日志轮转配置
	rotationCmd := "ls -la /etc/logrotate.d/ 2>/dev/null || echo 'logrotate 未配置'"
	rotationOutput, _ := ctx.SSH.Exec(host, rotationCmd)

	// 检查系统日志
	journalCmd := fmt.Sprintf("journalctl --disk-usage")
	journalOutput, _ := ctx.SSH.Exec(host, journalCmd)

	// 分析日志增长速度
	growthAnalysis := t.analyzeLogGrowth(logOutput)

	// 生成建议
	suggestions := t.generateLogSuggestions(growthAnalysis)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"log_files":   strings.TrimSpace(logOutput),
		"rotation":    strings.TrimSpace(rotationOutput),
		"journal":     strings.TrimSpace(journalOutput),
		"growth":      growthAnalysis,
		"suggestions": strings.Join(suggestions, "\n"),
		"timestamp":   time.Now().Format(time.RFC3339),
	}, "日志趋势分析完成"), nil
}

// analyzeNetworkTrend 分析网络趋势
func (t *TrendAnalysisTool) analyzeNetworkTrend(ctx *tool.Context, host string, days int) (*tool.Result, error) {
	// 获取网卡流量
	networkCmd := `
		cat /proc/net/dev | grep -v "Inter\|face" | while read line; do
			interface=$(echo $line | cut -d: -f1 | tr -d ' ')
			rx=$(echo $line | awk '{print $2}')
			tx=$(echo $line | awk '{print $10}')
			echo "$interface $rx $tx"
		done
	`
	networkOutput, err := ctx.SSH.Exec(host, networkCmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("获取网络信息失败: %v", err)), nil
	}

	// 获取连接统计
	connCmd := "ss -s"
	connOutput, _ := ctx.SSH.Exec(host, connCmd)

	// 获取网络接口详细信息
	ifConfigCmd := "ip addr show 2>/dev/null || ifconfig 2>/dev/null"
	ifConfigOutput, _ := ctx.SSH.Exec(host, ifConfigCmd)

	// 分析网络趋势
	trendAnalysis := t.analyzeNetworkTrendData(networkOutput)

	// 生成建议
	suggestions := t.generateNetworkSuggestions(trendAnalysis)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"network":     strings.TrimSpace(networkOutput),
		"connections": strings.TrimSpace(connOutput),
		"interfaces":  strings.TrimSpace(ifConfigOutput),
		"trend":       trendAnalysis,
		"suggestions": strings.Join(suggestions, "\n"),
		"timestamp":   time.Now().Format(time.RFC3339),
	}, "网络趋势分析完成"), nil
}

// predictDiskFull 预测磁盘满时间
func (t *TrendAnalysisTool) predictDiskFull(diskOutput string) map[string]interface{} {
	lines := strings.Split(diskOutput, "\n")
	predictions := make([]map[string]interface{}, 0)

	for _, line := range lines {
		if strings.Contains(line, "/") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				mountPoint := fields[len(fields)-1]
				usageStr := strings.TrimSuffix(fields[4], "%")

				if usage, err := strconv.Atoi(usageStr); err == nil {
					var daysUntilFull int
					var riskLevel string
					var actionRequired string

					switch {
					case usage < 50:
						daysUntilFull = 999
						riskLevel = "低"
						actionRequired = "无需操作"
					case usage < 70:
						daysUntilFull = 60
						riskLevel = "中"
						actionRequired = "定期监控"
					case usage < 85:
						daysUntilFull = 30
						riskLevel = "高"
						actionRequired = "开始清理或规划扩容"
					case usage < 95:
						daysUntilFull = 7
						riskLevel = "严重"
						actionRequired = "立即清理或扩容"
					default:
						daysUntilFull = 1
						riskLevel = "紧急"
						actionRequired = "紧急处理！磁盘即将满"
					}

					predictions = append(predictions, map[string]interface{}{
						"mount_point":     mountPoint,
						"current_usage":   usage,
						"risk_level":      riskLevel,
						"days_until_full": daysUntilFull,
						"action_required": actionRequired,
					})
				}
			}
		}
	}

	return map[string]interface{}{
		"predictions": predictions,
		"total":       len(predictions),
	}
}

// analyzeMemoryTrendData 分析内存趋势数据
func (t *TrendAnalysisTool) analyzeMemoryTrendData(output string) map[string]interface{} {
	lines := strings.Split(output, "\n")

	var memPercent int
	var swapPercent int
	var memTotal, memUsed, memAvailable string
	var swapTotal, swapUsed string

	for _, line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				memTotal = fields[1]
				memUsed = fields[2]
				memAvailable = fields[3]

				// 尝试解析内存使用率
				totalStr := strings.TrimSuffix(memTotal, "G")
				totalStr = strings.TrimSuffix(totalStr, "M")
				usedStr := strings.TrimSuffix(memUsed, "G")
				usedStr = strings.TrimSuffix(usedStr, "M")

				if total, err1 := strconv.ParseFloat(totalStr, 64); err1 == nil {
					if used, err2 := strconv.ParseFloat(usedStr, 64); err2 == nil {
						if total > 0 {
							memPercent = int((used / total) * 100)
						}
					}
				}
			}
		}

		if strings.HasPrefix(line, "Swap:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				swapTotal = fields[1]
				swapUsed = fields[2]

				// 计算 Swap 使用率
				totalStr := strings.TrimSuffix(swapTotal, "G")
				totalStr = strings.TrimSuffix(totalStr, "M")
				usedStr := strings.TrimSuffix(swapUsed, "G")
				usedStr = strings.TrimSuffix(usedStr, "M")

				if total, err1 := strconv.ParseFloat(totalStr, 64); err1 == nil && total > 0 {
					if used, err2 := strconv.ParseFloat(usedStr, 64); err2 == nil {
						swapPercent = int((used / total) * 100)
					}
				}
			}
		}
	}

	// 评估趋势
	trend := t.getMemoryTrend(memPercent)

	// 评估 OOM 风险
	oomRisk := t.assessOOMRisk(memPercent, swapPercent)

	return map[string]interface{}{
		"memory_total":     memTotal,
		"memory_used":      memUsed,
		"memory_available": memAvailable,
		"memory_percent":   memPercent,
		"swap_total":       swapTotal,
		"swap_used":        swapUsed,
		"swap_percent":     swapPercent,
		"trend":            trend,
		"oom_risk":         oomRisk,
	}
}

// analyzeLogGrowth 分析日志增长
func (t *TrendAnalysisTool) analyzeLogGrowth(output string) map[string]interface{} {
	lines := strings.Split(output, "\n")

	totalSize := int64(0)
	fileCount := 0
	largestFiles := make([]map[string]string, 0, 5)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			sizeStr := fields[0]
			filePath := fields[len(fields)-1]

			// 解析大小
			if size := t.parseSize(sizeStr); size > 0 {
				totalSize += size
				fileCount++

				// 记录最大的 5 个文件
				if len(largestFiles) < 5 {
					largestFiles = append(largestFiles, map[string]string{
						"file": filePath,
						"size": sizeStr,
					})
				}
			}
		}
	}

	avgSize := int64(0)
	if fileCount > 0 {
		avgSize = totalSize / int64(fileCount)
	}

	// 评估风险
	risk := "低"
	totalMB := totalSize / (1024 * 1024)
	switch {
	case totalMB > 1024: // > 1GB
		risk = "紧急"
	case totalMB > 512: // > 512MB
		risk = "严重"
	case totalMB > 100: // > 100MB
		risk = "高"
	case totalMB > 50: // > 50MB
		risk = "中"
	}

	return map[string]interface{}{
		"total_size_mb": totalMB,
		"avg_size_mb":   avgSize / (1024 * 1024),
		"file_count":    fileCount,
		"risk_level":    risk,
		"largest_files": largestFiles,
	}
}

// analyzeNetworkTrendData 分析网络趋势
func (t *TrendAnalysisTool) analyzeNetworkTrendData(output string) map[string]interface{} {
	lines := strings.Split(output, "\n")

	interfaces := make([]map[string]interface{}, 0)
	totalRx := int64(0)
	totalTx := int64(0)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			iface := fields[0]
			rx := parseInt64(fields[1])
			tx := parseInt64(fields[2])

			totalRx += rx
			totalTx += tx

			interfaces = append(interfaces, map[string]interface{}{
				"name":  iface,
				"rx":    formatBytes(rx),
				"tx":    formatBytes(tx),
				"rx_gb": float64(rx) / (1024 * 1024 * 1024),
				"tx_gb": float64(tx) / (1024 * 1024 * 1024),
			})
		}
	}

	return map[string]interface{}{
		"interfaces":      interfaces,
		"total_rx":        formatBytes(totalRx),
		"total_tx":        formatBytes(totalTx),
		"interface_count": len(interfaces),
	}
}

// 辅助方法

// parseSize 解析大小字符串为字节数
func (t *TrendAnalysisTool) parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(sizeStr)

	re := regexp.MustCompile(`(\d+(?:\.\d+)?)([KMGT]?B?)`)
	matches := re.FindStringSubmatch(sizeStr)

	if len(matches) < 3 {
		return 0
	}

	value, _ := strconv.ParseFloat(matches[1], 64)
	unit := matches[2]

	switch unit {
	case "T", "TB":
		return int64(value * 1024 * 1024 * 1024 * 1024)
	case "G", "GB":
		return int64(value * 1024 * 1024 * 1024)
	case "M", "MB":
		return int64(value * 1024 * 1024)
	case "K", "KB":
		return int64(value * 1024)
	default:
		return int64(value)
	}
}

// getMemoryTrend 获取内存趋势描述
func (t *TrendAnalysisTool) getMemoryTrend(percent int) string {
	switch {
	case percent < 50:
		return "稳定"
	case percent < 70:
		return "上升"
	case percent < 85:
		return "快速增长"
	default:
		return "危险"
	}
}

// assessOOMRisk 评估 OOM 风险
func (t *TrendAnalysisTool) assessOOMRisk(memPercent, swapPercent int) string {
	if swapPercent > 50 && memPercent > 90 {
		return "极高风险 - 即将发生 OOM"
	}
	if swapPercent > 30 && memPercent > 85 {
		return "高风险 - OOM 可能性大"
	}
	if swapPercent > 10 && memPercent > 80 {
		return "中风险 - 需要关注"
	}
	if memPercent > 90 {
		return "中风险 - 内存压力高"
	}
	return "低风险"
}

// generateDiskSuggestions 生成磁盘建议
func (t *TrendAnalysisTool) generateDiskSuggestions(predictions map[string]interface{}) []string {
	suggestions := []string{
		"1. 设置日志轮转: 配置 logrotate",
		"2. 定期清理临时文件: 使用 tmpwatch 或 tmpreaper",
		"3. 清理包管理器缓存: yum clean all / apt-get clean",
		"4. 清理旧日志: journalctl --vacuum-time=30d",
		"5. 监控大文件: find / -size +100M -type f",
		"6. 清理 Docker 镜像和容器: docker system prune -a",
	}

	// 根据预测添加特定建议
	if preds, ok := predictions["predictions"].([]map[string]interface{}); ok {
		for _, pred := range preds {
			if risk, ok := pred["risk_level"].(string); ok && (risk == "严重" || risk == "紧急") {
				mountPoint := pred["mount_point"].(string)
				suggestions = append([]string{
					fmt.Sprintf("⚠️  警告: %s 使用率 %d%% (%s)", mountPoint, pred["current_usage"], risk),
					fmt.Sprintf("   %s", pred["action_required"]),
				}, suggestions...)
			}
		}
	}

	return suggestions
}

// generateMemorySuggestions 生成内存建议
func (t *TrendAnalysisTool) generateMemorySuggestions(trend map[string]interface{}) []string {
	percent := trend["memory_percent"].(int)
	oomRisk := trend["oom_risk"].(string)

	suggestions := []string{
		"1. 定期监控内存使用情况",
		"2. 检查是否有内存泄漏: 使用 top/ps/valgrind",
	}

	if percent > 70 {
		suggestions = append(suggestions,
			"3. 分析大内存进程: ps aux --sort=-%mem | head -20",
			"4. 考虑增加物理内存",
			"5. 优化应用内存使用",
			"6. 检查 Swap 使用是否正常",
		)
	} else {
		suggestions = append(suggestions, "2. 当前内存使用正常，继续监控")
	}

	if oomRisk != "低风险" {
		suggestions = append([]string{
			fmt.Sprintf("⚠️  警告: %s", oomRisk),
		}, suggestions...)
	}

	return suggestions
}

// generateLogSuggestions 生成日志建议
func (t *TrendAnalysisTool) generateLogSuggestions(growth map[string]interface{}) []string {
	totalSizeMB := growth["total_size_mb"].(int64)
	risk := growth["risk_level"].(string)

	suggestions := []string{
		"1. 配置 logrotate 自动轮转日志",
		"2. 设置合适的日志保留策略",
	}

	if totalSizeMB > 1024 { // > 1GB
		suggestions = append(suggestions,
			"3. 日志总量较大，建议立即清理",
			"4. 考虑减少日志级别",
			"5. 使用日志中心集中管理 (ELK/Loki)",
			"6. 清理 journalctl: journalctl --vacuum-size=500M",
		)
	} else if totalSizeMB > 100 {
		suggestions = append(suggestions,
			"3. 日志量较大，建议清理旧日志",
			"4. 检查日志轮转是否正常工作",
		)
	} else {
		suggestions = append(suggestions, "3. 当前日志量正常")
	}

	if risk == "严重" || risk == "紧急" {
		suggestions = append([]string{
			fmt.Sprintf("⚠️  警告: 日志风险等级 %s, 总量 %d MB", risk, totalSizeMB),
		}, suggestions...)
	}

	return suggestions
}

// generateNetworkSuggestions 生成网络建议
func (t *TrendAnalysisTool) generateNetworkSuggestions(trend map[string]interface{}) []string {
	suggestions := []string{
		"1. 监控网络流量趋势",
		"2. 设置流量告警阈值",
		"3. 优化网络带宽使用",
		"4. 定期检查网络连接数: ss -s",
		"5. 分析异常连接: ss -tunap",
	}

	if interfaces, ok := trend["interfaces"].([]map[string]interface{}); ok && len(interfaces) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("6. 检测到 %d 个网络接口", len(interfaces)))
	}

	return suggestions
}

// 工具函数

// parseInt64 安全地解析 int64
func parseInt64(s string) int64 {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// formatBytes 格式化字节数为可读格式
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func NewTrendAnalysisTool() *TrendAnalysisTool {
	return &TrendAnalysisTool{}
}
