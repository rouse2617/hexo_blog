# ReAct Agent 快速开始指南

## 5 分钟快速上手

### 步骤 1: 启用 ReAct 模式

在 `cmd/server/main.go` 中添加一行代码：

```go
chatService.SetEnableReAct(true)
```

### 步骤 2: 启动服务器

```bash
go run cmd/server/main.go
```

### 步骤 3: 发送请求

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "检查所有主机的状态"}'
```

### 步骤 4: 查看响应

```json
{
  "reply": "已完成主机状态检查...",
  "thinking": "[1] 💭 思考: 需要获取主机列表\n[2] ⚡ 行动: list_hosts\n[3] 👁 观察: 找到 3 台主机...",
  "react_steps": [...]
}
```

完成！🎉

## 核心概念

### ReAct 循环

```
Thought (思考) → Action (行动) → Observation (观察) → 循环
```

1. **Thought**: AI 分析问题，决定下一步
2. **Action**: 调用工具获取信息
3. **Observation**: 分析工具返回的结果
4. **循环**: 重复直到有足够信息给出答案

### 与普通模式的区别

| 普通模式 | ReAct 模式 |
|---------|-----------|
| 直接执行工具 | 先思考再行动 |
| 不显示推理过程 | 显示完整思考链 |
| 适合简单任务 | 适合复杂多步骤任务 |

## 实用示例

### 示例 1: 主机监控

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "检查所有主机的 CPU 和内存使用情况，如果有异常告诉我"
  }'
```

**ReAct 过程:**
```
💭 思考: 需要先获取主机列表
⚡ 行动: list_hosts
👁 观察: 找到 3 台主机

💭 思考: 现在检查 CPU 和内存
⚡ 行动: check_cpu, check_memory
👁 观察: host1 CPU 85%, host2 内存 90%

💭 思考: 发现异常，需要进一步检查
⚡ 行动: check_processes
👁 观察: host1 上有高 CPU 进程

✅ 最终答案: host1 CPU 使用率过高(85%)，由 java 进程导致；host2 内存使用率过高(90%)，建议清理...
```

### 示例 2: 故障诊断

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "web-01 主机响应很慢，帮我诊断问题",
    "hosts": ["web-01"]
  }'
```

**ReAct 过程:**
```
💭 思考: 响应慢可能的原因: CPU、内存、磁盘IO、网络
⚡ 行动: check_cpu
👁 观察: CPU 45%, 正常

💭 思考: CPU 正常，检查内存
⚡ 行动: check_memory
👁 观察: 内存 82%, 略高但不是主要原因

💭 思考: 检查磁盘 IO
⚡ 行动: check_disk_io
👁 观察: 磁盘 IO 使用率 95%，很高！

💭 思考: 找到问题根源，检查哪个进程导致高 IO
⚡ 行动: list_processes
👁 观察: 数据库进程在大量读写

✅ 最终答案: 诊断结果：web-01 磁盘 IO 使用率过高(95%)，由数据库进程导致。建议检查数据库查询优化...
```

### 示例 3: 批量操作

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "清理所有主机的临时文件，超过 7 天的"
  }'
```

**ReAct 过程:**
```
💭 思考: 需要在所有主机上执行清理操作
⚡ 行动: list_hosts
👁 观察: 找到 5 台主机

💭 思考: 批量清理临时文件
⚡ 行动: clean_temp_files (hosts: all)
👁 观察: 清理完成，释放 2.3GB 空间

✅ 最终答案: 已在 5 台主机上清理超过 7 天的临时文件，共释放 2.3GB 空间...
```

## 前端集成示例

### React 示例

```jsx
import { useState } from 'react';

function ChatWithReAct() {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState('');

  const sendMessage = async () => {
    const response = await fetch('/api/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: input, stream: false }),
    });

    const data = await response.json();
    setMessages([...messages, { user: input, ai: data.data }]);
  };

  return (
    <div>
      {messages.map((msg, i) => (
        <div key={i}>
          <p>User: {msg.user}</p>

          {/* ReAct 步骤 */}
          {msg.ai.react_steps && (
            <div className="react-steps">
              {msg.ai.react_steps.map((step, j) => (
                <div key={j} className={`step-${step.phase}`}>
                  {step.phase === 'thought' && `💭 ${step.content}`}
                  {step.phase === 'action' && `⚡ ${step.tool_calls.map(t => t.function.name).join(', ')}`}
                  {step.phase === 'observation' && `👁 ${step.content}`}
                </div>
              ))}
            </div>
          )}

          {/* 最终回复 */}
          <p>AI: {msg.ai.reply}</p>
        </div>
      ))}

      <input value={input} onChange={e => setInput(e.target.value)} />
      <button onClick={sendMessage}>Send</button>
    </div>
  );
}
```

## 配置调优

### 调整循环次数

```go
// 对于简单任务，减少循环
reactAgent.SetMaxLoops(5)

// 对于复杂任务，增加循环
reactAgent.SetMaxLoops(15)
```

### 调整超时时间

```go
// 对于快速查询
reactAgent.SetTimeout(1 * time.Minute)

// 对于耗时操作
reactAgent.SetTimeout(10 * time.Minute)
```

### 切换模式

```go
// 普通模式 (最快)
chatService.SetEnableThinking(false)
chatService.SetEnableReAct(false)

// Thinking 模式 (平衡)
chatService.SetEnableThinking(true)
chatService.SetEnableReAct(false)

// ReAct 模式 (最详细)
chatService.SetEnableThinking(false)
chatService.SetEnableReAct(true)
```

## 故障排查

### 问题: Agent 循环太多

**解决方案:**
```go
// 减少最大循环次数
reactAgent.SetMaxLoops(5)
```

### 问题: 不显示思考过程

**解决方案:**
```go
// 确保启用了 ReAct
chatService.SetEnableReAct(true)

// 检查响应中是否包含 react_steps
console.log(response.react_steps)
```

### 问题: Token 消耗过多

**解决方案:**
```go
// 1. 减少循环次数
reactAgent.SetMaxLoops(5)

// 2. 限制工具结果长度
// 在 react.go 中修改:
if len(resultPreview) > 200 {  // 从 500 改为 200
    resultPreview = resultPreview[:200] + "...(truncated)"
}

// 3. 使用简洁的系统提示词
```

## 最佳实践

### ✅ 推荐做法

1. **复杂任务使用 ReAct**
   - 多步骤诊断
   - 需要详细分析
   - 透明度要求高

2. **简单任务使用普通模式**
   - 直接查询
   - 快速响应
   - Token 预算紧张

3. **合理配置循环次数**
   - 简单任务: 3-5 次
   - 中等任务: 5-10 次
   - 复杂任务: 10-15 次

### ❌ 避免做法

1. **不要在简单查询中使用 ReAct**
   ```bash
   # ❌ 不好
   message: "现在几点了"

   # ✅ 好
   message: "检查为什么服务器变慢"
   ```

2. **不要设置过大的循环次数**
   ```go
   // ❌ 不好
   reactAgent.SetMaxLoops(100)

   // ✅ 好
   reactAgent.SetMaxLoops(10)
   ```

3. **不要忽视错误处理**
   ```go
   // ❌ 不好
   resp, _ := reactAgent.ChatReAct(ctx, req)

   // ✅ 好
   resp, err := reactAgent.ChatReAct(ctx, req)
   if err != nil {
       log.Error("ReAct 失败", err)
       return
   }
   ```

## 进阶技巧

### 自定义 ReAct 提示词

```go
func (a *ReActAgent) buildCustomSystemPrompt() string {
    return `
# 专业运维 ReAct Agent

## 你的专长
- Linux 系统管理
- Docker 容器编排
- 数据库性能优化

## 工作流程
1. Thought: 明确问题，收集信息
2. Action: 执行诊断命令
3. Observation: 分析输出，判断异常

## 回答格式
- 使用专业的运维术语
- 提供具体的命令和参数
- 给出可操作的建议
`
}
```

### 添加自定义步骤

```go
type CustomReActStep struct {
    agent.ReActStep
    Confidence float64 `json:"confidence"` // 置信度
    Priority   string  `json:"priority"`   // 优先级
}
```

### 集成知识库

```go
func (a *ReActAgent) ChatReActWithKnowledge(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // 从知识库检索相关信息
    knowledge := a.knowledgeBase.Search(req.Message)

    // 将知识库内容添加到系统提示
    systemPrompt := a.buildReActSystemPrompt()
    systemPrompt += fmt.Sprintf("\n## 相关知识\n%s\n", knowledge)

    // 继续正常流程...
}
```

## 下一步

- 📖 阅读完整文档: [REACT_AGENT.md](./REACT_AGENT.md)
- 💡 查看更多示例: [REACT_EXAMPLES.md](./REACT_EXAMPLES.md)
- 🚀 开始构建你的应用！

## 支持

遇到问题？查看：
- 文档: `/docs`
- 示例: `/docs/REACT_EXAMPLES.md`
- Issue: GitHub Issues

祝使用愉快！🎉
