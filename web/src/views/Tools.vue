<template>
  <div class="tools-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <h2 class="page-title">工具列表</h2>
        <p class="page-description">管理内置工具和脚本工具</p>
      </div>
    </div>

    <!-- 统计卡片 - 改进版 -->
    <div class="stats-cards">
      <div class="stat-card primary">
        <div class="stat-icon">
          <el-icon><Tools /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ toolStore.tools.length }}</div>
          <div class="stat-label">全部工具</div>
        </div>
      </div>
      <div class="stat-card success">
        <div class="stat-icon">
          <el-icon><SetUp /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ toolStore.builtinTools.length }}</div>
          <div class="stat-label">内置工具</div>
        </div>
      </div>
      <div class="stat-card warning">
        <div class="stat-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ toolStore.scriptTools.length }}</div>
          <div class="stat-label">脚本工具</div>
        </div>
      </div>
      <div class="stat-card info">
        <div class="stat-icon">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ toolStore.enabledTools.length }}</div>
          <div class="stat-label">已启用</div>
        </div>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <!-- 类型筛选器 - 使用单选组 -->
        <el-radio-group v-model="activeTab" @change="handleTabChange" class="type-filter">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="builtin">内置工具</el-radio-button>
          <el-radio-button value="script">脚本工具</el-radio-button>
        </el-radio-group>

        <!-- 搜索框 -->
        <el-input
          v-model="keyword"
          placeholder="搜索工具名称或描述..."
          :prefix-icon="Search"
          clearable
          class="search-box"
        >
          <template #suffix>
            <el-tag v-if="keyword" size="small" type="info">
              {{ filteredTools.length }} 结果
            </el-tag>
          </template>
        </el-input>
      </div>

      <div class="toolbar-right">
        <el-button :icon="Refresh" @click="toolStore.loadTools()">刷新</el-button>
      </div>
    </div>

    <!-- 工具列表 - 优化版网格 -->
    <div class="tools-container">
      <div v-if="toolStore.loading" class="loading-state">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <span>加载中...</span>
      </div>
      <div v-else-if="filteredTools.length === 0" class="empty-state">
        <el-empty :description="keyword ? '未找到匹配的工具' : '暂无工具'">
          <el-button v-if="!keyword" type="primary">添加第一个工具</el-button>
        </el-empty>
      </div>
      <div v-else class="tools-grid-enhanced">
        <ToolCard
          v-for="tool in filteredTools"
          :key="tool.name"
          :tool="tool"
          @toggle="handleToggle"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Search, Refresh, Loading, Tools,
  SetUp, Document, CircleCheck
} from '@element-plus/icons-vue'
import { useToolStore } from '@/stores/tool'
import ToolCard from '@/components/tool/ToolCard.vue'

const toolStore = useToolStore()

const activeTab = ref('all')
const keyword = ref('')

onMounted(() => {
  toolStore.loadTools()
})

const filteredTools = computed(() => {
  let tools = toolStore.tools

  if (activeTab.value === 'builtin') {
    tools = toolStore.builtinTools
  } else if (activeTab.value === 'script') {
    tools = toolStore.scriptTools
  }

  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    tools = tools.filter(
      t => t.name.toLowerCase().includes(kw) ||
           t.description.toLowerCase().includes(kw)
    )
  }

  return tools
})

const handleTabChange = () => {
  // 切换 tab 时可以做一些处理
}

const handleToggle = async (name: string, enabled: boolean) => {
  try {
    await toolStore.toggle(name, enabled)
    ElMessage.success(enabled ? '已启用' : '已禁用')
  } catch (error) {
    // 错误已在 store 中处理
  }
}
</script>

<style scoped>
.tools-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-5);
}

.page-header {
  margin-bottom: var(--spacing-4);
}

.page-title {
  font-size: var(--text-3xl);
  font-weight: var(--font-semibold);
  color: var(--color-gray-800);
  margin: 0 0 var(--spacing-1) 0;
}

.page-description {
  font-size: var(--text-sm);
  color: var(--color-gray-500);
  margin: 0;
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--spacing-4);
}

.stat-card {
  background: #fff;
  border-radius: var(--radius-xl);
  padding: var(--spacing-5);
  display: flex;
  align-items: center;
  gap: var(--spacing-4);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--color-gray-200);
  position: relative;
  overflow: hidden;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
}

.stat-card.primary::before { background: linear-gradient(90deg, #3b82f6, #60a5fa); }
.stat-card.success::before { background: linear-gradient(90deg, #22c55e, #4ade80); }
.stat-card.warning::before { background: linear-gradient(90deg, #f59e0b, #fbbf24); }
.stat-card.info::before { background: linear-gradient(90deg, #06b6d4, #22d3ee); }

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.stat-card.primary .stat-icon { background: #dbeafe; color: #2563eb; }
.stat-card.success .stat-icon { background: #dcfce7; color: #16a34a; }
.stat-card.warning .stat-icon { background: #fef3c7; color: #d97706; }
.stat-card.info .stat-icon { background: #cffafe; color: #0891b2; }

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--color-gray-800);
  line-height: 1;
}

.stat-label {
  font-size: var(--text-sm);
  color: var(--color-gray-500);
  margin-top: var(--spacing-2);
}

/* 工具栏 */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  padding: var(--spacing-4);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--color-gray-200);
  flex-wrap: wrap;
  gap: var(--spacing-4);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  flex: 1;
}

.type-filter :deep(.el-radio-button__inner) {
  padding: var(--spacing-3) var(--spacing-4);
}

.search-box {
  width: 280px;
}

.toolbar-right {
  display: flex;
  gap: var(--spacing-3);
}

/* 工具网格 */
.tools-container {
  min-height: 400px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: var(--spacing-4);
  color: var(--color-gray-500);
}

.tools-grid-enhanced {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: var(--spacing-5);
}
</style>
