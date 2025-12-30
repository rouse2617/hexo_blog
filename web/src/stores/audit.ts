/**
 * Audit Log Store - Audit Log State Management
 *
 * Manages audit log state including logs, filters, statistics, and loading states.
 * Provides actions for querying, filtering, exporting, and managing audit logs.
 *
 * @module stores/audit
 * @requirements 10.1, 10.2, 10.3, 10.6, 10.7
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  AuditLogEntry,
  AuditLogFilters,
  AuditStatistics,
  AuditLogState,
  RiskLevel
} from '@/types/chat-ui'
import * as auditApi from '@/api/audit'

// ============================================================================
// Constants
// ============================================================================

/** Default time range for queries (last 7 days) */
const DEFAULT_TIME_RANGE_DAYS = 7

/** Default page size for queries */
const DEFAULT_PAGE_SIZE = 50

/** Maximum cached logs in memory */
const MAX_CACHED_LOGS = 1000

// ============================================================================
// Store Definition
// ============================================================================

export const useAuditStore = defineStore('audit', () => {
  // ============================================================================
  // State (Req 10.6)
  // ============================================================================

  /** All loaded audit logs */
  const logs = ref<AuditLogEntry[]>([])

  /** Filtered logs based on current filters */
  const filteredLogs = ref<AuditLogEntry[]>([])

  /** Current active filters */
  const filters = ref<AuditLogFilters>({
    limit: DEFAULT_PAGE_SIZE,
    offset: 0
  })

  /** Loading state for async operations */
  const loading = ref(false)

  /** Error state for failed operations */
  const error = ref<Error | null>(null)

  /** Audit statistics for dashboard */
  const statistics = ref<AuditStatistics>({
    totalCommands: 0,
    highRiskCommands: 0,
    failureRate: 0,
    mostActiveHosts: []
  })

  /** Total count of logs (for pagination) */
  const totalCount = ref(0)

  /** Whether there are more logs to load */
  const hasMore = ref(false)

  // ============================================================================
  // Computed Properties
  // ============================================================================

  /** Whether any logs are currently loaded */
  const hasLogs = computed(() => logs.value.length > 0)

  /** Number of logs in current filtered view */
  const filteredCount = computed(() => filteredLogs.value.length)

  /** Whether logs are currently being loaded */
  const isLoading = computed(() => loading.value)

  /** Whether there's an error */
  const hasError = computed(() => error.value !== null)

  /** Recent logs (last 10) */
  const recentLogs = computed(() => {
    return [...logs.value]
      .sort((a, b) => b.timestamp - a.timestamp)
      .slice(0, 10)
  })

  /** High risk logs */
  const highRiskLogs = computed(() => {
    return logs.value.filter(log =>
      log.riskLevel === 'high' || log.riskLevel === 'critical'
    )
  })

  /** Failed command logs */
  const failedLogs = computed(() => {
    return logs.value.filter(log =>
      log.executionResult.status === 'failure'
    )
  })

  /** Current filter summary for display */
  const filterSummary = computed(() => {
    const summary: string[] = []

    if (filters.value.timeRange) {
      const [start, end] = filters.value.timeRange
      summary.push(`${start.toLocaleDateString()} - ${end.toLocaleDateString()}`)
    }

    if (filters.value.userId) {
      summary.push(`用户: ${filters.value.userId}`)
    }

    if (filters.value.riskLevel) {
      summary.push(`风险等级: ${filters.value.riskLevel}`)
    }

    if (filters.value.host) {
      summary.push(`主机: ${filters.value.host}`)
    }

    if (filters.value.commandPattern) {
      summary.push(`命令模式: ${filters.value.commandPattern}`)
    }

    return summary.length > 0 ? summary.join(' | ') : '所有日志'
  })

  // ============================================================================
  // Actions
  // ============================================================================

  /**
   * Add a new audit log entry
   * @param entry - The log entry to add
   */
  async function addLog(entry: Omit<AuditLogEntry, 'id' | 'timestamp' | 'riskLevel'>): Promise<void> {
    try {
      loading.value = true
      error.value = null

      // Call API to log the command
      const result = await auditApi.logCommand({
        userId: entry.userId,
        sessionId: entry.sessionId,
        command: entry.command,
        targetHosts: entry.targetHosts,
        executionResult: entry.executionResult,
        metadata: {
          source: entry.metadata.source,
          triggeredBy: entry.metadata.triggeredBy,
          relatedToolCall: entry.metadata.relatedToolCall
        }
      })

      // Add to local logs
      const newLog: AuditLogEntry = {
        ...entry,
        id: result.id,
        timestamp: Date.now(),
        riskLevel: auditApi.assessCommandRiskLevel(entry.command)
      }

      logs.value.unshift(newLog)

      // Enforce max cache size
      if (logs.value.length > MAX_CACHED_LOGS) {
        logs.value = logs.value.slice(0, MAX_CACHED_LOGS)
      }

      // Reapply filters
      applyFilters()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error('Failed to add log')
      throw error.value
    } finally {
      loading.value = false
    }
  }

  /**
   * Query audit logs with current filters (Req 10.6)
   * @param append - Whether to append results to existing logs (for pagination)
   */
  async function queryLogs(append: boolean = false): Promise<void> {
    try {
      loading.value = true
      error.value = null

      const response = await auditApi.queryLogs(filters.value)

      if (append) {
        // Append for pagination
        logs.value = [...logs.value, ...response.logs]
      } else {
        // Replace for new query
        logs.value = response.logs
      }

      totalCount.value = response.total
      hasMore.value = response.hasMore

      // Apply filters to get filtered logs
      applyFilters()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error('Failed to query logs')
      throw error.value
    } finally {
      loading.value = false
    }
  }

  /**
   * Export audit logs in specified format (Req 10.7)
   * @param format - Export format (csv or json)
   * @param customFilters - Optional custom filters for export
   * @returns Blob of exported data
   */
  async function exportLogs(
    format: 'csv' | 'json' = 'csv',
    customFilters?: AuditLogFilters
  ): Promise<Blob> {
    try {
      loading.value = true
      error.value = null

      const exportFilters = customFilters || filters.value
      return await auditApi.exportLogs(exportFilters, format)
    } catch (err) {
      error.value = err instanceof Error ? err : new Error('Failed to export logs')
      throw error.value
    } finally {
      loading.value = false
    }
  }

  /**
   * Load audit statistics for current time range
   * @param timeRange - Optional custom time range
   */
  async function loadStatistics(timeRange?: [Date, Date]): Promise<void> {
    try {
      loading.value = true
      error.value = null

      const range = timeRange || filters.value.timeRange || getDefaultTimeRange()
      statistics.value = await auditApi.getStatistics(range)
    } catch (err) {
      error.value = err instanceof Error ? err : new Error('Failed to load statistics')
      throw error.value
    } finally {
      loading.value = false
    }
  }

  /**
   * Clear all logs from local state
   */
  function clearLogs(): void {
    logs.value = []
    filteredLogs.value = []
    totalCount.value = 0
    hasMore.value = false
    error.value = null
  }

  /**
   * Apply current filters to logs
   * @param newFilters - New filters to apply (optional)
   */
  function applyFilters(newFilters?: AuditLogFilters): void {
    if (newFilters) {
      filters.value = { ...filters.value, ...newFilters }
    }

    let result = [...logs.value]

    // Apply time range filter
    if (filters.value.timeRange) {
      const [start, end] = filters.value.timeRange
      result = result.filter(log => {
        const timestamp = new Date(log.timestamp)
        return timestamp >= start && timestamp <= end
      })
    }

    // Apply user ID filter
    if (filters.value.userId) {
      result = result.filter(log =>
        log.userId === filters.value.userId
      )
    }

    // Apply risk level filter
    if (filters.value.riskLevel) {
      result = result.filter(log =>
        log.riskLevel === filters.value.riskLevel
      )
    }

    // Apply host filter
    if (filters.value.host) {
      result = result.filter(log =>
        log.targetHosts.includes(filters.value.host!)
      )
    }

    // Apply command pattern filter
    if (filters.value.commandPattern) {
      const pattern = new RegExp(filters.value.commandPattern, 'i')
      result = result.filter(log =>
        pattern.test(log.command)
      )
    }

    // Apply limit and offset
    const offset = filters.value.offset || 0
    const limit = filters.value.limit || result.length

    filteredLogs.value = result.slice(offset, offset + limit)
  }

  /**
   * Update filters and refresh logs
   * @param newFilters - New filter values
   * @param autoQuery - Whether to automatically query after updating filters
   */
  async function updateFilters(newFilters: Partial<AuditLogFilters>, autoQuery: boolean = true): Promise<void> {
    // Reset offset when filters change
    filters.value = {
      ...filters.value,
      ...newFilters,
      offset: 0
    }

    applyFilters()

    if (autoQuery) {
      await queryLogs()
    }
  }

  /**
   * Reset filters to default values
   */
  function resetFilters(): void {
    filters.value = {
      limit: DEFAULT_PAGE_SIZE,
      offset: 0,
      timeRange: getDefaultTimeRange()
    }

    applyFilters()
  }

  /**
   * Load more logs (pagination)
   */
  async function loadMore(): Promise<void> {
    if (!hasMore.value || loading.value) {
      return
    }

    filters.value.offset = logs.value.length
    await queryLogs(true)
  }

  /**
   * Refresh logs with current filters
   */
  async function refresh(): Promise<void> {
    filters.value.offset = 0
    await queryLogs()
    await loadStatistics()
  }

  /**
   * Get a specific log entry by ID
   * @param logId - The log entry ID
   * @returns The log entry or undefined if not found
   */
  function getLogById(logId: string): AuditLogEntry | undefined {
    return logs.value.find(log => log.id === logId)
  }

  /**
   * Get logs by risk level
   * @param riskLevel - The risk level to filter by
   * @returns Array of log entries with the specified risk level
   */
  function getLogsByRiskLevel(riskLevel: RiskLevel): AuditLogEntry[] {
    return logs.value.filter(log => log.riskLevel === riskLevel)
  }

  /**
   * Get logs for a specific host
   * @param host - The host name
   * @returns Array of log entries for the specified host
   */
  function getLogsByHost(host: string): AuditLogEntry[] {
    return logs.value.filter(log =>
      log.targetHosts.includes(host)
    )
  }

  /**
   * Get logs for a specific session
   * @param sessionId - The session ID
   * @returns Array of log entries for the specified session
   */
  function getLogsBySession(sessionId: string): AuditLogEntry[] {
    return logs.value.filter(log =>
      log.sessionId === sessionId
    )
  }

  /**
   * Get logs for a specific user
   * @param userId - The user ID
   * @returns Array of log entries for the specified user
   */
  function getLogsByUser(userId: string): AuditLogEntry[] {
    return logs.value.filter(log =>
      log.userId === userId
    )
  }

  // ============================================================================
  // Utility Functions
  // ============================================================================

  /**
   * Get default time range (last N days)
   * @returns Date range [start, end]
   */
  function getDefaultTimeRange(): [Date, Date] {
    const end = new Date()
    const start = new Date()
    start.setDate(start.getDate() - DEFAULT_TIME_RANGE_DAYS)
    return [start, end]
  }

  /**
   * Set time range preset
   * @param preset - Preset name (today, week, month, custom)
   * @param customRange - Custom date range if preset is 'custom'
   */
  async function setTimeRangePreset(
    preset: 'today' | 'week' | 'month' | 'custom',
    customRange?: [Date, Date]
  ): Promise<void> {
    const now = new Date()
    const start = new Date()

    switch (preset) {
      case 'today':
        start.setHours(0, 0, 0, 0)
        break
      case 'week':
        start.setDate(now.getDate() - 7)
        break
      case 'month':
        start.setDate(now.getDate() - 30)
        break
      case 'custom':
        if (!customRange) {
          throw new Error('Custom range is required for custom preset')
        }
        await updateFilters({ timeRange: customRange })
        return
    }

    await updateFilters({ timeRange: [start, now] })
  }

  /**
   * Get current state (for debugging/testing)
   * @returns Current store state
   */
  function getState(): AuditLogState {
    return {
      logs: logs.value,
      filteredLogs: filteredLogs.value,
      filters: filters.value,
      loading: loading.value,
      error: error.value || undefined,
      statistics: statistics.value
    }
  }

  // ============================================================================
  // Initialization
  // ============================================================================

  /**
   * Initialize the audit store with default time range
   */
  async function init(): Promise<void> {
    resetFilters()
    try {
      await queryLogs()
      await loadStatistics()
    } catch (err) {
      console.error('Failed to initialize audit store:', err)
      // Don't throw - allow app to continue even if init fails
    }
  }

  // ============================================================================
  // Return Store Interface
  // ============================================================================

  return {
    // State
    logs,
    filteredLogs,
    filters,
    loading,
    error,
    statistics,
    totalCount,
    hasMore,

    // Computed
    hasLogs,
    filteredCount,
    isLoading,
    hasError,
    recentLogs,
    highRiskLogs,
    failedLogs,
    filterSummary,

    // Actions
    addLog,
    queryLogs,
    exportLogs,
    loadStatistics,
    clearLogs,
    applyFilters,
    updateFilters,
    resetFilters,
    loadMore,
    refresh,
    getLogById,
    getLogsByRiskLevel,
    getLogsByHost,
    getLogsBySession,
    getLogsByUser,
    setTimeRangePreset,
    init,
    getState
  }
}, {
  persist: {
    key: 'ai-pro-audit',
    paths: ['filters'] // Persist filter settings only
  }
})
