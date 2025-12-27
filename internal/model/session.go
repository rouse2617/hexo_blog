package model

import (
	"time"
)

// Session 对话会话模型
type Session struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Hosts     []string  `json:"hosts" gorm:"serializer:json"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

