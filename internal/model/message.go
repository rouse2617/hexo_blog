package model

import (
	"time"
)

// Message 对话消息模型
type Message struct {
	ID        string     `json:"id" gorm:"primaryKey"`
	SessionID string     `json:"session_id" gorm:"index;not null"`
	Role      string     `json:"role"`
	Content   string     `json:"content" gorm:"type:text"`
	ToolCalls []ToolCall `json:"tool_calls" gorm:"serializer:json"`
	CreatedAt time.Time  `json:"created_at"`
}

// ToolCall 工具调用记录
type ToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
	Result string                 `json:"result"`
	Error  string                 `json:"error,omitempty"`
}

// MessageRole 消息角色常量
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	RoleSystem    = "system"
)

