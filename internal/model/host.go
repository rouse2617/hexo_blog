package model

import (
	"time"
)

// Host 主机模型
type Host struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	IP        string    `json:"ip" gorm:"not null"`
	Port      int       `json:"port" gorm:"default:22"`
	User      string    `json:"user" gorm:"default:root"`
	Group     string    `json:"group" gorm:"index"`
	Tags      []string  `json:"tags" gorm:"serializer:json"`
	AuthType  string    `json:"auth_type" gorm:"default:key"`
	Password  string    `json:"-" gorm:""`
	KeyPath   string    `json:"key_path" gorm:""`
	KeyContent string   `json:"-" gorm:""`
	Status    string    `json:"status" gorm:"default:unknown"`
	LastCheck *time.Time `json:"last_check"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HostStatus 主机状态常量
const (
	HostStatusOnline  = "online"
	HostStatusOffline = "offline"
	HostStatusUnknown = "unknown"
)

// AuthType 认证类型常量
const (
	AuthTypeAuto       = "auto"
	AuthTypePassword   = "password"
	AuthTypeKey        = "key"
	AuthTypeKeyContent = "key_content"
)

