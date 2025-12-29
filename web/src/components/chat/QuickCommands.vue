<template>
  <div class="quick-commands">
    <div class="commands-header">
      <el-icon><Lightning /></el-icon>
      <span class="header-title">快速命令</span>
      <el-button
        type="text"
        size="small"
        @click="showAll = !showAll"
        class="toggle-btn"
      >
        {{ showAll ? '收起' : '展开全部' }}
      </el-button>
    </div>

    <el-collapse-transition>
      <div v-show="showAll" class="commands-body">
        <!-- 分类标签 -->
        <el-tabs v-model="activeCategory" type="card" class="category-tabs">
          <el-tab-pane
            v-for="category in commandCategories"
            :key="category.key"
            :label="category.label"
            :name="category.key"
          >
            <div class="commands-grid">
              <div
                v-for="cmd in category.commands"
                :key="cmd.name"
                class="command-card"
                @click="handleCommandClick(cmd)"
              >
                <div class="command-header">
                  <div class="command-icon" :style="{ background: cmd.color }">
                    <el-icon>
                      <component :is="cmd.icon" />
                    </el-icon>
                  </div>
                  <div class="command-info">
                    <div class="command-name">{{ cmd.name }}</div>
                    <div class="command-desc">{{ cmd.description }}</div>
                  </div>
                </div>

                <!-- 预览命令 -->
                <div class="command-preview" v-if="cmd.preview">
                  <code>{{ cmd.preview }}</code>
                </div>

                <!-- 快捷键提示 -->
                <div class="command-shortcut" v-if="cmd.shortcut">
                  <el-tag size="small" type="info" effect="plain">
                    {{ cmd.shortcut }}
                  </el-tag>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>

        <!-- 自定义命令 -->
        <div class="custom-commands">
          <div class="custom-header">
            <span>自定义命令</span>
            <el-button type="text" size="small" :icon="Plus">
              添加
            </el-button>
          </div>
          <div v-if="customCommands.length === 0" class="empty-state">
            暂无自定义命令，点击右上角添加
          </div>
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Lightning, Plus, Monitor, Document, Tools,
  Search, Warning, TrendCharts, Setting, FolderOpened
} from '@element-plus/icons-vue'

interface Command {
  name: string
  description: string
  prompt: string
  preview?: string
  icon?: any
  color?: string
  shortcut?: string
}

interface CommandCategory {
  key: string
  label: string
  commands: Command[]
}

const emit = defineEmits<{
  select: [prompt: string]
}>()

const showAll = ref(false)
const activeCategory = ref('monitor')
const customCommands = ref<Command[]>([])

// 命令分类
const commandCategories = computed<CommandCategory[]>(() => [
  {
    key: 'monitor',
    label: '监控',
    commands: [
      {
        name: '系统概览',
        description: '查看整体系统状态',
        prompt: '显示系统整体概览，包括 CPU、内存、磁盘、网络',
        preview: 'top -bn1 | head -20',
        icon: Monitor,
        color: 'linear-gradient(135deg, #667eea, #764ba2)',
        shortcut: 'Ctrl+1'
      },
      {
        name: 'CPU 使用率',
        description: '查看 CPU 详细使用情况',
        prompt: '显示 CPU 使用率详情，包括各核心使用情况',
        preview: 'mpstat 1 5',
        icon: TrendCharts,
        color: 'linear-gradient(135deg, #f093fb, #f5576c)'
      },
      {
        name: '内存分析',
        description: '分析内存使用详情',
        prompt: '显示内存使用详情，找出占用内存最多的进程',
        preview: 'free -h && ps aux --sort=-%mem | head -10',
        icon: Monitor,
        color: 'linear-gradient(135deg, #4facfe, #00f2fe)'
      },
      {
        name: '磁盘检查',
        description: '检查磁盘空间和 IO',
        prompt: '检查磁盘使用情况和 IO 状态',
        preview: 'df -h && iostat -x 1 3',
        icon: FolderOpened,
        color: 'linear-gradient(135deg, #43e97b, #38f9d7)'
      }
    ]
  },
  {
    key: 'log',
    label: '日志',
    commands: [
      {
        name: '实时日志',
        description: '查看系统实时日志',
        prompt: '显示系统实时日志，监控错误和警告',
        preview: 'tail -f /var/log/syslog',
        icon: Document,
        color: 'linear-gradient(135deg, #fa709a, #fee140)'
      },
      {
        name: '错误日志',
        description: '查找最近的错误',
        prompt: '查找最近 1 小时内的所有错误日志',
        preview: 'journalctl -p err -since "1 hour ago"',
        icon: Warning,
        color: 'linear-gradient(135deg, #f5576c, #f093fb)'
      },
      {
        name: '服务日志',
        description: '查看特定服务日志',
        prompt: '查看指定服务的日志',
        icon: Document,
        color: 'linear-gradient(135deg, #a1c4fd, #c2e9fb)'
      },
      {
        name: '日志分析',
        description: '统计分析日志',
        prompt: '对日志进行统计分析，找出常见问题',
        icon: TrendCharts,
        color: 'linear-gradient(135deg, #d4fc79, #96e6a1)'
      }
    ]
  },
  {
    key: 'operation',
    label: '操作',
    commands: [
      {
        name: '重启服务',
        description: '安全重启指定服务',
        prompt: '帮我重启 [服务名]，并检查状态',
        icon: Setting,
        color: 'linear-gradient(135deg, #a8edea, #fed6e3)'
      },
      {
        name: '进程管理',
        description: '查看和管理进程',
        prompt: '显示所有运行中的进程，并找出占用资源最多的',
        preview: 'ps aux | sort -nk3 | tail -10',
        icon: Tools,
        color: 'linear-gradient(135deg, #ff9a9e, #fecfef)'
      },
      {
        name: '端口检查',
        description: '检查端口占用情况',
        prompt: '检查指定端口的占用情况',
        preview: 'netstat -tlnp | grep :8080',
        icon: Search,
        color: 'linear-gradient(135deg, #667eea, #764ba2)'
      },
      {
        name: '网络连接',
        description: '查看网络连接状态',
        prompt: '显示当前所有网络连接',
        preview: 'ss -s',
        icon: Monitor,
        color: 'linear-gradient(135deg, #e0c3fc, #8ec5fc)'
      }
    ]
  },
  {
    key: 'troubleshoot',
    label: '故障排查',
    commands: [
      {
        name: '性能诊断',
        description: '诊断系统性能问题',
        prompt: '帮我诊断系统性能问题，找出瓶颈',
        icon: TrendCharts,
        color: 'linear-gradient(135deg, #fddb92, #d1fdff)'
      },
      {
        name: '磁盘清理',
        description: '查找可清理的文件',
        prompt: '查找占用空间大的临时文件和日志',
        icon: FolderOpened,
        color: 'linear-gradient(135deg, #9890e3, #b1f4cf)'
      },
      {
        name: '安全检查',
        description: '检查系统安全状态',
        prompt: '检查系统安全状态，包括登录记录、防火墙等',
        icon: Warning,
        color: 'linear-gradient(135deg, #ffecd2, #fcb69f)'
      },
      {
        name: '备份检查',
        description: '检查备份任务状态',
        prompt: '检查备份任务的执行情况',
        icon: Document,
        color: 'linear-gradient(135deg, #a1c4fd, #c2e9fb)'
      }
    ]
  }
])

const handleCommandClick = (cmd: Command) => {
  emit('select', cmd.prompt)
}
</script>

<style scoped>
.quick-commands {
  margin: 12px 20px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  overflow: hidden;
}

.commands-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: linear-gradient(to right, #f9fafb, #fff);
  border-bottom: 1px solid #e5e7eb;
  cursor: pointer;
  user-select: none;
}

.commands-header:hover {
  background: linear-gradient(to right, #f3f4f6, #f9fafb);
}

.header-title {
  flex: 1;
  font-weight: 600;
  color: #374151;
  font-size: 14px;
}

.toggle-btn {
  color: #6b7280;
}

.commands-body {
  padding: 16px;
}

.category-tabs {
  border: none;
}

.category-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
  background: #f9fafb;
  border-radius: 8px;
  padding: 4px;
}

.commands-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.command-card {
  display: flex;
  flex-direction: column;
  padding: 14px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s;
}

.command-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
  transform: translateY(-2px);
}

.command-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 10px;
}

.command-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  color: #fff;
  flex-shrink: 0;
}

.command-info {
  flex: 1;
  min-width: 0;
}

.command-name {
  font-weight: 500;
  color: #1f2937;
  font-size: 14px;
  margin-bottom: 2px;
}

.command-desc {
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
}

.command-preview {
  background: #1e293b;
  border-radius: 6px;
  padding: 8px 10px;
  margin: 8px 0;
  overflow: hidden;
}

.command-preview code {
  color: #e2e8f0;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.command-shortcut {
  margin-top: 8px;
}

.custom-commands {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.custom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-weight: 500;
  color: #374151;
  font-size: 13px;
}

.empty-state {
  padding: 24px;
  text-align: center;
  color: #9ca3af;
  font-size: 13px;
}

@media (max-width: 768px) {
  .commands-grid {
    grid-template-columns: 1fr;
  }
}
</style>
