package repository

import (
	"ai-ops/internal/model"
	"strings"

	"gorm.io/gorm"
)

// HostRepository 主机仓库接口
type HostRepository interface {
	Create(host *model.Host) error
	Update(host *model.Host) error
	Delete(id string) error
	GetByID(id string) (*model.Host, error)
	GetByName(name string) (*model.Host, error)
	List(filter HostFilter) ([]*model.Host, error)
	UpdateStatus(id string, status string) error
}

// HostFilter 主机过滤条件
type HostFilter struct {
	Group   string
	Keyword string
	Status  string
	Tags    []string
}

// hostRepository 主机仓库实现
type hostRepository struct {
	db *gorm.DB
}

// NewHostRepository 创建主机仓库
func NewHostRepository(db *gorm.DB) HostRepository {
	return &hostRepository{db: db}
}

// Create 创建主机
func (r *hostRepository) Create(host *model.Host) error {
	return r.db.Create(host).Error
}

// Update 更新主机
func (r *hostRepository) Update(host *model.Host) error {
	return r.db.Save(host).Error
}

// Delete 删除主机
func (r *hostRepository) Delete(id string) error {
	return r.db.Delete(&model.Host{}, "id = ?", id).Error
}

// GetByID 根据ID获取主机
func (r *hostRepository) GetByID(id string) (*model.Host, error) {
	var host model.Host
	err := r.db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// GetByName 根据名称获取主机
func (r *hostRepository) GetByName(name string) (*model.Host, error) {
	var host model.Host
	err := r.db.Where("name = ?", name).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// List 列表查询
func (r *hostRepository) List(filter HostFilter) ([]*model.Host, error) {
	var hosts []*model.Host
	query := r.db.Model(&model.Host{})

	if filter.Group != "" {
		query = query.Where("`group` = ?", filter.Group)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR user LIKE ?", keyword, keyword, keyword)
	}
	if len(filter.Tags) > 0 {
		// TODO: 实现标签过滤（需要 JSON 查询）
	}

	err := query.Order("created_at DESC").Find(&hosts).Error
	return hosts, err
}

// UpdateStatus 更新主机状态
func (r *hostRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&model.Host{}).Where("id = ?", id).Update("status", status).Error
}

// ContainsKeyword 检查主机是否包含关键字（用于内存过滤）
func ContainsKeyword(host *model.Host, keyword string) bool {
	if keyword == "" {
		return true
	}
	keyword = strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(host.Name), keyword) ||
		strings.Contains(strings.ToLower(host.IP), keyword) ||
		strings.Contains(strings.ToLower(host.User), keyword)
}

