package builtin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/tool"
)

// InodeCheckTool Inode 检查工具
type InodeCheckTool struct{}

// NewInodeCheckTool 创建 Inode 检查工具
func NewInodeCheckTool() *InodeCheckTool {
	return &InodeCheckTool{}
}

// Name 工具名称
func (t *InodeCheckTool) Name() string {
	return "check_inode"
}

// Description 工具描述
func (t *InodeCheckTool) Description() string {
	return `# Inode 使用情况检查

## 功能
- 检查各分区 inode 使用率
- 识别 inode 不足的分区
- 查找大量小文件的目录
- 评估 inode 使用风险

## 使用场景
- 磁盘有空间但无法创建文件时
- 系统提示 "No space left on device" 但磁盘未满时
- 定期预防性检查
- 分析小文件过多的目录`
}

// Parameters 工具参数
func (t *InodeCheckTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "hosts",
			Type:        "array",
			Description: "主机列表",
			Required:    true,
		},
		{
			Name:        "threshold",
			Type:        "integer",
			Description: "告警阈值（百分比），默认 80",
			Required:    false,
			Default:     80,
		},
		{
			Name:        "find_large_dirs",
			Type:        "boolean",
			Description: "是否查找 inode 使用高的目录",
			Required:    false,
			Default:     true,
		},
	}
}

// Execute 执行工具
func (t *InodeCheckTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	hostsParam := params["hosts"]
	threshold := tool.GetIntParam(params, "threshold", 80)
	findLargeDirs := tool.GetBoolParam(params, "find_large_dirs", true)

	var hosts []string

	// 处理主机列表参数
	switch v := hostsParam.(type) {
	case []string:
		hosts = v
	case []interface{}:
		hosts = make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				hosts = append(hosts, str)
			}
		}
	case string:
		hosts = []string{v}
	default:
		return tool.NewErrorResult("hosts 参数格式错误，需要字符串数组"), nil
	}

	if len(hosts) == 0 {
		return tool.NewErrorResult("至少需要提供一个主机"), nil
	}

	results := make([]map[string]interface{}, 0)
	criticalHosts := make([]string, 0)

	for _, host := range hosts {
		// 检查 inode
		cmd := "df -i"
		output, err := ctx.SSH.Exec(host, cmd)
		if err != nil {
			results = append(results, map[string]interface{}{
				"host":   host,
				"error":  err.Error(),
				"status": "检查失败",
			})
			continue
		}

		// 解析结果
		inodeInfo := t.parseInodeInfo(output, threshold)

		// 如果有分区使用率超过阈值，查找大目录
		if findLargeDirs && inodeInfo["critical_partitions"] != nil && len(inodeInfo["critical_partitions"].([]map[string]interface{})) > 0 {
			largeDirs := t.findLargeInodeDirectories(ctx, host)
			inodeInfo["large_directories"] = largeDirs
		}

		hostResult := map[string]interface{}{
			"host":      host,
			"inode_info": inodeInfo,
			"raw_output": output,
		}

		// 添加状态
		if critical, ok := inodeInfo["critical"].(bool); ok && critical {
			hostResult["status"] = "告警"
			hostResult["risk_level"] = "高"
			criticalHosts = append(criticalHosts, host)
		} else if warning, ok := inodeInfo["warning"].(bool); ok && warning {
			hostResult["status"] = "警告"
			hostResult["risk_level"] = "中"
		} else {
			hostResult["status"] = "正常"
			hostResult["risk_level"] = "低"
		}

		results = append(results, hostResult)
	}

	// 生成总体评估
	summary := t.generateSummary(results, threshold, len(criticalHosts))

	return tool.NewResult(map[string]interface{}{
		"results":   results,
		"summary":   summary,
		"threshold": threshold,
		"timestamp": time.Now().Format(time.RFC3339),
	}, fmt.Sprintf("完成 %d 台主机的 Inode 检查", len(hosts))), nil
}

// parseInodeInfo 解析 inode 信息
func (t *InodeCheckTool) parseInodeInfo(output string, threshold int) map[string]interface{} {
	lines := strings.Split(output, "\n")
	result := make(map[string]interface{})

	partitions := make([]map[string]interface{}, 0)
	criticalPartitions := make([]map[string]interface{}, 0)
	warningPartitions := make([]map[string]interface{}, 0)

	hasCritical := false
	hasWarning := false

	// 跳过标题行，从第二行开始处理
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		// 跳过包含 "Filesystem" 或 "Mounted" 的标题行
		if strings.Contains(line, "Filesystem") || strings.Contains(line, "IUse%") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		partition := make(map[string]interface{})

		// df -i 输出格式示例:
		// Filesystem      Inodes IUsed   IFree IUse% Mounted on
		// /dev/sda1      1310720 12345 1298375    1% /
		partition["filesystem"] = fields[0]

		// 解析总 inode 数
		if total, err := strconv.Atoi(fields[1]); err == nil {
			partition["total"] = total
		}

		// 解析已用 inode 数
		if used, err := strconv.Atoi(fields[2]); err == nil {
			partition["used"] = used
		}

		// 解析空闲 inode 数
		if free, err := strconv.Atoi(fields[3]); err == nil {
			partition["free"] = free
		}

		// 解析使用率百分比
		usageStr := strings.TrimSuffix(fields[4], "%")
		if usage, err := strconv.Atoi(usageStr); err == nil {
			partition["usage_percent"] = usage

			// 判断风险等级
			if usage >= threshold {
				partition["status"] = "严重"
				partition["risk"] = "高"
				criticalPartitions = append(criticalPartitions, partition)
				hasCritical = true
			} else if usage >= threshold-10 {
				partition["status"] = "警告"
				partition["risk"] = "中"
				warningPartitions = append(warningPartitions, partition)
				hasWarning = true
			} else {
				partition["status"] = "正常"
				partition["risk"] = "低"
			}
		}

		// 挂载点
		if len(fields) >= 6 {
			partition["mountpoint"] = strings.Join(fields[5:], " ")
		}

		partitions = append(partitions, partition)
	}

	result["partitions"] = partitions
	result["critical_partitions"] = criticalPartitions
	result["warning_partitions"] = warningPartitions
	result["critical"] = hasCritical
	result["warning"] = hasWarning

	return result
}

// findLargeInodeDirectories 查找 inode 使用率高的目录
func (t *InodeCheckTool) findLargeInodeDirectories(ctx *tool.Context, host string) map[string]interface{} {
	directories := make([]map[string]interface{}, 0)

	// 常见的高 inode 使用目录
	searchPaths := []string{"/var", "/tmp", "/home", "/opt", "/usr", "/"}

	for _, path := range searchPaths {
		// 查找该路径下文件数最多的目录
		cmd := fmt.Sprintf("find %s -xdev -printf '%%h\\n' 2>/dev/null | sort | uniq -c | sort -rn | head -10", path)
		output, _ := ctx.SSH.Exec(host, cmd)

		if strings.TrimSpace(output) != "" {
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					count, err := strconv.Atoi(fields[0])
					if err == nil {
						dirPath := strings.Join(fields[1:], " ")
						directories = append(directories, map[string]interface{}{
							"path":       dirPath,
							"file_count": count,
						})
					}
				}
			}
		}
	}

	// 另一种方法：使用 du 统计目录下的文件数量
	cmd := "for d in /var /tmp /home /opt; do [ -d \"$d\" ] && echo \"$d: $(find \"$d\" -type f 2>/dev/null | wc -l)\"; done 2>/dev/null"
	output, _ := ctx.SSH.Exec(host, cmd)

	simpleStats := make([]map[string]interface{}, 0)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					simpleStats = append(simpleStats, map[string]interface{}{
						"path":       strings.TrimSpace(parts[0]),
						"file_count": count,
					})
				}
			}
		}
	}

	return map[string]interface{}{
		"detailed":     directories,
		"simple_stats": simpleStats,
	}
}

// generateSummary 生成总体评估
func (t *InodeCheckTool) generateSummary(results []map[string]interface{}, threshold int, criticalCount int) map[string]interface{} {
	totalHosts := len(results)
	normalCount := 0
	warningCount := 0
	alertCount := 0

	for _, result := range results {
		if status, ok := result["status"].(string); ok {
			switch status {
			case "正常":
				normalCount++
			case "警告":
				warningCount++
			case "告警":
				alertCount++
			}
		}
	}

	overallStatus := "正常"
	if alertCount > 0 {
		overallStatus = "告警"
	} else if warningCount > 0 {
		overallStatus = "警告"
	}

	// 生成建议
	suggestions := make([]string, 0)
	if alertCount > 0 {
		suggestions = append(suggestions,
			fmt.Sprintf("有 %d 台主机 inode 使用率超过 %d%%，需要立即处理", alertCount, threshold),
			"查找并删除大量小文件或临时文件",
			"考虑使用更大的文件系统或增加 inode 数量",
			"检查是否有日志文件或邮件队列文件过多")
	} else if warningCount > 0 {
		suggestions = append(suggestions,
			fmt.Sprintf("有 %d 台主机 inode 使用率接近阈值，建议关注", warningCount),
			"定期清理临时文件和缓存",
			"监控 inode 使用趋势")
	} else {
		suggestions = append(suggestions, "所有主机 inode 使用正常，继续保持监控")
	}

	return map[string]interface{}{
		"total_hosts":    totalHosts,
		"normal_count":   normalCount,
		"warning_count":  warningCount,
		"alert_count":    alertCount,
		"overall_status": overallStatus,
		"suggestions":    suggestions,
	}
}
