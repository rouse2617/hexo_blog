<template>
  <el-dropdown trigger="click" @command="setTheme">
    <el-button class="theme-toggle" :icon="themeIcon" circle />
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          :class="{ 'is-active': currentTheme === 'light' }"
          command="light"
        >
          <el-icon><Sunny /></el-icon>
          <span>Light</span>
        </el-dropdown-item>
        <el-dropdown-item
          :class="{ 'is-active': currentTheme === 'dark' }"
          command="dark"
        >
          <el-icon><Moon /></el-icon>
          <span>Dark</span>
        </el-dropdown-item>
        <el-dropdown-item
          :class="{ 'is-active': currentTheme === 'auto' }"
          command="auto"
        >
          <el-icon><Monitor /></el-icon>
          <span>Auto (System)</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Sunny, Moon, Monitor } from '@element-plus/icons-vue'
import { useTheme, type ThemeMode } from '@/composables/useTheme'

const { currentTheme, appliedTheme, setTheme: changeTheme } = useTheme()

const themeIcon = computed(() => {
  switch (appliedTheme.value) {
    case 'light':
      return Sunny
    case 'dark':
      return Moon
    default:
      return Monitor
  }
})

function setTheme(mode: ThemeMode) {
  changeTheme(mode)
}
</script>

<style scoped>
.theme-toggle {
  transition: all 0.3s ease;
}

.theme-toggle:hover {
  transform: scale(1.1);
}

.el-dropdown-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 140px;
}

.el-dropdown-menu__item.is-active {
  background-color: var(--color-primary-light);
  color: var(--color-primary);
}

.el-dropdown-menu__item .el-icon {
  font-size: 18px;
}
</style>
