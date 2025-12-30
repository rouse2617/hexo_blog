<template>
  <div class="ai-analysis">
    <div class="analysis-header">
      <h3>AI智能分析</h3>
      <el-button
        type="primary"
        @click="handleAnalyze"
        :loading="analyzing"
        :disabled="!hasResults"
      >
        开始分析
      </el-button>
    </div>

    <div v-if="analyzing" class="analyzing-tip">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>AI正在分析中，请稍候...</span>
    </div>

    <div v-if="currentAnalysis" class="analysis-result">
      <div class="result-header">
        <h4>分析结果</h4>
        <div class="result-actions">
          <el-button type="primary" link size="small" @click="handleCopyAnalysis">
            复制
          </el-button>
          <el-button type="primary" link size="small" @click="handleSaveAnalysis">
            保存
          </el-button>
        </div>
      </div>
      <div class="result-content">
        <div class="markdown-content" v-html="formattedAnalysis"></div>
      </div>
    </div>

    <div v-if="analysisHistory.length > 0" class="analysis-history">
      <div class="history-header">
        <h4>分析历史</h4>
        <el-button type="primary" link size="small" @click="loadHistory">
          刷新
        </el-button>
      </div>
      <el-timeline>
        <el-timeline-item
          v-for="item in analysisHistory"
          :key="item.id"
          :timestamp="formatTime(item.created_at)"
        >
          <div class="history-item">
            <div class="item-header">
              <span class="item-id">ID: {{ item.id.slice(0, 8) }}...</span>
              <el-button
                type="primary"
                link
                size="small"
                @click="handleViewHistory(item)"
              >
                查看
              </el-button>
              <el-button
                type="danger"
                link
                size="small"
                @click="handleDeleteHistory(item.id)"
              >
                删除
              </el-button>
            </div>
            <div v-if="item.question" class="item-question">
              问题: {{ item.question }}
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
    </div>

    <!-- 分析对话框 -->
    <el-dialog v-model="showAnalyzeDialog" title="AI分析" width="600px">
      <el-input
        v-model="analyzeQuestion"
        type="textarea"
        :rows="3"
        placeholder="请输入您特别关注的问题（可选）"
      />
      <template #footer>
        <el-button @click="showAnalyzeDialog = false">取消</el-button>
        <el-button type="primary" @click="confirmAnalyze" :loading="analyzing">
          确认分析
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { useConsoleStore } from '@/stores/console'
import { deleteAnalysis } from '@/api/analysis'
import type { Analysis } from '@/api/analysis'

const consoleStore = useConsoleStore()

const analyzing = ref(false)
const showAnalyzeDialog = ref(false)
const analyzeQuestion = ref('')

const hasResults = computed(() => consoleStore.executionResults.length > 0)
const currentAnalysis = computed(() => consoleStore.currentAnalysis)
const analysisHistory = computed(() => consoleStore.analysisHistory)

// 格式化分析结果为HTML（简单的换行处理）
const formattedAnalysis = computed(() => {
  if (!currentAnalysis.value?.analysis_result) {
    return ''
  }
  return currentAnalysis.value.analysis_result
    .replace(/\n/g, '<br>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
})

const handleAnalyze = () => {
  if (!hasResults.value) {
    ElMessage.warning('没有可分析的结果')
    return
  }
  showAnalyzeDialog.value = true
}

const confirmAnalyze = async () => {
  analyzing.value = true
  showAnalyzeDialog.value = false

  try {
    const response = await consoleStore.analyzeExecutionResults(
      analyzeQuestion.value || undefined
    )

    // 设置当前分析结果
    const analysis: Analysis = {
      id: response.id,
      analysis_result: response.analysis_result,
      question: response.question,
      created_at: response.created_at,
      results_data: '',
      session_id: ''
    }
    consoleStore.setCurrentAnalysis(analysis)

    ElMessage.success('分析完成')
    analyzeQuestion.value = ''
  } catch (error: any) {
    ElMessage.error(error.message || '分析失败')
  } finally {
    analyzing.value = false
  }
}

const handleCopyAnalysis = () => {
  if (!currentAnalysis.value?.analysis_result) {
    return
  }

  navigator.clipboard.writeText(currentAnalysis.value.analysis_result).then(() => {
    ElMessage.success('已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const handleSaveAnalysis = () => {
  ElMessage.success('分析结果已自动保存到历史记录')
}

const loadHistory = async () => {
  await consoleStore.loadAnalysisHistory()
}

const handleViewHistory = (item: Analysis) => {
  consoleStore.setCurrentAnalysis(item)
}

const handleDeleteHistory = async (id: string) => {
  try {
    await deleteAnalysis(id)
    await loadHistory()
    if (currentAnalysis.value?.id === id) {
      consoleStore.setCurrentAnalysis(null)
    }
    ElMessage.success('删除成功')
  } catch (error: any) {
    ElMessage.error(error.message || '删除失败')
  }
}

const formatTime = (time: string) => {
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadHistory()
})
</script>

<style scoped>
.ai-analysis {
  padding: 20px;
}

.analysis-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 15px;
  border-bottom: 1px solid var(--el-border-color);
}

.analysis-header h3 {
  margin: 0;
}

.analyzing-tip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  margin-bottom: 20px;
}

.analysis-result {
  margin-bottom: 30px;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.result-header h4 {
  margin: 0;
}

.result-content {
  background: var(--el-fill-color-light);
  padding: 15px;
  border-radius: 4px;
  min-height: 100px;
}

.markdown-content {
  line-height: 1.6;
  color: var(--el-text-color-primary);
}

.analysis-history {
  margin-top: 30px;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.history-header h4 {
  margin: 0;
}

.history-item {
  padding: 10px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
}

.item-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 5px;
}

.item-id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.item-question {
  color: var(--el-text-color-regular);
  font-size: 14px;
}
</style>






