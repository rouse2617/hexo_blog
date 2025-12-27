<template>
  <div class="app-sidebar">
    <div class="sidebar-header">
      <div class="logo">
        <el-icon :size="28" color="#409eff"><Monitor /></el-icon>
        <span class="logo-text">AI Pro</span>
      </div>
    </div>
    <el-menu
      :default-active="activeMenu"
      class="sidebar-menu"
      background-color="transparent"
      text-color="#a0a0b0"
      active-text-color="#409eff"
      router
    >
      <el-menu-item index="/chat">
        <el-icon><ChatDotRound /></el-icon>
        <span>智能对话</span>
      </el-menu-item>
      <el-menu-item index="/hosts">
        <el-icon><Monitor /></el-icon>
        <span>主机管理</span>
      </el-menu-item>
      <el-menu-item index="/tools">
        <el-icon><SetUp /></el-icon>
        <span>工具列表</span>
      </el-menu-item>
      <el-menu-item index="/scripts">
        <el-icon><Document /></el-icon>
        <span>脚本管理</span>
      </el-menu-item>
      <el-menu-item index="/tasks">
        <el-icon><List /></el-icon>
        <span>任务历史</span>
      </el-menu-item>
      <el-menu-item index="/console">
        <el-icon><Operation /></el-icon>
        <span>控制台</span>
      </el-menu-item>
      <el-menu-item index="/settings">
        <el-icon><Setting /></el-icon>
        <span>安全设置</span>
      </el-menu-item>
    </el-menu>
    <div class="sidebar-footer">
      <div class="version">v1.0.0</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { ChatDotRound, Monitor, SetUp, Document, List, Setting, Operation } from '@element-plus/icons-vue'

const route = useRoute()

const activeMenu = computed(() => route.path)
</script>

<style scoped>
.app-sidebar {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #2a2a4a;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-text {
  font-size: 20px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  padding: 10px 0;
}

.sidebar-menu .el-menu-item {
  margin: 4px 10px;
  border-radius: 8px;
  height: 48px;
  line-height: 48px;
}

.sidebar-menu .el-menu-item:hover {
  background-color: rgba(64, 158, 255, 0.1) !important;
}

.sidebar-menu .el-menu-item.is-active {
  background-color: rgba(64, 158, 255, 0.2) !important;
  color: #409eff !important;
}

.sidebar-footer {
  padding: 15px 20px;
  border-top: 1px solid #2a2a4a;
}

.version {
  color: #606266;
  font-size: 12px;
  text-align: center;
}
</style>
