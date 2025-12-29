import { ElMessage, ElNotification } from 'element-plus'

/**
 * 错误类型枚举
 */
export enum ErrorType {
  NETWORK = 'NETWORK_ERROR',
  SERVER = 'SERVER_ERROR',
  BUSINESS = 'BUSINESS_ERROR',
  VALIDATION = 'VALIDATION_ERROR',
  TIMEOUT = 'TIMEOUT_ERROR',
  UNKNOWN = 'UNKNOWN_ERROR'
}

/**
 * 错误级别
 */
export enum ErrorLevel {
  INFO = 'info',
  WARNING = 'warning',
  ERROR = 'error',
  FATAL = 'fatal'
}

/**
 * 自定义错误类
 */
export class AppError extends Error {
  public readonly type: ErrorType
  public readonly level: ErrorLevel
  public readonly userMessage: string
  public readonly originalError?: any
  public readonly statusCode?: number
  public readonly retryable: boolean

  constructor({
    message,
    type = ErrorType.UNKNOWN,
    level = ErrorLevel.ERROR,
    userMessage,
    originalError,
    statusCode,
    retryable = false
  }: {
    message: string
    type?: ErrorType
    level?: ErrorLevel
    userMessage?: string
    originalError?: any
    statusCode?: number
    retryable?: boolean
  }) {
    super(message)
    this.name = 'AppError'
    this.type = type
    this.level = level
    this.userMessage = userMessage || this.getDefaultUserMessage()
    this.originalError = originalError
    this.statusCode = statusCode
    this.retryable = retryable

    // 保持原型链
    Object.setPrototypeOf(this, AppError.prototype)
  }

  private getDefaultUserMessage(): string {
    switch (this.type) {
      case ErrorType.NETWORK:
        return '网络连接失败，请检查网络设置'
      case ErrorType.SERVER:
        return '服务器异常，请稍后重试'
      case ErrorType.TIMEOUT:
        return '请求超时，请稍后重试'
      case ErrorType.VALIDATION:
        return '数据格式错误，请检查输入'
      case ErrorType.BUSINESS:
        return this.message
      default:
        return '系统异常，请联系管理员'
    }
  }

  /**
   * 从 Axios 错误创建 AppError
   */
  static fromAxiosError(error: any): AppError {
    if (!error.response) {
      // 网络错误或超时
      if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
        return new AppError({
          message: error.message || '请求超时',
          type: ErrorType.TIMEOUT,
          level: ErrorLevel.WARNING,
          originalError: error,
          retryable: true
        })
      }
      return new AppError({
        message: error.message || '网络错误',
        type: ErrorType.NETWORK,
        level: ErrorLevel.ERROR,
        originalError: error,
        retryable: true
      })
    }

    const { status, data } = error.response
    const message = data?.message || data?.msg || error.message

    switch (status) {
      case 400:
        return new AppError({
          message,
          type: ErrorType.VALIDATION,
          level: ErrorLevel.WARNING,
          statusCode: status,
          originalError: error,
          userMessage: data?.message || '请求参数错误'
        })
      case 401:
        return new AppError({
          message: '未授权，请重新登录',
          type: ErrorType.BUSINESS,
          level: ErrorLevel.WARNING,
          statusCode: status,
          originalError: error,
          userMessage: '登录已过期，请重新登录',
          retryable: false
        })
      case 403:
        return new AppError({
          message: '无权限访问',
          type: ErrorType.BUSINESS,
          level: ErrorLevel.WARNING,
          statusCode: status,
          originalError: error,
          userMessage: '您没有权限执行此操作'
        })
      case 404:
        return new AppError({
          message: '资源不存在',
          type: ErrorType.BUSINESS,
          level: ErrorLevel.WARNING,
          statusCode: status,
          originalError: error,
          userMessage: '请求的资源不存在'
        })
      case 422:
        return new AppError({
          message,
          type: ErrorType.VALIDATION,
          level: ErrorLevel.WARNING,
          statusCode: status,
          originalError: error,
          userMessage: data?.message || '数据验证失败'
        })
      case 500:
      case 502:
      case 503:
      case 504:
        return new AppError({
          message,
          type: ErrorType.SERVER,
          level: ErrorLevel.ERROR,
          statusCode: status,
          originalError: error,
          retryable: true
        })
      default:
        return new AppError({
          message,
          type: ErrorType.UNKNOWN,
          level: ErrorLevel.ERROR,
          statusCode: status,
          originalError: error
        })
    }
  }
}

/**
 * 错误处理器配置
 */
export interface ErrorHandlerConfig {
  enableConsoleLog: boolean
  enableNotification: boolean
  enableMessage: boolean
  onError?: (error: AppError) => void
}

/**
 * 默认配置
 */
const defaultConfig: ErrorHandlerConfig = {
  enableConsoleLog: true,
  enableNotification: false,
  enableMessage: true
}

class ErrorHandler {
  private config: ErrorHandlerConfig = defaultConfig
  private errorQueue: Map<string, number> = new Map()
  private readonly ERROR_THROTTLE_TIME = 5000 // 相同错误5秒内只显示一次

  /**
   * 更新配置
   */
  setConfig(config: Partial<ErrorHandlerConfig>) {
    this.config = { ...this.config, ...config }
  }

  /**
   * 处理错误
   */
  handle(error: any, showMessage: boolean = true): AppError {
    // 转换为 AppError
    const appError = error instanceof AppError ? error : AppError.fromAxiosError(error)

    // 控制台日志
    if (this.config.enableConsoleLog) {
      this.logToConsole(appError)
    }

    // 节流：相同错误短时间内只显示一次
    const errorKey = this.getErrorKey(appError)
    if (this.shouldShowError(errorKey)) {
      // 显示错误提示
      if (showMessage) {
        this.showErrorNotification(appError)
      }

      // 记录错误时间
      this.errorQueue.set(errorKey, Date.now())
    }

    // 触发回调
    if (this.config.onError) {
      try {
        this.config.onError(appError)
      } catch (err) {
        console.error('Error handler callback failed:', err)
      }
    }

    return appError
  }

  /**
   * 生成错误唯一标识
   */
  private getErrorKey(error: AppError): string {
    return `${error.type}_${error.statusCode}_${error.message}`
  }

  /**
   * 判断是否应该显示错误（节流）
   */
  private shouldShowError(errorKey: string): boolean {
    const lastTime = this.errorQueue.get(errorKey)
    if (!lastTime) return true
    return Date.now() - lastTime > this.ERROR_THROTTLE_TIME
  }

  /**
   * 控制台日志
   */
  private logToConsole(error: AppError) {
    const logStyle = {
      [ErrorLevel.INFO]: 'color: #409EFF',
      [ErrorLevel.WARNING]: 'color: #E6A23C',
      [ErrorLevel.ERROR]: 'color: #F56C6C',
      [ErrorLevel.FATAL]: 'color: #F56C6C; font-weight: bold'
    }

    console.group(
      `%c[${error.level}] ${error.type}`,
      logStyle[error.level]
    )
    console.error('Message:', error.message)
    console.error('User Message:', error.userMessage)
    if (error.statusCode) console.error('Status:', error.statusCode)
    if (error.retryable) console.log('此错误可重试')
    if (error.originalError) console.error('Original Error:', error.originalError)
    console.groupEnd()
  }

  /**
   * 显示错误通知
   */
  private showErrorNotification(error: AppError) {
    if (this.config.enableNotification && error.level === ErrorLevel.FATAL) {
      ElNotification({
        title: '系统错误',
        message: error.userMessage,
        type: 'error',
        duration: 0
      })
    }

    if (this.config.enableMessage) {
      const messageType = {
        [ErrorLevel.INFO]: 'success' as const,
        [ErrorLevel.WARNING]: 'warning' as const,
        [ErrorLevel.ERROR]: 'error' as const,
        [ErrorLevel.FATAL]: 'error' as const
      }[error.level]

      ElMessage({
        message: error.userMessage,
        type: messageType,
        duration: error.level === ErrorLevel.FATAL ? 0 : 3000,
        showClose: true
      })
    }
  }

  /**
   * 清理过期的错误记录
   */
  clearErrorHistory() {
    const now = Date.now()
    for (const [key, time] of this.errorQueue.entries()) {
      if (now - time > this.ERROR_THROTTLE_TIME) {
        this.errorQueue.delete(key)
      }
    }
  }
}

// 创建全局单例
export const errorHandler = new ErrorHandler()

/**
 * 全局错误处理函数
 */
export function handleError(error: any, showMessage: boolean = true): AppError {
  return errorHandler.handle(error, showMessage)
}

/**
 * 异步函数包装器，自动处理错误
 */
export function withErrorHandling<T extends (...args: any[]) => any>(
  fn: T,
  showError: boolean = true
): T {
  return ((...args: any[]) => {
    try {
      const result = fn(...args)
      // 如果是 Promise，catch 错误
      if (result && typeof result.catch === 'function') {
        return result.catch((error: any) => {
          handleError(error, showError)
          throw error
        })
      }
      return result
    } catch (error) {
      handleError(error, showError)
      throw error
    }
  }) as T
}

/**
 * 创建带重试的异步函数
 */
export async function withRetry<T>(
  fn: () => Promise<T>,
  options: {
    maxRetries?: number
    retryDelay?: number
    backoffMultiplier?: number
    onRetry?: (error: AppError, attempt: number) => void
    shouldRetry?: (error: AppError) => boolean
  } = {}
): Promise<T> {
  const {
    maxRetries = 3,
    retryDelay = 1000,
    backoffMultiplier = 2,
    onRetry,
    shouldRetry
  } = options

  let lastError: AppError | null = null

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn()
    } catch (error) {
      const appError = error instanceof AppError ? error : handleError(error, false)

      // 检查是否应该重试
      const canRetry = shouldRetry ? shouldRetry(appError) : appError.retryable
      const isLastAttempt = attempt === maxRetries

      if (!canRetry || isLastAttempt) {
        handleError(appError)
        throw appError
      }

      lastError = appError

      // 触发重试回调
      if (onRetry) {
        onRetry(appError, attempt + 1)
      }

      // 指数退避
      const delay = retryDelay * Math.pow(backoffMultiplier, attempt)
      await new Promise(resolve => setTimeout(resolve, delay))
    }
  }

  throw lastError
}

// 定期清理错误历史
setInterval(() => {
  errorHandler.clearErrorHistory()
}, 10000)
