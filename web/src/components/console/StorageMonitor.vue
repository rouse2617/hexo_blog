<template>
  <div class="storage-monitor">
    <div class="monitor-header">
      <h3>存储系统监控</h3>
      <div class="header-actions">
        <el-button-group>
          <el-button
            size="small"
            @click="refreshAll"
            :loading="refreshing"
          >
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button
            size="small"
            type="primary"
            @click="showDiagDialog = true"
          >
            <el-icon><Search /></el-icon>
            诊断
          </el-button>
        </el-button-group>
      </div>
    </div>

    <!-- 存储健康概览 -->
    <div class="health-overview">
      <div class="overview-card filesystem">
        <div class="card-icon">
          <el-icon :size="32"><FolderOpened /></el-icon>
        </div>
        <div class="card-content">
          <div class="card-title">文件系统</div>
          <div class="card-value" :class="filesystemHealth.class">
            {{ filesystemHealth.text }}
          </div>
          <div class="card-subtitle">{{ filesystemHealth.count }} 个挂载点</div>
        </div>
      </div>

      <div class="overview-card raid">
        <div class="card-icon">
          <el-icon :size="32"><Files /></el-icon>
        </div>
        <div class="card-content">
          <div class="card-title">RAID 阵列</div>
          <div class="card-value" :class="raidHealth.class">
            {{ raidHealth.text }}
          </div>
          <div class="card-subtitle">{{ raidHealth.count }} 个阵列</div>
        </div>
      </div>

      <div class="overview-card disk">
        <div class="card-icon">
          <el-icon :size="32"><Coin /></el-icon>
        </div>
        <div class="card-content">
          <div class="card-title">磁盘空间</div>
          <div class="card-value" :class="diskHealth.class">
            {{ diskHealth.text }}
          </div>
          <div class="card-subtitle">{{ diskHealth.usedPercent }}% 已使用</div>
        </div>
      </div>

      <div class="overview-card io">
        <div class="card-icon">
          <el-icon :size="32"><Odometer /></el-icon>
        </div>
        <div class="card-content">
          <div class="card-title">I/O 状态</div>
          <div class="card-value" :class="ioHealth.class">
            {{ ioHealth.text }}
          </div>
          <div class="card-subtitle">{{ ioHealth.waitTime }}ms 等待</div>
        </div>
      </div>
    </div>

    <!-- 存储详情标签页 -->
    <el-tabs v-model="activeTab" class="storage-tabs">
      <el-tab-pane name="filesystem" label="文件系统">
        <template #label>
          <div class="tab-label">
            <el-icon><FolderOpened /></el-icon>
            <span>文件系统</span>
          </div>
        </template>
        <FilesystemPanel :selected-hosts="selectedHosts" />
      </el-tab-pane>

      <el-tab-pane name="raid" label="RAID">
        <template #label>
          <div class="tab-label">
            <el-icon><Files /></el-icon>
            <span>RAID 阵列</span>
          </div>
        </template>
        <RAIDPanel :selected-hosts="selectedHosts" />
      </el-tab-pane>

      <el-tab-pane name="lvm" label="LVM">
        <template #label>
          <div class="tab-label">
            <el-icon><Grid /></el-icon>
            <span>LVM</span>
          </div>
        </template>
        <LVMPanel :selected-hosts="selectedHosts" />
      </el-tab-pane>

      <el-tab-pane name="performance" label="性能">
        <template #label>
          <div class="tab-label">
            <el-icon><TrendCharts /></el-icon>
            <span>I/O 性能</span>
          </div>
        </template>
        <IOPerformancePanel :selected-hosts="selectedHosts" />
      </el-tab-pane>
    </el-tabs>

    <!-- 存储诊断对话框 -->
    <el-dialog
      v-model="showDiagDialog"
      title="存储系统诊断"
      width="600px"
    >
      <el-form label-width="100px">
        <el-form-item label="诊断类型">
          <el-select v-model="diagType" placeholder="选择诊断类型">
            <el-option label="文件系统检查" value="filesystem" />
            <el-option label="RAID 状态检查" value="raid" />
            <el-option label="磁盘健康检查" value="disk" />
            <el-option label="LVM 状态检查" value="lvm" />
            <el-option label="I/O 性能分析" value="io" />
            <el-option label="完整诊断" value="full" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标主机">
          <el-select v-model="diagHosts" multiple placeholder="选择主机">
            <el-option
              v-for="host in availableHosts"
              :key="host.id"
              :label="host.name"
              :value="host.name"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="详细输出">
          <el-switch v-model="diagDetail" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDiagDialog = false">取消</el-button>
        <el-button type="primary" @click="runDiagnosis" :loading="diagnosing">
          开始诊断
        </el-button>
      </template>
    </el-dialog>

    <!-- 诊断结果对话框 -->
    <el-dialog
      v-model="showResultDialog"
      title="诊断结果"
      width="800px"
    >
      <div class="diag-results">
        <div v-for="(result, index) in diagResults" :key="index" class="result-item">
          <div class="result-header">
            <span class="host-name">{{ result.host }}</span>
            <el-tag :type="result.status === 'success' ? 'success' : 'danger'" size="small">
              {{ result.status }}
            </el-tag>
          </div>
          <pre v-if="diagDetail" class="result-output">{{ result.output }}</pre>
          <div v-else class="result-summary">{{ result.summary }}</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showResultDialog = false">关闭</el-button>
        <el-button type="primary" @click="copyResults">复制结果</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh, Search, FolderOpened, Files, Coin, Odometer,
  Grid, TrendCharts
} from '@element-plus/icons-vue'
import { useConsoleStore } from '@/stores/console'
import { useHostStore } from '@/stores/host'
import FilesystemPanel from './storage/FilesystemPanel.vue'
import RAIDPanel from './storage/RAIDPanel.vue'
import LVMPanel from './storage/LVMPanel.vue'
import IOPerformancePanel from './storage/IOPerformancePanel.vue'

const consoleStore = useConsoleStore()
const hostStore = useHostStore()

const refreshing = ref(false)
const showDiagDialog = ref(false)
const showResultDialog = ref(false)
const diagnosing = ref(false)
const activeTab = ref('filesystem')
const diagType = ref('filesystem')
const diagHosts = ref<string[]>([])
const diagDetail = ref(false)
const diagResults = ref<any[]>([])

// 选中的主机
const selectedHosts = computed(() => consoleStore.selectedHosts)

// 可用主机列表
const availableHosts = computed(() => {
  return hostStore.allHosts.map(h => ({
    id: h.id || h.name,
    name: h.name
  }))
})

// 文件系统健康状态
const filesystemHealth = ref({
  text: '正常',
  class: 'healthy',
  count: 0
})

// RAID 健康状态
const raidHealth = ref({
  text: '正常',
  class: 'healthy',
  count: 0
})

// 磁盘空间健康状态
const diskHealth = ref({
  text: '正常',
  class: 'healthy',
  usedPercent: 45
})

// I/O 健康状态
const ioHealth = ref({
  text: '正常',
  class: 'healthy',
  waitTime: 5
})

// 刷新所有数据
const refreshAll = async () => {
  refreshing.value = true
  try {
    // 触发子组件刷新
    await Promise.all([
      // 这里可以调用各个子组件的刷新方法
    ])
    ElMessage.success('刷新成功')
  } catch (error: any) {
    ElMessage.error(error.message || '刷新失败')
  } finally {
    refreshing.value = false
  }
}

// 运行诊断
const runDiagnosis = async () => {
  if (diagHosts.value.length === 0) {
    ElMessage.warning('请选择至少一个主机')
    return
  }

  diagnosing.value = true
  showDiagDialog.value = false

  try {
    // 根据诊断类型执行不同的操作
    const operation = diagType.value === 'raid' ? 'check_raid' : 'check_filesystem'

    const results = await consoleStore.executeOperation(
      operation as any,
      diagHosts.value,
      { detail: diagDetail.value }
    )

    diagResults.value = results.map((r: any) => ({
      host: r.host,
      status: r.status,
      output: r.result,
      summary: r.result?.substring(0, 200) + '...'
    }))

    showResultDialog.value = true
    ElMessage.success('诊断完成')
  } catch (error: any) {
    ElMessage.error(error.message || '诊断失败')
  } finally {
    diagnosing.value = false
  }
}

// 复制诊断结果
const copyResults = () => {
  const text = diagResults.value.map(r =>
    `## ${r.host}\n${r.output}`
  ).join('\n\n')

  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 初始化
onMounted(() => {
  if (selectedHosts.value.length > 0) {
    diagHosts.value = [...selectedHosts.value]
  }
})
</script>

<style scoped>
.storage-monitor {
  padding: 20px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid var(--el-border-color);
}

.monitor-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.health-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.overview-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  transition: all 0.3s;
}

.overview-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 8px;
}

.filesystem .card-icon { background: #e3f2fd; color: #1976d2; }
.raid .card-icon { background: #f3e5f5; color: #7b1fa2; }
.disk .card-icon { background: #fff3e0; color: #f57c00; }
.io .card-icon { background: #e8f5e9; color: #388e3c; }

.card-content {
  flex: 1;
}

.card-title {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.card-value {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 4px;
}

.card-value.healthy { color: #22c55e; }
.card-value.warning { color: #f59e0b; }
.card-value.critical { color: #ef4444; }

.card-subtitle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.storage-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.storage-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.storage-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: auto;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.diag-results {
  max-height: 500px;
  overflow-y: auto;
}

.result-item {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.host-name {
  font-weight: 600;
}

.result-output {
  margin: 0;
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
}

.result-summary {
  color: var(--el-text-color-regular);
  font-size: 14px;
}
</style>
