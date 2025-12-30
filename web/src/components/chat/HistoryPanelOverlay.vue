<template>
  <!-- 移动端历史面板遮罩 -->
  <transition name="overlay-fade">
    <div
      v-if="visible"
      class="history-overlay-backdrop"
      @click="handleClose"
    />
  </transition>

  <!-- 移动端历史面板侧滑 -->
  <transition name="slide-left">
    <aside
      v-if="visible"
      ref="panelRef"
      class="history-panel-overlay"
      @touchstart="handleTouchStart"
      @touchmove="handleTouchMove"
      @touchend="handleTouchEnd"
    >
      <div class="panel-header">
        <h3 class="panel-title">
          <el-icon><ChatDotRound /></el-icon>
          <span>对话历史</span>
        </h3>
        <el-button
          :icon="Close"
          circle
          size="small"
          @click="handleClose"
        />
      </div>

      <div class="panel-actions">
        <el-button
          type="primary"
          :icon="Plus"
          @click="handleNewSession"
          style="width: 100%"
        >
          新建对话
        </el-button>
      </div>

      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="session-item"
          :class="{ active: session.id === currentSessionId }"
          @click="handleSessionSelect(session.id)"
        >
          <div class="session-info">
            <el-icon class="session-icon"><ChatDotRound /></el-icon>
            <span class="session-title">{{ session.title || '未命名对话' }}</span>
          </div>
          <el-icon
            class="session-delete"
            @click.stop="handleSessionDelete(session.id)"
          >
            <Delete />
          </el-icon>
        </div>

        <div v-if="sessions.length === 0" class="empty-sessions">
          <el-icon :size="32" color="#c0c4cc"><ChatDotRound /></el-icon>
          <p>暂无对话记录</p>
        </div>
      </div>
    </aside>
  </transition>
</template>

<script setup lang="ts">
/**
 * HistoryPanelOverlay - Mobile History Side Drawer
 *
 * A mobile-optimized side drawer component for displaying chat history with:
 * - Side drawer effect with smooth animations
 * - Click mask to close functionality
 * - Touch swipe gesture support for closing
 * - Full TypeScript typing
 *
 * @module components/chat/HistoryPanelOverlay
 * @requirements 17
 */

import { ref } from 'vue'
import { ChatDotRound, Close, Plus, Delete } from '@element-plus/icons-vue'
// import type { HistoryPanelOverlayProps } from '@/types/chat-ui'

// ============================================================================
// Props & Emits
// ============================================================================

interface Props {
  /** Whether the overlay is visible */
  visible: boolean
  /** List of chat sessions */
  sessions: Array<{
    id: string
    title: string
    createdAt: string | number
  }>
  /** Current active session ID */
  currentSessionId: string
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  sessions: () => [],
  currentSessionId: ''
})

// Props are used in template (TypeScript doesn't recognize this)
void props

interface Emits {
  /** Emitted when the overlay is closed */
  (e: 'close'): void
  /** Emitted when a session is selected */
  (e: 'sessionSelect', sessionId: string): void
  /** Emitted when a session is deleted */
  (e: 'sessionDelete', sessionId: string): void
}

const emit = defineEmits<Emits>()

// ============================================================================
// Touch Gesture State
// ============================================================================

const panelRef = ref<HTMLElement | null>(null)
let touchStartX = 0
let touchStartY = 0
let isDragging = false
const SWIPE_THRESHOLD = 50 // Minimum distance for swipe gesture

// ============================================================================
// Event Handlers
// ============================================================================

/**
 * Handle overlay close
 */
function handleClose(): void {
  emit('close')
}

/**
 * Handle new session creation
 */
function handleNewSession(): void {
  emit('sessionSelect', '__new__')
}

/**
 * Handle session selection
 */
function handleSessionSelect(sessionId: string): void {
  emit('sessionSelect', sessionId)
}

/**
 * Handle session deletion
 */
function handleSessionDelete(sessionId: string): void {
  emit('sessionDelete', sessionId)
}

// ============================================================================
// Touch Gesture Handlers
// ============================================================================

/**
 * Handle touch start - record initial position
 */
function handleTouchStart(event: TouchEvent): void {
  touchStartX = event.touches[0].clientX
  touchStartY = event.touches[0].clientY
  isDragging = false
}

/**
 * Handle touch move - track drag gesture
 */
function handleTouchMove(event: TouchEvent): void {
  const currentX = event.touches[0].clientX
  const currentY = event.touches[0].clientY
  const deltaX = currentX - touchStartX
  const deltaY = Math.abs(currentY - touchStartY)

  // Only trigger swipe if horizontal movement is greater than vertical
  if (Math.abs(deltaX) > deltaY && Math.abs(deltaX) > 10) {
    isDragging = true

    // Prevent default only if we're swiping horizontally
    event.preventDefault()

    // Apply visual feedback during drag
    if (panelRef.value && deltaX < 0) {
      const translateX = Math.max(deltaX, -window.innerWidth * 0.85)
      panelRef.value.style.transform = `translateX(${translateX}px)`
      panelRef.value.style.transition = 'none'
    }
  }
}

/**
 * Handle touch end - determine if swipe should close the drawer
 */
function handleTouchEnd(event: TouchEvent): void {
  if (!isDragging || !panelRef.value) {
    return
  }

  const touchEndX = event.changedTouches[0].clientX
  const deltaX = touchEndX - touchStartX

  // Reset transition
  panelRef.value.style.transition = ''

  // Check if swipe is from left to right (closing gesture)
  if (deltaX > SWIPE_THRESHOLD) {
    // Swipe right - close drawer
    panelRef.value.style.transform = ''
    handleClose()
  } else {
    // Swipe not strong enough - reset position
    panelRef.value.style.transform = ''
  }

  isDragging = false
}
</script>

<style scoped>
/* ============================================================================
 * Overlay Backdrop (Mask)
 * ============================================================================ */

.history-overlay-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: var(--z-modal-backdrop, 2000);
  -webkit-tap-highlight-color: transparent;
}

/* ============================================================================
 * Side Drawer Panel
 * ============================================================================ */

.history-panel-overlay {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 280px;
  max-width: 85vw;
  display: flex;
  flex-direction: column;
  background: linear-gradient(
    180deg,
    var(--sidebar-bg-start, #1e293b) 0%,
    var(--sidebar-bg-end, #0f172a) 100%
  );
  z-index: var(--z-modal, 2001);
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
  touch-action: pan-y; /* Allow vertical scrolling, handle horizontal manually */
  overflow: hidden;
}

/* ============================================================================
 * Panel Header
 * ============================================================================ */

.history-panel-overlay .panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4, 16px);
  border-bottom: 1px solid var(--sidebar-border, rgba(255, 255, 255, 0.1));
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.02);
}

.history-panel-overlay .panel-title {
  display: flex;
  align-items: center;
  gap: var(--spacing-2, 8px);
  font-size: var(--text-base, 16px);
  font-weight: var(--font-semibold, 600);
  color: var(--sidebar-text-hover, #f1f5f9);
  margin: 0;
}

/* ============================================================================
 * Panel Actions
 * ============================================================================ */

.history-panel-overlay .panel-actions {
  padding: var(--spacing-3, 12px) var(--spacing-4, 16px);
  border-bottom: 1px solid var(--sidebar-border, rgba(255, 255, 255, 0.1));
  flex-shrink: 0;
}

/* ============================================================================
 * Session List
 * ============================================================================ */

.history-panel-overlay .session-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--spacing-2, 8px);
  -webkit-overflow-scrolling: touch; /* Smooth scrolling on iOS */
}

/* Custom scrollbar for desktop */
.history-panel-overlay .session-list::-webkit-scrollbar {
  width: 6px;
}

.history-panel-overlay .session-list::-webkit-scrollbar-track {
  background: transparent;
}

.history-panel-overlay .session-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.history-panel-overlay .session-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* ============================================================================
 * Session Item
 * ============================================================================ */

.history-panel-overlay .session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-3, 12px) var(--spacing-3, 12px);
  margin-bottom: var(--spacing-1, 4px);
  border-radius: var(--radius-lg, 8px);
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s);
  color: var(--sidebar-text, #cbd5e1);
  user-select: none;
  -webkit-tap-highlight-color: transparent;
}

.history-panel-overlay .session-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--sidebar-text-hover, #f1f5f9);
}

.history-panel-overlay .session-item:active {
  background: rgba(255, 255, 255, 0.08);
  transform: scale(0.98);
}

.history-panel-overlay .session-item.active {
  background: var(--sidebar-active-bg, rgba(59, 130, 246, 0.2));
  color: var(--sidebar-active-text, #60a5fa);
}

.history-panel-overlay .session-info {
  display: flex;
  align-items: center;
  gap: var(--spacing-2, 8px);
  flex: 1;
  min-width: 0;
}

.history-panel-overlay .session-icon {
  flex-shrink: 0;
  opacity: 0.8;
}

.history-panel-overlay .session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--text-sm, 14px);
}

.history-panel-overlay .session-delete {
  opacity: 0;
  color: var(--sidebar-text, #cbd5e1);
  transition: all var(--transition-fast, 0.15s);
  flex-shrink: 0;
  padding: 4px;
  margin: -4px;
  border-radius: 4px;
}

.history-panel-overlay .session-item:hover .session-delete {
  opacity: 1;
}

.history-panel-overlay .session-delete:hover {
  color: var(--color-danger, #ef4444);
  background: rgba(239, 68, 68, 0.1);
}

/* ============================================================================
 * Empty State
 * ============================================================================ */

.history-panel-overlay .empty-sessions {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-8, 32px);
  color: var(--sidebar-text, #cbd5e1);
  text-align: center;
}

.history-panel-overlay .empty-sessions p {
  margin-top: var(--spacing-2, 8px);
  font-size: var(--text-sm, 14px);
}

/* ============================================================================
 * Animations
 * ============================================================================ */

/* Overlay fade in/out */
.overlay-fade-enter-active,
.overlay-fade-leave-active {
  transition: opacity var(--transition-base, 0.3s);
}

.overlay-fade-enter-from,
.overlay-fade-leave-to {
  opacity: 0;
}

/* Panel slide from left */
.slide-left-enter-active,
.slide-left-leave-active {
  transition: transform var(--transition-base, 0.3s) cubic-bezier(0.4, 0, 0.2, 1);
}

.slide-left-enter-from,
.slide-left-leave-to {
  transform: translateX(-100%);
}

/* ============================================================================
 * Responsive Adjustments
 * ============================================================================ */

@media (min-width: 768px) {
  .history-panel-overlay {
    /* On larger screens, slightly wider drawer */
    width: 320px;
  }
}

/* ============================================================================
 * Accessibility
 * ============================================================================ */

@media (prefers-reduced-motion: reduce) {
  .overlay-fade-enter-active,
  .overlay-fade-leave-active,
  .slide-left-enter-active,
  .slide-left-leave-active {
    transition: none;
  }

  .history-panel-overlay .session-item {
    transition: none;
  }
}
</style>
