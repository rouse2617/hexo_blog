<template>
  <div class="context-panel" :class="{ 'dark-mode': isDark }">
    <!-- Header with Silent Mode Toggle -->
    <div class="panel-header">
      <span class="panel-title">Context Monitor</span>
      <el-switch
        v-model="localSilentMode"
        @change="handleSilentModeToggle"
        active-text="Silent"
        inactive-text="Alert"
        :active-icon="MuteNotification"
        :inactive-icon="Bell"
        size="small"
      />
    </div>

    <!-- Host Metrics Section (max 5 hosts) -->
    <div
      v-for="host in displayHosts"
      :key="host.hostId"
      class="host-metrics"
      :class="{ 'pulse-warning': isAbnormal(host), 'expanded': isAbnormal(host) || isExpanded(host.hostId) }"
    >
      <div class="host-header" @click="toggleExpand(host.hostId)">
        <span class="host-name">{{ host.hostName }}</span>
        <div class="host-status">
          <el-tag
            :type="getStatusType(host)"
            size="small"
            effect="dark"
          >
            {{ host.status }}
          </el-tag>
        </div>
      </div>

      <transition name="expand">
        <div v-show="isAbnormal(host) || isExpanded(host.hostId)" class="metrics-details">
          <!-- CPU Usage -->
          <div class="metric-item">
            <div class="metric-label">CPU Usage</div>
            <div class="metric-value">
              <el-progress
                :percentage="host.cpu"
                :status="getProgressStatus(host.cpu)"
                :show-text="true"
                :stroke-width="8"
              />
            </div>
          </div>

          <!-- Memory Usage -->
          <div class="metric-item">
            <div class="metric-label">Memory Usage</div>
            <div class="metric-value">
              <el-progress
                :percentage="host.memory"
                :status="getProgressStatus(host.memory)"
                :show-text="true"
                :stroke-width="8"
              />
            </div>
          </div>

          <!-- Disk Usage -->
          <div class="metric-item">
            <div class="metric-label">Disk Usage</div>
            <div class="metric-value">
              <el-progress
                :percentage="host.disk"
                :status="getProgressStatus(host.disk)"
                :show-text="true"
                :stroke-width="8"
              />
            </div>
          </div>

          <!-- Alert Threshold Indicator -->
          <div v-if="isAbnormal(host)" class="alert-indicator">
            <el-alert
              :title="getAlertMessage(host)"
              type="warning"
              :closable="false"
              show-icon
            />
          </div>

          <!-- Last Updated -->
          <div class="last-updated">
            <span class="update-label">Last updated:</span>
            <span class="update-time">{{ formatTimestamp(host.lastUpdated) }}</span>
          </div>
        </div>
      </transition>
    </div>

    <!-- Related Files Section (max 3 files, AI mentions, clickable preview) -->
    <div v-if="displayRelatedFiles.length > 0" class="related-files">
      <div class="section-title">Related Files ({{ displayRelatedFiles.length }})</div>
      <div class="files-list">
        <div
          v-for="file in displayRelatedFiles"
          :key="file.path"
          class="file-item"
          :class="{ 'has-highlights': file.highlightLines && file.highlightLines.length > 0 }"
          @click="handleFileClick(file)"
          title="Click to preview file"
        >
          <el-icon><Document /></el-icon>
          <div class="file-info">
            <span class="file-path">{{ file.path }}</span>
            <span v-if="file.highlightLines && file.highlightLines.length > 0" class="highlight-info">
              Lines: {{ file.highlightLines.join(', ') }}
            </span>
          </div>
          <el-icon class="arrow-icon"><ArrowRight /></el-icon>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="displayHosts.length === 0 && displayRelatedFiles.length === 0" class="empty-state">
      <el-empty description="No monitoring data available" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Bell, MuteNotification, Document, ArrowRight } from '@element-plus/icons-vue'
import type { HostMetrics, FileReference } from '@/types/chat-ui'
import { useMetricsStore } from '@/stores/metrics'

// Props
interface Props {
  hosts?: HostMetrics[]
  relatedFiles?: FileReference[]
  silentMode?: boolean
  maxHosts?: number
}

const props = withDefaults(defineProps<Props>(), {
  hosts: () => [],
  relatedFiles: () => [],
  silentMode: false,
  maxHosts: 5
})

// Emits
const emit = defineEmits<{
  silentModeToggle: [value: boolean]
  fileClick: [file: FileReference]
}>()

// Store
const metricsStore = useMetricsStore()

// Local state
const localSilentMode = ref(props.silentMode)
const expandedHosts = ref<Set<string>>(new Set())

// Computed
const isDark = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Use metrics store data with fallback to props
const displayHosts = computed(() => {
  const hostsToDisplay = metricsStore.hostList.length > 0
    ? metricsStore.hostList
    : props.hosts

  return hostsToDisplay.slice(0, props.maxHosts)
})

const displayRelatedFiles = computed(() => {
  // Limit to 3 files max as per requirements
  return props.relatedFiles.slice(0, 3)
})

// Watch for prop changes
watch(() => props.silentMode, (newValue) => {
  localSilentMode.value = newValue
})

// Methods
const handleSilentModeToggle = (value: boolean) => {
  emit('silentModeToggle', value)
}

const handleFileClick = (file: FileReference) => {
  emit('fileClick', file)
}

const isAbnormal = (host: HostMetrics): boolean => {
  // Check thresholds: CPU > 80%, Memory > 85%
  return (
    host.cpu > 80 ||
    host.memory > 85 ||
    host.status === 'warning' ||
    host.status === 'critical'
  )
}

const getStatusType = (host: HostMetrics): 'success' | 'warning' | 'danger' | 'info' => {
  // Determine status based on thresholds
  if (host.cpu > 80 || host.memory > 85 || host.status === 'critical') {
    return 'danger'
  }
  if (host.status === 'warning') {
    return 'warning'
  }
  if (host.status === 'normal') {
    return 'success'
  }
  return 'info'
}

const getProgressStatus = (percentage: number): 'success' | 'exception' | 'warning' => {
  if (percentage >= 90) {
    return 'exception'
  } else if (percentage >= 80) {
    return 'warning'
  }
  return 'success'
}

const getAlertMessage = (host: HostMetrics): string => {
  const alerts: string[] = []
  // CPU threshold: > 80%
  if (host.cpu > 80) {
    alerts.push(`CPU usage is ${host.cpu.toFixed(1)}%`)
  }
  // Memory threshold: > 85%
  if (host.memory > 85) {
    alerts.push(`Memory usage is ${host.memory.toFixed(1)}%`)
  }
  // Disk threshold: > 90%
  if (host.disk > 90) {
    alerts.push(`Disk usage is ${host.disk.toFixed(1)}%`)
  }
  return alerts.join('; ')
}

const isExpanded = (hostId: string): boolean => {
  return expandedHosts.value.has(hostId)
}

const toggleExpand = (hostId: string) => {
  if (expandedHosts.value.has(hostId)) {
    expandedHosts.value.delete(hostId)
  } else {
    expandedHosts.value.add(hostId)
  }
}

const formatTimestamp = (timestamp: number): string => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) {
    return 'just now'
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000)
    return `${minutes}m ago`
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    return `${hours}h ago`
  } else {
    return date.toLocaleDateString()
  }
}

// Subscribe to hosts from props on mount
onMounted(() => {
  props.hosts.forEach(host => {
    metricsStore.subscribe(host.hostId)
  })
})

onUnmounted(() => {
  // Unsubscribe from all hosts
  props.hosts.forEach(host => {
    metricsStore.unsubscribe(host.hostId)
  })
})
</script>

<style scoped lang="scss">
.context-panel {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 16px;
  height: 100%;
  overflow-y: auto;
  border: 1px solid var(--el-border-color);
  transition: all 0.3s ease;

  &.dark-mode {
    background: #1a1a1a;
    border-color: #333;
  }
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color);
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.host-metrics {
  margin-bottom: 12px;
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
  overflow: hidden;
  transition: all 0.3s ease;

  &.pulse-warning {
    animation: pulse-border 2s ease-in-out infinite;
    border-color: var(--el-color-warning);
  }

  &.expanded {
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  }
}

@keyframes pulse-border {
  0%, 100% {
    border-color: var(--el-color-warning);
    box-shadow: 0 0 0 0 rgba(var(--el-color-warning-rgb), 0.4);
  }
  50% {
    border-color: var(--el-color-warning);
    box-shadow: 0 0 0 8px rgba(var(--el-color-warning-rgb), 0);
  }
}

.host-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;

  &:hover {
    background: var(--el-fill-color);
  }
}

.host-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.host-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.3s ease;
  max-height: 500px;
  opacity: 1;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

.metrics-details {
  padding: 12px;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
}

.metric-item {
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.metric-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.metric-value {
  :deep(.el-progress) {
    .el-progress__text {
      font-size: 12px !important;
    }
  }
}

.network-metrics {
  .metric-value {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: var(--el-text-color-regular);
  }
}

.network-stat {
  display: flex;
  align-items: center;
  gap: 4px;
}

.last-updated {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 11px;
  color: var(--el-text-color-secondary);
  display: flex;
  justify-content: space-between;
}

.update-label {
  font-weight: 500;
}

.update-time {
  color: var(--el-text-color-placeholder);
}

.alert-indicator {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);

  :deep(.el-alert) {
    --el-alert-padding: 8px 12px;
  }

  :deep(.el-alert__title) {
    font-size: 12px;
  }
}

.related-files {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color);
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.files-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: var(--el-fill-color);
    transform: translateX(4px);
  }

  &.has-highlights {
    border-left: 3px solid var(--el-color-warning);
  }

  .el-icon {
    font-size: 16px;
    color: var(--el-color-primary);
  }
}

.file-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-path {
  flex: 1;
  font-size: 12px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.highlight-info {
  font-size: 11px;
  color: var(--el-color-warning);
  font-weight: 500;
}

.arrow-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 200px;
}

// Dark mode specific styles
.dark-mode {
  .host-header {
    background: #2a2a2a;

    &:hover {
      background: #333;
    }
  }

  .metrics-details {
    background: #1a1a1a;
  }

  .file-item {
    background: #2a2a2a;

    &:hover {
      background: #333;
    }
  }

  .alert-indicator {
    border-top-color: #333;
  }

  .last-updated {
    border-top-color: #333;
  }
}

// Scrollbar styling
.context-panel::-webkit-scrollbar {
  width: 6px;
}

.context-panel::-webkit-scrollbar-track {
  background: transparent;
}

.context-panel::-webkit-scrollbar-thumb {
  background: var(--el-border-color-darker);
  border-radius: 3px;

  &:hover {
    background: var(--el-border-color-dark);
  }
}
</style>
