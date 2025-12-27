package repository

import (
	"ai-ops/internal/model"
	"time"

	"gorm.io/gorm"
)

// AnalysisRepository 分析历史仓库接口
type AnalysisRepository interface {
	Create(analysis *model.Analysis) error
	GetByID(id string) (*model.Analysis, error)
	List(sessionID string, limit int) ([]*model.Analysis, error)
	Delete(id string) error
}

// analysisRepository 分析历史仓库实现
type analysisRepository struct {
	db *gorm.DB
}

// NewAnalysisRepository 创建分析历史仓库
func NewAnalysisRepository(db *gorm.DB) AnalysisRepository {
	return &analysisRepository{db: db}
}

// Create 创建分析记录
func (r *analysisRepository) Create(analysis *model.Analysis) error {
	return r.db.Create(analysis).Error
}

// GetByID 根据ID获取分析记录
func (r *analysisRepository) GetByID(id string) (*model.Analysis, error) {
	var analysis model.Analysis
	err := r.db.Where("id = ?", id).First(&analysis).Error
	if err != nil {
		return nil, err
	}
	return &analysis, nil
}

// List 列表查询
func (r *analysisRepository) List(sessionID string, limit int) ([]*model.Analysis, error) {
	var analyses []*model.Analysis
	query := r.db.Model(&model.Analysis{})
	
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Order("created_at DESC").Find(&analyses).Error
	return analyses, err
}

// Delete 删除分析记录
func (r *analysisRepository) Delete(id string) error {
	return r.db.Delete(&model.Analysis{}, "id = ?", id).Error
}

