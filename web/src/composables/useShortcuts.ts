import { onMounted, onUnmounted } from 'vue'

export interface Shortcut {
  key: string
  ctrl?: boolean
  alt?: boolean
  shift?: boolean
  meta?: boolean
  description: string
  handler: (event: KeyboardEvent) => void
  category?: 'general' | 'chat' | 'navigation' | 'editing'
}

const shortcuts = new Map<string, Shortcut>()
const activeShortcuts: Set<string> = new Set()

function generateShortcutKey(shortcut: Shortcut): string {
  const modifiers = [
    shortcut.ctrl ? 'ctrl' : '',
    shortcut.alt ? 'alt' : '',
    shortcut.shift ? 'shift' : '',
    shortcut.meta ? 'meta' : '',
    shortcut.key.toLowerCase()
  ].filter(Boolean).join('+')

  return modifiers
}

function isInputFocused(): boolean {
  const activeElement = document.activeElement
  if (!activeElement) return false

  const tagName = activeElement.tagName.toLowerCase()
  const isInput = ['input', 'textarea', 'select'].includes(tagName)
  const isContentEditable = activeElement.getAttribute('contenteditable') === 'true'

  // Allow shortcuts in specific cases even when input is focused
  if (isInput || isContentEditable) {
    // Escape key always works
    return true
  }

  return false
}

function handleKeyPress(event: KeyboardEvent): void {
  const pressedKey = event.key.toLowerCase()

  // Build the shortcut key from the event
  const modifiers = [
    event.ctrlKey ? 'ctrl' : '',
    event.altKey ? 'alt' : '',
    event.shiftKey ? 'shift' : '',
    event.metaKey ? 'meta' : '',
    pressedKey
  ].filter(Boolean).join('+')

  const shortcut = shortcuts.get(modifiers)

  if (shortcut && activeShortcuts.has(modifiers)) {
    // Check if we should prevent default based on input focus
    const isInput = isInputFocused()

    // Some shortcuts work even in inputs (like Escape)
    const allowInInput = shortcut.key.toLowerCase() === 'escape'

    if (!isInput || allowInInput) {
      event.preventDefault()
      event.stopPropagation()
      shortcut.handler(event)
    }
  }
}

export function useShortcuts() {
  function register(shortcut: Shortcut): string {
    const key = generateShortcutKey(shortcut)
    shortcuts.set(key, shortcut)
    activeShortcuts.add(key)
    return key
  }

  function unregister(keyOrShortcut: string | Shortcut): void {
    const key = typeof keyOrShortcut === 'string'
      ? keyOrShortcut
      : generateShortcutKey(keyOrShortcut)

    shortcuts.delete(key)
    activeShortcuts.delete(key)
  }

  function registerMultiple(shortcutsList: Shortcut[]): string[] {
    return shortcutsList.map(s => register(s))
  }

  function unregisterAll(): void {
    shortcuts.clear()
    activeShortcuts.clear()
  }

  function getActiveShortcuts(): Shortcut[] {
    return Array.from(shortcuts.values())
  }

  function getShortcutsByCategory(category: Shortcut['category']): Shortcut[] {
    return Array.from(shortcuts.values()).filter(s => s.category === category)
  }

  function isShortcutRegistered(shortcut: Shortcut): boolean {
    const key = generateShortcutKey(shortcut)
    return shortcuts.has(key)
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeyPress, true)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyPress, true)
  })

  return {
    register,
    unregister,
    registerMultiple,
    unregisterAll,
    getActiveShortcuts,
    getShortcutsByCategory,
    isShortcutRegistered
  }
}

// Preset shortcuts for the application
export const presetShortcuts: Shortcut[] = [
  // General
  {
    key: '/',
    ctrl: true,
    description: 'Show keyboard shortcuts',
    category: 'general',
    handler: () => {}
  },

  // Chat
  {
    key: 'Enter',
    ctrl: true,
    description: 'Send message',
    category: 'chat',
    handler: () => {}
  },
  {
    key: 'k',
    ctrl: true,
    description: 'Clear chat',
    category: 'chat',
    handler: () => {}
  },
  {
    key: 'e',
    ctrl: true,
    description: 'Export chat',
    category: 'chat',
    handler: () => {}
  },

  // Navigation
  {
    key: 'h',
    ctrl: true,
    description: 'Toggle history panel',
    category: 'navigation',
    handler: () => {}
  },
  {
    key: 'm',
    ctrl: true,
    description: 'Toggle monitor panel',
    category: 'navigation',
    handler: () => {}
  },
  {
    key: 'b',
    ctrl: true,
    shift: true,
    description: 'Focus chat input',
    category: 'navigation',
    handler: () => {}
  },

  // Editing
  {
    key: 'ArrowUp',
    ctrl: true,
    description: 'Previous message in history',
    category: 'editing',
    handler: () => {}
  },
  {
    key: 'ArrowDown',
    ctrl: true,
    description: 'Next message in history',
    category: 'editing',
    handler: () => {}
  },
  {
    key: 'l',
    ctrl: true,
    shift: true,
    description: 'Clear input',
    category: 'editing',
    handler: () => {}
  }
]

export function formatShortcutDisplay(shortcut: Shortcut): string {
  const parts: string[] = []

  if (shortcut.ctrl) parts.push('Ctrl')
  if (shortcut.alt) parts.push('Alt')
  if (shortcut.shift) parts.push('Shift')
  if (shortcut.meta) parts.push('Cmd')

  const key = shortcut.key
  if (key === ' ') parts.push('Space')
  else if (key === 'ArrowUp') parts.push('↑')
  else if (key === 'ArrowDown') parts.push('↓')
  else if (key === 'ArrowLeft') parts.push('←')
  else if (key === 'ArrowRight') parts.push('→')
  else parts.push(key.charAt(0).toUpperCase() + key.slice(1))

  return parts.join(' + ')
}
