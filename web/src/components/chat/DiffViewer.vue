<template>
  <div class="diff-viewer" :class="{ 'dark-mode': isDark }">
    <!-- Header with file info and actions -->
    <div class="diff-header">
      <div class="file-info">
        <el-icon><Document /></el-icon>
        <span class="file-name">{{ fileName }}</span>
        <el-tag v-if="isLargeFile" type="warning" size="small" effect="plain">
          Large File ({{ totalLines }} lines)
        </el-tag>
      </div>
      <div class="diff-actions">
        <el-button
          text
          size="small"
          @click="toggleCollapseUnchanged"
          :icon="collapseUnchanged ? 'Expand' : 'Fold'"
        >
          {{ collapseUnchanged ? 'Expand All' : 'Collapse Unchanged' }}
        </el-button>
        <el-button
          type="success"
          size="small"
          @click="handleApply"
          :icon="Check"
        >
          Apply
        </el-button>
        <el-button
          type="danger"
          size="small"
          @click="handleReject"
          :icon="Close"
        >
          Reject
        </el-button>
      </div>
    </div>

    <!-- Diff stats -->
    <div class="diff-stats">
      <div class="stat-item added">
        <el-icon><Plus /></el-icon>
        <span>{{ addedCount }} additions</span>
      </div>
      <div class="stat-item removed">
        <el-icon><Minus /></el-icon>
        <span>{{ removedCount }} deletions</span>
      </div>
      <div class="stat-item unchanged">
        <el-icon><Document /></el-icon>
        <span>{{ unchangedCount }} unchanged</span>
      </div>
    </div>

    <!-- Diff content -->
    <div class="diff-content" :class="{ 'collapsed-unchanged': collapseUnchanged }">
      <div
        v-for="(line, index) in diffLines"
        :key="index"
        class="diff-line"
        :class="[
          `line-${line.type}`,
          { 'line-unchanged-collapsed': line.type === 'unchanged' && collapseUnchanged }
        ]"
      >
        <div class="line-numbers">
          <span v-if="line.lineNumber.old !== undefined" class="line-number old">
            {{ line.lineNumber.old }}
          </span>
          <span v-else class="line-number empty"></span>
          <span v-if="line.lineNumber.new !== undefined" class="line-number new">
            {{ line.lineNumber.new }}
          </span>
          <span v-else class="line-number empty"></span>
        </div>
        <div class="line-content">
          <pre><code>{{ line.content }}</code></pre>
        </div>
      </div>
    </div>

    <!-- Large file warning -->
    <el-dialog
      v-model="showLargeFileWarning"
      title="Large File Warning"
      width="500px"
    >
      <div class="warning-content">
        <el-alert
          type="warning"
          :closable="false"
          show-icon
        >
          <template #title>
            This file has {{ totalLines }} lines. Displaying may be slow.
          </template>
          <p>Consider using a dedicated diff viewer for large files.</p>
        </el-alert>
      </div>
      <template #footer>
        <el-button @click="showLargeFileWarning = false">Cancel</el-button>
        <el-button type="primary" @click="proceedWithLargeFile">Continue</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Document, Check, Close, Plus, Minus } from '@element-plus/icons-vue'
// import { Diff2HtmlUI } from 'diff2html/lib-esm/ui/js/diff2html-ui'
import 'diff2html/bundles/css/diff2html.min.css'

// Props
interface Props {
  oldContent: string
  newContent: string
  fileName: string
}

const props = withDefaults(defineProps<Props>(), {
  oldContent: '',
  newContent: '',
  fileName: 'unknown'
})

// Emits
const emit = defineEmits<{
  apply: []
  reject: []
}>()

// State
const collapseUnchanged = ref(true)
const showLargeFileWarning = ref(false)
const isLargeFile = ref(false)
const totalLines = ref(0)
const diffLines = ref<Array<{
  type: 'added' | 'removed' | 'unchanged'
  lineNumber: { old?: number; new?: number }
  content: string
}>>([])

// Computed
const isDark = computed(() => {
  return document.documentElement.classList.contains('dark')
})

const addedCount = computed(() => {
  return diffLines.value.filter(l => l.type === 'added').length
})

const removedCount = computed(() => {
  return diffLines.value.filter(l => l.type === 'removed').length
})

const unchangedCount = computed(() => {
  return diffLines.value.filter(l => l.type === 'unchanged').length
})

// Methods
function parseDiff() {
  const oldLines = props.oldContent.split('\n')
  const newLines = props.newContent.split('\n')
  totalLines.value = Math.max(oldLines.length, newLines.length)

  // Check for large file
  if (totalLines.value > 1000) {
    isLargeFile.value = true
    showLargeFileWarning.value = true
  }

  // Simple diff implementation (for demo - use diff2html in production)
  const lines: typeof diffLines.value = []
  let oldIndex = 1
  let newIndex = 1

  // This is a simplified diff algorithm
  // In production, use proper diff library like diff2html
  const maxLen = Math.max(oldLines.length, newLines.length)
  for (let i = 0; i < maxLen; i++) {
    const oldLine = oldLines[i]
    const newLine = newLines[i]

    if (oldLine === newLine) {
      lines.push({
        type: 'unchanged',
        lineNumber: { old: oldIndex++, new: newIndex++ },
        content: oldLine || ''
      })
    } else {
      if (oldLine !== undefined) {
        lines.push({
          type: 'removed',
          lineNumber: { old: oldIndex++ },
          content: oldLine
        })
      }
      if (newLine !== undefined) {
        lines.push({
          type: 'added',
          lineNumber: { new: newIndex++ },
          content: newLine
        })
      }
    }
  }

  diffLines.value = lines
}

function toggleCollapseUnchanged() {
  collapseUnchanged.value = !collapseUnchanged.value
}

function handleApply() {
  emit('apply')
}

function handleReject() {
  emit('reject')
}

function proceedWithLargeFile() {
  showLargeFileWarning.value = false
  // Continue with rendering
}

// Lifecycle
onMounted(() => {
  parseDiff()
})

// Watch for content changes
watch(() => [props.oldContent, props.newContent], () => {
  parseDiff()
}, { deep: true })
</script>

<style scoped lang="scss">
.diff-viewer {
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: 600px;

  &.dark-mode {
    background: #1a1a1a;
    border-color: #333;
  }
}

.diff-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);

  .file-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .file-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .diff-actions {
    display: flex;
    gap: 8px;
  }
}

.diff-stats {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color);

  .stat-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);

    &.added {
      color: var(--el-color-success);
    }

    &.removed {
      color: var(--el-color-danger);
    }

    &.unchanged {
      color: var(--el-text-color-secondary);
    }
  }
}

.diff-content {
  flex: 1;
  overflow-y: auto;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;

  &.collapsed-unchanged {
    .line-unchanged-collapsed {
      display: none;
    }
  }
}

.diff-line {
  display: flex;
  border-bottom: 1px solid var(--el-border-color-lighter);

  &.line-added {
    background: rgba(var(--el-color-success-rgb), 0.1);

    .line-content {
      color: var(--el-color-success-dark-2);
    }
  }

  &.line-removed {
    background: rgba(var(--el-color-danger-rgb), 0.1);

    .line-content {
      color: var(--el-color-danger-dark-2);
    }
  }

  &.line-unchanged {
    background: transparent;
  }
}

.line-numbers {
  display: flex;
  gap: 16px;
  padding: 4px 8px;
  background: var(--el-fill-color-lighter);
  border-right: 1px solid var(--el-border-color);
  min-width: 100px;
  user-select: none;

  .line-number {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-align: right;
    min-width: 40px;

    &.old {
      color: var(--el-color-danger);
    }

    &.new {
      color: var(--el-color-success);
    }

    &.empty {
      visibility: hidden;
    }
  }
}

.line-content {
  flex: 1;
  padding: 4px 12px;
  white-space: pre-wrap;
  word-break: break-all;

  pre {
    margin: 0;
    font-family: inherit;
  }

  code {
    background: transparent;
    padding: 0;
    font-family: inherit;
  }
}

// Dark mode
.dark-mode {
  .diff-stats {
    background: #1a1a1a;
    border-bottom-color: #333;
  }

  .diff-line {
    border-bottom-color: #333;

    &.line-added {
      background: rgba(var(--el-color-success-rgb), 0.15);
    }

    &.line-removed {
      background: rgba(var(--el-color-danger-rgb), 0.15);
    }
  }

  .line-numbers {
    background: #2a2a2a;
    border-right-color: #333;
  }

  .line-content {
    color: #e2e8f0;
  }
}

// Scrollbar
.diff-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.diff-content::-webkit-scrollbar-track {
  background: var(--el-fill-color-lighter);
}

.diff-content::-webkit-scrollbar-thumb {
  background: var(--el-border-color-darker);
  border-radius: 4px;

  &:hover {
    background: var(--el-border-color-dark);
  }
}
</style>
