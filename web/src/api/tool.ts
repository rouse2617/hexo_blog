import { request } from './request'

export interface Tool {
  name: string
  description: string
  type: 'builtin' | 'script'
  parameters?: ToolParameter[]
  enabled: boolean
  scriptId?: string
}

export interface ToolParameter {
  name: string
  type: string
  description: string
  required: boolean
  default?: any
}

// 获取所有工具列表
export function getTools() {
  return request.get<any>('/tools').then((res) => {
    if (Array.isArray(res)) return res as Tool[]
    return (res?.tools ?? []) as Tool[]
  })
}

// 获取内置工具列表
export function getBuiltinTools() {
  return request.get<Tool[]>('/tools/builtin')
}

// 获取脚本工具列表
export function getScriptTools() {
  return request.get<Tool[]>('/tools/script')
}

// 启用/禁用工具
export function toggleTool(name: string, enabled: boolean) {
  return request.put(`/tools/${name}/toggle`, { enabled })
}

// 获取工具详情
export function getToolDetail(name: string) {
  return request.get<Tool>(`/tools/${name}`)
}
