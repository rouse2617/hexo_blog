import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElNotification } from 'element-plus'
import type { HostMetrics, MetricAlert } from '@/types/chat-ui'

const REFRESH_INTERVAL = 60000 // 60 seconds
const MAX_CONSECUTIVE_FAILURES = 3
const MAX_HOSTS = 5

export const useMetricsStore = defineStore('metrics', () => {
  // State
  const hosts = ref<Map<string, HostMetrics>>(new Map())
  const lastRefresh = ref(0)
  const isRefreshing = ref(false)
  const refreshError = ref<Error | null>(null)
  const consecutiveFailures = ref(0)
  const subscriptions = ref<Set<string>>(new Set())
  const abnormalHosts = ref<Set<string>>(new Set())
  const alertHistory = ref<MetricAlert[]>([])

  let refreshTimer: number | null = null

  // Computed
  const refreshInterval = computed(() => REFRESH_INTERVAL)

  const hostList = computed(() => {
    return Array.from(hosts.value.values())
      .slice(0, MAX_HOSTS)
      .sort((a, b) => {
        // Sort abnormal hosts first
        const aAbnormal = abnormalHosts.value.has(a.hostId)
        const bAbnormal = abnormalHosts.value.has(b.hostId)
        if (aAbnormal && !bAbnormal) return -1
        if (!aAbnormal && bAbnormal) return 1
        return b.lastUpdated - a.lastUpdated
      })
  })

  const hasAbnormalHosts = computed(() => abnormalHosts.value.size > 0)

  const criticalAlerts = computed(() => {
    return alertHistory.value.filter(alert => !alert.acknowledged)
  })

  // Actions
  function subscribe(hostId: string) {
    subscriptions.value.add(hostId)
    if (subscriptions.value.size === 1) {
      startAutoRefresh()
    }
    // Immediately fetch metrics for newly subscribed host
    fetchMetrics([hostId])
  }

  function unsubscribe(hostId: string) {
    subscriptions.value.delete(hostId)
    hosts.value.delete(hostId)
    abnormalHosts.value.delete(hostId)
    if (subscriptions.value.size === 0) {
      stopAutoRefresh()
    }
  }

  function startAutoRefresh() {
    if (refreshTimer !== null) return

    refreshTimer = window.setInterval(async () => {
      await refreshMetrics()
    }, REFRESH_INTERVAL)

    // Initial refresh
    refreshMetrics()
  }

  function stopAutoRefresh() {
    if (refreshTimer !== null) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
    isRefreshing.value = false
  }

  async function refreshMetrics() {
    if (isRefreshing.value) return

    // Check if we should stop due to consecutive failures
    if (consecutiveFailures.value >= MAX_CONSECUTIVE_FAILURES) {
      console.warn('[Metrics] Stopping auto-refresh due to consecutive failures')
      stopAutoRefresh()
      ElNotification({
        title: 'Metrics Refresh Stopped',
        message: `Stopped after ${MAX_CONSECUTIVE_FAILURES} consecutive failures. Please manually refresh.`,
        type: 'warning',
        duration: 5000
      })
      return
    }

    isRefreshing.value = true
    refreshError.value = null

    try {
      const hostIds = Array.from(subscriptions.value)
      if (hostIds.length === 0) return

      await fetchMetrics(hostIds)

      // Reset failure counter on success
      consecutiveFailures.value = 0
    } catch (error) {
      const err = error as Error
      console.error('[Metrics] Refresh failed:', err)
      refreshError.value = err
      consecutiveFailures.value++

      ElNotification({
        title: 'Metrics Refresh Failed',
        message: `Failed to refresh metrics (${consecutiveFailures.value}/${MAX_CONSECUTIVE_FAILURES})`,
        type: 'error',
        duration: 3000
      })
    } finally {
      isRefreshing.value = false
      lastRefresh.value = Date.now()
    }
  }

  async function fetchMetrics(hostIds: string[]) {
    try {
      // Mock API call - replace with actual API endpoint
      // const response = await request.post('/api/metrics/batch', { hostIds })

      // Simulating metrics data for demo
      const mockMetrics = hostIds.map(hostId => ({
        hostId,
        hostName: hostId,
        cpu: Math.random() * 100,
        memory: Math.random() * 100,
        disk: Math.random() * 100,
        status: ['normal', 'warning', 'critical'][Math.floor(Math.random() * 3)] as 'normal' | 'warning' | 'critical',
        lastUpdated: Date.now()
      }))

      // Update hosts map
      mockMetrics.forEach(metric => {
        hosts.value.set(metric.hostId, metric)
        checkThresholds(metric)
      })
    } catch (error) {
      throw error
    }
  }

  function checkThresholds(host: HostMetrics) {
    const alerts: MetricAlert[] = []
    let hasAbnormal = false

    // CPU threshold check (>80%)
    if (host.cpu > 80) {
      alerts.push({
        hostId: host.hostId,
        metricType: 'cpu',
        value: host.cpu,
        threshold: 80,
        timestamp: Date.now(),
        acknowledged: false
      })
      hasAbnormal = true
    }

    // Memory threshold check (>85%)
    if (host.memory > 85) {
      alerts.push({
        hostId: host.hostId,
        metricType: 'memory',
        value: host.memory,
        threshold: 85,
        timestamp: Date.now(),
        acknowledged: false
      })
      hasAbnormal = true
    }

    // Disk threshold check (>90%)
    if (host.disk > 90) {
      alerts.push({
        hostId: host.hostId,
        metricType: 'disk',
        value: host.disk,
        threshold: 90,
        timestamp: Date.now(),
        acknowledged: false
      })
      hasAbnormal = true
    }

    // Update abnormal hosts
    if (hasAbnormal) {
      abnormalHosts.value.add(host.hostId)
    } else {
      abnormalHosts.value.delete(host.hostId)
    }

    // Add alerts to history
    if (alerts.length > 0) {
      alertHistory.value.push(...alerts)

      // Show notification for critical alerts
      if (host.status === 'critical') {
        ElNotification({
          title: `Critical Alert: ${host.hostName}`,
          message: alerts.map(a => `${a.metricType} at ${a.value.toFixed(1)}%`).join(', '),
          type: 'error',
          duration: 0, // Don't auto-close
        })
      }
    }
  }

  function acknowledgeAlert(alertId: string) {
    const alert = alertHistory.value.find(a => a.hostId === alertId)
    if (alert) {
      alert.acknowledged = true
    }
  }

  function acknowledgeAllAlerts() {
    alertHistory.value.forEach(alert => {
      alert.acknowledged = true
    })
  }

  function clearAlertHistory() {
    alertHistory.value = []
  }

  function updateHost(hostId: string, updates: Partial<HostMetrics>) {
    const existing = hosts.value.get(hostId)
    if (existing) {
      const updated = { ...existing, ...updates, lastUpdated: Date.now() }
      hosts.value.set(hostId, updated)
      checkThresholds(updated)
    }
  }

  function removeHost(hostId: string) {
    hosts.value.delete(hostId)
    abnormalHosts.value.delete(hostId)
    subscriptions.value.delete(hostId)

    // Stop auto-refresh if no more subscriptions
    if (subscriptions.value.size === 0) {
      stopAutoRefresh()
    }
  }

  function reset() {
    stopAutoRefresh()
    hosts.value.clear()
    abnormalHosts.value.clear()
    subscriptions.value.clear()
    alertHistory.value = []
    consecutiveFailures.value = 0
    refreshError.value = null
    lastRefresh.value = 0
    isRefreshing.value = false
  }

  return {
    // State
    hosts,
    lastRefresh,
    isRefreshing,
    refreshError,
    consecutiveFailures,
    subscriptions,
    abnormalHosts,
    alertHistory,

    // Computed
    refreshInterval,
    hostList,
    hasAbnormalHosts,
    criticalAlerts,

    // Actions
    subscribe,
    unsubscribe,
    startAutoRefresh,
    stopAutoRefresh,
    refreshMetrics,
    fetchMetrics,
    acknowledgeAlert,
    acknowledgeAllAlerts,
    clearAlertHistory,
    updateHost,
    removeHost,
    reset
  }
}, {
  persist: {
    key: 'ai-pro-metrics',
    paths: ['subscriptions', 'alertHistory']
  }
})
