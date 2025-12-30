import { PiniaPluginContext } from 'pinia'

/**
 * 持久化状态配置
 */
export interface PersistOptions {
  /**
   * 存储的 key，默认使用 store.$id
   */
  key?: string

  /**
   * 存储类型，默认 localStorage
   */
  storage?: 'localStorage' | 'sessionStorage'

  /**
   * 需要持久化的 state 路径数组
   * 例如: ['sidebarCollapsed', 'theme', 'user.profile']
   * 如果不指定，则持久化整个 state
   */
  paths?: string[]
}

/**
 * 扩展 Pinia store 定义，支持 persist 配置
 */
declare module 'pinia' {
  export interface DefineStoreOptionsBase<S, Store> {
    persist?: PersistOptions | boolean
  }
}

/**
 * 获取存储对象
 */
function getStorage(type: 'localStorage' | 'sessionStorage'): Storage {
  return type === 'sessionStorage' ? window.sessionStorage : window.localStorage
}

/**
 * 根据路径从对象中获取值
 */
function getValueByPath(obj: any, path: string): any {
  return path.split('.').reduce((acc, part) => acc?.[part], obj)
}

/**
 * 根据路径设置对象中的值
 */
function setValueByPath(obj: any, path: string, value: unknown): void {
  const parts = path.split('.')
  const lastPart = parts.pop()!
  const target = parts.reduce((acc, part) => {
    if (!acc[part]) acc[part] = {}
    return acc[part]
  }, obj)
  target[lastPart] = value
}

/**
 * Pinia 持久化插件
 */
export function createPersistPlugin(options?: {
  global?: boolean
  storage?: 'localStorage' | 'sessionStorage'
}) {
  return (context: PiniaPluginContext) => {
    const { store, options: storeOptions } = context

    // 获取 store 的持久化配置
    const persistConfig = storeOptions.persist as PersistOptions | boolean | undefined

    // 如果全局未开启且 store 未配置持久化，则跳过
    if (!options?.global && !persistConfig) {
      return
    }

    // 解析配置
    const isEnabled = typeof persistConfig === 'boolean' ? persistConfig : true
    if (!isEnabled) return

    const config: PersistOptions = typeof persistConfig === 'object' ? persistConfig : {}
    const key = config.key || `pinia-${store.$id}`
    const storage = getStorage(config.storage || options?.storage || 'localStorage')
    const paths = config.paths

    // 从存储中恢复状态
    const fromStorage = storage.getItem(key)
    if (fromStorage) {
      try {
        const savedState = JSON.parse(fromStorage)

        if (paths) {
          // 只恢复指定的路径
          paths.forEach(path => {
            const value = getValueByPath(savedState, path)
            if (value !== undefined) {
              setValueByPath(store.$state, path, value)
            }
          })
        } else {
          // 恢复整个状态
          store.$patch(savedState)
        }
      } catch (error) {
        console.error(`[PersistPlugin] Failed to restore state for ${store.$id}:`, error)
      }
    }

    // 监听状态变化并保存
    store.$subscribe((_mutation, state) => {
      try {
        let toSave: any

        if (paths) {
          // 只保存指定的路径
          toSave = {}
          paths.forEach(path => {
            const value = getValueByPath(state, path)
            setValueByPath(toSave, path, value)
          })
        } else {
          // 保存整个状态
          toSave = state
        }

        storage.setItem(key, JSON.stringify(toSave))
      } catch (error) {
        console.error(`[PersistPlugin] Failed to persist state for ${store.$id}:`, error)
      }
    }, { detached: true })
  }
}

/**
 * 清除指定 store 的持久化数据
 */
export function clearPersistedState(storeId: string, key?: string) {
  const storageKey = key || `pinia-${storeId}`
  localStorage.removeItem(storageKey)
  sessionStorage.removeItem(storageKey)
}

/**
 * 清除所有持久化数据
 */
export function clearAllPersistedState() {
  const keys = [
    ...Object.keys(localStorage),
    ...Object.keys(sessionStorage)
  ]

  keys.forEach(key => {
    if (key.startsWith('pinia-')) {
      localStorage.removeItem(key)
      sessionStorage.removeItem(key)
    }
  })
}
