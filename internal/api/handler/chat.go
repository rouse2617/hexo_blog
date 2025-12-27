package handler

import (
	"fmt"
	"io"
	"strings"
	"time"

	"ai-ops/internal/agent"
	"ai-ops/internal/model"
	"ai-ops/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	// 确保会话存在（如果提供了sessionID）
	if sessionID != "" {
		_, err := h.sessionRepo.GetByID(sessionID)
		if err != nil {
			// 会话不存在，创建新会话
			session := &model.Session{
				ID:        sessionID,
				Title:     strings.TrimSpace(req.Message),
				Hosts:     hosts,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if len(session.Title) > 50 {
				session.Title = session.Title[:50] + "..."
			}
			if session.Title == "" {
				session.Title = fmt.Sprintf("会话 %s", time.Now().Format("2006-01-02 15:04:05"))
			}
			h.sessionRepo.Create(session)
		}
	}

	// 保存用户消息
	if sessionID != "" {
		userMsg := &model.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      model.RoleUser,
			Content:   req.Message,
			CreatedAt: time.Now(),
		}
		h.sessionRepo.AddMessage(sessionID, userMsg)
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

	// 保存助手回复
	if sessionID != "" {
		// 转换 ToolCallRecord 到 model.ToolCall
		toolCalls := make([]model.ToolCall, 0, len(resp.ToolCalls))
		for _, tc := range resp.ToolCalls {
			toolCalls = append(toolCalls, model.ToolCall{
				Tool:   tc.Tool,
				Params: tc.Params,
				Result: tc.Result,
				Error:  tc.Error,
			})
		}
		assistantMsg := &model.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      model.RoleAssistant,
			Content:   resp.Reply,
			ToolCalls: toolCalls,
			CreatedAt: time.Now(),
		}
		h.sessionRepo.AddMessage(sessionID, assistantMsg)

		// 更新会话标题（如果是第一条消息）
		session, _ := h.sessionRepo.GetByID(sessionID)
		if session != nil && (session.Title == "" || strings.HasPrefix(session.Title, "会话 ")) {
			newTitle := strings.TrimSpace(req.Message)
			if len(newTitle) > 50 {
				newTitle = newTitle[:50] + "..."
			}
			if newTitle != "" {
				h.sessionRepo.UpdateTitle(sessionID, newTitle)
			}
		}
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

	// 确保会话存在
	if sessionID != "" {
		_, err := h.sessionRepo.GetByID(sessionID)
		if err != nil {
			// 会话不存在，创建新会话
			session := &model.Session{
				ID:        sessionID,
				Title:     strings.TrimSpace(req.Message),
				Hosts:     hosts,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if len(session.Title) > 50 {
				session.Title = session.Title[:50] + "..."
			}
			if session.Title == "" {
				session.Title = fmt.Sprintf("会话 %s", time.Now().Format("2006-01-02 15:04:05"))
			}
			h.sessionRepo.Create(session)
		}
	}

	// 保存用户消息
	var userMsgID string
	if sessionID != "" {
		userMsgID = uuid.New().String()
		userMsg := &model.Message{
			ID:        userMsgID,
			SessionID: sessionID,
			Role:      model.RoleUser,
			Content:   req.Message,
			CreatedAt: time.Now(),
		}
		h.sessionRepo.AddMessage(sessionID, userMsg)
	}

	// 用于累积助手回复内容
	var assistantContent strings.Builder
	var toolCalls []model.ToolCall

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
			assistantContent.WriteString(chunk.Content)
			c.SSEvent("content", gin.H{"content": chunk.Content})
		case "tool_call":
			c.SSEvent("tool_call", gin.H{
				"id":     chunk.ToolCall.ID,
				"tool":   chunk.ToolCall.Function.Name,
				"params": chunk.ToolCall.Function.Arguments,
				"status": chunk.Status,
			})
		case "tool_result":
			if chunk.ToolResult != nil {
				toolCalls = append(toolCalls, model.ToolCall{
					Tool:   chunk.ToolResult.Tool,
					Params: chunk.ToolResult.Params,
					Result: chunk.ToolResult.Result,
					Error:  chunk.ToolResult.Error,
				})
				c.SSEvent("tool_result", gin.H{
					"id":     chunk.ToolResult.ID,
					"tool":   chunk.ToolResult.Tool,
					"params": chunk.ToolResult.Params,
					"result": chunk.ToolResult.Result,
					"error":  chunk.ToolResult.Error,
				})
			}
		case "done":
			// 保存助手回复
			if sessionID != "" && assistantContent.Len() > 0 {
				assistantMsg := &model.Message{
					ID:        uuid.New().String(),
					SessionID: sessionID,
					Role:      model.RoleAssistant,
					Content:   assistantContent.String(),
					ToolCalls: toolCalls,
					CreatedAt: time.Now(),
				}
				h.sessionRepo.AddMessage(sessionID, assistantMsg)

				// 更新会话标题（如果是第一条消息）
				session, _ := h.sessionRepo.GetByID(sessionID)
				if session != nil && (session.Title == "" || strings.HasPrefix(session.Title, "会话 ")) {
					newTitle := strings.TrimSpace(req.Message)
					if len(newTitle) > 50 {
						newTitle = newTitle[:50] + "..."
					}
					if newTitle != "" {
						h.sessionRepo.UpdateTitle(sessionID, newTitle)
					}
				}
			}
			c.SSEvent("done", gin.H{"session_id": sessionID})
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
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if v, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || v != 1 {
			limit = 50
		}
	}

	sessions, err := h.sessionRepo.List(limit)
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

	// 生成会话 ID
	sessionID := uuid.New().String()

	// 生成标题（如果没有提供）
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("会话 %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	// 保存到数据库
	session := &model.Session{
		ID:        sessionID,
		Title:     title,
		Hosts:     req.Hosts,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.sessionRepo.Create(session); err != nil {
		InternalError(c, "创建会话失败: "+err.Error())
		return
	}

	Success(c, gin.H{
		"sessionId": sessionID,
		"session_id": sessionID,
		"title":     title,
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

	// 检查会话是否存在
	_, err := h.sessionRepo.GetByID(sessionID)
	if err != nil {
		NotFound(c, "会话不存在")
		return
	}

	// 从数据库获取对话历史
	messages, err := h.sessionRepo.GetMessages(sessionID)
	if err != nil {
		InternalError(c, "获取对话历史失败: "+err.Error())
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

	// 从数据库删除会话（会自动删除关联的消息）
	if err := h.sessionRepo.Delete(sessionID); err != nil {
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
