package handler

import (
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
)

// ToolHandler 工具处理器
type ToolHandler struct {
	registry *tool.Registry
	sshPool  *ssh.Pool
}

// NewToolHandler 创建工具处理器
func NewToolHandler(registry *tool.Registry, sshPool *ssh.Pool) *ToolHandler {
	return &ToolHandler{
		registry: registry,
		sshPool:  sshPool,
	}
}

// ListTools 获取工具列表
// GET /api/tools
func (h *ToolHandler) ListTools(c *gin.Context) {
	tools := h.registry.ListInfo()

	Success(c, gin.H{
		"tools": tools,
		"total": len(tools),
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
		Params map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 创建执行上下文
	ctx := &tool.Context{
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
