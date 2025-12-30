<template>
  <div class="message-item" :class="messageClass">
    <div class="message-card">
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
/**
 * MessageItem Styles - Width Optimization
 * 
 * Implements Requirement 6: 消息气泡宽度优化
 * - Max width 800px (1000px on screens > 1920px)
 * - Center-aligned message cards
 * - Code blocks max height 300px with overflow scrolling
 * - Consistent padding and spacing
 * - Responsive font sizing
 * 
 * @requirements 6.1, 6.2, 6.3, 6.4, 6.5
 */

/* Message container - centers the message card - Req 6.2 */
.message-item {
  display: flex;
  justify-content: center;
  padding: var(--spacing-4, 16px) var(--spacing-5, 20px);
  transition: background-color var(--transition-base, 0.2s);
}

.message-item:hover {
  background-color: var(--color-surface-hover, rgba(0, 0, 0, 0.02));
}

.message-user {
  background-color: var(--color-message-user-bg, #fff);
}

.message-assistant {
  background-color: var(--color-message-assistant-bg, #f8fafc);
}

/* Message card wrapper - applies max-width and centering - Req 6.1, 6.2 */
.message-card {
  display: flex;
  gap: var(--spacing-3, 12px);
  width: 100%;
  max-width: 800px;
}

/* Wider max-width for ultra-wide screens (> 1920px) - Req 6.1 */
@media (min-width: 1921px) {
  .message-card {
    max-width: 1000px;
  }
}

.message-avatar {
  flex-shrink: 0;
}

/* Message content container - Req 6.4 */
.message-content {
  flex: 1;
  min-width: 0;
}

/* Message header - Req 6.4 */
.message-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3, 10px);
  margin-bottom: var(--spacing-2, 8px);
}

.message-role {
  font-weight: var(--font-semibold, 600);
  color: var(--color-text-primary, #303133);
  font-size: var(--text-base, 14px);
}

.message-time {
  font-size: var(--text-xs, 12px);
  color: var(--color-text-secondary, #909399);
}

/* Message body - Req 6.4, 6.5 */
.message-body {
  color: var(--color-text-secondary, #606266);
  line-height: var(--leading-relaxed, 1.7);
}

.plain-text {
  white-space: pre-wrap;
  word-break: break-word;
}

/* Markdown body - Req 6.5 responsive font sizing */
.markdown-body {
  font-size: var(--text-base, 14px);
}

/* Responsive font sizing for different screens - Req 6.5 */
@media (min-width: 1200px) {
  .markdown-body {
    font-size: var(--text-base, 14px);
  }
}

@media (min-width: 1600px) {
  .markdown-body {
    font-size: 15px;
  }
}

@media (min-width: 1921px) {
  .markdown-body {
    font-size: var(--text-lg, 16px);
  }
}

.markdown-body :deep(p) {
  margin: 0 0 var(--spacing-3, 10px);
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

/* Code blocks - Req 6.3: max height 300px with overflow scrolling */
.markdown-body :deep(pre) {
  background: var(--color-code-bg, #1e1e1e);
  border-radius: var(--radius-lg, 6px);
  padding: var(--spacing-3, 12px) var(--spacing-4, 16px);
  overflow: auto;
  margin: var(--spacing-3, 10px) 0;
  max-height: 300px;
  position: relative;
}

/* Scrollbar styling for code blocks */
.markdown-body :deep(pre::-webkit-scrollbar) {
  width: 6px;
  height: 6px;
}

.markdown-body :deep(pre::-webkit-scrollbar-track) {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.markdown-body :deep(pre::-webkit-scrollbar-thumb) {
  background: rgba(255, 255, 255, 0.3);
  border-radius: 3px;
}

.markdown-body :deep(pre::-webkit-scrollbar-thumb:hover) {
  background: rgba(255, 255, 255, 0.5);
}

.markdown-body :deep(pre code) {
  color: var(--color-code-text, #d4d4d4);
  font-family: var(--font-family-mono, 'Monaco', 'Menlo', 'Ubuntu Mono', monospace);
  font-size: var(--text-sm, 13px);
  line-height: var(--leading-normal, 1.5);
  display: block;
}

.markdown-body :deep(code) {
  background: var(--color-gray-100, #f0f0f0);
  padding: 2px 6px;
  border-radius: var(--radius-md, 4px);
  font-family: var(--font-family-mono, 'Monaco', 'Menlo', 'Ubuntu Mono', monospace);
  font-size: var(--text-sm, 13px);
}

.markdown-body :deep(pre code) {
  background: transparent;
  padding: 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: var(--spacing-5, 20px);
  margin: var(--spacing-3, 10px) 0;
}

.markdown-body :deep(li) {
  margin: var(--spacing-1, 5px) 0;
}

.markdown-body :deep(blockquote) {
  border-left: 4px solid var(--color-primary, #409eff);
  padding-left: var(--spacing-4, 15px);
  margin: var(--spacing-3, 10px) 0;
  color: var(--color-text-tertiary, #909399);
}

.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: var(--spacing-3, 10px) 0;
  overflow-x: auto;
  display: block;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--color-border, #ebeef5);
  padding: var(--spacing-2, 8px) var(--spacing-3, 12px);
  text-align: left;
}

.markdown-body :deep(th) {
  background: var(--color-bg-secondary, #f5f7fa);
  font-weight: var(--font-semibold, 600);
}

.tool-calls {
  margin-top: var(--spacing-3, 12px);
}

/* Dark mode support */
[data-theme="dark"] .message-user {
  background-color: var(--color-message-user-bg);
}

[data-theme="dark"] .message-assistant {
  background-color: var(--color-message-assistant-bg);
}

[data-theme="dark"] .message-role {
  color: var(--color-text-primary);
}

[data-theme="dark"] .message-time {
  color: var(--color-text-secondary);
}

[data-theme="dark"] .message-body {
  color: var(--color-text-secondary);
}

[data-theme="dark"] .markdown-body :deep(code) {
  background: var(--color-gray-200, #404040);
}

[data-theme="dark"] .markdown-body :deep(th) {
  background: var(--color-bg-tertiary);
}

[data-theme="dark"] .markdown-body :deep(th),
[data-theme="dark"] .markdown-body :deep(td) {
  border-color: var(--color-border);
}
</style>
