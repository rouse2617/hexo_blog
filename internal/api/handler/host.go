package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/model"
	"ai-ops/internal/repository"
	"ai-ops/internal/ssh"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HostHandler 主机处理器
type HostHandler struct {
	sshPool   *ssh.Pool
	hostRepo  repository.HostRepository
	groupRepo repository.GroupRepository
}

// NewHostHandler 创建主机处理器
func NewHostHandler(sshPool *ssh.Pool, hostRepo repository.HostRepository, groupRepo repository.GroupRepository) *HostHandler {
	return &HostHandler{
		sshPool:   sshPool,
		hostRepo:  hostRepo,
		groupRepo: groupRepo,
	}
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
	// 默认使用 auto 模式，自动尝试所有认证方式
	return "auto"
}

// ListHosts 获取主机列表
// GET /api/hosts
func (h *HostHandler) ListHosts(c *gin.Context) {
	group := c.Query("group")
	keyword := c.Query("keyword")

	// 从数据库获取主机列表
	filter := repository.HostFilter{
		Group:   group,
		Keyword: keyword,
	}
	dbHosts, err := h.hostRepo.List(filter)
	if err != nil {
		InternalError(c, "获取主机列表失败: "+err.Error())
		return
	}

	// 同步到SSH Pool
	for _, dbHost := range dbHosts {
		h.sshPool.AddHost(ssh.HostInfo{
			Name:       dbHost.Name,
			Host:       dbHost.IP,
			Port:       dbHost.Port,
			User:       dbHost.User,
			Group:      dbHost.Group,
			Tags:       dbHost.Tags,
			AuthType:   dbHost.AuthType,
			Password:   dbHost.Password,
			KeyPath:    dbHost.KeyPath,
			KeyContent: dbHost.KeyContent,
		})
	}

	// 转换为响应格式
	hosts := make([]HostResponse, 0, len(dbHosts))
	for _, dbHost := range dbHosts {
		hosts = append(hosts, HostResponse{
			ID:       dbHost.ID,
			Name:     dbHost.Name,
			Host:     dbHost.IP,
			Port:     dbHost.Port,
			User:     dbHost.User,
			Username: dbHost.User,
			Group:    dbHost.Group,
			Tags:     dbHost.Tags,
			AuthType: dbHost.AuthType,
			Status:   dbHost.Status,
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
	// 从数据库获取所有主机
	dbHosts, err := h.hostRepo.List(repository.HostFilter{})
	if err != nil {
		InternalError(c, "获取主机列表失败: "+err.Error())
		return
	}

	// 同步到SSH Pool
	for _, dbHost := range dbHosts {
		h.sshPool.AddHost(ssh.HostInfo{
			Name:       dbHost.Name,
			Host:       dbHost.IP,
			Port:       dbHost.Port,
			User:       dbHost.User,
			Group:      dbHost.Group,
			Tags:       dbHost.Tags,
			AuthType:   dbHost.AuthType,
			Password:   dbHost.Password,
			KeyPath:    dbHost.KeyPath,
			KeyContent: dbHost.KeyContent,
		})
	}

	// 转换为响应格式
	hosts := make([]HostResponse, 0, len(dbHosts))
	for _, dbHost := range dbHosts {
		hosts = append(hosts, HostResponse{
			ID:       dbHost.ID,
			Name:     dbHost.Name,
			Host:     dbHost.IP,
			Port:     dbHost.Port,
			User:     dbHost.User,
			Username: dbHost.User,
			Group:    dbHost.Group,
			Tags:     dbHost.Tags,
			Status:   dbHost.Status,
		})
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

	// 去除名称和地址的前后空格
	req.Name = strings.TrimSpace(req.Name)
	req.Host = strings.TrimSpace(req.Host)

	// 设置默认值
	if req.Port == 0 {
		req.Port = 22
	}

	user := req.getUser()
	authType := req.getAuthType()

	// 检查主机名是否已存在
	if _, err := h.hostRepo.GetByName(req.Name); err == nil {
		ParamError(c, "主机名已存在: "+req.Name)
		return
	}

	// 生成ID（使用name作为ID，保持与SSH Pool一致）
	hostID := req.Name
	if req.ID != "" {
		hostID = strings.TrimSpace(req.ID)
	}

	// 保存到数据库
	hostModel := &model.Host{
		ID:         hostID,
		Name:       req.Name,
		IP:         req.Host,
		Port:       req.Port,
		User:       user,
		Group:      req.Group,
		Tags:       req.Tags,
		AuthType:   authType,
		Password:   req.Password,
		KeyPath:    req.KeyPath,
		KeyContent: req.PrivateKey,
		Status:     model.HostStatusUnknown,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := h.hostRepo.Create(hostModel); err != nil {
		InternalError(c, "保存主机失败: "+err.Error())
		return
	}

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

	// 返回完整的主机信息
	Success(c, HostResponse{
		ID:       hostID,
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		User:     user,
		Username: user,
		Group:    req.Group,
		Tags:     req.Tags,
		AuthType: authType,
		Status:   model.HostStatusUnknown,
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

	// 获取现有主机
	existingHost, err := h.hostRepo.GetByID(id)
	if err != nil {
		NotFound(c, "主机不存在")
		return
	}

	var req HostInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 去除名称和地址的前后空格
	req.Name = strings.TrimSpace(req.Name)
	req.Host = strings.TrimSpace(req.Host)

	user := req.getUser()
	authType := req.getAuthType()

	// 更新数据库（如果名称发生变化，也需要更新Name和ID）
	if req.Name != "" && req.Name != existingHost.Name {
		// 检查新名称是否已被使用
		if _, err := h.hostRepo.GetByName(req.Name); err == nil {
			ParamError(c, "主机名已存在: "+req.Name)
			return
		}
		existingHost.Name = req.Name
		// 注意：这里不更新ID，因为ID是主键，更新ID需要特殊处理
		// 如果确实需要更改ID，需要删除旧记录并创建新记录
	}
	existingHost.IP = req.Host
	existingHost.Port = req.Port
	existingHost.User = user
	existingHost.Group = req.Group
	existingHost.Tags = req.Tags
	existingHost.AuthType = authType
	existingHost.Password = req.Password
	existingHost.KeyPath = req.KeyPath
	existingHost.KeyContent = req.PrivateKey
	existingHost.UpdatedAt = time.Now()
	if err := h.hostRepo.Update(existingHost); err != nil {
		InternalError(c, "更新主机失败: "+err.Error())
		return
	}

	// 更新 SSH Pool
	h.sshPool.RemoveHost(id)
	h.sshPool.AddHost(ssh.HostInfo{
		Name:       existingHost.Name, // 使用 existingHost.Name 而不是 req.Name，因为Name可能没有更新
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

	// 返回更新后的主机信息
	SuccessWithMessage(c, "更新成功", HostResponse{
		ID:       existingHost.ID,
		Name:     existingHost.Name,
		Host:     existingHost.IP,
		Port:     existingHost.Port,
		User:     existingHost.User,
		Username: existingHost.User,
		Group:    existingHost.Group,
		Tags:     existingHost.Tags,
		AuthType: existingHost.AuthType,
		Status:   existingHost.Status,
	})
}

// DeleteHost 删除主机
// DELETE /api/hosts/:id
func (h *HostHandler) DeleteHost(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "id 不能为空")
		return
	}

	// 从数据库删除
	if err := h.hostRepo.Delete(id); err != nil {
		InternalError(c, "删除主机失败: "+err.Error())
		return
	}

	// 从 SSH Pool 删除
	h.sshPool.RemoveHost(id)

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

	// 从数据库获取
	dbHost, err := h.hostRepo.GetByID(id)
	if err != nil {
		NotFound(c, "主机不存在")
		return
	}

	// 同步到SSH Pool
	h.sshPool.AddHost(ssh.HostInfo{
		Name:       dbHost.Name,
		Host:       dbHost.IP,
		Port:       dbHost.Port,
		User:       dbHost.User,
		Group:      dbHost.Group,
		Tags:       dbHost.Tags,
		AuthType:   dbHost.AuthType,
		Password:   dbHost.Password,
		KeyPath:    dbHost.KeyPath,
		KeyContent: dbHost.KeyContent,
	})

	Success(c, HostResponse{
		ID:       dbHost.ID,
		Name:     dbHost.Name,
		Host:     dbHost.IP,
		Port:     dbHost.Port,
		User:     dbHost.User,
		Username: dbHost.User,
		Group:    dbHost.Group,
		Tags:     dbHost.Tags,
		AuthType: dbHost.AuthType,
		Status:   dbHost.Status,
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

	// 从数据库批量删除
	for _, id := range req.IDs {
		h.hostRepo.Delete(id)
		h.sshPool.RemoveHost(id)
	}

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
		Format string `json:"format" binding:"required"` // csv / json
		Data   string `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	var hosts []HostInfo
	var err error

	switch req.Format {
	case "json":
		hosts, err = parseJSONHosts(req.Data)
	case "csv":
		hosts, err = parseCSVHosts(req.Data)
	default:
		ParamError(c, "不支持的格式，仅支持 csv 或 json")
		return
	}

	if err != nil {
		ParamError(c, "解析数据失败: "+err.Error())
		return
	}

	// 导入主机
	imported := 0
	failed := 0
	for _, hostInfo := range hosts {
		// 去除名称和地址的前后空格
		hostInfo.Name = strings.TrimSpace(hostInfo.Name)
		hostInfo.Host = strings.TrimSpace(hostInfo.Host)

		// 检查主机名是否已存在
		if _, err := h.hostRepo.GetByName(hostInfo.Name); err == nil {
			failed++
			continue
		}

		// 设置默认值
		if hostInfo.Port == 0 {
			hostInfo.Port = 22
		}
		user := hostInfo.getUser()
		authType := hostInfo.getAuthType()

		// 保存到数据库
		hostModel := &model.Host{
			ID:         hostInfo.Name,
			Name:       hostInfo.Name,
			IP:         hostInfo.Host,
			Port:       hostInfo.Port,
			User:       user,
			Group:      hostInfo.Group,
			Tags:       hostInfo.Tags,
			AuthType:   authType,
			Password:   hostInfo.Password,
			KeyPath:    hostInfo.KeyPath,
			KeyContent: hostInfo.PrivateKey,
			Status:     model.HostStatusUnknown,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := h.hostRepo.Create(hostModel); err != nil {
			failed++
			continue
		}

		// 添加到SSH Pool
		h.sshPool.AddHost(ssh.HostInfo{
			Name:       hostInfo.Name,
			Host:       hostInfo.Host,
			Port:       hostInfo.Port,
			User:       user,
			Group:      hostInfo.Group,
			Tags:       hostInfo.Tags,
			AuthType:   authType,
			Password:   hostInfo.Password,
			KeyPath:    hostInfo.KeyPath,
			KeyContent: hostInfo.PrivateKey,
		})

		imported++
	}

	Success(c, gin.H{
		"imported": imported,
		"failed":   failed,
		"total":    len(hosts),
	})
}

// parseJSONHosts 解析JSON格式的主机数据
func parseJSONHosts(data string) ([]HostInfo, error) {
	var hosts []HostInfo
	if err := json.Unmarshal([]byte(data), &hosts); err != nil {
		return nil, err
	}
	return hosts, nil
}

// parseCSVHosts 解析CSV格式的主机数据
// CSV格式: name,host,port,user,group,auth_type,password,key_path
func parseCSVHosts(data string) ([]HostInfo, error) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("CSV数据为空")
	}

	var hosts []HostInfo
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			return nil, fmt.Errorf("第%d行格式错误，至少需要name和host字段", i+1)
		}

		host := HostInfo{
			Name: strings.TrimSpace(fields[0]),
			Host: strings.TrimSpace(fields[1]),
		}

		if len(fields) > 2 && strings.TrimSpace(fields[2]) != "" {
			if port, err := strconv.Atoi(strings.TrimSpace(fields[2])); err == nil {
				host.Port = port
			}
		}
		if len(fields) > 3 {
			host.User = strings.TrimSpace(fields[3])
		}
		if len(fields) > 4 {
			host.Group = strings.TrimSpace(fields[4])
		}
		if len(fields) > 5 {
			host.AuthType = strings.TrimSpace(fields[5])
		}
		if len(fields) > 6 {
			host.Password = strings.TrimSpace(fields[6])
		}
		if len(fields) > 7 {
			host.KeyPath = strings.TrimSpace(fields[7])
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

// GetGroups 获取分组列表
// GET /api/groups
func (h *HostHandler) GetGroups(c *gin.Context) {
	groups, err := h.groupRepo.List()
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

	// 检查分组名是否已存在
	if _, err := h.groupRepo.GetByName(req.Name); err == nil {
		ParamError(c, "分组名已存在: "+req.Name)
		return
	}

	// 保存到数据库
	group := &model.Group{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}
	if err := h.groupRepo.Create(group); err != nil {
		InternalError(c, "创建分组失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "创建成功", gin.H{
		"id":   group.ID,
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

	// 从数据库删除
	if err := h.groupRepo.DeleteByName(name); err != nil {
		InternalError(c, "删除分组失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", nil)
}
