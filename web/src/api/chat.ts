import { request } from './request'

export interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
  toolCalls?: ToolCall[]
  timestamp?: string | number
}

export interface ToolCall {
  id: string
  name: string
  arguments: string
  params?: Record<string, any>
  result?: string
  error?: string
  status?: 'pending' | 'running' | 'success' | 'error'
}

export interface ThinkingStatus {
  step: number
  status: 'calling_llm' | 'executing_tools' | 'analyzing_results'
  content: string
}

export interface ChatRequest {
  message: string
  sessionId?: string
  hostIds?: string[]
}

export interface ChatResponse {
  sessionId: string
  message: Message
}

// 流式事件类型
export type StreamEventType = 'content' | 'tool_call' | 'tool_result' | 'thinking' | 'done' | 'error'

// 发送消息（非流式）
export function sendMessage(data: ChatRequest) {
  return request.post<ChatResponse>('/chat/send', data)
}

// 获取会话历史
export function getChatHistory(sessionId: string) {
  return request.get<any>(`/chat/history/${sessionId}`).then((res) => {
    if (Array.isArray(res)) return res as Message[]
    return (res?.messages ?? []) as Message[]
  })
}

// 获取所有会话列表
export function getSessions() {
  return request.get<any>('/chat/sessions').then((res) => {
    if (Array.isArray(res)) return res as { id: string; title: string; createdAt: string }[]
    return (res?.sessions ?? []) as { id: string; title: string; createdAt: string }[]
  })
}

// 删除会话
export function deleteSession(sessionId: string) {
  return request.delete(`/chat/sessions/${sessionId}`)
}

// 创建新会话
export function createSession() {
  return request.post<{ sessionId: string }>('/chat/sessions')
}

// SSE 流式对话
export function streamChat(data: ChatRequest, onMessage: (event: MessageEvent) => void, onError?: (error: Event) => void) {
  const eventSource = new EventSource(`/api/chat/stream?message=${encodeURIComponent(data.message)}&sessionId=${data.sessionId || ''}&hostIds=${(data.hostIds || []).join(',')}`)

  eventSource.onmessage = onMessage
  eventSource.onerror = (error) => {
    if (onError) {
      onError(error)
    }
    eventSource.close()
  }

  return eventSource
}

// 使用 fetch 实现 SSE 流式对话（更灵活，支持取消）
export async function fetchStreamChat(
  data: ChatRequest,
  onChunk: (chunk: string, type: StreamEventType, rawData?: any) => void,
  abortSignal?: AbortSignal
) {
  console.log('[fetchStreamChat] Starting request', data)
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(data),
    signal: abortSignal
  })

  console.log('[fetchStreamChat] Response status:', response.status)

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('No reader available')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let currentEventType = ''

  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) {
        console.log('[fetchStreamChat] Stream done')
        break
      }

      const chunk = decoder.decode(value, { stream: true })
      console.log('[fetchStreamChat] Raw chunk:', chunk)
      buffer += chunk
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        console.log('[fetchStreamChat] Processing line:', line)
        // 解析 SSE 事件格式
        if (line.startsWith('event:')) {
          // 事件类型行
          currentEventType = line.slice(6).trim()
          console.log('[fetchStreamChat] Event type:', currentEventType)
          continue
        }
        if (line.startsWith('data:')) {
          const dataStr = line.slice(5).trim()
          console.log('[fetchStreamChat] Data:', dataStr, 'EventType:', currentEventType)
          if (dataStr === '[DONE]') {
            onChunk('', 'done')
            continue
          }
          try {
            const parsed = JSON.parse(dataStr)

            // 根据事件类型处理
            if (currentEventType === 'thinking') {
              onChunk(parsed.content || '', 'thinking', parsed)
            } else if (currentEventType === 'tool_call') {
              onChunk(JSON.stringify(parsed), 'tool_call', parsed)
            } else if (currentEventType === 'tool_result') {
              onChunk(JSON.stringify(parsed), 'tool_result', parsed)
            } else if (currentEventType === 'content') {
              onChunk(parsed.content || '', 'content', parsed)
            } else if (currentEventType === 'error') {
              onChunk(parsed.error || '', 'error', parsed)
            } else if (currentEventType === 'done') {
              onChunk('', 'done', parsed)
            } else {
              // 兼容没有 event 行的情况，根据数据内容判断
              if (parsed.step !== undefined && parsed.status) {
                onChunk(parsed.content || '', 'thinking', parsed)
              } else if (parsed.tool && parsed.result !== undefined) {
                onChunk(JSON.stringify(parsed), 'tool_result', parsed)
              } else if (parsed.tool && parsed.params !== undefined) {
                onChunk(JSON.stringify(parsed), 'tool_call', parsed)
              } else if (parsed.content !== undefined) {
                onChunk(parsed.content || '', 'content', parsed)
              } else if (parsed.error) {
                onChunk(parsed.error, 'error', parsed)
              } else if (parsed.session_id !== undefined) {
                onChunk('', 'done', parsed)
              }
            }
            // 重置事件类型
            currentEventType = ''
          } catch (e) {
            console.error('[fetchStreamChat] Parse error:', e)
            onChunk(dataStr, 'content')
          }
        }
      }
    }
  } finally {
    // 确保 reader 被释放
    reader.releaseLock()
  }
}
