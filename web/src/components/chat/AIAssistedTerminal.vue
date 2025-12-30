<template>
  <div class="ai-assisted-terminal" :class="{ 'dark-mode': isDark, 'collapsed': !isExpanded }">
    <!-- Terminal Header -->
    <div class="terminal-header" @click="toggleTerminal">
      <div class="header-left">
        <el-icon class="terminal-icon"><Monitor /></el-icon>
        <span class="terminal-title">AI-Assisted Terminal</span>
        <el-badge
          v-if="commandHistory.length > 0"
          :value="commandHistory.length"
          :max="99"
          class="history-badge"
        />
      </div>
      <div class="header-right">
        <el-tooltip content="Toggle Terminal (Ctrl+`)" placement="bottom">
          <el-icon class="toggle-icon">
            <component :is="isExpanded ? ArrowDown : ArrowUp" />
          </el-icon>
        </el-tooltip>
      </div>
    </div>

    <!-- Terminal Content -->
    <transition name="slide-down">
      <div v-show="isExpanded" class="terminal-content">
        <!-- Risk Assessment Panel -->
        <div
          v-if="currentRiskAssessment && currentRiskAssessment.level !== 'low'"
          class="risk-panel"
          :class="`risk-${currentRiskAssessment.level}`"
        >
          <div class="risk-header">
            <el-icon class="risk-icon">
              <Warning v-if="currentRiskAssessment.level === 'medium'" />
              <WarningFilled v-else />
            </el-icon>
            <span class="risk-title">
              {{ getRiskTitle(currentRiskAssessment.level) }} Risk Command
            </span>
          </div>
          <div class="risk-content">
            <p class="risk-reason">{{ currentRiskAssessment.reason }}</p>
            <ul v-if="currentRiskAssessment.warnings.length > 0" class="risk-warnings">
              <li v-for="(warning, index) in currentRiskAssessment.warnings" :key="index">
                {{ warning }}
              </li>
            </ul>
            <div v-if="currentRiskAssessment.estimatedImpact" class="risk-impact">
              <strong>Estimated Impact:</strong> {{ currentRiskAssessment.estimatedImpact }}
            </div>
          </div>
          <div class="risk-actions">
            <el-button
              v-if="currentRiskAssessment.level === 'critical'"
              type="danger"
              size="small"
              @click="confirmExecution"
              :loading="isExecuting"
            >
              I Understand, Execute
            </el-button>
            <el-button
              v-else-if="currentRiskAssessment.level === 'high'"
              type="warning"
              size="small"
              @click="confirmExecution"
              :loading="isExecuting"
            >
              Confirm Execution
            </el-button>
            <el-button
              v-else
              type="primary"
              size="small"
              @click="confirmExecution"
              :loading="isExecuting"
            >
              Execute Anyway
            </el-button>
            <el-button size="small" @click="cancelExecution">Cancel</el-button>
          </div>
        </div>

        <!-- Input Area -->
        <div class="terminal-input-area">
          <div class="prompt-line">
            <span class="prompt-symbol">$</span>
            <el-select
              v-model="selectedHost"
              placeholder="Select host"
              size="small"
              class="host-select"
              filterable
            >
              <el-option
                v-for="host in hosts"
                :key="host"
                :label="host"
                :value="host"
              />
            </el-select>
          </div>
          <div class="command-input-wrapper">
            <el-input
              ref="commandInputRef"
              v-model="currentCommand"
              type="textarea"
              :rows="1"
              placeholder="Type a command... (Use Up/Down for history)"
              class="command-input"
              :disabled="!selectedHost"
              @keydown="handleKeydown"
              @input="onCommandInput"
              resize="none"
            />
            <el-button
              type="primary"
              :icon="Position"
              @click="executeCommand"
              :loading="isExecuting"
              :disabled="!currentCommand.trim() || !selectedHost"
              class="execute-button"
              size="small"
            >
              Execute
            </el-button>
          </div>
          <div v-if="currentRiskAssessment && currentRiskAssessment.level === 'low'" class="risk-indicator low-risk">
            <el-icon><CircleCheck /></el-icon>
            <span>Low risk command</span>
          </div>
        </div>

        <!-- Output Area -->
        <div v-if="output.length > 0" class="terminal-output">
          <div
            v-for="(item, index) in output"
            :key="index"
            class="output-item"
            :class="`status-${item.status}`"
          >
            <div class="output-header">
              <span class="output-prompt">$ {{ item.command }}</span>
              <span class="output-host">{{ item.host }}</span>
              <span class="output-time">{{ formatTime(item.timestamp) }}</span>
            </div>
            <pre class="output-result">{{ item.result }}</pre>
          </div>
        </div>

        <!-- History Panel -->
        <div v-if="showHistory && commandHistory.length > 0" class="history-panel">
          <div class="history-header">
            <span>Command History ({{ commandHistory.length }})</span>
            <el-button text size="small" @click="showHistory = false">
              <el-icon><Close /></el-icon>
            </el-button>
          </div>
          <div class="history-list">
            <div
              v-for="(cmd, index) in commandHistory.slice().reverse()"
              :key="index"
              class="history-item"
              @click="useHistoryCommand(cmd)"
            >
              <span class="history-command">{{ cmd }}</span>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Monitor,
  ArrowUp,
  ArrowDown,
  Position,
  Warning,
  WarningFilled,
  CircleCheck,
  Close
} from '@element-plus/icons-vue'
import type { RiskAssessment, TerminalOutput } from '@/types/chat-ui'
import { ElNotification } from 'element-plus'

// Props
interface Props {
  hosts: string[]
}

const props = withDefaults(defineProps<Props>(), {
  hosts: () => []
})

// Emits
const emit = defineEmits<{
  execute: [command: string, host: string]
  assessRisk: [command: string]
}>()

// State
const isExpanded = ref(true)
const selectedHost = ref('')
const currentCommand = ref('')
const commandHistory = ref<string[]>([])
const historyIndex = ref(-1)
const output = ref<TerminalOutput[]>([])
const currentRiskAssessment = ref<RiskAssessment | null>(null)
const isExecuting = ref(false)
const showHistory = ref(false)
const commandInputRef = ref()
let autoCollapseTimer: number | null = null

// Computed
const isDark = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Methods
function toggleTerminal() {
  isExpanded.value = !isExpanded.value
}

function getRiskTitle(level: string): string {
  switch (level) {
    case 'critical':
      return 'Critical'
    case 'high':
      return 'High'
    case 'medium':
      return 'Medium'
    default:
      return 'Low'
  }
}

async function assessCommandRisk(command: string): Promise<RiskAssessment> {
  // Emit event to parent for risk assessment
  emit('assessRisk', command)

  // For demo, provide default assessment
  // In production, the parent component would handle the actual assessment
  return {
    level: 'low',
    score: 0,
    reason: 'Command appears safe',
    warnings: [],
    requiresConfirmation: false
  }
}

async function onCommandInput() {
  if (!currentCommand.value.trim()) {
    currentRiskAssessment.value = null
    return
  }

  // Assess risk in real-time
  try {
    currentRiskAssessment.value = await assessCommandRisk(currentCommand.value)
  } catch (error) {
    console.error('Risk assessment failed:', error)
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    navigateHistory('up')
  } else if (event.key === 'ArrowDown') {
    event.preventDefault()
    navigateHistory('down')
  } else if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    if (currentRiskAssessment.value?.requiresConfirmation) {
      // Show confirmation dialog
      return
    }
    executeCommand()
  } else if (event.key === 'Escape') {
    cancelExecution()
  }
}

function navigateHistory(direction: 'up' | 'down') {
  if (commandHistory.value.length === 0) return

  if (direction === 'up') {
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++
      currentCommand.value = commandHistory.value[commandHistory.value.length - 1 - historyIndex.value]
    }
  } else {
    if (historyIndex.value > 0) {
      historyIndex.value--
      currentCommand.value = commandHistory.value[commandHistory.value.length - 1 - historyIndex.value]
    } else if (historyIndex.value === 0) {
      historyIndex.value = -1
      currentCommand.value = ''
    }
  }
}

function useHistoryCommand(command: string) {
  currentCommand.value = command
  showHistory.value = false
  nextTick(() => {
    commandInputRef.value?.focus()
  })
}

async function executeCommand() {
  if (!currentCommand.value.trim() || !selectedHost.value) return

  // Check if confirmation is required
  if (currentRiskAssessment.value?.requiresConfirmation) {
    return // Wait for user to click confirm button
  }

  isExecuting.value = true
  const command = currentCommand.value.trim()
  const host = selectedHost.value

  try {
    // Add to history
    if (!commandHistory.value.includes(command)) {
      commandHistory.value.push(command)
    }

    // Emit execution event
    emit('execute', command, host)

    // Log to audit
    logToAudit(command, host, currentRiskAssessment.value)

    // Simulate output (in real app, this would come from the execution result)
    const result: TerminalOutput = {
      command,
      host,
      result: `Command executed on ${host}`,
      timestamp: Date.now(),
      status: 'success'
    }
    output.value.push(result)

    // Clear command
    currentCommand.value = ''
    currentRiskAssessment.value = null
    historyIndex.value = -1

    // Auto-collapse after 30 seconds
    resetAutoCollapseTimer()

    ElNotification({
      title: 'Command Executed',
      message: `Successfully executed on ${host}`,
      type: 'success',
      duration: 2000
    })
  } catch (error) {
    ElNotification({
      title: 'Execution Failed',
      message: error instanceof Error ? error.message : 'Unknown error',
      type: 'error',
      duration: 3000
    })
  } finally {
    isExecuting.value = false
  }
}

function confirmExecution() {
  if (currentRiskAssessment.value?.level === 'critical') {
    // Require explicit confirmation
    executeCommand()
  } else {
    executeCommand()
  }
}

function cancelExecution() {
  currentCommand.value = ''
  currentRiskAssessment.value = null
}

function logToAudit(command: string, host: string, risk: RiskAssessment | null) {
  // In a real application, send to audit log API
  const auditEntry = {
    timestamp: Date.now(),
    command,
    host,
    riskLevel: risk?.level || 'low',
    source: 'quick_terminal'
  }
  console.log('[Audit]', auditEntry)
}

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString()
}

function resetAutoCollapseTimer() {
  if (autoCollapseTimer !== null) {
    clearTimeout(autoCollapseTimer)
  }

  autoCollapseTimer = window.setTimeout(() => {
    if (!isExecuting.value) {
      isExpanded.value = false
    }
  }, 30000) // 30 seconds
}

// Lifecycle
onMounted(() => {
  // Set default host
  if (props.hosts.length > 0) {
    selectedHost.value = props.hosts[0]
  }

  // Global keyboard shortcut
  const handleKeydown = (e: KeyboardEvent) => {
    if (e.ctrlKey && e.key === '`') {
      e.preventDefault()
      toggleTerminal()
    }
  }

  window.addEventListener('keydown', handleKeydown)

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
    if (autoCollapseTimer !== null) {
      clearTimeout(autoCollapseTimer)
    }
  })
})
</script>

<style scoped lang="scss">
.ai-assisted-terminal {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s ease;

  &.dark-mode {
    background: #1a1a1a;
    border-color: #333;
  }

  &.collapsed {
    .terminal-header {
      border-radius: 8px;
    }
  }
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;

  &:hover {
    background: var(--el-fill-color);
  }
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-icon {
  font-size: 18px;
  color: var(--el-color-primary);
}

.terminal-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.history-badge {
  margin-left: 4px;
}

.toggle-icon {
  font-size: 16px;
  color: var(--el-text-color-secondary);
  transition: transform 0.3s;
}

.terminal-content {
  padding: 16px;
  max-height: 500px;
  overflow-y: auto;
}

// Risk Panel
.risk-panel {
  margin-bottom: 16px;
  padding: 12px;
  border-radius: 6px;
  border-left: 4px solid;

  &.risk-medium {
    background: rgba(var(--el-color-warning-rgb), 0.1);
    border-left-color: var(--el-color-warning);
  }

  &.risk-high {
    background: rgba(var(--el-color-danger-rgb), 0.1);
    border-left-color: var(--el-color-danger);
  }

  &.risk-critical {
    background: rgba(196, 86, 86, 0.1);
    border-left-color: #c45656;
  }
}

.risk-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.risk-icon {
  font-size: 20px;
}

.risk-title {
  font-size: 14px;
  font-weight: 600;
}

.risk-content {
  margin-bottom: 12px;
  font-size: 13px;

  .risk-reason {
    margin: 0 0 8px 0;
    color: var(--el-text-color-primary);
  }

  .risk-warnings {
    margin: 8px 0;
    padding-left: 20px;

    li {
      margin-bottom: 4px;
      color: var(--el-text-color-regular);
    }
  }

  .risk-impact {
    margin-top: 8px;
    padding: 8px;
    background: rgba(0, 0, 0, 0.05);
    border-radius: 4px;
    font-size: 12px;
  }
}

.risk-actions {
  display: flex;
  gap: 8px;
}

// Terminal Input
.terminal-input-area {
  margin-bottom: 16px;
}

.prompt-line {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.prompt-symbol {
  font-family: 'Consolas', monospace;
  font-size: 14px;
  font-weight: bold;
  color: var(--el-color-success);
}

.host-select {
  flex: 1;
  max-width: 200px;
}

.command-input-wrapper {
  display: flex;
  gap: 8px;
}

.command-input {
  flex: 1;

  :deep(.el-textarea__inner) {
    font-family: 'Consolas', monospace;
    font-size: 13px;
  }
}

.execute-button {
  align-self: flex-end;
}

.risk-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;

  &.low-risk {
    color: var(--el-color-success);
  }
}

// Output
.terminal-output {
  margin-bottom: 16px;
  max-height: 300px;
  overflow-y: auto;
  background: #0d0d0d;
  border-radius: 6px;
  padding: 12px;
  font-family: 'Consolas', monospace;
  font-size: 12px;
}

.output-item {
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #333;

  &:last-child {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }

  &.status-error {
    .output-result {
      color: var(--el-color-danger);
    }
  }

  &.status-success {
    .output-result {
      color: var(--el-color-success);
    }
  }
}

.output-header {
  display: flex;
  gap: 12px;
  margin-bottom: 6px;
  color: #888;
}

.output-prompt {
  color: var(--el-color-primary);
}

.output-host {
  color: var(--el-color-warning);
}

.output-time {
  color: #666;
  font-size: 11px;
}

.output-result {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  color: #e2e8f0;
}

// History Panel
.history-panel {
  background: var(--el-fill-color-light);
  border-radius: 6px;
  padding: 12px;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  font-size: 13px;
  font-weight: 600;
}

.history-list {
  max-height: 200px;
  overflow-y: auto;
}

.history-item {
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.2s;

  &:hover {
    background: var(--el-fill-color);
  }
}

.history-command {
  font-family: 'Consolas', monospace;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

// Animations
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
  max-height: 600px;
  overflow: hidden;
}

.slide-down-enter-from,
.slide-down-leave-to {
  max-height: 0;
  opacity: 0;
}

// Dark mode
.dark-mode {
  .terminal-header {
    background: #2a2a2a;

    &:hover {
      background: #333;
    }
  }

  .terminal-output {
    background: #0a0a0a;
  }

  .history-panel {
    background: #2a2a2a;
  }

  .history-item:hover {
    background: #333;
  }

  .risk-content .risk-impact {
    background: rgba(255, 255, 255, 0.05);
  }
}

// Scrollbar
.terminal-content::-webkit-scrollbar,
.terminal-output::-webkit-scrollbar,
.history-list::-webkit-scrollbar {
  width: 6px;
}

.terminal-content::-webkit-scrollbar-track,
.terminal-output::-webkit-scrollbar-track,
.history-list::-webkit-scrollbar-track {
  background: transparent;
}

.terminal-content::-webkit-scrollbar-thumb,
.terminal-output::-webkit-scrollbar-thumb,
.history-list::-webkit-scrollbar-thumb {
  background: var(--el-border-color-darker);
  border-radius: 3px;

  &:hover {
    background: var(--el-border-color-dark);
  }
}
</style>
