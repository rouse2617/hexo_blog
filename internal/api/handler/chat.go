package handler

import (
	"fmt"
	"io"
	"time"

	"ai-ops/internal/agent"
	"ai-ops/internal/repository"

	"github.com/gin-gonic/gin"
)

// ChatHandler 对话处理器
type ChatHandler struct {
	agent       *agent.Agent
	sessionRepo repository.SessionRepository
}

// NewChatHandler 创建对话处理器
func NewChatHandler(agent *agent.Agent, sessionRepo repository.SessionRepository) *ChatHandler {
	return &ChatHandler{
		agent:       agent,
		sessionRepo: sessionRepo,
	}
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

	hosts := req.Hosts
	if len(hosts) == 0 {
		if len(req.HostIDs) > 0 {
			hosts = req.HostIDs
		} else if len(req.HostIDs2) > 0 {
			hosts = req.HostIDs2
		}
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = req.SessionID2
	}

	// 流式响应
	if req.Stream {
		h.chatStream(c, req)
		return
	}

	// 普通响应
	resp, err := h.agent.Chat(c.Request.Context(), agent.ChatRequest{
		SessionID: sessionID,
		Message:   req.Message,
		Hosts:     hosts,
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
	hosts := req.Hosts
	if len(hosts) == 0 {
		if len(req.HostIDs) > 0 {
			hosts = req.HostIDs
		} else if len(req.HostIDs2) > 0 {
			hosts = req.HostIDs2
		}
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = req.SessionID2
	}
	err := h.agent.ChatStream(c.Request.Context(), agent.ChatRequest{
		SessionID: sessionID,
		Message:   req.Message,
		Hosts:     hosts,
	}, func(chunk agent.StreamChunk) {
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
			c.SSEvent("tool_call", gin.H{
				"id":     chunk.ToolCall.ID,
				"tool":   chunk.ToolCall.Function.Name,
				"params": chunk.ToolCall.Function.Arguments,
				"status": chunk.Status,
			})
		case "tool_result":
			c.SSEvent("tool_result", gin.H{
				"tool":   chunk.ToolResult.Tool,
				"params": chunk.ToolResult.Params,
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
	// 目前返回空数组
	Success(c, gin.H{
		"sessions": []interface{}{},
	})
}

// CreateSession 创建新会话
// POST /api/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	// 生成会话 ID
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	// TODO: 保存到数据库

	Success(c, gin.H{
		"sessionId": sessionID,
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

	// TODO: 从数据库获取对话历史
	// 目前返回空数组
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
