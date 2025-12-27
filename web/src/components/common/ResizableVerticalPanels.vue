<template>
  <div class="resizable-vertical-panels" :style="{ height: height }">
    <div
      class="panel top-panel"
      :style="{ height: topHeight + 'px', minHeight: minTopHeight + 'px', maxHeight: maxTopHeight + 'px' }"
    >
      <slot name="top"></slot>
    </div>
    <div
      class="resizer"
      @mousedown="handleMouseDown"
      :class="{ dragging: isDragging }"
    ></div>
    <div class="panel bottom-panel" :style="{ height: `calc(100% - ${topHeight + 8}px)` }">
      <slot name="bottom"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface Props {
  initialTopHeight?: number
  minTopHeight?: number
  maxTopHeight?: number
  height?: string
}

const props = withDefaults(defineProps<Props>(), {
  initialTopHeight: 300,
  minTopHeight: 200,
  maxTopHeight: 600,
  height: '100%'
})

const topHeight = ref(props.initialTopHeight)
const isDragging = ref(false)

const handleMouseDown = (e: MouseEvent) => {
  isDragging.value = true
  const startY = e.clientY
  const startHeight = topHeight.value

  const handleMouseMove = (e: MouseEvent) => {
    const diff = e.clientY - startY
    let newHeight = startHeight + diff
    
    if (newHeight < props.minTopHeight) {
      newHeight = props.minTopHeight
    } else if (newHeight > props.maxTopHeight) {
      newHeight = props.maxTopHeight
    }
    
    topHeight.value = newHeight
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
watch(() => props.initialTopHeight, (newVal) => {
  topHeight.value = newVal
})
</script>

<style scoped>
.resizable-vertical-panels {
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
  height: 100%;
}

.panel {
  overflow: hidden;
}

.top-panel {
  flex-shrink: 0;
}

.bottom-panel {
  flex: 1;
  min-height: 0;
}

.resizer {
  height: 8px;
  background: var(--el-border-color);
  cursor: row-resize;
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
  right: 0;
  height: 100%;
}
</style>

