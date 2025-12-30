package agent

import (
	"context"
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
	llmClient        llm.Client
	toolRegistry     *tool.Registry
	sshPool          *ssh.Pool
	maxLoops         int
	timeout          time.Duration
	promptVersion    string // 提示词版本: standard / enhanced
	errorRecovery    *ErrorRecoveryEngine
	reactAgent       *ReActAgent          // ReAct Agent 实例
	parallelExecutor *ParallelExecutor    // 并行执行器
	toolCache        *ToolCache           // 工具缓存
}

// Config Agent 配置
type Config struct {
	MaxLoops       int
	Timeout        time.Duration
	PromptVersion  string        // 提示词版本: standard / enhanced
	EnableThinking bool          // 启用思考过程
	CacheTTL       time.Duration // 工具缓存 TTL
	MaxConcurrent  int           // 最大并发工具执行数
}

// NewAgent 创建 Agent
func NewAgent(llmClient llm.Client, toolRegistry *tool.Registry, sshPool *ssh.Pool, cfg Config) *Agent {
	if cfg.MaxLoops == 0 {
		cfg.MaxLoops = 10
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "enhanced" // 默认使用增强版
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 30 * time.Second
	}
	if cfg.MaxConcurrent == 0 {
		cfg.MaxConcurrent = 10
	}

	// 创建工具缓存
	toolCache := NewToolCache(cfg.CacheTTL)

	// 创建并行执行器
	parallelExecutor := NewParallelExecutor(toolRegistry, sshPool, toolCache)
	parallelExecutor.SetMaxConcurrent(cfg.MaxConcurrent)

	return &Agent{
		llmClient:        llmClient,
		toolRegistry:     toolRegistry,
		sshPool:          sshPool,
		maxLoops:         cfg.MaxLoops,
		timeout:          cfg.Timeout,
		promptVersion:    cfg.PromptVersion,
		errorRecovery:    NewErrorRecoveryEngine(),
		reactAgent:       NewReActAgent(toolRegistry, llmClient, sshPool),
		parallelExecutor: parallelExecutor,
		toolCache:        toolCache,
	}
}

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID string        // 会话 ID
	Message   string        // 用户消息
	Hosts     []string      // 关联主机
	History   []llm.Message // 历史消息
}

// ChatResponse 对话响应
type ChatResponse struct {
	SessionID  string           // 会话 ID
	Reply      string           // 回复内容
	ToolCalls  []ToolCallRecord // 工具调用记录
	Thinking   string           // 思考过程（可选）
	Error      string           // 错误信息
	ReActSteps []ReActStep      // ReAct 步骤（可选）
}

// ToolCallRecord 工具调用记录
type ToolCallRecord struct {
	ID     string                 `json:"id,omitempty"` // 工具调用ID，用于匹配
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
	Result string                 `json:"result"`
	Error  string                 `json:"error,omitempty"`
}

// Chat 对话
func (a *Agent) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("Agent 未初始化")
	}
	if a.llmClient == nil || a.toolRegistry == nil {
		return nil, fmt.Errorf("Agent 依赖未初始化")
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 使用动态 Prompt 构建器（仅在 enhanced 模式下）
	var systemPrompt string
	if a.promptVersion == "enhanced" {
		promptBuilder := NewDynamicPromptBuilder(a.toolRegistry, req.Hosts)
		systemPrompt = promptBuilder.BuildDynamicPrompt(req.Message)

		logger.Info("动态 Prompt 构建",
			zap.String("task_type", promptBuilder.GetTaskType()),
			zap.Int("complexity", promptBuilder.GetComplexity()),
		)
	} else {
		systemPrompt = SelectSystemPrompt(a.promptVersion, a.toolRegistry, req.Hosts)
	}

	// 构建消息列表
	messages := a.buildMessagesWithPrompt(req, systemPrompt)

	// 获取工具定义
	toolDefs := a.buildToolDefinitions()

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

	// 系统提示词（根据配置选择版本）
	systemPrompt := SelectSystemPrompt(a.promptVersion, a.toolRegistry, req.Hosts)
	messages = append(messages, llm.NewSystemMessage(systemPrompt))

	// 历史消息
	messages = append(messages, req.History...)

	// 用户消息
	messages = append(messages, llm.NewUserMessage(req.Message))

	return messages
}

// buildMessagesWithPrompt 使用指定的 system prompt 构建消息列表
func (a *Agent) buildMessagesWithPrompt(req ChatRequest, systemPrompt string) []llm.Message {
	messages := make([]llm.Message, 0, len(req.History)+2)

	// 系统提示词
	messages = append(messages, llm.NewSystemMessage(systemPrompt))

	// 历史消息
	messages = append(messages, req.History...)

	// 用户消息
	messages = append(messages, llm.NewUserMessage(req.Message))

	return messages
}

// buildToolDefinitions 构建 LLM 工具定义
func (a *Agent) buildToolDefinitions() []llm.ToolDef {
	tools := a.toolRegistry.GenerateJSONSchema()
	toolDefs := make([]llm.ToolDef, 0, len(tools))

	for _, t := range tools {
		function, ok := t["function"].(map[string]interface{})
		if !ok {
			continue
		}

		name, nameOk := function["name"].(string)
		description, descOk := function["description"].(string)
		parameters, paramsOk := function["parameters"].(map[string]interface{})

		if !nameOk || !descOk || !paramsOk {
			continue
		}

		toolDefs = append(toolDefs, llm.ToolDef{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        name,
				Description: description,
				Parameters:  parameters,
			},
		})
	}

	return toolDefs
}

// executeToolCalls 执行工具调用（带错误恢复和并行执行）
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	// 使用并行执行器
	toolMessages, records := a.parallelExecutor.ExecuteParallel(ctx, toolCalls, hosts)

	// 对失败的工具调用尝试错误恢复
	for i, record := range records {
		if record.Error == "" {
			continue
		}

		shouldRetry, newCalls, recoveryPrompt, _ := a.errorRecovery.RecoverFromError(ctx, a, record, nil)

		if recoveryPrompt != "" {
			logger.Debug("错误恢复提示", zap.String("tool", record.Tool), zap.String("prompt", recoveryPrompt))
		}

		// 尝试替代工具
		if shouldRetry && len(newCalls) > 0 {
			altMessages, altRecords := a.parallelExecutor.ExecuteParallel(ctx, newCalls, hosts)
			for j, altRecord := range altRecords {
				if altRecord.Error == "" {
					// 替代工具成功
					records[i].Result = fmt.Sprintf("原工具失败，使用替代方案 %s 成功：%s", altRecord.Tool, altRecord.Result)
					records[i].Error = ""
					toolMessages[i] = altMessages[j]

					a.errorRecovery.LogRecovery(RecoveryLog{
						Timestamp:   time.Now(),
						ToolName:    record.Tool,
						Error:       record.Error,
						Strategy:    "alternative_tool",
						Success:     true,
						Alternative: altRecord.Tool,
					})
					break
				}
			}
		}
	}

	return toolMessages, records
}

// ChatStream 流式对话
func (a *Agent) ChatStream(ctx context.Context, req ChatRequest, callback func(chunk StreamChunk)) error {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// 使用动态 Prompt 构建器（仅在 enhanced 模式下）
	var systemPrompt string
	if a.promptVersion == "enhanced" {
		promptBuilder := NewDynamicPromptBuilder(a.toolRegistry, req.Hosts)
		systemPrompt = promptBuilder.BuildDynamicPrompt(req.Message)

		logger.Info("动态 Prompt 构建（流式）",
			zap.String("task_type", promptBuilder.GetTaskType()),
			zap.Int("complexity", promptBuilder.GetComplexity()),
		)
	} else {
		systemPrompt = SelectSystemPrompt(a.promptVersion, a.toolRegistry, req.Hosts)
	}

	// 构建消息列表
	messages := a.buildMessagesWithPrompt(req, systemPrompt)

	// 获取工具定义
	toolDefs := a.buildToolDefinitions()

	// Agent 循环
	for i := 0; i < a.maxLoops; i++ {
		var contentBuffer string
		var toolCalls []llm.ToolCall
		done := false

		// 通知前端：正在调用 LLM
		callback(StreamChunk{
			Type:    "thinking",
			Step:    i + 1,
			Status:  "calling_llm",
			Content: "正在分析您的问题...",
		})

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
			// 通知前端：准备执行工具
			callback(StreamChunk{
				Type:    "thinking",
				Step:    i + 1,
				Status:  "executing_tools",
				Content: fmt.Sprintf("需要执行 %d 个工具来获取信息...", len(toolCalls)),
			})

			// 通知前端工具调用
			for _, tc := range toolCalls {
				callback(StreamChunk{
					Type:     "tool_call",
					ToolCall: &tc,
					Status:   "pending",
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

			// 通知前端：工具执行完成，继续分析
			callback(StreamChunk{
				Type:    "thinking",
				Step:    i + 1,
				Status:  "analyzing_results",
				Content: "正在分析工具执行结果...",
			})

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
	Type       string          `json:"type"` // content, tool_call, tool_result, thinking, done, error
	Content    string          `json:"content,omitempty"`
	ToolCall   *llm.ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolCallRecord `json:"tool_result,omitempty"`
	Error      string          `json:"error,omitempty"`
	Step       int             `json:"step,omitempty"`   // 当前步骤
	Status     string          `json:"status,omitempty"` // 状态描述
}

// ChatWithReAct 使用 ReAct 模式进行对话
// ReAct (Reasoning + Acting) 模式通过显式的思考-行动-观察循环提供更透明的推理过程
func (a *Agent) ChatWithReAct(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if a.reactAgent == nil {
		return nil, fmt.Errorf("ReAct Agent 未初始化")
	}
	return a.reactAgent.ChatReAct(ctx, req)
}



// ClearCache 清空工具缓存
func (a *Agent) ClearCache() {
	if a.toolCache != nil {
		a.toolCache.Clear()
	}
}

// CacheStats 获取缓存统计
func (a *Agent) CacheStats() (size int, hits int) {
	if a.toolCache != nil {
		return a.toolCache.Stats()
	}
	return 0, 0
}
