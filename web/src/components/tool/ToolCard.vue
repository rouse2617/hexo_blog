<template>
  <div class="tool-card" :class="{ disabled: !tool.enabled }">
    <!-- 卡片头部 -->
    <div class="card-header">
      <div class="tool-identity">
        <div class="tool-icon-wrapper" :class="{ disabled: !tool.enabled }">
          <el-icon :size="24">
            <component :is="getToolIcon(tool.type)" />
          </el-icon>
        </div>
        <div class="tool-meta">
          <h3 class="tool-name">{{ tool.name }}</h3>
          <el-tag :type="tool.type === 'builtin' ? 'primary' : 'success'" size="small">
            {{ tool.type === 'builtin' ? '内置' : '脚本' }}
          </el-tag>
        </div>
      </div>
      <el-switch
        :model-value="tool.enabled"
        @change="handleToggle"
        :loading="toggling"
        size="large"
      />
    </div>

    <!-- 卡片内容 -->
    <div class="card-content">
      <p class="tool-description">{{ tool.description }}</p>

      <!-- 参数列表 - 可折叠 -->
      <el-collapse v-if="tool.parameters && tool.parameters.length > 0" class="params-collapse">
        <el-collapse-item name="params">
          <template #title>
            <div class="params-title">
              <el-icon><Document /></el-icon>
              <span>参数列表</span>
              <el-badge :value="tool.parameters.length" :max="99" />
            </div>
          </template>
          <div class="params-list">
            <div
              v-for="param in tool.parameters"
              :key="param.name"
              class="param-item"
            >
              <div class="param-header">
                <span class="param-name">
                  {{ param.name }}
                  <el-tag v-if="param.required" size="small" type="danger">必填</el-tag>
                </span>
                <el-tag size="small" type="info">{{ param.type }}</el-tag>
              </div>
              <div class="param-desc">{{ param.description }}</div>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
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
    setTimeout(() => {
      toggling.value = false
    }, 300)
  }
}
</script>

<style scoped>
.tool-card {
  background: #fff;
  border-radius: var(--radius-xl);
  border: 1px solid var(--color-gray-200);
  overflow: hidden;
  transition: all var(--transition-base);
}

.tool-card:hover {
  box-shadow: var(--shadow-xl);
  transform: translateY(-4px);
  border-color: var(--color-primary-light);
}

.tool-card.disabled {
  opacity: 0.6;
}

/* 卡片头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-5);
  background: linear-gradient(to bottom, var(--color-gray-50), transparent);
  border-bottom: 1px solid var(--color-gray-100);
}

.tool-identity {
  display: flex;
  align-items: center;
  gap: var(--spacing-4);
  flex: 1;
}

.tool-icon-wrapper {
  width: 52px;
  height: 52px;
  background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2563eb;
  transition: all var(--transition-base);
}

.tool-icon-wrapper.disabled {
  background: var(--color-gray-200);
  color: var(--color-gray-500);
}

.tool-card:hover .tool-icon-wrapper:not(.disabled) {
  background: linear-gradient(135deg, #60a5fa 0%, #3b82f6 100%);
  color: white;
  transform: rotate(5deg);
}

.tool-meta {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.tool-name {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-gray-800);
  margin: 0;
}

/* 卡片内容 */
.card-content {
  padding: var(--spacing-5);
}

.tool-description {
  font-size: var(--text-sm);
  color: var(--color-gray-600);
  line-height: var(--leading-relaxed);
  margin: 0 0 var(--spacing-4) 0;
}

/* 参数折叠 */
.params-collapse {
  border: none;
}

.params-collapse :deep(.el-collapse-item__header) {
  height: auto;
  padding: 0;
  border-bottom: none;
  background: transparent;
}

.params-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--text-sm);
  color: var(--color-gray-500);
  cursor: pointer;
  padding: var(--spacing-3);
  background: var(--color-gray-50);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.params-title:hover {
  background: var(--color-gray-100);
}

.params-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
  padding: var(--spacing-4);
  background: var(--color-gray-50);
  border-radius: var(--radius-md);
  margin-top: var(--spacing-3);
}

.param-item {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-1);
}

.param-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-2);
}

.param-name {
  font-weight: var(--font-medium);
  color: var(--color-gray-800);
  font-size: var(--text-sm);
}

.param-desc {
  font-size: var(--text-xs);
  color: var(--color-gray-500);
  line-height: var(--leading-normal);
}
</style>
