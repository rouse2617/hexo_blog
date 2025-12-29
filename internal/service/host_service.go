package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/cache"
	"ai-ops/internal/model"
	"ai-ops/internal/repository"
	"ai-ops/internal/ssh"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HostService 主机服务
type HostService struct {
	sshPool   *ssh.Pool
	hostRepo  repository.HostRepository
	groupRepo repository.GroupRepository
	cache     cache.Cache
	keyGen    *cache.CacheKey
}

// NewHostService 创建主机服务（cache 可选）
func NewHostService(sshPool *ssh.Pool, hostRepo repository.HostRepository, groupRepo repository.GroupRepository, cacheObj ...cache.Cache) *HostService {
	svc := &HostService{
		sshPool:   sshPool,
		hostRepo:  hostRepo,
		groupRepo: groupRepo,
	}
	if len(cacheObj) > 0 && cacheObj[0] != nil {
		svc.cache = cacheObj[0]
		svc.keyGen = cache.NewCacheKey("host")
	} else {
		svc.cache = nil
		svc.keyGen = nil
	}
	return svc
}

// HostInfo 主机信息
type HostInfo struct {
	ID          string
	Name        string
	Host        string
	Port        int
	User        string
	Username    string
	Group       string
	Tags        []string
	AuthType    string
	Password    string
	PrivateKey  string
	KeyPath     string
	Description string
}

// HostResponse 主机响应
type HostResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	User        string   `json:"user"`
	Username    string   `json:"username"`
	Group       string   `json:"group"`
	Tags        []string `json:"tags"`
	AuthType    string   `json:"authType"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
}

// ListHosts 获取主机列表（带缓存）
func (s *HostService) ListHosts(filter repository.HostFilter) ([]*HostResponse, error) {
	ctx := context.Background()

	// 尝试从缓存获取
	if s.cache != nil && s.keyGen != nil {
		cacheKey := s.keyGen.Build("list", filter.Group, filter.Keyword, filter.Status)
		var cachedHosts []*model.Host
		err := s.cache.Get(ctx, cacheKey, &cachedHosts)
		if err == nil {
			return s.convertToResponses(cachedHosts), nil
		}
	}

	// 从数据库获取
	dbHosts, err := s.hostRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("获取主机列表失败: %w", err)
	}

	// 同步到 SSH Pool
	s.syncDBHostsToSSHPool(dbHosts)

	// 设置缓存（5分钟过期）
	if s.cache != nil && s.keyGen != nil {
		cacheKey := s.keyGen.Build("list", filter.Group, filter.Keyword, filter.Status)
		if err := s.cache.Set(ctx, cacheKey, dbHosts, 5*time.Minute); err != nil {
			// 记录日志但不影响主流程
			fmt.Printf("Warning: failed to set cache: %v\n", err)
		}
	}

	return s.convertToResponses(dbHosts), nil
}

// CreateHost 创建主机
func (s *HostService) CreateHost(info *HostInfo) (*HostResponse, error) {
	ctx := context.Background()

	// 标准化输入
	info.Name = strings.TrimSpace(info.Name)
	info.Host = strings.TrimSpace(info.Host)

	// 设置默认值
	if info.Port == 0 {
		info.Port = 22
	}

	user := s.getUser(info)
	authType := s.getAuthType(info)

	// 检查主机名是否已存在
	if _, err := s.hostRepo.GetByName(info.Name); err == nil {
		return nil, fmt.Errorf("主机名已存在: %s", info.Name)
	}

	// 生成 ID
	hostID := info.Name
	if info.ID != "" {
		hostID = strings.TrimSpace(info.ID)
	}

	// 保存到数据库
	hostModel := &model.Host{
		ID:         hostID,
		Name:       info.Name,
		IP:         info.Host,
		Port:       info.Port,
		User:       user,
		Group:      info.Group,
		Tags:       info.Tags,
		AuthType:   authType,
		Password:   info.Password,
		KeyPath:    info.KeyPath,
		KeyContent: info.PrivateKey,
		Status:     model.HostStatusUnknown,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.hostRepo.Create(hostModel); err != nil {
		return nil, fmt.Errorf("保存主机失败: %w", err)
	}

	// 添加到 SSH Pool
	s.sshPool.AddHost(ssh.HostInfo{
		Name:       info.Name,
		Host:       info.Host,
		Port:       info.Port,
		User:       user,
		Group:      info.Group,
		Tags:       info.Tags,
		AuthType:   authType,
		Password:   info.Password,
		KeyPath:    info.KeyPath,
		KeyContent: info.PrivateKey,
	})

	// 清除相关缓存
	s.invalidateHostListCache(ctx)

	return &HostResponse{
		ID:       hostID,
		Name:     info.Name,
		Host:     info.Host,
		Port:     info.Port,
		User:     user,
		Username: user,
		Group:    info.Group,
		Tags:     info.Tags,
		AuthType: authType,
		Status:   model.HostStatusUnknown,
	}, nil
}

// UpdateHost 更新主机
func (s *HostService) UpdateHost(id string, info *HostInfo) (*HostResponse, error) {
	ctx := context.Background()

	// 获取现有主机
	existingHost, err := s.hostRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("主机不存在: %w", err)
	}

	// 标准化输入
	info.Name = strings.TrimSpace(info.Name)
	info.Host = strings.TrimSpace(info.Host)

	user := s.getUser(info)
	authType := s.getAuthType(info)

	// 检查名称是否变化
	if info.Name != "" && info.Name != existingHost.Name {
		if _, err := s.hostRepo.GetByName(info.Name); err == nil {
			return nil, fmt.Errorf("主机名已存在: %s", info.Name)
		}
		existingHost.Name = info.Name
	}

	// 更新字段
	existingHost.IP = info.Host
	existingHost.Port = info.Port
	existingHost.User = user
	existingHost.Group = info.Group
	existingHost.Tags = info.Tags
	existingHost.AuthType = authType

	// 只在提供了新值时才更新密码和私钥
	if info.Password != "" {
		existingHost.Password = info.Password
	}
	if info.KeyPath != "" {
		existingHost.KeyPath = info.KeyPath
	}
	if info.PrivateKey != "" {
		existingHost.KeyContent = info.PrivateKey
	}

	existingHost.UpdatedAt = time.Now()

	if err := s.hostRepo.Update(existingHost); err != nil {
		return nil, fmt.Errorf("更新主机失败: %w", err)
	}

	// 更新 SSH Pool
	s.sshPool.RemoveHost(id)
	s.sshPool.AddHost(ssh.HostInfo{
		Name:       existingHost.Name,
		Host:       info.Host,
		Port:       info.Port,
		User:       user,
		Group:      info.Group,
		Tags:       info.Tags,
		AuthType:   authType,
		Password:   info.Password,
		KeyPath:    info.KeyPath,
		KeyContent: info.PrivateKey,
	})

	// 清除相关缓存
	s.invalidateHostCache(ctx, id)
	s.invalidateHostListCache(ctx)

	return &HostResponse{
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
	}, nil
}

// DeleteHost 删除主机
func (s *HostService) DeleteHost(id string) error {
	ctx := context.Background()

	// 从数据库删除
	if err := s.hostRepo.Delete(id); err != nil {
		return fmt.Errorf("删除主机失败: %w", err)
	}

	// 从 SSH Pool 删除
	s.sshPool.RemoveHost(id)

	// 清除相关缓存
	s.invalidateHostCache(ctx, id)
	s.invalidateHostListCache(ctx)

	return nil
}

// GetHost 获取单个主机
func (s *HostService) GetHost(id string) (*HostResponse, error) {
	// 从数据库获取
	dbHost, err := s.hostRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("主机不存在: %w", err)
	}

	// 同步到 SSH Pool
	s.sshPool.AddHost(ssh.HostInfo{
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

	return &HostResponse{
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
	}, nil
}

// BatchDeleteHosts 批量删除主机
func (s *HostService) BatchDeleteHosts(ids []string) (int, error) {
	deleted := 0
	for _, id := range ids {
		if err := s.hostRepo.Delete(id); err != nil {
			zap.Error(fmt.Errorf("删除主机 %s 失败: %w", id, err))
			continue
		}
		s.sshPool.RemoveHost(id)
		deleted++
	}
	return deleted, nil
}

// TestConnection 测试主机连接
func (s *HostService) TestConnection(id string) (string, error) {
	output, err := s.sshPool.Exec(id, "uptime")
	if err != nil {
		return "", fmt.Errorf("连接失败: %w", err)
	}
	return output, nil
}

// ImportHosts 导入主机
func (s *HostService) ImportHosts(format string, data string) (imported int, failed int, err error) {
	var hosts []HostInfo

	switch format {
	case "json":
		hosts, err = s.parseJSONHosts(data)
	case "csv":
		hosts, err = s.parseCSVHosts(data)
	default:
		return 0, 0, fmt.Errorf("不支持的格式，仅支持 csv 或 json")
	}

	if err != nil {
		return 0, 0, fmt.Errorf("解析数据失败: %w", err)
	}

	// 导入主机
	for _, hostInfo := range hosts {
		hostInfo.Name = strings.TrimSpace(hostInfo.Name)
		hostInfo.Host = strings.TrimSpace(hostInfo.Host)

		// 检查主机名是否已存在
		if _, err := s.hostRepo.GetByName(hostInfo.Name); err == nil {
			failed++
			continue
		}

		// 设置默认值
		if hostInfo.Port == 0 {
			hostInfo.Port = 22
		}

		user := s.getUser(&hostInfo)
		authType := s.getAuthType(&hostInfo)

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

		if err := s.hostRepo.Create(hostModel); err != nil {
			failed++
			continue
		}

		// 添加到 SSH Pool
		s.sshPool.AddHost(ssh.HostInfo{
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

	return imported, failed, nil
}

// GetGroups 获取分组列表
func (s *HostService) GetGroups() ([]*model.Group, error) {
	return s.groupRepo.List()
}

// CreateGroup 创建分组
func (s *HostService) CreateGroup(name string, description string) (string, error) {
	// 检查分组名是否已存在
	if _, err := s.groupRepo.GetByName(name); err == nil {
		return "", fmt.Errorf("分组名已存在: %s", name)
	}

	group := &model.Group{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}

	if err := s.groupRepo.Create(group); err != nil {
		return "", fmt.Errorf("创建分组失败: %w", err)
	}

	return group.ID, nil
}

// DeleteGroup 删除分组
func (s *HostService) DeleteGroup(name string) error {
	return s.groupRepo.DeleteByName(name)
}

// syncDBHostsToSSHPool 同步数据库主机到 SSH Pool
func (s *HostService) syncDBHostsToSSHPool(dbHosts []*model.Host) {
	for _, dbHost := range dbHosts {
		s.sshPool.AddHost(ssh.HostInfo{
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
}

// getUser 获取用户名
func (s *HostService) getUser(info *HostInfo) string {
	if info.User != "" {
		return info.User
	}
	if info.Username != "" {
		return info.Username
	}
	return "root"
}

// getAuthType 获取认证类型
func (s *HostService) getAuthType(info *HostInfo) string {
	if info.AuthType != "" {
		return info.AuthType
	}
	return "auto"
}

// parseJSONHosts 解析 JSON 格式的主机数据
func (s *HostService) parseJSONHosts(data string) ([]HostInfo, error) {
	var hosts []HostInfo
	if err := json.Unmarshal([]byte(data), &hosts); err != nil {
		return nil, err
	}
	return hosts, nil
}

// parseCSVHosts 解析 CSV 格式的主机数据
// CSV 格式: name,host,port,user,group,auth_type,password,key_path
func (s *HostService) parseCSVHosts(data string) ([]HostInfo, error) {
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

// LoadHostsFromConfig 从配置加载主机到 SSH Pool 和数据库
func (s *HostService) LoadHostsFromConfig(configHosts []ConfigHost) {
	for _, h := range configHosts {
		port := h.Port
		if port == 0 {
			port = 22
		}
		user := h.User
		if user == "" {
			user = "root"
		}
		authType := h.AuthType
		if authType == "" {
			if h.Password != "" {
				authType = "password"
			} else {
				authType = "key"
			}
		}
		keyPath := h.KeyPath
		if keyPath == "" && authType == "key" {
			keyPath = "~/.ssh/id_rsa"
		}

		// 检查是否已存在于数据库
		_, err := s.hostRepo.GetByName(h.Name)
		if err != nil {
			// 不存在，添加到数据库
			hostModel := &model.Host{
				ID:        h.Name,
				Name:      h.Name,
				IP:        h.Host,
				Port:      port,
				User:      user,
				Group:     h.Group,
				Tags:      h.Tags,
				AuthType:  authType,
				Password:  h.Password,
				KeyPath:   keyPath,
				Status:    model.HostStatusUnknown,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := s.hostRepo.Create(hostModel); err != nil {
				fmt.Printf("Warning: 保存配置主机到数据库失败 name=%s error=%v\n", h.Name, err)
			}
		}

		// 添加到 SSH Pool
		s.sshPool.AddHost(ssh.HostInfo{
			Name:     h.Name,
			Host:     h.Host,
			Port:     port,
			User:     user,
			Group:    h.Group,
			Tags:     h.Tags,
			AuthType: authType,
			Password: h.Password,
			KeyPath:  keyPath,
		})
	}
}

// ConfigHost 配置文件中的主机定义
type ConfigHost struct {
	Name     string
	Host     string
	Port     int
	User     string
	Group    string
	Tags     []string
	AuthType string
	Password string
	KeyPath  string
}

// GetCacheMetrics 获取缓存指标
func (s *HostService) GetCacheMetrics() cache.Metrics {
	if s.cache == nil {
		return cache.Metrics{}
	}
	return s.cache.GetMetrics()
}

// ClearAllCache 清除所有主机相关缓存
func (s *HostService) ClearAllCache() error {
	ctx := context.Background()
	return s.cache.Clear(ctx)
}

// invalidateHostCache 使主机缓存失效
func (s *HostService) invalidateHostCache(ctx context.Context, id string) {
	if s.cache == nil {
		return
	}
	pattern := s.keyGen.Build("*", id)
	_ = s.cache.DeleteByPattern(ctx, pattern)
}

// invalidateHostListCache 使主机列表缓存失效
func (s *HostService) invalidateHostListCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	pattern := s.keyGen.Pattern("list")
	_ = s.cache.DeleteByPattern(ctx, pattern)
}

// convertToResponses 转换为响应格式
func (s *HostService) convertToResponses(dbHosts []*model.Host) []*HostResponse {
	hosts := make([]*HostResponse, 0, len(dbHosts))
	for _, dbHost := range dbHosts {
		hosts = append(hosts, &HostResponse{
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
	return hosts
}
