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
          <el-dropdown-item command="export" :icon="Download">
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
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { Message } from '@/api/chat'
import {
  MoreFilled, CopyDocument, RefreshRight, ChatLineRound,
  Select, CloseBold, Download, Share
} from '@element-plus/icons-vue'

const props = defineProps<{
  message: Message
  isLoading?: boolean
}>()

const emit = defineEmits<{
  copy: [message: Message]
  regenerate: [message: Message]
  continue: [message: Message]
  feedback: [message: Message, type: 'good' | 'bad']
  export: [message: Message]
  share: [message: Message]
}>()

const feedback = ref<'good' | 'bad' | null>(null)
const isVisible = ref(false)

const handleCommand = async (command: string) => {
  switch (command) {
    case 'copy':
      await copyMessage()
      emit('copy', props.message)
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
