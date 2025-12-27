<template>
  <div class="message-item" :class="messageClass">
    <div class="message-avatar">
      <el-avatar :size="36" :style="avatarStyle">
        <el-icon v-if="message.role === 'assistant'"><Monitor /></el-icon>
        <el-icon v-else><User /></el-icon>
      </el-avatar>
    </div>
    <div class="message-content">
      <div class="message-header">
        <span class="message-role">{{ roleName }}</span>
        <span class="message-time">{{ formatTime }}</span>
      </div>
      <div class="message-body">
        <div v-if="message.role === 'assistant'" class="markdown-body" v-html="renderedContent"></div>
        <div v-else class="plain-text">{{ message.content }}</div>

        <!-- 工具调用展示 -->
        <div v-if="message.toolCalls && message.toolCalls.length > 0" class="tool-calls">
          <ToolCallCard
            v-for="toolCall in message.toolCalls"
            :key="toolCall.id"
            :tool-call="toolCall"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import type { Message } from '@/api/chat'
import { Monitor, User } from '@element-plus/icons-vue'
import ToolCallCard from './ToolCallCard.vue'

const props = defineProps<{
  message: Message
}>()

// 配置 marked
marked.setOptions({
  breaks: true,
  gfm: true
})

// 自定义渲染器处理代码高亮 (marked v12+ API)
const renderer = new marked.Renderer()
renderer.code = function (code: string, infostring?: string) {
  const language = (infostring || '').trim().split(/\s+/)[0]
  if (language && hljs.getLanguage(language)) {
    try {
      const highlighted = hljs.highlight(code, { language }).value
      return `<pre><code class="hljs language-${language}">${highlighted}</code></pre>`
    } catch {
      // 忽略错误
    }
  }
  const highlighted = hljs.highlightAuto(code).value
  return `<pre><code class="hljs">${highlighted}</code></pre>`
}
marked.use({ renderer })

const messageClass = computed(() => ({
  'message-user': props.message.role === 'user',
  'message-assistant': props.message.role === 'assistant',
  'message-system': props.message.role === 'system'
}))

const avatarStyle = computed(() => {
  if (props.message.role === 'assistant') {
    return { backgroundColor: '#409eff' }
  }
  return { backgroundColor: '#67c23a' }
})

const roleName = computed(() => {
  switch (props.message.role) {
    case 'user':
      return '用户'
    case 'assistant':
      return 'AI 助手'
    case 'system':
      return '系统'
    default:
      return ''
  }
})

const formatTime = computed(() => {
  const timestamp = props.message.timestamp
  const date = timestamp ? new Date(timestamp) : new Date()
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  })
})

const renderedContent = computed(() => {
  if (!props.message.content) return ''
  try {
    return marked(props.message.content)
  } catch {
    return props.message.content
  }
})
</script>

<style scoped>
.message-item {
  display: flex;
  gap: 12px;
  padding: 16px 20px;
  transition: background-color 0.2s;
}

.message-item:hover {
  background-color: rgba(0, 0, 0, 0.02);
}

.message-user {
  background-color: #fff;
}

.message-assistant {
  background-color: #f8fafc;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.message-role {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.message-time {
  font-size: 12px;
  color: #909399;
}

.message-body {
  color: #606266;
  line-height: 1.7;
}

.plain-text {
  white-space: pre-wrap;
  word-break: break-word;
}

.markdown-body {
  font-size: 14px;
}

.markdown-body :deep(p) {
  margin: 0 0 10px;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(pre) {
  background: #1e1e1e;
  border-radius: 6px;
  padding: 12px 16px;
  overflow-x: auto;
  margin: 10px 0;
}

.markdown-body :deep(pre code) {
  color: #d4d4d4;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.markdown-body :deep(code) {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.markdown-body :deep(pre code) {
  background: transparent;
  padding: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 20px;
  margin: 10px 0;
}

.markdown-body :deep(li) {
  margin: 5px 0;
}

.markdown-body :deep(blockquote) {
  border-left: 4px solid #409eff;
  padding-left: 15px;
  margin: 10px 0;
  color: #909399;
}

.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid #ebeef5;
  padding: 8px 12px;
  text-align: left;
}

.markdown-body :deep(th) {
  background: #f5f7fa;
  font-weight: 600;
}

.tool-calls {
  margin-top: 12px;
}
</style>
