# 状态持久化快速参考

## 快速开始

### 1. 为新 Store 添加持久化

```typescript
import { defineStore } from 'pinia'

export const useMyStore = defineStore('my', () => {
  const name = ref('')
  const count = ref(0)

  return { name, count }
}, {
  persist: {
    key: 'my-store',        // 可选，默认 'pinia-my'
    paths: ['name'],        // 可选，持久化 name 字段
    storage: 'localStorage' // 可选，默认 localStorage
  }
})
```

### 2. 组件中使用

```vue
<script setup lang="ts">
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

// 读取状态（自动从 localStorage 恢复）
console.log(appStore.theme)

// 修改状态（自动保存到 localStorage）
appStore.setTheme('dark')
</script>
```

## 配置选项速查

| 选项 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `persist` | `boolean \| object` | `undefined` | 启用持久化 |
| `persist.key` | `string` | `'pinia-' + store.$id` | 存储键名 |
| `persist.paths` | `string[]` | `undefined` | 要持久化的字段 |
| `persist.storage` | `'localStorage' \| 'sessionStorage'` | `'localStorage'` | 存储类型 |

## 已配置的 Store

| Store | 持久化字段 | 存储键 |
|-------|-----------|--------|
| `app.ts` | sidebarCollapsed, theme, language, hostPageSize, sessionPageSize | `ai-pro-app` |
| `chat.ts` | sessions, currentSessionId, selectedHostIds | `ai-pro-chat` |
| `console.ts` | selectedHosts | `ai-pro-console` |

## 常用操作

### 读取状态

```typescript
import { useAppStore } from '@/stores/app'
const appStore = useAppStore()

// 整个 store
console.log(appStore.theme)

// 使用 storeToRefs（保持响应性）
import { storeToRefs } from 'pinia'
const { theme, sidebarCollapsed } = storeToRefs(appStore)
```

### 修改状态

```typescript
// 直接修改
appStore.theme = 'dark'

// 通过 action
appStore.setTheme('dark')
appStore.toggleSidebar()
```

### 清除持久化数据

```typescript
// 清除特定 store
import { clearPersistedState } from '@/stores/plugins/persist'
clearPersistedState('app', 'ai-pro-app')

// 清除所有
clearAllPersistedState()

// 清除消息缓存
import { useChatStore } from '@/stores/chat'
const chatStore = useChatStore()
chatStore.clearMessageCache()      // 所有缓存
chatStore.clearMessageCache('id')  // 特定会话
```

## 消息缓存

- **缓存键**: `chat-messages-{sessionId}`
- **有效期**: 30 分钟
- **自动缓存**: 调用 `loadHistory()` 时自动缓存
- **降级处理**: API 失败时从缓存读取

## 存储位置

### localStorage
- `ai-pro-app` - 应用配置
- `ai-pro-chat` - 聊天基础状态
- `ai-pro-console` - 控制台状态
- `chat-messages-*` - 会话消息缓存

### sessionStorage
- 按需配置（默认未使用）

## 注意事项

✅ **推荐持久化**
- 用户偏好设置（主题、语言）
- UI 状态（侧边栏、分页大小）
- 会话选择（当前会话、选中项）

❌ **不建议持久化**
- 临时状态（loading、error）
- 敏感信息（token、密码）
- 大型数据（完整列表、详细记录）
- 实时数据（状态会在短时间内变化）

## 故障排查

### 状态未恢复
1. 检查浏览器是否禁用 localStorage
2. 检查 `paths` 配置是否正确
3. 查看控制台是否有错误

### 性能问题
1. 使用 `paths` 只持久化必要字段
2. 避免频繁更新持久化状态
3. 大型数据考虑 IndexedDB

## 相关文件

- `src/stores/plugins/persist.ts` - 插件实现
- `src/stores/plugins/README.md` - 详细文档
- `src/composables/usePersistedState.ts` - 使用示例
- `PERSISTENCE_MIGRATION.md` - 实施总结
