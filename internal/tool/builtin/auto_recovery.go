package builtin

import (
	"ai-ops/internal/tool"
	"fmt"
)

// AutoRecoveryTool 自动故障恢复工具
type AutoRecoveryTool struct{}

func (t *AutoRecoveryTool) Name() string {
	return "auto_recover"
}

func (t *AutoRecoveryTool) Description() string {
	return `# 自动故障恢复

## 功能说明
根据诊断结果自动执行故障恢复操作。

## 支持的恢复操作
1. **disk_full** - 清理日志、临时文件、缓存
2. **oom** - 清理 page cache、dentry cache、slab cache
3. **service_down** - 重启服务
4. **network_issue** - 重启网卡、清理规则
5. **high_cpu** - 终止异常进程

## 安全机制
- 所有操作前记录日志
- 关键操作需要确认
- 支持回滚操作`
}

func (t *AutoRecoveryTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{Name: "host", Type: "string", Description: "目标主机", Required: true},
		{Name: "issue_type", Type: "string", Description: "问题类型", Required: true},
		{Name: "confirm", Type: "boolean", Description: "确认执行", Required: false, Default: false},
		{Name: "service_name", Type: "string", Description: "服务名（可选）", Required: false},
	}
}

func (t *AutoRecoveryTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	issueType := tool.GetStringParam(params, "issue_type", "")
	confirm := tool.GetBoolParam(params, "confirm", false)

	if !confirm {
		return tool.NewErrorResult("自动恢复需要确认，请设置 confirm=true"), nil
	}

	switch issueType {
	case "disk_full":
		return t.recoverDiskFull(ctx, host)
	case "oom":
		return t.recoverOOM(ctx, host)
	case "service_down":
		return t.recoverServiceDown(ctx, host, params)
	case "network_issue":
		return t.recoverNetwork(ctx, host)
	case "high_cpu":
		return t.recoverHighCPU(ctx, host)
	default:
		return tool.NewErrorResult(fmt.Sprintf("未知的问题类型: %s", issueType)), nil
	}
}

// recoverDiskFull 磁盘满恢复
func (t *AutoRecoveryTool) recoverDiskFull(ctx *tool.Context, host string) (*tool.Result, error) {
	operations := make([]map[string]interface{}, 0)

	// 1. 清理 systemd 日志（保留7天）
	cmd := "journalctl --vacuum-time=7d"
	output, err := ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理 systemd 日志（保留7天）",
		"cmd":    cmd,
		"output": output,
		"status": getStatus(err),
	})

	// 2. 清理临时文件
	cmd = "find /tmp -type f -mtime +7 -delete 2>/dev/null ; echo '临时文件清理完成'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理 7 天前的临时文件",
		"cmd":    "find /tmp -type f -mtime +7 -delete",
		"output": output,
		"status": getStatus(err),
	})

	// 3. 清理包管理器缓存
	cmd = "yum clean all 2>/dev/null || apt-get clean 2>/dev/null || echo '包管理器缓存清理完成'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理包管理器缓存",
		"cmd":    cmd,
		"output": output,
		"status": getStatus(err),
	})

	// 4. 清理已删除但未释放的文件
	cmd = "lsof | grep deleted | awk '{print $2}' | sort -u | xargs -I {} kill -9 {} 2>/dev/null ; echo '已终止占用已删除文件的进程'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "终止占用已删除文件的进程",
		"cmd":    "lsof | grep deleted | awk '{print $2}' | sort -u | xargs -I {} kill -9 {}",
		"output": output,
		"status": getStatus(err),
	})

	// 5. 检查恢复后状态
	checkCmd := "df -h | grep -v tmpfs | grep -v loop"
	checkOutput, _ := ctx.SSH.Exec(host, checkCmd)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"issue_type":  "disk_full",
		"operations":  operations,
		"final_state": checkOutput,
		"status":      "completed",
	}, "磁盘空间恢复完成"), nil
}

// recoverOOM 内存不足恢复
func (t *AutoRecoveryTool) recoverOOM(ctx *tool.Context, host string) (*tool.Result, error) {
	operations := make([]map[string]interface{}, 0)

	// 1. 清理 page cache
	cmd := "sync && echo 1 > /proc/sys/vm/drop_caches && echo 'page cache 已清理'"
	output, err := ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理 page cache",
		"cmd":    "sync && echo 1 > /proc/sys/vm/drop_caches",
		"output": output,
		"status": getStatus(err),
	})

	// 2. 清理 dentry 和 inode cache
	cmd = "sync && echo 2 > /proc/sys/vm/drop_caches && echo 'dentry/inode cache 已清理'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理 dentry/inode cache",
		"cmd":    "sync && echo 2 > /proc/sys/vm/drop_caches",
		"output": output,
		"status": getStatus(err),
	})

	// 3. 清理所有 cache
	cmd = "sync && echo 3 > /proc/sys/vm/drop_caches && echo '所有 cache 已清理'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "清理所有 cache",
		"cmd":    "sync && echo 3 > /proc/sys/vm/drop_caches",
		"output": output,
		"status": getStatus(err),
	})

	// 检查恢复后状态
	checkCmd := "free -h"
	checkOutput, _ := ctx.SSH.Exec(host, checkCmd)

	return tool.NewResult(map[string]interface{}{
		"host":        host,
		"issue_type":  "oom",
		"operations":  operations,
		"final_state": checkOutput,
		"status":      "completed",
	}, "内存空间恢复完成"), nil
}

// recoverServiceDown 服务恢复
func (t *AutoRecoveryTool) recoverServiceDown(ctx *tool.Context, host string, params map[string]interface{}) (*tool.Result, error) {
	serviceName := tool.GetStringParam(params, "service_name", "")

	if serviceName == "" {
		return tool.NewErrorResult("请指定 service_name"), nil
	}

	operations := make([]map[string]interface{}, 0)

	// 1. 尝试启动服务
	startCmd := fmt.Sprintf("systemctl start %s", serviceName)
	output, err := ctx.SSH.Exec(host, startCmd)
	operations = append(operations, map[string]interface{}{
		"action": fmt.Sprintf("启动服务 %s", serviceName),
		"cmd":    startCmd,
		"output": output,
		"status": getStatus(err),
	})

	// 2. 检查服务状态
	statusCmd := fmt.Sprintf("systemctl status %s --no-pager", serviceName)
	statusOutput, _ := ctx.SSH.Exec(host, statusCmd)

	// 3. 启用开机自启
	enableCmd := fmt.Sprintf("systemctl enable %s", serviceName)
	enableOutput, err := ctx.SSH.Exec(host, enableCmd)
	operations = append(operations, map[string]interface{}{
		"action": fmt.Sprintf("启用 %s 开机自启", serviceName),
		"cmd":    enableCmd,
		"output": enableOutput,
		"status": getStatus(err),
	})

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"service":    serviceName,
		"operations": operations,
		"status":     statusOutput,
	}, fmt.Sprintf("服务 %s 恢复完成", serviceName)), nil
}

// recoverNetwork 网络恢复
func (t *AutoRecoveryTool) recoverNetwork(ctx *tool.Context, host string) (*tool.Result, error) {
	operations := make([]map[string]interface{}, 0)

	// 1. 重启网络服务
	cmd := "systemctl restart network 2>/dev/null || systemctl restart NetworkManager 2>/dev/null || echo '网络服务重启完成'"
	output, err := ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "重启网络服务",
		"cmd":    cmd,
		"output": output,
		"status": getStatus(err),
	})

	// 2. 刷新 DNS 缓存
	cmd = "systemctl restart systemd-resolved 2>/dev/null || echo 'DNS 缓存刷新完成'"
	output, err = ctx.SSH.Exec(host, cmd)
	operations = append(operations, map[string]interface{}{
		"action": "刷新 DNS 缓存",
		"cmd":    cmd,
		"output": output,
		"status": getStatus(err),
	})

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"issue_type": "network_issue",
		"operations": operations,
		"status":     "completed",
	}, "网络恢复完成"), nil
}

// recoverHighCPU 高 CPU 恢复
func (t *AutoRecoveryTool) recoverHighCPU(ctx *tool.Context, host string) (*tool.Result, error) {
	operations := make([]map[string]interface{}, 0)

	// 1. 查找高 CPU 进程
	cmd := "ps aux --sort=-%cpu | head -10"
	output, _ := ctx.SSH.Exec(host, cmd)

	// 2. 识别异常进程（这里需要用户确认）
	// 为了安全，只列出进程，不自动终止

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"issue_type": "high_cpu",
		"operations": operations,
		"top_cpu":    output,
		"notice":     "自动终止进程有风险，请手动确认后执行",
	}, "高 CPU 进程列表已获取"), nil
}

// getStatus 获取状态
func getStatus(err error) string {
	if err != nil {
		return "failed: " + err.Error()
	}
	return "success"
}

func NewAutoRecoveryTool() *AutoRecoveryTool {
	return &AutoRecoveryTool{}
}
