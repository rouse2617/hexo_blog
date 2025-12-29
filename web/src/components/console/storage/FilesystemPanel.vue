<template>
  <div class="filesystem-panel">
    <div class="panel-toolbar">
      <el-button
        type="primary"
        size="small"
        @click="checkFilesystem"
        :loading="checking"
        :disabled="!hasHosts"
      >
        <el-icon><Search /></el-icon>
        检查文件系统
      </el-button>
      <el-button size="small" @click="exportData" :disabled="!hasData">
        <el-icon><Download /></el-icon>
        导出
      </el-button>
    </div>

    <div v-if="!hasData" class="empty-state">
      <el-empty description="选择主机并点击检查按钮开始"></el-empty>
    </div>

    <div v-else class="filesystem-list">
      <div
        v-for="(item, index) in filesystemData"
        :key="index"
        class="filesystem-item"
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
          <!-- 挂载点列表 -->
          <div v-if="item.data.mounts" class="mounts-list">
            <div
              v-for="(mount, idx) in item.data.mounts"
              :key="idx"
              class="mount-item"
            >
              <div class="mount-header">
                <span class="mount-point">{{ mount.mount }}</span>
                <el-tag
                  :type="getUsageType(mount.usePercent)"
                  size="small"
                >
                  {{ mount.usePercent }}%
                </el-tag>
              </div>
              <div class="mount-details">
                <div class="detail-row">
                  <span class="label">设备:</span>
                  <span class="value">{{ mount.device }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">类型:</span>
                  <span class="value">{{ mount.type }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">大小:</span>
                  <span class="value">{{ mount.size }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">已用:</span>
                  <span class="value">{{ mount.used }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">可用:</span>
                  <span class="value">{{ mount.avail }}</span>
                </div>
              </div>
              <!-- 使用率进度条 -->
              <div class="usage-bar">
                <el-progress
                  :percentage="parsePercent(mount.usePercent)"
                  :status="getProgressStatus(mount.usePercent)"
                  :show-text="false"
                />
              </div>
            </div>
          </div>

          <!-- 文件系统错误 -->
          <div v-if="item.data.errors && item.data.errors.length > 0" class="errors-section">
            <div class="section-title">检测到的问题</div>
            <div v-for="(error, idx) in item.data.errors" :key="idx" class="error-item">
              <el-icon class="error-icon"><WarningFilled /></el-icon>
              <span>{{ error }}</span>
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

interface FilesystemData {
  host: string
  status: 'success' | 'error'
  data?: {
    mounts?: Array<{
      device: string
      type: string
      size: string
      used: string
      avail: string
      usePercent: string
      mount: string
    }>
    errors?: string[]
  }
  error?: string
}

const props = defineProps<{
  selectedHosts: string[]
}>()

const consoleStore = useConsoleStore()
const checking = ref(false)
const filesystemData = ref<FilesystemData[]>([])

const hasHosts = computed(() => props.selectedHosts.length > 0)
const hasData = computed(() => filesystemData.value.length > 0)

// 检查文件系统
const checkFilesystem = async () => {
  if (!hasHosts.value) {
    ElMessage.warning('请先选择主机')
    return
  }

  checking.value = true
  try {
    const results = await consoleStore.executeOperation(
      'check_disk',
      props.selectedHosts,
      { detail: true }
    )

    filesystemData.value = results.map((r: any) => {
      if (r.status === 'success' && r.result) {
        // 解析 df 输出
        const mounts = parseDfOutput(r.result)
        const errors = detectStorageIssues(mounts)

        return {
          host: r.host,
          status: 'success',
          data: { mounts, errors }
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

// 解析 df 输出
const parseDfOutput = (output: string) => {
  const lines = output.split('\n').filter((l: string) => l.trim())
  const mounts: any[] = []

  // 跳过标题行
  for (let i = 1; i < lines.length; i++) {
    const parts = lines[i].split(/\s+/)
    if (parts.length >= 6) {
      mounts.push({
        device: parts[0],
        size: parts[1],
        used: parts[2],
        avail: parts[3],
        usePercent: parts[4].replace('%', ''),
        mount: parts[5]
      })
    }
  }

  return mounts
}

// 检测存储问题
const detectStorageIssues = (mounts: any[]) => {
  const errors: string[] = []

  for (const mount of mounts) {
    const percent = parsePercent(mount.usePercent)

    if (percent >= 95) {
      errors.push(`${mount.mount} 空间严重不足 (${mount.usePercent}%)`)
    } else if (percent >= 85) {
      errors.push(`${mount.mount} 空间不足 (${mount.usePercent}%)`)
    }

    // 检查只读文件系统
    if (mount.options && mount.options.includes('ro')) {
      errors.push(`${mount.mount} 挂载为只读模式`)
    }
  }

  return errors
}

// 解析百分比
const parsePercent = (value: string | number) => {
  if (typeof value === 'number') return value
  return parseInt(String(value).replace('%', '')) || 0
}

// 获取使用率类型
const getUsageType = (percent: string | number) => {
  const p = parsePercent(percent)
  if (p >= 95) return 'danger'
  if (p >= 85) return 'warning'
  return 'success'
}

// 获取进度条状态
const getProgressStatus = (percent: string | number) => {
  const p = parsePercent(percent)
  if (p >= 95) return 'exception'
  if (p >= 85) return 'warning'
  return undefined
}

// 导出数据
const exportData = () => {
  const data = JSON.stringify(filesystemData.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `filesystem_${Date.now()}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('导出成功')
}
</script>

<style scoped>
.filesystem-panel {
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

.filesystem-list {
  flex: 1;
  overflow-y: auto;
  padding-right: 8px;
}

.filesystem-item {
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

.mounts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mount-item {
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.mount-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.mount-point {
  font-weight: 600;
  font-size: 13px;
}

.mount-details {
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

.usage-bar {
  margin-top: 8px;
}

.errors-section {
  margin-top: 12px;
  padding: 12px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
}

.section-title {
  font-weight: 600;
  color: #dc2626;
  margin-bottom: 8px;
  font-size: 13px;
}

.error-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #dc2626;
  font-size: 12px;
  margin-bottom: 4px;
}

.error-icon {
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
