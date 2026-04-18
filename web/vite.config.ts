import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

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
      dirs: ['src/components', 'src/shared/components'],
      resolvers: [elementPlusResolver],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/src/architecture/presentation/views/FormView.vue')) {
            return 'FormView'
          }

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
          if (id.includes('vditor')) {
            return 'vendor-vditor'
          }
          if (id.includes('@tiptap') || id.includes('prosemirror')) {
            return 'vendor-tiptap'
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
    host: true, // 允许通过局域网 IP 访问（如 http://192.168.3.19:5173）
    proxy: {
      // Workspace API 通过网关代理（必须在 Vue Router 之前处理）
      // 注意：只代理 /workspace/api/* 路径，不代理 /workspace 页面路由
      '/workspace/api': {
        target: proxyTarget,
        changeOrigin: true,
        rewrite: (path) => path, // 不重写路径，直接转发
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
      // Hub API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/hub/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // Control API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/control/api': {
        target: proxyTarget,
        changeOrigin: true,
      },
      // HR API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/hr/api': {
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
      '/ai-agent-os': {
        target: minioTarget,
        changeOrigin: false,
      },
    },
  },
  }
})
