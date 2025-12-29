<template>
  <div class="lvm-panel">
    <div class="panel-toolbar">
      <el-button
        type="primary"
        size="small"
        @click="checkLVM"
        :loading="checking"
        :disabled="!hasHosts"
      >
        <el-icon><Search /></el-icon>
        检查 LVM
      </el-button>
      <el-button size="small" @click="exportData" :disabled="!hasData">
        <el-icon><Download /></el-icon>
        导出
      </el-button>
    </div>

    <div v-if="!hasData" class="empty-state">
      <el-empty description="选择主机并点击检查按钮开始"></el-empty>
    </div>

    <div v-else class="lvm-list">
      <div
        v-for="(item, index) in lvmData"
        :key="index"
        class="lvm-item"
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
          <!-- VG 列表 -->
          <div v-if="item.data.vgs && item.data.vgs.length > 0" class="vgs-section">
            <div class="section-title">卷组 (Volume Groups)</div>
            <div class="vgs-list">
              <div
                v-for="(vg, idx) in item.data.vgs"
                :key="idx"
                class="vg-item"
              >
                <div class="vg-header">
                  <span class="vg-name">{{ vg.name }}</span>
                  <el-tag size="small">{{ vg.lvs }} 个逻辑卷</el-tag>
                </div>
                <div class="vg-stats">
                  <div class="stat">
                    <span class="stat-label">PV 数量:</span>
                    <span class="stat-value">{{ vg.pvs }}</span>
                  </div>
                  <div class="stat">
                    <span class="stat-label">总大小:</span>
                    <span class="stat-value">{{ vg.size }}</span>
                  </div>
                  <div class="stat">
                    <span class="stat-label">可用:</span>
                    <span class="stat-value">{{ vg.free }}</span>
                  </div>
                  <div class="stat">
                    <span class="stat-label">使用率:</span>
                    <span class="stat-value" :class="getUsageClass(vg.usedPercent)">
                      {{ vg.usedPercent }}%
                    </span>
                  </div>
                </div>
                <!-- VG 使用率进度条 -->
                <div class="usage-bar">
                  <el-progress
                    :percentage="parsePercent(vg.usedPercent)"
                    :status="getProgressStatus(vg.usedPercent)"
                    :show-text="false"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- LV 列表 -->
          <div v-if="item.data.lvs && item.data.lvs.length > 0" class="lvs-section">
            <div class="section-title">逻辑卷 (Logical Volumes)</div>
            <div class="lvs-table">
              <div class="table-header">
                <span>LV 路径</span>
                <span>VG</span>
                <span>大小</span>
                <span>数据%</span>
                <span>状态</span>
              </div>
              <div
                v-for="(lv, idx) in item.data.lvs"
                :key="idx"
                class="table-row"
              >
                <span class="lv-path">{{ lv.path }}</span>
                <span class="lv-vg">{{ lv.vg }}</span>
                <span class="lv-size">{{ lv.size }}</span>
                <span class="lv-data" :class="getDataClass(lv.dataPercent)">
                  {{ lv.dataPercent }}%
                </span>
                <span class="lv-attr">
                  <el-tag
                    v-if="lv.attr"
                    size="small"
                    :type="lv.attr.includes('a') ? 'success' : 'info'"
                  >
                    {{ formatAttr(lv.attr) }}
                  </el-tag>
                </span>
              </div>
            </div>
          </div>

          <!-- Thin Pool 状态 -->
          <div v-if="item.data.thinPools && item.data.thinPools.length > 0" class="thin-section">
            <div class="section-title">Thin Pool 状态</div>
            <div
              v-for="(pool, idx) in item.data.thinPools"
              :key="idx"
              class="thin-pool-item"
            >
              <div class="pool-header">
                <span class="pool-name">{{ pool.name }}</span>
                <el-tag
                  :type="pool.dataPercent > 80 ? 'danger' : 'success'"
                  size="small"
                >
                  {{ pool.dataPercent }}% 已使用
                </el-tag>
              </div>
              <div class="pool-details">
                <div class="detail-row">
                  <span class="label">总大小:</span>
                  <span class="value">{{ pool.size }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">已用:</span>
                  <span class="value">{{ pool.used }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">可用:</span>
                  <span class="value">{{ pool.free }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 警告信息 -->
          <div v-if="item.data.warnings && item.data.warnings.length > 0" class="warnings-section">
            <div class="section-title">警告</div>
            <div
              v-for="(warning, idx) in item.data.warnings"
              :key="idx"
              class="warning-item"
            >
              <el-icon class="warning-icon"><WarningFilled /></el-icon>
              <span>{{ warning }}</span>
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
  Search, Download, Monitor, WarningFilled
} from '@element-plus/icons-vue'
import { useConsoleStore } from '@/stores/console'

interface LVMData {
  host: string
  status: 'success' | 'error'
  data?: {
    vgs?: Array<{
      name: string
      pvs: number
      lvs: number
      size: string
      free: string
      usedPercent: string
    }>
    lvs?: Array<{
      path: string
      vg: string
      size: string
      dataPercent: string
      attr?: string
    }>
    thinPools?: Array<{
      name: string
      size: string
      used: string
      free: string
      dataPercent: number
    }>
    warnings?: string[]
  }
  error?: string
}

const props = defineProps<{
  selectedHosts: string[]
}>()

const consoleStore = useConsoleStore()
const checking = ref(false)
const lvmData = ref<LVMData[]>([])

const hasHosts = computed(() => props.selectedHosts.length > 0)
const hasData = computed(() => lvmData.value.length > 0)

// 检查 LVM
const checkLVM = async () => {
  if (!hasHosts.value) {
    ElMessage.warning('请先选择主机')
    return
  }

  checking.value = true
  try {
    const results = await consoleStore.executeOperation(
      'check_disk',
      props.selectedHosts,
      { lvm: true }
    )

    lvmData.value = results.map((r: any) => {
      if (r.status === 'success' && r.result) {
        const vgs = parseVGSOutput(r.result)
        const lvs = parseLVSOutput(r.result)
        const thinPools = parseThinPools(r.result)
        const warnings = checkLVMHealth(vgs, lvs, thinPools)

        return {
          host: r.host,
          status: 'success',
          data: { vgs, lvs, thinPools, warnings }
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

// 解析 vgs 输出
const parseVGSOutput = (output: string) => {
  const vgs: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    if (line.includes('VG ') && !line.startsWith('  VG')) {
      const parts = line.trim().split(/\s+/)
      if (parts.length >= 6) {
        const name = parts[0]
        const pvs = parseInt(parts[1]) || 0
        const lvsCount = parseInt(parts[2]) || 0
        const size = parts[3]
        const free = parts[5]
        const usedPercent = '0'

        vgs.push({ name, pvs, lvs: lvsCount, size, free, usedPercent })
      }
    }
  }

  return vgs
}

// 解析 lvs 输出
const parseLVSOutput = (output: string) => {
  const lvs: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    if (line.includes('/dev/') && !line.includes('LV')) {
      const parts = line.trim().split(/\s+/)
      if (parts.length >= 5) {
        const path = parts[0]
        const vg = parts[1]
        const size = parts[2]
        const dataPercent = parts[3].replace(',', '.')
        const attr = parts[4] || ''

        lvs.push({ path, vg, size, dataPercent, attr })
      }
    }
  }

  return lvs
}

// 解析 thin pool 信息
const parseThinPools = (output: string) => {
  const pools: any[] = []
  const lines = output.split('\n')

  for (const line of lines) {
    if (line.includes('thin') && line.includes('pool')) {
      const parts = line.trim().split(/\s+/)
      if (parts.length >= 3) {
        pools.push({
          name: parts[0],
          size: parts[1],
          used: parts[2],
          free: parts[3] || '0',
          dataPercent: 0
        })
      }
    }
  }

  return pools
}

// 检查 LVM 健康状态
const checkLVMHealth = (vgs: any[], _lvs: any[], thinPools: any[]) => {
  const warnings: string[] = []

  for (const vg of vgs) {
    const percent = parsePercent(vg.usedPercent)
    if (percent >= 90) {
      warnings.push(`卷组 ${vg.name} 空间几乎已满 (${vg.usedPercent}%)`)
    }
  }

  for (const pool of thinPools) {
    if (pool.dataPercent > 80) {
      warnings.push(`Thin Pool ${pool.name} 使用率过高 (${pool.dataPercent}%)`)
    }
  }

  return warnings
}

// 解析百分比
const parsePercent = (value: string | number) => {
  if (typeof value === 'number') return value
  return parseInt(String(value).replace('%', '')) || 0
}

// 获取使用率样式
const getUsageClass = (percent: string | number) => {
  const p = parsePercent(percent)
  if (p >= 90) return 'critical'
  if (p >= 75) return 'warning'
  return 'normal'
}

// 获取数据百分比样式
const getDataClass = (percent: string | number) => {
  const p = parsePercent(percent)
  if (p >= 90) return 'critical'
  if (p >= 75) return 'warning'
  return 'normal'
}

// 获取进度条状态
const getProgressStatus = (percent: string | number) => {
  const p = parsePercent(percent)
  if (p >= 90) return 'exception'
  if (p >= 75) return 'warning'
  return undefined
}

// 格式化属性
const formatAttr = (attr: string) => {
  const attrs: string[] = []
  if (attr.includes('w')) attrs.push('可写')
  if (attr.includes('r')) attrs.push('只读')
  if (attr.includes('a')) attrs.push('活动')
  return attrs.join(' ') || attr
}

// 导出数据
const exportData = () => {
  const data = JSON.stringify(lvmData.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `lvm_${Date.now()}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}
</script>

<style scoped>
.lvm-panel {
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

.lvm-list {
  flex: 1;
  overflow-y: auto;
  padding-right: 8px;
}

.lvm-item {
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

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
}

.vgs-section,
.lvs-section,
.thin-section {
  margin-bottom: 16px;
}

.vgs-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.vg-item {
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.vg-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.vg-name {
  font-weight: 600;
  font-size: 13px;
}

.vg-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
  margin-bottom: 8px;
  font-size: 12px;
}

.stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}

.stat-value {
  font-weight: 600;
  font-size: 12px;
}

.stat-value.normal { color: var(--el-text-color-primary); }
.stat-value.warning { color: #f59e0b; }
.stat-value.critical { color: #ef4444; }

.usage-bar {
  margin-top: 8px;
}

.lvs-table {
  background: var(--el-bg-color);
  border-radius: 6px;
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  gap: 8px;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid var(--el-border-color);
  font-size: 12px;
}

.lv-path {
  font-family: monospace;
  font-weight: 500;
}

.lv-size {
  font-family: monospace;
  color: var(--el-text-color-secondary);
}

.lv-data.normal { color: var(--el-text-color-primary); }
.lv-data.warning { color: #f59e0b; font-weight: 600; }
.lv-data.critical { color: #ef4444; font-weight: 600; }

.thin-pool-item {
  padding: 12px;
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: 6px;
  margin-bottom: 8px;
}

.pool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.pool-name {
  font-weight: 600;
  font-size: 13px;
}

.pool-details {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px 16px;
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
  font-family: monospace;
}

.warnings-section {
  padding: 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
}

.warning-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #dc2626;
  font-size: 12px;
  margin-bottom: 4px;
}

.warning-icon {
  flex-shrink: 0;
}

.error-content {
  padding-left: 32px;
  color: var(--el-color-danger);
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
