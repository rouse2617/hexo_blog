<template>
  <div v-show="visible" class="message-search-bar" :class="{ 'dark-mode': isDark }">
    <!-- Search Input -->
    <div class="search-input-wrapper">
      <el-icon class="search-icon"><Search /></el-icon>
      <el-input
        ref="searchInputRef"
        v-model="searchText"
        placeholder="Search messages... (Ctrl+F)"
        class="search-input"
        clearable
        @input="handleSearchInput"
        @keydown="handleKeydown"
        @clear="handleClear"
      >
        <template #append>
          <el-dropdown trigger="click" @command="handleQuickFilter">
            <el-button :icon="Filter" />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="content">Content</el-dropdown-item>
                <el-dropdown-item command="commands">Commands</el-dropdown-item>
                <el-dropdown-item command="files">Files</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-input>
      <el-button
        :icon="Setting"
        circle
        size="small"
        class="settings-button"
        @click="showAdvancedFilters = !showAdvancedFilters"
      />
    </div>

    <!-- Advanced Filters -->
    <transition name="slide-down">
      <div v-show="showAdvancedFilters" class="advanced-filters">
        <div class="filter-row">
          <div class="filter-item">
            <label>Time Range</label>
            <el-date-picker
              v-model="filters.timeRange"
              type="datetimerange"
              range-separator="To"
              start-placeholder="Start"
              end-placeholder="End"
              size="small"
              @change="handleFilterChange"
            />
          </div>
          <div class="filter-item">
            <label>Tool Type</label>
            <el-select
              v-model="filters.toolType"
              multiple
              placeholder="Select tools"
              size="small"
              @change="handleFilterChange"
            >
              <el-option
                v-for="tool in availableTools"
                :key="tool"
                :label="tool"
                :value="tool"
              />
            </el-select>
          </div>
        </div>
        <div class="filter-row">
          <div class="filter-item">
            <label>Host</label>
            <el-select
              v-model="filters.hosts"
              multiple
              placeholder="Select hosts"
              size="small"
              @change="handleFilterChange"
            >
              <el-option
                v-for="host in availableHosts"
                :key="host"
                :label="host"
                :value="host"
              />
            </el-select>
          </div>
          <div class="filter-item">
            <label>Error Level</label>
            <el-select
              v-model="filters.errorLevel"
              placeholder="Select level"
              size="small"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="Info" value="info" />
              <el-option label="Warning" value="warning" />
              <el-option label="Error" value="error" />
            </el-select>
          </div>
        </div>
        <div class="filter-row">
          <div class="filter-item">
            <el-checkbox v-model="filters.caseSensitive" @change="handleFilterChange">
              Case Sensitive
            </el-checkbox>
          </div>
          <div class="filter-item">
            <el-checkbox v-model="filters.regex" @change="handleFilterChange">
              Use Regex
            </el-checkbox>
          </div>
          <div class="filter-item">
            <el-button type="primary" size="small" @click="applyFilters">
              Apply Filters
            </el-button>
            <el-button size="small" @click="resetFilters">
              Reset
            </el-button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Results Info -->
    <div v-if="hasResults" class="results-info">
      <div class="results-count">
        <span>{{ resultsCount }} results</span>
        <span v-if="currentIndex >= 0" class="current-index">
          ({{ currentIndex + 1 }} / {{ resultsCount }})
        </span>
      </div>
      <div class="navigation-controls">
        <el-button-group>
          <el-button
            size="small"
            :disabled="currentIndex <= 0"
            @click="navigateTo('prev')"
          >
            <el-icon><ArrowUp /></el-icon>
            Previous
          </el-button>
          <el-button
            size="small"
            :disabled="currentIndex >= resultsCount - 1"
            @click="navigateTo('next')"
          >
            Next
            <el-icon><ArrowDown /></el-icon>
          </el-button>
        </el-button-group>
        <el-button
          size="small"
          :icon="Close"
          @click="handleClose"
        >
          Close
        </el-button>
      </div>
    </div>

    <!-- Search Highlight Preview -->
    <transition name="fade">
      <div v-if="currentMatch" class="match-preview">
        <div class="match-header">
          <span class="match-field">{{ currentMatch.field }}</span>
          <span class="match-position">Line {{ currentMatch.position[0] }}</span>
        </div>
        <div class="match-content" v-html="highlightedText"></div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Search,
  Filter,
  Setting,
  ArrowUp,
  ArrowDown,
  Close
} from '@element-plus/icons-vue'
import type { SearchQuery, SearchMatch } from '@/types/chat-ui'

// Props
interface Props {
  visible: boolean
  resultsCount: number
  currentIndex: number
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  resultsCount: 0,
  currentIndex: -1
})

// Emits
const emit = defineEmits<{
  search: [query: SearchQuery]
  navigate: [direction: 'prev' | 'next']
  close: []
}>()

// State
const searchInputRef = ref()
const searchText = ref('')
const showAdvancedFilters = ref(false)
const currentMatch = ref<SearchMatch | null>(null)
const highlightedText = ref('')

const filters = ref({
  timeRange: null as [Date, Date] | null,
  toolType: [] as string[],
  hosts: [] as string[],
  errorLevel: null as 'info' | 'warning' | 'error' | null,
  caseSensitive: false,
  regex: false
})

// Mock data - in real app, these would come from props or store
const availableTools = ref(['bash', 'read', 'grep', 'write', 'analyze'])
const availableHosts = ref(['localhost', 'server1', 'server2'])

// Computed
const isDark = computed(() => {
  return document.documentElement.classList.contains('dark')
})

const hasResults = computed(() => {
  return props.resultsCount > 0
})

// Methods
function handleSearchInput() {
  // Debounce search input
  if (searchTimeout !== null) {
    clearTimeout(searchTimeout)
  }

  searchTimeout = window.setTimeout(() => {
    performSearch()
  }, 300)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    performSearch()
  } else if (event.key === 'F3' || (event.ctrlKey && event.key === 'g')) {
    event.preventDefault()
    navigateTo('next')
  } else if (event.shiftKey && event.key === 'F3') {
    event.preventDefault()
    navigateTo('prev')
  } else if (event.key === 'Escape') {
    handleClose()
  }
}

function handleClear() {
  searchText.value = ''
  currentMatch.value = null
  highlightedText.value = ''
  emit('search', {})
}

function handleQuickFilter(command: string) {
  switch (command) {
    case 'content':
    case 'commands':
    case 'files':
      // Apply quick filter
      performSearch()
      break
  }
}

function handleFilterChange() {
  // Auto-apply filters on change
  performSearch()
}

function applyFilters() {
  performSearch()
}

function resetFilters() {
  filters.value = {
    timeRange: null,
    toolType: [],
    hosts: [],
    errorLevel: null,
    caseSensitive: false,
    regex: false
  }
  performSearch()
}

function performSearch() {
  const query: SearchQuery = {
    text: searchText.value || undefined,
    timeRange: filters.value.timeRange || undefined,
    toolType: filters.value.toolType.length > 0 ? filters.value.toolType : undefined,
    hosts: filters.value.hosts.length > 0 ? filters.value.hosts : undefined,
    errorLevel: filters.value.errorLevel || undefined,
    caseSensitive: filters.value.caseSensitive,
    regex: filters.value.regex
  }

  emit('search', query)
}

function navigateTo(direction: 'prev' | 'next') {
  emit('navigate', direction)
}

function handleClose() {
  searchText.value = ''
  currentMatch.value = null
  highlightedText.value = ''
  showAdvancedFilters.value = false
  emit('close')
}

// function updateCurrentMatch(match: SearchMatch) {
//   currentMatch.value = match
//   highlightedText.value = highlightMatchText(match.text, match.position)
// }

function highlightMatchText(text: string, position: [number, number]): string {
  const [start, end] = position
  const before = text.substring(0, start)
  const matchText = text.substring(start, end)
  const after = text.substring(end)

  return `${before}<mark class="search-highlight">${matchText}</mark>${after}`
}

// Export for potential future use
void { highlightMatchText }

// Keyboard shortcut handler
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if (e.ctrlKey && e.key === 'f') {
    e.preventDefault()
    if (!props.visible) {
      emit('search', {})
    }
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  }
}

let searchTimeout: number | null = null

// Lifecycle
onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
})

// Watch for visibility changes
watch(() => props.visible, (visible) => {
  if (visible) {
    nextTick(() => {
      searchInputRef.value?.focus()
    })
  }
})
</script>

<style scoped lang="scss">
.message-search-bar {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);

  &.dark-mode {
    background: #1a1a1a;
    border-color: #333;
  }
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.search-icon {
  font-size: 18px;
  color: var(--el-text-color-secondary);
}

.search-input {
  flex: 1;

  :deep(.el-input__inner) {
    font-family: inherit;
  }
}

.settings-button {
  flex-shrink: 0;
}

// Advanced Filters
.advanced-filters {
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  margin-bottom: 8px;
}

.filter-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 4px;

  label {
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }

  > * {
    width: 180px;
  }

  .el-button-group {
    width: auto;
  }

  .el-checkbox {
    margin: 0;
  }
}

// Results Info
.results-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.results-count {
  font-size: 13px;
  color: var(--el-text-color-regular);

  .current-index {
    color: var(--el-color-primary);
    font-weight: 600;
  }
}

.navigation-controls {
  display: flex;
  gap: 8px;
}

// Match Preview
.match-preview {
  margin-top: 12px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
}

.match-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 12px;
}

.match-field {
  font-weight: 600;
  color: var(--el-color-primary);
}

.match-position {
  color: var(--el-text-color-secondary);
}

.match-content {
  padding: 8px;
  background: var(--el-bg-color);
  border-radius: 4px;
  font-family: 'Consolas', monospace;
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;

  :deep(.search-highlight) {
    background: rgba(var(--el-color-warning-rgb), 0.3);
    padding: 2px 4px;
    border-radius: 2px;
    font-weight: 600;
  }
}

// Dark mode
.dark-mode {
  .advanced-filters {
    background: #2a2a2a;
  }

  .match-preview {
    background: #2a2a2a;
    border-color: #333;
  }

  .match-content {
    background: #1a1a1a;
  }

  .results-info {
    border-top-color: #333;
  }
}

// Animations
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease;
  max-height: 300px;
  overflow: hidden;
}

.slide-down-enter-from,
.slide-down-leave-to {
  max-height: 0;
  opacity: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
