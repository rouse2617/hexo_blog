# 前端错误处理和重试机制

完整的错误处理和重试系统，提供统一的错误管理、网络状态监控和用户友好的错误提示。

## 功能特性

### 1. 统一错误处理 (`errorHandler.ts`)
- 自定义错误类 `AppError`
- 错误分类（网络、服务器、业务、验证、超时等）
- 错误级别（信息、警告、错误、致命）
- 错误节流（相同错误 5 秒内只显示一次）
- 控制台日志格式化
- 全局错误处理器

### 2. 网络状态监控 (`networkMonitor.ts`)
- 实时网络状态监听（在线/离线/慢速）
- 网络质量评估（优秀/良好/一般/较差）
- 自动健康检查（每 30 秒）
- 慢速网络检测
- Vue 3 Composable API 支持

### 3. HTTP 请求重试机制 (`request.ts`)
- 自动重试失败请求（默认 3 次）
- 指数退避策略
- 可配置重试条件
- 慢速网络自动增加超时时间
- 请求耗时监控
- 支持 `skipErrorHandler` 和 `skipRetry` 选项

### 4. 友好的错误提示组件 (`ErrorAlert.vue`)
- 美观的错误提示 UI
- 可折叠的详细信息
- 内置重试功能
- 自动关闭倒计时
- 鼠标悬停暂停

## 使用指南

### 基础使用

#### 1. HTTP 请求（自动错误处理）

```typescript
import { request } from '@/api/request'

// 自动错误处理和重试
const data = await request.get('/api/users')

// 禁用自动错误处理
const data = await request.get('/api/users', { skipErrorHandler: true })

// 禁用重试
const data = await request.get('/api/users', { skipRetry: true })

// 自定义重试次数
const data = await request.get('/api/users', { retryCount: 5 })
```

#### 2. 手动错误处理

```typescript
import { handleError, AppError, ErrorType, ErrorLevel } from '@/utils/errorHandler'

try {
  await someAsyncOperation()
} catch (error) {
  handleError(error)
}

// 创建自定义错误
const error = new AppError({
  message: '详细错误信息',
  type: ErrorType.BUSINESS,
  level: ErrorLevel.WARNING,
  userMessage: '用户友好的错误提示',
  statusCode: 400,
  retryable: true
})
handleError(error)
```

#### 3. 带重试的异步操作

```typescript
import { withRetry } from '@/utils/errorHandler'

const data = await withRetry(
  async () => {
    const response = await fetch('/api/data')
    return response.json()
  },
  {
    maxRetries: 3,
    retryDelay: 1000,
    backoffMultiplier: 2,
    onRetry: (error, attempt) => {
      console.log(`重试第 ${attempt} 次`, error)
    },
    shouldRetry: (error) => error.retryable
  }
)
```

#### 4. 网络状态监控

```typescript
import { useNetworkMonitor, isOnline, isSlowNetwork } from '@/utils/networkMonitor'

// 在 Vue 组件中使用
export default {
  setup() {
    const { networkInfo, refresh } = useNetworkMonitor()

    // 监听网络状态变化
    watch(() => networkInfo.value.online, (online) => {
      if (!online) {
        console.warn('网络已断开')
      }
    })

    return { networkInfo, refresh }
  }
}

// 在任何地方检查网络状态
if (isOnline()) {
  console.log('网络在线')
}

if (isSlowNetwork()) {
  console.log('当前网络较慢')
}
```

#### 5. 错误提示组件

```vue
<template>
  <ErrorAlert
    :error="error"
    :closable="true"
    :retryable="true"
    :auto-close="true"
    :duration="5000"
    @retry="handleRetry"
    @close="handleClose"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import ErrorAlert from '@/components/common/ErrorAlert.vue'
import { AppError } from '@/utils/errorHandler'

const error = ref<AppError | null>(null)

const handleRetry = async () => {
  // 重试逻辑
}

const handleClose = () => {
  error.value = null
}
</script>
```

### 高级配置

#### 1. 全局错误处理器配置

```typescript
import { errorHandler } from '@/utils/errorHandler'

errorHandler.setConfig({
  enableConsoleLog: true,      // 启用控制台日志
  enableNotification: false,   // 启用通知
  enableMessage: true,         // 启用消息提示
  onError: (error) => {
    // 自定义错误处理
    // 例如：上报到监控系统
    reportError(error)
  }
})
```

#### 2. HTTP 重试配置

```typescript
import { setRetryConfig } from '@/api/request'

setRetryConfig({
  maxRetries: 5,              // 最大重试次数
  retryDelay: 2000,           // 重试延迟（毫秒）
  retryCondition: (error) => {
    // 自定义重试条件
    return error.code === 'ECONNABORTED'
  }
})
```

#### 3. 错误边界处理

```typescript
import { withErrorHandling } from '@/utils/errorHandler'

// 包装函数，自动处理错误
const safeFetch = withErrorHandling(async () => {
  const response = await fetch('/api/data')
  return response.json()
}, true) // true 表示显示错误消息

// 使用
try {
  const data = await safeFetch()
} catch (error) {
  // 错误已被处理
  console.log('请求失败，但用户已收到提示')
}
```

## API 文档

### errorHandler

#### `handleError(error: any, showMessage?: boolean): AppError`
处理错误并显示用户友好的提示。

- `error`: 错误对象（可以是任意类型）
- `showMessage`: 是否显示错误消息（默认 true）
- 返回: `AppError` 实例

#### `withRetry<T>(fn: () => Promise<T>, options?: RetryOptions): Promise<T>`
执行带重试的异步操作。

- `fn`: 异步函数
- `options.maxRetries`: 最大重试次数（默认 3）
- `options.retryDelay`: 重试延迟（默认 1000ms）
- `options.backoffMultiplier`: 退避倍数（默认 2）
- `options.onRetry`: 重试回调
- `options.shouldRetry`: 是否应该重试的判断函数

### networkMonitor

#### `useNetworkMonitor()`
Vue 3 网络监控 Hook。

返回:
- `networkInfo`: 响应式网络信息
- `addListener(callback)`: 添加监听器
- `removeListener(callback)`: 移除监听器
- `refresh()`: 刷新网络状态

#### `isOnline(): boolean`
检查网络是否在线。

#### `isSlowNetwork(): boolean`
检查是否是慢速网络。

### request

#### `request.get<T>(url, config?)`
GET 请求。

#### `request.post<T>(url, data?, config?)`
POST 请求。

#### `request.put<T>(url, data?, config?)`
PUT 请求。

#### `request.delete<T>(url, config?)`
DELETE 请求。

#### `request.upload<T>(url, file, config?)`
文件上传。

#### `request.download(url, config?)`
文件下载。

#### `request.withRetry<T>(fn, options?)`
带重试的请求。

## 最佳实践

### 1. 错误处理策略

```typescript
// ✅ 推荐：使用统一的错误处理
try {
  await request.get('/api/data')
} catch (error) {
  // 错误已被自动处理
  console.error('操作失败:', error)
}

// ❌ 不推荐：手动处理每个错误
try {
  await axios.get('/api/data')
} catch (error) {
  if (error.response?.status === 401) {
    // ...
  } else if (error.response?.status === 500) {
    // ...
  }
}
```

### 2. 网络状态感知

```typescript
// ✅ 推荐：根据网络状态调整行为
import { isSlowNetwork } from '@/utils/networkMonitor'

const timeout = isSlowNetwork() ? 60000 : 30000
const data = await request.get('/api/data', { timeout })

// ❌ 不推荐：固定的超时时间
const data = await request.get('/api/data', { timeout: 30000 })
```

### 3. 重试策略

```typescript
// ✅ 推荐：重要操作使用重试
const result = await withRetry(
  () => request.post('/api/important-action', data),
  { maxRetries: 3 }
)

// ❌ 不推荐：重要操作不使用重试
const result = await request.post('/api/important-action', data)
```

### 4. 错误提示

```typescript
// ✅ 推荐：使用 ErrorAlert 组件
<ErrorAlert :error="error" @retry="retry" />

// ❌ 不推荐：直接使用 ElMessage
ElMessage.error(error.message)
```

## 错误类型说明

| 类型 | 说明 | 可重试 | 示例 |
|------|------|--------|------|
| `NETWORK` | 网络连接失败 | ✅ | 断网、DNS 解析失败 |
| `SERVER` | 服务器错误 | ✅ | 500、502、503 |
| `BUSINESS` | 业务逻辑错误 | ❌ | 权限不足、资源不存在 |
| `VALIDATION` | 数据验证失败 | ❌ | 参数错误、格式错误 |
| `TIMEOUT` | 请求超时 | ✅ | 网络延迟 |
| `UNKNOWN` | 未知错误 | ❌ | 未预期的错误 |

## 错误级别说明

| 级别 | 说明 | 表现形式 |
|------|------|----------|
| `INFO` | 信息提示 | 成功消息 |
| `WARNING` | 警告 | 警告消息，3 秒自动关闭 |
| `ERROR` | 错误 | 错误消息，3 秒自动关闭 |
| `FATAL` | 致命错误 | 通知，需要手动关闭 |

## 技术细节

### 重试机制

- 使用指数退避策略：延迟时间 = 基础延迟 × (2 ^ 重试次数)
- 默认重试 3 次，可自定义
- 只对可重试的错误进行重试（网络错误、超时、5xx 错误）
- 4xx 错误默认不重试

### 网络监控

- 使用 Network Information API（浏览器支持时）
- 每 30 秒进行一次健康检查
- 根据连接类型、下行速度、RTT 评估网络质量
- 监听在线/离线事件

### 错误节流

- 相同错误 5 秒内只显示一次
- 避免因重复请求导致的错误提示刷屏
- 基于错误类型、状态码、消息生成唯一标识

## 兼容性

- Vue 3.4+
- TypeScript 5.0+
- Element Plus 2.0+
- 现代浏览器（支持 ES2020）

## 性能影响

- 错误处理器：< 1ms
- 网络监控：~100ms/次（30 秒间隔）
- 重试机制：仅在失败时触发
- 总体性能影响：可忽略不计

## 维护建议

1. 定期检查错误日志，优化常见错误处理
2. 根据业务需求调整重试策略
3. 监控网络质量，优化慢速网络体验
4. 及时更新错误提示文案，提高用户体验
