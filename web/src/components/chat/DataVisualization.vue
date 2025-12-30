<template>
  <div class="data-visualization">
    <!-- Tab 切换不同类型的可视化 -->
    <el-tabs v-model="activeTab" type="border-card" class="viz-tabs">
      <!-- CPU 趋势图 -->
      <el-tab-pane label="CPU 趋势" name="cpu">
        <div class="chart-container">
          <div class="chart-header">
            <span class="chart-title">CPU 使用率趋势</span>
            <el-tag type="primary">{{ currentCpu }}%</el-tag>
          </div>
          <div class="sparkline" ref="cpuSparkline"></div>
          <div class="chart-stats">
            <div class="stat-item">
              <span class="stat-label">平均值</span>
              <span class="stat-value">{{ avgCpu }}%</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">峰值</span>
              <span class="stat-value">{{ maxCpu }}%</span>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 内存趋势图 -->
      <el-tab-pane label="内存使用" name="memory">
        <div class="chart-container">
          <div class="chart-header">
            <span class="chart-title">内存使用情况</span>
            <el-tag type="success">{{ currentMemory }}%</el-tag>
          </div>
          <!-- 进度条样式 -->
          <div class="memory-bars">
            <div
              v-for="(item, index) in displayMemoryData"
              :key="index"
              class="memory-bar-wrapper"
            >
              <div class="bar-label">{{ item.time }}</div>
              <el-progress
                :percentage="item.value"
                :color="getMemoryColor(item.value)"
                :show-text="false"
                :stroke-width="8"
              />
              <div class="bar-value">{{ item.value }}%</div>
            </div>
          </div>
          <div class="chart-stats">
            <div class="stat-item">
              <span class="stat-label">已用</span>
              <span class="stat-value">{{ usedMemory }} GB</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">总计</span>
              <span class="stat-value">{{ totalMemory }} GB</span>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 磁盘使用饼图 -->
      <el-tab-pane label="磁盘使用" name="disk">
        <div class="chart-container">
          <div class="chart-header">
            <span class="chart-title">磁盘空间分布</span>
          </div>
          <div class="disk-pie-charts">
            <div
              v-for="(disk, index) in displayDiskData"
              :key="index"
              class="disk-item"
            >
              <div class="disk-name">{{ disk.mount }}</div>
              <!-- 简单饼图可视化 -->
              <div class="pie-chart-wrapper">
                <svg class="pie-chart" viewBox="0 0 36 36">
                  <path
                    :d="getPiePath(disk.usedPercent)"
                    :stroke="getDiskColor(disk.usedPercent)"
                    stroke-width="3"
                    fill="none"
                  />
                  <text x="18" y="21" text-anchor="middle" class="pie-text">
                    {{ disk.usedPercent }}%
                  </text>
                </svg>
              </div>
              <div class="disk-info">
                <div class="disk-usage">
                  已用: {{ formatBytes(disk.used) }}
                </div>
                <div class="disk-total">
                  总计: {{ formatBytes(disk.total) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 进程资源占用 -->
      <el-tab-pane label="Top 进程" name="processes">
        <div class="chart-container">
          <div class="chart-header">
            <span class="chart-title">资源占用 Top 10</span>
          </div>
          <div class="process-list">
            <div
              v-for="(process, index) in displayProcessData"
              :key="process.pid"
              class="process-item"
            >
              <div class="process-rank">{{ index + 1 }}</div>
              <div class="process-info">
                <div class="process-name">{{ process.name }}</div>
                <div class="process-pid">PID: {{ process.pid }}</div>
              </div>
              <div class="process-metrics">
                <div class="metric">
                  <span class="metric-label">CPU</span>
                  <el-progress
                    :percentage="process.cpu"
                    :show-text="true"
                    :stroke-width="6"
                    class="metric-progress"
                  />
                </div>
                <div class="metric">
                  <span class="metric-label">MEM</span>
                  <el-progress
                    :percentage="process.memory"
                    :show-text="true"
                    :stroke-width="6"
                    color="#67c23a"
                    class="metric-progress"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 日志时间线 -->
      <el-tab-pane label="日志分析" name="logs">
        <div class="chart-container">
          <div class="chart-header">
            <span class="chart-title">日志时间线</span>
            <el-tag type="warning">{{ displayLogData.length }} 条</el-tag>
          </div>
          <div class="log-timeline">
            <div
              v-for="(log, index) in displayLogData"
              :key="index"
              class="log-entry"
              :class="`log-${log.level.toLowerCase()}`"
            >
              <div class="log-dot"></div>
              <div class="log-content">
                <div class="log-header-row">
                  <el-tag :type="getLogTagType(log.level)" size="small">
                    {{ log.level }}
                  </el-tag>
                  <span class="log-time">{{ log.time }}</span>
                </div>
                <div class="log-message">{{ log.message }}</div>
              </div>
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

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

const props = defineProps<{
  data?: {
    cpu?: CpuData[]
    memory?: MemoryData[]
    disk?: DiskData[]
    processes?: ProcessData[]
    logs?: LogData[]
  }
}>()

const activeTab = ref('cpu')

// 模拟数据（实际使用时从 props 传入）
const cpuData = ref<CpuData[]>([
  { time: '10:00', value: 35 },
  { time: '10:05', value: 42 },
  { time: '10:10', value: 38 },
  { time: '10:15', value: 55 },
  { time: '10:20', value: 48 },
  { time: '10:25', value: 62 },
  { time: '10:30', value: 45 }
])

const memoryData = ref<MemoryData[]>([
  { time: '10:00', value: 45 },
  { time: '10:05', value: 47 },
  { time: '10:10', value: 48 },
  { time: '10:15', value: 52 },
  { time: '10:20', value: 55 },
  { time: '10:25', value: 53 },
  { time: '10:30', value: 58 }
])

const diskData = ref<DiskData[]>([
  { mount: '/', used: 45 * 1024 * 1024 * 1024, total: 100 * 1024 * 1024 * 1024, usedPercent: 45 },
  { mount: '/home', used: 80 * 1024 * 1024 * 1024, total: 200 * 1024 * 1024 * 1024, usedPercent: 40 },
  { mount: '/var', used: 35 * 1024 * 1024 * 1024, total: 50 * 1024 * 1024 * 1024, usedPercent: 70 }
])

const processData = ref<ProcessData[]>([
  { pid: 1234, name: 'nginx', cpu: 12.5, memory: 8.3 },
  { pid: 5678, name: 'mysql', cpu: 8.2, memory: 45.6 },
  { pid: 9012, name: 'redis', cpu: 5.1, memory: 12.4 },
  { pid: 3456, name: 'node', cpu: 15.8, memory: 22.1 },
  { pid: 7890, name: 'python', cpu: 3.2, memory: 18.7 }
])

const logData = ref<LogData[]>([
  { level: 'ERROR', time: '10:28:35', message: 'Connection timeout to database server' },
  { level: 'WARN', time: '10:27:12', message: 'High memory usage detected: 85%' },
  { level: 'INFO', time: '10:26:45', message: 'User authentication successful' },
  { level: 'ERROR', time: '10:25:30', message: 'Failed to execute query: syntax error' },
  { level: 'INFO', time: '10:24:18', message: 'Cron job completed successfully' }
])

// 计算属性 - 优先使用 props 数据
const currentCpu = computed(() => {
  const data = props.data?.cpu
  if (data && data.length > 0) {
    return data[data.length - 1]?.value || 0
  }
  return cpuData.value[cpuData.value.length - 1]?.value || 0
})

const avgCpu = computed(() => {
  const data = props.data?.cpu
  if (data && data.length > 0) {
    const sum = data.reduce((acc, item) => acc + item.value, 0)
    return (sum / data.length).toFixed(1)
  }
  const sum = cpuData.value.reduce((acc, item) => acc + item.value, 0)
  return (sum / cpuData.value.length).toFixed(1)
})

const maxCpu = computed(() => {
  const data = props.data?.cpu
  if (data && data.length > 0) {
    return Math.max(...data.map(item => item.value))
  }
  return Math.max(...cpuData.value.map(item => item.value))
})

const currentMemory = computed(() => {
  const data = props.data?.memory
  if (data && data.length > 0) {
    return data[data.length - 1]?.value || 0
  }
  return memoryData.value[memoryData.value.length - 1]?.value || 0
})

// 实际使用的数据（优先 props）
const displayMemoryData = computed(() => props.data?.memory || memoryData.value)
const displayDiskData = computed(() => props.data?.disk || diskData.value)
const displayProcessData = computed(() => props.data?.processes || processData.value)
const displayLogData = computed(() => props.data?.logs || logData.value)

const usedMemory = computed(() => {
  return (16 * currentMemory.value / 100).toFixed(1)
})

const totalMemory = ref('16')

// 方法
const getMemoryColor = (percent: number) => {
  if (percent > 80) return '#f56c6c'
  if (percent > 60) return '#e6a23c'
  return '#67c23a'
}

const getDiskColor = (percent: number) => {
  if (percent > 80) return '#f56c6c'
  if (percent > 60) return '#e6a23c'
  return '#67c23a'
}

const formatBytes = (bytes: number) => {
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(2)} GB`
  const mb = bytes / (1024 * 1024)
  return `${mb.toFixed(2)} MB`
}

const getPiePath = (percent: number) => {
  const normalized = percent / 100
  const startX = 18
  const startY = 3
  const endX = 18 + 15 * Math.cos(2 * Math.PI * normalized - Math.PI / 2)
  const endY = 18 + 15 * Math.sin(2 * Math.PI * normalized - Math.PI / 2)
  const largeArc = normalized > 0.5 ? 1 : 0

  return `M ${startX} ${startY} A 15 15 0 ${largeArc} 1 ${endX} ${endY}`
}

const getLogTagType = (level: string) => {
  switch (level) {
    case 'ERROR': return 'danger'
    case 'WARN': return 'warning'
    case 'INFO': return 'info'
    default: return ''
  }
}
</script>

<style scoped>
.data-visualization {
  margin: 12px 20px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.viz-tabs {
  border: none;
}

.viz-tabs :deep(.el-tabs__content) {
  padding: 0;
}

.chart-container {
  padding: 20px;
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.chart-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.chart-stats {
  display: flex;
  gap: 24px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 12px;
  color: #6b7280;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

/* Sparkline */
.sparkline {
  height: 80px;
  display: flex;
  align-items: flex-end;
  gap: 4px;
  padding: 10px 0;
}

/* Memory Bars */
.memory-bars {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.memory-bar-wrapper {
  display: grid;
  grid-template-columns: 60px 1fr 50px;
  align-items: center;
  gap: 12px;
}

.bar-label {
  font-size: 12px;
  color: #6b7280;
}

.bar-value {
  font-size: 13px;
  font-weight: 500;
  color: #1f2937;
  text-align: right;
}

/* Disk Pie Charts */
.disk-pie-charts {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 20px;
}

.disk-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: #f9fafb;
  border-radius: 8px;
}

.disk-name {
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 12px;
}

.pie-chart-wrapper {
  position: relative;
  width: 80px;
  height: 80px;
}

.pie-chart {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.pie-text {
  transform: rotate(90deg);
  font-size: 8px;
  font-weight: 600;
  fill: #1f2937;
}

.disk-info {
  margin-top: 12px;
  text-align: center;
  font-size: 12px;
  color: #6b7280;
}

.disk-usage,
.disk-total {
  margin: 2px 0;
}

/* Process List */
.process-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.process-item {
  display: grid;
  grid-template-columns: 40px 1fr 180px;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f9fafb;
  border-radius: 8px;
  transition: background 0.2s;
}

.process-item:hover {
  background: #f3f4f6;
}

.process-rank {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: #fff;
  border-radius: 50%;
  font-weight: 600;
  font-size: 14px;
}

.process-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.process-name {
  font-weight: 500;
  color: #1f2937;
}

.process-pid {
  font-size: 12px;
  color: #6b7280;
}

.process-metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric {
  display: grid;
  grid-template-columns: 40px 1fr;
  align-items: center;
  gap: 8px;
}

.metric-label {
  font-size: 11px;
  color: #6b7280;
  text-transform: uppercase;
}

.metric-progress :deep(.el-progress__text) {
  font-size: 11px !important;
}

/* Log Timeline */
.log-timeline {
  position: relative;
  padding-left: 30px;
}

.log-timeline::before {
  content: '';
  position: absolute;
  left: 8px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: #e5e7eb;
}

.log-entry {
  position: relative;
  padding-bottom: 16px;
}

.log-dot {
  position: absolute;
  left: -26px;
  top: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #fff;
  border: 2px solid #6b7280;
}

.log-error .log-dot {
  background: #ef4444;
  border-color: #ef4444;
}

.log-warn .log-dot {
  background: #f59e0b;
  border-color: #f59e0b;
}

.log-info .log-dot {
  background: #3b82f6;
  border-color: #3b82f6;
}

.log-content {
  background: #f9fafb;
  padding: 10px 12px;
  border-radius: 6px;
  border-left: 3px solid transparent;
}

.log-error .log-content {
  border-left-color: #ef4444;
  background: #fef2f2;
}

.log-warn .log-content {
  border-left-color: #f59e0b;
  background: #fffbeb;
}

.log-info .log-content {
  border-left-color: #3b82f6;
}

.log-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.log-time {
  font-size: 12px;
  color: #6b7280;
}

.log-message {
  font-size: 13px;
  color: #374151;
  line-height: 1.5;
}
</style>
