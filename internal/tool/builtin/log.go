package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// QueryLogTool 日志查询工具
type QueryLogTool struct{}

// Name 工具名称
func (t *QueryLogTool) Name() string {
	return "query_log"
}

// Description 工具描述
func (t *QueryLogTool) Description() string {
	return `查询指定节点的日志文件。
支持的日志类型: nginx, app, system, custom
可以按关键词过滤，指定查看行数。
适用场景: 排查错误、查看访问记录、分析异常。
当用户询问"查看日志"、"日志报错"、"错误日志"等问题时使用。`
}

// Parameters 参数定义
func (t *QueryLogTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称",
			Required:    true,
		},
		{
			Name:        "log_type",
			Type:        "string",
			Description: "日志类型",
			Required:    true,
			Enum:        []interface{}{"nginx", "app", "system", "custom"},
		},
		{
			Name:        "lines",
			Type:        "int",
			Description: "查看行数",
			Required:    false,
			Default:     100,
		},
		{
			Name:        "keyword",
			Type:        "string",
			Description: "过滤关键词",
			Required:    false,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "自定义日志路径 (log_type=custom 时使用)",
			Required:    false,
		},
		{
			Name:        "level",
			Type:        "string",
			Description: "日志级别过滤: error, warn, info",
			Required:    false,
			Enum:        []interface{}{"error", "warn", "info"},
		},
	}
}

// Execute 执行工具
func (t *QueryLogTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	// 获取参数
	host := tool.GetStringParam(params, "host", "")
	logType := tool.GetStringParam(params, "log_type", "")
	lines := tool.GetIntParam(params, "lines", 100)
	keyword := tool.GetStringParam(params, "keyword", "")
	customPath := tool.GetStringParam(params, "path", "")
	level := tool.GetStringParam(params, "level", "")

	// 参数验证
	if host == "" {
		return tool.NewErrorResult("参数 host 不能为空"), nil
	}
	if logType == "" {
		return tool.NewErrorResult("参数 log_type 不能为空"), nil
	}

	// 确定日志路径
	logPath := t.getLogPath(logType, customPath)
	if logPath == "" {
		return tool.NewErrorResult("无法确定日志路径，请指定 path 参数"), nil
	}

	// 构建命令
	cmd := t.buildCommand(logPath, lines, keyword, level)

	// 检查 SSH 上下文
	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	// 执行命令
	output, err := ctx.SSH.Exec(host, cmd)
	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
	}

	// 处理空结果
	if strings.TrimSpace(output) == "" {
		return tool.NewResult("", fmt.Sprintf("未找到匹配的日志内容 (日志路径: %s)", logPath)), nil
	}

	return tool.NewResult(output, fmt.Sprintf("成功获取 %s 的 %s 日志 (%d 行)", host, logType, lines)), nil
}

// getLogPath 根据日志类型获取路径
func (t *QueryLogTool) getLogPath(logType string, customPath string) string {
	switch logType {
	case "nginx":
		return "/var/log/nginx/error.log"
	case "app":
		return "/var/log/app/app.log"
	case "system":
		return "/var/log/syslog"
	case "custom":
		return customPath
	default:
		return ""
	}
}

// buildCommand 构建查询命令
func (t *QueryLogTool) buildCommand(logPath string, lines int, keyword string, level string) string {
	// 基础命令
	var cmd string

	// 如果有关键词或级别过滤
	filters := make([]string, 0)
	if keyword != "" {
		filters = append(filters, keyword)
	}
	if level != "" {
		// 根据级别添加过滤
		switch level {
		case "error":
			filters = append(filters, "[Ee]rror\\|ERROR\\|[Ff]atal\\|FATAL")
		case "warn":
			filters = append(filters, "[Ww]arn\\|WARN\\|[Ee]rror\\|ERROR")
		case "info":
			filters = append(filters, "[Ii]nfo\\|INFO")
		}
	}

	if len(filters) > 0 {
		// 使用 grep 过滤
		grepPattern := strings.Join(filters, "\\|")
		cmd = fmt.Sprintf("grep -E '%s' %s | tail -%d", grepPattern, logPath, lines)
	} else {
		// 直接 tail
		cmd = fmt.Sprintf("tail -%d %s", lines, logPath)
	}

	return cmd
}

// NewQueryLogTool 创建日志查询工具
func NewQueryLogTool() *QueryLogTool {
	return &QueryLogTool{}
}
