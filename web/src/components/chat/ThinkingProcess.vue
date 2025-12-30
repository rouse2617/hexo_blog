<template>
  <div class="thinking-process" v-if="isVisible">
    <div class="thinking-header" @click="toggleExpand">
      <div class="thinking-status">
        <el-icon class="is-loading" :size="16"><Loading /></el-icon>
        <span class="status-text">{{ currentStatusText }}</span>
        <span v-if="totalElapsedTime" class="elapsed-time">{{ formatElapsedTime(totalElapsedTime) }}</span>
      </div>
      <el-icon class="expand-icon" :class="{ expanded: isExpanded }">
        <ArrowDown />
      </el-icon>
    </div>

    <el-collapse-transition>
      <div v-show="isExpanded" class="thinking-details">
        <el-timeline>
          <el-timeline-item
            v-for="(step, index) in enhancedSteps"
            :key="step.step"
            :type="getStepType(step)"
            :hollow="step.status === 'calling_llm'"
            :timestamp="formatTime(step.timestamp)"
            placement="top"
          >
            <div class="step-content">
              <div class="step-header">
                <div class="step-title">
                  <el-icon v-if="step.status === 'calling_llm'" class="is-loading"><Loading /></el-icon>
                  <el-icon v-else-if="step.status === 'executing_tools'" class="tool-icon"><Tools /></el-icon>
                  <el-icon v-else-if="step.status === 'reading_file'" class="file-icon"><Document /></el-icon>
                  <el-icon v-else-if="step.status === 'searching'" class="search-icon"><Search /></el-icon>
                  <el-icon v-else-if="step.status === 'error'" class="error-icon"><CircleClose /></el-icon>
                  <el-icon v-else><DataAnalysis /></el-icon>
                  <span>{{ getStepTitle(step) }}</span>
                </div>
                <span v-if="step.elapsedTime" class="step-elapsed">{{ formatElapsedTime(step.elapsedTime) }}</span>
              </div>
              <div class="step-description">{{ getSemanticDescription(step) }}</div>

              <!-- 文件引用列表 -->
              <div v-if="step.fileReferences && step.fileReferences.length > 0" class="file-references">
                <div
                  v-for="file in step.fileReferences"
                  :key="file.path + (file.lineNumber || '')"
                  class="file-reference-item"
                  @click="handleFileClick(file)"
                >
                  <el-icon class="file-icon"><Document /></el-icon>
                  <span class="file-path">{{ file.name || file.path }}</span>
                  <span v-if="file.lineNumber" class="line-number">:{{ file.lineNumber }}</span>
                  <span v-else-if="file.lineRange" class="line-range">:{{ file.lineRange[0] }}-{{ file.lineRange[1] }}</span>
                </div>
              </div>

              <!-- 工具调用详情 -->
              <div v-if="step.toolCalls && step.toolCalls.length > 0" class="tool-calls-list">
                <div
                  v-for="toolCall in step.toolCalls"
                  :key="toolCall.id"
                  class="tool-call-item"
                  :class="{ 'is-error': toolCall.status === 'error' }"
                >
                  <div class="tool-call-header">
                    <div class="tool-call-status">
                      <el-icon v-if="toolCall.status === 'running'" class="is-loading"><Loading /></el-icon>
                      <el-icon v-else-if="toolCall.status === 'success'" class="success"><CircleCheck /></el-icon>
                      <el-icon v-else-if="toolCall.status === 'error'" class="error"><CircleClose /></el-icon>
                      <el-icon v-else class="pending"><Clock /></el-icon>
                      <span class="tool-name">{{ toolCall.tool }}</span>
                    </div>
                    <span v-if="toolCall.elapsedTime" class="tool-elapsed">{{ formatElapsedTime(toolCall.elapsedTime) }}</span>
                  </div>
                  <div class="tool-call-params">
                    <el-text type="info" size="small">参数: </el-text>
                    <code>{{ formatParams(toolCall.params) }}</code>
                  </div>
                  <div v-if="toolCall.result" class="tool-call-result">
                    <el-text type="info" size="small">结果: </el-text>
                    <code class="result-code">{{ formatResult(toolCall.result) }}</code>
                  </div>
                  <div v-if="toolCall.error" class="tool-call-error">
                    <el-text type="danger" size="small">错误: {{ toolCall.error }}</el-text>
                  </div>
                </div>
              </div>

              <!-- 错误信息和重试选项 -->
              <div v-if="step.error" class="step-error">
                <div class="error-content">
                  <el-icon class="error-icon"><WarningFilled /></el-icon>
                  <div class="error-details">
                    <span class="error-message">{{ step.error.message }}</span>
                    <span v-if="step.error.code" class="error-code">错误码: {{ step.error.code }}</span>
                    <span v-if="step.error.details" class="error-extra">{{ step.error.details }}</span>
                  </div>
                </div>
                <el-button
                  v-if="step.error.retryable"
                  type="primary"
                  size="small"
                  @click.stop="handleRetry(index)"
                  class="retry-button"
                >
                  <el-icon><RefreshRight /></el-icon>
                  重试
                </el-button>
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { 
  Loading, 
  ArrowDown, 
  Tools, 
  DataAnalysis, 
  CircleCheck, 
  CircleClose,
  Document,
  Search,
  Clock,
  WarningFilled,
  RefreshRight
} from '@element-plus/icons-vue'
import type { ThinkingStep } from '@/stores/chat'
import type { ThinkingStatus } from '@/api/chat'
import type { 
  EnhancedThinkingStep, 
  ThinkingFileReference,
  ThinkingStepStatus
} from '@/types/chat-ui'
import { getThinkingStepDescription } from '@/types/chat-ui'

const props = defineProps<{
  thinkingSteps: ThinkingStep[]
  currentStatus: ThinkingStatus | null
  isLoading: boolean
}>()

const emit = defineEmits<{
  (e: 'retry', stepIndex: number): void
  (e: 'fileClick', file: ThinkingFileReference): void
}>()

const isExpanded = ref(true)

// Track start times for elapsed time calculation
const stepStartTimes = ref<Map<number, number>>(new Map())

const isVisible = computed(() => {
  return props.isLoading && (props.currentStatus || props.thinkingSteps.length > 0)
})

// Enhanced steps with elapsed time and file references
const enhancedSteps = computed<EnhancedThinkingStep[]>(() => {
  return props.thinkingSteps.map((step, index) => {
    const startTime = stepStartTimes.value.get(step.step) || step.timestamp
    const endTime = getStepEndTime(step, index)
    const elapsedTime = endTime ? endTime - startTime : Date.now() - startTime
    
    // Extract file references from step content and tool calls
    const fileReferences = extractFileReferences(step)
    
    // Enhance tool calls with elapsed time
    const enhancedToolCalls = step.toolCalls?.map(tc => ({
      ...tc,
      elapsedTime: tc.status === 'success' || tc.status === 'error' 
        ? calculateToolElapsedTime(tc)
        : undefined
    }))
    
    // Check for errors
    const hasError = step.toolCalls?.some(tc => tc.status === 'error')
    const errorToolCall = step.toolCalls?.find(tc => tc.status === 'error')
    
    return {
      ...step,
      startTime,
      endTime,
      elapsedTime: step.status !== 'calling_llm' || endTime ? elapsedTime : undefined,
      fileReferences,
      toolCalls: enhancedToolCalls,
      error: hasError && errorToolCall ? {
        message: errorToolCall.error || '执行失败',
        retryable: true
      } : undefined
    } as EnhancedThinkingStep
  })
})

// Calculate total elapsed time
const totalElapsedTime = computed(() => {
  if (props.thinkingSteps.length === 0) return 0
  const firstStep = props.thinkingSteps[0]
  const startTime = stepStartTimes.value.get(firstStep.step) || firstStep.timestamp
  return Date.now() - startTime
})

// Watch for new steps to record start times
watch(() => props.thinkingSteps, (newSteps) => {
  newSteps.forEach(step => {
    if (!stepStartTimes.value.has(step.step)) {
      stepStartTimes.value.set(step.step, Date.now())
    }
  })
}, { deep: true, immediate: true })

// Get semantic description based on step status and context
const currentStatusText = computed(() => {
  if (!props.currentStatus) {
    return 'AI 正在思考...'
  }
  
  const status = props.currentStatus.status as ThinkingStepStatus
  const content = props.currentStatus.content
  
  // Try to extract context from content
  const context = extractContextFromContent(content)
  
  return getThinkingStepDescription(status, context)
})

function getSemanticDescription(step: EnhancedThinkingStep): string {
  const context = extractContextFromContent(step.content)
  
  // If there are tool calls, use tool name as context
  if (step.toolCalls && step.toolCalls.length > 0) {
    const runningTool = step.toolCalls.find(tc => tc.status === 'running')
    if (runningTool) {
      context.toolName = runningTool.tool
    }
  }
  
  return getThinkingStepDescription(step.status as ThinkingStepStatus, context)
}

function extractContextFromContent(content: string): { filename?: string; hostname?: string; command?: string; toolName?: string } {
  const context: { filename?: string; hostname?: string; command?: string; toolName?: string } = {}
  
  // Extract filename patterns
  const fileMatch = content.match(/(?:读取|查看|分析)\s*[`"]?([\/\w\-\.]+\.\w+)[`"]?/i)
  if (fileMatch) {
    context.filename = fileMatch[1]
  }
  
  // Extract hostname patterns
  const hostMatch = content.match(/(?:主机|服务器|host)\s*[`"]?([a-zA-Z0-9\-\.]+)[`"]?/i)
  if (hostMatch) {
    context.hostname = hostMatch[1]
  }
  
  // Extract command patterns
  const cmdMatch = content.match(/(?:执行|运行|command)\s*[`"]?([a-zA-Z0-9\-_]+)[`"]?/i)
  if (cmdMatch) {
    context.command = cmdMatch[1]
  }
  
  return context
}

function extractFileReferences(step: ThinkingStep): ThinkingFileReference[] {
  const references: ThinkingFileReference[] = []
  
  // Extract from content using regex patterns
  const filePatterns = [
    /(?:文件|file|读取|查看)\s*[`"]?([\/\w\-\.]+(?:\.\w+)?)[`"]?(?::(\d+))?/gi,
    /([\/\w\-\.]+\.\w+)(?::(\d+)(?:-(\d+))?)?/g
  ]
  
  for (const pattern of filePatterns) {
    let match
    while ((match = pattern.exec(step.content)) !== null) {
      const path = match[1]
      // Filter out common non-file patterns
      if (path && !path.startsWith('http') && path.includes('/') || path.includes('.')) {
        const ref: ThinkingFileReference = {
          path,
          name: path.split('/').pop() || path,
          mentionedAt: step.timestamp
        }
        
        if (match[2]) {
          ref.lineNumber = parseInt(match[2], 10)
        }
        if (match[3]) {
          ref.lineRange = [parseInt(match[2], 10), parseInt(match[3], 10)]
        }
        
        // Avoid duplicates
        if (!references.some(r => r.path === ref.path && r.lineNumber === ref.lineNumber)) {
          references.push(ref)
        }
      }
    }
  }
  
  // Also extract from tool call params
  if (step.toolCalls) {
    for (const tc of step.toolCalls) {
      try {
        const params = JSON.parse(tc.params)
        if (params.file || params.path || params.filename) {
          const path = params.file || params.path || params.filename
          const ref: ThinkingFileReference = {
            path,
            name: path.split('/').pop() || path,
            lineNumber: params.line || params.lineNumber,
            mentionedAt: step.timestamp
          }
          if (!references.some(r => r.path === ref.path)) {
            references.push(ref)
          }
        }
      } catch {
        // Ignore parse errors
      }
    }
  }
  
  return references
}

function getStepEndTime(step: ThinkingStep, index: number): number | undefined {
  // If all tool calls are complete, use the latest completion time
  if (step.toolCalls && step.toolCalls.length > 0) {
    const allComplete = step.toolCalls.every(tc => tc.status === 'success' || tc.status === 'error')
    if (allComplete) {
      return Date.now()
    }
  }
  
  // If there's a next step, use its start time
  if (index < props.thinkingSteps.length - 1) {
    return props.thinkingSteps[index + 1].timestamp
  }
  
  // If step is analyzing_results, it's likely complete
  if (step.status === 'analyzing_results') {
    return Date.now()
  }
  
  return undefined
}

function calculateToolElapsedTime(_toolCall: NonNullable<ThinkingStep['toolCalls']>[number]): number | undefined {
  // This is a simplified calculation - in a real implementation,
  // you'd track actual start/end times from the SSE events
  return undefined
}

const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}

const getStepType = (step: EnhancedThinkingStep) => {
  if (step.error || step.toolCalls?.some(t => t.status === 'error')) {
    return 'danger'
  }
  if (step.status === 'analyzing_results') {
    return 'success'
  }
  if (step.status === 'executing_tools') {
    return 'warning'
  }
  return 'primary'
}

const getStepTitle = (step: EnhancedThinkingStep) => {
  switch (step.status) {
    case 'calling_llm':
      return `第 ${step.step} 轮 - 调用 AI 模型`
    case 'executing_tools':
      return `第 ${step.step} 轮 - 执行工具`
    case 'analyzing_results':
      return `第 ${step.step} 轮 - 分析结果`
    case 'reading_file':
      return `第 ${step.step} 轮 - 读取文件`
    case 'searching':
      return `第 ${step.step} 轮 - 检索方案`
    case 'error':
      return `第 ${step.step} 轮 - 执行出错`
    default:
      return `第 ${step.step} 轮`
  }
}

const formatTime = (timestamp: number) => {
  return new Date(timestamp).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const formatElapsedTime = (ms: number) => {
  if (ms < 1000) {
    return `${ms}ms`
  }
  const seconds = ms / 1000
  if (seconds < 60) {
    return `${seconds.toFixed(1)}s`
  }
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds.toFixed(0)}s`
}

const formatParams = (params: string) => {
  try {
    const parsed = JSON.parse(params)
    return JSON.stringify(parsed, null, 2)
  } catch {
    return params
  }
}

const formatResult = (result: string) => {
  try {
    const parsed = JSON.parse(result)
    const str = JSON.stringify(parsed, null, 2)
    // 截断过长的结果
    if (str.length > 500) {
      return str.substring(0, 500) + '...'
    }
    return str
  } catch {
    if (result.length > 500) {
      return result.substring(0, 500) + '...'
    }
    return result
  }
}

const handleRetry = (stepIndex: number) => {
  emit('retry', stepIndex)
}

const handleFileClick = (file: ThinkingFileReference) => {
  emit('fileClick', file)
}
</script>


<style scoped>
.thinking-process {
  background: linear-gradient(135deg, #f0f7ff 0%, #e8f4f8 100%);
  border: 1px solid #d4e5f7;
  border-radius: 8px;
  margin: 12px 20px;
  overflow: hidden;
}

.thinking-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s;
}

.thinking-header:hover {
  background-color: rgba(64, 158, 255, 0.05);
}

.thinking-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #409eff;
  font-size: 14px;
  font-weight: 500;
}

.status-text {
  color: #606266;
}

.elapsed-time {
  color: #909399;
  font-size: 12px;
  font-weight: normal;
  padding: 2px 6px;
  background: rgba(144, 147, 153, 0.1);
  border-radius: 4px;
}

.expand-icon {
  color: #909399;
  transition: transform 0.3s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.thinking-details {
  padding: 0 16px 16px;
  border-top: 1px solid #e4e7ed;
  background: #fff;
}

.thinking-details :deep(.el-timeline) {
  padding-top: 16px;
}

.thinking-details :deep(.el-timeline-item__wrapper) {
  padding-left: 20px;
}

.step-content {
  background: #fafafa;
  border-radius: 6px;
  padding: 12px;
}

.step-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.step-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #303133;
}

.step-elapsed {
  color: #909399;
  font-size: 12px;
  padding: 2px 6px;
  background: rgba(144, 147, 153, 0.1);
  border-radius: 4px;
}

.step-description {
  color: #606266;
  font-size: 13px;
  margin-bottom: 8px;
}

/* File references */
.file-references {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 8px 0;
}

.file-reference-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: #ecf5ff;
  border: 1px solid #d9ecff;
  border-radius: 4px;
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
  transition: all 0.2s;
}

.file-reference-item:hover {
  background: #d9ecff;
  border-color: #409eff;
}

.file-reference-item .file-icon {
  font-size: 14px;
}

.file-reference-item .file-path {
  font-family: 'Monaco', 'Menlo', monospace;
}

.file-reference-item .line-number,
.file-reference-item .line-range {
  color: #67c23a;
  font-weight: 500;
}

/* Tool calls */
.tool-calls-list {
  margin-top: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tool-call-item {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 10px;
  font-size: 12px;
}

.tool-call-item.is-error {
  border-color: #f56c6c;
  background: #fef0f0;
}

.tool-call-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.tool-call-status {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tool-name {
  font-weight: 600;
  color: #409eff;
}

.tool-elapsed {
  color: #909399;
  font-size: 11px;
}

.tool-call-header .success {
  color: #67c23a;
}

.tool-call-header .error {
  color: #f56c6c;
}

.tool-call-header .pending {
  color: #909399;
}

.tool-call-params,
.tool-call-result {
  margin-top: 4px;
}

.tool-call-params code,
.tool-call-result code {
  display: block;
  background: #f5f7fa;
  padding: 6px 8px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 150px;
  overflow-y: auto;
  margin-top: 4px;
}

.tool-call-error {
  margin-top: 6px;
  padding: 6px 8px;
  background: #fef0f0;
  border-radius: 4px;
}

/* Step error */
.step-error {
  margin-top: 12px;
  padding: 12px;
  background: #fef0f0;
  border: 1px solid #fbc4c4;
  border-radius: 6px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.error-content {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex: 1;
}

.error-content .error-icon {
  color: #f56c6c;
  font-size: 18px;
  flex-shrink: 0;
  margin-top: 2px;
}

.error-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.error-message {
  color: #f56c6c;
  font-weight: 500;
  font-size: 13px;
}

.error-code {
  color: #909399;
  font-size: 11px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.error-extra {
  color: #606266;
  font-size: 12px;
}

.retry-button {
  flex-shrink: 0;
}

/* Icon colors */
.tool-icon {
  color: #e6a23c;
}

.file-icon {
  color: #409eff;
}

.search-icon {
  color: #67c23a;
}

.error-icon {
  color: #f56c6c;
}

/* Animations */
.is-loading {
  animation: rotating 1.5s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Dark mode support */
:root.dark .thinking-process {
  background: linear-gradient(135deg, #1a2332 0%, #1e2a3a 100%);
  border-color: #3a4a5a;
}

:root.dark .thinking-header:hover {
  background-color: rgba(64, 158, 255, 0.1);
}

:root.dark .thinking-details {
  background: #1a1a1a;
  border-color: #3a4a5a;
}

:root.dark .step-content {
  background: #2d2d2d;
}

:root.dark .step-title {
  color: #e2e8f0;
}

:root.dark .step-description {
  color: #94a3b8;
}

:root.dark .file-reference-item {
  background: #1e3a5f;
  border-color: #2d4a6f;
}

:root.dark .file-reference-item:hover {
  background: #2d4a6f;
  border-color: #409eff;
}

:root.dark .tool-call-item {
  background: #2d2d2d;
  border-color: #404040;
}

:root.dark .tool-call-item.is-error {
  background: #3d2020;
  border-color: #f56c6c;
}

:root.dark .tool-call-params code,
:root.dark .tool-call-result code {
  background: #1a1a1a;
  color: #e2e8f0;
}

:root.dark .step-error {
  background: #3d2020;
  border-color: #5a3030;
}
</style>
