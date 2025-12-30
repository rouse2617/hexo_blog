# 设计文档分析报告
# Design Document Analysis Report

**日期 / Date**: 2025-12-31
**分析者 / Analyst**: 前端架构师视角
**文档版本 / Version**: design.md vs requirements.md v2.0

---

## 执行摘要 / Executive Summary

本报告从专业前端架构师的角度,对 `design.md` 进行全面评估,并与优化后的 `requirements.md` 进行一致性分析。总体评价:

- **架构设计**: ⭐⭐⭐⭐ (4/5) - 组件划分合理,但缺少部分关键模块
- **接口设计**: ⭐⭐⭐ (3/5) - 基础类型完整,但安全性和扩展性不足
- **数据流设计**: ⭐⭐⭐ (3/5) - Pinia store 基础可用,需增强状态管理
- **正确性属性**: ⭐⭐⭐⭐ (4/5) - 12 条属性覆盖核心需求,但遗漏新增需求

---

## 第一部分: 设计质量评估 / Design Quality Assessment

### 1. 架构设计评估 / Architecture Design

#### ✅ 优点 / Strengths

1. **清晰的层次结构**
   ```
   ChatWindow (根容器)
   ├── HistoryPanel (左栏 - 会话历史)
   ├── ChatPanel (中栏 - 主对话区)
   └── ContextPanel (右栏 - 监控面板)
   ```
   - 符合运维控制台的三栏布局范式
   - 关注点分离清晰

2. **合理的响应式设计**
   - 断点定义科学 (Mobile: <768px, Desktop: 768-1200px, XL: >1200px)
   - 渐进式降级策略合理

3. **组件职责单一**
   - `MessageCard`: 消息展示
   - `ToolCallCard`: 工具调用
   - `EnhancedInputBox`: 输入增强
   - `QuickTerminal`: 快捷终端

#### ⚠️ 问题与建议 / Issues & Recommendations

**问题 1: 缺少关键新增需求的组件**

| 需求 | 缺失组件 | 影响 |
|------|---------|------|
| Req 10: 审计日志 | `AuditLogPanel` / `AuditLogViewer` | 无法记录和追溯危险操作 |
| Req 11: 二次确认 | `ConfirmationDialog` | 缺少安全防护机制 |
| Req 13: 深色模式 | `ThemeProvider` / `ThemeToggle` | 用户体验缺失 |
| Req 14: 快捷键 | `KeyboardShortcutManager` | 效率功能缺失 |
| Req 15: 连接状态 | `ConnectionStatusIndicator` | 网络状态不可见 |
| Req 16: 消息搜索 | `MessageSearchBar` / `SearchResults` | 历史检索困难 |
| Req 17: 小屏访问 | `HistoryPanelOverlay` | 移动端体验差 |

**建议补充的组件结构**:

```typescript
// 1. 审计日志组件
interface AuditLogViewerProps {
  filters: {
    timeRange?: [Date, Date]
    userId?: string
    riskLevel?: 'low' | 'medium' | 'high'
    host?: string
  }
  onExport: (format: 'csv' | 'json') => void
}

// 2. 二次确认对话框
interface ConfirmationDialogProps {
  command: string
  riskLevel: 'low' | 'medium' | 'high'
  affectedHosts: string[]
  requiresTyping: boolean // 高风险需输入 "CONFIRM"
  countdown?: number // 倒计时
  onConfirm: () => void
  onCancel: () => void
}

// 3. 主题提供者
interface ThemeProviderProps {
  defaultTheme: 'light' | 'dark' | 'auto'
  children: ReactNode | VNode
}

interface ThemeContextValue {
  theme: 'light' | 'dark' | 'auto'
  setTheme: (theme: 'light' | 'dark' | 'auto') => void
  resolvedTheme: 'light' | 'dark' // 实际应用的主题
}

// 4. 快捷键管理器
interface ShortcutConfig {
  [key: string]: {
    description: string
    handler: () => void
    disabled?: boolean
  }
}

interface ShortcutManagerProps {
  shortcuts: ShortcutConfig
  helpModal?: boolean
}

// 5. 连接状态指示器
interface ConnectionStatusProps {
  status: 'connected' | 'connecting' | 'disconnected'
  latency?: number
  queueSize?: number
  onClick: () => void
}

// 6. 消息搜索栏
interface MessageSearchBarProps {
  onSearch: (query: SearchQuery) => void
  onNavigate: (direction: 'prev' | 'next') => void
  resultsCount: number
  currentIndex: number
}

interface SearchQuery {
  text?: string
  timeRange?: [Date, Date]
  toolType?: string[]
  hosts?: string[]
  errorLevel?: 'info' | 'warning' | 'error'
}

// 7. 历史面板遮罩 (移动端)
interface HistoryPanelOverlayProps {
  visible: boolean
  sessions: Session[]
  currentSessionId: string
  onClose: () => void
  onSessionSelect: (sessionId: string) => void
}
```

**问题 2: 缺少错误边界组件**

```typescript
// 建议添加
interface ErrorBoundaryProps {
  fallback?: ComponentType<{ error: Error; retry: () => void }>
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  children: ReactNode | VNode
}

// 用途: 捕获组件树中的错误,防止整个应用崩溃
```

**问题 3: `ChatPanel` 组件未在架构图中明确**

当前设计直接跳到 `MessageList` 和 `InputArea`,缺少 `ChatPanel` 作为中间层的抽象。

```typescript
// 建议添加
interface ChatPanelProps {
  sessionId: string
  messages: Message[]
  isLoading: boolean
  thinkingSteps: ThinkingStep[]
  onSendMessage: (content: string) => void
  onClearMessages: () => void
}

// ChatPanel 负责协调 MessageList + InputArea + QuickTerminal
```

---

### 2. 接口设计评估 / Interface Design

#### ✅ 优点

1. **TypeScript 类型定义清晰**
   - 使用 interface 定义核心数据结构
   - 联合类型 (union types) 使用得当 (如 `status: 'pending' | 'running' | 'success' | 'error'`)

2. **组件 Props 定义完整**
   - 每个 Props 接口都有清晰的字段说明
   - 回调函数命名语义化 (如 `onToggleExpand`, `onRerun`)

#### ⚠️ 问题与建议

**问题 1: 类型安全性不足 - 缺少严格模式**

```typescript
// ❌ 当前设计 - 宽松类型
interface ToolCall {
  arguments: string  // 任意字符串
  result?: string    // 可选字段
}

// ✅ 建议改进 - 严格类型
interface ToolCall {
  id: string
  name: ToolName // 使用字面量类型,而非任意 string
  arguments: Record<string, unknown> // 结构化参数
  result?: ToolResult
  error?: ToolError
  status: ToolCallStatus
  metadata?: {
    startTime: number
    endTime?: number
    duration?: number
    retryCount?: number
  }
}

// 使用字面量类型限制工具名称
type ToolName =
  | 'bash'
  | 'read_file'
  | 'write_file'
  | 'search_files'
  | 'analyze_log'
  // ... 其他工具

// 使用字面量类型限制状态
type ToolCallStatus = 'pending' | 'running' | 'success' | 'error' | 'cancelled'

// 定义工具结果类型
type ToolResult =
  | { type: 'text'; content: string }
  | { type: 'json'; data: unknown }
  | { type: 'file'; path: string }

// 定义工具错误类型
interface ToolError {
  code: string
  message: string
  details?: unknown
}
```

**问题 2: 缺少审计日志相关类型**

```typescript
// ✅ 应该添加
interface AuditLogEntry {
  id: string
  timestamp: number
  userId: string
  sessionId: string
  command: string
  targetHosts: string[]
  executionResult: ExecutionResult
  riskLevel: RiskLevel
  metadata?: {
    source: 'chat' | 'quick_terminal' | 'slash_command'
    userAgent?: string
    ipAddress?: string
  }
}

type RiskLevel = 'low' | 'medium' | 'high'

interface ExecutionResult {
  status: 'success' | 'failure' | 'cancelled'
  exitCode?: number
  duration: number
  output?: string
  error?: string
}
```

**问题 3: `Message` 接口缺少关键字段**

```typescript
// ❌ 当前设计
interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  timestamp: number
}

// ✅ 建议补充
interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  timestamp: number

  // 新增字段
  sessionId: string           // 所属会话
  status?: 'sending' | 'sent' | 'failed'  // 发送状态
  error?: ErrorInfo           // 错误信息
  metadata?: {
    regenerated?: boolean     // 是否为重新生成
    edited?: boolean          // 是否被编辑
    searchHighlight?: boolean // 搜索高亮
  }
  relatedFiles?: FileReference[] // 关联的文件 (Req 4)
  diffContent?: DiffContent      // Diff 内容 (Req 7)
}

interface ErrorInfo {
  code: string
  message: string
  retryable: boolean
}
```

**问题 4: `HostResult` 接口设计不完整**

```typescript
// ❌ 当前设计
interface HostResult {
  host: string
  status: 'pending' | 'running' | 'success' | 'error'
  output?: string
  error?: string
}

// ✅ 建议改进
interface HostResult {
  host: string
  status: HostResultStatus
  output?: string
  error?: HostError

  // 新增性能相关字段
  metrics?: {
    startTime: number
    endTime?: number
    duration?: number
    retryCount: number
  }

  // 新增风险信息 (Req 10, 11)
  riskAssessment?: {
    level: RiskLevel
    warnings: string[]
    requiresConfirmation: boolean
  }
}

type HostResultStatus = 'pending' | 'running' | 'success' | 'error' | 'timeout' | 'cancelled'

interface HostError {
  code: string
  message: string
  details?: unknown
  stderr?: string
}
```

**问题 5: `SlashCommand` 接口缺少参数定义**

```typescript
// ❌ 当前设计
interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string
}

// ✅ 建议改进
interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string | ComponentType  // 支持组件图标

  // 新增字段
  category: CommandCategory
  parameters?: CommandParameter[]
  examples: string[]
  riskLevel?: RiskLevel

  // 执行相关
  handler?: (params: Record<string, unknown>) => Promise<void>
  requiresHost?: boolean     // 是否需要选择主机
  allowBatch?: boolean       // 是否支持批量执行
}

type CommandCategory = 'monitor' | 'log' | 'analysis' | 'command' | 'troubleshoot'

interface CommandParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'array'
  description: string
  required: boolean
  defaultValue?: unknown
  validate?: (value: unknown) => boolean | string
}
```

**问题 6: 缺少 UI 主题相关类型**

```typescript
// ✅ 应该添加 (Req 13)
type ThemeMode = 'light' | 'dark' | 'auto'

type ColorScheme = 'light' | 'dark'

interface ThemeConfig {
  mode: ThemeMode
  colors: {
    primary: string
    secondary: string
    background: string
    surface: string
    error: string
    warning: string
    success: string
  }
  typography: {
    fontFamily: string
    fontSize: Record<string, string>
  }
  spacing: Record<string, string>
  borderRadius: Record<string, string>
}
```

**问题 7: 缺少 WebSocket 相关类型**

```typescript
// ✅ 应该添加 (Req 15)
interface WebSocketConfig {
  url: string
  reconnectInterval: number
  maxReconnectAttempts: number
  heartbeatInterval: number
}

interface WebSocketState {
  status: ConnectionStatus
  latency?: number
  queueSize: number
  lastConnected?: number
  lastDisconnected?: number
}

type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error'

interface WebSocketMessage<T = unknown> {
  type: string
  payload: T
  timestamp: number
}
```

---

### 3. 数据流设计评估 / Data Flow Design

#### ✅ 优点

1. **Pinia 作为状态管理选择合理**
   - Vue 3 官方推荐
   - TypeScript 支持良好
   - DevTools 调试方便

2. **状态分层清晰**
   ```typescript
   stores/
   ├── chat.ts      // 会话和消息状态
   └── metrics.ts   // 监控指标状态
   ```

#### ⚠️ 问题与建议

**问题 1: `ChatState` 缺少新增需求的状态字段**

```typescript
// ❌ 当前设计
interface ChatState {
  layout: LayoutType
  silentMode: boolean
  relatedFiles: FileReference[]
  hostMetrics: Map<string, HostMetrics>
  quickTerminalHistory: string[]
}

// ✅ 建议补充
interface ChatState {
  // 原有字段
  layout: LayoutType
  silentMode: boolean
  relatedFiles: FileReference[]
  hostMetrics: Map<string, HostMetrics>
  quickTerminalHistory: string[]

  // 新增字段 (Req 10: 审计日志)
  auditLogs: AuditLogEntry[]
  auditLogFilters: AuditLogFilters

  // 新增字段 (Req 11: 二次确认)
  pendingConfirmations: ConfirmationRequest[]

  // 新增字段 (Req 13: 主题)
  theme: ThemeMode

  // 新增字段 (Req 14: 快捷键)
  shortcuts: ShortcutConfig
  shortcutHelpVisible: boolean

  // 新增字段 (Req 15: 连接状态)
  connectionStatus: ConnectionStatus
  connectionMetrics: ConnectionMetrics

  // 新增字段 (Req 16: 搜索)
  searchState: SearchState
  searchResults: Message[]

  // 新增字段 (Req 17: 移动端)
  historyPanelVisible: boolean
  historyPanelOverlay: boolean
}

// 补充类型定义
interface AuditLogFilters {
  timeRange?: [Date, Date]
  userId?: string
  riskLevel?: RiskLevel
  host?: string
}

interface ConfirmationRequest {
  id: string
  command: string
  riskLevel: RiskLevel
  affectedHosts: string[]
  requiresTyping: boolean
  countdown: number
  timestamp: number
}

interface ConnectionMetrics {
  latency: number
  queueSize: number
  reconnectAttempts: number
  lastConnected: number
}

interface SearchState {
  query: SearchQuery
  active: boolean
  currentIndex: number
  totalCount: number
}
```

**问题 2: `MetricsState` 设计过于简单**

```typescript
// ❌ 当前设计
interface MetricsState {
  hosts: Map<string, HostMetrics>
  refreshInterval: number
  lastRefresh: number
  isRefreshing: boolean
}

// ✅ 建议改进
interface MetricsState {
  hosts: Map<string, HostMetrics>

  // 刷新控制
  refreshInterval: number      // 刷新间隔 (ms)
  lastRefresh: number          // 上次刷新时间戳
  isRefreshing: boolean        // 是否正在刷新
  refreshError?: Error         // 刷新错误信息
  consecutiveFailures: number  // 连续失败次数 (Req 4.8)

  // 订阅管理
  subscriptions: Set<string>   // 已订阅的主机 ID
  websocketConnected: boolean  // WebSocket 连接状态

  // 异常检测 (Req 8)
  abnormalHosts: Set<string>   // 异常主机列表
  alertHistory: MetricAlert[]  // 告警历史
}

interface MetricAlert {
  hostId: string
  metricType: 'cpu' | 'memory' | 'disk'
  value: number
  threshold: number
  timestamp: number
  acknowledged: boolean
}
```

**问题 3: 缺少全局 UI 状态管理**

```typescript
// ✅ 建议添加新的 store
interface UIState {
  // 布局相关
  sidebarCollapsed: boolean
  rightPanelCollapsed: boolean
  leftPanelCollapsed: boolean

  // 模态框相关
  activeModal: ModalType | null
  modalStack: ModalType[]  // 支持多层模态框

  // 通知相关
  notifications: Notification[]
  notificationIdCounter: number

  // 加载状态
  globalLoading: boolean
  loadingMessage?: string
}

type ModalType =
  | 'confirmation'
  | 'shortcut-help'
  | 'connection-info'
  | 'file-preview'
  | 'audit-log'

interface Notification {
  id: number
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
  timestamp: number
}
```

**问题 4: 缺少用户偏好设置 store**

```typescript
// ✅ 建议添加
interface UserPreferencesState {
  // 主题 (Req 13)
  theme: ThemeMode

  // 快捷键 (Req 14)
  customShortcuts: Partial<ShortcutConfig>

  // 输入偏好
  sendOnEnter: boolean       // 是否按 Enter 发送
  autoCollapseOutput: boolean // 是否自动折叠输出
  maxOutputHeight: number    // 最大输出高度 (px)

  // 通知偏好
  enableNotifications: boolean
  soundEnabled: boolean

  // 隐私偏好
  saveHistory: boolean       // 是否保存历史记录
  analyticsEnabled: boolean  // 是否启用分析
}

// stores/preferences.ts
export const usePreferencesStore = defineStore('preferences', () => {
  // ... 状态管理

  // 持久化到 localStorage
  return {
    // ... actions
  }
}, {
  persist: {
    key: 'ai-pro-preferences',
    paths: ['theme', 'customShortcuts', 'sendOnEnter', 'saveHistory']
  }
})
```

**问题 5: 状态更新逻辑缺少错误恢复**

```typescript
// ❌ 当前设计缺少错误恢复机制

// ✅ 建议添加乐观更新和回滚机制
interface OptimisticUpdateState {
  pendingUpdates: Map<string, {
    originalValue: unknown
    newValue: unknown
    timestamp: number
  }>
}

// 示例: 乐观更新工具调用状态
function updateToolCallOptimistically(toolCallId: string, updates: Partial<ToolCall>) {
  const original = currentToolCalls.value.find(t => t.id === toolCallId)
  if (!original) return

  // 保存原始值
  pendingUpdates.set(toolCallId, {
    originalValue: { ...original },
    newValue: { ...original, ...updates },
    timestamp: Date.now()
  })

  // 乐观更新 UI
  Object.assign(original, updates)

  // 设置超时回滚
  setTimeout(() => {
    if (pendingUpdates.has(toolCallId)) {
      // 回滚到原始值
      Object.assign(original, pendingUpdates.get(toolCallId)!.originalValue)
      pendingUpdates.delete(toolCallId)
    }
  }, 5000)
}
```

---

### 4. 正确性属性评估 / Correctness Properties

#### ✅ 优点

1. **12 条属性覆盖了核心需求**
   - Property 1-12 对应 Req 1-9 的关键功能
   - 使用形式化语言描述,易于机器验证

2. **属性表述清晰**
   - 使用 *For any* ... *shall* 模式
   - 明确验证条件

#### ⚠️ 问题与建议

**问题 1: 缺少新增需求的正确性属性**

优化后的 `requirements.md` 新增了 8 个需求 (Req 10-17),但设计文档缺少对应的正确性属性。

**建议补充的属性**:

```text
### Property 13: Audit Log Completeness

*For any* command execution, THE System SHALL log an Audit_Log_Entry containing all required fields: timestamp, user_id, command, target_hosts, execution_result, risk_level.

**Validates: Requirements 10.1, 10.2**

### Property 14: High-Risk Command Confirmation

*For any* command with risk_level = 'high', THE System SHALL display a Confirmation_Dialog AND require typing "CONFIRM" before execution.

**Validates: Requirements 11.1, 11.3, 11.4**

### Property 15: Session Persistence Accuracy

*For any* page reload, THE System SHALL restore the previous session state with 100% message accuracy AND restore input draft content.

**Validates: Requirements 12.1, 12.2, 12.3**

### Property 16: Theme Consistency

*For any* theme change, THE System SHALL apply the new theme to ALL UI components WITHOUT page reload AND persist the preference.

**Validates: Requirements 13.1, 13.5, 13.6**

### Property 17: Keyboard Shortcut Conflict Resolution

*For any* keyboard shortcut, IF there is a conflict with browser defaults, THE System SHALL prevent default behavior AND execute the custom handler.

**Validates: Requirements 14.1, 14.3**

### Property 18: WebSocket Reconnection Behavior

*For any* connection loss, THE System SHALL attempt reconnection with exponential backoff (1s, 2s, 4s, 8s, max 30s) AND notify user when reconnection succeeds.

**Validates: Requirements 15.4, 15.5, 15.6**

### Property 19: Search Result Accuracy

*For any* search query, THE System SHALL return ALL messages matching the search criteria AND highlight matching text.

**Validates: Requirements 16.2, 16.3**

### Property 20: Mobile History Panel Behavior

*For any* screen width < 1200px, THE System SHALL display a hamburger menu button AND show History_Panel_Overlay when clicked.

**Validates: Requirements 17.1, 17.2**
```

**问题 2: Property 10 与 Req 9.3 不一致**

```text
❌ Property 10 (当前设计):
"For any command entered in Quick_Terminal, the system should execute directly
on the selected host without invoking AI processing, and return raw command output."

✅ 应该修正为 (匹配 Req 9):
"For any command entered in Quick_Terminal, the system shall:
1. Display AI risk assessment and command explanation
2. Require user confirmation for medium or high risk commands
3. Execute directly on the selected host without full AI processing
4. Log the execution to the audit system"
```

**问题 3: Property 2 和 Property 11 的高度限制不一致**

```text
❌ Property 2: "more than 3 lines" -> collapsed
❌ Property 11 (implied): Diff view max height undefined

✅ 应该统一 (匹配 Req 2.2 和 Req 6.3):
"For any command output OR code block OR diff view exceeding 300px height,
the UI shall limit display to 300px with overflow scrolling AND show expand button."

**Validates: Requirements 2.2, 6.3 (FIXED: unified to 300px)**
```

**问题 4: 缺少性能相关的正确性属性**

```text
### Property 21: Message Rendering Performance

*For any* message list with up to 1000 messages, THE System SHALL render
the initial viewport within 100ms AND maintain 60fps during scrolling.

**Validates: Performance Requirements**

### Property 22: Metrics Update Latency

*For any* WebSocket metrics update, THE System SHALL update the Context_Panel
within 200ms of receiving the data.

**Validates: Real-time Requirements**
```

---

## 第二部分: 与需求的一致性分析 / Requirements Consistency

### 需求覆盖矩阵 / Requirements Coverage Matrix

| Req | 需求描述 | 设计文档 | 状态 | 缺失内容 |
|-----|---------|---------|------|---------|
| **Req 1** | 三栏布局改造 | ✅ | 完整 | - |
| **Req 2** | 对话卡片运维化 | ✅ | 完整 | - |
| **Req 3** | 输入框增强 | ✅ | 完整 | 缺少参数验证 |
| **Req 4** | 实时资源监控 | ✅ | 完整 | 缺少失败处理逻辑 |
| **Req 5** | AI 思考状态细化 | ✅ | 完整 | - |
| **Req 6** | 消息气泡宽度优化 | ✅ | 完整 | - |
| **Req 7** | 配置文件 Diff 视图 | ✅ | 完整 | - |
| **Req 8** | 监控面板静默模式 | ✅ | 完整 | - |
| **Req 9** | AI 辅助快捷终端 | ⚠️ | **需修正** | Property 10 与需求不一致 |
| **Req 10** | 命令执行审计日志 | ❌ | **完全缺失** | AuditLogViewer, AuditLogEntry 类型 |
| **Req 11** | 敏感操作二次确认 | ❌ | **完全缺失** | ConfirmationDialog, 风险评估逻辑 |
| **Req 12** | 会话持久化与恢复 | ⚠️ | **部分覆盖** | 只有 localStorage,缺少 IndexedDB |
| **Req 13** | 深色模式支持 | ❌ | **完全缺失** | ThemeProvider, 主题类型 |
| **Req 14** | 快捷键系统 | ❌ | **完全缺失** | ShortcutManager, 快捷键配置 |
| **Req 15** | WebSocket 连接状态指示器 | ❌ | **完全缺失** | ConnectionStatusIndicator |
| **Req 16** | 消息搜索与过滤 | ❌ | **完全缺失** | MessageSearchBar, 搜索逻辑 |
| **Req 17** | 小屏幕历史面板访问 | ❌ | **完全缺失** | HistoryPanelOverlay |

**统计**:
- ✅ 完整覆盖: 7/17 (41%)
- ⚠️ 部分覆盖/需修正: 2/17 (12%)
- ❌ 完全缺失: 8/17 (47%)

---

### 关键缺失项详解 / Critical Missing Features

#### 1. Req 9: AI 辅助快捷终端 - 需修正

**当前设计问题**:
```typescript
// ❌ Property 10 描述过于简化
interface QuickTerminalProps {
  hosts: string[]
  onExecute: (command: string, host: string) => Promise<string>  // 直接执行!
}

// ✅ 应该包含风险评估
interface QuickTerminalProps {
  hosts: string[]
  onExecute: (command: string, host: string) => Promise<CommandResult>
  onAssessRisk: (command: string) => Promise<RiskAssessment>
  onConfirmExecution: (request: ExecutionRequest) => Promise<void>
}

interface RiskAssessment {
  level: RiskLevel
  reason: string
  warnings: string[]
  requiresConfirmation: boolean
  affectedResources?: string[]
}

interface ExecutionRequest {
  command: string
  host: string
  riskLevel: RiskLevel
  userConfirmed: boolean
  confirmationTimestamp?: number
}
```

**修正建议**:
- 将 `QuickTerminal` 改为 `AIAssistedTerminal`,强调 AI 辅助
- 添加风险评估流程: `Command -> Risk Assessment -> Confirmation -> Execution`
- 所有命令必须记录到审计日志 (Req 10)

#### 2. Req 10: 命令执行审计日志 - 完全缺失

**影响**:
- 无法追溯危险操作
- 安全合规性不足
- 故障排查困难

**补充设计**:

```typescript
// 组件
interface AuditLogViewerProps {
  logs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  onFilterChange: (filters: AuditLogFilters) => void
  onExport: (format: 'csv' | 'json') => void
  onViewDetails: (logId: string) => void
}

// Store (stores/audit.ts)
interface AuditLogState {
  logs: AuditLogEntry[]
  filteredLogs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  error?: Error

  // 统计信息
  statistics: {
    totalCommands: number
    highRiskCommands: number
    failureRate: number
    mostActiveHosts: Array<{ host: string; count: number }>
  }
}

// API
interface AuditLogAPI {
  // 记录命令执行
  logCommand(entry: Omit<AuditLogEntry, 'id' | 'timestamp'>): Promise<void>

  // 查询日志
  queryLogs(filters: AuditLogFilters): Promise<AuditLogEntry[]>

  // 导出日志
  exportLogs(filters: AuditLogFilters, format: 'csv' | 'json'): Promise<Blob>

  // 获取统计信息
  getStatistics(timeRange: [Date, Date]): Promise<AuditLogStatistics>
}

// 风险评估逻辑 (utils/riskAssessment.ts)
interface RiskAssessmentConfig {
  dangerousPatterns: Array<{
    pattern: RegExp
    riskLevel: RiskLevel
    reason: string
  }>
}

function assessCommandRisk(command: string): RiskAssessment {
  const patterns = [
    { pattern: /\brm\s+-rf\s+\//, risk: 'high', reason: '删除根目录文件' },
    { pattern: /\bdd\s+if=/, risk: 'high', reason: '磁盘覆盖操作' },
    { pattern: /\bmkfs\./, risk: 'high', reason: '格式化文件系统' },
    { pattern: /\bshutdown\b/, risk: 'medium', reason: '关闭系统' },
    { pattern: /\breboot\b/, risk: 'medium', reason: '重启系统' },
  ]

  // ... 匹配逻辑
}
```

#### 3. Req 11: 敏感操作二次确认 - 完全缺失

**影响**:
- 高风险操作无防护
- 可能导致灾难性误操作

**补充设计**:

```typescript
// 组件
interface ConfirmationDialogProps {
  visible: boolean
  request: ConfirmationRequest
  onConfirm: () => void
  onCancel: () => void
}

interface ConfirmationRequest {
  id: string
  command: string
  riskLevel: RiskLevel
  affectedHosts: string[]
  requiresTyping: boolean  // 高风险需输入 "CONFIRM"
  countdown?: number       // 倒计时
  estimatedImpact?: string  // 预估影响
}

// 样式设计 (基于风险级别)
const dialogStyles = {
  low: {
    title: '确认执行',
    icon: 'Info',
    color: '#409EFF'
  },
  medium: {
    title: '警告: 潜在风险',
    icon: 'Warning',
    color: '#E6A23C'
  },
  high: {
    title: '危险操作确认',
    icon: 'WarningFilled',
    color: '#F56C6C',
    countdown: 5
  }
}

// 确认逻辑
function handleHighRiskConfirmation(request: ConfirmationRequest) {
  // 1. 显示红色警告对话框
  // 2. 列出受影响主机
  // 3. 显示完整命令
  // 4. 启用 5 秒倒计时
  // 5. 要求输入 "CONFIRM"
  // 6. 默认焦点在"取消"按钮
}
```

#### 4. Req 12: 会话持久化 - 部分覆盖

**当前设计问题**:
```typescript
// ❌ 只使用 localStorage,容量限制 5-10MB
function saveMessageCache(sessionId: string, messages: Message[]): void {
  localStorage.setItem(key, JSON.stringify(data))
}

// ✅ 应该使用 IndexedDB,容量更大,支持结构化数据
import Dexie from 'dexie'

class ChatDatabase extends Dexie {
  messages!: Table<Message>
  sessions!: Table<Session>
  drafts!: Table<InputDraft>

  constructor() {
    super('AI-Pro-ChatDB')
    this.version(1).stores({
      messages: 'sessionId, timestamp, role',
      sessions: 'id, createdAt',
      drafts: 'sessionId'
    })
  }
}

const db = new ChatDatabase()

// 自动保存逻辑
async function autoSaveMessages(sessionId: string, messages: Message[]) {
  await db.messages.bulkPut(messages)
  console.log(`[Persist] Saved ${messages.length} messages for session ${sessionId}`)
}

// 恢复逻辑
async function restoreSession(sessionId: string): Promise<SessionState> {
  const messages = await db.messages.where('sessionId').equals(sessionId).toArray()
  const draft = await db.drafts.get(sessionId)

  return {
    messages,
    inputDraft: draft?.content,
    lastSaved: Date.now()
  }
}
```

#### 5. Req 13: 深色模式 - 完全缺失

**补充设计**:

```typescript
// 主题上下文 (composables/useTheme.ts)
import { ref, computed } from 'vue'

type ThemeMode = 'light' | 'dark' | 'auto'

export function useTheme() {
  const mode = ref<ThemeMode>('auto')
  const resolvedTheme = ref<'light' | 'dark'>('light')

  // 监听系统主题变化
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

  function updateResolvedTheme() {
    if (mode.value === 'auto') {
      resolvedTheme.value = mediaQuery.matches ? 'dark' : 'light'
    } else {
      resolvedTheme.value = mode.value
    }
  }

  mediaQuery.addEventListener('change', updateResolvedTheme)

  // 应用主题到 DOM
  function applyTheme(theme: 'light' | 'dark') {
    document.documentElement.setAttribute('data-theme', theme)
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }

  // 设置主题
  function setTheme(newMode: ThemeMode) {
    mode.value = newMode
    updateResolvedTheme()
    applyTheme(resolvedTheme.value)

    // 持久化到 localStorage
    localStorage.setItem('theme-preference', newMode)
  }

  return {
    mode,
    resolvedTheme,
    setTheme,
    toggleTheme: () => setTheme(mode.value === 'light' ? 'dark' : 'light')
  }
}

// CSS 变量定义 (styles/theme.css)
:root {
  --color-background: #ffffff;
  --color-surface: #f8fafc;
  --color-text: #1e293b;
  --color-text-secondary: #64748b;
  --color-border: #e2e8f0;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

[data-theme='dark'] {
  --color-background: #1a1a1a;
  --color-surface: #2d2d2d;
  --color-text: #e2e8f0;
  --color-text-secondary: #94a3b8;
  --color-border: #404040;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.5);
}
```

#### 6. Req 14: 快捷键系统 - 完全缺失

**补充设计**:

```typescript
// 快捷键管理器 (composables/useShortcuts.ts)
interface ShortcutConfig {
  [key: string]: {
    description: string
    handler: (event: KeyboardEvent) => void
    disabled?: Ref<boolean>
  }
}

export function useShortcuts() {
  const shortcuts = ref<ShortcutConfig>({})
  const helpVisible = ref(false)

  function register(key: string, config: ShortcutConfig[string]) {
    shortcuts.value[key] = config
  }

  function unregister(key: string) {
    delete shortcuts.value[key]
  }

  function handleKeyDown(event: KeyboardEvent) {
    const key = buildKeyString(event)
    const shortcut = shortcuts.value[key]

    if (shortcut && (!shortcut.disabled || !shortcut.disabled.value)) {
      event.preventDefault()
      shortcut.handler(event)
    }
  }

  function buildKeyString(event: KeyboardEvent): string {
    const parts: string[] = []

    if (event.ctrlKey) parts.push('ctrl')
    if (event.metaKey) parts.push('cmd')
    if (event.shiftKey) parts.push('shift')
    if (event.altKey) parts.push('alt')

    parts.push(event.key.toLowerCase())

    return parts.join('+')
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
  })

  return {
    shortcuts,
    helpVisible,
    register,
    unregister,
    toggleHelp: () => helpVisible.value = !helpVisible.value
  }
}

// 默认快捷键配置 (shortcuts/index.ts)
export const defaultShortcuts: ShortcutConfig = {
  'ctrl+enter': {
    description: '发送消息',
    handler: () => chatStore.sendMessage(inputBoxRef.value?.content)
  },
  'ctrl+k': {
    description: '清空输入框',
    handler: () => inputBoxRef.value?.clear()
  },
  'ctrl+h': {
    description: '切换历史面板',
    handler: () => uiStore.toggleLeftPanel()
  },
  'ctrl+m': {
    description: '切换监控面板',
    handler: () => uiStore.toggleRightPanel()
  },
  'ctrl+shift+c': {
    description: '复制上次命令输出',
    handler: () => copyLastCommandOutput()
  },
  'escape': {
    description: '关闭模态框/下拉菜单',
    handler: () => closeAllModals()
  },
  'ctrl+/': {
    description: '显示快捷键帮助',
    handler: () => shortcutManager.toggleHelp()
  }
}
```

#### 7. Req 15: WebSocket 连接状态 - 完全缺失

**补充设计**:

```typescript
// 连接状态指示器组件
interface ConnectionStatusProps {
  status: ConnectionStatus
  latency?: number
  queueSize?: number
  onClick: () => void
}

// WebSocket 管理 (composables/useWebSocket.ts)
interface WebSocketOptions {
  url: string
  reconnectInterval?: number
  maxReconnectAttempts?: number
  heartbeatInterval?: number
}

export function useWebSocket(options: WebSocketOptions) {
  const status = ref<ConnectionStatus>('disconnected')
  const latency = ref<number>()
  const queueSize = ref(0)
  const reconnectAttempts = ref(0)

  let ws: WebSocket | null = null
  let heartbeatTimer: number | null = null
  let reconnectTimer: number | null = null

  function connect() {
    if (ws?.readyState === WebSocket.OPEN) return

    status.value = 'connecting'
    ws = new WebSocket(options.url)

    ws.onopen = () => {
      status.value = 'connected'
      reconnectAttempts.value = 0
      startHeartbeat()
      showToast('连接成功', 'success')
    }

    ws.onclose = () => {
      status.value = 'disconnected'
      stopHeartbeat()
      scheduleReconnect()
    }

    ws.onerror = () => {
      status.value = 'error'
      showToast('连接错误', 'error')
    }

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)

      // 处理心跳响应
      if (data.type === 'pong') {
        latency.value = Date.now() - data.timestamp
        return
      }

      // 处理业务数据
      handleMessage(data)
    }
  }

  function scheduleReconnect() {
    const delay = Math.min(
      options.reconnectInterval! * Math.pow(2, reconnectAttempts.value),
      30000
    )

    reconnectTimer = setTimeout(() => {
      if (reconnectAttempts.value < options.maxReconnectAttempts!) {
        reconnectAttempts.value++
        console.log(`[WebSocket] Reconnecting... (${reconnectAttempts.value})`)
        connect()
      } else {
        showToast('连接失败,请刷新页面', 'error')
      }
    }, delay)
  }

  function startHeartbeat() {
    heartbeatTimer = setInterval(() => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping', timestamp: Date.now() }))
      }
    }, options.heartbeatInterval)
  }

  function disconnect() {
    ws?.close()
    stopHeartbeat()
    clearTimeout(reconnectTimer!)
  }

  onMounted(connect)
  onUnmounted(disconnect)

  return {
    status,
    latency,
    queueSize,
    reconnectAttempts,
    connect,
    disconnect
  }
}
```

#### 8. Req 16: 消息搜索 - 完全缺失

**补充设计**:

```typescript
// 搜索组件
interface MessageSearchBarProps {
  visible: boolean
  resultsCount: number
  currentIndex: number
  onSearch: (query: SearchQuery) => void
  onNavigate: (direction: 'prev' | 'next') => void
  onClose: () => void
}

// 搜索逻辑 (utils/messageSearch.ts)
interface SearchQuery {
  text?: string
  timeRange?: [Date, Date]
  toolType?: string[]
  hosts?: string[]
  errorLevel?: 'info' | 'warning' | 'error'
  caseSensitive?: boolean
  regex?: boolean
}

interface SearchResult {
  message: Message
  matches: Array<{
    field: 'content' | 'command' | 'filename'
    text: string
    position: [number, number]  // [start, end]
  }>
  score: number  // 相关性评分
}

function searchMessages(messages: Message[], query: SearchQuery): SearchResult[] {
  const results: SearchResult[] = []

  for (const message of messages) {
    const matches = findMatches(message, query)
    if (matches.length > 0) {
      results.push({
        message,
        matches,
        score: calculateScore(message, matches)
      })
    }
  }

  // 按相关性排序
  return results.sort((a, b) => b.score - a.score)
}

function findMatches(message: Message, query: SearchQuery): Array<...> {
  const matches = []

  // 搜索消息内容
  if (query.text) {
    const contentMatches = findTextMatches(message.content, query.text, query)
    matches.push(...contentMatches.map(pos => ({
      field: 'content',
      text: message.content.substring(pos[0], pos[1]),
      position: pos
    })))
  }

  // 搜索工具调用
  if (message.toolCalls) {
    for (const toolCall of message.toolCalls) {
      if (query.toolType && !query.toolType.includes(toolCall.name)) {
        continue
      }

      if (toolCall.result) {
        const resultMatches = findTextMatches(toolCall.result, query.text, query)
        matches.push(...resultMatches.map(pos => ({
          field: 'command',
          text: toolCall.result.substring(pos[0], pos[1]),
          position: pos
        })))
      }
    }
  }

  return matches
}

// 高亮显示 (utils/highlight.ts)
function highlightText(text: string, matches: Array<{ position: [number, number] }>): VNode {
  const parts = []
  let lastIndex = 0

  for (const match of matches) {
    const [start, end] = match.position

    // 添加匹配前的文本
    parts.push(text.substring(lastIndex, start))

    // 添加高亮的匹配文本
    parts.push(h('mark', { class: 'search-highlight' }, text.substring(start, end)))

    lastIndex = end
  }

  // 添加剩余文本
  parts.push(text.substring(lastIndex))

  return h('span', parts)
}
```

#### 9. Req 17: 小屏幕历史面板 - 完全缺失

**补充设计**:

```typescript
// 历史面板遮罩组件
<template>
  <Teleport to="body">
    <Transition name="slide">
      <div v-if="visible" class="history-panel-overlay">
        <!-- 遮罩背景 -->
        <div class="overlay-backdrop" @click="handleClose" />

        <!-- 侧滑面板 -->
        <div class="history-panel-drawer">
          <div class="drawer-header">
            <h2>历史会话</h2>
            <el-button @click="handleClose" :icon="Close" circle />
          </div>

          <div class="drawer-content">
            <SessionList
              :sessions="sessions"
              :current-session-id="currentSessionId"
              @select="handleSessionSelect"
            />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  sessions: Session[]
  currentSessionId: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  sessionSelect: [sessionId: string]
}>()

// 支持滑动关闭
const drawerRef = ref()
let touchStartX = 0
let touchStartY = 0

function handleTouchStart(event: TouchEvent) {
  touchStartX = event.touches[0].clientX
  touchStartY = event.touches[0].clientY
}

function handleTouchMove(event: TouchEvent) {
  const deltaX = event.touches[0].clientX - touchStartX
  const deltaY = event.touches[0].clientY - touchStartY

  // 只响应水平滑动
  if (Math.abs(deltaX) > Math.abs(deltaY) && deltaX > 50) {
    handleClose()
  }
}
</script>

<style scoped>
.history-panel-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
}

.overlay-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
}

.history-panel-drawer {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  max-width: 80vw;
  background: white;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(-100%);
}

/* 响应式断点 */
@media (min-width: 1200px) {
  .history-panel-overlay {
    display: none;  /* 大屏幕不显示遮罩 */
  }
}
</style>
```

---

## 第三部分: 具体建议 / Specific Recommendations

### 1. 需要补充的组件清单 / Components Checklist

| 组件名称 | 对应需求 | 优先级 | 复杂度 | 预计工时 |
|---------|---------|-------|-------|---------|
| `AuditLogViewer` | Req 10 | P0 | ⭐⭐⭐ | 2-3 天 |
| `ConfirmationDialog` | Req 11 | P0 | ⭐⭐ | 1-2 天 |
| `ThemeProvider` | Req 13 | P1 | ⭐⭐ | 1 天 |
| `ShortcutManager` | Req 14 | P1 | ⭐⭐⭐ | 2 天 |
| `ConnectionStatusIndicator` | Req 15 | P1 | ⭐⭐ | 1 天 |
| `MessageSearchBar` | Req 16 | P2 | ⭐⭐⭐⭐ | 3-4 天 |
| `HistoryPanelOverlay` | Req 17 | P1 | ⭐⭐ | 1-2 天 |
| `ErrorBoundary` | 通用 | P0 | ⭐⭐ | 1 天 |
| `ChatPanel` | 架构优化 | P1 | ⭐⭐ | 1 天 |

**总工时估算**: 13-18 天

---

### 2. 需要修正的类型定义 / Type Definitions to Fix

#### 修正 1: `ToolCall` 接口增强

```typescript
// ❌ 当前设计 (design.md:125-133)
interface ToolCall {
  id: string
  name: string
  arguments: string
  result?: string
  status: 'pending' | 'running' | 'success' | 'error'
  hosts?: string[]
  hostResults?: HostResult[]
}

// ✅ 修正后 (增强类型安全和元数据)
interface ToolCall {
  id: string
  name: ToolName  // 使用字面量类型
  arguments: Record<string, unknown>  // 结构化参数
  result?: ToolResult  // 类型化结果
  status: ToolCallStatus
  hosts?: string[]
  hostResults?: EnhancedHostResult[]

  // 新增字段
  riskAssessment?: RiskAssessment
  metadata?: {
    startTime: number
    endTime?: number
    duration?: number
    retryCount: number
    triggeredBy: 'user' | 'ai' | 'auto'
  }
}

type ToolName =
  | 'bash'
  | 'read_file'
  | 'write_file'
  | 'analyze_log'
  | 'check_disk'
  | 'check_memory'
  | string  // 允许扩展

type ToolCallStatus =
  | 'pending'
  | 'running'
  | 'success'
  | 'error'
  | 'timeout'
  | 'cancelled'

type ToolResult =
  | { type: 'text'; content: string }
  | { type: 'json'; data: unknown }
  | { type: 'file'; path: string; preview?: string }
  | { type: 'error'; error: string }

interface EnhancedHostResult extends HostResult {
  metrics: {
    startTime: number
    endTime?: number
    duration?: number
  }
  riskAssessment?: RiskAssessment
}
```

#### 修正 2: `SlashCommand` 接口增强

```typescript
// ❌ 当前设计 (design.md:183-188)
interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string
}

// ✅ 修正后 (增加参数和验证)
interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string | Component  // 支持组件

  // 新增字段
  category: CommandCategory
  parameters: CommandParameter[]
  examples: string[]
  riskLevel: RiskLevel
  requiresHost: boolean
  allowBatch: boolean

  // 执行相关
  handler?: CommandHandler
  validate?: (params: Record<string, unknown>) => ValidationResult
}

type CommandCategory =
  | 'monitor'
  | 'log'
  | 'analysis'
  | 'command'
  | 'troubleshoot'

interface CommandParameter {
  name: string
  type: 'string' | 'number' | 'boolean' | 'array' | 'object'
  description: string
  required: boolean
  defaultValue?: unknown
  validate?: (value: unknown) => boolean | string
  options?: unknown[]  // 枚举选项
}

interface ValidationResult {
  valid: boolean
  errors?: string[]
}

type CommandHandler = (params: Record<string, unknown>) => Promise<void>
```

#### 修正 3: 新增审计日志类型

```typescript
// ✅ 新增类型定义 (Req 10)
interface AuditLogEntry {
  id: string
  timestamp: number
  userId: string
  sessionId: string
  command: string
  targetHosts: string[]
  executionResult: ExecutionResult
  riskLevel: RiskLevel
  metadata: AuditMetadata
}

interface ExecutionResult {
  status: 'success' | 'failure' | 'cancelled'
  exitCode?: number
  duration: number
  output?: string
  error?: string
}

interface AuditMetadata {
  source: 'chat' | 'quick_terminal' | 'slash_command'
  userAgent?: string
  ipAddress?: string
  triggeredBy: 'user' | 'ai' | 'auto'
  relatedToolCall?: string
}

interface AuditLogFilters {
  timeRange?: [Date, Date]
  userId?: string
  riskLevel?: RiskLevel
  host?: string
  commandPattern?: string
  limit?: number
  offset?: number
}
```

#### 修正 4: 新增风险评估类型

```typescript
// ✅ 新增类型定义 (Req 9, 10, 11)
interface RiskAssessment {
  level: RiskLevel
  score: number  // 0-100
  reason: string
  warnings: string[]
  requiresConfirmation: boolean
  affectedResources?: string[]
  estimatedImpact?: string
  mitigation?: string[]
}

type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

interface RiskAssessmentConfig {
  dangerousPatterns: DangerousPattern[]
  customRules?: CustomRule[]
}

interface DangerousPattern {
  pattern: RegExp
  riskLevel: RiskLevel
  reason: string
  category: 'data_loss' | 'system_shutdown' | 'service_disruption' | 'security'
}

interface CustomRule {
  name: string
  matcher: (command: string) => boolean
  assessor: (command: string) => RiskAssessment
}
```

#### 修正 5: 新增 UI 状态类型

```typescript
// ✅ 新增类型定义 (全局 UI 状态)
interface UIState {
  // 布局
  layout: LayoutState

  // 模态框
  modals: ModalStack

  // 通知
  notifications: Notification[]

  // 加载
  loading: LoadingState

  // 主题 (Req 13)
  theme: ThemeState

  // 快捷键 (Req 14)
  shortcuts: ShortcutState

  // 连接 (Req 15)
  connection: ConnectionState
}

interface LayoutState {
  leftPanelVisible: boolean
  rightPanelVisible: boolean
  leftPanelWidth: number
  rightPanelWidth: number
  leftPanelCollapsed: boolean
  rightPanelCollapsed: boolean
}

type ModalType =
  | 'confirmation'
  | 'shortcut-help'
  | 'connection-info'
  | 'file-preview'
  | 'audit-log'

interface ModalStack {
  items: Array<{
    type: ModalType
    props: Record<string, unknown>
    closable: boolean
  }>
  maxDepth: number
}

interface Notification {
  id: number
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
  timestamp: number
  actions?: Array<{
    label: string
    handler: () => void
  }>
}

interface ThemeState {
  mode: ThemeMode
  resolvedTheme: 'light' | 'dark'
  systemPreference: 'light' | 'dark'
}

interface ShortcutState {
  registered: ShortcutConfig
  helpVisible: boolean
  conflicting: string[]  // 与浏览器冲突的快捷键
}

interface ConnectionState {
  status: ConnectionStatus
  latency?: number
  queueSize: number
  reconnectAttempts: number
  lastConnected?: number
  lastDisconnected?: number
}
```

---

### 3. 测试策略建议 / Testing Strategy Recommendations

#### 当前测试设计的不足

设计文档中只提到使用 `fast-check` 进行属性测试,但缺少:
1. 单元测试框架选择
2. 集成测试工具链
3. E2E 测试方案
4. 视觉回归测试
5. 性能测试基准

#### 建议的测试策略

**阶段 1: 单元测试 (Vitest)**

```typescript
// __tests__/components/MessageCard.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageCard from '@/components/chat/MessageCard.vue'

describe('MessageCard', () => {
  it('should render user message correctly', () => {
    const wrapper = mount(MessageCard, {
      props: {
        message: {
          role: 'user',
          content: 'Hello AI',
          timestamp: Date.now()
        }
      }
    })

    expect(wrapper.text()).toContain('Hello AI')
    expect(wrapper.find('.user-message').exists()).toBe(true)
  })

  it('should collapse long command output', () => {
    const longOutput = 'Line 1\n'.repeat(100)

    const wrapper = mount(MessageCard, {
      props: {
        message: {
          role: 'assistant',
          content: '',
          toolCalls: [{
            id: 'tool-1',
            name: 'bash',
            arguments: '',
            result: longOutput,
            status: 'success'
          }],
          timestamp: Date.now()
        }
      }
    })

    expect(wrapper.find('.collapsed-output').exists()).toBe(true)
    expect(wrapper.find('.expand-button').exists()).toBe(true)
  })

  it('should expand on error', () => {
    const wrapper = mount(MessageCard, {
      props: {
        message: {
          role: 'assistant',
          content: '',
          toolCalls: [{
            id: 'tool-1',
            name: 'bash',
            arguments: '',
            result: '',
            error: 'Command failed',
            status: 'error'
          }],
          timestamp: Date.now()
        }
      }
    })

    expect(wrapper.find('.expanded-output').exists()).toBe(true)
    expect(wrapper.find('.error-message').text()).toContain('Command failed')
  })
})
```

**阶段 2: 属性测试 (fast-check)**

```typescript
// __tests__/properties/layout.test.ts
import fc from 'fast-check'
import { calculateLayout } from '@/utils/layout'

describe('Property: Responsive Layout', () => {
  it('Property 1: should return correct layout for any viewport width', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 320, max: 3840 }),
        (width) => {
          const layout = calculateLayout(width)

          if (width > 1200) {
            return layout === 'three-column'
          } else if (width >= 768) {
            return layout === 'two-column'
          } else {
            return layout === 'single-column'
          }
        }
      ),
      { numRuns: 1000 }  // 增加测试次数
    )
  })
})

// __tests__/properties/command-collapse.test.ts
describe('Property: Command Output Collapse', () => {
  it('Property 2: should collapse output with more than 5 lines', () => {
    fc.assert(
      fc.property(
        fc.array(fc.asciiString(), { minLength: 6 }),
        (lines) => {
          const output = lines.join('\n')
          const config = calculateOutputConfig(output)

          return config.collapsed === true &&
                 config.maxHeight === 300 &&
                 config.showExpandButton === true
        }
      )
    )
  })

  it('Property 3: should aggregate batch operations', () => {
    fc.assert(
      fc.property(
        fc.array(fc.string(), { minLength: 2, maxLength: 10 }),
        (hosts) => {
          const toolCall = createBatchToolCall(hosts)

          return toolCall.aggregated === true &&
                 toolCall.hostResults?.length === hosts.length &&
                 toolCall.displayMode === 'progress-bar'
        }
      )
    )
  })
})

// __tests__/properties/slash-commands.test.ts
describe('Property: Slash Command Trigger', () => {
  it('Property 4: should trigger slash menu only at valid positions', () => {
    fc.assert(
      fc.property(
        fc.string(),
        fc.integer({ min: 0, max: 100 }),
        (text, cursorPosition) => {
          const result = shouldTriggerSlashMenu(text, cursorPosition)

          // 在开头或空格后触发
          const atStart = cursorPosition === 0
          const afterSpace = cursorPosition > 0 && text[cursorPosition - 1] === ' '

          // 在文件路径中不触发 (如 "cat /home/file" 不应该触发)
          const inFilePath = cursorPosition > 0 &&
                            text[cursorPosition - 1] === '/' &&
                            /[a-zA-Z]/.test(text[cursorPosition - 2] || '')

          if (inFilePath) {
            return result === false
          }

          return result === (atStart || afterSpace)
        }
      )
    )
  })
})
```

**阶段 3: 集成测试 (Vue Test Utils)**

```typescript
// __tests__/integration/chat-flow.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ChatWindow from '@/components/chat/ChatWindow.vue'
import * as api from '@/api/chat'

// Mock API
vi.mock('@/api/chat', () => ({
  fetchStreamChat: vi.fn(),
  getSessions: vi.fn(),
  createSession: vi.fn()
}))

describe('Integration: Chat Flow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should complete send-receive flow', async () => {
    const mockStream = async function* () {
      yield { type: 'thinking', data: { step: 1, status: 'calling_llm', content: 'Thinking...' } }
      yield { type: 'content', data: { content: 'Hello!' } }
      yield { type: 'done', data: {} }
    }

    vi.mocked(api.fetchStreamChat).mockImplementation(mockStream)

    const wrapper = mount(ChatWindow)

    // 输入消息
    await wrapper.find('input').setValue('Hello AI')
    await wrapper.find('button.send').trigger('click')

    // 等待响应
    await wrapper.vm.$nextTick()

    // 验证消息列表
    expect(wrapper.find('.user-message').text()).toBe('Hello AI')
    expect(wrapper.find('.assistant-message').text()).toBe('Hello!')
  })

  it('should handle tool execution', async () => {
    const mockStream = async function* () {
      yield {
        type: 'tool_call',
        data: { id: 'tool-1', tool: 'bash', params: { command: 'ls -la' } }
      }
      yield {
        type: 'tool_result',
        data: { id: 'tool-1', result: 'file1.txt\nfile2.txt', error: null }
      }
      yield { type: 'done', data: {} }
    }

    vi.mocked(api.fetchStreamChat).mockImplementation(mockStream)

    const wrapper = mount(ChatWindow)
    await wrapper.find('input').setValue('List files')
    await wrapper.find('button.send').trigger('click')

    await wrapper.vm.$nextTick()

    // 验证工具调用卡片
    expect(wrapper.find('.tool-call-card').exists()).toBe(true)
    expect(wrapper.find('.tool-output').text()).toContain('file1.txt')
  })
})
```

**阶段 4: E2E 测试 (Playwright)**

```typescript
// e2e/chat.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Chat UI E2E', () => {
  test('should send message and receive response', async ({ page }) => {
    await page.goto('http://localhost:5173')

    // 等待页面加载
    await expect(page.locator('.chat-window')).toBeVisible()

    // 输入消息
    await page.fill('input[placeholder*="输入"]', 'Hello AI')
    await page.click('button[aria-label="发送"]')

    // 等待响应
    await expect(page.locator('.assistant-message')).toBeVisible()
    await expect(page.locator('.assistant-message')).toContainText('Hello')
  })

  test('should display tool call with collapsed output', async ({ page }) => {
    await page.goto('http://localhost:5173')

    await page.fill('input', 'Execute ls -la')
    await page.click('button[aria-label="发送"]')

    // 等待工具调用卡片出现
    await expect(page.locator('.tool-call-card')).toBeVisible()

    // 验证默认折叠
    const output = page.locator('.tool-output')
    await expect(output).toHaveCSS('max-height', '300px')

    // 点击展开
    await page.click('.expand-button')
    await expect(output).not.toHaveCSS('max-height', '300px')
  })

  test('should use slash commands', async ({ page }) => {
    await page.goto('http://localhost:5173')

    // 触发 slash 菜单
    const input = page.locator('input[placeholder*="输入"]')
    await input.fill('/')

    // 验证菜单出现
    await expect(page.locator('.slash-menu')).toBeVisible()
    await expect(page.locator('.slash-menu-item')).toHaveCount(6)

    // 选择命令
    await page.click('.slash-menu-item:has-text("/top")')

    // 验证自动完成
    await expect(input).toHaveValue(/显示系统负载/)
  })

  test('should confirm high-risk command', async ({ page }) => {
    await page.goto('http://localhost:5173')

    // 输入危险命令
    await page.fill('input', 'rm -rf /tmp/test')
    await page.click('button[aria-label="发送"]')

    // 等待确认对话框
    await expect(page.locator('.confirmation-dialog')).toBeVisible()
    await expect(page.locator('.confirmation-dialog')).toHaveClass(/high-risk/)

    // 验证倒计时
    const countdown = page.locator('.countdown')
    await expect(countdown).toContainText(/5|4|3|2|1/)

    // 等待倒计时结束
    await expect(page.locator('.confirm-button')).toBeDisabled()
    await page.waitForTimeout(5000)
    await expect(page.locator('.confirm-button')).toBeEnabled()

    // 输入 CONFIRM
    await page.fill('.confirm-input', 'CONFIRM')
    await page.click('.confirm-button')

    // 验证执行
    await expect(page.locator('.tool-call-card')).toBeVisible()
  })
})

test.describe('Responsive Layout', () => {
  test('should show three columns on large screens', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.goto('http://localhost:5173')

    await expect(page.locator('.history-panel')).toBeVisible()
    await expect(page.locator('.chat-panel')).toBeVisible()
    await expect(page.locator('.context-panel')).toBeVisible()
  })

  test('should hide left panel on medium screens', async ({ page }) => {
    await page.setViewportSize({ width: 1000, height: 1080 })
    await page.goto('http://localhost:5173')

    await expect(page.locator('.history-panel')).not.toBeVisible()
    await expect(page.locator('.chat-panel')).toBeVisible()
    await expect(page.locator('.context-panel')).toBeVisible()
  })

  test('should show single column on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 })
    await page.goto('http://localhost:5173')

    await expect(page.locator('.history-panel')).not.toBeVisible()
    await expect(page.locator('.chat-panel')).toBeVisible()
    await expect(page.locator('.context-panel')).not.toBeVisible()

    // 验证汉堡菜单
    await expect(page.locator('.hamburger-menu')).toBeVisible()
  })
})

test.describe('Theme Switching', () => {
  test('should switch to dark mode', async ({ page }) => {
    await page.goto('http://localhost:5173')

    // 切换主题
    await page.click('[aria-label="切换主题"]')
    await page.click('text:深色模式')

    // 验证主题应用
    await expect(page.locator('html')).toHaveClass(/dark/)
    await expect(page.locator('.chat-window')).toHaveCSS(
      'background-color',
      'rgb(26, 26, 26)'
    )
  })
})
```

**阶段 5: 性能测试**

```typescript
// __tests__/performance/rendering.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageList from '@/components/chat/MessageList.vue'

describe('Performance: Rendering', () => {
  it('should render 1000 messages within 100ms', () => {
    const messages = Array.from({ length: 1000 }, (_, i) => ({
      role: i % 2 === 0 ? 'user' : 'assistant',
      content: `Message ${i}`,
      timestamp: Date.now() + i
    }))

    const start = performance.now()
    const wrapper = mount(MessageList, {
      props: { messages }
    })
    const end = performance.now()

    expect(end - start).toBeLessThan(100)
  })

  it('should maintain 60fps during scroll', async () => {
    const messages = Array.from({ length: 1000 }, (_, i) => ({
      role: i % 2 === 0 ? 'user' : 'assistant',
      content: `Message ${i}\nLine 2\nLine 3`,
      timestamp: Date.now() + i
    }))

    const wrapper = mount(MessageList, {
      props: { messages }
    })

    const frameTimes: number[] = []

    for (let i = 0; i < 60; i++) {
      const frameStart = performance.now()

      wrapper.vm.$el.scrollTop += 100
      await wrapper.vm.$nextTick()

      const frameEnd = performance.now()
      frameTimes.push(frameEnd - frameStart)
    }

    const avgFrameTime = frameTimes.reduce((a, b) => a + b) / frameTimes.length
    const fps = 1000 / avgFrameTime

    expect(fps).toBeGreaterThan(55)  // 允许小幅波动
  })
})
```

---

### 4. 潜在技术债务与风险 / Technical Debt & Risks

#### 技术债务 1: 缺少错误边界机制

**风险等级**: 🔴 高

**问题描述**:
- 当前设计没有错误边界组件
- 任何组件崩溃可能导致整个应用白屏
- 用户未保存的输入可能丢失

**解决方案**:

```typescript
// components/ErrorBoundary.vue
<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'

const hasError = ref(false)
const error = ref<Error | null>(null)

onErrorCaptured((err, instance, info) => {
  hasError.value = true
  error.value = err

  // 上报错误到监控系统
  reportError(err, { component: instance?.type?.name, info })

  // 返回 false 阻止错误继续传播
  return false
})

function retry() {
  hasError.value = false
  error.value = null
  // 触发重新渲染
}
</script>

<template>
  <slot v-if="!hasError" />
  <div v-else class="error-boundary">
    <h2>出错了</h2>
    <p>{{ error?.message }}</p>
    <button @click="retry">重试</button>
  </div>
</template>

// 使用
<ErrorBoundary>
  <ChatWindow />
</ErrorBoundary>
```

#### 技术债务 2: 缺少乐观更新机制

**风险等级**: 🟡 中

**问题描述**:
- 所有操作都等待后端响应才更新 UI
- 用户感觉延迟明显
- 网络慢时体验差

**解决方案**:

```typescript
// stores/optimistic.ts
export function useOptimisticUpdate<T>() {
  const pendingUpdates = ref<Map<string, T>>(new Map())

  function updateOptimistically(
    id: string,
    updateFn: (original: T) => T,
    commitFn: () => Promise<void>
  ) {
    // 保存原始值
    const original = items.value.find(item => item.id === id)
    if (original) {
      pendingUpdates.value.set(id, original)
    }

    // 乐观更新
    items.value = items.value.map(item =>
      item.id === id ? { ...item, ...updateFn(item) } : item
    )

    // 提交到服务器
    commitFn()
      .then(() => {
        pendingUpdates.value.delete(id)
      })
      .catch(err => {
        // 回滚
        const original = pendingUpdates.value.get(id)
        if (original) {
          items.value = items.value.map(item =>
            item.id === id ? original : item
          )
        }
        showError('操作失败,请重试')
      })
  }

  return { updateOptimistically }
}
```

#### 技术债务 3: 缺少虚拟滚动优化

**风险等级**: 🟡 中

**问题描述**:
- 长对话 (1000+ 消息) 渲染性能差
- DOM 节点过多导致内存占用高
- 滚动卡顿

**解决方案**:

```typescript
// 使用 vue-virtual-scroller
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

<RecycleScroller
  class="message-list"
  :items="messages"
  :item-size="80"
  key-field="id"
  v-slot="{ item }"
>
  <MessageCard :message="item" />
</RecycleScroller>
```

#### 技术债务 4: 缺少请求去重机制

**风险等级**: 🟡 中

**问题描述**:
- 用户快速点击可能发送重复请求
- 浪费服务器资源
- 可能导致数据不一致

**解决方案**:

```typescript
// utils/requestDeduplication.ts
const pendingRequests = new Map<string, Promise<any>>()

export function deduplicateRequest<T>(
  key: string,
  requestFn: () => Promise<T>
): Promise<T> {
  // 如果请求正在进行,返回现有 Promise
  if (pendingRequests.has(key)) {
    return pendingRequests.get(key)!
  }

  // 创建新请求
  const promise = requestFn().finally(() => {
    pendingRequests.delete(key)
  })

  pendingRequests.set(key, promise)
  return promise
}

// 使用
async function sendMessage(content: string) {
  const key = `send:${currentSessionId.value}:${content}`

  return deduplicateRequest(key, async () => {
    // 实际发送逻辑
  })
}
```

#### 技术债务 5: 缺少内存泄漏防护

**风险等级**: 🟡 中

**问题描述**:
- 定时器未清理
- 事件监听器未移除
- WebSocket 连接未关闭
- 长时间运行后内存占用增长

**解决方案**:

```typescript
// composables/useLifecycle.ts
export function useLifecycle() {
  const timers = new Set<ReturnType<typeof setTimeout>>()
  const intervals = new Set<ReturnType<typeof setInterval>>()
  const cleanupFns = new Array<() => void>()

  function setTimeoutSafe(callback: () => void, delay: number) {
    const timer = setTimeout(() => {
      callback()
      timers.delete(timer)
    }, delay)
    timers.add(timer)
    return timer
  }

  function setIntervalSafe(callback: () => void, delay: number) {
    const interval = setInterval(callback, delay)
    intervals.add(interval)
    return interval
  }

  function onCleanup(fn: () => void) {
    cleanupFns.push(fn)
  }

  // 组件卸载时清理
  onUnmounted(() => {
    timers.forEach(clearTimeout)
    intervals.forEach(clearInterval)
    cleanupFns.forEach(fn => fn())
  })

  return { setTimeoutSafe, setIntervalSafe, onCleanup }
}
```

#### 技术债务 6: 缺少防抖节流优化

**风险等级**: 🟢 低

**问题描述**:
- 输入框 `@input` 事件频繁触发
- 滚动事件未节流
- WebSocket 消息未防抖

**解决方案**:

```typescript
// utils/performance.ts
import { debounce, throttle } from 'lodash-es'

// 输入框搜索防抖
const search = debounce((query: string) => {
  performSearch(query)
}, 300)

// 滚动事件节流
const handleScroll = throttle((event: Event) => {
  updateScrollPosition(event)
}, 100)

// WebSocket 消息防抖
const processMetrics = debounce((metrics: HostMetrics[]) => {
  updateMetrics(metrics)
}, 200)
```

---

## 第四部分: 实施路线图 / Implementation Roadmap

### 阶段 0: 准备阶段 (1 周)

- [ ] 补充设计文档中的缺失组件定义
- [ ] 更新 TypeScript 类型定义
- [ ] 建立测试基础设施 (Vitest + Playwright)
- [ ] 设置 ESLint + Prettier 规范
- [ ] 建立 CI/CD 流程

### 阶段 1: 安全优先 (P0) - 2 周

**Week 1**:
- [ ] 实现 `ConfirmationDialog` 组件 (Req 11)
- [ ] 实现风险评估逻辑 (`riskAssessment.ts`)
- [ ] 实现 `AuditLogViewer` 组件 (Req 10)
- [ ] 建立审计日志 API
- [ ] 添加单元测试和 E2E 测试

**Week 2**:
- [ ] 实现 `ErrorBoundary` 组件
- [ ] 集成到所有关键路径
- [ ] 实现 `ChatPanel` 中间层 (架构优化)
- [ ] 修正 `Property 10` (AI 辅助终端)
- [ ] 代码审查和测试

### 阶段 2: 用户体验 (P1) - 2 周

**Week 3**:
- [ ] 实现主题系统 `ThemeProvider` (Req 13)
- [ ] 实现 `ShortcutManager` (Req 14)
- [ ] 实现 `ConnectionStatusIndicator` (Req 15)
- [ ] 优化响应式布局 (Req 1, Req 17)
- [ ] 添加 E2E 测试

**Week 4**:
- [ ] 实现 `HistoryPanelOverlay` (移动端)
- [ ] 实现深色模式样式
- [ ] 实现快捷键帮助模态框
- [ ] 实现连接重连逻辑
- [ ] 性能优化和测试

### 阶段 3: 高级功能 (P2) - 2 周

**Week 5**:
- [ ] 实现 `MessageSearchBar` (Req 16)
- [ ] 实现搜索算法和高亮显示
- [ ] 实现虚拟滚动优化
- [ ] 实现乐观更新机制
- [ ] 添加性能测试

**Week 6**:
- [ ] 完善 WebSocket 实时监控 (Req 4)
- [ ] 实现 IndexedDB 持久化 (Req 12)
- [ ] 实现防抖节流优化
- [ ] 全面回归测试
- [ ] 文档更新

### 阶段 4: 稳定与优化 (1 周)

- [ ] 修复所有已知 bug
- [ ] 性能调优
- [ ] 可访问性审计 (WCAG AA)
- [ ] 安全审计
- [ ] 准备发布

**总工时**: 6 周 (1 人)
**并行开发建议**: 2-3 人可缩短至 3-4 周

---

## 第五部分: 优先级建议 / Priority Recommendations

### 立即修复 (本周内)

1. **修正 `Property 10`** - 与 Req 9 保持一致
2. **补充 `AuditLogEntry` 类型** - Req 10 基础
3. **添加 `RiskAssessment` 接口** - Req 9/10/11 共用
4. **更新 `ToolCall` 接口** - 增加类型安全

### 短期实施 (2 周内)

1. **`ConfirmationDialog` 组件** - 安全必需
2. **`AuditLogViewer` 组件** - 审计必需
3. **`ErrorBoundary` 组件** - 稳定性必需
4. **完善测试覆盖** - 质量保证

### 中期实施 (1 个月内)

1. **主题系统** - 用户体验
2. **快捷键系统** - 效率提升
3. **连接状态指示器** - 可靠性
4. **响应式优化** - 移动端支持

### 长期规划 (2 个月内)

1. **消息搜索** - 高级功能
2. **虚拟滚动** - 性能优化
3. **乐观更新** - 交互体验
4. **全面审计** - 安全合规

---

## 第六部分: 总结与评分 / Summary & Scoring

### 设计质量总评

| 维度 | 评分 | 权重 | 加权分 |
|------|------|------|--------|
| 架构设计 | 4/5 | 25% | 1.00 |
| 接口设计 | 3/5 | 25% | 0.75 |
| 数据流设计 | 3/5 | 20% | 0.60 |
| 正确性属性 | 4/5 | 20% | 0.80 |
| 测试策略 | 2/5 | 10% | 0.20 |

**总分**: 3.35/5 ⭐⭐⭐ (3.5 星)

**优势**:
- ✅ 架构清晰,组件划分合理
- ✅ 正确性属性形式化描述
- ✅ 响应式设计考虑周全

**劣势**:
- ❌ 47% 的需求在设计文档中缺失
- ❌ 类型安全性不足
- ❌ 测试策略不够全面
- ❌ 安全机制设计缺失

### 关键建议

1. **立即补充缺失的组件设计** - 特别是安全和稳定性相关
2. **增强 TypeScript 类型定义** - 提升类型安全
3. **完善测试策略** - 单元、集成、E2E、性能全覆盖
4. **更新设计文档** - 与优化后的需求保持一致
5. **实施技术债务清理** - 错误边界、虚拟滚动、防抖节流

### 下一步行动

1. **本周**: 组织设计评审会议,讨论缺失组件
2. **下周**: 开始 P0 安全功能开发
3. **持续**: 每周同步设计文档更新状态

---

## 附录: 快速参考 / Quick Reference

### 补充的文件清单

```
建议新增的文件:

web/src/
├── types/
│   ├── audit.ts          # Req 10: 审计日志类型
│   ├── risk.ts           # Req 9/10/11: 风险评估类型
│   ├── theme.ts          # Req 13: 主题类型
│   ├── shortcuts.ts      # Req 14: 快捷键类型
│   └── connection.ts     # Req 15: 连接状态类型
├── components/
│   ├── audit/
│   │   ├── AuditLogViewer.vue
│   │   └── AuditLogFilters.vue
│   ├── confirmation/
│   │   └── ConfirmationDialog.vue
│   ├── theme/
│   │   ├── ThemeProvider.vue
│   │   └── ThemeToggle.vue
│   ├── shortcuts/
│   │   ├── ShortcutManager.vue
│   │   └── ShortcutHelpModal.vue
│   ├── connection/
│   │   └── ConnectionStatusIndicator.vue
│   ├── search/
│   │   ├── MessageSearchBar.vue
│   │   └── SearchResults.vue
│   └── mobile/
│       └── HistoryPanelOverlay.vue
├── stores/
│   ├── audit.ts
│   ├── ui.ts
│   └── preferences.ts
├── composables/
│   ├── useTheme.ts
│   ├── useShortcuts.ts
│   ├── useWebSocket.ts
│   ├── useLifecycle.ts
│   └── useOptimisticUpdate.ts
├── utils/
│   ├── riskAssessment.ts
│   ├── messageSearch.ts
│   ├── requestDeduplication.ts
│   └── performance.ts
└── api/
    ├── audit.ts
    └── terminal.ts
```

---

**报告结束**

*生成工具: Claude Code (Sonnet 4.5)*
*分析方法: 静态代码分析 + 架构评审 + 需求追溯*
*置信度: 高 (基于完整的源代码和文档分析)*
