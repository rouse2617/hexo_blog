package handler

import (
	"ai-ops/internal/ssh"
	"strings"

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
	PrivateKey  string   `json:"privateKey,omitempty"`  // 私钥内容（前端传入）
	KeyPath     string   `json:"key_path,omitempty"`    // 私钥路径
	Description string   `json:"description,omitempty"` // 描述
	Status      string   `json:"status"`
}

// HostResponse 返回给前端的主机信息（使用 username 字段）
type HostResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	User        string   `json:"user,omitempty"`
	Username    string   `json:"username"`
	Group       string   `json:"group,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	AuthType    string   `json:"auth_type,omitempty"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
}

// getUser 获取用户名（兼容 user 和 username 字段）
func (h *HostInfo) getUser() string {
	if h.User != "" {
		return h.User
	}
	if h.Username != "" {
		return h.Username
	}
	return "root"
}

// getAuthType 获取认证类型
func (h *HostInfo) getAuthType() string {
	if h.AuthType != "" {
		return h.AuthType
	}
	// 根据是否有密码或私钥自动判断
	if h.Password != "" {
		return "password"
	}
	if h.PrivateKey != "" {
		return "key_content"
	}
	return "key"
}

// ListHosts 获取主机列表
// GET /api/hosts
func (h *HostHandler) ListHosts(c *gin.Context) {
	group := c.Query("group")
	keyword := c.Query("keyword")

	// 从 SSH Pool 获取主机列表
	poolHosts := h.sshPool.ListHosts()
	hosts := make([]HostResponse, 0, len(poolHosts))

	for _, info := range poolHosts {
		// 过滤逻辑
		if group != "" && info.Group != group {
			continue
		}
		if keyword != "" && !containsKeyword(info, keyword) {
			continue
		}

		hosts = append(hosts, HostResponse{
			ID:       info.Name,
			Name:     info.Name,
			Host:     info.Host,
			Port:     info.Port,
			User:     info.User,
			Username: info.User,
			Group:    info.Group,
			Tags:     info.Tags,
			AuthType: info.AuthType,
			Status:   "unknown",
		})
	}

	Success(c, gin.H{
		"list":  hosts,
		"total": len(hosts),
	})
}

// GetAllHosts 获取所有主机（不分页，用于选择器）
// GET /api/hosts/all
func (h *HostHandler) GetAllHosts(c *gin.Context) {
	poolHosts := h.sshPool.ListHosts()
	hosts := make([]HostResponse, 0, len(poolHosts))

	for _, info := range poolHosts {
		hosts = append(hosts, HostResponse{
			ID:       info.Name,
			Name:     info.Name,
			Host:     info.Host,
			Port:     info.Port,
			User:     info.User,
			Username: info.User,
			Group:    info.Group,
			Tags:     info.Tags,
			Status:   "unknown",
		})
	}

	Success(c, hosts)
}

// containsKeyword 检查主机信息是否包含关键字
func containsKeyword(info ssh.HostInfo, keyword string) bool {
	return strings.Contains(info.Name, keyword) ||
		strings.Contains(info.Host, keyword) ||
		strings.Contains(info.User, keyword)
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

	user := req.getUser()
	authType := req.getAuthType()

	// 添加到 SSH Pool
	h.sshPool.AddHost(ssh.HostInfo{
		Name:       req.Name,
		Host:       req.Host,
		Port:       req.Port,
		User:       user,
		Group:      req.Group,
		Tags:       req.Tags,
		AuthType:   authType,
		Password:   req.Password,
		KeyPath:    req.KeyPath,
		KeyContent: req.PrivateKey,
	})

	// TODO: 保存到数据库

	// 返回完整的主机信息
	Success(c, HostResponse{
		ID:       req.Name,
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		User:     user,
		Username: user,
		Group:    req.Group,
		Tags:     req.Tags,
		AuthType: authType,
		Status:   "unknown",
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

	Success(c, HostResponse{
		ID:       info.Name,
		Name:     info.Name,
		Host:     info.Host,
		Port:     info.Port,
		User:     info.User,
		Username: info.User,
		Group:    info.Group,
		Tags:     info.Tags,
		AuthType: info.AuthType,
		Status:   "unknown",
	})
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

	for _, id := range req.IDs {
		h.sshPool.RemoveHost(id)
	}

	// TODO: 从数据库删除

	SuccessWithMessage(c, "删除成功", gin.H{
		"deleted": len(req.IDs),
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

	// 执行简单命令测试连接（命令需符合只读白名单策略）
	output, err := h.sshPool.Exec(id, "uptime")
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
