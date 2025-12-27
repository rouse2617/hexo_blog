<template>
  <div class="tool-call-card" :class="statusClass">
    <div class="tool-header">
      <div class="tool-info">
        <el-icon class="tool-icon" :size="18">
          <component :is="statusIcon" />
        </el-icon>
        <span class="tool-name">{{ toolCall.name }}</span>
      </div>
      <el-tag :type="statusTagType" size="small">{{ statusText }}</el-tag>
    </div>
    <div class="tool-body">
      <div class="tool-section" v-if="toolCall.arguments">
        <div class="section-title">参数</div>
        <pre class="section-content"><code>{{ formattedArguments }}</code></pre>
      </div>
      <div class="tool-section" v-if="toolCall.result">
        <div class="section-title">结果</div>
        <pre class="section-content result"><code>{{ toolCall.result }}</code></pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ToolCall } from '@/api/chat'
import { Loading, Check, Close, Clock } from '@element-plus/icons-vue'

const props = defineProps<{
  toolCall: ToolCall
}>()

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
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
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

.section-content code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #606266;
}

.section-content.result {
  background: #f0f9eb;
  border-color: #e1f3d8;
}

.status-error .section-content.result {
  background: #fef0f0;
  border-color: #fde2e2;
}
</style>
