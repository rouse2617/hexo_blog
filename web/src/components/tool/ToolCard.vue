<template>
  <div class="tool-card" :class="{ disabled: !tool.enabled }">
    <div class="tool-header">
      <div class="tool-icon">
        <el-icon :size="24" :color="tool.enabled ? '#409eff' : '#c0c4cc'">
          <component :is="getToolIcon(tool.type)" />
        </el-icon>
      </div>
      <div class="tool-info">
        <h3 class="tool-name">{{ tool.name }}</h3>
        <el-tag :type="tool.type === 'builtin' ? 'primary' : 'success'" size="small">
          {{ tool.type === 'builtin' ? '内置' : '脚本' }}
        </el-tag>
      </div>
      <el-switch
        :model-value="tool.enabled"
        @change="handleToggle"
        :loading="toggling"
      />
    </div>
    <div class="tool-body">
      <p class="tool-description">{{ tool.description }}</p>
      <div v-if="tool.parameters && tool.parameters.length > 0" class="tool-params">
        <div class="params-title">参数列表</div>
        <div class="params-list">
          <div
            v-for="param in tool.parameters"
            :key="param.name"
            class="param-item"
          >
            <span class="param-name">
              {{ param.name }}
              <span v-if="param.required" class="required">*</span>
            </span>
            <span class="param-type">{{ param.type }}</span>
            <span class="param-desc">{{ param.description }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Tool } from '@/api/tool'
import { SetUp, Document } from '@element-plus/icons-vue'

const props = defineProps<{
  tool: Tool
}>()

const emit = defineEmits<{
  toggle: [name: string, enabled: boolean]
}>()

const toggling = ref(false)

const getToolIcon = (type: string) => {
  return type === 'builtin' ? SetUp : Document
}

const handleToggle = async (enabled: boolean) => {
  toggling.value = true
  try {
    emit('toggle', props.tool.name, enabled)
  } finally {
    toggling.value = false
  }
}
</script>

<style scoped>
.tool-card {
  background: #fff;
  border-radius: 10px;
  border: 1px solid #ebeef5;
  padding: 20px;
  transition: all 0.3s;
}

.tool-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.tool-card.disabled {
  opacity: 0.6;
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 15px;
}

.tool-icon {
  width: 48px;
  height: 48px;
  background: #f5f7fa;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tool-info {
  flex: 1;
}

.tool-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 6px;
}

.tool-body {
  color: #606266;
}

.tool-description {
  font-size: 14px;
  line-height: 1.6;
  margin: 0 0 15px;
}

.tool-params {
  background: #f8fafc;
  border-radius: 6px;
  padding: 12px;
}

.params-title {
  font-size: 12px;
  color: #909399;
  margin-bottom: 10px;
}

.params-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.param-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}

.param-name {
  font-weight: 500;
  color: #303133;
  min-width: 100px;
}

.param-name .required {
  color: #f56c6c;
}

.param-type {
  background: #e4e7ed;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
}

.param-desc {
  color: #909399;
  flex: 1;
}
</style>
