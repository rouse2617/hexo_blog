<template>
  <div class="connection-status-container">
    <el-tooltip
      :content="tooltipContent"
      placement="bottom"
      :disabled="!showTooltip"
    >
      <div
        class="status-indicator"
        :class="statusClass"
        @click="handleClick"
      >
        <div class="status-dot" :class="{ pulsing: isConnecting }"></div>
        <span class="status-text">{{ statusText }}</span>
        <el-icon
          v-if="showDetailsIcon"
          class="details-icon"
          :class="{ 'is-rotated': detailsVisible }"
        >
          <ArrowDown />
        </el-icon>
      </div>
    </el-tooltip>

    <!-- Connection Details Panel -->
    <el-popover
      v-model:visible="detailsVisible"
      placement="bottom-end"
      :width="320"
      trigger="click"
      popper-class="connection-details-popover"
    >
      <template #reference>
        <!-- Hidden reference, popover is controlled by status indicator click -->
      </template>

      <div class="connection-details">
        <div class="details-header">
          <h4>Connection Details</h4>
          <el-button
            link
            type="primary"
            @click="handleReconnect"
            :loading="isConnecting"
            :disabled="isConnected"
          >
            {{ isConnecting ? 'Connecting...' : 'Reconnect' }}
          </el-button>
        </div>

        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="Status">
            <el-tag
              :type="statusTagType"
              size="small"
              effect="plain"
            >
              {{ statusText }}
            </el-tag>
          </el-descriptions-item>

          <el-descriptions-item label="Server">
            {{ serverUrl }}
          </el-descriptions-item>

          <el-descriptions-item label="Last Connected">
            {{ lastConnectedTime }}
          </el-descriptions-item>

          <el-descriptions-item v-if="isConnected" label="Uptime">
            {{ uptime }}
          </el-descriptions-item>

          <el-descriptions-item v-if="isConnecting" label="Reconnecting">
            Attempt {{ nextReconnectIn }}s
          </el-descriptions-item>

          <el-descriptions-item v-if="isError" label="Error">
            <span class="error-message">{{ errorMessage }}</span>
          </el-descriptions-item>
        </el-descriptions>

        <!-- Connection Quality Indicator -->
        <div v-if="isConnected" class="connection-quality">
          <div class="quality-label">Connection Quality</div>
          <div class="quality-bars">
            <div
              v-for="i in 4"
              :key="i"
              class="quality-bar"
              :class="{ active: i <= qualityLevel }"
            ></div>
          </div>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'
import { useWebSocket } from '@/composables/useWebSocket'

interface Props {
  serverUrl?: string
}

const props = withDefaults(defineProps<Props>(), {
  serverUrl: 'ws://localhost:8080/ws'
})

// WebSocket state
const {
  connectionState,
  statusText,
  connectionUptime,
  lastConnected,
  getStatusDetails
} = useWebSocket({ url: props.serverUrl })

const detailsVisible = ref(false)
const showTooltip = ref(false)

const isConnected = computed(() => connectionState.value === 'connected')
const isConnecting = computed(() => connectionState.value === 'connecting')
const isDisconnected = computed(() => connectionState.value === 'disconnected')
const isError = computed(() => connectionState.value === 'error')

const statusClass = computed(() => {
  return {
    'status-connected': isConnected.value,
    'status-connecting': isConnecting.value,
    'status-disconnected': isDisconnected.value,
    'status-error': isError.value
  }
})

const statusTagType = computed(() => {
  switch (connectionState.value) {
    case 'connected':
      return 'success'
    case 'connecting':
      return 'warning'
    case 'disconnected':
      return 'info'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
})

const showDetailsIcon = computed(() => {
  return isConnected.value || isError.value
})

const tooltipContent = computed(() => {
  return `${statusText.value} - Click for details`
})

const lastConnectedTime = computed(() => {
  if (!lastConnected.value) return 'Never'

  const now = new Date()
  const diff = now.getTime() - lastConnected.value.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return lastConnected.value.toLocaleString()
})

const uptime = computed(() => {
  const ms = connectionUptime.value
  if (ms < 1000) return '0s'

  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`

  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`

  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
})

const nextReconnectIn = computed(() => {
  const details = getStatusDetails()
  return Math.ceil(details.nextReconnectIn / 1000)
})

const errorMessage = computed(() => {
  const details = getStatusDetails()
  return details.error?.toString() || 'Unknown error'
})

// Simulated connection quality (4 bars max)
const qualityLevel = computed(() => {
  if (!isConnected.value) return 0

  // In real implementation, calculate based on latency, packet loss, etc.
  // For now, return max quality when connected
  return 4
})

function handleClick() {
  if (showDetailsIcon.value) {
    detailsVisible.value = !detailsVisible.value
  }
}

function handleReconnect() {
  // Reconnect logic would be handled by useWebSocket
  window.location.reload()
}

// Auto-hide tooltip after 3 seconds
let tooltipTimer: number | null = null

function showTooltipTemporarily() {
  showTooltip.value = true

  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
  }

  tooltipTimer = window.setTimeout(() => {
    showTooltip.value = false
  }, 3000)
}

onMounted(() => {
  // Show tooltip briefly on mount
  showTooltipTemporarily()
})

onUnmounted(() => {
  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
  }
})
</script>

<style scoped>
.connection-status-container {
  display: inline-block;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border-primary);
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.3s ease;
  user-select: none;
}

.status-indicator:hover {
  background-color: var(--color-bg-tertiary);
  border-color: var(--color-primary);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  animation: pulse 2s ease-in-out infinite;
}

.status-dot.pulsing {
  animation: pulse-pulsing 1s ease-in-out infinite;
}

.status-connected .status-dot {
  background-color: var(--color-status-connected);
}

.status-connecting .status-dot {
  background-color: var(--color-status-connecting);
}

.status-disconnected .status-dot {
  background-color: var(--color-status-disconnected);
  animation: none;
}

.status-error .status-dot {
  background-color: var(--color-status-disconnected);
  animation: none;
}

.status-text {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.details-icon {
  margin-left: 4px;
  font-size: 14px;
  transition: transform 0.3s ease;
  color: var(--color-text-secondary);
}

.details-icon.is-rotated {
  transform: rotate(180deg);
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}

@keyframes pulse-pulsing {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.2);
    opacity: 0.8;
  }
}

/* Connection Details Panel */
.connection-details {
  padding: 4px 0;
}

.details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-secondary);
}

.details-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.error-message {
  color: var(--color-error);
  font-size: 12px;
  word-break: break-word;
}

/* Connection Quality Indicator */
.connection-quality {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border-secondary);
}

.quality-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.quality-bars {
  display: flex;
  gap: 4px;
  align-items: flex-end;
  height: 16px;
}

.quality-bar {
  flex: 1;
  height: 8px;
  background-color: var(--color-border-secondary);
  border-radius: 2px;
  transition: all 0.3s ease;
}

.quality-bar.active {
  background-color: var(--color-status-connected);
}

.quality-bar:nth-child(1).active {
  height: 8px;
}

.quality-bar:nth-child(2).active {
  height: 10px;
}

.quality-bar:nth-child(3).active {
  height: 12px;
}

.quality-bar:nth-child(4).active {
  height: 16px;
}
</style>

<style>
.connection-details-popover {
  padding: 16px !important;
}
</style>
