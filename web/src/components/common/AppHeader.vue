<template>
  <div class="app-header-content">
    <div class="header-left">
      <h2 class="page-title">{{ pageTitle }}</h2>
    </div>

    <div class="header-right">
      <!-- 搜索框 -->
      <el-input
        v-model="searchKeyword"
        placeholder="搜索... (Ctrl+K)"
        :prefix-icon="Search"
        clearable
        class="header-search"
        @focus="showSearchPanel = true"
      />

      <!-- 刷新 -->
      <el-tooltip content="刷新" placement="bottom">
        <el-button
          :icon="Refresh"
          circle
          :loading="refreshing"
          @click="handleRefresh"
        />
      </el-tooltip>

      <!-- 全屏 -->
      <el-tooltip content="全屏" placement="bottom">
        <el-button :icon="FullScreen" circle @click="toggleFullscreen" />
      </el-tooltip>

      <!-- 用户菜单 -->
      <el-dropdown trigger="click">
        <el-button circle>
          <el-icon :size="18"><User /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>
              <el-icon><User /></el-icon>
              个人设置
            </el-dropdown-item>
            <el-dropdown-item divided>
              <el-icon><SwitchButton /></el-icon>
              退出登录
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Refresh, FullScreen, User, Search, SwitchButton } from '@element-plus/icons-vue'

const route = useRoute()
const searchKeyword = ref('')
const refreshing = ref(false)
const showSearchPanel = ref(false)

const pageTitle = computed(() => {
  return (route.meta.title as string) || 'AI Pro'
})

const handleRefresh = () => {
  refreshing.value = true
  setTimeout(() => {
    window.location.reload()
  }, 500)
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
  gap: var(--spacing-6);
}

.header-left {
  flex: 1;
  min-width: 0;
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-semibold);
  color: var(--color-gray-800);
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  flex-shrink: 0;
}

.header-search {
  width: 200px;
}

.header-search :deep(.el-input__wrapper) {
  border-radius: var(--radius-full);
  background: var(--color-gray-100);
  border: none;
  box-shadow: none;
}

.header-search :deep(.el-input__wrapper:focus) {
  background: var(--color-gray-200);
}

.header-search :deep(.el-input__inner) {
  color: var(--color-gray-700);
}

.header-search :deep(.el-input__inner::placeholder) {
  color: var(--color-gray-500);
}
</style>
