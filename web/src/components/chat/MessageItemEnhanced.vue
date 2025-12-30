<template>
  <div
    class="message-item-enhanced"
    :class="messageClass"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
    :role="message.role === 'assistant' ? 'article' : 'comment'"
    :aria-label="`${roleName} message at ${formatTime}`"
  >
    <!-- 消息主体 - 居中容器 -->
    <div class="message-card">
      <div class="message-main">
        <!-- 头像 -->
        <div class="message-avatar" :aria-hidden="true">
          <el-avatar :size="40" :style="avatarStyle">
            <el-icon v-if="message.role === 'assistant'"><Monitor /></el-icon>
            <el-icon v-else><User /></el-icon>
          </el-avatar>
        </div>

        <!-- 内容区域 -->
        <div class="message-content">
          <!-- 头部信息 -->
          <div class="message-header">
            <div class="message-meta">
              <span class="message-role">{{ roleName }}</span>
              <span class="message-time">{{ formatTime }}</span>
              <span v-if="executionTime" class="message-duration">
                <el-icon><Clock /></el-icon>
                {{ executionTime }}
              </span>
            </div>

            <!-- 快捷操作 -->
            <MessageActions
              ref="actionsRef"
              :message="message"
              :is-loading="isLoading"
              @copy="handleCopy"
              @regenerate="handleRegenerate"
              @continue="handleContinue"
              @feedback="handleFeedback"
              @export="handleExport"
              @share="handleShare"
            />
          </div>

          <!-- 消息正文 -->
          <div class="message-body">
            <!-- Markdown 内容 -->
            <div v-if="message.role === 'assistant'" class="markdown-body" v-html="renderedContent"></div>
            <div v-else class="plain-text">{{ message.content }}</div>

            <!-- 工具调用展示（使用增强版卡片） -->
            <div v-if="message.toolCalls && message.toolCalls.length > 0" class="tool-calls">
              <EnhancedToolCallCard
                v-for="toolCall in message.toolCalls"
                :key="toolCall.id"
                :tool-call="toolCall"
                @rerun="handleToolRerun"
              />
            </div>

            <!-- 数据可视化（条件显示） -->
            <DataVisualization
              v-if="shouldShowVisualization"
              :data="visualizationData"
            />
          </div>
        </div>
      </div>

      <!-- 追问建议（仅助手消息且完成加载） -->
      <FollowUpQuestions
        v-if="message.role === 'assistant' && !isLoading && showFollowUp"
        :last-message="message.content"
        :last-tool="lastToolName"
        :context="extractContext()"
        @select="handleFollowUpSelect"
      />
    </div>

    <!-- 消息分隔线（多条消息之间） -->
    <div v-if="showDivider" class="message-divider">
      <span class="divider-text">{{ dividerText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, defineAsyncComponent } from 'vue'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import type { Message } from '@/api/chat'
import { Monitor, User, Clock } from '@element-plus/icons-vue'

// Lazy load heavy components for better performance
const MessageActions = defineAsyncComponent(() => import('./MessageActions.vue'))
const EnhancedToolCallCard = defineAsyncComponent(() => import('./EnhancedToolCallCard.vue'))
const DataVisualization = defineAsyncComponent(() => import('./DataVisualization.vue'))
const FollowUpQuestions = defineAsyncComponent(() => import('./FollowUpQuestions.vue'))

const props = defineProps<{
  message: Message
  isLoading?: boolean
  showDivider?: boolean
  dividerText?: string
}>()

const emit = defineEmits<{
  copy: [message: Message]
  regenerate: [message: Message]
  continue: [message: Message, question: string]
  feedback: [message: Message, type: 'good' | 'bad']
  export: [message: Message]
  share: [message: Message]
  'tool-rerun': [toolCall: any]
  'follow-up': [question: string]
}>()

const actionsRef = ref()
const isHovered = ref(false)

// 配置 marked
marked.setOptions({
  breaks: true,
  gfm: true
})

// 自定义渲染器处理代码高亮
const renderer = new marked.Renderer()
renderer.code = function (code: string, infostring?: string) {
  const language = (infostring || '').trim().split(/\s+/)[0]
  if (language && hljs.getLanguage(language)) {
    try {
      const highlighted = hljs.highlight(code, { language }).value
      // 添加复制按钮
      return `<div class="code-block-wrapper">
        <pre><code class="hljs language-${language}">${highlighted}</code></pre>
        <button class="copy-code-btn" onclick="copyCodeBlock(this)">复制</button>
      </div>`
    } catch {
      // 忽略错误
    }
  }
  const highlighted = hljs.highlightAuto(code).value
  return `<pre><code class="hljs">${highlighted}</code></pre>`
}
marked.use({ renderer })

// 计算属性
const messageClass = computed(() => ({
  'message-user': props.message.role === 'user',
  'message-assistant': props.message.role === 'assistant',
  'message-system': props.message.role === 'system',
  'is-hovered': isHovered.value
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
    minute: '2-digit',
    second: '2-digit'
  })
})

const executionTime = computed(() => {
  // 如果有工具调用，计算执行时间
  if (props.message.toolCalls && props.message.toolCalls.length > 0) {
    // 这里需要根据实际的工具调用时间计算
    // 暂时返回 null
    return null
  }
  return null
})

const renderedContent = computed(() => {
  if (!props.message.content) return ''
  try {
    return marked(props.message.content)
  } catch {
    return props.message.content
  }
})

const lastToolName = computed(() => {
  return props.message.toolCalls?.[0]?.name || ''
})

const showFollowUp = computed(() => {
  // 只在有内容且完成加载时显示
  return props.message.content &&
         props.message.content.length > 50 &&
         !props.isLoading
})

// 数据可视化相关
const shouldShowVisualization = computed(() => {
  // 检查是否包含可可视化的数据
  const hasMonitoringTool = props.message.toolCalls?.some(tc =>
    tc.name.includes('monitor') ||
    tc.name.includes('status') ||
    tc.name.includes('top')
  )
  return hasMonitoringTool && props.message.toolCalls?.[0]?.result
})

const visualizationData = computed(() => {
  // 解析工具调用结果
  if (!shouldShowVisualization.value) return null

  const toolResult = props.message.toolCalls?.[0]?.result
  if (!toolResult) return null

  try {
    return JSON.parse(toolResult)
  } catch {
    return null
  }
})

// 方法
const handleMouseEnter = () => {
  isHovered.value = true
  actionsRef.value?.show()
}

const handleMouseLeave = () => {
  isHovered.value = false
  actionsRef.value?.hide()
}

const handleCopy = (message: Message) => {
  emit('copy', message)
}

const handleRegenerate = (message: Message) => {
  emit('regenerate', message)
}

const handleContinue = (message: Message) => {
  const defaultQuestion = '请继续详细说明'
  emit('continue', message, defaultQuestion)
}

const handleFeedback = (message: Message, type: 'good' | 'bad') => {
  emit('feedback', message, type)
}

const handleExport = (message: Message) => {
  emit('export', message)
}

const handleShare = (message: Message) => {
  emit('share', message)
}

const handleToolRerun = (toolCall: any) => {
  emit('tool-rerun', toolCall)
}

const handleFollowUpSelect = (question: string) => {
  emit('follow-up', question)
}

const extractContext = () => {
  // 从消息内容中提取上下文关键词
  const content = props.message.content.toLowerCase()
  const keywords: string[] = []

  if (content.includes('cpu') || content.includes('memory')) {
    keywords.push('monitor')
  }
  if (content.includes('error') || content.includes('fail')) {
    keywords.push('error')
  }
  if (content.includes('log')) {
    keywords.push('log')
  }

  return keywords
}

// 暴露给父组件的方法
defineExpose({
  scrollToView: () => {
    // 滚动到视图
  }
})

// 全局复制代码块函数
if (typeof window !== 'undefined') {
  (window as any).copyCodeBlock = async (btn: HTMLElement) => {
    const wrapper = btn.parentElement
    const code = wrapper?.querySelector('code')?.textContent
    if (code) {
      try {
        await navigator.clipboard.writeText(code)
        btn.textContent = '已复制!'
        setTimeout(() => {
          btn.textContent = '复制'
        }, 2000)
      } catch {
        ElMessage.error('复制失败')
      }
    }
  }
}
</script>

<style scoped>
/**
 * MessageItemEnhanced Styles - Width Optimization
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
.message-item-enhanced {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--spacing-5, 20px);
  transition: background-color var(--transition-base, 0.2s);
}

.message-item-enhanced:hover {
  background-color: var(--color-surface-hover, rgba(0, 0, 0, 0.015));
}

.message-user {
  background-color: var(--color-message-user-bg, #fff);
}

.message-assistant {
  background-color: var(--color-message-assistant-bg, #f8fafc);
}

/* Message card wrapper - applies max-width and centering - Req 6.1, 6.2 */
.message-card {
  width: 100%;
  max-width: 800px;
}

/* Wider max-width for ultra-wide screens (> 1920px) - Req 6.1 */
@media (min-width: 1921px) {
  .message-card {
    max-width: 1000px;
  }
}

.message-main {
  display: flex;
  gap: var(--spacing-4, 16px);
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  flex: 1;
  min-width: 0;
}

/* Message header - Req 6.4 */
.message-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: var(--spacing-2, 8px);
  gap: var(--spacing-3, 12px);
}

.message-meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-3, 10px);
  flex-wrap: wrap;
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

.message-duration {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--text-xs, 12px);
  color: var(--color-success, #67c23a);
  background: #f0f9ff;
  padding: 2px 8px;
  border-radius: var(--radius-xl, 12px);
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

/* 代码块样式 - Req 6.3: max height 300px with overflow scrolling */
.markdown-body :deep(.code-block-wrapper) {
  position: relative;
  margin: var(--spacing-3, 10px) 0;
}

.markdown-body :deep(.copy-code-btn) {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 4px 12px;
  font-size: var(--text-xs, 12px);
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-md, 4px);
  cursor: pointer;
  transition: all var(--transition-base, 0.2s);
  z-index: 1;
}

.markdown-body :deep(.copy-code-btn:hover) {
  background: rgba(255, 255, 255, 0.2);
}

.markdown-body :deep(pre) {
  background: var(--color-code-bg, #1e1e1e);
  border-radius: var(--radius-lg, 8px);
  padding: var(--spacing-3, 12px) var(--spacing-4, 16px);
  overflow: auto;
  max-height: 300px;
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
  background: #f0f9ff;
  padding: var(--spacing-3, 10px) var(--spacing-4, 15px);
  border-radius: var(--radius-md, 4px);
}

.markdown-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: var(--spacing-3, 10px) 0;
  overflow: hidden;
  border-radius: var(--radius-lg, 8px);
  display: block;
  overflow-x: auto;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border: 1px solid var(--color-border, #ebeef5);
  padding: var(--spacing-3, 10px) var(--spacing-3, 12px);
  text-align: left;
}

.markdown-body :deep(th) {
  background: var(--color-bg-secondary, #f5f7fa);
  font-weight: var(--font-semibold, 600);
}

.markdown-body :deep(tr:hover) {
  background: var(--color-surface-hover, #f9fafb);
}

.tool-calls {
  margin-top: var(--spacing-4, 16px);
}

.message-divider {
  display: flex;
  align-items: center;
  margin: var(--spacing-6, 24px) 0 var(--spacing-4, 16px);
  padding: 0 var(--spacing-5, 20px);
  width: 100%;
  max-width: 800px;
}

@media (min-width: 1921px) {
  .message-divider {
    max-width: 1000px;
  }
}

.message-divider::before,
.message-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: linear-gradient(
    to right,
    transparent,
    var(--color-border, #e5e7eb),
    transparent
  );
}

.divider-text {
  padding: 0 var(--spacing-4, 16px);
  font-size: var(--text-xs, 12px);
  color: var(--color-text-tertiary, #9ca3af);
  background: inherit;
}

/* 响应式 */
@media (max-width: 768px) {
  .message-item-enhanced {
    padding: var(--spacing-4, 16px) var(--spacing-3, 12px);
  }

  .message-main {
    gap: var(--spacing-3, 12px);
  }

  .message-meta {
    gap: var(--spacing-2, 8px);
  }
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

[data-theme="dark"] .message-duration {
  background: rgba(103, 194, 58, 0.1);
}

[data-theme="dark"] .markdown-body :deep(code) {
  background: var(--color-gray-200, #404040);
}

[data-theme="dark"] .markdown-body :deep(blockquote) {
  background: rgba(64, 158, 255, 0.1);
}

[data-theme="dark"] .markdown-body :deep(th) {
  background: var(--color-bg-tertiary);
}

[data-theme="dark"] .markdown-body :deep(th),
[data-theme="dark"] .markdown-body :deep(td) {
  border-color: var(--color-border);
}

[data-theme="dark"] .markdown-body :deep(tr:hover) {
  background: var(--color-surface-hover);
}
</style>
