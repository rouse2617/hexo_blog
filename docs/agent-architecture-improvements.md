# AI-Ops Agent 架构改进文档

## 一、现状分析

### 当前架构
```
用户输入 → [Agent Loop] → 输出
              ↓
         LLM 调用
              ↓
         工具执行
              ↓
         继续循环
```

**特点**：
- 单 Agent ReAct 循环
- 简单的系统提示词 (~500 tokens)
- 基础的工具描述
- 使用标准的 OpenAI 兼容 API (glm-4.7)

**局限**：
1. 系统提示词过于简单，缺少详细的决策指导
2. 工具描述只说明"是什么"，没说明"何时用"、"怎么用"
3. 缺少显式的思考步骤（思考、规划、反思）
4. 没有工具调用策略（何时串行、何时并行）

---

## 二、改进方案

### 2.1 增强系统提示词

**当前** (`internal/agent/prompt.go`):
```go
const SystemPromptTemplate = `你是一个专业的运维助手 Agent，名为 AI-Ops。

## 你的能力
- 你可以通过调用工具来执行各种运维操作
...

## 可用工具
%s

## 注意事项
- 执行危险操作前要确认
...`
```

**改进后** - 新建 `internal/agent/prompt_enhanced.go`:
```go
package agent

// EnhancedSystemPromptTemplate 增强版系统提示词
const EnhancedSystemPromptTemplate = `# AI-Ops 智能运维助手

你是一个专业的运维 Agent，具有强大的系统监控、诊断和问题解决能力。

## 核心能力
1. **系统监控**：CPU、内存、磁盘、网络、进程监控
2. **日志分析**：查看和分析系统/应用日志
3. **故障诊断**：定位性能瓶颈、资源占用问题
4. **批量操作**：同时在多台服务器上执行操作

## 工作流程（严格遵循）
对于每个用户请求，你必须按以下步骤进行：

### 第一步：理解与规划
1. 分析用户意图：用户想了解什么？想解决什么问题？
2. 识别关键信息：涉及哪些主机？需要什么数据？
3. 制定执行计划：需要调用哪些工具？按什么顺序？
4. 评估风险：操作是否安全？是否需要用户确认？

### 第二步：信息收集
1. 优先使用只读工具获取信息（check_* 系列工具）
2. 批量操作时使用 hosts 参数而非多次调用
3. 收集足够信息后再做决策

### 第三步：分析诊断
1. 综合分析收集到的数据
2. 识别异常模式和问题根因
3. 提出具体的解决建议

### 第四步：执行与验证（如需）
1. 如需执行操作，先说明操作目的和预期结果
2. 执行后验证结果
3. 如失败，分析原因并尝试替代方案

### 第五步：总结报告
1. 总结执行的操作
2. 报告发现的问题
3. 给出具体的建议

## 工具使用策略

### 并行执行策略
当满足以下条件时，应使用 hosts 参数批量执行：
- 检查多个主机的相同指标（如检查所有主机的 CPU）
- 执行相同的只读命令
- 操作之间无依赖关系

示例：
```
❌ 错误：分 3 次调用
1. check_cpu(host="node1")
2. check_cpu(host="node2")
3. check_cpu(host="node3")

✅ 正确：一次批量调用
check_cpu(hosts=["node1", "node2", "node3"])
```

### 串行执行策略
当满足以下条件时，应分步调用：
- 后续操作依赖前一步的结果
- 需要根据前一步结果决定后续操作
- 操作涉及不同类型的数据收集

示例：
```
用户："检查 node1 的磁盘，如果使用率超过 80%，列出大文件"

✅ 正确：分步执行
1. check_disk(host="node1")  # 先检查
2. # 如果发现 /var 使用率 85%，继续
3. run_command(host="node1", command="du -sh /var/* | sort -hr | head -20")
```

### 工具选择优先级
1. **优先使用专用工具**：
   - 检查 CPU → check_cpu（而非 run_command "top"）
   - 检查内存 → check_memory（而非 run_command "free"）
   - 检查磁盘 → check_disk（而非 run_command "df"）

2. **run_command 作为补充**：
   - 仅在专用工具无法满足需求时使用
   - 用于执行自定义脚本或特殊查询

3. **list_hosts 用于上下文**：
   - 不确定主机名称时先调用 list_hosts
   - 了解环境后制定精准的操作计划

## 输出格式规范

### 信息收集类请求
```
# [主机名] - 指标名称
- 当前值：...
- 状态：正常/警告/危险
- 建议：...
```

### 问题诊断类请求
```
# 问题诊断

## 发现的问题
1. [主机名] - [问题描述]
   - 详细数据：...
   - 可能原因：...

## 诊断结论
[综合分析后的问题根因]

## 建议措施
1. [具体措施1]
2. [具体措施2]
```

### 执行操作类请求
```
# 执行操作

## 操作计划
[说明要执行的操作及原因]

## 执行结果
- [主机1]：成功/失败
- [主机2]：成功/失败

## 验证结果
[操作后的验证结果]

## 总结
[操作总结和注意事项]
```

## 错误处理策略

### 工具执行失败
1. 分析失败原因：参数错误？权限问题？主机不可达？
2. 尝试恢复：修正参数、使用替代方法
3. 无法恢复时：向用户清晰说明情况

### 数据异常
1. 识别异常模式：数值异常、趋势异常
2. 关联分析：结合多个指标判断
3. 给出告警：明确标注异常等级

## 安全准则

### 危险操作识别
以下操作属于危险操作，需要特别说明风险：
- 删除文件/目录（rm 命令）
- 修改系统配置（修改 /etc 下的文件）
- 重启/关机操作
- 清理日志/缓存
- 修改权限

### 风险告知格式
```
⚠️ 警告：此操作存在风险

操作：[操作描述]
风险：[可能的影响]
建议：[安全建议]

是否继续执行？（建议用户确认）
```

## 可用工具

%s

## 关键原则
1. **信息优先**：先收集信息，再下结论
2. **批量高效**：能批量就不要分次
3. **安全第一**：危险操作必须说明风险
4. **结果导向**：给用户可操作的建议，而非只是数据
5. **清晰简洁**：使用结构化输出，便于阅读

---
记住：你的目标是帮助用户快速定位和解决运维问题，而不仅仅是执行命令。
`
```

---

### 2.2 改进工具描述

**当前** (`internal/tool/builtin/system.go`):
```go
func (t *CheckCPUTool) Description() string {
    return `检查节点的 CPU 使用情况。
返回 CPU 使用率、负载等信息。
可以检查单个节点或所有节点。
当用户询问"CPU 使用率"、"负载高"、"性能问题"时使用。`
}
```

**改进后** - 创建 `internal/tool/builtin/system_enhanced.go`:
```go
package builtin

func (t *CheckCPUTool) Description() string {
    return `# 检查 CPU 使用情况和系统负载

## 功能说明
检查指定主机的 CPU 使用率、核心数、系统负载等信息。

## 适用场景
- 用户询问"CPU 使用率"、"负载情况"
- 性能问题诊断：系统变慢、响应延迟
- 资源规划：评估当前资源使用情况
- 故障排查：判断是否存在 CPU 瓶颈

## 输出数据解读
- CPU 使用率：用户进程 + 系统进程的占用比例
- Load Average：1/5/15 分钟平均负载，对比核心数判断压力
- 进程列表：按 CPU 占用排序的进程信息

## 判断标准
- 正常：CPU < 70%, Load < 核心数
- 警告：CPU 70-85%, Load 接近核心数
- 危险：CPU > 85%, Load > 核心数 * 1.5

## 使用建议
- 单主机检查：使用 host 参数
- 多主机检查：使用 hosts 参数批量执行
- 配合 check_memory 一起使用可全面评估系统状态
- 发现 CPU 高时，继续用 check_process 找出占用进程`
}
```

**其他工具类似改进**，每个工具描述包含：
1. 功能说明
2. 适用场景
3. 输出数据解读
4. 判断标准/阈值
5. 使用建议

---

### 2.3 增强 ReAct 循环（添加思考步骤）

**当前** (`internal/agent/agent.go` 第 99-131 行):
```go
for i := 0; i < a.maxLoops; i++ {
    // 直接调用 LLM
    resp, err := a.llmClient.ChatWithTools(ctx, messages, toolDefs)
    if err != nil {
        return nil, fmt.Errorf("LLM 调用失败: %w", err)
    }

    // 检查是否有工具调用
    if resp.HasToolCalls() {
        // 执行工具...
    }

    // 没有工具调用，返回最终回复
    finalReply = resp.Message.Content
    break
}
```

**改进后** - 新建 `internal/agent/agent_enhanced.go`:

```go
package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "ai-ops/internal/llm"
    "ai-ops/internal/ssh"
    "ai-ops/internal/tool"
    "ai-ops/pkg/logger"

    "go.uber.org/zap"
)

// ThinkingStep 思考步骤
type ThinkingStep struct {
    Step    string `json:"step"`    // 步骤类型: analyze, plan, execute, reflect, summarize
    Content string `json:"content"` // 思考内容
}

// EnhancedAgent 增强版 Agent
type EnhancedAgent struct {
    llmClient    llm.Client
    toolRegistry *tool.Registry
    sshPool      *ssh.Pool
    maxLoops     int
    timeout      time.Duration
}

// NewEnhancedAgent 创建增强版 Agent
func NewEnhancedAgent(llmClient llm.Client, toolRegistry *tool.Registry, sshPool *ssh.Pool, cfg Config) *EnhancedAgent {
    if cfg.MaxLoops == 0 {
        cfg.MaxLoops = 10
    }
    if cfg.Timeout == 0 {
        cfg.Timeout = 5 * time.Minute
    }

    return &EnhancedAgent{
        llmClient:    llmClient,
        toolRegistry: toolRegistry,
        sshPool:      sshPool,
        maxLoops:     cfg.MaxLoops,
        timeout:      cfg.Timeout,
    }
}

// ChatWithThinking 带思考过程的对话
func (a *EnhancedAgent) ChatWithThinking(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, a.timeout)
    defer cancel()

    messages := a.buildMessages(req)
    tools := a.buildToolDefs()

    var toolCallRecords []ToolCallRecord
    var thinkingSteps []ThinkingStep
    var finalReply string

    for i := 0; i < a.maxLoops; i++ {
        logger.Debug("Agent 循环", zap.Int("loop", i+1))

        // === 第一步：分析 ===
        if i == 0 {
            analyzeStep := a.analyzeRequest(req.Message, req.Hosts)
            thinkingSteps = append(thinkingSteps, analyzeStep)
            logger.Info("分析用户请求", zap.String("thinking", analyzeStep.Content))
        }

        // === 第二步：规划（如有工具调用历史） ===
        if len(toolCallRecords) > 0 {
            planStep := a.planNextAction(toolCallRecords)
            thinkingSteps = append(thinkingSteps, planStep)
            logger.Info("规划下一步", zap.String("thinking", planStep.Content))
        }

        // === 第三步：执行（调用 LLM） ===
        resp, err := a.llmClient.ChatWithTools(ctx, messages, tools)
        if err != nil {
            return nil, fmt.Errorf("LLM 调用失败: %w", err)
        }

        if resp.HasToolCalls() {
            // 优化工具调用：检测是否可以批量
            optimizedCalls := a.optimizeToolCalls(resp.Message.ToolCalls, req.Hosts)

            // 执行工具
            toolResults, records := a.executeToolCalls(ctx, optimizedCalls, req.Hosts)
            toolCallRecords = append(toolCallRecords, records...)

            messages = append(messages, resp.Message)
            messages = append(messages, toolResults...)

            // === 第四步：反思 ===
            reflectStep := a.reflectOnResults(records)
            if reflectStep.Content != "" {
                thinkingSteps = append(thinkingSteps, reflectStep)
                logger.Info("反思结果", zap.String("thinking", reflectStep.Content))
            }

            continue
        }

        // === 第五步：总结 ===
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
        Thinking:  formatThinkingSteps(thinkingSteps),
    }, nil
}

// analyzeRequest 分析用户请求
func (a *EnhancedAgent) analyzeRequest(userMessage string, hosts []string) ThinkingStep {
    // 使用简单的规则分析（也可以调用 LLM）
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
func (a *EnhancedAgent) planNextAction(records []ToolCallRecord) ThinkingStep {
    // 分析上一步结果，规划下一步
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
func (a *EnhancedAgent) reflectOnResults(records []ToolCallRecord) ThinkingStep {
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
        Content: fmt.Sprintf("执行结果反思：\n%s", joinString(insights, "\n")),
    }
}

// optimizeToolCalls 优化工具调用（检测批量机会）
func (a *EnhancedAgent) optimizeToolCalls(calls []llm.ToolCall, contextHosts []string) []llm.ToolCall {
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
        if a.canBatch(toolName) {
            merged := a.mergeToolCalls(toolName, calls, contextHosts)
            optimized = append(optimized, merged)
        } else {
            optimized = append(optimized, calls...)
        }
    }

    return optimized
}

// canBatch 检查工具是否支持批量
func (a *EnhancedAgent) canBatch(toolName string) bool {
    batchableTools := map[string]bool{
        "check_cpu":    true,
        "check_memory": true,
        "check_disk":   true,
        "list_hosts":   true,
    }
    return batchableTools[toolName]
}

// mergeToolCalls 合并工具调用为批量调用
func (a *EnhancedAgent) mergeToolCalls(toolName string, calls []llm.ToolCall, contextHosts []string) llm.ToolCall {
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

        // 合并其他参数
        for k, v := range params {
            if k != "host" && k != "hosts" {
                paramsMap[k] = v
            }
        }
    }

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

// formatThinkingSteps 格式化思考步骤
func formatThinkingSteps(steps []ThinkingStep) string {
    if len(steps) == 0 {
        return ""
    }

    var result string
    result = "## 思考过程\n\n"

    for i, step := range steps {
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

func joinString(strs []string, sep string) string {
    if len(strs) == 0 {
        return ""
    }
    result := strs[0]
    for i := 1; i < len(strs); i++ {
        result += sep + strs[i]
    }
    return result
}

// 继承原有的方法...
func (a *EnhancedAgent) buildMessages(req ChatRequest) []llm.Message { ... }
func (a *EnhancedAgent) buildToolDefs() []llm.ToolDef { ... }
func (a *EnhancedAgent) executeToolCalls(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) { ... }
```

---

### 2.4 可选：Subagent 模式（高级功能）

对于复杂场景，可以引入专门的 Subagent：

**新建 `internal/agent/subagent.go`**:

```go
package agent

import "fmt"

// SubagentType Subagent 类型
type SubagentType string

const (
    SubagentTypePlan    SubagentType = "plan"    // 规划 Agent
    SubagentTypeExplore SubagentType = "explore" // 探索 Agent
    SubagentTypeTask    SubagentType = "task"    // 任务 Agent
)

// SubagentConfig Subagent 配置
type SubagentConfig struct {
    Name         string
    Type         SubagentType
    SystemPrompt string
    AllowedTools []string // 只能访问这些工具
}

// Built-in Subagent 配置
var SubagentConfigs = []SubagentConfig{
    {
        Name: "planner",
        Type: SubagentTypePlan,
        SystemPrompt: `# 规划 Agent

你是专门负责制定执行计划的 Agent。

## 职责
1. 理解用户的最终目标
2. 分析当前环境状态
3. 制定分步执行计划
4. 识别潜在风险

## 约束
- 只能使用只读工具（list_*、check_*）
- 不执行任何修改操作
- 输出结构化的执行计划

## 输出格式
\`\`\`
# 执行计划

## 目标
[用户目标的描述]

## 当前状态
[环境状态概述]

## 执行步骤
1. [步骤1] - 使用工具xxx - 预期结果
2. [步骤2] - 使用工具xxx - 预期结果
...

## 风险提示
[潜在风险]
\`\`\`
`,
        AllowedTools: []string{"list_hosts", "check_cpu", "check_memory", "check_disk"},
    },
    {
        Name: "explorer",
        Type: SubagentTypeExplore,
        SystemPrompt: `# 探索 Agent

你是专门负责快速浏览和理解代码库/系统的 Agent。

## 职责
1. 快速浏览目录结构
2. 识别关键文件和配置
3. 理解系统架构
4. 定位相关组件

## 约束
- 只进行只读操作
- 快速给出概览，不做深度分析

## 输出格式
\`\`\`
# 系统探索报告

## 架构概览
[系统架构描述]

## 关键组件
- [组件1]: [说明]
- [组件2]: [说明]

## 目录结构
\`\`\`
`,
        AllowedTools: []string{"run_command"}, // 用于 ls, find 等命令
    },
    {
        Name: "executor",
        Type: SubagentTypeTask,
        SystemPrompt: `# 执行 Agent

你是专门负责执行具体任务的 Agent。

## 职责
1. 根据计划执行操作
2. 处理执行过程中的异常
3. 验证执行结果
4. 报告执行状态

## 工具使用
- 可以使用所有工具
- 遵循安全准则
- 危险操作需要说明风险

## 输出格式
\`\`\`
# 执行报告

## 执行的操作
1. [操作1] - 结果
2. [操作2] - 结果

## 验证结果
[验证信息]

## 建议
[后续建议]
\`\`\`
`,
        AllowedTools: []string{"*"}, // 所有工具
    },
}

// GetSubagentConfig 获取 Subagent 配置
func GetSubagentConfig(name string) (*SubagentConfig, error) {
    for _, cfg := range SubagentConfigs {
        if cfg.Name == name {
            return &cfg, nil
        }
    }
    return nil, fmt.Errorf("subagent not found: %s", name)
}

// GetSubagentConfigByType 根据 Type 获取 Subagent 配置
func GetSubagentConfigByType(typ SubagentType) *SubagentConfig {
    for _, cfg := range SubagentConfigs {
        if cfg.Type == typ {
            return &cfg
        }
    }
    return nil
}
```

---

### 2.5 配置改进

**修改 `internal/config/config.go`**，添加 Agent 配置选项：

```go
// AgentConfig Agent 配置
type AgentConfig struct {
    MaxLoops      int            `yaml:"max_loops"`       // 最大循环次数
    Timeout       time.Duration  `yaml:"timeout"`         // 单次请求超时
    EnableThinking bool          `yaml:"enable_thinking"` // 启用思考过程
    EnableSubagent bool          `yaml:"enable_subagent"` // 启用 Subagent 模式
    PromptVersion string         `yaml:"prompt_version"`  // 提示词版本: standard / enhanced
}
```

**修改配置默认值**：

```go
func (c *Config) setDefaults() {
    // ...

    // Agent 默认值
    if c.Agent.MaxLoops == 0 {
        c.Agent.MaxLoops = 10
    }
    if c.Agent.Timeout == 0 {
        c.Agent.Timeout = 5 * time.Minute
    }
    if c.Agent.PromptVersion == "" {
        c.Agent.PromptVersion = "enhanced" // 默认使用增强版
    }
    c.Agent.EnableThinking = true // 默认启用思考过程
}
```

**修改 `config.yaml`**：

```yaml
agent:
  max_loops: 10
  timeout: 5m
  prompt_version: enhanced  # standard / enhanced
  enable_thinking: true    # 启用思考过程输出
  enable_subagent: false   # Subagent 模式（实验性功能）
```

---

## 三、实施计划

### Phase 1: 基础改进（必须）
1. ✅ 创建 `internal/agent/prompt_enhanced.go` - 增强系统提示词
2. ✅ 创建 `internal/tool/builtin/*_enhanced.go` - 改进工具描述
3. ✅ 修改 `internal/agent/prompt.go` 添加版本选择
4. ✅ 修改 `internal/config/config.go` 添加配置选项

### Phase 2: 思考过程（推荐）
1. ✅ 创建 `internal/agent/agent_enhanced.go` - 添加思考步骤
2. ✅ 修改 `internal/agent/agent.go` 添加 `ChatWithThinking` 方法
3. ✅ 修改 API handler 支持返回思考过程

### Phase 3: 高级功能（可选）
1. ⭕ 创建 `internal/agent/subagent.go` - Subagent 支持
2. ⭕ 实现多 Agent 协调逻辑
3. ⭕ 添加 Subagent 性能监控

---

## 四、测试验证

### 单元测试
```bash
# 测试增强版提示词
go test -v ./internal/agent/... -run TestEnhancedPrompt

# 测试工具调用优化
go test -v ./internal/agent/... -run TestOptimizeToolCalls
```

### 集成测试
```bash
# 启动服务
go run cmd/server/main.go

# 测试场景
# 1. 简单查询："检查所有主机的 CPU"
# 2. 问题诊断："node1 变慢了，帮我看看"
# 3. 批量操作："检查所有主机的磁盘使用率"
```

---

## 五、效果对比

| 指标 | 改进前 | 改进后 |
|------|--------|--------|
| 系统提示词 | ~500 tokens | ~3000 tokens |
| 工具描述 | 1-2 句话 | 结构化 5 部分 |
| 思考过程 | 无 | 5 步骤输出 |
| 工具调用优化 | 无 | 自动检测批量 |
| 输出格式 | 自由文本 | 结构化报告 |
| 安全意识 | 基础 | 明确的风险提示 |

---

## 六、参考资源

1. [Claude Code: Best practices for agentic coding](https://www.anthropic.com/engineering/claude-code-best-practices)
2. [Making Claude Code more secure and autonomous](https://www.anthropic.com/engineering/claude-code-sandboxing)
3. [How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
4. [Piebald-AI/claude-code-system-prompts (GitHub)](https://github.com/Piebald-AI/claude-code-system-prompts)
