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

    // 取消正在进行的请求
    cancelCurrentRequest()
    currentSessionId.value = sessionId
    thinkingSteps.value = []
    currentThinkingStatus.value = null

    try {
      await loadHistory(sessionId)
    } finally {
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
    } catch (error) {
      console.error('加载历史消息失败:', error)
      messages.value = []
    }
  }

  // 删除会话
  async function removeSession(sessionId: string) {
    try {
      await deleteSession(sessionId)
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
    console.log('[sendMessage] Called with:', content)
    if (!content.trim() || isLoading.value) {
      console.log('[sendMessage] Skipped - empty or loading', { empty: !content.trim(), loading: isLoading.value })
      return
    }

    // 检查是否正在切换会话
    if (isSwitchingSession) {
      console.log('[sendMessage] Skipped - session is switching')
      return
    }

    console.log('[sendMessage] Processing message')

    // 取消之前的请求
    cancelCurrentRequest()

    // 创建新的 AbortController
    abortController = new AbortController()

    // 添加用户消息
    const userMessage: Message = {
      role: 'user',
      content: content.trim(),
      timestamp: Date.now()
    }
    messages.value.push(userMessage)

    // 准备助手消息
    const assistantMessage: Message = {
      role: 'assistant',
      content: '',
      toolCalls: [],
      timestamp: Date.now()
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
        (chunk: string, type: StreamEventType, rawData?: any) => {
          console.log('[SSE Event]', type, rawData) // 调试日志
          const lastMessage = messages.value[messages.value.length - 1]
          if (lastMessage.role !== 'assistant') return

          switch (type) {
            case 'thinking':
              console.log('[Thinking]', rawData) // 调试日志
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
              console.log('[ThinkingSteps]', thinkingSteps.value) // 调试日志
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
    } catch (error: any) {
      // 忽略取消错误
      if (error.name === 'AbortError') {
        console.log('[sendMessage] Request was cancelled')
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
    loadSessions,
    newSession,
    switchSession,
    loadHistory,
    removeSession,
    sendMessage,
    setSelectedHosts,
    clearMessages,
    cancelCurrentRequest
  }
})
