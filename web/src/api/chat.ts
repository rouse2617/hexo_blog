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
  params?: Record<string, unknown>
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

export interface Session {
  id: string
  title: string
  createdAt: string
}

export type StreamEventType = 'content' | 'tool_call' | 'tool_result' | 'thinking' | 'done' | 'error'

interface StreamEventData {
  content?: string
  step?: number
  status?: string
  tool?: string
  params?: unknown
  result?: unknown
  error?: string
  session_id?: string
}

// ============================================================================
// API Functions
// ============================================================================

export function sendMessage(data: ChatRequest): Promise<ChatResponse> {
  return request.post<ChatResponse>('/chat/send', data)
}

export function getChatHistory(sessionId: string): Promise<Message[]> {
  return request.get<{ messages?: Message[]; list?: Message[] }>(`/chat/history/${sessionId}`)
    .then((res) => res.messages ?? res.list ?? [])
}

export function getSessions(): Promise<Session[]> {
  return request.get<{ sessions?: Session[]; list?: Session[] }>('/chat/sessions')
    .then((res) => res.sessions ?? res.list ?? [])
}

export function deleteSession(sessionId: string): Promise<void> {
  return request.delete(`/chat/sessions/${sessionId}`)
}

export function createSession(): Promise<{ sessionId: string }> {
  return request.post<{ sessionId: string }>('/chat/sessions')
}

// ============================================================================
// Streaming Chat (EventSource)
// ============================================================================

export function streamChat(
  data: ChatRequest,
  onMessage: (event: MessageEvent) => void,
  onError?: (error: Event) => void
): EventSource {
  const params = new URLSearchParams({
    message: data.message,
    sessionId: data.sessionId || '',
    hostIds: (data.hostIds || []).join(',')
  })

  const eventSource = new EventSource(`/api/chat/stream?${params}`)

  eventSource.onmessage = onMessage
  eventSource.onerror = (error) => {
    onError?.(error)
    eventSource.close()
  }

  return eventSource
}

// ============================================================================
// Fetch-based Streaming (with better error handling)
// ============================================================================

export async function fetchStreamChat(
  data: ChatRequest,
  onChunk: (chunk: string, type: StreamEventType, rawData?: StreamEventData) => void,
  abortSignal?: AbortSignal
): Promise<void> {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
    signal: abortSignal
  })

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
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.trim()) continue

        if (line.startsWith('event:')) {
          currentEventType = line.slice(6).trim()
          continue
        }

        if (line.startsWith('data:')) {
          const dataStr = line.slice(5).trim()

          if (dataStr === '[DONE]') {
            onChunk('', 'done')
            continue
          }

          try {
            const parsed = JSON.parse(dataStr) as StreamEventData
            handleStreamEvent(parsed, currentEventType, onChunk)
            currentEventType = ''
          } catch {
            onChunk(dataStr, 'content')
          }
        }
      }
    }
  } finally {
    reader.releaseLock()
  }
}

// ============================================================================
// Stream Event Handler
// ============================================================================

function handleStreamEvent(
  parsed: StreamEventData,
  eventType: string,
  onChunk: (chunk: string, type: StreamEventType, rawData?: StreamEventData) => void
): void {
  if (eventType) {
    switch (eventType) {
      case 'thinking':
        onChunk(parsed.content || '', 'thinking', parsed)
        return
      case 'tool_call':
        onChunk(JSON.stringify(parsed), 'tool_call', parsed)
        return
      case 'tool_result':
        onChunk(JSON.stringify(parsed), 'tool_result', parsed)
        return
      case 'content':
        onChunk(parsed.content || '', 'content', parsed)
        return
      case 'error':
        onChunk(parsed.error || '', 'error', parsed)
        return
      case 'done':
        onChunk('', 'done', parsed)
        return
    }
  }

  // Auto-detect event type
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
