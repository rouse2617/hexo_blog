<template>
  <div class="script-table">
    <el-table
      :data="scripts"
      v-loading="loading"
      stripe
      style="width: 100%"
    >
      <el-table-column prop="name" label="名称" min-width="150">
        <template #default="{ row }">
          <div class="script-name-cell">
            <el-icon :color="row.enabled ? '#67c23a' : '#909399'">
              <Document />
            </el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="language" label="语言" width="120">
        <template #default="{ row }">
          <el-tag :type="getLanguageType(row.language)" size="small">
            {{ row.language }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column prop="enabled" label="状态" width="100">
        <template #default="{ row }">
          <el-switch
            :model-value="row.enabled"
            @change="(val: boolean) => $emit('toggle', row.id, val)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="updatedAt" label="更新时间" width="180">
        <template #default="{ row }">
          {{ formatDate(row.updatedAt) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click="$emit('view', row)">
            查看
          </el-button>
          <el-button type="primary" link size="small" @click="$emit('edit', row)">
            编辑
          </el-button>
          <el-button type="warning" link size="small" @click="$emit('test', row)">
            测试
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
import { ref } from 'vue'
import type { Script } from '@/api/script'
import { Document } from '@element-plus/icons-vue'

defineProps<{
  scripts: Script[]
  total: number
  loading: boolean
}>()

const emit = defineEmits<{
  view: [script: Script]
  edit: [script: Script]
  delete: [script: Script]
  test: [script: Script]
  toggle: [id: string, enabled: boolean]
  'page-change': [page: number, pageSize: number]
}>()

const currentPage = ref(1)
const pageSize = ref(10)

const getLanguageType = (language: string) => {
  switch (language) {
    case 'bash':
      return 'success'
    case 'python':
      return 'primary'
    case 'powershell':
      return 'warning'
    default:
      return 'info'
  }
}

const formatDate = (date?: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
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
.script-table {
  background: #fff;
  border-radius: 8px;
}

.script-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}
</style>
