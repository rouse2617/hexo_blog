package model

import (
	"time"
)

// Group 主机分组模型
type Group struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

