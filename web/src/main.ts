import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/inter/700.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'

import './architecture/presentation/assets/main.css'
import 'element-plus/dist/index.css'
import './architecture/presentation/styles/theme.scss'
import './architecture/presentation/styles/widgets.css'
import './architecture/presentation/assets/theme-workstation-sci-fi.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/notification/style/css'
import 'element-plus/es/components/message-box/style/css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import { ElLoadingDirective } from 'element-plus'

import App from './App.vue'
import router from './architecture/presentation/router'
import { useAuthStore } from './architecture/infrastructure/stores/auth'
import { useLocaleStore } from './architecture/infrastructure/stores/locale'
import { useThemeStore } from './architecture/infrastructure/stores/theme'
import { useUserInfoStore } from './architecture/infrastructure/stores/userInfo'
import { i18n } from './architecture/shared/i18n'
import { registerWidgetInitializers } from './architecture/presentation/widgets/initializers/registerInitializers'

const app = createApp(App)
const pinia = createPinia()

// 配置持久化插件
pinia.use(piniaPluginPersistedstate)

app.use(pinia)
app.use(router)
app.use(i18n)
app.directive('loading', ElLoadingDirective)

// 初始化认证状态
const authStore = useAuthStore()
authStore.initAuth()

// 初始化主题
const themeStore = useThemeStore()
themeStore.initTheme()

const localeStore = useLocaleStore()
localeStore.initLocale()

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
}

app.mount('#app')
