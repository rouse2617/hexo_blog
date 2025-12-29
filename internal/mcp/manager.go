package mcp

import (
	"fmt"
	"sort"
	"sync"

	"ai-ops/internal/tool"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// Manager MCP 管理器
type Manager struct {
	clients  map[string]*Client
	adapters map[string]*Adapter
	registry *tool.Registry
	mu       sync.RWMutex
}

// NewManager 创建 MCP 管理器
func NewManager(registry *tool.Registry) *Manager {
	return &Manager{
		clients:  make(map[string]*Client),
		adapters: make(map[string]*Adapter),
		registry: registry,
	}
}

// RegisterClient 注册 MCP 客户端
func (m *Manager) RegisterClient(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !cfg.Enabled {
		logger.Info("MCP server 已禁用，跳过", zap.String("name", cfg.Name))
		return nil
	}

	client, err := NewClient(cfg)
	if err != nil {
		return fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}

	// 测试连接
	if err := client.GetHealth(nil); err != nil {
		logger.Warn("MCP server 健康检查失败",
			zap.String("name", cfg.Name),
			zap.Error(err),
		)
		// 不阻止启动，只记录警告
	}

	m.clients[cfg.Name] = client

	// 加载该 MCP server 的工具
	tools, err := client.ListTools(nil)
	if err != nil {
		logger.Warn("获取 MCP 工具列表失败",
			zap.String("server", cfg.Name),
			zap.Error(err),
		)
		return err
	}

	// 创建适配器并注册到工具注册中心
	for _, mcpTool := range tools {
		adapter := NewAdapter(client, mcpTool)
		m.adapters[adapter.Name()] = adapter

		if err := m.registry.Register(adapter, "mcp"); err != nil {
			logger.Warn("注册 MCP 工具失败",
				zap.String("tool", adapter.Name()),
				zap.Error(err),
			)
			continue
		}

		logger.Info("注册 MCP 工具",
			zap.String("server", cfg.Name),
			zap.String("tool", adapter.Name()),
			zap.String("description", mcpTool.Description),
		)
	}

	return nil
}

// UnregisterClient 注销 MCP 客户端
func (m *Manager) UnregisterClient(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 注销相关工具
	for toolName, adapter := range m.adapters {
		if adapter.mcpClient.name == name {
			m.registry.Unregister(toolName)
			delete(m.adapters, toolName)

			logger.Info("注销 MCP 工具",
				zap.String("server", name),
				zap.String("tool", toolName),
			)
		}
	}

	delete(m.clients, name)

	logger.Info("注销 MCP 客户端",
		zap.String("name", name),
	)
}

// ListClients 列出所有客户端
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetClient 获取客户端
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[name]
	return client, ok
}

// GetStats 获取统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["clients"] = len(m.clients)
	stats["adapters"] = len(m.adapters)

	// 按服务器统计工具数量
	toolsByServer := make(map[string]int)
	for _, adapter := range m.adapters {
		serverName := adapter.mcpClient.name
		toolsByServer[serverName]++
	}
	stats["tools_by_server"] = toolsByServer

	return stats
}

// HealthCheck 检查所有 MCP servers 健康状态
func (m *Manager) HealthCheck() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, client := range m.clients {
		wg.Add(1)
		go func(name string, client *Client) {
			defer wg.Done()
			err := client.GetHealth(nil)
			mu.Lock()
			results[name] = err
			mu.Unlock()
		}(name, client)
	}

	wg.Wait()
	return results
}
