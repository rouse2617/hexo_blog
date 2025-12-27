<template>
  <div v-if="error" class="error-boundary">
    <div class="error-content">
      <el-icon :size="64" color="#f56c6c"><WarningFilled /></el-icon>
      <h2 class="error-title">页面出现错误</h2>
      <p class="error-message">{{ error.message || '发生了未知错误' }}</p>
      <div class="error-actions">
        <el-button type="primary" @click="handleRetry">重试</el-button>
        <el-button @click="handleGoHome">返回首页</el-button>
      </div>
      <details v-if="errorInfo" class="error-details">
        <summary>错误详情</summary>
        <pre>{{ errorInfo }}</pre>
      </details>
    </div>
  </div>
  <slot v-else></slot>
</template>

<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import { useRouter } from 'vue-router'
import { WarningFilled } from '@element-plus/icons-vue'

const router = useRouter()
const error = ref<Error | null>(null)
const errorInfo = ref<string>('')

onErrorCaptured((err: Error, instance, info: string) => {
  error.value = err
  errorInfo.value = `组件: ${instance?.$options?.name || '未知'}\n信息: ${info}\n堆栈: ${err.stack || ''}`
  console.error('ErrorBoundary caught:', err, info)
  // 返回 false 阻止错误继续传播
  return false
})

const handleRetry = () => {
  error.value = null
  errorInfo.value = ''
}

const handleGoHome = () => {
  error.value = null
  errorInfo.value = ''
  router.push('/')
}
</script>

<style scoped>
.error-boundary {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  padding: 40px;
}

.error-content {
  text-align: center;
  max-width: 500px;
}

.error-title {
  font-size: 24px;
  color: #303133;
  margin: 20px 0 10px;
}

.error-message {
  font-size: 14px;
  color: #909399;
  margin-bottom: 20px;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-bottom: 20px;
}

.error-details {
  text-align: left;
  background: #f5f7fa;
  border-radius: 8px;
  padding: 12px;
  margin-top: 20px;
}

.error-details summary {
  cursor: pointer;
  color: #606266;
  font-size: 14px;
  margin-bottom: 10px;
}

.error-details pre {
  font-size: 12px;
  color: #909399;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
</style>
