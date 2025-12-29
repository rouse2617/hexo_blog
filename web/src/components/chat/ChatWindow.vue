<template>
  <div class="chat-window">
    <!-- 聊天头部 -->
    <div class="chat-header">
      <div class="session-section">
        <el-dropdown trigger="click" @command="handleSessionCommand">
          <el-button type="primary" class="session-selector">
            <el-icon><ChatDotRound /></el-icon>
            <span class="session-title">{{ currentSessionTitle }}</span>
            <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="new">
                <el-icon><Plus /></el-icon>
                <span>新建对话</span>
              </el-dropdown-item>
              <el-dropdown-item divided v-for="session in sessions" :key="session.id" :command="session.id">
                <div class="session-item">
                  <div class="session-info">
                    <el-icon><ChatDotRound /></el-icon>
                    <span class="session-title-text">{{ session.title || '未命名对话' }}</span>
                  </div>
                  <el-icon
                    class="session-delete"
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
    </div>

    <!-- 消息列表 -->
    <MessageList
      :messages="messages"
      :is-loading="isLoading"
      :thinking-steps="thinkingSteps"
      :current-thinking-status="currentThinkingStatus"
      ref="messageListRef"
    />

    <!-- 输入区域 -->
    <div class="input-area">
      <InputBox
        ref="inputBoxRef"
        :disabled="isLoading"
        @send="handleSend"
      />

      <!-- 提示信息 -->
      <div class="input-tips">
        <el-space :size="8">
          <el-tag size="small" type="info">
            <el-icon><Position /></el-icon>
            Enter 发送
          </el-tag>
          <el-tag size="small" type="info">
            <el-icon><TopRight /></el-icon>
            Shift+Enter 换行
          </el-tag>
        </el-space>
        <span class="tips-text">您可以询问服务器状态、执行命令、查看日志等运维相关问题</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useChatStore } from '@/stores/chat'
import {
  ChatDotRound, ArrowDown, Plus, Delete,
  Monitor, Position, TopRight
} from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import MessageList from './MessageList.vue'
import InputBox from './InputBox.vue'
import HostSelector from './HostSelector.vue'

const chatStore = useChatStore()
const messageListRef = ref()
const inputBoxRef = ref()

const messages = computed(() => chatStore.messages)
const isLoading = computed(() => chatStore.isLoading)
const sessions = computed(() => chatStore.sessions)
const thinkingSteps = computed(() => chatStore.thinkingSteps)
const currentThinkingStatus = computed(() => chatStore.currentThinkingStatus)

const selectedHostIds = computed({
  get: () => chatStore.selectedHostIds,
  set: (val) => chatStore.setSelectedHosts(val)
})

const currentSessionTitle = computed(() => {
  const session = sessions.value.find(s => s.id === chatStore.currentSessionId)
  return session?.title || '新对话'
})

onMounted(async () => {
  await chatStore.loadSessions()
  if (!chatStore.currentSessionId) {
    await chatStore.newSession()
  }
})

const handleSessionCommand = async (command: string) => {
  if (command === 'new') {
    await chatStore.newSession()
  } else {
    await chatStore.switchSession(command)
  }
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

const handleSend = async (message: string) => {
  try {
    await chatStore.sendMessage(message)
    inputBoxRef.value?.clearInput()
  } catch (error) {
    console.error('发送消息失败:', error)
  }
}
</script>

<style scoped>
.chat-window {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  border-radius: var(--radius-2xl);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

/* 聊天头部 */
.chat-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-4);
  padding: var(--spacing-4) var(--spacing-5);
  background: linear-gradient(to bottom, #f8fafc, #fff);
  border-bottom: 1px solid var(--color-gray-200);
}

.session-selector {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-3) var(--spacing-4);
  border-radius: var(--radius-lg);
}

.session-title {
  font-weight: var(--font-medium);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 240px;
  max-width: 320px;
  gap: var(--spacing-4);
}

.session-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  flex: 1;
  min-width: 0;
}

.session-title-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-delete {
  color: var(--color-gray-400);
  transition: color var(--transition-fast);
}

.session-delete:hover {
  color: #ef4444;
}

/* 主机选择器 */
.host-section {
  flex: 1;
  max-width: 420px;
}

.host-selector-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-3) var(--spacing-4);
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
}

/* 输入区域 */
.input-area {
  border-top: 1px solid var(--color-gray-200);
  background: #fff;
  padding: var(--spacing-4) var(--spacing-5);
}

.input-tips {
  display: flex;
  align-items: center;
  gap: var(--spacing-4);
  margin-top: var(--spacing-3);
}

.tips-text {
  flex: 1;
  font-size: var(--text-xs);
  color: var(--color-gray-500);
}
</style>
