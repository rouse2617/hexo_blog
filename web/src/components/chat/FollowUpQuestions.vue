<template>
  <div class="follow-up-questions" v-if="questions.length > 0">
    <div class="follow-up-header">
      <el-icon><ChatDotRound /></el-icon>
      <span class="header-title">相关问题</span>
    </div>
    <div class="questions-grid">
      <div
        v-for="(question, index) in questions"
        :key="index"
        class="question-card"
        @click="handleQuestionClick(question)"
      >
        <div class="question-icon" :style="{ background: question.color }">
          <el-icon>
            <component :is="question.icon" />
          </el-icon>
        </div>
        <div class="question-content">
          <div class="question-text">{{ question.text }}</div>
          <div class="question-hint">{{ question.hint }}</div>
        </div>
        <el-icon class="question-arrow"><ArrowRight /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  ChatDotRound, ArrowRight, TrendCharts, Document,
  Warning, Tools, Monitor, Setting
} from '@element-plus/icons-vue'

interface FollowUpQuestion {
  text: string
  hint: string
  icon: any
  color: string
}

const props = defineProps<{
  lastMessage?: string
  lastTool?: string
  context?: string[]
}>()

const emit = defineEmits<{
  select: [question: string]
}>()

// 预定义的追问模板库
const questionTemplates: Record<string, FollowUpQuestion[]> = {
  // 监控相关
  monitor: [
    {
      text: '查看历史趋势',
      hint: '过去 24 小时的数据变化',
      icon: TrendCharts,
      color: 'linear-gradient(135deg, #667eea, #764ba2)'
    },
    {
      text: '设置告警阈值',
      hint: '配置自动告警规则',
      icon: Warning,
      color: 'linear-gradient(135deg, #f093fb, #f5576c)'
    },
    {
      text: '性能优化建议',
      hint: '基于当前状态的优化方案',
      icon: TrendCharts,
      color: 'linear-gradient(135deg, #4facfe, #00f2fe)'
    }
  ],

  // 日志相关
  log: [
    {
      text: '错误统计汇总',
      hint: '按类型统计错误出现频率',
      icon: Document,
      color: 'linear-gradient(135deg, #fa709a, #fee140)'
    },
    {
      text: '关联日志分析',
      hint: '查找相关的错误日志',
      icon: Document,
      color: 'linear-gradient(135deg, #a8edea, #fed6e3)'
    },
    {
      text: '日志导出报告',
      hint: '生成可分享的日志报告',
      icon: Document,
      color: 'linear-gradient(135deg, #ff9a9e, #fecfef)'
    }
  ],

  // 命令执行相关
  command: [
    {
      text: '解释命令含义',
      hint: '详细解释刚才执行命令的作用',
      icon: Document,
      color: 'linear-gradient(135deg, #a1c4fd, #c2e9fb)'
    },
    {
      text: '相关命令推荐',
      hint: '类似场景的其他命令',
      icon: Tools,
      color: 'linear-gradient(135deg, #d4fc79, #96e6a1)'
    },
    {
      text: '添加到脚本库',
      hint: '将命令保存为可复用脚本',
      icon: Setting,
      color: 'linear-gradient(135deg, #84fab0, #8fd3f4)'
    }
  ],

  // 故障排查相关
  troubleshoot: [
    {
      text: '深入诊断',
      hint: '更详细的系统诊断',
      icon: Monitor,
      color: 'linear-gradient(135deg, #e0c3fc, #8ec5fc)'
    },
    {
      text: '查看历史问题',
      hint: '类似的过去问题和解决方案',
      icon: Document,
      color: 'linear-gradient(135deg, #fddb92, #d1fdff)'
    },
    {
      text: '生成修复方案',
      hint: '基于 AI 分析的修复建议',
      icon: Tools,
      color: 'linear-gradient(135deg, #9890e3, #b1f4cf)'
    }
  ],

  // 通用问题
  general: [
    {
      text: '详细说明',
      hint: '更详细地解释刚才的内容',
      icon: Document,
      color: 'linear-gradient(135deg, #667eea, #764ba2)'
    },
    {
      text: '提供示例',
      hint: '给出具体的操作示例',
      icon: Document,
      color: 'linear-gradient(135deg, #f093fb, #f5576c)'
    },
    {
      text: '最佳实践',
      hint: '推荐的标准做法',
      icon: TrendCharts,
      color: 'linear-gradient(135deg, #4facfe, #00f2fe)'
    }
  ]
}

// 智能生成相关问题
const questions = computed<FollowUpQuestion[]>(() => {
  const lastTool = props.lastTool?.toLowerCase() || ''
  const lastMessage = props.lastMessage?.toLowerCase() || ''

  // 根据最后执行的工具类型选择模板
  let templateKey = 'general'

  if (lastTool.includes('execute') || lastTool.includes('command')) {
    templateKey = 'command'
  } else if (lastTool.includes('log')) {
    templateKey = 'log'
  } else if (lastTool.includes('monitor') || lastTool.includes('status')) {
    templateKey = 'monitor'
  }

  // 根据消息内容检测场景
  if (lastMessage.includes('error') || lastMessage.includes('fail') || lastMessage.includes('错误')) {
    templateKey = 'troubleshoot'
  } else if (lastMessage.includes('cpu') || lastMessage.includes('memory') || lastMessage.includes('disk')) {
    templateKey = 'monitor'
  } else if (lastMessage.includes('log') || lastMessage.includes('日志')) {
    templateKey = 'log'
  }

  const baseQuestions = questionTemplates[templateKey] || questionTemplates.general

  // 限制显示数量
  return baseQuestions.slice(0, 3)
})

const handleQuestionClick = (question: FollowUpQuestion) => {
  emit('select', question.text)
}
</script>

<style scoped>
.follow-up-questions {
  margin: 16px 20px;
  background: linear-gradient(135deg, #fdfbfb 0%, #ebedee 100%);
  border-radius: 12px;
  padding: 16px;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.follow-up-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  color: #4b5563;
  font-weight: 600;
  font-size: 13px;
}

.header-title {
  flex: 1;
}

.questions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 12px;
}

.question-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s;
  user-select: none;
}

.question-card:hover {
  background: #fff;
  border-color: #3b82f6;
  transform: translateY(-3px);
  box-shadow: 0 8px 20px rgba(59, 130, 246, 0.15);
}

.question-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  color: #fff;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.question-content {
  flex: 1;
  min-width: 0;
}

.question-text {
  font-weight: 500;
  color: #1f2937;
  font-size: 14px;
  margin-bottom: 2px;
}

.question-hint {
  font-size: 12px;
  color: #6b7280;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.question-arrow {
  color: #d1d5db;
  flex-shrink: 0;
  transition: all 0.3s;
}

.question-card:hover .question-arrow {
  color: #3b82f6;
  transform: translateX(4px);
}

/* 响应式 */
@media (max-width: 768px) {
  .questions-grid {
    grid-template-columns: 1fr;
  }
}
</style>
