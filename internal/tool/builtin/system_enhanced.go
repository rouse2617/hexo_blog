package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// CheckCPUToolEnhanced 增强版 CPU 检查工具
type CheckCPUToolEnhanced struct{}

func (t *CheckCPUToolEnhanced) Name() string { return "check_cpu" }

func (t *CheckCPUToolEnhanced) Description() string {
	return `# 检查 CPU 使用情况和系统负载

## 功能说明
检查指定主机的 CPU 使用率、核心数、系统负载等信息。

## 适用场景
- 用户询问"CPU 使用率"、"负载情况"、"系统压力"
- 性能问题诊断：系统变慢、响应延迟
- 资源规划：评估当前资源使用情况
- 故障排查：判断是否存在 CPU 瓶颈
- 容量规划：评估是否需要扩容

## 输出数据解读
- CPU 使用率：用户进程 + 系统进程的占用比例
- Load Average：1/5/15 分钟平均负载
- CPU 核心数：通过逻辑处理器数量判断
- 进程列表：按 CPU 占用排序的进程信息

## 判断标准
- **正常**：CPU < 70%, Load < 核心数
- **警告**：CPU 70-85%, Load 接近核心数
- **危险**：CPU > 85%, Load > 核心数 * 1.5
- **严重**：CPU > 95%, Load > 核心数 * 2

## 使用建议
- 单主机检查：使用 host 参数
- 多主机检查：使用 hosts 参数批量执行（推荐）
- 配合 check_memory 一起使用可全面评估系统状态
- 发现 CPU 高时，继续用 check_process 找出占用进程
- Load Average 持续高于核心数说明存在 CPU 争用`
}

func (t *CheckCPUToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，单主机检查时使用",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点，批量检查时使用（推荐）",
			Required:    false,
		},
	}
}

func (t *CheckCPUToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	cpuCmd := `top -bn1`
	loadCmd := `uptime`

	if host != "" {
		cpuOut, err := ctx.SSH.Exec(host, cpuCmd)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}
		cpuOut = strings.Join(strings.Split(cpuOut, "\n")[:minInt(20, len(strings.Split(cpuOut, "\n")))], "\n")
		loadOut, err := ctx.SSH.Exec(host, loadCmd)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}
		return tool.NewResult(map[string]interface{}{
			"host":   host,
			"cpu":    strings.TrimSpace(cpuOut),
			"load":   strings.TrimSpace(loadOut),
		}, fmt.Sprintf("成功获取 %s 的 CPU 信息", host)), nil
	}

	if len(hosts) > 0 {
		data := make([]map[string]interface{}, 0, len(hosts))
		for _, h := range hosts {
			item := map[string]interface{}{
				"host": h,
			}
			cpuOut, err := ctx.SSH.Exec(h, cpuCmd)
			if err != nil {
				item["error"] = err.Error()
				data = append(data, item)
				continue
			}
			cpuOut = strings.Join(strings.Split(cpuOut, "\n")[:minInt(20, len(strings.Split(cpuOut, "\n")))], "\n")
			loadOut, err := ctx.SSH.Exec(h, loadCmd)
			if err != nil {
				item["error"] = err.Error()
				item["cpu"] = strings.TrimSpace(cpuOut)
				data = append(data, item)
				continue
			}
			item["cpu"] = strings.TrimSpace(cpuOut)
			item["load"] = strings.TrimSpace(loadOut)
			data = append(data, item)
		}
		return tool.NewResult(data, fmt.Sprintf("成功检查 %d 个节点的 CPU", len(hosts))), nil
	}

	return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
}

func NewCheckCPUToolEnhanced() *CheckCPUToolEnhanced { return &CheckCPUToolEnhanced{} }

// CheckMemoryToolEnhanced 增强版内存检查工具
type CheckMemoryToolEnhanced struct{}

func (t *CheckMemoryToolEnhanced) Name() string { return "check_memory" }

func (t *CheckMemoryToolEnhanced) Description() string {
	return `# 检查内存使用情况

## 功能说明
检查指定主机的内存总量、已用、可用、使用率等信息。

## 适用场景
- 用户询问"内存使用"、"内存不足"、"OOM"
- 性能问题诊断：系统变慢、频繁 swap
- 资源规划：评估内存容量
- 故障排查：判断是否存在内存泄漏

## 输出数据解读
- MemTotal: 物理内存总量
- MemUsed: 已用内存（包含缓存和缓冲）
- MemFree: 空闲内存
- SwapTotal: 交换分区总量
- SwapUsed: 已用交换分区
- Usage: 内存使用率百分比

## 判断标准
- **正常**：内存使用率 < 70%, Swap 使用 < 10%
- **警告**：内存使用率 70-85%, Swap 使用 10-30%
- **危险**：内存使用率 > 85%, Swap 使用 > 30%
- **严重**：内存使用率 > 95%, Swap 持续增长

## 使用建议
- 单主机检查：使用 host 参数
- 多主机检查：使用 hosts 参数批量执行（推荐）
- Linux 系统的缓存会占用内存，这是正常的
- 关注 Swap 使用率，Swap 使用高说明物理内存不足
- 发现内存不足时，用 check_process 找出占用高的进程
- 配合 check_cpu 一起使用可全面评估系统状态`
}

func (t *CheckMemoryToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，单主机检查时使用",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点，批量检查时使用（推荐）",
			Required:    false,
		},
	}
}

func (t *CheckMemoryToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	cmd := `free -h | awk 'NR==1{print $1,$2,$3,$4} NR==2{printf "Memory: Total=%s, Used=%s, Free=%s, Usage=%.1f%%\n", $2, $3, $4, $3/$2*100} NR==3{printf "Swap: Total=%s, Used=%s, Free=%s\n", $2, $3, $4}'`

	if host != "" {
		output, err := ctx.SSH.Exec(host, cmd)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}
		return tool.NewResult(map[string]interface{}{
			"host":   host,
			"output": strings.TrimSpace(output),
		}, fmt.Sprintf("成功获取 %s 的内存信息", host)), nil
	}

	if len(hosts) > 0 {
		results := ctx.SSH.BatchExec(hosts, cmd)
		data := make([]map[string]interface{}, len(results))
		for i, r := range results {
			item := map[string]interface{}{
				"host":   r.Host,
				"output": strings.TrimSpace(r.Output),
			}
			if r.Error != nil {
				item["error"] = r.Error.Error()
			}
			data[i] = item
		}
		return tool.NewResult(data, fmt.Sprintf("成功检查 %d 个节点的内存", len(hosts))), nil
	}

	return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
}

func NewCheckMemoryToolEnhanced() *CheckMemoryToolEnhanced { return &CheckMemoryToolEnhanced{} }

// CheckDiskToolEnhanced 增强版磁盘检查工具
type CheckDiskToolEnhanced struct{}

func (t *CheckDiskToolEnhanced) Name() string { return "check_disk" }

func (t *CheckDiskToolEnhanced) Description() string {
	return `# 检查磁盘使用情况

## 功能说明
检查指定主机的磁盘分区、总容量、已用空间、使用率等信息。

## 适用场景
- 用户询问"磁盘满了"、"空间不够"、"存储"
- 排查磁盘空间不足
- 日常巡检
- 容量规划
- 日志清理前检查

## 输出数据解读
- Filesystem: 文件系统路径
- Size: 总容量
- Used: 已用空间
- Avail: 可用空间
- Use%: 使用率百分比
- Mounted on: 挂载点

## 判断标准
- **正常**：磁盘使用率 < 70%
- **警告**：磁盘使用率 70-85%
- **危险**：磁盘使用率 > 85%
- **严重**：磁盘使用率 > 95%

## 常见挂载点
- /: 根分区，系统文件
- /var: 日志、缓存文件
- /home: 用户文件
- /tmp: 临时文件
- /data: 数据目录

## 使用建议
- 单主机检查：使用 host 参数
- 多主机检查：使用 hosts 参数批量执行（推荐）
- 默认检查所有分区（排除 tmpfs 和 loop）
- 使用 path 参数指定检查特定路径
- 发现 /var 分区满时，检查日志文件
- 发现 /home 分区满时，检查用户大文件
- 配合 run_command 的 du 命令找出大文件`
}

func (t *CheckDiskToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，单主机检查时使用",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点，批量检查时使用（推荐）",
			Required:    false,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "指定检查的路径（不填则检查所有分区）",
			Required:    false,
		},
	}
}

func (t *CheckDiskToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	path := tool.GetStringParam(params, "path", "")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	var cmd string
	if path != "" {
		cmd = fmt.Sprintf("df -h %s", path)
	} else {
		cmd = "df -h | grep -v tmpfs | grep -v loop"
	}

	if host != "" {
		output, err := ctx.SSH.Exec(host, cmd)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}
		return tool.NewResult(map[string]interface{}{
			"host":   host,
			"output": strings.TrimSpace(output),
		}, fmt.Sprintf("成功获取 %s 的磁盘信息", host)), nil
	}

	if len(hosts) > 0 {
		results := ctx.SSH.BatchExec(hosts, cmd)
		data := make([]map[string]interface{}, len(results))
		for i, r := range results {
			item := map[string]interface{}{
				"host":   r.Host,
				"output": strings.TrimSpace(r.Output),
			}
			if r.Error != nil {
				item["error"] = r.Error.Error()
			}
			data[i] = item
		}
		return tool.NewResult(data, fmt.Sprintf("成功检查 %d 个节点的磁盘", len(hosts))), nil
	}

	return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
}

func NewCheckDiskToolEnhanced() *CheckDiskToolEnhanced { return &CheckDiskToolEnhanced{} }
