import './assets/main.css'
import './styles/theme.scss'
import './styles/widgets.css'
import './assets/theme-workstation-sci-fi.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/notification/style/css'
import 'element-plus/es/components/message-box/style/css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import { ElLoadingDirective } from 'element-plus'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

dayjs.locale('zh-cn')

import App from './App.vue'
import router from './architecture/infrastructure/router'
import { useAuthStore } from './architecture/infrastructure/stores/auth'
import { useThemeStore } from './architecture/infrastructure/stores/theme'
import { useUserInfoStore } from './architecture/infrastructure/stores/userInfo'
import { registerWidgetInitializers } from './architecture/presentation/widgets/initializers/registerInitializers'
import { ensureInitialized } from './architecture/infrastructure/widgetRegistry'

const app = createApp(App)
const pinia = createPinia()

// 配置持久化插件
pinia.use(piniaPluginPersistedstate)

app.use(pinia)
app.use(router)
app.directive('loading', ElLoadingDirective)

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
}

    // 所有组件注册完成，挂载应用
    app.mount('#app')
  })
  .catch((err) => {
    console.error('[main.ts] Widget 组件工厂初始化失败，应用仍将启动', err)
    // 即使初始化失败，也挂载应用（基础组件已经同步注册，大部分功能仍可用）
app.mount('#app')
  })
