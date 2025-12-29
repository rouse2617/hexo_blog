<template>
  <div class="json-viewer">
    <TreeNode
      v-for="(value, key) in displayData"
      :key="key"
      :node-key="String(key)"
      :value="value"
      :is-root="true"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import TreeNode from './TreeNode.vue'

const props = defineProps<{
  data: any
}>()

const displayData = computed(() => {
  if (Array.isArray(props.data)) {
    return props.data
  }
  return props.data
})
</script>

<style scoped>
.json-viewer {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
}
</style>
