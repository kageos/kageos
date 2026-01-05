import './assets/main.css'
import 'element-plus/dist/index.css'
import './styles/theme.scss'
import './styles/widgets.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'

import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import { useThemeStore } from './stores/theme'
import { useUserInfoStore } from './stores/userInfo'
import { registerWidgetInitializers } from './architecture/presentation/widgets/initializers/registerInitializers'
import { ensureInitialized } from './architecture/infrastructure/widgetRegistry'

const app = createApp(App)
const pinia = createPinia()

// 配置持久化插件
pinia.use(piniaPluginPersistedstate)

// 配置 Element Plus 中文语言包
app.use(ElementPlus, {
  locale: zhCn
})

app.use(pinia)
app.use(router)

// 初始化认证状态
const authStore = useAuthStore()
authStore.initAuth()

// 初始化主题
const themeStore = useThemeStore()
themeStore.initTheme()

// 🔥 注册所有 Widget 初始化器（组件自治，符合依赖倒置原则）
registerWidgetInitializers()

// 🔥 确保 Widget 组件工厂初始化完成后再挂载应用
// 这样可以避免刷新时出现"组件未找到"的闪现问题
// 注意：基础组件已经在模块加载时同步注册，这里只需要等待容器组件（FormWidget、TableWidget）注册完成
ensureInitialized()
  .then(() => {
    // 🔥 开发环境：将 stores 挂载到 window 对象，方便在控制台调试
    if (import.meta.env.DEV) {
      const userInfoStore = useUserInfoStore()
      ;(window as any).__stores__ = {
        authStore,
        themeStore,
        userInfoStore
      }
      console.log('[Dev] Stores 已挂载到 window.__stores__，可以在控制台访问：')
      console.log('  - window.__stores__.userInfoStore.getCacheStats()')
      console.log('  - window.__stores__.userInfoStore.clearCache()')
      console.log('  - window.__stores__.userInfoStore.refreshCache()')
    }

    // 所有组件注册完成，挂载应用
    app.mount('#app')
  })
  .catch((err) => {
    console.error('[main.ts] Widget 组件工厂初始化失败，应用仍将启动', err)
    // 即使初始化失败，也挂载应用（基础组件已经同步注册，大部分功能仍可用）
    app.mount('#app')
  })
