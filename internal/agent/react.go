package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// ReActAgent 基于 ReAct (Reasoning + Acting) 模式的 Agent
// ReAct 模式: Thought -> Action -> Observation 循环
type ReActAgent struct {
	registry *tool.Registry
	llmClient llm.Client
	sshPool   *ssh.Pool
	maxLoops  int
	timeout   time.Duration
}

// ReActStep ReAct 步骤
type ReActStep struct {
	Phase      string             `json:"phase"`      // thought, action, observation
	Content    string             `json:"content"`
	ToolCalls  []llm.ToolCall     `json:"tool_calls,omitempty"`
	ToolResult string             `json:"tool_result,omitempty"`
	Timestamp  int64              `json:"timestamp"`
}

// NewReActAgent 创建 ReAct Agent
func NewReActAgent(registry *tool.Registry, llmClient llm.Client, sshPool *ssh.Pool) *ReActAgent {
	return &ReActAgent{
		registry:  registry,
		llmClient: llmClient,
		sshPool:   sshPool,
		maxLoops:  10,
		timeout:   5 * time.Minute,
	}
}

// ChatReAct 使用 ReAct 模式进行对话
func (a *ReActAgent) ChatReAct(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	var reactSteps []ReActStep
	var toolCallRecords []ToolCallRecord

	// 构建 ReAct 系统提示
	systemPrompt := a.buildReActSystemPrompt()
	messages := []llm.Message{
		llm.NewSystemMessage(systemPrompt),
		llm.NewUserMessage(req.Message),
	}

	tools := a.registry.GenerateJSONSchema()
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

	var finalResp *llm.ChatResponse

	for i := 0; i < a.maxLoops; i++ {
		logger.Info("ReAct 循环", zap.Int("loop", i+1))

		// 调用 LLM
		resp, err := a.llmClient.ChatWithTools(ctx, messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}
		finalResp = resp

		// 记录思考步骤
		thought := strings.TrimSpace(resp.Message.Content)
		if thought != "" {
			reactSteps = append(reactSteps, ReActStep{
				Phase:     "thought",
				Content:   thought,
				Timestamp: time.Now().Unix(),
			})
			logger.Info("ReAct 思考", zap.String("thought", thought))
		}

		// 如果没有工具调用，说明准备好给出最终答案
		if !resp.HasToolCalls() {
			logger.Info("ReAct 完成，无工具调用")
			break
		}

		// 记录行动步骤
		reactSteps = append(reactSteps, ReActStep{
			Phase:     "action",
			Content:   fmt.Sprintf("执行 %d 个工具调用", len(resp.Message.ToolCalls)),
			ToolCalls: resp.Message.ToolCalls,
			Timestamp: time.Now().Unix(),
		})
		logger.Info("ReAct 行动", zap.Int("tool_calls", len(resp.Message.ToolCalls)))

		// 执行工具调用
		toolResults, records := a.executeToolCalls(ctx, resp.Message.ToolCalls, req.Hosts)
		toolCallRecords = append(toolCallRecords, records...)

		// 记录观察结果
		for _, record := range records {
			var resultStr string
			if record.Error != "" {
				resultStr = fmt.Sprintf("错误: %s", record.Error)
			} else {
				// 限制结果长度
				resultPreview := record.Result
				if len(resultPreview) > 500 {
					resultPreview = resultPreview[:500] + "...(truncated)"
				}
				resultStr = fmt.Sprintf("成功: %s", resultPreview)
			}

			reactSteps = append(reactSteps, ReActStep{
				Phase:      "observation",
				Content:    resultStr,
				Timestamp:  time.Now().Unix(),
			})
			logger.Info("ReAct 观察", zap.String("tool", record.Tool), zap.String("result", resultStr))
		}

		// 将助手消息和工具结果添加到对话历史
		messages = append(messages, resp.Message)
		messages = append(messages, toolResults...)
	}

	// 生成最终回复（包含完整的 ReAct 过程）
	var finalReply string
	if finalResp != nil {
		finalReply = finalResp.Message.Content
	}

	// 如果没有最终回复但有 ReAct 步骤，生成格式化输出
	if finalReply == "" && len(reactSteps) > 0 {
		finalReply = a.formatReActResponse(reactSteps, "已完成所有工具调用，请查看上述详细过程。")
	} else if len(reactSteps) > 0 {
		// 如果有最终回复，可以选择是否包含完整的 ReAct 过程
		// 这里我们只返回最终回复，但保留 ReAct 步骤在前端展示
	}

	return &ChatResponse{
		SessionID:  req.SessionID,
		Reply:      finalReply,
		ToolCalls:  toolCallRecords,
		Thinking:   a.formatReActSteps(reactSteps),
		ReActSteps: reactSteps,
	}, nil
}

// buildReActSystemPrompt 构建 ReAct 系统提示
func (a *ReActAgent) buildReActSystemPrompt() string {
	toolDesc := a.registry.GeneratePrompt()

	return fmt.Sprintf(`# AI-Ops ReAct Agent

你是一个使用 ReAct (Reasoning + Acting) 模式的智能运维助手。

## 工作模式

对于每个问题，你必须按以下循环进行：

1. **Thought（思考）**：明确当前知道什么，需要什么信息
2. **Action（行动）**：选择合适的工具收集信息或执行操作
3. **Observation（观察）**：分析工具返回的结果
4. **循环**：如果信息不足，回到 Thought；如果充足，给出答案

## 思考模板

在 Thought 阶段，你应该明确说明：
- **已知信息**：我已经知道什么
- **需要信息**：我还需要知道什么
- **下一步行动**：我应该调用什么工具，为什么

示例：
"已知用户询问服务器 CPU 使用率，但不知道具体主机信息。需要先获取主机列表，然后检查各主机的 CPU 状态。下一步：调用 list_hosts 工具。"

## 行动原则

1. **一次只调用必要的工具**：不要盲目调用多个工具
2. **优先使用批量参数**：如果工具支持 hosts 参数，优先使用而不是多次调用
3. **根据观察结果调整**：每次观察后重新评估，不要机械地执行预设计划
4. **明确工具选择理由**：在选择工具时，说明为什么选择这个工具

## 观察要点

在获得工具结果后，你应该分析：
- 工具是否成功执行
- 返回的数据是否异常
- 是否需要进一步验证
- 下一步应该做什么

示例：
"观察到主机列表返回了 3 台主机。现在需要检查每台主机的 CPU 使用率。下一步：批量调用 check_cpu 工具。"

## 可用工具

%s

## 回答格式

当有足够信息时，给出清晰的最终答案：
- 总结你的发现
- 提供具体的数据和指标
- 如果发现异常，给出可能的原因和建议
- 如果需要更多信息，明确说明

## 重要提醒

- **思考在前**：每一步都要明确思考，不要盲目行动
- **观察在后**：仔细分析工具返回的结果
- **迭代优化**：根据观察结果不断调整策略
- **明确输出**：最终答案要清晰、具体、可操作

记住：ReAct 的核心是推理与行动的结合，不要跳过思考步骤。`, toolDesc)
}

// formatReActResponse 格式化 ReAct 响应为完整报告
func (a *ReActAgent) formatReActResponse(steps []ReActStep, finalAnswer string) string {
	var sb strings.Builder

	sb.WriteString("## ReAct 分析过程\n\n")

	for i, step := range steps {
		switch step.Phase {
		case "thought":
			sb.WriteString(fmt.Sprintf("### 步骤 %d: 思考\n%s\n\n", i+1, step.Content))
		case "action":
			sb.WriteString(fmt.Sprintf("### 步骤 %d: 行动\n", i+1))
			for _, tc := range step.ToolCalls {
				sb.WriteString(fmt.Sprintf("- 调用工具: **%s**\n", tc.Function.Name))
				// 可以选择是否显示参数
				// sb.WriteString(fmt.Sprintf("  参数: %s\n", tc.Function.Arguments))
			}
			sb.WriteString("\n")
		case "observation":
			sb.WriteString(fmt.Sprintf("### 步骤 %d: 观察\n%s\n\n", i+1, step.Content))
		}
	}

	if finalAnswer != "" {
		sb.WriteString("## 结论\n\n")
		sb.WriteString(finalAnswer)
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatReActSteps 格式化 ReAct 步骤用于显示（简化版）
func (a *ReActAgent) formatReActSteps(steps []ReActStep) string {
	var sb strings.Builder

	for i, step := range steps {
		switch step.Phase {
		case "thought":
			sb.WriteString(fmt.Sprintf("[%d] 💭 思考: %s\n", i+1, step.Content))
		case "action":
			sb.WriteString(fmt.Sprintf("[%d] ⚡ 行动: ", i+1))
			for j, tc := range step.ToolCalls {
				if j > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(tc.Function.Name)
			}
			sb.WriteString("\n")
		case "observation":
			sb.WriteString(fmt.Sprintf("[%d] 👁 观察: %s\n", i+1, step.Content))
		}
	}

	return sb.String()
}

// executeToolCalls 执行工具调用
func (a *ReActAgent) executeToolCalls(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	var toolMessages []llm.Message
	var records []ToolCallRecord

	// 创建工具执行上下文
	toolCtx := &tool.Context{
		Hosts:   hosts,
		SSH:     a.sshPool,
		Timeout: 30 * time.Second,
	}

	for _, tc := range toolCalls {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			logger.Error("解析工具参数失败",
				zap.Error(err),
				zap.String("tool", tc.Function.Name),
			)

			record := ToolCallRecord{
				ID:     tc.ID,
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

		logger.Info("ReAct 执行工具",
			zap.String("tool", tc.Function.Name),
			zap.Any("params", params),
		)

		// 执行工具
		result, err := a.registry.Execute(toolCtx, tc.Function.Name, params)

		record := ToolCallRecord{
			ID:     tc.ID,
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

// SetMaxLoops 设置最大循环次数
func (a *ReActAgent) SetMaxLoops(maxLoops int) {
	a.maxLoops = maxLoops
}

// SetTimeout 设置超时时间
func (a *ReActAgent) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
}
