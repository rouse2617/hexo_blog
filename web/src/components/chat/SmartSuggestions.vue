<template>
  <div class="smart-suggestions" v-if="suggestions.length > 0">
    <div class="suggestions-header">
      <el-icon><Star /></el-icon>
      <span class="header-title">智能推荐</span>
    </div>
    <div class="suggestions-list">
      <div
        v-for="(suggestion, index) in suggestions"
        :key="index"
        class="suggestion-item"
        @click="handleSuggestionClick(suggestion)"
      >
        <div class="suggestion-icon">
          <el-icon>
            <component :is="suggestion.icon" />
          </el-icon>
        </div>
        <div class="suggestion-content">
          <div class="suggestion-title">{{ suggestion.title }}</div>
          <div class="suggestion-desc">{{ suggestion.description }}</div>
        </div>
        <el-icon class="suggestion-arrow"><ArrowRight /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  Star, ArrowRight, Monitor, Document,
  DataAnalysis, Tools, Warning
} from '@element-plus/icons-vue'

interface Suggestion {
  title: string
  description: string
  prompt: string
  icon: any
  category: 'monitor' | 'log' | 'analysis' | 'command' | 'troubleshoot'
}

const props = defineProps<{
  currentContext?: {
    lastTool?: string
    lastCommand?: string
    hasError?: boolean
    selectedHosts?: string[]
  }
}>()

const emit = defineEmits<{
  select: [prompt: string]
}>()

// 预定义的运维场景建议库
const suggestionTemplates: Suggestion[] = [
  {
    title: '系统健康检查',
    description: '查看 CPU、内存、磁盘使用情况',
    prompt: '帮我检查一下服务器的整体健康状况',
    icon: Monitor,
    category: 'monitor'
  },
  {
    title: '查看最近错误日志',
    description: '分析最近的系统错误和异常',
    prompt: '查看最近 1 小时内的错误日志',
    icon: Warning,
    category: 'log'
  },
  {
    title: '进程资源占用',
    description: '查看占用资源最多的进程',
    prompt: '列出占用 CPU 和内存最多的前 10 个进程',
    icon: DataAnalysis,
    category: 'analysis'
  },
  {
    title: '磁盘空间分析',
    description: '查找占用空间大的目录',
    prompt: '分析磁盘使用情况，找出占用空间最大的目录',
    icon: Document,
    category: 'analysis'
  },
  {
    title: '网络连接状态',
    description: '查看当前网络连接和端口监听',
    prompt: '检查当前的网络连接和监听端口',
    icon: Monitor,
    category: 'monitor'
  },
  {
    title: '系统负载趋势',
    description: '查看过去 5 分钟的负载变化',
    prompt: '显示最近 5 分钟的系统负载变化',
    icon: DataAnalysis,
    category: 'analysis'
  }
]

// 根据上下文动态生成建议
const suggestions = computed<Suggestion[]>(() => {
  const context = props.currentContext || {}
  const baseSuggestions: Suggestion[] = []

  // 如果刚刚执行了监控命令，推荐后续分析命令
  if (context.lastTool === 'execute_command') {
    baseSuggestions.push({
      title: '深入分析',
      description: '对刚才的结果进行详细分析',
      prompt: '对上面的结果进行分析，找出潜在问题',
      icon: DataAnalysis,
      category: 'analysis'
    })
  }

  // 如果有错误，推荐故障排查命令
  if (context.hasError) {
    baseSuggestions.push({
      title: '故障排查',
      description: '诊断可能的问题原因',
      prompt: '根据上面的错误信息，帮我分析可能的原因和解决方案',
      icon: Tools,
      category: 'troubleshoot'
    })
  }

  // 如果有选中的主机，推荐主机特定命令
  if (context.selectedHosts && context.selectedHosts.length > 0) {
    baseSuggestions.push({
      title: '批量操作',
      description: `对 ${context.selectedHosts.length} 台主机执行相同操作`,
      prompt: '帮我在这几台主机上同时检查系统状态',
      icon: Monitor,
      category: 'command'
    })
  }

  // 添加通用建议（如果数量不足 4 个）
  const remainingCount = 4 - baseSuggestions.length
  if (remainingCount > 0) {
    baseSuggestions.push(...suggestionTemplates.slice(0, remainingCount))
  }

  // 去重并限制数量
  const uniqueSuggestions = Array.from(
    new Map(baseSuggestions.map(s => [s.title, s])).values()
  )

  return uniqueSuggestions.slice(0, 6)
})

const handleSuggestionClick = (suggestion: Suggestion) => {
  emit('select', suggestion.prompt)
}
</script>

<style scoped>
.smart-suggestions {
  margin: 12px 20px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 1px solid #bae6fd;
  border-radius: 12px;
  padding: 16px;
  animation: slideIn 0.3s ease-out;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.suggestions-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: #0369a1;
  font-weight: 600;
  font-size: 14px;
}

.header-title {
  flex: 1;
}

.suggestions-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.suggestion-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: #fff;
  border: 1px solid #e0f2fe;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.suggestion-item:hover {
  background: #f0f9ff;
  border-color: #0ea5e9;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(14, 165, 233, 0.15);
}

.suggestion-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #0ea5e9, #0284c7);
  border-radius: 8px;
  color: #fff;
  flex-shrink: 0;
}

.suggestion-content {
  flex: 1;
  min-width: 0;
}

.suggestion-title {
  font-weight: 500;
  color: #0c4a6e;
  font-size: 13px;
  margin-bottom: 2px;
}

.suggestion-desc {
  font-size: 12px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.suggestion-arrow {
  color: #94a3b8;
  flex-shrink: 0;
  transition: transform 0.2s;
}

.suggestion-item:hover .suggestion-arrow {
  transform: translateX(4px);
  color: #0ea5e9;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .suggestions-list {
    grid-template-columns: 1fr;
  }
}
</style>
