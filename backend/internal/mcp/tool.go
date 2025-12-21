package mcp

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ToolCaller handles tool calling with timeout and error handling.
type ToolCaller struct {
	client Client
	logger *zap.Logger
}

// NewToolCaller creates a new ToolCaller.
func NewToolCaller(client Client, logger *zap.Logger) *ToolCaller {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ToolCaller{
		client: client,
		logger: logger,
	}
}

// CallTool calls a tool with timeout control and error handling.
func (tc *ToolCaller) CallTool(toolName string, args map[string]interface{}) (*ToolResult, error) {
	// Create context with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tc.logger.Info("Calling tool",
		zap.String("tool", toolName),
		zap.Any("args", args))

	startTime := time.Now()

	// Call the tool
	result, err := tc.client.CallTool(ctx, toolName, args)

	duration := time.Since(startTime)

	if err != nil {
		tc.logger.Error("Tool call failed",
			zap.String("tool", toolName),
			zap.Error(err),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	if !result.Success {
		tc.logger.Warn("Tool call returned error",
			zap.String("tool", toolName),
			zap.String("error", result.Error),
			zap.Duration("duration", duration))
		return result, nil
	}

	tc.logger.Info("Tool call succeeded",
		zap.String("tool", toolName),
		zap.Duration("duration", duration))

	return result, nil
}

// CallToolWithContext calls a tool with a custom context.
func (tc *ToolCaller) CallToolWithContext(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
	tc.logger.Info("Calling tool with context",
		zap.String("tool", toolName),
		zap.Any("args", args))

	startTime := time.Now()

	// Call the tool
	result, err := tc.client.CallTool(ctx, toolName, args)

	duration := time.Since(startTime)

	if err != nil {
		tc.logger.Error("Tool call failed",
			zap.String("tool", toolName),
			zap.Error(err),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("tool call failed: %w", err)
	}

	if !result.Success {
		tc.logger.Warn("Tool call returned error",
			zap.String("tool", toolName),
			zap.String("error", result.Error),
			zap.Duration("duration", duration))
		return result, nil
	}

	tc.logger.Info("Tool call succeeded",
		zap.String("tool", toolName),
		zap.Duration("duration", duration))

	return result, nil
}

// IsToolReadonly checks if a tool is readonly based on its metadata.
func (tc *ToolCaller) IsToolReadonly(toolName string) (bool, error) {
	tools, err := tc.client.ListTools()
	if err != nil {
		return false, fmt.Errorf("failed to list tools: %w", err)
	}

	for _, tool := range tools {
		if tool.Name == toolName {
			return tool.Readonly, nil
		}
	}

	return false, fmt.Errorf("tool not found: %s", toolName)
}

// ValidateToolArgs validates tool arguments against the tool's input schema.
func (tc *ToolCaller) ValidateToolArgs(toolName string, args map[string]interface{}) error {
	tools, err := tc.client.ListTools()
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	var tool *Tool
	for i := range tools {
		if tools[i].Name == toolName {
			tool = &tools[i]
			break
		}
	}

	if tool == nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	// Basic validation - check if required properties exist
	if schema, ok := tool.InputSchema["properties"].(map[string]interface{}); ok {
		for propName, propDef := range schema {
			if propDefMap, ok := propDef.(map[string]interface{}); ok {
				if required, ok := propDefMap["required"].(bool); ok && required {
					if _, exists := args[propName]; !exists {
						return fmt.Errorf("missing required argument: %s", propName)
					}
				}
			}
		}
	}

	return nil
}
