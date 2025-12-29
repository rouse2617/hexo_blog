package builtin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-ops/internal/tool"
)

// PerformanceAnalysisTool 性能分析工具
type PerformanceAnalysisTool struct{}

// NewPerformanceAnalysisTool 创建性能分析工具
func NewPerformanceAnalysisTool() *PerformanceAnalysisTool {
	return &PerformanceAnalysisTool{}
}

// Name 工具名称
func (t *PerformanceAnalysisTool) Name() string {
	return "analyze_performance"
}

// Description 工具描述
func (t *PerformanceAnalysisTool) Description() string {
	return `# 综合性能分析与瓶颈定位

## 功能说明
对指定主机进行全面性能分析，自动识别性能瓶颈类型并给出优化建议。

## 分析维度
1. **CPU 分析** - 使用率、负载、上下文切换、iowait
2. **内存分析** - 物理内存、Swap、缓存、内存压力
3. **I/O 分析** - 磁盘读写、IOPS、队列深度、延迟
4. **网络分析** - 网卡流量、连接数、TCP 状态
5. **进程 Top N** - CPU/内存占用最高的进程

## 输出内容
- 性能评分（0-100）
- 瓶颈类型判定
- 具体指标数据
- 优化建议清单
- 风险等级评估`
}

// Parameters 工具参数
func (t *PerformanceAnalysisTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标主机名称",
			Required:    true,
		},
		{
			Name:        "duration",
			Type:        "integer",
			Description: "采集时长（秒），默认 5 秒",
			Required:    false,
			Default:     5,
		},
		{
			Name:        "top_n",
			Type:        "integer",
			Description: "显示 Top N 进程",
			Required:    false,
			Default:     10,
		},
	}
}

// Execute 执行工具
func (t *PerformanceAnalysisTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	duration := tool.GetIntParam(params, "duration", 5)
	topN := tool.GetIntParam(params, "top_n", 10)

	if host == "" {
		return tool.NewErrorResult("host 参数必填"), nil
	}

	// 执行综合分析
	diagnosis := t.performAnalysis(ctx, host, duration, topN)

	return tool.NewResult(diagnosis, fmt.Sprintf("完成 %s 的性能分析", host)), nil
}

// performAnalysis 执行分析
func (t *PerformanceAnalysisTool) performAnalysis(ctx *tool.Context, host string, duration, topN int) map[string]interface{} {
	var wg sync.WaitGroup
	var cpuStats, memStats, ioStats, netStats, processStats string
	var mu sync.Mutex

	wg.Add(5)

	// 并发采集
	go func() {
		defer wg.Done()
		cmd := "echo '=== CPU ===' && top -bn1 | grep 'Cpu(s)' && uptime && nproc"
		result, _ := ctx.SSH.Exec(host, cmd)
		mu.Lock()
		cpuStats = result
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		cmd := "free -h"
		result, _ := ctx.SSH.Exec(host, cmd)
		mu.Lock()
		memStats = result
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		cmd := "iostat -x 1 2 2>/dev/null || echo 'iostat 未安装，请安装 sysstat 包'"
		result, _ := ctx.SSH.Exec(host, cmd)
		mu.Lock()
		ioStats = result
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		cmd := "cat /proc/net/dev | grep -v 'Inter\\|face' | head -5"
		result, _ := ctx.SSH.Exec(host, cmd)
		mu.Lock()
		netStats = result
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		cmd := fmt.Sprintf("ps aux --sort=-%%cpu | head -%d && echo '=== MEMORY ===' && ps aux --sort=-%%mem | head -%d", topN+1, topN+1)
		result, _ := ctx.SSH.Exec(host, cmd)
		mu.Lock()
		processStats = result
		mu.Unlock()
	}()

	wg.Wait()

	// 计算性能评分和瓶颈
	score := t.calculateScore(cpuStats, memStats, ioStats)
	bottleneck := t.identifyBottleneck(cpuStats, memStats, ioStats)
	suggestions := t.generateSuggestions(bottleneck)
	riskLevel := t.getRiskLevel(score)

	return map[string]interface{}{
		"host":        host,
		"score":       score,
		"bottleneck":  bottleneck,
		"suggestions": suggestions,
		"risk_level":  riskLevel,
		"cpu":         cpuStats,
		"memory":      memStats,
		"io":          ioStats,
		"network":     netStats,
		"processes":   processStats,
		"timestamp":   time.Now().Format(time.RFC3339),
	}
}

// calculateScore 计算性能评分
func (t *PerformanceAnalysisTool) calculateScore(cpu, mem, io string) int {
	score := 100

	// CPU 评分
	cpuUsage := t.parseCPUUsage(cpu)
	if cpuUsage > 90 {
		score -= 30
	} else if cpuUsage > 70 {
		score -= 15
	} else if cpuUsage > 50 {
		score -= 5
	}

	// 内存评分
	memUsage := t.parseMemoryUsage(mem)
	if memUsage > 90 {
		score -= 30
	} else if memUsage > 80 {
		score -= 15
	} else if memUsage > 60 {
		score -= 5
	}

	// I/O 评分
	if strings.Contains(strings.ToLower(io), "iowait") {
		iowait := t.parseIOWait(io)
		if iowait > 20 {
			score -= 25
		} else if iowait > 10 {
			score -= 10
		}
	}

	// 确保评分在 0-100 范围内
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// parseCPUUsage 解析 CPU 使用率
func (t *PerformanceAnalysisTool) parseCPUUsage(cpuOutput string) float64 {
	// 解析 top 命令输出的 CPU 使用率
	// 格式: %Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.5 id,  0.2 wa,  0.0 hi,  0.0 si,  0.0 st
	lines := strings.Split(cpuOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s)") || strings.Contains(line, "%Cpu") {
			// 提取 us (user) 和 sy (system) 的和
			re := regexp.MustCompile(`(\d+\.?\d*)\s*us`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				if us, err := strconv.ParseFloat(matches[1], 64); err == nil {
					re = regexp.MustCompile(`(\d+\.?\d*)\s*sy`)
					if matches := re.FindStringSubmatch(line); len(matches) > 1 {
						if sy, err := strconv.ParseFloat(matches[1], 64); err == nil {
							return us + sy
						}
					}
					return us
				}
			}
		}
	}
	return 0
}

// parseMemoryUsage 解析内存使用率
func (t *PerformanceAnalysisTool) parseMemoryUsage(memOutput string) float64 {
	// 解析 free 命令输出
	lines := strings.Split(memOutput, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				// 格式: Mem: total used free
				total, err1 := strconv.ParseFloat(strings.TrimSuffix(fields[1], "G"), 64)
				used, err2 := strconv.ParseFloat(strings.TrimSuffix(fields[2], "G"), 64)
				if err1 == nil && err2 == nil && total > 0 {
					return (used / total) * 100
				}
			}
		}
	}
	return 0
}

// parseIOWait 解析 IO 等待
func (t *PerformanceAnalysisTool) parseIOWait(ioOutput string) float64 {
	// 从 iostat 或 top 输出中解析 iowait
	re := regexp.MustCompile(`(\d+\.?\d*)\s*%iowait`)
	if matches := re.FindStringSubmatch(ioOutput); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	// 尝试从 top 输出解析 wa (wait)
	re = regexp.MustCompile(`(\d+\.?\d*)\s*wa`)
	if matches := re.FindStringSubmatch(ioOutput); len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return val
		}
	}

	return 0
}

// identifyBottleneck 识别瓶颈
func (t *PerformanceAnalysisTool) identifyBottleneck(cpu, mem, io string) string {
	iowait := t.parseIOWait(io)
	cpuUsage := t.parseCPUUsage(cpu)
	memUsage := t.parseMemoryUsage(mem)

	// 判断主要瓶颈
	if iowait > 10 {
		return "I/O密集型瓶颈"
	}
	if memUsage > 80 {
		return "内存瓶颈"
	}
	if cpuUsage > 70 {
		return "CPU密集型瓶颈"
	}

	return "系统运行正常"
}

// generateSuggestions 生成建议
func (t *PerformanceAnalysisTool) generateSuggestions(bottleneck string) []string {
	suggestions := []string{
		"定期监控系统资源使用情况",
		"设置告警阈值，及时发现问题",
	}

	switch bottleneck {
	case "I/O密集型瓶颈":
		suggestions = append(suggestions,
			"考虑使用 SSD 替代 HDD",
			"优化数据库查询，减少磁盘 I/O",
			"增加磁盘缓存或使用 Redis 等缓存系统",
			"检查是否有大量小文件读写操作",
			"考虑使用 RAID 0 或 RAID 10 提升性能")
	case "内存瓶颈":
		suggestions = append(suggestions,
			"增加物理内存",
			"优化应用内存使用，检查内存泄漏",
			"调整 Swap 分区大小",
			"使用内存缓存时注意控制大小",
			"考虑使用内存分析工具定位问题")
	case "CPU密集型瓶颈":
		suggestions = append(suggestions,
			"优化代码性能，减少不必要的计算",
			"增加 CPU 核心数或使用负载均衡",
			"检查是否有异常进程占用 CPU",
			"考虑使用异步处理或消息队列",
			"使用性能分析工具定位热点代码")
	default:
		suggestions = append(suggestions, "系统运行良好，继续保持监控")
	}

	return suggestions
}

// getRiskLevel 获取风险等级
func (t *PerformanceAnalysisTool) getRiskLevel(score int) string {
	if score >= 80 {
		return "低风险"
	} else if score >= 60 {
		return "中风险"
	} else {
		return "高风险"
	}
}
