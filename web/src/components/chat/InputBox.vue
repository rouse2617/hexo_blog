<template>
  <div class="input-box">
    <div class="input-wrapper">
      <el-input
        v-model="inputText"
        type="textarea"
        :rows="1"
        :autosize="{ minRows: 1, maxRows: 6 }"
        placeholder="输入您的问题，按 Enter 发送，Shift + Enter 换行"
        resize="none"
        @keydown="handleKeydown"
        :disabled="disabled"
      />
      <div class="input-actions">
        <el-tooltip content="清空对话" placement="top">
          <el-button
            :icon="Delete"
            circle
            size="small"
            @click="$emit('clear')"
            :disabled="disabled"
          />
        </el-tooltip>
        <el-button
          type="primary"
          :icon="Promotion"
          :loading="disabled"
          @click="handleSend"
          :disabled="!inputText.trim() || disabled"
        >
          发送
        </el-button>
      </div>
    </div>
    <div class="input-tips">
      <span>提示：您可以询问服务器状态、执行命令、查看日志等运维相关问题</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Promotion, Delete } from '@element-plus/icons-vue'

defineProps<{
  disabled: boolean
}>()

const emit = defineEmits<{
  send: [message: string]
  clear: []
}>()

const inputText = ref('')

const handleSend = () => {
  if (!inputText.value.trim()) return
  emit('send', inputText.value)
  inputText.value = ''
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}
</script>

<style scoped>
.input-box {
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #ebeef5;
}

.input-wrapper {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.input-wrapper :deep(.el-textarea__inner) {
  padding: 12px 15px;
  font-size: 14px;
  line-height: 1.5;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.input-wrapper :deep(.el-textarea__inner:focus) {
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.2);
}

.input-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.input-tips {
  margin-top: 10px;
  font-size: 12px;
  color: #909399;
}
</style>
