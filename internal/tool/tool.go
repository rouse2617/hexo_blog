package tool

import (
	"time"

	"ai-ops/internal/ssh"
)

// Tool 工具接口
type Tool interface {
	// Name 工具唯一标识
	Name() string

	// Description 工具描述（给 LLM 看，决定何时调用）
	Description() string

	// Parameters 参数定义
	Parameters() []Parameter

	// Execute 执行工具
	Execute(ctx *Context, params map[string]interface{}) (*Result, error)
}

// Parameter 参数定义
type Parameter struct {
	Name        string        `json:"name"`              // 参数名
	Type        string        `json:"type"`              // 类型: string, int, bool, []string
	Description string        `json:"description"`       // 参数描述（给 LLM 看）
	Required    bool          `json:"required"`          // 是否必填
	Default     interface{}   `json:"default,omitempty"` // 默认值
	Enum        []interface{} `json:"enum,omitempty"`    // 可选值枚举
}

// Context 执行上下文
type Context struct {
	SessionID string        // 会话 ID
	Hosts     []string      // 关联主机（用于参数规范化/默认注入）
	SSH       *ssh.Pool     // SSH 连接池
	Timeout   time.Duration // 超时时间
}

// Result 执行结果
type Result struct {
	Success bool        `json:"success"`         // 是否成功
	Data    interface{} `json:"data,omitempty"`  // 返回数据
	Message string      `json:"message"`         // 结果描述
	Error   string      `json:"error,omitempty"` // 错误信息
}

// NewResult 创建成功结果
func NewResult(data interface{}, message string) *Result {
	return &Result{
		Success: true,
		Data:    data,
		Message: message,
	}
}

// NewErrorResult 创建错误结果
func NewErrorResult(err string) *Result {
	return &Result{
		Success: false,
		Error:   err,
	}
}

// ToolInfo 工具信息（用于 API 返回）
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"` // builtin / script / remote
	Parameters  []Parameter `json:"parameters"`
}

// GetStringParam 获取字符串参数
func GetStringParam(params map[string]interface{}, name string, defaultVal string) string {
	if v, ok := params[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetIntParam 获取整数参数
func GetIntParam(params map[string]interface{}, name string, defaultVal int) int {
	if v, ok := params[name]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return defaultVal
}

// GetBoolParam 获取布尔参数
func GetBoolParam(params map[string]interface{}, name string, defaultVal bool) bool {
	if v, ok := params[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// GetStringSliceParam 获取字符串数组参数
func GetStringSliceParam(params map[string]interface{}, name string) []string {
	if v, ok := params[name]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case []interface{}:
			result := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// GetFloatParam 获取浮点数参数
func GetFloatParam(params map[string]interface{}, name string, defaultVal float64) float64 {
	if v, ok := params[name]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		}
	}
	return defaultVal
}

// GetArrayParam 获取数组参数（通用）
func GetArrayParam(params map[string]interface{}, name string) []interface{} {
	if v, ok := params[name]; ok {
		if arr, ok := v.([]interface{}); ok {
			return arr
		}
	}
	return nil
}
