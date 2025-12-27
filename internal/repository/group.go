package repository

import (
	"ai-ops/internal/model"

	"gorm.io/gorm"
)

// GroupRepository 分组仓库接口
type GroupRepository interface {
	Create(group *model.Group) error
	GetByID(id string) (*model.Group, error)
	GetByName(name string) (*model.Group, error)
	List() ([]*model.Group, error)
	Delete(id string) error
	DeleteByName(name string) error
}

// groupRepository 分组仓库实现
type groupRepository struct {
	db *gorm.DB
}

// NewGroupRepository 创建分组仓库
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Create 创建分组
func (r *groupRepository) Create(group *model.Group) error {
	return r.db.Create(group).Error
}

// GetByID 根据ID获取分组
func (r *groupRepository) GetByID(id string) (*model.Group, error) {
	var group model.Group
	err := r.db.Where("id = ?", id).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByName 根据名称获取分组
func (r *groupRepository) GetByName(name string) (*model.Group, error) {
	var group model.Group
	err := r.db.Where("name = ?", name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// List 列表查询
func (r *groupRepository) List() ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.Order("created_at DESC").Find(&groups).Error
	return groups, err
}

// Delete 删除分组（根据ID）
func (r *groupRepository) Delete(id string) error {
	return r.db.Delete(&model.Group{}, "id = ?", id).Error
}

// DeleteByName 删除分组（根据名称）
func (r *groupRepository) DeleteByName(name string) error {
	return r.db.Delete(&model.Group{}, "name = ?", name).Error
}

