<template>
  <div class="host-grid">
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-skeleton">
      <div v-for="i in 6" :key="i" class="skeleton-card">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="rect" style="width: 100%; height: 160px" />
          </template>
        </el-skeleton>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="hosts.length === 0" class="empty-state">
      <el-empty description="暂无主机数据">
        <el-button type="primary">添加主机</el-button>
      </el-empty>
    </div>

    <!-- 卡片网格 -->
    <div v-else class="grid-container">
      <div
        v-for="host in hosts"
        :key="host.id"
        class="host-card"
        :class="{
          selected: selectedHosts.includes(host),
          offline: host.status === 'offline'
        }"
      >
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="host-info">
            <div class="status-indicator" :class="host.status || 'unknown'"></div>
            <h3 class="host-name">{{ host.name }}</h3>
          </div>
          <el-checkbox
            :model-value="selectedHosts.includes(host)"
            @change="handleSelect(host, $event)"
          />
        </div>

        <!-- 卡片内容 -->
        <div class="card-body">
          <div class="host-detail">
            <el-icon><Connection /></el-icon>
            <span>{{ host.host }}:{{ host.port }}</span>
          </div>
          <div class="host-detail">
            <el-icon><User /></el-icon>
            <span>{{ host.username }}</span>
          </div>

          <!-- 标签 -->
          <div v-if="host.tags?.length" class="host-tags">
            <el-tag
              v-for="tag in host.tags.slice(0, 3)"
              :key="tag"
              size="small"
              type="info"
            >
              {{ tag }}
            </el-tag>
            <el-tag v-if="host.tags.length > 3" size="small">
              +{{ host.tags.length - 3 }}
            </el-tag>
          </div>
        </div>

        <!-- 卡片底部操作 -->
        <div class="card-footer">
          <el-button
            size="small"
            :icon="Refresh"
            :loading="testingHost === host.id"
            @click="handleTest(host)"
          >
            测试
          </el-button>
          <el-button
            size="small"
            :icon="Edit"
            @click="handleEdit(host)"
          >
            编辑
          </el-button>
          <el-button
            size="small"
            :icon="Delete"
            type="danger"
            link
            @click="handleDelete(host)"
          >
            删除
          </el-button>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="grid-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[12, 24, 48, 96]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Host } from '@/api/host'
import { Connection, User, Refresh, Edit, Delete } from '@element-plus/icons-vue'

defineProps<{
  hosts: Host[]
  total: number
  loading: boolean
}>()

const emit = defineEmits<{
  edit: [host: Host]
  delete: [host: Host]
  test: [host: Host]
  'selection-change': [hosts: Host[]]
  'page-change': [page: number, pageSize: number]
}>()

const selectedHosts = ref<Host[]>([])
const testingHost = ref<string | null>(null)
const currentPage = ref(1)
const pageSize = ref(12)

const handleSelect = (host: Host, checked: boolean) => {
  if (checked) {
    selectedHosts.value.push(host)
  } else {
    selectedHosts.value = selectedHosts.value.filter(h => h.id !== host.id)
  }
  emit('selection-change', selectedHosts.value)
}

const handleTest = (host: Host) => {
  testingHost.value = host.id
  emit('test', host)
  setTimeout(() => {
    testingHost.value = null
  }, 2000)
}

const handleEdit = (host: Host) => {
  emit('edit', host)
}

const handleDelete = (host: Host) => {
  emit('delete', host)
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  emit('page-change', currentPage.value, size)
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
  emit('page-change', page, pageSize.value)
}
</script>

<style scoped>
.host-grid {
  min-height: 400px;
}

/* 骨架屏 */
.loading-skeleton {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--spacing-4);
}

.skeleton-card {
  background: #fff;
  border-radius: var(--radius-xl);
  padding: var(--spacing-4);
  border: 1px solid var(--color-gray-200);
}

/* 空状态 */
.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
}

/* 网格容器 */
.grid-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--spacing-4);
}

/* 主机卡片 */
.host-card {
  background: #fff;
  border-radius: var(--radius-xl);
  border: 2px solid var(--color-gray-200);
  overflow: hidden;
  transition: all var(--transition-base);
  cursor: pointer;
}

.host-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.host-card.selected {
  border-color: var(--color-primary);
  background: var(--color-primary-lighter);
}

.host-card.offline {
  opacity: 0.7;
}

/* 卡片头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-4);
  background: var(--color-gray-50);
  border-bottom: 1px solid var(--color-gray-200);
}

.host-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  position: relative;
}

.status-indicator.online {
  background: #22c55e;
  box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.2);
}

.status-indicator.offline {
  background: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
}

.status-indicator.unknown {
  background: #9ca3af;
}

.host-name {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-gray-800);
  margin: 0;
}

/* 卡片内容 */
.card-body {
  padding: var(--spacing-4);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

.host-detail {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--text-sm);
  color: var(--color-gray-600);
}

.host-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-2);
}

/* 卡片底部 */
.card-footer {
  padding: var(--spacing-4);
  border-top: 1px solid var(--color-gray-200);
  display: flex;
  gap: var(--spacing-2);
}

/* 分页 */
.grid-pagination {
  display: flex;
  justify-content: center;
  padding: var(--spacing-6) 0;
}
</style>
