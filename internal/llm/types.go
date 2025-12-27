package llm

import "context"

// Client LLM 客户端接口
type Client interface {
	// Chat 普通对话
	Chat(ctx context.Context, messages []Message) (*ChatResponse, error)

	// ChatWithTools 带工具调用的对话
	ChatWithTools(ctx context.Context, messages []Message, tools []ToolDef) (*ChatResponse, error)

	// ChatStream 流式对话
	ChatStream(ctx context.Context, messages []Message, callback StreamCallback) error

	// ChatStreamWithTools 带工具调用的流式对话
	ChatStreamWithTools(ctx context.Context, messages []Message, tools []ToolDef, callback StreamCallback) error
}

// StreamCallback 流式回调
type StreamCallback func(chunk StreamChunk)

// StreamChunk 流式数据块
type StreamChunk struct {
	Type         string    `json:"type"`                    // content, tool_call, done, error
	Content      string    `json:"content,omitempty"`       // 文本内容
	ToolCall     *ToolCall `json:"tool_call,omitempty"`     // 工具调用
	Error        string    `json:"error,omitempty"`         // 错误信息
	FinishReason string    `json:"finish_reason,omitempty"` // 结束原因
}

// Message 消息
type Message struct {
	Role       string     `json:"role"`                   // system, user, assistant, tool
	Content    string     `json:"content"`                // 消息内容
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // 工具调用（assistant 消息）
	ToolCallID string     `json:"tool_call_id,omitempty"` // 工具调用 ID（tool 消息）
	Name       string     `json:"name,omitempty"`         // 工具名称（tool 消息）
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // function
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// ToolDef 工具定义（用于 function calling）
type ToolDef struct {
	Type     string      `json:"type"` // function
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	ID           string  `json:"id"`
	Model        string  `json:"model"`
	Message      Message `json:"message"`
	Usage        Usage   `json:"usage"`
	FinishReason string  `json:"finish_reason"` // stop, tool_calls, length
}

// Usage token 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Role 常量
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// FinishReason 常量
const (
	FinishReasonStop      = "stop"
	FinishReasonToolCalls = "tool_calls"
	FinishReasonLength    = "length"
)

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

// NewAssistantToolCallMessage 创建带工具调用的助手消息
func NewAssistantToolCallMessage(toolCalls []ToolCall) Message {
	return Message{Role: RoleAssistant, ToolCalls: toolCalls}
}

// NewToolMessage 创建工具结果消息
func NewToolMessage(toolCallID string, name string, content string) Message {
	return Message{
		Role:       RoleTool,
		Content:    content,
		ToolCallID: toolCallID,
		Name:       name,
	}
}

// HasToolCalls 检查响应是否包含工具调用
func (r *ChatResponse) HasToolCalls() bool {
	return len(r.Message.ToolCalls) > 0
}
