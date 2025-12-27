<template>
  <div class="result-detail">
    <div class="detail-header">
      <div class="header-left">
        <el-button size="small" @click="handleExpandAll">展开全部</el-button>
        <el-button size="small" @click="handleCollapseAll">折叠全部</el-button>
      </div>
      <div class="header-right">
        <span class="result-count">共 {{ results.length }} 条结果</span>
      </div>
    </div>

    <div class="detail-content">
      <el-collapse v-model="activeNames" accordion>
        <el-collapse-item
          v-for="(result, index) in results"
          :key="index"
          :name="index.toString()"
        >
          <template #title>
            <div class="collapse-title">
              <el-icon :color="result.status === 'success' ? '#67c23a' : '#f56c6c'">
                <CircleCheck v-if="result.status === 'success'" />
                <CircleClose v-else />
              </el-icon>
              <span class="host-name">{{ result.host }}</span>
              <el-tag
                :type="result.status === 'success' ? 'success' : 'danger'"
                size="small"
                style="margin-left: 10px"
              >
                {{ result.status === 'success' ? '成功' : '失败' }}
              </el-tag>
              <span class="elapsed" style="margin-left: 10px; color: var(--el-text-color-secondary)">
                耗时: {{ result.elapsed }}
              </span>
            </div>
          </template>

          <div class="result-content">
            <div v-if="result.error" class="error-section">
              <h4>错误信息</h4>
              <pre class="error-text">{{ result.error }}</pre>
            </div>

            <div v-if="result.result" class="result-section">
              <div class="section-header">
                <h4>执行结果</h4>
                <div class="section-actions">
                  <el-button
                    type="primary"
                    link
                    size="small"
                    @click="handleCopy(result)"
                  >
                    复制
                  </el-button>
                  <el-button
                    type="primary"
                    link
                    size="small"
                    @click="handleDownload(result)"
                  >
                    下载
                  </el-button>
                  <el-button
                    type="primary"
                    link
                    size="small"
                    @click="handleRetry(result)"
                  >
                    重试
                  </el-button>
                </div>
              </div>

              <!-- 日志内容 -->
              <pre v-if="result.result.log" class="log-content">{{ result.result.log }}</pre>
              
              <!-- 命令输出 -->
              <pre v-if="result.result.output" class="output-content">{{ result.result.output }}</pre>
              
              <!-- CPU信息 -->
              <div v-if="result.result.cpu" class="cpu-content">
                <pre>{{ result.result.cpu }}</pre>
              </div>
              
              <!-- 内存信息 -->
              <div v-if="result.result.output" class="memory-content">
                <pre>{{ result.result.output }}</pre>
              </div>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck, CircleClose } from '@element-plus/icons-vue'
import type { BatchExecuteResult } from '@/api/operations'
import { useConsoleStore } from '@/stores/console'

const consoleStore = useConsoleStore()

const activeNames = ref<string[]>([])

const results = computed(() => consoleStore.executionResults)

const handleExpandAll = () => {
  activeNames.value = results.value.map((_, index) => index.toString())
}

const handleCollapseAll = () => {
  activeNames.value = []
}

const handleCopy = (result: BatchExecuteResult) => {
  let text = ''
  if (result.result?.log) {
    text = result.result.log
  } else if (result.result?.output) {
    text = result.result.output
  } else if (result.error) {
    text = result.error
  }

  if (text) {
    navigator.clipboard.writeText(text).then(() => {
      ElMessage.success('已复制到剪贴板')
    }).catch(() => {
      ElMessage.error('复制失败')
    })
  }
}

const handleDownload = (result: BatchExecuteResult) => {
  let content = ''
  let filename = `${result.host}_result.txt`

  if (result.result?.log) {
    content = result.result.log
    filename = `${result.host}_log.txt`
  } else if (result.result?.output) {
    content = result.result.output
    filename = `${result.host}_output.txt`
  } else if (result.error) {
    content = result.error
    filename = `${result.host}_error.txt`
  }

  if (content) {
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('下载成功')
  }
}

const handleRetry = (result: BatchExecuteResult) => {
  emit('retry', result)
}

const emit = defineEmits<{
  retry: [result: BatchExecuteResult]
}>()
</script>

<style scoped>
.result-detail {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color);
}

.result-count {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
}

.collapse-title {
  display: flex;
  align-items: center;
  width: 100%;
}

.host-name {
  font-weight: 500;
  margin-left: 8px;
}

.result-content {
  padding: 15px;
}

.error-section {
  margin-bottom: 15px;
}

.error-section h4 {
  margin: 0 0 10px 0;
  color: var(--el-color-danger);
}

.error-text {
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
  color: var(--el-color-danger);
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.result-section h4 {
  margin: 0 0 10px 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.section-actions {
  display: flex;
  gap: 5px;
}

.log-content,
.output-content,
.cpu-content pre,
.memory-content pre {
  background: var(--el-fill-color-light);
  padding: 10px;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  max-height: 500px;
  overflow-y: auto;
}
</style>

