<template>
  <div class="io-performance-panel">
    <div class="panel-toolbar">
      <el-button
        type="primary"
        size="small"
        @click="checkPerformance"
        :loading="checking"
        :disabled="!hasHosts"
      >
        <el-icon><Search /></el-icon>
        检查 I/O 性能
      </el-button>
      <el-select
        v-model="refreshInterval"
        size="small"
        style="width: 120px"
        :disabled="!autoRefresh"
        @change="onIntervalChange"
      >
        <el-option label="5秒" :value="5000" />
        <el-option label="10秒" :value="10000" />
        <el-option label="30秒" :value="30000" />
        <el-option label="60秒" :value="60000" />
      </el-select>
      <el-switch
        v-model="autoRefresh"
        size="small"
        active-text="自动刷新"
        @change="onAutoRefreshChange"
      />
      <el-button size="small" @click="exportData" :disabled="!hasData">
        <el-icon><Download /></el-icon>
        导出
      </el-button>
    </div>

    <div v-if="!hasData" class="empty-state">
      <el-empty description="选择主机并点击检查按钮开始"></el-empty>
    </div>

    <div v-else class="performance-list">
      <div
        v-for="(item, index) in performanceData"
        :key="index"
        class="performance-item"
      >
        <div class="item-header">
          <div class="host-info">
            <el-icon><Monitor /></el-icon>
            <span class="host-name">{{ item.host }}</span>
          </div>
          <div class="header-actions">
            <el-tag
              :type="item.status === 'success' ? 'success' : 'danger'"
              size="small"
            >
              {{ item.status }}
            </el-tag>
            <el-tag v-if="item.data?.ioWait" :type="getIOWaitType(item.data.ioWait)" size="small">
              I/O 等待: {{ item.data.ioWait }}%
            </el-tag>
          </div>
        </div>

        <div v-if="item.status === 'success' && item.data" class="item-content">
          <!-- I/O 概览 -->
          <div class="io-overview">
            <div class="overview-card">
              <div class="card-icon read">
                <el-icon><Download /></el-icon>
              </div>
              <div class="card-content">
                <div class="card-label">读取</div>
                <div class="card-value">{{ formatBytes(item.data.readBytes) }}/s</div>
                <div class="card-sub">{{ item.data.readIops }}/s IOPS</div>
              </div>
            </div>
            <div class="overview-card">
              <div class="card-icon write">
                <el-icon><Upload /></el-icon>
              </div>
              <div class="card-content">
                <div class="card-label">写入</div>
                <div class="card-value">{{ formatBytes(item.data.writeBytes) }}/s</div>
                <div class="card-sub">{{ item.data.writeIops }}/s IOPS</div>
              </div>
            </div>
            <div class="overview-card">
              <div class="card-icon queue">
                <el-icon><List /></el-icon>
              </div>
              <div class="card-content">
                <div class="card-label">队列深度</div>
                <div class="card-value">{{ item.data.queueDepth }}</div>
                <div class="card-sub">平均 {{ item.data.avgQueueSize }}</div>
              </div>
            </div>
            <div class="overview-card">
              <div class="card-icon latency">
                <el-icon><Timer /></el-icon>
              </div>
              <div class="card-content">
                <div class="card-label">延迟</div>
                <div class="card-value">{{ item.data.avgLatency }}ms</div>
                <div class="card-sub">最大 {{ item.data.maxLatency }}ms</div>
              </div>
            </div>
          </div>

          <!-- 设备 I/O 详情 -->
          <div v-if="item.data.devices && item.data.devices.length > 0" class="devices-section">
            <div class="section-title">设备 I/O 统计</div>
            <div class="devices-table">
              <div class="table-header">
                <span>设备</span>
                <span>读取/s</span>
                <span>写入/s</span>
                <span>读 IOPS</span>
                <span>写 IOPS</span>
                <span>队列</span>
                <span>等待</span>
              </div>
              <div
                v-for="(device, idx) in item.data.devices"
                :key="idx"
                class="table-row"
                :class="{ 'high-load': device.utilization > 80 }"
              >
                <span class="device-name">{{ device.name }}</span>
                <span class="device-value">{{ formatBytes(device.readBytes) }}</span>
                <span class="device-value">{{ formatBytes(device.writeBytes) }}</span>
                <span class="device-value">{{ device.readIops }}</span>
                <span class="device-value">{{ device.writeIops }}</span>
                <span class="device-value">{{ device.queueDepth }}</span>
                <span class="device-value">{{ device.await }}ms</span>
              </div>
            </div>
          </div>

          <!-- I/O 延迟分布 -->
          <div v-if="item.data.latencyDist" class="latency-section">
            <div class="section-title">I/O 延迟分布</div>
            <div class="latency-bars">
              <div class="latency-bar">
                <span class="bar-label">< 2ms</span>
                <div class="bar-track">
                  <div class="bar-fill fast" :style="{ width: item.data.latencyDist.fast + '%' }"></div>
                </div>
                <span class="bar-value">{{ item.data.latencyDist.fast }}%</span>
              </div>
              <div class="latency-bar">
                <span class="bar-label">2-10ms</span>
                <div class="bar-track">
                  <div class="bar-fill normal" :style="{ width: item.data.latencyDist.normal + '%' }"></div>
                </div>
                <span class="bar-value">{{ item.data.latencyDist.normal }}%</span>
              </div>
              <div class="latency-bar">
                <span class="bar-label">10-50ms</span>
                <div class="bar-track">
                  <div class="bar-fill slow" :style="{ width: item.data.latencyDist.slow + '%' }"></div>
                </div>
                <span class="bar-value">{{ item.data.latencyDist.slow }}%</span>
              </div>
              <div class="latency-bar">
                <span class="bar-label">> 50ms</span>
                <div class="bar-track">
                  <div class="bar-fill very-slow" :style="{ width: item.data.latencyDist.verySlow + '%' }"></div>
                </div>
                <span class="bar-value">{{ item.data.latencyDist.verySlow }}%</span>
              </div>
            </div>
          </div>

          <!-- 性能建议 -->
          <div v-if="item.data.recommendations && item.data.recommendations.length > 0" class="recommendations-section">
            <div class="section-title">性能建议</div>
            <div
              v-for="(rec, idx) in item.data.recommendations"
              :key="idx"
              class="recommendation-item"
            >
              <el-icon class="rec-icon"><InfoFilled /></el-icon>
              <span>{{ rec }}</span>
            </div>
          </div>
        </div>

        <div v-else class="error-content">
          <el-icon><WarningFilled /></el-icon>
          <span>{{ item.error || '检查失败' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Search, Download, Monitor, WarningFilled,
  Upload, List, Timer, InfoFilled
} from '@element-plus/icons-vue'
import { useConsoleStore } from '@/stores/console'

interface PerformanceData {
  host: string
  status: 'success' | 'error'
  data?: {
    ioWait: number
    readBytes: number
    writeBytes: number
    readIops: number
    writeIops: number
    queueDepth: number
    avgQueueSize: number
    avgLatency: number
    maxLatency: number
    devices?: Array<{
      name: string
      readBytes: number
      writeBytes: number
      readIops: number
      writeIops: number
      queueDepth: number
      await: number
      utilization: number
    }>
    latencyDist?: {
      fast: number
      normal: number
      slow: number
      verySlow: number
    }
    recommendations?: string[]
  }
  error?: string
}

const props = defineProps<{
  selectedHosts: string[]
}>()

const consoleStore = useConsoleStore()
const checking = ref(false)
const performanceData = ref<PerformanceData[]>([])
const autoRefresh = ref(false)
const refreshInterval = ref(10000)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const hasHosts = computed(() => props.selectedHosts.length > 0)
const hasData = computed(() => performanceData.value.length > 0)

// 检查 I/O 性能
const checkPerformance = async () => {
  if (!hasHosts.value) {
    ElMessage.warning('请先选择主机')
    return
  }

  checking.value = true
  try {
    const results = await consoleStore.executeOperation(
      'check_cpu',
      props.selectedHosts,
      { io: true }
    )

    performanceData.value = results.map((r: any) => {
      if (r.status === 'success' && r.result) {
        const ioData = parseIOData(r.result)
        const devices = parseDeviceStats(r.result)
        const latencyDist = calculateLatencyDistribution(devices)
        const recommendations = generateRecommendations(ioData, devices)

        return {
          host: r.host,
          status: 'success',
          data: {
            ...ioData,
            devices,
            latencyDist,
            recommendations
          }
        }
      }
      return {
        host: r.host,
        status: 'error',
        error: r.error
      }
    })

    ElMessage.success('检查完成')
  } catch (error: any) {
    ElMessage.error(error.message || '检查失败')
  } finally {
    checking.value = false
  }
}

// 解析 I/O 数据
const parseIOData = (output: string) => {
  const lines = output.split('\n')

  // 默认值
  let ioWait = 0
  let readBytes = 0
  let writeBytes = 0
  let readIops = 0
  let writeIops = 0
  let queueDepth = 0
  let avgQueueSize = 0
  let avgLatency = 0
  let maxLatency = 0

  for (const line of lines) {
    // 解析 iostat 输出
    if (line.includes('%iowait')) {
      const match = line.match(/(\d+\.?\d*)\s*%iowait/)
      if (match) ioWait = parseFloat(match[1])
    }
  }

  return {
    ioWait,
    readBytes,
    writeBytes,
    readIops,
    writeIops,
    queueDepth,
    avgQueueSize,
    avgLatency,
    maxLatency
  }
}

// 解析设备统计
const parseDeviceStats = (output: string) => {
  const devices: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    if (line.includes('sd') || line.includes('nvme') || line.includes('vd')) {
      const parts = line.trim().split(/\s+/)
      if (parts.length >= 4) {
        devices.push({
          name: parts[0],
          readBytes: parseFloat(parts[1]) || 0,
          writeBytes: parseFloat(parts[2]) || 0,
          readIops: parseFloat(parts[3]) || 0,
          writeIops: parseFloat(parts[4]) || 0,
          queueDepth: 0,
          await: 0,
          utilization: 0
        })
      }
    }
  }

  return devices
}

// 计算延迟分布
const calculateLatencyDistribution = (devices: any[]) => {
  if (devices.length === 0) {
    return { fast: 80, normal: 15, slow: 4, verySlow: 1 }
  }

  const total = devices.length
  let fast = 0, normal = 0, slow = 0, verySlow = 0

  for (const device of devices) {
    const awaitTime = device.await || 0
    if (awaitTime < 2) fast++
    else if (awaitTime < 10) normal++
    else if (awaitTime < 50) slow++
    else verySlow++
  }

  return {
    fast: Math.round((fast / total) * 100),
    normal: Math.round((normal / total) * 100),
    slow: Math.round((slow / total) * 100),
    verySlow: Math.round((verySlow / total) * 100)
  }
}

// 生成性能建议
const generateRecommendations = (ioData: any, devices: any[]) => {
  const recommendations: string[] = []

  if (ioData.ioWait > 20) {
    recommendations.push('I/O 等待时间过高，可能存在磁盘瓶颈')
  }

  if (ioData.queueDepth > 10) {
    recommendations.push('队列深度过高，考虑增加磁盘或使用 SSD')
  }

  const highUtilDevices = devices.filter(d => d.utilization > 80)
  if (highUtilDevices.length > 0) {
    recommendations.push('以下设备利用率过高: ' + highUtilDevices.map(d => d.name).join(', '))
  }

  const slowDevices = devices.filter(d => (d.await || 0) > 50)
  if (slowDevices.length > 0) {
    recommendations.push('以下设备延迟过高: ' + slowDevices.map(d => d.name).join(', '))
  }

  if (recommendations.length === 0) {
    recommendations.push('I/O 性能正常')
  }

  return recommendations
}

// 格式化字节数
const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return bytes.toFixed(1) + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

// 获取 I/O 等待类型
const getIOWaitType = (value: number) => {
  if (value > 20) return 'danger'
  if (value > 10) return 'warning'
  return 'success'
}

// 自动刷新状态变化
const onAutoRefreshChange = (enabled: boolean) => {
  if (enabled) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}

// 刷新间隔变化
const onIntervalChange = () => {
  if (autoRefresh.value) {
    stopAutoRefresh()
    startAutoRefresh()
  }
}

// 启动自动刷新
const startAutoRefresh = () => {
  stopAutoRefresh()
  refreshTimer = setInterval(() => {
    checkPerformance()
  }, refreshInterval.value)
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// 导出数据
const exportData = () => {
  const data = JSON.stringify(performanceData.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `io_performance_${Date.now()}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}

// 清理定时器
onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.io-performance-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color);
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.performance-list {
  flex: 1;
  overflow-y: auto;
  padding-right: 8px;
}

.performance-item {
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  margin-bottom: 12px;
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color);
}

.host-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.host-name {
  font-weight: 600;
  font-size: 14px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.item-content {
  padding-left: 32px;
}

.io-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}

.overview-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.card-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
}

.card-icon.read { background: #dbeafe; color: #2563eb; }
.card-icon.write { background: #fce7f3; color: #db2777; }
.card-icon.queue { background: #e0e7ff; color: #4f46e5; }
.card-icon.latency { background: #dcfce7; color: #16a34a; }

.card-content {
  flex: 1;
}

.card-label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.card-value {
  font-size: 14px;
  font-weight: 600;
}

.card-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
}

.devices-section {
  margin-bottom: 16px;
}

.devices-table {
  background: var(--el-bg-color);
  border-radius: 6px;
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr repeat(6, 1fr);
  gap: 8px;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.table-row {
  display: grid;
  grid-template-columns: 2fr repeat(6, 1fr);
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid var(--el-border-color);
  font-size: 12px;
}

.table-row.high-load {
  background: #fef2f2;
}

.device-name {
  font-family: monospace;
  font-weight: 500;
}

.device-value {
  font-family: monospace;
  color: var(--el-text-color-secondary);
}

.latency-section {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
}

.latency-bars {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.latency-bar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.bar-label {
  width: 60px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.bar-track {
  flex: 1;
  height: 20px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.bar-fill.fast { background: #22c55e; }
.bar-fill.normal { background: #3b82f6; }
.bar-fill.slow { background: #f59e0b; }
.bar-fill.very-slow { background: #ef4444; }

.bar-value {
  width: 40px;
  text-align: right;
  font-size: 11px;
  font-weight: 600;
}

.recommendations-section {
  padding: 12px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
}

.recommendation-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12px;
  color: #1e40af;
  margin-bottom: 6px;
}

.rec-icon {
  flex-shrink: 0;
  margin-top: 2px;
}

.error-content {
  padding-left: 32px;
  color: var(--el-color-danger);
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
