package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIClient OpenAI 兼容客户端（支持 Ollama）
type OpenAIClient struct {
	endpoint   string
	model      string
	apiKey     string
	timeout    time.Duration
	maxTokens  int
	httpClient *http.Client
}

// OpenAIConfig 客户端配置
type OpenAIConfig struct {
	Endpoint  string
	Model     string
	APIKey    string
	Timeout   time.Duration
	MaxTokens int
}

// NewOpenAIClient 创建 OpenAI 兼容客户端
func NewOpenAIClient(cfg OpenAIConfig) *OpenAIClient {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:11434/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "qwen2.5:14b"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 4096
	}

	return &OpenAIClient{
		endpoint:  strings.TrimSuffix(cfg.Endpoint, "/"),
		model:     cfg.Model,
		apiKey:    cfg.APIKey,
		timeout:   cfg.Timeout,
		maxTokens: cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// openAIRequest OpenAI API 请求结构
type openAIRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Tools          []ToolDef `json:"tools,omitempty"`
	MaxTokens      int       `json:"max_tokens,omitempty"`
	Temperature    float64   `json:"temperature,omitempty"`
	Stream         bool      `json:"stream,omitempty"`
	EnableThinking *bool     `json:"enable_thinking,omitempty"`
}

// openAIResponse OpenAI API 响应结构
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// openAIStreamResponse 流式响应结构
type openAIStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string     `json:"role,omitempty"`
			Content   string     `json:"content,omitempty"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// Chat 普通对话
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message) (*ChatResponse, error) {
	return c.ChatWithTools(ctx, messages, nil)
}

// ChatWithTools 带工具调用的对话
func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDef) (*ChatResponse, error) {
	reqBody := openAIRequest{
		Model:     c.model,
		Messages:  messages,
		MaxTokens: c.maxTokens,
		Stream:    false,
	}
	{
		v := false
		reqBody.EnableThinking = &v
	}

	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("响应中没有 choices")
	}

	choice := openAIResp.Choices[0]
	return &ChatResponse{
		ID:           openAIResp.ID,
		Model:        openAIResp.Model,
		Message:      choice.Message,
		Usage:        openAIResp.Usage,
		FinishReason: choice.FinishReason,
	}, nil
}

// ChatStream 流式对话
func (c *OpenAIClient) ChatStream(ctx context.Context, messages []Message, callback StreamCallback) error {
	return c.ChatStreamWithTools(ctx, messages, nil, callback)
}

// ChatStreamWithTools 带工具调用的流式对话
func (c *OpenAIClient) ChatStreamWithTools(ctx context.Context, messages []Message, tools []ToolDef, callback StreamCallback) error {
	reqBody := openAIRequest{
		Model:     c.model,
		Messages:  messages,
		MaxTokens: c.maxTokens,
		Stream:    true,
	}

	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析 SSE 流
	reader := bufio.NewReader(resp.Body)
	var toolCalls []ToolCall

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取流失败: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// SSE 格式: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			callback(StreamChunk{Type: "done"})
			break
		}

		var streamResp openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue // 忽略解析错误
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// 处理内容
		if choice.Delta.Content != "" {
			callback(StreamChunk{
				Type:    "content",
				Content: choice.Delta.Content,
			})
		}

		// 处理工具调用
		if len(choice.Delta.ToolCalls) > 0 {
			for _, tc := range choice.Delta.ToolCalls {
				// 累积工具调用
				if tc.ID != "" {
					toolCalls = append(toolCalls, tc)
				} else if len(toolCalls) > 0 {
					// 追加参数到最后一个工具调用
					last := &toolCalls[len(toolCalls)-1]
					last.Function.Arguments += tc.Function.Arguments
				}
			}
		}

		// 处理结束
		if choice.FinishReason != "" {
			if choice.FinishReason == FinishReasonToolCalls && len(toolCalls) > 0 {
				for _, tc := range toolCalls {
					callback(StreamChunk{
						Type:     "tool_call",
						ToolCall: &tc,
					})
				}
			}
			callback(StreamChunk{
				Type:         "done",
				FinishReason: choice.FinishReason,
			})
			break
		}
	}

	return nil
}

// GetModel 获取当前模型
func (c *OpenAIClient) GetModel() string {
	return c.model
}

// SetModel 设置模型
func (c *OpenAIClient) SetModel(model string) {
	c.model = model
}
