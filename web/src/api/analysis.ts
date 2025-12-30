import { request } from './request'

export interface AnalyzeRequest {
  results: any[]
  question?: string
}

export interface AnalyzeResponse {
  id: string
  analysis_result: string
  question?: string
  created_at: string
}

export interface Analysis {
  id: string
  session_id?: string
  results_data: string
  analysis_result: string
  question?: string
  created_at: string
}

// AI分析批量执行结果
export function analyzeResults(data: AnalyzeRequest) {
  return request.post<AnalyzeResponse>('/analysis/analyze', data)
}

// 获取分析历史
export function getAnalysisHistory(params?: { session_id?: string; limit?: number }) {
  return request.get<Analysis[]>('/analysis/history', { params })
}

// 获取单个分析记录
export function getAnalysis(id: string) {
  return request.get<Analysis>(`/analysis/${id}`)
}

// 删除分析记录
export function deleteAnalysis(id: string) {
  return request.delete(`/analysis/${id}`)
}






