# Tool 系统设计

## 核心理念

Tool 是 Agent 的"手"，是 Agent 与外部世界交互的唯一方式。

```
Agent 能做什么 = Agent 有什么 Tool
```

## Tool 接口定义

```go
// Tool 工具接口
type Tool interface {
    // 基本信息
    Name() string        // 工具唯一标识
    Description() string // 工具描述（给 LLM 看，决定何时调用）

    // 参数定义
    Parameters() []Parameter

    // 执行
    Execute(ctx *Context, params map[string]any) (*Result, error)
}

// Parameter 参数定义
type Parameter struct {
    Name        string // 参数名
    Type        string // 类型: string, int, bool, []string
    Description string // 参数描述（给 LLM 看）
    Required    bool   // 是否必填
    Default     any    // 默认值
    Enum        []any  // 可选值枚举
}

// Context 执行上下文
type Context struct {
    SessionID  string          // 会话 ID
    SSH        *SSHExecutor    // SSH 执行器
    HostMgr    *HostManager    // 主机管理器
    Logger     *zap.Logger     // 日志
    Timeout    time.Duration   // 超时时间
}

// Result 执行结果
type Result struct {
    Success bool   // 是否成功
    Data    any    // 返回数据
    Message string // 结果描述
    Error   string // 错误信息
}
```

## Tool 注册中心

```go
// Registry 工具注册中心
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

// Register 注册工具
func (r *Registry) Register(tool Tool) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    name := tool.Name()
    if _, exists := r.tools[name]; exists {
        return fmt.Errorf("tool %s already registered", name)
    }
    r.tools[name] = tool
    return nil
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    tool, ok := r.tools[name]
    return tool, ok
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
            sb.WriteString(fmt.Sprintf("  - %s (%s): %s%s\n",
                p.Name, p.Type, p.Description, required))
        }
        sb.WriteString("\n")
    }

    return sb.String()
}
```

## 三层 Tool 架构

```
┌─────────────────────────────────────────────────────────┐
│                    Tool 注册中心                         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              第一层: 内置 Tool                    │   │
│  │                  (Go 代码实现)                    │   │
│  │                                                  │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐        │   │
│  │  │query_log │ │check_cpu │ │check_disk│ ...    │   │
│  │  └──────────┘ └──────────┘ └──────────┘        │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              第二层: 脚本 Tool                    │   │
│  │               (动态加载脚本)                      │   │
│  │                                                  │   │
│  │  scripts/                                        │   │
│  │  ├── check_redis.yaml + check_redis.sh          │   │
│  │  ├── backup_db.yaml + backup_db.py              │   │
│  │  └── ...                                        │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
│  ┌─────────────────────────────────────────────────┐   │
│  │              第三层: 远程 Tool                    │   │
│  │              (HTTP/gRPC 调用)                    │   │
│  │                                                  │   │
│  │  ┌──────────────┐ ┌──────────────┐              │   │
│  │  │ K8s API Tool │ │ 云厂商 Tool  │ ...          │   │
│  │  └──────────────┘ └──────────────┘              │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## 第一层: 内置 Tool

### query_log - 日志查询

```go
type QueryLogTool struct{}

func (t *QueryLogTool) Name() string { return "query_log" }

func (t *QueryLogTool) Description() string {
    return `查询指定节点的日志文件。
支持的日志类型: nginx, app, system, custom
可以按关键词过滤，指定查看行数。
适用场景: 排查错误、查看访问记录、分析异常。`
}

func (t *QueryLogTool) Parameters() []Parameter {
    return []Parameter{
        {Name: "host", Type: "string", Description: "目标节点名称或IP", Required: true},
        {Name: "log_type", Type: "string", Description: "日志类型: nginx/app/system/custom", Required: true, Enum: []any{"nginx", "app", "system", "custom"}},
        {Name: "lines", Type: "int", Description: "查看行数", Required: false, Default: 100},
        {Name: "keyword", Type: "string", Description: "过滤关键词", Required: false},
        {Name: "path", Type: "string", Description: "自定义日志路径(log_type=custom时使用)", Required: false},
    }
}

func (t *QueryLogTool) Execute(ctx *Context, params map[string]any) (*Result, error) {
    host := params["host"].(string)
    logType := params["log_type"].(string)
    lines := getIntParam(params, "lines", 100)
    keyword := getStringParam(params, "keyword", "")

    // 根据日志类型确定路径
    logPath := t.getLogPath(logType, params)

    // 构建命令
    cmd := fmt.Sprintf("tail -%d %s", lines, logPath)
    if keyword != "" {
        cmd = fmt.Sprintf("grep '%s' %s | tail -%d", keyword, logPath, lines)
    }

    // 执行
    output, err := ctx.SSH.Exec(host, cmd)
    if err != nil {
        return &Result{Success: false, Error: err.Error()}, nil
    }

    return &Result{
        Success: true,
        Data:    output,
        Message: fmt.Sprintf("成功获取 %s 的 %s 日志", host, logType),
    }, nil
}
```

### check_cpu - CPU 检查

```go
type CheckCPUTool struct{}

func (t *CheckCPUTool) Name() string { return "check_cpu" }

func (t *CheckCPUTool) Description() string {
    return `检查节点的 CPU 使用情况。
返回 CPU 使用率、负载等信息。
可以检查单个节点或所有节点。`
}

func (t *CheckCPUTool) Parameters() []Parameter {
    return []Parameter{
        {Name: "host", Type: "string", Description: "目标节点，不填则检查所有节点", Required: false},
    }
}

func (t *CheckCPUTool) Execute(ctx *Context, params map[string]any) (*Result, error) {
    host := getStringParam(params, "host", "")

    cmd := "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1"

    var results []map[string]any

    if host != "" {
        // 单节点
        output, err := ctx.SSH.Exec(host, cmd)
        if err != nil {
            return &Result{Success: false, Error: err.Error()}, nil
        }
        results = append(results, map[string]any{
            "host":      host,
            "cpu_usage": strings.TrimSpace(output),
        })
    } else {
        // 所有节点
        hosts := ctx.HostMgr.GetAllHosts()
        results = ctx.SSH.BatchExec(hosts, cmd)
    }

    return &Result{
        Success: true,
        Data:    results,
        Message: "CPU 检查完成",
    }, nil
}
```

### 内置 Tool 列表

| Tool | 功能 | 关键参数 |
|------|------|----------|
| `query_log` | 查询日志 | host, log_type, lines, keyword |
| `check_cpu` | 检查 CPU | host (可选) |
| `check_memory` | 检查内存 | host (可选) |
| `check_disk` | 检查磁盘 | host (可选), path |
| `check_process` | 查看进程 | host, name |
| `run_command` | 执行命令 | host, command |
| `list_hosts` | 列出主机 | group (可选) |

## 第二层: 脚本 Tool

### 脚本定义格式

```yaml
# scripts/check_redis.yaml
name: check_redis
description: |
  检查 Redis 服务状态。
  返回 Redis 运行状态、内存使用、连接数、键数量等信息。
  适用于 Redis 健康检查和性能分析。

parameters:
  - name: host
    type: string
    description: 目标节点
    required: true
  - name: port
    type: int
    description: Redis 端口
    required: false
    default: 6379
  - name: password
    type: string
    description: Redis 密码
    required: false

script: check_redis.sh
timeout: 30s
```

```bash
#!/bin/bash
# scripts/check_redis.sh

PORT=${2:-6379}
PASSWORD=$3

if [ -n "$PASSWORD" ]; then
    AUTH="-a $PASSWORD"
fi

redis-cli -p $PORT $AUTH INFO | grep -E "used_memory_human|connected_clients|db0:keys"
```

### 脚本 Tool 加载器

```go
// ScriptTool 脚本工具
type ScriptTool struct {
    name        string
    description string
    parameters  []Parameter
    scriptPath  string
    timeout     time.Duration
}

// ScriptLoader 脚本加载器
type ScriptLoader struct {
    scriptDir string
    registry  *Registry
}

func (l *ScriptLoader) LoadAll() error {
    files, _ := filepath.Glob(filepath.Join(l.scriptDir, "*.yaml"))

    for _, f := range files {
        tool, err := l.loadScript(f)
        if err != nil {
            log.Printf("加载脚本失败 %s: %v", f, err)
            continue
        }
        l.registry.Register(tool)
    }
    return nil
}

func (l *ScriptLoader) loadScript(yamlPath string) (*ScriptTool, error) {
    data, _ := os.ReadFile(yamlPath)

    var meta struct {
        Name        string      `yaml:"name"`
        Description string      `yaml:"description"`
        Parameters  []Parameter `yaml:"parameters"`
        Script      string      `yaml:"script"`
        Timeout     string      `yaml:"timeout"`
    }
    yaml.Unmarshal(data, &meta)

    return &ScriptTool{
        name:        meta.Name,
        description: meta.Description,
        parameters:  meta.Parameters,
        scriptPath:  filepath.Join(filepath.Dir(yamlPath), meta.Script),
        timeout:     parseDuration(meta.Timeout),
    }, nil
}
```

## 第三层: 远程 Tool

```go
// RemoteTool 远程工具 (HTTP 调用)
type RemoteTool struct {
    name        string
    description string
    parameters  []Parameter
    endpoint    string // http://localhost:8081/api/k8s/pods
    method      string // GET/POST
}

func (t *RemoteTool) Execute(ctx *Context, params map[string]any) (*Result, error) {
    // 构建请求
    body, _ := json.Marshal(params)
    req, _ := http.NewRequest(t.method, t.endpoint, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    // 发送请求
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return &Result{Success: false, Error: err.Error()}, nil
    }
    defer resp.Body.Close()

    // 解析响应
    var result Result
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```

## Tool 描述的重要性

**Tool 的 Description 决定了 LLM 什么时候调用它！**

```
❌ 差的描述:
name: check_disk
description: 检查磁盘

❓ LLM 不知道:
- 什么时候该用这个工具？
- 能检查什么？
- 返回什么信息？

✅ 好的描述:
name: check_disk
description: |
  检查节点的磁盘使用情况。
  返回各分区的总容量、已用空间、使用率。
  适用场景：
  - 排查磁盘空间不足问题
  - 日常巡检磁盘使用率
  - 查找大文件占用

  当用户询问"磁盘满了"、"空间不够"、"存储"相关问题时使用。
```

## 扩展新 Tool 的流程

### 方式一: 内置 Tool (Go 代码)

```
1. 在 internal/tool/builtin/ 下创建新文件
2. 实现 Tool 接口
3. 在 registry 初始化时注册
4. 重新编译部署
```

### 方式二: 脚本 Tool (运维人员)

```
1. 编写脚本 (bash/python/...)
2. 编写 yaml 描述文件
3. 放到 scripts/ 目录
4. 系统自动加载，无需重启
```

### 方式三: 远程 Tool (对接外部系统)

```
1. 编写 yaml 配置文件，指定 endpoint
2. 或通过 API 动态注册
3. Agent 通过 HTTP 调用外部服务
```
