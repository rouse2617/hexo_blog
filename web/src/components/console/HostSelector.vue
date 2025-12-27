<template>
  <div class="host-selector">
    <div class="selector-header">
      <div class="header-left">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索主机名称或地址"
          clearable
          style="width: 200px; margin-right: 10px"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select
          v-model="statusFilter"
          placeholder="状态筛选"
          clearable
          style="width: 120px; margin-right: 10px"
        >
          <el-option label="全部" value="" />
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
      </div>
      <div class="header-right">
        <el-button size="small" @click="handleSelectAll">全选</el-button>
        <el-button size="small" @click="handleSelectNone">清空</el-button>
        <el-button size="small" @click="handleSelectInverse">反选</el-button>
        <span class="selected-count">已选择: {{ selectedHosts.length }}</span>
      </div>
    </div>

    <el-table
      ref="tableRef"
      :data="filteredHosts"
      v-loading="loading"
      stripe
      height="calc(100vh - 280px)"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="name" label="名称" min-width="120">
        <template #default="{ row }">
          <div class="host-name-cell">
            <el-icon :color="row.status === 'online' ? '#67c23a' : '#909399'">
              <Monitor />
            </el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="host" label="地址" min-width="150">
        <template #default="{ row }">
          {{ row.host }}:{{ row.port }}
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)" size="small">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="tags" label="标签" min-width="150">
        <template #default="{ row }">
          <el-tag
            v-for="tag in row.tags"
            :key="tag"
            size="small"
            type="info"
            style="margin-right: 5px"
          >
            {{ tag }}
          </el-tag>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { ElTable } from 'element-plus'
import { Monitor, Search } from '@element-plus/icons-vue'
import type { Host } from '@/api/host'
import { useHostStore } from '@/stores/host'
import { useConsoleStore } from '@/stores/console'

const hostStore = useHostStore()
const consoleStore = useConsoleStore()

const tableRef = ref<InstanceType<typeof ElTable>>()
const searchKeyword = ref('')
const statusFilter = ref('')

const loading = computed(() => hostStore.loading)
const allHosts = computed(() => hostStore.allHosts)

// 过滤后的主机列表
const filteredHosts = computed(() => {
  let hosts = allHosts.value

  // 状态筛选
  if (statusFilter.value) {
    hosts = hosts.filter(h => h.status === statusFilter.value)
  }

  // 关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    hosts = hosts.filter(
      h =>
        h.name.toLowerCase().includes(keyword) ||
        h.host.toLowerCase().includes(keyword) ||
        (h.description || '').toLowerCase().includes(keyword)
    )
  }

  return hosts
})

// 选中的主机
const selectedHosts = computed(() => consoleStore.selectedHosts)

// 监听选中变化，同步到store
watch(
  () => consoleStore.selectedHosts,
  (newHosts) => {
    // 更新表格选中状态
    if (tableRef.value) {
      tableRef.value.clearSelection()
      nextTick(() => {
        filteredHosts.value.forEach((host) => {
          if (newHosts.includes(host.id || host.name)) {
            tableRef.value?.toggleRowSelection(host, true)
          }
        })
      })
    }
  },
  { deep: true }
)

// 选择变化处理
const handleSelectionChange = (hosts: Host[]) => {
  const hostIds = hosts.map(h => h.id || h.name)
  consoleStore.selectHosts(hostIds)
}

// 全选
const handleSelectAll = () => {
  tableRef.value?.toggleAllSelection()
}

// 清空选择
const handleSelectNone = () => {
  tableRef.value?.clearSelection()
  consoleStore.clearSelection()
}

// 反选
const handleSelectInverse = () => {
  const currentSelected = new Set(selectedHosts.value)
  const newSelected = filteredHosts.value
    .filter(h => !currentSelected.has(h.id || h.name))
    .map(h => h.id || h.name)
  consoleStore.selectHosts(newSelected)
  
  // 更新表格选中状态
  nextTick(() => {
    tableRef.value?.clearSelection()
    filteredHosts.value.forEach((host) => {
      if (newSelected.includes(host.id || host.name)) {
        tableRef.value?.toggleRowSelection(host, true)
      }
    })
  })
}

const getStatusType = (status?: string) => {
  switch (status) {
    case 'online':
      return 'success'
    case 'offline':
      return 'danger'
    default:
      return 'info'
  }
}

const getStatusText = (status?: string) => {
  switch (status) {
    case 'online':
      return '在线'
    case 'offline':
      return '离线'
    default:
      return '未知'
  }
}

// 加载主机列表
onMounted(async () => {
  await hostStore.loadAllHosts()
})
</script>

<style scoped>
.host-selector {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color);
}

.header-left {
  display: flex;
  align-items: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.selected-count {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.host-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>

