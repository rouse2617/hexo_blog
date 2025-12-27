package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// CheckCPUTool CPU 检查工具
type CheckCPUTool struct{}

func (t *CheckCPUTool) Name() string { return "check_cpu" }

func (t *CheckCPUTool) Description() string {
	return `检查节点的 CPU 使用情况。
返回 CPU 使用率、负载等信息。
可以检查单个节点或所有节点。
当用户询问"CPU 使用率"、"负载高"、"性能问题"时使用。`
}

func (t *CheckCPUTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，不填则需要指定 hosts",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点",
			Required:    false,
		},
	}
}

func (t *CheckCPUTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	// CPU 检查命令：拆分执行，避免触发只读白名单的 unsafe_shell_syntax（如 && / ; 等）
	// 且避免使用管道（|），以便匹配默认白名单中的 top 规则。
	cpuCmd := `top -bn1`
	loadCmd := `uptime`

	if host != "" {
		// 单节点
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewCheckCPUTool() *CheckCPUTool { return &CheckCPUTool{} }

// CheckMemoryTool 内存检查工具
type CheckMemoryTool struct{}

func (t *CheckMemoryTool) Name() string { return "check_memory" }

func (t *CheckMemoryTool) Description() string {
	return `检查节点的内存使用情况。
返回内存总量、已用、可用、使用率等信息。
当用户询问"内存使用"、"内存不足"、"OOM"时使用。`
}

func (t *CheckMemoryTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点",
			Required:    false,
		},
	}
}

func (t *CheckMemoryTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	// 内存检查命令
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

func NewCheckMemoryTool() *CheckMemoryTool { return &CheckMemoryTool{} }

// CheckDiskTool 磁盘检查工具
type CheckDiskTool struct{}

func (t *CheckDiskTool) Name() string { return "check_disk" }

func (t *CheckDiskTool) Description() string {
	return `检查节点的磁盘使用情况。
返回各分区的总容量、已用空间、使用率。
适用场景：排查磁盘空间不足、日常巡检。
当用户询问"磁盘满了"、"空间不够"、"存储"时使用。`
}

func (t *CheckDiskTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点",
			Required:    false,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "指定检查的路径",
			Required:    false,
		},
	}
}

func (t *CheckDiskTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	path := tool.GetStringParam(params, "path", "")

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	// 磁盘检查命令
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

func NewCheckDiskTool() *CheckDiskTool { return &CheckDiskTool{} }
