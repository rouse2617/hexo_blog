<template>
  <div class="audit-log-viewer">
    <!-- Filter Panel -->
    <div class="audit-log-viewer__filters">
      <el-form :inline="true" :model="localFilters" class="filter-form">
        <!-- Time Range Filter -->
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="localFilters.timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            :clearable="true"
            @change="handleFilterChange"
          />
        </el-form-item>

        <!-- User Filter -->
        <el-form-item label="用户">
          <el-input
            v-model="localFilters.userId"
            placeholder="输入用户ID"
            clearable
            @change="handleFilterChange"
          />
        </el-form-item>

        <!-- Host Filter -->
        <el-form-item label="主机">
          <el-select
            v-model="localFilters.host"
            placeholder="选择主机"
            clearable
            filterable
            @change="handleFilterChange"
          >
            <el-option
              v-for="host in uniqueHosts"
              :key="host"
              :label="host"
              :value="host"
            />
          </el-select>
        </el-form-item>

        <!-- Risk Level Filter -->
        <el-form-item label="风险级别">
          <el-select
            v-model="localFilters.riskLevel"
            placeholder="选择风险级别"
            clearable
            @change="handleFilterChange"
          >
            <el-option label="低风险" value="low" />
            <el-option label="中风险" value="medium" />
            <el-option label="高风险" value="high" />
            <el-option label="极高风险" value="critical" />
          </el-select>
        </el-form-item>

        <!-- Search Command Pattern -->
        <el-form-item label="命令">
          <el-input
            v-model="localFilters.commandPattern"
            placeholder="搜索命令"
            clearable
            @change="handleFilterChange"
          />
        </el-form-item>

        <!-- Export Buttons -->
        <el-form-item>
          <el-button-group>
            <el-button
              type="primary"
              :icon="Download"
              @click="handleExport('csv')"
              :disabled="filteredLogs.length === 0"
            >
              导出 CSV
            </el-button>
            <el-button
              type="success"
              :icon="DocumentCopy"
              @click="handleExport('json')"
              :disabled="filteredLogs.length === 0"
            >
              导出 JSON
            </el-button>
          </el-button-group>
        </el-form-item>
      </el-form>
    </div>

    <!-- Statistics Summary -->
    <div class="audit-log-viewer__stats">
      <el-row :gutter="16">
        <el-col :span="6">
          <el-statistic title="总记录数" :value="filteredLogs.length" />
        </el-col>
        <el-col :span="6">
          <el-statistic
            title="高风险操作"
            :value="highRiskCount"
            :value-style="{ color: '#F56C6C' }"
          />
        </el-col>
        <el-col :span="6">
          <el-statistic
            title="失败率"
            :value="failureRate"
            suffix="%"
            :precision="1"
            :value-style="{ color: failureRate > 10 ? '#F56C6C' : '#67C23A' }"
          />
        </el-col>
        <el-col :span="6">
          <el-statistic
            title="涉及主机数"
            :value="uniqueHosts.length"
          />
        </el-col>
      </el-row>
    </div>

    <!-- Audit Log Table -->
    <div class="audit-log-viewer__table">
      <el-table
        :data="paginatedLogs"
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        :default-sort="{ prop: 'timestamp', order: 'descending' }"
        @sort-change="handleSortChange"
      >
        <!-- Timestamp -->
        <el-table-column
          prop="timestamp"
          label="时间"
          width="180"
          sortable="custom"
        >
          <template #default="{ row }">
            <span class="timestamp-text">
              {{ formatTimestamp(row.timestamp) }}
            </span>
          </template>
        </el-table-column>

        <!-- User ID -->
        <el-table-column
          prop="userId"
          label="用户"
          width="120"
        />

        <!-- Command -->
        <el-table-column
          prop="command"
          label="命令"
          min-width="200"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <code class="command-text">{{ row.command }}</code>
          </template>
        </el-table-column>

        <!-- Target Hosts -->
        <el-table-column
          prop="targetHosts"
          label="目标主机"
          width="150"
        >
          <template #default="{ row }">
            <el-tag
              v-for="host in row.targetHosts"
              :key="host"
              size="small"
              type="info"
              style="margin: 2px"
            >
              {{ host }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Risk Level -->
        <el-table-column
          prop="riskLevel"
          label="风险级别"
          width="120"
          sortable="custom"
        >
          <template #default="{ row }">
            <el-tag :type="getRiskTagType(row.riskLevel)">
              {{ getRiskLabel(row.riskLevel) }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Execution Status -->
        <el-table-column
          prop="executionResult.status"
          label="执行状态"
          width="120"
        >
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.executionResult.status)">
              {{ getStatusLabel(row.executionResult.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Duration -->
        <el-table-column
          prop="executionResult.duration"
          label="耗时"
          width="100"
          sortable="custom"
        >
          <template #default="{ row }">
            {{ formatDuration(row.executionResult.duration) }}
          </template>
        </el-table-column>

        <!-- Source -->
        <el-table-column
          prop="metadata.source"
          label="来源"
          width="120"
        >
          <template #default="{ row }">
            <el-tag size="small" type="info">
              {{ getSourceLabel(row.metadata.source) }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- Actions -->
        <el-table-column
          label="操作"
          width="100"
          fixed="right"
        >
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              text
              @click="handleViewDetails(row.id)"
            >
              查看
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100, 200]"
          :total="filteredLogs.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Download, DocumentCopy } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type {
  AuditLogEntry,
  AuditLogFilters,
  RiskLevel,
  ExecutionResult
} from '@/types/chat-ui'

// ============================================================================
// Props & Emits
// ============================================================================

interface Props {
  logs?: AuditLogEntry[]
  filters?: Partial<AuditLogFilters>
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  logs: () => [],
  filters: () => ({}),
  loading: false
})

const emit = defineEmits<{
  filterChange: [filters: AuditLogFilters]
  export: [format: 'csv' | 'json', data: AuditLogEntry[]]
  viewDetails: [logId: string]
}>()

// ============================================================================
// Local State
// ============================================================================

const localFilters = ref<Partial<AuditLogFilters>>({
  timeRange: undefined,
  userId: undefined,
  riskLevel: undefined,
  host: undefined,
  commandPattern: undefined,
  limit: undefined,
  offset: undefined
})

const currentPage = ref(1)
const pageSize = ref(20)
const sortProp = ref<'timestamp' | 'riskLevel' | 'duration'>('timestamp')
const sortOrder = ref<'ascending' | 'descending' | null>('descending')

// ============================================================================
// Computed Properties
// ============================================================================

// Apply filters to logs
const filteredLogs = computed(() => {
  let logs = [...props.logs]

  // Filter by time range
  if (localFilters.value.timeRange) {
    const [start, end] = localFilters.value.timeRange
    const startTime = new Date(start as string | Date).getTime()
    const endTime = new Date(end as string | Date).getTime()
    logs = logs.filter(log => log.timestamp >= startTime && log.timestamp <= endTime)
  }

  // Filter by user ID
  if (localFilters.value.userId) {
    const userId = localFilters.value.userId.toLowerCase()
    logs = logs.filter(log => log.userId.toLowerCase().includes(userId))
  }

  // Filter by host
  if (localFilters.value.host) {
    const host = localFilters.value.host
    logs = logs.filter(log => log.targetHosts.includes(host))
  }

  // Filter by risk level
  if (localFilters.value.riskLevel) {
    logs = logs.filter(log => log.riskLevel === localFilters.value.riskLevel)
  }

  // Filter by command pattern
  if (localFilters.value.commandPattern) {
    const pattern = localFilters.value.commandPattern.toLowerCase()
    logs = logs.filter(log => log.command.toLowerCase().includes(pattern))
  }

  // Apply sorting
  logs.sort((a, b) => {
    let comparison = 0

    switch (sortProp.value) {
      case 'timestamp':
        comparison = a.timestamp - b.timestamp
        break
      case 'riskLevel':
        const riskOrder = { low: 1, medium: 2, high: 3, critical: 4 }
        comparison = riskOrder[a.riskLevel] - riskOrder[b.riskLevel]
        break
      case 'duration':
        comparison = a.executionResult.duration - b.executionResult.duration
        break
    }

    return sortOrder.value === 'ascending' ? comparison : -comparison
  })

  return logs
})

// Get unique hosts for filter dropdown
const uniqueHosts = computed(() => {
  const hosts = new Set<string>()
  props.logs.forEach(log => {
    log.targetHosts.forEach(host => hosts.add(host))
  })
  return Array.from(hosts).sort()
})

// Paginated logs
const paginatedLogs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredLogs.value.slice(start, end)
})

// Statistics
const highRiskCount = computed(() => {
  return filteredLogs.value.filter(log =>
    log.riskLevel === 'high' || log.riskLevel === 'critical'
  ).length
})

const failureRate = computed(() => {
  if (filteredLogs.value.length === 0) return 0
  const failures = filteredLogs.value.filter(log =>
    log.executionResult.status === 'failure'
  ).length
  return (failures / filteredLogs.value.length) * 100
})

// ============================================================================
// Methods
// ============================================================================

// Format timestamp
const formatTimestamp = (timestamp: number): string => {
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

// Format duration
const formatDuration = (duration: number): string => {
  if (duration < 1000) {
    return `${duration}ms`
  } else if (duration < 60000) {
    return `${(duration / 1000).toFixed(1)}s`
  } else {
    const minutes = Math.floor(duration / 60000)
    const seconds = ((duration % 60000) / 1000).toFixed(0)
    return `${minutes}m ${seconds}s`
  }
}

// Get risk tag type
const getRiskTagType = (level: RiskLevel) => {
  const typeMap = {
    low: 'success',
    medium: 'warning',
    high: 'danger',
    critical: 'danger'
  }
  return typeMap[level] || 'info'
}

// Get risk label
const getRiskLabel = (level: RiskLevel): string => {
  const labelMap = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '极高风险'
  }
  return labelMap[level] || level
}

// Get status tag type
const getStatusTagType = (status: ExecutionResult['status']) => {
  const typeMap = {
    success: 'success',
    failure: 'danger',
    cancelled: 'info'
  }
  return typeMap[status] || 'info'
}

// Get status label
const getStatusLabel = (status: ExecutionResult['status']): string => {
  const labelMap = {
    success: '成功',
    failure: '失败',
    cancelled: '已取消'
  }
  return labelMap[status] || status
}

// Get source label
const getSourceLabel = (source: string): string => {
  const labelMap: Record<string, string> = {
    chat: '聊天',
    quick_terminal: '快速终端',
    slash_command: '斜杠命令'
  }
  return labelMap[source] || source
}

// Handle filter change
const handleFilterChange = () => {
  currentPage.value = 1
  emit('filterChange', localFilters.value as AuditLogFilters)
}

// Handle export
const handleExport = (format: 'csv' | 'json') => {
  const data = filteredLogs.value

  if (data.length === 0) {
    ElMessage.warning('没有可导出的数据')
    return
  }

  try {
    if (format === 'csv') {
      exportCSV(data)
    } else {
      exportJSON(data)
    }
    emit('export', format, data)
    ElMessage.success(`成功导出 ${data.length} 条记录为 ${format.toUpperCase()}`)
  } catch (error) {
    console.error('Export failed:', error)
    ElMessage.error('导出失败')
  }
}

// Export to CSV
const exportCSV = (data: AuditLogEntry[]) => {
  const headers = [
    '时间',
    '用户ID',
    '会话ID',
    '命令',
    '目标主机',
    '风险级别',
    '执行状态',
    '退出码',
    '耗时(ms)',
    '来源',
    '触发方式'
  ]

  const rows = data.map(log => [
    formatTimestamp(log.timestamp),
    log.userId,
    log.sessionId,
    `"${log.command.replace(/"/g, '""')}"`,
    log.targetHosts.join(';'),
    getRiskLabel(log.riskLevel),
    getStatusLabel(log.executionResult.status),
    log.executionResult.exitCode?.toString() || '',
    log.executionResult.duration.toString(),
    getSourceLabel(log.metadata.source),
    log.metadata.triggeredBy
  ])

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.join(','))
  ].join('\n')

  downloadFile(csvContent, `audit_logs_${Date.now()}.csv`, 'text/csv;charset=utf-8;')
}

// Export to JSON
const exportJSON = (data: AuditLogEntry[]) => {
  const jsonContent = JSON.stringify(data, null, 2)
  downloadFile(jsonContent, `audit_logs_${Date.now()}.json`, 'application/json;charset=utf-8;')
}

// Download file helper
const downloadFile = (content: string, filename: string, mimeType: string) => {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// Handle view details
const handleViewDetails = (logId: string) => {
  emit('viewDetails', logId)
}

// Handle sort change
const handleSortChange = ({ prop, order }: { prop: string; order: string | null }) => {
  sortProp.value = (prop || 'timestamp') as 'timestamp' | 'riskLevel' | 'duration'
  sortOrder.value = (order || 'descending') as 'ascending' | 'descending' | null
}

// Handle page size change
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
}

// Handle current page change
const handleCurrentChange = (page: number) => {
  currentPage.value = page
}

// ============================================================================
// Watchers
// ============================================================================

watch(() => props.filters, (newFilters) => {
  localFilters.value = { ...newFilters }
}, { deep: true, immediate: true })
</script>

<script lang="ts">
export default {
  name: 'AuditLogViewer'
}
</script>

<style scoped>
.audit-log-viewer {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.audit-log-viewer__filters {
  padding: 16px;
  background-color: var(--el-bg-color-page);
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
}

.filter-form {
  margin: 0;
}

.filter-form :deep(.el-form-item) {
  margin-bottom: 12px;
}

.filter-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.audit-log-viewer__stats {
  padding: 16px;
  background-color: var(--el-bg-color);
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
}

.audit-log-viewer__stats :deep(.el-statistic) {
  text-align: center;
}

.audit-log-viewer__stats :deep(.el-statistic__head) {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.audit-log-viewer__stats :deep(.el-statistic__content) {
  font-size: 24px;
  font-weight: 600;
}

.audit-log-viewer__table {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.audit-log-viewer__table :deep(.el-table) {
  flex: 1;
}

.timestamp-text {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.command-text {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  background-color: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 3px;
  color: var(--el-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
  max-width: 100%;
}

.pagination-container {
  padding: 16px 0;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid var(--el-border-color);
  background-color: var(--el-bg-color);
}

/* Risk level custom colors */
.el-tag--success {
  --el-tag-bg-color: #f0f9ff;
  --el-tag-border-color: #67c23a;
  --el-tag-text-color: #67c23a;
}

.el-tag--warning {
  --el-tag-bg-color: #fdf6ec;
  --el-tag-border-color: #e6a23c;
  --el-tag-text-color: #e6a23c;
}

.el-tag--danger {
  --el-tag-bg-color: #fef0f0;
  --el-tag-border-color: #f56c6c;
  --el-tag-text-color: #f56c6c;
}

/* Critical risk special styling */
.el-tag--danger.critical {
  --el-tag-bg-color: #fee;
  --el-tag-border-color: #c45656;
  --el-tag-text-color: #c45656;
  font-weight: 600;
}

/* Responsive adjustments */
@media (max-width: 1200px) {
  .audit-log-viewer__stats :deep(.el-col) {
    margin-bottom: 8px;
  }
}
</style>
