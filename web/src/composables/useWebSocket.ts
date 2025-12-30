import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'

export interface WebSocketOptions {
  url: string
  protocols?: string | string[]
  reconnectInterval?: number
  maxReconnectInterval?: number
  heartbeatInterval?: number
  heartbeatMessage?: string
  reconnectDecay?: number
  maxReconnectAttempts?: number
}

export interface WebSocketMessage {
  type: string
  data: any
  timestamp?: number
}

const DEFAULT_OPTIONS: Required<WebSocketOptions> = {
  url: '',
  protocols: '',
  reconnectInterval: 1000,      // Start with 1s
  maxReconnectInterval: 30000,  // Cap at 30s
  heartbeatInterval: 30000,      // Send heartbeat every 30s
  heartbeatMessage: JSON.stringify({ type: 'ping' }),
  reconnectDecay: 2,             // Double each time: 1s, 2s, 4s, 8s, 16s, 30s, 30s...
  maxReconnectAttempts: Infinity
}

let ws: WebSocket | null = null
let heartbeatTimer: number | null = null
let reconnectTimer: number | null = null
let reconnectAttempts = 0
let currentReconnectInterval = DEFAULT_OPTIONS.reconnectInterval
let manualClose = false

const connectionState = ref<ConnectionState>('disconnected')
const connectionError = ref<Event | null>(null)
const lastConnected = ref<Date | null>(null)
const lastDisconnected = ref<Date | null>(null)

const messages = ref<WebSocketMessage[]>([])

export function useWebSocket(options: Partial<WebSocketOptions> = {}) {
  const opts = { ...DEFAULT_OPTIONS, ...options }

  const isConnected = computed(() => connectionState.value === 'connected')
  const isConnecting = computed(() => connectionState.value === 'connecting')
  const isDisconnected = computed(() => connectionState.value === 'disconnected')
  const isError = computed(() => connectionState.value === 'error')

  const connectionUptime = computed(() => {
    if (!lastConnected.value || connectionState.value !== 'connected') {
      return 0
    }
    return Date.now() - lastConnected.value.getTime()
  })

  const statusText = computed(() => {
    switch (connectionState.value) {
      case 'connected':
        return 'Connected'
      case 'connecting':
        return 'Connecting...'
      case 'disconnected':
        return 'Disconnected'
      case 'error':
        return 'Connection Error'
      default:
        return 'Unknown'
    }
  })

  function setState(state: ConnectionState, error?: Event) {
    connectionState.value = state
    if (error) {
      connectionError.value = error
    }
  }

  function clearHeartbeat() {
    if (heartbeatTimer !== null) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  function clearReconnect() {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function startHeartbeat() {
    clearHeartbeat()

    heartbeatTimer = window.setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        try {
          ws.send(opts.heartbeatMessage)
        } catch (error) {
          console.error('Heartbeat failed:', error)
          close()
        }
      }
    }, opts.heartbeatInterval)
  }

  function scheduleReconnect() {
    if (manualClose) return
    if (reconnectAttempts >= opts.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      ElMessage.error('Connection failed. Please refresh the page.')
      return
    }

    clearReconnect()

    reconnectTimer = window.setTimeout(() => {
      reconnectAttempts++
      console.log(`Reconnection attempt ${reconnectAttempts}, delay: ${currentReconnectInterval}ms`)
      setState('connecting')
      connect()

      // Exponential backoff: double the interval each time, up to max
      currentReconnectInterval = Math.min(
        currentReconnectInterval * opts.reconnectDecay,
        opts.maxReconnectInterval
      )
    }, currentReconnectInterval)
  }

  function connect() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      console.log('WebSocket already connected or connecting')
      return
    }

    try {
      setState('connecting')
      ws = new WebSocket(opts.url, opts.protocols)

      ws.onopen = () => {
        console.log('WebSocket connected')
        setState('connected')
        lastConnected.value = new Date()
        connectionError.value = null
        reconnectAttempts = 0
        currentReconnectInterval = opts.reconnectInterval
        startHeartbeat()
        ElMessage.success('Connected to server')
      }

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          messages.value.push(message)

          // Handle pong response
          if (message.type === 'pong') {
            // Heartbeat acknowledged
            return
          }

          // Emit custom event for components to listen
          window.dispatchEvent(new CustomEvent('websocket-message', { detail: message }))
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      ws.onerror = (event) => {
        console.error('WebSocket error:', event)
        setState('error', event)
        connectionError.value = event
      }

      ws.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason)
        clearHeartbeat()
        lastDisconnected.value = new Date()

        if (!manualClose) {
          setState('disconnected')
          scheduleReconnect()
        } else {
          setState('disconnected')
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      setState('error')
      connectionError.value = error as Event
      scheduleReconnect()
    }
  }

  function disconnect() {
    manualClose = true
    clearHeartbeat()
    clearReconnect()

    if (ws) {
      ws.close(1000, 'User disconnected')
      ws = null
    }

    setState('disconnected')
  }

  function send(message: WebSocketMessage | string): boolean {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.warn('Cannot send message: WebSocket not connected')
      return false
    }

    try {
      const data = typeof message === 'string' ? message : JSON.stringify(message)
      ws.send(data)
      return true
    } catch (error) {
      console.error('Failed to send WebSocket message:', error)
      return false
    }
  }

  function reconnect() {
    manualClose = false
    reconnectAttempts = 0
    currentReconnectInterval = opts.reconnectInterval

    if (ws) {
      ws.close()
    }

    connect()
  }

  function addMessageListener(callback: (message: WebSocketMessage) => void) {
    const handler = (event: CustomEvent) => callback(event.detail)
    window.addEventListener('websocket-message', handler as EventListener)

    return () => {
      window.removeEventListener('websocket-message', handler as EventListener)
    }
  }

  function getStatusDetails() {
    return {
      state: connectionState.value,
      statusText: statusText.value,
      isConnected: isConnected.value,
      isConnecting: isConnecting.value,
      isDisconnected: isDisconnected.value,
      isError: isError.value,
      lastConnected: lastConnected.value,
      lastDisconnected: lastDisconnected.value,
      connectionUptime: connectionUptime.value,
      reconnectAttempts: reconnectAttempts,
      nextReconnectIn: reconnectTimer ? currentReconnectInterval : 0,
      error: connectionError.value
    }
  }

  onMounted(() => {
    if (opts.url) {
      connect()
    }
  })

  onUnmounted(() => {
    disconnect()
  })

  return {
    // State
    connectionState,
    connectionError,
    messages,
    isConnected,
    isConnecting,
    isDisconnected,
    isError,
    statusText,
    connectionUptime,
    lastConnected,
    lastDisconnected,

    // Methods
    connect,
    disconnect,
    send,
    reconnect,
    addMessageListener,
    getStatusDetails
  }
}
