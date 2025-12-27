<template>
  <div class="console-page">
    <div class="page-header">
      <h2>批量操作控制台</h2>
      <p class="page-desc">选择多个主机节点，执行批量操作，查看执行结果，使用AI智能分析</p>
    </div>

    <div class="console-layout">
      <!-- 左侧：主机选择 -->
      <div class="left-panel">
        <HostSelector />
      </div>

      <!-- 右侧：操作和结果区域 -->
      <div class="right-panel">
        <!-- 操作区域 -->
        <div class="operation-section">
          <OperationPanel />
        </div>

        <!-- 结果区域 -->
        <div v-if="hasResults" class="result-section">
          <el-tabs v-model="resultTab">
            <el-tab-pane label="结果概览" name="overview">
              <ResultOverview @view-detail="handleViewDetail" />
            </el-tab-pane>
            <el-tab-pane label="详细结果" name="detail">
              <ResultDetail @retry="handleRetry" />
            </el-tab-pane>
            <el-tab-pane label="AI分析" name="analysis">
              <AIAnalysis />
            </el-tab-pane>
          </el-tabs>
        </div>

        <!-- 空状态 -->
        <div v-else class="empty-state">
          <el-empty description="请选择主机并执行操作，结果将显示在这里" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import HostSelector from '@/components/console/HostSelector.vue'
import OperationPanel from '@/components/console/OperationPanel.vue'
import ResultOverview from '@/components/console/ResultOverview.vue'
import ResultDetail from '@/components/console/ResultDetail.vue'
import AIAnalysis from '@/components/console/AIAnalysis.vue'
import { useConsoleStore } from '@/stores/console'
import type { BatchExecuteResult } from '@/api/operations'

const consoleStore = useConsoleStore()

const resultTab = ref('overview')

const hasResults = computed(() => consoleStore.executionResults.length > 0)

const handleViewDetail = (_result: BatchExecuteResult) => {
  resultTab.value = 'detail'
  // 可以滚动到对应结果
}

const handleRetry = async (result: BatchExecuteResult) => {
  if (!consoleStore.currentOperation) {
    ElMessage.warning('无法重试：未知操作类型')
    return
  }

  try {
    await consoleStore.executeOperation(
      consoleStore.currentOperation,
      [result.host],
      consoleStore.operationParams
    )
    ElMessage.success('重试成功')
    resultTab.value = 'detail'
  } catch (error: any) {
    ElMessage.error(error.message || '重试失败')
  }
}
</script>

<style scoped>
.console-page {
  height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
  padding: 20px;
  overflow: hidden;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
}

.page-desc {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.console-layout {
  flex: 1;
  display: flex;
  gap: 20px;
  overflow: hidden;
}

.left-panel {
  width: 30%;
  min-width: 300px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow: hidden;
}

.operation-section {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.result-section {
  flex: 1;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-bg-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.result-section :deep(.el-tabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.result-section :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
}

.result-section :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-bg-color);
}
</style>
