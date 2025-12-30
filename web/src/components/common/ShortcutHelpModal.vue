<template>
  <el-dialog
    v-model="visible"
    title="Keyboard Shortcuts"
    width="600px"
    :close-on-click-modal="true"
    :close-on-press-escape="true"
    @close="handleClose"
  >
    <div class="shortcuts-container">
      <el-alert
        type="info"
        :closable="false"
        show-icon
      >
        Press <kbd>Ctrl</kbd> + <kbd>/</kbd> to toggle this dialog
      </el-alert>

      <div class="shortcuts-content">
        <div v-for="category in categorizedShortcuts" :key="category.name" class="category-section">
          <h3 class="category-title">{{ category.title }}</h3>
          <div class="shortcuts-list">
            <div
              v-for="shortcut in category.shortcuts"
              :key="shortcut.key"
              class="shortcut-item"
            >
              <div class="shortcut-description">
                {{ shortcut.description }}
              </div>
              <div class="shortcut-keys">
                <kbd
                  v-for="(key, index) in shortcut.keys"
                  :key="index"
                  class="key-badge"
                >
                  {{ key }}
                </kbd>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="shortcuts-footer">
        <el-text type="info" size="small">
          Note: Some shortcuts may not work when typing in input fields
        </el-text>
      </div>
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">Close</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useShortcuts, formatShortcutDisplay, type Shortcut, presetShortcuts } from '@/composables/useShortcuts'

const visible = ref(false)
const { register, unregister } = useShortcuts()

const categorizedShortcuts = computed(() => {
  const categories = [
    { name: 'general', title: 'General', shortcuts: [] as (Shortcut & { keys: string[] })[] },
    { name: 'chat', title: 'Chat', shortcuts: [] as (Shortcut & { keys: string[] })[] },
    { name: 'navigation', title: 'Navigation', shortcuts: [] as (Shortcut & { keys: string[] })[] },
    { name: 'editing', title: 'Editing', shortcuts: [] as (Shortcut & { keys: string[] })[] }
  ]

  // Group preset shortcuts by category
  presetShortcuts.forEach(shortcut => {
    const category = categories.find(c => c.name === shortcut.category)
    if (category) {
      category.shortcuts.push({
        ...shortcut,
        keys: formatShortcutDisplay(shortcut).split(' + ')
      })
    }
  })

  return categories.filter(c => c.shortcuts.length > 0)
})

function open() {
  visible.value = true
}

function close() {
  visible.value = false
}

function handleClose() {
  close()
}

function toggle() {
  if (visible.value) {
    close()
  } else {
    open()
  }
}

// Register keyboard shortcut to open/close modal
let shortcutKey = ''

onMounted(() => {
  const toggleShortcut: Shortcut = {
    key: '/',
    ctrl: true,
    description: 'Show keyboard shortcuts',
    category: 'general',
    handler: toggle
  }

  shortcutKey = register(toggleShortcut)
})

onUnmounted(() => {
  if (shortcutKey) {
    unregister(shortcutKey)
  }
})

// Expose toggle method for parent components
defineExpose({
  open,
  close,
  toggle
})
</script>

<style scoped>
.shortcuts-container {
  padding: 12px 0;
}

.shortcuts-content {
  margin-top: 20px;
  max-height: 500px;
  overflow-y: auto;
}

.category-section {
  margin-bottom: 24px;
}

.category-section:last-child {
  margin-bottom: 0;
}

.category-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 12px 0;
  padding-bottom: 8px;
  border-bottom: 2px solid var(--color-border-secondary);
}

.shortcuts-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.shortcut-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--color-bg-secondary);
  border-radius: 6px;
  transition: background-color 0.2s ease;
}

.shortcut-item:hover {
  background-color: var(--color-bg-tertiary);
}

.shortcut-description {
  flex: 1;
  font-size: 14px;
  color: var(--color-text-primary);
}

.shortcut-keys {
  display: flex;
  gap: 4px;
  align-items: center;
}

.key-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 28px;
  padding: 0 8px;
  font-size: 12px;
  font-weight: 500;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  color: var(--color-text-primary);
  background-color: var(--color-bg-elevated);
  border: 1px solid var(--color-border-primary);
  border-radius: 4px;
  box-shadow: 0 2px 0 var(--color-border-secondary);
}

.shortcuts-footer {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border-secondary);
  text-align: center;
}

/* Scrollbar styling for shortcuts content */
.shortcuts-content::-webkit-scrollbar {
  width: 6px;
}

.shortcuts-content::-webkit-scrollbar-track {
  background: var(--color-scrollbar-track);
  border-radius: 3px;
}

.shortcuts-content::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar-thumb);
  border-radius: 3px;
}

.shortcuts-content::-webkit-scrollbar-thumb:hover {
  background: var(--color-scrollbar-thumb-hover);
}

/* kbd element styling */
kbd {
  display: inline-block;
  padding: 2px 6px;
  font-size: 11px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  color: var(--color-text-primary);
  background-color: var(--color-bg-elevated);
  border: 1px solid var(--color-border-primary);
  border-radius: 3px;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.1);
}
</style>
