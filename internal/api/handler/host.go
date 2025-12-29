package handler

import (
	"ai-ops/internal/repository"
	"ai-ops/internal/service"

	"github.com/gin-gonic/gin"
)

// HostHandler 主机处理器
type HostHandler struct {
	hostService *service.HostService
}

// NewHostHandler 创建主机处理器
func NewHostHandler(hostService *service.HostService) *HostHandler {
	return &HostHandler{
		hostService: hostService,
	}
}

// HostInfo 主机信息（请求）
type HostInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name" binding:"required"`
	Host        string   `json:"host" binding:"required"`
	Port        int      `json:"port"`
	User        string   `json:"user"`
	Username    string   `json:"username"` // 前端字段兼容
	Group       string   `json:"group"`
	Tags        []string `json:"tags"`
	AuthType    string   `json:"auth_type"` // password / key
	Password    string   `json:"password,omitempty"`
	PrivateKey  string   `json:"privateKey,omitempty"` // 私钥内容（前端传入）
	KeyPath     string   `json:"key_path,omitempty"`   // 私钥路径
	Description string   `json:"description,omitempty"` // 描述
	Status      string   `json:"status"`
}

// ListHosts 获取主机列表
// GET /api/hosts
func (h *HostHandler) ListHosts(c *gin.Context) {
	group := c.Query("group")
	keyword := c.Query("keyword")

	// 调用 Service 层
	hosts, err := h.hostService.ListHosts(repository.HostFilter{
		Group:   group,
		Keyword: keyword,
	})
	if err != nil {
		InternalError(c, "获取主机列表失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"list":  hosts,
		"total": len(hosts),
	})
}

// GetAllHosts 获取所有主机（不分页，用于选择器）
// GET /api/hosts/all
func (h *HostHandler) GetAllHosts(c *gin.Context) {
	hosts, err := h.hostService.ListHosts(repository.HostFilter{})
	if err != nil {
		InternalError(c, "获取主机列表失败: "+err.Error())
		return
	}

	Success(c, hosts)
}

// CreateHost 添加主机
// POST /api/hosts
func (h *HostHandler) CreateHost(c *gin.Context) {
	var req HostInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 转换为 Service 层的 HostInfo
	info := &service.HostInfo{
		ID:          req.ID,
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		User:        req.User,
		Username:    req.Username,
		Group:       req.Group,
		Tags:        req.Tags,
		AuthType:    req.AuthType,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		KeyPath:     req.KeyPath,
		Description: req.Description,
	}

	host, err := h.hostService.CreateHost(info)
	if err != nil {
		ParamError(c, err.Error())
		return
	}

	Success(c, host)
}

// UpdateHost 更新主机
// PUT /api/hosts/:id
func (h *HostHandler) UpdateHost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	var req HostInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 转换为 Service 层的 HostInfo
	info := &service.HostInfo{
		ID:          id,
		Name:        req.Name,
		Host:        req.Host,
		Port:        req.Port,
		User:        req.User,
		Username:    req.Username,
		Group:       req.Group,
		Tags:        req.Tags,
		AuthType:    req.AuthType,
		Password:    req.Password,
		PrivateKey:  req.PrivateKey,
		KeyPath:     req.KeyPath,
		Description: req.Description,
	}

	host, err := h.hostService.UpdateHost(id, info)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "更新成功", host)
}

// DeleteHost 删除主机
// DELETE /api/hosts/:id
func (h *HostHandler) DeleteHost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	if err := h.hostService.DeleteHost(id); err != nil {
		InternalError(c, "删除主机失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", nil)
}

// GetHost 获取单个主机
// GET /api/hosts/:id
func (h *HostHandler) GetHost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	host, err := h.hostService.GetHost(id)
	if err != nil {
		NotFound(c, "主机不存在")
		return
	}

	Success(c, host)
}

// BatchDeleteHosts 批量删除主机
// POST /api/hosts/batch-delete
func (h *HostHandler) BatchDeleteHosts(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	deleted, err := h.hostService.BatchDeleteHosts(req.IDs)
	if err != nil {
		InternalError(c, "批量删除失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", gin.H{
		"deleted": deleted,
	})
}

// TestConnection 测试主机连接
// POST /api/hosts/:id/test
func (h *HostHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	output, err := h.hostService.TestConnection(id)
	if err != nil {
		SSHError(c, "连接失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"success": true,
		"message": "连接成功",
		"output":  output,
	})
}

// ImportHosts 批量导入主机
// POST /api/hosts/import
func (h *HostHandler) ImportHosts(c *gin.Context) {
	var req struct {
		Format string `json:"format" binding:"required"` // csv / json
		Data   string `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	imported, failed, err := h.hostService.ImportHosts(req.Format, req.Data)
	if err != nil {
		ParamError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"imported": imported,
		"failed":   failed,
		"total":    imported + failed,
	})
}

// GetGroups 获取分组列表
// GET /api/groups
func (h *HostHandler) GetGroups(c *gin.Context) {
	groups, err := h.hostService.GetGroups()
	if err != nil {
		InternalError(c, "获取分组列表失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"groups": groups,
	})
}

// CreateGroup 创建分组
// POST /api/groups
func (h *HostHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	groupID, err := h.hostService.CreateGroup(req.Name, req.Description)
	if err != nil {
		ParamError(c, err.Error())
		return
	}

	SuccessWithMessage(c, "创建成功", gin.H{
		"id":   groupID,
		"name": req.Name,
	})
}

// DeleteGroup 删除分组
// DELETE /api/groups/:name
func (h *HostHandler) DeleteGroup(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		ParamError(c, "name 不能为空")
		return
	}

	if err := h.hostService.DeleteGroup(name); err != nil {
		InternalError(c, "删除分组失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", nil)
}
