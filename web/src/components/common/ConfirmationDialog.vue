<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogStyle.title"
    :width="480"
    :close-on-click-modal="false"
    :close-on-press-escape="true"
    :show-close="true"
    class="confirmation-dialog"
    :class="`confirmation-dialog--${request?.riskLevel || 'low'}`"
    @close="handleCancel"
    @opened="handleDialogOpened"
    role="alertdialog"
    aria-modal="true"
    :aria-labelledby="`${request?.riskLevel || 'low'}-dialog-title`"
    :aria-describedby="`${request?.riskLevel || 'low'}-dialog-content`"
  >
    <!-- Header with icon -->
    <div class="confirmation-dialog__header">
      <div class="confirmation-dialog__icon" :style="{ color: dialogStyle.color }" aria-hidden="true">
        <el-icon :size="48">
          <component :is="iconComponent" />
        </el-icon>
      </div>
    </div>

    <!-- Content -->
    <div class="confirmation-dialog__content" :id="`${request?.riskLevel || 'low'}-dialog-content`">
      <!-- Command display (Req 11.2) -->
      <div class="confirmation-dialog__command">
        <div class="confirmation-dialog__label">即将执行的命令:</div>
        <div class="confirmation-dialog__command-text">
          <code>{{ request?.command || '' }}</code>
        </div>
      </div>

      <!-- Affected hosts (Req 11.3) -->
      <div v-if="request?.affectedHosts?.length" class="confirmation-dialog__hosts">
        <div class="confirmation-dialog__label">影响的主机:</div>
        <div class="confirmation-dialog__host-list">
          <el-tag
            v-for="host in request.affectedHosts"
            :key="host"
            size="small"
            type="info"
            class="confirmation-dialog__host-tag"
          >
            {{ host }}
          </el-tag>
        </div>
      </div>

      <!-- Estimated impact -->
      <div v-if="request?.estimatedImpact" class="confirmation-dialog__impact">
        <div class="confirmation-dialog__label">预估影响:</div>
        <div class="confirmation-dialog__impact-text">{{ request.estimatedImpact }}</div>
      </div>

      <!-- Warning message for high/critical risk -->
      <div v-if="isHighRisk" class="confirmation-dialog__warning">
        <el-icon><WarningFilled /></el-icon>
        <span>此操作具有高风险，可能导致数据丢失或系统不可用。请谨慎操作！</span>
      </div>

      <!-- Typing confirmation for high risk (Req 11.4) -->
      <div v-if="request?.requiresTyping" class="confirmation-dialog__typing">
        <div class="confirmation-dialog__label">
          请输入 <strong>CONFIRM</strong> 以确认执行:
        </div>
        <el-input
          ref="confirmInputRef"
          v-model="confirmText"
          placeholder="输入 CONFIRM"
          :class="{ 'is-error': confirmText && confirmText !== 'CONFIRM' }"
          @keyup.enter="handleConfirm"
        />
        <div v-if="confirmText && confirmText !== 'CONFIRM'" class="confirmation-dialog__typing-hint">
          请输入正确的确认文本
        </div>
      </div>
    </div>

    <!-- Footer with buttons -->
    <template #footer>
      <div class="confirmation-dialog__footer">
        <!-- Countdown display (Req 11.6) -->
        <div v-if="countdown > 0" class="confirmation-dialog__countdown">
          <el-progress
            :percentage="countdownPercentage"
            :stroke-width="4"
            :show-text="false"
            :color="dialogStyle.color"
          />
          <span class="confirmation-dialog__countdown-text">
            {{ countdown }}秒后可确认
          </span>
        </div>

        <div class="confirmation-dialog__buttons">
          <!-- Cancel button - default focus (Req 11.5) -->
          <el-button
            ref="cancelButtonRef"
            @click="handleCancel"
          >
            取消
          </el-button>

          <!-- Confirm button -->
          <el-button
            :type="confirmButtonType"
            :disabled="!canConfirm"
            @click="handleConfirm"
          >
            {{ confirmButtonText }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted, nextTick } from 'vue'
import {
  InfoFilled,
  Warning,
  WarningFilled,
  CircleCloseFilled
} from '@element-plus/icons-vue'
import type { ConfirmationRequest, RiskLevel, DialogStyle } from '@/types/chat-ui'
import { DIALOG_STYLES } from '@/types/chat-ui'

// ============================================================================
// Props and Emits
// ============================================================================

interface Props {
  visible: boolean
  request: ConfirmationRequest | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: []
  cancel: []
}>()

// ============================================================================
// Refs
// ============================================================================

const confirmText = ref('')
const countdown = ref(0)
const cancelButtonRef = ref<InstanceType<typeof import('element-plus')['ElButton']> | null>(null)
const confirmInputRef = ref<InstanceType<typeof import('element-plus')['ElInput']> | null>(null)

let countdownTimer: ReturnType<typeof setInterval> | null = null

// ============================================================================
// Computed Properties
// ============================================================================

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const riskLevel = computed<RiskLevel>(() => props.request?.riskLevel || 'low')

const dialogStyle = computed<DialogStyle>(() => {
  return DIALOG_STYLES[riskLevel.value]
})

const iconComponent = computed(() => {
  const iconMap: Record<string, typeof InfoFilled> = {
    'Info': InfoFilled,
    'Warning': Warning,
    'WarningFilled': WarningFilled,
    'CircleCloseFilled': CircleCloseFilled
  }
  return iconMap[dialogStyle.value.icon] || InfoFilled
})

const isHighRisk = computed(() => {
  return riskLevel.value === 'high' || riskLevel.value === 'critical'
})

const confirmButtonType = computed(() => {
  if (riskLevel.value === 'critical') return 'danger'
  if (riskLevel.value === 'high') return 'warning'
  return 'primary'
})

const confirmButtonText = computed(() => {
  if (countdown.value > 0) return `确认 (${countdown.value}s)`
  return '确认执行'
})

const countdownPercentage = computed(() => {
  const initialCountdown = props.request?.countdown || dialogStyle.value.countdown
  if (initialCountdown <= 0) return 100
  return ((initialCountdown - countdown.value) / initialCountdown) * 100
})

const canConfirm = computed(() => {
  // Must wait for countdown to finish
  if (countdown.value > 0) return false
  
  // If typing is required, must match "CONFIRM"
  if (props.request?.requiresTyping) {
    return confirmText.value === 'CONFIRM'
  }
  
  return true
})

// ============================================================================
// Methods
// ============================================================================

function startCountdown() {
  // Clear any existing timer
  stopCountdown()
  
  // Get countdown duration from request or dialog style
  const duration = props.request?.countdown ?? dialogStyle.value.countdown
  
  if (duration > 0) {
    countdown.value = duration
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        stopCountdown()
      }
    }, 1000)
  }
}

function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function resetState() {
  confirmText.value = ''
  countdown.value = 0
  stopCountdown()
}

function handleDialogOpened() {
  // Start countdown when dialog opens
  startCountdown()
  
  // Focus on cancel button by default (Req 11.5)
  nextTick(() => {
    if (cancelButtonRef.value?.$el) {
      cancelButtonRef.value.$el.focus()
    }
  })
}

function handleConfirm() {
  if (!canConfirm.value) return
  
  emit('confirm')
  dialogVisible.value = false
  resetState()
}

function handleCancel() {
  emit('cancel')
  dialogVisible.value = false
  resetState()
}

// ============================================================================
// Watchers
// ============================================================================

watch(() => props.visible, (newVisible) => {
  if (newVisible) {
    // Reset state when dialog opens
    resetState()
  } else {
    // Clean up when dialog closes
    stopCountdown()
  }
})

// ============================================================================
// Lifecycle
// ============================================================================

onUnmounted(() => {
  stopCountdown()
})
</script>

<script lang="ts">
export default {
  name: 'ConfirmationDialog'
}
</script>


<style scoped>
/* ============================================================================
   Base Dialog Styles
   ============================================================================ */

.confirmation-dialog {
  --dialog-icon-size: 48px;
  --dialog-spacing: 16px;
}

.confirmation-dialog__header {
  display: flex;
  justify-content: center;
  margin-bottom: var(--dialog-spacing);
}

.confirmation-dialog__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background-color: var(--el-fill-color-light);
}

/* ============================================================================
   Content Styles
   ============================================================================ */

.confirmation-dialog__content {
  display: flex;
  flex-direction: column;
  gap: var(--dialog-spacing);
}

.confirmation-dialog__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.confirmation-dialog__command {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
}

.confirmation-dialog__command-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-all;
  white-space: pre-wrap;
}

.confirmation-dialog__command-text code {
  color: var(--el-color-danger);
  background: transparent;
  padding: 0;
}

/* ============================================================================
   Hosts List Styles
   ============================================================================ */

.confirmation-dialog__hosts {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.confirmation-dialog__host-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.confirmation-dialog__host-tag {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

/* ============================================================================
   Impact Styles
   ============================================================================ */

.confirmation-dialog__impact {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.confirmation-dialog__impact-text {
  font-size: 14px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
}

/* ============================================================================
   Warning Styles
   ============================================================================ */

.confirmation-dialog__warning {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px;
  background-color: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-5);
  border-radius: 8px;
  color: var(--el-color-warning-dark-2);
  font-size: 14px;
  line-height: 1.5;
}

.confirmation-dialog__warning .el-icon {
  flex-shrink: 0;
  font-size: 18px;
  margin-top: 2px;
}

/* ============================================================================
   Typing Confirmation Styles
   ============================================================================ */

.confirmation-dialog__typing {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 8px;
}

.confirmation-dialog__typing .el-input {
  margin-top: 8px;
}

.confirmation-dialog__typing .el-input.is-error :deep(.el-input__wrapper) {
  box-shadow: 0 0 0 1px var(--el-color-danger) inset;
}

.confirmation-dialog__typing-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-color-danger);
}

.confirmation-dialog__typing strong {
  color: var(--el-color-danger);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

/* ============================================================================
   Footer Styles
   ============================================================================ */

.confirmation-dialog__footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.confirmation-dialog__countdown {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.confirmation-dialog__countdown .el-progress {
  width: 100%;
}

.confirmation-dialog__countdown-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.confirmation-dialog__buttons {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* ============================================================================
   Risk Level Specific Styles (Req 11.1)
   ============================================================================ */

/* Low Risk */
.confirmation-dialog--low .confirmation-dialog__icon {
  background-color: var(--el-color-primary-light-9);
}

/* Medium Risk */
.confirmation-dialog--medium .confirmation-dialog__icon {
  background-color: var(--el-color-warning-light-9);
}

/* High Risk */
.confirmation-dialog--high .confirmation-dialog__icon {
  background-color: var(--el-color-danger-light-9);
}

.confirmation-dialog--high :deep(.el-dialog__header) {
  background-color: var(--el-color-danger-light-9);
  border-bottom: 2px solid var(--el-color-danger);
}

.confirmation-dialog--high :deep(.el-dialog__title) {
  color: var(--el-color-danger);
  font-weight: 600;
}

/* Critical Risk */
.confirmation-dialog--critical .confirmation-dialog__icon {
  background-color: var(--el-color-danger-light-8);
}

.confirmation-dialog--critical :deep(.el-dialog__header) {
  background-color: var(--el-color-danger-light-8);
  border-bottom: 3px solid #C45656;
}

.confirmation-dialog--critical :deep(.el-dialog__title) {
  color: #C45656;
  font-weight: 700;
}

.confirmation-dialog--critical .confirmation-dialog__command-text code {
  color: #C45656;
}

/* ============================================================================
   Dark Mode Support
   ============================================================================ */

:root[data-theme='dark'] .confirmation-dialog__command,
:root[data-theme='dark'] .confirmation-dialog__hosts,
:root[data-theme='dark'] .confirmation-dialog__impact,
:root[data-theme='dark'] .confirmation-dialog__typing {
  background-color: var(--el-fill-color-dark);
  border-color: var(--el-border-color-darker);
}

:root[data-theme='dark'] .confirmation-dialog__warning {
  background-color: rgba(230, 162, 60, 0.1);
  border-color: rgba(230, 162, 60, 0.3);
}

:root[data-theme='dark'] .confirmation-dialog--high :deep(.el-dialog__header),
:root[data-theme='dark'] .confirmation-dialog--critical :deep(.el-dialog__header) {
  background-color: rgba(245, 108, 108, 0.1);
}

/* ============================================================================
   Responsive Styles
   ============================================================================ */

@media (max-width: 768px) {
  .confirmation-dialog :deep(.el-dialog) {
    width: 90% !important;
    max-width: 480px;
  }
  
  .confirmation-dialog__buttons {
    flex-direction: column-reverse;
  }
  
  .confirmation-dialog__buttons .el-button {
    width: 100%;
    margin: 0;
  }
}
</style>
