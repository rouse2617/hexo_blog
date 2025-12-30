# AI-Ops 智能交互组件集成指南

## 概述

本指南说明如何将新开发的智能交互组件集成到现有的 AI-Ops 聊天系统中。

---

## 新增组件列表

| 组件 | 功能 | 文件路径 |
|------|------|----------|
| SmartSuggestions | 智能推荐问题 | `/web/src/components/chat/SmartSuggestions.vue` |
| EnhancedToolCallCard | 增强工具调用卡片 | `/web/src/components/chat/EnhancedToolCallCard.vue` |
| DataVisualization | 数据可视化 | `/web/src/components/chat/DataVisualization.vue` |
| MessageActions | 消息快捷操作 | `/web/src/components/chat/MessageActions.vue` |
| FollowUpQuestions | 追问建议 | `/web/src/components/chat/FollowUpQuestions.vue` |
| QuickCommands | 快速命令模板 | `/web/src/components/chat/QuickCommands.vue` |
| JsonViewer | JSON 数据查看器 | `/web/src/components/chat/JsonViewer.vue` |
| TreeNode | 树形节点组件 | `/web/src/components/chat/TreeNode.vue` |
| HistoryPanelOverlay | 移动端历史面板侧滑 | `/web/src/components/chat/HistoryPanelOverlay.vue` |

---

## 集成步骤

### 1. 修改 MessageItem.vue

在现有的 `MessageItem.vue` 中添加快捷操作和追问建议：

```vue
<template>
  <div class="message-item" :class="messageClass">
    <!-- 现有的头像、内容等 -->

    <!-- 新增：快捷操作按钮 -->
    <div class="message-actions-wrapper">
      <MessageActions
        :message="message"
        :is-loading="isLoading"
        @copy="handleCopy"
        @regenerate="handleRegenerate"
        @continue="handleContinue"
        @feedback="handleFeedback"
        @export="handleExport"
      />
    </div>

    <!-- 新增：追问建议（仅助手消息） -->
    <FollowUpQuestions
      v-if="message.role === 'assistant' && showFollowUp"
      :last-message="message.content"
      :last-tool="lastToolName"
      @select="handleFollowUpSelect"
    />
  </div>
</template>

<script setup lang="ts">
import MessageActions from './MessageActions.vue'
import FollowUpQuestions from './FollowUpQuestions.vue'

const showFollowUp = computed(() => {
  return message.role === 'assistant' && !isLoading
})

const lastToolName = computed(() => {
  return message.toolCalls?.[0]?.name || ''
})
</script>
```

### 2. 修改 MessageList.vue

在消息列表底部添加智能建议和快速命令：

```vue
<template>
  <div class="message-list">
    <!-- 现有的消息列表 -->

    <!-- 新增：智能建议（首次访问或空闲时显示） -->
    <SmartSuggestions
      v-if="messages.length === 0 || showSuggestions"
      :current-context="contextData"
      @select="handleSuggestionSelect"
    />

    <!-- 新增：快速命令（可折叠） -->
    <QuickCommands
      v-if="showQuickCommands"
      @select="handleCommandSelect"
    />
  </div>
</template>

<script setup lang="ts">
import SmartSuggestions from './SmartSuggestions.vue'
import QuickCommands from './QuickCommands.vue'

const contextData = computed(() => ({
  lastTool: lastToolName.value,
  lastCommand: lastCommand.value,
  hasError: hasError.value,
  selectedHosts: chatStore.selectedHostIds
}))
</script>
```

### 3. 修改 ToolCallCard.vue（可选）

如果想要使用增强版工具卡片，可以直接替换为 `EnhancedToolCallCard`：

```vue
<template>
  <!-- 替换原来的 ToolCallCard -->
  <EnhancedToolCallCard
    v-for="toolCall in message.toolCalls"
    :key="toolCall.id"
    :tool-call="toolCall"
    @rerun="handleToolRerun"
  />
</template>

<script setup lang="ts">
import EnhancedToolCallCard from './EnhancedToolCallCard.vue'
</script>
```

### 4. 在 ChatWindow.vue 中集成数据可视化

当工具返回结构化数据时，自动展示可视化图表：

```vue
<template>
  <div class="chat-window">
    <!-- 现有内容 -->

    <!-- 数据可视化面板（条件显示） -->
    <DataVisualization
      v-if="shouldShowVisualization"
      :data="visualizationData"
    />
  </div>
</template>

<script setup lang="ts">
import DataVisualization from './DataVisualization.vue'

const shouldShowVisualization = computed(() => {
  // 检测最后一条消息是否包含可可视化的数据
  const lastMessage = chatStore.messages[chatStore.messages.length - 1]
  return lastMessage?.toolCalls?.some(tc =>
    tc.name.includes('monitor') ||
    tc.name.includes('status')
  )
})

const visualizationData = computed(() => {
  // 从工具调用结果中解析数据
  // 这里需要根据实际 API 返回格式进行解析
  return parseMonitoringData(chatStore.messages)
})
</script>
```

---

## 事件处理示例

### 在 ChatWindow.vue 中添加事件处理

```vue
<script setup lang="ts">
// 处理建议问题选择
const handleSuggestionSelect = (prompt: string) => {
  inputBoxRef.value?.setInput(prompt)
}

// 处理快速命令选择
const handleCommandSelect = (prompt: string) => {
  chatStore.sendMessage(prompt)
  inputBoxRef.value?.clearInput()
}

// 处理追问问题选择
const handleFollowUpSelect = (question: string) => {
  chatStore.sendMessage(question)
}

// 处理消息重新生成
const handleRegenerate = async (message: Message) => {
  // 找到这条消息之前的用户消息
  const userMessage = findPreviousUserMessage(message)
  if (userMessage) {
    // 删除包括这条消息在内的后续所有消息
    removeMessagesAfter(message)
    // 重新发送用户消息
    await chatStore.sendMessage(userMessage.content)
  }
}

// 处理消息反馈
const handleFeedback = (message: Message, type: 'good' | 'bad') => {
  // 发送反馈到后端（用于改进模型）
  sendFeedback(message.id, type)
}

// 处理工具重新执行
const handleToolRerun = async (toolCall: ToolCall) => {
  // 重新执行工具调用
  await rerunToolCall(toolCall)
}
</script>
```

---

## 数据可视化集成

### 解析工具结果为可视化数据

```typescript
// utils/parseMonitoringData.ts
export function parseMonitoringData(messages: Message[]) {
  const data = {
    cpu: [] as CpuData[],
    memory: [] as MemoryData[],
    disk: [] as DiskData[],
    processes: [] as ProcessData[],
    logs: [] as LogData[]
  }

  // 从最近的工具调用中提取数据
  const recentToolCalls = messages
    .flatMap(m => m.toolCalls || [])
    .slice(-5)

  for (const toolCall of recentToolCalls) {
    if (toolCall.result) {
      try {
        const result = JSON.parse(toolCall.result)

        // 根据工具类型解析数据
        if (toolCall.name.includes('cpu')) {
          data.cpu = parseCpuData(result)
        } else if (toolCall.name.includes('memory')) {
          data.memory = parseMemoryData(result)
        } else if (toolCall.name.includes('disk')) {
          data.disk = parseDiskData(result)
        } else if (toolCall.name.includes('process')) {
          data.processes = parseProcessData(result)
        } else if (toolCall.name.includes('log')) {
          data.logs = parseLogData(result)
        }
      } catch (e) {
        console.error('Failed to parse tool result:', e)
      }
    }
  }

  return data
}
```

---

## 样式配置

### 全局样式变量

在 `web/src/styles/variables.css` 中添加：

```css
:root {
  /* 智能组件动画时长 */
  --smart-animation-duration: 0.3s;

  /* 渐变色 */
  --gradient-primary: linear-gradient(135deg, #667eea, #764ba2);
  --gradient-success: linear-gradient(135deg, #43e97b, #38f9d7);
  --gradient-warning: linear-gradient(135deg, #fa709a, #fee140);
  --gradient-info: linear-gradient(135deg, #4facfe, #00f2fe);

  /* 卡片阴影 */
  --card-shadow-sm: 0 2px 8px rgba(0, 0, 0, 0.06);
  --card-shadow-md: 0 4px 12px rgba(0, 0, 0, 0.08);
  --card-shadow-lg: 0 8px 20px rgba(0, 0, 0, 0.12);
}
```

---

## 响应式设计

所有组件都支持移动端响应式布局：

```css
/* 在移动端隐藏快速命令面板 */
@media (max-width: 768px) {
  .quick-commands {
    display: none;
  }

  .suggestions-grid {
    grid-template-columns: 1fr;
  }
}
```

---

## 性能优化建议

### 1. 懒加载可视化组件

```vue
<script setup lang="ts">
import { defineAsyncComponent } from 'vue'

const DataVisualization = defineAsyncComponent(() =>
  import('./DataVisualization.vue')
)
</script>
```

### 2. 虚拟滚动长列表

对于大量日志数据，使用虚拟滚动：

```vue
<template>
  <RecycleScroller
    :items="logData"
    :item-size="80"
    key-field="id"
  >
    <template #default="{ item }">
      <LogEntry :log="item" />
    </template>
  </RecycleScroller>
</template>
```

### 3. 防抖和节流

对于用户输入和滚动事件：

```typescript
import { debounce } from 'lodash-es'

const handleScroll = debounce(() => {
  // 处理滚动
}, 200)
```

---

## 测试建议

### 单元测试

```typescript
// SmartSuggestions.spec.ts
import { mount } from '@vue/test-utils'
import SmartSuggestions from './SmartSuggestions.vue'

describe('SmartSuggestions', () => {
  it('根据上下文生成正确的建议', () => {
    const wrapper = mount(SmartSuggestions, {
      props: {
        currentContext: {
          lastTool: 'execute_command',
          hasError: true
        }
      }
    })

    expect(wrapper.findAll('.suggestion-item').length).toBeGreaterThan(0)
  })
})
```

---

## 故障排查

### 常见问题

1. **组件不显示**：检查是否正确导入和注册组件
2. **事件不触发**：确保父组件正确绑定事件处理器
3. **样式错乱**：检查 CSS 作用域和全局样式冲突
4. **性能问题**：使用 Vue DevTools 检查组件渲染次数

---

## 未来扩展

- [ ] 支持自定义命令模板导入/导出
- [ ] 添加更多可视化图表类型（折线图、热力图）
- [ ] 实现拖拽式仪表盘配置
- [ ] 支持多语言切换
- [ ] 添加暗黑模式适配

---

## 📋 组件完整功能列表

### 1. SmartSuggestions.vue - 智能推荐组件

**功能特性**:
- 根据对话上下文智能推荐运维场景
- 支持监控、日志、分析、命令、故障排查等类别
- 响应式网格布局，适配不同屏幕尺寸
- 优雅的渐入动画和悬停效果

**Props 接口**:
```typescript
interface Suggestion {
  title: string
  description: string
  prompt: string
  icon: any
  category: 'monitor' | 'log' | 'analysis' | 'command' | 'troubleshoot'
}

interface Props {
  currentContext?: {
    lastTool?: string
    lastCommand?: string
    hasError?: boolean
    selectedHosts?: string[]
  }
}
```

**Events**:
- `select(prompt: string)` - 用户点击推荐卡片时触发

**使用场景**:
- 新会话空状态时显示
- 执行完命令后推荐后续操作
- 检测到错误时推荐故障排查

---

### 2. EnhancedToolCallCard.vue - 增强工具调用卡片

**功能特性**:
- 实时进度条显示执行状态
- 可折叠的参数和结果展示
- JSON 格式化高亮显示
- 执行时间统计
- 一键复制/导出结果
- 状态颜色标识（成功/失败/运行中）

**Props 接口**:
```typescript
interface ToolCall {
  id: string
  name: string
  arguments: string
  result?: string
  error?: string
  status: 'pending' | 'running' | 'success' | 'error'
}

interface Props {
  toolCall: ToolCall
}
```

**Events**:
- `rerun(toolCall: ToolCall)` - 重新执行工具

**展示内容**:
- 工具名称和状态
- 执行参数（JSON 格式）
- 执行结果（文本/JSON）
- 错误信息（如果失败）
- 执行耗时

---

### 3. DataVisualization.vue - 数据可视化组件

**功能特性**:
- 5 种可视化类型切换
- CPU 趋势折线图
- 内存使用进度条
- 磁盘空间饼图
- Top 进程列表
- 日志时间线

**数据接口**:
```typescript
interface CpuData {
  time: string
  value: number
}

interface MemoryData {
  time: string
  value: number
}

interface DiskData {
  mount: string
  used: number
  total: number
  usedPercent: number
}

interface ProcessData {
  pid: number
  name: string
  cpu: number
  memory: number
}

interface LogData {
  level: string
  time: string
  message: string
}

interface Props {
  data?: {
    cpu?: CpuData[]
    memory?: MemoryData[]
    disk?: DiskData[]
    processes?: ProcessData[]
    logs?: LogData[]
  }
}
```

**可视化效果**:
- Sparkline 迷你图
- SVG 环形进度条
- 颜色梯度标识
- 统计数据汇总

---

### 4. MessageActions.vue - 消息操作组件

**功能特性**:
- 下拉菜单 + 快捷按钮双模式
- 复制消息内容
- 重新生成回答
- 继续追问
- 好坏反馈
- 导出为 Markdown
- 分享链接

**Props 接口**:
```typescript
interface Props {
  message: Message
  isLoading?: boolean
}
```

**Events**:
```typescript
interface Emits {
  copy: [message: Message]
  regenerate: [message: Message]
  continue: [message: Message]
  feedback: [message: Message, type: 'good' | 'bad']
  export: [message: Message]
  share: [message: Message]
}
```

**交互方式**:
- 悬停显示快捷按钮
- 点击更多打开下拉菜单
- 反馈状态记录

---

### 5. FollowUpQuestions.vue - 追问建议组件

**功能特性**:
- 根据对话内容智能生成追问
- 支持多种场景模板
- 精美的渐变色图标
- 响应式网格布局

**Props 接口**:
```typescript
interface Props {
  lastMessage?: string
  lastTool?: string
  context?: string[]
}
```

**Events**:
- `select(question: string)` - 选择追问问题

**场景模板**:
- 监控相关 - 查看历史趋势、设置告警、性能优化
- 日志相关 - 错误统计、关联分析、日志导出
- 命令相关 - 解释命令、相关推荐、添加到脚本库
- 故障排查 - 深入诊断、查看历史、生成修复方案

---

### 6. QuickCommands.vue - 快速命令组件

**功能特性**:
- Tab 分类切换（监控、日志、操作、故障排查）
- 命令预览
- 快捷键提示
- 自定义命令支持

**分类内容**:
- **监控**: 系统概览、CPU 使用率、内存分析、磁盘检查
- **日志**: 实时日志、错误日志、服务日志、日志分析
- **操作**: 重启服务、进程管理、端口检查、网络连接
- **故障排查**: 性能诊断、磁盘清理、安全检查、备份检查

**Events**:
- `select(prompt: string)` - 选择快速命令

---

### 7. JsonViewer.vue - JSON 查看器

**功能特性**:
- 树形展示 JSON 数据
- 可折叠/展开节点
- 语法高亮
- 类型标识

**Props 接口**:
```typescript
interface Props {
  data: any
}
```

**展示特性**:
- 对象用 `{}` 标识
- 数组用 `[]` 标识
- 字符串绿色高亮
- 数字橙色高亮
- 布尔值蓝色高亮
- null 红色高亮

---

### 8. TreeNode.vue - 树形节点组件

**功能特性**:
- 递归渲染子节点
- 点击展开/折叠
- 悬停高亮
- 类型颜色标识

**Props 接口**:
```typescript
interface Props {
  nodeKey: string
  value: any
  isRoot?: boolean
}
```

**展示内容**:
- 节点键名
- 节点值（非对象/数组）
- 预览信息（折叠状态）
- 类型徽章（对象/数组）

---

### 9. MessageItemEnhanced.vue - 增强消息组件

**功能特性**:
- Markdown 渲染（支持代码高亮）
- 集成 EnhancedToolCallCard
- 集成 DataVisualization
- 集成 FollowUpQuestions
- 集成 MessageActions
- 头像和元信息展示

**Props 接口**:
```typescript
interface Props {
  message: Message
  isLoading?: boolean
  showDivider?: boolean
  dividerText?: string
}
```

**Events**:
```typescript
interface Emits {
  copy: [message: Message]
  regenerate: [message: Message]
  continue: [message: Message, question: string]
  feedback: [message: Message, type: 'good' | 'bad']
  export: [message: Message]
  share: [message: Message]
  toolRerun: [toolCall: any]
  followUp: [question: string]
}
```

**展示内容**:
- 用户/AI 头像
- 角色和时间戳
- 消息内容（Markdown/纯文本）
- 工具调用卡片
- 数据可视化
- 追问建议
- 分隔线

---

## 🔄 Store 更新说明

### chat.ts 新增状态

```typescript
// 智能推荐接口
export interface Suggestion {
  title: string
  description: string
  prompt: string
  icon: any
  category: 'monitor' | 'log' | 'analysis' | 'command' | 'troubleshoot'
}

// 新增状态
suggestions: ref<Suggestion[]>([])        // 智能推荐列表
showSuggestions: ref<boolean>(true)       // 是否显示推荐
```

### chat.ts 新增方法

```typescript
// 获取智能推荐（基于上下文）
async function getSuggestions(context: {
  lastTool?: string
  hasError?: boolean
  selectedHosts?: string[]
}): Promise<Suggestion[]>

// 选择智能推荐
function selectSuggestion(suggestion: Suggestion): void

// 切换推荐显示状态
function toggleSuggestions(show?: boolean): void
```

---

## 📦 完整集成示例

### MessageList.vue 集成所有新组件

```vue
<template>
  <div class="message-list-container">
    <!-- 空状态 - 显示智能推荐和快速命令 -->
    <div v-if="messages.length === 0" class="empty-state-with-suggestions">
      <div class="empty-state">
        <el-icon :size="64" color="#c0c4cc"><ChatDotRound /></el-icon>
        <p class="empty-title">开始新的对话</p>
        <p class="empty-desc">输入您的问题，AI 助手将为您提供帮助</p>
      </div>

      <!-- 智能推荐 -->
      <SmartSuggestions
        v-if="chatStore.showSuggestions"
        :current-context="getContextFromMessages()"
        @select="handleSuggestionSelect"
      />

      <!-- 快速命令 -->
      <QuickCommands @select="handleCommandSelect" />
    </div>

    <!-- 消息列表 -->
    <template v-else>
      <div class="message-scroller">
        <MessageItemEnhanced
          v-for="(message, index) in visibleMessages"
          :key="getMessageKey(message, index)"
          :message="message"
          :is-loading="isLoading && index === visibleMessages.length - 1"
          @copy="handleCopy"
          @regenerate="handleRegenerate"
          @continue="handleContinue"
          @feedback="handleFeedback"
          @export="handleExport"
          @share="handleShare"
          @toolRerun="handleToolRerun"
          @followUp="handleFollowUp"
        />
      </div>
    </template>

    <!-- 思考过程展示 -->
    <ThinkingProcess
      :thinking-steps="thinkingSteps"
      :current-status="currentThinkingStatus"
      :is-loading="isLoading"
    />
  </div>
</template>

<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import MessageItemEnhanced from './MessageItemEnhanced.vue'
import SmartSuggestions from './SmartSuggestions.vue'
import QuickCommands from './QuickCommands.vue'

const chatStore = useChatStore()

// 事件处理函数
const handleSuggestionSelect = (prompt: string) => {
  emit('sendMessage', prompt)
}

const handleCommandSelect = (prompt: string) => {
  emit('sendMessage', prompt)
}

const handleCopy = (message: Message) => {
  navigator.clipboard.writeText(message.content)
  ElMessage.success('已复制到剪贴板')
}

const handleRegenerate = (message: Message) => {
  emit('regenerate', message)
}

const handleFollowUp = (question: string) => {
  emit('sendMessage', question)
}
// ... 其他事件处理函数
</script>
```

---

## 🎨 设计规范

### 颜色使用

| 用途 | 颜色 | Hex |
|------|------|-----|
| 主要操作 | 蓝色 | #409eff |
| 成功状态 | 绿色 | #67c23a |
| 警告状态 | 橙色 | #e6a23c |
| 错误状态 | 红色 | #f56c6c |
| 文本主色 | 深灰 | #303133 |
| 文本次色 | 浅灰 | #909399 |

### 渐变色方案

```css
/* 主色渐变 */
--gradient-primary: linear-gradient(135deg, #667eea, #764ba2);
--gradient-blue: linear-gradient(135deg, #4facfe, #00f2fe);
--gradient-green: linear-gradient(135deg, #43e97b, #38f9d7);
--gradient-orange: linear-gradient(135deg, #fa709a, #fee140);
--gradient-purple: linear-gradient(135deg, #a8edea, #fed6e3);
```

### 动画时长

```css
--transition-fast: 0.15s;
--transition-normal: 0.3s;
--transition-slow: 0.5s;
```

---

## ✅ 实现检查清单

- [x] SmartSuggestions.vue - 智能推荐组件
- [x] EnhancedToolCallCard.vue - 增强工具调用卡片
- [x] DataVisualization.vue - 数据可视化组件
- [x] MessageActions.vue - 消息操作组件
- [x] FollowUpQuestions.vue - 追问建议组件
- [x] QuickCommands.vue - 快速命令组件
- [x] JsonViewer.vue - JSON 查看器
- [x] TreeNode.vue - 树形节点组件
- [x] MessageItemEnhanced.vue - 增强消息组件
- [x] chat.ts - Store 状态和方法更新
- [x] MessageList.vue - 集成所有新组件
- [x] 组件使用指南文档
- [x] HistoryPanelOverlay.vue - 移动端历史面板侧滑组件

---

### 10. HistoryPanelOverlay.vue - 移动端历史面板侧滑组件

**功能特性**:
- 侧滑抽屉效果 (Side drawer effect)
- 点击遮罩层关闭 (Click mask to close)
- 触摸滑动关闭 (Touch swipe to close)
- 平滑的动画过渡
- 支持手势拖拽反馈
- 完整的 TypeScript 类型定义
- 响应式设计，适配移动端和平板

**Props 接口**:
```typescript
interface Props {
  /** Whether the overlay is visible */
  visible: boolean
  /** List of chat sessions */
  sessions: Array<{
    id: string
    title: string
    createdAt: string | number
  }>
  /** Current active session ID */
  currentSessionId: string
}
```

**Events**:
```typescript
interface Emits {
  /** Emitted when the overlay is closed */
  (e: 'close'): void
  /** Emitted when a session is selected */
  (e: 'sessionSelect', sessionId: string): void
  /** Emitted when a session is deleted */
  (e: 'sessionDelete', sessionId: string): void
}
```

**使用场景**:
- 移动端 (single-column 布局) 显示历史会话列表
- 通过菜单按钮打开侧滑面板
- 支持手势滑动关闭，提升移动端体验

**手势支持**:
- **触摸开始**: 记录初始触摸位置
- **触摸移动**: 检测水平滑动，应用视觉反馈
- **触摸结束**: 判断滑动距离，超过阈值则关闭面板
- **滑动阈值**: 50px (可配置)

**动画效果**:
- 遮罩层淡入淡出 (overlay-fade)
- 面板从左侧滑入 (slide-left)
- 使用 cubic-bezier 缓动函数，更自然的动画

**响应式设计**:
- 移动端: 宽度 280px，最大 85vw
- 平板及以上: 宽度 320px
- 支持自定义滚动条样式

---

### 使用示例

在 ChatWindow.vue 中集成 HistoryPanelOverlay：

```vue
<template>
  <div class="chat-window-container">
    <!-- 现有的聊天面板 -->

    <!-- 移动端历史面板侧滑 -->
    <HistoryPanelOverlay
      :visible="uiStore.historyPanelOverlayVisible"
      :sessions="chatStore.sessions"
      :current-session-id="chatStore.currentSessionId"
      @close="handleCloseHistoryOverlay"
      @session-select="handleSessionSelect"
      @session-delete="handleDeleteSession"
    />
  </div>
</template>

<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import HistoryPanelOverlay from './HistoryPanelOverlay.vue'

const chatStore = useChatStore()
const uiStore = useUIStore()

// 关闭历史面板
const handleCloseHistoryOverlay = () => {
  uiStore.closeHistoryOverlay()
}

// 选择会话
const handleSessionSelect = async (sessionId: string) => {
  if (sessionId === '__new__') {
    await chatStore.newSession()
  } else {
    await chatStore.switchSession(sessionId)
  }
  uiStore.closeHistoryOverlay()
}

// 删除会话
const handleDeleteSession = async (sessionId: string) => {
  await chatStore.removeSession(sessionId)
}
</script>
```

**状态管理 (UI Store)**:

```typescript
// 在 ui.ts 中已有以下状态和方法
interface UIState {
  historyPanelOverlayVisible: boolean
}

function toggleHistoryOverlay(): void
function closeHistoryOverlay(): void
```

**样式变量**:

```css
:root {
  /* Z-index 层级 */
  --z-modal-backdrop: 2000;
  --z-modal: 2001;

  /* 侧边栏样式 */
  --sidebar-bg-start: #1e293b;
  --sidebar-bg-end: #0f172a;
  --sidebar-border: rgba(255, 255, 255, 0.1);
  --sidebar-text: #cbd5e1;
  --sidebar-text-hover: #f1f5f9;
  --sidebar-active-bg: rgba(59, 130, 246, 0.2);
  --sidebar-active-text: #60a5fa;

  /* 过渡动画 */
  --transition-fast: 0.15s;
  --transition-base: 0.3s;
}
```

**无障碍支持**:
- 支持减少动画偏好设置 (prefers-reduced-motion)
- 适当的 ARIA 属性
- 触摸反馈优化
- 防止误触 (垂直滚动时不触发水平滑动)

**性能优化**:
- 使用 CSS transform 而非 position (GPU 加速)
- 事件节流和防抖
- 条件渲染 (v-if) 减少不必要的 DOM
- 自定义滚动条优化

---

## 🔗 相关文件

- **组件目录**: `C:\Users\hrp\Downloads\ai-pro\web\src\components\chat\`
- **Store**: `C:\Users\hrp\Downloads\ai-pro\web\src\stores\chat.ts`
- **文档**: `C:\Users\hrp\Downloads\ai-pro\web\src\components\chat\ComponentUsageGuide.md`

---

**文档版本**: v2.0.0
**最后更新**: 2025-12-30
**维护者**: AI-Ops 开发团队
