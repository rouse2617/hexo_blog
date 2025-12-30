# Design Document: Chat UI Optimization

## Overview

本设计文档描述 AI-Ops 智能运维平台聊天界面的优化方案。核心目标是将当前的单栏宽屏布局改造为专业的三栏运维控制台风格，提升信息密度和操作效率。

### 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v1.0 | 初始版本 | 原始 9 个需求的设计 |
| v2.0 | 2024-12-31 | 基于分析报告优化：新增 8 个需求组件、修正类型定义、补充正确性属性 |

### 设计原则

1. **信息聚焦** - 对话流限制在 800px 宽度，减少视觉疲劳
2. **上下文关联** - 右侧面板实时展示与对话相关的服务器状态
3. **操作闭环** - 从 AI 建议到一键执行的完整工作流
4. **响应式适配** - 支持从移动端到超宽屏的多种设备
5. **安全优先** - 所有命令执行需经过风险评估和审计记录
6. **可访问性** - 支持键盘导航、深色模式、WCAG AA 标准

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              ChatWindow (Root)                                   │
│  ┌─────────────────────────────────────────────────────────────────────────────┐│
│  │ Header: [≡] 智能对话控制台  [🔍 搜索] [🌙/☀️ 主题] [🔌 连接状态] [👤] [⚙️] ││
│  └─────────────────────────────────────────────────────────────────────────────┘│
├──────────────┬────────────────────────────────────────┬─────────────────────────┤
│              │                                        │                         │
│  HistoryPanel│         ChatPanel (Center)             │    ContextPanel         │
│   (280px)    │           (max 800px)                  │     (min 300px)         │
│              │                                        │                         │
│ ┌──────────┐ │ ┌────────────────────────────────────┐ │ ┌─────────────────────┐ │
│ │ Sessions │ │ │        MessageList                 │ │ │  ResourceMonitor    │ │
│ │  List    │ │ │  ┌──────────────────────────────┐  │ │ │  - CPU Gauge        │ │
│ │          │ │ │  │ MessageCard                  │  │ │ │  - Memory Gauge     │ │
│ │          │ │ │  │ - ToolCallCard               │  │ │ │  - Disk Usage       │ │
│ │          │ │ │  │ - DiffViewer                 │  │ │ ├─────────────────────┤ │
│ │          │ │ │  │ - ActionButtons              │  │ │ │  RelatedFiles       │ │
│ │          │ │ │  │ - AggregatedProgressBar      │  │ │ │  - File List        │ │
│ │          │ │ │  └──────────────────────────────┘  │ │ │  - Preview Modal    │ │
│ │          │ │ │                                    │ │ ├─────────────────────┤ │
│ │          │ │ ├────────────────────────────────────┤ │ │  SilentModeToggle   │ │
│ │          │ │ │        InputArea                   │ │ │                     │ │
│ │          │ │ │  ┌──────────────────────────────┐  │ │ └─────────────────────┘ │
│ │          │ │ │  │ HostTagBar                   │  │ │                         │
│ │          │ │ │  ├──────────────────────────────┤  │ │                         │
│ │          │ │ │  │ EnhancedInputBox             │  │ │                         │
│ │          │ │ │  │ - SlashCommands              │  │ │                         │
│ │          │ │ │  │ - HostMention                │  │ │                         │
│ │          │ │ │  ├──────────────────────────────┤  │ │                         │
│ │          │ │ │  │ AIAssistedTerminal           │  │ │                         │
│ └──────────┘ │ │  └──────────────────────────────┘  │ │                         │
│              │ └────────────────────────────────────┘ │                         │
├──────────────┴────────────────────────────────────────┴─────────────────────────┤
│  Global Components: ConfirmationDialog | ShortcutHelpModal | AuditLogViewer     │
│                     MessageSearchBar | HistoryPanelOverlay | ConnectionInfo     │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 响应式断点

| 断点 | 宽度范围 | 布局 |
|------|----------|------|
| Desktop XL | > 1200px | 三栏布局 |
| Desktop | 768px - 1200px | 两栏布局（隐藏左侧） |
| Mobile | < 768px | 单栏布局 |

## Components and Interfaces

### 1. ChatWindow (根组件)

```typescript
interface ChatWindowProps {
  sessionId: string
  selectedHosts: string[]
}

interface ChatWindowState {
  layout: 'three-column' | 'two-column' | 'single-column'
  leftPanelVisible: boolean
  rightPanelVisible: boolean
  silentMode: boolean
}
```

### 2. HistoryPanel (历史记录面板)

```typescript
interface HistoryPanelProps {
  sessions: Session[]
  currentSessionId: string
  onSessionSelect: (sessionId: string) => void
  onSessionDelete: (sessionId: string) => void
}

interface Session {
  id: string
  title: string
  lastMessage: string
  timestamp: number
  messageCount: number
}
```

### 3. MessageCard (消息卡片)

```typescript
interface MessageCardProps {
  message: Message
  onCopy: () => void
  onExecute: (command: string) => void
  onSaveScript: () => void
  onRegenerate: () => void
}

interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  diffContent?: DiffContent
  fileReferences?: FileReference[]
  timestamp: number
}
```

### 4. ToolCallCard (工具调用卡片 - 增强版)

```typescript
interface ToolCallCardProps {
  toolCall: ToolCall
  isExpanded: boolean
  onToggleExpand: () => void
  onRerun: () => void
}

interface ToolCall {
  id: string
  name: string
  arguments: string
  result?: string
  status: 'pending' | 'running' | 'success' | 'error'
  hosts?: string[]  // 批量操作的主机列表
  hostResults?: HostResult[]  // 每个主机的执行结果
}

interface HostResult {
  host: string
  status: 'pending' | 'running' | 'success' | 'error'
  output?: string
  error?: string
}
```

### 5. AggregatedProgressBar (聚合进度条)

```typescript
interface AggregatedProgressBarProps {
  totalHosts: number
  completedHosts: number
  failedHosts: number
  onExpand: () => void
}
```

### 6. DiffViewer (Diff 视图)

```typescript
interface DiffViewerProps {
  oldContent: string
  newContent: string
  fileName: string
  onApply: () => void
  onReject: () => void
}

interface DiffLine {
  type: 'added' | 'removed' | 'unchanged'
  lineNumber: { old?: number; new?: number }
  content: string
}
```

### 7. EnhancedInputBox (增强输入框)

```typescript
interface EnhancedInputBoxProps {
  disabled: boolean
  selectedHosts: string[]
  onSend: (message: string) => void
  onHostSelect: (hosts: string[]) => void
  onSlashCommand: (command: SlashCommand) => void
}

interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string
}

const SLASH_COMMANDS: SlashCommand[] = [
  { name: '/top', description: '查看系统负载', template: '显示系统负载和 CPU 使用情况', icon: 'TrendCharts' },
  { name: '/tail', description: '查看日志', template: '查看最近的系统日志', icon: 'Document' },
  { name: '/check', description: '健康检查', template: '执行系统健康检查', icon: 'CircleCheck' },
  { name: '/disk', description: '磁盘检查', template: '检查磁盘使用情况', icon: 'FolderOpened' },
  { name: '/memory', description: '内存检查', template: '检查内存使用情况', icon: 'Cpu' },
  { name: '/process', description: '进程管理', template: '查看运行中的进程', icon: 'Operation' },
]
```

### 8. ContextPanel (上下文面板)

```typescript
interface ContextPanelProps {
  hosts: HostMetrics[]
  relatedFiles: FileReference[]
  silentMode: boolean
  onSilentModeToggle: () => void
  onFileClick: (file: FileReference) => void
}

interface HostMetrics {
  hostId: string
  hostName: string
  cpu: number
  memory: number
  disk: number
  status: 'normal' | 'warning' | 'critical'
  lastUpdated: number
}

interface FileReference {
  path: string
  name: string
  highlightLines?: number[]
  mentionedAt: number
}
```

### 9. QuickTerminal (快捷终端)

```typescript
interface QuickTerminalProps {
  hosts: string[]
  onExecute: (command: string, host: string) => Promise<string>
}

interface QuickTerminalState {
  isExpanded: boolean
  commandHistory: string[]
  historyIndex: number
  currentCommand: string
  output: TerminalOutput[]
}

interface TerminalOutput {
  command: string
  host: string
  result: string
  timestamp: number
  status: 'success' | 'error'
}
```

### 10. AIAssistedTerminal (AI 辅助终端 - Req 9 增强)

```typescript
// 替代原 QuickTerminal，增加风险评估功能
interface AIAssistedTerminalProps {
  hosts: string[]
  onExecute: (command: string, host: string) => Promise<CommandResult>
  onAssessRisk: (command: string) => Promise<RiskAssessment>
  onConfirmExecution: (request: ExecutionRequest) => Promise<void>
}

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

interface ExecutionRequest {
  command: string
  host: string
  riskLevel: RiskLevel
  userConfirmed: boolean
  confirmationTimestamp?: number
}
```

### 11. AuditLogViewer (审计日志查看器 - Req 10)

```typescript
interface AuditLogViewerProps {
  logs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  onFilterChange: (filters: AuditLogFilters) => void
  onExport: (format: 'csv' | 'json') => void
  onViewDetails: (logId: string) => void
}

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

### 12. ConfirmationDialog (二次确认对话框 - Req 11)

```typescript
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
  countdown?: number       // 倒计时秒数
  estimatedImpact?: string
}

// 样式配置 (基于风险级别)
const DIALOG_STYLES: Record<RiskLevel, DialogStyle> = {
  low: {
    title: '确认执行',
    icon: 'Info',
    color: '#409EFF',
    countdown: 0
  },
  medium: {
    title: '警告: 潜在风险',
    icon: 'Warning',
    color: '#E6A23C',
    countdown: 0
  },
  high: {
    title: '危险操作确认',
    icon: 'WarningFilled',
    color: '#F56C6C',
    countdown: 5
  },
  critical: {
    title: '极高风险操作',
    icon: 'CircleCloseFilled',
    color: '#C45656',
    countdown: 10
  }
}

interface DialogStyle {
  title: string
  icon: string
  color: string
  countdown: number
}
```

### 13. ThemeProvider (主题提供者 - Req 13)

```typescript
type ThemeMode = 'light' | 'dark' | 'auto'

interface ThemeProviderProps {
  defaultTheme?: ThemeMode
}

interface ThemeContextValue {
  mode: ThemeMode
  resolvedTheme: 'light' | 'dark'
  setTheme: (theme: ThemeMode) => void
  toggleTheme: () => void
}

interface ThemeConfig {
  mode: ThemeMode
  colors: {
    primary: string
    secondary: string
    background: string
    surface: string
    text: string
    textSecondary: string
    border: string
    error: string
    warning: string
    success: string
  }
}

// CSS 变量映射
const THEME_VARIABLES = {
  light: {
    '--color-background': '#ffffff',
    '--color-surface': '#f8fafc',
    '--color-text': '#1e293b',
    '--color-text-secondary': '#64748b',
    '--color-border': '#e2e8f0',
  },
  dark: {
    '--color-background': '#1a1a1a',
    '--color-surface': '#2d2d2d',
    '--color-text': '#e2e8f0',
    '--color-text-secondary': '#94a3b8',
    '--color-border': '#404040',
  }
}
```

### 14. ShortcutManager (快捷键管理器 - Req 14)

```typescript
interface ShortcutConfig {
  [key: string]: {
    description: string
    handler: (event: KeyboardEvent) => void
    disabled?: boolean
    category?: ShortcutCategory
  }
}

type ShortcutCategory = 'navigation' | 'editing' | 'actions' | 'panels'

interface ShortcutManagerProps {
  shortcuts: ShortcutConfig
  helpModalVisible?: boolean
  onHelpToggle?: () => void
}

// 默认快捷键配置
const DEFAULT_SHORTCUTS: ShortcutConfig = {
  'ctrl+enter': { description: '发送消息', handler: sendMessage, category: 'actions' },
  'cmd+enter': { description: '发送消息 (Mac)', handler: sendMessage, category: 'actions' },
  'ctrl+k': { description: '清空输入框', handler: clearInput, category: 'editing' },
  'ctrl+h': { description: '切换历史面板', handler: toggleHistoryPanel, category: 'panels' },
  'ctrl+m': { description: '切换监控面板', handler: toggleMonitorPanel, category: 'panels' },
  'ctrl+shift+c': { description: '复制上次命令输出', handler: copyLastOutput, category: 'actions' },
  'ctrl+f': { description: '搜索消息', handler: openSearch, category: 'navigation' },
  'escape': { description: '关闭模态框/下拉菜单', handler: closeModals, category: 'navigation' },
  'ctrl+/': { description: '显示快捷键帮助', handler: toggleShortcutHelp, category: 'navigation' },
  'up': { description: '上一条历史命令', handler: prevHistory, category: 'editing' },
  'down': { description: '下一条历史命令', handler: nextHistory, category: 'editing' },
  'tab': { description: '接受自动补全', handler: acceptAutocomplete, category: 'editing' },
}
```

### 15. ConnectionStatusIndicator (连接状态指示器 - Req 15)

```typescript
type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error'

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus
  latency?: number
  queueSize?: number
  onClick: () => void
}

interface ConnectionInfoModalProps {
  visible: boolean
  status: ConnectionStatus
  metrics: ConnectionMetrics
  onClose: () => void
  onReconnect: () => void
}

interface ConnectionMetrics {
  latency: number
  queueSize: number
  reconnectAttempts: number
  lastConnected?: number
  lastDisconnected?: number
  messagesReceived: number
  messagesSent: number
}

// WebSocket 配置
interface WebSocketConfig {
  url: string
  reconnectInterval: number      // 初始重连间隔 (ms)
  maxReconnectAttempts: number   // 最大重连次数
  heartbeatInterval: number      // 心跳间隔 (ms)
  maxBackoff: number             // 最大重连间隔 (ms)
}

const DEFAULT_WS_CONFIG: WebSocketConfig = {
  url: '/ws/chat',
  reconnectInterval: 1000,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  maxBackoff: 30000
}
```

### 16. MessageSearchBar (消息搜索栏 - Req 16)

```typescript
interface MessageSearchBarProps {
  visible: boolean
  resultsCount: number
  currentIndex: number
  onSearch: (query: SearchQuery) => void
  onNavigate: (direction: 'prev' | 'next') => void
  onClose: () => void
}

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
  matches: SearchMatch[]
  score: number  // 相关性评分
}

interface SearchMatch {
  field: 'content' | 'command' | 'filename' | 'output'
  text: string
  position: [number, number]  // [start, end]
}

// 搜索状态
interface SearchState {
  query: SearchQuery
  active: boolean
  results: SearchResult[]
  currentIndex: number
  totalCount: number
  loading: boolean
}
```

### 17. HistoryPanelOverlay (历史面板遮罩 - Req 17)

```typescript
interface HistoryPanelOverlayProps {
  visible: boolean
  sessions: Session[]
  currentSessionId: string
  onClose: () => void
  onSessionSelect: (sessionId: string) => void
  onSessionDelete: (sessionId: string) => void
}

// 支持触摸滑动关闭
interface TouchGestureConfig {
  swipeThreshold: number      // 滑动触发阈值 (px)
  swipeDirection: 'left' | 'right'
  onSwipeClose: () => void
}
```

### 18. SessionPersistence (会话持久化 - Req 12)

```typescript
// IndexedDB 数据库结构
interface ChatDatabase {
  messages: Table<Message>
  sessions: Table<Session>
  drafts: Table<InputDraft>
}

interface InputDraft {
  sessionId: string
  content: string
  selectedHosts: string[]
  timestamp: number
}

interface PersistenceConfig {
  autoSaveInterval: number    // 自动保存间隔 (ms)
  maxStorageSize: number      // 最大存储大小 (bytes)
  retentionDays: number       // 数据保留天数
}

const DEFAULT_PERSISTENCE_CONFIG: PersistenceConfig = {
  autoSaveInterval: 10000,    // 10 秒
  maxStorageSize: 50 * 1024 * 1024,  // 50MB
  retentionDays: 30
}

// 持久化 API
interface PersistenceAPI {
  saveMessages(sessionId: string, messages: Message[]): Promise<void>
  saveDraft(sessionId: string, draft: InputDraft): Promise<void>
  restoreSession(sessionId: string): Promise<{ messages: Message[], draft?: InputDraft }>
  exportConversation(sessionId: string, format: 'markdown' | 'json'): Promise<Blob>
  importConversation(file: File): Promise<Session>
  getLastSavedTime(sessionId: string): Promise<number | null>
}
```

### 19. ErrorBoundary (错误边界 - 通用)

```typescript
interface ErrorBoundaryProps {
  fallback?: (error: Error, retry: () => void) => VNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
}

interface ErrorInfo {
  componentStack: string
  timestamp: number
}

// 默认错误回退 UI
const DefaultErrorFallback = (error: Error, retry: () => void) => (
  <div class="error-boundary-fallback">
    <Icon name="WarningFilled" color="#F56C6C" size={48} />
    <h3>组件加载失败</h3>
    <p>{error.message}</p>
    <ElButton type="primary" onClick={retry}>重试</ElButton>
  </div>
)
```

## Data Models

### Store 设计 (Pinia)

```typescript
// stores/chat.ts 扩展
interface ChatState {
  // 现有字段
  sessions: Session[]
  currentSessionId: string
  messages: Message[]
  
  // 布局相关
  layout: LayoutType
  leftPanelVisible: boolean
  rightPanelVisible: boolean
  
  // 监控相关
  silentMode: boolean
  relatedFiles: FileReference[]
  hostMetrics: Map<string, HostMetrics>
  
  // 终端相关
  quickTerminalHistory: string[]
  quickTerminalExpanded: boolean
  
  // 搜索相关 (Req 16)
  searchState: SearchState
  
  // 持久化相关 (Req 12)
  lastSavedTime: number | null
  inputDraft: string
}

type LayoutType = 'three-column' | 'two-column' | 'single-column'

// stores/metrics.ts 扩展
interface MetricsState {
  hosts: Map<string, HostMetrics>
  refreshInterval: number
  lastRefresh: number
  isRefreshing: boolean
  
  // 新增字段
  refreshError?: Error
  consecutiveFailures: number  // 连续失败次数 (Req 4.8)
  subscriptions: Set<string>   // 已订阅的主机 ID
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

// stores/audit.ts 新增 (Req 10)
interface AuditLogState {
  logs: AuditLogEntry[]
  filteredLogs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  error?: Error
  statistics: AuditStatistics
}

interface AuditStatistics {
  totalCommands: number
  highRiskCommands: number
  failureRate: number
  mostActiveHosts: Array<{ host: string; count: number }>
}

// stores/ui.ts 新增 (全局 UI 状态)
interface UIState {
  // 主题 (Req 13)
  theme: ThemeMode
  resolvedTheme: 'light' | 'dark'
  
  // 快捷键 (Req 14)
  shortcutHelpVisible: boolean
  customShortcuts: Partial<ShortcutConfig>
  
  // 连接状态 (Req 15)
  connectionStatus: ConnectionStatus
  connectionMetrics: ConnectionMetrics
  
  // 模态框
  activeModal: ModalType | null
  modalStack: ModalType[]
  
  // 确认对话框 (Req 11)
  pendingConfirmations: ConfirmationRequest[]
  
  // 通知
  notifications: Notification[]
  
  // 移动端 (Req 17)
  historyPanelOverlayVisible: boolean
}

type ModalType = 
  | 'confirmation'
  | 'shortcut-help'
  | 'connection-info'
  | 'file-preview'
  | 'audit-log'
  | 'export-conversation'

interface Notification {
  id: number
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
  timestamp: number
  actions?: Array<{ label: string; handler: () => void }>
}

// stores/preferences.ts 新增 (用户偏好)
interface UserPreferencesState {
  // 主题 (Req 13)
  theme: ThemeMode
  
  // 快捷键 (Req 14)
  customShortcuts: Partial<ShortcutConfig>
  
  // 输入偏好
  sendOnEnter: boolean
  autoCollapseOutput: boolean
  maxOutputHeight: number
  
  // 通知偏好
  enableNotifications: boolean
  soundEnabled: boolean
  
  // 隐私偏好
  saveHistory: boolean
  analyticsEnabled: boolean
}
```

### API 接口扩展

```typescript
// api/metrics.ts 新增
interface MetricsAPI {
  // 获取主机实时指标
  getHostMetrics(hostIds: string[]): Promise<HostMetrics[]>
  
  // 订阅指标更新 (WebSocket)
  subscribeMetrics(hostIds: string[], callback: (metrics: HostMetrics) => void): () => void
}

// api/terminal.ts 新增
interface TerminalAPI {
  // 直接执行命令（不经过 AI）
  executeCommand(host: string, command: string): Promise<CommandResult>
  
  // 风险评估 (Req 9, 11)
  assessCommandRisk(command: string): Promise<RiskAssessment>
}

interface CommandResult {
  stdout: string
  stderr: string
  exitCode: number
  duration: number
}

// api/audit.ts 新增 (Req 10)
interface AuditLogAPI {
  // 记录命令执行
  logCommand(entry: Omit<AuditLogEntry, 'id' | 'timestamp'>): Promise<void>
  
  // 查询日志
  queryLogs(filters: AuditLogFilters): Promise<{ logs: AuditLogEntry[], total: number }>
  
  // 导出日志
  exportLogs(filters: AuditLogFilters, format: 'csv' | 'json'): Promise<Blob>
  
  // 获取统计信息
  getStatistics(timeRange: [Date, Date]): Promise<AuditStatistics>
}

// api/persistence.ts 新增 (Req 12)
interface PersistenceAPI {
  // 保存消息到 IndexedDB
  saveMessages(sessionId: string, messages: Message[]): Promise<void>
  
  // 保存输入草稿
  saveDraft(sessionId: string, draft: InputDraft): Promise<void>
  
  // 恢复会话
  restoreSession(sessionId: string): Promise<{ messages: Message[], draft?: InputDraft }>
  
  // 导出对话
  exportConversation(sessionId: string, format: 'markdown' | 'json'): Promise<Blob>
  
  // 导入对话
  importConversation(file: File): Promise<Session>
  
  // 获取最后保存时间
  getLastSavedTime(sessionId: string): Promise<number | null>
}

// api/websocket.ts 新增 (Req 15)
interface WebSocketAPI {
  // 连接管理
  connect(): Promise<void>
  disconnect(): void
  reconnect(): Promise<void>
  
  // 状态查询
  getStatus(): ConnectionStatus
  getMetrics(): ConnectionMetrics
  
  // 消息发送
  send(message: WebSocketMessage): void
  
  // 事件监听
  on(event: 'message' | 'open' | 'close' | 'error', handler: Function): () => void
}

interface WebSocketMessage<T = unknown> {
  type: string
  payload: T
  timestamp: number
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Responsive Layout Consistency

*For any* viewport width, the Chat_Window layout should match the expected configuration:
- width > 1200px → three-column layout
- 768px ≤ width ≤ 1200px → two-column layout  
- width < 768px → single-column layout

**Validates: Requirements 1.1, 1.2, 1.3**

### Property 2: Command Output Collapse Behavior

*For any* command output with more than 3 lines, the Tool_Call_Card should render in collapsed state by default, showing only the first 3 lines with an expand button.

**Validates: Requirements 2.1**

### Property 3: Batch Operation Aggregation

*For any* tool execution targeting N hosts where N > 1, the UI should display a single aggregated progress bar instead of N separate Tool_Call_Cards.

**Validates: Requirements 2.3**

### Property 4: Slash Command Trigger

*For any* input text starting with `/`, the Input_Box should display a filtered dropdown menu of matching Slash_Commands within 100ms.

**Validates: Requirements 3.1, 3.3**

### Property 5: Host Mention Trigger

*For any* input text containing `@`, the Input_Box should display a dropdown menu of available hosts filtered by the text after `@`.

**Validates: Requirements 3.4, 3.5**

### Property 6: Metrics Display Completeness

*For any* selected host, the Context_Panel should display both CPU and memory usage percentages.

**Validates: Requirements 4.1, 4.2**

### Property 7: Threshold Alert Highlighting

*For any* metric value exceeding its threshold (CPU > 80% OR Memory > 85%), the Context_Panel should apply visual highlighting (color change + optional animation).

**Validates: Requirements 4.3**

### Property 8: File Reference Extraction

*For any* AI response containing file path patterns (e.g., `/var/log/*`, `/etc/*`), the Context_Panel should extract and display these files in the "相关文件" section.

**Validates: Requirements 4.4**

### Property 9: Silent Mode Behavior

*For any* state where silent mode is enabled:
- If all metrics are normal → panel should be collapsed
- If any metric exceeds threshold → panel should auto-expand

**Validates: Requirements 8.2, 8.3, 8.5**

### Property 10: AI Assisted Terminal Execution (修正)

*For any* command entered in AIAssistedTerminal, THE System SHALL:
1. First display AI risk assessment with risk level and explanation
2. For low-risk commands: execute directly after user clicks "执行"
3. For medium/high-risk commands: require confirmation dialog before execution
4. Log all executions to the audit system

**Validates: Requirements 9.3, 9.4, 9.8**

### Property 11: Diff View Correctness

*For any* configuration modification suggestion, the Diff_View should correctly identify and display:
- Deleted lines (red background)
- Added lines (green background)
- Line numbers for both old and new content

**Validates: Requirements 7.1, 7.2, 7.3**

### Property 12: Action Buttons Presence

*For any* AI assistant message, the Message_Card should display action buttons: "复制命令", "推荐执行", "保存为脚本".

**Validates: Requirements 2.6**

### Property 13: Audit Log Completeness (Req 10)

*For any* command execution through any interface (Chat, Quick_Terminal, Slash_Command), THE System SHALL log an Audit_Log_Entry containing all required fields: timestamp, user_id, command, target_hosts, execution_result, risk_level.

**Validates: Requirements 10.1, 10.2**

### Property 14: High-Risk Command Confirmation (Req 11)

*For any* command with risk_level = 'high' or 'critical', THE System SHALL display a Confirmation_Dialog AND require typing "CONFIRM" before execution. The confirm button SHALL be disabled for a countdown period.

**Validates: Requirements 11.1, 11.3, 11.4, 11.6**

### Property 15: Session Persistence Accuracy (Req 12)

*For any* page reload, THE System SHALL restore the previous session state with 100% message accuracy AND restore input draft content. Auto-save SHALL occur every 10 seconds.

**Validates: Requirements 12.1, 12.2, 12.3**

### Property 16: Theme Consistency (Req 13)

*For any* theme change, THE System SHALL apply the new theme to ALL UI components WITHOUT page reload AND persist the preference to localStorage. Dark mode SHALL use #1a1a1a background (not pure black).

**Validates: Requirements 13.1, 13.2, 13.5, 13.6**

### Property 17: Keyboard Shortcut Execution (Req 14)

*For any* registered keyboard shortcut, IF there is a conflict with browser defaults, THE System SHALL prevent default behavior AND execute the custom handler. A toast notification SHALL be shown when action is performed.

**Validates: Requirements 14.1, 14.3, 14.4**

### Property 18: WebSocket Reconnection Behavior (Req 15)

*For any* connection loss, THE System SHALL:
1. Display disconnected status indicator (red)
2. Attempt reconnection with exponential backoff (1s, 2s, 4s, 8s, max 30s)
3. Show toast notification on reconnection success

**Validates: Requirements 15.2, 15.4, 15.5, 15.6**

### Property 19: Search Result Accuracy (Req 16)

*For any* search query, THE System SHALL return ALL messages matching the search criteria AND highlight matching text. Search SHALL support content, command output, and file names.

**Validates: Requirements 16.2, 16.3**

### Property 20: Mobile History Panel Behavior (Req 17)

*For any* screen width < 1200px, THE System SHALL display a hamburger menu button AND show History_Panel_Overlay when clicked. The overlay SHALL support swipe-to-close on touch devices.

**Validates: Requirements 17.1, 17.2, 17.5**

### Property 21: AI Terminal Risk Assessment (Req 9 修正)

*For any* command entered in AIAssistedTerminal, THE System SHALL:
1. Display AI risk assessment and command explanation
2. Require user confirmation for medium or high risk commands
3. Log the execution to the audit system

**Validates: Requirements 9.3, 9.4, 9.8**

### Property 22: Output Height Consistency

*For any* command output OR code block OR diff view exceeding 300px height, THE UI SHALL limit display to 300px with overflow scrolling AND show expand button.

**Validates: Requirements 2.2, 6.3**

## Error Handling

### 网络错误处理

| 场景 | 处理方式 |
|------|----------|
| 指标获取失败 | 显示上次成功数据 + 错误提示，60秒后重试，连续 3 次失败后停止自动刷新 |
| WebSocket 断开 | 自动重连，指数退避 (1s, 2s, 4s, 8s, max 30s)，显示连接状态指示器 |
| 命令执行超时 | 显示超时提示，允许重试或取消 |
| 主机不可达 | 在 Context_Panel 显示离线状态，暂停该主机的指标刷新 |
| IndexedDB 写入失败 | 降级到 localStorage，显示警告提示 |
| 审计日志写入失败 | 本地缓存，网络恢复后重试 |

### 用户输入错误处理

| 场景 | 处理方式 |
|------|----------|
| 无效的 Slash Command | 显示"未知命令"提示，列出可用命令 |
| 无效的主机名 | 高亮显示，提示"主机不存在" |
| 空消息发送 | 禁用发送按钮，不做处理 |
| 危险命令检测 | 显示风险评估，要求确认 |
| 搜索无结果 | 显示"未找到匹配结果"，建议调整搜索条件 |

### 安全错误处理

| 场景 | 处理方式 |
|------|----------|
| 高风险命令 | 显示红色警告对话框，要求输入 "CONFIRM"，5秒倒计时 |
| 权限不足 | 显示权限错误，建议联系管理员 |
| 会话过期 | 提示重新登录，保存当前草稿 |

## Testing Strategy

### 单元测试 (Vitest)

- 组件渲染测试：验证各组件在不同 props 下的渲染结果
- 状态管理测试：验证 Pinia store 的状态变更逻辑
- 工具函数测试：验证 Diff 计算、文件路径提取、风险评估等工具函数
- 主题切换测试：验证 CSS 变量正确应用
- 快捷键测试：验证快捷键注册和触发

### 属性测试 (Property-Based Testing)

使用 `fast-check` 库进行属性测试：

```typescript
import fc from 'fast-check'

// Property 1: 响应式布局
fc.assert(
  fc.property(fc.integer({ min: 320, max: 2560 }), (width) => {
    const layout = calculateLayout(width)
    if (width > 1200) return layout === 'three-column'
    if (width >= 768) return layout === 'two-column'
    return layout === 'single-column'
  })
)

// Property 2: 命令输出折叠
fc.assert(
  fc.property(fc.array(fc.string(), { minLength: 6 }), (lines) => {
    const output = lines.join('\n')
    const card = renderToolCallCard({ result: output })
    return card.isCollapsed === true && card.visibleLines <= 5
  })
)

// Property 14: 高风险命令确认
fc.assert(
  fc.property(
    fc.record({
      command: fc.string(),
      riskLevel: fc.constantFrom('low', 'medium', 'high', 'critical')
    }),
    ({ command, riskLevel }) => {
      const dialog = renderConfirmationDialog({ command, riskLevel })
      if (riskLevel === 'high' || riskLevel === 'critical') {
        return dialog.requiresTyping === true && dialog.countdown > 0
      }
      return true
    }
  )
)

// Property 18: WebSocket 重连
fc.assert(
  fc.property(fc.integer({ min: 0, max: 10 }), (attempts) => {
    const delay = calculateReconnectDelay(attempts)
    const expectedDelay = Math.min(1000 * Math.pow(2, attempts), 30000)
    return delay === expectedDelay
  })
)
```

### 集成测试

- E2E 测试：使用 Cypress 测试完整用户流程
- 视觉回归测试：使用 Percy 或 Chromatic 检测 UI 变化
- 主题测试：验证深色/浅色模式下的视觉一致性
- 响应式测试：验证不同断点下的布局正确性

### 安全测试

- 危险命令检测测试：验证 `rm -rf`, `dd`, `mkfs` 等模式被正确识别
- 确认对话框测试：验证高风险操作需要正确的确认流程
- 审计日志测试：验证所有命令执行被正确记录

### 测试配置

- 属性测试最小迭代次数：100
- 每个属性测试需标注对应的设计文档属性编号
- 标签格式：`**Feature: chat-ui-optimization, Property {number}: {property_text}**`

## 需求覆盖矩阵

| Req | 需求描述 | 组件 | 正确性属性 | 状态 |
|-----|---------|------|-----------|------|
| Req 1 | 三栏布局改造 | ChatWindow | Property 1 | ✅ |
| Req 2 | 对话卡片运维化 | ToolCallCard, AggregatedProgressBar | Property 2, 3, 12, 22 | ✅ |
| Req 3 | 输入框增强 | EnhancedInputBox | Property 4, 5 | ✅ |
| Req 4 | 实时资源监控 | ContextPanel | Property 6, 7, 8 | ✅ |
| Req 5 | AI 思考状态细化 | MessageCard | - | ✅ |
| Req 6 | 消息气泡宽度优化 | MessageCard | Property 22 | ✅ |
| Req 7 | 配置文件 Diff 视图 | DiffViewer | Property 11 | ✅ |
| Req 8 | 监控面板静默模式 | ContextPanel | Property 9 | ✅ |
| Req 9 | AI 辅助快捷终端 | AIAssistedTerminal | Property 10, 21 | ✅ |
| Req 10 | 命令执行审计日志 | AuditLogViewer | Property 13 | ✅ |
| Req 11 | 敏感操作二次确认 | ConfirmationDialog | Property 14 | ✅ |
| Req 12 | 会话持久化与恢复 | SessionPersistence | Property 15 | ✅ |
| Req 13 | 深色模式支持 | ThemeProvider | Property 16 | ✅ |
| Req 14 | 快捷键系统 | ShortcutManager | Property 17 | ✅ |
| Req 15 | WebSocket 连接状态 | ConnectionStatusIndicator | Property 18 | ✅ |
| Req 16 | 消息搜索与过滤 | MessageSearchBar | Property 19 | ✅ |
| Req 17 | 小屏幕历史面板访问 | HistoryPanelOverlay | Property 20 | ✅ |

## 实施优先级

| 阶段 | 需求 | 组件 | 预计工时 |
|------|------|------|---------|
| **P0 - MVP** | Req 1, 6, 5 | ChatWindow, MessageCard | 3-4 天 |
| **P0 - 安全** | Req 10, 11 | AuditLogViewer, ConfirmationDialog | 3-4 天 |
| **P1 - 交互** | Req 2, 3, 8, 17 | ToolCallCard, EnhancedInputBox, HistoryPanelOverlay | 5-6 天 |
| **P1 - 体验** | Req 12, 13, 14, 15 | SessionPersistence, ThemeProvider, ShortcutManager, ConnectionStatusIndicator | 4-5 天 |
| **P2 - 高级** | Req 4, 7, 9, 16 | ContextPanel, DiffViewer, AIAssistedTerminal, MessageSearchBar | 6-8 天 |

**总预计工时**: 21-27 天
