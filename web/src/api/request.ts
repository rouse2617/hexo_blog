import axios, { AxiosError } from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { handleError, AppError, withRetry } from '@/utils/errorHandler'
import { isSlowNetwork } from '@/utils/networkMonitor'

/**
 * 请求配置接口
 */
interface RequestConfig extends AxiosRequestConfig {
  skipErrorHandler?: boolean // 跳过错误处理
  retry?: boolean // 是否启用重试
  retryCount?: number // 重试次数
  skipRetry?: boolean // 跳过重试
}

/**
 * 重试配置
 */
interface RetryConfig {
  maxRetries: number
  retryDelay: number
  retryCondition?: (error: AxiosError) => boolean
}

// 默认重试配置
const defaultRetryConfig: RetryConfig = {
  maxRetries: 3,
  retryDelay: 1000,
  retryCondition: (error: AxiosError) => {
    // 网络错误、超时、5xx 错误可重试
    return !error.response ||
      error.code === 'ECONNABORTED' ||
      error.code === 'ETIMEDOUT' ||
      (error.response.status >= 500 && error.response.status < 600)
  }
}

// 创建 axios 实例
const service: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 可以在这里添加 token 等认证信息
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // 如果是慢速网络，增加超时时间
    if (isSlowNetwork()) {
      config.timeout = 60000 // 60秒
    }

    // 添加请求开始时间（用于计算请求耗时）
    ;(config as any).metadata = { startTime: Date.now() }

    return config
  },
  (error: AxiosError) => {
    const appError = handleError(error)
    return Promise.reject(appError)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    // 计算请求耗时
    const metadata = (response.config as any).metadata
    if (metadata) {
      const duration = Date.now() - metadata.startTime
      if (duration > 5000) {
        console.warn(`[Slow Request] ${response.config.url?.split('?')[0]} took ${duration}ms`)
      }
    }

    const res = response.data

    // 如果返回的是文件流或其他非 JSON 数据
    if (response.config.responseType === 'blob') {
      return response
    }

    // 统一处理业务错误（当后端返回 { code, message, data } 格式时）
    if (res.code !== undefined && res.code !== 0 && res.code !== 200) {
      const error = new AppError({
        message: res.message || '请求失败',
        type: 'BUSINESS_ERROR' as any,
        level: 'WARNING' as any,
        userMessage: res.message || '请求失败',
        statusCode: res.code,
        retryable: false
      })

      const config = response.config as RequestConfig
      if (!config.skipErrorHandler) {
        handleError(error)
      }

      return Promise.reject(error)
    }

    // 如果有 data 字段，返回 data；否则返回整个响应
    return res.data !== undefined ? res.data : res
  },
  async (error: AxiosError) => {
    const config = error.config as RequestConfig & { _retry?: number; _retryCount?: number }

    // 检查是否应该重试
    const shouldRetry = config.retry !== false &&
                        !config.skipRetry &&
                        defaultRetryConfig.retryCondition?.(error) &&
                        (config._retryCount || 0) < (config.retryCount || defaultRetryConfig.maxRetries)

    if (shouldRetry) {
      // 更新重试计数
      config._retryCount = (config._retryCount || 0) + 1

      console.info(`[Retry] ${config.url?.split('?')[0]} - Attempt ${config._retryCount}`)

      // 指数退避延迟
      const delay = defaultRetryConfig.retryDelay * Math.pow(2, config._retryCount - 1)
      await new Promise(resolve => setTimeout(resolve, delay))

      // 重新发起请求
      return service(config)
    }

    // 处理错误
    const appError = handleError(error, !config.skipErrorHandler)

    return Promise.reject(appError)
  }
)

/**
 * 封装请求方法
 */
export const request = {
  get<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return service.get(url, config)
  },

  post<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return service.post(url, data, config)
  },

  put<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return service.put(url, data, config)
  },

  delete<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return service.delete(url, config)
  },

  /**
   * 文件上传
   */
  upload<T = any>(url: string, file: File | FormData, config?: RequestConfig): Promise<T> {
    const formData = file instanceof FormData ? file : new FormData()
    if (file instanceof File) {
      formData.append('file', file)
    }

    return service.post(url, formData, {
      ...config,
      headers: {
        'Content-Type': 'multipart/form-data',
        ...config?.headers
      }
    })
  },

  /**
   * 文件下载
   */
  download(url: string, config?: RequestConfig): Promise<Blob> {
    return service.get(url, {
      ...config,
      responseType: 'blob'
    })
  },

  /**
   * 带重试的请求
   */
  async withRetry<T>(
    fn: () => Promise<T>,
    options?: {
      maxRetries?: number
      retryDelay?: number
      onRetry?: (error: AppError, attempt: number) => void
    }
  ): Promise<T> {
    return withRetry(fn, options)
  }
}

/**
 * 设置请求重试配置
 */
export function setRetryConfig(config: Partial<RetryConfig>) {
  Object.assign(defaultRetryConfig, config)
}

/**
 * 取消所有请求
 */
export function cancelAllRequests() {
  // 可以实现一个请求队列来取消所有进行中的请求
  console.warn('Cancel all requests called - implement request queue if needed')
}

/**
 * 创建取消令牌
 */
export function createCancelToken() {
  const source = axios.CancelToken.source()
  return {
    token: source.token,
    cancel: source.cancel
  }
}

export default service
