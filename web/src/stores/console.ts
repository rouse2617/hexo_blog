import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { BatchExecuteResult } from '@/api/operations'
import type { Analysis } from '@/api/analysis'
import { batchExecute } from '@/api/operations'
import { analyzeResults, getAnalysisHistory } from '@/api/analysis'

export const useConsoleStore = defineStore('console', () => {
  // 选中的主机列表
  const selectedHosts = ref<string[]>([])
  
  // 执行结果
  const executionResults = ref<BatchExecuteResult[]>([])
  
  // 执行状态
  const executing = ref(false)
  
  // 当前操作类型
  const currentOperation = ref<'query_log' | 'run_command' | 'check_cpu' | 'check_memory' | 'check_disk' | 'check_process' | 'check_filesystem' | 'check_raid' | 'check_lvm' | 'check_io' | null>(null)
  
  // 当前操作参数
  const operationParams = ref<Record<string, any>>({})
  
  // AI分析历史
  const analysisHistory = ref<Analysis[]>([])
  
  // 当前分析结果
  const currentAnalysis = ref<Analysis | null>(null)
  
  // 分析中
  const analyzing = ref(false)

  // 选择主机
  function selectHosts(hosts: string[]) {
    selectedHosts.value = hosts
  }

  // 清空选择
  function clearSelection() {
    selectedHosts.value = []
  }

  // 执行批量操作
  async function executeOperation(
    operation: 'query_log' | 'run_command' | 'check_cpu' | 'check_memory' | 'check_disk' | 'check_process' | 'check_filesystem' | 'check_raid' | 'check_lvm' | 'check_io',
    hosts: string[],
    params?: Record<string, any>
  ) {
    if (hosts.length === 0) {
      throw new Error('请至少选择一个主机')
    }

    executing.value = true
    currentOperation.value = operation
    operationParams.value = params || {}

    try {
      const response = await batchExecute({
        operation,
        hosts,
        params
      })

      executionResults.value = response || []
      return response || []
    } catch (error) {
      console.error('批量执行失败:', error)
      throw error
    } finally {
      executing.value = false
    }
  }

  // 清空结果
  function clearResults() {
    executionResults.value = []
    currentOperation.value = null
    operationParams.value = {}
  }

  // AI分析结果
  async function analyzeExecutionResults(question?: string) {
    if (executionResults.value.length === 0) {
      throw new Error('没有可分析的结果')
    }

    analyzing.value = true
    try {
      const response = await analyzeResults({
        results: executionResults.value.map(r => ({
          host: r.host,
          status: r.status,
          result: r.result,
          error: r.error,
          elapsed: r.elapsed
        })),
        question
      })

      // 加载分析历史
      await loadAnalysisHistory()

      return response
    } catch (error) {
      console.error('AI分析失败:', error)
      throw error
    } finally {
      analyzing.value = false
    }
  }

  // 加载分析历史
  async function loadAnalysisHistory(limit = 20) {
    try {
      const history = await getAnalysisHistory({ limit })
      analysisHistory.value = history || []
    } catch (error) {
      console.error('加载分析历史失败:', error)
    }
  }

  // 设置当前分析
  function setCurrentAnalysis(analysis: Analysis | null) {
    currentAnalysis.value = analysis
  }

  return {
    selectedHosts,
    executionResults,
    executing,
    currentOperation,
    operationParams,
    analysisHistory,
    currentAnalysis,
    analyzing,
    selectHosts,
    clearSelection,
    executeOperation,
    clearResults,
    analyzeExecutionResults,
    loadAnalysisHistory,
    setCurrentAnalysis
  }
}, {
  persist: {
    key: 'ai-pro-console',
    paths: ['selectedHosts']
  }
})


