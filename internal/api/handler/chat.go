package handler

import (
	"fmt"
	"io"
	"ai-ops/internal/service"

	"github.com/gin-gonic/gin"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler 创建对话处理器
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// SetEnableThinking 设置是否启用思考过程
func (h *ChatHandler) SetEnableThinking(enable bool) {
	h.chatService.SetEnableThinking(enable)
}

// SetEnableReAct 设置是否启用 ReAct 模式
func (h *ChatHandler) SetEnableReAct(enable bool) {
	h.chatService.SetEnableReAct(enable)
}


// ChatRequest 对话请求
type ChatRequest struct {
	SessionID  string   `json:"session_id"`
	SessionID2 string   `json:"sessionId"`
	Message    string   `json:"message" binding:"required"`
	Hosts      []string `json:"hosts"`
	HostIDs    []string `json:"hostIds"`
	HostIDs2   []string `json:"host_ids"`
	Stream     bool     `json:"stream"`
}

// ChatResponseData 对话响应
type ChatResponseData struct {
	SessionID string      `json:"session_id"`
	Reply     string      `json:"reply"`
	ToolCalls interface{} `json:"tool_calls,omitempty"`
	Thinking  string      `json:"thinking,omitempty"`
}

// Chat 对话接口
// POST /api/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	hosts := h.normalizeHosts(req)
	sessionID := h.normalizeSessionID(req)

	// 流式响应
	if req.Stream {
		h.chatStream(c, req)
		return
	}

	// 调用 Service 层
	resp, err := h.chatService.Chat(c.Request.Context(), service.ChatRequest{
		SessionID: sessionID,
		Message:   req.Message,
		Hosts:     hosts,
		Stream:    false,
	})

	if err != nil {
		LLMError(c, "对话失败: "+err.Error())
		return
	}

	Success(c, ChatResponseData{
		SessionID: resp.SessionID,
		Reply:     resp.Reply,
		ToolCalls: resp.ToolCalls,
		Thinking:  resp.Thinking,
	})
}

// chatStream 流式对话
func (h *ChatHandler) chatStream(c *gin.Context, req ChatRequest) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	hosts := h.normalizeHosts(req)
	sessionID := h.normalizeSessionID(req)

	// 调用 Service 层流式接口
	err := h.chatService.ChatStream(c.Request.Context(), service.ChatRequest{
		SessionID: sessionID,
		Message:   req.Message,
		Hosts:     hosts,
		Stream:    true,
	}, func(chunk service.StreamChunk) {
		// 发送 SSE 事件
		switch chunk.Type {
		case "thinking":
			c.SSEvent("thinking", gin.H{
				"step":    chunk.Step,
				"status":  chunk.Status,
				"content": chunk.Content,
			})
		case "content":
			c.SSEvent("content", gin.H{"content": chunk.Content})
		case "tool_call":
			if chunk.ToolCall != nil {
				c.SSEvent("tool_call", gin.H{
					"id":     chunk.ToolCall.ID,
					"tool":   chunk.ToolCall.Tool,
					"params": chunk.ToolCall.Params,
					"status": chunk.Status,
				})
			}
		case "tool_result":
			if chunk.ToolResult != nil {
				c.SSEvent("tool_result", gin.H{
					"id":     chunk.ToolResult.ID,
					"tool":   chunk.ToolResult.Tool,
					"params": chunk.ToolResult.Params,
					"result": chunk.ToolResult.Result,
					"error":  chunk.ToolResult.Error,
				})
			}
		case "done":
			c.SSEvent("done", gin.H{})
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
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || v != 1 {
			limit = 50
		}
	}

	sessions, err := h.chatService.GetSessions(limit)
	if err != nil {
		InternalError(c, "获取会话列表失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"sessions": sessions,
	})
}

// CreateSession 创建新会话
// POST /api/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	var req struct {
		Title string   `json:"title"`
		Hosts []string `json:"hosts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空body
	}

	sessionID, err := h.chatService.CreateSession(req.Title, req.Hosts)
	if err != nil {
		InternalError(c, "创建会话失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"sessionId":  sessionID,
		"session_id": sessionID,
		"title":      req.Title,
	})
}

// GetHistory 获取对话历史
// GET /api/chat/history/:session_id
func (h *ChatHandler) GetHistory(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		// 兼容 query 参数
		sessionID = c.Query("session_id")
	}
	if sessionID == "" {
		ParamError(c, "session_id 不能为空")
		return
	}

	messages, err := h.chatService.GetHistory(sessionID)
	if err != nil {
		NotFound(c, "会话不存在")
		return
	}

	Success(c, gin.H{
		"session_id": sessionID,
		"messages":   messages,
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

	if err := h.chatService.DeleteSession(sessionID); err != nil {
		InternalError(c, "删除会话失败: "+err.Error())
		return
	}

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

// normalizeHosts 标准化主机列表
func (h *ChatHandler) normalizeHosts(req ChatRequest) []string {
	hosts := req.Hosts
	if len(hosts) == 0 {
		if len(req.HostIDs) > 0 {
			hosts = req.HostIDs
		} else if len(req.HostIDs2) > 0 {
			hosts = req.HostIDs2
		}
	}
	return hosts
}

// normalizeSessionID 标准化会话 ID
func (h *ChatHandler) normalizeSessionID(req ChatRequest) string {
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = req.SessionID2
	}
	return sessionID
}
