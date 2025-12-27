import { request } from './request'

export interface CommandRule {
  name: string
  pattern: string
  enabled: boolean
}

export interface CommandPolicy {
  enabled: boolean
  rules: CommandRule[]
}

export interface CommandAuditEvent {
  time: string
  session_id?: string
  tool?: string
  host: string
  command: string
  allowed: boolean
  reason?: string
  elapsed_ms?: number
  error?: string
}

export function getCommandPolicy() {
  return request.get<CommandPolicy>('/system/command-policy')
}

export function updateCommandPolicy(policy: CommandPolicy) {
  return request.put<CommandPolicy>('/system/command-policy', policy)
}

export function getCommandAudit(limit = 200) {
  return request.get<{ events: CommandAuditEvent[] }>(`/system/audit/commands?limit=${limit}`)
}
