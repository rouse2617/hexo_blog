package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config MCP 客户端配置
type Config struct {
	Name    string        `json:"name"`    // MCP server 名称
	URL     string        `json:"url"`     // MCP server 地址
	Timeout time.Duration `json:"timeout"` // 请求超时
	Enabled bool          `json:"enabled"` // 是否启用
}

// Client MCP 客户端
type Client struct {
	name    string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewClient 创建 MCP 客户端
func NewClient(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("MCP server URL 不能为空")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		name:    cfg.Name,
		baseURL: cfg.URL,
		timeout: cfg.Timeout,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Tool MCP 工具定义（符合 MCP 协议）
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult MCP 工具执行结果
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock 内容块
type ContentBlock struct {
	Type string `json:"type"` // text, image, resource
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// ListToolsResponse 列出工具响应
type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest 调用工具请求
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ListTools 列出 MCP server 提供的所有工具
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	url := fmt.Sprintf("%s/tools", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 MCP server 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP server 返回错误: %d, %s", resp.StatusCode, string(body))
	}

	var result ListToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 MCP 响应失败: %w", err)
	}

	return result.Tools, nil
}

// CallTool 调用 MCP 工具
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
	url := fmt.Sprintf("%s/tools/%s", c.baseURL, toolName)

	reqBody := CallToolRequest{
		Name:      toolName,
		Arguments: args,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 MCP 工具失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP 工具返回错误: %d, %s", resp.StatusCode, string(respBody))
	}

	var result ToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析工具结果失败: %w", err)
	}

	return &result, nil
}

// GetHealth 检查 MCP server 健康状态
func (c *Client) GetHealth(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MCP server 不健康: %d", resp.StatusCode)
	}

	return nil
}
