<template>
  <div class="result-container">
    <div class="result-container-header">
      <h3>结果展示</h3>
    </div>
    <div class="result-container-content">
      <p>结果容器组件（待实现）</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue'

defineProps<{
  results: any[]
  loading: boolean
}>()
</script>

<style scoped>
.result-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f5f5f5;
  border-radius: 8px;
  padding: 16px;
}

.result-container-header {
  margin-bottom: 16px;
}

.result-container-content {
  flex: 1;
  overflow: auto;
}
</style>

