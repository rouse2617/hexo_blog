<template>
  <div class="operation-panel">
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <el-tab-pane label="查看日志" name="query_log">
        <el-form :model="logForm" label-width="100px">
          <el-form-item label="日志类型">
            <el-select v-model="logForm.log_type" placeholder="选择日志类型">
              <el-option label="Nginx" value="nginx" />
              <el-option label="应用日志" value="app" />
              <el-option label="系统日志" value="system" />
              <el-option label="自定义" value="custom" />
            </el-select>
          </el-form-item>
          <el-form-item label="日志路径" v-if="logForm.log_type === 'custom'">
            <el-input v-model="logForm.path" placeholder="请输入日志路径" />
          </el-form-item>
          <el-form-item label="查看行数">
            <el-input-number v-model="logForm.lines" :min="1" :max="10000" />
          </el-form-item>
          <el-form-item label="关键词">
            <el-input v-model="logForm.keyword" placeholder="过滤关键词（可选）" />
          </el-form-item>
          <el-form-item label="日志级别">
            <el-select v-model="logForm.level" placeholder="选择级别（可选）" clearable>
              <el-option label="错误" value="error" />
              <el-option label="警告" value="warn" />
              <el-option label="信息" value="info" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleExecuteLog" :loading="executing">
              执行查询
            </el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="执行命令" name="run_command">
        <el-form :model="commandForm" label-width="100px">
          <el-form-item label="命令">
            <el-input
              v-model="commandForm.command"
              type="textarea"
              :rows="4"
              placeholder="请输入要执行的命令"
            />
          </el-form-item>
          <el-form-item label="超时时间">
            <el-input-number v-model="commandForm.timeout" :min="1" :max="300" />
            <span style="margin-left: 10px; color: var(--el-text-color-secondary)">秒</span>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleExecuteCommand" :loading="executing">
              执行命令
            </el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="系统信息" name="system">
        <div class="system-actions">
          <el-button type="primary" @click="handleCheckCPU" :loading="executing">
            检查CPU
          </el-button>
          <el-button type="primary" @click="handleCheckMemory" :loading="executing">
            检查内存
          </el-button>
          <el-button type="primary" @click="handleCheckDisk" :loading="executing">
            检查磁盘
          </el-button>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useConsoleStore } from '@/stores/console'

const consoleStore = useConsoleStore()

const activeTab = ref('query_log')
const executing = ref(false)

const logForm = ref({
  log_type: 'nginx',
  path: '',
  lines: 100,
  keyword: '',
  level: ''
})

const commandForm = ref({
  command: '',
  timeout: 30
})

const handleTabChange = (tab: string) => {
  // 切换标签时清空表单
  if (tab === 'query_log') {
    logForm.value = {
      log_type: 'nginx',
      path: '',
      lines: 100,
      keyword: '',
      level: ''
    }
  } else if (tab === 'run_command') {
    commandForm.value = {
      command: '',
      timeout: 30
    }
  }
}

const executeOperation = async (
  operation: 'query_log' | 'run_command' | 'check_cpu' | 'check_memory' | 'check_disk',
  params?: Record<string, any>
) => {
  if (consoleStore.selectedHosts.length === 0) {
    ElMessage.warning('请先选择主机')
    return
  }

  executing.value = true
  try {
    await consoleStore.executeOperation(operation, consoleStore.selectedHosts, params)
    ElMessage.success('执行成功')
  } catch (error: any) {
    ElMessage.error(error.message || '执行失败')
  } finally {
    executing.value = false
  }
}

const handleExecuteLog = () => {
  const params: Record<string, any> = {
    log_type: logForm.value.log_type,
    lines: logForm.value.lines
  }
  if (logForm.value.path) {
    params.path = logForm.value.path
  }
  if (logForm.value.keyword) {
    params.keyword = logForm.value.keyword
  }
  if (logForm.value.level) {
    params.level = logForm.value.level
  }
  executeOperation('query_log', params)
}

const handleExecuteCommand = () => {
  if (!commandForm.value.command.trim()) {
    ElMessage.warning('请输入命令')
    return
  }
  executeOperation('run_command', {
    command: commandForm.value.command,
    timeout: commandForm.value.timeout
  })
}

const handleCheckCPU = () => {
  executeOperation('check_cpu')
}

const handleCheckMemory = () => {
  executeOperation('check_memory')
}

const handleCheckDisk = () => {
  executeOperation('check_disk')
}
</script>

<style scoped>
.operation-panel {
  padding: 20px;
}

.system-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
</style>
