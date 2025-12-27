package repository

import (
	"ai-ops/internal/model"
	"time"

	"gorm.io/gorm"
)

// ConfigRepository 配置仓库接口
type ConfigRepository interface {
	Get(key string) (*model.Config, error)
	Set(key string, value string) error
	GetAll() (map[string]string, error)
}

// configRepository 配置仓库实现
type configRepository struct {
	db *gorm.DB
}

// NewConfigRepository 创建配置仓库
func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{db: db}
}

// Get 获取配置
func (r *configRepository) Get(key string) (*model.Config, error) {
	var config model.Config
	err := r.db.Where("key = ?", key).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Set 设置配置
func (r *configRepository) Set(key string, value string) error {
	config := &model.Config{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return r.db.Save(config).Error
}

// GetAll 获取所有配置
func (r *configRepository) GetAll() (map[string]string, error) {
	var configs []model.Config
	err := r.db.Find(&configs).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.Key] = config.Value
	}
	return result, nil
}

