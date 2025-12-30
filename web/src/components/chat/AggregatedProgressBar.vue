<template>
  <div class="aggregated-progress-bar">
    <!-- Overall progress -->
    <div class="overall-section">
      <div class="section-header">
        <div class="header-info">
          <el-icon :size="16" :class="{ spinning: isInProgress }">
            <Loading />
          </el-icon>
          <span class="operation-title">{{ operationTitle }}</span>
          <el-tag size="small" :type="overallStatusType">{{ overallStatusText }}</el-tag>
        </div>
        <div class="header-stats">
          <span class="stat-text">{{ completedCount }}/{{ totalCount }} 完成</span>
          <span class="stat-text" v-if="hasErrors">
            {{ errorCount }} 失败
          </span>
        </div>
      </div>

      <!-- Main progress bar -->
      <div class="progress-container">
        <el-progress
          :percentage="overallProgress"
          :status="progressStatus"
          :stroke-width="8"
          :show-text="false"
        />
        <span class="progress-text">{{ overallProgress }}%</span>
      </div>
    </div>

    <!-- Detailed breakdown (collapsible) -->
    <el-collapse-transition>
      <div class="details-section" v-show="showDetails">
        <div class="section-header">
          <span class="details-title">详细进度</span>
          <el-button type="text" size="small" @click="showDetails = false">
            收起
          </el-button>
        </div>

        <!-- Batch operation groups -->
        <div class="batch-groups">
          <div
            v-for="(group, index) in operationGroups"
            :key="index"
            class="batch-group"
          >
            <div class="group-header">
              <span class="group-name">{{ group.name }}</span>
              <div class="group-info">
                <el-tag size="small" :type="getGroupStatusType(group)">
                  {{ getGroupStatusText(group) }}
                </el-tag>
                <span class="group-count">{{ group.completed }}/{{ group.total }}</span>
              </div>
            </div>

            <!-- Group progress bar -->
            <div class="group-progress">
              <el-progress
                :percentage="getGroupProgress(group)"
                :status="getGroupProgressStatus(group)"
                :stroke-width="6"
                :show-text="false"
              />
            </div>

            <!-- Individual items within group (expandable) -->
            <el-collapse-transition>
              <div class="items-list" v-show="group.expanded">
                <div
                  v-for="item in group.items"
                  :key="item.id"
                  class="item-row"
                  :class="`status-${item.status}`"
                >
                  <div class="item-info">
                    <el-icon :size="14" class="item-icon">
                      <component :is="getItemIcon(item.status)" />
                    </el-icon>
                    <span class="item-name">{{ item.name }}</span>
                    <span v-if="item.error" class="item-error">{{ item.error }}</span>
                  </div>
                  <div class="item-actions">
                    <el-button
                      v-if="item.status === 'failed'"
                      type="text"
                      size="small"
                      @click="$emit('retry', item)"
                    >
                      重试
                    </el-button>
                    <el-button
                      type="text"
                      size="small"
                      @click="$emit('showLogs', item)"
                    >
                      日志
                    </el-button>
                  </div>
                </div>
              </div>
            </el-collapse-transition>

            <div class="group-toggle" @click="toggleGroup(index)">
              <el-icon :size="14" :class="{ expanded: group.expanded }">
                <ArrowDown />
              </el-icon>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="action-buttons">
          <el-button
            v-if="hasErrors"
            type="warning"
            size="small"
            :icon="RefreshRight"
            @click="$emit('retryFailed')"
          >
            重试失败项
          </el-button>
          <el-button
            v-if="isInProgress"
            type="danger"
            size="small"
            :icon="Close"
            @click="$emit('cancel')"
          >
            取消操作
          </el-button>
          <el-button
            v-if="isCompleted"
            type="primary"
            size="small"
            :icon="Check"
            @click="$emit('confirm')"
          >
            确认完成
          </el-button>
        </div>
      </div>
    </el-collapse-transition>

    <!-- Show details button (when collapsed) -->
    <div class="show-details-btn" v-show="!showDetails" @click="showDetails = true">
      <span>查看详情</span>
      <el-icon :size="14">
        <ArrowDown />
      </el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Loading, Check, Close, Clock, ArrowDown, RefreshRight } from '@element-plus/icons-vue'

interface OperationItem {
  id: string
  name: string
  status: 'pending' | 'running' | 'success' | 'failed'
  error?: string
  result?: any
}

interface OperationGroup {
  name: string
  items: OperationItem[]
  expanded?: boolean
}

const props = defineProps<{
  operationName?: string
  groups: OperationGroup[]
}>()

defineEmits<{
  retry: [item: OperationItem]
  retryFailed: []
  cancel: []
  confirm: []
  showLogs: [item: OperationItem]
}>()

const showDetails = ref(true)

const operationTitle = computed(() => {
  return props.operationName || '批量操作'
})

const totalCount = computed(() => {
  return props.groups.reduce((sum, group) => sum + group.items.length, 0)
})

const completedCount = computed(() => {
  return props.groups.reduce(
    (sum, group) => sum + group.items.filter(i => i.status === 'success').length,
    0
  )
})

const errorCount = computed(() => {
  return props.groups.reduce(
    (sum, group) => sum + group.items.filter(i => i.status === 'failed').length,
    0
  )
})

const hasErrors = computed(() => errorCount.value > 0)

const runningCount = computed(() => {
  return props.groups.reduce(
    (sum, group) => sum + group.items.filter(i => i.status === 'running').length,
    0
  )
})

const isInProgress = computed(() => runningCount.value > 0)

const isCompleted = computed(() => {
  return completedCount.value + errorCount.value === totalCount.value && totalCount.value > 0
})

const overallProgress = computed(() => {
  if (totalCount.value === 0) return 0
  return Math.round((completedCount.value / totalCount.value) * 100)
})

const progressStatus = computed(() => {
  if (isInProgress.value) return undefined
  if (hasErrors.value) return 'exception'
  if (isCompleted.value) return 'success'
  return undefined
})

const overallStatusText = computed(() => {
  if (isInProgress.value) return '进行中'
  if (isCompleted.value) return hasErrors.value ? '部分失败' : '已完成'
  return '等待中'
})

const overallStatusType = computed(() => {
  if (isInProgress.value) return 'warning'
  if (isCompleted.value) return hasErrors.value ? 'danger' : 'success'
  return 'info'
})

const operationGroups = computed(() => {
  return props.groups.map(group => ({
    ...group,
    expanded: group.expanded ?? false,
    total: group.items.length,
    completed: group.items.filter(i => i.status === 'success').length,
    failed: group.items.filter(i => i.status === 'failed').length,
    running: group.items.filter(i => i.status === 'running').length
  }))
})

const getGroupProgress = (group: any) => {
  if (group.total === 0) return 0
  return Math.round((group.completed / group.total) * 100)
}

const getGroupProgressStatus = (group: any) => {
  if (group.running > 0) return undefined
  if (group.failed > 0) return 'exception'
  if (group.completed === group.total && group.total > 0) return 'success'
  return undefined
}

const getGroupStatusType = (group: any) => {
  if (group.running > 0) return 'warning'
  if (group.failed > 0) return 'danger'
  if (group.completed === group.total && group.total > 0) return 'success'
  return 'info'
}

const getGroupStatusText = (group: any) => {
  if (group.running > 0) return '进行中'
  if (group.failed > 0) return '部分失败'
  if (group.completed === group.total && group.total > 0) return '已完成'
  return '等待中'
}

const getItemIcon = (status: string) => {
  switch (status) {
    case 'running':
      return Loading
    case 'success':
      return Check
    case 'failed':
      return Close
    default:
      return Clock
  }
}

const toggleGroup = (index: number) => {
  const group = operationGroups.value[index]
  if (group) {
    group.expanded = !group.expanded
  }
}
</script>

<style scoped>
.aggregated-progress-bar {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  margin: 12px 0;
}

.overall-section {
  margin-bottom: 12px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-stats {
  display: flex;
  gap: 12px;
}

.stat-text {
  font-size: 12px;
  color: #6b7280;
}

.operation-title {
  font-weight: 600;
  font-size: 14px;
  color: #1f2937;
}

.progress-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-text {
  font-size: 12px;
  font-weight: 600;
  color: #6b7280;
  min-width: 45px;
  text-align: right;
}

.details-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.details-title {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.batch-groups {
  margin-top: 12px;
}

.batch-group {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.group-name {
  font-weight: 600;
  font-size: 13px;
  color: #374151;
}

.group-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-count {
  font-size: 12px;
  color: #6b7280;
}

.group-progress {
  margin: 8px 0;
}

.items-list {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #e5e7eb;
}

.item-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  background: #fff;
  border-radius: 4px;
  margin-bottom: 4px;
  font-size: 12px;
}

.item-row:last-child {
  margin-bottom: 0;
}

.item-row.status-running {
  border-left: 2px solid #e6a23c;
}

.item-row.status-success {
  border-left: 2px solid #67c23a;
}

.item-row.status-failed {
  border-left: 2px solid #f56c6c;
}

.item-info {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.item-name {
  color: #374151;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-error {
  color: #f56c6c;
  font-size: 11px;
}

.item-icon {
  flex-shrink: 0;
}

.item-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.item-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.group-toggle {
  display: flex;
  justify-content: center;
  margin-top: 8px;
  cursor: pointer;
  color: #6b7280;
  transition: color 0.2s;
}

.group-toggle:hover {
  color: #409eff;
}

.group-toggle .el-icon {
  transition: transform 0.3s;
}

.group-toggle .el-icon.expanded {
  transform: rotate(180deg);
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.show-details-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px;
  color: #409eff;
  cursor: pointer;
  font-size: 13px;
  transition: color 0.2s;
}

.show-details-btn:hover {
  color: #66b1ff;
}
</style>
