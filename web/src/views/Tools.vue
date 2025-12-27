<template>
  <div class="tools-page">
    <div class="page-header">
      <div class="header-left">
        <el-radio-group v-model="activeTab" @change="handleTabChange">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="builtin">内置工具</el-radio-button>
          <el-radio-button value="script">脚本工具</el-radio-button>
        </el-radio-group>
      </div>
      <div class="header-right">
        <el-input
          v-model="keyword"
          placeholder="搜索工具名称"
          :prefix-icon="Search"
          clearable
          style="width: 200px"
        />
      </div>
    </div>

    <div class="tools-stats">
      <div class="stat-item">
        <span class="stat-value">{{ toolStore.tools.length }}</span>
        <span class="stat-label">全部工具</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ toolStore.builtinTools.length }}</span>
        <span class="stat-label">内置工具</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ toolStore.scriptTools.length }}</span>
        <span class="stat-label">脚本工具</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ toolStore.enabledTools.length }}</span>
        <span class="stat-label">已启用</span>
      </div>
    </div>

    <ToolList
      :tools="filteredTools"
      :loading="toolStore.loading"
      @toggle="handleToggle"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { useToolStore } from '@/stores/tool'
import ToolList from '@/components/tool/ToolList.vue'

const toolStore = useToolStore()

const activeTab = ref('all')
const keyword = ref('')

onMounted(() => {
  toolStore.loadTools()
})

const filteredTools = computed(() => {
  let tools = toolStore.tools

  // 按类型筛选
  if (activeTab.value === 'builtin') {
    tools = toolStore.builtinTools
  } else if (activeTab.value === 'script') {
    tools = toolStore.scriptTools
  }

  // 按关键词筛选
  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    tools = tools.filter(
      t => t.name.toLowerCase().includes(kw) || t.description.toLowerCase().includes(kw)
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
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.tools-stats {
  display: flex;
  gap: 30px;
  padding: 20px;
  background: #f8fafc;
  border-radius: 10px;
  margin-bottom: 20px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #409eff;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 5px;
}
</style>
