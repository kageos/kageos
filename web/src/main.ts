import './assets/main.css'
import 'element-plus/dist/index.css'
import './styles/theme.scss'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import { useThemeStore } from './stores/theme'

const app = createApp(App)
const pinia = createPinia()

// 配置持久化插件
pinia.use(piniaPluginPersistedstate)

app.use(pinia)
app.use(router)

// 初始化认证状态
const authStore = useAuthStore()
authStore.initAuth()

// 初始化主题
const themeStore = useThemeStore()
themeStore.initTheme()

app.mount('#app')
