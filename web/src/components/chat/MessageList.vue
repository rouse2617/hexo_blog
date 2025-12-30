<template>
  <div class="message-list-container" role="log" aria-live="polite" aria-label="Chat messages">
    <!-- 空状态 - 显示智能推荐 -->
    <div v-if="messages.length === 0" class="empty-state-with-suggestions" role="status" aria-label="No messages">
      <div class="empty-state">
        <el-icon :size="64" color="#c0c4cc" aria-hidden="true"><ChatDotRound /></el-icon>
        <p class="empty-title">开始新的对话</p>
        <p class="empty-desc">输入您的问题，AI 助手将为您提供帮助</p>
      </div>

      <!-- 智能推荐（空状态时显示） -->
      <SmartSuggestions
        v-if="chatStore.showSuggestions"
        :current-context="getContextFromMessages()"
        @select="handleSuggestionSelect"
      />

      <!-- 快速命令 -->
      <QuickCommands @select="handleCommandSelect" />
    </div>

    <!-- 虚拟滚动列表 -->
    <template v-else>
      <!-- 加载更多历史消息按钮 -->
      <div v-if="hasMoreMessages" class="load-more">
        <el-button
          text
          :loading="loadingMore"
          @click="loadMoreMessages"
        >
          加载更多历史消息
        </el-button>
      </div>

      <div class="message-scroller" ref="messageListRef" @scroll="handleScroll" role="region" aria-label="Message list" tabindex="0">
        <!-- 会话切换加载骨架屏 -->
        <template v-if="sessionLoading">
          <div v-for="i in 3" :key="i" class="message-skeleton" role="status" :aria-label="`Loading message ${i}`">
            <div class="skeleton-avatar" aria-hidden="true"></div>
            <div class="skeleton-content" aria-hidden="true">
              <div class="skeleton-line skeleton-line-short"></div>
              <div class="skeleton-line"></div>
              <div class="skeleton-line skeleton-line-medium"></div>
            </div>
          </div>
        </template>

        <template v-else>
          <div
            v-for="(message, index) in visibleMessages"
            :key="getMessageKey(message, index)"
            class="message-item-wrapper"
          >
            <!-- 增强版消息组件 -->
            <MessageItemEnhanced
              :message="message"
              :is-loading="isLoading && index === visibleMessages.length - 1"
              :ref="el => setMessageRef(el, index)"
              @copy="handleCopy"
              @regenerate="handleRegenerate"
              @continue="handleContinue"
              @feedback="handleFeedback"
              @export="handleExport"
              @share="handleShare"
              @tool-rerun="handleToolRerun"
              @follow-up="handleFollowUp"
            />
          </div>

          <!-- 思考过程展示（移到滚动区域内） -->
          <ThinkingProcess
            :thinking-steps="thinkingSteps"
            :current-status="currentThinkingStatus"
            :is-loading="isLoading"
            @retry="handleThinkingRetry"
            @file-click="handleThinkingFileClick"
          />
        </template>
      </div>
    </template>

    <!-- 简单加载指示器（当没有思考步骤时显示） -->
    <div v-if="isLoading && props.thinkingSteps.length === 0 && !currentThinkingStatus" class="loading-indicator" role="status" aria-live="polite" aria-label="AI is thinking">
      <el-icon class="is-loading" :size="20" aria-hidden="true"><Loading /></el-icon>
      <span>AI 正在思考...</span>
    </div>

    <!-- 滚动到底部按钮 -->
    <transition name="fade">
      <div
        v-if="showScrollButton"
        class="scroll-to-bottom"
        @click="scrollToBottom"
        role="button"
        tabindex="0"
        aria-label="Scroll to bottom"
        @keydown.enter="scrollToBottom"
        @keydown.space.prevent="scrollToBottom"
      >
        <el-icon aria-hidden="true"><ArrowDown /></el-icon>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted, defineAsyncComponent } from 'vue'
import type { Message } from '@/api/chat'
import type { ThinkingStep } from '@/stores/chat'
import type { ThinkingStatus } from '@/api/chat'
import type { ThinkingFileReference } from '@/types/chat-ui'
import { ChatDotRound, Loading, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useChatStore } from '@/stores/chat'

// Lazy load heavy components for better performance
const MessageItemEnhanced = defineAsyncComponent(() => import('./MessageItemEnhanced.vue'))
const ThinkingProcess = defineAsyncComponent(() => import('./ThinkingProcess.vue'))
const SmartSuggestions = defineAsyncComponent(() => import('./SmartSuggestions.vue'))
const QuickCommands = defineAsyncComponent(() => import('./QuickCommands.vue'))

const props = withDefaults(defineProps<{
  messages: Message[]
  isLoading: boolean
  thinkingSteps?: ThinkingStep[]
  currentThinkingStatus?: ThinkingStatus | null
  sessionLoading?: boolean
}>(), {
  thinkingSteps: () => [],
  currentThinkingStatus: null,
  sessionLoading: false
})

const emit = defineEmits<{
  'send-message': [content: string]
  regenerate: [message: Message]
  'tool-rerun': [toolCall: any]
}>()

const chatStore = useChatStore()
const messageListRef = ref<HTMLElement | null>(null)
const messageRefs = ref<Map<number, any>>(new Map())
const showScrollButton = ref(false)
const loadingMore = ref(false)
const displayCount = ref(50) // 初始显示的消息数量
const BATCH_SIZE = 30 // 每次加载更多的数量

// 生成消息的唯一 key（解决 timestamp 重复问题）
const getMessageKey = (message: Message, index: number): string => {
  return `${message.role}-${message.timestamp}-${index}`
}

// 计算是否有更多消息可加载
const hasMoreMessages = computed(() => {
  return props.messages.length > displayCount.value
})

// 计算可见的消息（从最新的开始显示）
const visibleMessages = computed(() => {
  const total = props.messages.length
  if (total <= displayCount.value) {
    return props.messages
  }
  // 显示最后 displayCount 条消息
  return props.messages.slice(total - displayCount.value)
})

// 设置消息元素引用
const setMessageRef = (el: any, index: number) => {
  if (el) {
    messageRefs.value.set(index, el)
  } else {
    messageRefs.value.delete(index)
  }
}

// 从消息历史中提取上下文信息（用于智能推荐）
const getContextFromMessages = () => {
  const lastMessage = props.messages[props.messages.length - 1]
  const lastTool = lastMessage?.toolCalls?.[0]?.name
  const hasError = props.messages.some(m =>
    m.content?.toLowerCase().includes('error') ||
    m.toolCalls?.some(tc => tc.status === 'error')
  )

  return {
    lastTool,
    hasError,
    selectedHosts: chatStore.selectedHostIds
  }
}

// 智能推荐选择处理
const handleSuggestionSelect = (prompt: string) => {
  emit('send-message', prompt)
}

// 快速命令选择处理
const handleCommandSelect = (prompt: string) => {
  emit('send-message', prompt)
}

// 消息操作处理函数
const handleCopy = (message: Message) => {
  navigator.clipboard.writeText(message.content)
  ElMessage.success('已复制到剪贴板')
}

const handleRegenerate = (message: Message) => {
  emit('regenerate', message)
}

const handleContinue = (_message: Message, question: string) => {
  emit('send-message', question)
}

const handleFeedback = (message: Message, type: 'good' | 'bad') => {
  // TODO: 实现反馈提交
  console.log('Feedback:', type, message)
  ElMessage.success(type === 'good' ? '感谢您的反馈！' : '感谢您的反馈，我们会改进！')
}

const handleExport = (message: Message) => {
  const blob = new Blob([message.content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `message_${Date.now()}.md`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  ElMessage.success('消息已导出')
}

const handleShare = (message: Message) => {
  const shareText = message.content.substring(0, 100)
  navigator.clipboard.writeText(shareText)
  ElMessage.success('分享内容已复制')
}

const handleToolRerun = (toolCall: any) => {
  emit('tool-rerun', toolCall)
}

const handleFollowUp = (question: string) => {
  emit('send-message', question)
}

// 思考过程重试处理
const handleThinkingRetry = (stepIndex: number) => {
  console.log('Retry thinking step:', stepIndex)
  // TODO: Implement retry logic - this would typically re-trigger the AI processing
  ElMessage.info('正在重试...')
}

// 思考过程文件点击处理
const handleThinkingFileClick = (file: ThinkingFileReference) => {
  console.log('File clicked:', file)
  // TODO: Implement file preview modal
  // For now, show a message with the file info
  const lineInfo = file.lineNumber 
    ? `:${file.lineNumber}` 
    : file.lineRange 
      ? `:${file.lineRange[0]}-${file.lineRange[1]}` 
      : ''
  ElMessage.info(`打开文件: ${file.path}${lineInfo}`)
}

// 加载更多历史消息
const loadMoreMessages = async () => {
  loadingMore.value = true

  await nextTick()
  displayCount.value += BATCH_SIZE

  // 等待 DOM 更新
  await nextTick()
  loadingMore.value = false
}

// 自动滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (messageListRef.value) {
      messageListRef.value.scrollTop = messageListRef.value.scrollHeight
    }
  })
}

// 检查是否在底部附近
const isNearBottom = () => {
  if (!messageListRef.value) return true
  const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
  // 如果距离底部小于 100px，认为在底部附近
  return scrollHeight - scrollTop - clientHeight < 100
}

// 处理滚动事件
const handleScroll = () => {
  if (!messageListRef.value) return
  showScrollButton.value = !isNearBottom()
}

// 监听消息变化，自动滚动
watch(
  () => props.messages.length,
  (newLength, oldLength) => {
    // 只有在消息数量增加时才自动滚动
    if (newLength > (oldLength || 0)) {
      // 新消息时，如果显示数量小于初始值则重置
      if (displayCount.value < 50) {
        displayCount.value = 50
      }
      // 只有在底部附近时才自动滚动
      if (isNearBottom()) {
        scrollToBottom()
      }
    }
  }
)

// 监听最后一条消息内容变化（流式输出）
watch(
  () => {
    const lastMessage = props.messages[props.messages.length - 1]
    return lastMessage?.content
  },
  () => {
    if (isNearBottom()) {
      scrollToBottom()
    }
  }
)

// 监听加载状态
watch(
  () => props.isLoading,
  (loading) => {
    if (loading && isNearBottom()) {
      scrollToBottom()
    }
  }
)

// 监听思考步骤变化
watch(
  () => props.thinkingSteps?.length,
  () => {
    if (isNearBottom()) {
      scrollToBottom()
    }
  }
)

// 监听当前思考状态变化
watch(
  () => props.currentThinkingStatus,
  () => {
    if (isNearBottom()) {
      scrollToBottom()
    }
  },
  { deep: true }
)

onMounted(() => {
  // 初始化时滚动到底部
  nextTick(() => {
    scrollToBottom()
  })
})

onUnmounted(() => {
  // 清理工作
  messageRefs.value.clear()
})

defineExpose({
  scrollToBottom
})
</script>

<style scoped>
.message-list-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.empty-state-with-suggestions {
  flex: 1;
  overflow-y: auto;
  padding: 20px 0;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #909399;
}

.empty-title {
  font-size: 18px;
  font-weight: 500;
  margin: 20px 0 10px;
  color: #606266;
}

.empty-desc {
  font-size: 14px;
  color: #909399;
}

.load-more {
  display: flex;
  justify-content: center;
  padding: 12px;
  border-bottom: 1px solid #ebeef5;
  background: #fff;
  flex-shrink: 0;
}

.message-scroller {
  flex: 1;
  overflow-y: auto;
  background: #fff;
  position: relative;
  height: 100%;
}

/* vue3-virtual-scroll-list 样式 */
.message-item-wrapper {
  padding: 12px 16px;
  border-bottom: 1px solid #f5f5f5;
}

.message-item-wrapper:hover {
  background: #fafafa;
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: #409eff;
  font-size: 14px;
  background: #fff;
  flex-shrink: 0;
}

.loading-indicator .is-loading {
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

.scroll-to-bottom {
  position: absolute;
  bottom: 20px;
  right: 20px;
  width: 40px;
  height: 40px;
  background: #409eff;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.4);
  transition: all 0.3s;
  z-index: 100;
}

.scroll-to-bottom:hover {
  background: #66b1ff;
  transform: scale(1.1);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 会话切换骨架屏样式 */
.message-skeleton {
  display: flex;
  gap: 12px;
  padding: 16px;
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}

.skeleton-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s infinite;
  flex-shrink: 0;
}

.skeleton-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skeleton-line {
  height: 14px;
  border-radius: 4px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s infinite;
}

.skeleton-line-short {
  width: 30%;
}

.skeleton-line-medium {
  width: 70%;
}

@keyframes skeleton-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@keyframes skeleton-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.7;
  }
}
</style>
