# MCP 集成分析与设计方案

## 一、当前场景分析

### 1.1 现有工具体系

| 工具 | 功能 | 实现方式 |
|------|------|----------|
| `list_hosts` | 列出主机 | 读取 SSH Pool |
| `check_cpu` | 检查 CPU | SSH 执行 `top` |
| `check_memory` | 检查内存 | SSH 执行 `free` |
| `check_disk` | 检查磁盘 | SSH 执行 `df` |
| `check_process` | 检查进程 | SSH 执行 `ps` |
| `query_log` | 查询日志 | SSH 执行 `tail/grep` |
| `run_command` | 执行命令 | SSH 执行任意命令 |

**特点**：
- ✅ 所有操作都是**远程 SSH** 执行
- ✅ 覆盖了核心运维场景
- ❌ 没有**本地**文件访问能力
- ❌ 没有外部系统集成（监控、日志平台）

---

## 二、是否需要 MCP？

### 2.1 当前场景评估

| 评估维度 | 结论 |
|----------|------|
| **纯 SSH 运维** | ❌ 不需要 MCP，现有实现已足够 |
| **需要读取本地配置** | ✅ MCP filesystem 有用 |
| **需要对接外部 API** | ✅ MCP 可以标准化集成 |
| **希望复用社区工具** | ✅ MCP 生态丰富 |
| **需要动态扩展** | ✅ MCP 支持热加载 |

### 2.2 实际场景举例

#### 场景 A：纯远程运维（不需要 MCP）
```
用户: "检查所有主机的 CPU 使用率"
→ check_cpu(hosts=["node1","node2","node3"])
→ 现有工具完全够用
```

#### 场景 B：需要读取本地配置（建议 MCP）
```
用户: "帮我看看本地有哪些部署脚本"
→ 需要 filesystem MCP server
→ 读取 ./scripts/ 目录
```

#### 场景 C：对接 Prometheus 监控（建议 MCP）
```
用户: "Prometheus 上 node1 的 CPU 告警情况如何？"
→ 需要 Prometheus MCP server
→ 查询 metrics，获取告警历史
```

#### 场景 D：对接外部日志平台（建议 MCP）
```
用户: "ELK 里最近一小时的错误日志有多少？"
→ 需要 Elasticsearch MCP server
→ 查询日志聚合数据
```

---

## 三、MCP 集成方案设计

### 3.1 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      AI-Ops Agent                           │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   Builtin    │  │   MCP Tools  │  │  Script Tools    │  │
│  │   Tools      │  │   (动态)      │  │   (未来)         │  │
│  └──────┬───────┘  └──────┬───────┘  └──────────────────┘  │
│         │                 │                                  │
│         └────────┬────────┘                                  │
│                  ▼                                           │
│         ┌─────────────────┐                                 │
│         │  Tool Registry  │                                 │
│         └────────┬────────┘                                 │
└──────────────────┼──────────────────────────────────────────┘
                   ▼
         ┌─────────────────────┐
         │   统一工具接口       │
         │   - Execute()       │
         │   - Parameters()    │
         └─────────────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
    ┌─────────┐      ┌──────────┐      ┌─────────┐
    │  SSH    │      │   MCP    │      │  Local  │
    │  Pool   │      │ Clients  │      │  Files  │
    └─────────┘      └──────────┘      └─────────┘
```

### 3.2 MCP Client 设计

**新建 `internal/mcp/client.go`**:

```go
package mcp

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "ai-ops/pkg/logger"

    "go.uber.org/zap"
)

// Client MCP 客户端
type Client struct {
    name    string
    baseURL string
    timeout time.Duration
    client  *http.Client
}

// Config MCP 客户端配置
type Config struct {
    Name    string        `yaml:"name"`     // MCP server 名称
    URL     string        `yaml:"url"`      // MCP server 地址
    Timeout time.Duration `yaml:"timeout"`  // 请求超时
    Enabled bool          `yaml:"enabled"`  // 是否启用
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

    var result struct {
        Tools []Tool `json:"tools"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("解析 MCP 响应失败: %w", err)
    }

    logger.Info("获取 MCP 工具列表",
        zap.String("server", c.name),
        zap.Int("count", len(result.Tools)),
    )

    return result.Tools, nil
}

// CallTool 调用 MCP 工具
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*ToolResult, error) {
    url := fmt.Sprintf("%s/tools/%s", c.baseURL, toolName)

    body := map[string]interface{}{
        "arguments": args,
    }

    jsonData, err := json.Marshal(body)
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
```

### 3.3 MCP Tool 适配器

**新建 `internal/mcp/adapter.go`**:

```go
package mcp

import (
    "context"
    "encoding/json"
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
```

### 3.4 MCP 管理器

**新建 `internal/mcp/manager.go`**:

```go
package mcp

import (
    "sync"

    "ai-ops/internal/tool"
    "ai-ops/pkg/logger"

    "go.uber.org/zap"
)

// Manager MCP 管理器
type Manager struct {
    clients   map[string]*Client
    adapters  map[string]*Adapter
    registry  *tool.Registry
    mu        sync.RWMutex
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
        return err
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
        }
    }

    delete(m.clients, name)
}

// ListClients 列出所有客户端
func (m *Manager) ListClients() []string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    names := make([]string, 0, len(m.clients))
    for name := range m.clients {
        names = append(names, name)
    }
    return names
}
```

### 3.5 配置支持

**修改 `internal/config/config.go`**:

```go
// Config 应用配置
type Config struct {
    Server   ServerConfig    `yaml:"server"`
    LLM      LLMConfig       `yaml:"llm"`
    SSH      SSHConfig       `yaml:"ssh"`
    Hosts    []HostConfig    `yaml:"hosts"`
    Database DatabaseConfig  `yaml:"database"`
    Scripts  ScriptsConfig   `yaml:"scripts"`
    Agent    AgentConfig     `yaml:"agent"`
    Log      LogConfig       `yaml:"log"`
    MCP      []MCPConfig     `yaml:"mcp"`  // 新增：MCP 配置
}

// MCPConfig MCP Server 配置
type MCPConfig struct {
    Name    string        `yaml:"name"`     // MCP server 名称
    URL     string        `yaml:"url"`      // MCP server 地址
    Timeout time.Duration `yaml:"timeout"`  // 请求超时
    Enabled bool          `yaml:"enabled"`  // 是否启用
}
```

**修改 `config.yaml`**:

```yaml
# MCP 集成配置
mcp:
  # 本地文件系统 MCP server（用于读取本地脚本、配置）
  - name: filesystem
    url: http://localhost:3000
    timeout: 30s
    enabled: false  # 按需启用

  # Prometheus MCP server（用于查询监控指标）
  - name: prometheus
    url: http://localhost:3001
    timeout: 30s
    enabled: false

  # 自定义 MCP server
  - name: custom-ops
    url: http://localhost:3002
    timeout: 60s
    enabled: false
```

### 3.6 主程序集成

**修改 `cmd/server/main.go`**:

```go
import (
    "ai-ops/internal/mcp"
    // ...
)

func main() {
    // ... 现有初始化代码 ...

    // 创建工具注册中心
    toolRegistry := tool.NewRegistry()

    // 注册内置工具
    getHostsFunc := func(group string) []builtin.HostBasicInfo {
        // ...
    }
    if err := builtin.RegisterAll(toolRegistry, getHostsFunc); err != nil {
        logger.Fatal("注册内置工具失败", zap.Error(err))
    }

    // === 新增：MCP 集成 ===
    mcpManager := mcp.NewManager(toolRegistry)

    // 注册配置的 MCP servers
    for _, mcpCfg := range cfg.MCP {
        if err := mcpManager.RegisterClient(mcp.Config{
            Name:    mcpCfg.Name,
            URL:     mcpCfg.URL,
            Timeout: mcpCfg.Timeout,
            Enabled: mcpCfg.Enabled,
        }); err != nil {
            logger.Warn("注册 MCP server 失败",
                zap.String("name", mcpCfg.Name),
                zap.Error(err),
            )
        }
    }

    logger.Info("工具注册完成",
        zap.Int("builtin", toolRegistry.Count()),
        zap.Int("mcp_servers", len(mcpManager.ListClients())),
    )
    // === MCP 集成结束 ===

    // ... 后续代码 ...
}
```

---

## 四、推荐使用的 MCP Servers

### 4.1 官方/社区 MCP Servers

| MCP Server | 功能 | 适用场景 |
|------------|------|----------|
| `@modelcontextprotocol/server-filesystem` | 本地文件系统 | 读取本地脚本、配置文件 |
| `@modelcontextprotocol/server-postgres` | PostgreSQL 查询 | 查询数据库 |
| `@modelcontextprotocol/server-sqlite` | SQLite 查询 | 本地数据查询 |
| `@modelcontextprotocol/server-github` | GitHub 操作 | 查询 Issue、PR |
| `@modelcontextprotocol/server-git` | Git 操作 | 查看代码历史 |
| `@modelcontextprotocol/server-brave-search` | 网页搜索 | 搜索技术文档 |

### 4.2 自定义 MCP Server 示例

**Prometheus MCP Server** (`mcp-servers/prometheus/src/index.ts`):

```typescript
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';

const server = new Server({
  name: 'prometheus-mcp-server',
  version: '1.0.0',
});

// 工具：query_prometheus
server.registerTool({
  name: 'query_prometheus',
  description: '查询 Prometheus 监控指标',
  inputSchema: {
    type: 'object',
    properties: {
      query: {
        type: 'string',
        description: 'PromQL 查询语句',
      },
      range: {
        type: 'string',
        description: '时间范围，如 1h, 24h',
      },
    },
    required: ['query'],
  },
}, async (args) => {
  const response = await fetch(`${PROMETHEUS_URL}/api/v1/query_range?query=${args.query}&range=${args.range}`);
  const data = await response.json();

  return {
    content: [{
      type: 'text',
      text: JSON.stringify(data, null, 2),
    }],
  };
});

// 启动 HTTP 服务（而非 stdio）
const express = require('express');
const app = express();
app.use(express.json());

app.get('/tools', (req, res) => {
  res.json({ tools: [/* tool definitions */] });
});

app.post('/tools/:name', async (req, res) => {
  // 调用对应的工具
});

app.listen(3000, () => {
  console.log('Prometheus MCP server listening on port 3000');
});
```

---

## 五、实施建议

### 5.1 阶段一：基础集成（最小化）

```
目标：支持 MCP 协议，不强制使用

任务：
1. ✅ 实现 MCP Client 基础功能
2. ✅ 实现 MCP Tool 适配器
3. ✅ 修改配置文件支持 MCP
4. ✅ 在 main.go 中集成 MCP Manager

验证：
- 启动服务，MCP 相关配置 enabled=false 时不影响现有功能
- enabled=true 时能正确连接 MCP server
```

### 5.2 阶段二：实际应用

```
按需集成 MCP servers：

场景 A：需要读取本地脚本
→ 启用 filesystem MCP server

场景 B：需要对接 Prometheus
→ 启用 Prometheus MCP server

场景 C：需要查询数据库
→ 启用 postgres/sqlite MCP server
```

### 5.3 阶段三：高级功能

```
1. MCP 热加载/卸载
2. MCP 工具调用监控
3. MCP Server 健康检查
4. MCP 调用结果缓存
```

---

## 六、总结

### 6.1 是否需要 MCP？

| 场景 | 建议 |
|------|------|
| **纯 SSH 远程运维** | ❌ 不需要，现有工具够用 |
| **需要读取本地文件** | ✅ MCP filesystem 有用 |
| **需要对接外部系统** | ✅ MCP 标准化集成 |
| **希望复用社区工具** | ✅ MCP 生态丰富 |

### 6.2 MCP 价值

1. **标准化**：统一的工具协议
2. **扩展性**：动态加载，无需重启
3. **生态**：复用社区已有的 MCP servers
4. **解耦**：工具实现与 Agent 分离

### 6.3 实施优先级

```
P0 (立即): 实现 MCP 基础框架（不影响现有功能）
P1 (按需): 根据实际场景集成具体的 MCP servers
P2 (未来): 优化 MCP 调用性能和监控
```

---

## 七、代码文件清单

### 新增文件
```
internal/mcp/
├── client.go      # MCP 客户端
├── adapter.go     # MCP 工具适配器
├── manager.go     # MCP 管理器
└── types.go       # MCP 类型定义

docs/
└── mcp-integration-analysis.md  # 本文档
```

### 修改文件
```
internal/config/config.go    # 添加 MCPConfig
config.yaml                  # 添加 MCP 配置
cmd/server/main.go           # 集成 MCP Manager
```

### 示例 MCP Server
```
mcp-servers/
├── filesystem/    # 文件系统 MCP server
├── prometheus/    # Prometheus MCP server
└── elasticsearch/ # Elasticsearch MCP server
```
