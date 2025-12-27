<template>
  <div class="resizable-panels" :style="{ height: height }">
    <div
      class="panel left-panel"
      :style="{ width: leftWidth + 'px', minWidth: minLeftWidth + 'px', maxWidth: maxLeftWidth + 'px' }"
    >
      <slot name="left"></slot>
    </div>
    <div
      class="resizer"
      @mousedown="handleMouseDown"
      :class="{ dragging: isDragging }"
    ></div>
    <div class="panel right-panel" :style="{ width: `calc(100% - ${leftWidth + 8}px)` }">
      <slot name="right"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  initialLeftWidth?: number
  minLeftWidth?: number
  maxLeftWidth?: number
  height?: string
}

const props = withDefaults(defineProps<Props>(), {
  initialLeftWidth: 300,
  minLeftWidth: 200,
  maxLeftWidth: 800,
  height: '100%'
})

const leftWidth = ref(props.initialLeftWidth)
const isDragging = ref(false)

const handleMouseDown = (e: MouseEvent) => {
  isDragging.value = true
  const startX = e.clientX
  const startWidth = leftWidth.value

  const handleMouseMove = (e: MouseEvent) => {
    const diff = e.clientX - startX
    let newWidth = startWidth + diff
    
    if (newWidth < props.minLeftWidth) {
      newWidth = props.minLeftWidth
    } else if (newWidth > props.maxLeftWidth) {
      newWidth = props.maxLeftWidth
    }
    
    leftWidth.value = newWidth
  }

  const handleMouseUp = () => {
    isDragging.value = false
    document.removeEventListener('mousemove', handleMouseMove)
    document.removeEventListener('mouseup', handleMouseUp)
  }

  document.addEventListener('mousemove', handleMouseMove)
  document.addEventListener('mouseup', handleMouseUp)
}

// 监听props变化
watch(() => props.initialLeftWidth, (newVal) => {
  leftWidth.value = newVal
})
</script>

<style scoped>
.resizable-panels {
  display: flex;
  position: relative;
  width: 100%;
}

.panel {
  overflow: hidden;
}

.left-panel {
  flex-shrink: 0;
}

.right-panel {
  flex: 1;
  min-width: 0;
}

.resizer {
  width: 8px;
  background: var(--el-border-color);
  cursor: col-resize;
  flex-shrink: 0;
  position: relative;
  user-select: none;
}

.resizer:hover {
  background: var(--el-color-primary);
}

.resizer.dragging {
  background: var(--el-color-primary);
}

.resizer::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 100%;
}
</style>

