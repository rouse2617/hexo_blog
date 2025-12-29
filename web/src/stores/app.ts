import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'light' | 'dark' | 'auto'

export const useAppStore = defineStore('app', () => {
  // 侧边栏折叠状态
  const sidebarCollapsed = ref(false)

  // 主题模式
  const theme = ref<Theme>('light')

  // 语言
  const language = ref<string>('zh-CN')

  // 主机列表每页数量
  const hostPageSize = ref<number>(20)

  // 会话列表每页数量
  const sessionPageSize = ref<number>(20)

  // 切换侧边栏
  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  // 设置侧边栏状态
  function setSidebarCollapsed(collapsed: boolean) {
    sidebarCollapsed.value = collapsed
  }

  // 设置主题
  function setTheme(newTheme: Theme) {
    theme.value = newTheme
    applyTheme(newTheme)
  }

  // 应用主题
  function applyTheme(currentTheme: Theme) {
    const actualTheme = currentTheme === 'auto'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : currentTheme

    document.documentElement.setAttribute('data-theme', actualTheme)

    // 更新 CSS 变量
    if (actualTheme === 'dark') {
      document.documentElement.style.setProperty('--color-bg-primary', '#1a1a1a')
      document.documentElement.style.setProperty('--color-bg-secondary', '#2d2d2d')
      document.documentElement.style.setProperty('--color-text-primary', '#e5e5e5')
      document.documentElement.style.setProperty('--color-text-secondary', '#a3a3a3')
      document.documentElement.style.setProperty('--sidebar-bg-start', '#1a1a1a')
      document.documentElement.style.setProperty('--sidebar-bg-end', '#262626')
      document.documentElement.style.setProperty('--color-gray-50', '#171717')
      document.documentElement.style.setProperty('--color-gray-100', '#262626')
      document.documentElement.style.setProperty('--color-gray-200', '#404040')
    } else {
      document.documentElement.style.setProperty('--color-bg-primary', '#ffffff')
      document.documentElement.style.setProperty('--color-bg-secondary', '#f5f5f5')
      document.documentElement.style.setProperty('--color-text-primary', '#171717')
      document.documentElement.style.setProperty('--color-text-secondary', '#737373')
      document.documentElement.style.setProperty('--sidebar-bg-start', '#ffffff')
      document.documentElement.style.setProperty('--sidebar-bg-end', '#fafafa')
      document.documentElement.style.setProperty('--color-gray-50', '#fafafa')
      document.documentElement.style.setProperty('--color-gray-100', '#f5f5f5')
      document.documentElement.style.setProperty('--color-gray-200', '#e5e5e5')
    }
  }

  // 设置语言
  function setLanguage(lang: string) {
    language.value = lang
  }

  // 初始化主题
  function initTheme() {
    applyTheme(theme.value)

    // 监听系统主题变化（当设置为 auto 时）
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'auto') {
        applyTheme('auto')
      }
    })
  }

  return {
    sidebarCollapsed,
    theme,
    language,
    hostPageSize,
    sessionPageSize,
    toggleSidebar,
    setSidebarCollapsed,
    setTheme,
    setLanguage,
    initTheme
  }
}, {
  persist: {
    key: 'ai-pro-app',
    paths: ['sidebarCollapsed', 'theme', 'language', 'hostPageSize', 'sessionPageSize']
  }
})
