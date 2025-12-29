<template>
  <div class="enhanced-tool-card" :class="statusClass">
    <!-- 工具头部 -->
    <div class="tool-header">
      <div class="tool-main-info">
        <div class="tool-icon-wrapper">
          <el-icon class="tool-icon" :size="20">
            <component :is="toolIcon" />
          </el-icon>
        </div>
        <div class="tool-meta">
          <div class="tool-name-row">
            <span class="tool-name">{{ toolCall.name }}</span>
            <el-tag :type="statusTagType" size="small" effect="plain">
              {{ statusText }}
            </el-tag>
          </div>
          <div class="tool-time" v-if="executionTime">
            <el-icon><Clock /></el-icon>
            <span>执行耗时: {{ executionTime }}</span>
          </div>
        </div>
      </div>

      <!-- 快捷操作 -->
      <div class="tool-actions">
        <el-dropdown trigger="click" @command="handleAction">
          <el-button type="text" :icon="MoreFilled" circle />
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="copy" :icon="CopyDocument">
                复制结果
              </el-dropdown-item>
              <el-dropdown-item command="export" :icon="Download">
                导出为文件
              </el-dropdown-item>
              <el-dropdown-item command="rerun" :icon="RefreshRight">
                重新执行
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 进度条（执行中） -->
    <div class="tool-progress" v-if="toolCall.status === 'running'">
      <el-progress
        :percentage="progress"
        :indeterminate="true"
        :show-text="false"
        :stroke-width="3"
      />
    </div>

    <!-- 可折叠内容 -->
    <el-collapse-transition>
      <div v-show="isExpanded" class="tool-body">
        <!-- 参数展示 -->
        <div class="tool-section" v-if="toolCall.arguments">
          <div class="section-header" @click="toggleSection('args')">
            <el-icon class="section-icon" :class="{ expanded: expandedSections.args }">
              <ArrowRight />
            </el-icon>
            <span class="section-title">执行参数</span>
            <el-tag size="small" type="info" effect="plain">
              {{ argCount }} 个参数
            </el-tag>
          </div>
          <el-collapse-transition>
            <div v-show="expandedSections.args" class="section-content">
              <pre class="code-block"><code>{{ formattedArguments }}</code></pre>
              <div class="section-actions">
                <el-button
                  type="text"
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(formattedArguments, '参数')"
                >
                  复制
                </el-button>
              </div>
            </div>
          </el-collapse-transition>
        </div>

        <!-- 结果展示 -->
        <div class="tool-section" v-if="toolCall.result">
          <div class="section-header" @click="toggleSection('result')">
            <el-icon class="section-icon" :class="{ expanded: expandedSections.result }">
              <ArrowRight />
            </el-icon>
            <span class="section-title">执行结果</span>
            <div class="result-stats">
              <el-tag size="small" type="success" effect="plain">
                {{ resultLineCount }} 行
              </el-tag>
              <el-tag size="small" type="info" effect="plain" v-if="resultSize">
                {{ resultSize }}
              </el-tag>
            </div>
          </div>
          <el-collapse-transition>
            <div v-show="expandedSections.result" class="section-content">
              <!-- 尝试解析为 JSON 并格式化 -->
              <div v-if="isJsonResult" class="result-viewer">
                <JsonViewer :data="parsedResult" />
              </div>
              <!-- 普通文本结果 -->
              <pre v-else class="code-block result-code"><code>{{ toolCall.result }}</code></pre>

              <div class="section-actions">
                <el-button
                  type="text"
                  size="small"
                  :icon="CopyDocument"
                  @click="copyToClipboard(toolCall.result, '结果')"
                >
                  复制结果
                </el-button>
                <el-button
                  type="text"
                  size="small"
                  :icon="Download"
                  @click="exportResult"
                >
                  导出
                </el-button>
              </div>
            </div>
          </el-collapse-transition>
        </div>

        <!-- 错误信息 -->
        <div class="tool-section error-section" v-if="toolCall.error">
          <div class="section-header" @click="toggleSection('error')">
            <el-icon class="section-icon" :class="{ expanded: expandedSections.error }">
              <ArrowRight />
            </el-icon>
            <span class="section-title">错误信息</span>
            <el-tag size="small" type="danger" effect="plain">
              执行失败
            </el-tag>
          </div>
          <el-collapse-transition>
            <div v-show="expandedSections.error" class="section-content error-content">
              <pre class="code-block error-code"><code>{{ toolCall.error }}</code></pre>
            </div>
          </el-collapse-transition>
        </div>
      </div>
    </el-collapse-transition>

    <!-- 折叠按钮 -->
    <div class="collapse-trigger" @click="toggleExpand" v-if="toolCall.result || toolCall.error">
      <el-icon :class="{ expanded: isExpanded }">
        <ArrowDown />
      </el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { ToolCall } from '@/api/chat'
import {
  Clock, MoreFilled,
  CopyDocument, Download, RefreshRight, ArrowRight, ArrowDown,
  Monitor, Tools, Document, Warning
} from '@element-plus/icons-vue'
import JsonViewer from './JsonViewer.vue'

const props = defineProps<{
  toolCall: ToolCall
}>()

const emit = defineEmits<{
  rerun: [toolCall: ToolCall]
}>()

// 折叠状态
const isExpanded = ref(true)
const expandedSections = ref({
  args: true,
  result: true,
  error: true
})

// 执行时间追踪
const startTime = ref<number>(0)
const endTime = ref<number>(0)
const progress = ref(0)

// 进度动画
let progressTimer: number | null = null

onMounted(() => {
  if (props.toolCall.status === 'running') {
    startTime.value = Date.now()
    startProgressAnimation()
  } else if (props.toolCall.status === 'success' || props.toolCall.status === 'error') {
    endTime.value = Date.now()
    // 假设平均执行时间 2-5 秒
    startTime.value = endTime.value - 3000
  }
})

onUnmounted(() => {
  if (progressTimer) {
    clearInterval(progressTimer)
  }
})

const startProgressAnimation = () => {
  progress.value = 0
  progressTimer = setInterval(() => {
    if (progress.value < 90) {
      progress.value += Math.random() * 10
    }
  }, 500) as unknown as number
}

// 当工具完成时
if (props.toolCall.status !== 'running') {
  progress.value = 100
  if (progressTimer) {
    clearInterval(progressTimer)
  }
}

const statusClass = computed(() => `status-${props.toolCall.status || 'pending'}`)

const toolIcon = computed(() => {
  const name = props.toolCall.name.toLowerCase()
  if (name.includes('execute') || name.includes('command')) return Tools
  if (name.includes('monitor') || name.includes('status')) return Monitor
  if (name.includes('log') || name.includes('file')) return Document
  return Warning
})

const statusTagType = computed(() => {
  switch (props.toolCall.status) {
    case 'running': return 'warning'
    case 'success': return 'success'
    case 'error': return 'danger'
    default: return 'info'
  }
})

const statusText = computed(() => {
  switch (props.toolCall.status) {
    case 'running': return '执行中...'
    case 'success': return '执行成功'
    case 'error': return '执行失败'
    default: return '等待中'
  }
})

const executionTime = computed(() => {
  if (!startTime.value) return null
  const end = endTime.value || Date.now()
  const duration = end - startTime.value
  if (duration < 1000) return `${duration}ms`
  return `${(duration / 1000).toFixed(2)}s`
})

const formattedArguments = computed(() => {
  try {
    const args = JSON.parse(props.toolCall.arguments)
    return JSON.stringify(args, null, 2)
  } catch {
    return props.toolCall.arguments
  }
})

const argCount = computed(() => {
  try {
    const args = JSON.parse(props.toolCall.arguments)
    return Object.keys(args).length
  } catch {
    return 1
  }
})

const resultLineCount = computed(() => {
  if (!props.toolCall.result) return 0
  return props.toolCall.result.split('\n').length
})

const resultSize = computed(() => {
  if (!props.toolCall.result) return null
  const bytes = new Blob([props.toolCall.result]).size
  if (bytes < 1024) return `${bytes}B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)}MB`
})

const isJsonResult = computed(() => {
  if (!props.toolCall.result) return false
  try {
    JSON.parse(props.toolCall.result)
    return true
  } catch {
    return false
  }
})

const parsedResult = computed(() => {
  if (!isJsonResult.value) return null
  try {
    return JSON.parse(props.toolCall.result!)
  } catch {
    return null
  }
})

const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}

const toggleSection = (section: 'args' | 'result' | 'error') => {
  expandedSections.value[section] = !expandedSections.value[section]
}

const copyToClipboard = async (text: string, label: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${label}已复制到剪贴板`)
  } catch {
    ElMessage.error('复制失败')
  }
}

const exportResult = () => {
  if (!props.toolCall.result) return

  const blob = new Blob([props.toolCall.result], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.toolCall.name}_${Date.now()}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  ElMessage.success('文件已导出')
}

const handleAction = (command: string) => {
  switch (command) {
    case 'copy':
      if (props.toolCall.result) {
        copyToClipboard(props.toolCall.result, '结果')
      }
      break
    case 'export':
      exportResult()
      break
    case 'rerun':
      emit('rerun', props.toolCall)
      break
  }
}
</script>

<style scoped>
.enhanced-tool-card {
  background: #fff;
  border-radius: 10px;
  border: 1px solid #e5e7eb;
  margin: 12px 0;
  overflow: hidden;
  transition: all 0.3s;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.enhanced-tool-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.enhanced-tool-card.status-running {
  border-left: 4px solid #f59e0b;
}

.enhanced-tool-card.status-success {
  border-left: 4px solid #10b981;
}

.enhanced-tool-card.status-error {
  border-left: 4px solid #ef4444;
}

.tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  background: linear-gradient(to bottom, #f9fafb, #fff);
  border-bottom: 1px solid #e5e7eb;
}

.tool-main-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.tool-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border-radius: 10px;
  color: #fff;
  flex-shrink: 0;
}

.status-running .tool-icon-wrapper {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  animation: pulse 2s ease-in-out infinite;
}

.status-success .tool-icon-wrapper {
  background: linear-gradient(135deg, #10b981, #059669);
}

.status-error .tool-icon-wrapper {
  background: linear-gradient(135deg, #ef4444, #dc2626);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

.tool-meta {
  flex: 1;
  min-width: 0;
}

.tool-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.tool-name {
  font-weight: 600;
  color: #1f2937;
  font-size: 14px;
}

.tool-time {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #6b7280;
}

.tool-progress {
  padding: 0 16px;
  background: #fff;
}

.tool-body {
  padding: 12px 16px;
  background: #f9fafb;
}

.tool-section {
  margin-bottom: 12px;
  border-radius: 6px;
  background: #fff;
  border: 1px solid #e5e7eb;
  overflow: hidden;
}

.tool-section:last-child {
  margin-bottom: 0;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: #f9fafb;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}

.section-header:hover {
  background: #f3f4f6;
}

.section-icon {
  transition: transform 0.3s;
}

.section-icon.expanded {
  transform: rotate(90deg);
}

.section-title {
  flex: 1;
  font-weight: 500;
  color: #374151;
  font-size: 13px;
}

.result-stats {
  display: flex;
  gap: 6px;
}

.section-content {
  padding: 12px;
  border-top: 1px solid #e5e7eb;
}

.code-block {
  background: #1e293b;
  border-radius: 6px;
  padding: 12px;
  margin: 0 0 8px 0;
  overflow-x: auto;
  max-height: 300px;
  overflow-y: auto;
}

.code-block code {
  color: #e2e8f0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre;
}

.result-code {
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
}

.result-code code {
  color: #166534;
}

.error-section {
  border-color: #fecaca;
}

.error-content {
  background: #fef2f2;
}

.error-code {
  background: #fee2e2;
  border-color: #fca5a5;
}

.error-code code {
  color: #991b1b;
}

.section-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.collapse-trigger {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px;
  background: #fff;
  cursor: pointer;
  transition: all 0.3s;
}

.collapse-trigger:hover {
  background: #f9fafb;
}

.collapse-trigger .el-icon {
  transition: transform 0.3s;
  color: #6b7280;
}

.collapse-trigger .el-icon.expanded {
  transform: rotate(180deg);
}

.result-viewer {
  max-height: 400px;
  overflow-y: auto;
}
</style>
