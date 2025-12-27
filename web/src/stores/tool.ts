import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Tool } from '@/api/tool'
import { getTools, getBuiltinTools, getScriptTools, toggleTool } from '@/api/tool'

export const useToolStore = defineStore('tool', () => {
  const tools = ref<Tool[]>([])
  const loading = ref(false)

  // 内置工具
  const builtinTools = computed(() => tools.value.filter(t => t.type === 'builtin'))

  // 脚本工具
  const scriptTools = computed(() => tools.value.filter(t => t.type === 'script'))

  // 已启用的工具
  const enabledTools = computed(() => tools.value.filter(t => t.enabled))

  // 加载所有工具
  async function loadTools() {
    loading.value = true
    try {
      const data = await getTools()
      tools.value = data
    } catch (error) {
      console.error('加载工具列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 加载内置工具
  async function loadBuiltinTools() {
    try {
      const data = await getBuiltinTools()
      // 更新内置工具
      const scriptToolsList = tools.value.filter(t => t.type === 'script')
      tools.value = [...data, ...scriptToolsList]
    } catch (error) {
      console.error('加载内置工具失败:', error)
    }
  }

  // 加载脚本工具
  async function loadScriptTools() {
    try {
      const data = await getScriptTools()
      // 更新脚本工具
      const builtinToolsList = tools.value.filter(t => t.type === 'builtin')
      tools.value = [...builtinToolsList, ...data]
    } catch (error) {
      console.error('加载脚本工具失败:', error)
    }
  }

  // 切换工具启用状态
  async function toggle(name: string, enabled: boolean) {
    try {
      await toggleTool(name, enabled)
      const tool = tools.value.find(t => t.name === name)
      if (tool) {
        tool.enabled = enabled
      }
    } catch (error) {
      console.error('切换工具状态失败:', error)
      throw error
    }
  }

  return {
    tools,
    loading,
    builtinTools,
    scriptTools,
    enabledTools,
    loadTools,
    loadBuiltinTools,
    loadScriptTools,
    toggle
  }
})
