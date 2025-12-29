<template>
  <div class="raid-panel">
    <div class="panel-toolbar">
      <el-button
        type="primary"
        size="small"
        @click="checkRAID"
        :loading="checking"
        :disabled="!hasHosts"
      >
        <el-icon><Search /></el-icon>
        检查 RAID
      </el-button>
      <el-button size="small" @click="exportData" :disabled="!hasData">
        <el-icon><Download /></el-icon>
        导出
      </el-button>
    </div>

    <div v-if="!hasData" class="empty-state">
      <el-empty description="选择主机并点击检查按钮开始"></el-empty>
    </div>

    <div v-else class="raid-list">
      <div
        v-for="(item, index) in raidData"
        :key="index"
        class="raid-item"
      >
        <div class="item-header">
          <div class="host-info">
            <el-icon><Monitor /></el-icon>
            <span class="host-name">{{ item.host }}</span>
          </div>
          <el-tag
            :type="item.status === 'success' ? 'success' : 'danger'"
            size="small"
          >
            {{ item.status }}
          </el-tag>
        </div>

        <div v-if="item.status === 'success' && item.data" class="item-content">
          <!-- RAID 概览 -->
          <div v-if="item.data.summary" class="raid-summary">
            <div class="summary-stat">
              <span class="stat-label">RAID 阵列数:</span>
              <span class="stat-value">{{ item.data.summary.arrayCount }}</span>
            </div>
            <div class="summary-stat">
              <span class="stat-label">状态:</span>
              <span class="stat-value" :class="item.data.summary.healthClass">
                {{ item.data.summary.health }}
              </span>
            </div>
          </div>

          <!-- RAID 阵列列表 -->
          <div v-if="item.data.arrays && item.data.arrays.length > 0" class="arrays-list">
            <div
              v-for="(array, idx) in item.data.arrays"
              :key="idx"
              class="array-item"
            >
              <div class="array-header">
                <span class="array-name">{{ array.device }}</span>
                <el-tag
                  :type="array.status === 'clean' || array.status === 'active' ? 'success' : 'warning'"
                  size="small"
                >
                  {{ array.status }}
                </el-tag>
              </div>
              <div class="array-details">
                <div class="detail-row">
                  <span class="label">RAID 级别:</span>
                  <span class="value">{{ array.level }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">设备数量:</span>
                  <span class="value">{{ array.devices }} / {{ array.totalDevices }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">块大小:</span>
                  <span class="value">{{ array.chunkSize }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">大小:</span>
                  <span class="value">{{ array.size }}</span>
                </div>
              </div>

              <!-- 成员设备 -->
              <div v-if="array.members" class="members-section">
                <div class="section-title">成员设备</div>
                <div class="members-grid">
                  <div
                    v-for="(member, midx) in array.members"
                    :key="midx"
                    class="member-item"
                    :class="getMemberClass(member.state)"
                  >
                    <span class="member-device">{{ member.device }}</span>
                    <span class="member-state">{{ member.state }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- SMART 状态 -->
          <div v-if="item.data.smartStatus" class="smart-section">
            <div class="section-title">磁盘 SMART 状态</div>
            <div class="smart-list">
              <div
                v-for="(disk, idx) in item.data.smartStatus"
                :key="idx"
                class="smart-item"
              >
                <div class="disk-info">
                  <el-icon><Coin /></el-icon>
                  <span class="disk-device">{{ disk.device }}</span>
                </div>
                <div class="disk-health">
                  <el-tag
                    :type="disk.health === 'PASSED' ? 'success' : 'danger'"
                    size="small"
                  >
                    {{ disk.health }}
                  </el-tag>
                </div>
                <div v-if="disk.temp" class="disk-temp">
                  温度: {{ disk.temp }}°C
                </div>
              </div>
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
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Search, Download, Monitor, WarningFilled, Coin
} from '@element-plus/icons-vue'
import { useConsoleStore } from '@/stores/console'

interface RAIDData {
  host: string
  status: 'success' | 'error'
  data?: {
    summary?: {
      arrayCount: number
      health: string
      healthClass: string
    }
    arrays?: Array<{
      device: string
      level: string
      status: string
      devices: number
      totalDevices: number
      chunkSize: string
      size: string
      members?: Array<{
        device: string
        state: string
      }>
    }>
    smartStatus?: Array<{
      device: string
      health: string
      temp?: number
    }>
  }
  error?: string
}

const props = defineProps<{
  selectedHosts: string[]
}>()

const consoleStore = useConsoleStore()
const checking = ref(false)
const raidData = ref<RAIDData[]>([])

const hasHosts = computed(() => props.selectedHosts.length > 0)
const hasData = computed(() => raidData.value.length > 0)

// 检查 RAID
const checkRAID = async () => {
  if (!hasHosts.value) {
    ElMessage.warning('请先选择主机')
    return
  }

  checking.value = true
  try {
    const results = await consoleStore.executeOperation(
      'check_disk',
      props.selectedHosts,
      { raid: true, smart: true }
    )

    raidData.value = results.map((r: any) => {
      if (r.status === 'success' && r.result) {
        const arrays = parseMDStat(r.result)
        const summary = createRAIDSummary(arrays)
        const smartStatus = parseSMARTStatus(r.result)

        return {
          host: r.host,
          status: 'success',
          data: { summary, arrays, smartStatus }
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

// 解析 /proc/mdstat
const parseMDStat = (output: string) => {
  const arrays: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    // 匹配 md 设备行
    const mdMatch = line.match(/^(\w+)\s*:\s+(.+)$/)
    if (mdMatch) {
      const device = mdMatch[1]
      const info = mdMatch[2]

      // 解析 RAID 级别和状态
      const levelMatch = info.match(/(\w+)\s+raid(\w+)/)
      const level = levelMatch ? `RAID${levelMatch[2].toUpperCase()}` : 'Unknown'

      const statusMatch = info.match(/\[(\w+)\]/)
      const status = statusMatch ? statusMatch[1] : 'unknown'

      arrays.push({
        device: `/dev/${device}`,
        level,
        status,
        devices: 0,
        totalDevices: 0,
        chunkSize: 'unknown',
        size: 'unknown',
        members: []
      })
    }
  }

  return arrays
}

// 创建 RAID 概要
const createRAIDSummary = (arrays: any[]) => {
  const hasIssues = arrays.some(a => a.status !== 'clean' && a.status !== 'active')

  return {
    arrayCount: arrays.length,
    health: hasIssues ? '警告' : '正常',
    healthClass: hasIssues ? 'warning' : 'healthy'
  }
}

// 解析 SMART 状态
const parseSMARTStatus = (output: string) => {
  const disks: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    if (line.includes('SMART overall-health self-assessment')) {
      const healthMatch = line.match(/test result:\s*(\w+)/)
      if (healthMatch) {
        disks.push({
          device: 'unknown',
          health: healthMatch[1]
        })
      }
    }
  }

  return disks
}

// 获取成员设备状态样式
const getMemberClass = (state: string) => {
  if (state === 'active' || state === 'sync') return 'member-active'
  if (state === 'failed' || state === 'faulty') return 'member-failed'
  if (state === 'spare') return 'member-spare'
  return 'member-unknown'
}

// 导出数据
const exportData = () => {
  const data = JSON.stringify(raidData.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `raid_${Date.now()}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}
</script>

<style scoped>
.raid-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-toolbar {
  display: flex;
  gap: 8px;
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

.raid-list {
  flex: 1;
  overflow-y: auto;
  padding-right: 8px;
}

.raid-item {
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

.item-content {
  padding-left: 32px;
}

.raid-summary {
  display: flex;
  gap: 24px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
}

.summary-stat {
  display: flex;
  gap: 8px;
}

.stat-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.stat-value {
  font-weight: 600;
  font-size: 13px;
}

.stat-value.healthy { color: #22c55e; }
.stat-value.warning { color: #f59e0b; }
.stat-value.critical { color: #ef4444; }

.arrays-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.array-item {
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.array-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.array-name {
  font-weight: 600;
  font-size: 13px;
}

.array-details {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 4px 16px;
  margin-bottom: 8px;
  font-size: 12px;
}

.detail-row {
  display: flex;
  gap: 8px;
}

.detail-row .label {
  color: var(--el-text-color-secondary);
}

.detail-row .value {
  color: var(--el-text-color-regular);
  font-family: monospace;
}

.members-section {
  margin-top: 8px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
}

.member-item {
  display: flex;
  flex-direction: column;
  padding: 6px 8px;
  border-radius: 4px;
  font-size: 11px;
}

.member-item.member-active {
  background: #dcfce7;
  color: #166534;
}

.member-item.member-failed {
  background: #fee2e2;
  color: #991b1b;
}

.member-item.member-spare {
  background: #e0e7ff;
  color: #3730a3;
}

.member-item.member-unknown {
  background: #f3f4f6;
  color: #6b7280;
}

.member-device {
  font-family: monospace;
  font-weight: 600;
}

.member-state {
  font-size: 10px;
  opacity: 0.8;
}

.smart-section {
  margin-top: 12px;
  padding: 12px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 6px;
}

.smart-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.smart-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  background: var(--el-bg-color);
  border-radius: 4px;
  font-size: 12px;
}

.disk-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.disk-device {
  font-family: monospace;
  font-weight: 600;
}

.disk-health {
  flex-shrink: 0;
}

.disk-temp {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.error-content {
  padding-left: 32px;
  color: var(--el-color-danger);
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
