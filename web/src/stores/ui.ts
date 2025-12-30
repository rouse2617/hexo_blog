/**
 * UI Store - Global UI State Management
 * 
 * Manages global UI state including theme, connection status, modals, and notifications.
 * 
 * @module stores/ui
 * @requirements 13.5, 14.4, 15.2
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type {
  ThemeMode,
  ConnectionStatus,
  ConnectionMetrics,
  ModalType,
  ConfirmationRequest,
  Notification,
  ShortcutConfig,
  UIState,
  LayoutType
} from '@/types/chat-ui'

/** Default connection metrics */
const DEFAULT_CONNECTION_METRICS: ConnectionMetrics = {
  latency: 0,
  queueSize: 0,
  reconnectAttempts: 0,
  messagesReceived: 0,
  messagesSent: 0
}

/** Breakpoints for responsive layout */
const BREAKPOINTS = {
  mobile: 768,
  desktop: 1200
}

export const useUIStore = defineStore('ui', () => {
  // ============================================================================
  // Theme State (Req 13.5)
  // ============================================================================
  
  const theme = ref<ThemeMode>('light')
  const resolvedTheme = ref<'light' | 'dark'>('light')

  // ============================================================================
  // Layout State
  // ============================================================================
  
  const layout = ref<LayoutType>('three-column')
  const leftPanelVisible = ref(true)
  const rightPanelVisible = ref(true)
  const historyPanelOverlayVisible = ref(false)

  // ============================================================================
  // Connection State (Req 15.2)
  // ============================================================================
  
  const connectionStatus = ref<ConnectionStatus>('disconnected')
  const connectionMetrics = ref<ConnectionMetrics>({ ...DEFAULT_CONNECTION_METRICS })

  // ============================================================================
  // Modal State
  // ============================================================================
  
  const activeModal = ref<ModalType | null>(null)
  const modalStack = ref<ModalType[]>([])
  const shortcutHelpVisible = ref(false)

  // ============================================================================
  // Confirmation State
  // ============================================================================
  
  const pendingConfirmations = ref<ConfirmationRequest[]>([])

  // ============================================================================
  // Notification State (Req 14.4)
  // ============================================================================
  
  const notifications = ref<Notification[]>([])
  let notificationIdCounter = 0

  // ============================================================================
  // Shortcut State
  // ============================================================================
  
  const customShortcuts = ref<Partial<ShortcutConfig>>({})

  // ============================================================================
  // Computed Properties
  // ============================================================================
  
  const isConnected = computed(() => connectionStatus.value === 'connected')
  const hasActiveModal = computed(() => activeModal.value !== null)
  const hasPendingConfirmations = computed(() => pendingConfirmations.value.length > 0)
  const isMobileLayout = computed(() => layout.value === 'single-column')
  const isTabletLayout = computed(() => layout.value === 'two-column')
  const isDesktopLayout = computed(() => layout.value === 'three-column')

  // ============================================================================
  // Theme Actions (Req 13.5)
  // ============================================================================
  
  /**
   * Set the theme mode
   * @param newTheme - The theme mode to set
   */
  function setTheme(newTheme: ThemeMode) {
    theme.value = newTheme
    updateResolvedTheme()
    applyTheme()
    persistTheme()
  }

  /**
   * Toggle between light and dark themes
   */
  function toggleTheme() {
    if (theme.value === 'light') {
      setTheme('dark')
    } else if (theme.value === 'dark') {
      setTheme('light')
    } else {
      // If auto, switch to the opposite of current resolved theme
      setTheme(resolvedTheme.value === 'light' ? 'dark' : 'light')
    }
  }

  /**
   * Update the resolved theme based on current mode and system preference
   */
  function updateResolvedTheme() {
    if (theme.value === 'auto') {
      resolvedTheme.value = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    } else {
      resolvedTheme.value = theme.value
    }
  }

  /**
   * Apply the current theme to the document
   */
  function applyTheme() {
    const root = document.documentElement
    root.setAttribute('data-theme', resolvedTheme.value)
    
    // Apply CSS variables based on theme
    if (resolvedTheme.value === 'dark') {
      root.style.setProperty('--color-background', '#1a1a1a')
      root.style.setProperty('--color-surface', '#2d2d2d')
      root.style.setProperty('--color-text', '#e2e8f0')
      root.style.setProperty('--color-text-secondary', '#94a3b8')
      root.style.setProperty('--color-border', '#404040')
    } else {
      root.style.setProperty('--color-background', '#ffffff')
      root.style.setProperty('--color-surface', '#f8fafc')
      root.style.setProperty('--color-text', '#1e293b')
      root.style.setProperty('--color-text-secondary', '#64748b')
      root.style.setProperty('--color-border', '#e2e8f0')
    }
  }

  /**
   * Persist theme preference to localStorage
   */
  function persistTheme() {
    localStorage.setItem('ui-theme', theme.value)
  }

  /**
   * Initialize theme from localStorage and system preference
   */
  function initTheme() {
    const savedTheme = localStorage.getItem('ui-theme') as ThemeMode | null
    if (savedTheme && ['light', 'dark', 'auto'].includes(savedTheme)) {
      theme.value = savedTheme
    }
    updateResolvedTheme()
    applyTheme()

    // Listen for system theme changes when in auto mode
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'auto') {
        updateResolvedTheme()
        applyTheme()
      }
    })
  }

  // ============================================================================
  // Layout Actions
  // ============================================================================
  
  /**
   * Update layout based on window width
   * @param width - Current window width
   */
  function updateLayout(width: number) {
    if (width < BREAKPOINTS.mobile) {
      layout.value = 'single-column'
      leftPanelVisible.value = false
      rightPanelVisible.value = false
    } else if (width < BREAKPOINTS.desktop) {
      layout.value = 'two-column'
      leftPanelVisible.value = false
      rightPanelVisible.value = true
    } else {
      layout.value = 'three-column'
      leftPanelVisible.value = true
      rightPanelVisible.value = true
    }
  }

  /**
   * Toggle a specific panel
   * @param panel - The panel to toggle ('left' | 'right')
   */
  function togglePanel(panel: 'left' | 'right') {
    if (panel === 'left') {
      if (layout.value === 'single-column' || layout.value === 'two-column') {
        // On mobile/tablet, use overlay
        historyPanelOverlayVisible.value = !historyPanelOverlayVisible.value
      } else {
        leftPanelVisible.value = !leftPanelVisible.value
      }
    } else {
      rightPanelVisible.value = !rightPanelVisible.value
    }
  }

  /**
   * Set panel visibility
   * @param panel - The panel to set
   * @param visible - Whether the panel should be visible
   */
  function setPanelVisible(panel: 'left' | 'right', visible: boolean) {
    if (panel === 'left') {
      leftPanelVisible.value = visible
    } else {
      rightPanelVisible.value = visible
    }
  }

  /**
   * Toggle history panel overlay (for mobile)
   */
  function toggleHistoryOverlay() {
    historyPanelOverlayVisible.value = !historyPanelOverlayVisible.value
  }

  /**
   * Close history panel overlay
   */
  function closeHistoryOverlay() {
    historyPanelOverlayVisible.value = false
  }

  // ============================================================================
  // Connection Actions (Req 15.2)
  // ============================================================================
  
  /**
   * Set connection status
   * @param status - The new connection status
   */
  function setConnectionStatus(status: ConnectionStatus) {
    const previousStatus = connectionStatus.value
    connectionStatus.value = status

    // Update metrics
    if (status === 'connected') {
      connectionMetrics.value.lastConnected = Date.now()
      connectionMetrics.value.reconnectAttempts = 0
    } else if (status === 'disconnected') {
      connectionMetrics.value.lastDisconnected = Date.now()
    }

    // Show notification on status change
    if (previousStatus !== status) {
      if (status === 'disconnected' && previousStatus === 'connected') {
        showNotification({
          type: 'warning',
          title: '连接已断开',
          message: '正在尝试重新连接...',
          duration: 3000
        })
      } else if (status === 'connected' && previousStatus !== 'connected') {
        showNotification({
          type: 'success',
          title: '连接已恢复',
          duration: 2000
        })
      }
    }
  }

  /**
   * Update connection metrics
   * @param metrics - Partial metrics to update
   */
  function updateConnectionMetrics(metrics: Partial<ConnectionMetrics>) {
    connectionMetrics.value = { ...connectionMetrics.value, ...metrics }
  }

  /**
   * Increment reconnect attempts
   */
  function incrementReconnectAttempts() {
    connectionMetrics.value.reconnectAttempts++
  }

  // ============================================================================
  // Modal Actions
  // ============================================================================
  
  /**
   * Open a modal
   * @param modal - The modal type to open
   */
  function openModal(modal: ModalType) {
    if (activeModal.value) {
      modalStack.value.push(activeModal.value)
    }
    activeModal.value = modal
  }

  /**
   * Close the current modal
   */
  function closeModal() {
    if (modalStack.value.length > 0) {
      activeModal.value = modalStack.value.pop() || null
    } else {
      activeModal.value = null
    }
  }

  /**
   * Close all modals
   */
  function closeAllModals() {
    activeModal.value = null
    modalStack.value = []
    shortcutHelpVisible.value = false
  }

  /**
   * Toggle shortcut help modal
   */
  function toggleShortcutHelp() {
    shortcutHelpVisible.value = !shortcutHelpVisible.value
  }

  // ============================================================================
  // Confirmation Actions
  // ============================================================================
  
  /**
   * Add a confirmation request
   * @param request - The confirmation request to add
   */
  function addConfirmation(request: ConfirmationRequest) {
    pendingConfirmations.value.push(request)
  }

  /**
   * Remove a confirmation request
   * @param id - The ID of the confirmation to remove
   */
  function removeConfirmation(id: string) {
    const index = pendingConfirmations.value.findIndex(c => c.id === id)
    if (index !== -1) {
      pendingConfirmations.value.splice(index, 1)
    }
  }

  /**
   * Clear all pending confirmations
   */
  function clearConfirmations() {
    pendingConfirmations.value = []
  }

  // ============================================================================
  // Notification Actions (Req 14.4)
  // ============================================================================
  
  /**
   * Show a notification
   * @param options - Notification options
   * @returns The notification ID
   */
  function showNotification(options: Omit<Notification, 'id' | 'timestamp'>): number {
    const id = ++notificationIdCounter
    const notification: Notification = {
      ...options,
      id,
      timestamp: Date.now(),
      duration: options.duration ?? 3000
    }
    notifications.value.push(notification)

    // Auto-remove after duration
    if (notification.duration && notification.duration > 0) {
      setTimeout(() => {
        removeNotification(id)
      }, notification.duration)
    }

    return id
  }

  /**
   * Remove a notification
   * @param id - The notification ID to remove
   */
  function removeNotification(id: number) {
    const index = notifications.value.findIndex(n => n.id === id)
    if (index !== -1) {
      notifications.value.splice(index, 1)
    }
  }

  /**
   * Clear all notifications
   */
  function clearNotifications() {
    notifications.value = []
  }

  // ============================================================================
  // Shortcut Actions
  // ============================================================================
  
  /**
   * Set custom shortcuts
   * @param shortcuts - Custom shortcut configuration
   */
  function setCustomShortcuts(shortcuts: Partial<ShortcutConfig>) {
    customShortcuts.value = shortcuts
    localStorage.setItem('ui-shortcuts', JSON.stringify(shortcuts))
  }

  /**
   * Load custom shortcuts from localStorage
   */
  function loadCustomShortcuts() {
    const saved = localStorage.getItem('ui-shortcuts')
    if (saved) {
      try {
        customShortcuts.value = JSON.parse(saved)
      } catch (e) {
        console.error('Failed to load custom shortcuts:', e)
      }
    }
  }

  // ============================================================================
  // Initialization
  // ============================================================================
  
  /**
   * Initialize the UI store
   */
  function init() {
    initTheme()
    loadCustomShortcuts()
    
    // Set initial layout based on window width
    updateLayout(window.innerWidth)
    
    // Listen for window resize
    window.addEventListener('resize', () => {
      updateLayout(window.innerWidth)
    })
  }

  // ============================================================================
  // State Getter (for debugging/testing)
  // ============================================================================
  
  function getState(): UIState {
    return {
      theme: theme.value,
      resolvedTheme: resolvedTheme.value,
      shortcutHelpVisible: shortcutHelpVisible.value,
      customShortcuts: customShortcuts.value,
      connectionStatus: connectionStatus.value,
      connectionMetrics: connectionMetrics.value,
      activeModal: activeModal.value,
      modalStack: modalStack.value,
      pendingConfirmations: pendingConfirmations.value,
      notifications: notifications.value,
      historyPanelOverlayVisible: historyPanelOverlayVisible.value
    }
  }

  return {
    // State
    theme,
    resolvedTheme,
    layout,
    leftPanelVisible,
    rightPanelVisible,
    historyPanelOverlayVisible,
    connectionStatus,
    connectionMetrics,
    activeModal,
    modalStack,
    shortcutHelpVisible,
    pendingConfirmations,
    notifications,
    customShortcuts,

    // Computed
    isConnected,
    hasActiveModal,
    hasPendingConfirmations,
    isMobileLayout,
    isTabletLayout,
    isDesktopLayout,

    // Theme Actions
    setTheme,
    toggleTheme,
    initTheme,

    // Layout Actions
    updateLayout,
    togglePanel,
    setPanelVisible,
    toggleHistoryOverlay,
    closeHistoryOverlay,

    // Connection Actions
    setConnectionStatus,
    updateConnectionMetrics,
    incrementReconnectAttempts,

    // Modal Actions
    openModal,
    closeModal,
    closeAllModals,
    toggleShortcutHelp,

    // Confirmation Actions
    addConfirmation,
    removeConfirmation,
    clearConfirmations,

    // Notification Actions
    showNotification,
    removeNotification,
    clearNotifications,

    // Shortcut Actions
    setCustomShortcuts,
    loadCustomShortcuts,

    // Initialization
    init,
    getState
  }
}, {
  persist: {
    key: 'ai-pro-ui',
    paths: ['theme', 'leftPanelVisible', 'rightPanelVisible']
  }
})
