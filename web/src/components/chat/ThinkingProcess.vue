<template>
  <div class="thinking-process" v-if="isVisible">
    <div class="thinking-header" @click="toggleExpand">
      <div class="thinking-status">
        <el-icon class="is-loading" :size="16"><Loading /></el-icon>
        <span class="status-text">{{ currentStatusText }}</span>
      </div>
      <el-icon class="expand-icon" :class="{ expanded: isExpanded }">
        <ArrowDown />
      </el-icon>
    </div>

    <el-collapse-transition>
      <div v-show="isExpanded" class="thinking-details">
        <el-timeline>
          <el-timeline-item
            v-for="step in thinkingSteps"
            :key="step.step"
            :type="getStepType(step)"
            :hollow="step.status === 'calling_llm'"
            :timestamp="formatTime(step.timestamp)"
            placement="top"
          >
            <div class="step-content">
              <div class="step-title">
                <el-icon v-if="step.status === 'calling_llm'" class="is-loading"><Loading /></el-icon>
                <el-icon v-else-if="step.status === 'executing_tools'"><Tools /></el-icon>
                <el-icon v-else><DataAnalysis /></el-icon>
                <span>{{ getStepTitle(step) }}</span>
              </div>
              <div class="step-description">{{ step.content }}</div>

              <!-- 工具调用详情 -->
              <div v-if="step.toolCalls && step.toolCalls.length > 0" class="tool-calls-list">
                <div
                  v-for="toolCall in step.toolCalls"
                  :key="toolCall.id"
                  class="tool-call-item"
                  :class="{ 'is-error': toolCall.status === 'error' }"
                >
                  <div class="tool-call-header">
                    <el-icon v-if="toolCall.status === 'running'" class="is-loading"><Loading /></el-icon>
                    <el-icon v-else-if="toolCall.status === 'success'" class="success"><CircleCheck /></el-icon>
                    <el-icon v-else-if="toolCall.status === 'error'" class="error"><CircleClose /></el-icon>
                    <span class="tool-name">{{ toolCall.tool }}</span>
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
            </div>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Loading, ArrowDown, Tools, DataAnalysis, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import type { ThinkingStep } from '@/stores/chat'
import type { ThinkingStatus } from '@/api/chat'

const props = defineProps<{
  thinkingSteps: ThinkingStep[]
  currentStatus: ThinkingStatus | null
  isLoading: boolean
}>()

const isExpanded = ref(true)

const isVisible = computed(() => {
  return props.isLoading && (props.currentStatus || props.thinkingSteps.length > 0)
})

const currentStatusText = computed(() => {
  if (!props.currentStatus) {
    return 'AI 正在思考...'
  }
  switch (props.currentStatus.status) {
    case 'calling_llm':
      return '正在分析问题...'
    case 'executing_tools':
      return '正在执行工具...'
    case 'analyzing_results':
      return '正在分析结果...'
    default:
      return 'AI 正在思考...'
  }
})

const toggleExpand = () => {
  isExpanded.value = !isExpanded.value
}

const getStepType = (step: ThinkingStep) => {
  if (step.toolCalls?.some(t => t.status === 'error')) {
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

const getStepTitle = (step: ThinkingStep) => {
  switch (step.status) {
    case 'calling_llm':
      return `第 ${step.step} 轮 - 调用 AI 模型`
    case 'executing_tools':
      return `第 ${step.step} 轮 - 执行工具`
    case 'analyzing_results':
      return `第 ${step.step} 轮 - 分析结果`
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

.step-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 6px;
}

.step-description {
  color: #606266;
  font-size: 13px;
  margin-bottom: 8px;
}

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
  gap: 6px;
  margin-bottom: 6px;
}

.tool-name {
  font-weight: 600;
  color: #409eff;
}

.tool-call-header .success {
  color: #67c23a;
}

.tool-call-header .error {
  color: #f56c6c;
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
</style>
