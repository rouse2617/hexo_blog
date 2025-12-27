<template>
  <div class="tool-list">
    <div v-if="loading" class="loading-state">
      <el-icon class="is-loading" :size="32"><Loading /></el-icon>
      <span>加载中...</span>
    </div>
    <div v-else-if="tools.length === 0" class="empty-state">
      <el-icon :size="48" color="#c0c4cc"><SetUp /></el-icon>
      <p>暂无工具</p>
    </div>
    <div v-else class="tool-grid">
      <ToolCard
        v-for="tool in tools"
        :key="tool.name"
        :tool="tool"
        @toggle="handleToggle"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Tool } from '@/api/tool'
import { Loading, SetUp } from '@element-plus/icons-vue'
import ToolCard from './ToolCard.vue'

defineProps<{
  tools: Tool[]
  loading: boolean
}>()

const emit = defineEmits<{
  toggle: [name: string, enabled: boolean]
}>()

const handleToggle = (name: string, enabled: boolean) => {
  emit('toggle', name, enabled)
}
</script>

<style scoped>
.tool-list {
  min-height: 200px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: #909399;
  gap: 15px;
}

.loading-state .is-loading {
  animation: rotating 1.5s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.tool-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
}
</style>
