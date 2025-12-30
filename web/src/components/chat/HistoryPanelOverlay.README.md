# HistoryPanelOverlay Component

**移动端历史面板侧滑组件** - Mobile History Panel Side Drawer

## 快速概览

一个专为移动端优化的历史会话侧滑面板组件，提供流畅的触摸手势支持和优雅的动画效果。

## 核心功能

### 1. 侧滑抽屉效果 (Side Drawer)
- 从左侧滑入的抽屉面板
- 固定定位，覆盖在主内容之上
- 最大宽度 85vw，防止在小屏幕上占用过多空间

### 2. 点击遮罩关闭 (Click Mask to Close)
- 半透明黑色背景遮罩 (rgba(0, 0, 0, 0.5))
- 点击遮罩区域即可关闭面板
- 平滑的淡入淡出动画

### 3. 触摸滑动关闭 (Touch Swipe to Close)
- 从左向右滑动即可关闭面板
- 滑动阈值: 50px
- 实时视觉反馈（拖拽时面板跟随手指移动）
- 智能区分垂直滚动和水平滑动

### 4. 完整的 TypeScript 支持
- 严格的类型定义
- 清晰的 Props 和 Emits 接口
- JSDoc 注释完善

## Props

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `visible` | `boolean` | `false` | 控制面板显示/隐藏 |
| `sessions` | `Array<Session>` | `[]` | 会话列表 |
| `currentSessionId` | `string` | `''` | 当前激活的会话 ID |

### Session 类型

```typescript
interface Session {
  id: string
  title: string
  createdAt: string | number
}
```

## Events

| 事件 | 参数 | 说明 |
|------|------|------|
| `close` | - | 面板关闭时触发 |
| `sessionSelect` | `sessionId: string` | 选择会话时触发 |
| `sessionDelete` | `sessionId: string` | 删除会话时触发 |

## 使用示例

### 基础用法

```vue
<template>
  <HistoryPanelOverlay
    :visible="isVisible"
    :sessions="sessions"
    :current-session-id="currentId"
    @close="handleClose"
    @session-select="handleSelect"
    @session-delete="handleDelete"
  />
</template>

<script setup lang="ts">
import { ref } from 'vue'
import HistoryPanelOverlay from '@/components/chat/HistoryPanelOverlay.vue'

const isVisible = ref(false)
const sessions = ref([
  { id: '1', title: '系统监控', createdAt: '2025-12-31' },
  { id: '2', title: '日志分析', createdAt: '2025-12-30' }
])
const currentId = ref('1')

const handleClose = () => {
  isVisible.value = false
}

const handleSelect = (sessionId: string) => {
  console.log('Selected session:', sessionId)
  currentId.value = sessionId
  isVisible.value = false
}

const handleDelete = (sessionId: string) => {
  console.log('Delete session:', sessionId)
}
</script>
```

### 与 Store 集成

```vue
<template>
  <HistoryPanelOverlay
    :visible="uiStore.historyPanelOverlayVisible"
    :sessions="chatStore.sessions"
    :current-session-id="chatStore.currentSessionId"
    @close="uiStore.closeHistoryOverlay()"
    @session-select="handleSessionSelect"
    @session-delete="chatStore.removeSession"
  />
</template>

<script setup lang="ts">
import { useChatStore } from '@/stores/chat'
import { useUIStore } from '@/stores/ui'

const chatStore = useChatStore()
const uiStore = useUIStore()

const handleSessionSelect = async (sessionId: string) => {
  if (sessionId === '__new__') {
    await chatStore.newSession()
  } else {
    await chatStore.switchSession(sessionId)
  }
  uiStore.closeHistoryOverlay()
}
</script>
```

## 手势详解

### 触摸事件处理

```typescript
// 1. touchstart - 记录初始位置
handleTouchStart(event: TouchEvent) {
  touchStartX = event.touches[0].clientX
  touchStartY = event.touches[0].clientY
}

// 2. touchmove - 检测水平滑动
handleTouchMove(event: TouchEvent) {
  const deltaX = currentX - touchStartX

  // 仅在水平移动大于垂直移动时触发
  if (Math.abs(deltaX) > deltaY) {
    // 应用视觉反馈
    panelRef.value.style.transform = `translateX(${deltaX}px)`
  }
}

// 3. touchend - 判断是否关闭
handleTouchEnd(event: TouchEvent) {
  const deltaX = touchEndX - touchStartX

  // 超过阈值则关闭
  if (deltaX > 50) {
    handleClose()
  } else {
    // 重置位置
    panelRef.value.style.transform = ''
  }
}
```

## 样式定制

### CSS 变量

组件使用以下 CSS 变量，可以在全局样式中覆盖：

```css
:root {
  /* Z-index 层级 */
  --z-modal-backdrop: 2000;
  --z-modal: 2001;

  /* 侧边栏样式 */
  --sidebar-bg-start: #1e293b;
  --sidebar-bg-end: #0f172a;
  --sidebar-border: rgba(255, 255, 255, 0.1);
  --sidebar-text: #cbd5e1;
  --sidebar-text-hover: #f1f5f9;
  --sidebar-active-bg: rgba(59, 130, 246, 0.2);
  --sidebar-active-text: #60a5fa;

  /* 间距 */
  --spacing-1: 4px;
  --spacing-2: 8px;
  --spacing-3: 12px;
  --spacing-4: 16px;
  --spacing-8: 32px;

  /* 圆角 */
  --radius-lg: 8px;

  /* 过渡动画 */
  --transition-fast: 0.15s;
  --transition-base: 0.3s;

  /* 字体 */
  --text-sm: 14px;
  --text-base: 16px;
  --font-semibold: 600;
}
```

### 自定义样式

```vue
<style scoped>
/* 修改面板宽度 */
.history-panel-overlay {
  width: 320px; /* 默认 280px */
}

/* 修改遮罩颜色 */
.history-overlay-backdrop {
  background: rgba(0, 0, 0, 0.7); /* 默认 0.5 */
}

/* 修改动画时长 */
.slide-left-enter-active,
.slide-left-leave-active {
  transition: transform 0.5s; /* 默认 0.3s */
}
</style>
```

## 响应式断点

```css
/* 移动端 */
@media (max-width: 767px) {
  .history-panel-overlay {
    width: 280px;
    max-width: 85vw;
  }
}

/* 平板及以上 */
@media (min-width: 768px) {
  .history-panel-overlay {
    width: 320px;
  }
}
```

## 性能优化

1. **GPU 加速**: 使用 `transform` 而非 `position`
2. **条件渲染**: 使用 `v-if` 而非 `v-show`
3. **事件优化**: 智能判断滑动方向，避免误触发
4. **滚动优化**: `-webkit-overflow-scrolling: touch` (iOS)

## 无障碍支持

```css
/* 尊重用户的减少动画偏好 */
@media (prefers-reduced-motion: reduce) {
  .overlay-fade-enter-active,
  .overlay-fade-leave-active,
  .slide-left-enter-active,
  .slide-left-leave-active {
    transition: none;
  }
}
```

## 常见问题

### Q: 如何修改滑动阈值？

A: 修改组件中的 `SWIPE_THRESHOLD` 常量：

```typescript
const SWIPE_THRESHOLD = 80 // 默认 50
```

### Q: 如何禁用滑动关闭？

A: 注释掉模板中的触摸事件监听：

```vue
<aside
  v-if="visible"
  ref="panelRef"
  class="history-panel-overlay"
  <!-- @touchstart="handleTouchStart"
  @touchmove="handleTouchMove"
  @touchend="handleTouchEnd" -->
>
```

### Q: 如何添加滑动打开手势？

A: 在父组件中监听主区域的触摸事件：

```vue
<template>
  <div
    class="main-content"
    @touchstart="handleSwipeStart"
    @touchmove="handleSwipeMove"
    @touchend="handleSwipeEnd"
  >
    <!-- 主内容 -->
  </div>
</template>

<script setup lang="ts">
let swipeStartX = 0

const handleSwipeStart = (e: TouchEvent) => {
  swipeStartX = e.touches[0].clientX
}

const handleSwipeMove = (e: TouchEvent) => {
  const deltaX = e.touches[0].clientX - swipeStartX
  // 从左边缘向右滑动
  if (deltaX > 50 && swipeStartX < 50) {
    uiStore.toggleHistoryOverlay()
  }
}
</script>
```

## 相关文件

- **组件位置**: `/web/src/components/chat/HistoryPanelOverlay.vue`
- **类型定义**: `/web/src/types/chat-ui.ts`
- **UI Store**: `/web/src/stores/ui.ts`
- **Chat Store**: `/web/src/stores/chat.ts`
- **使用指南**: `/web/src/components/chat/ComponentUsageGuide.md`

## 依赖

- Vue 3 (Composition API)
- Element Plus (Icons)
- TypeScript

## 版本历史

- **v1.0.0** (2025-12-31): 初始版本
  - 侧滑抽屉效果
  - 点击遮罩关闭
  - 触摸滑动关闭
  - 完整 TypeScript 支持
