# 状态持久化实施总结

## 已完成的工作

### 1. 创建持久化插件

**文件**: `web/src/stores/plugins/persist.ts`

- 实现了 Pinia 持久化插件
- 支持 localStorage 和 sessionStorage
- 支持选择性持久化（通过 paths 配置）
- 自动类型扩展（TypeScript）
- 内置错误处理

### 2. 创建应用配置 Store

**文件**: `web/src/stores/app.ts`

持久化的状态：
- `sidebarCollapsed`: 侧边栏折叠状态
- `theme`: 主题（light/dark/auto）
- `language`: 语言设置
- `hostPageSize`: 主机列表每页数量
- `sessionPageSize`: 会话列表每页数量

功能：
- 主题切换和应用
- 侧边栏状态管理
- 语言切换
- 自动主题初始化

### 3. 增强 Chat Store

**文件**: `web/src/stores/chat.ts`

持久化的状态：
- `sessions`: 会话列表
- `currentSessionId`: 当前会话 ID
- `selectedHostIds`: 选中的主机 ID

新增功能：
- 消息缓存机制（30分钟有效期）
- 缓存读取和保存
- 缓存清除
- API 失败时从缓存恢复

### 4. 增强 Console Store

**文件**: `web/src/stores/console.ts`

持久化的状态：
- `selectedHosts`: 选中的主机列表

### 5. 注册插件

**文件**: `web/src/main.ts`

- 创建 Pinia 实例
- 注册持久化插件
- 初始化应用主题

### 6. 更新 App.vue

**文件**: `web/src/App.vue`

- 使用 `useAppStore` 管理侧边栏状态
- 状态自动持久化

### 7. 创建文档和示例

**文件**:
- `web/src/stores/plugins/README.md`: 完整的使用文档
- `web/src/stores/__tests__/persist.test.ts`: 测试示例
- `web/src/composables/usePersistedState.ts`: 使用示例

## 数据存储结构

### localStorage Keys

```
ai-pro-app          → 应用配置
ai-pro-chat         → 聊天基础状态
ai-pro-console      → 控制台状态
chat-messages-xxx   → 会话消息缓存（xxx = sessionId）
```

## 使用方法

### 为新 Store 添加持久化

```typescript
import { defineStore } from 'pinia'

export const useMyStore = defineStore('my', () => {
  const data = ref('')

  return { data }
}, {
  persist: {
    key: 'my-store-key',
    paths: ['data'],
    storage: 'localStorage'
  }
})
```

### 在组件中使用

```typescript
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

// 读取状态（自动从 localStorage 恢复）
console.log(appStore.theme)

// 修改状态（自动保存到 localStorage）
appStore.setTheme('dark')
```

## 配置说明

### 全局配置（main.ts）

```typescript
pinia.use(createPersistPlugin({
  global: false,  // 不全局启用，每个 store 单独配置
  storage: 'localStorage'
}))
```

### Store 配置

```typescript
{
  persist: {
    key: 'custom-key',        // 可选，默认使用 store.$id
    paths: ['field1', 'field2'],  // 可选，持久化的字段
    storage: 'localStorage'   // 可选，默认 localStorage
  }
}
```

## 注意事项

1. **敏感数据**: 不要持久化 token、密码等敏感信息
2. **数据大小**: localStorage 有 5-10MB 限制
3. **错误处理**: 插件已内置错误处理，不会影响应用运行
4. **缓存时效**: 消息缓存默认 30 分钟过期
5. **跨标签页**: localStorage 会在同源标签页间共享状态

## 清除持久化数据

### 清除特定 Store

```typescript
import { clearPersistedState } from '@/stores/plugins/persist'

clearPersistedState('app', 'ai-pro-app')
```

### 清除所有数据

```typescript
import { clearAllPersistedState } from '@/stores/plugins/persist'

clearAllPersistedState()
```

### 清除消息缓存

```typescript
import { useChatStore } from '@/stores/chat'

const chatStore = useChatStore()

// 清除所有缓存
chatStore.clearMessageCache()

// 清除特定会话缓存
chatStore.clearMessageCache('session-id')
```

## 测试

运行以下命令测试构建：

```bash
cd web
npm run build
```

持久化相关的 stores 没有 TypeScript 错误。

## 后续优化建议

1. **数据迁移**: 当 store 结构变化时，添加版本管理和迁移逻辑
2. **压缩**: 对于大型数据，考虑压缩后存储
3. **IndexedDB**: 对于超过 5MB 的数据，使用 IndexedDB 代替 localStorage
4. **加密**: 敏感数据加密后再存储
5. **同步**: 考虑多标签页间的状态同步机制

## 相关文件

- `web/src/stores/plugins/persist.ts` - 持久化插件核心实现
- `web/src/stores/app.ts` - 应用配置 store
- `web/src/stores/chat.ts` - 聊天 store（含消息缓存）
- `web/src/stores/console.ts` - 控制台 store
- `web/src/main.ts` - 插件注册
- `web/src/App.vue` - 应用入口（使用持久化状态）
- `web/src/stores/plugins/README.md` - 详细文档
- `web/src/composables/usePersistedState.ts` - 使用示例
