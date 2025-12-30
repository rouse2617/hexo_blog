import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import { createPersistPlugin } from './stores/plugins/persist'
import { useAppStore } from './stores/app'
import { initNetworkMonitor, destroyNetworkMonitor } from './utils/networkMonitor'
import { errorHandler, ErrorLevel } from './utils/errorHandler'
import './styles/design-tokens.css'
import './styles/theme.css'

// 初始化网络监控
initNetworkMonitor()

// 配置全局错误处理器
errorHandler.setConfig({
  enableConsoleLog: import.meta.env.DEV, // 开发环境启用控制台日志
  enableNotification: true,
  enableMessage: true,
  onError: (error) => {
    // 可以在这里添加错误上报逻辑
    if (error.level === ErrorLevel.FATAL) {
      // 严重错误可以上报到监控系统
      console.error('[Fatal Error]', error)
    }
  }
})

const app = createApp(App)

// 创建 Pinia 实例并注册持久化插件
const pinia = createPinia()
pinia.use(createPersistPlugin({
  global: false, // 不全局启用，需要每个 store 单独配置
  storage: 'localStorage'
}))

// 注册所有 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(pinia)
app.use(router)

// 全局未捕获的错误处理
app.config.errorHandler = (err, instance, info) => {
  console.error('Vue Error:', err)
  console.error('Component:', instance)
  console.error('Info:', info)

  errorHandler.handle(err)
}

// 初始化应用状态（主题等）
const appStore = useAppStore()
appStore.initTheme()

app.mount('#app')

// 全局未捕获的 Promise rejection
window.addEventListener('unhandledrejection', (event) => {
  console.error('Unhandled Rejection:', event.reason)
  errorHandler.handle(event.reason)

  // 阻止默认的控制台错误输出
  event.preventDefault()
})

// 全局 JavaScript 错误
window.addEventListener('error', (event) => {
  console.error('Global Error:', event.error)
  errorHandler.handle(event.error)
})

// 应用卸载时清理
window.addEventListener('beforeunload', () => {
  destroyNetworkMonitor()
})
