package handler

import (
	"ai-ops/internal/ssh"

	"github.com/gin-gonic/gin"
)

// HostHandler 主机处理器
type HostHandler struct {
	sshPool *ssh.Pool
}

// NewHostHandler 创建主机处理器
func NewHostHandler(sshPool *ssh.Pool) *HostHandler {
	return &HostHandler{sshPool: sshPool}
}

// HostInfo 主机信息
type HostInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name" binding:"required"`
	Host     string   `json:"host" binding:"required"`
	Port     int      `json:"port"`
	User     string   `json:"user"`
	Group    string   `json:"group"`
	Tags     []string `json:"tags"`
	AuthType string   `json:"auth_type"` // password / key
	Password string   `json:"password,omitempty"`
	KeyPath  string   `json:"key_path,omitempty"`
	Status   string   `json:"status"`
}

// ListHosts 获取主机列表
// GET /api/hosts
func (h *HostHandler) ListHosts(c *gin.Context) {
	group := c.Query("group")
	keyword := c.Query("keyword")

	// 从 SSH Pool 获取主机列表
	// 这里简化处理，实际应该从数据库获取
	hosts := make([]HostInfo, 0)

	// TODO: 实现过滤逻辑
	_ = group
	_ = keyword

	Success(c, gin.H{
		"hosts": hosts,
		"total": len(hosts),
	})
}

// CreateHost 添加主机
// POST /api/hosts
func (h *HostHandler) CreateHost(c *gin.Context) {
	var req HostInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 设置默认值
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}
	if req.AuthType == "" {
		req.AuthType = "key"
	}

	// 添加到 SSH Pool
	h.sshPool.AddHost(ssh.HostInfo{
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		AuthType: req.AuthType,
		Password: req.Password,
		KeyPath:  req.KeyPath,
	})

	// TODO: 保存到数据库

	SuccessWithMessage(c, "添加成功", gin.H{
		"id":   req.Name, // 暂时用 name 作为 id
		"name": req.Name,
	})
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

	// 先移除旧的
	h.sshPool.RemoveHost(id)

	// 添加新的
	h.sshPool.AddHost(ssh.HostInfo{
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		AuthType: req.AuthType,
		Password: req.Password,
		KeyPath:  req.KeyPath,
	})

	// TODO: 更新数据库

	SuccessWithMessage(c, "更新成功", nil)
}

// DeleteHost 删除主机
// DELETE /api/hosts/:id
func (h *HostHandler) DeleteHost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	h.sshPool.RemoveHost(id)

	// TODO: 从数据库删除

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

	info, ok := h.sshPool.GetHost(id)
	if !ok {
		NotFound(c, "主机不存在")
		return
	}

	Success(c, HostInfo{
		ID:       info.Name,
		Name:     info.Name,
		Host:     info.Host,
		Port:     info.Port,
		User:     info.User,
		AuthType: info.AuthType,
		KeyPath:  info.KeyPath,
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

	// 执行简单命令测试连接
	output, err := h.sshPool.Exec(id, "echo 'connection test'")
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
		Format string `json:"format"` // csv / json
		Data   string `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// TODO: 实现导入逻辑
	SuccessWithMessage(c, "导入功能开发中", nil)
}

// GetGroups 获取分组列表
// GET /api/groups
func (h *HostHandler) GetGroups(c *gin.Context) {
	// TODO: 从数据库获取分组
	Success(c, gin.H{
		"groups": []interface{}{},
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

	// TODO: 保存到数据库
	SuccessWithMessage(c, "创建成功", gin.H{
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

	// TODO: 从数据库删除
	SuccessWithMessage(c, "删除成功", nil)
}
