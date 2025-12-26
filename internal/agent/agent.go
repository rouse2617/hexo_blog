package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// Agent AI 运维助手
type Agent struct {
	llmClient    llm.Client
	toolRegistry *tool.Registry
	sshPool      *ssh.Pool
	maxLoops     int
	timeout      time.Duration
}

// Config Agent 配置
type Config struct {
	MaxLoops int
	Timeout  time.Duration
}

// NewAgent 创建 Agent
func NewAgent(llmClient llm.Client, toolRegistry *tool.Registry, sshPool *ssh.Pool, cfg Config) *Agent {
	if cfg.MaxLoops == 0 {
		cfg.MaxLoops = 10
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}

	return &Agent{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
		sshPool:      sshPool,
		maxLoops:     cfg.MaxLoops,
		timeout:      cfg.Timeout,
	}
}

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID string   // 会话 ID
	Message   string   // 用户消息
	Hosts     []string // 关联主机
	History   []llm.Message // 历史消息
}

// ChatResponse 对话响应
type ChatResponse struct {
	SessionID  string      // 会话 ID
	Reply      string      // 回复内容
	ToolCalls  []ToolCallRecord // 工具调用记录
	Thinking   string      // 思考过程（可选）
	Error      string      // 错误信息
}

// ToolCallRecord 工具调用记录
type ToolCallRecord struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
	Result string                 `json:"result"`
	Error  string                 `json:"error,omitempty"`
}

// Chat 对话
func (a *Agent) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 构建消息列表
	messages := a.buildMessages(req)

	// 获取工具定义
	tools := a.toolRegistry.GenerateJSONSchema()
	toolDefs := make([]llm.ToolDef, len(tools))
	for i, t := range tools {
		toolDefs[i] = llm.ToolDef{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t["function"].(map[string]interface{})["name"].(string),
				Description: t["function"].(map[string]interface{})["description"].(string),
				Parameters:  t["function"].(map[string]interface{})["parameters"].(map[string]interface{}),
			},
		}
	}

	// Agent 循环
	var toolCallRecords []ToolCallRecord
	var finalReply string

	for i := 0; i < a.maxLoops; i++ {
		logger.Debug("Agent 循环", zap.Int("loop", i+1), zap.Int("messages", len(messages)))

		// 调用 LLM
		resp, err := a.llmClient.ChatWithTools(ctx, messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}

		// 检查是否有工具调用
		if resp.HasToolCalls() {
			// 处理工具调用
			toolResults, records := a.executeToolCalls(ctx, resp.Message.ToolCalls, req.Hosts)
			toolCallRecords = append(toolCallRecords, records...)

			// 添加助手消息（包含工具调用）
			messages = append(messages, resp.Message)

			// 添加工具结果消息
			messages = append(messages, toolResults...)

			logger.Debug("工具调用完成", zap.Int("tools", len(records)))
			continue
		}

		// 没有工具调用，返回最终回复
		finalReply = resp.Message.Content
		break
	}

	// 如果达到最大循环次数但没有最终回复
	if finalReply == "" && len(toolCallRecords) > 0 {
		finalReply = "已执行相关操作，请查看工具调用结果。"
	}

	return &ChatResponse{
		SessionID: req.SessionID,
		Reply:     finalReply,
		ToolCalls: toolCallRecords,
	}, nil
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages(req ChatRequest) []llm.Message {
	messages := make([]llm.Message, 0)

	// 系统提示词
	systemPrompt := BuildSystemPromptWithHosts(a.toolRegistry, req.Hosts)
	messages = append(messages, llm.NewSystemMessage(systemPrompt))

	// 历史消息
	messages = append(messages, req.History...)

	// 用户消息
	messages = append(messages, llm.NewUserMessage(req.Message))

	return messages
}

// executeToolCalls 执行工具调用
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	var toolMessages []llm.Message
	var records []ToolCallRecord

	// 创建工具执行上下文
	toolCtx := &tool.Context{
		SSH:     a.sshPool,
		Timeout: 30 * time.Second,
	}

	for _, tc := range toolCalls {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			logger.Error("解析工具参数失败", zap.Error(err), zap.String("tool", tc.Function.Name))

			record := ToolCallRecord{
				Tool:   tc.Function.Name,
				Params: nil,
				Error:  fmt.Sprintf("参数解析失败: %v", err),
			}
			records = append(records, record)

			toolMessages = append(toolMessages, llm.NewToolMessage(
				tc.ID,
				tc.Function.Name,
				fmt.Sprintf("错误: 参数解析失败 - %v", err),
			))
			continue
		}

		logger.Info("执行工具",
			zap.String("tool", tc.Function.Name),
			zap.Any("params", params),
		)

		// 执行工具
		result, err := a.toolRegistry.Execute(toolCtx, tc.Function.Name, params)

		record := ToolCallRecord{
			Tool:   tc.Function.Name,
			Params: params,
		}

		var resultStr string
		if err != nil {
			record.Error = err.Error()
			resultStr = fmt.Sprintf("执行错误: %v", err)
		} else if !result.Success {
			record.Error = result.Error
			resultStr = fmt.Sprintf("执行失败: %s", result.Error)
		} else {
			// 序列化结果
			resultBytes, _ := json.Marshal(result.Data)
			record.Result = string(resultBytes)
			resultStr = fmt.Sprintf("执行成功: %s\n结果: %s", result.Message, string(resultBytes))
		}

		records = append(records, record)
		toolMessages = append(toolMessages, llm.NewToolMessage(tc.ID, tc.Function.Name, resultStr))
	}

	return toolMessages, records
}

// ChatStream 流式对话
func (a *Agent) ChatStream(ctx context.Context, req ChatRequest, callback func(chunk StreamChunk)) error {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 构建消息列表
	messages := a.buildMessages(req)

	// 获取工具定义
	tools := a.toolRegistry.GenerateJSONSchema()
	toolDefs := make([]llm.ToolDef, len(tools))
	for i, t := range tools {
		toolDefs[i] = llm.ToolDef{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t["function"].(map[string]interface{})["name"].(string),
				Description: t["function"].(map[string]interface{})["description"].(string),
				Parameters:  t["function"].(map[string]interface{})["parameters"].(map[string]interface{}),
			},
		}
	}

	// Agent 循环
	for i := 0; i < a.maxLoops; i++ {
		var contentBuffer string
		var toolCalls []llm.ToolCall
		done := false

		// 流式调用 LLM
		err := a.llmClient.ChatStreamWithTools(ctx, messages, toolDefs, func(chunk llm.StreamChunk) {
			switch chunk.Type {
			case "content":
				contentBuffer += chunk.Content
				callback(StreamChunk{
					Type:    "content",
					Content: chunk.Content,
				})
			case "tool_call":
				if chunk.ToolCall != nil {
					toolCalls = append(toolCalls, *chunk.ToolCall)
				}
			case "done":
				done = true
			}
		})

		if err != nil {
			return fmt.Errorf("LLM 流式调用失败: %w", err)
		}

		// 检查是否有工具调用
		if len(toolCalls) > 0 {
			// 通知前端工具调用
			for _, tc := range toolCalls {
				callback(StreamChunk{
					Type:     "tool_call",
					ToolCall: &tc,
				})
			}

			// 执行工具
			toolResults, records := a.executeToolCalls(ctx, toolCalls, req.Hosts)

			// 通知前端工具结果
			for _, r := range records {
				callback(StreamChunk{
					Type:       "tool_result",
					ToolResult: &r,
				})
			}

			// 添加消息继续对话
			messages = append(messages, llm.NewAssistantToolCallMessage(toolCalls))
			messages = append(messages, toolResults...)
			continue
		}

		// 没有工具调用，对话结束
		if done {
			callback(StreamChunk{Type: "done"})
			break
		}
	}

	return nil
}

// StreamChunk 流式数据块
type StreamChunk struct {
	Type       string          `json:"type"` // content, tool_call, tool_result, done, error
	Content    string          `json:"content,omitempty"`
	ToolCall   *llm.ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolCallRecord `json:"tool_result,omitempty"`
	Error      string          `json:"error,omitempty"`
}
