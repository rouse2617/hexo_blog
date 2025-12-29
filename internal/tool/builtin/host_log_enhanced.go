package builtin

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// ListHostsToolEnhanced 增强版主机列表工具
type ListHostsToolEnhanced struct {
	GetHostsFunc func(group string) []HostBasicInfo
}

func (t *ListHostsToolEnhanced) Name() string { return "list_hosts" }

func (t *ListHostsToolEnhanced) Description() string {
	return `# 列出可用主机

## 功能说明
列出系统中配置的所有主机节点，可以按分组过滤。

## 适用场景
- 用户询问"有哪些主机"、"节点列表"、"服务器"
- 不确定主机名称时先查看
- 按分组查看特定类型的主机
- 了解当前可管理的节点范围

## 输出数据解读
- name: 主机名称（唯一标识）
- host: IP 地址或域名
- port: SSH 端口
- user: SSH 用户名
- group: 分组名称
- status: 主机状态

## 分组类型
- 常见分组：web, db, cache, mq 等
- 未分组的主机 group 为空

## 使用建议
- 不带参数调用时列出所有主机
- 使用 group 参数按分组过滤
- 执行操作前建议先确认主机名称
- 配合其他工具使用时，主机名称必须准确
- 新增主机后可能需要刷新列表`
}

func (t *ListHostsToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "group",
			Type:        "string",
			Description: "按分组过滤（不填则显示所有主机）",
			Required:    false,
		},
	}
}

func (t *ListHostsToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	group := tool.GetStringParam(params, "group", "")

	if ctx != nil && ctx.SSH != nil {
		poolHosts := ctx.SSH.ListHosts()
		hosts := make([]HostBasicInfo, 0, len(poolHosts))
		for _, h := range poolHosts {
			if group != "" && h.Group != group {
				continue
			}
			hosts = append(hosts, HostBasicInfo{
				Name:   h.Name,
				Host:   h.Host,
				Port:   h.Port,
				User:   h.User,
				Group:  h.Group,
				Status: "unknown",
			})
		}
		if len(hosts) == 0 {
			if group != "" {
				return tool.NewResult([]HostBasicInfo{}, "未找到该分组的主机"), nil
			}
			return tool.NewResult([]HostBasicInfo{}, "暂无可用主机"), nil
		}
		message := "主机列表"
		if group != "" {
			message = "分组 " + group + " 的主机列表"
		}
		return tool.NewResult(hosts, message), nil
	}

	if t.GetHostsFunc == nil {
		return tool.NewErrorResult("主机列表功能未配置"), nil
	}

	hosts := t.GetHostsFunc(group)

	if len(hosts) == 0 {
		if group != "" {
			return tool.NewResult([]HostBasicInfo{}, "未找到该分组的主机"), nil
		}
		return tool.NewResult([]HostBasicInfo{}, "暂无可用主机"), nil
	}

	message := "主机列表"
	if group != "" {
		message = "分组 " + group + " 的主机列表"
	}

	return tool.NewResult(hosts, message), nil
}

func NewListHostsToolEnhanced(getHostsFunc func(group string) []HostBasicInfo) *ListHostsToolEnhanced {
	return &ListHostsToolEnhanced{
		GetHostsFunc: getHostsFunc,
	}
}

// QueryLogToolEnhanced 增强版日志查询工具
type QueryLogToolEnhanced struct{}

func (t *QueryLogToolEnhanced) Name() string { return "query_log" }

func (t *QueryLogToolEnhanced) Description() string {
	return `# 查询日志文件

## 功能说明
查询指定节点的日志文件，支持按关键词、日志级别过滤。

## 适用场景
- 用户询问"查看日志"、"日志报错"、"错误日志"
- 排查应用错误
- 分析访问记录
- 查找特定异常
- 日常巡检

## 日志类型
- **nginx**: Nginx 访问/错误日志（默认 /var/log/nginx/error.log）
- **app**: 应用日志（默认 /var/log/app/app.log）
- **system**: 系统日志（默认 /var/log/syslog）
- **custom**: 自定义路径（需指定 path 参数）

## 过滤选项
- **keyword**: 关键词过滤（模糊匹配）
- **level**: 日志级别
  - error: 只显示错误日志
  - warn: 显示警告和错误
  - info: 显示所有级别
- **lines**: 显示行数（默认 100）

## 输出数据解读
- 匹配的日志行内容
- 空结果表示未找到匹配的日志

## 使用建议
- 单主机查询：使用 host 参数
- 多主机查询：使用 hosts 参数批量执行（推荐）
- 指定 log_type 为必需参数
- 使用 keyword 过滤特定内容
- 使用 level 过滤日志级别
- 自定义路径使用 log_type=custom 并指定 path
- 发现错误时继续分析错误上下文
- 日志量大时减少 lines 参数提高性能`
}

func (t *QueryLogToolEnhanced) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "目标节点名称，单主机查询时使用",
			Required:    false,
		},
		{
			Name:        "hosts",
			Type:        "[]string",
			Description: "多个目标节点，批量查询时使用（推荐）",
			Required:    false,
		},
		{
			Name:        "log_type",
			Type:        "string",
			Description: "日志类型：nginx, app, system, custom（必填）",
			Required:    true,
			Enum:        []interface{}{"nginx", "app", "system", "custom"},
		},
		{
			Name:        "lines",
			Type:        "int",
			Description: "查看行数（默认 100）",
			Required:    false,
			Default:     100,
		},
		{
			Name:        "keyword",
			Type:        "string",
			Description: "过滤关键词（模糊匹配）",
			Required:    false,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "自定义日志路径（log_type=custom 时使用）",
			Required:    false,
		},
		{
			Name:        "level",
			Type:        "string",
			Description: "日志级别过滤：error, warn, info",
			Required:    false,
			Enum:        []interface{}{"error", "warn", "info"},
		},
	}
}

func (t *QueryLogToolEnhanced) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	hosts := tool.GetStringSliceParam(params, "hosts")
	logType := tool.GetStringParam(params, "log_type", "")
	lines := tool.GetIntParam(params, "lines", 100)
	keyword := tool.GetStringParam(params, "keyword", "")
	customPath := tool.GetStringParam(params, "path", "")
	level := tool.GetStringParam(params, "level", "")

	if host == "" && len(hosts) == 0 {
		return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
	}
	if logType == "" {
		return tool.NewErrorResult("参数 log_type 不能为空"), nil
	}

	logPath := t.getLogPath(logType, customPath)
	if logPath == "" {
		return tool.NewErrorResult("无法确定日志路径，请指定 path 参数"), nil
	}

	cmd := t.buildCommand(logPath, lines, keyword, level)

	if ctx == nil || ctx.SSH == nil {
		return tool.NewErrorResult("SSH 上下文未初始化"), nil
	}

	if host != "" {
		output, err := ctx.SSH.Exec(host, cmd)
		if err != nil {
			return tool.NewErrorResult(fmt.Sprintf("执行失败: %v", err)), nil
		}

		if strings.TrimSpace(output) == "" {
			return tool.NewResult(map[string]interface{}{
				"host":    host,
				"log":     "",
				"path":    logPath,
				"message": "未找到匹配的日志内容",
			}, fmt.Sprintf("未找到匹配的日志内容 (日志路径: %s)", logPath)), nil
		}

		return tool.NewResult(map[string]interface{}{
			"host":  host,
			"log":   strings.TrimSpace(output),
			"path":  logPath,
			"lines": lines,
		}, fmt.Sprintf("成功获取 %s 的 %s 日志 (%d 行)", host, logType, lines)), nil
	}

	if len(hosts) > 0 {
		data := make([]map[string]interface{}, 0, len(hosts))
		for _, h := range hosts {
			item := map[string]interface{}{
				"host": h,
				"path": logPath,
			}
			output, err := ctx.SSH.Exec(h, cmd)
			if err != nil {
				item["error"] = err.Error()
				data = append(data, item)
				continue
			}

			if strings.TrimSpace(output) == "" {
				item["log"] = ""
				item["message"] = "未找到匹配的日志内容"
			} else {
				item["log"] = strings.TrimSpace(output)
				item["lines"] = lines
			}
			data = append(data, item)
		}
		return tool.NewResult(data, fmt.Sprintf("成功获取 %d 个节点的 %s 日志", len(hosts), logType)), nil
	}

	return tool.NewErrorResult("请指定 host 或 hosts 参数"), nil
}

func (t *QueryLogToolEnhanced) getLogPath(logType string, customPath string) string {
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

func (t *QueryLogToolEnhanced) buildCommand(logPath string, lines int, keyword string, level string) string {
	filters := make([]string, 0)
	if keyword != "" {
		filters = append(filters, keyword)
	}
	if level != "" {
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
		grepPattern := strings.Join(filters, "\\|")
		return fmt.Sprintf("grep -E '%s' %s | tail -%d", grepPattern, logPath, lines)
	}

	return fmt.Sprintf("tail -%d %s", lines, logPath)
}

func NewQueryLogToolEnhanced() *QueryLogToolEnhanced {
	return &QueryLogToolEnhanced{}
}
