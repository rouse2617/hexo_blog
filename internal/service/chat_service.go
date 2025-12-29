package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/agent"
	"ai-ops/internal/model"
	"ai-ops/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ChatService 聊天服务
type ChatService struct {
	agent          *agent.Agent
	sessionRepo    repository.SessionRepository
	enableThinking bool
	enableReAct    bool // 是否启用 ReAct 模式
}

// NewChatService 创建聊天服务
func NewChatService(agent *agent.Agent, sessionRepo repository.SessionRepository, enableThinking bool) *ChatService {
	return &ChatService{
		agent:          agent,
		sessionRepo:    sessionRepo,
		enableThinking: enableThinking,
	}
}

// SetEnableThinking 设置是否启用思考过程
func (s *ChatService) SetEnableThinking(enable bool) {
	s.enableThinking = enable
}

// SetEnableReAct 设置是否启用 ReAct 模式
func (s *ChatService) SetEnableReAct(enable bool) {
	s.enableReAct = enable
}


// ChatRequest 聊天请求
type ChatRequest struct {
	SessionID string
	Message   string
	Hosts     []string
	Stream    bool
}

// ChatResponse 聊天响应
type ChatResponse struct {
	SessionID string
	Reply     string
	ToolCalls []ToolCallRecord
	Thinking  string
}

// ToolCallRecord 工具调用记录（导出版本）
type ToolCallRecord = agent.ToolCallRecord

// Chat 执行聊天
func (s *ChatService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 标准化请求
	req = s.normalizeRequest(req)

	// 确保会话存在
	if err := s.ensureSessionExists(req); err != nil {
		return nil, fmt.Errorf("确保会话存在失败: %w", err)
	}

	// 保存用户消息
	if err := s.saveUserMessage(req); err != nil {
		return nil, fmt.Errorf("保存用户消息失败: %w", err)
	}

	// 调用 Agent
	agentResp, err := s.callAgent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("调用 Agent 失败: %w", err)
	}

	// 保存助手回复
	if err := s.saveAssistantMessage(req, agentResp); err != nil {
		return nil, fmt.Errorf("保存助手消息失败: %w", err)
	}

	// 更新会话标题
	s.updateSessionTitleIfNeeded(req)

	return &ChatResponse{
		SessionID: agentResp.SessionID,
		Reply:     agentResp.Reply,
		ToolCalls: agentResp.ToolCalls,
		Thinking:  agentResp.Thinking,
	}, nil
}

// ChatStream 流式聊天
func (s *ChatService) ChatStream(ctx context.Context, req ChatRequest, callback func(chunk StreamChunk)) error {
	// 标准化请求
	req = s.normalizeRequest(req)

	// 确保会话存在
	if err := s.ensureSessionExists(req); err != nil {
		return fmt.Errorf("确保会话存在失败: %w", err)
	}

	// 保存用户消息
	userMsgID := uuid.New().String()
	if err := s.saveUserMessageWithID(req, userMsgID); err != nil {
		return fmt.Errorf("保存用户消息失败: %w", err)
	}

	// 用于累积助手回复
	var assistantContent strings.Builder
	var toolCalls []model.ToolCall

	// 流式处理回调
	streamCallback := func(chunk agent.StreamChunk) {
		switch chunk.Type {
		case "thinking":
			if s.enableThinking {
				callback(StreamChunk{
					Type:    "thinking",
					Step:    chunk.Step,
					Status:  chunk.Status,
					Content: chunk.Content,
				})
			}
		case "content":
			assistantContent.WriteString(chunk.Content)
			callback(StreamChunk{
				Type:    "content",
				Content: chunk.Content,
			})
		case "tool_call":
			// 转换 agent.ToolCall 到 service.ToolCallRecord
			if chunk.ToolCall != nil {
				toolCallRecord := ToolCallRecord{
					ID:     chunk.ToolCall.ID,
					Tool:   chunk.ToolCall.Function.Name,
					Params: make(map[string]interface{}),
				}
				// 解析 JSON 参数
				if chunk.ToolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(chunk.ToolCall.Function.Arguments), &toolCallRecord.Params)
				}

				callback(StreamChunk{
					Type:     "tool_call",
					ToolCall: &toolCallRecord,
					Status:   chunk.Status,
				})
			}
		case "tool_result":
			if chunk.ToolResult != nil {
				toolCalls = append(toolCalls, model.ToolCall{
					Tool:   chunk.ToolResult.Tool,
					Params: chunk.ToolResult.Params,
					Result: chunk.ToolResult.Result,
					Error:  chunk.ToolResult.Error,
				})
				callback(StreamChunk{
					Type:       "tool_result",
					ToolResult: chunk.ToolResult,
				})
			}
		case "done":
			// 保存助手回复
			if assistantContent.Len() > 0 {
				assistantMsg := &model.Message{
					ID:        uuid.New().String(),
					SessionID: req.SessionID,
					Role:      model.RoleAssistant,
					Content:   assistantContent.String(),
					ToolCalls: toolCalls,
					CreatedAt: time.Now(),
				}
				if err := s.sessionRepo.AddMessage(req.SessionID, assistantMsg); err != nil {
					// 记录错误但不中断流
					zap.Error(fmt.Errorf("保存助手消息失败: %w", err))
				}
			}
			// 更新会话标题
			s.updateSessionTitleIfNeeded(req)
			callback(StreamChunk{Type: "done"})
		case "error":
			callback(StreamChunk{
				Type:  "error",
				Error: chunk.Error,
			})
		}
	}

	// 调用 Agent 流式接口
	var err error
	if s.enableThinking {
		err = s.agent.ChatStreamWithThinking(ctx, agent.ChatRequest{
			SessionID: req.SessionID,
			Message:   req.Message,
			Hosts:     req.Hosts,
		}, streamCallback)
	} else {
		err = s.agent.ChatStream(ctx, agent.ChatRequest{
			SessionID: req.SessionID,
			Message:   req.Message,
			Hosts:     req.Hosts,
		}, streamCallback)
	}

	return err
}

// StreamChunk 流式数据块
type StreamChunk struct {
	Type       string         `json:"type"`
	Content    string         `json:"content,omitempty"`
	ToolCall   *ToolCallRecord `json:"tool_call,omitempty"`
	ToolResult *ToolCallRecord `json:"tool_result,omitempty"`
	Error      string         `json:"error,omitempty"`
	Step       int            `json:"step,omitempty"`
	Status     string         `json:"status,omitempty"`
}

// normalizeRequest 标准化请求
func (s *ChatService) normalizeRequest(req ChatRequest) ChatRequest {
	// 去除消息前后空格
	req.Message = strings.TrimSpace(req.Message)
	return req
}

// ensureSessionExists 确保会话存在
func (s *ChatService) ensureSessionExists(req ChatRequest) error {
	if req.SessionID == "" {
		return nil
	}

	_, err := s.sessionRepo.GetByID(req.SessionID)
	if err == nil {
		return nil // 会话已存在
	}

	// 创建新会话
	session := &model.Session{
		ID:        req.SessionID,
		Title:     s.generateSessionTitle(req.Message),
		Hosts:     req.Hosts,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	return s.sessionRepo.Create(session)
}

// generateSessionTitle 生成会话标题
func (s *ChatService) generateSessionTitle(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 50 {
		return message[:50] + "..."
	}
	if message == "" {
		return fmt.Sprintf("会话 %s", time.Now().Format("2006-01-02 15:04:05"))
	}
	return message
}

// saveUserMessage 保存用户消息
func (s *ChatService) saveUserMessage(req ChatRequest) error {
	if req.SessionID == "" {
		return nil
	}

	userMsg := &model.Message{
		ID:        uuid.New().String(),
		SessionID: req.SessionID,
		Role:      model.RoleUser,
		Content:   req.Message,
		CreatedAt: time.Now(),
	}
	return s.sessionRepo.AddMessage(req.SessionID, userMsg)
}

// saveUserMessageWithID 保存用户消息（指定ID）
func (s *ChatService) saveUserMessageWithID(req ChatRequest, msgID string) error {
	if req.SessionID == "" {
		return nil
	}

	userMsg := &model.Message{
		ID:        msgID,
		SessionID: req.SessionID,
		Role:      model.RoleUser,
		Content:   req.Message,
		CreatedAt: time.Now(),
	}
	return s.sessionRepo.AddMessage(req.SessionID, userMsg)
}

// callAgent 调用 Agent
func (s *ChatService) callAgent(ctx context.Context, req ChatRequest) (*agent.ChatResponse, error) {
	// 优先级: ReAct > Thinking > 普通
	if s.enableReAct {
		return s.agent.ChatWithReAct(ctx, agent.ChatRequest{
			SessionID: req.SessionID,
			Message:   req.Message,
			Hosts:     req.Hosts,
		})
	}

	if s.enableThinking {
		return s.agent.ChatWithThinking(ctx, agent.ChatRequest{
			SessionID: req.SessionID,
			Message:   req.Message,
			Hosts:     req.Hosts,
		})
	}

	return s.agent.Chat(ctx, agent.ChatRequest{
		SessionID: req.SessionID,
		Message:   req.Message,
		Hosts:     req.Hosts,
	})
}

// saveAssistantMessage 保存助手消息
func (s *ChatService) saveAssistantMessage(req ChatRequest, resp *agent.ChatResponse) error {
	if req.SessionID == "" {
		return nil
	}

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
		SessionID: req.SessionID,
		Role:      model.RoleAssistant,
		Content:   resp.Reply,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
	}
	return s.sessionRepo.AddMessage(req.SessionID, assistantMsg)
}

// updateSessionTitleIfNeeded 更新会话标题（如果需要）
func (s *ChatService) updateSessionTitleIfNeeded(req ChatRequest) {
	if req.SessionID == "" {
		return
	}

	session, err := s.sessionRepo.GetByID(req.SessionID)
	if err != nil {
		return
	}

	// 仅当标题为默认值时才更新
	if session.Title == "" || strings.HasPrefix(session.Title, "会话 ") {
		newTitle := s.generateSessionTitle(req.Message)
		if newTitle != "" {
			_ = s.sessionRepo.UpdateTitle(req.SessionID, newTitle)
		}
	}
}

// GetSessions 获取会话列表
func (s *ChatService) GetSessions(limit int) ([]*model.Session, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.sessionRepo.List(limit)
}

// CreateSession 创建新会话
func (s *ChatService) CreateSession(title string, hosts []string) (string, error) {
	sessionID := uuid.New().String()

	if title == "" {
		title = fmt.Sprintf("会话 %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	session := &model.Session{
		ID:        sessionID,
		Title:     title,
		Hosts:     hosts,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return "", err
	}

	return sessionID, nil
}

// GetHistory 获取对话历史
func (s *ChatService) GetHistory(sessionID string) ([]*model.Message, error) {
	// 检查会话是否存在
	_, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("会话不存在: %w", err)
	}

	return s.sessionRepo.GetMessages(sessionID)
}

// DeleteSession 删除会话
func (s *ChatService) DeleteSession(sessionID string) error {
	return s.sessionRepo.Delete(sessionID)
}
