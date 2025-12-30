import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElNotification } from 'element-plus'
import type { Message, ToolCall, ThinkingStatus, StreamEventType } from '@/api/chat'
import { fetchStreamChat, getChatHistory, getSessions, createSession, deleteSession } from '@/api/chat'

export interface ChatSession {
  id: string
  title: string
  createdAt: string
}

// 智能推荐接口
export interface Suggestion {
  title: string
  description: string
  prompt: string
  icon: unknown
  category: 'monitor' | 'log' | 'analysis' | 'command' | 'troubleshoot'
}

// 思考过程步骤
export interface ThinkingStep {
  step: number
  status: 'calling_llm' | 'executing_tools' | 'analyzing_results'
  content: string
  timestamp: number
  toolCalls?: {
    id: string
    tool: string
    params: string
    status: 'pending' | 'running' | 'success' | 'error'
    result?: string
    error?: string
  }[]
}

/**
 * 会话消息缓存
 */
const MESSAGE_CACHE_PREFIX = 'chat-messages-'
const CACHE_DURATION = 30 * 60 * 1000 // 30分钟

/**
 * 从 localStorage 读取消息缓存
 */
function loadMessageCache(sessionId: string): Message[] | null {
  try {
    const key = MESSAGE_CACHE_PREFIX + sessionId
    const cached = localStorage.getItem(key)
    if (!cached) return null

    const data = JSON.parse(cached) as { messages: Message[]; timestamp: number }
    const now = Date.now()

    // 检查缓存是否过期
    if (now - data.timestamp > CACHE_DURATION) {
      localStorage.removeItem(key)
      return null
    }

    return data.messages
  } catch (error) {
    console.error('[MessageCache] Failed to load cache:', error)
    return null
  }
}

/**
 * 保存消息到 localStorage
 */
function saveMessageCache(sessionId: string, messages: Message[]): void {
  try {
    const key = MESSAGE_CACHE_PREFIX + sessionId
    const data = {
      messages,
      timestamp: Date.now()
    }
    localStorage.setItem(key, JSON.stringify(data))
  } catch (error) {
    console.error('[MessageCache] Failed to save cache:', error)
  }
}

/**
 * 清除消息缓存
 */
function clearMessageCache(sessionId?: string): void {
  try {
    if (sessionId) {
      localStorage.removeItem(MESSAGE_CACHE_PREFIX + sessionId)
    } else {
      // 清除所有消息缓存
      const keys = Object.keys(localStorage)
      keys.forEach(key => {
        if (key.startsWith(MESSAGE_CACHE_PREFIX)) {
          localStorage.removeItem(key)
        }
      })
    }
  } catch (error) {
    console.error('[MessageCache] Failed to clear cache:', error)
  }
}

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const currentSessionId = ref<string>('')
  const messages = ref<Message[]>([])
  const isLoading = ref(false)
  const currentToolCalls = ref<ToolCall[]>([])
  const selectedHostIds = ref<string[]>([])

  // 思考过程状态
  const thinkingSteps = ref<ThinkingStep[]>([])
  const currentThinkingStatus = ref<ThinkingStatus | null>(null)

  // 智能推荐状态
  const suggestions = ref<Suggestion[]>([])
  const showSuggestions = ref(true)

  // 会话切换加载状态
  const sessionLoading = ref(false)

  // 请求取消控制器
  let abortController: AbortController | null = null

  // 会话切换锁 - 防止切换时发送消息
  let isSwitchingSession = false

  // 加载会话列表
  async function loadSessions() {
    try {
      const data = await getSessions()
      sessions.value = data
      if (data.length > 0 && !currentSessionId.value) {
        currentSessionId.value = data[0].id
        await loadHistory(data[0].id)
      }
    } catch (error) {
      console.error('加载会话列表失败:', error)
    }
  }

  // 创建新会话
  async function newSession() {
    try {
      const { sessionId } = await createSession()
      currentSessionId.value = sessionId
      messages.value = []
      currentToolCalls.value = []
      thinkingSteps.value = []
      currentThinkingStatus.value = null
      await loadSessions()
      return sessionId
    } catch (error) {
      console.error('创建会话失败:', error)
      // 本地创建临时会话
      const tempId = `temp-${Date.now()}`
      currentSessionId.value = tempId
      messages.value = []
      currentToolCalls.value = []
      thinkingSteps.value = []
      currentThinkingStatus.value = null
      return tempId
    }
  }

  // 切换会话
  async function switchSession(sessionId: string) {
    // 设置切换锁
    isSwitchingSession = true
    // 设置加载状态
    sessionLoading.value = true

    // 取消正在进行的请求
    cancelCurrentRequest()
    currentSessionId.value = sessionId
    thinkingSteps.value = []
    currentThinkingStatus.value = null

    try {
      await loadHistory(sessionId)
    } finally {
      // 关闭加载状态
      sessionLoading.value = false
      // 等待下一个 tick 后释放锁，确保状态已更新
      setTimeout(() => {
        isSwitchingSession = false
      }, 100)
    }
  }

  // 加载历史消息
  async function loadHistory(sessionId: string) {
    try {
      const data = await getChatHistory(sessionId)
      messages.value = data
      // 缓存消息
      saveMessageCache(sessionId, data)
    } catch (error) {
      console.error('加载历史消息失败:', error)
      // 尝试从缓存加载
      const cached = loadMessageCache(sessionId)
      if (cached) {
        messages.value = cached
      } else {
        messages.value = []
      }
    }
  }

  // 删除会话
  async function removeSession(sessionId: string) {
    try {
      await deleteSession(sessionId)
      // 清除消息缓存
      clearMessageCache(sessionId)
      await loadSessions()
      if (currentSessionId.value === sessionId) {
        if (sessions.value.length > 0) {
          await switchSession(sessions.value[0].id)
        } else {
          await newSession()
        }
      }
    } catch (error) {
      console.error('删除会话失败:', error)
    }
  }

  // 发送消息（流式）
  async function sendMessage(content: string) {
    if (!content.trim() || isLoading.value) {
      return
    }

    // 检查是否正在切换会话
    if (isSwitchingSession) {
      return
    }


    // 取消之前的请求
    cancelCurrentRequest()

    // 创建新的 AbortController
    abortController = new AbortController()

    // 添加用户消息（使用递增时间戳确保唯一性）
    const baseTimestamp = Date.now()
    const userMessage: Message = {
      role: 'user',
      content: content.trim(),
      timestamp: baseTimestamp
    }
    messages.value.push(userMessage)

    // 准备助手消息（加 1ms 确保时间戳唯一）
    const assistantMessage: Message = {
      role: 'assistant',
      content: '',
      toolCalls: [],
      timestamp: baseTimestamp + 1
    }
    messages.value.push(assistantMessage)

    isLoading.value = true
    currentToolCalls.value = []
    thinkingSteps.value = []
    currentThinkingStatus.value = null

    // 当前步骤的工具调用
    let currentStepToolCalls: ThinkingStep['toolCalls'] = []

    try {
      await fetchStreamChat(
        {
          message: content,
          sessionId: currentSessionId.value,
          hostIds: selectedHostIds.value
        },
        (chunk: string, type: StreamEventType, rawData?: unknown) => {
          const lastMessage = messages.value[messages.value.length - 1]
          if (lastMessage.role !== 'assistant') return

          switch (type) {
            case 'thinking':
              // 更新思考状态
              currentThinkingStatus.value = {
                step: rawData.step,
                status: rawData.status,
                content: rawData.content
              }

              // 如果是新步骤，添加到步骤列表
              const existingStep = thinkingSteps.value.find(s => s.step === rawData.step)
              if (!existingStep) {
                currentStepToolCalls = []
                thinkingSteps.value.push({
                  step: rawData.step,
                  status: rawData.status,
                  content: rawData.content,
                  timestamp: Date.now(),
                  toolCalls: currentStepToolCalls
                })
              } else {
                // 更新现有步骤
                existingStep.status = rawData.status
                existingStep.content = rawData.content
              }
              break

            case 'content':
              lastMessage.content += chunk
              // 收到内容后清除思考状态
              currentThinkingStatus.value = null
              break

            case 'tool_call':
              try {
                const toolCallData = rawData || JSON.parse(chunk)
                const toolCall: ToolCall = {
                  id: toolCallData.id || `tool-${Date.now()}`,
                  name: toolCallData.tool,
                  arguments: typeof toolCallData.params === 'string'
                    ? toolCallData.params
                    : JSON.stringify(toolCallData.params),
                  status: 'running'
                }
                if (!lastMessage.toolCalls) {
                  lastMessage.toolCalls = []
                }
                lastMessage.toolCalls.push(toolCall)
                currentToolCalls.value.push(toolCall)

                // 添加到当前步骤的工具调用
                const currentStep = thinkingSteps.value[thinkingSteps.value.length - 1]
                if (currentStep) {
                  if (!currentStep.toolCalls) {
                    currentStep.toolCalls = []
                  }
                  currentStep.toolCalls.push({
                    id: toolCall.id,
                    tool: toolCall.name,
                    params: toolCall.arguments,
                    status: 'running'
                  })
                }
              } catch (e) {
                console.error('解析工具调用失败:', e)
              }
              break

            case 'tool_result':
              try {
                const result = rawData || JSON.parse(chunk)
                // 更新 message 中的工具调用状态（优先使用ID匹配，如果没有ID则使用名称）
                const toolCall = result.id
                  ? currentToolCalls.value.find(t => t.id === result.id)
                  : currentToolCalls.value.find(t => t.name === result.tool)
                if (toolCall) {
                  toolCall.result = typeof result.result === 'string'
                    ? result.result
                    : JSON.stringify(result.result)
                  toolCall.status = result.error ? 'error' : 'success'
                }

                // 更新思考步骤中的工具调用状态（优先使用ID匹配）
                for (const step of thinkingSteps.value) {
                  const stepToolCall = result.id
                    ? step.toolCalls?.find(t => t.id === result.id)
                    : step.toolCalls?.find(t => t.tool === result.tool)
                  if (stepToolCall) {
                    stepToolCall.result = typeof result.result === 'string'
                      ? result.result
                      : JSON.stringify(result.result)
                    stepToolCall.error = result.error
                    stepToolCall.status = result.error ? 'error' : 'success'
                  }
                }
              } catch (e) {
                console.error('解析工具结果失败:', e)
              }
              break

            case 'done':
              isLoading.value = false
              currentThinkingStatus.value = null
              break

            case 'error':
              lastMessage.content += '\n\n[错误] ' + chunk
              isLoading.value = false
              currentThinkingStatus.value = null
              break
          }
        },
        abortController.signal
      )
    } catch (error: unknown) {
      // 忽略取消错误
      if (error.name === 'AbortError') {
        // 显示取消通知
        ElNotification({
          title: '请求已取消',
          message: '当前请求已被取消',
          type: 'info',
          duration: 2000
        })
        return
      }
      console.error('发送消息失败:', error)
      const lastMessage = messages.value[messages.value.length - 1]
      if (lastMessage.role === 'assistant') {
        lastMessage.content = '抱歉，发送消息时出现错误，请稍后重试。'
        ElNotification({
          title: '发送失败',
          message: '发送消息时出现错误，请稍后重试',
          type: 'error',
          duration: 3000
        })
      }
    } finally {
      isLoading.value = false
      currentThinkingStatus.value = null
      abortController = null
    }
  }

  // 取消当前请求
  function cancelCurrentRequest() {
    if (abortController) {
      abortController.abort()
      abortController = null
      isLoading.value = false
      currentThinkingStatus.value = null
    }
  }

  // 设置选中的主机
  function setSelectedHosts(hostIds: string[]) {
    selectedHostIds.value = hostIds
  }

  // 清空当前会话消息
  function clearMessages() {
    messages.value = []
    currentToolCalls.value = []
    thinkingSteps.value = []
    currentThinkingStatus.value = null
    showSuggestions.value = true
  }

  // 获取智能推荐（基于上下文）
  async function getSuggestions(context: { lastTool?: string; hasError?: boolean; selectedHosts?: string[] }): Promise<Suggestion[]> {
    // 这里可以根据上下文动态生成推荐
    // 目前使用预设的推荐列表
    const baseSuggestions: Suggestion[] = [
      {
        title: '系统健康检查',
        description: '查看 CPU、内存、磁盘使用情况',
        prompt: '帮我检查一下服务器的整体健康状况',
        icon: null, // 会在组件中设置
        category: 'monitor'
      },
      {
        title: '查看最近错误日志',
        description: '分析最近的系统错误和异常',
        prompt: '查看最近 1 小时内的错误日志',
        icon: null,
        category: 'log'
      },
      {
        title: '进程资源占用',
        description: '查看占用资源最多的进程',
        prompt: '列出占用 CPU 和内存最多的前 10 个进程',
        icon: null,
        category: 'analysis'
      }
    ]

    // 如果刚刚执行了命令，推荐后续分析
    if (context.lastTool) {
      baseSuggestions.unshift({
        title: '深入分析结果',
        description: '对刚才的结果进行详细分析',
        prompt: '对上面的结果进行分析，找出潜在问题',
        icon: null,
        category: 'analysis'
      })
    }

    // 如果有错误，推荐故障排查
    if (context.hasError) {
      baseSuggestions.unshift({
        title: '故障排查',
        description: '诊断可能的问题原因',
        prompt: '根据上面的错误信息，帮我分析可能的原因和解决方案',
        icon: null,
        category: 'troubleshoot'
      })
    }

    suggestions.value = baseSuggestions.slice(0, 6)
    return suggestions.value
  }

  // 选择智能推荐
  function selectSuggestion(suggestion: Suggestion) {
    sendMessage(suggestion.prompt)
    showSuggestions.value = false
  }

  // 切换推荐显示状态
  function toggleSuggestions(show?: boolean) {
    showSuggestions.value = show !== undefined ? show : !showSuggestions.value
  }

  return {
    sessions,
    currentSessionId,
    messages,
    isLoading,
    currentToolCalls,
    selectedHostIds,
    thinkingSteps,
    currentThinkingStatus,
    suggestions,
    showSuggestions,
    sessionLoading,
    loadSessions,
    newSession,
    switchSession,
    loadHistory,
    removeSession,
    sendMessage,
    setSelectedHosts,
    clearMessages,
    cancelCurrentRequest,
    clearMessageCache,
    getSuggestions,
    selectSuggestion,
    toggleSuggestions
  }
}, {
  persist: {
    key: 'ai-pro-chat',
    paths: ['sessions', 'currentSessionId', 'selectedHostIds']
  }
})
