<template>
  <div class="hosts-page">
    <div class="page-header">
      <div class="header-left">
        <el-input
          v-model="keyword"
          placeholder="搜索主机名称或地址"
          :prefix-icon="Search"
          clearable
          style="width: 250px"
          @input="handleSearch"
        />
      </div>
      <div class="header-right">
        <el-button type="primary" :icon="Plus" @click="handleAdd">
          添加主机
        </el-button>
        <el-button :icon="Upload" @click="showImport = true">
          批量导入
        </el-button>
        <el-button
          :icon="Delete"
          :disabled="selectedHosts.length === 0"
          @click="handleBatchDelete"
        >
          批量删除
        </el-button>
      </div>
    </div>

    <HostTable
      :hosts="hostStore.hosts"
      :total="hostStore.total"
      :loading="hostStore.loading"
      @edit="handleEdit"
      @delete="handleDelete"
      @test="handleTest"
      @selection-change="handleSelectionChange"
      @page-change="handlePageChange"
    />

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
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, Upload, Delete } from '@element-plus/icons-vue'
import { useHostStore } from '@/stores/host'
import type { Host } from '@/api/host'
import HostTable from '@/components/host/HostTable.vue'
import HostForm from '@/components/host/HostForm.vue'
import HostImport from '@/components/host/HostImport.vue'

const hostStore = useHostStore()

const keyword = ref('')
const showForm = ref(false)
const showImport = ref(false)
const currentHost = ref<Host | null>(null)
const selectedHosts = ref<Host[]>([])

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

.header-right {
  display: flex;
  gap: 10px;
}
</style>
