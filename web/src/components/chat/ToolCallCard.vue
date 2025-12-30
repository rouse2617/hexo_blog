<template>
  <div class="tool-call-card" :class="statusClass">
    <div class="tool-header" @click="toggleCollapse">
      <div class="tool-info">
        <el-icon class="tool-icon" :size="18">
          <component :is="statusIcon" />
        </el-icon>
        <span class="tool-name">{{ toolCall.name }}</span>
      </div>
      <div class="header-actions">
        <el-tag :type="statusTagType" size="small">{{ statusText }}</el-tag>
        <el-icon class="collapse-icon" :class="{ collapsed: isCollapsed }">
          <ArrowDown />
        </el-icon>
      </div>
    </div>

    <transition name="collapse">
      <div class="tool-body" v-show="!isCollapsed" ref="bodyRef">
        <div class="tool-section" v-if="toolCall.arguments">
          <div class="section-title">
            <span>参数</span>
            <el-button
              type="text"
              size="small"
              :icon="CopyDocument"
              @click="copyArguments"
              class="copy-btn"
            >
              复制
            </el-button>
          </div>
          <pre class="section-content" :class="{ 'has-scroll': needsScroll }"><code>{{ formattedArguments }}</code></pre>
        </div>
        <div class="tool-section" v-if="toolCall.result">
          <div class="section-title">
            <span>结果</span>
            <div class="result-actions">
              <el-button
                type="text"
                size="small"
                :icon="CopyDocument"
                @click="copyResult"
                class="copy-btn"
              >
                复制
              </el-button>
              <el-button
                v-if="isCommand"
                type="text"
                size="small"
                :icon="Promotion"
                @click="recommendExecute"
                class="copy-btn"
              >
                推荐执行
              </el-button>
            </div>
          </div>
          <pre class="section-content result" :class="{ 'has-scroll': needsScroll }"><code>{{ toolCall.result }}</code></pre>
        </div>
        <div class="tool-section" v-if="toolCall.error">
          <div class="section-title error-title">
            <span>错误</span>
            <el-button
              type="text"
              size="small"
              :icon="CopyDocument"
              @click="copyError"
              class="copy-btn"
            >
              复制
            </el-button>
          </div>
          <pre class="section-content error-content"><code>{{ toolCall.error }}</code></pre>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import type { ToolCall } from '@/api/chat'
import { Loading, Check, Close, Clock, ArrowDown, CopyDocument, Promotion } from '@element-plus/icons-vue'

const props = defineProps<{
  toolCall: ToolCall
}>()

const emit = defineEmits<{
  recommendExecute: [command: string]
}>()

const bodyRef = ref<HTMLElement>()
const isCollapsed = ref(false)
const needsScroll = ref(false)

// Auto-collapse logic: collapse if content > 5 lines or height > 300px
// Error state always expanded
onMounted(async () => {
  await nextTick()
  if (props.toolCall.status === 'error') {
    isCollapsed.value = false
    return
  }

  const bodyElement = bodyRef.value
  if (bodyElement) {
    const contentElements = bodyElement.querySelectorAll('.section-content code')
    contentElements.forEach((el) => {
      const lines = el.textContent?.split('\n').length || 0
      const height = el.scrollHeight
      if (lines > 5 || height > 300) {
        needsScroll.value = true
        isCollapsed.value = true
      }
    })
  }
})

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}

const statusClass = computed(() => {
  return `status-${props.toolCall.status || 'pending'}`
})

const statusIcon = computed(() => {
  switch (props.toolCall.status) {
    case 'running':
      return Loading
    case 'success':
      return Check
    case 'error':
      return Close
    default:
      return Clock
  }
})

const statusTagType = computed(() => {
  switch (props.toolCall.status) {
    case 'running':
      return 'warning'
    case 'success':
      return 'success'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
})

const statusText = computed(() => {
  switch (props.toolCall.status) {
    case 'running':
      return '执行中'
    case 'success':
      return '成功'
    case 'error':
      return '失败'
    default:
      return '等待中'
  }
})

const formattedArguments = computed(() => {
  try {
    const args = JSON.parse(props.toolCall.arguments)
    return JSON.stringify(args, null, 2)
  } catch {
    return props.toolCall.arguments
  }
})

// Check if result looks like a shell command
const isCommand = computed(() => {
  if (!props.toolCall.result) return false
  const result = props.toolCall.result.trim()
  return /^[a-zA-Z0-9_\-\/]+\s+/.test(result) || result.startsWith('sudo') || result.startsWith('kubectl')
})

const copyArguments = async () => {
  try {
    await navigator.clipboard.writeText(formattedArguments.value)
    ElMessage.success('参数已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyResult = async () => {
  try {
    await navigator.clipboard.writeText(props.toolCall.result || '')
    ElMessage.success('结果已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyError = async () => {
  try {
    await navigator.clipboard.writeText(props.toolCall.error || '')
    ElMessage.success('错误信息已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const recommendExecute = () => {
  if (props.toolCall.result) {
    emit('recommendExecute', props.toolCall.result)
  }
}
</script>

<style scoped>
.tool-call-card {
  background: #f8f9fa;
  border-radius: 8px;
  border-left: 3px solid #909399;
  margin: 10px 0;
  overflow: hidden;
}

.tool-call-card.status-running {
  border-left-color: #e6a23c;
}

.tool-call-card.status-success {
  border-left-color: #67c23a;
}

.tool-call-card.status-error {
  border-left-color: #f56c6c;
}

.tool-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid #ebeef5;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s;
}

.tool-header:hover {
  background: rgba(0, 0, 0, 0.04);
}

.tool-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tool-icon {
  color: #409eff;
}

.tool-name {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.collapse-icon {
  transition: transform 0.3s;
  color: #909399;
}

.collapse-icon.collapsed {
  transform: rotate(-90deg);
}

.tool-body {
  padding: 12px 15px;
}

.tool-section {
  margin-bottom: 10px;
}

.tool-section:last-child {
  margin-bottom: 0;
}

.section-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
}

.section-title.error-title {
  color: #f56c6c;
}

.result-actions {
  display: flex;
  gap: 4px;
}

.copy-btn {
  font-size: 11px;
  padding: 2px 8px;
  height: auto;
  color: #409eff;
}

.copy-btn:hover {
  color: #66b1ff;
}

.section-content {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px;
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
}

.section-content.has-scroll {
  max-height: 300px;
  overflow-y: auto;
}

.section-content code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #606266;
  white-space: pre;
}

.section-content.result {
  background: #f0f9eb;
  border-color: #e1f3d8;
}

.status-error .section-content.result {
  background: #fef0f0;
  border-color: #fde2e2;
}

.error-content {
  background: #fef0f0;
  border-color: #fde2e2;
  color: #f56c6c;
}

/* Collapse transition */
.collapse-enter-active,
.collapse-leave-active {
  transition: all 0.3s ease;
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  max-height: 0;
  opacity: 0;
}

.collapse-enter-to,
.collapse-leave-from {
  max-height: 500px;
  opacity: 1;
}
</style>
