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

  const normalizeHost = (h: Host): Host => {
    const anyHost = h as any
    const id = anyHost?.id ?? anyHost?.name
    return { ...h, id }
  }

  // 加载主机列表（分页）
  async function loadHosts(params?: { page?: number; pageSize?: number; keyword?: string }) {
    loading.value = true
    try {
      const data = await getHosts(params)
      hosts.value = (data.list || []).map(normalizeHost)
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
      allHosts.value = (data || []).map(normalizeHost)
      // 加载后立即检查状态
      await refreshAllHostStatus()
    } catch (error) {
      console.error('加载所有主机失败:', error)
    }
  }

  // 刷新所有主机状态
  async function refreshAllHostStatus() {
    if (allHosts.value.length === 0) {
      return
    }

    // 批量测试所有主机状态
    const promises = allHosts.value.map(async (host) => {
      try {
        const result = await testHostConnection(host.id || host.name)
        host.status = result.success ? 'online' : 'offline'
        // 同时更新hosts列表中的状态
        const index = hosts.value.findIndex(h => h.id === host.id)
        if (index !== -1) {
          hosts.value[index].status = host.status
        }
      } catch (error) {
        host.status = 'offline'
        const index = hosts.value.findIndex(h => h.id === host.id)
        if (index !== -1) {
          hosts.value[index].status = 'offline'
        }
      }
    })

    await Promise.allSettled(promises)
  }

  // 启动状态自动刷新
  let statusRefreshTimer: number | null = null

  function startStatusAutoRefresh() {
    // 立即执行一次
    refreshAllHostStatus()

    // 每1分钟刷新一次
    statusRefreshTimer = window.setInterval(() => {
      refreshAllHostStatus()
    }, 60000) // 60秒 = 60000毫秒
  }

  function stopStatusAutoRefresh() {
    if (statusRefreshTimer !== null) {
      clearInterval(statusRefreshTimer)
      statusRefreshTimer = null
    }
  }

  // 获取单个主机
  async function fetchHost(id: string) {
    try {
      currentHost.value = normalizeHost(await getHost(id))
      return currentHost.value
    } catch (error) {
      console.error('获取主机详情失败:', error)
      return null
    }
  }

  // 添加主机
  async function addHost(data: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>) {
    try {
      const newHost = normalizeHost(await createHost(data))
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
      const response = await updateHost(id, data) as any
      // Check if response contains valid host data (name and host fields)
      if (response && typeof response === 'object' && 'name' in response && 'host' in response && response.name && response.host) {
        // Response has valid host data
        const updated = normalizeHost(response)
        const index = hosts.value.findIndex(h => h.id === id)
        if (index !== -1) {
          hosts.value[index] = updated
        }
        return updated
      } else {
        // Response is just { code, message } without data
        // Don't modify the hosts array - preserve existing data
        const existing = hosts.value.find(h => h.id === id)
        if (existing) {
          // Update the existing host with the submitted data
          Object.assign(existing, data)
          return existing
        }
        return null
      }
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
    batchImport,
    refreshAllHostStatus,
    startStatusAutoRefresh,
    stopStatusAutoRefresh
  }
})
