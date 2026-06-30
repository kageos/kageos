import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import ElementPlus from 'unplugin-element-plus/vite'

// https://vite.dev/config/
export default defineConfig(({ command, mode }) => {
  // 支持前端开发时「连线上后端」：.env.development.local 中设置 VITE_PROXY_TARGET=https://你的线上网关
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:9090'
  const minioTarget = env.VITE_MINIO_PROXY_TARGET || 'http://localhost:9000'
  const elementPlusResolver = ElementPlusResolver({ importStyle: 'css' })

  return {
  plugins: [
    vue(),
    vueJsx(),
    ...(command === 'serve' ? [vueDevTools()] : []),
    AutoImport({
      imports: ['vue', 'vue-router'],
      resolvers: [elementPlusResolver],
    }),
    Components({
      dirs: ['src/architecture/presentation/components', 'src/architecture/presentation/shared/components'],
      resolvers: [elementPlusResolver],
    }),
    // 给「<script> 里显式 import { ElXxx } from 'element-plus'」的组件自动补按需样式。
    // resolver 只处理模板自动导入的组件，本插件覆盖显式导入这条路径，二者互补。
    // 默认即按需注入编译好的 css（.../style/css），无需额外配置。
    ElementPlus({}),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  // 预声明依赖：让 Vite 在 dev 启动时一次性预构建，避免边浏览边发现 EP 组件样式 /
  // echarts 深层 install 模块而反复 re-optimize + 整页 reload（reload 半途会让
  // echarts 坐标系注册出现竞态，表现为 "cartesian2d cannot be found"）。
  optimizeDeps: {
    include: [
      // ElementPlus 按需样式（importStyle: 'css'），用 glob 一次性纳入
      'element-plus/es/components/*/style/css',
      // ChartRenderer 使用完整 ECharts runtime，避免 core/components/charts
      // 分包注册边界导致 dataZoom 拖动时 cartesian2d 丢失。
      'echarts',
    ],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return

          // Vue / Element Plus / icons 之间耦合很重，强行手拆容易形成循环依赖。
          // 这里只保留少数真正大的三方包单独分块，其余交给 Vite/Rollup 自动决策。
          if (id.includes('/zrender/')) {
            return 'vendor-zrender'
          }
          if (id.includes('echarts/lib/chart/bar/')) {
            return 'vendor-echarts-bar'
          }
          if (id.includes('echarts/lib/chart/line/')) {
            return 'vendor-echarts-line'
          }
          if (id.includes('echarts/lib/chart/pie/')) {
            return 'vendor-echarts-pie'
          }
          if (id.includes('echarts/lib/chart/gauge/')) {
            return 'vendor-echarts-gauge'
          }
          if (id.includes('echarts') || id.includes('vue-echarts')) {
            return 'vendor-echarts-core'
          }
          if (id.includes('monaco-editor') || id.includes('@monaco-editor')) {
            return 'vendor-monaco'
          }
          if (id.includes('marked')) {
            return 'vendor-markdown'
          }
        },
      },
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    proxy: {
      // Workspace API 通过网关代理（必须在 Vue Router 之前处理）
      // 注意：只代理 /workspace/api/* 路径，不代理 /workspace 页面路由
      '/workspace/api': {
        target: proxyTarget,
        changeOrigin: true,
        rewrite: (path) => path, // 不重写路径，直接转发
      },
      '/public/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Agent API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/agent/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Storage API：转发时带上浏览器原始 Host，供后端生成预签名 URL 与 PUT 一致（避免本地 403）
      '/storage/api': {
        target: proxyTarget,
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const host = req.headers.host
            if (host) proxyReq.setHeader('X-Forwarded-Host', host)
          })
        },
      },
      // HR API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/hr/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Connector API / OAuth callback 通过网关代理
      '/connector/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      '/connector/oauth': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Timer Scheduler API 通过网关代理
      '/timer/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Message API 通过网关代理
      '/message/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // 统一通过网关代理所有 API 请求（兜底，用于兼容旧路径）
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Swagger 文档也通过网关
      '/swagger': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // MinIO 文件代理。需保留浏览器 Host（changeOrigin: false）以便预签名与 MinIO 收到的 Host 一致；本地配置 cdn_domain 为前端 origin 如 http://localhost:5173
      '/kageos': {
        target: minioTarget,
        changeOrigin: false,
      },
    },
  },
  }
})
