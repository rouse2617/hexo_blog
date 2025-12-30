<template>
  <div class="input-box" role="form" aria-label="Chat input">
    <div class="input-wrapper">
      <div class="input-container">
        <!-- Host Chips Display -->
        <div v-if="selectedHostChips.length > 0" class="host-chips" role="list" aria-label="Selected hosts">
          <el-tag
            v-for="host in selectedHostChips"
            :key="host.id"
            closable
            @close="removeHostChip(host.id)"
            class="host-chip"
            role="listitem"
            :aria-label="`Remove ${host.name}`"
          >
            {{ host.name }}
          </el-tag>
        </div>

        <!-- Main Input -->
        <el-input
          ref="inputRef"
          v-model="inputText"
          type="textarea"
          :rows="1"
          :autosize="{ minRows: 1, maxRows: 6 }"
          placeholder="输入您的问题，按 Enter 发送，Shift + Enter 换行，使用 / 选择命令，使用 @ 选择主机"
          resize="none"
          @keydown="handleKeydown"
          @input="handleInput"
          :disabled="disabled"
          aria-label="Message input"
          role="textbox"
          aria-multiline="true"
        />

        <!-- Slash Command Dropdown -->
        <div
          v-if="showSlashMenu"
          class="dropdown-menu slash-menu"
          :style="menuPosition"
          role="listbox"
          aria-label="Available commands"
        >
          <div
            v-for="(cmd, index) in filteredCommands"
            :key="cmd.name"
            class="menu-item"
            :class="{ active: index === slashMenuIndex }"
            @click="selectSlashCommand(cmd)"
            @mouseenter="slashMenuIndex = index"
            role="option"
            :aria-selected="index === slashMenuIndex"
            :id="`slash-cmd-${index}`"
            tabindex="-1"
          >
            <div class="menu-item-content">
              <el-icon aria-hidden="true"><component :is="cmd.icon" /></el-icon>
              <div class="menu-item-text">
                <span class="command-name">{{ cmd.name }}</span>
                <span class="command-desc">{{ cmd.description }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Host Mention Dropdown -->
        <div
          v-if="showHostMenu"
          class="dropdown-menu host-menu"
          :style="menuPosition"
          role="listbox"
          aria-label="Available hosts"
        >
          <div
            v-for="(host, index) in filteredHosts"
            :key="host.id"
            class="menu-item"
            :class="{ active: index === hostMenuIndex }"
            @click="selectHostMention(host)"
            @mouseenter="hostMenuIndex = index"
            role="option"
            :aria-selected="index === hostMenuIndex"
            :id="`host-mention-${index}`"
            tabindex="-1"
          >
            <div class="menu-item-content">
              <el-tag
                :type="host.status === 'online' ? 'success' : 'info'"
                size="small"
              >
                {{ host.status === 'online' ? '在线' : '离线' }}
              </el-tag>
              <div class="menu-item-text">
                <span class="host-name">{{ host.name }}</span>
                <span class="host-address">{{ host.host }}:{{ host.port }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="input-actions">
        <el-tooltip content="清空对话" placement="top">
          <el-button
            :icon="Delete"
            circle
            size="small"
            @click="$emit('clear')"
            :disabled="disabled"
            aria-label="Clear conversation"
          />
        </el-tooltip>
        <el-button
          type="primary"
          :icon="Promotion"
          :loading="disabled"
          @click="handleSend"
          aria-label="Send message"
          :disabled="!canSend || disabled"
        >
          发送
        </el-button>
      </div>
    </div>
    <div class="input-hints">
      <div class="hint-tags">
        <el-tag size="small" type="info" effect="plain">
          Enter 发送
        </el-tag>
        <el-tag size="small" type="info" effect="plain">
          Shift+Enter 换行
        </el-tag>
        <el-tag size="small" type="info" effect="plain">
          / 命令
        </el-tag>
        <el-tag size="small" type="info" effect="plain">
          @ 主机
        </el-tag>
      </div>
      <span class="hint-text">您可以询问服务器状态、执行命令、查看日志等运维相关问题</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'
import { Promotion, Delete, TrendCharts, Document, CircleCheck, FolderOpened, Cpu, Operation } from '@element-plus/icons-vue'
import { SLASH_COMMANDS, type SlashCommand } from '@/types/chat-ui'
import { useHostStore } from '@/stores/host'
import type { Host } from '@/api/host'

defineProps<{
  disabled: boolean
}>()

const emit = defineEmits<{
  send: [message: string]
  success: []
  clear: []
}>()

const hostStore = useHostStore()
const inputRef = ref<any>()
const inputText = ref('')

// Icon mapping for slash commands
const iconMap: Record<string, any> = {
  TrendCharts,
  Document,
  CircleCheck,
  FolderOpened,
  Cpu,
  Operation
}

// Enhanced slash commands with icon components
const slashCommands = ref<SlashCommand[]>(
  SLASH_COMMANDS.map(cmd => ({
    ...cmd,
    icon: iconMap[cmd.icon] || TrendCharts
  }))
)

// Host mention state
const mentionedHosts = ref<Map<string, Host>>(new Map())
const selectedHostChips = computed(() => Array.from(mentionedHosts.value.values()))

// Menu state
const showSlashMenu = ref(false)
const showHostMenu = ref(false)
const slashMenuIndex = ref(0)
const hostMenuIndex = ref(0)
const menuPosition = ref({ top: '0px', left: '0px' })

// Computed properties
const filteredCommands = computed(() => {
  const match = inputText.value.match(/\/(\w*)$/)
  if (!match) return []
  const query = match[1].toLowerCase()
  return slashCommands.value.filter(cmd =>
    cmd.name.toLowerCase().startsWith(query) || cmd.description.toLowerCase().includes(query)
  )
})

const filteredHosts = computed(() => {
  const match = inputText.value.match(/@(\w*)$/)
  if (!match) return []
  const query = match[1].toLowerCase()
  return hostStore.allHosts.filter(host =>
    host.name.toLowerCase().includes(query) ||
    host.host.toLowerCase().includes(query)
  )
})

const canSend = computed(() => {
  return inputText.value.trim() && !showSlashMenu.value && !showHostMenu.value
})

// Load hosts on mount
onMounted(async () => {
  await hostStore.loadAllHosts()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// Input handler
const handleInput = () => {
  const value = inputText.value

  // Check for slash command
  const slashMatch = value.match(/\/(\w*)$/)
  if (slashMatch && value.trim().startsWith('/')) {
    showSlashMenu.value = true
    showHostMenu.value = false
    slashMenuIndex.value = 0
    updateMenuPosition()
    return
  }

  // Check for host mention
  const hostMatch = value.match(/@(\w*)$/)
  if (hostMatch) {
    showHostMenu.value = true
    showSlashMenu.value = false
    hostMenuIndex.value = 0
    updateMenuPosition()
    return
  }

  // Hide menus if no match
  showSlashMenu.value = false
  showHostMenu.value = false
}

// Update menu position
const updateMenuPosition = () => {
  nextTick(() => {
    const textarea = inputRef.value?.$el?.querySelector('textarea')
    if (textarea) {
      const rect = textarea.getBoundingClientRect()
      menuPosition.value = {
        top: `${rect.bottom + 4}px`,
        left: `${rect.left}px`
      }
    }
  })
}

// Select slash command
const selectSlashCommand = (command: SlashCommand) => {
  const value = inputText.value
  const match = value.match(/^(.*)\/\w*$/)
  if (match) {
    inputText.value = match[1] + command.template
  }
  showSlashMenu.value = false
  inputRef.value?.focus()
}

// Select host mention
const selectHostMention = (host: Host) => {
  const value = inputText.value
  const match = value.match(/^(.*)@\w*$/)
  if (match) {
    inputText.value = match[1]
    // Add to mentioned hosts
    mentionedHosts.value.set(host.id, host)
  }
  showHostMenu.value = false
  inputRef.value?.focus()
}

// Remove host chip
const removeHostChip = (hostId: string) => {
  mentionedHosts.value.delete(hostId)
}

// Keyboard navigation
const handleKeydown = (e: KeyboardEvent) => {
  // Handle menu navigation
  if (showSlashMenu.value) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      slashMenuIndex.value = (slashMenuIndex.value + 1) % filteredCommands.value.length
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      slashMenuIndex.value = (slashMenuIndex.value - 1 + filteredCommands.value.length) % filteredCommands.value.length
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const selected = filteredCommands.value[slashMenuIndex.value]
      if (selected) {
        selectSlashCommand(selected)
      }
    } else if (e.key === 'Escape') {
      showSlashMenu.value = false
    }
    return
  }

  if (showHostMenu.value) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      hostMenuIndex.value = (hostMenuIndex.value + 1) % filteredHosts.value.length
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      hostMenuIndex.value = (hostMenuIndex.value - 1 + filteredHosts.value.length) % filteredHosts.value.length
    } else if (e.key === 'Enter') {
      e.preventDefault()
      const selected = filteredHosts.value[hostMenuIndex.value]
      if (selected) {
        selectHostMention(selected)
      }
    } else if (e.key === 'Escape') {
      showHostMenu.value = false
    }
    return
  }

  // Handle send
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

// Send message
const handleSend = () => {
  if (!canSend.value) return

  // Include mentioned hosts in the message
  let message = inputText.value.trim()

  if (selectedHostChips.value.length > 0) {
    const hostNames = selectedHostChips.value.map(h => h.name).join(', ')
    message = `[目标主机: ${hostNames}]\n${message}`
  }

  emit('send', message)

  // Clear input and reset state
  inputText.value = ''
  mentionedHosts.value.clear()
}

// Clear input function
function clearInput() {
  inputText.value = ''
  mentionedHosts.value.clear()
  showSlashMenu.value = false
  showHostMenu.value = false
}

defineExpose({
  clearInput
})

// Close menus when clicking outside
const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as Node
  const inputContainer = inputRef.value?.$el
  const dropdownMenus = document.querySelectorAll('.dropdown-menu')

  let clickedInsideMenu = false
  dropdownMenus.forEach(menu => {
    if (menu.contains(target)) {
      clickedInsideMenu = true
    }
  })

  if (!clickedInsideMenu && inputContainer && !inputContainer.contains(target)) {
    showSlashMenu.value = false
    showHostMenu.value = false
  }
}
</script>

<style scoped>
.input-box {
  padding: 16px 20px;
  background: #fff;
  position: relative;
}

.input-wrapper {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  position: relative;
}

.input-container {
  flex: 1;
  min-width: 0;
  position: relative;
}

.host-chips {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  padding: 8px 0;
  margin-bottom: 4px;
}

.host-chip {
  user-select: none;
  transition: all 0.2s ease;
}

.host-chip:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.input-container :deep(.el-textarea__inner) {
  padding: 12px 15px;
  font-size: 14px;
  line-height: 1.5;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  transition: all 0.3s ease;
}

.input-container :deep(.el-textarea__inner:focus) {
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.2);
}

/* Dropdown Menu Styles */
.dropdown-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  max-height: 300px;
  overflow-y: auto;
  z-index: 2000;
  padding: 6px 0;
  animation: dropdownFadeIn 0.2s ease;
}

@keyframes dropdownFadeIn {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.menu-item {
  padding: 10px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
}

.menu-item:hover,
.menu-item.active {
  background: #f5f7fa;
}

.menu-item.active {
  background: #ecf5ff;
}

.menu-item-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.menu-item-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.command-name {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
}

.command-desc {
  font-size: 12px;
  color: #909399;
}

.host-name {
  font-weight: 500;
  color: #303133;
  font-size: 14px;
}

.host-address {
  font-size: 12px;
  color: #909399;
}

/* Slash Command Menu Specifics */
.slash-menu {
  min-width: 280px;
}

.slash-menu .el-icon {
  font-size: 20px;
  color: #409eff;
}

/* Host Mention Menu Specifics */
.host-menu {
  min-width: 320px;
}

.input-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.input-hints {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
}

.hint-tags {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.hint-text {
  font-size: 12px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Scrollbar styling for dropdown menus */
.dropdown-menu::-webkit-scrollbar {
  width: 6px;
}

.dropdown-menu::-webkit-scrollbar-track {
  background: #f5f7fa;
  border-radius: 3px;
}

.dropdown-menu::-webkit-scrollbar-thumb {
  background: #dcdfe6;
  border-radius: 3px;
}

.dropdown-menu::-webkit-scrollbar-thumb:hover {
  background: #c0c4cc;
}
</style>
