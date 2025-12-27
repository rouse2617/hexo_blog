<template>
  <div class="result-overview">
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
        <span class="result-count">
          共 {{ filteredResults.length }} 条结果
          <span v-if="statusFilter">
            (成功: {{ successCount }}, 失败: {{ errorCount }})
          </span>
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
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { BatchExecuteResult } from '@/api/operations'
import { useConsoleStore } from '@/stores/console'

const consoleStore = useConsoleStore()

const statusFilter = ref('')
const selectedRows = ref<BatchExecuteResult[]>([])

const results = computed(() => consoleStore.executionResults)

const filteredResults = computed(() => {
  if (!statusFilter.value) {
    return results.value
  }
  return results.value.filter(r => r.status === statusFilter.value)
})

const successCount = computed(() => {
  return results.value.filter(r => r.status === 'success').length
})

const errorCount = computed(() => {
  return results.value.filter(r => r.status === 'error').length
})

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

