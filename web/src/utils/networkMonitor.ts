import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

/**
 * 网络状态类型
 */
export enum NetworkStatus {
  ONLINE = 'online',
  OFFLINE = 'offline',
  SLOW = 'slow'
}

/**
 * 网络质量等级
 */
export enum NetworkQuality {
  EXCELLENT = 'excellent', // 优秀
  GOOD = 'good',           // 良好
  FAIR = 'fair',           // 一般
  POOR = 'poor'            // 较差
}

/**
 * 网络信息接口
 */
interface NetworkInfo {
  online: boolean
  status: NetworkStatus
  quality: NetworkQuality
  effectiveType?: string // 有效连接类型 (4g, 3g, 2g, slow-2g)
  downlink?: number      // 下行速度 (Mbps)
  rtt?: number           // 往返时间 (ms)
  saveData?: boolean     // 节省数据模式
}

// 全局网络状态
const networkInfo = ref<NetworkInfo>({
  online: true,
  status: NetworkStatus.ONLINE,
  quality: NetworkQuality.GOOD
})

// 监听器列表
const listeners: Set<(info: NetworkInfo) => void> = new Set()

/**
 * 网络监控类
 */
class NetworkMonitor {
  private connection: any = null
  private checkInterval: number | null = null
  private lastCheckTime = 0
  private pingUrl = '/api/health' // 健康检查端点

  constructor() {
    this.init()
  }

  /**
   * 初始化网络监控
   */
  private init() {
    // 检查浏览器是否支持 Network Information API
    if ('connection' in navigator) {
      this.connection = (navigator as any).connection
      this.setupConnectionListeners()
    }

    // 监听在线/离线事件
    window.addEventListener('online', this.handleOnline)
    window.addEventListener('offline', this.handleOffline)

    // 初始状态检测
    this.updateNetworkInfo()
    this.startPeriodicCheck()
  }

  /**
   * 设置网络连接变化监听器
   */
  private setupConnectionListeners() {
    if (!this.connection) return

    const events = ['change', 'typechange']
    events.forEach(event => {
      this.connection?.addEventListener(event, () => {
        this.updateNetworkInfo()
      })
    })
  }

  /**
   * 处理网络连接恢复
   */
  private handleOnline = () => {
    this.updateNetworkInfo()
    ElMessage.success('网络已恢复连接')
    this.notifyListeners()
  }

  /**
   * 处理网络断开
   */
  private handleOffline = () => {
    networkInfo.value.online = false
    networkInfo.value.status = NetworkStatus.OFFLINE
    ElMessage.warning('网络连接已断开，请检查网络设置')
    this.notifyListeners()
  }

  /**
   * 更新网络信息
   */
  private updateNetworkInfo() {
    const info = this.getCurrentNetworkInfo()
    networkInfo.value = info
    this.notifyListeners()
  }

  /**
   * 获取当前网络信息
   */
  private getCurrentNetworkInfo(): NetworkInfo {
    const isOnline = navigator.onLine

    if (!isOnline) {
      return {
        online: false,
        status: NetworkStatus.OFFLINE,
        quality: NetworkQuality.POOR
      }
    }

    // 如果支持 Network Information API
    if (this.connection) {
      const quality = this.calculateQuality(this.connection)
      const status = this.calculateStatus(quality)

      return {
        online: true,
        status,
        quality,
        effectiveType: this.connection.effectiveType,
        downlink: this.connection.downlink,
        rtt: this.connection.rtt,
        saveData: this.connection.saveData
      }
    }

    // 默认返回良好状态
    return {
      online: true,
      status: NetworkStatus.ONLINE,
      quality: NetworkQuality.GOOD
    }
  }

  /**
   * 计算网络质量
   */
  private calculateQuality(connection: any): NetworkQuality {
    const { effectiveType, downlink, rtt } = connection

    // 根据有效类型判断
    if (effectiveType === 'slow-2g' || effectiveType === '2g') {
      return NetworkQuality.POOR
    }

    if (effectiveType === '3g') {
      return NetworkQuality.FAIR
    }

    // 根据下行速度判断
    if (downlink !== undefined) {
      if (downlink >= 10) return NetworkQuality.EXCELLENT
      if (downlink >= 5) return NetworkQuality.GOOD
      if (downlink >= 1.5) return NetworkQuality.FAIR
      return NetworkQuality.POOR
    }

    // 根据往返时间判断
    if (rtt !== undefined) {
      if (rtt <= 50) return NetworkQuality.EXCELLENT
      if (rtt <= 150) return NetworkQuality.GOOD
      if (rtt <= 300) return NetworkQuality.FAIR
      return NetworkQuality.POOR
    }

    return NetworkQuality.GOOD
  }

  /**
   * 计算网络状态
   */
  private calculateStatus(quality: NetworkQuality): NetworkStatus {
    if (quality === NetworkQuality.POOR) {
      return NetworkStatus.SLOW
    }
    return NetworkStatus.ONLINE
  }

  /**
   * 通知所有监听器
   */
  private notifyListeners() {
    listeners.forEach(listener => {
      try {
        listener(networkInfo.value)
      } catch (error) {
        console.error('Network listener error:', error)
      }
    })
  }

  /**
   * 启动定期检查
   */
  private startPeriodicCheck() {
    // 每30秒检查一次
    this.checkInterval = setInterval(() => {
      this.checkNetworkHealth()
    }, 30000) as unknown as number
  }

  /**
   * 停止定期检查
   */
  private stopPeriodicCheck() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval)
      this.checkInterval = null
    }
  }

  /**
   * 检查网络健康度
   */
  private async checkNetworkHealth() {
    const now = Date.now()
    const timeSinceLastCheck = now - this.lastCheckTime

    // 避免频繁检查
    if (timeSinceLastCheck < 10000) return

    this.lastCheckTime = now

    try {
      const startTime = Date.now()
      const response = await fetch(this.pingUrl, {
        method: 'HEAD',
        cache: 'no-cache',
        signal: AbortSignal.timeout(5000) // 5秒超时
      })
      const endTime = Date.now()
      const rtt = endTime - startTime

      if (response.ok) {
        networkInfo.value.online = true
        // 根据 RTT 更新质量
        if (rtt <= 50) {
          networkInfo.value.quality = NetworkQuality.EXCELLENT
        } else if (rtt <= 150) {
          networkInfo.value.quality = NetworkQuality.GOOD
        } else if (rtt <= 300) {
          networkInfo.value.quality = NetworkQuality.FAIR
        } else {
          networkInfo.value.quality = NetworkQuality.POOR
          networkInfo.value.status = NetworkStatus.SLOW
        }
      } else {
        networkInfo.value.status = NetworkStatus.SLOW
        networkInfo.value.quality = NetworkQuality.POOR
      }
    } catch (error) {
      console.error('Network health check failed:', error)
      networkInfo.value.status = NetworkStatus.SLOW
      networkInfo.value.quality = NetworkQuality.POOR
    }

    this.notifyListeners()
  }

  /**
   * 手动刷新网络状态
   */
  refresh() {
    this.updateNetworkInfo()
    this.checkNetworkHealth()
  }

  /**
   * 销毁监控
   */
  destroy() {
    this.stopPeriodicCheck()
    window.removeEventListener('online', this.handleOnline)
    window.removeEventListener('offline', this.handleOffline)
    listeners.clear()
  }
}

// 创建全局单例
let monitorInstance: NetworkMonitor | null = null

/**
 * 获取网络监控实例
 */
function getMonitor(): NetworkMonitor {
  if (!monitorInstance) {
    monitorInstance = new NetworkMonitor()
  }
  return monitorInstance
}

/**
 * 网络监控 Hook (用于 Vue 组件)
 */
export function useNetworkMonitor() {
  const info = networkInfo

  // 添加监听器
  const addListener = (callback: (info: NetworkInfo) => void) => {
    listeners.add(callback)
    // 立即回调当前状态
    callback(info.value)
  }

  // 移除监听器
  const removeListener = (callback: (info: NetworkInfo) => void) => {
    listeners.delete(callback)
  }

  onMounted(() => {
    getMonitor()
  })

  onUnmounted(() => {
    // 组件卸载时不销毁全局监控，只移除监听器
  })

  return {
    networkInfo: info,
    addListener,
    removeListener,
    refresh: () => getMonitor().refresh()
  }
}

/**
 * 获取当前网络信息 (非响应式)
 */
export function getNetworkInfo(): NetworkInfo {
  return networkInfo.value
}

/**
 * 检查网络是否在线
 */
export function isOnline(): boolean {
  return networkInfo.value.online
}

/**
 * 检查网络是否慢速
 */
export function isSlowNetwork(): boolean {
  return networkInfo.value.status === NetworkStatus.SLOW ||
         networkInfo.value.quality === NetworkQuality.POOR
}

/**
 * 添加网络状态变化监听器
 */
export function addNetworkListener(callback: (info: NetworkInfo) => void): () => void {
  listeners.add(callback)
  // 立即回调当前状态
  callback(networkInfo.value)

  // 返回取消监听函数
  return () => {
    listeners.delete(callback)
  }
}

/**
 * 初始化网络监控
 */
export function initNetworkMonitor() {
  getMonitor()
}

/**
 * 销毁网络监控
 */
export function destroyNetworkMonitor() {
  if (monitorInstance) {
    monitorInstance.destroy()
    monitorInstance = null
  }
}

export { NetworkMonitor, getMonitor }
