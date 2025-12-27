import { request } from './request'

export interface Host {
  id: string
  name: string
  host: string
  port: number
  username: string
  password?: string
  privateKey?: string
  description?: string
  tags?: string[]
  status?: 'online' | 'offline' | 'unknown'
  createdAt?: string
  updatedAt?: string
}

export interface HostListResponse {
  list: Host[]
  total: number
}

// 获取主机列表
export function getHosts(params?: { page?: number; pageSize?: number; keyword?: string }) {
  return request.get<HostListResponse>('/hosts', { params })
}

// 获取单个主机
export function getHost(id: string) {
  return request.get<Host>(`/hosts/${id}`)
}

// 创建主机
export function createHost(data: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>) {
  return request.post<Host>('/hosts', data)
}

// 更新主机
export function updateHost(id: string, data: Partial<Host>) {
  return request.put<Host>(`/hosts/${id}`, data)
}

// 删除主机
export function deleteHost(id: string) {
  return request.delete(`/hosts/${id}`)
}

// 测试主机连接
export function testHostConnection(id: string) {
  return request.post<{ success: boolean; message: string }>(`/hosts/${id}/test`)
}

// 批量导入主机
export function importHosts(data: { hosts: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>[] }) {
  return request.post<{ success: number; failed: number; errors?: string[] }>('/hosts/import', data)
}

// 批量删除主机
export function batchDeleteHosts(ids: string[]) {
  return request.post('/hosts/batch-delete', { ids })
}

// 获取所有主机（不分页，用于选择器）
export function getAllHosts() {
  return request.get<Host[]>('/hosts/all')
}
