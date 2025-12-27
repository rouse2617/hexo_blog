import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Script } from '@/api/script'
import {
  getScripts,
  getScript,
  createScript,
  updateScript,
  deleteScript,
  toggleScript,
  uploadScript,
  testScript
} from '@/api/script'

export const useScriptStore = defineStore('script', () => {
  const scripts = ref<Script[]>([])
  const total = ref(0)
  const loading = ref(false)
  const currentScript = ref<Script | null>(null)

  // 加载脚本列表
  async function loadScripts(params?: { page?: number; pageSize?: number; keyword?: string }) {
    loading.value = true
    try {
      const data = await getScripts(params)
      scripts.value = data.list
      total.value = data.total
    } catch (error) {
      console.error('加载脚本列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 获取单个脚本
  async function fetchScript(id: string) {
    try {
      currentScript.value = await getScript(id)
      return currentScript.value
    } catch (error) {
      console.error('获取脚本详情失败:', error)
      return null
    }
  }

  // 添加脚本
  async function addScript(data: Omit<Script, 'id' | 'createdAt' | 'updatedAt'>) {
    try {
      const newScript = await createScript(data)
      scripts.value.unshift(newScript)
      total.value++
      return newScript
    } catch (error) {
      console.error('添加脚本失败:', error)
      throw error
    }
  }

  // 更新脚本
  async function editScript(id: string, data: Partial<Script>) {
    try {
      const updated = await updateScript(id, data)
      const index = scripts.value.findIndex(s => s.id === id)
      if (index !== -1) {
        scripts.value[index] = updated
      }
      return updated
    } catch (error) {
      console.error('更新脚本失败:', error)
      throw error
    }
  }

  // 删除脚本
  async function removeScript(id: string) {
    try {
      await deleteScript(id)
      scripts.value = scripts.value.filter(s => s.id !== id)
      total.value--
    } catch (error) {
      console.error('删除脚本失败:', error)
      throw error
    }
  }

  // 切换启用状态
  async function toggle(id: string, enabled: boolean) {
    try {
      await toggleScript(id, enabled)
      const script = scripts.value.find(s => s.id === id)
      if (script) {
        script.enabled = enabled
      }
    } catch (error) {
      console.error('切换脚本状态失败:', error)
      throw error
    }
  }

  // 上传脚本
  async function upload(file: File) {
    try {
      const newScript = await uploadScript(file)
      scripts.value.unshift(newScript)
      total.value++
      return newScript
    } catch (error) {
      console.error('上传脚本失败:', error)
      throw error
    }
  }

  // 测试脚本
  async function test(id: string, params?: Record<string, any>) {
    try {
      return await testScript(id, params)
    } catch (error) {
      console.error('测试脚本失败:', error)
      throw error
    }
  }

  return {
    scripts,
    total,
    loading,
    currentScript,
    loadScripts,
    fetchScript,
    addScript,
    editScript,
    removeScript,
    toggle,
    upload,
    test
  }
})
