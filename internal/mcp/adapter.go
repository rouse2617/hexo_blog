package mcp

import (
	"context"
	"fmt"

	"ai-ops/internal/tool"
)

// Adapter 将 MCP 工具适配为内部 Tool 接口
type Adapter struct {
	mcpClient *Client
	mcpTool   Tool
}

// NewAdapter 创建 MCP 工具适配器
func NewAdapter(client *Client, mcpTool Tool) *Adapter {
	return &Adapter{
		mcpClient: client,
		mcpTool:   mcpTool,
	}
}

// Name 返回工具名称
func (a *Adapter) Name() string {
	return fmt.Sprintf("mcp_%s_%s", a.mcpClient.name, a.mcpTool.Name)
}

// Description 返回工具描述
func (a *Adapter) Description() string {
	desc := a.mcpTool.Description
	return fmt.Sprintf("[MCP:%s] %s", a.mcpClient.name, desc)
}

// Parameters 返回参数定义（将 MCP Schema 转换为内部格式）
func (a *Adapter) Parameters() []tool.Parameter {
	schema := a.mcpTool.InputSchema
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return []tool.Parameter{}
	}

	required, _ := schema["required"].([]interface{})
	requiredMap := make(map[string]bool)
	for _, r := range required {
		if s, ok := r.(string); ok {
			requiredMap[s] = true
		}
	}

	params := make([]tool.Parameter, 0, len(properties))

	for name, prop := range properties {
		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}

		paramType := "string"
		if t, ok := propMap["type"].(string); ok {
			paramType = t
		}

		description := ""
		if d, ok := propMap["description"].(string); ok {
			description = d
		}

		p := tool.Parameter{
			Name:        name,
			Type:        paramType,
			Description: description,
			Required:    requiredMap[name],
		}

		// 处理 enum
		if enum, ok := propMap["enum"].([]interface{}); ok {
			p.Enum = enum
		}

		// 处理 default
		if def, ok := propMap["default"]; ok {
			p.Default = def
		}

		params = append(params, p)
	}

	return params
}

// Execute 执行工具
func (a *Adapter) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	// 调用 MCP server
	result, err := a.mcpClient.CallTool(context.Background(), a.mcpTool.Name, params)
	if err != nil {
		return tool.NewErrorResult(err.Error()), nil
	}

	if result.IsError {
		// 提取错误文本
		for _, block := range result.Content {
			if block.Type == "text" {
				return tool.NewErrorResult(block.Text), nil
			}
		}
		return tool.NewErrorResult("MCP 工具执行失败"), nil
	}

	// 提取结果文本
	var textContent string
	for _, block := range result.Content {
		if block.Type == "text" {
			textContent += block.Text
		}
	}

	return tool.NewResult(map[string]interface{}{
		"content": textContent,
		"blocks":  result.Content,
	}, "MCP 工具执行成功"), nil
}
