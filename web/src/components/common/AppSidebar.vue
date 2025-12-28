<template>
  <div class="app-sidebar" :class="{ collapsed }">
    <div class="sidebar-header">
      <div class="logo" v-if="!collapsed">
        <div class="logo-icon">
          <el-icon :size="20"><Monitor /></el-icon>
        </div>
        <span class="logo-text sidebar-text">AI Pro</span>
      </div>
      <el-icon v-else :size="24" color="#60a5fa"><Monitor /></el-icon>
    </div>

    <el-menu
      :default-active="activeMenu"
      class="sidebar-menu"
      :collapse="collapsed"
      background-color="transparent"
      text-color="var(--sidebar-text)"
      active-text-color="var(--sidebar-active-text)"
      router
    >
      <el-menu-item
        v-for="item in menuItems"
        :key="item.path"
        :index="item.path"
      >
        <el-icon>
          <component :is="item.icon" />
        </el-icon>
        <template #title>
          <span class="menu-text sidebar-text">{{ item.title }}</span>
        </template>
      </el-menu-item>
    </el-menu>

    <div class="sidebar-footer">
      <div class="version sidebar-text" v-if="!collapsed">v1.0.0</div>
      <el-tooltip v-else content="AI Pro v1.0.0" placement="right">
        <el-icon :size="14" color="#94a3b8"><InfoFilled /></el-icon>
      </el-tooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  ChatDotRound, Monitor, SetUp, Document,
  List, Setting, Operation, InfoFilled
} from '@element-plus/icons-vue'

defineProps<{
  collapsed?: boolean
}>()

const route = useRoute()

const activeMenu = computed(() => route.path)

const menuItems = [
  { path: '/chat', title: '智能对话', icon: ChatDotRound },
  { path: '/hosts', title: '主机管理', icon: Monitor },
  { path: '/tools', title: '工具列表', icon: SetUp },
  { path: '/scripts', title: '脚本管理', icon: Document },
  { path: '/tasks', title: '任务历史', icon: List },
  { path: '/console', title: '控制台', icon: Operation },
  { path: '/settings', title: '安全设置', icon: Setting },
]
</script>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: linear-gradient(180deg, var(--sidebar-bg-start) 0%, var(--sidebar-bg-end) 100%);
  transition: all var(--transition-base);
}

.sidebar-header {
  padding: var(--spacing-5);
  border-bottom: 1px solid var(--sidebar-border);
  display: flex;
  align-items: center;
  justify-content: center;
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.logo-icon {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.logo-text {
  font-size: var(--text-xl);
  font-weight: var(--font-bold);
  color: #fff;
  letter-spacing: 0.5px;
  transition: all var(--transition-base);
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  padding: var(--spacing-3) 0;
}

.sidebar-menu :deep(.el-menu-item) {
  margin: var(--spacing-1) var(--spacing-3);
  border-radius: var(--radius-lg);
  height: 44px;
  line-height: 44px;
  color: var(--sidebar-text);
  transition: all var(--transition-fast);
}

.sidebar-menu :deep(.el-menu-item:hover) {
  background: var(--sidebar-active-bg);
  color: var(--sidebar-text-hover);
  transform: translateX(2px);
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background: var(--sidebar-active-bg);
  color: var(--sidebar-active-text);
  position: relative;
}

.sidebar-menu :deep(.el-menu-item.is-active::before) {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 60%;
  background: #60a5fa;
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.menu-text {
  position: relative;
}

.menu-badge {
  position: absolute;
  right: -8px;
  top: -8px;
}

.sidebar-footer {
  padding: var(--spacing-4);
  border-top: 1px solid var(--sidebar-border);
  display: flex;
  align-items: center;
  justify-content: center;
}

.version {
  color: var(--sidebar-text);
  font-size: var(--text-xs);
  padding: var(--spacing-1) var(--spacing-3);
  background: rgba(255, 255, 255, 0.05);
  border-radius: var(--radius-md);
  transition: all var(--transition-base);
}

/* 折叠状态 */
.collapsed .sidebar-text {
  opacity: 0;
  width: 0;
  overflow: hidden;
}
</style>
