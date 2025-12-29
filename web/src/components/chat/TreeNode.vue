<template>
  <div class="tree-node" :class="{ 'is-root': isRoot }">
    <div class="node-content" @click="toggleExpand">
      <!-- 展开/折叠图标 -->
      <span
        v-if="isExpandable"
        class="node-toggle"
        :class="{ expanded: isExpanded }"
      >
        <el-icon><ArrowRight /></el-icon>
      </span>
      <span v-else class="node-toggle-placeholder"></span>

      <!-- Key -->
      <span v-if="!isRoot" class="node-key">{{ nodeKey }}:</span>

      <!-- Value preview (折叠状态) -->
      <span v-if="isExpandable && !isExpanded" class="node-preview">
        {{ valuePreview }}
      </span>

      <!-- Value (非对象/数组) -->
      <span v-else-if="!isExpandable" class="node-value" :class="valueClass">
        {{ formattedValue }}
      </span>

      <!-- Type badge -->
      <span v-if="isExpandable && !isExpanded" class="node-type-badge">
        {{ itemType }}
      </span>
    </div>

    <!-- 展开的子节点 -->
    <div v-if="isExpandable && isExpanded" class="node-children">
      <TreeNode
        v-for="(childValue, childKey) in children"
        :key="childKey"
        :node-key="String(childKey)"
        :value="childValue"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ArrowRight } from '@element-plus/icons-vue'

const props = defineProps<{
  nodeKey: string
  value: any
  isRoot?: boolean
}>()

const isExpanded = ref(!props.isRoot) // 根节点默认折叠

const isExpandable = computed(() => {
  return typeof props.value === 'object' && props.value !== null
})

const itemType = computed(() => {
  if (Array.isArray(props.value)) {
    return `Array(${props.value.length})`
  }
  const keys = Object.keys(props.value)
  return `Object{${keys.length}}`
})

const valuePreview = computed(() => {
  if (Array.isArray(props.value)) {
    return `[${props.value.length} items]`
  }
  const keys = Object.keys(props.value)
  return `{${keys.length} keys}`
})

const formattedValue = computed(() => {
  if (props.value === null) return 'null'
  if (props.value === undefined) return 'undefined'
  if (typeof props.value === 'string') return `"${props.value}"`
  return String(props.value)
})

const valueClass = computed(() => {
  if (props.value === null) return 'value-null'
  if (typeof props.value === 'boolean') return 'value-boolean'
  if (typeof props.value === 'number') return 'value-number'
  if (typeof props.value === 'string') return 'value-string'
  return ''
})

const children = computed(() => {
  return props.value
})

const toggleExpand = () => {
  if (isExpandable.value) {
    isExpanded.value = !isExpanded.value
  }
}
</script>

<style scoped>
.tree-node {
  margin-left: 0;
}

.tree-node.is-root {
  margin-left: 0;
}

.tree-node:not(.is-root) {
  margin-left: 20px;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s;
  border-radius: 4px;
}

.node-content:hover {
  background: rgba(0, 0, 0, 0.04);
}

.node-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  transition: transform 0.2s;
  color: #6b7280;
  flex-shrink: 0;
}

.node-toggle.expanded {
  transform: rotate(90deg);
}

.node-toggle-placeholder {
  width: 16px;
  flex-shrink: 0;
}

.node-key {
  color: #8b5cf6;
  font-weight: 500;
}

.node-value {
  word-break: break-all;
}

.value-string {
  color: #10b981;
}

.value-number {
  color: #f59e0b;
}

.value-boolean {
  color: #3b82f6;
}

.value-null {
  color: #ef4444;
}

.node-preview {
  color: #6b7280;
  font-style: italic;
}

.node-type-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  background: #e5e7eb;
  border-radius: 4px;
  font-size: 11px;
  color: #4b5563;
  margin-left: 4px;
}

.node-children {
  margin-top: 2px;
}
</style>
