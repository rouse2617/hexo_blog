<template>
  <ErrorBoundary>
    <el-container class="app-container">
      <el-aside width="220px" class="app-aside">
        <AppSidebar />
      </el-aside>
      <el-container>
        <el-header class="app-header">
          <AppHeader />
        </el-header>
        <el-main class="app-main">
          <ErrorBoundary>
            <router-view />
          </ErrorBoundary>
        </el-main>
      </el-container>
    </el-container>
  </ErrorBoundary>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import AppHeader from '@/components/common/AppHeader.vue'
import AppSidebar from '@/components/common/AppSidebar.vue'
import ErrorBoundary from '@/components/common/ErrorBoundary.vue'

// 全局未捕获错误处理
onMounted(() => {
  // 处理未捕获的 Promise 错误
  window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled Promise Rejection:', event.reason)
    ElMessage.error('操作失败: ' + (event.reason?.message || '未知错误'))
    event.preventDefault()
  })

  // 处理全局 JS 错误
  window.addEventListener('error', (event) => {
    console.error('Global Error:', event.error)
    // 忽略资源加载错误
    if (event.target && (event.target as HTMLElement).tagName) {
      return
    }
    ElMessage.error('发生错误: ' + (event.error?.message || '未知错误'))
  })
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  width: 100%;
}

.app-container {
  height: 100%;
}

.app-aside {
  background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
  border-right: 1px solid #2a2a4a;
}

.app-header {
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  padding: 0 20px;
  height: 60px;
}

.app-main {
  background: #f5f7fa;
  padding: 20px;
  overflow-y: auto;
}
</style>
