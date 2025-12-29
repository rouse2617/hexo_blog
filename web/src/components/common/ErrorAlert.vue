<template>
  <transition name="el-zoom-in-top">
    <div
      v-if="visible"
      :class="['error-alert', `error-alert--${type}`, { 'is-closable': closable }]"
      @mouseenter="handleMouseEnter"
      @mouseleave="handleMouseLeave"
    >
      <div class="error-alert__content">
        <!-- 图标 -->
        <div class="error-alert__icon">
          <el-icon v-if="type === 'error'">
            <CircleClose />
          </el-icon>
          <el-icon v-else-if="type === 'warning'">
            <Warning />
          </el-icon>
          <el-icon v-else>
            <InfoFilled />
          </el-icon>
        </div>

        <!-- 错误信息 -->
        <div class="error-alert__message">
          <div v-if="title" class="error-alert__title">{{ title }}</div>
          <div class="error-alert__description">{{ message }}</div>

          <!-- 额外信息 -->
          <div v-if="details" class="error-alert__details">
            <el-button
              text
              size="small"
              @click="showDetails = !showDetails"
            >
              {{ showDetails ? '隐藏' : '查看' }}详情
            </el-button>
            <el-collapse-transition>
              <div v-show="showDetails" class="error-alert__detail-text">
                <pre>{{ details }}</pre>
              </div>
            </el-collapse-transition>
          </div>

          <!-- 重试按钮 -->
          <div v-if="retryable && onRetry" class="error-alert__actions">
            <el-button
              :loading="retrying"
              type="primary"
              size="small"
              @click="handleRetry"
            >
              {{ retrying ? '重试中...' : `重试${retryCount > 0 ? ` (${retryCount})` : ''}` }}
            </el-button>
          </div>
        </div>

        <!-- 关闭按钮 -->
        <div v-if="closable" class="error-alert__close">
          <el-icon @click="close">
            <Close />
          </el-icon>
        </div>
      </div>

      <!-- 进度条 -->
      <div v-if="autoClose && !retrying" class="error-alert__progress">
        <div
          class="error-alert__progress-bar"
          :style="{ animationDuration: `${duration}ms` }"
        />
      </div>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { CircleClose, Warning, InfoFilled, Close } from '@element-plus/icons-vue'
import { AppError } from '@/utils/errorHandler'

interface Props {
  error?: AppError | Error | string
  title?: string
  message?: string
  type?: 'error' | 'warning' | 'info'
  details?: string
  closable?: boolean
  retryable?: boolean
  autoClose?: boolean
  duration?: number
  onClose?: () => void
  onRetry?: () => Promise<void> | void
}

const props = withDefaults(defineProps<Props>(), {
  type: 'error',
  closable: true,
  retryable: false,
  autoClose: false,
  duration: 5000
})

const emit = defineEmits<{
  close: []
  retry: []
}>()

const visible = ref(true)
const showDetails = ref(false)
const retrying = ref(false)
const retryCount = ref(0)
let timer: number | null = null


// 关闭提示
const close = () => {
  visible.value = false
  if (timer) clearTimeout(timer)
  emit('close')
}

// 重试
const handleRetry = async () => {
  if (!props.onRetry || retrying.value) return

  retrying.value = true
  try {
    await props.onRetry()
    retryCount.value = 0
    visible.value = false
    emit('retry')
  } catch (error) {
    retryCount.value++
    console.error('Retry failed:', error)
  } finally {
    retrying.value = false
  }
}

// 自动关闭
const startTimer = () => {
  if (props.autoClose && props.duration > 0) {
    timer = setTimeout(() => {
      close()
    }, props.duration) as unknown as number
  }
}

const stopTimer = () => {
  if (timer) {
    clearTimeout(timer)
    timer = null
  }
}

// 鼠标悬停时暂停自动关闭
const handleMouseEnter = () => {
  stopTimer()
}

const handleMouseLeave = () => {
  if (visible.value) {
    startTimer()
  }
}

onMounted(() => {
  startTimer()
})

onUnmounted(() => {
  stopTimer()
})

watch(() => props.autoClose, () => {
  stopTimer()
  if (visible.value) {
    startTimer()
  }
})
</script>

<script lang="ts">
export default {
  name: 'ErrorAlert'
}
</script>

<style scoped>
.error-alert {
  position: relative;
  padding: 12px 16px;
  margin-bottom: 12px;
  border-radius: 4px;
  border-left: 4px solid;
  background-color: var(--el-bg-color);
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.error-alert--error {
  border-left-color: var(--el-color-danger);
  background-color: #fef0f0;
}

.error-alert--warning {
  border-left-color: var(--el-color-warning);
  background-color: #fdf6ec;
}

.error-alert--info {
  border-left-color: var(--el-color-info);
  background-color: #f4f4f5;
}

.error-alert__content {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.error-alert__icon {
  flex-shrink: 0;
  font-size: 20px;
  margin-top: 2px;
}

.error-alert--error .error-alert__icon {
  color: var(--el-color-danger);
}

.error-alert--warning .error-alert__icon {
  color: var(--el-color-warning);
}

.error-alert--info .error-alert__icon {
  color: var(--el-color-info);
}

.error-alert__message {
  flex: 1;
  min-width: 0;
}

.error-alert__title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
  color: var(--el-text-color-primary);
}

.error-alert__description {
  font-size: 14px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}

.error-alert__details {
  margin-top: 8px;
}

.error-alert__detail-text {
  margin-top: 8px;
  padding: 8px;
  background-color: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
  overflow: auto;
}

.error-alert__detail-text pre {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.error-alert__actions {
  margin-top: 12px;
}

.error-alert__close {
  flex-shrink: 0;
  cursor: pointer;
  font-size: 16px;
  color: var(--el-text-color-secondary);
  transition: color 0.3s;
}

.error-alert__close:hover {
  color: var(--el-text-color-primary);
}

.error-alert__progress {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 3px;
  background-color: rgba(0, 0, 0, 0.1);
}

.error-alert__progress-bar {
  height: 100%;
  background-color: currentColor;
  animation: progress linear forwards;
  opacity: 0.3;
}

.error-alert--error .error-alert__progress-bar {
  background-color: var(--el-color-danger);
}

.error-alert--warning .error-alert__progress-bar {
  background-color: var(--el-color-warning);
}

.error-alert--info .error-alert__progress-bar {
  background-color: var(--el-color-info);
}

@keyframes progress {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
</style>
