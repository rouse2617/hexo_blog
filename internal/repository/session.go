package repository

import (
	"ai-ops/internal/model"
	"time"

	"gorm.io/gorm"
)

// SessionRepository 会话仓库接口
type SessionRepository interface {
	Create(session *model.Session) error
	GetByID(id string) (*model.Session, error)
	List(limit int) ([]*model.Session, error)
	Delete(id string) error
	AddMessage(sessionID string, msg *model.Message) error
	GetMessages(sessionID string) ([]*model.Message, error)
	UpdateTitle(sessionID string, title string) error
}

// sessionRepository 会话仓库实现
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 创建会话仓库
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create 创建会话
func (r *sessionRepository) Create(session *model.Session) error {
	return r.db.Create(session).Error
}

// GetByID 根据ID获取会话
func (r *sessionRepository) GetByID(id string) (*model.Session, error) {
	var session model.Session
	err := r.db.Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// List 列表查询
func (r *sessionRepository) List(limit int) ([]*model.Session, error) {
	var sessions []*model.Session
	query := r.db.Model(&model.Session{})
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("updated_at DESC").Find(&sessions).Error
	return sessions, err
}

// Delete 删除会话
func (r *sessionRepository) Delete(id string) error {
	// 先删除关联的消息
	if err := r.db.Where("session_id = ?", id).Delete(&model.Message{}).Error; err != nil {
		return err
	}
	// 删除会话
	return r.db.Delete(&model.Session{}, "id = ?", id).Error
}

// AddMessage 添加消息
func (r *sessionRepository) AddMessage(sessionID string, msg *model.Message) error {
	msg.SessionID = sessionID
	if err := r.db.Create(msg).Error; err != nil {
		return err
	}
	// 更新会话的 updated_at
	return r.db.Model(&model.Session{}).Where("id = ?", sessionID).Update("updated_at", time.Now()).Error
}

// GetMessages 获取会话的所有消息
func (r *sessionRepository) GetMessages(sessionID string) ([]*model.Message, error) {
	var messages []*model.Message
	err := r.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

// UpdateTitle 更新会话标题
func (r *sessionRepository) UpdateTitle(sessionID string, title string) error {
	return r.db.Model(&model.Session{}).Where("id = ?", sessionID).Update("title", title).Error
}

