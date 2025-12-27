<template>
  <div class="tasks-page">
    <div class="page-header">
      <div class="header-left">
        <el-input
          v-model="keyword"
          placeholder="搜索任务"
          :prefix-icon="Search"
          clearable
          style="width: 200px"
          @input="handleSearch"
        />
        <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 120px" @change="handleSearch">
          <el-option label="全部" value="" />
          <el-option label="运行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-select v-model="typeFilter" placeholder="类型筛选" clearable style="width: 120px" @change="handleSearch">
          <el-option label="全部" value="" />
          <el-option label="对话" value="chat" />
          <el-option label="脚本" value="script" />
          <el-option label="命令" value="command" />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          style="width: 260px"
          @change="handleSearch"
        />
      </div>
      <div class="header-right">
        <el-button :icon="Refresh" @click="loadData">刷新</el-button>
        <el-button
          type="danger"
          :icon="Delete"
          :disabled="selectedTasks.length === 0"
          @click="handleBatchDelete"
        >
          批量删除
        </el-button>
      </div>
    </div>

    <div class="tasks-stats">
      <div class="stat-item">
        <span class="stat-value">{{ stats.total }}</span>
        <span class="stat-label">总任务</span>
      </div>
      <div class="stat-item success">
        <span class="stat-value">{{ stats.success }}</span>
        <span class="stat-label">成功</span>
      </div>
      <div class="stat-item danger">
        <span class="stat-value">{{ stats.failed }}</span>
        <span class="stat-label">失败</span>
      </div>
      <div class="stat-item warning">
        <span class="stat-value">{{ stats.running }}</span>
        <span class="stat-label">运行中</span>
      </div>
    </div>

    <el-table
      :data="tasks"
      v-loading="loading"
      stripe
      style="width: 100%"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" />
      <el-table-column prop="id" label="任务ID" width="180" show-overflow-tooltip />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="getTypeTagType(row.type)" size="small">
            {{ getTypeText(row.type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="input" label="输入" min-width="200" show-overflow-tooltip />
      <el-table-column prop="hostName" label="目标主机" width="120" />
      <el-table-column prop="toolName" label="工具" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusTagType(row.status)" size="small">
            {{ getStatusText(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="startTime" label="开始时间" width="180">
        <template #default="{ row }">
          {{ formatDate(row.startTime) }}
        </template>
      </el-table-column>
      <el-table-column prop="duration" label="耗时" width="100">
        <template #default="{ row }">
          {{ formatDuration(row.duration) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link size="small" @click="handleView(row)">
            详情
          </el-button>
          <el-button
            v-if="row.status === 'running'"
            type="warning"
            link
            size="small"
            @click="handleCancel(row)"
          >
            取消
          </el-button>
          <el-button
            v-if="row.status === 'failed'"
            type="primary"
            link
            size="small"
            @click="handleRetry(row)"
          >
            重试
          </el-button>
          <el-button type="danger" link size="small" @click="handleDelete(row)">
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

    <!-- 任务详情对话框 -->
    <el-dialog
      v-model="showDetail"
      title="任务详情"
      width="700px"
    >
      <el-descriptions v-if="currentTask" :column="2" border>
        <el-descriptions-item label="任务ID">{{ currentTask.id }}</el-descriptions-item>
        <el-descriptions-item label="类型">
          <el-tag :type="getTypeTagType(currentTask.type)" size="small">
            {{ getTypeText(currentTask.type) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="getStatusTagType(currentTask.status)" size="small">
            {{ getStatusText(currentTask.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="目标主机">{{ currentTask.hostName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="工具">{{ currentTask.toolName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ formatDuration(currentTask.duration) }}</el-descriptions-item>
        <el-descriptions-item label="开始时间">{{ formatDate(currentTask.startTime) }}</el-descriptions-item>
        <el-descriptions-item label="结束时间">{{ formatDate(currentTask.endTime) }}</el-descriptions-item>
        <el-descriptions-item label="输入" :span="2">
          <pre class="detail-content">{{ currentTask.input }}</pre>
        </el-descriptions-item>
        <el-descriptions-item label="输出" :span="2">
          <pre class="detail-content output">{{ currentTask.output || '-' }}</pre>
        </el-descriptions-item>
        <el-descriptions-item v-if="currentTask.error" label="错误" :span="2">
          <pre class="detail-content error">{{ currentTask.error }}</pre>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="showDetail = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Delete } from '@element-plus/icons-vue'
import { getTasks, getTask, cancelTask, retryTask, deleteTask, batchDeleteTasks, getTaskStats } from '@/api/task'
import type { Task } from '@/api/task'

const keyword = ref('')
const statusFilter = ref('')
const typeFilter = ref('')
const dateRange = ref<[Date, Date] | null>(null)
const tasks = ref<Task[]>([])
const total = ref(0)
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const selectedTasks = ref<Task[]>([])
const showDetail = ref(false)
const currentTask = ref<Task | null>(null)
const stats = reactive({
  total: 0,
  success: 0,
  failed: 0,
  running: 0
})

onMounted(() => {
  loadData()
  loadStats()
})

const loadData = async () => {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      pageSize: pageSize.value
    }
    if (statusFilter.value) params.status = statusFilter.value
    if (typeFilter.value) params.type = typeFilter.value
    if (dateRange.value) {
      params.startDate = dateRange.value[0].toISOString()
      params.endDate = dateRange.value[1].toISOString()
    }

    const data = await getTasks(params)
    tasks.value = data.list
    total.value = data.total
  } catch (error) {
    console.error('加载任务列表失败:', error)
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    const data = await getTaskStats()
    Object.assign(stats, data)
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadData()
}

const handleSelectionChange = (selection: Task[]) => {
  selectedTasks.value = selection
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  loadData()
}

const handleCurrentChange = (page: number) => {
  currentPage.value = page
  loadData()
}

const handleView = async (task: Task) => {
  try {
    currentTask.value = await getTask(task.id)
    showDetail.value = true
  } catch (error) {
    ElMessage.error('获取任务详情失败')
  }
}

const handleCancel = async (task: Task) => {
  try {
    await ElMessageBox.confirm('确定要取消这个任务吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await cancelTask(task.id)
    ElMessage.success('任务已取消')
    loadData()
    loadStats()
  } catch {
    // 取消操作
  }
}

const handleRetry = async (task: Task) => {
  try {
    await retryTask(task.id)
    ElMessage.success('任务已重新提交')
    loadData()
    loadStats()
  } catch (error) {
    ElMessage.error('重试失败')
  }
}

const handleDelete = async (task: Task) => {
  try {
    await ElMessageBox.confirm('确定要删除这条任务记录吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteTask(task.id)
    ElMessage.success('删除成功')
    loadData()
    loadStats()
  } catch {
    // 取消删除
  }
}

const handleBatchDelete = async () => {
  if (selectedTasks.value.length === 0) return

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTasks.value.length} 条任务记录吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await batchDeleteTasks(selectedTasks.value.map(t => t.id))
    ElMessage.success('批量删除成功')
    selectedTasks.value = []
    loadData()
    loadStats()
  } catch {
    // 取消删除
  }
}

const getTypeTagType = (type: string) => {
  switch (type) {
    case 'chat':
      return 'primary'
    case 'script':
      return 'success'
    case 'command':
      return 'warning'
    default:
      return 'info'
  }
}

const getTypeText = (type: string) => {
  switch (type) {
    case 'chat':
      return '对话'
    case 'script':
      return '脚本'
    case 'command':
      return '命令'
    default:
      return type
  }
}

const getStatusTagType = (status: string) => {
  switch (status) {
    case 'running':
      return 'warning'
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'info'
    default:
      return 'info'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'pending':
      return '等待中'
    case 'running':
      return '运行中'
    case 'success':
      return '成功'
    case 'failed':
      return '失败'
    case 'cancelled':
      return '已取消'
    default:
      return status
  }
}

const formatDate = (date?: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const formatDuration = (duration?: number) => {
  if (!duration) return '-'
  if (duration < 1000) return `${duration}ms`
  if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`
  return `${(duration / 60000).toFixed(1)}min`
}
</script>

<style scoped>
.tasks-page {
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
  flex-wrap: wrap;
  gap: 10px;
}

.header-left {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.header-right {
  display: flex;
  gap: 10px;
}

.tasks-stats {
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

.stat-item.success .stat-value {
  color: #67c23a;
}

.stat-item.danger .stat-value {
  color: #f56c6c;
}

.stat-item.warning .stat-value {
  color: #e6a23c;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 5px;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
}

.detail-content {
  background: #f5f7fa;
  padding: 10px;
  border-radius: 4px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  max-height: 200px;
  overflow: auto;
}

.detail-content.output {
  background: #f0f9eb;
}

.detail-content.error {
  background: #fef0f0;
  color: #f56c6c;
}
</style>
