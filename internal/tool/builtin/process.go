package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// CheckProcessTool 进程检查工具
type CheckProcessTool struct{}

func (t *CheckProcessTool) Name() string { return "check_process" }

func (t *CheckProcessTool) Description() string {
	return `查看节点的进程信息。
可以按进程名过滤，查看进程状态、CPU、内存占用。
当用户询问"进程状态"、"服务是否运行"、"哪个进程占用高"时使用。`
}

func (t *CheckProcessTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称",
			Required:    true,
		},
		{
			Name:        "name",
			Type:        "string",
			Description: "进程名称过滤（支持模糊匹配）",
			Required:    false,
		},
		{
			Name:        "top",
			Type:        "int",
			Description: "显示资源占用最高的 N 个进程",
			Required:    false,
			Default:     10,
		},
		{
			Name:        "sort_by",
			Type:        "string",
			Description: "排序方式: cpu, memory",
			Required:    false,
			Default:     "cpu",
			Enum:        []interface{}{"cpu", "memory"},
		},
	}
}

func (t *CheckProcessTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	name := tool.GetStringParam(params, "name", "")
	top := tool.GetIntParam(params, "top", 10)
	sortBy := tool.GetStringParam(params, "sort_by", "cpu")

	if host == "" {
		return tool.NewErrorResult("参数 host 不能为空"), nil
	}

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	var cmd string
	if name != "" {
		// 按进程名过滤
		cmd = fmt.Sprintf("ps aux | grep -i '%s' | grep -v grep | head -%d", name, top)
	} else {
		// 按资源占用排序
		sortFlag := "-%cpu"
		if sortBy == "memory" {
			sortFlag = "-%mem"
		}
		cmd = fmt.Sprintf("ps aux --sort=%s | head -%d", sortFlag, top+1) // +1 包含标题行
	}

	output, err := ctx.SSH.Exec(host, cmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
	}

	if strings.TrimSpace(output) == "" {
		if name != "" {
			return tool.NewResult("", fmt.Sprintf("未找到名为 '%s' 的进程", name)), nil
		}
		return tool.NewResult("", "未获取到进程信息"), nil
	}

	return tool.NewResult(map[string]interface{}{
		"host":   host,
		"output": strings.TrimSpace(output),
	}, fmt.Sprintf("成功获取 %s 的进程信息", host)), nil
}

func NewCheckProcessTool() *CheckProcessTool { return &CheckProcessTool{} }

// RunCommandTool 命令执行工具
type RunCommandTool struct {
	// 危险命令列表
	dangerousCommands []string
}

func (t *RunCommandTool) Name() string { return "run_command" }

func (t *RunCommandTool) Description() string {
	return `在指定节点上执行 Shell 命令。
这是一个通用工具，当其他专用工具无法满足需求时使用。
注意：危险命令会被拦截。
可以执行单个节点或批量执行多个节点。
适用场景：执行自定义命令、查看特定信息、批量操作。`
}

func (t *RunCommandTool) Parameters() []tool.Parameter {
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
			Description: "多个目标节点名称",
			Required:    false,
		},
		{
			Name:        "command",
			Type:        "string",
			Description: "要执行的 Shell 命令",
			Required:    true,
		},
	}
}

func (t *RunCommandTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	command := tool.GetStringParam(params, "command", "")

	if host == "" && len(hosts) == 0 {
		return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
	}
	if command == "" {
		return tool.NewErrorResult("参数 command 不能为空"), nil
	}

	// 检查危险命令
	if err := t.checkDangerousCommand(command); err != nil {
		return tool.NewErrorResult(err.Error()), nil
	}

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	if host != "" {
		// 单节点执行
		output, err := ctx.SSH.Exec(host, command)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}

		return tool.NewResult(map[string]interface{}{
			"host":    host,
			"command": command,
			"output":  strings.TrimSpace(output),
		}, fmt.Sprintf("成功在 %s 上执行命令", host)), nil
	}

	if len(hosts) > 0 {
		// 多节点批量执行
		results := ctx.SSH.BatchExec(hosts, command)
		data := make([]map[string]interface{}, 0, len(results))
		for _, result := range results {
			item := map[string]interface{}{
				"host":    result.Host,
				"command": command,
			}
			if result.Error != nil {
				item["error"] = result.Error.Error()
				item["output"] = ""
			} else {
				item["output"] = strings.TrimSpace(result.Output)
			}
			item["elapsed"] = result.Elapsed.String()
			data = append(data, item)
		}

		return tool.NewResult(data, fmt.Sprintf("成功在 %d 个节点上执行命令", len(hosts))), nil
	}

	return tool.NewErrorResult("未指定执行节点"), nil
}

// checkDangerousCommand 检查危险命令
func (t *RunCommandTool) checkDangerousCommand(cmd string) error {
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=",
		"> /dev/sd",
		"shutdown",
		"reboot",
		"init 0",
		"init 6",
		":(){ :|:& };:", // fork bomb
		"chmod -R 777 /",
		"chown -R",
		"> /etc/passwd",
		"> /etc/shadow",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, strings.ToLower(pattern)) {
			return fmt.Errorf("危险命令被拦截: 包含 '%s'", pattern)
		}
	}

	return nil
}

func NewRunCommandTool() *RunCommandTool {
	return &RunCommandTool{}
}
