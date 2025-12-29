package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// CheckProcessToolEnhanced 增强版进程检查工具
type CheckProcessToolEnhanced struct{}

func (t *CheckProcessToolEnhanced) Name() string { return "check_process" }

func (t *CheckProcessToolEnhanced) Description() string {
	return `# 检查进程信息

## 功能说明
查看指定主机的进程信息，可以按进程名过滤，或按资源占用排序。

## 适用场景
- 用户询问"进程状态"、"服务是否运行"
- 查找占用资源高的进程
- 检查特定服务是否启动
- 性能问题排查
- 杀死僵尸进程前查看进程状态

## 输出数据解读
- USER: 进程所属用户
- PID: 进程 ID
- %CPU: CPU 使用率
- %MEM: 内存使用率
- VSZ: 虚拟内存大小
- RSS: 常驻内存大小
- COMMAND: 启动命令

## 排序方式
- cpu: 按 CPU 使用率降序排列（默认）
- memory: 按内存使用率降序排列

## 使用建议
- 必须指定 host 参数（不支持批量）
- 使用 name 参数按进程名模糊匹配
- 使用 top 参数显示前 N 个进程（默认 10）
- 发现资源占用高时，可以结合其他工具进一步诊断
- 查看 nginx/mysql 等服务进程时使用 name 过滤
- 不确定进程名时先列出所有进程`
}

func (t *CheckProcessToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称（必填）",
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
			Description: "排序方式: cpu（默认）, memory",
			Required:    false,
			Default:     "cpu",
			Enum:        []interface{}{"cpu", "memory"},
		},
	}
}

func (t *CheckProcessToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
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
		cmd = fmt.Sprintf("ps aux | grep -i '%s' | grep -v grep | head -%d", name, top)
	} else {
		sortFlag := "-%cpu"
		if sortBy == "memory" {
			sortFlag = "-%mem"
		}
		cmd = fmt.Sprintf("ps aux --sort=%s | head -%d", sortFlag, top+1)
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

func NewCheckProcessToolEnhanced() *CheckProcessToolEnhanced { return &CheckProcessToolEnhanced{} }

// RunCommandToolEnhanced 增强版命令执行工具
type RunCommandToolEnhanced struct{}

func (t *RunCommandToolEnhanced) Name() string { return "run_command" }

func (t *RunCommandToolEnhanced) Description() string {
	return `# 执行 Shell 命令

## 功能说明
在指定节点上执行任意 Shell 命令，这是一个通用工具。

## 适用场景
- 专用工具无法满足需求时使用
- 执行自定义脚本
- 查看特定文件内容
- 执行特殊命令或组合命令

## 注意事项
- **危险命令会被拦截**：rm -rf /, mkfs, shutdown, reboot 等
- 优先使用专用工具（check_*）而不是 run_command
- 批量操作使用 hosts 参数
- 复杂命令建议使用单引号包裹

## 常见使用场景
- 查看文件内容：run_command(host="node1", command="cat /etc/hosts")
- 统计日志：run_command(host="node1", command="wc -l /var/log/app.log")
- 查找大文件：run_command(host="node1", command="du -sh /var/* | sort -hr")
- 测试网络：run_command(host="node1", command="ping -c 3 google.com")

## 安全提醒
- 以下操作会被拦截：删除系统文件、格式化磁盘、重启系统
- 建议先使用只读命令查看信息
- 修改操作前先确认影响范围`
}

func (t *RunCommandToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，单主机执行时使用",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点名称，批量执行时使用",
			Required:    false,
		},
		{
			Name:        "command",
			Type:        "string",
			Description: "要执行的 Shell 命令（必填）",
			Required:    true,
		},
	}
}

func (t *RunCommandToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	command := tool.GetStringParam(params, "command", "")

	if host == "" && len(hosts) == 0 {
		return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
	}
	if command == "" {
		return tool.NewErrorResult("参数 command 不能为空"), nil
	}

	if err := t.checkDangerousCommand(command); err != nil {
		return tool.NewErrorResult(err.Error()), nil
	}

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	if host != "" {
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

func (t *RunCommandToolEnhanced) checkDangerousCommand(cmd string) error {
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
		":(){ :|:& };:",
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

func NewRunCommandToolEnhanced() *RunCommandToolEnhanced { return &RunCommandToolEnhanced{} }
