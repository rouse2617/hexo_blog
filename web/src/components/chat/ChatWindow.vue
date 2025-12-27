<template>
  <div class="chat-window">
    <div class="chat-header">
      <div class="session-info">
        <el-dropdown trigger="click" @command="handleSessionCommand">
          <el-button type="primary" plain>
            <el-icon><ChatDotRound /></el-icon>
            <span>{{ currentSessionTitle }}</span>
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="new">
                <el-icon><Plus /></el-icon>
                新建对话
              </el-dropdown-item>
              <el-dropdown-item divided v-for="session in sessions" :key="session.id" :command="session.id">
                <div class="session-item">
                  <span class="session-title">{{ session.title || '未命名对话' }}</span>
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
      <div class="host-select">
        <HostSelector v-model="selectedHostIds" />
      </div>
    </div>
    <MessageList
      :messages="messages"
      :is-loading="isLoading"
      :thinking-steps="thinkingSteps"
      :current-thinking-status="currentThinkingStatus"
      ref="messageListRef"
    />
    <InputBox
      :disabled="isLoading"
      @send="handleSend"
      @clear="handleClear"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import { ChatDotRound, ArrowDown, Plus, Delete } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import MessageList from './MessageList.vue'
import InputBox from './InputBox.vue'
import HostSelector from './HostSelector.vue'

const chatStore = useChatStore()
const messageListRef = ref()

const messages = computed(() => chatStore.messages)
const isLoading = computed(() => chatStore.isLoading)
const sessions = computed(() => chatStore.sessions)
const thinkingSteps = computed(() => chatStore.thinkingSteps)
const currentThinkingStatus = computed(() => chatStore.currentThinkingStatus)
const selectedHostIds = ref<string[]>([])

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

watch(selectedHostIds, (val) => {
  chatStore.setSelectedHosts(val)
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
  await chatStore.sendMessage(message)
}

const handleClear = async () => {
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
</script>

<style scoped>
.chat-window {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #f8fafc;
  border-bottom: 1px solid #ebeef5;
  gap: 20px;
}

.session-info {
  flex-shrink: 0;
}

.host-select {
  flex: 1;
  max-width: 400px;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 200px;
}

.session-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-delete {
  color: #909399;
  margin-left: 10px;
}

.session-delete:hover {
  color: #f56c6c;
}
</style>
