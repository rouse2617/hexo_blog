/**
 * 持久化状态使用示例
 *
 * 此文件展示如何在组件中使用持久化的 Pinia 状态
 */

import { watch, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useConsoleStore } from '@/stores/console'

/**
 * 示例 1: 使用应用配置（侧边栏、主题等）
 */
export function useAppConfig() {
  const appStore = useAppStore()
  const { sidebarCollapsed, theme, language } = storeToRefs(appStore)

  // 这些状态会自动持久化到 localStorage
  // 刷新页面后会自动恢复

  const toggleSidebar = () => {
    appStore.toggleSidebar() // 状态变化会自动保存
  }

  const changeTheme = (newTheme: 'light' | 'dark' | 'auto') => {
    appStore.setTheme(newTheme) // 主题设置会自动保存
  }

  const changeLanguage = (lang: string) => {
    appStore.setLanguage(lang) // 语言设置会自动保存
  }

  return {
    sidebarCollapsed,
    theme,
    language,
    toggleSidebar,
    changeTheme,
    changeLanguage
  }
}

/**
 * 示例 2: 使用聊天会话状态
 */
export function useChatPersistence() {
  const chatStore = useChatStore()
  const { sessions, currentSessionId, selectedHostIds } = storeToRefs(chatStore)

  // 会话列表和当前会话 ID 会自动持久化
  // 即使刷新页面，也会恢复到之前的会话

  const switchToSession = (sessionId: string) => {
    chatStore.switchSession(sessionId)
  }

  const selectHosts = (hostIds: string[]) => {
    chatStore.setSelectedHosts(hostIds) // 选中的主机会自动保存
  }

  return {
    sessions,
    currentSessionId,
    selectedHostIds,
    switchToSession,
    selectHosts
  }
}

/**
 * 示例 3: 使用控制台选中状态
 */
export function useConsolePersistence() {
  const consoleStore = useConsoleStore()
  const { selectedHosts } = storeToRefs(consoleStore)

  // 选中的主机列表会自动持久化
  // 切换到其他页面后再回来，选择状态会保持

  const selectHosts = (hosts: string[]) => {
    consoleStore.selectHosts(hosts) // 选择会自动保存
  }

  const clearSelection = () => {
    consoleStore.clearSelection() // 清除选择会自动保存
  }

  return {
    selectedHosts,
    selectHosts,
    clearSelection
  }
}

/**
 * 示例 4: 监听持久化状态变化
 */
export function usePersistedWatcher() {
  const appStore = useAppStore()
  const { theme, sidebarCollapsed } = storeToRefs(appStore)

  // 监听主题变化
  watch(theme, (newTheme) => {
    console.log('主题已更改:', newTheme)
    // 新的主题值已自动保存到 localStorage
  })

  // 监听侧边栏状态
  watch(sidebarCollapsed, (collapsed) => {
    console.log('侧边栏状态:', collapsed)
    // 新的状态已自动保存
  })

  return {
    theme,
    sidebarCollapsed
  }
}

/**
 * 示例 5: 组件中的实际使用
 */
export function setupPersistedState() {
  const appStore = useAppStore()
  const chatStore = useChatStore()

  // 初始化时，这些值会从 localStorage 恢复
  const { sidebarCollapsed, theme } = storeToRefs(appStore)
  const { currentSessionId } = storeToRefs(chatStore)

  // 组件挂载时的操作
  onMounted(() => {
    console.log('当前主题:', theme.value) // 从持久化恢复的值
    console.log('侧边栏状态:', sidebarCollapsed.value) // 从持久化恢复的值
    console.log('当前会话:', currentSessionId.value) // 从持久化恢复的值
  })

  return {
    sidebarCollapsed,
    theme,
    currentSessionId
  }
}

/**
 * 示例 6: 清除持久化数据
 */
export function usePersistedDataManagement() {
  const appStore = useAppStore()
  const chatStore = useChatStore()

  // 重置应用配置到默认值
  const resetAppConfig = () => {
    appStore.$reset() // 这会清除 app store 的持久化数据
  }

  // 清除聊天会话缓存
  const clearChatCache = () => {
    chatStore.clearMessageCache() // 清除所有消息缓存
  }

  // 清除特定会话的缓存
  const clearSessionCache = (sessionId: string) => {
    chatStore.clearMessageCache(sessionId)
  }

  return {
    resetAppConfig,
    clearChatCache,
    clearSessionCache
  }
}

/**
 * 示例 7: 在 Vue 组件中的完整使用
 */
/*
<template>
  <div>
    <p>当前主题: {{ theme }}</p>
    <p>侧边栏: {{ sidebarCollapsed ? '收起' : '展开' }}</p>

    <button @click="toggleSidebar">
      {{ sidebarCollapsed ? '展开' : '收起' }}侧边栏
    </button>

    <button @click="changeTheme('dark')">
      切换到暗色主题
    </button>
  </div>
</template>

<script setup lang="ts">
import { useAppConfig } from '@/composables/usePersistedState'

const {
  sidebarCollapsed,
  theme,
  toggleSidebar,
  changeTheme
} = useAppConfig()

// 所有状态变化都会自动保存到 localStorage
// 刷新页面后会自动恢复
</script>
*/
