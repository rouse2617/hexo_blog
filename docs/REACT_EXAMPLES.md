# ReAct Agent 使用示例

## 基础使用

### 启用 ReAct 模式

在 `cmd/server/main.go` 中：

```go
package main

import (
    "ai-ops/internal/api/handler"
    "ai-ops/internal/service"
    "ai-ops/internal/agent"
    // ... 其他导入
)

func main() {
    // ... 初始化代码

    // 创建 ChatService
    chatService := service.NewChatService(
        agentInstance,
        sessionRepo,
        false, // enableThinking = false
    )

    // 启用 ReAct 模式
    chatService.SetEnableReAct(true)

    // 创建 ChatHandler
    chatHandler := handler.NewChatHandler(chatService)

    // ... 路由设置等
}
```

### 通过配置文件启用

在 `config.yaml` 中：

```yaml
server:
  port: 8080

agent:
  mode: react  # normal, thinking, react
  max_loops: 10
  timeout: 5m
  enable_thinking: false
  enable_react: true

llm:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: gpt-4
  base_url: https://api.openai.com/v1
```

在 `cmd/server/main.go` 中读取配置：

```go
func setupChatService(cfg *config.Config, agentInstance *agent.Agent, sessionRepo repository.SessionRepository) *service.ChatService {
    chatService := service.NewChatService(
        agentInstance,
        sessionRepo,
        cfg.Agent.EnableThinking,
    )

    // 根据配置启用 ReAct
    if cfg.Agent.Mode == "react" || cfg.Agent.EnableReAct {
        chatService.SetEnableReAct(true)
    }

    return chatService
}
```

## API 调用示例

### cURL 示例

```bash
# 1. 简单查询
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "列出所有主机",
    "stream": false
  }'

# 2. 带主机的查询
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "检查这些主机的 CPU 使用率",
    "hosts": ["server1", "server2", "server3"],
    "stream": false
  }'

# 3. 复杂多步骤任务
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "检查所有主机的磁盘使用情况，如果超过 80% 告诉我哪些文件占用空间大",
    "stream": false
  }'
```

### JavaScript/TypeScript 示例

```typescript
interface ChatRequest {
  message: string;
  session_id?: string;
  hosts?: string[];
  stream?: boolean;
}

interface ReActStep {
  phase: 'thought' | 'action' | 'observation';
  content: string;
  timestamp: number;
  tool_calls?: any[];
}

interface ChatResponse {
  session_id: string;
  reply: string;
  tool_calls: any[];
  thinking: string;
  react_steps: ReActStep[];
}

async function chatWithReAct(message: string, hosts?: string[]): Promise<ChatResponse> {
  const response = await fetch('http://localhost:8080/api/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message,
      hosts,
      stream: false,
    }),
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data = await response.json();
  return data.data;
}

// 使用示例
async function main() {
  const result = await chatWithReAct(
    '检查所有主机的内存使用情况',
  );

  console.log('最终答案:', result.reply);
  console.log('\nReAct 步骤:');
  result.react_steps.forEach((step, i) => {
    console.log(`\n[步骤 ${i + 1}] ${step.phase.toUpperCase()}`);
    console.log(step.content);
  });
}

main();
```

### Vue 3 组件示例

```vue
<template>
  <div class="react-chat">
    <!-- 消息列表 -->
    <div class="messages">
      <div v-for="(msg, index) in messages" :key="index" class="message">
        <div class="user-message">{{ msg.user }}</div>

        <div class="assistant-message">
          <!-- ReAct 步骤展示 -->
          <div v-if="msg.reactSteps && msg.reactSteps.length > 0" class="react-steps">
            <div
              v-for="(step, stepIndex) in msg.reactSteps"
              :key="stepIndex"
              class="react-step"
              :class="`step-${step.phase}`"
            >
              <div class="step-header">
                <span class="step-icon">{{ getStepIcon(step.phase) }}</span>
                <span class="step-title">{{ getStepTitle(step.phase) }}</span>
                <span class="step-time">{{ formatTime(step.timestamp) }}</span>
              </div>

              <div class="step-content">
                {{ step.content }}
              </div>

              <!-- Action 阶段显示工具调用 -->
              <div v-if="step.phase === 'action' && step.tool_calls" class="tool-calls">
                <div v-for="tool in step.tool_calls" :key="tool.id" class="tool-call">
                  <code>{{ tool.function.name }}</code>
                </div>
              </div>
            </div>
          </div>

          <!-- 最终回复 -->
          <div class="final-reply">
            {{ msg.reply }}
          </div>
        </div>
      </div>
    </div>

    <!-- 输入框 -->
    <div class="input-area">
      <el-input
        v-model="userInput"
        type="textarea"
        placeholder="输入您的问题..."
        @keydown.ctrl.enter="sendMessage"
      />
      <el-button type="primary" @click="sendMessage">发送 (Ctrl+Enter)</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { chatApi } from '@/api/chat';

interface ReActStep {
  phase: 'thought' | 'action' | 'observation';
  content: string;
  timestamp: number;
  tool_calls?: any[];
}

interface Message {
  user: string;
  reply: string;
  reactSteps?: ReActStep[];
}

const userInput = ref('');
const messages = ref<Message[]>([]);

function getStepIcon(phase: string): string {
  const icons = {
    thought: '💭',
    action: '⚡',
    observation: '👁',
  };
  return icons[phase] || '•';
}

function getStepTitle(phase: string): string {
  const titles = {
    thought: '思考',
    action: '行动',
    observation: '观察',
  };
  return titles[phase] || phase;
}

function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString();
}

async function sendMessage() {
  if (!userInput.value.trim()) return;

  const userMessage = userInput.value;
  userInput.value = '';

  // 添加用户消息
  messages.value.push({
    user: userMessage,
    reply: '',
    reactSteps: [],
  });

  try {
    const response = await chatApi.chat({
      message: userMessage,
      stream: false,
    });

    // 更新最后一条消息
    const lastMessage = messages.value[messages.value.length - 1];
    lastMessage.reply = response.reply;
    lastMessage.reactSteps = response.react_steps;
  } catch (error) {
    console.error('发送消息失败:', error);
  }
}
</script>

<style scoped>
.react-chat {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.message {
  margin-bottom: 20px;
}

.user-message {
  background: #e3f2fd;
  padding: 10px;
  border-radius: 8px;
  margin-bottom: 10px;
}

.assistant-message {
  background: #f5f5f5;
  padding: 15px;
  border-radius: 8px;
}

.react-steps {
  margin-bottom: 15px;
}

.react-step {
  margin-bottom: 10px;
  padding: 10px;
  border-left: 3px solid #ccc;
  background: white;
}

.step-thought {
  border-left-color: #2196f3;
}

.step-action {
  border-left-color: #ff9800;
}

.step-observation {
  border-left-color: #4caf50;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-weight: bold;
}

.step-icon {
  font-size: 1.2em;
}

.step-time {
  margin-left: auto;
  font-size: 0.8em;
  color: #999;
  font-weight: normal;
}

.step-content {
  color: #333;
  line-height: 1.6;
}

.tool-calls {
  margin-top: 8px;
}

.tool-call {
  display: inline-block;
  margin-right: 8px;
  padding: 4px 8px;
  background: #fff3cd;
  border-radius: 4px;
}

.tool-call code {
  color: #856404;
}

.final-reply {
  margin-top: 15px;
  padding-top: 15px;
  border-top: 2px solid #ddd;
  line-height: 1.8;
}

.input-area {
  display: flex;
  gap: 10px;
  padding: 20px;
  border-top: 1px solid #ddd;
}
</style>
```

## 测试 ReAct Agent

### 单元测试

```go
package agent_test

import (
    "context"
    "testing"
    "time"

    "ai-ops/internal/agent"
    "ai-ops/internal/llm"
    "ai-ops/internal/ssh"
    "ai-ops/internal/tool"
)

// Mock LLM Client for testing
type MockLLMClient struct {
    responses []llm.ChatResponse
    callCount int
}

func (m *MockLLMClient) ChatWithTools(ctx context.Context, messages []llm.Message, tools []llm.ToolDef) (*llm.ChatResponse, error) {
    resp := m.responses[m.callCount%len(m.responses)]
    m.callCount++
    return &resp, nil
}

func (m *MockLLMClient) ChatStreamWithTools(ctx context.Context, messages []llm.Message, tools []llm.ToolDef, callback func(chunk llm.StreamChunk)) error {
    // 实现流式响应...
    return nil
}

func TestReActAgent(t *testing.T) {
    // 创建工具注册表
    registry := tool.NewRegistry()

    // 注册测试工具
    registry.RegisterBuiltin(&tool.ListHostsTool{})

    // 创建 Mock LLM Client
    mockLLM := &MockLLMClient{
        responses: []llm.ChatResponse{
            {
                Message: llm.Message{
                    Role: "assistant",
                    Content: "我需要先获取主机列表",
                    ToolCalls: []llm.ToolCall{
                        {
                            ID: "call_1",
                            Function: llm.FunctionCall{
                                Name: "list_hosts",
                                Arguments: "{}",
                            },
                        },
                    },
                },
            },
            {
                Message: llm.Message{
                    Role: "assistant",
                    Content: "已找到 3 台主机，现在检查它们的 CPU 使用率",
                },
            },
        },
    }

    // 创建 ReAct Agent
    reactAgent := agent.NewReActAgent(registry, mockLLM, nil)

    // 测试 ChatReAct
    req := agent.ChatRequest{
        SessionID: "test-session",
        Message:   "检查所有主机的状态",
        Hosts:     []string{},
    }

    resp, err := reactAgent.ChatReAct(context.Background(), req)
    if err != nil {
        t.Fatalf("ChatReAct failed: %v", err)
    }

    // 验证响应
    if resp.SessionID != req.SessionID {
        t.Errorf("SessionID mismatch: got %s, want %s", resp.SessionID, req.SessionID)
    }

    if len(resp.ReActSteps) == 0 {
        t.Error("Expected ReActSteps, got none")
    }

    // 验证 ReAct 步骤
    hasThought := false
    hasAction := false
    hasObservation := false

    for _, step := range resp.ReActSteps {
        switch step.Phase {
        case "thought":
            hasThought = true
        case "action":
            hasAction = true
        case "observation":
            hasObservation = true
        }
    }

    if !hasThought {
        t.Error("Expected at least one thought step")
    }
    if !hasAction {
        t.Error("Expected at least one action step")
    }
}

func TestReActAgentMaxLoops(t *testing.T) {
    registry := tool.NewRegistry()
    mockLLM := &MockLLMClient{
        responses: []llm.ChatResponse{
            {
                Message: llm.Message{
                    Role: "assistant",
                    Content: "继续思考",
                    ToolCalls: []llm.ToolCall{
                        {
                            ID: "call_1",
                            Function: llm.FunctionCall{
                                Name: "list_hosts",
                                Arguments: "{}",
                            },
                        },
                    },
                },
            },
        },
    }

    reactAgent := agent.NewReActAgent(registry, mockLLM, nil)
    reactAgent.SetMaxLoops(3) // 限制为 3 次循环

    req := agent.ChatRequest{
        SessionID: "test-session",
        Message:   "测试循环限制",
        Hosts:     []string{},
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    resp, err := reactAgent.ChatReAct(ctx, req)
    if err != nil {
        t.Fatalf("ChatReAct failed: %v", err)
    }

    // 验证不会超过最大循环次数
    if len(resp.ReActSteps) > 9 { // 3 loops * 3 steps max
        t.Errorf("Too many steps: got %d, expected at most 9", len(resp.ReActSteps))
    }
}
```

## 性能对比测试

```go
func BenchmarkReActAgent(b *testing.B) {
    registry := setupTestRegistry()
    mockLLM := setupMockLLM()
    reactAgent := agent.NewReActAgent(registry, mockLLM, nil)

    req := agent.ChatRequest{
        Message: "测试查询",
        Hosts:   []string{"host1", "host2"},
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := reactAgent.ChatReAct(context.Background(), req)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkNormalAgent(b *testing.B) {
    // 普通Agent的性能测试...
}
```

## 调试技巧

### 1. 查看详细日志

```go
import "go.uber.org/zap"

// 在初始化时设置日志级别
logger, _ := zap.NewDevelopment()
zap.ReplaceGlobals(logger)
```

### 2. 打印 ReAct 步骤

```go
resp, err := reactAgent.ChatReAct(ctx, req)
if err != nil {
    log.Fatal(err)
}

// 打印所有 ReAct 步骤
for i, step := range resp.ReActSteps {
    fmt.Printf("\n=== Step %d: %s ===\n", i+1, step.Phase)
    fmt.Printf("Content: %s\n", step.Content)
    if len(step.ToolCalls) > 0 {
        fmt.Printf("Tool Calls: %+v\n", step.ToolCalls)
    }
    fmt.Printf("Time: %s\n", time.Unix(step.Timestamp, 0).Format(time.RFC3339))
}
```

### 3. 分析 Token 消耗

```go
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

func (a *ReActAgent) ChatReActWithMetrics(ctx context.Context, req ChatRequest) (*ChatResponse, *TokenUsage, error) {
    var totalUsage TokenUsage

    // 在循环中累积 token 使用
    for i := 0; i < a.maxLoops; i++ {
        resp, err := a.llmClient.ChatWithTools(ctx, messages, tools)
        if err != nil {
            return nil, nil, err
        }

        // 累积 token 使用
        totalUsage.PromptTokens += resp.Usage.PromptTokens
        totalUsage.CompletionTokens += resp.Usage.CompletionTokens
        totalUsage.TotalTokens += resp.Usage.TotalTokens

        // ... 其余逻辑
    }

    return response, &totalUsage, nil
}
```

## 常见问题解决

### Q: ReAct Agent 不调用工具

**A**: 检查：
1. 系统提示词是否正确
2. 工具描述是否清晰
3. LLM 是否支持 function calling

### Q: 循环次数过多

**A**: 调整配置：
```go
reactAgent.SetMaxLoops(5)  // 减少最大循环次数
```

### Q: 观察结果不准确

**A**: 改进工具返回：
```go
// 确保工具返回清晰的结果
result := &tool.Result{
    Success: true,
    Message: "检查完成",
    Data: map[string]interface{}{
        "summary": "3 台主机正常",
        "details": hostsData,
    },
}
```

## 总结

ReAct Agent 提供了强大的推理能力，特别适合复杂的多步骤任务。通过合理配置和使用，可以显著提升 AI-Ops 系统的智能化水平。
