/**
 * Audit Log API
 * 
 * Provides interfaces for logging, querying, and exporting command execution audit logs.
 * Implements dangerous command pattern detection for security compliance.
 * 
 * @module api/audit
 * @requirements 10.1, 10.2, 10.3, 10.4
 */

import { request } from './request'
import type {
  AuditLogEntry,
  AuditLogFilters,
  AuditStatistics,
  RiskLevel,
  ExecutionResult,
  AuditMetadata
} from '@/types/chat-ui'

// ============================================================================
// Dangerous Command Patterns (Req 10.4)
// ============================================================================

/**
 * Dangerous command patterns for risk detection
 * These patterns identify potentially destructive operations
 */
export const DANGEROUS_COMMAND_PATTERNS: Array<{
  pattern: RegExp
  riskLevel: RiskLevel
  description: string
  category: string
}> = [
  // Critical - Destructive file operations
  {
    pattern: /\brm\s+(-rf|-r\s+-f|-fr|-f\s+-r)\s+/i,
    riskLevel: 'critical',
    description: '递归强制删除文件',
    category: 'file_deletion'
  },
  {
    pattern: /\brm\s+-rf\s+\/(?:\s|$)/i,
    riskLevel: 'critical',
    description: '删除根目录',
    category: 'file_deletion'
  },
  
  // Critical - Disk operations
  {
    pattern: /\bdd\s+if=/i,
    riskLevel: 'critical',
    description: '磁盘写入操作',
    category: 'disk_operation'
  },
  {
    pattern: /\bmkfs\./i,
    riskLevel: 'critical',
    description: '格式化文件系统',
    category: 'disk_operation'
  },
  {
    pattern: /\bfdisk\s+/i,
    riskLevel: 'high',
    description: '磁盘分区操作',
    category: 'disk_operation'
  },
  
  // Critical - System control
  {
    pattern: /\bshutdown\b/i,
    riskLevel: 'critical',
    description: '系统关机',
    category: 'system_control'
  },
  {
    pattern: /\breboot\b/i,
    riskLevel: 'critical',
    description: '系统重启',
    category: 'system_control'
  },
  {
    pattern: /\bhalt\b/i,
    riskLevel: 'critical',
    description: '系统停止',
    category: 'system_control'
  },
  {
    pattern: /\bpoweroff\b/i,
    riskLevel: 'critical',
    description: '系统断电',
    category: 'system_control'
  },
  {
    pattern: /\binit\s+[06]\b/i,
    riskLevel: 'critical',
    description: '切换运行级别',
    category: 'system_control'
  },
  
  // Critical - Fork bomb
  {
    pattern: /:\(\)\{:\|:&\};:/,
    riskLevel: 'critical',
    description: 'Fork 炸弹',
    category: 'malicious'
  },
  
  // High - Service operations
  {
    pattern: /\bsystemctl\s+(stop|disable|mask)\s+/i,
    riskLevel: 'high',
    description: '停止/禁用系统服务',
    category: 'service_control'
  },
  {
    pattern: /\bservice\s+\S+\s+stop\b/i,
    riskLevel: 'high',
    description: '停止服务',
    category: 'service_control'
  },
  
  // High - Process termination
  {
    pattern: /\bkillall\s+/i,
    riskLevel: 'high',
    description: '终止所有匹配进程',
    category: 'process_control'
  },
  {
    pattern: /\bkill\s+-9\s+/i,
    riskLevel: 'high',
    description: '强制终止进程',
    category: 'process_control'
  },
  {
    pattern: /\bpkill\s+-9\s+/i,
    riskLevel: 'high',
    description: '强制终止匹配进程',
    category: 'process_control'
  },
  
  // High - Network operations
  {
    pattern: /\biptables\s+(-F|-X|--flush)/i,
    riskLevel: 'high',
    description: '清空防火墙规则',
    category: 'network'
  },
  {
    pattern: /\broute\s+(del|delete)\s+/i,
    riskLevel: 'high',
    description: '删除路由',
    category: 'network'
  },
  {
    pattern: /\bip\s+route\s+(del|delete)\s+/i,
    riskLevel: 'high',
    description: '删除 IP 路由',
    category: 'network'
  },
  
  // High - Permission changes
  {
    pattern: /\bchmod\s+(-R\s+)?000\s+/i,
    riskLevel: 'high',
    description: '移除所有权限',
    category: 'permission'
  },
  {
    pattern: /\bchown\s+-R\s+/i,
    riskLevel: 'medium',
    description: '递归更改所有者',
    category: 'permission'
  },
  
  // High - Database operations
  {
    pattern: /\bdrop\s+(database|table|schema)\s+/i,
    riskLevel: 'high',
    description: '删除数据库/表',
    category: 'database'
  },
  {
    pattern: /\btruncate\s+table\s+/i,
    riskLevel: 'high',
    description: '清空数据表',
    category: 'database'
  },
  {
    pattern: /\bdelete\s+from\s+\S+\s*(;|$|where\s+1\s*=\s*1)/i,
    riskLevel: 'high',
    description: '删除所有数据',
    category: 'database'
  },
  
  // Medium - File operations
  {
    pattern: /\brm\s+(-r|-f)\s+/i,
    riskLevel: 'medium',
    description: '删除文件/目录',
    category: 'file_deletion'
  },
  {
    pattern: /\b>\s*\/\S+/i,
    riskLevel: 'medium',
    description: '覆盖文件内容',
    category: 'file_modification'
  },
  
  // Medium - Package operations
  {
    pattern: /\b(apt|yum|dnf)\s+(remove|purge|autoremove)\s+/i,
    riskLevel: 'medium',
    description: '卸载软件包',
    category: 'package'
  },
  
  // Medium - User operations
  {
    pattern: /\buserdel\s+/i,
    riskLevel: 'medium',
    description: '删除用户',
    category: 'user_management'
  },
  {
    pattern: /\bpasswd\s+/i,
    riskLevel: 'medium',
    description: '修改密码',
    category: 'user_management'
  }
]

/**
 * Read-only command patterns (low risk)
 */
export const READ_ONLY_PATTERNS: RegExp[] = [
  /^\s*(ls|ll|dir)\b/i,
  /^\s*cat\s+/i,
  /^\s*head\s+/i,
  /^\s*tail\s+/i,
  /^\s*less\s+/i,
  /^\s*more\s+/i,
  /^\s*grep\s+/i,
  /^\s*find\s+/i,
  /^\s*locate\s+/i,
  /^\s*which\s+/i,
  /^\s*whereis\s+/i,
  /^\s*whoami\b/i,
  /^\s*who\b/i,
  /^\s*w\b/i,
  /^\s*id\b/i,
  /^\s*pwd\b/i,
  /^\s*date\b/i,
  /^\s*uptime\b/i,
  /^\s*hostname\b/i,
  /^\s*uname\b/i,
  /^\s*df\b/i,
  /^\s*du\b/i,
  /^\s*free\b/i,
  /^\s*top\b/i,
  /^\s*htop\b/i,
  /^\s*ps\b/i,
  /^\s*netstat\b/i,
  /^\s*ss\b/i,
  /^\s*ip\s+(addr|link|route)\s*(show)?/i,
  /^\s*ifconfig\b/i,
  /^\s*ping\s+/i,
  /^\s*traceroute\s+/i,
  /^\s*nslookup\s+/i,
  /^\s*dig\s+/i,
  /^\s*host\s+/i,
  /^\s*systemctl\s+status\s+/i,
  /^\s*service\s+\S+\s+status\b/i,
  /^\s*journalctl\b/i,
  /^\s*dmesg\b/i,
  /^\s*history\b/i,
  /^\s*env\b/i,
  /^\s*printenv\b/i,
  /^\s*echo\s+/i,
  /^\s*test\s+/i,
  /^\s*\[\s+/i
]

// ============================================================================
// Risk Assessment Functions
// ============================================================================

/**
 * Detect dangerous command patterns and return risk assessment
 * @param command - The command to analyze
 * @returns Risk assessment result with level, description, and category
 */
export function detectDangerousPattern(command: string): {
  isDangerous: boolean
  riskLevel: RiskLevel
  matchedPatterns: Array<{ description: string; category: string }>
} {
  if (!command || !command.trim()) {
    return {
      isDangerous: false,
      riskLevel: 'low',
      matchedPatterns: []
    }
  }

  const trimmedCommand = command.trim()
  const matchedPatterns: Array<{ description: string; category: string; riskLevel: RiskLevel }> = []

  // Check against dangerous patterns
  for (const { pattern, riskLevel, description, category } of DANGEROUS_COMMAND_PATTERNS) {
    if (pattern.test(trimmedCommand)) {
      matchedPatterns.push({ description, category, riskLevel })
    }
  }

  if (matchedPatterns.length === 0) {
    return {
      isDangerous: false,
      riskLevel: 'low',
      matchedPatterns: []
    }
  }

  // Determine highest risk level
  const riskPriority: Record<RiskLevel, number> = {
    critical: 4,
    high: 3,
    medium: 2,
    low: 1
  }

  const highestRisk = matchedPatterns.reduce((max, current) => {
    return riskPriority[current.riskLevel] > riskPriority[max.riskLevel] ? current : max
  }, matchedPatterns[0])

  return {
    isDangerous: true,
    riskLevel: highestRisk.riskLevel,
    matchedPatterns: matchedPatterns.map(({ description, category }) => ({ description, category }))
  }
}

/**
 * Determine risk level for a command
 * @param command - The command to assess
 * @returns Risk level (low, medium, high, critical)
 */
export function assessCommandRiskLevel(command: string): RiskLevel {
  if (!command || !command.trim()) {
    return 'low'
  }

  const trimmedCommand = command.trim()

  // Check if it's a read-only command
  const isReadOnly = READ_ONLY_PATTERNS.some(pattern => pattern.test(trimmedCommand))
  if (isReadOnly) {
    return 'low'
  }

  // Check dangerous patterns
  const { riskLevel } = detectDangerousPattern(trimmedCommand)
  return riskLevel
}

/**
 * Check if a command requires confirmation before execution
 * @param command - The command to check
 * @returns Whether confirmation is required
 */
export function requiresConfirmation(command: string): boolean {
  const riskLevel = assessCommandRiskLevel(command)
  return riskLevel === 'high' || riskLevel === 'critical'
}

/**
 * Check if a command requires typing "CONFIRM" to execute
 * @param command - The command to check
 * @returns Whether typing confirmation is required
 */
export function requiresTypingConfirmation(command: string): boolean {
  const riskLevel = assessCommandRiskLevel(command)
  return riskLevel === 'critical'
}

// ============================================================================
// API Interfaces
// ============================================================================

/**
 * Log command entry input (without auto-generated fields)
 */
export interface LogCommandInput {
  userId: string
  sessionId: string
  command: string
  targetHosts: string[]
  executionResult: ExecutionResult
  metadata: Omit<AuditMetadata, 'ipAddress' | 'userAgent'>
}

/**
 * Query logs response
 */
export interface QueryLogsResponse {
  logs: AuditLogEntry[]
  total: number
  hasMore: boolean
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * Log a command execution to the audit system
 * @param entry - The audit log entry to record
 * @returns Promise resolving to the created log entry ID
 * @requirements 10.1, 10.2
 */
export async function logCommand(entry: LogCommandInput): Promise<{ id: string }> {
  // Assess risk level based on command
  const riskLevel = assessCommandRiskLevel(entry.command)
  
  const fullEntry = {
    ...entry,
    riskLevel,
    metadata: {
      ...entry.metadata,
      userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined
    }
  }

  return request.post<{ id: string }>('/audit/logs', fullEntry)
}

/**
 * Query audit logs with filters
 * @param filters - Filter criteria for the query
 * @returns Promise resolving to filtered logs and total count
 * @requirements 10.6
 */
export async function queryLogs(filters: AuditLogFilters): Promise<QueryLogsResponse> {
  // Convert Date objects to ISO strings for API
  const apiFilters: Record<string, any> = {
    ...filters,
    timeRange: filters.timeRange
      ? [filters.timeRange[0].toISOString(), filters.timeRange[1].toISOString()]
      : undefined
  }

  return request.get<QueryLogsResponse>('/audit/logs', { params: apiFilters })
}

/**
 * Export audit logs in specified format
 * @param filters - Filter criteria for export
 * @param format - Export format (csv or json)
 * @returns Promise resolving to Blob of exported data
 * @requirements 10.7
 */
export async function exportLogs(
  filters: AuditLogFilters,
  format: 'csv' | 'json'
): Promise<Blob> {
  const apiFilters: Record<string, any> = {
    ...filters,
    format,
    timeRange: filters.timeRange
      ? [filters.timeRange[0].toISOString(), filters.timeRange[1].toISOString()]
      : undefined
  }

  return request.download('/audit/logs/export', { params: apiFilters })
}

/**
 * Get audit statistics for a time range
 * @param timeRange - Start and end dates for statistics
 * @returns Promise resolving to audit statistics
 */
export async function getStatistics(timeRange: [Date, Date]): Promise<AuditStatistics> {
  return request.get<AuditStatistics>('/audit/statistics', {
    params: {
      startTime: timeRange[0].toISOString(),
      endTime: timeRange[1].toISOString()
    }
  })
}

/**
 * Get a single audit log entry by ID
 * @param logId - The ID of the log entry
 * @returns Promise resolving to the log entry
 */
export async function getLogById(logId: string): Promise<AuditLogEntry> {
  return request.get<AuditLogEntry>(`/audit/logs/${logId}`)
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Format audit log entry for display
 * @param entry - The audit log entry
 * @returns Formatted string representation
 */
export function formatAuditLogEntry(entry: AuditLogEntry): string {
  const timestamp = new Date(entry.timestamp).toLocaleString()
  const hosts = entry.targetHosts.join(', ')
  const status = entry.executionResult.status
  
  return `[${timestamp}] ${entry.userId} executed "${entry.command}" on ${hosts} - ${status}`
}

/**
 * Get risk level display configuration
 * @param level - The risk level
 * @returns Display configuration with color and label
 */
export function getRiskLevelDisplay(level: RiskLevel): {
  color: string
  label: string
  icon: string
} {
  const displays: Record<RiskLevel, { color: string; label: string; icon: string }> = {
    low: { color: '#67C23A', label: '低风险', icon: 'CircleCheck' },
    medium: { color: '#E6A23C', label: '中风险', icon: 'Warning' },
    high: { color: '#F56C6C', label: '高风险', icon: 'WarningFilled' },
    critical: { color: '#C45656', label: '极高风险', icon: 'CircleCloseFilled' }
  }
  
  return displays[level]
}

/**
 * Create a local audit log entry (for offline/local storage)
 * @param input - The log input data
 * @returns Complete audit log entry with generated fields
 */
export function createLocalAuditEntry(input: LogCommandInput): AuditLogEntry {
  return {
    id: `local-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    timestamp: Date.now(),
    userId: input.userId,
    sessionId: input.sessionId,
    command: input.command,
    targetHosts: input.targetHosts,
    executionResult: input.executionResult,
    riskLevel: assessCommandRiskLevel(input.command),
    metadata: {
      ...input.metadata,
      userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined
    }
  }
}

// ============================================================================
// Export Default API Object
// ============================================================================

export const auditApi = {
  logCommand,
  queryLogs,
  exportLogs,
  getStatistics,
  getLogById,
  detectDangerousPattern,
  assessCommandRiskLevel,
  requiresConfirmation,
  requiresTypingConfirmation,
  formatAuditLogEntry,
  getRiskLevelDisplay,
  createLocalAuditEntry
}

export default auditApi
