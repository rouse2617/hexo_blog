<template>
  <div class="app-header-content">
    <div class="header-left">
      <h2 class="page-title">{{ pageTitle }}</h2>
    </div>
    <div class="header-right">
      <el-tooltip content="刷新" placement="bottom">
        <el-button :icon="Refresh" circle @click="handleRefresh" />
      </el-tooltip>
      <el-tooltip content="全屏" placement="bottom">
        <el-button :icon="FullScreen" circle @click="toggleFullscreen" />
      </el-tooltip>
      <el-dropdown trigger="click">
        <el-button :icon="User" circle />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>个人设置</el-dropdown-item>
            <el-dropdown-item divided>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Refresh, FullScreen, User } from '@element-plus/icons-vue'

const route = useRoute()

const pageTitle = computed(() => {
  return (route.meta.title as string) || 'AI Pro'
})

const handleRefresh = () => {
  window.location.reload()
}

const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }
}
</script>

<style scoped>
.app-header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-left {
  display: flex;
  align-items: center;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
</style>
