import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Host } from '@/api/host'
import {
  getHosts,
  getHost,
  createHost,
  updateHost,
  deleteHost,
  testHostConnection,
  importHosts,
  getAllHosts
} from '@/api/host'

export const useHostStore = defineStore('host', () => {
  const hosts = ref<Host[]>([])
  const allHosts = ref<Host[]>([])
  const total = ref(0)
  const loading = ref(false)
  const currentHost = ref<Host | null>(null)

  // 加载主机列表（分页）
  async function loadHosts(params?: { page?: number; pageSize?: number; keyword?: string }) {
    loading.value = true
    try {
      const data = await getHosts(params)
      hosts.value = data.list
      total.value = data.total
    } catch (error) {
      console.error('加载主机列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 加载所有主机（用于选择器）
  async function loadAllHosts() {
    try {
      const data = await getAllHosts()
      allHosts.value = data
    } catch (error) {
      console.error('加载所有主机失败:', error)
    }
  }

  // 获取单个主机
  async function fetchHost(id: string) {
    try {
      currentHost.value = await getHost(id)
      return currentHost.value
    } catch (error) {
      console.error('获取主机详情失败:', error)
      return null
    }
  }

  // 添加主机
  async function addHost(data: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>) {
    try {
      const newHost = await createHost(data)
      hosts.value.unshift(newHost)
      total.value++
      return newHost
    } catch (error) {
      console.error('添加主机失败:', error)
      throw error
    }
  }

  // 更新主机
  async function editHost(id: string, data: Partial<Host>) {
    try {
      const updated = await updateHost(id, data)
      const index = hosts.value.findIndex(h => h.id === id)
      if (index !== -1) {
        hosts.value[index] = updated
      }
      return updated
    } catch (error) {
      console.error('更新主机失败:', error)
      throw error
    }
  }

  // 删除主机
  async function removeHost(id: string) {
    try {
      await deleteHost(id)
      hosts.value = hosts.value.filter(h => h.id !== id)
      total.value--
    } catch (error) {
      console.error('删除主机失败:', error)
      throw error
    }
  }

  // 测试连接
  async function testConnection(id: string) {
    try {
      const result = await testHostConnection(id)
      // 更新主机状态
      const host = hosts.value.find(h => h.id === id)
      if (host) {
        host.status = result.success ? 'online' : 'offline'
      }
      return result
    } catch (error) {
      console.error('测试连接失败:', error)
      throw error
    }
  }

  // 批量导入
  async function batchImport(hostsData: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>[]) {
    try {
      const result = await importHosts({ hosts: hostsData })
      await loadHosts()
      return result
    } catch (error) {
      console.error('批量导入失败:', error)
      throw error
    }
  }

  return {
    hosts,
    allHosts,
    total,
    loading,
    currentHost,
    loadHosts,
    loadAllHosts,
    fetchHost,
    addHost,
    editHost,
    removeHost,
    testConnection,
    batchImport
  }
})
