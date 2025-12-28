<template>
  <div class="host-table">
    <el-table
      :data="hosts"
      v-loading="loading"
      stripe
      style="width: 100%"
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
      <el-table-column prop="username" label="用户名" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <div class="status-badge" :class="row.status || 'unknown'">
            <el-icon class="status-icon">
              <component :is="getStatusIcon(row.status)" />
            </el-icon>
            <span class="status-text">{{ getStatusText(row.status) }}</span>
          </div>
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
      <el-table-column prop="description" label="描述" min-width="150" show-overflow-tooltip />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click="$emit('test', row)">
            测试
          </el-button>
          <el-button type="primary" link size="small" @click="$emit('edit', row)">
            编辑
          </el-button>
          <el-button type="danger" link size="small" @click="$emit('delete', row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="table-footer">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Host } from '@/api/host'
import { Monitor, CircleCheck, CircleClose, WarningFilled } from '@element-plus/icons-vue'

const props = defineProps<{
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

const currentPage = ref(1)
const pageSize = ref(10)

const getStatusIcon = (status?: string) => {
  switch (status) {
    case 'online':
      return CircleCheck
    case 'offline':
      return CircleClose
    default:
      return WarningFilled
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

const handleSelectionChange = (selection: Host[]) => {
  emit('selection-change', selection)
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  emit('page-change', currentPage.value, size)
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
  emit('page-change', page, pageSize.value)
}

watch(
  () => props.total,
  () => {
    const maxPage = Math.ceil(props.total / pageSize.value)
    if (currentPage.value > maxPage && maxPage > 0) {
      currentPage.value = 1
    }
  }
)
</script>

<style scoped>
.host-table {
  background: #fff;
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--color-gray-200);
  overflow: hidden;
}

.host-name-cell {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

/* 状态徽章样式 */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.status-badge.online {
  background: var(--badge-online-bg);
  color: var(--badge-online-text);
  border: 1px solid var(--badge-online-border);
}

.status-badge.offline {
  background: var(--badge-offline-bg);
  color: var(--badge-offline-text);
  border: 1px solid var(--badge-offline-border);
}

.status-badge.unknown {
  background: var(--badge-unknown-bg);
  color: var(--badge-unknown-text);
  border: 1px solid var(--badge-unknown-border);
}

.status-icon {
  font-size: 14px;
}

.status-text {
  line-height: 1;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  padding: var(--spacing-4);
  border-top: 1px solid var(--color-gray-200);
}
</style>
