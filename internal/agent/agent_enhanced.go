package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-ops/internal/llm"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// ThinkingStep 思考步骤
type ThinkingStep struct {
	Step    string `json:"step"`    // analyze, plan, execute, reflect, summarize
	Content string `json:"content"` // 思考内容
}

// ThinkingSteps 思考步骤列表
type ThinkingSteps []ThinkingStep

// Format 格式化思考步骤为可读文本
func (ts ThinkingSteps) Format() string {
	if len(ts) == 0 {
		return ""
	}

	var result string
	result = "## 思考过程\n\n"

	for _, step := range ts {
		icon := map[string]string{
			"analyze":   "🔍",
			"plan":      "📋",
			"execute":   "⚡",
			"reflect":   "💭",
			"summarize": "✅",
		}[step.Step]

		result += fmt.Sprintf("### %s %s\n%s\n\n", icon, step.Title(), step.Content)
	}

	return result
}

// Title 返回步骤标题
func (s ThinkingStep) Title() string {
	titles := map[string]string{
		"analyze":   "分析请求",
		"plan":      "规划行动",
		"execute":   "执行操作",
		"reflect":   "反思结果",
		"summarize": "总结完成",
	}
	return titles[s.Step]
}

// analyzeRequest 分析用户请求
func analyzeRequest(userMessage string, hosts []string) ThinkingStep {
	content := fmt.Sprintf("分析用户请求：%s\n", userMessage)
	if len(hosts) > 0 {
		content += fmt.Sprintf("涉及主机：%d 台\n", len(hosts))
	}
	content += "需要确定：信息收集 / 问题诊断 / 操作执行"

	return ThinkingStep{
		Step:    "analyze",
		Content: content,
	}
}

// planNextAction 规划下一步行动
func planNextAction(records []ToolCallRecord) ThinkingStep {
	failedCount := 0
	for _, r := range records {
		if r.Error != "" {
			failedCount++
		}
	}

	content := fmt.Sprintf("已执行 %d 个工具调用", len(records))
	if failedCount > 0 {
		content += fmt.Sprintf("，其中 %d 个失败，需要分析原因", failedCount)
	} else {
		content += "，全部成功，继续收集信息或执行下一步"
	}

	return ThinkingStep{
		Step:    "plan",
		Content: content,
	}
}

// reflectOnResults 反思工具执行结果
func reflectOnResults(records []ToolCallRecord) ThinkingStep {
	var insights []string

	for _, r := range records {
		if r.Error != "" {
			insights = append(insights, fmt.Sprintf("- %s 失败: %s", r.Tool, r.Error))
		}
	}

	if len(insights) == 0 {
		return ThinkingStep{Step: "reflect", Content: ""}
	}

	return ThinkingStep{
		Step:    "reflect",
		Content: fmt.Sprintf("执行结果反思：\n%s", strings.Join(insights, "\n")),
	}
}

// optimizeToolCalls 优化工具调用（检测批量机会）
func optimizeToolCalls(calls []llm.ToolCall, contextHosts []string) []llm.ToolCall {
	if len(calls) <= 1 {
		return calls
	}

	// 检测连续调用同一工具的不同主机
	toolCalls := make(map[string][]llm.ToolCall)

	for _, call := range calls {
		toolName := call.Function.Name
		toolCalls[toolName] = append(toolCalls[toolName], call)
	}

	var optimized []llm.ToolCall

	for toolName, calls := range toolCalls {
		if len(calls) == 1 {
			optimized = append(optimized, calls...)
			continue
		}

		// 检查是否可以合并为批量调用
		if canBatch(toolName) {
			merged := mergeToolCalls(toolName, calls, contextHosts)
			optimized = append(optimized, merged)
			logger.Info("优化工具调用",
				zap.String("tool", toolName),
				zap.Int("original_count", len(calls)),
				zap.Int("optimized_count", 1),
			)
		} else {
			optimized = append(optimized, calls...)
		}
	}

	return optimized
}

// canBatch 检查工具是否支持批量
func canBatch(toolName string) bool {
	batchableTools := map[string]bool{
		"check_cpu":    true,
		"check_memory": true,
		"check_disk":   true,
		"list_hosts":   true,
		"query_log":    true,
	}
	return batchableTools[toolName]
}

// mergeToolCalls 合并工具调用为批量调用
func mergeToolCalls(toolName string, calls []llm.ToolCall, contextHosts []string) llm.ToolCall {
	// 收集所有主机
	var hosts []string
	paramsMap := make(map[string]interface{})

	for _, call := range calls {
		var params map[string]interface{}
		json.Unmarshal([]byte(call.Function.Arguments), &params)

		if h, ok := params["host"].(string); ok && h != "" {
			hosts = append(hosts, h)
		}
		if hs, ok := params["hosts"].([]string); ok {
			hosts = append(hosts, hs...)
		}
		if hs, ok := params["hosts"].([]interface{}); ok {
			for _, h := range hs {
				if s, ok := h.(string); ok {
					hosts = append(hosts, s)
				}
			}
		}

		// 合并其他参数
		for k, v := range params {
			if k != "host" && k != "hosts" {
				paramsMap[k] = v
			}
		}
	}

	// 去重
	hosts = uniqueStrings(hosts)

	// 使用 hosts 参数
	paramsMap["hosts"] = hosts

	args, _ := json.Marshal(paramsMap)

	return llm.ToolCall{
		ID: calls[0].ID,
		Function: llm.FunctionCall{
			Name:      toolName,
			Arguments: string(args),
		},
	}
}

// uniqueStrings 去重字符串切片
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// ChatWithThinking 带思考过程的对话
// 这是对 Agent.Chat 的增强版本，添加思考步骤记录
func (a *Agent) ChatWithThinking(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	messages := a.buildMessages(req)

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

	var toolCallRecords []ToolCallRecord
	var thinkingSteps ThinkingSteps
	var finalReply string

	// 第一步：分析请求
	analyzeStep := analyzeRequest(req.Message, req.Hosts)
	thinkingSteps = append(thinkingSteps, analyzeStep)
	logger.Info("分析用户请求", zap.String("thinking", analyzeStep.Content))

	for i := 0; i < a.maxLoops; i++ {
		_ = i // 避免未使用变量警告
		logger.Debug("Agent 循环", zap.Int("loop", i+1))

		// 第二步：规划行动（如果有工具调用历史）
		if len(toolCallRecords) > 0 {
			planStep := planNextAction(toolCallRecords)
			thinkingSteps = append(thinkingSteps, planStep)
			logger.Info("规划下一步", zap.String("thinking", planStep.Content))
		}

		// 第三步：执行 LLM
		resp, err := a.llmClient.ChatWithTools(ctx, messages, toolDefs)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}

		if resp.HasToolCalls() {
			// 优化工具调用：检测是否可以批量
			optimizedCalls := resp.Message.ToolCalls
			if a.promptVersion == "enhanced" {
				optimizedCalls = optimizeToolCalls(resp.Message.ToolCalls, req.Hosts)
			}

			// 执行工具
			toolResults, records := a.executeToolCalls(ctx, optimizedCalls, req.Hosts)
			toolCallRecords = append(toolCallRecords, records...)

			messages = append(messages, resp.Message)
			messages = append(messages, toolResults...)

			// 第四步：反思结果
			reflectStep := reflectOnResults(records)
			if reflectStep.Content != "" {
				thinkingSteps = append(thinkingSteps, reflectStep)
				logger.Info("反思结果", zap.String("thinking", reflectStep.Content))
			}

			continue
		}

		// 第五步：总结
		finalReply = resp.Message.Content
		if len(thinkingSteps) > 0 {
			summarizeStep := ThinkingStep{
				Step:    "summarize",
				Content: fmt.Sprintf("任务完成，共执行 %d 次工具调用", len(toolCallRecords)),
			}
			thinkingSteps = append(thinkingSteps, summarizeStep)
		}
		break
	}

	if finalReply == "" && len(toolCallRecords) > 0 {
		finalReply = "已执行相关操作，请查看工具调用结果。"
	}

	return &ChatResponse{
		SessionID: req.SessionID,
		Reply:     finalReply,
		ToolCalls: toolCallRecords,
		Thinking:  thinkingSteps.Format(),
	}, nil
}

// ChatStreamWithThinking 带思考过程的流式对话
func (a *Agent) ChatStreamWithThinking(ctx context.Context, req ChatRequest, callback func(chunk StreamChunk)) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	messages := a.buildMessages(req)

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

	var thinkingSteps ThinkingSteps

	// 第一步：分析请求
	analyzeStep := analyzeRequest(req.Message, req.Hosts)
	thinkingSteps = append(thinkingSteps, analyzeStep)
	callback(StreamChunk{
		Type:    "thinking",
		Content: analyzeStep.Content,
		Step:    len(thinkingSteps),
		Status:  "analyze",
	})

	for i := 0; i < a.maxLoops; i++ {
		_ = i // 避免未使用变量警告
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
			// 优化工具调用
			optimizedCalls := toolCalls
			if a.promptVersion == "enhanced" {
				optimizedCalls = optimizeToolCalls(toolCalls, req.Hosts)
			}

			// 通知前端：准备执行工具
			callback(StreamChunk{
				Type:    "thinking",
				Step:    i + 1,
				Status:  "executing_tools",
				Content: fmt.Sprintf("需要执行 %d 个工具来获取信息...", len(optimizedCalls)),
			})

			// 通知前端工具调用
			for _, tc := range optimizedCalls {
				callback(StreamChunk{
					Type:     "tool_call",
					ToolCall: &tc,
					Status:   "pending",
				})
			}

			// 执行工具
			toolResults, records := a.executeToolCalls(ctx, optimizedCalls, req.Hosts)

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

			// 添加反思步骤
			reflectStep := reflectOnResults(records)
			if reflectStep.Content != "" {
				thinkingSteps = append(thinkingSteps, reflectStep)
				callback(StreamChunk{
					Type:    "thinking",
					Content: reflectStep.Content,
					Status:  "reflect",
				})
			}

			messages = append(messages, llm.NewAssistantToolCallMessage(optimizedCalls))
			messages = append(messages, toolResults...)
			continue
		}

		// 没有工具调用，对话结束
		if done {
			// 总结步骤
			summarizeStep := ThinkingStep{
				Step:    "summarize",
				Content: fmt.Sprintf("任务完成，共执行 %d 次工具调用", len(thinkingSteps)-1),
			}
			thinkingSteps = append(thinkingSteps, summarizeStep)
			callback(StreamChunk{
				Type:    "thinking",
				Content: summarizeStep.Content,
				Status:  "summarize",
			})

			callback(StreamChunk{Type: "done"})
			break
		}
	}

	return nil
}
