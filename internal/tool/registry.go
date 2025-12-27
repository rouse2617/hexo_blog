package tool

import (
	"fmt"
	"strings"
	"sync"
)

// Registry 工具注册中心
type Registry struct {
	tools map[string]Tool
	types map[string]string // 工具类型: builtin / script / remote
	mu    sync.RWMutex
}

// NewRegistry 创建注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		types: make(map[string]string),
	}
}

// Register 注册工具
func (r *Registry) Register(tool Tool, toolType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("工具已存在: %s", name)
	}

	r.tools[name] = tool
	r.types[name] = toolType
	return nil
}

// RegisterBuiltin 注册内置工具
func (r *Registry) RegisterBuiltin(tool Tool) error {
	return r.Register(tool, "builtin")
}

// RegisterScript 注册脚本工具
func (r *Registry) RegisterScript(tool Tool) error {
	return r.Register(tool, "script")
}

// Unregister 注销工具
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tools, name)
	delete(r.types, name)
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// List 列出所有工具
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListInfo 列出所有工具信息
func (r *Registry) ListInfo() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for name, tool := range r.tools {
		infos = append(infos, ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Type:        r.types[name],
			Parameters:  tool.Parameters(),
		})
	}
	return infos
}

// Count 获取工具数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Has 检查工具是否存在
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}

func hasNonEmptyHostParams(params map[string]interface{}) bool {
	if params == nil {
		return false
	}
	if v, ok := params["host"]; ok {
		if v != nil {
			if s, ok := v.(string); ok {
				if s != "" {
					return true
				}
			} else {
				return true
			}
		}
	}
	if v, ok := params["hosts"]; ok {
		switch vv := v.(type) {
		case []string:
			return len(vv) > 0
		case []interface{}:
			return len(vv) > 0
		}
	}
	return false
}

func shouldInjectHosts(paramsDef []Parameter) (bool, bool) {
	supportsHost := false
	supportsHosts := false
	for _, p := range paramsDef {
		switch p.Name {
		case "host":
			supportsHost = true
		case "hosts":
			supportsHosts = true
		}
	}
	return supportsHost, supportsHosts
}

// Execute 执行工具
func (r *Registry) Execute(ctx *Context, name string, params map[string]interface{}) (*Result, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("工具不存在: %s", name)
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	if ctx != nil && len(ctx.Hosts) > 0 {
		supportsHost, supportsHosts := shouldInjectHosts(tool.Parameters())
		if (supportsHost || supportsHosts) && !hasNonEmptyHostParams(params) {
			if len(ctx.Hosts) == 1 {
				if supportsHost {
					params["host"] = ctx.Hosts[0]
				} else {
					params["hosts"] = ctx.Hosts
				}
			} else {
				if supportsHosts {
					params["hosts"] = ctx.Hosts
				} else if supportsHost {
					params["host"] = ctx.Hosts[0]
				}
			}
		}
	}

	return tool.Execute(ctx, params)
}

// GeneratePrompt 生成工具描述（给 LLM）
func (r *Registry) GeneratePrompt() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("你有以下工具可用：\n\n")

	for _, tool := range r.tools {
		sb.WriteString(fmt.Sprintf("## %s\n", tool.Name()))
		sb.WriteString(fmt.Sprintf("描述: %s\n", tool.Description()))
		sb.WriteString("参数:\n")

		for _, p := range tool.Parameters() {
			required := ""
			if p.Required {
				required = " (必填)"
			}
			defaultVal := ""
			if p.Default != nil {
				defaultVal = fmt.Sprintf(", 默认: %v", p.Default)
			}
			enumVal := ""
			if len(p.Enum) > 0 {
				enumVal = fmt.Sprintf(", 可选值: %v", p.Enum)
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s): %s%s%s%s\n",
				p.Name, p.Type, p.Description, required, defaultVal, enumVal))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateJSONSchema 生成 JSON Schema 格式的工具描述（用于 OpenAI function calling）
func (r *Registry) GenerateJSONSchema() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make([]map[string]interface{}, 0, len(r.tools))

	for _, tool := range r.tools {
		properties := make(map[string]interface{})
		required := make([]string, 0)

		for _, p := range tool.Parameters() {
			prop := map[string]interface{}{
				"type":        convertType(p.Type),
				"description": p.Description,
			}
			if len(p.Enum) > 0 {
				prop["enum"] = p.Enum
			}
			properties[p.Name] = prop

			if p.Required {
				required = append(required, p.Name)
			}
		}

		schema := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": properties,
					"required":   required,
				},
			},
		}
		schemas = append(schemas, schema)
	}

	return schemas
}

// convertType 转换类型到 JSON Schema 类型
func convertType(t string) string {
	switch t {
	case "int", "integer":
		return "integer"
	case "float", "number":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "[]string", "array":
		return "array"
	default:
		return "string"
	}
}
