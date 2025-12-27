package handler

import (
	"ai-ops/internal/repository"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ToolHandler 工具处理器
type ToolHandler struct {
	registry   *tool.Registry
	sshPool    *ssh.Pool
	configRepo repository.ConfigRepository
}

// NewToolHandler 创建工具处理器
func NewToolHandler(registry *tool.Registry, sshPool *ssh.Pool, configRepo repository.ConfigRepository) *ToolHandler {
	handler := &ToolHandler{
		registry:   registry,
		sshPool:    sshPool,
		configRepo: configRepo,
	}

	// 设置启用状态检查函数
	registry.SetEnableChecker(handler.isToolEnabled)

	return handler
}

// isToolEnabled 检查工具是否启用（从配置读取）
func (h *ToolHandler) isToolEnabled(name string) bool {
	config, err := h.configRepo.Get("tool." + name + ".enabled")
	if err != nil || config == nil {
		// 默认启用
		return true
	}
	return config.Value == "true"
}

// ListTools 获取工具列表
// GET /api/tools
func (h *ToolHandler) ListTools(c *gin.Context) {
	infos := h.registry.ListInfo()

	// 转换为前端期望的格式，添加 enabled 字段
	tools := make([]gin.H, 0, len(infos))
	for _, info := range infos {
		tools = append(tools, gin.H{
			"name":        info.Name,
			"description": info.Description,
			"type":        info.Type,
			"parameters":  info.Parameters,
			"enabled":     h.isToolEnabled(info.Name),
		})
	}

	Success(c, gin.H{
		"total": len(tools),
		"tools": tools,
	})
}

// ListBuiltinTools 获取内置工具列表
// GET /api/tools/builtin
func (h *ToolHandler) ListBuiltinTools(c *gin.Context) {
	infos := h.registry.ListInfo()

	tools := make([]gin.H, 0)
	for _, info := range infos {
		if info.Type == "builtin" {
			tools = append(tools, gin.H{
				"name":        info.Name,
				"description": info.Description,
				"type":        info.Type,
				"parameters":  info.Parameters,
				"enabled":     h.isToolEnabled(info.Name),
			})
		}
	}

	Success(c, tools)
}

// ListScriptTools 获取脚本工具列表
// GET /api/tools/script
func (h *ToolHandler) ListScriptTools(c *gin.Context) {
	infos := h.registry.ListInfo()

	tools := make([]gin.H, 0)
	for _, info := range infos {
		if info.Type == "script" {
			tools = append(tools, gin.H{
				"name":        info.Name,
				"description": info.Description,
				"type":        info.Type,
				"parameters":  info.Parameters,
				"enabled":     h.isToolEnabled(info.Name),
			})
		}
	}

	Success(c, tools)
}

// ToggleTool 启用/禁用工具
// PUT /api/tools/:name/toggle
func (h *ToolHandler) ToggleTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		ParamError(c, "name 不能为空")
		return
	}

	// 检查工具是否存在
	_, ok := h.registry.Get(name)
	if !ok {
		NotFound(c, "工具不存在")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 保存工具的启用状态到配置
	configKey := "tool." + name + ".enabled"
	value := strconv.FormatBool(req.Enabled)
	if err := h.configRepo.Set(configKey, value); err != nil {
		InternalError(c, "保存工具状态失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "操作成功", gin.H{
		"name":    name,
		"enabled": req.Enabled,
	})
}

// GetTool 获取单个工具信息
// GET /api/tools/:name
func (h *ToolHandler) GetTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		ParamError(c, "name 不能为空")
		return
	}

	t, ok := h.registry.Get(name)
	if !ok {
		NotFound(c, "工具不存在")
		return
	}

	Success(c, tool.ToolInfo{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	})
}

// ExecuteTool 手动执行工具
// POST /api/tools/:name/execute
func (h *ToolHandler) ExecuteTool(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		ParamError(c, "name 不能为空")
		return
	}

	var req struct {
		Params   map[string]interface{} `json:"params"`
		Hosts    []string               `json:"hosts"`
		HostIDs  []string               `json:"host_ids"`
		HostIDs2 []string               `json:"hostIds"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 创建执行上下文
	ctx := &tool.Context{
		Hosts: func() []string {
			if len(req.Hosts) > 0 {
				return req.Hosts
			}
			if len(req.HostIDs) > 0 {
				return req.HostIDs
			}
			return req.HostIDs2
		}(),
		SSH: h.sshPool,
	}

	// 执行工具
	result, err := h.registry.Execute(ctx, name, req.Params)
	if err != nil {
		ExecError(c, "执行失败: "+err.Error())
		return
	}

	if !result.Success {
		ExecError(c, result.Error)
		return
	}

	Success(c, gin.H{
		"success": result.Success,
		"message": result.Message,
		"data":    result.Data,
	})
}
