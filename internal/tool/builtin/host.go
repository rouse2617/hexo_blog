package builtin

import (
	"ai-ops/internal/tool"
)

// ListHostsTool 主机列表工具
type ListHostsTool struct {
	// GetHostsFunc 获取主机列表的函数（由外部注入）
	GetHostsFunc func(group string) []HostBasicInfo
}

// HostBasicInfo 主机基本信息
type HostBasicInfo struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"user"`
	Group  string `json:"group"`
	Status string `json:"status"`
}

func (t *ListHostsTool) Name() string { return "list_hosts" }

func (t *ListHostsTool) Description() string {
	return `列出可用的主机节点。
可以按分组过滤，查看主机名称、IP、状态等信息。
当用户询问"有哪些主机"、"节点列表"、"服务器"时使用。`
}

func (t *ListHostsTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "group",
			Type:        "string",
			Description: "按分组过滤",
			Required:    false,
		},
	}
}

func (t *ListHostsTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	group := tool.GetStringParam(params, "group", "")

	// 优先使用上下文中的 SSH Pool（与 Host API 同源）
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

func NewListHostsTool(getHostsFunc func(group string) []HostBasicInfo) *ListHostsTool {
	return &ListHostsTool{
		GetHostsFunc: getHostsFunc,
	}
}
