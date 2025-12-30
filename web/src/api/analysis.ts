import { request } from './request'

export interface AnalyzeRequest {
  results: unknown[]
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

export interface AnalysisHistoryParams {
  session_id?: string
  limit?: number
}

// ============================================================================
// Analysis API Functions
// ============================================================================

export function analyzeResults(data: AnalyzeRequest): Promise<AnalyzeResponse> {
  return request.post<AnalyzeResponse>('/analysis/analyze', data)
}

export function getAnalysisHistory(params?: AnalysisHistoryParams): Promise<Analysis[]> {
  return request.get<Analysis[]>('/analysis/history', { params })
}

export function getAnalysis(id: string): Promise<Analysis> {
  return request.get<Analysis>(`/analysis/${id}`)
}

export function deleteAnalysis(id: string): Promise<void> {
  return request.delete(`/analysis/${id}`)
}
