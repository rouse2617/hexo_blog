/**
 * Chat UI Optimization - Type Definitions
 * 
 * This file contains all type definitions for the Chat UI optimization feature.
 * It serves as the foundation for all components, stores, and APIs.
 * 
 * @module types/chat-ui
 */

// ============================================================================
// Core Enums and Constants
// ============================================================================

/** Risk level for command execution */
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

/** Connection status for WebSocket */
export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected' | 'error'

/** Theme mode options */
export type ThemeMode = 'light' | 'dark' | 'auto'

/** Layout type based on screen width */
export type LayoutType = 'three-column' | 'two-column' | 'single-column'

/** Modal types for UI state management */
export type ModalType =
  | 'confirmation'
  | 'shortcut-help'
  | 'connection-info'
  | 'file-preview'
  | 'audit-log'
  | 'export-conversation'

/** Shortcut categories */
export type ShortcutCategory = 'navigation' | 'editing' | 'actions' | 'panels'

/** Audit log source types */
export type AuditSource = 'chat' | 'quick_terminal' | 'slash_command'

/** Audit trigger types */
export type AuditTrigger = 'user' | 'ai' | 'auto'

// ============================================================================
// Message and Chat Types
// ============================================================================

/** Base message interface */
export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  diffContent?: DiffContent
  fileReferences?: FileReference[]
  timestamp: number
}

/** Tool call interface */
export interface ToolCall {
  id: string
  name: string
  arguments: string
  result?: string
  status: 'pending' | 'running' | 'success' | 'error'
  hosts?: string[]
  hostResults?: HostResult[]
}

/** Host execution result */
export interface HostResult {
  host: string
  status: 'pending' | 'running' | 'success' | 'error'
  output?: string
  error?: string
}

/** Diff content for configuration changes */
export interface DiffContent {
  oldContent: string
  newContent: string
  fileName: string
}

/** File reference in messages */
export interface FileReference {
  path: string
  name: string
  highlightLines?: number[]
  mentionedAt: number
}

/** Chat session interface */
export interface Session {
  id: string
  title: string
  lastMessage: string
  timestamp: number
  messageCount: number
}

// ============================================================================
// Host and Metrics Types
// ============================================================================

/** Host metrics interface */
export interface HostMetrics {
  hostId: string
  hostName: string
  cpu: number
  memory: number
  disk: number
  status: 'normal' | 'warning' | 'critical'
  lastUpdated: number
}

/** Metric alert interface */
export interface MetricAlert {
  hostId: string
  metricType: 'cpu' | 'memory' | 'disk'
  value: number
  threshold: number
  timestamp: number
  acknowledged: boolean
}

// ============================================================================
// Risk Assessment and Audit Types
// ============================================================================

/** Risk assessment result */
export interface RiskAssessment {
  level: RiskLevel
  score: number
  reason: string
  warnings: string[]
  requiresConfirmation: boolean
  affectedResources?: string[]
  estimatedImpact?: string
  mitigation?: string[]
}

/** Execution request for commands */
export interface ExecutionRequest {
  command: string
  host: string
  riskLevel: RiskLevel
  userConfirmed: boolean
  confirmationTimestamp?: number
}

/** Command execution result */
export interface CommandResult {
  stdout: string
  stderr: string
  exitCode: number
  duration: number
}

/** Audit log entry */
export interface AuditLogEntry {
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

/** Execution result for audit */
export interface ExecutionResult {
  status: 'success' | 'failure' | 'cancelled'
  exitCode?: number
  duration: number
  output?: string
  error?: string
}

/** Audit metadata */
export interface AuditMetadata {
  source: AuditSource
  userAgent?: string
  ipAddress?: string
  triggeredBy: AuditTrigger
  relatedToolCall?: string
}

/** Audit log filters */
export interface AuditLogFilters {
  timeRange?: [Date, Date]
  userId?: string
  riskLevel?: RiskLevel
  host?: string
  commandPattern?: string
  limit?: number
  offset?: number
}

/** Audit statistics */
export interface AuditStatistics {
  totalCommands: number
  highRiskCommands: number
  failureRate: number
  mostActiveHosts: Array<{ host: string; count: number }>
}

// ============================================================================
// Confirmation Dialog Types
// ============================================================================

/** Confirmation request */
export interface ConfirmationRequest {
  id: string
  command: string
  riskLevel: RiskLevel
  affectedHosts: string[]
  requiresTyping: boolean
  countdown?: number
  estimatedImpact?: string
}

/** Dialog style configuration */
export interface DialogStyle {
  title: string
  icon: string
  color: string
  countdown: number
}

/** Dialog styles by risk level */
export const DIALOG_STYLES: Record<RiskLevel, DialogStyle> = {
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

// ============================================================================
// Input and Slash Command Types
// ============================================================================

/** Slash command definition */
export interface SlashCommand {
  name: string
  description: string
  template: string
  icon: string
}

/** Default slash commands */
export const SLASH_COMMANDS: SlashCommand[] = [
  { name: '/top', description: '查看系统负载', template: '显示系统负载和 CPU 使用情况', icon: 'TrendCharts' },
  { name: '/tail', description: '查看日志', template: '查看最近的系统日志', icon: 'Document' },
  { name: '/check', description: '健康检查', template: '执行系统健康检查', icon: 'CircleCheck' },
  { name: '/disk', description: '磁盘检查', template: '检查磁盘使用情况', icon: 'FolderOpened' },
  { name: '/memory', description: '内存检查', template: '检查内存使用情况', icon: 'Cpu' },
  { name: '/process', description: '进程管理', template: '查看运行中的进程', icon: 'Operation' },
]

// ============================================================================
// Search Types
// ============================================================================

/** Search query interface */
export interface SearchQuery {
  text?: string
  timeRange?: [Date, Date]
  toolType?: string[]
  hosts?: string[]
  errorLevel?: 'info' | 'warning' | 'error'
  caseSensitive?: boolean
  regex?: boolean
}

/** Search result interface */
export interface SearchResult {
  message: Message
  matches: SearchMatch[]
  score: number
}

/** Search match interface */
export interface SearchMatch {
  field: 'content' | 'command' | 'filename' | 'output'
  text: string
  position: [number, number]
}

/** Search state */
export interface SearchState {
  query: SearchQuery
  active: boolean
  results: SearchResult[]
  currentIndex: number
  totalCount: number
  loading: boolean
}

// ============================================================================
// WebSocket and Connection Types
// ============================================================================

/** Connection metrics */
export interface ConnectionMetrics {
  latency: number
  queueSize: number
  reconnectAttempts: number
  lastConnected?: number
  lastDisconnected?: number
  messagesReceived: number
  messagesSent: number
}

/** WebSocket configuration */
export interface WebSocketConfig {
  url: string
  reconnectInterval: number
  maxReconnectAttempts: number
  heartbeatInterval: number
  maxBackoff: number
}

/** Default WebSocket configuration */
export const DEFAULT_WS_CONFIG: WebSocketConfig = {
  url: '/ws/chat',
  reconnectInterval: 1000,
  maxReconnectAttempts: 10,
  heartbeatInterval: 30000,
  maxBackoff: 30000
}

/** WebSocket message */
export interface WebSocketMessage<T = unknown> {
  type: string
  payload: T
  timestamp: number
}

// ============================================================================
// Shortcut Types
// ============================================================================

/** Shortcut configuration */
export interface ShortcutConfig {
  [key: string]: {
    description: string
    handler: (event: KeyboardEvent) => void
    disabled?: boolean
    category?: ShortcutCategory
  }
}

// ============================================================================
// Persistence Types
// ============================================================================

/** Input draft for persistence */
export interface InputDraft {
  sessionId: string
  content: string
  selectedHosts: string[]
  timestamp: number
}

/** Persistence configuration */
export interface PersistenceConfig {
  autoSaveInterval: number
  maxStorageSize: number
  retentionDays: number
}

/** Default persistence configuration */
export const DEFAULT_PERSISTENCE_CONFIG: PersistenceConfig = {
  autoSaveInterval: 10000,
  maxStorageSize: 50 * 1024 * 1024,
  retentionDays: 30
}

// ============================================================================
// Notification Types
// ============================================================================

/** Notification interface */
export interface Notification {
  id: number
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message?: string
  duration?: number
  timestamp: number
  actions?: Array<{ label: string; handler: () => void }>
}

// ============================================================================
// Theme Types
// ============================================================================

/** Theme configuration */
export interface ThemeConfig {
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

/** CSS variables for themes */
export const THEME_VARIABLES = {
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
} as const

// ============================================================================
// Store State Interfaces
// ============================================================================

/** Chat window state */
export interface ChatWindowState {
  layout: LayoutType
  leftPanelVisible: boolean
  rightPanelVisible: boolean
  silentMode: boolean
}

/** Chat state for store */
export interface ChatState {
  sessions: Session[]
  currentSessionId: string
  messages: Message[]
  layout: LayoutType
  leftPanelVisible: boolean
  rightPanelVisible: boolean
  silentMode: boolean
  relatedFiles: FileReference[]
  hostMetrics: Map<string, HostMetrics>
  quickTerminalHistory: string[]
  quickTerminalExpanded: boolean
  searchState: SearchState
  lastSavedTime: number | null
  inputDraft: string
}

/** Metrics state for store */
export interface MetricsState {
  hosts: Map<string, HostMetrics>
  refreshInterval: number
  lastRefresh: number
  isRefreshing: boolean
  refreshError?: Error
  consecutiveFailures: number
  subscriptions: Set<string>
  abnormalHosts: Set<string>
  alertHistory: MetricAlert[]
}

/** Audit log state for store */
export interface AuditLogState {
  logs: AuditLogEntry[]
  filteredLogs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  error?: Error
  statistics: AuditStatistics
}

/** UI state for store */
export interface UIState {
  theme: ThemeMode
  resolvedTheme: 'light' | 'dark'
  shortcutHelpVisible: boolean
  customShortcuts: Partial<ShortcutConfig>
  connectionStatus: ConnectionStatus
  connectionMetrics: ConnectionMetrics
  activeModal: ModalType | null
  modalStack: ModalType[]
  pendingConfirmations: ConfirmationRequest[]
  notifications: Notification[]
  historyPanelOverlayVisible: boolean
}

/** User preferences state */
export interface UserPreferencesState {
  theme: ThemeMode
  customShortcuts: Partial<ShortcutConfig>
  sendOnEnter: boolean
  autoCollapseOutput: boolean
  maxOutputHeight: number
  enableNotifications: boolean
  soundEnabled: boolean
  saveHistory: boolean
  analyticsEnabled: boolean
}

// ============================================================================
// Component Props Interfaces
// ============================================================================

/** ChatWindow props */
export interface ChatWindowProps {
  sessionId: string
  selectedHosts: string[]
}

/** HistoryPanel props */
export interface HistoryPanelProps {
  sessions: Session[]
  currentSessionId: string
  onSessionSelect: (sessionId: string) => void
  onSessionDelete: (sessionId: string) => void
}

/** MessageCard props */
export interface MessageCardProps {
  message: Message
  onCopy: () => void
  onExecute: (command: string) => void
  onSaveScript: () => void
  onRegenerate: () => void
}

/** ToolCallCard props */
export interface ToolCallCardProps {
  toolCall: ToolCall
  isExpanded: boolean
  onToggleExpand: () => void
  onRerun: () => void
}

/** AggregatedProgressBar props */
export interface AggregatedProgressBarProps {
  totalHosts: number
  completedHosts: number
  failedHosts: number
  onExpand: () => void
}

/** DiffViewer props */
export interface DiffViewerProps {
  oldContent: string
  newContent: string
  fileName: string
  onApply: () => void
  onReject: () => void
}

/** ContextPanel props */
export interface ContextPanelProps {
  hosts: HostMetrics[]
  relatedFiles: FileReference[]
  silentMode: boolean
  onSilentModeToggle: () => void
  onFileClick: (file: FileReference) => void
}

/** AIAssistedTerminal props */
export interface AIAssistedTerminalProps {
  hosts: string[]
  onExecute: (command: string, host: string) => Promise<CommandResult>
  onAssessRisk: (command: string) => Promise<RiskAssessment>
  onConfirmExecution: (request: ExecutionRequest) => Promise<void>
}

/** AuditLogViewer props */
export interface AuditLogViewerProps {
  logs: AuditLogEntry[]
  filters: AuditLogFilters
  loading: boolean
  onFilterChange: (filters: AuditLogFilters) => void
  onExport: (format: 'csv' | 'json') => void
  onViewDetails: (logId: string) => void
}

/** ConfirmationDialog props */
export interface ConfirmationDialogProps {
  visible: boolean
  request: ConfirmationRequest
  onConfirm: () => void
  onCancel: () => void
}

/** ConnectionStatusIndicator props */
export interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus
  latency?: number
  queueSize?: number
  onClick: () => void
}

/** MessageSearchBar props */
export interface MessageSearchBarProps {
  visible: boolean
  resultsCount: number
  currentIndex: number
  onSearch: (query: SearchQuery) => void
  onNavigate: (direction: 'prev' | 'next') => void
  onClose: () => void
}

/** HistoryPanelOverlay props */
export interface HistoryPanelOverlayProps {
  visible: boolean
  sessions: Session[]
  currentSessionId: string
  onClose: () => void
  onSessionSelect: (sessionId: string) => void
  onSessionDelete: (sessionId: string) => void
}

/** Touch gesture configuration */
export interface TouchGestureConfig {
  swipeThreshold: number
  swipeDirection: 'left' | 'right'
  onSwipeClose: () => void
}

// ============================================================================
// Diff Types
// ============================================================================

/** Diff line interface */
export interface DiffLine {
  type: 'added' | 'removed' | 'unchanged'
  lineNumber: { old?: number; new?: number }
  content: string
}

// ============================================================================
// Terminal Types
// ============================================================================

/** Terminal output interface */
export interface TerminalOutput {
  command: string
  host: string
  result: string
  timestamp: number
  status: 'success' | 'error'
}

/** Quick terminal state */
export interface QuickTerminalState {
  isExpanded: boolean
  commandHistory: string[]
  historyIndex: number
  currentCommand: string
  output: TerminalOutput[]
}

// ============================================================================
// Error Boundary Types
// ============================================================================

/** Error info interface */
export interface ErrorInfo {
  componentStack: string
  timestamp: number
}

/** Error boundary props */
export interface ErrorBoundaryProps {
  fallback?: (error: Error, retry: () => void) => any
  onError?: (error: Error, errorInfo: ErrorInfo) => void
}

// ============================================================================
// Thinking Process Types (Req 5)
// ============================================================================

/** Thinking step status */
export type ThinkingStepStatus = 'calling_llm' | 'executing_tools' | 'analyzing_results' | 'reading_file' | 'searching' | 'error'

/** Enhanced thinking step with detailed information */
export interface EnhancedThinkingStep {
  step: number
  status: ThinkingStepStatus
  content: string
  timestamp: number
  startTime: number
  endTime?: number
  elapsedTime?: number
  toolCalls?: EnhancedToolCallInfo[]
  fileReferences?: ThinkingFileReference[]
  error?: ThinkingStepError
}

/** Enhanced tool call info for thinking process */
export interface EnhancedToolCallInfo {
  id: string
  tool: string
  params: string
  status: 'pending' | 'running' | 'success' | 'error'
  result?: string
  error?: string
  startTime?: number
  endTime?: number
  elapsedTime?: number
}

/** File reference in thinking process */
export interface ThinkingFileReference {
  path: string
  name: string
  lineNumber?: number
  lineRange?: [number, number]
  mentionedAt: number
}

/** Error information for thinking step */
export interface ThinkingStepError {
  message: string
  code?: string
  retryable: boolean
  details?: string
}

/** Semantic step descriptions mapping */
export const THINKING_STEP_DESCRIPTIONS: Record<ThinkingStepStatus, string> = {
  calling_llm: '正在分析问题...',
  executing_tools: '正在执行工具...',
  analyzing_results: '正在分析结果...',
  reading_file: '正在读取文件...',
  searching: '正在检索解决方案...',
  error: '执行出错'
}

/** Get semantic description for a thinking step */
export function getThinkingStepDescription(
  status: ThinkingStepStatus,
  context?: { filename?: string; hostname?: string; command?: string; toolName?: string }
): string {
  switch (status) {
    case 'reading_file':
      return context?.filename ? `正在读取 ${context.filename}...` : '正在读取文件...'
    case 'executing_tools':
      if (context?.hostname && context?.command) {
        return `正在主机 ${context.hostname} 上执行 ${context.command}...`
      }
      if (context?.toolName) {
        return `正在执行 ${context.toolName}...`
      }
      return '正在执行工具...'
    case 'searching':
      return '正在检索解决方案...'
    case 'calling_llm':
      return '正在分析问题...'
    case 'analyzing_results':
      return '正在分析结果...'
    case 'error':
      return '执行出错'
    default:
      return THINKING_STEP_DESCRIPTIONS[status] || 'AI 正在思考...'
  }
}

/** ThinkingProcess component props */
export interface ThinkingProcessProps {
  thinkingSteps: EnhancedThinkingStep[]
  currentStatus: {
    step: number
    status: ThinkingStepStatus
    content: string
  } | null
  isLoading: boolean
  onRetry?: (stepIndex: number) => void
  onFileClick?: (file: ThinkingFileReference) => void
}
