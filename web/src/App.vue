<template>
  <ErrorBoundary>
    <el-container class="app-container">
      <!-- 可折叠侧边栏 -->
      <el-aside
        :width="sidebarCollapsed ? '64px' : '220px'"
        class="app-aside"
        :class="{ collapsed: sidebarCollapsed }"
      >
        <AppSidebar :collapsed="sidebarCollapsed" />
      </el-aside>

      <el-container>
        <el-header class="app-header">
          <!-- 侧边栏折叠按钮 -->
          <el-button
            :icon="sidebarCollapsed ? Expand : Fold"
            text
            @click="toggleSidebar"
            class="collapse-btn"
          />
          <AppHeader />
        </el-header>

        <!-- 顶部进度条 -->
        <div class="page-progress" v-if="isLoading">
          <div class="progress-bar" :style="{ width: progress + '%' }"></div>
        </div>

        <el-main class="app-main">
          <ErrorBoundary>
            <router-view v-slot="{ Component }">
              <transition name="fade" mode="out-in">
                <component :is="Component" />
              </transition>
            </router-view>
          </ErrorBoundary>
        </el-main>
      </el-container>
    </el-container>
  </ErrorBoundary>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Expand, Fold } from '@element-plus/icons-vue'
import AppHeader from '@/components/common/AppHeader.vue'
import AppSidebar from '@/components/common/AppSidebar.vue'
import ErrorBoundary from '@/components/common/ErrorBoundary.vue'
import { useAppStore } from '@/stores/app'

const route = useRoute()
const appStore = useAppStore()

const { sidebarCollapsed, toggleSidebar } = appStore

const isLoading = ref(false)
const progress = ref(0)

// 监听路由变化，显示进度
watch(() => route.path, () => {
  isLoading.value = true
  progress.value = 0

  const timer = setInterval(() => {
    progress.value += 10
    if (progress.value >= 90) {
      clearInterval(timer)
    }
  }, 50)

  setTimeout(() => {
    progress.value = 100
    setTimeout(() => {
      isLoading.value = false
    }, 200)
  }, 500)
})
</script>

<style>
/* 侧边栏折叠动画 */
.app-aside {
  transition: width var(--transition-base);
}

.app-aside.collapsed :deep(.sidebar-text) {
  opacity: 0;
  width: 0;
  overflow: hidden;
}

/* 页面切换动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--transition-base), transform var(--transition-base);
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

/* 顶部进度条 */
.page-progress {
  position: fixed;
  top: 60px;
  left: 0;
  right: 0;
  height: 2px;
  background: transparent;
  z-index: 1000;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, var(--color-primary), var(--color-primary-light));
  transition: width var(--transition-base);
}

/* 折叠按钮 */
.collapse-btn {
  margin-right: var(--spacing-3);
}
</style>

<style scoped>
.app-container {
  height: 100%;
}

.app-aside {
  background: linear-gradient(180deg, var(--sidebar-bg-start) 0%, var(--sidebar-bg-end) 100%);
  border-right: 1px solid var(--sidebar-border);
}

.app-header {
  background: #fff;
  border-bottom: 1px solid var(--color-gray-200);
  display: flex;
  align-items: center;
  padding: 0 var(--spacing-5);
  height: 60px;
}

.app-main {
  background: var(--color-gray-50);
  padding: var(--spacing-5);
  overflow-y: auto;
}
</style>
