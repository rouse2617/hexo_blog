<template>
  <div class="ops-console-v2">
    <!-- ===== 主体内容区 ===== -->
    <div class="main-content">
      <!-- 左侧主机筛选树 -->
      <div class="host-sidebar">
        <!-- 搜索栏 -->
        <div class="sidebar-search">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <path d="m21 21-4.35-4.35"/>
          </svg>
          <input
            v-model="searchQuery"
            placeholder="搜索主机..."
            class="search-input"
          />
          <button
            v-if="!hostStore.loading"
            class="refresh-btn"
            @click="hostStore.loadAllHosts()"
            title="刷新主机列表"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 2v6h-6M3 12a9 9 0 0 1 15-6.7L21 8M3 22v-6h6M21 12a9 9 0 0 1-15 6.7L3 16"/>
            </svg>
          </button>
          <div v-else class="loading-spinner"></div>
        </div>

        <!-- 快速过滤 - Segment Control 样式 -->
        <div class="quick-filters">
          <div class="segment-control">
            <button
              v-for="filter in filters"
              :key="filter.key"
              :class="['segment-item', { active: activeFilter === filter.key }]"
              @click="activeFilter = filter.key"
            >
              {{ filter.label }}
              <span class="segment-count">{{ filter.count }}</span>
            </button>
          </div>
          <button
            class="refresh-status-btn"
            :class="{ loading: refreshingStatus }"
            @click="refreshHostStatus"
            title="刷新主机状态"
          >
            <svg v-if="!refreshingStatus" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 2v6h-6M3 12a9 9 0 0 1 15-6.7L21 8M3 22v-6h6M21 12a9 9 0 0 1-15 6.7L3 16"/>
            </svg>
            <div v-else class="mini-spinner"></div>
          </button>
        </div>

        <!-- 主机树 -->
        <div class="host-tree-container">
          <!-- 加载状态 -->
          <div v-if="hostStore.loading" class="loading-state">
            <div class="loading-spinner"></div>
            <p>加载主机列表中...</p>
          </div>

          <!-- 空状态 -->
          <div v-else-if="hostGroups.length === 0" class="empty-state">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="2" width="20" height="8" rx="2"/>
              <rect x="2" y="14" width="20" height="8" rx="2"/>
              <line x1="6" y1="6" x2="6" y2="6"/>
              <line x1="6" y1="18" x2="6" y2="18"/>
            </svg>
            <p>暂无主机</p>
            <button class="add-hosts-btn" @click="$router.push('/hosts')">
              添加主机
            </button>
          </div>

          <!-- 主机列表 -->
          <div
            v-for="group in filteredHostGroups"
            :key="group.id"
            class="host-group"
          >
            <div class="group-header" @click="toggleGroup(group.id)">
              <svg class="chevron" :class="{ expanded: expandedGroups.has(group.id) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
              <span class="group-label">{{ group.label }}</span>
              <span class="group-count">{{ group.hosts.length }}</span>
            </div>
            <div v-show="expandedGroups.has(group.id)" class="group-hosts">
              <div
                v-for="host in group.hosts"
                :key="host.id"
                :class="['host-item', { selected: selectedHosts.includes(host.id), online: host.online }]"
                @click="toggleHost(host.id)"
              >
                <div class="host-checkbox">
                  <svg v-if="selectedHosts.includes(host.id)" class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                </div>
                <div class="host-info">
                  <span class="host-name">{{ host.name }}</span>
                  <span class="host-ip">{{ getHostIp(host.id) }}</span>
                </div>
                <span v-if="!host.online" class="host-status-badge">离线</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 底部统计 -->
        <div class="sidebar-footer">
          <div class="stat-item">
            <span class="stat-dot online"></span>
            <span class="stat-label">在线</span>
            <span class="stat-value">{{ onlineCount }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-dot offline"></span>
            <span class="stat-label">离线</span>
            <span class="stat-value">{{ offlineCount }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-dot selected"></span>
            <span class="stat-label">已选</span>
            <span class="stat-value">{{ selectedHosts.length }}</span>
          </div>
        </div>
      </div>

      <!-- 右侧操作面板 -->
      <div class="ops-panel">
        <!-- 面板内容 -->
        <div class="panel-content">
          <!-- 命令执行区域 -->
          <div class="command-pane">
            <div class="command-section">
              <!-- 多行命令输入 -->
              <div class="command-input-wrapper">
                <div class="terminal-header">
                  <span class="terminal-title">命令输入</span>
                  <div class="terminal-actions">
                    <button class="terminal-btn" @click="command = ''" title="清空">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 6 6 18M6 6l12 12"/>
                      </svg>
                    </button>
                  </div>
                </div>
                <textarea
                  ref="commandInput"
                  v-model="command"
                  placeholder="输入要执行的命令...
例如:
  ls -la /var/log
  systemctl status nginx
  ps aux | grep nginx"
                  @keydown.ctrl.enter="handleExecute"
                  class="command-input"
                  rows="6"
                ></textarea>
                <div class="command-footer">
                  <span class="command-hint">Ctrl + Enter 执行</span>
                  <span class="command-lines">{{ command.split('\n').filter(l => l.trim()).length }} 行</span>
                </div>
              </div>

              <div class="command-options">
                <div class="option-group">
                  <label class="option-label">超时时间</label>
                  <select v-model="timeout" class="option-select">
                    <option value="10">10 秒</option>
                    <option value="30">30 秒</option>
                    <option value="60">1 分钟</option>
                    <option value="120">2 分钟</option>
                    <option value="300">5 分钟</option>
                  </select>
                </div>
                <button
                  class="execute-btn"
                  :disabled="!command.trim() || selectedHosts.length === 0"
                  @click="handleExecute"
                >
                  <svg viewBox="0 0 24 24" fill="currentColor">
                    <polygon points="5 3 19 12 5 21 5 3"/>
                  </svg>
                  执行命令
                  <span class="execute-count">({{ selectedHosts.length }} 台)</span>
                </button>
              </div>

              <!-- 快速命令建议 -->
              <div class="quick-commands">
                <span class="quick-label">快速:</span>
                <button
                  v-for="cmd in quickCommands"
                  :key="cmd"
                  class="quick-btn"
                  @click="command = cmd"
                >
                  {{ cmd }}
                </button>
              </div>
            </div>

            <!-- 执行结果 - 终端风格 -->
            <div v-if="results.length > 0" class="results-section">
              <div class="results-header">
                <span class="results-title">执行结果 ({{ results.length }})</span>
                <button class="clear-btn" @click="results = []">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M18 6 6 18M6 6l12 12"/>
                  </svg>
                  清空
                </button>
              </div>
              <div class="results-list">
                <div
                  v-for="result in results"
                  :key="result.id"
                  :class="['result-item', result.status]"
                >
                  <div class="result-header">
                    <div class="result-host">
                      <svg class="host-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="2" y="2" width="20" height="8" rx="2"/>
                        <rect x="2" y="14" width="20" height="8" rx="2"/>
                        <line x1="6" y1="6" x2="6" y2="6"/>
                        <line x1="6" y1="18" x2="6" y2="18"/>
                      </svg>
                      {{ result.host }}
                    </div>
                    <div class="result-meta">
                      <span :class="['result-status-badge', result.status]">
                        <svg v-if="result.status === 'success'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <polyline points="20 6 9 17 4 12"/>
                        </svg>
                        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <circle cx="12" cy="12" r="10"/>
                          <line x1="15" y1="9" x2="9" y2="15"/>
                          <line x1="9" y1="9" x2="15" y2="15"/>
                        </svg>
                        {{ result.status === 'success' ? '成功' : '失败' }}
                      </span>
                      <span class="result-time">{{ result.elapsed }}</span>
                    </div>
                  </div>
                  <pre class="result-output">{{ result.output }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useHostStore } from '@/stores/host'
import type { Host } from '@/api/host'
import { batchExecute } from '@/api/operations'

// ===== 数据类型 =====
interface HostNode {
  id: string
  name: string
  online: boolean
  group?: string
}

interface HostGroup {
  id: string
  label: string
  hosts: HostNode[]
}

interface Result {
  id: string
  host: string
  status: 'success' | 'error'
  output: string
  elapsed: string  // 改为字符串类型，如 "283ms"
}

// ===== 状态 =====
const activeFilter = ref('all')
const searchQuery = ref('')
const selectedHosts = ref<string[]>([])
const expandedGroups = ref(new Set<string>())
const command = ref('')
const timeout = ref(30)
const executing = ref(false)
const refreshingStatus = ref(false)

// 使用真实主机数据
const hostStore = useHostStore()

// 将主机按分组组织
const hostGroups = ref<HostGroup[]>([])

// 快速命令
const quickCommands = [
  'systemctl status nginx',
  'df -h',
  'tail -100 /var/log/syslog',
  'ps aux | grep nginx'
]

// 过滤器
const filters = ref([
  { key: 'all', label: '全部', count: 0, color: 'gray' },
  { key: 'online', label: '在线', count: 0, color: 'green' },
  { key: 'offline', label: '离线', count: 0, color: 'red' },
  { key: 'selected', label: '已选', count: 0, color: 'blue' }
])

// 执行结果
const results = ref<Result[]>([])

// ===== 计算属性 =====
const filteredHostGroups = computed(() => {
  let groups = hostGroups.value

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    groups = groups.map(g => ({
      ...g,
      hosts: g.hosts.filter(h => h.name.toLowerCase().includes(query))
    })).filter(g => g.hosts.length > 0)
  }

  // 状态过滤
  if (activeFilter.value === 'online') {
    groups = groups.map(g => ({
      ...g,
      hosts: g.hosts.filter(h => h.online)
    })).filter(g => g.hosts.length > 0)
  } else if (activeFilter.value === 'offline') {
    groups = groups.map(g => ({
      ...g,
      hosts: g.hosts.filter(h => !h.online)
    })).filter(g => g.hosts.length > 0)
  } else if (activeFilter.value === 'selected') {
    groups = groups.map(g => ({
      ...g,
      hosts: g.hosts.filter(h => selectedHosts.value.includes(h.id))
    })).filter(g => g.hosts.length > 0)
  }

  return groups
})

const onlineCount = computed(() => {
  return hostStore.allHosts.filter((h: Host) => h.status === 'online').length
})

const offlineCount = computed(() => {
  return hostStore.allHosts.filter((h: Host) => h.status === 'offline' || h.status === 'unknown').length
})

// 更新过滤器计数
const updateFilterCounts = () => {
  filters.value[0].count = hostStore.allHosts.length
  filters.value[1].count = onlineCount.value
  filters.value[2].count = offlineCount.value
  filters.value[3].count = selectedHosts.value.length
}

// 监听主机数据变化，自动组织分组
watch(() => hostStore.allHosts, (hosts: Host[]) => {
  // 按分组组织主机
  const grouped = new Map<string, Host[]>()

  hosts.forEach(host => {
    const groupName = host.group || '未分组'
    if (!grouped.has(groupName)) {
      grouped.set(groupName, [])
    }
    grouped.get(groupName)!.push(host)
  })

  // 转换为 HostGroup 格式
  const groups: HostGroup[] = []
  let groupIndex = 0

  // 先添加有分组的主机
  grouped.forEach((hostsList, groupName) => {
    if (groupName !== '未分组') {
      groups.push({
        id: `group-${groupIndex++}`,
        label: groupName,
        hosts: hostsList.map((h: Host) => ({
          id: h.id || h.name,
          name: h.name,
          online: h.status === 'online',
          group: h.group
        }))
      })
    }
  })

  // 添加未分组的主机
  if (grouped.has('未分组') && grouped.get('未分组')!.length > 0) {
    groups.push({
      id: `group-${groupIndex++}`,
      label: '未分组',
      hosts: grouped.get('未分组')!.map((h: Host) => ({
        id: h.id || h.name,
        name: h.name,
        online: h.status === 'online',
        group: h.group
      }))
    })
  }

  hostGroups.value = groups

  // 默认展开所有分组
  expandedGroups.value = new Set(groups.map(g => g.id))

  // 更新过滤器计数
  updateFilterCounts()
}, { immediate: true })

// ===== 方法 =====
const toggleGroup = (id: string) => {
  if (expandedGroups.value.has(id)) {
    expandedGroups.value.delete(id)
  } else {
    expandedGroups.value.add(id)
  }
}

const toggleHost = (id: string) => {
  const index = selectedHosts.value.indexOf(id)
  if (index > -1) {
    selectedHosts.value.splice(index, 1)
  } else {
    selectedHosts.value.push(id)
  }
  // 更新过滤计数
  filters.value[3].count = selectedHosts.value.length
}

const getHostById = (id: string) => {
  return hostStore.allHosts.find((h: Host) => (h.id || h.name) === id) || null
}

const getHostIp = (id: string) => {
  const host = getHostById(id)
  return host?.host || ''
}

// 刷新主机状态
const refreshHostStatus = async () => {
  refreshingStatus.value = true
  try {
    await hostStore.refreshAllHostStatus()
    ElMessage.success('主机状态已刷新')
  } catch (error: any) {
    console.error('刷新状态失败:', error)
    ElMessage.error('刷新状态失败: ' + (error.message || '未知错误'))
  } finally {
    refreshingStatus.value = false
  }
}

const handleExecute = async () => {
  if (!command.value.trim()) {
    ElMessage.warning('请输入命令')
    return
  }
  if (selectedHosts.value.length === 0) {
    ElMessage.warning('请先选择主机')
    return
  }

  executing.value = true

  try {
    // 获取主机名称列表
    const hostNames = selectedHosts.value.map(id => {
      const host = getHostById(id)
      return host?.name || id
    })

    // 调用后端 API 批量执行命令
    const response = await batchExecute({
      operation: 'run_command',
      hosts: hostNames,
      params: {
        command: command.value,
        timeout: timeout.value
      }
    })

    // 处理执行结果
    const successCount = response.filter(r => r.status === 'success').length
    const errorCount = response.filter(r => r.status === 'error').length

    // 将结果添加到结果列表
    for (const result of response) {
      results.value.unshift({
        id: Date.now() + Math.random().toString(),
        host: result.host,
        status: result.status,
        output: result.status === 'success'
          ? (result.result?.output as string || '执行成功')
          : (result.error || '执行失败'),
        elapsed: result.elapsed || '0ms'
      })
    }

    // 显示执行结果
    if (errorCount === 0) {
      ElMessage.success(`命令已在 ${successCount} 台主机上执行成功`)
    } else if (successCount === 0) {
      ElMessage.error(`命令在 ${errorCount} 台主机上执行失败`)
    } else {
      ElMessage.warning(`命令执行完成：${successCount} 台成功，${errorCount} 台失败`)
    }
  } catch (error: any) {
    console.error('执行命令失败:', error)
    ElMessage.error(error.message || '执行命令失败，请稍后重试')
  } finally {
    executing.value = false
  }
}

onMounted(async () => {
  // 加载真实主机数据
  if (hostStore.allHosts.length === 0) {
    await hostStore.loadAllHosts()
  }
  // 自动启动状态刷新
  hostStore.startStatusAutoRefresh()
})
</script>

<style scoped>
/* ===== 全局变量 ===== */
.ops-console-v2 {
  --color-bg: #f8fafc;
  --color-card: #ffffff;
  --color-border: #e2e8f0;
  --color-text-primary: #1e293b;
  --color-text-secondary: #64748b;
  --color-text-muted: #94a3b8;
  --color-primary: #6366f1;
  --color-primary-light: #818cf8;
  --color-success: #22c55e;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-info: #3b82f6;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1);
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;

  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: var(--color-bg);
  color: var(--color-text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

/* ===== 主体内容 ===== */
.main-content {
  display: flex;
  flex: 1;
  gap: 0;
  overflow: hidden;
}

/* ===== 左侧边栏 ===== */
.host-sidebar {
  width: 280px;
  display: flex;
  flex-direction: column;
  background: var(--color-card);
  border-right: 1px solid var(--color-border);
}

.sidebar-search {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px;
  border-bottom: 1px solid var(--color-border);
}

.search-icon {
  width: 18px;
  height: 18px;
  color: var(--color-text-muted);
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 14px;
  color: var(--color-text-primary);
  background: transparent;
}

.search-input::placeholder {
  color: var(--color-text-muted);
}

.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: none;
  border: none;
  border-radius: var(--radius-md);
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s;
}

.refresh-btn:hover {
  background: #f1f5f9;
  color: var(--color-primary);
}

.refresh-btn svg {
  width: 16px;
  height: 16px;
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.quick-filters {
  display: flex;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
}

/* Segment Control */
.segment-control {
  display: flex;
  width: 100%;
  background: var(--color-bg);
  border-radius: var(--radius-md);
  padding: 3px;
  gap: 2px;
}

.segment-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.segment-item:hover {
  color: var(--color-text-primary);
  background: rgba(0, 0, 0, 0.03);
}

.segment-item.active {
  background: white;
  color: var(--color-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.segment-count {
  font-size: 11px;
  padding: 2px 6px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  min-width: 20px;
  text-align: center;
}

.segment-item.active .segment-count {
  background: var(--color-primary);
  color: white;
}

/* 刷新状态按钮 */
.refresh-status-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.refresh-status-btn:hover {
  background: #f1f5f9;
  color: var(--color-primary);
}

.refresh-status-btn svg {
  width: 16px;
  height: 16px;
}

.refresh-status-btn.loading {
  pointer-events: none;
  opacity: 0.7;
}

.mini-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.host-tree-container {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--color-text-muted);
}

.loading-state p {
  margin: 12px 0 0 0;
  font-size: 13px;
}

.host-group {
  margin-bottom: 8px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: background 0.2s;
}

.group-header:hover {
  background: var(--color-bg);
}

.chevron {
  width: 16px;
  height: 16px;
  color: var(--color-text-muted);
  transition: transform 0.2s;
}

.chevron.expanded {
  transform: rotate(90deg);
}

.group-label {
  flex: 1;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.group-count {
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-bg);
  padding: 2px 8px;
  border-radius: 10px;
}

.group-hosts {
  margin-left: 24px;
}

.host-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  margin: 1px 0;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
}

.host-item:hover {
  background: #f1f5f9;
}

.host-item.selected {
  background: #eef2ff;
  box-shadow: 0 0 0 1px var(--color-primary);
}

/* 复选框 */
.host-checkbox {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  border: 2px solid var(--color-border);
  border-radius: 4px;
  background: white;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.host-item.online .host-checkbox {
  border-color: var(--color-success);
}

.host-item.selected .host-checkbox {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.check-icon {
  width: 14px;
  height: 14px;
  color: white;
}

/* 主机信息 - 两行布局 */
.host-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.host-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host-ip {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 离线状态标签 */
.host-status-badge {
  padding: 3px 8px;
  font-size: 11px;
  color: var(--color-text-muted);
  background: #f1f5f9;
  border-radius: 4px;
  flex-shrink: 0;
}

.sidebar-footer {
  display: flex;
  justify-content: space-around;
  padding: 16px;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.stat-dot.online { background: var(--color-success); }
.stat-dot.offline { background: var(--color-danger); }
.stat-dot.selected { background: var(--color-primary); }

.stat-label {
  font-size: 11px;
  color: var(--color-text-muted);
}

.stat-value {
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}

/* ===== 右侧操作面板 ===== */
.ops-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--color-card);
  overflow: hidden;
}

.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

/* ===== 命令面板 ===== */
.command-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 命令输入 - 终端风格 */
.command-input-wrapper {
  display: flex;
  flex-direction: column;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: all 0.2s;
}

.command-input-wrapper:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
}

.terminal-title {
  font-size: 12px;
  font-weight: 600;
  color: #7d8590;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.terminal-actions {
  display: flex;
  gap: 8px;
}

.terminal-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: #7d8590;
  cursor: pointer;
  transition: all 0.2s;
}

.terminal-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #e5e5e5;
}

.terminal-btn svg {
  width: 16px;
  height: 16px;
}

.command-input {
  flex: 1;
  min-height: 150px;
  padding: 16px;
  background: #0d1117;
  border: none;
  outline: none;
  resize: vertical;
  font-size: 14px;
  font-family: 'JetBrains Mono', 'SF Mono', 'Consolas', 'Menlo', 'Courier New', monospace;
  color: #e5e5e5;
  line-height: 1.6;
}

.command-input::placeholder {
  color: #6b7280;
}

.command-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #161b22;
  border-top: 1px solid #30363d;
}

.command-hint {
  font-size: 12px;
  color: #7d8590;
}

.command-lines {
  font-size: 11px;
  color: #7d8590;
  font-family: 'JetBrains Mono', monospace;
}

.command-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.option-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.option-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.option-select {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 14px;
  color: var(--color-text-primary);
  background: var(--color-card);
  cursor: pointer;
}

.execute-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 24px;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.execute-btn:hover:not(:disabled) {
  background: #4f46e5;
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.execute-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.execute-btn svg {
  width: 16px;
  height: 16px;
}

.execute-count {
  font-size: 12px;
  opacity: 0.9;
}

.quick-commands {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.quick-label {
  font-size: 13px;
  color: var(--color-text-secondary);
}

.quick-btn {
  padding: 6px 14px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.quick-btn:hover {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: white;
}

/* ===== 结果列表 - 终端风格 ===== */
.results-section {
  margin-top: 24px;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.results-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.clear-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.clear-btn:hover {
  background: #fee2e2;
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.clear-btn svg {
  width: 14px;
  height: 14px;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.result-item {
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.result-item.success {
  border-color: #238636;
}

.result-item.error {
  border-color: #da3633;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
}

.result-host {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #e5e5e5;
}

.result-host .host-icon {
  width: 16px;
  height: 16px;
  color: #7d8590;
}

.result-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.result-status-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
}

.result-status-badge.success {
  background: #238636;
  color: white;
}

.result-status-badge.error {
  background: #da3633;
  color: white;
}

.result-status-badge svg {
  width: 14px;
  height: 14px;
}

.result-time {
  font-size: 12px;
  color: #7d8590;
  font-family: 'JetBrains Mono', monospace;
}

.result-output {
  margin: 0;
  padding: 16px;
  font-family: 'JetBrains Mono', 'SF Mono', 'Consolas', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  color: #e5e5e5;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  overflow-x: auto;
}

.result-item.error .result-output {
  color: #ff7b72;
}
</style>
