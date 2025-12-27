import { request } from './request'

export interface Task {
  id: string
  sessionId: string
  type: 'chat' | 'script' | 'command'
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled'
  input: string
  output?: string
  error?: string
  hostId?: string
  hostName?: string
  toolName?: string
  startTime: string
  endTime?: string
  duration?: number
}

export interface TaskListResponse {
  list: Task[]
  total: number
}

// 获取任务列表
export function getTasks(params?: {
  page?: number
  pageSize?: number
  status?: string
  type?: string
  startDate?: string
  endDate?: string
}) {
  return request.get<TaskListResponse>('/tasks', { params })
}

// 获取单个任务详情
export function getTask(id: string) {
  return request.get<Task>(`/tasks/${id}`)
}

// 取消任务
export function cancelTask(id: string) {
  return request.post(`/tasks/${id}/cancel`)
}

// 重试任务
export function retryTask(id: string) {
  return request.post<Task>(`/tasks/${id}/retry`)
}

// 删除任务记录
export function deleteTask(id: string) {
  return request.delete(`/tasks/${id}`)
}

// 批量删除任务
export function batchDeleteTasks(ids: string[]) {
  return request.post('/tasks/batch-delete', { ids })
}

// 获取任务统计
export function getTaskStats() {
  return request.get<{
    total: number
    success: number
    failed: number
    running: number
  }>('/tasks/stats')
}
