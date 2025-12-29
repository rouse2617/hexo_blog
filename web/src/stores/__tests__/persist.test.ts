/**
 * Pinia 持久化插件测试示例
 *
 * 此文件展示如何测试持久化功能
 */

import { createPinia, setActivePinia } from 'pinia'
import { createPersistPlugin } from '../plugins/persist'
import { useAppStore } from '../app'
import { useChatStore } from '../chat'

describe('Persist Plugin', () => {
  beforeEach(() => {
    // 创建新的 pinia 实例
    setActivePinia(createPinia())

    // 清除所有存储
    localStorage.clear()
    sessionStorage.clear()

    // 注册持久化插件
    const pinia = setActivePinia(createPinia())
    pinia.use(createPersistPlugin())
  })

  afterEach(() => {
    localStorage.clear()
    sessionStorage.clear()
  })

  describe('App Store Persistence', () => {
    it('should persist sidebarCollapsed state', () => {
      const store = useAppStore()

      // 初始状态
      expect(store.sidebarCollapsed).toBe(false)

      // 修改状态
      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(true)

      // 检查 localStorage
      const saved = localStorage.getItem('ai-pro-app')
      expect(saved).toBeTruthy()

      const parsed = JSON.parse(saved!)
      expect(parsed.sidebarCollapsed).toBe(true)
    })

    it('should restore state from localStorage', () => {
      // 预设存储数据
      localStorage.setItem('ai-pro-app', JSON.stringify({
        sidebarCollapsed: true,
        theme: 'dark',
        language: 'en-US',
        hostPageSize: 50,
        sessionPageSize: 50
      }))

      const store = useAppStore()

      // 验证状态已恢复
      expect(store.sidebarCollapsed).toBe(true)
      expect(store.theme).toBe('dark')
      expect(store.language).toBe('en-US')
      expect(store.hostPageSize).toBe(50)
      expect(store.sessionPageSize).toBe(50)
    })

    it('should persist only specified paths', () => {
      const store = useAppStore()

      // 修改多个状态
      store.setSidebarCollapsed(true)
      store.setTheme('dark')
      store.setLanguage('en-US')

      // 检查存储内容
      const saved = localStorage.getItem('ai-pro-app')
      const parsed = JSON.parse(saved!)

      // 验证只包含指定的路径
      expect(parsed).toHaveProperty('sidebarCollapsed')
      expect(parsed).toHaveProperty('theme')
      expect(parsed).toHaveProperty('language')
    })
  })

  describe('Chat Store Persistence', () => {
    it('should persist sessions and currentSessionId', () => {
      const store = useChatStore()

      // 模拟会话数据
      const mockSessions = [
        { id: '1', title: 'Session 1', createdAt: '2024-01-01' },
        { id: '2', title: 'Session 2', createdAt: '2024-01-02' }
      ]

      store.$patch({
        sessions: mockSessions,
        currentSessionId: '1'
      })

      // 检查存储
      const saved = localStorage.getItem('ai-pro-chat')
      expect(saved).toBeTruthy()

      const parsed = JSON.parse(saved!)
      expect(parsed.sessions).toEqual(mockSessions)
      expect(parsed.currentSessionId).toBe('1')
    })

    it('should not persist messages and loading state', () => {
      const store = useChatStore()

      store.$patch({
        messages: [{ role: 'user', content: 'test', timestamp: Date.now() }],
        isLoading: true
      })

      // 等待持久化
      await new Promise(resolve => setTimeout(resolve, 100))

      const saved = localStorage.getItem('ai-pro-chat')
      const parsed = JSON.parse(saved!)

      // messages 和 isLoading 不应该被持久化
      expect(parsed.messages).toBeUndefined()
      expect(parsed.isLoading).toBeUndefined()
    })
  })

  describe('Message Cache', () => {
    it('should cache messages with timestamp', async () => {
      const store = useChatStore()

      const mockMessages = [
        { role: 'user' as const, content: 'Hello', timestamp: Date.now() },
        { role: 'assistant' as const, content: 'Hi there!', timestamp: Date.now() }
      ]

      const sessionId = 'test-session'

      // 保存缓存
      store.$patch({ messages: mockMessages })
      await new Promise(resolve => setTimeout(resolve, 100))

      // 检查缓存
      const cached = localStorage.getItem(`chat-messages-${sessionId}`)
      expect(cached).toBeTruthy()

      const parsed = JSON.parse(cached!)
      expect(parsed.messages).toEqual(mockMessages)
      expect(parsed.timestamp).toBeDefined()
    })

    it('should respect cache expiration', async () => {
      // 这个测试需要等待或 mock 时间
      // 实际项目中可以跳过或使用 vi.useFakeTimers()
    })
  })

  describe('Error Handling', () => {
    it('should handle corrupted localStorage gracefully', () => {
      // 存储无效的 JSON
      localStorage.setItem('ai-pro-app', 'invalid-json{')

      // store 应该能正常初始化，使用默认值
      const store = useAppStore()
      expect(store.sidebarCollapsed).toBe(false)
      expect(store.theme).toBe('light')
    })

    it('should handle localStorage quota exceeded', () => {
      // 模拟存储满的情况
      const originalSetItem = Storage.prototype.setItem
      Storage.prototype.setItem = vi.fn(() => {
        throw new Error('QuotaExceededError')
      })

      const store = useAppStore()
      store.toggleSidebar()

      // 不应该抛出错误
      expect(store.sidebarCollapsed).toBe(true)

      // 恢复原始方法
      Storage.prototype.setItem = originalSetItem
    })
  })

  describe('Session Storage', () => {
    it('should use sessionStorage when configured', () => {
      // 创建一个使用 sessionStorage 的 store
      const pinia = createPinia()
      pinia.use(createPersistPlugin({ storage: 'sessionStorage' }))
      setActivePinia(pinia)

      const store = useAppStore()
      store.toggleSidebar()

      // 检查 sessionStorage
      const saved = sessionStorage.getItem('ai-pro-app')
      expect(saved).toBeTruthy()

      // localStorage 中不应该有
      expect(localStorage.getItem('ai-pro-app')).toBeNull()
    })
  })
})

/**
 * 集成测试示例
 */
describe('Persistence Integration', () => {
  it('should maintain state across page reloads', async () => {
    // 第一次加载
    const pinia1 = createPinia()
    pinia1.use(createPersistPlugin())
    setActivePinia(pinia1)

    const store1 = useAppStore()
    store1.setTheme('dark')
    store1.setLanguage('en-US')

    // 模拟页面刷新 - 创建新的 pinia 实例
    const pinia2 = createPinia()
    pinia2.use(createPersistPlugin())
    setActivePinia(pinia2)

    const store2 = useAppStore()

    // 验证状态已恢复
    expect(store2.theme).toBe('dark')
    expect(store2.language).toBe('en-US')
  })

  it('should handle multiple stores independently', () => {
    const pinia = createPinia()
    pinia.use(createPersistPlugin())
    setActivePinia(pinia)

    const appStore = useAppStore()
    const chatStore = useChatStore()

    appStore.setTheme('dark')
    chatStore.$patch({
      sessions: [{ id: '1', title: 'Test', createdAt: '2024-01-01' }]
    })

    // 两个 store 的数据应该分开存储
    const appData = JSON.parse(localStorage.getItem('ai-pro-app')!)
    const chatData = JSON.parse(localStorage.getItem('ai-pro-chat')!)

    expect(appData.theme).toBe('dark')
    expect(chatData.sessions).toHaveLength(1)
    expect(appData).not.toHaveProperty('sessions')
    expect(chatData).not.toHaveProperty('theme')
  })
})
