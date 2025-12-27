<template>
  <div class="result-overview">
    <!-- 状态统计仪表盘 -->
    <div class="status-dashboard">
      <div class="stat-item success">
        <div class="stat-value">{{ successCount }}</div>
        <div class="stat-label">成功</div>
      </div>
      <div class="stat-item error">
        <div class="stat-value">{{ errorCount }}</div>
        <div class="stat-label">失败</div>
      </div>
      <div class="stat-item total">
        <div class="stat-value">{{ results.length }}</div>
        <div class="stat-label">总计</div>
      </div>
      <div class="stat-item elapsed">
        <div class="stat-value">{{ averageElapsed }}</div>
        <div class="stat-label">平均耗时</div>
      </div>
    </div>
    
    <div class="overview-header">
      <div class="header-left">
        <el-select
          v-model="statusFilter"
          placeholder="状态筛选"
          clearable
          style="width: 120px; margin-right: 10px"
        >
          <el-option label="全部" value="" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="error" />
        </el-select>
        <el-button
          v-if="errorCount > 0"
          type="danger"
          size="small"
          @click="handleRetryAllFailed"
          style="margin-left: 10px"
        >
          重试所有失败项 ({{ errorCount }})
        </el-button>
        <span class="result-count">
          共 {{ filteredResults.length }} 条结果
        </span>
      </div>
      <div class="header-right">
        <el-button size="small" @click="handleCopySelected" :disabled="selectedRows.length === 0">
          复制选中
        </el-button>
        <el-button size="small" @click="handleExportSelected" :disabled="selectedRows.length === 0">
          导出选中
        </el-button>
      </div>
    </div>

    <el-table
      :data="filteredResults"
      stripe
      @selection-change="handleSelectionChange"
      @row-click="handleRowClick"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="host" label="主机" min-width="150" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
            {{ row.status === 'success' ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="elapsed" label="耗时" width="100" />
      <el-table-column prop="error" label="错误信息" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.error" style="color: var(--el-color-danger)">{{ row.error }}</span>
          <span v-else style="color: var(--el-color-success)">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click.stop="handleViewDetail(row)">
            查看详情
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { BatchExecuteResult } from '@/api/operations'
import { useConsoleStore } from '@/stores/console'

const consoleStore = useConsoleStore()

const statusFilter = ref('')
const selectedRows = ref<BatchExecuteResult[]>([])

const results = computed(() => consoleStore.executionResults)

// 默认显示失败项（如果有失败的话）
onMounted(() => {
  if (results.value.some(r => r.status === 'error')) {
    statusFilter.value = 'error'
  }
})

// 监听结果变化，如果有新失败项，自动切换到失败筛选
watch(() => results.value, (newResults) => {
  if (newResults.some(r => r.status === 'error') && !statusFilter.value) {
    statusFilter.value = 'error'
  }
}, { deep: true })

const filteredResults = computed(() => {
  if (!statusFilter.value) {
    // 失败项置顶
    const sorted = [...results.value]
    return sorted.sort((a, b) => {
      if (a.status === 'error' && b.status !== 'error') return -1
      if (a.status !== 'error' && b.status === 'error') return 1
      return 0
    })
  }
  return results.value.filter(r => r.status === statusFilter.value)
})

const successCount = computed(() => {
  return results.value.filter(r => r.status === 'success').length
})

const errorCount = computed(() => {
  return results.value.filter(r => r.status === 'error').length
})

// 计算平均耗时
const averageElapsed = computed(() => {
  if (results.value.length === 0) return '0ms'
  const total = results.value.reduce((sum, r) => {
    const elapsed = parseElapsed(r.elapsed)
    return sum + elapsed
  }, 0)
  const avg = total / results.value.length
  return formatElapsed(avg)
})

// 解析耗时字符串为毫秒数
function parseElapsed(elapsed: string): number {
  if (!elapsed) return 0
  const match = elapsed.match(/(\d+\.?\d*)(ms|s|m)/)
  if (!match) return 0
  const value = parseFloat(match[1])
  const unit = match[2]
  switch (unit) {
    case 'ms':
      return value
    case 's':
      return value * 1000
    case 'm':
      return value * 60 * 1000
    default:
      return 0
  }
}

// 格式化毫秒数为字符串
function formatElapsed(ms: number): string {
  if (ms < 1000) {
    return `${Math.round(ms)}ms`
  } else if (ms < 60000) {
    return `${(ms / 1000).toFixed(2)}s`
  } else {
    return `${(ms / 60000).toFixed(2)}m`
  }
}

// 重试所有失败项
const handleRetryAllFailed = async () => {
  const failedResults = results.value.filter(r => r.status === 'error')
  if (failedResults.length === 0) {
    ElMessage.warning('没有失败项需要重试')
    return
  }
  
  if (!consoleStore.currentOperation) {
    ElMessage.warning('无法重试：未知操作类型')
    return
  }
  
  const failedHosts = failedResults.map(r => r.host)
  
  try {
    await consoleStore.executeOperation(
      consoleStore.currentOperation,
      failedHosts,
      consoleStore.operationParams
    )
    ElMessage.success(`成功重试 ${failedHosts.length} 个失败的主机`)
    // 重置筛选，显示所有结果
    statusFilter.value = ''
  } catch (error: any) {
    ElMessage.error(error.message || '重试失败')
  }
}

const handleSelectionChange = (rows: BatchExecuteResult[]) => {
  selectedRows.value = rows
}

const handleRowClick = (row: BatchExecuteResult) => {
  // 可以触发查看详情
  emit('view-detail', row)
}

const handleViewDetail = (row: BatchExecuteResult) => {
  emit('view-detail', row)
}

const handleCopySelected = () => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请先选择要复制的结果')
    return
  }

  const text = selectedRows.value
    .map(r => `${r.host}\t${r.status}\t${r.elapsed}\t${r.error || '-'}`)
    .join('\n')

  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const handleExportSelected = () => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请先选择要导出的结果')
    return
  }

  const data = selectedRows.value.map(r => ({
    主机: r.host,
    状态: r.status === 'success' ? '成功' : '失败',
    耗时: r.elapsed,
    错误信息: r.error || '-'
  }))

  const json = JSON.stringify(data, null, 2)
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `批量操作结果_${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)

  ElMessage.success('导出成功')
}

const emit = defineEmits<{
  'view-detail': [result: BatchExecuteResult]
}>()
</script>

<style scoped>
.result-overview {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.status-dashboard {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: linear-gradient(to right, #f5f7fa, #fff);
  border-bottom: 1px solid var(--el-border-color);
}

.stat-item {
  flex: 1;
  padding: 12px;
  background: #fff;
  border-radius: 8px;
  text-align: center;
  border: 1px solid var(--el-border-color);
  transition: all 0.3s;
}

.stat-item:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.stat-item.success {
  border-left: 4px solid #67c23a;
}

.stat-item.error {
  border-left: 4px solid #f56c6c;
}

.stat-item.total {
  border-left: 4px solid #409eff;
}

.stat-item.elapsed {
  border-left: 4px solid #e6a23c;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 4px;
}

.stat-item.success .stat-value {
  color: #67c23a;
}

.stat-item.error .stat-value {
  color: #f56c6c;
}

.stat-item.total .stat-value {
  color: #409eff;
}

.stat-item.elapsed .stat-value {
  color: #e6a23c;
}

.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.overview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color);
}

.header-left {
  display: flex;
  align-items: center;
}

.result-count {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  margin-left: 10px;
}
</style>


