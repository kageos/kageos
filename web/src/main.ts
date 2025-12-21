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
import { registerWidgetInitializers } from './core/widgets-v2/initializers/registerInitializers'

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

app.mount('#app')
