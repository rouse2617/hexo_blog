<template>
  <div class="hosts-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">主机管理</h2>
        <p class="page-description">管理您的服务器主机，支持批量操作</p>
      </div>
      <div class="header-actions">
        <el-button type="primary" :icon="Plus" @click="handleAdd">
          添加主机
        </el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon online">
          <el-icon><CircleCheck /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ onlineCount }}</div>
          <div class="stat-label">在线主机</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon offline">
          <el-icon><CircleClose /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ offlineCount }}</div>
          <div class="stat-label">离线主机</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon warning">
          <el-icon><Warning /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ unknownCount }}</div>
          <div class="stat-label">状态未知</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon info">
          <el-icon><Monitor /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ hostStore.total }}</div>
          <div class="stat-label">总主机数</div>
        </div>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="keyword"
          placeholder="搜索主机名称、地址或标签..."
          :prefix-icon="Search"
          clearable
          class="search-input"
          @input="handleSearch"
        />

        <!-- 视图切换 -->
        <el-radio-group v-model="viewMode" class="view-toggle">
          <el-tooltip content="列表视图" placement="top">
            <el-radio-button value="table">
              <el-icon><List /></el-icon>
            </el-radio-button>
          </el-tooltip>
          <el-tooltip content="卡片视图" placement="top">
            <el-radio-button value="card">
              <el-icon><Grid /></el-icon>
            </el-radio-button>
          </el-tooltip>
        </el-radio-group>

        <!-- 筛选器 -->
        <el-select v-model="statusFilter" placeholder="状态" clearable>
          <el-option label="全部" value="" />
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="未知" value="unknown" />
        </el-select>
      </div>

      <div class="toolbar-right">
        <el-text v-if="selectedHosts.length > 0" type="info">
          已选择 {{ selectedHosts.length }} 台主机
        </el-text>
        <el-button
          v-if="selectedHosts.length > 0"
          :icon="Upload"
          @click="showImport = true"
        >
          批量导入
        </el-button>
        <el-button
          v-if="selectedHosts.length > 0"
          :icon="Delete"
          type="danger"
          @click="handleBatchDelete"
        >
          批量删除
        </el-button>
      </div>
    </div>

    <!-- 内容区域 - 动态切换视图 -->
    <HostTable
      v-if="viewMode === 'table'"
      :hosts="filteredHosts"
      :total="filteredTotal"
      :loading="hostStore.loading"
      @edit="handleEdit"
      @delete="handleDelete"
      @test="handleTest"
      @selection-change="handleSelectionChange"
      @page-change="handlePageChange"
    />

    <HostGrid
      v-else
      :hosts="filteredHosts"
      :total="filteredTotal"
      :loading="hostStore.loading"
      @edit="handleEdit"
      @delete="handleDelete"
      @test="handleTest"
      @selection-change="handleSelectionChange"
      @page-change="handlePageChange"
    />

    <!-- 弹窗 -->
    <HostForm
      v-model:visible="showForm"
      :host="currentHost"
      @submit="handleFormSubmit"
    />

    <HostImport
      v-model:visible="showImport"
      @submit="handleImportSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search, Plus, Upload, Delete,
  CircleCheck, CircleClose, Warning, Monitor,
  List, Grid
} from '@element-plus/icons-vue'
import { useHostStore } from '@/stores/host'
import type { Host } from '@/api/host'
import HostTable from '@/components/host/HostTable.vue'
import HostGrid from '@/components/host/HostGrid.vue'
import HostForm from '@/components/host/HostForm.vue'
import HostImport from '@/components/host/HostImport.vue'

const hostStore = useHostStore()

const keyword = ref('')
const statusFilter = ref('')
const viewMode = ref<'table' | 'card'>('table')
const showForm = ref(false)
const showImport = ref(false)
const currentHost = ref<Host | null>(null)
const selectedHosts = ref<Host[]>([])

// 统计数据
const onlineCount = computed(() =>
  hostStore.hosts.filter(h => h.status === 'online').length
)
const offlineCount = computed(() =>
  hostStore.hosts.filter(h => h.status === 'offline').length
)
const unknownCount = computed(() =>
  hostStore.hosts.filter(h => !h.status || h.status === 'unknown').length
)

// 过滤后的主机
const filteredHosts = computed(() => {
  let hosts = hostStore.hosts

  if (statusFilter.value) {
    hosts = hosts.filter(h => h.status === statusFilter.value)
  }

  if (keyword.value) {
    const kw = keyword.value.toLowerCase()
    hosts = hosts.filter(h =>
      h.name.toLowerCase().includes(kw) ||
      h.host.toLowerCase().includes(kw) ||
      (h.tags || []).some(tag => tag.toLowerCase().includes(kw))
    )
  }

  return hosts
})

const filteredTotal = computed(() => filteredHosts.value.length)

onMounted(() => {
  loadData()
})

const loadData = (page = 1, pageSize = 10) => {
  hostStore.loadHosts({ page, pageSize, keyword: keyword.value })
}

const handleSearch = () => {
  loadData()
}

const handleAdd = () => {
  currentHost.value = null
  showForm.value = true
}

const handleEdit = (host: Host) => {
  currentHost.value = host
  showForm.value = true
}

const handleDelete = async (host: Host) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除主机 "${host.name}" 吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await hostStore.removeHost(host.id)
    ElMessage.success('删除成功')
  } catch {
    // 取消删除
  }
}

const handleTest = async (host: Host) => {
  try {
    ElMessage.info('正在测试连接...')
    const result = await hostStore.testConnection(host.id)
    if (result.success) {
      ElMessage.success('连接成功')
    } else {
      ElMessage.error(result.message || '连接失败')
    }
  } catch (error) {
    ElMessage.error('测试连接失败')
  }
}

const handleSelectionChange = (hosts: Host[]) => {
  selectedHosts.value = hosts
}

const handleBatchDelete = async () => {
  if (selectedHosts.value.length === 0) return

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedHosts.value.length} 台主机吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    for (const host of selectedHosts.value) {
      await hostStore.removeHost(host.id)
    }
    ElMessage.success('批量删除成功')
    selectedHosts.value = []
  } catch {
    // 取消删除
  }
}

const handlePageChange = (page: number, pageSize: number) => {
  loadData(page, pageSize)
}

const handleFormSubmit = async (data: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>) => {
  try {
    if (currentHost.value) {
      await hostStore.editHost(currentHost.value.id, data)
      ElMessage.success('更新成功')
    } else {
      await hostStore.addHost(data)
      ElMessage.success('添加成功')
    }
  } catch (error) {
    // 错误已在 store 中处理
  }
}

const handleImportSubmit = async (hosts: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>[]) => {
  try {
    const result = await hostStore.batchImport(hosts)
    ElMessage.success(`成功导入 ${result.success} 台主机${result.failed > 0 ? `，失败 ${result.failed} 台` : ''}`)
  } catch (error) {
    // 错误已在 store 中处理
  }
}
</script>

<style scoped>
.hosts-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-5);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
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

.header-actions {
  display: flex;
  gap: var(--spacing-3);
}

/* 统计卡片 */
.stats-row {
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
  transition: all var(--transition-fast);
}

.stat-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
}

.stat-icon.online {
  background: var(--badge-online-bg);
  color: var(--badge-online-text);
}

.stat-icon.offline {
  background: var(--badge-offline-bg);
  color: var(--badge-offline-text);
}

.stat-icon.warning {
  background: var(--color-warning-light);
  color: var(--color-warning);
}

.stat-icon.info {
  background: var(--color-info-light);
  color: var(--color-info);
}

.stat-content {
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
  min-width: 0;
}

.search-input {
  width: 240px;
  max-width: 100%;
}

.view-toggle :deep(.el-radio-button__inner) {
  padding: 8px 12px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}
</style>
