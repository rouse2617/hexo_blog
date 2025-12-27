<template>
  <div class="host-selector">
    <el-select
      v-model="selectedHosts"
      multiple
      collapse-tags
      collapse-tags-tooltip
      placeholder="选择目标主机（可多选）"
      style="width: 100%"
      @change="handleChange"
    >
      <el-option
        v-for="host in hosts"
        :key="host.id"
        :label="host.name"
        :value="host.id"
      >
        <div class="host-option">
          <span class="host-name">{{ host.name }}</span>
          <span class="host-address">{{ host.host }}:{{ host.port }}</span>
          <el-tag
            :type="host.status === 'online' ? 'success' : 'info'"
            size="small"
          >
            {{ host.status === 'online' ? '在线' : '离线' }}
          </el-tag>
        </div>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useHostStore } from '@/stores/host'

const props = defineProps<{
  modelValue: string[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const hostStore = useHostStore()
const hosts = computed(() => hostStore.allHosts)
const selectedHosts = ref<string[]>(props.modelValue || [])

onMounted(async () => {
  await hostStore.loadAllHosts()
})

watch(
  () => props.modelValue,
  (val) => {
    selectedHosts.value = val || []
  }
)

const handleChange = (val: string[]) => {
  emit('update:modelValue', val)
}
</script>

<style scoped>
.host-selector {
  width: 100%;
}

.host-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.host-name {
  font-weight: 500;
}

.host-address {
  color: #909399;
  font-size: 12px;
  flex: 1;
}
</style>
