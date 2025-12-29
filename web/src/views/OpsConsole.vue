<template>
  <div class="ops-console">
    <!-- 顶部 Command Bar - 紧凑单行 -->
    <div class="command-bar">
      <div class="command-left">
        <!-- 快速命令输入 -->
        <div class="quick-command">
          <el-icon><Monitor /></el-icon>
          <input
            ref="commandInput"
            v-model="quickCommand"
            type="text"
            placeholder="输入命令或搜索... (Ctrl+K)"
            @keydown.enter="handleQuickCommand"
            @keydown.ctrl.k="handleCommandFocus"
            class="command-input"
          />
          <el-tag v-if="selectedHosts.length > 0" size="small" type="info">
            {{ selectedHosts.length }} hosts
          </el-tag>
        </div>
      </div>

      <div class="command-right">
        <!-- 快速操作按钮组 -->
        <el-button-group>
          <el-tooltip content="刷新 (F5)">
            <el-button size="small" :icon="Refresh" @click="handleRefresh" />
          </el-tooltip>
          <el-tooltip content="批量命令 (Ctrl+B)">
            <el-button size="small" :icon="Lightning" @click="showBatchDialog = true" />
          </el-tooltip>
          <el-tooltip content="存储监控">
            <el-button size="small" :icon="FolderOpened" @click="showStoragePanel = true" />
          </el-tooltip>
          <el-tooltip content="设置">
            <el-button size="small" :icon="Setting" @click="showSettings = true" />
          </el-tooltip>
        </el-button-group>

        <!-- 状态指示 -->
        <div class="status-indicators">
          <div class="indicator" :class="{ active: isOnline }">
            <span class="dot"></span>
            <span class="count">{{ onlineCount }}</span>
          </div>
          <div class="indicator offline">
            <span class="dot"></span>
            <span class="count">{{ offlineCount }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 主体内容 -->
    <div class="console-body">
      <!-- 左侧主机树 - 可收缩 -->
      <div class="host-sidebar" :class="{ collapsed: isSidebarCollapsed }">
        <div class="sidebar-header">
          <transition name="fade">
            <span v-if="!isSidebarCollapsed" class="sidebar-title">主机</span>
          </transition>
          <el-button
            text
            size="small"
            @click="isSidebarCollapsed = !isSidebarCollapsed"
            class="collapse-btn"
          >
            <el-icon :size="16">
              <DArrowLeft v-if="!isSidebarCollapsed" />
              <DArrowRight v-else />
            </el-icon>
          </el-button>
        </div>

        <transition name="slide">
          <div v-if="!isSidebarCollapsed" class="sidebar-content">
            <!-- 分组主机树 -->
            <el-tree
              ref="hostTreeRef"
              :data="hostTreeData"
              :props="treeProps"
              show-checkbox
              node-key="id"
              :default-checked-keys="selectedHosts"
              @check-change="handleHostCheck"
              class="host-tree"
            >
              <template #default="{ node, data }">
                <div class="tree-node">
                  <el-icon v-if="data.type === 'group'"><Folder /></el-icon>
                  <el-icon v-else>
                    <Monitor :color="data.status === 'online' ? '#22c55e' : '#6b7280'" />
                  </el-icon>
                  <span class="node-label">{{ node.label }}</span>
                  <span v-if="data.status" class="node-status" :class="data.status">
                    {{ data.status === 'online' ? '在线' : '离线' }}
                  </span>
                </div>
              </template>
            </el-tree>
          </div>
        </transition>
      </div>

      <!-- 中间表格区域 -->
      <div class="main-content">
        <!-- 工具栏 -->
        <div class="table-toolbar">
          <div class="toolbar-left">
            <el-radio-group v-model="viewMode" size="small">
              <el-radio-button value="table">表格</el-radio-button>
              <el-radio-button value="grid">网格</el-radio-button>
              <el-radio-button value="timeline">时间线</el-radio-button>
            </el-radio-group>
            <el-divider direction="vertical" />
            <el-select v-model="timeFilter" size="small" placeholder="时间范围" style="width: 120px">
              <el-option label="全部" value="all" />
              <el-option label="最近5分钟" value="5m" />
              <el-option label="最近1小时" value="1h" />
              <el-option label="今天" value="today" />
            </el-select>
          </div>
          <div class="toolbar-right">
            <el-input
              v-model="searchKeyword"
              size="small"
              placeholder="搜索..."
              prefix-icon="Search"
              style="width: 200px"
              clearable
            />
          </div>
        </div>

        <!-- 紧凑型表格 -->
        <div class="table-container">
          <el-table
            ref="tableRef"
            :data="filteredTableData"
            :height="tableHeight"
            stripe
            size="small"
            :row-height="40"
            @row-click="handleRowClick"
            class="compact-table"
          >
            <el-table-column type="index" width="50" align="center" />
            <el-table-column prop="host" label="主机" width="150" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="host-cell">
                  <span class="host-dot" :class="row.hostStatus"></span>
                  {{ row.host }}
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="command" label="命令" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)" size="small">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="elapsed" label="耗时" width="100" align="right">
              <template #default="{ row }">
                <span class="elapsed-text">{{ formatElapsed(row.elapsed) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="timestamp" label="时间" width="160" />
            <el-table-column label="操作" width="100" align="center" fixed="right">
              <template #default="{ row }">
                <el-button text size="small" @click.stop="handleViewDetail(row)">
                  <el-icon><View /></el-icon>
                </el-button>
                <el-button text size="small" @click.stop="handleCopy(row)">
                  <el-icon><CopyDocument /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 分页 -->
        <div class="table-footer">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[20, 50, 100, 200]"
            :total="totalItems"
            layout="total, sizes, prev, pager, next, jumper"
            size="small"
            background
          />
        </div>
      </div>

      <!-- 右侧详情侧边栏 - 抽屉式 -->
      <el-drawer
        v-model="detailDrawerVisible"
        direction="rtl"
        :size="500"
        :show-close="true"
        class="detail-drawer"
      >
        <template #header>
          <div class="drawer-header">
            <div class="header-info">
              <el-icon :size="20"><Monitor /></el-icon>
              <span class="host-name">{{ selectedRow?.host }}</span>
              <el-tag :type="getStatusType(selectedRow?.status)" size="small">
                {{ getStatusText(selectedRow?.status) }}
              </el-tag>
            </div>
          </div>
        </template>

        <div class="drawer-content">
          <!-- 命令信息 -->
          <div class="info-section">
            <h4 class="section-title">
              <el-icon><Monitor /></el-icon>
              执行命令
            </h4>
            <div class="command-display">
              <pre>{{ selectedRow?.command }}</pre>
            </div>
          </div>

          <!-- 执行结果 -->
          <div class="info-section">
            <h4 class="section-title">
              <el-icon><Document /></el-icon>
              执行结果
            </h4>
            <div class="result-display" :class="{ error: selectedRow?.status === 'error' }">
              <pre>{{ selectedRow?.result || '无输出' }}</pre>
            </div>
          </div>

          <!-- 统计信息 -->
          <div class="info-section">
            <h4 class="section-title">
              <el-icon><DataAnalysis /></el-icon>
              统计信息
            </h4>
            <div class="stats-grid">
              <div class="stat-item">
                <span class="stat-label">执行时间</span>
                <span class="stat-value">{{ selectedRow?.timestamp }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">耗时</span>
                <span class="stat-value">{{ formatElapsed(selectedRow?.elapsed) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">退出码</span>
                <span class="stat-value">{{ selectedRow?.exitCode || 'N/A' }}</span>
              </div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="action-section">
            <el-button type="primary" @click="handleReRun" :icon="Refresh">
              重新执行
            </el-button>
            <el-button @click="handleCopyResult" :icon="CopyDocument">
              复制结果
            </el-button>
            <el-button @click="handleDownload" :icon="Download">
              下载日志
            </el-button>
          </div>
        </div>
      </el-drawer>
    </div>

    <!-- 批量命令对话框 -->
    <el-dialog v-model="showBatchDialog" title="批量执行命令" width="600px" :dark="true">
      <el-form :model="batchForm" label-width="80px">
        <el-form-item label="命令">
          <el-input
            v-model="batchForm.command"
            type="textarea"
            :rows="4"
            placeholder="输入要批量执行的命令"
          />
        </el-form-item>
        <el-form-item label="超时时间">
          <el-slider v-model="batchForm.timeout" :min="5" :max="300" :marks="{ 5: '5s', 60: '1m', 300: '5m' }" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBatchDialog = false">取消</el-button>
        <el-button type="primary" @click="handleBatchExecute" :loading="batchExecuting">
          执行 ({{ selectedHosts.length }} 台主机)
        </el-button>
      </template>
    </el-dialog>

    <!-- 存储监控面板对话框 -->
    <el-dialog v-model="showStoragePanel" title="存储监控" width="900px" :dark="true">
      <StorageMonitorEmbedded />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Refresh, Lightning, FolderOpened, Setting,
  DArrowLeft, DArrowRight, Folder, Monitor,
  View, CopyDocument, Document, DataAnalysis, Download
} from '@element-plus/icons-vue'

// Import Terminal icon separately - not in Element Plus, using Monitor as replacement
import { ElMessage } from 'element-plus'

// ===== 数据结构 =====
interface HostNode {
  id: string
  label: string
  type: 'group' | 'host'
  status?: 'online' | 'offline'
  children?: HostNode[]
}

interface TableData {
  id: string
  host: string
  hostStatus: 'online' | 'offline'
  command: string
  status: 'success' | 'error' | 'running'
  elapsed: number
  timestamp: string
  result?: string
  exitCode?: number
}

// ===== 状态管理 =====
const quickCommand = ref('')
const isSidebarCollapsed = ref(false)
const viewMode = ref('table')
const timeFilter = ref('all')
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(50)
const detailDrawerVisible = ref(false)
const selectedRow = ref<TableData | null>(null)
const selectedHosts = ref<string[]>([])
const showBatchDialog = ref(false)
const showStoragePanel = ref(false)
const showSettings = ref(false)
const batchExecuting = ref(false)

const batchForm = ref({
  command: '',
  timeout: 30
})

// 模拟数据
const hostTreeData = ref<HostNode[]>([
  {
    id: 'prod',
    label: '生产环境',
    type: 'group',
    children: [
      { id: 'prod-web-01', label: 'prod-web-01', type: 'host', status: 'online' },
      { id: 'prod-web-02', label: 'prod-web-02', type: 'host', status: 'online' },
      { id: 'prod-db-01', label: 'prod-db-01', type: 'host', status: 'online' },
      { id: 'prod-db-02', label: 'prod-db-02', type: 'host', status: 'offline' }
    ]
  },
  {
    id: 'staging',
    label: '测试环境',
    type: 'group',
    children: [
      { id: 'staging-web-01', label: 'staging-web-01', type: 'host', status: 'online' },
      { id: 'staging-db-01', label: 'staging-db-01', type: 'host', status: 'online' }
    ]
  }
])

const tableData = ref<TableData[]>([
  {
    id: '1',
    host: 'prod-web-01',
    hostStatus: 'online',
    command: 'tail -100 /var/log/nginx/access.log',
    status: 'success',
    elapsed: 234,
    timestamp: '2025-12-30 14:32:15',
    result: '192.168.1.100 - - [30/Dec/2025:14:32:10] "GET /api/users HTTP/1.1" 200',
    exitCode: 0
  },
  {
    id: '2',
    host: 'prod-web-02',
    hostStatus: 'online',
    command: 'systemctl status nginx',
    status: 'success',
    elapsed: 567,
    timestamp: '2025-12-30 14:32:20',
    result: 'nginx.service - A high performance web server\n   Loaded: loaded (/usr/lib/systemd/system/nginx.service; enabled)',
    exitCode: 0
  },
  {
    id: '3',
    host: 'prod-db-02',
    hostStatus: 'offline',
    command: 'mysql -e "SHOW PROCESSLIST"',
    status: 'error',
    elapsed: 5000,
    timestamp: '2025-12-30 14:32:25',
    result: 'ERROR 2003 (HY000): Can\'t connect to MySQL server on \'localhost\' (111)',
    exitCode: 1
  }
])

const treeProps = {
  children: 'children',
  label: 'label'
}

// ===== 计算属性 =====
const onlineCount = computed(() => {
  let count = 0
  const countHosts = (nodes: HostNode[]) => {
    nodes.forEach(node => {
      if (node.type === 'host' && node.status === 'online') count++
      if (node.children) countHosts(node.children)
    })
  }
  countHosts(hostTreeData.value)
  return count
})

const offlineCount = computed(() => {
  let count = 0
  const countHosts = (nodes: HostNode[]) => {
    nodes.forEach(node => {
      if (node.type === 'host' && node.status === 'offline') count++
      if (node.children) countHosts(node.children)
    })
  }
  countHosts(hostTreeData.value)
  return count
})

const isOnline = computed(() => onlineCount.value > 0)

const tableHeight = computed(() => {
  // 动态计算表格高度 = 视口高度 - header - toolbar - footer
  return `calc(100vh - 140px)`
})

const filteredTableData = computed(() => {
  let data = tableData.value

  // 时间过滤
  if (timeFilter.value !== 'all') {
    const now = Date.now()
    const timeMap: Record<string, number> = {
      '5m': 5 * 60 * 1000,
      '1h': 60 * 60 * 1000,
      'today': 24 * 60 * 60 * 1000
    }
    const cutoff = now - (timeMap[timeFilter.value] || 0)
    data = data.filter(row => {
      const rowTime = new Date(row.timestamp).getTime()
      return rowTime >= cutoff
    })
  }

  // 关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    data = data.filter(row =>
      row.host.toLowerCase().includes(keyword) ||
      row.command.toLowerCase().includes(keyword) ||
      (row.result && row.result.toLowerCase().includes(keyword))
    )
  }

  return data
})

const totalItems = computed(() => filteredTableData.value.length)

// ===== 方法 =====
const handleCommandFocus = () => {
  nextTick(() => {
    // 聚焦命令输入框
  })
}

const handleQuickCommand = () => {
  if (!quickCommand.value.trim()) return
  ElMessage.info(`执行快速命令: ${quickCommand.value}`)
  quickCommand.value = ''
}

const handleRefresh = () => {
  ElMessage.success('已刷新')
}

const handleHostCheck = () => {
  // 更新选中的主机
  ElMessage.info(`已选择 ${selectedHosts.value.length} 台主机`)
}

const handleRowClick = (row: TableData) => {
  selectedRow.value = row
  detailDrawerVisible.value = true
}

const handleViewDetail = (row: TableData) => {
  selectedRow.value = row
  detailDrawerVisible.value = true
}

const handleCopy = (row: TableData) => {
  navigator.clipboard.writeText(row.command)
  ElMessage.success('命令已复制')
}

const getStatusType = (status: string | undefined) => {
  const map: Record<string, any> = {
    'success': 'success',
    'error': 'danger',
    'running': 'warning'
  }
  return map[status || ''] || 'info'
}

const getStatusText = (status: string | undefined) => {
  const map: Record<string, string> = {
    'success': '成功',
    'error': '失败',
    'running': '执行中'
  }
  return map[status || ''] || '未知'
}

const formatElapsed = (ms: number | undefined) => {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

const handleReRun = () => {
  ElMessage.info('重新执行命令')
}

const handleCopyResult = () => {
  if (selectedRow.value?.result) {
    navigator.clipboard.writeText(selectedRow.value.result)
    ElMessage.success('结果已复制')
  }
}

const handleDownload = () => {
  ElMessage.success('下载日志')
}

const handleBatchExecute = () => {
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请先选择主机')
    return
  }
  batchExecuting.value = true
  setTimeout(() => {
    batchExecuting.value = false
    showBatchDialog.value = false
    ElMessage.success(`已在 ${selectedHosts.value.length} 台主机上执行命令`)
  }, 2000)
}

// ===== 键盘快捷键 =====
const handleKeydown = (e: KeyboardEvent) => {
  // Ctrl+K: 聚焦命令输入
  if (e.ctrlKey && e.key === 'k') {
    e.preventDefault()
    handleCommandFocus()
  }
  // F5: 刷新
  if (e.key === 'F5') {
    e.preventDefault()
    handleRefresh()
  }
  // Ctrl+B: 批量命令
  if (e.ctrlKey && e.key === 'b') {
    e.preventDefault()
    showBatchDialog.value = true
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
/* ===== GitHub Codespaces 深色主题 ===== */
.ops-console {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #0d1117;
  color: #c9d1d9;
}

/* ===== Command Bar ===== */
.command-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 48px;
  padding: 0 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  flex-shrink: 0;
}

.command-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.quick-command {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  max-width: 600px;
  padding: 6px 12px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  transition: border-color 0.2s;
}

.quick-command:focus-within {
  border-color: #58a6ff;
}

.quick-command .el-icon {
  color: #8b949e;
}

.command-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: #c9d1d9;
  font-size: 14px;
}

.command-input::placeholder {
  color: #8b949e;
}

.command-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-indicators {
  display: flex;
  gap: 12px;
}

.indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #8b949e;
}

.indicator.active {
  color: #22c55e;
}

.indicator.offline {
  color: #f85149;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.indicator.active .dot {
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.count {
  font-weight: 500;
}

/* ===== Console Body ===== */
.console-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* ===== Host Sidebar ===== */
.host-sidebar {
  display: flex;
  flex-direction: column;
  width: 240px;
  background: #161b22;
  border-right: 1px solid #30363d;
  transition: width 0.3s ease;
  flex-shrink: 0;
}

.host-sidebar.collapsed {
  width: 48px;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 48px;
  padding: 0 12px;
  border-bottom: 1px solid #21262d;
}

.sidebar-title {
  font-size: 14px;
  font-weight: 600;
  color: #f0f6fc;
}

.collapse-btn {
  color: #8b949e;
}

.collapse-btn:hover {
  color: #c9d1d9;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.host-tree {
  background: transparent;
}

.host-tree :deep(.el-tree-node__content) {
  height: 36px;
  color: #c9d1d9;
}

.host-tree :deep(.el-tree-node__content:hover) {
  background: #21262d;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-label {
  flex: 1;
}

.node-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
}

.node-status.online {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
}

.node-status.offline {
  color: #f85149;
  background: rgba(248, 81, 73, 0.1);
}

/* ===== Main Content ===== */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 48px;
  padding: 0 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.table-container {
  flex: 1;
  overflow: hidden;
}

/* ===== Compact Table ===== */
.compact-table {
  background: transparent;
}

.compact-table :deep(.el-table__header) {
  background: #161b22;
}

.compact-table :deep(.el-table__header th) {
  background: #161b22;
  border-color: #30363d;
  color: #8b949e;
  font-weight: 600;
  padding: 8px 0;
}

.compact-table :deep(.el-table__body) {
  background: transparent;
}

.compact-table :deep(.el-table__body tr) {
  background: transparent;
}

.compact-table :deep(.el-table__body tr:hover td) {
  background: #21262d;
}

.compact-table :deep(.el-table__body td) {
  border-color: #21262d;
  color: #c9d1d9;
  padding: 8px 0;
}

.compact-table :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background: rgba(48, 54, 61, 0.3);
}

.compact-table :deep(.el-table__row) {
  cursor: pointer;
}

.host-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.host-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.host-dot.online { background: #22c55e; }
.host-dot.offline { background: #f85149; }

.elapsed-text {
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 12px;
  color: #8b949e;
}

/* ===== Table Footer ===== */
.table-footer {
  display: flex;
  justify-content: center;
  padding: 12px 16px;
  background: #161b22;
  border-top: 1px solid #30363d;
}

.table-footer :deep(.el-pagination) {
  --el-pagination-bg-color: transparent;
  --el-pagination-text-color: #8b949e;
  --el-pagination-button-bg-color: #21262d;
  --el-pagination-button-color: #c9d1d9;
}

/* ===== Detail Drawer ===== */
.detail-drawer :deep(.el-drawer) {
  background: #161b22;
}

.detail-drawer :deep(.el-drawer__header) {
  background: #0d1117;
  border-bottom: 1px solid #30363d;
  padding: 16px 20px;
}

.drawer-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.host-name {
  font-size: 16px;
  font-weight: 600;
  color: #f0f6fc;
}

.drawer-content {
  padding: 20px;
}

.info-section {
  margin-bottom: 24px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #f0f6fc;
}

.command-display,
.result-display {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  padding: 12px;
  overflow-x: auto;
}

.command-display pre,
.result-display pre {
  margin: 0;
  font-family: 'SF Mono', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #c9d1d9;
  white-space: pre-wrap;
  word-break: break-all;
}

.result-display.error {
  border-color: #f85149;
  background: rgba(248, 81, 73, 0.05);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.stat-item {
  padding: 12px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
}

.stat-label {
  display: block;
  font-size: 12px;
  color: #8b949e;
  margin-bottom: 4px;
}

.stat-value {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #f0f6fc;
}

.action-section {
  display: flex;
  gap: 12px;
}

/* ===== Transitions ===== */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

.slide-enter-active, .slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from, .slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* ===== Element Plus Dark Theme Overrides ===== */
:deep(.el-button) {
  --el-button-bg-color: #21262d;
  --el-button-border-color: #30363d;
  --el-button-text-color: #c9d1d9;
  --el-button-hover-bg-color: #30363d;
  --el-button-hover-border-color: #8b949e;
  --el-button-hover-text-color: #f0f6fc;
}

:deep(.el-input__wrapper) {
  background: #0d1117;
  border-color: #30363d;
}

:deep(.el-input__inner) {
  color: #c9d1d9;
}

:deep(.el-input__wrapper.is-focus) {
  border-color: #58a6ff;
}

:deep(.el-dialog) {
  background: #161b22;
  border: 1px solid #30363d;
}

:deep(.el-dialog__header) {
  border-bottom: 1px solid #30363d;
}

:deep(.el-tag) {
  background: #21262d;
  border-color: #30363d;
  color: #c9d1d9;
}

:deep(.el-tag--success) {
  background: rgba(34, 197, 94, 0.15);
  border-color: #22c55e;
  color: #22c55e;
}

:deep(.el-tag--danger) {
  background: rgba(248, 81, 73, 0.15);
  border-color: #f85149;
  color: #f85149;
}
</style>
