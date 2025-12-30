<template>
  <div 
    class="chat-window-container"
    :class="[
      `layout-${layout}`,
      { 'left-panel-hidden': !leftPanelVisible, 'right-panel-hidden': !rightPanelVisible }
    ]"
  >
    <!-- 左侧面板 - 历史记录 (仅桌面端显示) -->
    <aside 
      v-if="layout === 'three-column' && leftPanelVisible" 
      class="history-panel"
    >
      <div class="panel-header">
        <h3 class="panel-title">
          <el-icon><ChatDotRound /></el-icon>
          <span>对话历史</span>
        </h3>
        <el-button 
          type="primary" 
          size="small" 
          @click="handleNewSession"
          :icon="Plus"
        >
          新建
        </el-button>
      </div>
      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="session-item"
          :class="{ active: session.id === currentSessionId }"
          @click="handleSessionSelect(session.id)"
        >
          <div class="session-info">
            <el-icon class="session-icon"><ChatDotRound /></el-icon>
            <span class="session-title">{{ session.title || '未命名对话' }}</span>
          </div>
          <el-icon
            class="session-delete"
            @click.stop="handleDeleteSession(session.id)"
          >
            <Delete />
          </el-icon>
        </div>
        <div v-if="sessions.length === 0" class="empty-sessions">
          <el-icon :size="32" color="#c0c4cc"><ChatDotRound /></el-icon>
          <p>暂无对话记录</p>
        </div>
      </div>
    </aside>

    <!-- 中间面板 - 聊天主区域 -->
    <main class="chat-panel">
      <!-- 聊天头部 -->
      <div class="chat-header">
        <!-- 移动端/平板端菜单按钮 -->
        <el-button
          v-if="layout !== 'three-column'"
          class="menu-button"
          :icon="Menu"
          @click="handleToggleHistoryOverlay"
          circle
        />

        <div class="session-section">
          <el-dropdown trigger="click" @command="handleSessionCommand">
            <el-button type="primary" class="session-selector">
              <el-icon><ChatDotRound /></el-icon>
              <span class="session-title-text">{{ currentSessionTitle }}</span>
              <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="new">
                  <el-icon><Plus /></el-icon>
                  <span>新建对话</span>
                </el-dropdown-item>
                <el-dropdown-item divided v-for="session in sessions" :key="session.id" :command="session.id">
                  <div class="dropdown-session-item">
                    <div class="dropdown-session-info">
                      <el-icon><ChatDotRound /></el-icon>
                      <span class="dropdown-session-title">{{ session.title || '未命名对话' }}</span>
                    </div>
                    <el-icon
                      class="dropdown-session-delete"
                      @click.stop="handleDeleteSession(session.id)"
                    >
                      <Delete />
                    </el-icon>
                  </div>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <!-- 主机选择器 -->
        <div class="host-section">
          <div class="host-selector-wrapper">
            <el-icon class="selector-icon"><Monitor /></el-icon>
            <HostSelector v-model="selectedHostIds" />
            <el-badge v-if="selectedHostIds.length > 0" :value="selectedHostIds.length" />
          </div>
        </div>

        <!-- 右侧面板切换按钮 -->
        <el-button
          v-if="layout !== 'single-column'"
          class="panel-toggle-button"
          :icon="rightPanelVisible ? Expand : Fold"
          @click="handleToggleRightPanel"
          circle
          title="切换监控面板"
        />
      </div>

      <!-- 消息列表 -->
      <div class="message-area">
        <MessageList
          :messages="messages"
          :is-loading="isLoading"
          :thinking-steps="thinkingSteps"
          :current-thinking-status="currentThinkingStatus"
          :session-loading="sessionLoading"
          ref="messageListRef"
          @send-message="handleSendFromList"
          @regenerate="handleRegenerate"
          @tool-rerun="handleToolRerun"
        />
      </div>

      <!-- 输入区域 -->
      <div class="input-area">
        <InputBox
          ref="inputBoxRef"
          :disabled="isLoading"
          @send="handleSend"
          @clear="handleClearMessages"
        />
      </div>
    </main>

    <!-- 右侧面板 - 上下文/监控 (桌面端和平板端显示) -->
    <aside 
      v-if="layout !== 'single-column' && rightPanelVisible" 
      class="context-panel"
    >
      <div class="panel-header">
        <h3 class="panel-title">
          <el-icon><DataAnalysis /></el-icon>
          <span>实时监控</span>
        </h3>
        <el-switch
          v-model="silentMode"
          size="small"
          active-text="静默"
          inactive-text=""
        />
      </div>
      <div class="context-content">
        <!-- 监控内容占位 - 后续任务实现 -->
        <div class="monitor-placeholder">
          <el-icon :size="48" color="#c0c4cc"><DataAnalysis /></el-icon>
          <p>选择主机查看监控数据</p>
          <p class="hint">监控面板将在后续任务中实现</p>
        </div>
      </div>
    </aside>

    <!-- 移动端历史面板遮罩 -->
    <transition name="overlay-fade">
      <div 
        v-if="historyPanelOverlayVisible" 
        class="history-overlay-backdrop"
        @click="handleCloseHistoryOverlay"
      />
    </transition>
    
    <!-- 移动端历史面板侧滑 -->
    <transition name="slide-left">
      <aside 
        v-if="historyPanelOverlayVisible" 
        class="history-panel-overlay"
      >
        <div class="panel-header">
          <h3 class="panel-title">
            <el-icon><ChatDotRound /></el-icon>
            <span>对话历史</span>
          </h3>
          <el-button 
            :icon="Close" 
            circle 
            size="small"
            @click="handleCloseHistoryOverlay"
          />
        </div>
        <div class="panel-actions">
          <el-button 
            type="primary" 
            @click="handleNewSessionFromOverlay"
            :icon="Plus"
            style="width: 100%"
          >
            新建对话
          </el-button>
        </div>
        <div class="session-list">
          <div
            v-for="session in sessions"
            :key="session.id"
            class="session-item"
            :class="{ active: session.id === currentSessionId }"
            @click="handleSessionSelectFromOverlay(session.id)"
          >
            <div class="session-info">
              <el-icon class="session-icon"><ChatDotRound /></el-icon>
              <span class="session-title">{{ session.title || '未命名对话' }}</span>
            </div>
            <el-icon
              class="session-delete"
              @click.stop="handleDeleteSession(session.id)"
            >
              <Delete />
            </el-icon>
          </div>
          <div v-if="sessions.length === 0" class="empty-sessions">
            <el-icon :size="32" color="#c0c4cc"><ChatDotRound /></el-icon>
            <p>暂无对话记录</p>
          </div>
        </div>
      </aside>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'
import type { Message } from '@/api/chat'
import {
  ChatDotRound, ArrowDown, Plus, Delete,
  Monitor, Menu, Close, Expand, Fold, DataAnalysis
} from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import MessageList from './MessageList.vue'
import InputBox from './InputBox.vue'
import HostSelector from './HostSelector.vue'

const chatStore = useChatStore()
const uiStore = useUIStore()
const messageListRef = ref()
const inputBoxRef = ref()

// Chat store state
const messages = computed(() => chatStore.messages)
const isLoading = computed(() => chatStore.isLoading)
const sessions = computed(() => chatStore.sessions)
const thinkingSteps = computed(() => chatStore.thinkingSteps)
const currentThinkingStatus = computed(() => chatStore.currentThinkingStatus)
const sessionLoading = computed(() => chatStore.sessionLoading)
const currentSessionId = computed(() => chatStore.currentSessionId)

// UI store state
const layout = computed(() => uiStore.layout)
const leftPanelVisible = computed(() => uiStore.leftPanelVisible)
const rightPanelVisible = computed(() => uiStore.rightPanelVisible)
const historyPanelOverlayVisible = computed(() => uiStore.historyPanelOverlayVisible)

// Local state
const silentMode = ref(false)

const selectedHostIds = computed({
  get: () => chatStore.selectedHostIds,
  set: (val) => chatStore.setSelectedHosts(val)
})

const currentSessionTitle = computed(() => {
  const session = sessions.value.find(s => s.id === chatStore.currentSessionId)
  return session?.title || '新对话'
})

// Initialize UI store and load sessions
onMounted(async () => {
  uiStore.init()
  await chatStore.loadSessions()
  if (!chatStore.currentSessionId) {
    await chatStore.newSession()
  }
})

// Session handlers
const handleSessionCommand = async (command: string) => {
  if (command === 'new') {
    await chatStore.newSession()
  } else {
    await chatStore.switchSession(command)
  }
}

const handleSessionSelect = async (sessionId: string) => {
  await chatStore.switchSession(sessionId)
}

const handleNewSession = async () => {
  await chatStore.newSession()
}

const handleDeleteSession = async (sessionId: string) => {
  try {
    await ElMessageBox.confirm('确定要删除这个对话吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await chatStore.removeSession(sessionId)
  } catch {
    // 取消删除
  }
}

// Message handlers
const handleSend = async (message: string) => {
  try {
    await chatStore.sendMessage(message)
    inputBoxRef.value?.clearInput()
  } catch (error) {
    console.error('发送消息失败:', error)
  }
}

const handleSendFromList = async (message: string) => {
  try {
    await chatStore.sendMessage(message)
  } catch (error) {
    console.error('发送消息失败:', error)
  }
}

const handleRegenerate = async (message: Message) => {
  try {
    const messageIndex = messages.value.findIndex(m => m === message)
    if (messageIndex > 0) {
      const userMessage = messages.value[messageIndex - 1]
      if (userMessage.role === 'user') {
        messages.value.splice(messageIndex, 1)
        await chatStore.sendMessage(userMessage.content)
      }
    }
  } catch (error) {
    console.error('重新生成失败:', error)
    ElMessage.error('重新生成失败，请稍后重试')
  }
}

const handleToolRerun = async (toolCall: any) => {
  try {
    ElMessage.info(`正在重新执行工具: ${toolCall.name}`)
  } catch (error) {
    console.error('工具重新执行失败:', error)
    ElMessage.error('工具重新执行失败，请稍后重试')
  }
}

const handleClearMessages = async () => {
  try {
    await ElMessageBox.confirm('确定要清空当前对话吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    chatStore.clearMessages()
  } catch {
    // 取消清空
  }
}

// Panel toggle handlers
const handleToggleRightPanel = () => {
  uiStore.togglePanel('right')
}

const handleToggleHistoryOverlay = () => {
  uiStore.toggleHistoryOverlay()
}

const handleCloseHistoryOverlay = () => {
  uiStore.closeHistoryOverlay()
}

const handleNewSessionFromOverlay = async () => {
  await chatStore.newSession()
  uiStore.closeHistoryOverlay()
}

const handleSessionSelectFromOverlay = async (sessionId: string) => {
  await chatStore.switchSession(sessionId)
  uiStore.closeHistoryOverlay()
}
</script>

<style scoped>
/* ============================================
 * 三栏布局容器
 * ============================================ */
.chat-window-container {
  display: grid;
  height: 100%;
  width: 100%;
  background: var(--color-gray-50);
  overflow: hidden;
}

/* 三栏布局 (>1200px) */
.layout-three-column {
  grid-template-columns: 280px 1fr minmax(300px, 360px);
  grid-template-areas: "history chat context";
}

.layout-three-column.left-panel-hidden {
  grid-template-columns: 0 1fr minmax(300px, 360px);
}

.layout-three-column.right-panel-hidden {
  grid-template-columns: 280px 1fr 0;
}

.layout-three-column.left-panel-hidden.right-panel-hidden {
  grid-template-columns: 0 1fr 0;
}

/* 两栏布局 (768px - 1200px) */
.layout-two-column {
  grid-template-columns: 1fr minmax(280px, 320px);
  grid-template-areas: "chat context";
}

.layout-two-column.right-panel-hidden {
  grid-template-columns: 1fr 0;
}

/* 单栏布局 (<768px) */
.layout-single-column {
  grid-template-columns: 1fr;
  grid-template-areas: "chat";
}

/* ============================================
 * 左侧历史面板
 * ============================================ */
.history-panel {
  grid-area: history;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, var(--sidebar-bg-start) 0%, var(--sidebar-bg-end) 100%);
  border-right: 1px solid var(--sidebar-border);
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4);
  border-bottom: 1px solid var(--sidebar-border);
  flex-shrink: 0;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--sidebar-text-hover);
  margin: 0;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2);
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-3) var(--spacing-3);
  margin-bottom: var(--spacing-1);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--sidebar-text);
}

.session-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--sidebar-text-hover);
}

.session-item.active {
  background: var(--sidebar-active-bg);
  color: var(--sidebar-active-text);
}

.session-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  flex: 1;
  min-width: 0;
}

.session-icon {
  flex-shrink: 0;
}

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--text-sm);
}

.session-delete {
  opacity: 0;
  color: var(--sidebar-text);
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.session-item:hover .session-delete {
  opacity: 1;
}

.session-delete:hover {
  color: var(--color-danger);
}

.empty-sessions {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-8);
  color: var(--sidebar-text);
  text-align: center;
}

.empty-sessions p {
  margin-top: var(--spacing-2);
  font-size: var(--text-sm);
}

/* ============================================
 * 中间聊天面板
 * ============================================ */
.chat-panel {
  grid-area: chat;
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
  min-width: 0;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3) var(--spacing-4);
  background: linear-gradient(to bottom, #f8fafc, #fff);
  border-bottom: 1px solid var(--color-gray-200);
  flex-shrink: 0;
}

.menu-button {
  flex-shrink: 0;
}

.session-section {
  flex-shrink: 0;
}

.session-selector {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  border-radius: var(--radius-lg);
}

.session-title-text {
  font-weight: var(--font-medium);
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 200px;
  max-width: 280px;
  gap: var(--spacing-3);
}

.dropdown-session-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  flex: 1;
  min-width: 0;
}

.dropdown-session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-session-delete {
  color: var(--color-gray-400);
  transition: color var(--transition-fast);
}

.dropdown-session-delete:hover {
  color: var(--color-danger);
}

.host-section {
  flex: 1;
  max-width: 360px;
  min-width: 0;
}

.host-selector-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  background: #fff;
  border: 1px solid var(--color-gray-300);
  border-radius: var(--radius-lg);
  transition: all var(--transition-fast);
}

.host-selector-wrapper:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1);
}

.selector-icon {
  color: var(--color-gray-500);
  flex-shrink: 0;
}

.panel-toggle-button {
  flex-shrink: 0;
  margin-left: auto;
}

.message-area {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 消息区域内容居中，最大宽度800px */
.message-area :deep(.message-list-container) {
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

/* 超宽屏幕 (>1920px) 时最大宽度1000px */
@media (min-width: 1920px) {
  .message-area :deep(.message-list-container) {
    max-width: 1000px;
  }
}

.input-area {
  border-top: 1px solid var(--color-gray-200);
  background: #fff;
  flex-shrink: 0;
}

/* ============================================
 * 右侧上下文面板
 * ============================================ */
.context-panel {
  grid-area: context;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-left: 1px solid var(--color-gray-200);
  overflow: hidden;
}

.context-panel .panel-header {
  background: linear-gradient(to bottom, #f8fafc, #fff);
  border-bottom: 1px solid var(--color-gray-200);
}

.context-panel .panel-title {
  color: var(--color-gray-700);
}

.context-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-4);
}

.monitor-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-gray-400);
  text-align: center;
}

.monitor-placeholder p {
  margin-top: var(--spacing-2);
  font-size: var(--text-sm);
}

.monitor-placeholder .hint {
  font-size: var(--text-xs);
  color: var(--color-gray-300);
}

/* ============================================
 * 移动端历史面板遮罩
 * ============================================ */
.history-overlay-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: var(--z-modal-backdrop);
}

.history-panel-overlay {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  max-width: 85vw;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, var(--sidebar-bg-start) 0%, var(--sidebar-bg-end) 100%);
  z-index: var(--z-modal);
  box-shadow: var(--shadow-xl);
}

.history-panel-overlay .panel-header {
  border-bottom: 1px solid var(--sidebar-border);
}

.history-panel-overlay .panel-actions {
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--sidebar-border);
}

/* ============================================
 * 动画
 * ============================================ */
.overlay-fade-enter-active,
.overlay-fade-leave-active {
  transition: opacity var(--transition-base);
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
  opacity: 0;
}

.slide-left-enter-active,
.slide-left-leave-active {
  transition: transform var(--transition-base);
}

.slide-left-enter-from,
.slide-left-leave-to {
  transform: translateX(-100%);
}

/* ============================================
 * 响应式调整
 * ============================================ */
@media (max-width: 1200px) {
  .session-title-text {
    max-width: 120px;
  }
  
  .host-section {
    max-width: 280px;
  }
}

@media (max-width: 768px) {
  .chat-header {
    padding: var(--spacing-2) var(--spacing-3);
  }
  
  .session-title-text {
    max-width: 100px;
  }
  
  .host-section {
    flex: 1;
    max-width: none;
  }
  
  .host-selector-wrapper {
    padding: var(--spacing-2);
  }
}
</style>
