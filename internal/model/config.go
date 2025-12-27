package model

import (
	"time"
)

// Config 系统配置模型
type Config struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

