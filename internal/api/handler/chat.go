package handler

import (
	"io"

	"ai-ops/internal/agent"

	"github.com/gin-gonic/gin"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	agent *agent.Agent
}

// NewChatHandler 创建对话处理器
func NewChatHandler(agent *agent.Agent) *ChatHandler {
	return &ChatHandler{agent: agent}
}

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID string   `json:"session_id"`
	Message   string   `json:"message" binding:"required"`
	Hosts     []string `json:"hosts"`
	Stream    bool     `json:"stream"`
}

// ChatResponseData 对话响应
type ChatResponseData struct {
	SessionID string                 `json:"session_id"`
	Reply     string                 `json:"reply"`
	ToolCalls []agent.ToolCallRecord `json:"tool_calls,omitempty"`
}

// Chat 对话接口
// POST /api/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 流式响应
	if req.Stream {
		h.chatStream(c, req)
		return
	}

	// 普通响应
	resp, err := h.agent.Chat(c.Request.Context(), agent.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		Hosts:     req.Hosts,
	})

	if err != nil {
		LLMError(c, "对话失败: "+err.Error())
		return
	}

	Success(c, ChatResponseData{
		SessionID: resp.SessionID,
		Reply:     resp.Reply,
		ToolCalls: resp.ToolCalls,
	})
}

// chatStream 流式对话
func (h *ChatHandler) chatStream(c *gin.Context, req ChatRequest) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 流式处理
	err := h.agent.ChatStream(c.Request.Context(), agent.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		Hosts:     req.Hosts,
	}, func(chunk agent.StreamChunk) {
		// 发送 SSE 事件
		switch chunk.Type {
		case "content":
			c.SSEvent("content", gin.H{"content": chunk.Content})
		case "tool_call":
			c.SSEvent("tool_call", gin.H{
				"tool":   chunk.ToolCall.Function.Name,
				"params": chunk.ToolCall.Function.Arguments,
			})
		case "tool_result":
			c.SSEvent("tool_result", gin.H{
				"tool":   chunk.ToolResult.Tool,
				"result": chunk.ToolResult.Result,
				"error":  chunk.ToolResult.Error,
			})
		case "done":
			c.SSEvent("done", gin.H{"session_id": req.SessionID})
		case "error":
			c.SSEvent("error", gin.H{"error": chunk.Error})
		}
		c.Writer.Flush()
	})

	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
	}
}

// StreamChat 流式对话接口 (SSE)
// POST /api/chat/stream
func (h *ChatHandler) StreamChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	req.Stream = true
	h.chatStream(c, req)
}

// GetSessions 获取会话列表
// GET /api/chat/sessions
func (h *ChatHandler) GetSessions(c *gin.Context) {
	// TODO: 从数据库获取会话列表
	// 目前返回空列表
	Success(c, gin.H{
		"sessions": []interface{}{},
	})
}

// GetHistory 获取对话历史
// GET /api/chat/history?session_id=xxx
func (h *ChatHandler) GetHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		ParamError(c, "session_id 不能为空")
		return
	}

	// TODO: 从数据库获取对话历史
	// 目前返回空列表
	Success(c, gin.H{
		"session_id": sessionID,
		"messages":   []interface{}{},
	})
}

// DeleteSession 删除会话
// DELETE /api/chat/sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		ParamError(c, "session_id 不能为空")
		return
	}

	// TODO: 从数据库删除会话
	SuccessWithMessage(c, "删除成功", nil)
}

// HealthCheck 健康检查（用于 SSE 连接测试）
func (h *ChatHandler) HealthCheck(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		c.SSEvent("ping", gin.H{"status": "ok"})
		return false
	})
}
