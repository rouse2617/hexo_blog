<template>
  <div class="message-actions" :class="{ visible: isVisible }">
    <el-dropdown trigger="click" @command="handleCommand">
      <el-button type="text" :icon="MoreFilled" circle size="small" />
      <template #dropdown>
        <el-dropdown-menu>
          <!-- 复制 -->
          <el-dropdown-item command="copy" :icon="CopyDocument">
            复制消息
          </el-dropdown-item>

          <!-- 运维功能：复制命令 -->
          <el-dropdown-item
            v-if="hasCommands"
            command="copyCommand"
            :icon="DocumentCopy"
          >
            复制命令
          </el-dropdown-item>

          <!-- 运维功能：推荐执行 -->
          <el-dropdown-item
            v-if="hasCommands"
            command="recommendExecute"
            :icon="Promotion"
          >
            推荐执行
          </el-dropdown-item>

          <!-- 运维功能：保存为脚本 -->
          <el-dropdown-item
            v-if="hasCommands"
            command="saveAsScript"
            :icon="Download"
          >
            保存为脚本
          </el-dropdown-item>

          <!-- 分隔线 -->
          <el-dropdown-item divided v-if="hasCommands" />

          <!-- 重新生成 -->
          <el-dropdown-item
            v-if="message.role === 'assistant'"
            command="regenerate"
            :icon="RefreshRight"
            :disabled="isLoading"
          >
            重新生成
          </el-dropdown-item>

          <!-- 继续提问 -->
          <el-dropdown-item
            v-if="message.role === 'assistant' && !isLoading"
            command="continue"
            :icon="ChatLineRound"
          >
            继续追问
          </el-dropdown-item>

          <!-- 分隔线 -->
          <el-dropdown-item divided v-if="message.role === 'assistant'" />

          <!-- 好的/坏的反馈 -->
          <el-dropdown-item
            v-if="message.role === 'assistant'"
            command="good"
            :icon="Select"
          >
            <span class="feedback-item">
              <span>有帮助</span>
              <el-tag v-if="feedback === 'good'" type="success" size="small">
                已反馈
              </el-tag>
            </span>
          </el-dropdown-item>
          <el-dropdown-item
            v-if="message.role === 'assistant'"
            command="bad"
            :icon="CloseBold"
          >
            <span class="feedback-item">
              <span>没帮助</span>
              <el-tag v-if="feedback === 'bad'" type="danger" size="small">
                已反馈
              </el-tag>
            </span>
          </el-dropdown-item>

          <!-- 分隔线 -->
          <el-dropdown-item divided />

          <!-- 导出 -->
          <el-dropdown-item command="export" :icon="FolderOpened">
            导出为文件
          </el-dropdown-item>

          <!-- 分享 -->
          <el-dropdown-item command="share" :icon="Share">
            复制分享链接
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <!-- 快速操作按钮组（hover 时显示） -->
    <div class="quick-actions" v-if="message.role === 'assistant'">
      <el-tooltip content="复制" placement="top">
        <el-button
          type="text"
          :icon="CopyDocument"
          circle
          size="small"
          @click="handleCommand('copy')"
        />
      </el-tooltip>

      <!-- 运维功能：复制命令 -->
      <el-tooltip v-if="hasCommands" content="复制命令" placement="top">
        <el-button
          type="text"
          :icon="DocumentCopy"
          circle
          size="small"
          @click="handleCommand('copyCommand')"
        />
      </el-tooltip>

      <!-- 运维功能：推荐执行 -->
      <el-tooltip v-if="hasCommands" content="推荐执行" placement="top">
        <el-button
          type="text"
          :icon="Promotion"
          circle
          size="small"
          @click="handleCommand('recommendExecute')"
        />
      </el-tooltip>

      <el-tooltip content="重新生成" placement="top">
        <el-button
          type="text"
          :icon="RefreshRight"
          circle
          size="small"
          :disabled="isLoading"
          @click="handleCommand('regenerate')"
        />
      </el-tooltip>

      <el-tooltip content="好的" placement="top">
        <el-button
          type="text"
          :icon="Select"
          circle
          size="small"
          :class="{ active: feedback === 'good' }"
          @click="handleCommand('good')"
        />
      </el-tooltip>

      <el-tooltip content="坏的" placement="top">
        <el-button
          type="text"
          :icon="CloseBold"
          circle
          size="small"
          :class="{ active: feedback === 'bad' }"
          @click="handleCommand('bad')"
        />
      </el-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { Message } from '@/api/chat'
import {
  MoreFilled, CopyDocument, DocumentCopy, Promotion, RefreshRight,
  ChatLineRound, Select, CloseBold, Download, FolderOpened, Share
} from '@element-plus/icons-vue'

const props = defineProps<{
  message: Message
  isLoading?: boolean
}>()

const emit = defineEmits<{
  copy: [message: Message]
  copyCommand: [message: Message]
  recommendExecute: [message: Message]
  saveAsScript: [message: Message]
  regenerate: [message: Message]
  continue: [message: Message]
  feedback: [message: Message, type: 'good' | 'bad']
  export: [message: Message]
  share: [message: Message]
}>()

const feedback = ref<'good' | 'bad' | null>(null)
const isVisible = ref(false)

// Extract commands from message content using regex
const hasCommands = computed(() => {
  const content = props.message.content || ''
  // Match common command patterns (shell code blocks or inline commands)
  const commandPatterns = [
    /```(?:bash|shell|sh)\n([\s\S]*?)```/gi,
    /`([^`\n]+)`/g,
    /^\s*(sudo|kubectl|docker|git|npm|yum|apt|systemctl|ssh|curl|wget)\s+/gim
  ]

  for (const pattern of commandPatterns) {
    if (pattern.test(content)) {
      return true
    }
  }
  return false
})

// Extract all commands from the message
const extractCommands = (): string[] => {
  const content = props.message.content || ''
  const commands: string[] = []

  // Extract from code blocks
  const codeBlockRegex = /```(?:bash|shell|sh)\n([\s\S]*?)```/gi
  let match
  while ((match = codeBlockRegex.exec(content)) !== null) {
    const blockCommands = match[1]
      .split('\n')
      .map(line => line.trim())
      .filter(line => line && !line.startsWith('#'))
    commands.push(...blockCommands)
  }

  // Extract inline commands
  const inlineCommandRegex = /`([^`\n]+)`/g
  while ((match = inlineCommandRegex.exec(content)) !== null) {
    const cmd = match[1].trim()
    if (/^(sudo|kubectl|docker|git|npm|yum|apt|systemctl|ssh|curl|wget)\s+/.test(cmd)) {
      commands.push(cmd)
    }
  }

  return commands
}

const handleCommand = async (command: string) => {
  switch (command) {
    case 'copy':
      await copyMessage()
      emit('copy', props.message)
      break

    case 'copyCommand':
      await copyCommands()
      emit('copyCommand', props.message)
      break

    case 'recommendExecute':
      recommendExecuteCommands()
      emit('recommendExecute', props.message)
      break

    case 'saveAsScript':
      saveAsScript()
      emit('saveAsScript', props.message)
      break

    case 'regenerate':
      emit('regenerate', props.message)
      break

    case 'continue':
      emit('continue', props.message)
      break

    case 'good':
      feedback.value = feedback.value === 'good' ? null : 'good'
      if (feedback.value) {
        ElMessage.success('感谢您的反馈！')
      }
      emit('feedback', props.message, 'good')
      break

    case 'bad':
      feedback.value = feedback.value === 'bad' ? null : 'bad'
      if (feedback.value) {
        ElMessage.warning('感谢您的反馈，我们会改进！')
      }
      emit('feedback', props.message, 'bad')
      break

    case 'export':
      exportMessage()
      emit('export', props.message)
      break

    case 'share':
      await shareMessage()
      emit('share', props.message)
      break
  }
}

const copyMessage = async () => {
  try {
    await navigator.clipboard.writeText(props.message.content)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyCommands = async () => {
  const commands = extractCommands()
  if (commands.length === 0) {
    ElMessage.warning('未找到可执行的命令')
    return
  }

  const commandText = commands.join('\n')
  try {
    await navigator.clipboard.writeText(commandText)
    ElMessage.success(`已复制 ${commands.length} 条命令到剪贴板`)
  } catch {
    ElMessage.error('复制失败')
  }
}

const recommendExecuteCommands = () => {
  const commands = extractCommands()
  if (commands.length === 0) {
    ElMessage.warning('未找到可执行的命令')
    return
  }

  ElMessage.success(`已推荐 ${commands.length} 条命令到执行队列`)
}

const saveAsScript = () => {
  const commands = extractCommands()
  if (commands.length === 0) {
    ElMessage.warning('未找到可执行的命令')
    return
  }

  // Add shebang and proper formatting
  const scriptContent = `#!/bin/bash
# Auto-generated script from AI conversation
# Generated at: ${new Date().toISOString()}

set -e  # Exit on error

${commands.join('\n')}
`

  const blob = new Blob([scriptContent], { type: 'text/x-shellscript' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `ai_script_${Date.now()}.sh`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  ElMessage.success('脚本已保存')
}

const exportMessage = () => {
  const blob = new Blob([props.message.content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `message_${Date.now()}.md`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  ElMessage.success('消息已导出')
}

const shareMessage = async () => {
  const shareText = `${props.message.role === 'user' ? '用户' : 'AI'}: ${props.message.content.substring(0, 100)}...`

  try {
    await navigator.clipboard.writeText(shareText)
    ElMessage.success('分享内容已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

// 暴露方法供父组件调用
defineExpose({
  show: () => { isVisible.value = true },
  hide: () => { isVisible.value = false }
})
</script>

<style scoped>
.message-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.message-actions.visible {
  opacity: 1;
}

/* 鼠标悬停消息项时显示操作按钮 */
.message-item:hover .message-actions {
  opacity: 1;
}

.quick-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 8px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.quick-actions .el-button {
  color: #6b7280;
}

.quick-actions .el-button:hover {
  color: #3b82f6;
  background: #eff6ff;
}

.quick-actions .el-button.active {
  color: #3b82f6;
  background: #dbeafe;
}

.feedback-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 140px;
}
</style>
