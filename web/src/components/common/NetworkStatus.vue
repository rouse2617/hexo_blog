<template>
  <div class="network-status">
    <el-tooltip :content="statusText" placement="bottom">
      <div :class="['network-indicator', statusClass]">
        <el-icon>
          <component :is="statusIcon" />
        </el-icon>
        <span v-if="showText" class="network-text">{{ statusText }}</span>
      </div>
    </el-tooltip>

    <!-- 网络详情弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      title="网络状态详情"
      width="400px"
    >
      <div class="network-details">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="状态">
            <el-tag :type="networkInfo.online ? 'success' : 'danger'">
              {{ networkInfo.online ? '在线' : '离线' }}
            </el-tag>
          </el-descriptions-item>

          <el-descriptions-item label="质量">
            <el-tag :type="qualityTagType">
              {{ qualityText }}
            </el-tag>
          </el-descriptions-item>

          <el-descriptions-item v-if="networkInfo.effectiveType" label="网络类型">
            {{ networkInfo.effectiveType.toUpperCase() }}
          </el-descriptions-item>

          <el-descriptions-item v-if="networkInfo.downlink" label="下行速度">
            {{ networkInfo.downlink }} Mbps
          </el-descriptions-item>

          <el-descriptions-item v-if="networkInfo.rtt" label="往返时间">
            {{ networkInfo.rtt }} ms
          </el-descriptions-item>

          <el-descriptions-item v-if="networkInfo.saveData !== undefined" label="省流模式">
            <el-tag :type="networkInfo.saveData ? 'warning' : 'info'">
              {{ networkInfo.saveData ? '已开启' : '未开启' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <div class="network-actions">
          <el-button type="primary" @click="handleRefresh">
            刷新状态
          </el-button>
          <el-button @click="dialogVisible = false">
            关闭
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useNetworkMonitor, NetworkQuality } from '@/utils/networkMonitor'
import { Connection, Warning, CircleClose } from '@element-plus/icons-vue'

interface Props {
  showText?: boolean
  clickable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showText: false,
  clickable: true
})

const { networkInfo, refresh } = useNetworkMonitor()
const dialogVisible = ref(false)

// 状态文本
const statusText = computed(() => {
  if (!networkInfo.value.online) {
    return '网络已断开'
  }
  if (networkInfo.value.quality === NetworkQuality.POOR) {
    return '网络较差'
  }
  return '网络正常'
})

// 状态类名
const statusClass = computed(() => {
  if (!networkInfo.value.online) return 'is-offline'
  if (networkInfo.value.quality === NetworkQuality.POOR) return 'is-poor'
  return 'is-online'
})

// 状态图标
const statusIcon = computed(() => {
  if (!networkInfo.value.online) return CircleClose
  if (networkInfo.value.quality === NetworkQuality.POOR) return Warning
  return Connection
})

// 质量文本
const qualityText = computed(() => {
  const qualityMap = {
    [NetworkQuality.EXCELLENT]: '优秀',
    [NetworkQuality.GOOD]: '良好',
    [NetworkQuality.FAIR]: '一般',
    [NetworkQuality.POOR]: '较差'
  }
  return qualityMap[networkInfo.value.quality] || '未知'
})

// 质量标签类型
const qualityTagType = computed(() => {
  const typeMap = {
    [NetworkQuality.EXCELLENT]: 'success',
    [NetworkQuality.GOOD]: 'success',
    [NetworkQuality.FAIR]: 'warning',
    [NetworkQuality.POOR]: 'danger'
  }
  return typeMap[networkInfo.value.quality] || 'info'
})

// 刷新状态
const handleRefresh = () => {
  refresh()
}

// 点击处理
const handleClick = () => {
  if (props.clickable) {
    dialogVisible.value = true
  }
}

// 添加点击事件
onMounted(() => {
  if (props.clickable) {
    document.querySelector('.network-indicator')?.addEventListener('click', handleClick)
  }
})

onUnmounted(() => {
  if (props.clickable) {
    document.querySelector('.network-indicator')?.removeEventListener('click', handleClick)
  }
})
</script>

<script lang="ts">
export default {
  name: 'NetworkStatus'
}
</script>

<style scoped>
.network-status {
  display: inline-block;
}

.network-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s;
  font-size: 14px;
}

.network-indicator:hover {
  background-color: var(--el-fill-color-light);
}

.network-indicator.is-online {
  color: var(--el-color-success);
}

.network-indicator.is-poor {
  color: var(--el-color-warning);
}

.network-indicator.is-offline {
  color: var(--el-color-danger);
}

.network-text {
  font-size: 12px;
}

.network-details {
  padding: 12px 0;
}

.network-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
}
</style>
