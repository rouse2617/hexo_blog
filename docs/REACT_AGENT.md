# ReAct Agent 实现文档

## 概述

ReAct (Reasoning + Acting) 是一种增强的 AI Agent 模式，通过显式的 **思考-行动-观察** 循环提供更透明、更可控的推理过程。

## 架构

### 核心组件

1. **ReActAgent** (`internal/agent/react.go`)
   - 实现完整的 ReAct 循环逻辑
   - 管理思考步骤和工具执行
   - 记录完整的推理链

2. **ReActStep** 数据结构
   ```go
   type ReActStep struct {
       Phase      string         // thought, action, observation
       Content    string         // 步骤内容
       ToolCalls  []llm.ToolCall // 工具调用（action 阶段）
       ToolResult string         // 工具结果（observation 阶段）
       Timestamp  int64          // 时间戳
   }
   ```

3. **集成点**
   - `Agent` 结构体包含 `ReActAgent` 实例
   - `ChatService` 支持切换 ReAct 模式
   - `ChatHandler` 提供 API 控制

## ReAct 循环流程

```
用户请求
    ↓
┌─────────────────────┐
│ 1. Thought (思考)   │ ← LLM 分析：已知什么？需要什么？下一步？
└─────────────────────┘
    ↓
┌─────────────────────┐
│ 2. Action (行动)    │ ← 选择并执行工具
└─────────────────────┘
    ↓
┌─────────────────────┐
│ 3. Observation (观察)│ ← 分析工具返回结果
└─────────────────────┘
    ↓
   需要更多信息？ ──Yes──> 回到 Thought
    |
   No
    ↓
给出最终答案
```

## 使用方法

### 1. 启用 ReAct 模式

在 `cmd/server/main.go` 中：

```go
// 方式 1: 通过 Handler 启用
chatHandler := handler.NewChatHandler(chatService)
chatHandler.SetEnableReAct(true)

// 方式 2: 直接通过 Service 启用
chatService.SetEnableReAct(true)
```

### 2. 环境变量配置（推荐）

在 `config.yaml` 中添加：

```yaml
agent:
  mode: react  # 可选值: normal, thinking, react
  max_loops: 10
  timeout: 5m
```

### 3. API 请求示例

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "检查所有主机的 CPU 使用率",
    "stream": false
  }'
```

### 4. 响应格式

```json
{
  "session_id": "xxx",
  "reply": "已检查 3 台主机，CPU 使用率正常...",
  "tool_calls": [
    {
      "tool": "check_cpu",
      "params": {"hosts": ["host1", "host2"]},
      "result": "{...}"
    }
  ],
  "thinking": "[1] 💭 思考: 需要获取主机列表...\n[2] ⚡ 行动: list_hosts\n[3] 👁 观察: 发现 3 台主机...",
  "react_steps": [
    {
      "phase": "thought",
      "content": "需要先获取主机列表...",
      "timestamp": 1234567890
    },
    {
      "phase": "action",
      "content": "执行 1 个工具调用",
      "tool_calls": [...],
      "timestamp": 1234567891
    },
    {
      "phase": "observation",
      "content": "成功: 返回 3 台主机",
      "timestamp": 1234567892
    }
  ]
}
```

## ReAct vs 普通 Agent vs Thinking Agent

| 特性 | 普通 Agent | Thinking Agent | ReAct Agent |
|------|-----------|----------------|-------------|
| 推理透明度 | 低 | 中 | **高** |
| 步骤记录 | 无 | 简单 | **详细** |
| 工具选择理由 | 隐式 | 部分 | **显式** |
| 观察分析 | 无 | 基础 | **深入** |
| 适用场景 | 简单任务 | 中等复杂度 | **复杂多步骤** |
| Token 消耗 | 低 | 中 | 高 |

## 配置选项

### ReActAgent 配置

```go
reactAgent := NewReActAgent(registry, llmClient, sshPool)

// 设置最大循环次数（防止无限循环）
reactAgent.SetMaxLoops(10)

// 设置超时时间
reactAgent.SetTimeout(5 * time.Minute)
```

### 提示词定制

`buildReActSystemPrompt()` 方法构建了详细的系统提示，包含：

- **工作模式说明**：Thought → Action → Observation 循环
- **思考模板**：引导 LLM 进行结构化思考
- **行动原则**：如何选择和使用工具
- **观察要点**：如何分析工具返回结果
- **回答格式**：最终答案的呈现方式

## 最佳实践

### 1. 适用场景

ReAct 模式最适合：
- **多步骤诊断**：需要多次信息收集和分析
- **复杂决策**：需要权衡多个因素
- **透明度要求高**：需要向用户展示推理过程
- **调试和优化**：需要理解 Agent 的决策逻辑

### 2. 不适用场景

- **简单问答**：直接查询类请求
- **实时性要求高**：流式响应优先的场景
- **Token 预算紧张**：ReAct 会消耗更多 Token

### 3. 提示词优化

```go
// 根据具体业务定制系统提示
func (a *ReActAgent) buildCustomSystemPrompt() string {
    return fmt.Sprintf(`
# 自定义 ReAct Agent

## 领域知识
- 重点关注：%s
- 常见问题：%s

## 工作流程
%s
    `, domainKnowledge, commonIssues, workFlow)
}
```

### 4. 错误处理

ReAct Agent 会自动处理工具调用错误：

```go
// 在 Observation 阶段
if record.Error != "" {
    resultStr = fmt.Sprintf("错误: %s", record.Error)
    // Agent 会在下一个 Thought 阶段分析错误并调整策略
}
```

## 性能优化

### 1. 批量工具调用

ReAct Agent 集成了工具调用优化：

```go
// 检测到多个主机上的同一工具调用
// 自动合并为批量调用
optimizedCalls := optimizeToolCalls(toolCalls, hosts)
```

### 2. 结果截断

防止观察结果过长：

```go
resultPreview := record.Result
if len(resultPreview) > 500 {
    resultPreview = resultPreview[:500] + "...(truncated)"
}
```

### 3. 循环限制

设置合理的最大循环次数：

```go
maxLoops := 10  // 根据任务复杂度调整
```

## 监控和调试

### 日志输出

```go
logger.Info("ReAct 循环", zap.Int("loop", i+1))
logger.Info("ReAct 思考", zap.String("thought", thought))
logger.Info("ReAct 行动", zap.Int("tool_calls", len(toolCalls)))
logger.Info("ReAct 观察", zap.String("result", resultStr))
```

### 思考步骤查看

前端可以展示 ReAct 步骤：

```vue
<div v-for="(step, index) in reactSteps" :key="index">
  <div v-if="step.phase === 'thought'">
    💭 思考: {{ step.content }}
  </div>
  <div v-if="step.phase === 'action'">
    ⚡ 行动: {{ step.tool_calls.map(t => t.function.name).join(', ') }}
  </div>
  <div v-if="step.phase === 'observation'">
    👁 观察: {{ step.content }}
  </div>
</div>
```

## 扩展和定制

### 自定义 ReAct 步骤

```go
type CustomReActStep struct {
    ReActStep
    CustomField string `json:"custom_field"`
    Metrics     map[string]float64 `json:"metrics"`
}
```

### 添加新的阶段

```go
const (
    PhaseThought     = "thought"
    PhaseAction      = "action"
    PhaseObservation = "observation"
    PhaseReflection  = "reflection"  // 新增：反思阶段
    PhasePlanning    = "planning"    // 新增：规划阶段
)
```

### 集成外部知识库

```go
func (a *ReActAgent) enhanceWithKnowledge(ctx context.Context, query string) string {
    // 从知识库检索相关信息
    knowledge := a.knowledgeBase.Search(query)
    return fmt.Sprintf("\n## 相关知识\n%s\n", knowledge)
}
```

## 测试

### 单元测试示例

```go
func TestReActAgent(t *testing.T) {
    registry := tool.NewRegistry()
    // 注册测试工具...

    llmClient := &MockLLMClient{}
    reactAgent := NewReActAgent(registry, llmClient, nil)

    req := ChatRequest{
        Message: "检查主机状态",
        Hosts:   []string{"host1"},
    }

    resp, err := reactAgent.ChatReAct(context.Background(), req)
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.Reply)
    assert.Greater(t, len(resp.ReActSteps), 0)
}
```

### 集成测试

```go
func TestReActAgentIntegration(t *testing.T) {
    // 测试完整的 ReAct 循环
    // 验证工具调用、错误处理、步骤记录等
}
```

## 故障排查

### 问题 1: 无限循环

**症状**: Agent 达到最大循环次数仍未结束

**解决**:
1. 检查工具返回结果是否清晰
2. 优化系统提示词
3. 增加 `maxLoops` 或改进终止条件

### 问题 2: 工具选择不当

**症状**: Agent 重复调用无效工具

**解决**:
1. 检查工具描述是否清晰
2. 在 Observation 阶段加强错误分析
3. 添加工具使用示例到系统提示

### 问题 3: Token 消耗过高

**症状**: 单次对话 Token 使用量过大

**解决**:
1. 减少 `maxLoops`
2. 限制工具结果长度
3. 使用简洁的系统提示词

## 未来改进

- [ ] 支持 ReAct 流式响应
- [ ] 添加 ReAct 步骤可视化
- [ ] 实现 ReAct 性能分析
- [ ] 支持 ReAct 步骤回溯和修正
- [ ] 集成更多工具优化策略
- [ ] 添加 ReAct 模板库

## 参考资料

- [ReAct: Synergizing Reasoning and Acting in Language Models](https://arxiv.org/abs/2210.03629)
- [Effective Go - Agent 模式](https://go.dev/doc/effective_go)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)

## 贡献

欢迎提交 Issue 和 PR 来改进 ReAct Agent 实现！
