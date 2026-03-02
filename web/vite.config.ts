import { fileURLToPath, URL } from 'node:url'

import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // 支持前端开发时「连线上后端」：.env.development.local 中设置 VITE_PROXY_TARGET=https://你的线上网关
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:9090'
  const minioTarget = env.VITE_MINIO_PROXY_TARGET || 'http://localhost:9000'

  return {
  plugins: [
    vue(),
    vueJsx(),
    vueDevTools(),
    AutoImport({
      imports: ['vue', 'vue-router'],
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
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
      // Storage API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/storage/api': {
        target: proxyTarget,
        changeOrigin: true,
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
      // MinIO 文件代理（开发环境直连本地 MinIO，连线上时可不配或指向线上 MinIO 代理路径）
      '/ai-agent-os': {
        target: minioTarget,
        changeOrigin: true,
      },
    },
  },
  }
})
