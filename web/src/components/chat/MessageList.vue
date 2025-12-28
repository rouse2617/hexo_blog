<template>
  <div class="message-list" ref="listRef">
    <div v-if="messages.length === 0" class="empty-state">
      <el-icon :size="64" color="#c0c4cc"><ChatDotRound /></el-icon>
      <p class="empty-title">开始新的对话</p>
      <p class="empty-desc">输入您的问题，AI 助手将为您提供帮助</p>
    </div>
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

      <!-- 消息列表 -->
      <MessageItem
        v-for="(message, index) in visibleMessages"
        :key="message.timestamp || index"
        :message="message"
        :ref="el => setMessageRef(el, index)"
      />
    </template>

    <!-- 思考过程展示 -->
    <ThinkingProcess
      :thinking-steps="thinkingSteps"
      :current-status="currentThinkingStatus"
      :is-loading="isLoading"
    />

    <!-- 简单加载指示器（当没有思考步骤时显示） -->
    <div v-if="isLoading && props.thinkingSteps.length === 0 && !currentThinkingStatus" class="loading-indicator">
      <el-icon class="is-loading" :size="20"><Loading /></el-icon>
      <span>AI 正在思考...</span>
    </div>

    <!-- 滚动到底部按钮 -->
    <transition name="fade">
      <div v-if="showScrollButton" class="scroll-to-bottom" @click="scrollToBottom">
        <el-icon><ArrowDown /></el-icon>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import type { Message } from '@/api/chat'
import type { ThinkingStep } from '@/stores/chat'
import type { ThinkingStatus } from '@/api/chat'
import { ChatDotRound, Loading, ArrowDown } from '@element-plus/icons-vue'
import MessageItem from './MessageItem.vue'
import ThinkingProcess from './ThinkingProcess.vue'

const props = withDefaults(defineProps<{
  messages: Message[]
  isLoading: boolean
  thinkingSteps?: ThinkingStep[]
  currentThinkingStatus?: ThinkingStatus | null
}>(), {
  thinkingSteps: () => [],
  currentThinkingStatus: null
})

const listRef = ref<HTMLElement | null>(null)
const messageRefs = ref<Map<number, any>>(new Map())
const showScrollButton = ref(false)
const loadingMore = ref(false)
const displayCount = ref(50) // 初始显示的消息数量
const BATCH_SIZE = 30 // 每次加载更多的数量

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

// 加载更多历史消息
const loadMoreMessages = async () => {
  loadingMore.value = true
  // 记录当前滚动位置
  const scrollHeight = listRef.value?.scrollHeight || 0

  await nextTick()
  displayCount.value += BATCH_SIZE

  // 保持滚动位置
  await nextTick()
  if (listRef.value) {
    const newScrollHeight = listRef.value.scrollHeight
    listRef.value.scrollTop = newScrollHeight - scrollHeight
  }
  loadingMore.value = false
}

// 自动滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (listRef.value) {
      listRef.value.scrollTo({
        top: listRef.value.scrollHeight,
        behavior: 'smooth'
      })
    }
  })
}

// 检查是否在底部附近
const isNearBottom = () => {
  if (!listRef.value) return true
  const { scrollTop, scrollHeight, clientHeight } = listRef.value
  return scrollHeight - scrollTop - clientHeight < 100
}

// 处理滚动事件
const handleScroll = () => {
  if (!listRef.value) return
  const { scrollTop, scrollHeight, clientHeight } = listRef.value
  // 距离底部超过 300px 时显示滚动按钮
  showScrollButton.value = scrollHeight - scrollTop - clientHeight > 300
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
  () => props.messages[props.messages.length - 1]?.content,
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
  listRef.value?.addEventListener('scroll', handleScroll, { passive: true })
})

onUnmounted(() => {
  listRef.value?.removeEventListener('scroll', handleScroll)
})

defineExpose({
  scrollToBottom
})
</script>

<style scoped>
.message-list {
  flex: 1;
  overflow-y: auto;
  background: #fff;
  position: relative;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
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
}

.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: #409eff;
  font-size: 14px;
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
</style>
