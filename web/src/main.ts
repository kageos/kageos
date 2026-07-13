import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/inter/700.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'

import './architecture/presentation/assets/main.css'
import './architecture/presentation/styles/theme.scss'
import './architecture/presentation/styles/widgets.css'
import './architecture/presentation/assets/theme-workstation-sci-fi.css'
// Element Plus 组件样式由 unplugin（ElementPlusResolver importStyle: 'css'）按需注入，
// 这里只手动补充无法被 resolver 捕获的命令式/指令式组件样式。
import 'element-plus/es/components/loading/style/css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/notification/style/css'
import 'element-plus/es/components/message-box/style/css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
// 从深层路径导入指令，避免 unplugin-element-plus 按名字推出不存在的
// 'loading-directive/style/css'（其样式已在上面手动导入 loading/style/css）。
import { ElLoadingDirective } from 'element-plus/es/components/loading/index'

import App from './App.vue'
import router from './architecture/presentation/router'
import { useAuthStore } from './architecture/infrastructure/stores/auth'
import { useLocaleStore } from './architecture/infrastructure/stores/locale'
import { useThemeStore } from './architecture/infrastructure/stores/theme'
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

app.mount('#app')
