package agent

import (
	"ai-ops/internal/llm"
)

// buildMessagesWithPrompt 使用指定的 Prompt 构建消息列表
func (a *Agent) buildMessagesWithPrompt(req ChatRequest, systemPrompt string) []llm.Message {
	messages := make([]llm.Message, 0)

	// 使用提供的系统提示词
	messages = append(messages, llm.NewSystemMessage(systemPrompt))

	// 历史消息
	messages = append(messages, req.History...)

	// 用户消息
	messages = append(messages, llm.NewUserMessage(req.Message))

	return messages
}
