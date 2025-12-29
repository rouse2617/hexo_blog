# Pinia 状态持久化插件

## 概述

本项目使用自定义的 Pinia 插件实现状态持久化，支持将 store 状态保存到 localStorage 或 sessionStorage。

## 功能特性

- **灵活配置**: 可为每个 store 单独配置持久化策略
- **路径选择**: 支持指定需要持久化的 state 路径
- **存储选择**: 支持 localStorage 和 sessionStorage
- **自动恢复**: 应用启动时自动从存储中恢复状态
- **类型安全**: 完整的 TypeScript 类型支持

## 使用方法

### 1. 基础配置

在 store 定义中添加 `persist` 配置：

```typescript
import { defineStore } from 'pinia'

export const useAppStore = defineStore('app', () => {
  const theme = ref('light')
  const sidebarCollapsed = ref(false)

  return {
    theme,
    sidebarCollapsed
  }
}, {
  persist: {
    key: 'my-app-key',      // 自定义存储 key
    paths: ['theme', 'sidebarCollapsed'],  // 需要持久化的路径
    storage: 'localStorage'  // 存储类型（默认 localStorage）
  }
})
```

### 2. 配置选项

#### `persist: boolean`
启用持久化，使用默认配置：

```typescript
{
  persist: true  // 持久化整个 state
}
```

#### `persist: PersistOptions`
详细配置：

```typescript
interface PersistOptions {
  /**
   * 存储的 key，默认使用 store.$id
   */
  key?: string

  /**
   * 存储类型，默认 localStorage
   */
  storage?: 'localStorage' | 'sessionStorage'

  /**
   * 需要持久化的 state 路径数组
   * 例如: ['sidebarCollapsed', 'theme', 'user.profile']
   * 如果不指定，则持久化整个 state
   */
  paths?: string[]
}
```

### 3. 使用示例

#### 示例 1: 持久化整个 store

```typescript
export const useSettingsStore = defineStore('settings', () => {
  const language = ref('zh-CN')
  const fontSize = ref(14)

  return {
    language,
    fontSize
  }
}, {
  persist: true  // 持久化所有状态
})
```

#### 示例 2: 持久化部分状态

```typescript
export const useUserStore = defineStore('user', () => {
  const token = ref('')
  const userInfo = ref(null)
  const temporaryData = ref([])  // 不需要持久化

  return {
    token,
    userInfo,
    temporaryData
  }
}, {
  persist: {
    paths: ['token', 'userInfo']  // 只持久化这两个字段
  }
})
```

#### 示例 3: 使用 sessionStorage

```typescript
export const useTempStore = defineStore('temp', () => {
  const draftData = ref('')

  return {
    draftData
  }
}, {
  persist: {
    storage: 'sessionStorage'  // 使用 sessionStorage，关闭标签页后清除
  }
})
```

#### 示例 4: 嵌套路径持久化

```typescript
export const useAppStore = defineStore('app', () => {
  const user = ref({
    profile: {
      name: '',
      avatar: ''
    },
    settings: {
      notifications: true
    }
  })

  return {
    user
  }
}, {
  persist: {
    paths: [
      'user.profile',      // 持久化用户信息
      'user.settings.notifications'  // 持久化通知设置
    ]
  }
})
```

## 项目中的实际应用

### app.ts - 应用配置

```typescript
export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(false)
  const theme = ref<Theme>('light')
  const language = ref<string>('zh-CN')

  return {
    sidebarCollapsed,
    theme,
    language
  }
}, {
  persist: {
    key: 'ai-pro-app',
    paths: ['sidebarCollapsed', 'theme', 'language', 'hostPageSize', 'sessionPageSize']
  }
})
```

持久化的状态：
- 侧边栏折叠状态
- 主题设置
- 语言设置
- 分页设置

### chat.ts - 聊天会话

```typescript
export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const currentSessionId = ref<string>('')
  const selectedHostIds = ref<string[]>([])

  return {
    sessions,
    currentSessionId,
    selectedHostIds
  }
}, {
  persist: {
    key: 'ai-pro-chat',
    paths: ['sessions', 'currentSessionId', 'selectedHostIds']
  }
})
```

持久化的状态：
- 会话列表
- 当前会话 ID
- 选中的主机 ID

**消息缓存**: 聊天消息通过单独的缓存机制管理，带有时效性（30分钟）：

```typescript
// 加载历史消息时会自动缓存
async function loadHistory(sessionId: string) {
  const data = await getChatHistory(sessionId)
  messages.value = data
  saveMessageCache(sessionId, data)  // 缓存到 localStorage
}

// 当 API 失败时，从缓存加载
catch (error) {
  const cached = loadMessageCache(sessionId)
  if (cached) {
    messages.value = cached
  }
}
```

### console.ts - 控制台

```typescript
export const useConsoleStore = defineStore('console', () => {
  const selectedHosts = ref<string[]>([])

  return {
    selectedHosts
  }
}, {
  persist: {
    key: 'ai-pro-console',
    paths: ['selectedHosts']
  }
})
```

持久化的状态：
- 选中的主机列表

## 工具函数

### 清除持久化数据

```typescript
import { clearPersistedState, clearAllPersistedState } from '@/stores/plugins/persist'

// 清除指定 store 的持久化数据
clearPersistedState('app', 'ai-pro-app')

// 清除所有持久化数据
clearAllPersistedState()
```

### 聊天消息缓存管理

```typescript
import { useChatStore } from '@/stores/chat'

const chatStore = useChatStore()

// 清除指定会话的消息缓存
chatStore.clearMessageCache('session-id')

// 清除所有消息缓存
chatStore.clearMessageCache()
```

## 存储结构

### localStorage

```
ai-pro-app          → 应用配置
ai-pro-chat         → 聊天会话基础信息
ai-pro-console      → 控制台选中主机
chat-messages-xxx   → 会话消息缓存（xxx 为 session id）
```

## 注意事项

1. **敏感数据**: 不要持久化敏感信息（如密码、token 除非必要）
2. **数据大小**: localStorage 有 5-10MB 限制，避免存储大量数据
3. **时效性**: 消息缓存设置了 30 分钟过期时间
4. **跨标签页**: 使用 localStorage 时，状态会在同源的所有标签页间共享
5. **清理机制**: 定期清理过期的缓存数据

## 最佳实践

1. **选择性持久化**: 只持久化必要的状态，避免存储临时状态
2. **合理命名**: 使用清晰的 key 命名，避免冲突
3. **错误处理**: 插件已内置错误处理，不会因存储异常影响应用运行
4. **版本迁移**: 当 state 结构变化时，考虑添加版本号进行迁移
5. **测试**: 在隐私模式下测试应用，确保降级体验正常

## 故障排查

### 状态未恢复

1. 检查浏览器是否禁用了 localStorage
2. 检查是否配置了 `paths` 但状态路径不正确
3. 查看控制台是否有存储错误

### 状态过期

- 检查是否手动清除了浏览器缓存
- 检查是否使用了 sessionStorage（关闭标签页会清除）

### 性能问题

- 避免频繁更新需要持久化的状态
- 对于大型数据，考虑使用 IndexedDB 代替 localStorage
- 使用 `paths` 选项只持久化必要的字段
